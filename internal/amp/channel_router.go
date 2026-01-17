package amp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"strconv"

	"ampmanager/internal/model"
	"ampmanager/internal/service"
	"ampmanager/internal/translator"
	"ampmanager/internal/translator/filters"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// sharedChannelTransport 是共享的 Channel Proxy Transport，用于连接复用
var sharedChannelTransport = NewStreamingTransport()

// translationContextKey is used to store translation info in context
type translationContextKey struct{}

// responseWriterContextKey is used to store ResponseWriter in context for SSE keep-alive
type responseWriterContextKey struct{}

// WithResponseWriter stores ResponseWriter in context
func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, responseWriterContextKey{}, w)
}

// GetResponseWriter retrieves ResponseWriter from context
func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	if val := ctx.Value(responseWriterContextKey{}); val != nil {
		if w, ok := val.(http.ResponseWriter); ok {
			return w
		}
	}
	return nil
}

// TranslationInfo holds translation state for request/response conversion
type TranslationInfo struct {
	NeedsConversion     bool
	IncomingFormat      translator.Format
	OutgoingFormat      translator.Format
	OriginalRequestBody []byte
	ConvertedBody       []byte
	IsStreaming         bool
	Model               string // originalModel - for response rewriting
	UpstreamModel       string // mappedModel - for upstream URL path
	ResponseParam       *any
}

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

// WithTranslationInfo stores translation info in context
func WithTranslationInfo(ctx context.Context, info *TranslationInfo) context.Context {
	return context.WithValue(ctx, translationContextKey{}, info)
}

// GetTranslationInfo retrieves translation info from context
func GetTranslationInfo(ctx context.Context) *TranslationInfo {
	if val := ctx.Value(translationContextKey{}); val != nil {
		if info, ok := val.(*TranslationInfo); ok {
			return info
		}
	}
	return nil
}

// detectIncomingFormat determines the request format based on the request path
func detectIncomingFormat(path string) translator.Format {
	switch {
	case strings.Contains(path, "/v1/chat/completions"):
		return translator.FormatOpenAIChat
	case strings.Contains(path, "/v1/responses"):
		return translator.FormatOpenAIResponses
	case strings.Contains(path, "/v1/messages"):
		return translator.FormatClaude
	case strings.Contains(path, "/v1beta/models/") || strings.Contains(path, "/v1beta1/publishers/google/models/"):
		return translator.FormatGemini
	default:
		return translator.FormatOpenAI // Default to OpenAI format
	}
}

// channelTypeToFormat converts channel type and endpoint to translator format
func channelTypeToFormat(channel *model.Channel) translator.Format {
	if channel == nil {
		return translator.FormatOpenAI
	}
	switch channel.Type {
	case model.ChannelTypeOpenAI:
		if channel.Endpoint == model.ChannelEndpointResponses {
			return translator.FormatOpenAIResponses
		}
		return translator.FormatOpenAIChat
	case model.ChannelTypeClaude:
		return translator.FormatClaude
	case model.ChannelTypeGemini:
		return translator.FormatGemini
	default:
		return translator.FormatOpenAI
	}
}

// needsFormatConversion checks if request/response format conversion is needed
func needsFormatConversion(incoming, outgoing translator.Format) bool {
	return incoming != outgoing
}

// getTargetEndpointPath returns the correct endpoint path for the target format
func getTargetEndpointPath(targetFormat translator.Format, channel *model.Channel) string {
	switch targetFormat {
	case translator.FormatOpenAI:
		if channel != nil && channel.Endpoint == model.ChannelEndpointResponses {
			return "/v1/responses"
		}
		return "/v1/chat/completions"
	case translator.FormatOpenAIChat:
		return "/v1/chat/completions"
	case translator.FormatOpenAIResponses:
		return "/v1/responses"
	case translator.FormatClaude:
		return "/v1/messages"
	case translator.FormatGemini:
		// Gemini paths are handled separately with model name in path
		return ""
	default:
		return "/v1/chat/completions"
	}
}

// sanitizeURL removes sensitive query parameters from URL for safe logging
func sanitizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid-url]"
	}
	q := parsed.Query()
	sensitiveKeys := []string{"key", "api_key", "apikey", "token"}
	for _, k := range sensitiveKeys {
		if q.Has(k) {
			q.Set(k, "[REDACTED]")
		}
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

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

	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024))
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
		// Security guard: ensure authentication was performed via proxy middleware
		if GetProxyConfig(c.Request.Context()) == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			return
		}

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

		// Detect incoming and outgoing formats for potential translation
		// First check if XMLTagRoutingMiddleware detected a format from body structure
		incomingFormat := GetXMLTagDetectedFormat(c)
		if incomingFormat == "" {
			incomingFormat = detectIncomingFormat(c.Request.URL.Path)
		}
		outgoingFormat := channelTypeToFormat(channel)
		needsTranslation := needsFormatConversion(incomingFormat, outgoingFormat)

		// Check if translation is possible when needed
		if needsTranslation {
			// Check if we have a request transformer registered
			testBody := []byte(`{"model":"test"}`)
			convertedTest, _ := translator.TranslateRequest(incomingFormat, outgoingFormat, "test", testBody, false)
			if bytes.Equal(testBody, convertedTest) {
				// No transformer registered, return error
				log.Warnf("channel proxy: format conversion from %s to %s not supported", incomingFormat, outgoingFormat)
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("format conversion from %s to %s is not supported", incomingFormat, outgoingFormat),
				})
				return
			}
			log.Infof("channel proxy: translating request from %s to %s", incomingFormat, outgoingFormat)
		}

		// Read and cache original request body for translation
		var originalRequestBody []byte
		var convertedBody []byte
		isStreaming := false
		if needsTranslation && c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024))
			c.Request.Body.Close()
			if err != nil {
				log.Errorf("channel proxy: failed to read request body for translation: %v", err)
				c.JSON(http.StatusInternalServerError, NewStandardError(http.StatusInternalServerError, "failed to read request body"))
				return
			}
			originalRequestBody = bodyBytes

			// Check if streaming
			var payload struct {
				Stream bool `json:"stream"`
			}
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				isStreaming = payload.Stream
			}

			// Translate request
			convertedBody, err = translator.TranslateRequest(incomingFormat, outgoingFormat, channelCfg.Model, bodyBytes, isStreaming)
			if err != nil {
				log.Warnf("channel proxy: request translation failed: %v, using original request", err)
				convertedBody = bodyBytes
			}

			// Apply outgoing format filters (e.g., Claude system string to array)
			filteredBody, filterErr := filters.ApplyFilters(outgoingFormat, convertedBody)
			if filterErr != nil {
				log.Warnf("channel proxy: filter application failed: %v, using unfiltered body", filterErr)
			} else {
				convertedBody = filteredBody
			}

			// /v1/responses: if client asked for non-stream, force upstream stream=true.
			// We'll aggregate the upstream SSE back to a single non-stream JSON response.
			if !isStreaming && strings.Contains(c.Request.URL.Path, "/v1/responses") {
				forcedBody, forced := forceJSONStreamTrue(convertedBody)
				if forced {
					convertedBody = forcedBody
					isStreaming = true
				}
			}

			// Restore body with converted content
			c.Request.Body = io.NopCloser(bytes.NewReader(convertedBody))
			c.Request.ContentLength = int64(len(convertedBody))
			c.Request.Header.Set("Content-Length", fmt.Sprintf("%d", len(convertedBody)))
		} else if !needsTranslation && c.Request.Body != nil && c.Request.ContentLength > 0 {
			// No translation needed, but still apply filters for the outgoing format
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024))
			c.Request.Body.Close()
			if err != nil {
				log.Errorf("channel proxy: failed to read request body for filtering: %v", err)
				c.JSON(http.StatusInternalServerError, NewStandardError(http.StatusInternalServerError, "failed to read request body"))
				return
			}
			originalRequestBody = bodyBytes
			convertedBody = bodyBytes

			// Check if streaming
			var payload struct {
				Stream bool `json:"stream"`
			}
			if err := json.Unmarshal(bodyBytes, &payload); err == nil {
				isStreaming = payload.Stream
			}

			// Apply outgoing format filters
			filteredBody, filterErr := filters.ApplyFilters(outgoingFormat, bodyBytes)
			if filterErr != nil {
				log.Warnf("channel proxy: filter application failed: %v, using unfiltered body", filterErr)
			} else if !bytes.Equal(filteredBody, bodyBytes) {
				convertedBody = filteredBody
				c.Request.Body = io.NopCloser(bytes.NewReader(convertedBody))
				c.Request.ContentLength = int64(len(convertedBody))
				c.Request.Header.Set("Content-Length", fmt.Sprintf("%d", len(convertedBody)))
			} else {
				// No changes, restore original body
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			// /v1/responses: if client asked for non-stream, force upstream stream=true.
			// We'll aggregate the upstream SSE back to a single non-stream JSON response.
			if !isStreaming && strings.Contains(c.Request.URL.Path, "/v1/responses") {
				forcedBody, forced := forceJSONStreamTrue(bodyBytes)
				if forced {
					convertedBody = forcedBody
					isStreaming = true
					c.Request.Body = io.NopCloser(bytes.NewReader(convertedBody))
					c.Request.ContentLength = int64(len(convertedBody))
					c.Request.Header.Set("Content-Length", fmt.Sprintf("%d", len(convertedBody)))
				}
			}
		}

		// Store translation info in context for response processing
		var responseParam any
		translationInfo := &TranslationInfo{
			NeedsConversion:     needsTranslation,
			IncomingFormat:      incomingFormat,
			OutgoingFormat:      outgoingFormat,
			OriginalRequestBody: originalRequestBody,
			ConvertedBody:       convertedBody,
			IsStreaming:         isStreaming,
			Model:               originalModel,    // 用于响应重写
			UpstreamModel:       channelCfg.Model, // 用于上游路径
			ResponseParam:       &responseParam,
		}
		c.Request = c.Request.WithContext(WithTranslationInfo(c.Request.Context(), translationInfo))

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
				// Set thinking level if applied
				if thinkingLevel := GetThinkingLevel(c); thinkingLevel != "" {
					trace.SetThinkingLevel(thinkingLevel)
				}
				// Store trace in context
				c.Request = c.Request.WithContext(WithRequestTrace(c.Request.Context(), trace))

				// Write pending record to database immediately
				if writer := GetLogWriter(); writer != nil {
					writer.WritePendingFromTrace(trace)
				}

				// Capture request detail for logging (same as amp upstream proxy)
				if captureData := GetCaptureData(c.Request.Context()); captureData != nil {
					StoreRequestDetail(trace.RequestID, captureData.RequestHeaders, captureData.RequestBody)
				}

				// Store translated request body if different from original
				if transInfo := GetTranslationInfo(c.Request.Context()); transInfo != nil && transInfo.NeedsConversion && len(transInfo.ConvertedBody) > 0 {
					StoreTranslatedRequestBody(trace.RequestID, transInfo.ConvertedBody)
				}

				log.Infof("channel proxy: model invocation %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, sanitizeURL(targetURL), originalModel)
			}
		} else {
			log.Debugf("channel proxy: %s %s -> %s (model: %s)", c.Request.Method, c.Request.URL.Path, sanitizeURL(targetURL), originalModel)
		}

		proxy := &httputil.ReverseProxy{
			// 使用共享的流式 Transport，支持连接复用
			Transport: sharedChannelTransport,
			Director: func(req *http.Request) {
				req.URL.Scheme = parsed.Scheme
				req.URL.Host = parsed.Host
				req.URL.Path = parsed.Path
				req.URL.RawQuery = parsed.RawQuery
				req.Host = parsed.Host

				// Inject ProviderInfo into request context for token extraction
				*req = *req.WithContext(WithProviderInfo(req.Context(), providerInfo))

				// Inject ResponseWriter for SSE keep-alive support
				*req = *req.WithContext(WithResponseWriter(req.Context(), c.Writer))

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

				// Track original client streaming preference for response conversion.
				// For /v1/responses we may force upstream streaming and later aggregate SSE back to non-stream JSON.
				mode := StreamMode{}
				if transInfo := GetTranslationInfo(req.Context()); transInfo != nil {
					mode.ClientWantsStream = transInfo.IsStreaming
				}
				*req = *req.WithContext(WithStreamMode(req.Context(), mode))

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
				transInfo := GetTranslationInfo(resp.Request.Context())
				isStreaming := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

				// /v1/responses: if client requested non-stream but upstream responded with SSE,
				// aggregate the SSE into a single JSON response.
				if isStreaming && strings.Contains(resp.Request.URL.Path, "/v1/responses") {
					if mode, ok := GetStreamMode(resp.Request.Context()); ok && !mode.ClientWantsStream {
						jsonBody, aggErr := aggregateOpenAIResponsesSSEToJSON(resp.Request.Context(), resp.Body)
						_ = resp.Body.Close()
						if aggErr != nil {
							return aggErr
						}
						resp.Body = io.NopCloser(bytes.NewReader(jsonBody))
						resp.Header.Set("Content-Type", "application/json")
						resp.Header.Del("Content-Encoding")
						resp.Header.Del("Transfer-Encoding")
						resp.TransferEncoding = nil
						resp.ContentLength = int64(len(jsonBody))
						resp.Header.Set("Content-Length", strconv.Itoa(len(jsonBody)))
						isStreaming = false
					}
				}

				// Log non-2xx responses
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					log.Warnf("channel proxy: upstream returned status %d for %s", resp.StatusCode, sanitizeURL(targetURL))
					if trace != nil {
						trace.SetError("upstream_error")
						resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
					}
					return nil
				}

				// For non-streaming responses, read the complete body upfront,
				// apply all transformations, then reset body with correct Content-Length
				if !isStreaming {
					return handleNonStreamingResponse(resp, trace, transInfo, originalModel)
				}

				// Streaming response handling (existing logic)
				if trace != nil {
					info, _ := GetProviderInfo(resp.Request.Context())
					resp.Body = WrapResponseBodyForTokenExtraction(resp.Body, isStreaming, trace, info)
					resp.Body = NewResponseCaptureWrapper(resp.Body, trace.RequestID, resp.Header)
					resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
				}

				// Apply response translation if needed for streaming
				// Note: Registry keys are [IncomingFormat][OutgoingFormat] based on request direction
				// For response translation: IncomingFormat is client format, OutgoingFormat is upstream format
				if transInfo != nil && transInfo.NeedsConversion {
					resp.Body = newTranslatingResponseBody(
						resp.Request.Context(),
						resp.Body,
						transInfo.IncomingFormat, // from = client format (registry key)
						transInfo.OutgoingFormat, // to = upstream format (registry key)
						transInfo.Model,
						transInfo.OriginalRequestBody,
						transInfo.ConvertedBody,
						true, // isStreaming
						transInfo.ResponseParam,
					)
					log.Debugf("channel proxy: translating response from %s to %s", transInfo.OutgoingFormat, transInfo.IncomingFormat)
				}

				// Wrap SSE responses with keep-alive for long-running streams
				if rw := GetResponseWriter(resp.Request.Context()); rw != nil {
					if wrapper := NewSSEKeepAliveWrapper(resp.Body, rw, resp.Request.Context(), nil); wrapper != nil {
						resp.Body = wrapper
						log.Debugf("channel proxy: enabled SSE keep-alive for streaming response")
					}
				}

				return nil
			},
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				log.Errorf("channel proxy: upstream request failed: %v", err)
				// Update error log (pending record was already written)
				if trace != nil {
					trace.SetError("upstream_request_failed")
					trace.SetResponse(http.StatusBadGateway)
					if writer := GetLogWriter(); writer != nil {
						writer.UpdateFromTrace(trace)
					}
				}
				// 使用清理后的错误消息，防止泄露敏感信息
				safeMsg := SanitizeError(err)
				WriteErrorResponse(rw, http.StatusBadGateway, "Upstream request failed: "+safeMsg)
			},
		}

		// Wrap ResponseWriter to rewrite model names in responses
		wrappedWriter := newRewritingResponseWriter(c.Writer, originalModel)
		proxy.ServeHTTP(wrappedWriter, c.Request)
		wrappedWriter.Flush() // 确保非流式响应被发送给客户端
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
		// For streaming requests, add alt=sse to get SSE format responses
		if strings.Contains(upstreamPath, "streamGenerateContent") {
			q.Set("alt", "sse")
		}
		parsed.RawQuery = q.Encode()
	}

	return parsed.String(), nil
}

func getEndpointPath(channel *model.Channel, req *http.Request) string {
	originalPath := req.URL.Path

	// Check if we need format conversion (OpenAI request -> Gemini channel)
	transInfo := GetTranslationInfo(req.Context())

	switch channel.Type {
	case model.ChannelTypeOpenAI:
		if channel.Endpoint == model.ChannelEndpointResponses {
			return "/v1/responses"
		}
		return "/v1/chat/completions"

	case model.ChannelTypeClaude:
		return "/v1/messages"

	case model.ChannelTypeGemini:
		// When format conversion is needed (e.g., OpenAI request -> Gemini channel),
		// we need to construct the correct Gemini path with model name
		if transInfo != nil && transInfo.NeedsConversion {
			incomingFormat := transInfo.IncomingFormat
			// OpenAI/Claude format incoming, need to build Gemini path
			if incomingFormat == translator.FormatOpenAI || incomingFormat == translator.FormatClaude {
				modelName := transInfo.UpstreamModel // 使用 UpstreamModel
				if modelName != "" {
					// 移除可能存在的 models/ 前缀
					modelName = strings.TrimPrefix(modelName, "models/")
					// Gemini uses :streamGenerateContent for streaming, :generateContent for non-streaming
					action := "generateContent"
					if transInfo.IsStreaming {
						action = "streamGenerateContent"
					}
					return fmt.Sprintf("/v1beta/models/%s:%s", modelName, action)
				}
			}
		}

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

	bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, 10*1024*1024))
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

// translatingResponseBody wraps an io.ReadCloser to translate response data
type translatingResponseBody struct {
	ctx                 context.Context
	reader              io.ReadCloser
	from                translator.Format
	to                  translator.Format
	model               string
	originalRequestBody []byte
	convertedBody       []byte
	isStreaming         bool
	param               *any
	buffer              bytes.Buffer
	translatedBuffer    bytes.Buffer
	sseBuffer           bytes.Buffer // Buffer for accumulating SSE events (split by \n\n or \r\n\r\n)
	eofReached          bool         // Track if EOF has been reached
}

// maxSSEBufferSize is the maximum size limit for SSE buffer to prevent OOM (10MB)
const maxSSEBufferSize = 10 * 1024 * 1024

// findSSEDelimiter finds the earliest SSE delimiter (\n\n or \r\n\r\n) in data
func findSSEDelimiter(data []byte) (idx int, delimLen int) {
	lfIdx := bytes.Index(data, []byte("\n\n"))
	crlfIdx := bytes.Index(data, []byte("\r\n\r\n"))

	if lfIdx == -1 && crlfIdx == -1 {
		return -1, 0
	}
	if lfIdx == -1 {
		return crlfIdx, 4
	}
	if crlfIdx == -1 {
		return lfIdx, 2
	}
	if lfIdx < crlfIdx {
		return lfIdx, 2
	}
	return crlfIdx, 4
}

func newTranslatingResponseBody(
	ctx context.Context,
	reader io.ReadCloser,
	from, to translator.Format,
	model string,
	originalRequestBody, convertedBody []byte,
	isStreaming bool,
	param *any,
) *translatingResponseBody {
	return &translatingResponseBody{
		ctx:                 ctx,
		reader:              reader,
		from:                from,
		to:                  to,
		model:               model,
		originalRequestBody: originalRequestBody,
		convertedBody:       convertedBody,
		isStreaming:         isStreaming,
		param:               param,
	}
}

func (t *translatingResponseBody) Read(p []byte) (int, error) {
	// If we have translated data buffered, return that first
	if t.translatedBuffer.Len() > 0 {
		return t.translatedBuffer.Read(p)
	}

	if t.isStreaming {
		return t.readStreaming(p)
	}
	return t.readNonStreaming(p)
}

func (t *translatingResponseBody) readStreaming(p []byte) (int, error) {
	// If we have translated data buffered, return that first
	if t.translatedBuffer.Len() > 0 {
		return t.translatedBuffer.Read(p)
	}

	// If EOF already reached, return EOF
	if t.eofReached {
		return 0, io.EOF
	}

	// Loop until we have data to return or encounter an error
	for {
		// Check SSE buffer size limit to prevent OOM
		if t.sseBuffer.Len() > maxSSEBufferSize {
			return 0, fmt.Errorf("SSE buffer overflow: exceeds %d bytes", maxSSEBufferSize)
		}

		// Read from upstream
		buf := make([]byte, 4096)
		n, err := t.reader.Read(buf)
		if n > 0 {
			t.sseBuffer.Write(buf[:n])
		}

		// Process complete SSE events (delimited by \n\n or \r\n\r\n)
		for {
			data := t.sseBuffer.Bytes()
			idx, delimLen := findSSEDelimiter(data)
			if idx == -1 {
				break
			}

			// Extract complete event (including the delimiter)
			event := make([]byte, idx+delimLen)
			copy(event, data[:idx+delimLen])

			// Remove processed data from buffer
			t.sseBuffer.Reset()
			t.sseBuffer.Write(data[idx+delimLen:])

			// Translate this SSE event
			translated, translateErr := translator.TranslateStream(
				t.ctx,
				t.from,
				t.to,
				t.model,
				t.originalRequestBody,
				t.convertedBody,
				event,
				t.param,
			)
			if translateErr != nil {
				log.Warnf("channel proxy: stream translation failed: %v, using original data", translateErr)
				translated = []string{string(event)}
			}

			for _, chunk := range translated {
				t.translatedBuffer.WriteString(chunk)
				// 记录翻译后的响应用于调试
				if store := GetRequestDetailStore(); store != nil {
					if trace := GetRequestTrace(t.ctx); trace != nil {
						store.AppendTranslatedResponse(trace.RequestID, []byte(chunk))
					}
				}
			}
		}

		// Return translated data if available
		if t.translatedBuffer.Len() > 0 {
			return t.translatedBuffer.Read(p)
		}

		// Handle EOF
		if err == io.EOF {
			t.eofReached = true
			// Process any remaining data in buffer
			if t.sseBuffer.Len() > 0 {
				remaining := t.sseBuffer.Bytes()
				t.sseBuffer.Reset()
				translated, translateErr := translator.TranslateStream(
					t.ctx,
					t.from,
					t.to,
					t.model,
					t.originalRequestBody,
					t.convertedBody,
					remaining,
					t.param,
				)
				if translateErr != nil {
					log.Warnf("channel proxy: stream translation failed: %v, using original data", translateErr)
					translated = []string{string(remaining)}
				}
				for _, chunk := range translated {
					t.translatedBuffer.WriteString(chunk)
				}
				if t.translatedBuffer.Len() > 0 {
					return t.translatedBuffer.Read(p)
				}
			}
			return 0, io.EOF
		}

		// Handle other errors
		if err != nil {
			return 0, err
		}
		// No error and no translated data yet, continue reading
	}
}

// maxNonStreamingResponseSize is the maximum size limit for non-streaming responses (100MB)
const maxNonStreamingResponseSize = 100 * 1024 * 1024

func (t *translatingResponseBody) readNonStreaming(p []byte) (int, error) {
	// If already processed, only read from translatedBuffer
	if t.eofReached {
		if t.translatedBuffer.Len() > 0 {
			return t.translatedBuffer.Read(p)
		}
		return 0, io.EOF
	}

	// First call: read the complete upstream response
	buf := make([]byte, 4096)
	for {
		// Check size limit before reading more data
		if t.buffer.Len() >= maxNonStreamingResponseSize {
			return 0, fmt.Errorf("response too large: exceeds %d bytes", maxNonStreamingResponseSize)
		}

		n, err := t.reader.Read(buf)
		if n > 0 {
			t.buffer.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	// Translate the complete response
	translated, err := translator.TranslateNonStream(
		t.ctx,
		t.from,
		t.to,
		t.model,
		t.originalRequestBody,
		t.convertedBody,
		t.buffer.Bytes(),
		t.param,
	)
	if err != nil {
		log.Warnf("channel proxy: response translation failed: %v, using original response", err)
		translated = string(t.buffer.Bytes())
	}

	t.eofReached = true // Mark as processed
	t.translatedBuffer.WriteString(translated)
	return t.translatedBuffer.Read(p)
}

func (t *translatingResponseBody) Close() error {
	return t.reader.Close()
}

// MaxNonStreamingResponseSize is the maximum size for non-streaming response body (10MB)
const MaxNonStreamingResponseSize = 10 * 1024 * 1024

// handleNonStreamingResponse reads the complete upstream response, applies transformations,
// and resets resp.Body with correct Content-Length to avoid JSON truncation issues
func handleNonStreamingResponse(resp *http.Response, trace *RequestTrace, transInfo *TranslationInfo, originalModel string) error {
	// Read complete upstream body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxNonStreamingResponseSize))
	resp.Body.Close()
	if err != nil {
		log.Errorf("channel proxy: failed to read non-streaming response: %v", err)
		resp.Body = io.NopCloser(bytes.NewReader([]byte(`{"error":"failed to read upstream response"}`)))
		resp.ContentLength = -1
		resp.Header.Del("Content-Length")
		return nil
	}

	// Extract token usage for logging (before any transformation)
	if trace != nil {
		info, _ := GetProviderInfo(resp.Request.Context())
		extractTokenUsageFromBody(body, trace, &info)
	}

	// Apply format translation if needed
	// Note: Registry keys are [IncomingFormat][OutgoingFormat] based on request direction
	if transInfo != nil && transInfo.NeedsConversion {
		translated, translateErr := translator.TranslateNonStream(
			resp.Request.Context(),
			transInfo.IncomingFormat, // from = client format (registry key)
			transInfo.OutgoingFormat, // to = upstream format (registry key)
			transInfo.Model,
			transInfo.OriginalRequestBody,
			transInfo.ConvertedBody,
			body,
			transInfo.ResponseParam,
		)
		if translateErr != nil {
			log.Warnf("channel proxy: response translation failed: %v, using original response", translateErr)
		} else {
			body = []byte(translated)
			log.Debugf("channel proxy: translated non-streaming response from %s to %s", transInfo.OutgoingFormat, transInfo.IncomingFormat)
		}
	}

	// Apply model name rewriting
	body = RewriteModelInResponseData(body, originalModel)

	// Capture response for logging
	if trace != nil {
		captureResponseForLogging(trace.RequestID, resp.Header, body)
	}

	// Reset body with correct Content-Length
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	// Log completion
	if trace != nil {
		trace.SetResponse(resp.StatusCode)
		if writer := GetLogWriter(); writer != nil {
			writer.UpdateFromTrace(trace)
		}
	}

	return nil
}

// extractTokenUsageFromBody extracts token usage from response body for logging
func extractTokenUsageFromBody(body []byte, trace *RequestTrace, info *ProviderInfo) {
	if len(body) == 0 {
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}

	// Extract usage using existing logic
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		return
	}

	var inputTokens, outputTokens int

	if v, ok := usage["input_tokens"].(float64); ok {
		inputTokens = int(v)
	} else if v, ok := usage["prompt_tokens"].(float64); ok {
		inputTokens = int(v)
	}

	if v, ok := usage["output_tokens"].(float64); ok {
		outputTokens = int(v)
	} else if v, ok := usage["completion_tokens"].(float64); ok {
		outputTokens = int(v)
	}

	if inputTokens > 0 || outputTokens > 0 {
		trace.SetUsage(&inputTokens, &outputTokens, nil, nil)
		log.Debugf("channel proxy: extracted tokens from non-streaming response: input=%d, output=%d", inputTokens, outputTokens)
	}
}

// captureResponseForLogging captures the first part of response for debug logging
func captureResponseForLogging(requestID string, header http.Header, body []byte) {
	if log.GetLevel() < log.DebugLevel {
		return
	}

	maxCapture := 2048
	if len(body) > maxCapture {
		log.Debugf("[%s] response (first %d bytes): %s...", requestID, maxCapture, string(body[:maxCapture]))
	} else {
		log.Debugf("[%s] response: %s", requestID, string(body))
	}
}
