package amp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/tidwall/gjson"
)

const maxResponsesSSEAggregateBytes = 50 * 1024 * 1024 // 50MB

// aggregateOpenAIResponsesSSEToJSON consumes an OpenAI Responses SSE stream and returns a single non-stream
// Responses JSON body (object: "response").
//
// It prefers the embedded "response" object from response.completed/response.done events.
func aggregateOpenAIResponsesSSEToJSON(ctx context.Context, r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	var sseBuffer bytes.Buffer
	var totalRead int64
	var finalResponseRaw string

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tmp := make([]byte, 4096)
		n, err := r.Read(tmp)
		if n > 0 {
			totalRead += int64(n)
			if totalRead > maxResponsesSSEAggregateBytes {
				return nil, fmt.Errorf("responses sse aggregate: exceeded max bytes (%d)", maxResponsesSSEAggregateBytes)
			}
			sseBuffer.Write(tmp[:n])
		}

		for {
			data := sseBuffer.Bytes()
			idx, delimLen := findSSEDelimiter(data)
			if idx == -1 {
				break
			}

			event := make([]byte, idx+delimLen)
			copy(event, data[:idx+delimLen])
			sseBuffer.Reset()
			sseBuffer.Write(data[idx+delimLen:])

			_, payload, done := parseSSEEvent(event)
			if done {
				// Stop consuming once [DONE] is received.
				goto FINISH
			}
			if len(payload) == 0 {
				continue
			}

			// Keep the latest full response snapshot if present.
			// Common shape: {"type":"...","response":{...}} (including response.completed).
			if resp := gjson.GetBytes(payload, "response"); resp.Exists() && resp.IsObject() {
				finalResponseRaw = resp.Raw
				continue
			}

			// Some providers may send the response object directly.
			if gjson.GetBytes(payload, "object").String() == "response" {
				finalResponseRaw = string(bytes.TrimSpace(payload))
				continue
			}

			// Keep a copy of raw events for debugging if we fail to find a final response.
			buf.Write(payload)
			buf.WriteByte('\n')
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

FINISH:
	if strings.TrimSpace(finalResponseRaw) == "" {
		return nil, fmt.Errorf("responses sse aggregate: missing final response.completed event")
	}
	return []byte(finalResponseRaw), nil
}

func parseSSEEvent(event []byte) (eventName string, payload []byte, done bool) {
	// SSE event is a sequence of lines terminated by a blank line.
	lines := bytes.Split(event, []byte("\n"))
	var dataLines [][]byte
	for _, line := range lines {
		line = bytes.TrimRight(line, "\r")
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("event:")) {
			eventName = strings.TrimSpace(string(bytes.TrimPrefix(trimmed, []byte("event:"))))
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("data:")) {
			data := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
			if bytes.Equal(data, []byte("[DONE]")) {
				return eventName, nil, true
			}
			if len(data) > 0 {
				dataLines = append(dataLines, data)
			}
		}
	}
	if len(dataLines) == 0 {
		return eventName, nil, false
	}
	// SSE spec: multiple data: lines are joined with \n.
	payload = bytes.Join(dataLines, []byte("\n"))
	return eventName, payload, false
}
