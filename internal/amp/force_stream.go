package amp

import (
	"encoding/json"
)

func parseStreamFlag(body []byte) bool {
	var tmp struct {
		Stream *bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &tmp); err != nil || tmp.Stream == nil {
		return false
	}
	return *tmp.Stream
}

func forceJSONStreamTrue(body []byte) ([]byte, bool) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return body, false
	}
	if v, ok := m["stream"].(bool); ok && v {
		return body, false
	}
	m["stream"] = true
	out, err := json.Marshal(m)
	if err != nil {
		return body, false
	}
	return out, true
}
