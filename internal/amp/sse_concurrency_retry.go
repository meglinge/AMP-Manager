package amp

import (
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	sseConcurrencyRetryMax     = 3
	sseConcurrencyRetryBaseWait = 2 * time.Second
)

// isSSERetryableError checks if an SSE event frame contains a retryable upstream error.
// Matched patterns:
//
//  1. Concurrency limit exceeded (event: error):
//     event: error
//     data: {"error":{"message":"Concurrency limit exceeded ...","type":"rate_limit_error"}}
//
//  2. Stream read error (no event line, top-level type:"error"):
//     data: {"error":{"code":"stream_read_error",...},"type":"error"}
func isSSERetryableError(frame []byte) bool {
	eventName, payload, _ := parseSSEEvent(frame)
	if len(payload) == 0 {
		return false
	}

	// Pattern 1: event: error + rate_limit_error with concurrency limit
	if eventName == "error" {
		errType := gjson.GetBytes(payload, "error.type").String()
		errMsg := gjson.GetBytes(payload, "error.message").String()
		if errType == "rate_limit_error" && strings.Contains(errMsg, "Concurrency limit exceeded") {
			return true
		}
	}

	// Pattern 2: top-level "type":"error" with stream_read_error code
	if gjson.GetBytes(payload, "type").String() == "error" {
		errCode := gjson.GetBytes(payload, "error.code").String()
		if errCode == "stream_read_error" {
			return true
		}
	}

	return false
}

// SSEConcurrencyRetryWrapper wraps an upstream SSE stream and transparently
// retries the request when the first event is a concurrency-limit error.
// Once real data starts flowing, it passes through without buffering.
type SSEConcurrencyRetryWrapper struct {
	upstream  io.ReadCloser
	retryFunc func() (io.ReadCloser, error)

	// internal state
	buf       []byte // raw bytes not yet consumed
	pending   []byte // transformed data ready to return to caller
	started   bool   // true once we've forwarded at least one real event
	exhausted bool   // true once upstream hit EOF
	retries   int
}

// NewSSEConcurrencyRetryWrapper creates a wrapper.
// retryFunc should make a fresh upstream request and return the new body.
func NewSSEConcurrencyRetryWrapper(upstream io.ReadCloser, retryFunc func() (io.ReadCloser, error)) io.ReadCloser {
	if retryFunc == nil {
		return upstream
	}
	return &SSEConcurrencyRetryWrapper{
		upstream:  upstream,
		retryFunc: retryFunc,
	}
}

func (w *SSEConcurrencyRetryWrapper) Read(p []byte) (int, error) {
	// Serve any pending transformed data first.
	if len(w.pending) > 0 {
		n := copy(p, w.pending)
		w.pending = w.pending[n:]
		return n, nil
	}

	for {
		// Read more from upstream.
		if !w.exhausted {
			tmp := make([]byte, 8*1024)
			n, err := w.upstream.Read(tmp)
			if n > 0 {
				w.buf = append(w.buf, tmp[:n]...)
			}
			if err == io.EOF {
				w.exhausted = true
			} else if err != nil {
				return 0, err
			}
		}

		// If we've already started forwarding, just pass through.
		if w.started {
			if len(w.buf) > 0 {
				n := copy(p, w.buf)
				w.buf = w.buf[n:]
				return n, nil
			}
			if w.exhausted {
				return 0, io.EOF
			}
			continue
		}

		// Not yet started — look for complete SSE frames.
		idx, delimLen := findSSEDelimiter(w.buf)
		if idx < 0 {
			if w.exhausted {
				// No complete frame, just forward whatever remains.
				w.started = true
				if len(w.buf) > 0 {
					n := copy(p, w.buf)
					w.buf = w.buf[n:]
					return n, nil
				}
				return 0, io.EOF
			}
			// Need more data.
			continue
		}

		frame := w.buf[:idx+delimLen]
		rest := w.buf[idx+delimLen:]

		if isSSERetryableError(frame) {
			w.retries++
			if w.retries > sseConcurrencyRetryMax {
				log.Warnf("sse-concurrency-retry: max retries (%d) exhausted, forwarding error to client", sseConcurrencyRetryMax)
				w.started = true
				// Forward the error frame.
				n := copy(p, w.buf)
				w.buf = w.buf[n:]
				return n, nil
			}

			wait := sseConcurrencyRetryBaseWait * time.Duration(w.retries)
			log.Warnf("sse-concurrency-retry: concurrency limit hit, retry %d/%d after %v", w.retries, sseConcurrencyRetryMax, wait)
			time.Sleep(wait)

			// Discard all buffered data and close old upstream.
			w.upstream.Close()
			w.buf = nil
			w.exhausted = false

			newBody, err := w.retryFunc()
			if err != nil {
				log.Errorf("sse-concurrency-retry: retry request failed: %v, forwarding original error", err)
				// Return the original error frame.
				w.started = true
				w.pending = append(frame, rest...)
				n := copy(p, w.pending)
				w.pending = w.pending[n:]
				return n, nil
			}
			w.upstream = newBody
			continue
		}

		// Not a concurrency error — this is real data, start forwarding.
		w.started = true
		n := copy(p, w.buf)
		w.buf = w.buf[n:]
		return n, nil
	}
}

func (w *SSEConcurrencyRetryWrapper) Close() error {
	return w.upstream.Close()
}

// isRetryableResponseBody reads the response body and checks if it contains
// a retryable error (concurrency limit exceeded or rate_limit_error).
// The body is consumed and cannot be read again after this call.
func isRetryableResponseBody(body io.ReadCloser) bool {
	data, err := io.ReadAll(io.LimitReader(body, 8*1024))
	if err != nil || len(data) == 0 {
		return false
	}
	s := string(data)

	// Check JSON error body: {"error":{"message":"Concurrency limit exceeded...","type":"rate_limit_error"}}
	errType := gjson.Get(s, "error.type").String()
	errMsg := gjson.Get(s, "error.message").String()
	if errType == "rate_limit_error" && strings.Contains(errMsg, "Concurrency limit exceeded") {
		return true
	}

	// Also match generic upstream_error / stream_read_error
	errCode := gjson.Get(s, "error.code").String()
	if errCode == "stream_read_error" {
		return true
	}

	return false
}
