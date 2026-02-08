package amp

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	pseudoNonStreamKeepAliveInterval = 5 * time.Second
	pseudoNonStreamKeepAliveMsg      = ": keep-alive\n\n"
	pseudoNonStreamMaxRetries        = 2
)

// PseudoNonStreamBodyWrapper wraps an upstream io.ReadCloser to buffer all SSE data
// before returning it to the ReverseProxy. During buffering, it sends keep-alive
// heartbeats to the client via the ResponseWriter stored in context.
type PseudoNonStreamBodyWrapper struct {
	upstream      io.ReadCloser
	rw            http.ResponseWriter
	modelName     string
	auditKeywords []string

	// Retry support
	retryFunc  func() (io.ReadCloser, error)
	maxRetries int

	once   sync.Once
	buf    bytes.Buffer
	reader *bytes.Reader
	bufErr error
}

// NewPseudoNonStreamBodyWrapper creates a new wrapper that will buffer the entire
// upstream response before allowing reads. modelName is the mapped model name
// used for model-specific audit rules.
func NewPseudoNonStreamBodyWrapper(upstream io.ReadCloser, rw http.ResponseWriter, modelName string, opts ...PseudoNonStreamOption) *PseudoNonStreamBodyWrapper {
	w := &PseudoNonStreamBodyWrapper{
		upstream:   upstream,
		rw:         rw,
		modelName:  modelName,
		maxRetries: pseudoNonStreamMaxRetries,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

type PseudoNonStreamOption func(*PseudoNonStreamBodyWrapper)

func WithAuditKeywordsOption(keywords []string) PseudoNonStreamOption {
	return func(w *PseudoNonStreamBodyWrapper) {
		w.auditKeywords = keywords
	}
}

func WithRetryFunc(f func() (io.ReadCloser, error)) PseudoNonStreamOption {
	return func(w *PseudoNonStreamBodyWrapper) {
		w.retryFunc = f
	}
}

// Read implements io.Reader. On first call, it reads the entire upstream body into
// a buffer while sending keep-alive heartbeats, then serves subsequent reads from
// the buffer.
func (w *PseudoNonStreamBodyWrapper) Read(p []byte) (int, error) {
	w.once.Do(func() {
		w.bufErr = w.bufferUpstream()
	})

	if w.bufErr != nil {
		return 0, w.bufErr
	}

	return w.reader.Read(p)
}

// Close implements io.Closer.
func (w *PseudoNonStreamBodyWrapper) Close() error {
	return w.upstream.Close()
}

// bufferUpstream reads all data from the upstream body into the internal buffer,
// sending keep-alive heartbeats to the client while buffering. Supports retry
// on audit failure.
func (w *PseudoNonStreamBodyWrapper) bufferUpstream() error {
	ticker := time.NewTicker(pseudoNonStreamKeepAliveInterval)
	defer ticker.Stop()

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			log.Infof("pseudo-non-stream: retry attempt %d/%d", attempt, w.maxRetries)
		}

		// Buffer current upstream
		w.buf.Reset()
		readErr := w.bufferFromReader(w.upstream, ticker)
		if readErr != nil {
			log.Errorf("pseudo-non-stream: error reading upstream (attempt %d): %v", attempt, readErr)
			return readErr
		}

		log.Infof("pseudo-non-stream: buffered %d bytes from upstream (attempt %d)", w.buf.Len(), attempt)

		// Audit
		approved, reason := AuditResponse(w.buf.Bytes(), w.modelName, w.auditKeywords)
		if approved {
			log.Debugf("pseudo-non-stream: audit passed (attempt %d)", attempt)
			w.reader = bytes.NewReader(w.buf.Bytes())
			return nil
		}

		log.Warnf("pseudo-non-stream: audit REJECTED (attempt %d, reason: %s)", attempt, reason)

		// Check if we can retry
		if attempt < w.maxRetries && w.retryFunc != nil {
			w.upstream.Close()
			newBody, err := w.retryFunc()
			if err != nil {
				log.Errorf("pseudo-non-stream: retry request failed: %v, using last response", err)
				break
			}
			w.upstream = newBody
			continue
		}

		// No more retries or no retry func, use last response
		break
	}

	// Fallback: use whatever we have
	log.Warnf("pseudo-non-stream: all retries exhausted or no retry available, using last response")
	w.reader = bytes.NewReader(w.buf.Bytes())
	return nil
}

// bufferFromReader reads all data from reader into w.buf while sending heartbeats
func (w *PseudoNonStreamBodyWrapper) bufferFromReader(reader io.Reader, ticker *time.Ticker) error {
	done := make(chan struct{})
	var readErr error

	go func() {
		defer close(done)
		_, readErr = io.Copy(&w.buf, reader)
	}()

	for {
		select {
		case <-done:
			return readErr
		case <-ticker.C:
			w.sendKeepAlive()
		}
	}
}

// sendKeepAlive writes a SSE keep-alive comment to the client.
func (w *PseudoNonStreamBodyWrapper) sendKeepAlive() {
	if w.rw == nil {
		return
	}
	_, err := w.rw.Write([]byte(pseudoNonStreamKeepAliveMsg))
	if err != nil {
		log.Debugf("pseudo-non-stream: failed to send keep-alive: %v", err)
		return
	}
	if flusher, ok := w.rw.(http.Flusher); ok {
		flusher.Flush()
	}
	log.Debugf("pseudo-non-stream: sent keep-alive heartbeat")
}

// ========== Response Audit ==========

var (
	auditToolCallPatterns []*regexp.Regexp
	auditPatternsOnce     sync.Once
)

func initAuditPatterns() {
	auditPatternsOnce.Do(func() {
		patterns := []string{
			`(?i)to=multi_tool_use`,
			`(?i)to=functions\.`,
			`"tool_uses"\s*:`,
			`"recipient_name"\s*:`,
			`"parameters"\s*:`,
			`(?i)functions\.(grep|glob|read_file|ls|Read|Grep|finder|edit_file|create_file|Bash|mcp_|shell_command)`,
			`(?i)\bcall_id\b`,
			`(?i)\btool_call\b`,
		}
		for _, p := range patterns {
			auditToolCallPatterns = append(auditToolCallPatterns, regexp.MustCompile(p))
		}
	})
}

// chineseSpamTokens contains representative spam/gambling Chinese phrases
// extracted from GPT-4o token vocabulary that indicate hallucinated garbage output.
// Source: https://gist.github.com/PkuCuipy/d01e833cf3d57ea67f2b7a645c5c3cb5
var chineseSpamTokens = []string{
	"天天中彩票",
	"彩神争霸",
	"大发快三",
	"大发时时彩",
	"北京赛车",
	"毛片免费视频",
	"无码不卡高清",
	"免费视频在线观看",
	"高清毛片在线",
	"久久免费热在线",
	"一本道高清无码",
	"久久综合久久爱",
	"不卡免费播放",
	"毛片高清免费视频",
	"无码一区二区",
	"日本毛片免费",
	"日本一级特黄",
	"夫妻性生活影片",
	"最新高清无码专区",
	"精品一区二区三区",
	"在线观看中文字幕",
	"热这里只有精品",
	"久久精品国产",
	"彩票天天送钱",
	"全民彩票天天",
	"福利彩票天天",
	"重庆时时彩",
	"天天彩票中奖",
	"大发彩票官网",
	"大发游戏官网",
}

// hasChineseSpamTokens checks if text contains Chinese spam/gambling tokens
func hasChineseSpamTokens(text string, extraKeywords []string) (bool, string) {
	for _, token := range chineseSpamTokens {
		if strings.Contains(text, token) {
			return true, token
		}
	}
	for _, token := range extraKeywords {
		if token != "" && strings.Contains(text, token) {
			return true, token
		}
	}
	return false, ""
}

// hasToolCallInText checks if the text content contains patterns that look like
// tool call syntax leaked into plain text output (common with gpt-5.3-codex)
func hasToolCallInText(text string) (bool, string) {
	initAuditPatterns()

	for _, re := range auditToolCallPatterns {
		if re.MatchString(text) {
			return true, re.String()
		}
	}
	return false, ""
}

// extractTextFromSSE extracts all text content from buffered SSE data.
// It parses SSE events looking for response.output_text.delta events and
// also checks the final response.completed snapshot.
func extractTextFromSSE(data []byte) string {
	var textBuilder strings.Builder
	remaining := data

	for len(remaining) > 0 {
		idx, delimLen := findSSEDelimiter(remaining)
		if idx == -1 {
			break
		}

		event := remaining[:idx+delimLen]
		remaining = remaining[idx+delimLen:]

		_, payload, done := parseSSEEvent(event)
		if done {
			break
		}
		if len(payload) == 0 {
			continue
		}

		eventType := gjson.GetBytes(payload, "type").String()

		switch eventType {
		case "response.output_text.delta":
			delta := gjson.GetBytes(payload, "delta").String()
			if delta != "" {
				textBuilder.WriteString(delta)
			}

		case "response.completed", "response.done":
			resp := gjson.GetBytes(payload, "response")
			if !resp.Exists() {
				continue
			}
			output := resp.Get("output")
			if !output.Exists() || !output.IsArray() {
				continue
			}
			for _, item := range output.Array() {
				if item.Get("type").String() != "message" {
					continue
				}
				content := item.Get("content")
				if !content.IsArray() {
					continue
				}
				for _, part := range content.Array() {
					partType := part.Get("type").String()
					if partType == "output_text" || partType == "text" {
						txt := part.Get("text").String()
						if txt != "" {
							textBuilder.WriteString(txt)
						}
					}
				}
			}
		}
	}

	return textBuilder.String()
}

// AuditResponse inspects buffered SSE response data for problematic content.
// Returns (approved, reason). If approved=false, reason describes what was detected.
//
// Rules:
// 1. Tool call syntax leaked into text output — all models
// 2. Chinese spam/gambling token hallucinations — all models
// 3. Extra user-configured audit keywords — all models
func AuditResponse(data []byte, modelName string, extraKeywords []string) (approved bool, reason string) {
	text := extractTextFromSSE(data)
	if text == "" {
		return true, ""
	}

	if matched, pattern := hasToolCallInText(text); matched {
		return false, "tool_call_in_text: " + pattern
	}

	if matched, token := hasChineseSpamTokens(text, extraKeywords); matched {
		return false, "chinese_spam_token: " + token
	}

	return true, ""
}
