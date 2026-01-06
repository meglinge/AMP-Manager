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
	"github.com/google/uuid"
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

		// Use original model from context if mapping was applied, otherwise use channelCfg.Model
		// This ensures response rewriting uses the original requested model name
		originalModel := channelCfg.Model
		if IsModelMappingApplied(c) {
			if origModel := GetOriginalModel(c); origModel != "" {
				originalModel = origModel
				log.Debugf("channel proxy: using original model '%s' for response rewriting (mapped to '%s')", origModel, GetMappedModel(c))
			}
		}

		targetURL, err := buildUpstreamURL(channel, c.Request)
		if err != nil {
			log.Errorf("channel proxy: failed to build upstream URL: %v", err)
			c.JSON(http.StatusInternalServerError, NewStandardError(http.StatusInternalServerError, "failed to build upstream URL"))
			return
		}

		parsed, err := url.Parse(targetURL)
		if err != nil {
			log.Errorf("channel proxy: failed to parse target URL: %v", err)
			c.JSON(http.StatusInternalServerError, NewStandardError(http.StatusInternalServerError, "invalid upstream URL"))
			return
		}

		// Get provider info for token extraction
		providerInfo := ProviderInfoFromChannel(channel)

		// Create RequestTrace for logging (only for model invocations)
		var trace *RequestTrace
		if IsModelInvocation(c.Request.Method, c.Request.URL.Path) {
			if cfg := GetProxyConfig(c.Request.Context()); cfg != nil {
				trace = NewRequestTrace(
					uuid.New().String(),
					cfg.UserID,
					cfg.APIKeyID,
					c.Request.Method,
					c.Request.URL.Path,
				)
				// Set channel info
				trace.SetChannel(channel.ID, string(channel.Type), channel.BaseURL)
				// Set model info
				mappedModel := channelCfg.Model
				if IsModelMappingApplied(c) {
					if m := GetMappedModel(c); m != "" {
						mappedModel = m
					}
				}
				trace.SetModels(originalModel, mappedModel)
				// Store trace in context
				c.Request = c.Request.WithContext(WithRequestTrace(c.Request.Context(), trace))
				log.Infof("channel proxy: model invocation %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, targetURL, originalModel)
			}
		} else {
			log.Debugf("channel proxy: %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, targetURL, originalModel)
		}

		proxy := &httputil.ReverseProxy{
			// 使用针对 AI 流式请求优化的 Transport，解决 60 秒超时问题
			Transport: NewStreamingTransport(),
			Director: func(req *http.Request) {
				req.URL.Scheme = parsed.Scheme
				req.URL.Host = parsed.Host
				req.URL.Path = parsed.Path
				req.URL.RawQuery = parsed.RawQuery
				req.Host = parsed.Host

				// Inject ProviderInfo into request context for token extraction
				*req = *req.WithContext(WithProviderInfo(req.Context(), providerInfo))

				// Remove client auth headers (ReverseProxy handles hop-by-hop headers automatically)
				req.Header.Del("Authorization")
				req.Header.Del("X-Api-Key")
				req.Header.Del("x-api-key")
				req.Header.Del("X-Goog-Api-Key")
				req.Header.Del("x-goog-api-key")

				// Filter Anthropic-Beta header for local/channel handling paths
				filterAntropicBetaHeader(req)

				// Apply channel-specific authentication
				applyChannelAuth(channel, req)

				// For OpenAI Chat, inject stream_options.include_usage=true for streaming requests
				if channel.Type == model.ChannelTypeOpenAI && channel.Endpoint != model.ChannelEndpointResponses {
					injectOpenAIStreamOptions(req)
				}

				// Apply custom headers from channel config
				var headersMap map[string]string
				if err := json.Unmarshal([]byte(channel.HeadersJSON), &headersMap); err == nil {
					for k, v := range headersMap {
						req.Header.Set(k, v)
					}
				}

				// For Gemini, ensure no auth headers conflict with query key
				if channel.Type == model.ChannelTypeGemini {
					req.Header.Del("Authorization")
					req.Header.Del("X-Api-Key")
					req.Header.Del("x-api-key")
					req.Header.Del("X-Goog-Api-Key")
					req.Header.Del("x-goog-api-key")
				}
			},
			FlushInterval: -1, // Flush immediately for SSE streaming support
			ModifyResponse: func(resp *http.Response) error {
				trace := GetRequestTrace(resp.Request.Context())

				// Log non-2xx responses
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					log.Warnf("channel proxy: upstream returned status %d for %s", resp.StatusCode, targetURL)
					if trace != nil {
						trace.SetError("upstream_error")
						resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
					}
					return nil
				}

				// Extract token usage from response and wrap for logging
				if trace != nil {
					info, _ := GetProviderInfo(resp.Request.Context())
					isStreaming := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
					resp.Body = WrapResponseBodyForTokenExtraction(resp.Body, isStreaming, trace, info)
					// Wrap again for logging on close
					resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
				}

				return nil
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				log.Errorf("channel proxy: upstream request failed: %v", err)
				// Write error log
				if trace != nil {
					trace.SetError("upstream_request_failed")
					trace.SetResponse(http.StatusBadGateway)
					if writer := GetLogWriter(); writer != nil {
						writer.WriteFromTrace(trace)
					}
				}
				WriteErrorResponse(rw, http.StatusBadGateway, "Upstream request failed: "+err.Error())
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

// injectOpenAIStreamOptions 为 OpenAI Chat 流式请求注入 stream_options.include_usage=true
// 这是获取 streaming 响应中 usage 数据的必要条件
func injectOpenAIStreamOptions(req *http.Request) {
	if req.Body == nil || req.ContentLength == 0 {
		return
	}

	contentType := req.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	req.Body.Close()

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		return
	}

	// 检查是否为流式请求
	stream, ok := payload["stream"].(bool)
	if !ok || !stream {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		return
	}

	// 检查 stream_options 是否已存在
	streamOptions, ok := payload["stream_options"].(map[string]interface{})
	if !ok {
		streamOptions = make(map[string]interface{})
		payload["stream_options"] = streamOptions
	}

	// 设置 include_usage = true
	if _, exists := streamOptions["include_usage"]; !exists {
		streamOptions["include_usage"] = true
		log.Debugf("channel proxy: injected stream_options.include_usage=true for OpenAI streaming")
	}

	newBody, err := json.Marshal(payload)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		return
	}

	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
}
