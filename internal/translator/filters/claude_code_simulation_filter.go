package filters

import (
	"ampmanager/internal/translator"
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

const (
	claudeCodePreamble = "You are Claude Code, Anthropic's official CLI for Claude."
)

var claudeBrandSanitizeRe = regexp.MustCompile(`(?i)\b(?:opencode|amp(?:-?code)?)\b`)

type ClaudeCodeSimulationFilter struct{}

func (f *ClaudeCodeSimulationFilter) Name() string {
	return "claude_code_simulation"
}

func (f *ClaudeCodeSimulationFilter) Applies(outgoingFormat translator.Format) bool {
	return outgoingFormat == translator.FormatClaude
}

func sanitizeClaudeBrandText(s string) string {
	return claudeBrandSanitizeRe.ReplaceAllString(s, "Claude Code")
}

func normalizeClaudeCacheControl(obj map[string]any) bool {
	if _, ok := obj["cache_control"]; !ok {
		return false
	}
	obj["cache_control"] = map[string]any{
		"type": "ephemeral",
		"ttl":  "1h",
	}
	return true
}

func (f *ClaudeCodeSimulationFilter) Apply(body []byte) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}
	if !json.Valid(body) {
		return body, false, nil
	}

	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body, false, nil
	}

	if model, ok := root["model"].(string); ok {
		if strings.Contains(strings.ToLower(model), "haiku") {
			return body, false, nil
		}
	}

	changed := false

	// 0) system 清洗 + 注入 Claude Code 身份声明 + cache_control 统一
	systemVal, hasSystem := root["system"]
	if !hasSystem {
		root["system"] = []any{map[string]any{"type": "text", "text": claudeCodePreamble}}
		changed = true
	} else {
		switch v := systemVal.(type) {
		case string:
			cleaned := sanitizeClaudeBrandText(v)
			items := []any{map[string]any{"type": "text", "text": claudeCodePreamble}}

			rest := cleaned
			if strings.HasPrefix(rest, claudeCodePreamble) {
				rest = strings.TrimPrefix(rest, claudeCodePreamble)
				rest = strings.TrimLeft(rest, "\r\n")
			}
			if rest != "" {
				items = append(items, map[string]any{"type": "text", "text": rest})
			}
			root["system"] = items
			changed = true
		case []any:
			for _, item := range v {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if normalizeClaudeCacheControl(obj) {
					changed = true
				}
				if t, ok := obj["type"].(string); ok && t == "text" {
					if text, ok := obj["text"].(string); ok {
						cleaned := sanitizeClaudeBrandText(text)
						if cleaned != text {
							obj["text"] = cleaned
							changed = true
						}
					}
				}
			}

			alreadyPrefixed := false
			if len(v) > 0 {
				if first, ok := v[0].(map[string]any); ok {
					if t, _ := first["type"].(string); t == "text" {
						if text, _ := first["text"].(string); text == claudeCodePreamble {
							alreadyPrefixed = true
						}
					}
				}
			}
			if !alreadyPrefixed {
				root["system"] = append(
					[]any{map[string]any{"type": "text", "text": claudeCodePreamble}},
					v...,
				)
				changed = true
			}
		}
	}

	// 1) tools[].cache_control 统一
	if tools, ok := root["tools"].([]any); ok {
		for _, tool := range tools {
			obj, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			if normalizeClaudeCacheControl(obj) {
				changed = true
			}
		}
	}

	// 2) messages[].content[] cache_control 统一
	if messages, ok := root["messages"].([]any); ok {
		for _, msg := range messages {
			msgObj, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			content, ok := msgObj["content"].([]any)
			if !ok {
				continue
			}
			for _, item := range content {
				itemObj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if normalizeClaudeCacheControl(itemObj) {
					changed = true
				}
			}
		}
	}

	if !changed {
		return body, false, nil
	}

	out, err := json.Marshal(root)
	if err != nil {
		return body, false, nil
	}
	if bytes.Equal(out, body) {
		return body, false, nil
	}
	return out, true, nil
}
