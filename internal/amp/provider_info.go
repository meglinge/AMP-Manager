package amp

import (
	"context"

	"ampmanager/internal/model"
)

type ProviderKind string

const (
	ProviderAnthropic       ProviderKind = "anthropic"
	ProviderOpenAIChat      ProviderKind = "openai_chat"
	ProviderOpenAIResponses ProviderKind = "openai_responses"
	ProviderGemini          ProviderKind = "gemini"
)

type ProviderInfo struct {
	Provider ProviderKind
	Endpoint string
}

type providerInfoKey struct{}

func WithProviderInfo(ctx context.Context, info ProviderInfo) context.Context {
	return context.WithValue(ctx, providerInfoKey{}, info)
}

func GetProviderInfo(ctx context.Context) (ProviderInfo, bool) {
	if val := ctx.Value(providerInfoKey{}); val != nil {
		if info, ok := val.(ProviderInfo); ok {
			return info, true
		}
	}
	return ProviderInfo{}, false
}

func ProviderInfoFromChannel(channel *model.Channel) ProviderInfo {
	if channel == nil {
		return ProviderInfo{Provider: ProviderAnthropic}
	}

	switch channel.Type {
	case model.ChannelTypeClaude:
		return ProviderInfo{
			Provider: ProviderAnthropic,
			Endpoint: string(channel.Endpoint),
		}
	case model.ChannelTypeOpenAI:
		if channel.Endpoint == model.ChannelEndpointResponses {
			return ProviderInfo{
				Provider: ProviderOpenAIResponses,
				Endpoint: "responses",
			}
		}
		return ProviderInfo{
			Provider: ProviderOpenAIChat,
			Endpoint: "chat_completions",
		}
	case model.ChannelTypeGemini:
		return ProviderInfo{
			Provider: ProviderGemini,
			Endpoint: "generate_content",
		}
	default:
		return ProviderInfo{Provider: ProviderAnthropic}
	}
}
