// Package responses provides translation between OpenAI Chat Completions API and OpenAI Responses API.
// This enables transparent conversion between /v1/chat/completions and /v1/responses endpoints.
package responses

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertChatToResponsesRequest converts an OpenAI Chat Completions request to Responses API format.
// Chat Completions format: { model, messages, tools, ... }
// Responses format: { model, input, tools, ... }
func ConvertChatToResponsesRequest(modelName string, rawJSON []byte, stream bool) ([]byte, error) {
	result := gjson.ParseBytes(rawJSON)

	// Start building the Responses API request
	var output []byte
	var err error

	// Copy model
	if m := result.Get("model"); m.Exists() {
		output, err = sjson.SetBytes(output, "model", m.String())
		if err != nil {
			return rawJSON, err
		}
	} else if modelName != "" {
		output, err = sjson.SetBytes(output, "model", modelName)
		if err != nil {
			return rawJSON, err
		}
	}

	// Convert messages to input format
	messages := result.Get("messages")
	if messages.Exists() && messages.IsArray() {
		input, err := convertMessagesToInput(messages)
		if err != nil {
			return rawJSON, err
		}
		output, err = sjson.SetRawBytes(output, "input", input)
		if err != nil {
			return rawJSON, err
		}
	}

	// Copy tools (format is similar between both APIs)
	if tools := result.Get("tools"); tools.Exists() {
		output, err = sjson.SetRawBytes(output, "tools", []byte(tools.Raw))
		if err != nil {
			return rawJSON, err
		}
	}

	// Map common parameters
	if temp := result.Get("temperature"); temp.Exists() {
		output, _ = sjson.SetBytes(output, "temperature", temp.Float())
	}
	if topP := result.Get("top_p"); topP.Exists() {
		output, _ = sjson.SetBytes(output, "top_p", topP.Float())
	}
	if maxTokens := result.Get("max_tokens"); maxTokens.Exists() {
		output, _ = sjson.SetBytes(output, "max_output_tokens", maxTokens.Int())
	}
	if maxCompletionTokens := result.Get("max_completion_tokens"); maxCompletionTokens.Exists() {
		output, _ = sjson.SetBytes(output, "max_output_tokens", maxCompletionTokens.Int())
	}

	// Set stream
	output, _ = sjson.SetBytes(output, "stream", stream)

	// Copy tool_choice if present
	if tc := result.Get("tool_choice"); tc.Exists() {
		output, _ = sjson.SetRawBytes(output, "tool_choice", []byte(tc.Raw))
	}

	// Copy parallel_tool_calls if present
	if ptc := result.Get("parallel_tool_calls"); ptc.Exists() {
		output, _ = sjson.SetBytes(output, "parallel_tool_calls", ptc.Bool())
	}

	return output, nil
}

// ConvertResponsesToChatRequest converts an OpenAI Responses API request to Chat Completions format.
// Responses format: { model, input, tools, ... }
// Chat Completions format: { model, messages, tools, ... }
func ConvertResponsesToChatRequest(modelName string, rawJSON []byte, stream bool) ([]byte, error) {
	result := gjson.ParseBytes(rawJSON)

	var output []byte
	var err error

	// Copy model
	if m := result.Get("model"); m.Exists() {
		output, err = sjson.SetBytes(output, "model", m.String())
		if err != nil {
			return rawJSON, err
		}
	} else if modelName != "" {
		output, err = sjson.SetBytes(output, "model", modelName)
		if err != nil {
			return rawJSON, err
		}
	}

	// Convert input to messages format
	input := result.Get("input")
	if input.Exists() {
		messages, err := convertInputToMessages(input)
		if err != nil {
			return rawJSON, err
		}
		output, err = sjson.SetRawBytes(output, "messages", messages)
		if err != nil {
			return rawJSON, err
		}
	}

	// Copy tools (format is similar)
	if tools := result.Get("tools"); tools.Exists() {
		output, _ = sjson.SetRawBytes(output, "tools", []byte(tools.Raw))
	}

	// Map parameters
	if temp := result.Get("temperature"); temp.Exists() {
		output, _ = sjson.SetBytes(output, "temperature", temp.Float())
	}
	if topP := result.Get("top_p"); topP.Exists() {
		output, _ = sjson.SetBytes(output, "top_p", topP.Float())
	}
	if maxOutputTokens := result.Get("max_output_tokens"); maxOutputTokens.Exists() {
		output, _ = sjson.SetBytes(output, "max_tokens", maxOutputTokens.Int())
	}

	// Set stream
	output, _ = sjson.SetBytes(output, "stream", stream)

	// Copy tool_choice if present
	if tc := result.Get("tool_choice"); tc.Exists() {
		output, _ = sjson.SetRawBytes(output, "tool_choice", []byte(tc.Raw))
	}

	// Copy parallel_tool_calls if present
	if ptc := result.Get("parallel_tool_calls"); ptc.Exists() {
		output, _ = sjson.SetBytes(output, "parallel_tool_calls", ptc.Bool())
	}

	return output, nil
}

// convertMessagesToInput converts Chat Completions messages array to Responses API input format
func convertMessagesToInput(messages gjson.Result) ([]byte, error) {
	var inputItems []map[string]interface{}

	for _, msg := range messages.Array() {
		role := msg.Get("role").String()
		content := msg.Get("content")

		switch role {
		case "system":
			// System messages become system input items
			item := map[string]interface{}{
				"type": "message",
				"role": "system",
			}
			if content.Type == gjson.String {
				item["content"] = []map[string]interface{}{
					{"type": "input_text", "text": content.String()},
				}
			} else if content.IsArray() {
				item["content"] = convertContentArray(content)
			}
			inputItems = append(inputItems, item)

		case "user":
			item := map[string]interface{}{
				"type": "message",
				"role": "user",
			}
			if content.Type == gjson.String {
				item["content"] = []map[string]interface{}{
					{"type": "input_text", "text": content.String()},
				}
			} else if content.IsArray() {
				item["content"] = convertContentArray(content)
			}
			inputItems = append(inputItems, item)

		case "assistant":
			item := map[string]interface{}{
				"type": "message",
				"role": "assistant",
			}

			var contentParts []map[string]interface{}
			if content.Type == gjson.String {
				contentParts = append(contentParts, map[string]interface{}{
					"type": "output_text",
					"text": content.String(),
				})
			} else if content.IsArray() {
				for _, c := range content.Array() {
					if c.Get("type").String() == "text" {
						contentParts = append(contentParts, map[string]interface{}{
							"type": "output_text",
							"text": c.Get("text").String(),
						})
					}
				}
			}

			// Handle tool_calls
			toolCalls := msg.Get("tool_calls")
			if toolCalls.Exists() && toolCalls.IsArray() {
				for _, tc := range toolCalls.Array() {
					fcItem := map[string]interface{}{
						"type":      "function_call",
						"id":        tc.Get("id").String(),
						"name":      tc.Get("function.name").String(),
						"arguments": tc.Get("function.arguments").String(),
					}
					contentParts = append(contentParts, fcItem)
				}
			}

			if len(contentParts) > 0 {
				item["content"] = contentParts
			}
			inputItems = append(inputItems, item)

		case "tool":
			// Tool response messages
			item := map[string]interface{}{
				"type":    "function_call_output",
				"call_id": msg.Get("tool_call_id").String(),
				"output":  content.String(),
			}
			inputItems = append(inputItems, item)
		}
	}

	return json.Marshal(inputItems)
}

// convertInputToMessages converts Responses API input to Chat Completions messages format
func convertInputToMessages(input gjson.Result) ([]byte, error) {
	var messages []map[string]interface{}

	// Handle string input
	if input.Type == gjson.String {
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": input.String(),
		})
		return json.Marshal(messages)
	}

	// Handle array input
	if !input.IsArray() {
		return json.Marshal(messages)
	}

	for _, item := range input.Array() {
		itemType := item.Get("type").String()

		switch itemType {
		case "message":
			role := item.Get("role").String()
			content := item.Get("content")

			msg := map[string]interface{}{
				"role": role,
			}

			var textParts []string
			var toolCalls []map[string]interface{}

			if content.IsArray() {
				for _, c := range content.Array() {
					cType := c.Get("type").String()
					switch cType {
					case "input_text", "output_text", "text":
						textParts = append(textParts, c.Get("text").String())
					case "function_call":
						toolCalls = append(toolCalls, map[string]interface{}{
							"id":   c.Get("id").String(),
							"type": "function",
							"function": map[string]interface{}{
								"name":      c.Get("name").String(),
								"arguments": c.Get("arguments").String(),
							},
						})
					}
				}
			}

			if len(textParts) > 0 {
				// Join all text parts
				fullText := ""
				for i, t := range textParts {
					if i > 0 {
						fullText += "\n"
					}
					fullText += t
				}
				msg["content"] = fullText
			}

			if len(toolCalls) > 0 {
				msg["tool_calls"] = toolCalls
			}

			messages = append(messages, msg)

		case "function_call_output":
			msg := map[string]interface{}{
				"role":         "tool",
				"tool_call_id": item.Get("call_id").String(),
				"content":      item.Get("output").String(),
			}
			messages = append(messages, msg)
		}
	}

	return json.Marshal(messages)
}

// convertContentArray converts a Chat Completions content array to Responses format
func convertContentArray(content gjson.Result) []map[string]interface{} {
	var result []map[string]interface{}
	for _, c := range content.Array() {
		cType := c.Get("type").String()
		switch cType {
		case "text":
			result = append(result, map[string]interface{}{
				"type": "input_text",
				"text": c.Get("text").String(),
			})
		case "image_url":
			result = append(result, map[string]interface{}{
				"type": "input_image",
				"url":  c.Get("image_url.url").String(),
			})
		}
	}
	return result
}
