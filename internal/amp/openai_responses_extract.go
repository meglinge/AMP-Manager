package amp

import (
	"strings"

	"github.com/tidwall/gjson"
)

const maxResponseTextBytes = 16 * 1024 // stored in request_logs.response_text

// ExtractOpenAIResponsesOutputText extracts assistant-visible output text from an OpenAI Responses response object.
//
// It returns an empty string if the payload is not a Responses object.
func ExtractOpenAIResponsesOutputText(body []byte) string {
	root := gjson.ParseBytes(body)
	if root.Get("object").String() != "response" {
		return ""
	}

	out := root.Get("output")
	if !out.Exists() || !out.IsArray() {
		return ""
	}

	var b strings.Builder
	for _, item := range out.Array() {
		if item.Get("type").String() != "message" {
			continue
		}
		if role := item.Get("role").String(); role != "assistant" {
			continue
		}
		content := item.Get("content")
		if !content.Exists() || !content.IsArray() {
			continue
		}
		for _, part := range content.Array() {
			if part.Get("type").String() != "output_text" {
				continue
			}
			txt := part.Get("text").String()
			if txt == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(txt)
			if b.Len() >= maxResponseTextBytes {
				return b.String()[:maxResponseTextBytes]
			}
		}
	}

	return b.String()
}
