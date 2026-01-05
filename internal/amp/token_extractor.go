package amp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// TokenUsage 从 API 响应中提取的 token 使用量
type TokenUsage struct {
	InputTokens              *int `json:"input_tokens,omitempty"`
	OutputTokens             *int `json:"output_tokens,omitempty"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens,omitempty"`
}

// anthropicResponse 用于解析 Anthropic API 响应
type anthropicResponse struct {
	Usage *TokenUsage `json:"usage,omitempty"`
}

// anthropicStreamEvent 用于解析 Anthropic 流式事件
type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Message *struct {
		Usage *TokenUsage `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	Usage *TokenUsage `json:"usage,omitempty"`
}

// ExtractTokenUsage 从非流式响应体中提取 token 使用量
func ExtractTokenUsage(body []byte) *TokenUsage {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Debugf("token extractor: failed to parse response: %v", err)
		return nil
	}
	return resp.Usage
}

// SSETokenExtractor 从 SSE 流中提取 token 使用量
// 实现 io.ReadCloser 接口，边转发边解析
type SSETokenExtractor struct {
	reader    io.ReadCloser
	trace     *RequestTrace
	buffer    bytes.Buffer
	mu        sync.Mutex
	extracted bool
}

// NewSSETokenExtractor 创建 SSE token 提取器
func NewSSETokenExtractor(reader io.ReadCloser, trace *RequestTrace) *SSETokenExtractor {
	return &SSETokenExtractor{
		reader: reader,
		trace:  trace,
	}
}

// Read 实现 io.Reader 接口
func (e *SSETokenExtractor) Read(p []byte) (int, error) {
	n, err := e.reader.Read(p)
	if n > 0 {
		e.processChunk(p[:n])
	}
	return n, err
}

// Close 实现 io.Closer 接口
func (e *SSETokenExtractor) Close() error {
	return e.reader.Close()
}

// processChunk 处理 SSE 数据块
func (e *SSETokenExtractor) processChunk(chunk []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.buffer.Write(chunk)

	for {
		line, err := e.buffer.ReadString('\n')
		if err != nil {
			if len(line) > 0 {
				var remaining bytes.Buffer
				remaining.WriteString(line)
				remaining.Write(e.buffer.Bytes())
				e.buffer = remaining
			}
			break
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}
			e.parseSSEData(data)
		}
	}
}

// parseSSEData 解析单个 SSE 数据事件
func (e *SSETokenExtractor) parseSSEData(data string) {
	var event anthropicStreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return
	}

	switch event.Type {
	case "message_start":
		if event.Message != nil && event.Message.Usage != nil {
			usage := event.Message.Usage
			e.trace.SetUsage(usage.InputTokens, nil, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
			log.Debugf("token extractor: message_start - input=%v, cache_read=%v, cache_creation=%v",
				ptrToInt(usage.InputTokens), ptrToInt(usage.CacheReadInputTokens), ptrToInt(usage.CacheCreationInputTokens))
		}
	case "message_delta":
		if event.Usage != nil && event.Usage.OutputTokens != nil {
			e.trace.UpdateOutputTokens(*event.Usage.OutputTokens)
			log.Debugf("token extractor: message_delta - output=%d", *event.Usage.OutputTokens)
		}
	}
}

// ptrToInt 辅助函数，将指针转为值（用于日志）
func ptrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// WrapResponseBodyForTokenExtraction 包装响应体以提取 token
// 对于非流式响应，读取并解析后放回
// 对于流式响应，返回包装的 reader
func WrapResponseBodyForTokenExtraction(body io.ReadCloser, isStreaming bool, trace *RequestTrace) io.ReadCloser {
	if trace == nil {
		return body
	}

	if isStreaming {
		trace.SetStreaming(true)
		return NewSSETokenExtractor(body, trace)
	}

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		log.Debugf("token extractor: failed to read body: %v", err)
		return io.NopCloser(bytes.NewReader(data))
	}

	usage := ExtractTokenUsage(data)
	if usage != nil {
		trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
		log.Debugf("token extractor: non-streaming - input=%v, output=%v, cache_read=%v, cache_creation=%v",
			ptrToInt(usage.InputTokens), ptrToInt(usage.OutputTokens),
			ptrToInt(usage.CacheReadInputTokens), ptrToInt(usage.CacheCreationInputTokens))
	}

	return io.NopCloser(bytes.NewReader(data))
}

// StreamingSSEReader 用于流式响应的 SSE 解析 reader
type StreamingSSEReader struct {
	source  io.ReadCloser
	scanner *bufio.Scanner
	trace   *RequestTrace
	buffer  bytes.Buffer
	done    bool
}

// NewStreamingSSEReader 创建流式 SSE reader
func NewStreamingSSEReader(source io.ReadCloser, trace *RequestTrace) *StreamingSSEReader {
	return &StreamingSSEReader{
		source:  source,
		scanner: bufio.NewScanner(source),
		trace:   trace,
	}
}
