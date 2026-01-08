package amp

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// ConnectionHealthConfig 连接健康检查配置
type ConnectionHealthConfig struct {
	// 写入超时 - 检测客户端断开
	WriteTimeout time.Duration
	// 读取空闲超时 - 上游无数据时的最大等待时间
	ReadIdleTimeout time.Duration
	// 是否启用详细日志
	VerboseLogging bool
}

// DefaultConnectionHealthConfig 默认配置
func DefaultConnectionHealthConfig() *ConnectionHealthConfig {
	return &ConnectionHealthConfig{
		WriteTimeout:    30 * time.Second,
		ReadIdleTimeout: 300 * time.Second, // 5分钟，允许AI长时间思考
		VerboseLogging:  false,
	}
}

// HealthyStreamWrapper 健康流包装器
// 解决 "terminated" 问题的关键组件
type HealthyStreamWrapper struct {
	reader       io.ReadCloser
	trace        *RequestTrace
	cfg          *ConnectionHealthConfig
	lastReadTime time.Time
	mu           sync.Mutex
	closed       bool
	closeOnce    sync.Once
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewHealthyStreamWrapper 创建健康流包装器
func NewHealthyStreamWrapper(ctx context.Context, reader io.ReadCloser, trace *RequestTrace, cfg *ConnectionHealthConfig) *HealthyStreamWrapper {
	if cfg == nil {
		cfg = DefaultConnectionHealthConfig()
	}
	wrapCtx, cancel := context.WithCancel(ctx)
	return &HealthyStreamWrapper{
		reader:       reader,
		trace:        trace,
		cfg:          cfg,
		lastReadTime: time.Now(),
		ctx:          wrapCtx,
		cancel:       cancel,
	}
}

// Read 带健康检查的读取
func (h *HealthyStreamWrapper) Read(p []byte) (int, error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	h.mu.Unlock()

	// 检查上下文是否已取消
	select {
	case <-h.ctx.Done():
		return 0, h.ctx.Err()
	default:
	}

	// 带超时的读取
	type readResult struct {
		n   int
		err error
	}
	ch := make(chan readResult, 1)

	go func() {
		n, err := h.reader.Read(p)
		ch <- readResult{n, err}
	}()

	timer := time.NewTimer(h.cfg.ReadIdleTimeout)
	defer timer.Stop()

	select {
	case res := <-ch:
		if res.n > 0 {
			h.mu.Lock()
			h.lastReadTime = time.Now()
			h.mu.Unlock()
		}
		return res.n, res.err
	case <-timer.C:
		// 读取超时 - 上游可能已断开
		h.logConnectionIssue("read idle timeout", nil)
		h.forceClose()
		return 0, &StreamTimeoutError{
			Duration: h.cfg.ReadIdleTimeout,
			Phase:    "read_idle",
		}
	case <-h.ctx.Done():
		return 0, h.ctx.Err()
	}
}

// Close 安全关闭
func (h *HealthyStreamWrapper) Close() error {
	var closeErr error
	h.closeOnce.Do(func() {
		h.mu.Lock()
		h.closed = true
		h.mu.Unlock()
		h.cancel()
		closeErr = h.reader.Close()

		// 更新 trace 状态
		if h.trace != nil {
			h.trace.SetResponse(200) // 正常关闭
		}
	})
	return closeErr
}

// forceClose 强制关闭（错误场景）
func (h *HealthyStreamWrapper) forceClose() {
	h.closeOnce.Do(func() {
		h.mu.Lock()
		h.closed = true
		h.mu.Unlock()
		h.cancel()
		_ = h.reader.Close()

		// 更新 trace 状态为错误
		if h.trace != nil {
			h.trace.SetError("stream_timeout")
		}
	})
}

func (h *HealthyStreamWrapper) logConnectionIssue(msg string, err error) {
	if h.cfg.VerboseLogging {
		fields := log.Fields{
			"lastRead": h.lastReadTime,
			"elapsed":  time.Since(h.lastReadTime),
		}
		if err != nil {
			fields["error"] = err.Error()
		}
		log.WithFields(fields).Warnf("stream health: %s", msg)
	}
}

// StreamTimeoutError 流超时错误
type StreamTimeoutError struct {
	Duration time.Duration
	Phase    string
}

func (e *StreamTimeoutError) Error() string {
	return "stream " + e.Phase + " timeout after " + e.Duration.String()
}

func (e *StreamTimeoutError) Timeout() bool   { return true }
func (e *StreamTimeoutError) Temporary() bool { return true }

// IsClientDisconnect 判断错误是否是客户端主动断开
func IsClientDisconnect(err error) bool {
	if err == nil {
		return false
	}

	// 检查常见的客户端断开错误
	if errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	// 检查网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// 检查系统调用错误
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNRESET) ||
			errors.Is(opErr.Err, syscall.EPIPE) ||
			errors.Is(opErr.Err, syscall.ECONNABORTED) {
			return true
		}
	}

	return false
}

// LogStreamError 记录流错误（带分类）
func LogStreamError(err error, requestID string) {
	if err == nil {
		return
	}

	category := ClassifyStreamError(err)
	fields := log.Fields{
		"requestID": requestID,
		"category":  category,
	}

	switch category {
	case "client_disconnect":
		log.WithFields(fields).Debug("stream: client disconnected")
	case "upstream_timeout":
		log.WithFields(fields).Warn("stream: upstream timeout")
	case "network_error":
		log.WithFields(fields).Warn("stream: network error: " + err.Error())
	default:
		log.WithFields(fields).Error("stream: unexpected error: " + err.Error())
	}
}

// ClassifyStreamError 分类流错误
func ClassifyStreamError(err error) string {
	if err == nil {
		return "none"
	}

	if IsClientDisconnect(err) {
		return "client_disconnect"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "upstream_timeout"
	}

	var streamTimeout *StreamTimeoutError
	if errors.As(err, &streamTimeout) {
		return "upstream_timeout"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "upstream_timeout"
		}
		return "network_error"
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "upstream_closed"
	}

	return "unknown"
}
