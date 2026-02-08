package amp

import (
	"io"
	"strings"
	"testing"
)

type nopReadCloser struct{ io.Reader }

func (n nopReadCloser) Close() error { return nil }

func TestSSETransformWrapperStripsMCPPrefix(t *testing.T) {
	sse := "event: message\n" +
		"data: {\"type\":\"tool_use\",\"name\":\"mcp_extractWebPageContent\"}\n\n"

	rc := nopReadCloser{Reader: strings.NewReader(sse)}
	wrapped := NewSSETransformWrapper(rc, func(b []byte) []byte {
		out, _ := UnprefixClaudeToolNamesWithMap(b, ClaudeToolNameMap{"mcp_extractWebPageContent": "extractWebPageContent"})
		return out
	})
	defer wrapped.Close()

	out, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !strings.Contains(string(out), `"name":"extractWebPageContent"`) {
		t.Fatalf("expected prefix stripped, got: %s", string(out))
	}
}
