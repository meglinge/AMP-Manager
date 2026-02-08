// Package translator provides format detection for different AI API schemas.
// NOTE: Cross-platform and cross-format translation has been removed.
// Only same-platform, same-format requests are supported.
package translator

// Format constants
const (
	FormatOpenAI          Format = "openai"           // Generic OpenAI (for backward compatibility)
	FormatOpenAIChat      Format = "openai-chat"      // /v1/chat/completions
	FormatOpenAIResponses Format = "openai-responses" // /v1/responses
	FormatClaude          Format = "claude"
	FormatGemini          Format = "gemini"
)

// RegisterAll is a no-op since translation is no longer supported.
// Only same-platform, same-format requests are allowed.
func RegisterAll(registry *Registry) {
	// No translators registered - cross-format conversion is not supported
}
