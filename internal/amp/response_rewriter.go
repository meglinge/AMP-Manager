package amp

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResponseRewriter wraps a gin.ResponseWriter to intercept and modify streaming response data
// For streaming (SSE) responses: rewrites model names in real-time
// For non-streaming responses: passes through directly (model rewriting handled in ModifyResponse)
type ResponseRewriter struct {
	gin.ResponseWriter
	originalModel     string
	isStreaming       bool
	streamingDetected bool
}

// NewResponseRewriter creates a new response rewriter for model name substitution
func NewResponseRewriter(w gin.ResponseWriter, originalModel string) *ResponseRewriter {
	return &ResponseRewriter{
		ResponseWriter: w,
		originalModel:  originalModel,
	}
}

// Write intercepts response writes
// For streaming: rewrites model names in SSE chunks
// For non-streaming: passes through directly without buffering
func (rw *ResponseRewriter) Write(data []byte) (int, error) {
	// Detect streaming on first write
	if !rw.streamingDetected {
		rw.streamingDetected = true
		contentType := rw.Header().Get("Content-Type")
		rw.isStreaming = strings.Contains(contentType, "text/event-stream") ||
			strings.Contains(contentType, "stream")
	}

	if rw.isStreaming {
		// For streaming responses, rewrite model names in real-time
		n, err := rw.ResponseWriter.Write(rw.rewriteStreamChunk(data))
		if err == nil {
			if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		return n, err
	}

	// For non-streaming responses, pass through directly without buffering
	// Model name rewriting is already handled in ModifyResponse via translatingResponseBody
	return rw.ResponseWriter.Write(data)
}

// Flush flushes the underlying ResponseWriter
// For streaming: ensures SSE data is sent immediately
// For non-streaming: data is already written directly, just flush the underlying writer
func (rw *ResponseRewriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// modelFieldPaths lists all JSON paths where model name may appear
var modelFieldPaths = []string{"model", "modelVersion", "response.modelVersion", "message.model"}

// suppressThinkingIfToolUse suppresses thinking blocks when tool_use is detected
// Amp client has rendering issues when it sees both thinking and tool_use blocks
func suppressThinkingIfToolUse(data []byte) []byte {
	// Check if tool_use exists
	if !gjson.GetBytes(data, `content.#(type=="tool_use")`).Exists() {
		return data
	}

	// Filter out thinking and redacted_thinking blocks
	filtered := gjson.GetBytes(data, `content.#(type!="thinking")#`)
	if !filtered.Exists() {
		return data
	}

	// Also filter redacted_thinking
	result := filtered.Value()
	if arr, ok := result.([]interface{}); ok {
		var newArr []interface{}
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "redacted_thinking" {
					continue
				}
			}
			newArr = append(newArr, item)
		}
		result = newArr
	}

	originalCount := gjson.GetBytes(data, "content.#").Int()
	if arr, ok := result.([]interface{}); ok && int64(len(arr)) < originalCount {
		newData, err := sjson.SetBytes(data, "content", result)
		if err != nil {
			log.Warnf("response rewriter: failed to suppress thinking blocks: %v", err)
			return data
		}
		log.Debugf("response rewriter: suppressed %d thinking blocks due to tool usage", originalCount-int64(len(arr)))
		return newData
	}

	return data
}

// rewriteModelInResponse replaces all occurrences of the mapped model with the original model in JSON
func (rw *ResponseRewriter) rewriteModelInResponse(data []byte) []byte {
	return RewriteModelInResponseData(data, rw.originalModel)
}

// RewriteModelInResponseData is a standalone function to rewrite model names in JSON response data
// It can be used outside of ResponseRewriter for direct response manipulation
func RewriteModelInResponseData(data []byte, originalModel string) []byte {
	// Suppress thinking blocks first (if tool_use exists)
	data = suppressThinkingIfToolUse(data)

	if originalModel == "" {
		return data
	}
	for _, path := range modelFieldPaths {
		if gjson.GetBytes(data, path).Exists() {
			data, _ = sjson.SetBytes(data, path, originalModel)
		}
	}
	return data
}

// rewriteStreamChunk rewrites model names in SSE stream chunks
func (rw *ResponseRewriter) rewriteStreamChunk(chunk []byte) []byte {
	// SSE format: "data: {json}\n\n"
	lines := bytes.Split(chunk, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("data: ")) {
			jsonData := bytes.TrimPrefix(line, []byte("data: "))
			if len(jsonData) > 0 && jsonData[0] == '{' {
				// Suppress thinking blocks first
				jsonData = suppressThinkingIfToolUse(jsonData)
				// Then rewrite model
				rewritten := rw.rewriteModelInResponse(jsonData)
				lines[i] = append([]byte("data: "), rewritten...)
			}
		}
	}

	return bytes.Join(lines, []byte("\n"))
}
