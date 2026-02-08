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
	mappedModel       string
	isStreaming       bool
	streamingDetected bool
}

// NewResponseRewriter creates a new response rewriter for model name substitution
func NewResponseRewriter(w gin.ResponseWriter, originalModel, mappedModel string) *ResponseRewriter {
	return &ResponseRewriter{
		ResponseWriter: w,
		originalModel:  originalModel,
		mappedModel:    mappedModel,
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
		// For streaming responses, rewrite model names in real-time.
		// NOTE: ReverseProxy streams via io.Copy and treats a short write (n != len(data)) as an error.
		// Since rewriting can change the chunk length, we must report that we consumed all of `data` on success.
		rewritten := rw.rewriteStreamChunk(data)
		_, err := rw.ResponseWriter.Write(rewritten)
		if err == nil {
			if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
				flusher.Flush()
			}
			return len(data), nil
		}
		return 0, err
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

// RewriteModelInResponseData rewrites model names in JSON response data using simple string replacement.
// mappedModel is the upstream model name to find, originalModel is what to replace it with.
// If mappedModel is empty, no replacement is done (no mapping was applied).
func RewriteModelInResponseData(data []byte, originalModel, mappedModel string) []byte {
	data = suppressThinkingIfToolUse(data)

	if originalModel == "" || mappedModel == "" || originalModel == mappedModel {
		return data
	}

	return bytes.ReplaceAll(data, []byte(mappedModel), []byte(originalModel))
}

// rewriteStreamChunk rewrites model names in SSE stream chunks
func (rw *ResponseRewriter) rewriteStreamChunk(chunk []byte) []byte {
	chunk = suppressThinkingIfToolUse(chunk)

	if rw.originalModel == "" || rw.mappedModel == "" || rw.originalModel == rw.mappedModel {
		return chunk
	}

	return bytes.ReplaceAll(chunk, []byte(rw.mappedModel), []byte(rw.originalModel))
}
