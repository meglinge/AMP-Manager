package amp

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ampmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Constants for free tier request interception
const (
	webSearchQuery             = "webSearch2"
	extractWebPageContentQuery = "extractWebPageContent"
)

// Pre-compiled regex for matching isFreeTierRequest field
var isFreeTierRequestRegex = regexp.MustCompile(`"isFreeTierRequest"\s*:\s*false`)

var (
	apiKeyRepo   = repository.NewAPIKeyRepository()
	settingsRepo = repository.NewAmpSettingsRepository()
)

func APIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing api key",
			})
			return
		}

		keyHash := hashAPIKey(apiKey)

		apiKeyRecord, err := apiKeyRepo.GetByKeyHash(keyHash)
		if err != nil {
			log.Errorf("amp api key auth: db error: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		if apiKeyRecord == nil {
			log.Warnf("amp api key auth: invalid key (prefix: %s...)", maskAPIKey(apiKey))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid api key",
			})
			return
		}

		if apiKeyRecord.RevokedAt != nil {
			log.Warnf("amp api key auth: revoked key used (id: %s)", apiKeyRecord.ID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "api key revoked",
			})
			return
		}

		if apiKeyRecord.ExpiresAt != nil && time.Now().After(*apiKeyRecord.ExpiresAt) {
			log.Warnf("amp api key auth: expired key used (id: %s)", apiKeyRecord.ID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "api key expired",
			})
			return
		}

		settings, err := settingsRepo.GetByUserID(apiKeyRecord.UserID)
		if err != nil {
			log.Errorf("amp api key auth: failed to load settings: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		if settings == nil || !settings.Enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "amp proxy not enabled for this user",
			})
			return
		}

		if settings.UpstreamURL == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "upstream not configured",
			})
			return
		}

		proxyCfg := &ProxyConfig{
			UserID:             apiKeyRecord.UserID,
			APIKeyID:           apiKeyRecord.ID,
			UpstreamURL:        settings.UpstreamURL,
			UpstreamAPIKey:     settings.UpstreamAPIKey,
			ModelMappingsJSON:  settings.ModelMappingsJSON,
			ForceModelMappings: settings.ForceModelMappings,
		}
		ctx := WithProxyConfig(c.Request.Context(), proxyCfg)
		c.Request = c.Request.WithContext(ctx)

		go func() {
			if err := apiKeyRepo.UpdateLastUsed(apiKeyRecord.ID); err != nil {
				log.Warnf("amp api key auth: failed to update last_used_at: %v", err)
			}
		}()

		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	xApiKey := c.GetHeader("X-Api-Key")
	if xApiKey != "" {
		return xApiKey
	}

	return ""
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// DebugInternalAPIMiddleware logs request/response details for webSearch2 and extractWebPageContent
func DebugInternalAPIMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.RawQuery
		if query != webSearchQuery && query != extractWebPageContentQuery {
			c.Next()
			return
		}

		// Log request headers
		log.Infof("=== DEBUG %s REQUEST ===", query)
		log.Infof("URL: %s", c.Request.URL.String())
		log.Infof("Method: %s", c.Request.Method)
		log.Infof("--- Request Headers ---")
		for k, v := range c.Request.Header {
			log.Infof("  %s: %v", k, v)
		}

		// Read and log request body
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				log.Infof("--- Request Body ---")
				log.Infof("%s", string(bodyBytes))
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Create response writer wrapper to capture response
		rw := &responseLogWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = rw

		c.Next()

		// Log response
		log.Infof("--- Response Status ---")
		log.Infof("Status: %d", rw.Status())
		log.Infof("--- Response Headers ---")
		for k, v := range rw.Header() {
			log.Infof("  %s: %v", k, v)
		}
		log.Infof("--- Response Body ---")
		// Decompress gzip if needed
		respBody := rw.body.Bytes()
		if rw.Header().Get("Content-Encoding") == "gzip" && len(respBody) > 0 {
			gr, err := gzip.NewReader(bytes.NewReader(respBody))
			if err == nil {
				decompressed, err := io.ReadAll(gr)
				gr.Close()
				if err == nil {
					respBody = decompressed
				}
			}
		}
		log.Infof("%s", string(respBody))
		log.Infof("=== END DEBUG %s ===", query)
	}
}

type responseLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ForceFreeTierMiddleware forces webSearch2 and extractWebPageContent requests to use free tier
func ForceFreeTierMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.RawQuery
		if query != webSearchQuery && query != extractWebPageContentQuery {
			c.Next()
			return
		}

		if c.Request.Body == nil {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Warnf("amp: could not read request body for %s, proxying as-is: %v", query, err)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		if isFreeTierRequestRegex.Match(bodyBytes) {
			modifiedBody := isFreeTierRequestRegex.ReplaceAll(bodyBytes, []byte(`"isFreeTierRequest":true`))
			c.Request.ContentLength = int64(len(modifiedBody))
			c.Request.Header.Set("Content-Length", strconv.Itoa(len(modifiedBody)))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
			log.Debugf("amp: %s request modified to use free tier", query)
		} else {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()
	}
}

// RequestLoggingMiddleware 请求日志记录中间件
// 在请求开始时创建 RequestTrace，请求结束后写入日志
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取代理配置（由 APIKeyAuthMiddleware 设置）
		cfg := GetProxyConfig(c.Request.Context())
		if cfg == nil {
			c.Next()
			return
		}

		// 创建请求追踪
		requestID := uuid.New().String()
		trace := NewRequestTrace(
			requestID,
			cfg.UserID,
			cfg.APIKeyID,
			c.Request.Method,
			c.Request.URL.Path,
		)

		// 将 trace 存入 context
		ctx := WithRequestTrace(c.Request.Context(), trace)
		c.Request = c.Request.WithContext(ctx)

		// 执行后续处理
		c.Next()

		// 设置响应状态
		trace.SetResponse(c.Writer.Status())

		// 从 model mapping 上下文获取模型信息
		if original := GetOriginalModel(c); original != "" {
			if mapped := GetMappedModel(c); mapped != "" {
				trace.SetModels(original, mapped)
			} else {
				trace.SetModels(original, original)
			}
		}

		// 异步写入日志
		if writer := GetLogWriter(); writer != nil {
			writer.WriteFromTrace(trace)
		}
	}
}
