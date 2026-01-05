package amp

import (
	"net/http"
	"strings"
)

// modelInvocationPaths 模型推理请求路径白名单
// 只有这些路径的 POST 请求会被记录到日志中
var modelInvocationPaths = []string{
	// OpenAI compatible endpoints
	"/v1/chat/completions",
	"/v1/completions",
	"/v1/responses",
	// Anthropic compatible endpoints
	"/v1/messages",
	// Gemini compatible endpoints (prefix match)
	"/v1beta/models/",
	"/v1beta1/models/",
	"/v1beta1/publishers/google/models/",
}

// modelInvocationSuffixes Gemini action 后缀
var modelInvocationSuffixes = []string{
	":generateContent",
	":streamGenerateContent",
}

// IsModelInvocation 判断请求是否是模型推理调用
// 只有 POST 请求且路径匹配白名单才返回 true
func IsModelInvocation(method, path string) bool {
	if method != http.MethodPost {
		return false
	}

	// 处理 /api/provider/:provider/* 前缀的请求
	// 例如 /api/provider/anthropic/v1/messages -> /v1/messages
	normalizedPath := normalizeProviderPath(path)

	// 精确匹配
	for _, p := range modelInvocationPaths {
		if normalizedPath == p {
			return true
		}
	}

	// 前缀匹配 (用于 Gemini 的 /v1beta/models/xxx:generateContent)
	for _, prefix := range modelInvocationPaths {
		if strings.HasPrefix(normalizedPath, prefix) {
			// 检查是否是 Gemini action
			for _, suffix := range modelInvocationSuffixes {
				if strings.HasSuffix(normalizedPath, suffix) {
					return true
				}
			}
			// 或者就是 /v1beta/models/xxx 这种形式的 POST
			if strings.HasPrefix(prefix, "/v1beta") {
				return true
			}
		}
	}

	return false
}

// normalizeProviderPath 标准化 provider 路径
// /api/provider/anthropic/v1/messages -> /v1/messages
// /api/provider/openai/v1/chat/completions -> /v1/chat/completions
func normalizeProviderPath(path string) string {
	if !strings.HasPrefix(path, "/api/provider/") {
		return path
	}

	// /api/provider/:provider/... -> 去掉前缀
	parts := strings.SplitN(path, "/", 5) // ["", "api", "provider", ":provider", "rest..."]
	if len(parts) >= 5 {
		return "/" + parts[4]
	}
	return path
}
