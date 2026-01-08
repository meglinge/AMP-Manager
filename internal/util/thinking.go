// Package util provides utility functions for the translator package.
package util

import (
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// Walk recursively traverses a JSON structure to find all occurrences of a specific field.
// It builds paths to each occurrence and adds them to the provided paths slice.
//
// Parameters:
//   - value: The gjson.Result object to traverse
//   - path: The current path in the JSON structure (empty string for root)
//   - field: The field name to search for
//   - paths: Pointer to a slice where found paths will be stored
func Walk(value gjson.Result, path, field string, paths *[]string) {
	switch value.Type {
	case gjson.JSON:
		value.ForEach(func(key, val gjson.Result) bool {
			var childPath string
			var keyReplacer = strings.NewReplacer(".", "\\.", "*", "\\*", "?", "\\?")
			safeKey := keyReplacer.Replace(key.String())

			if path == "" {
				childPath = safeKey
			} else {
				childPath = path + "." + safeKey
			}
			if key.String() == field {
				*paths = append(*paths, childPath)
			}
			Walk(val, childPath, field, paths)
			return true
		})
	case gjson.String, gjson.Number, gjson.True, gjson.False, gjson.Null:
		// Terminal types - no further traversal needed
	}
}

// NormalizeThinkingBudget clamps the requested thinking budget to a supported range.
// Returns the original budget if no normalization is needed.
func NormalizeThinkingBudget(model string, budget int) int {
	// Define reasonable default ranges for thinking budget
	const minBudget = 1024
	const maxBudget = 128000

	if budget == -1 {
		// dynamic/auto mode
		return -1
	}
	if budget == 0 {
		return 0
	}
	if budget < minBudget {
		return minBudget
	}
	if budget > maxBudget {
		return maxBudget
	}
	return budget
}

// ModelSupportsThinking checks if the model supports thinking/reasoning.
// TODO: Implement actual model checking logic based on your requirements.
func ModelSupportsThinking(modelName string) bool {
	modelLower := strings.ToLower(modelName)
	return strings.Contains(modelLower, "claude-3") ||
		strings.Contains(modelLower, "claude-sonnet") ||
		strings.Contains(modelLower, "claude-opus") ||
		strings.Contains(modelLower, "o1") ||
		strings.Contains(modelLower, "o3") ||
		strings.Contains(modelLower, "deepseek")
}

// ModelUsesThinkingLevels checks if the model uses thinking levels instead of budget tokens.
// TODO: Implement actual model checking logic based on your requirements.
func ModelUsesThinkingLevels(modelName string) bool {
	modelLower := strings.ToLower(modelName)
	return strings.Contains(modelLower, "o1") || strings.Contains(modelLower, "o3")
}

// ThinkingEffortToBudget converts a thinking effort level to a budget token count.
// Returns the budget and true if successful, or 0 and false if the effort is not recognized.
func ThinkingEffortToBudget(modelName, effort string) (int, bool) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	switch effort {
	case "none", "disabled", "off":
		return 0, true
	case "low", "minimal":
		return 4000, true
	case "medium", "moderate", "default":
		return 10000, true
	case "high", "extensive":
		return 20000, true
	case "max", "maximum", "auto":
		return -1, true // -1 indicates auto/maximum
	default:
		return 0, false
	}
}

// ThinkingBudgetToEffort converts a budget token count to a thinking effort level.
// Returns the effort level and true if successful.
func ThinkingBudgetToEffort(modelName string, budget int) (string, bool) {
	switch {
	case budget == 0:
		return "none", true
	case budget < 0:
		return "high", true // auto/maximum maps to high
	case budget <= 4000:
		return "low", true
	case budget <= 10000:
		return "medium", true
	case budget <= 20000:
		return "high", true
	default:
		return "high", true
	}
}

// GetThinkingText extracts the thinking text from a content part.
func GetThinkingText(part gjson.Result) string {
	if thinking := part.Get("thinking"); thinking.Exists() {
		return thinking.String()
	}
	if text := part.Get("text"); text.Exists() {
		return text.String()
	}
	return ""
}

// FixJSON attempts to fix common JSON formatting issues.
func FixJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "{}"
	}
	// Remove trailing commas before closing brackets
	re := regexp.MustCompile(`,\s*([}\]])`)
	s = re.ReplaceAllString(s, "$1")
	return s
}
