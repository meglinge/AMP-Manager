package amp

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type proxyConfigKey struct{}

type ProxyConfig struct {
	UserID             string
	APIKeyID           string
	UpstreamURL        string
	UpstreamAPIKey     string
	ModelMappingsJSON  string
	ForceModelMappings bool
}

func WithProxyConfig(ctx context.Context, cfg *ProxyConfig) context.Context {
	return context.WithValue(ctx, proxyConfigKey{}, cfg)
}

func GetProxyConfig(ctx context.Context) *ProxyConfig {
	if val := ctx.Value(proxyConfigKey{}); val != nil {
		if cfg, ok := val.(*ProxyConfig); ok {
			return cfg
		}
	}
	return nil
}

// readCloser wraps a reader and forwards Close to a separate closer
type readCloser struct {
	r io.Reader
	c io.Closer
}

func (rc *readCloser) Read(p []byte) (int, error) { return rc.r.Read(p) }
func (rc *readCloser) Close() error               { return rc.c.Close() }

// 全局 RetryTransport 实例，支持动态配置更新
var globalRetryTransport *RetryTransport

// GetRetryTransport 获取全局 RetryTransport 实例
func GetRetryTransport() *RetryTransport {
	return globalRetryTransport
}

// NewStreamingTransport 创建针对 AI 流式请求优化的 HTTP Transport
// 解决 60 秒左右连接中断的问题
func NewStreamingTransport() *http.Transport {
	return &http.Transport{
		// 连接设置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second, // TCP Keep-Alive 间隔
		}).DialContext,

		// TLS 设置
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		TLSHandshakeTimeout: 15 * time.Second,

		// 连接池设置
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     0, // 无限制

		// 关键：延长空闲连接超时，避免连接过早关闭
		IdleConnTimeout: 120 * time.Second,

		// 关键：不设置 ResponseHeaderTimeout，允许 AI 模型长时间思考
		// ResponseHeaderTimeout: 0 表示无超时
		ResponseHeaderTimeout: 0,

		// 关键：不设置 ExpectContinueTimeout，避免 100-continue 超时
		ExpectContinueTimeout: 0,

		// 禁用压缩以支持流式响应（gzip 缓冲会导致 SSE 流看起来无数据，触发 idle timeout）
		DisableCompression: true,

		// 保持连接复用
		DisableKeepAlives: false,

		// 强制尝试 HTTP/2
		ForceAttemptHTTP2: true,
	}
}

// InitRetryTransportConfig 初始化重试配置（从数据库加载）
// 需要在 CreateDynamicReverseProxy 之后调用
func InitRetryTransportConfig(configJSON string) {
	if globalRetryTransport == nil || configJSON == "" {
		return
	}

	var cfg struct {
		Enabled           bool  `json:"enabled"`
		MaxAttempts       int   `json:"maxAttempts"`
		GateTimeoutMs     int64 `json:"gateTimeoutMs"`
		MaxBodyBytes      int64 `json:"maxBodyBytes"`
		BackoffBaseMs     int64 `json:"backoffBaseMs"`
		BackoffMaxMs      int64 `json:"backoffMaxMs"`
		RetryOn429        bool  `json:"retryOn429"`
		RetryOn5xx        bool  `json:"retryOn5xx"`
		RespectRetryAfter bool  `json:"respectRetryAfter"`
		RetryOnEmptyBody  bool  `json:"retryOnEmptyBody"`
	}

	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		log.Warnf("retry: 解析配置失败，使用默认配置: %v", err)
		return
	}

	globalRetryTransport.UpdateConfig(&RetryConfig{
		Enabled:           cfg.Enabled,
		MaxAttempts:       cfg.MaxAttempts,
		GateTimeout:       time.Duration(cfg.GateTimeoutMs) * time.Millisecond,
		MaxBodyBytes:      cfg.MaxBodyBytes,
		BackoffBase:       time.Duration(cfg.BackoffBaseMs) * time.Millisecond,
		BackoffMax:        time.Duration(cfg.BackoffMaxMs) * time.Millisecond,
		RetryOn429:        cfg.RetryOn429,
		RetryOn5xx:        cfg.RetryOn5xx,
		RespectRetryAfter: cfg.RespectRetryAfter,
		RetryOnEmptyBody:  cfg.RetryOnEmptyBody,
	})

	log.WithFields(log.Fields{
		"enabled":     cfg.Enabled,
		"maxAttempts": cfg.MaxAttempts,
	}).Info("retry: 已加载保存的重试配置")
}

// CreateDynamicReverseProxy creates a reverse proxy for ampcode.com upstream
// Following CLIProxyAPI pattern: does NOT filter Anthropic-Beta headers
// Users going through ampcode.com are paying for the service and should get all features
func CreateDynamicReverseProxy() *httputil.ReverseProxy {
	// 使用针对 AI 流式请求优化的 Transport，解决 60 秒超时问题
	streamingTransport := NewStreamingTransport()

	// 初始化全局 RetryTransport
	globalRetryTransport = NewRetryTransport(streamingTransport, DefaultRetryConfig())

	proxy := &httputil.ReverseProxy{
		// 使用带重试功能的 Transport
		Transport: globalRetryTransport,
		// FlushInterval 设为 -1 确保流式响应（SSE）立即刷新到客户端
		// 避免缓冲导致 "request ended without sending any chunks" 错误
		FlushInterval: -1,
		Director: func(req *http.Request) {
			cfg := GetProxyConfig(req.Context())
			if cfg == nil {
				log.Warn("amp proxy: no config in context")
				return
			}

			parsed, err := url.Parse(cfg.UpstreamURL)
			if err != nil {
				log.Errorf("amp proxy: invalid upstream url: %v", err)
				return
			}

			req.URL.Scheme = parsed.Scheme
			req.URL.Host = parsed.Host
			req.Host = parsed.Host

			log.Debugf("amp proxy: %s %s -> %s%s", req.Method, req.URL.Path, req.URL.Host, req.URL.Path)

			// Only create trace for model invocation requests
			if IsModelInvocation(req.Method, req.URL.Path) {
				trace := NewRequestTrace(
					uuid.New().String(),
					cfg.UserID,
					cfg.APIKeyID,
					req.Method,
					req.URL.Path,
				)
				// Set provider info (amp upstream defaults to Anthropic)
				trace.SetChannel("", string(ProviderAnthropic), cfg.UpstreamURL)
				// Get model info from context if available
				if modelInfo := GetModelInfo(req.Context()); modelInfo != nil {
					trace.SetModels(modelInfo.OriginalModel, modelInfo.MappedModel)
				}
				// Store trace in context
				ctx := WithRequestTrace(req.Context(), trace)
				// Store ProviderInfo in context for token extraction
				ctx = WithProviderInfo(ctx, ProviderInfo{Provider: ProviderAnthropic})
				*req = *req.WithContext(ctx)

				// Write pending record to database immediately
				if writer := GetLogWriter(); writer != nil {
					writer.WritePendingFromTrace(trace)
				}

				log.Infof("amp proxy: model invocation %s %s -> %s", req.Method, req.URL.Path, req.URL.Host)
			}

			// Remove client auth headers and hop-by-hop headers
			req.Header.Del("Authorization")
			req.Header.Del("X-Api-Key")
			req.Header.Del("Transfer-Encoding")
			req.TransferEncoding = nil

			// NOTE: Following CLIProxyAPI pattern - we do NOT filter Anthropic-Beta headers here
			// Users going through ampcode.com proxy are paying for the service and should get all features
			// including 1M context window (context-1m-2025-08-07)

			// Set upstream API key
			if cfg.UpstreamAPIKey != "" {
				req.Header.Set("X-Api-Key", cfg.UpstreamAPIKey)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.UpstreamAPIKey))
			}
		},
		ModifyResponse: modifyResponse,
		ErrorHandler:   errorHandler,
	}
	return proxy
}

// modifyResponse handles gzip decompression and token extraction
// Does NOT attempt to rewrite context_length as that's not where Amp gets the value
func modifyResponse(resp *http.Response) error {
	trace := GetRequestTrace(resp.Request.Context())

	// 获取 provider 信息（amp upstream 默认为 Anthropic）
	info, ok := GetProviderInfo(resp.Request.Context())
	if !ok {
		info = ProviderInfo{Provider: ProviderAnthropic}
	}

	// 处理流式响应 - 包装 body 以提取 token
	if isStreamingResponse(resp) {
		if trace != nil {
			trace.SetStreaming(true)
			resp.Body = NewSSETokenExtractor(resp.Body, trace, info)
			// Wrap for logging on close
			resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
		}
		return nil
	}

	// Handle non-2xx responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if trace != nil {
			trace.SetError("upstream_error")
			resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
		}
		return nil
	}

	// 处理非流式响应：读取完整 body 并提取 token
	originalBody := resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")

	// 读取完整响应体（限制 10MB + 1 字节用于检测截断）
	const maxResponseSize = 10 * 1024 * 1024
	compressedData, err := io.ReadAll(io.LimitReader(originalBody, maxResponseSize+1))
	_ = originalBody.Close()
	if err != nil {
		log.Warnf("amp proxy: failed to read response body: %v", err)
		resp.Body = io.NopCloser(bytes.NewReader(compressedData))
		if trace != nil {
			resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
		}
		return nil
	}

	// 检测响应是否被截断
	if len(compressedData) > maxResponseSize {
		log.Warnf("amp proxy: response too large (%d bytes), skipping token extraction", len(compressedData))
		// 截断到限制大小并透传
		resp.Body = io.NopCloser(bytes.NewReader(compressedData[:maxResponseSize]))
		resp.ContentLength = int64(maxResponseSize)
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		if trace != nil {
			resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
		}
		return nil
	}

	var bodyData []byte
	const maxDecompressedSize = 50 * 1024 * 1024 // 50MB 解压限制，防止 zip 炸弹

	// 根据 Content-Encoding 解压
	switch strings.ToLower(contentEncoding) {
	case "gzip":
		gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
		if err != nil {
			log.Warnf("amp proxy: failed to create gzip reader: %v", err)
			bodyData = compressedData
		} else {
			decompressed, err := io.ReadAll(io.LimitReader(gzipReader, maxDecompressedSize+1))
			_ = gzipReader.Close()
			if err != nil {
				log.Warnf("amp proxy: failed to decompress gzip: %v", err)
				bodyData = compressedData
			} else if len(decompressed) > maxDecompressedSize {
				log.Warnf("amp proxy: decompressed response too large (%d bytes), skipping token extraction", len(decompressed))
				// 解压后太大，使用原始压缩数据
				bodyData = compressedData
			} else {
				bodyData = decompressed
				resp.Header.Del("Content-Encoding")
				log.Debugf("amp proxy: decompressed gzip response (%d -> %d bytes)", len(compressedData), len(bodyData))
			}
		}
	case "":
		// 无压缩，但可能是隐式 gzip（检查 magic bytes）
		if len(compressedData) >= 2 && compressedData[0] == 0x1f && compressedData[1] == 0x8b {
			gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
			if err == nil {
				decompressed, err := io.ReadAll(io.LimitReader(gzipReader, maxDecompressedSize+1))
				_ = gzipReader.Close()
				if err == nil {
					if len(decompressed) > maxDecompressedSize {
						log.Warnf("amp proxy: implicit gzip decompressed too large (%d bytes), skipping token extraction", len(decompressed))
						bodyData = compressedData
					} else {
						bodyData = decompressed
						log.Debugf("amp proxy: decompressed implicit gzip response (%d -> %d bytes)", len(compressedData), len(bodyData))
					}
				} else {
					bodyData = compressedData
				}
			} else {
				bodyData = compressedData
			}
		} else {
			bodyData = compressedData
		}
	default:
		// 不支持的压缩格式（如 br），直接使用原始数据
		log.Debugf("amp proxy: unsupported Content-Encoding: %s, skipping token extraction", contentEncoding)
		bodyData = compressedData
	}

	// 提取 token 使用量
	if trace != nil && len(bodyData) > 0 {
		if usage := ExtractTokenUsage(bodyData, info); usage != nil {
			trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
			log.Debugf("amp proxy: extracted non-streaming token usage - input=%v, output=%v",
				ptrToInt(usage.InputTokens), ptrToInt(usage.OutputTokens))
		}
	}

	// 设置响应体（返回解压后的数据）
	resp.Body = io.NopCloser(bytes.NewReader(bodyData))
	resp.ContentLength = int64(len(bodyData))
	resp.Header.Del("Content-Length")
	resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))

	// Wrap for logging on close
	if trace != nil {
		resp.Body = NewLoggingBodyWrapper(resp.Body, trace, resp.StatusCode)
	}

	return nil
}

func isStreamingResponse(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream")
}

func errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	log.Errorf("amp upstream proxy error for %s %s: %v", req.Method, req.URL.Path, err)
	// Update error log (pending record was already written in Director)
	if trace := GetRequestTrace(req.Context()); trace != nil {
		trace.SetError("upstream_request_failed")
		trace.SetResponse(http.StatusBadGateway)
		if writer := GetLogWriter(); writer != nil {
			writer.UpdateFromTrace(trace)
		}
	}
	// 使用清理后的错误消息，防止泄露敏感信息
	safeMsg := SanitizeError(err)
	WriteErrorResponse(rw, http.StatusBadGateway, "Failed to reach Amp upstream: "+safeMsg)
}

func ProxyHandler(proxy *httputil.ReverseProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetProxyConfig(c.Request.Context()) == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing proxy configuration"})
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// filterAntropicBetaHeader removes the context-1m-2025-08-07 beta feature
// This should ONLY be called for local/channel handling paths, NOT for ampcode.com proxy
func filterAntropicBetaHeader(req *http.Request) {
	betaHeader := req.Header.Get("Anthropic-Beta")
	if betaHeader == "" {
		return
	}

	filtered := filterBetaFeatures(betaHeader, "context-1m-2025-08-07")
	if filtered != "" {
		req.Header.Set("Anthropic-Beta", filtered)
		log.Debugf("channel proxy: filtered Anthropic-Beta header: %s -> %s", betaHeader, filtered)
	} else {
		req.Header.Del("Anthropic-Beta")
		log.Debugf("channel proxy: removed Anthropic-Beta header (was: %s)", betaHeader)
	}
}

// filterBetaFeatures removes a specific feature from comma-separated beta features list
func filterBetaFeatures(header, featureToRemove string) string {
	features := strings.Split(header, ",")
	filtered := make([]string, 0, len(features))

	for _, feature := range features {
		trimmed := strings.TrimSpace(feature)
		if trimmed != "" && trimmed != featureToRemove {
			filtered = append(filtered, trimmed)
		}
	}

	return strings.Join(filtered, ",")
}
