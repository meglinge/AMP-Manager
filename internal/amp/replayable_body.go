package amp

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

const (
	DefaultMaxBodySize = 10 << 20 // 10MB
	bufferPoolSize     = 32 << 10 // 32KB buffer for pool
)

var (
	ErrBodyTooLarge   = errors.New("request body exceeds maximum size limit")
	ErrBodyTruncated  = errors.New("body was truncated due to size limit")
	ErrAlreadyClosed  = errors.New("body already closed")
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, bufferPoolSize)
		return &buf
	},
}

// ReplayableBody 支持多次读取的请求体封装
// 第一次读取时缓存数据，后续可创建新的 Reader 重复消费
type ReplayableBody struct {
	data      []byte
	maxSize   int64
	truncated bool
	size      int64
	mu        sync.RWMutex
	closed    bool
}

// NewReplayableBody 从 io.ReadCloser 创建可重放的 body
// maxSize 为最大缓存大小，超出时返回错误；传 0 使用默认值 (10MB)
func NewReplayableBody(r io.ReadCloser, maxSize int64) (*ReplayableBody, error) {
	if r == nil {
		return &ReplayableBody{
			data:    nil,
			maxSize: maxSize,
		}, nil
	}

	if maxSize <= 0 {
		maxSize = DefaultMaxBodySize
	}

	rb := &ReplayableBody{
		maxSize: maxSize,
	}

	// 使用 buffer pool 读取
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	var result bytes.Buffer
	result.Grow(int(min(maxSize, 64<<10))) // 预分配最多 64KB

	var totalRead int64
	for {
		n, err := r.Read(buf)
		if n > 0 {
			remaining := maxSize - totalRead
			if int64(n) > remaining {
				// 超限：写入剩余允许的部分
				writeSize := int(remaining) // 安全：remaining <= maxSize <= 10MB
				result.Write(buf[:writeSize])
				totalRead = maxSize
				rb.truncated = true
				_ = r.Close()
				return nil, ErrBodyTooLarge
			}
			result.Write(buf[:n])
			totalRead += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = r.Close()
			return nil, err
		}
	}

	_ = r.Close()
	rb.data = result.Bytes()
	rb.size = int64(len(rb.data))
	return rb, nil
}

// NewReplayableBodyWithTruncation 创建可重放 body，超限时截断而非返回错误
func NewReplayableBodyWithTruncation(r io.ReadCloser, maxSize int64) (*ReplayableBody, error) {
	if r == nil {
		return &ReplayableBody{
			data:    nil,
			maxSize: maxSize,
		}, nil
	}

	if maxSize <= 0 {
		maxSize = DefaultMaxBodySize
	}

	rb := &ReplayableBody{
		maxSize: maxSize,
	}

	// 使用 LimitReader 强制限制读取大小
	lr := io.LimitReader(r, maxSize+1) // 多读1字节检测是否超限

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	var result bytes.Buffer
	result.Grow(int(min(maxSize, 64<<10)))

	var totalRead int64
	for {
		n, err := lr.Read(buf)
		if n > 0 {
			if totalRead+int64(n) > maxSize {
				// 超限，截断
				remaining := maxSize - totalRead
				if remaining > 0 {
					writeSize := int(remaining) // 安全：remaining <= maxSize <= 10MB
					result.Write(buf[:writeSize])
				}
				rb.truncated = true
				totalRead = maxSize
				break
			}
			result.Write(buf[:n])
			totalRead += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = r.Close()
			return nil, err
		}
	}

	_ = r.Close()
	rb.data = result.Bytes()
	rb.size = int64(len(rb.data))
	return rb, nil
}

// NewReader 创建新的 Reader 用于消费 body 数据
// 每次调用返回独立的 Reader，可并发使用
func (rb *ReplayableBody) NewReader() io.Reader {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.data == nil {
		return bytes.NewReader(nil)
	}
	return bytes.NewReader(rb.data)
}

// NewReadCloser 创建新的 ReadCloser，用于替换 http.Request.Body
func (rb *ReplayableBody) NewReadCloser() io.ReadCloser {
	return io.NopCloser(rb.NewReader())
}

// Bytes 返回缓存的原始字节切片
// 注意：返回的是内部数据的引用，调用者不应修改
func (rb *ReplayableBody) Bytes() []byte {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.data
}

// BytesCopy 返回缓存数据的副本（安全修改）
func (rb *ReplayableBody) BytesCopy() []byte {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.data == nil {
		return nil
	}
	cp := make([]byte, len(rb.data))
	copy(cp, rb.data)
	return cp
}

// IsTruncated 返回 body 是否因超限被截断
func (rb *ReplayableBody) IsTruncated() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.truncated
}

// Size 返回缓存的 body 大小
func (rb *ReplayableBody) Size() int64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// MaxSize 返回最大允许大小
func (rb *ReplayableBody) MaxSize() int64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.maxSize
}

// IsEmpty 返回 body 是否为空
func (rb *ReplayableBody) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.data == nil || len(rb.data) == 0
}

// Clear 释放缓存的数据（用于主动释放内存）
func (rb *ReplayableBody) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.data = nil
	rb.size = 0
	rb.closed = true
}

// String 返回 body 内容的字符串表示
func (rb *ReplayableBody) String() string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return string(rb.data)
}

// WrapRequest 将 ReplayableBody 应用到 http.Request
// 返回一个新 Reader 并重置请求的 Body 和 ContentLength
func (rb *ReplayableBody) WrapRequest(bodyPtr *io.ReadCloser, contentLengthPtr *int64) {
	if bodyPtr == nil {
		return
	}

	rb.mu.RLock()
	defer rb.mu.RUnlock()

	*bodyPtr = io.NopCloser(bytes.NewReader(rb.data))
	if contentLengthPtr != nil {
		*contentLengthPtr = rb.size
	}
}
