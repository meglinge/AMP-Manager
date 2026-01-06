package amp

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// StreamConfig 流式响应配置
type StreamConfig struct {
	KeepAliveInterval time.Duration `json:"keepAliveInterval"`
	EnableKeepAlive   bool          `json:"enableKeepAlive"`
}

// DefaultStreamConfig 默认流式配置
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		KeepAliveInterval: 15 * time.Second,
		EnableKeepAlive:   true,
	}
}

// SSEStreamHandler 处理 SSE 流式响应，支持心跳和终端错误
type SSEStreamHandler struct {
	writer          http.ResponseWriter
	flusher         http.Flusher
	ctx             context.Context
	cancel          context.CancelFunc
	keepAlive       *time.Ticker
	mu              sync.Mutex
	closed          bool
	wroteFirstChunk bool
	cfg             *StreamConfig
}

// NewSSEStreamHandler 创建 SSE 流处理器
func NewSSEStreamHandler(w http.ResponseWriter, ctx context.Context, cfg *StreamConfig) *SSEStreamHandler {
	if cfg == nil {
		cfg = DefaultStreamConfig()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Warn("sse stream: ResponseWriter does not support Flusher")
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)

	handler := &SSEStreamHandler{
		writer:  w,
		flusher: flusher,
		ctx:     streamCtx,
		cancel:  cancel,
		cfg:     cfg,
	}

	if cfg.EnableKeepAlive && cfg.KeepAliveInterval > 0 {
		handler.keepAlive = time.NewTicker(cfg.KeepAliveInterval)
		go handler.keepAliveLoop()
	}

	return handler
}

// keepAliveLoop 发送心跳保持连接
func (h *SSEStreamHandler) keepAliveLoop() {
	if h.keepAlive == nil {
		return
	}
	defer h.keepAlive.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.keepAlive.C:
			h.mu.Lock()
			if h.closed {
				h.mu.Unlock()
				return
			}
			// SSE 心跳格式: 注释行
			_, err := h.writer.Write([]byte(": keep-alive\n\n"))
			if err != nil {
				log.Debugf("sse stream: keep-alive write failed: %v", err)
				h.mu.Unlock()
				return
			}
			h.flusher.Flush()
			h.mu.Unlock()
		}
	}
}

// WriteChunk 写入数据块
func (h *SSEStreamHandler) WriteChunk(data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return io.ErrClosedPipe
	}

	_, err := h.writer.Write(data)
	if err != nil {
		return err
	}
	h.flusher.Flush()
	h.wroteFirstChunk = true
	return nil
}

// WriteTerminalError 在流末尾写入错误信息
// 只有在已经开始发送数据后才使用此方法
func (h *SSEStreamHandler) WriteTerminalError(statusCode int, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	errPayload := BuildSSEErrorEvent(statusCode, message)
	_, _ = h.writer.Write(errPayload)
	h.flusher.Flush()
}

// Close 关闭流处理器
func (h *SSEStreamHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true
	h.cancel()
	if h.keepAlive != nil {
		h.keepAlive.Stop()
	}
}

// WroteFirstChunk 返回是否已写入第一个数据块
func (h *SSEStreamHandler) WroteFirstChunk() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.wroteFirstChunk
}

// BuildSSEErrorEvent 构建 SSE 格式的错误事件
func BuildSSEErrorEvent(statusCode int, message string) []byte {
	errBody := BuildErrorResponseBody(statusCode, message)
	return []byte("event: error\ndata: " + string(errBody) + "\n\n")
}

// SSEKeepAliveWrapper 包装 io.ReadCloser，在读取时支持心跳写入
type SSEKeepAliveWrapper struct {
	reader          io.ReadCloser
	writer          http.ResponseWriter
	flusher         http.Flusher
	ctx             context.Context
	cancel          context.CancelFunc
	keepAlive       *time.Ticker
	mu              sync.Mutex
	closed          bool
	wroteData       bool
	lastActivity    time.Time
	cfg             *StreamConfig
}

// NewSSEKeepAliveWrapper 创建 SSE Keep-Alive 包装器
func NewSSEKeepAliveWrapper(reader io.ReadCloser, w http.ResponseWriter, ctx context.Context, cfg *StreamConfig) *SSEKeepAliveWrapper {
	if cfg == nil {
		cfg = DefaultStreamConfig()
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	wrapCtx, cancel := context.WithCancel(ctx)

	wrapper := &SSEKeepAliveWrapper{
		reader:       reader,
		writer:       w,
		flusher:      flusher,
		ctx:          wrapCtx,
		cancel:       cancel,
		cfg:          cfg,
		lastActivity: time.Now(),
	}

	if cfg.EnableKeepAlive && cfg.KeepAliveInterval > 0 {
		wrapper.keepAlive = time.NewTicker(cfg.KeepAliveInterval)
		go wrapper.keepAliveLoop()
	}

	return wrapper
}

// keepAliveLoop 心跳循环
func (w *SSEKeepAliveWrapper) keepAliveLoop() {
	if w.keepAlive == nil {
		return
	}
	defer w.keepAlive.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.keepAlive.C:
			w.mu.Lock()
			if w.closed {
				w.mu.Unlock()
				return
			}
			// 只有超过间隔时间没有活动才发送心跳
			if time.Since(w.lastActivity) >= w.cfg.KeepAliveInterval {
				_, err := w.writer.Write([]byte(": keep-alive\n\n"))
				if err != nil {
					log.Debugf("sse keep-alive: write failed: %v", err)
					w.mu.Unlock()
					return
				}
				w.flusher.Flush()
			}
			w.mu.Unlock()
		}
	}
}

// Read 实现 io.Reader
func (w *SSEKeepAliveWrapper) Read(p []byte) (int, error) {
	n, err := w.reader.Read(p)
	if n > 0 {
		w.mu.Lock()
		w.lastActivity = time.Now()
		w.wroteData = true
		w.mu.Unlock()
	}
	return n, err
}

// Close 实现 io.Closer
func (w *SSEKeepAliveWrapper) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.cancel()
	if w.keepAlive != nil {
		w.keepAlive.Stop()
	}
	w.mu.Unlock()

	return w.reader.Close()
}

// WriteTerminalError 写入终端错误
func (w *SSEKeepAliveWrapper) WriteTerminalError(statusCode int, message string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	errPayload := BuildSSEErrorEvent(statusCode, message)
	_, _ = w.writer.Write(errPayload)
	w.flusher.Flush()
}
