// Package responses provides translation between OpenAI Chat Completions API and OpenAI Responses API.
// This file handles response translation.
package responses

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertResponsesStreamToChat converts OpenAI Responses API streaming events to Chat Completions format.
func ConvertResponsesStreamToChat(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) ([]string, error) {
	text := string(rawJSON)
	var results []string

	// Process each SSE line
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				results = append(results, "data: [DONE]\n\n")
				continue
			}

			converted, err := convertResponsesEventToChat(data, model)
			if err != nil {
				results = append(results, line+"\n")
				continue
			}
			if converted != "" {
				results = append(results, "data: "+converted+"\n\n")
			}
		} else if strings.HasPrefix(line, "event: ") {
			// Skip event type lines, we only care about data
			continue
		} else {
			results = append(results, line+"\n")
		}
	}

	return results, nil
}

// ConvertResponsesNonStreamToChat converts a non-streaming Responses API response to Chat Completions format.
func ConvertResponsesNonStreamToChat(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) (string, error) {
	result := gjson.ParseBytes(rawJSON)

	// Build Chat Completions response
	var output []byte

	// Copy id
	if id := result.Get("id"); id.Exists() {
		output, _ = sjson.SetBytes(output, "id", id.String())
	}

	output, _ = sjson.SetBytes(output, "object", "chat.completion")

	if created := result.Get("created_at"); created.Exists() {
		// Parse ISO timestamp to unix
		output, _ = sjson.SetBytes(output, "created", created.Int())
	}

	output, _ = sjson.SetBytes(output, "model", model)

	// Convert output to choices
	outputArr := result.Get("output")
	if outputArr.Exists() && outputArr.IsArray() {
		choices := convertOutputToChoices(outputArr)
		choicesJSON, _ := json.Marshal(choices)
		output, _ = sjson.SetRawBytes(output, "choices", choicesJSON)
	}

	// Copy usage
	if usage := result.Get("usage"); usage.Exists() {
		usageMap := map[string]interface{}{
			"prompt_tokens":     usage.Get("input_tokens").Int(),
			"completion_tokens": usage.Get("output_tokens").Int(),
			"total_tokens":      usage.Get("input_tokens").Int() + usage.Get("output_tokens").Int(),
		}
		usageJSON, _ := json.Marshal(usageMap)
		output, _ = sjson.SetRawBytes(output, "usage", usageJSON)
	}

	return string(output), nil
}

// ConvertChatStreamToResponses converts Chat Completions streaming events to Responses API format.
func ConvertChatStreamToResponses(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) ([]string, error) {
	text := string(rawJSON)
	var results []string

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				results = append(results, "data: [DONE]\n\n")
				continue
			}

			converted, err := convertChatEventToResponses(data, model)
			if err != nil {
				results = append(results, line+"\n")
				continue
			}
			if converted != "" {
				results = append(results, "data: "+converted+"\n\n")
			}
		} else if strings.HasPrefix(line, "event: ") {
			continue
		} else {
			results = append(results, line+"\n")
		}
	}

	return results, nil
}

// ConvertChatNonStreamToResponses converts a non-streaming Chat Completions response to Responses API format.
func ConvertChatNonStreamToResponses(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) (string, error) {
	result := gjson.ParseBytes(rawJSON)

	var output []byte

	// Copy id
	if id := result.Get("id"); id.Exists() {
		output, _ = sjson.SetBytes(output, "id", id.String())
	}

	output, _ = sjson.SetBytes(output, "object", "response")

	if created := result.Get("created"); created.Exists() {
		output, _ = sjson.SetBytes(output, "created_at", created.Int())
	}

	output, _ = sjson.SetBytes(output, "model", model)
	output, _ = sjson.SetBytes(output, "status", "completed")

	// Convert choices to output
	choices := result.Get("choices")
	if choices.Exists() && choices.IsArray() {
		outputItems := convertChoicesToOutput(choices)
		outputJSON, _ := json.Marshal(outputItems)
		output, _ = sjson.SetRawBytes(output, "output", outputJSON)
	}

	// Convert usage
	if usage := result.Get("usage"); usage.Exists() {
		usageMap := map[string]interface{}{
			"input_tokens":  usage.Get("prompt_tokens").Int(),
			"output_tokens": usage.Get("completion_tokens").Int(),
			"total_tokens":  usage.Get("total_tokens").Int(),
		}
		usageJSON, _ := json.Marshal(usageMap)
		output, _ = sjson.SetRawBytes(output, "usage", usageJSON)
	}

	return string(output), nil
}

// convertResponsesEventToChat converts a single Responses API event to Chat Completions format
func convertResponsesEventToChat(data string, model string) (string, error) {
	result := gjson.Parse(data)

	eventType := result.Get("type").String()

	switch eventType {
	case "response.created", "response.in_progress":
		// Skip these events
		return "", nil

	case "response.output_item.added":
		// Skip, we'll handle the actual content
		return "", nil

	case "response.content_part.added":
		return "", nil

	case "response.output_text.delta":
		// Text delta
		delta := result.Get("delta").String()
		chunk := map[string]interface{}{
			"id":      result.Get("response_id").String(),
			"object":  "chat.completion.chunk",
			"created": 0,
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index": result.Get("output_index").Int(),
					"delta": map[string]interface{}{
						"content": delta,
					},
					"finish_reason": nil,
				},
			},
		}
		chunkJSON, _ := json.Marshal(chunk)
		return string(chunkJSON), nil

	case "response.function_call_arguments.delta":
		// Function call arguments delta
		delta := result.Get("delta").String()
		chunk := map[string]interface{}{
			"id":      result.Get("response_id").String(),
			"object":  "chat.completion.chunk",
			"created": 0,
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index": result.Get("output_index").Int(),
					"delta": map[string]interface{}{
						"tool_calls": []map[string]interface{}{
							{
								"index": 0,
								"function": map[string]interface{}{
									"arguments": delta,
								},
							},
						},
					},
					"finish_reason": nil,
				},
			},
		}
		chunkJSON, _ := json.Marshal(chunk)
		return string(chunkJSON), nil

	case "response.output_item.done":
		item := result.Get("item")
		itemType := item.Get("type").String()

		if itemType == "function_call" {
			// Complete function call
			chunk := map[string]interface{}{
				"id":      result.Get("response_id").String(),
				"object":  "chat.completion.chunk",
				"created": 0,
				"model":   model,
				"choices": []map[string]interface{}{
					{
						"index": result.Get("output_index").Int(),
						"delta": map[string]interface{}{
							"tool_calls": []map[string]interface{}{
								{
									"index": 0,
									"id":    item.Get("id").String(),
									"type":  "function",
									"function": map[string]interface{}{
										"name":      item.Get("name").String(),
										"arguments": item.Get("arguments").String(),
									},
								},
							},
						},
						"finish_reason": nil,
					},
				},
			}
			chunkJSON, _ := json.Marshal(chunk)
			return string(chunkJSON), nil
		}
		return "", nil

	case "response.completed", "response.done":
		// Final event with finish reason
		response := result.Get("response")
		finishReason := "stop"
		if sr := response.Get("stop_reason"); sr.Exists() {
			if sr.String() == "tool_use" {
				finishReason = "tool_calls"
			}
		}

		chunk := map[string]interface{}{
			"id":      response.Get("id").String(),
			"object":  "chat.completion.chunk",
			"created": 0,
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": finishReason,
				},
			},
		}

		// Include usage if available
		if usage := response.Get("usage"); usage.Exists() {
			chunk["usage"] = map[string]interface{}{
				"prompt_tokens":     usage.Get("input_tokens").Int(),
				"completion_tokens": usage.Get("output_tokens").Int(),
				"total_tokens":      usage.Get("input_tokens").Int() + usage.Get("output_tokens").Int(),
			}
		}

		chunkJSON, _ := json.Marshal(chunk)
		return string(chunkJSON), nil

	default:
		// Pass through unknown events
		return "", nil
	}
}

// convertChatEventToResponses converts a single Chat Completions event to Responses API format
func convertChatEventToResponses(data string, model string) (string, error) {
	result := gjson.Parse(data)

	choices := result.Get("choices")
	if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
		return "", nil
	}

	choice := choices.Array()[0]
	delta := choice.Get("delta")
	finishReason := choice.Get("finish_reason")

	// Handle content delta
	if content := delta.Get("content"); content.Exists() && content.String() != "" {
		event := map[string]interface{}{
			"type":         "response.output_text.delta",
			"response_id":  result.Get("id").String(),
			"output_index": choice.Get("index").Int(),
			"delta":        content.String(),
		}
		eventJSON, _ := json.Marshal(event)
		return string(eventJSON), nil
	}

	// Handle tool calls delta
	if toolCalls := delta.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
		for _, tc := range toolCalls.Array() {
			if args := tc.Get("function.arguments"); args.Exists() && args.String() != "" {
				event := map[string]interface{}{
					"type":         "response.function_call_arguments.delta",
					"response_id":  result.Get("id").String(),
					"output_index": choice.Get("index").Int(),
					"delta":        args.String(),
				}
				eventJSON, _ := json.Marshal(event)
				return string(eventJSON), nil
			}
			if name := tc.Get("function.name"); name.Exists() && name.String() != "" {
				// Function call started
				event := map[string]interface{}{
					"type":         "response.output_item.added",
					"response_id":  result.Get("id").String(),
					"output_index": choice.Get("index").Int(),
					"item": map[string]interface{}{
						"type": "function_call",
						"id":   tc.Get("id").String(),
						"name": name.String(),
					},
				}
				eventJSON, _ := json.Marshal(event)
				return string(eventJSON), nil
			}
		}
	}

	// Handle finish reason
	if finishReason.Exists() && finishReason.String() != "" {
		status := "completed"
		stopReason := "end_turn"
		if finishReason.String() == "tool_calls" {
			stopReason = "tool_use"
		} else if finishReason.String() == "length" {
			stopReason = "max_tokens"
		}

		event := map[string]interface{}{
			"type": "response.completed",
			"response": map[string]interface{}{
				"id":          result.Get("id").String(),
				"model":       model,
				"status":      status,
				"stop_reason": stopReason,
			},
		}

		// Include usage if available
		if usage := result.Get("usage"); usage.Exists() {
			event["response"].(map[string]interface{})["usage"] = map[string]interface{}{
				"input_tokens":  usage.Get("prompt_tokens").Int(),
				"output_tokens": usage.Get("completion_tokens").Int(),
			}
		}

		eventJSON, _ := json.Marshal(event)
		return string(eventJSON), nil
	}

	return "", nil
}

// convertOutputToChoices converts Responses API output array to Chat Completions choices
func convertOutputToChoices(output gjson.Result) []map[string]interface{} {
	var choices []map[string]interface{}

	var textContent string
	var toolCalls []map[string]interface{}

	for _, item := range output.Array() {
		itemType := item.Get("type").String()

		switch itemType {
		case "message":
			content := item.Get("content")
			if content.IsArray() {
				for _, c := range content.Array() {
					cType := c.Get("type").String()
					if cType == "output_text" || cType == "text" {
						textContent += c.Get("text").String()
					}
				}
			}

		case "function_call":
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   item.Get("id").String(),
				"type": "function",
				"function": map[string]interface{}{
					"name":      item.Get("name").String(),
					"arguments": item.Get("arguments").String(),
				},
			})
		}
	}

	choice := map[string]interface{}{
		"index": 0,
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": textContent,
		},
		"finish_reason": "stop",
	}

	if len(toolCalls) > 0 {
		choice["message"].(map[string]interface{})["tool_calls"] = toolCalls
		choice["finish_reason"] = "tool_calls"
	}

	choices = append(choices, choice)
	return choices
}

// convertChoicesToOutput converts Chat Completions choices to Responses API output
func convertChoicesToOutput(choices gjson.Result) []map[string]interface{} {
	var output []map[string]interface{}

	for _, choice := range choices.Array() {
		msg := choice.Get("message")

		// Handle text content
		if content := msg.Get("content"); content.Exists() && content.String() != "" {
			output = append(output, map[string]interface{}{
				"type": "message",
				"role": "assistant",
				"content": []map[string]interface{}{
					{
						"type": "output_text",
						"text": content.String(),
					},
				},
			})
		}

		// Handle tool calls
		if toolCalls := msg.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
			for _, tc := range toolCalls.Array() {
				output = append(output, map[string]interface{}{
					"type":      "function_call",
					"id":        tc.Get("id").String(),
					"name":      tc.Get("function.name").String(),
					"arguments": tc.Get("function.arguments").String(),
				})
			}
		}
	}

	return output
}
