package translator

// Format identifies a request/response schema used inside the proxy.
type Format string

// FromString converts an arbitrary identifier to a translator format.
func FromString(v string) Format {
	return Format(v)
}

// String returns the raw schema identifier.
func (f Format) String() string {
	return string(f)
}

// Platform returns the platform family of the format.
// OpenAI, OpenAI-Chat, OpenAI-Responses all belong to "openai" platform.
func (f Format) Platform() string {
	switch f {
	case FormatOpenAI, FormatOpenAIChat, FormatOpenAIResponses:
		return "openai"
	case FormatClaude:
		return "claude"
	case FormatGemini:
		return "gemini"
	default:
		return string(f)
	}
}

// IsSamePlatform checks if two formats belong to the same platform.
// Same-platform format conversion is allowed (e.g., OpenAI Chat ↔ OpenAI Responses).
// Cross-platform conversion is NOT allowed (e.g., OpenAI ↔ Claude).
func IsSamePlatform(from, to Format) bool {
	return from.Platform() == to.Platform()
}
