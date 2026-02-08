package amp

import (
	"bytes"
	"encoding/json"
	"strings"
)

const claudeShimPrefix = "mcp_"

// PrefixClaudeToolNamesWithMap prefixes Claude tool names with mcp_ when needed,
// and returns a reverse map for safe unprefixing on the way back.
//
// Rules:
// - Only affects tools[].name and messages[].content[].(tool_use|tool_result).name
// - Does NOT modify names already starting with mcp_
// - Avoids collisions: if mcp_<name> already exists, leaves <name> untouched
func PrefixClaudeToolNamesWithMap(body []byte) ([]byte, ClaudeToolNameMap, bool) {
	if len(body) == 0 || !json.Valid(body) {
		return body, nil, false
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return body, nil, false
	}

	rootObj, ok := root.(map[string]any)
	if !ok {
		return body, nil, false
	}

	changed := false
	nameSet := make(map[string]struct{})

	// Collect existing tool names for collision avoidance
	if tools, ok := rootObj["tools"].([]any); ok {
		for _, t := range tools {
			if obj, ok := t.(map[string]any); ok {
				if name, ok := obj["name"].(string); ok {
					nameSet[name] = struct{}{}
				}
			}
		}
	}

	reverse := ClaudeToolNameMap{}

	prefixName := func(name string) (string, bool) {
		if strings.HasPrefix(name, claudeShimPrefix) {
			return name, false
		}
		candidate := claudeShimPrefix + name
		if _, exists := nameSet[candidate]; exists {
			return name, false
		}
		nameSet[candidate] = struct{}{}
		reverse[candidate] = name
		return candidate, true
	}

	// tools[].name
	if tools, ok := rootObj["tools"].([]any); ok {
		for _, t := range tools {
			obj, ok := t.(map[string]any)
			if !ok {
				continue
			}
			name, ok := obj["name"].(string)
			if !ok || name == "" {
				continue
			}
			if newName, did := prefixName(name); did {
				obj["name"] = newName
				changed = true
			}
		}
	}

	// messages[].content[].(tool_use|tool_result).name
	if messages, ok := rootObj["messages"].([]any); ok {
		for _, m := range messages {
			msgObj, ok := m.(map[string]any)
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
				t, _ := itemObj["type"].(string)
				if t != "tool_use" && t != "tool_result" {
					continue
				}
				name, ok := itemObj["name"].(string)
				if !ok || name == "" {
					continue
				}
				if newName, did := prefixName(name); did {
					itemObj["name"] = newName
					changed = true
				}
			}
		}
	}

	if !changed {
		return body, nil, false
	}

	out, err := json.Marshal(rootObj)
	if err != nil || bytes.Equal(out, body) {
		return body, nil, false
	}
	if len(reverse) == 0 {
		return out, nil, true
	}
	return out, reverse, true
}

// UnprefixClaudeToolNamesWithMap rewrites tool_use/tool_result names using the reverse map.
// Only mapped names are rewritten; everything else is left intact.
func UnprefixClaudeToolNamesWithMap(body []byte, reverse ClaudeToolNameMap) ([]byte, bool) {
	if len(body) == 0 || len(reverse) == 0 || !json.Valid(body) {
		return body, false
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return body, false
	}

	changed := false
	var walk func(any)
	walk = func(v any) {
		switch node := v.(type) {
		case map[string]any:
			// If this is a tool_use/tool_result object, try rewrite its name.
			if t, _ := node["type"].(string); t == "tool_use" || t == "tool_result" {
				if name, ok := node["name"].(string); ok {
					if orig, ok := reverse[name]; ok {
						node["name"] = orig
						changed = true
					}
				}
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}

	walk(root)
	if !changed {
		return body, false
	}
	out, err := json.Marshal(root)
	if err != nil || bytes.Equal(out, body) {
		return body, false
	}
	return out, true
}
