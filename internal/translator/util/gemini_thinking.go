// Package util provides Gemini-specific thinking configuration utilities.
package util

import (
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Gemini model family detection patterns
var (
	gemini3Pattern      = regexp.MustCompile(`(?i)^gemini[_-]?3[_-]`)
	gemini3ProPattern   = regexp.MustCompile(`(?i)^gemini[_-]?3[_-]pro`)
	gemini3FlashPattern = regexp.MustCompile(`(?i)^gemini[_-]?3[_-]flash`)
	gemini25Pattern     = regexp.MustCompile(`(?i)^gemini[_-]?2\.5[_-]`)
)

// IsGemini3Model returns true if the model is a Gemini 3 family model.
// Gemini 3 models should use thinkingLevel (string) instead of thinkingBudget (number).
func IsGemini3Model(model string) bool {
	return gemini3Pattern.MatchString(model)
}

// IsGemini3ProModel returns true if the model is a Gemini 3 Pro variant.
func IsGemini3ProModel(model string) bool {
	return gemini3ProPattern.MatchString(model)
}

// IsGemini3FlashModel returns true if the model is a Gemini 3 Flash variant.
func IsGemini3FlashModel(model string) bool {
	return gemini3FlashPattern.MatchString(model)
}

// IsGemini25Model returns true if the model is a Gemini 2.5 family model.
func IsGemini25Model(model string) bool {
	return gemini25Pattern.MatchString(model)
}

// Gemini3ProThinkingLevels are the valid thinkingLevel values for Gemini 3 Pro models.
var Gemini3ProThinkingLevels = []string{"low", "high"}

// Gemini3FlashThinkingLevels are the valid thinkingLevel values for Gemini 3 Flash models.
var Gemini3FlashThinkingLevels = []string{"minimal", "low", "medium", "high"}

// ApplyGeminiThinkingLevel applies thinkingLevel config for Gemini 3 models.
// For standard Gemini API format (generationConfig.thinkingConfig path).
func ApplyGeminiThinkingLevel(body []byte, level string, includeThoughts *bool) []byte {
	if level == "" && includeThoughts == nil {
		return body
	}
	updated := body
	if level != "" {
		valuePath := "generationConfig.thinkingConfig.thinkingLevel"
		rewritten, err := sjson.SetBytes(updated, valuePath, level)
		if err == nil {
			updated = rewritten
		}
	}
	// Default to including thoughts when a level is set but no explicit include flag is provided.
	incl := includeThoughts
	if incl == nil && level != "" {
		defaultInclude := true
		incl = &defaultInclude
	}
	if incl != nil {
		if !gjson.GetBytes(updated, "generationConfig.thinkingConfig.includeThoughts").Exists() &&
			!gjson.GetBytes(updated, "generationConfig.thinkingConfig.include_thoughts").Exists() {
			valuePath := "generationConfig.thinkingConfig.includeThoughts"
			rewritten, err := sjson.SetBytes(updated, valuePath, *incl)
			if err == nil {
				updated = rewritten
			}
		}
	}
	if tb := gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget"); tb.Exists() {
		updated, _ = sjson.DeleteBytes(updated, "generationConfig.thinkingConfig.thinkingBudget")
	}
	return updated
}

// ValidateGemini3ThinkingLevel validates that the thinkingLevel is valid for the Gemini 3 model variant.
// Returns the validated level (normalized to lowercase) and true if valid, or empty string and false if invalid.
func ValidateGemini3ThinkingLevel(model, level string) (string, bool) {
	if level == "" {
		return "", false
	}
	normalized := strings.ToLower(strings.TrimSpace(level))

	var validLevels []string
	if IsGemini3ProModel(model) {
		validLevels = Gemini3ProThinkingLevels
	} else if IsGemini3FlashModel(model) {
		validLevels = Gemini3FlashThinkingLevels
	} else if IsGemini3Model(model) {
		// Unknown Gemini 3 variant - allow all levels as fallback
		validLevels = Gemini3FlashThinkingLevels
	} else {
		return "", false
	}

	for _, valid := range validLevels {
		if normalized == valid {
			return normalized, true
		}
	}
	return "", false
}

// ReasoningEffortBudgetMapping defines the thinkingBudget values for each reasoning effort level.
var ReasoningEffortBudgetMapping = map[string]int{
	"none":    0,
	"auto":    -1,
	"minimal": 512,
	"low":     1024,
	"medium":  8192,
	"high":    24576,
	"xhigh":   32768,
}

// ApplyReasoningEffortToGemini applies OpenAI reasoning_effort to Gemini thinkingConfig
// for standard Gemini API format (generationConfig.thinkingConfig path).
// Returns the modified body with thinkingBudget and include_thoughts set.
func ApplyReasoningEffortToGemini(body []byte, effort string) []byte {
	normalized := strings.ToLower(strings.TrimSpace(effort))
	if normalized == "" {
		return body
	}

	budgetPath := "generationConfig.thinkingConfig.thinkingBudget"
	includePath := "generationConfig.thinkingConfig.include_thoughts"

	if normalized == "none" {
		body, _ = sjson.DeleteBytes(body, "generationConfig.thinkingConfig")
		return body
	}

	budget, ok := ReasoningEffortBudgetMapping[normalized]
	if !ok {
		return body
	}

	body, _ = sjson.SetBytes(body, budgetPath, budget)
	body, _ = sjson.SetBytes(body, includePath, true)
	return body
}

// ThinkingBudgetToGemini3Level converts a thinkingBudget to a thinkingLevel for Gemini 3 models.
// This provides backward compatibility when thinkingBudget is provided for Gemini 3 models.
// Returns the appropriate thinkingLevel and true if conversion is possible.
func ThinkingBudgetToGemini3Level(model string, budget int) (string, bool) {
	if !IsGemini3Model(model) {
		return "", false
	}

	switch {
	case budget == -1:
		return "high", true
	case budget == 0:
		if IsGemini3FlashModel(model) {
			return "minimal", true
		}
		return "low", true
	case budget > 0 && budget <= 512:
		if IsGemini3FlashModel(model) {
			return "minimal", true
		}
		return "low", true
	case budget <= 1024:
		return "low", true
	case budget <= 8192:
		if IsGemini3FlashModel(model) {
			return "medium", true
		}
		return "high", true
	default:
		return "high", true
	}
}
