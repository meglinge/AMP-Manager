package filters

import (
	"testing"

	"ampmanager/internal/translator"

	"github.com/tidwall/gjson"
)

func TestClaudeCodeSimulationFilterNormalizesCacheControlAndInjectsPreamble(t *testing.T) {
	f := &ClaudeCodeSimulationFilter{}
	if !f.Applies(translator.FormatClaude) {
		t.Fatalf("expected filter to apply to claude format")
	}

	body := []byte(`{
		"model":"claude-3-7-sonnet",
		"system":[{"type":"text","text":"Use opencode with amp-code","cache_control":{"type":"ephemeral","ttl":"1m"}}],
		"tools":[{"name":"webSearch2","cache_control":{"type":"ephemeral","ttl":"1m"}}],
		"messages":[{"role":"user","content":[{"type":"tool_use","name":"extractWebPageContent","cache_control":{"type":"ephemeral","ttl":"1m"}}]}]
	}`)

	out, changed, err := f.Apply(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}

	if got := gjson.GetBytes(out, "system.0.text").String(); got != claudeCodePreamble {
		t.Fatalf("expected preamble at system[0], got %q", got)
	}
	if got := gjson.GetBytes(out, "system.1.text").String(); got != "Use Claude Code with Claude Code" {
		t.Fatalf("expected sanitized system text, got %q", got)
	}
	if got := gjson.GetBytes(out, "system.1.cache_control.ttl").String(); got != "1h" {
		t.Fatalf("expected system cache_control ttl=1h, got %q", got)
	}

	if got := gjson.GetBytes(out, "tools.0.name").String(); got != "webSearch2" {
		t.Fatalf("expected tools[0].name unchanged, got %q", got)
	}
	if got := gjson.GetBytes(out, "messages.0.content.0.name").String(); got != "extractWebPageContent" {
		t.Fatalf("expected tool_use name unchanged, got %q", got)
	}
}

func TestClaudeCodeSimulationFilterSkipsHaiku(t *testing.T) {
	f := &ClaudeCodeSimulationFilter{}
	body := []byte(`{"model":"claude-3-haiku","tools":[{"name":"webSearch2"}]}`)

	out, changed, err := f.Apply(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if changed {
		t.Fatalf("expected changed=false")
	}
	if string(out) != string(body) {
		t.Fatalf("expected body unchanged")
	}
}

func TestClaudeCodeSimulationFilterCacheTTL5m(t *testing.T) {
	old := GetCacheTTLOverride()
	defer SetCacheTTLOverride(old)

	SetCacheTTLOverride("5m")

	f := &ClaudeCodeSimulationFilter{}
	body := []byte(`{
		"model":"claude-3-7-sonnet",
		"system":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral","ttl":"1m"}}],
		"tools":[{"name":"t1","cache_control":{"type":"ephemeral","ttl":"1m"}}],
		"messages":[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral","ttl":"1m"}}]}]
	}`)

	out, changed, err := f.Apply(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true")
	}

	if got := gjson.GetBytes(out, "system.1.cache_control.ttl").String(); got != "5m" {
		t.Fatalf("expected system cache_control ttl=5m, got %q", got)
	}
	if got := gjson.GetBytes(out, "tools.0.cache_control.ttl").String(); got != "5m" {
		t.Fatalf("expected tools cache_control ttl=5m, got %q", got)
	}
	if got := gjson.GetBytes(out, "messages.0.content.0.cache_control.ttl").String(); got != "5m" {
		t.Fatalf("expected messages cache_control ttl=5m, got %q", got)
	}
}

func TestClaudeCodeSimulationFilterCacheTTLEmpty(t *testing.T) {
	old := GetCacheTTLOverride()
	defer SetCacheTTLOverride(old)

	SetCacheTTLOverride("")

	f := &ClaudeCodeSimulationFilter{}
	body := []byte(`{
		"model":"claude-3-7-sonnet",
		"system":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral","ttl":"1m"}}],
		"tools":[{"name":"t1","cache_control":{"type":"ephemeral","ttl":"1m"}}]
	}`)

	out, changed, err := f.Apply(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if got := gjson.GetBytes(out, "system.1.cache_control.ttl").String(); got != "1m" {
		t.Fatalf("expected original ttl=1m preserved, got %q", got)
	}
	if got := gjson.GetBytes(out, "tools.0.cache_control.ttl").String(); got != "1m" {
		t.Fatalf("expected original tools ttl=1m preserved, got %q", got)
	}
	_ = changed
}
