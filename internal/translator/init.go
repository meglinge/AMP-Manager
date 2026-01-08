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
)

// Format constants
const (
	FormatOpenAI Format = "openai"
	FormatClaude Format = "claude"
	FormatGemini Format = "gemini"
)

func init() {
	// ===== OpenAI ↔ Claude =====
	// When client sends Claude format, channel is OpenAI
	// Request: Claude -> OpenAI, Response: OpenAI -> Claude
	Register(
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
	Register(
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
	Register(
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
	Register(
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
	Register(
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
	Register(
		FormatClaude, // from (incoming request format)
		FormatGemini, // to (channel/outgoing format)
		geminiClaude.ConvertClaudeRequestToGemini,
		ResponseTransform{
			Stream:     geminiClaude.ConvertGeminiResponseToClaude,
			NonStream:  geminiClaude.ConvertGeminiResponseToClaudeNonStream,
			TokenCount: geminiClaude.ClaudeTokenCount,
		},
	)
}
