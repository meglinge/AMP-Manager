package amp

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"
)

// ErrorClass 错误分类枚举
type ErrorClass string

const (
	ErrorClassTimeout       ErrorClass = "timeout"
	ErrorClassClientClosed  ErrorClass = "client_closed"
	ErrorClassServerCancel  ErrorClass = "server_cancel"
	ErrorClassBackpressure  ErrorClass = "backpressure"
	ErrorClassUpstreamReset ErrorClass = "upstream_reset"
	ErrorClassProtocol      ErrorClass = "protocol"
	ErrorClassCompleted     ErrorClass = "completed" // 正常完成（如读取上游响应流结束）
	ErrorClassUnknown       ErrorClass = "unknown"
)

// ClassifyError 统一错误分类器
// op 参数表示操作类型，如 "read", "write", "dial" 等，用于辅助判断
func ClassifyError(err error, op string) ErrorClass {
	if err == nil {
		return ErrorClassUnknown
	}

	// 1. 首先检查明确的超时错误
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassTimeout
	}

	var streamTimeout *StreamTimeoutError
	if errors.As(err, &streamTimeout) {
		return ErrorClassTimeout
	}

	// 检查 net.Error 的 Timeout() 方法
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrorClassTimeout
	}

	// 2. 检查服务端主动取消
	if errors.Is(err, context.Canceled) {
		return ErrorClassServerCancel
	}

	// 3. 检查 EOF - 根据操作类型区分
	// read 操作时 EOF 表示上游正常结束
	// write 操作时 EOF 表示客户端断开
	if errors.Is(err, io.EOF) {
		if op == "read" {
			return ErrorClassCompleted
		}
		return ErrorClassClientClosed
	}

	// 4. 检查客户端关闭连接的明确信号
	if isClientClosedError(err, op) {
		return ErrorClassClientClosed
	}

	// 5. 检查上游重置
	if isUpstreamResetError(err) {
		return ErrorClassUpstreamReset
	}

	// 6. 检查协议错误
	if isProtocolError(err) {
		return ErrorClassProtocol
	}

	// 7. 检查背压相关错误
	if isBackpressureError(err, op) {
		return ErrorClassBackpressure
	}

	// 默认归为未知
	return ErrorClassUnknown
}

// isClientClosedError 判断是否为客户端主动关闭连接
// 只对明确的客户端断开场景返回 true
// op 参数用于区分操作类型：write 操作时 broken pipe 等错误归为客户端断开
func isClientClosedError(err error, op string) bool {
	if err == nil {
		return false
	}

	// 关闭的管道
	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	// 检查系统调用错误 - 这些是客户端断开的明确信号
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// ECONNRESET - 连接被对端重置
		if errors.Is(opErr.Err, syscall.ECONNRESET) {
			return true
		}
		// EPIPE - 写入已关闭的管道（仅 write 操作时视为客户端断开）
		if errors.Is(opErr.Err, syscall.EPIPE) && op == "write" {
			return true
		}
		// ECONNABORTED - 连接被中止
		if errors.Is(opErr.Err, syscall.ECONNABORTED) {
			return true
		}

		// 检查内部错误的字符串表示
		if opErr.Err != nil {
			errStr := strings.ToLower(opErr.Err.Error())
			// broken pipe 仅在 write 操作时视为客户端断开
			if strings.Contains(errStr, "broken pipe") && op == "write" {
				return true
			}
			if strings.Contains(errStr, "connection reset by peer") ||
				strings.Contains(errStr, "forcibly closed") {
				return true
			}
		}
	}

	// 检查错误信息字符串 (作为最后的备选方案)
	errStr := strings.ToLower(err.Error())
	// broken pipe 仅在 write 操作时视为客户端断开
	if strings.Contains(errStr, "broken pipe") && op == "write" {
		return true
	}
	if strings.Contains(errStr, "connection reset by peer") {
		return true
	}

	return false
}

// isUpstreamResetError 判断是否为上游重置连接
func isUpstreamResetError(err error) bool {
	if err == nil {
		return false
	}

	// 非预期的 EOF 通常表示上游异常关闭
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "upstream") ||
		strings.Contains(errStr, "server reset") {
		return true
	}

	return false
}

// isProtocolError 判断是否为协议错误
func isProtocolError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "protocol") ||
		strings.Contains(errStr, "malformed") ||
		strings.Contains(errStr, "invalid header") ||
		strings.Contains(errStr, "bad request")
}

// isBackpressureError 判断是否为背压相关错误
func isBackpressureError(err error, op string) bool {
	if err == nil {
		return false
	}

	// 写操作时的临时错误可能是背压
	if op == "write" {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Temporary() {
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "buffer full") ||
		strings.Contains(errStr, "would block") ||
		strings.Contains(errStr, "resource temporarily unavailable")
}

// ErrorClassInfo 错误分类详情
type ErrorClassInfo struct {
	Class       ErrorClass
	Description string
	Retryable   bool
}

// GetErrorClassInfo 获取错误分类的详细信息
func GetErrorClassInfo(class ErrorClass) ErrorClassInfo {
	switch class {
	case ErrorClassTimeout:
		return ErrorClassInfo{
			Class:       class,
			Description: "操作超时",
			Retryable:   true,
		}
	case ErrorClassClientClosed:
		return ErrorClassInfo{
			Class:       class,
			Description: "客户端主动关闭连接",
			Retryable:   false,
		}
	case ErrorClassServerCancel:
		return ErrorClassInfo{
			Class:       class,
			Description: "服务端取消操作",
			Retryable:   false,
		}
	case ErrorClassBackpressure:
		return ErrorClassInfo{
			Class:       class,
			Description: "背压/缓冲区满",
			Retryable:   true,
		}
	case ErrorClassUpstreamReset:
		return ErrorClassInfo{
			Class:       class,
			Description: "上游重置连接",
			Retryable:   true,
		}
	case ErrorClassProtocol:
		return ErrorClassInfo{
			Class:       class,
			Description: "协议错误",
			Retryable:   false,
		}
	case ErrorClassCompleted:
		return ErrorClassInfo{
			Class:       class,
			Description: "正常完成",
			Retryable:   false,
		}
	default:
		return ErrorClassInfo{
			Class:       ErrorClassUnknown,
			Description: "未知错误",
			Retryable:   false,
		}
	}
}
