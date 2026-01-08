package amp

import (
	"net/http"
	"strings"

	"ampmanager/internal/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ModelsHandler returns local model list based on provider
// This prevents Amp client from using upstream's context_length (which causes 968k issue)
func ModelsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := strings.ToLower(c.Param("provider"))

		switch provider {
		case "google":
			handleGeminiModels(c)
		case "anthropic":
			handleClaudeModels(c)
		default:
			handleOpenAIModels(c)
		}
	}
}

// handleOpenAIModels returns OpenAI-compatible model list
func handleOpenAIModels(c *gin.Context) {
	models := getModelsForProvider("openai")

	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":                    m.ModelPattern,
			"object":                "model",
			"owned_by":              "openai",
			"context_length":        m.ContextLength,
			"max_completion_tokens": m.MaxCompletionTokens,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// handleClaudeModels returns Anthropic-compatible model list
func handleClaudeModels(c *gin.Context) {
	models := getModelsForProvider("anthropic")

	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":           m.ModelPattern,
			"object":       "model",
			"display_name": m.DisplayName,
			"owned_by":     "anthropic",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// handleGeminiModels returns Gemini-compatible model list
func handleGeminiModels(c *gin.Context) {
	models := getModelsForProvider("google")

	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		modelID := m.ModelPattern
		if !strings.HasPrefix(modelID, "models/") {
			modelID = "models/" + modelID
		}

		data = append(data, gin.H{
			"name":                       modelID,
			"displayName":                m.DisplayName,
			"inputTokenLimit":            m.ContextLength,
			"outputTokenLimit":           m.MaxCompletionTokens,
			"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": data,
	})
}

// modelInfo holds model metadata for handler responses
type modelInfo struct {
	ModelPattern        string
	DisplayName         string
	ContextLength       int
	MaxCompletionTokens int
	Provider            string
}

// getModelsForProvider returns models filtered by provider
func getModelsForProvider(provider string) []modelInfo {
	repo := repository.NewModelMetadataRepository()
	allModels, err := repo.List()
	if err != nil {
		log.Warnf("models handler: failed to list models: %v", err)
		return getBuiltinModelsForProvider(provider)
	}

	var result []modelInfo
	for _, m := range allModels {
		if strings.EqualFold(m.Provider, provider) {
			result = append(result, modelInfo{
				ModelPattern:        m.ModelPattern,
				DisplayName:         m.DisplayName,
				ContextLength:       m.ContextLength,
				MaxCompletionTokens: m.MaxCompletionTokens,
				Provider:            m.Provider,
			})
		}
	}

	if len(result) == 0 {
		return getBuiltinModelsForProvider(provider)
	}

	log.Debugf("models handler: returning %d models for provider %s", len(result), provider)
	return result
}

// getBuiltinModelsForProvider returns hardcoded models as fallback
func getBuiltinModelsForProvider(provider string) []modelInfo {
	builtins := GetBuiltinModelMetadata()
	var result []modelInfo

	for name, meta := range builtins {
		modelProvider := inferProvider(name)
		if strings.EqualFold(modelProvider, provider) {
			result = append(result, modelInfo{
				ModelPattern:        name,
				DisplayName:         name,
				ContextLength:       meta.ContextLength,
				MaxCompletionTokens: meta.MaxCompletionTokens,
				Provider:            modelProvider,
			})
		}
	}

	return result
}

// inferProvider guesses provider from model name
func inferProvider(modelName string) string {
	name := strings.ToLower(modelName)
	switch {
	case strings.HasPrefix(name, "claude"):
		return "anthropic"
	case strings.HasPrefix(name, "gemini"):
		return "google"
	case strings.HasPrefix(name, "gpt"):
		return "openai"
	case strings.HasPrefix(name, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(name, "qwen"):
		return "alibaba"
	default:
		return "openai"
	}
}
