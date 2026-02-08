package amp

import "context"

type claudeToolNameMapKey struct{}

// ClaudeToolNameMap maps "mcp_<name>" back to "<name>" for this request.
type ClaudeToolNameMap map[string]string

func WithClaudeToolNameMap(ctx context.Context, m ClaudeToolNameMap) context.Context {
	return context.WithValue(ctx, claudeToolNameMapKey{}, m)
}

func GetClaudeToolNameMap(ctx context.Context) (ClaudeToolNameMap, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(claudeToolNameMapKey{}); v != nil {
		if m, ok := v.(ClaudeToolNameMap); ok {
			return m, true
		}
	}
	return nil, false
}
