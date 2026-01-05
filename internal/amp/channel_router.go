package amp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func extractModelName(c *gin.Context) string {
	path := c.Request.URL.Path
	if strings.Contains(path, "/v1beta/models/") {
		parts := strings.Split(path, "/v1beta/models/")
		if len(parts) > 1 {
			modelPart := parts[1]
			if idx := strings.Index(modelPart, ":"); idx > 0 {
				return modelPart[:idx]
			}
			if idx := strings.Index(modelPart, "/"); idx > 0 {
				return modelPart[:idx]
			}
			return modelPart
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

	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return ""
	}

	return payload.Model
}

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
		originalModel := channelCfg.Model // Store original model for response rewriting

		targetURL, err := buildUpstreamURL(channel, c.Request)
		if err != nil {
			log.Errorf("channel proxy: failed to build upstream URL: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		log.Infof("channel proxy: %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, targetURL, originalModel)

		proxyReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			log.Errorf("channel proxy: failed to create request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		for key, values := range c.Request.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		proxyReq.Header.Del("Authorization")
		proxyReq.Header.Del("X-Api-Key")
		proxyReq.Header.Del("x-api-key")
		proxyReq.Header.Del("Transfer-Encoding")

		// Filter Anthropic-Beta header for local/channel handling paths
		// This prevents 1M context from being used when going through local channels
		filterAntropicBetaHeader(proxyReq)

		applyChannelAuth(channel, proxyReq)

		var headersMap map[string]string
		if err := json.Unmarshal([]byte(channel.HeadersJSON), &headersMap); err == nil {
			for k, v := range headersMap {
				proxyReq.Header.Set(k, v)
			}
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Errorf("channel proxy: upstream request failed: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request failed"})
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(key, value)
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)

		// Use ResponseRewriter to rewrite model name back to original
		// This is the key mechanism: Amp client uses the returned model name to determine context length
		rewriter := NewResponseRewriter(c.Writer, originalModel)

		if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			flusher, ok := c.Writer.(http.Flusher)
			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					rewriter.Write(buf[:n])
					if ok {
						flusher.Flush()
					}
				}
				if err != nil {
					break
				}
			}
		} else {
			body, _ := io.ReadAll(resp.Body)
			rewriter.Write(body)
			rewriter.Flush()
		}
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
