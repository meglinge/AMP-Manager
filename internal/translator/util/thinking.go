// Package util provides utility functions for translator operations.
package util

import (
	"regexp"
	"strings"
)

// Model patterns for thinking support detection
var (
	// Models that support thinking/reasoning
	thinkingModelsPattern = regexp.MustCompile(`(?i)(gemini[_-]?(2\.5|3)|claude[_-]?(3\.5|4|sonnet)|gpt[_-]?(4o|5)|o[134]-|deepseek[_-]?r)`)

	// Models that use discrete thinking levels instead of numeric budgets
	thinkingLevelsModelsPattern = regexp.MustCompile(`(?i)(gpt[_-]?5|o[134]-|claude[_-]?4)`)
)

// ModelSupportsThinking reports whether the given model has Thinking capability.
// This is a simplified version that uses pattern matching instead of a registry.
func ModelSupportsThinking(model string) bool {
	if model == "" {
		return false
	}
	return thinkingModelsPattern.MatchString(model)
}

// ModelUsesThinkingLevels reports whether the model uses discrete reasoning
// effort levels instead of numeric budgets.
func ModelUsesThinkingLevels(model string) bool {
	if model == "" {
		return false
	}
	return thinkingLevelsModelsPattern.MatchString(model)
}

// GetModelThinkingLevels returns the discrete reasoning effort levels for the model.
// Returns nil if the model has no thinking support or no levels defined.
func GetModelThinkingLevels(model string) []string {
	if !ModelUsesThinkingLevels(model) {
		return nil
	}
	// Default levels for models that use discrete levels
	return []string{"none", "low", "medium", "high"}
}

// NormalizeThinkingBudget clamps the requested thinking budget to a reasonable range.
func NormalizeThinkingBudget(model string, budget int) int {
	if budget == -1 { // dynamic
		return -1
	}
	if budget == 0 {
		return 0
	}
	// Clamp to reasonable range [128, 32768]
	if budget < 128 {
		return 128
	}
	if budget > 32768 {
		return 32768
	}
	return budget
}

// ThinkingEffortToBudget maps a reasoning effort level to a numeric thinking budget (tokens).
//
// Mappings:
//   - "none"    -> 0
//   - "auto"    -> -1
//   - "minimal" -> 512
//   - "low"     -> 1024
//   - "medium"  -> 8192
//   - "high"    -> 24576
//   - "xhigh"   -> 32768
//
// Returns false when the effort level is empty or unsupported.
func ThinkingEffortToBudget(model, effort string) (int, bool) {
	if effort == "" {
		return 0, false
	}
	normalized := strings.ToLower(strings.TrimSpace(effort))
	switch normalized {
	case "none":
		return 0, true
	case "auto":
		return NormalizeThinkingBudget(model, -1), true
	case "minimal":
		return NormalizeThinkingBudget(model, 512), true
	case "low":
		return NormalizeThinkingBudget(model, 1024), true
	case "medium":
		return NormalizeThinkingBudget(model, 8192), true
	case "high":
		return NormalizeThinkingBudget(model, 24576), true
	case "xhigh":
		return NormalizeThinkingBudget(model, 32768), true
	default:
		return 0, false
	}
}

// ThinkingLevelToBudget maps a Gemini thinkingLevel to a numeric thinking budget (tokens).
//
// Mappings:
//   - "minimal" -> 512
//   - "low"     -> 1024
//   - "medium"  -> 8192
//   - "high"    -> 32768
//
// Returns false when the level is empty or unsupported.
func ThinkingLevelToBudget(level string) (int, bool) {
	if level == "" {
		return 0, false
	}
	normalized := strings.ToLower(strings.TrimSpace(level))
	switch normalized {
	case "minimal":
		return 512, true
	case "low":
		return 1024, true
	case "medium":
		return 8192, true
	case "high":
		return 32768, true
	default:
		return 0, false
	}
}

// ThinkingBudgetToEffort maps a numeric thinking budget (tokens)
// to a reasoning effort level.
//
// Mappings:
//   - 0            -> "none"
//   - -1           -> "auto"
//   - 1..1024      -> "low"
//   - 1025..8192   -> "medium"
//   - 8193..24576  -> "high"
//   - 24577..      -> "xhigh"
//
// Returns false when the budget is unsupported (negative values other than -1).
func ThinkingBudgetToEffort(model string, budget int) (string, bool) {
	switch {
	case budget == -1:
		return "auto", true
	case budget < -1:
		return "", false
	case budget == 0:
		if levels := GetModelThinkingLevels(model); len(levels) > 0 {
			return levels[0], true
		}
		return "none", true
	case budget > 0 && budget <= 1024:
		return "low", true
	case budget <= 8192:
		return "medium", true
	case budget <= 24576:
		return "high", true
	case budget > 24576:
		if levels := GetModelThinkingLevels(model); len(levels) > 0 {
			return levels[len(levels)-1], true
		}
		return "xhigh", true
	default:
		return "", false
	}
}
