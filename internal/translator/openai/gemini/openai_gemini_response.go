// Package gemini provides response translation functionality for OpenAI to Gemini API.
// This package handles the conversion of OpenAI Chat Completions API responses into Gemini API-compatible
// JSON format, transforming streaming events and non-streaming responses into the format
// expected by Gemini API clients. It supports both streaming and non-streaming modes,
// handling text content, tool calls, and usage metadata appropriately.
package gemini

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertOpenAIResponseToGeminiParams holds parameters for response conversion
type ConvertOpenAIResponseToGeminiParams struct {
	// Tool calls accumulator for streaming
	ToolCallsAccumulator map[int]*ToolCallAccumulator
	// Content accumulator for streaming
	ContentAccumulator strings.Builder
	// Track if this is the first chunk
	IsFirstChunk bool
}

// ToolCallAccumulator holds the state for accumulating tool call data
type ToolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// ConvertOpenAIResponseToGemini converts OpenAI Chat Completions streaming response format to Gemini API format.
// This function processes OpenAI streaming chunks and transforms them into Gemini-compatible JSON responses.
// It handles text content, tool calls, and usage metadata, outputting responses that match the Gemini API format.
//
// Parameters:
//   - ctx: The context for the request.
//   - modelName: The name of the model.
//   - rawJSON: The raw JSON response from the OpenAI API.
//   - param: A pointer to a parameter object for the conversion.
//
// Returns:
//   - []string: A slice of strings, each containing a Gemini-compatible JSON response.
func ConvertOpenAIResponseToGemini(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) ([]string, error) {
	if *param == nil {
		*param = &ConvertOpenAIResponseToGeminiParams{
			ToolCallsAccumulator: nil,
			ContentAccumulator:   strings.Builder{},
			IsFirstChunk:         false,
		}
	}

	// Handle [DONE] marker
	if strings.TrimSpace(string(rawJSON)) == "[DONE]" {
		return []string{}, nil
	}

	if bytes.HasPrefix(rawJSON, []byte("data:")) {
		rawJSON = bytes.TrimSpace(rawJSON[5:])
	}

	root := gjson.ParseBytes(rawJSON)

	// Initialize accumulators if needed
	if (*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator == nil {
		(*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator = make(map[int]*ToolCallAccumulator)
	}

	// Process choices
	if choices := root.Get("choices"); choices.Exists() && choices.IsArray() {
		// Handle empty choices array (usage-only chunk)
		if len(choices.Array()) == 0 {
			// This is a usage-only chunk, handle usage and return
			if usage := root.Get("usage"); usage.Exists() {
				template := `{"candidates":[],"usageMetadata":{}}`

				// Set model if available
				if model := root.Get("model"); model.Exists() {
					template, _ = sjson.Set(template, "model", model.String())
				}

				template, _ = sjson.Set(template, "usageMetadata.promptTokenCount", usage.Get("prompt_tokens").Int())
				template, _ = sjson.Set(template, "usageMetadata.candidatesTokenCount", usage.Get("completion_tokens").Int())
				template, _ = sjson.Set(template, "usageMetadata.totalTokenCount", usage.Get("total_tokens").Int())
				if reasoningTokens := reasoningTokensFromUsage(usage); reasoningTokens > 0 {
					template, _ = sjson.Set(template, "usageMetadata.thoughtsTokenCount", reasoningTokens)
				}
				return []string{template}, nil
			}
			return []string{}, nil
		}

		var results []string

		choices.ForEach(func(choiceIndex, choice gjson.Result) bool {
			// Base Gemini response template without finishReason; set when known
			template := `{"candidates":[{"content":{"parts":[],"role":"model"},"index":0}]}`

			// Set model if available
			if model := root.Get("model"); model.Exists() {
				template, _ = sjson.Set(template, "model", model.String())
			}

			_ = int(choice.Get("index").Int()) // choiceIdx not used in streaming
			delta := choice.Get("delta")
			baseTemplate := template

			// Handle role (only in first chunk)
			if role := delta.Get("role"); role.Exists() && (*param).(*ConvertOpenAIResponseToGeminiParams).IsFirstChunk {
				// OpenAI assistant -> Gemini model
				if role.String() == "assistant" {
					template, _ = sjson.Set(template, "candidates.0.content.role", "model")
				}
				(*param).(*ConvertOpenAIResponseToGeminiParams).IsFirstChunk = false
				results = append(results, template)
				return true
			}

			var chunkOutputs []string

			// Handle reasoning/thinking delta
			if reasoning := delta.Get("reasoning_content"); reasoning.Exists() {
				for _, reasoningText := range extractReasoningTexts(reasoning) {
					if reasoningText == "" {
						continue
					}
					reasoningTemplate := baseTemplate
					reasoningTemplate, _ = sjson.Set(reasoningTemplate, "candidates.0.content.parts.0.thought", true)
					reasoningTemplate, _ = sjson.Set(reasoningTemplate, "candidates.0.content.parts.0.text", reasoningText)
					chunkOutputs = append(chunkOutputs, reasoningTemplate)
				}
			}

			// Handle content delta
			if content := delta.Get("content"); content.Exists() && content.String() != "" {
				contentText := content.String()
				(*param).(*ConvertOpenAIResponseToGeminiParams).ContentAccumulator.WriteString(contentText)

				// Create text part for this delta
				contentTemplate := baseTemplate
				contentTemplate, _ = sjson.Set(contentTemplate, "candidates.0.content.parts.0.text", contentText)
				chunkOutputs = append(chunkOutputs, contentTemplate)
			}

			if len(chunkOutputs) > 0 {
				results = append(results, chunkOutputs...)
				return true
			}

			// Handle tool calls delta
			if toolCalls := delta.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
				toolCalls.ForEach(func(_, toolCall gjson.Result) bool {
					toolIndex := int(toolCall.Get("index").Int())
					toolID := toolCall.Get("id").String()
					toolType := toolCall.Get("type").String()
					function := toolCall.Get("function")

					// Skip non-function tool calls explicitly marked as other types.
					if toolType != "" && toolType != "function" {
						return true
					}

					// OpenAI streaming deltas may omit the type field while still carrying function data.
					if !function.Exists() {
						return true
					}

					functionName := function.Get("name").String()
					functionArgs := function.Get("arguments").String()

					// Initialize accumulator if needed so later deltas without type can append arguments.
					if _, exists := (*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator[toolIndex]; !exists {
						(*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator[toolIndex] = &ToolCallAccumulator{
							ID:   toolID,
							Name: functionName,
						}
					}

					acc := (*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator[toolIndex]

					// Update ID if provided
					if toolID != "" {
						acc.ID = toolID
					}

					// Update name if provided
					if functionName != "" {
						acc.Name = functionName
					}

					// Accumulate arguments
					if functionArgs != "" {
						acc.Arguments.WriteString(functionArgs)
					}

					return true
				})

				// Don't output anything for tool call deltas - wait for completion
				return true
			}

			// Handle finish reason
			if finishReason := choice.Get("finish_reason"); finishReason.Exists() {
				geminiFinishReason := mapOpenAIFinishReasonToGemini(finishReason.String())
				template, _ = sjson.Set(template, "candidates.0.finishReason", geminiFinishReason)

				// If we have accumulated tool calls, output them now
				if len((*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator) > 0 {
					partIndex := 0
					for _, accumulator := range (*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator {
						namePath := fmt.Sprintf("candidates.0.content.parts.%d.functionCall.name", partIndex)
						argsPath := fmt.Sprintf("candidates.0.content.parts.%d.functionCall.args", partIndex)
						template, _ = sjson.Set(template, namePath, accumulator.Name)
						template, _ = sjson.SetRaw(template, argsPath, parseArgsToObjectRaw(accumulator.Arguments.String()))
						partIndex++
					}

					// Clear accumulators
					(*param).(*ConvertOpenAIResponseToGeminiParams).ToolCallsAccumulator = make(map[int]*ToolCallAccumulator)
				}

				results = append(results, template)
				return true
			}

			// Handle usage information
			if usage := root.Get("usage"); usage.Exists() {
				template, _ = sjson.Set(template, "usageMetadata.promptTokenCount", usage.Get("prompt_tokens").Int())
				template, _ = sjson.Set(template, "usageMetadata.candidatesTokenCount", usage.Get("completion_tokens").Int())
				template, _ = sjson.Set(template, "usageMetadata.totalTokenCount", usage.Get("total_tokens").Int())
				if reasoningTokens := reasoningTokensFromUsage(usage); reasoningTokens > 0 {
					template, _ = sjson.Set(template, "usageMetadata.thoughtsTokenCount", reasoningTokens)
				}
				results = append(results, template)
				return true
			}

			return true
		})
		return results, nil
	}
	return []string{}, nil
}

// mapOpenAIFinishReasonToGemini maps OpenAI finish reasons to Gemini finish reasons
func mapOpenAIFinishReasonToGemini(reason string) string {
	switch reason {
	case "stop":
		return "STOP"
	case "length":
		return "MAX_TOKENS"
	case "tool_calls", "function_call":
		return "STOP"
	case "content_filter":
		return "SAFETY"
	default:
		return "STOP"
	}
}

// parseArgsToObjectRaw converts an arguments string to a raw JSON object.
// It parses the JSON and returns it as raw JSON if valid, otherwise returns an empty object.
func parseArgsToObjectRaw(argsStr string) string {
	if argsStr == "" {
		return "{}"
	}
	// Try to parse as JSON and return as-is if valid
	if gjson.Valid(argsStr) {
		return argsStr
	}
	// If not valid JSON, try to fix common issues or return empty object
	return "{}"
}

// ConvertOpenAIResponseToGeminiNonStream converts a non-streaming OpenAI response to a non-streaming Gemini response.
//
// Parameters:
//   - ctx: The context for the request.
//   - modelName: The name of the model.
//   - rawJSON: The raw JSON response from the OpenAI API.
//   - param: A pointer to a parameter object for the conversion.
//
// Returns:
//   - string: A Gemini-compatible JSON response.
func ConvertOpenAIResponseToGeminiNonStream(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, _ *any) (string, error) {
	root := gjson.ParseBytes(rawJSON)

	// Base Gemini response template without finishReason; set when known
	out := `{"candidates":[{"content":{"parts":[],"role":"model"},"index":0}]}`

	// Set model if available
	if model := root.Get("model"); model.Exists() {
		out, _ = sjson.Set(out, "model", model.String())
	}

	// Process choices
	if choices := root.Get("choices"); choices.Exists() && choices.IsArray() {
		choices.ForEach(func(choiceIndex, choice gjson.Result) bool {
			choiceIdx := int(choice.Get("index").Int())
			message := choice.Get("message")

			// Set role
			if role := message.Get("role"); role.Exists() {
				if role.String() == "assistant" {
					out, _ = sjson.Set(out, "candidates.0.content.role", "model")
				}
			}

			partIndex := 0

			// Handle reasoning content before visible text
			if reasoning := message.Get("reasoning_content"); reasoning.Exists() {
				for _, reasoningText := range extractReasoningTexts(reasoning) {
					if reasoningText == "" {
						continue
					}
					out, _ = sjson.Set(out, fmt.Sprintf("candidates.0.content.parts.%d.thought", partIndex), true)
					out, _ = sjson.Set(out, fmt.Sprintf("candidates.0.content.parts.%d.text", partIndex), reasoningText)
					partIndex++
				}
			}

			// Handle content first
			if content := message.Get("content"); content.Exists() && content.String() != "" {
				out, _ = sjson.Set(out, fmt.Sprintf("candidates.0.content.parts.%d.text", partIndex), content.String())
				partIndex++
			}

			// Handle tool calls
			if toolCalls := message.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
				toolCalls.ForEach(func(_, toolCall gjson.Result) bool {
					if toolCall.Get("type").String() == "function" {
						function := toolCall.Get("function")
						functionName := function.Get("name").String()
						functionArgs := function.Get("arguments").String()

						namePath := fmt.Sprintf("candidates.0.content.parts.%d.functionCall.name", partIndex)
						argsPath := fmt.Sprintf("candidates.0.content.parts.%d.functionCall.args", partIndex)
						out, _ = sjson.Set(out, namePath, functionName)
						out, _ = sjson.SetRaw(out, argsPath, parseArgsToObjectRaw(functionArgs))
						partIndex++
					}
					return true
				})
			}

			// Handle finish reason
			if finishReason := choice.Get("finish_reason"); finishReason.Exists() {
				geminiFinishReason := mapOpenAIFinishReasonToGemini(finishReason.String())
				out, _ = sjson.Set(out, "candidates.0.finishReason", geminiFinishReason)
			}

			// Set index
			out, _ = sjson.Set(out, "candidates.0.index", choiceIdx)

			return true
		})
	}

	// Handle usage information
	if usage := root.Get("usage"); usage.Exists() {
		out, _ = sjson.Set(out, "usageMetadata.promptTokenCount", usage.Get("prompt_tokens").Int())
		out, _ = sjson.Set(out, "usageMetadata.candidatesTokenCount", usage.Get("completion_tokens").Int())
		out, _ = sjson.Set(out, "usageMetadata.totalTokenCount", usage.Get("total_tokens").Int())
		if reasoningTokens := reasoningTokensFromUsage(usage); reasoningTokens > 0 {
			out, _ = sjson.Set(out, "usageMetadata.thoughtsTokenCount", reasoningTokens)
		}
	}

	return out, nil
}

func GeminiTokenCount(ctx context.Context, count int64) string {
	return fmt.Sprintf(`{"totalTokens":%d,"promptTokensDetails":[{"modality":"TEXT","tokenCount":%d}]}`, count, count)
}

func reasoningTokensFromUsage(usage gjson.Result) int64 {
	if usage.Exists() {
		if v := usage.Get("completion_tokens_details.reasoning_tokens"); v.Exists() {
			return v.Int()
		}
		if v := usage.Get("output_tokens_details.reasoning_tokens"); v.Exists() {
			return v.Int()
		}
	}
	return 0
}

func extractReasoningTexts(node gjson.Result) []string {
	var texts []string
	if !node.Exists() {
		return texts
	}

	if node.IsArray() {
		node.ForEach(func(_, value gjson.Result) bool {
			texts = append(texts, extractReasoningTexts(value)...)
			return true
		})
		return texts
	}

	switch node.Type {
	case gjson.String:
		texts = append(texts, node.String())
	case gjson.JSON:
		if text := node.Get("text"); text.Exists() {
			texts = append(texts, text.String())
		} else if raw := strings.TrimSpace(node.Raw); raw != "" && !strings.HasPrefix(raw, "{") && !strings.HasPrefix(raw, "[") {
			texts = append(texts, raw)
		}
	}

	return texts
}

// tryParseNumber attempts to parse a string as an int or float.
func tryParseNumber(s string) (interface{}, bool) {
	if s == "" {
		return nil, false
	}
	// Try integer
	if i64, errParseInt := strconv.ParseInt(s, 10, 64); errParseInt == nil {
		return i64, true
	}
	if u64, errParseUInt := strconv.ParseUint(s, 10, 64); errParseUInt == nil {
		return u64, true
	}
	if f64, errParseFloat := strconv.ParseFloat(s, 64); errParseFloat == nil {
		return f64, true
	}
	return nil, false
}
