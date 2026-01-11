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

var (
	globalStreamConfig *StreamConfig
	streamConfigMu     sync.RWMutex
)

// GetStreamConfig 获取当前流配置
func GetStreamConfig() *StreamConfig {
	streamConfigMu.RLock()
	defer streamConfigMu.RUnlock()
	if globalStreamConfig == nil {
		return DefaultStreamConfig()
	}
	return globalStreamConfig
}

// UpdateStreamConfig 更新流配置
func UpdateStreamConfig(keepAliveInterval time.Duration) {
	streamConfigMu.Lock()
	defer streamConfigMu.Unlock()
	globalStreamConfig = &StreamConfig{
		KeepAliveInterval: keepAliveInterval,
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
	mu              sync.Mutex    // 保护共享状态
	writeMu         sync.Mutex    // 串行化写入操作，防止交错
	closed          bool
	wroteFirstChunk bool
	cfg             *StreamConfig
}

// NewSSEStreamHandler 创建 SSE 流处理器
func NewSSEStreamHandler(w http.ResponseWriter, ctx context.Context, cfg *StreamConfig) *SSEStreamHandler {
	if cfg == nil {
		cfg = GetStreamConfig()
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

	keepAliveData := []byte(": keep-alive\n\n")

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.keepAlive.C:
			// 锁内只检查状态
			h.mu.Lock()
			if h.closed {
				h.mu.Unlock()
				return
			}
			h.mu.Unlock()

			// 写操作在状态锁外执行，使用写锁串行化
			h.writeMu.Lock()
			_, err := h.writer.Write(keepAliveData)
			if err == nil {
				h.flusher.Flush()
			}
			h.writeMu.Unlock()

			if err != nil {
				log.Debugf("sse stream: keep-alive write failed: %v", err)
				h.Close() // 写失败时调用 Close() 统一设置 closed=true 并 cancel()
				return
			}
		}
	}
}

// WriteChunk 写入数据块
func (h *SSEStreamHandler) WriteChunk(data []byte) error {
	// 先检查 context 是否已取消
	select {
	case <-h.ctx.Done():
		return h.ctx.Err()
	default:
	}

	// 锁内只检查状态
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return io.ErrClosedPipe
	}
	h.mu.Unlock()

	// 复制数据用于锁外写入（避免调用方修改）
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	// 写操作在状态锁外执行，使用写锁串行化
	h.writeMu.Lock()
	_, err := h.writer.Write(dataCopy)
	if err == nil {
		h.flusher.Flush()
	}
	h.writeMu.Unlock()

	if err != nil {
		return err
	}

	// 更新状态
	h.mu.Lock()
	h.wroteFirstChunk = true
	h.mu.Unlock()

	return nil
}

// WriteTerminalError 在流末尾写入错误信息
// 只有在已经开始发送数据后才使用此方法
func (h *SSEStreamHandler) WriteTerminalError(statusCode int, message string) {
	// 先检查 context 是否已取消
	select {
	case <-h.ctx.Done():
		return
	default:
	}

	// 锁内只检查状态
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	// 准备数据
	errPayload := BuildSSEErrorEvent(statusCode, message)

	// 写操作在状态锁外执行，使用写锁串行化
	h.writeMu.Lock()
	_, _ = h.writer.Write(errPayload)
	h.flusher.Flush()
	h.writeMu.Unlock()
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
	reader       io.ReadCloser
	writer       http.ResponseWriter
	flusher      http.Flusher
	ctx          context.Context
	cancel       context.CancelFunc
	keepAlive    *time.Ticker
	mu           sync.Mutex // 保护共享状态
	writeMu      sync.Mutex // 串行化写入操作
	closed       bool
	wroteData    bool
	lastActivity time.Time
	cfg          *StreamConfig
}

// NewSSEKeepAliveWrapper 创建 SSE Keep-Alive 包装器
func NewSSEKeepAliveWrapper(reader io.ReadCloser, w http.ResponseWriter, ctx context.Context, cfg *StreamConfig) *SSEKeepAliveWrapper {
	if cfg == nil {
		cfg = GetStreamConfig()
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

	keepAliveData := []byte(": keep-alive\n\n")

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.keepAlive.C:
			// 锁内只检查状态和时间
			w.mu.Lock()
			if w.closed {
				w.mu.Unlock()
				return
			}
			shouldSend := time.Since(w.lastActivity) >= w.cfg.KeepAliveInterval
			w.mu.Unlock()

			if !shouldSend {
				continue
			}

			// 写操作在状态锁外执行，使用写锁串行化
			w.writeMu.Lock()
			_, err := w.writer.Write(keepAliveData)
			if err == nil {
				w.flusher.Flush()
			}
			w.writeMu.Unlock()

			if err != nil {
				log.Debugf("sse keep-alive: write failed: %v", err)
				w.Close() // 写失败时调用 Close() 统一设置 closed=true 并 cancel()
				return
			}
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
	// 先检查 context 是否已取消
	select {
	case <-w.ctx.Done():
		return
	default:
	}

	// 锁内只检查状态
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// 准备数据
	errPayload := BuildSSEErrorEvent(statusCode, message)

	// 写操作在状态锁外执行，使用写锁串行化
	w.writeMu.Lock()
	_, _ = w.writer.Write(errPayload)
	w.flusher.Flush()
	w.writeMu.Unlock()
}
