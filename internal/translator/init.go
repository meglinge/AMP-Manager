// Package translator provides format translation between different AI API schemas.
// This file initializes all registered translators.
package translator

import (
	// claude/gemini: converts Gemini ↔ Claude
	claudeGemini "ampmanager/internal/translator/claude/gemini"

	// claude/openai: converts OpenAI ↔ Claude
	claudeOpenai "ampmanager/internal/translator/claude/openai"

	// gemini/claude: converts Claude ↔ Gemini
	geminiClaude "ampmanager/internal/translator/gemini/claude"

	// gemini/openai: converts OpenAI ↔ Gemini
	geminiOpenai "ampmanager/internal/translator/gemini/openai"

	// openai/claude: converts Claude ↔ OpenAI
	openaiClaude "ampmanager/internal/translator/openai/claude"

	// openai/gemini: converts Gemini ↔ OpenAI
	openaiGemini "ampmanager/internal/translator/openai/gemini"

	// openai/responses: converts OpenAI Chat ↔ OpenAI Responses
	openaiResponses "ampmanager/internal/translator/openai/responses"
)

// Format constants
const (
	FormatOpenAI          Format = "openai"           // Generic OpenAI (for backward compatibility)
	FormatOpenAIChat      Format = "openai-chat"      // /v1/chat/completions
	FormatOpenAIResponses Format = "openai-responses" // /v1/responses
	FormatClaude          Format = "claude"
	FormatGemini          Format = "gemini"
)

// RegisterAll registers all translator transforms to the given registry.
func RegisterAll(registry *Registry) {
	// ===== OpenAI ↔ Claude =====
	// When client sends Claude format, channel is OpenAI
	// Request: Claude -> OpenAI, Response: OpenAI -> Claude
	registry.Register(
		FormatClaude, // from (incoming request format)
		FormatOpenAI, // to (channel/outgoing format)
		openaiClaude.ConvertClaudeRequestToOpenAI,
		ResponseTransform{
			Stream:     openaiClaude.ConvertOpenAIResponseToClaude,
			NonStream:  openaiClaude.ConvertOpenAIResponseToClaudeNonStream,
			TokenCount: openaiClaude.ClaudeTokenCount,
		},
	)

	// When client sends OpenAI format, channel is Claude
	// Request: OpenAI -> Claude, Response: Claude -> OpenAI
	registry.Register(
		FormatOpenAI, // from (incoming request format)
		FormatClaude, // to (channel/outgoing format)
		claudeOpenai.ConvertOpenAIRequestToClaude,
		ResponseTransform{
			Stream:    claudeOpenai.ConvertClaudeResponseToOpenAI,
			NonStream: claudeOpenai.ConvertClaudeResponseToOpenAINonStream,
		},
	)

	// ===== OpenAI ↔ Gemini =====
	// When client sends Gemini format, channel is OpenAI
	// Request: Gemini -> OpenAI, Response: OpenAI -> Gemini
	registry.Register(
		FormatGemini, // from (incoming request format)
		FormatOpenAI, // to (channel/outgoing format)
		openaiGemini.ConvertGeminiRequestToOpenAI,
		ResponseTransform{
			Stream:    openaiGemini.ConvertOpenAIResponseToGemini,
			NonStream: openaiGemini.ConvertOpenAIResponseToGeminiNonStream,
		},
	)

	// When client sends OpenAI format, channel is Gemini
	// Request: OpenAI -> Gemini, Response: Gemini -> OpenAI
	registry.Register(
		FormatOpenAI, // from (incoming request format)
		FormatGemini, // to (channel/outgoing format)
		geminiOpenai.ConvertOpenAIRequestToGemini,
		ResponseTransform{
			Stream:    geminiOpenai.ConvertGeminiResponseToOpenAI,
			NonStream: geminiOpenai.ConvertGeminiResponseToOpenAINonStream,
		},
	)

	// ===== Claude ↔ Gemini =====
	// When client sends Gemini format, channel is Claude
	// Request: Gemini -> Claude, Response: Claude -> Gemini
	registry.Register(
		FormatGemini, // from (incoming request format)
		FormatClaude, // to (channel/outgoing format)
		claudeGemini.ConvertGeminiRequestToClaude,
		ResponseTransform{
			Stream:    claudeGemini.ConvertClaudeResponseToGemini,
			NonStream: claudeGemini.ConvertClaudeResponseToGeminiNonStream,
		},
	)

	// When client sends Claude format, channel is Gemini
	// Request: Claude -> Gemini, Response: Gemini -> Claude
	registry.Register(
		FormatClaude, // from (incoming request format)
		FormatGemini, // to (channel/outgoing format)
		geminiClaude.ConvertClaudeRequestToGemini,
		ResponseTransform{
			Stream:     geminiClaude.ConvertGeminiResponseToClaude,
			NonStream:  geminiClaude.ConvertGeminiResponseToClaudeNonStream,
			TokenCount: geminiClaude.ClaudeTokenCount,
		},
	)

	// ===== OpenAI Chat ↔ OpenAI Responses =====
	// When client sends Chat format, channel is Responses
	// Request: Chat -> Responses, Response: Responses -> Chat
	registry.Register(
		FormatOpenAIChat,      // from (incoming request format)
		FormatOpenAIResponses, // to (channel/outgoing format)
		openaiResponses.ConvertChatToResponsesRequest,
		ResponseTransform{
			Stream:    openaiResponses.ConvertResponsesStreamToChat,
			NonStream: openaiResponses.ConvertResponsesNonStreamToChat,
		},
	)

	// When client sends Responses format, channel is Chat
	// Request: Responses -> Chat, Response: Chat -> Responses
	registry.Register(
		FormatOpenAIResponses, // from (incoming request format)
		FormatOpenAIChat,      // to (channel/outgoing format)
		openaiResponses.ConvertResponsesToChatRequest,
		ResponseTransform{
			Stream:    openaiResponses.ConvertChatStreamToResponses,
			NonStream: openaiResponses.ConvertChatNonStreamToResponses,
		},
	)

	// ===== OpenAIChat ↔ Claude =====
	// When client sends OpenAI Chat format, channel is Claude
	registry.Register(
		FormatOpenAIChat, // from
		FormatClaude,     // to
		claudeOpenai.ConvertOpenAIRequestToClaude,
		ResponseTransform{
			Stream:    claudeOpenai.ConvertClaudeResponseToOpenAI,
			NonStream: claudeOpenai.ConvertClaudeResponseToOpenAINonStream,
		},
	)

	// When client sends Claude format, channel is OpenAI Chat
	registry.Register(
		FormatClaude,     // from
		FormatOpenAIChat, // to
		openaiClaude.ConvertClaudeRequestToOpenAI,
		ResponseTransform{
			Stream:     openaiClaude.ConvertOpenAIResponseToClaude,
			NonStream:  openaiClaude.ConvertOpenAIResponseToClaudeNonStream,
			TokenCount: openaiClaude.ClaudeTokenCount,
		},
	)

	// ===== OpenAIResponses ↔ Claude =====
	// When client sends OpenAI Responses format, channel is Claude
	registry.Register(
		FormatOpenAIResponses, // from
		FormatClaude,          // to
		claudeOpenai.ConvertOpenAIRequestToClaude,
		ResponseTransform{
			Stream:    claudeOpenai.ConvertClaudeResponseToOpenAI,
			NonStream: claudeOpenai.ConvertClaudeResponseToOpenAINonStream,
		},
	)

	// When client sends Claude format, channel is OpenAI Responses
	registry.Register(
		FormatClaude,          // from
		FormatOpenAIResponses, // to
		openaiClaude.ConvertClaudeRequestToOpenAI,
		ResponseTransform{
			Stream:     openaiClaude.ConvertOpenAIResponseToClaude,
			NonStream:  openaiClaude.ConvertOpenAIResponseToClaudeNonStream,
			TokenCount: openaiClaude.ClaudeTokenCount,
		},
	)

	// ===== OpenAIChat ↔ Gemini =====
	// When client sends OpenAI Chat format, channel is Gemini
	registry.Register(
		FormatOpenAIChat, // from
		FormatGemini,     // to
		geminiOpenai.ConvertOpenAIRequestToGemini,
		ResponseTransform{
			Stream:    geminiOpenai.ConvertGeminiResponseToOpenAI,
			NonStream: geminiOpenai.ConvertGeminiResponseToOpenAINonStream,
		},
	)

	// When client sends Gemini format, channel is OpenAI Chat
	registry.Register(
		FormatGemini,     // from
		FormatOpenAIChat, // to
		openaiGemini.ConvertGeminiRequestToOpenAI,
		ResponseTransform{
			Stream:    openaiGemini.ConvertOpenAIResponseToGemini,
			NonStream: openaiGemini.ConvertOpenAIResponseToGeminiNonStream,
		},
	)

	// ===== OpenAIResponses ↔ Gemini =====
	// When client sends OpenAI Responses format, channel is Gemini
	registry.Register(
		FormatOpenAIResponses, // from
		FormatGemini,          // to
		geminiOpenai.ConvertOpenAIRequestToGemini,
		ResponseTransform{
			Stream:    geminiOpenai.ConvertGeminiResponseToOpenAI,
			NonStream: geminiOpenai.ConvertGeminiResponseToOpenAINonStream,
		},
	)

	// When client sends Gemini format, channel is OpenAI Responses
	registry.Register(
		FormatGemini,          // from
		FormatOpenAIResponses, // to
		openaiGemini.ConvertGeminiRequestToOpenAI,
		ResponseTransform{
			Stream:    openaiGemini.ConvertOpenAIResponseToGemini,
			NonStream: openaiGemini.ConvertOpenAIResponseToGeminiNonStream,
		},
	)
}
