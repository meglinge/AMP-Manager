package amp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"ampmanager/internal/model"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type channelConfigKey struct{}

type ChannelConfig struct {
	Channel *model.Channel
	Model   string
}

func WithChannelConfig(c *gin.Context, cfg *ChannelConfig) {
	c.Set("channel_config", cfg)
}

func GetChannelConfig(c *gin.Context) *ChannelConfig {
	if val, exists := c.Get("channel_config"); exists {
		if cfg, ok := val.(*ChannelConfig); ok {
			return cfg
		}
	}
	return nil
}

var channelService = service.NewChannelService()

func ChannelRouterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelName := extractModelName(c)
		if modelName == "" {
			c.Next()
			return
		}

		channel, err := channelService.SelectChannelForModel(modelName)
		if err != nil {
			log.Errorf("channel router: failed to select channel: %v", err)
			c.Next()
			return
		}

		if channel == nil {
			c.Next()
			return
		}

		log.Infof("channel router: routing model '%s' to channel '%s' (%s)", modelName, channel.Name, channel.Type)
		WithChannelConfig(c, &ChannelConfig{
			Channel: channel,
			Model:   modelName,
		})

		c.Next()
	}
}

// extractModelFromPathPart extracts model name from path segment like "gemini-3-flash:generateContent"
func extractModelFromPathPart(modelPart string) string {
	if idx := strings.Index(modelPart, ":"); idx > 0 {
		return modelPart[:idx]
	}
	if idx := strings.Index(modelPart, "/"); idx > 0 {
		return modelPart[:idx]
	}
	return modelPart
}

func extractModelName(c *gin.Context) string {
	path := c.Request.URL.Path

	// Handle v1beta1/publishers/google/models/ path (used by Amp CLI sub-agents)
	if strings.Contains(path, "/v1beta1/publishers/google/models/") {
		parts := strings.Split(path, "/v1beta1/publishers/google/models/")
		if len(parts) > 1 {
			return extractModelFromPathPart(parts[1])
		}
	}

	// Handle v1beta/models/ path
	if strings.Contains(path, "/v1beta/models/") {
		parts := strings.Split(path, "/v1beta/models/")
		if len(parts) > 1 {
			return extractModelFromPathPart(parts[1])
		}
	}

	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return ""
	}

	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ""
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Request.ContentLength = int64(len(bodyBytes))
	c.Request.TransferEncoding = nil

	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return ""
	}

	return payload.Model
}

// rewritingResponseWriter wraps gin.ResponseWriter to rewrite model names in responses
type rewritingResponseWriter struct {
	gin.ResponseWriter
	rewriter *ResponseRewriter
}

func newRewritingResponseWriter(w gin.ResponseWriter, originalModel string) *rewritingResponseWriter {
	return &rewritingResponseWriter{
		ResponseWriter: w,
		rewriter:       NewResponseRewriter(w, originalModel),
	}
}

func (rw *rewritingResponseWriter) Write(data []byte) (int, error) {
	return rw.rewriter.Write(data)
}

func (rw *rewritingResponseWriter) Flush() {
	rw.rewriter.Flush()
	rw.ResponseWriter.Flush()
}

// ChannelProxyHandler creates a handler using httputil.ReverseProxy for robust proxying
func ChannelProxyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		channelCfg := GetChannelConfig(c)
		if channelCfg == nil || channelCfg.Channel == nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "no channel available for this request",
			})
			return
		}

		channel := channelCfg.Channel
		originalModel := channelCfg.Model

		targetURL, err := buildUpstreamURL(channel, c.Request)
		if err != nil {
			log.Errorf("channel proxy: failed to build upstream URL: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		parsed, err := url.Parse(targetURL)
		if err != nil {
			log.Errorf("channel proxy: failed to parse target URL: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		log.Infof("channel proxy: %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, targetURL, originalModel)

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = parsed.Scheme
				req.URL.Host = parsed.Host
				req.URL.Path = parsed.Path
				req.URL.RawQuery = parsed.RawQuery
				req.Host = parsed.Host

				// Remove client auth headers (ReverseProxy handles hop-by-hop headers automatically)
				req.Header.Del("Authorization")
				req.Header.Del("X-Api-Key")
				req.Header.Del("x-api-key")

				// Filter Anthropic-Beta header for local/channel handling paths
				filterAntropicBetaHeader(req)

				// Apply channel-specific authentication
				applyChannelAuth(channel, req)

				// Apply custom headers from channel config
				var headersMap map[string]string
				if err := json.Unmarshal([]byte(channel.HeadersJSON), &headersMap); err == nil {
					for k, v := range headersMap {
						req.Header.Set(k, v)
					}
				}
			},
			FlushInterval: -1, // Flush immediately for SSE streaming support
			ModifyResponse: func(resp *http.Response) error {
				// Log non-2xx responses for debugging
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					log.Warnf("channel proxy: upstream returned status %d for %s", resp.StatusCode, targetURL)
				}
				return nil
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				log.Errorf("channel proxy: upstream request failed: %v", err)
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusBadGateway)
				_, _ = rw.Write([]byte(`{"error":"upstream request failed"}`))
			},
		}

		// Wrap ResponseWriter to rewrite model names in responses
		wrappedWriter := newRewritingResponseWriter(c.Writer, originalModel)
		proxy.ServeHTTP(wrappedWriter, c.Request)
	}
}

func buildUpstreamURL(channel *model.Channel, req *http.Request) (string, error) {
	parsed, err := url.Parse(channel.BaseURL)
	if err != nil {
		return "", err
	}

	upstreamPath := getEndpointPath(channel, req)

	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + upstreamPath
	parsed.RawQuery = req.URL.RawQuery

	if channel.Type == model.ChannelTypeGemini {
		q := parsed.Query()
		q.Set("key", channel.APIKey)
		parsed.RawQuery = q.Encode()
	}

	return parsed.String(), nil
}

func getEndpointPath(channel *model.Channel, req *http.Request) string {
	originalPath := req.URL.Path

	switch channel.Type {
	case model.ChannelTypeOpenAI:
		if channel.Endpoint == model.ChannelEndpointResponses {
			return "/v1/responses"
		}
		return "/v1/chat/completions"

	case model.ChannelTypeClaude:
		return "/v1/messages"

	case model.ChannelTypeGemini:
		// Convert v1beta1/publishers/google/models/X:action to v1beta/models/X:action
		if strings.Contains(originalPath, "/v1beta1/publishers/google/models/") {
			parts := strings.Split(originalPath, "/v1beta1/publishers/google/models/")
			if len(parts) > 1 {
				return "/v1beta/models/" + parts[1]
			}
		}
		if strings.Contains(originalPath, "/v1beta/models/") {
			return originalPath
		}
		return originalPath
	}

	return originalPath
}

func applyChannelAuth(channel *model.Channel, req *http.Request) {
	switch channel.Type {
	case model.ChannelTypeOpenAI:
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.APIKey))
	case model.ChannelTypeClaude:
		req.Header.Set("x-api-key", channel.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case model.ChannelTypeGemini:
	}
}
