package amp

import (
	"net/http"
	"sort"
	"strings"
)

var requiredAnthropicBetas = []string{
	"oauth-2025-04-20",
	"interleaved-thinking-2025-05-14",
}

func ensureRequiredAnthropicBetas(req *http.Request) {
	if req == nil {
		return
	}

	seen := make(map[string]struct{})
	existing := req.Header.Get("Anthropic-Beta")
	if existing != "" {
		for _, part := range strings.Split(existing, ",") {
			p := strings.TrimSpace(part)
			if p != "" {
				seen[p] = struct{}{}
			}
		}
	}

	for _, b := range requiredAnthropicBetas {
		seen[b] = struct{}{}
	}

	list := make([]string, 0, len(seen))
	for k := range seen {
		list = append(list, k)
	}
	sort.Strings(list)
	req.Header.Set("Anthropic-Beta", strings.Join(list, ","))
}
