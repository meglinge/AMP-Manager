package amp

import (
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	sseConcurrencyRetryMax = 3
)

var (
	sseConcurrencyRetryBaseWait = 2 * time.Second
	sseConcurrencyRetrySleep    = time.Sleep
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
	return isRetryableErrorPayload(eventName, payload)
}

func isRetryableErrorPayload(eventName string, payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	topType := strings.ToLower(gjson.GetBytes(payload, "type").String())

	// Pattern 1: event: error + rate_limit_error with concurrency limit
	if eventName == "error" {
		errType := gjson.GetBytes(payload, "error.type").String()
		errMsg := strings.ToLower(gjson.GetBytes(payload, "error.message").String())
		if errType == "rate_limit_error" && strings.Contains(errMsg, "concurrency limit exceeded") {
			return true
		}
	}

	// Pattern 2: stream_read_error (code/message), usually with top-level type:"error".
	errCode := strings.ToLower(gjson.GetBytes(payload, "error.code").String())
	errMsg := strings.ToLower(gjson.GetBytes(payload, "error.message").String())
	if errCode == "stream_read_error" || strings.Contains(errMsg, "stream_read_error") {
		if topType == "" || topType == "error" || eventName == "error" {
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
	staged    []byte // prelude frames buffered until retry/no-retry decision
	pending   []byte // transformed data ready to return to caller
	started   bool   // true once we've forwarded at least one real event
	exhausted bool   // true once upstream hit EOF
	retries   int
}

func isRetryPreludeFrame(eventName string, payload []byte) bool {
	if len(payload) == 0 {
		return false
	}

	eventType := strings.ToLower(gjson.GetBytes(payload, "type").String())
	if eventType == "" {
		eventType = strings.ToLower(eventName)
	}

	switch eventType {
	case "response.created", "response.in_progress", "response.output_item.added", "response.output_item.done", "response.content_part.added", "response.content_part.done":
		return true
	default:
		return false
	}
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
				if len(w.staged) > 0 {
					w.pending = append(w.pending, w.staged...)
					w.staged = nil
				}
				if len(w.buf) > 0 {
					w.pending = append(w.pending, w.buf...)
					w.buf = nil
				}
				if len(w.pending) > 0 {
					n := copy(p, w.pending)
					w.pending = w.pending[n:]
					return n, nil
				}
				return 0, io.EOF
			}
			// Need more data.
			continue
		}

		frame := w.buf[:idx+delimLen]
		rest := w.buf[idx+delimLen:]
		eventName, payload, done := parseSSEEvent(frame)

		if len(payload) == 0 && !done {
			// Keep-alive/comment frame (e.g. ":\n\n") during prelude.
			// Buffer it so we can discard if this attempt is retried.
			w.staged = append(w.staged, frame...)
			w.buf = rest
			continue
		}

		if isRetryableErrorPayload(eventName, payload) {
			w.retries++
			log.Warnf("sse-concurrency-retry: detected retryable SSE error in frame (attempt %d), frame=%s", w.retries, string(frame))
			if w.retries > sseConcurrencyRetryMax {
				log.Warnf("sse-concurrency-retry: max retries (%d) exhausted, forwarding error to client", sseConcurrencyRetryMax)
				w.started = true
				w.pending = append(w.pending, w.staged...)
				w.staged = nil
				w.pending = append(w.pending, frame...)
				w.pending = append(w.pending, rest...)
				w.buf = nil
				n := copy(p, w.pending)
				w.pending = w.pending[n:]
				return n, nil
			}

			wait := sseConcurrencyRetryBaseWait * time.Duration(w.retries)
			log.Warnf("sse-concurrency-retry: concurrency limit hit, retry %d/%d after %v", w.retries, sseConcurrencyRetryMax, wait)
			sseConcurrencyRetrySleep(wait)

			// Discard all buffered data and close old upstream.
			w.upstream.Close()
			w.buf = nil
			w.exhausted = false

			newBody, err := w.retryFunc()
			if err != nil {
				log.Errorf("sse-concurrency-retry: retry request failed: %v, forwarding original error", err)
				// Return the original stream prelude + error frame.
				w.started = true
				w.pending = append(w.pending, w.staged...)
				w.staged = nil
				w.pending = append(w.pending, frame...)
				w.pending = append(w.pending, rest...)
				n := copy(p, w.pending)
				w.pending = w.pending[n:]
				return n, nil
			}
			w.staged = nil
			w.upstream = newBody
			continue
		}

		if isRetryPreludeFrame(eventName, payload) {
			// Responses API prelude events may be followed by a retryable error.
			// Buffer until we either hit retryable error or first substantive data.
			w.staged = append(w.staged, frame...)
			w.buf = rest
			continue
		}

		// Not a retryable error — this is real stream data (or [DONE]), start forwarding.
		log.Debugf("sse-concurrency-retry: first data frame is not retryable, passing through (len=%d)", len(frame))
		w.started = true
		w.pending = append(w.pending, w.staged...)
		w.staged = nil
		w.pending = append(w.pending, w.buf...)
		w.buf = nil
		n := copy(p, w.pending)
		w.pending = w.pending[n:]
		return n, nil
	}
}

func (w *SSEConcurrencyRetryWrapper) Close() error {
	return w.upstream.Close()
}

// isRetryableBytes checks if a response body (as bytes) contains
// a retryable error (concurrency limit exceeded, rate_limit_error, or stream_read_error).
func isRetryableBytes(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	s := string(data)

	// Check JSON error body: {"error":{"message":"Concurrency limit exceeded...","type":"rate_limit_error"}}
	errType := strings.ToLower(gjson.Get(s, "error.type").String())
	errMsg := strings.ToLower(gjson.Get(s, "error.message").String())
	if errType == "rate_limit_error" && strings.Contains(errMsg, "concurrency limit exceeded") {
		return true
	}

	// Also match generic upstream_error / stream_read_error
	errCode := strings.ToLower(gjson.Get(s, "error.code").String())
	errMsg = strings.ToLower(gjson.Get(s, "error.message").String())
	if errCode == "stream_read_error" || strings.Contains(errMsg, "stream_read_error") {
		return true
	}

	return false
}
