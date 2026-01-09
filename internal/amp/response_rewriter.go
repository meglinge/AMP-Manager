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

// ResponseRewriter wraps a gin.ResponseWriter to intercept and modify the response body
// It's used to rewrite model names in responses when model mapping is used
// This is the key mechanism: by returning the original model name (e.g., claude-opus),
// Amp client will use that model's context length (168k/200k) instead of the mapped model's (968k)
type ResponseRewriter struct {
	gin.ResponseWriter
	body          *bytes.Buffer
	originalModel string
	isStreaming   bool
}

// NewResponseRewriter creates a new response rewriter for model name substitution
func NewResponseRewriter(w gin.ResponseWriter, originalModel string) *ResponseRewriter {
	return &ResponseRewriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		originalModel:  originalModel,
	}
}

// Write intercepts response writes and buffers them for model name replacement
func (rw *ResponseRewriter) Write(data []byte) (int, error) {
	// Detect streaming on first write
	if rw.body.Len() == 0 && !rw.isStreaming {
		contentType := rw.Header().Get("Content-Type")
		rw.isStreaming = strings.Contains(contentType, "text/event-stream") ||
			strings.Contains(contentType, "stream")
	}

	if rw.isStreaming {
		n, err := rw.ResponseWriter.Write(rw.rewriteStreamChunk(data))
		if err == nil {
			if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		return n, err
	}
	return rw.body.Write(data)
}

// Flush writes the buffered response with model names rewritten
func (rw *ResponseRewriter) Flush() {
	if rw.isStreaming {
		if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}
	if rw.body.Len() > 0 {
		if _, err := rw.ResponseWriter.Write(rw.rewriteModelInResponse(rw.body.Bytes())); err != nil {
			log.Warnf("amp response rewriter: failed to write rewritten response: %v", err)
		}
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
	// Suppress thinking blocks first (if tool_use exists)
	data = suppressThinkingIfToolUse(data)

	if rw.originalModel == "" {
		return data
	}
	for _, path := range modelFieldPaths {
		if gjson.GetBytes(data, path).Exists() {
			data, _ = sjson.SetBytes(data, path, rw.originalModel)
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
