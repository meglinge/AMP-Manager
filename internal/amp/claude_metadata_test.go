package amp

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestEnsureClaudeMetadataUserIDStableForSameMessages(t *testing.T) {
	body := []byte(`{"model":"claude-3-7-sonnet","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"world"}]}`)
	ua := "test-agent"
	key := "sk-test"

	out, changed := ensureClaudeMetadataUserID(body, ua, key)
	if !changed {
		t.Fatalf("expected changed=true")
	}
	userID := gjson.GetBytes(out, "metadata.user_id").String()
	if userID == "" {
		t.Fatalf("expected metadata.user_id to be set")
	}

	expectedHash := generateClaudeUserHash(ua, key)
	if !strings.Contains(userID, "user_"+expectedHash+"_account__session_") {
		t.Fatalf("unexpected user_id: %s", userID)
	}

	out2, changed2 := ensureClaudeMetadataUserID(out, ua, key)
	if changed2 {
		t.Fatalf("expected changed=false on second pass")
	}
	if string(out2) != string(out) {
		t.Fatalf("expected body unchanged on second pass")
	}
}

func TestEnsureClaudeMetadataUserIDNoopIfPresent(t *testing.T) {
	body := []byte(`{"metadata":{"user_id":"abc"}}`)
	out, changed := ensureClaudeMetadataUserID(body, "ua", "key")
	if changed {
		t.Fatalf("expected changed=false")
	}
	if string(out) != string(body) {
		t.Fatalf("expected body unchanged")
	}
}
