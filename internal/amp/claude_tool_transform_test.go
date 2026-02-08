package amp

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestPrefixAndUnprefixClaudeToolNamesWithMap(t *testing.T) {
	body := []byte(`{
		"model":"claude-3-7-sonnet",
		"tools":[{"name":"webSearch2"},{"name":"mcp_Read"}],
		"messages":[{"role":"user","content":[{"type":"tool_use","name":"extractWebPageContent"},{"type":"tool_use","name":"mcp_Read"}]}]
	}`)

	prefixed, m, changed := PrefixClaudeToolNamesWithMap(body)
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if gjson.GetBytes(prefixed, "tools.0.name").String() != "mcp_webSearch2" {
		t.Fatalf("expected tools[0] prefixed")
	}
	if gjson.GetBytes(prefixed, "tools.1.name").String() != "mcp_Read" {
		t.Fatalf("expected tools[1] unchanged")
	}
	if gjson.GetBytes(prefixed, "messages.0.content.0.name").String() != "mcp_extractWebPageContent" {
		t.Fatalf("expected content[0] prefixed")
	}
	if gjson.GetBytes(prefixed, "messages.0.content.1.name").String() != "mcp_Read" {
		t.Fatalf("expected content[1] unchanged")
	}
	if m["mcp_webSearch2"] != "webSearch2" {
		t.Fatalf("missing reverse map for webSearch2")
	}
	if _, ok := m["mcp_Read"]; ok {
		t.Fatalf("should not map already-prefixed tool")
	}

	unprefixed, uChanged := UnprefixClaudeToolNamesWithMap(prefixed, m)
	if !uChanged {
		t.Fatalf("expected unprefix changed=true")
	}
	if gjson.GetBytes(unprefixed, "messages.0.content.0.name").String() != "extractWebPageContent" {
		t.Fatalf("expected unprefixed tool_use name")
	}
	if gjson.GetBytes(unprefixed, "messages.0.content.1.name").String() != "mcp_Read" {
		t.Fatalf("expected original mcp_Read untouched")
	}
}
