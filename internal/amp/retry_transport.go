package amp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// RetryConfig 重试配置（可通过管理员界面配置）
type RetryConfig struct {
	Enabled           bool          `json:"enabled"`
	MaxAttempts       int           `json:"maxAttempts"`
	GateTimeout       time.Duration `json:"gateTimeout"`
	MaxBodyBytes      int64         `json:"maxBodyBytes"`
	BackoffBase       time.Duration `json:"backoffBase"`
	BackoffMax        time.Duration `json:"backoffMax"`
	RetryOn429        bool          `json:"retryOn429"`
	RetryOn5xx        bool          `json:"retryOn5xx"`
	RespectRetryAfter bool          `json:"respectRetryAfter"`
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Enabled:           true,
		MaxAttempts:       3,
		GateTimeout:       10 * time.Second,
		MaxBodyBytes:      10 << 20, // 10MB
		BackoffBase:       100 * time.Millisecond,
		BackoffMax:        2 * time.Second,
		RetryOn429:        true,
		RetryOn5xx:        true,
		RespectRetryAfter: true,
	}
}

// RetryConfigFromDB 从数据库配置创建 RetryConfig
func RetryConfigFromDB(enabled bool, maxAttempts int, gateTimeoutMs, maxBodyBytes, backoffBaseMs, backoffMaxMs int64, retryOn429, retryOn5xx, respectRetryAfter bool) *RetryConfig {
	return &RetryConfig{
		Enabled:           enabled,
		MaxAttempts:       maxAttempts,
		GateTimeout:       time.Duration(gateTimeoutMs) * time.Millisecond,
		MaxBodyBytes:      maxBodyBytes,
		BackoffBase:       time.Duration(backoffBaseMs) * time.Millisecond,
		BackoffMax:        time.Duration(backoffMaxMs) * time.Millisecond,
		RetryOn429:        retryOn429,
		RetryOn5xx:        retryOn5xx,
		RespectRetryAfter: respectRetryAfter,
	}
}

// RetryTransport 实现首包门控重试的 HTTP RoundTripper
type RetryTransport struct {
	Base http.RoundTripper
	cfg  *RetryConfig
	mu   sync.RWMutex
}

// NewRetryTransport 创建重试 Transport
func NewRetryTransport(base http.RoundTripper, cfg *RetryConfig) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}
	return &RetryTransport{
		Base: base,
		cfg:  cfg,
	}
}

// UpdateConfig 动态更新配置（线程安全）
func (rt *RetryTransport) UpdateConfig(cfg *RetryConfig) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cfg = cfg
}

// getConfig 获取当前配置（线程安全）
func (rt *RetryTransport) getConfig() *RetryConfig {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cfg
}

// RetryExhaustedError 重试耗尽错误
type RetryExhaustedError struct {
	Attempts int
	LastErr  error
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("retry exhausted after %d attempts: %v", e.Attempts, e.LastErr)
}

func (e *RetryExhaustedError) Unwrap() error {
	return e.LastErr
}

// RoundTrip 实现 http.RoundTripper 接口
func (rt *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cfg := rt.getConfig()
	
	// 如果禁用重试或只允许1次，直接调用底层
	if !cfg.Enabled || cfg.MaxAttempts <= 1 {
		return rt.Base.RoundTrip(req)
	}

	// 缓存请求体以支持重放
	bodyBytes, canRetry, err := rt.cacheRequestBody(req, cfg.MaxBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to cache request body: %w", err)
	}
	if !canRetry {
		// 请求体太大，无法重试
		log.Debug("retry: request body too large, skipping retry")
		return rt.Base.RoundTrip(req)
	}

	var lastErr error
	var lastResp *http.Response
	
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// 检查 context 是否已取消
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		// 为本次尝试准备请求
		attemptReq := rt.cloneRequest(req, bodyBytes)

		// 发起请求
		resp, err := rt.Base.RoundTrip(attemptReq)
		
		if err != nil {
			lastErr = err
			if rt.shouldRetryError(err) && attempt < cfg.MaxAttempts {
				rt.logRetryAttempt(req, attempt, cfg.MaxAttempts, err, nil)
				rt.backoff(req.Context(), attempt, cfg, nil)
				continue
			}
			return nil, err
		}

		// 检查是否需要根据状态码重试
		if rt.shouldRetryStatusCode(resp.StatusCode, cfg) && attempt < cfg.MaxAttempts {
			retryAfter := rt.parseRetryAfter(resp, cfg)
			rt.logRetryAttempt(req, attempt, cfg.MaxAttempts, nil, resp)
			_ = resp.Body.Close()
			rt.backoff(req.Context(), attempt, cfg, retryAfter)
			continue
		}

		// 对流式响应进行首字节门控
		if rt.shouldGateResponse(resp) {
			firstByte, probeErr := rt.probeFirstByte(req.Context(), resp.Body, cfg.GateTimeout)
			if probeErr != nil {
				lastErr = probeErr
				_ = resp.Body.Close()
				if attempt < cfg.MaxAttempts {
					rt.logRetryAttempt(req, attempt, cfg.MaxAttempts, probeErr, resp)
					rt.backoff(req.Context(), attempt, cfg, nil)
					continue
				}
				return nil, &RetryExhaustedError{Attempts: attempt, LastErr: probeErr}
			}

			// 成功读取首字节，回填到响应体
			resp.Body = &readCloser{
				r: io.MultiReader(bytes.NewReader(firstByte), resp.Body),
				c: resp.Body,
			}
		}

		// 成功
		if attempt > 1 {
			log.Infof("retry: request succeeded after %d attempts: %s %s", attempt, req.Method, req.URL.Path)
		}
		return resp, nil
	}

	// 所有重试都失败
	if lastResp != nil {
		_ = lastResp.Body.Close()
	}
	return nil, &RetryExhaustedError{Attempts: cfg.MaxAttempts, LastErr: lastErr}
}

// cacheRequestBody 缓存请求体以支持重放
func (rt *RetryTransport) cacheRequestBody(req *http.Request, maxBytes int64) ([]byte, bool, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, true, nil
	}

	// 如果已有 GetBody，使用它
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, false, err
		}
		data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
		_ = body.Close()
		if err != nil {
			return nil, false, err
		}
		if int64(len(data)) > maxBytes {
			return nil, false, nil // 太大，不重试
		}
		return data, true, nil
	}

	// 读取并缓存请求体
	data, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	_ = req.Body.Close()

	if int64(len(data)) > maxBytes {
		// 太大，重建 body 并返回不可重试
		req.Body = io.NopCloser(bytes.NewReader(data))
		return nil, false, nil
	}

	// 设置 GetBody 以支持重放
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))

	return data, true, nil
}

// cloneRequest 克隆请求并设置新的请求体
func (rt *RetryTransport) cloneRequest(req *http.Request, bodyBytes []byte) *http.Request {
	clone := req.Clone(req.Context())
	if bodyBytes == nil {
		clone.Body = http.NoBody
		clone.ContentLength = 0
	} else {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		clone.ContentLength = int64(len(bodyBytes))
	}
	return clone
}

// shouldRetryError 判断网络错误是否应该重试
func (rt *RetryTransport) shouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	// EOF 或 unexpected EOF
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// 网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		// 超时或临时错误可重试
		if netErr.Timeout() {
			return true
		}
	}

	// 系统调用错误
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// 连接重置、连接被拒绝等
		if errors.Is(opErr.Err, syscall.ECONNRESET) ||
			errors.Is(opErr.Err, syscall.ECONNREFUSED) ||
			errors.Is(opErr.Err, syscall.ETIMEDOUT) ||
			errors.Is(opErr.Err, syscall.EPIPE) {
			return true
		}
	}

	// 检查错误字符串（兜底）
	errStr := err.Error()
	retryableStrings := []string{
		"connection reset",
		"connection refused",
		"i/o timeout",
		"no such host",
		"temporary failure",
		"EOF",
		"broken pipe",
	}
	for _, s := range retryableStrings {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(s)) {
			return true
		}
	}

	return false
}

// shouldRetryStatusCode 判断状态码是否应该重试
func (rt *RetryTransport) shouldRetryStatusCode(statusCode int, cfg *RetryConfig) bool {
	if statusCode == 429 && cfg.RetryOn429 {
		return true
	}
	if statusCode >= 500 && statusCode < 600 && cfg.RetryOn5xx {
		// 500, 502, 503, 504 可重试
		return statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
	}
	return false
}

// shouldGateResponse 判断是否需要对响应进行首字节门控
func (rt *RetryTransport) shouldGateResponse(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream")
}

// probeFirstByte 探测响应的第一个字节
func (rt *RetryTransport) probeFirstByte(ctx context.Context, body io.ReadCloser, timeout time.Duration) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		buf := make([]byte, 1)
		n, err := body.Read(buf)
		if n > 0 {
			ch <- result{data: buf[:n], err: nil}
		} else {
			ch <- result{data: nil, err: err}
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-ch:
		if res.data != nil {
			return res.data, nil
		}
		if res.err == nil {
			res.err = io.EOF
		}
		return nil, res.err
	case <-timer.C:
		// 超时但不一定是错误，可能上游在处理中
		// 返回特殊错误让上层决定是否重试
		return nil, fmt.Errorf("first byte timeout after %v", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// parseRetryAfter 解析 Retry-After 头
func (rt *RetryTransport) parseRetryAfter(resp *http.Response, cfg *RetryConfig) *time.Duration {
	if !cfg.RespectRetryAfter {
		return nil
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return nil
	}

	// 尝试解析为秒数
	var seconds int
	if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err == nil {
		d := time.Duration(seconds) * time.Second
		return &d
	}

	// 尝试解析为 HTTP 日期
	if t, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		d := time.Until(t)
		if d > 0 {
			return &d
		}
	}

	return nil
}

// backoff 执行退避等待
func (rt *RetryTransport) backoff(ctx context.Context, attempt int, cfg *RetryConfig, retryAfter *time.Duration) {
	var delay time.Duration

	if retryAfter != nil {
		delay = *retryAfter
	} else {
		// 指数退避 + 抖动
		delay = cfg.BackoffBase * (1 << (attempt - 1))
		if delay > cfg.BackoffMax {
			delay = cfg.BackoffMax
		}
		// 添加 ±25% 的抖动
		jitter := time.Duration(rand.Float64()*float64(delay)*0.5) - delay/4
		delay += jitter
	}

	log.Debugf("retry: backing off for %v before attempt %d", delay, attempt+1)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

// logRetryAttempt 记录重试日志
func (rt *RetryTransport) logRetryAttempt(req *http.Request, attempt, maxAttempts int, err error, resp *http.Response) {
	fields := log.Fields{
		"method":      req.Method,
		"path":        req.URL.Path,
		"attempt":     attempt,
		"maxAttempts": maxAttempts,
	}

	if err != nil {
		fields["error"] = err.Error()
		fields["errorClass"] = classifyError(err)
	}

	if resp != nil {
		fields["statusCode"] = resp.StatusCode
	}

	log.WithFields(fields).Warnf("retry: attempt %d/%d failed, will retry", attempt, maxAttempts)
}

// classifyError 将错误分类为可观测的类型
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "eof"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network"
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline"
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "connection reset") {
		return "reset"
	}
	if strings.Contains(errStr, "connection refused") {
		return "refused"
	}
	if strings.Contains(errStr, "dns") || strings.Contains(errStr, "no such host") {
		return "dns"
	}
	if strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") {
		return "tls"
	}

	return "unknown"
}
