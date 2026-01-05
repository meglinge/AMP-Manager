package amp

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// CreateDynamicReverseProxy creates a reverse proxy for ampcode.com upstream
// Following CLIProxyAPI pattern: does NOT filter Anthropic-Beta headers
// Users going through ampcode.com are paying for the service and should get all features
func CreateDynamicReverseProxy() *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		// FlushInterval 确保流式响应（SSE）立即刷新到客户端
		// 防止 Oracle 等子代理因响应缓冲导致 terminated
		FlushInterval: 10 * time.Millisecond,
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

			log.Infof("amp proxy: %s %s -> %s%s", req.Method, req.URL.Path, req.URL.Host, req.URL.Path)

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

	// 处理流式响应 - 包装 body 以提取 token
	if isStreamingResponse(resp) {
		if trace != nil {
			trace.SetStreaming(true)
			resp.Body = NewSSETokenExtractor(resp.Body, trace)
		}
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	// Skip if already marked as gzip
	if resp.Header.Get("Content-Encoding") != "" {
		return nil
	}

	// Save reference to original upstream body
	originalBody := resp.Body

	// Peek at first 2 bytes to detect gzip magic bytes
	header := make([]byte, 2)
	n, _ := io.ReadFull(originalBody, header)

	// Check for gzip magic bytes (0x1f 0x8b)
	if n >= 2 && header[0] == 0x1f && header[1] == 0x8b {
		// Read the rest of the body
		rest, err := io.ReadAll(originalBody)
		if err != nil {
			resp.Body = &readCloser{
				r: io.MultiReader(bytes.NewReader(header[:n]), originalBody),
				c: originalBody,
			}
			return nil
		}

		// Reconstruct complete gzipped data
		gzippedData := append(header[:n], rest...)

		// Decompress
		gzipReader, err := gzip.NewReader(bytes.NewReader(gzippedData))
		if err != nil {
			_ = originalBody.Close()
			resp.Body = io.NopCloser(bytes.NewReader(gzippedData))
			return nil
		}

		decompressed, err := io.ReadAll(gzipReader)
		_ = gzipReader.Close()
		if err != nil {
			_ = originalBody.Close()
			resp.Body = io.NopCloser(bytes.NewReader(gzippedData))
			return nil
		}

		_ = originalBody.Close()

		// 提取 token 使用量
		if trace != nil {
			if usage := ExtractTokenUsage(decompressed); usage != nil {
				trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
			}
		}

		resp.Body = io.NopCloser(bytes.NewReader(decompressed))
		resp.ContentLength = int64(len(decompressed))
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))

		log.Debugf("amp proxy: decompressed gzip response (%d -> %d bytes)", len(gzippedData), len(decompressed))
	} else {
		// Not gzip - 读取完整响应以提取 token
		rest, err := io.ReadAll(originalBody)
		_ = originalBody.Close()
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(header[:n]))
			return nil
		}

		fullBody := append(header[:n], rest...)

		// 提取 token 使用量
		if trace != nil {
			if usage := ExtractTokenUsage(fullBody); usage != nil {
				trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
			}
		}

		resp.Body = io.NopCloser(bytes.NewReader(fullBody))
	}

	return nil
}

func isStreamingResponse(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream")
}

func errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	log.Errorf("amp upstream proxy error for %s %s: %v", req.Method, req.URL.Path, err)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusBadGateway)
	_, _ = rw.Write([]byte(`{"error":"amp_upstream_proxy_error","message":"Failed to reach Amp upstream"}`))
}

func ProxyHandler(proxy *httputil.ReverseProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
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
