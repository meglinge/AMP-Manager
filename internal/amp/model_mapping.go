package amp

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"

	"ampmanager/internal/model"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type MappingResult struct {
	OriginalModel string
	MappedModel   string
	ThinkingLevel string
	Applied       bool
}

func ApplyModelMappingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := GetProxyConfig(c.Request.Context())
		if cfg == nil || cfg.ModelMappingsJSON == "" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.Next()
			return
		}

		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		modelName, ok := payload["model"].(string)
		if !ok || modelName == "" {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		var mappings []model.ModelMapping
		if err := json.Unmarshal([]byte(cfg.ModelMappingsJSON), &mappings); err != nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		result := applyMapping(modelName, mappings)

		if result.Applied {
			payload["model"] = result.MappedModel
			log.Infof("model mapping: %s -> %s", result.OriginalModel, result.MappedModel)

			if result.ThinkingLevel != "" {
				applyThinkingLevel(payload, result.ThinkingLevel)
				log.Infof("model mapping: applied thinking level '%s'", result.ThinkingLevel)
			}

			newBody, err := json.Marshal(payload)
			if err != nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				c.Next()
				return
			}

			c.Request.Body = io.NopCloser(bytes.NewBuffer(newBody))
			c.Request.ContentLength = int64(len(newBody))
		} else {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()
	}
}

func applyMapping(modelName string, mappings []model.ModelMapping) MappingResult {
	for _, m := range mappings {
		if m.From == "" {
			continue
		}

		matched := false
		if m.Regex {
			re, err := regexp.Compile(m.From)
			if err == nil && re.MatchString(modelName) {
				matched = true
			}
		} else {
			if strings.EqualFold(m.From, modelName) || m.From == modelName {
				matched = true
			}
		}

		if matched {
			targetModel := m.To
			if targetModel == "" {
				targetModel = modelName
			}

			return MappingResult{
				OriginalModel: modelName,
				MappedModel:   targetModel,
				ThinkingLevel: m.ThinkingLevel,
				Applied:       true,
			}
		}
	}

	return MappingResult{
		OriginalModel: modelName,
		MappedModel:   modelName,
		Applied:       false,
	}
}

func applyThinkingLevel(payload map[string]interface{}, level string) {
	if level == "" {
		return
	}

	modelName, _ := payload["model"].(string)
	modelLower := strings.ToLower(modelName)

	if strings.HasPrefix(modelLower, "o1") || strings.HasPrefix(modelLower, "o3") || strings.HasPrefix(modelLower, "o4") {
		payload["reasoning_effort"] = level
		return
	}

	if strings.HasPrefix(modelLower, "gpt") {
		payload["reasoning_effort"] = level
		return
	}

	if strings.HasPrefix(modelLower, "claude") {
		budgetTokens := thinkingLevelToBudget(level, "claude")
		if budgetTokens > 0 {
			thinking := map[string]interface{}{
				"type":          "enabled",
				"budget_tokens": budgetTokens,
			}
			payload["thinking"] = thinking
		}
		return
	}

	if strings.HasPrefix(modelLower, "gemini") {
		budgetTokens := thinkingLevelToBudget(level, "gemini")
		generationConfig, ok := payload["generationConfig"].(map[string]interface{})
		if !ok {
			generationConfig = make(map[string]interface{})
		}
		thinkingConfig := map[string]interface{}{
			"thinkingBudget": budgetTokens,
		}
		generationConfig["thinkingConfig"] = thinkingConfig
		payload["generationConfig"] = generationConfig
		return
	}

	payload["reasoning_effort"] = level
}

func thinkingLevelToBudget(level string, provider string) int {
	levelLower := strings.ToLower(level)

	if provider == "claude" {
		switch levelLower {
		case "low":
			return 1024
		case "medium":
			return 8192
		case "high":
			return 32768
		case "xhigh":
			return 100000
		default:
			return 0
		}
	}

	if provider == "gemini" {
		switch levelLower {
		case "low":
			return 1024
		case "medium":
			return 8192
		case "high":
			return 24576
		case "xhigh":
			return 32768
		default:
			return 0
		}
	}

	return 0
}
