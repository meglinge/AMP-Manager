package amp

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// ErrorResponse 标准错误响应格式（OpenAI 兼容）
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// BuildErrorResponseBody 构建 OpenAI 兼容的 JSON 错误响应体
// 如果 errText 已经是有效的 JSON，则直接返回以保留上游错误格式
func BuildErrorResponseBody(status int, errText string) []byte {
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	if strings.TrimSpace(errText) == "" {
		errText = http.StatusText(status)
	}

	// 如果已经是有效 JSON，直接返回
	trimmed := strings.TrimSpace(errText)
	if trimmed != "" && json.Valid([]byte(trimmed)) {
		return []byte(trimmed)
	}

	// 根据状态码确定错误类型
	errType := "invalid_request_error"
	var code string
	switch status {
	case http.StatusUnauthorized:
		errType = "authentication_error"
		code = "invalid_api_key"
	case http.StatusForbidden:
		errType = "permission_error"
		code = "insufficient_quota"
	case http.StatusTooManyRequests:
		errType = "rate_limit_error"
		code = "rate_limit_exceeded"
	case http.StatusNotFound:
		errType = "invalid_request_error"
		code = "model_not_found"
	case http.StatusBadRequest:
		errType = "invalid_request_error"
		code = "bad_request"
	case http.StatusServiceUnavailable:
		errType = "server_error"
		code = "service_unavailable"
	case http.StatusBadGateway:
		errType = "server_error"
		code = "upstream_error"
	case http.StatusGatewayTimeout:
		errType = "server_error"
		code = "timeout"
	default:
		if status >= http.StatusInternalServerError {
			errType = "server_error"
			code = "internal_server_error"
		}
	}

	payload, err := json.Marshal(ErrorResponse{
		Error: ErrorDetail{
			Message: errText,
			Type:    errType,
			Code:    code,
		},
	})
	if err != nil {
		return []byte(`{"error":{"message":"` + escapeJSON(errText) + `","type":"server_error","code":"internal_server_error"}}`)
	}
	return payload
}

// WriteErrorResponse 写入标准化错误响应
func WriteErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(BuildErrorResponseBody(status, message))
}

// BuildUpstreamErrorResponse 从上游错误构建错误响应
// 尝试解析上游错误体，如果是有效 JSON 则透传，否则包装为标准格式
func BuildUpstreamErrorResponse(status int, upstreamBody []byte) []byte {
	if len(upstreamBody) == 0 {
		return BuildErrorResponseBody(status, "")
	}

	// 尝试解析为 JSON
	trimmed := strings.TrimSpace(string(upstreamBody))
	if json.Valid([]byte(trimmed)) {
		return []byte(trimmed)
	}

	// 不是有效 JSON，包装为标准格式
	return BuildErrorResponseBody(status, trimmed)
}

// escapeJSON 简单的 JSON 字符串转义
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// MapHTTPStatusToErrorType 将 HTTP 状态码映射到错误类型
func MapHTTPStatusToErrorType(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusNotFound:
		return "invalid_request_error"
	case http.StatusBadRequest:
		return "invalid_request_error"
	default:
		if status >= http.StatusInternalServerError {
			return "server_error"
		}
		return "invalid_request_error"
	}
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusGatewayTimeout,
		http.StatusInternalServerError:
		return true
	default:
		return false
	}
}

// sensitivePatterns 敏感信息正则模式（编译一次复用）
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)key=[^&\s]+`),
	regexp.MustCompile(`(?i)api_key=[^&\s]+`),
	regexp.MustCompile(`(?i)token=[^&\s]+`),
	regexp.MustCompile(`(?i)Bearer\s+[^\s]+`),
	regexp.MustCompile(`(?i)x-api-key:\s*[^\s]+`),
	regexp.MustCompile(`(?i)authorization:\s*[^\s]+`),
	regexp.MustCompile(`sk-[a-zA-Z0-9-_]+`),
	regexp.MustCompile(`(?i)password=[^&\s]+`),
	regexp.MustCompile(`(?i)secret=[^&\s]+`),
}

// SanitizeError 清理错误消息中的敏感信息
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return SanitizeErrorMessage(err.Error())
}

// SanitizeErrorMessage 清理错误消息字符串中的敏感信息
func SanitizeErrorMessage(msg string) string {
	for _, pattern := range sensitivePatterns {
		msg = pattern.ReplaceAllString(msg, "[REDACTED]")
	}
	return msg
}

// NewStandardError 创建标准化错误响应对象（用于 Gin 的 AbortWithStatusJSON）
func NewStandardError(status int, message string) ErrorResponse {
	errType := MapHTTPStatusToErrorType(status)
	var code string
	switch status {
	case http.StatusUnauthorized:
		code = "invalid_api_key"
	case http.StatusForbidden:
		code = "insufficient_quota"
	case http.StatusTooManyRequests:
		code = "rate_limit_exceeded"
	case http.StatusNotFound:
		code = "model_not_found"
	case http.StatusBadRequest:
		code = "bad_request"
	case http.StatusServiceUnavailable:
		code = "service_unavailable"
	case http.StatusBadGateway:
		code = "upstream_error"
	case http.StatusGatewayTimeout:
		code = "timeout"
	default:
		if status >= http.StatusInternalServerError {
			code = "internal_server_error"
		}
	}
	return ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	}
}
