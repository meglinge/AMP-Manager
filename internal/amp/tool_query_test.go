package amp

import "testing"

func TestDetectLocalToolQuery(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		wantTool string
		wantOK   bool
	}{
		{"empty", "", "", false},
		{"webSearch2", "webSearch2", webSearchQuery, true},
		{"mcp_webSearch2", "mcp_webSearch2", webSearchQuery, true},
		{"extract", "extractWebPageContent", extractWebPageContentQuery, true},
		{"mcp_extract", "mcp_extractWebPageContent", extractWebPageContentQuery, true},
		{"with_params", "foo=bar&webSearch2&x=y", webSearchQuery, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTool, gotOK := detectLocalToolQuery(tc.rawQuery)
			if gotOK != tc.wantOK {
				t.Fatalf("ok mismatch: got %v want %v", gotOK, tc.wantOK)
			}
			if gotTool != tc.wantTool {
				t.Fatalf("tool mismatch: got %q want %q", gotTool, tc.wantTool)
			}
		})
	}
}
