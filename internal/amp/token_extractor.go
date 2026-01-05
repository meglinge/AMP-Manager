package amp

import (
	"bytes"
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

// ExtractTokenUsage 从非流式响应体中提取 token 使用量（使用指定 provider）
func ExtractTokenUsage(body []byte, info ProviderInfo) *TokenUsage {
	parser := NewUsageParser(info)
	usage, ok := parser.ParseResponse(body)
	if !ok {
		log.Debugf("token extractor: failed to parse response for provider %s", info.Provider)
		return nil
	}
	return usage
}

// SSETokenExtractor 从 SSE 流中提取 token 使用量
// 实现 io.ReadCloser 接口，边转发边解析
type SSETokenExtractor struct {
	reader       io.ReadCloser
	trace        *RequestTrace
	parser       UsageParser
	buffer       bytes.Buffer
	mu           sync.Mutex
	extracted    bool
	currentEvent string // 当前 SSE event 名称
}

// NewSSETokenExtractor 创建 SSE token 提取器
func NewSSETokenExtractor(reader io.ReadCloser, trace *RequestTrace, info ProviderInfo) *SSETokenExtractor {
	return &SSETokenExtractor{
		reader: reader,
		trace:  trace,
		parser: NewUsageParser(info),
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

		// 处理 SSE event: 行
		if strings.HasPrefix(line, "event: ") {
			e.currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				e.currentEvent = ""
				continue
			}
			e.parseSSEData(data)
			e.currentEvent = "" // 重置 event
		}
	}
}

// parseSSEData 解析单个 SSE 数据事件
func (e *SSETokenExtractor) parseSSEData(data string) {
	usage, final, ok := e.parser.ConsumeSSE(e.currentEvent, []byte(data))
	if !ok {
		return
	}

	if usage != nil && e.trace != nil {
		e.trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
	}

	if final {
		e.extracted = true
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
func WrapResponseBodyForTokenExtraction(body io.ReadCloser, isStreaming bool, trace *RequestTrace, info ProviderInfo) io.ReadCloser {
	if trace == nil {
		return body
	}

	if isStreaming {
		trace.SetStreaming(true)
		return NewSSETokenExtractor(body, trace, info)
	}

	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		log.Debugf("token extractor: failed to read body: %v", err)
		return io.NopCloser(bytes.NewReader(data))
	}

	usage := ExtractTokenUsage(data, info)
	if usage != nil {
		trace.SetUsage(usage.InputTokens, usage.OutputTokens, usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
		log.Debugf("token extractor: non-streaming [%s] - input=%v, output=%v, cache_read=%v, cache_creation=%v",
			info.Provider,
			ptrToInt(usage.InputTokens), ptrToInt(usage.OutputTokens),
			ptrToInt(usage.CacheReadInputTokens), ptrToInt(usage.CacheCreationInputTokens))
	}

	return io.NopCloser(bytes.NewReader(data))
}
