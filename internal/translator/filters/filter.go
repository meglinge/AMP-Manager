package filters

import "ampmanager/internal/translator"

// RequestFilter defines the interface for request filters
type RequestFilter interface {
	// Name returns the filter name for logging
	Name() string
	// Applies checks if this filter should be applied
	Applies(outgoingFormat translator.Format) bool
	// Apply transforms the request body, returns new body and whether it was changed
	Apply(body []byte) ([]byte, bool, error)
}

// registry holds filters grouped by target format
var registry = make(map[translator.Format][]RequestFilter)

// Register adds a filter for a specific outgoing format
func Register(format translator.Format, filter RequestFilter) {
	registry[format] = append(registry[format], filter)
}

// ApplyFilters applies all registered filters for the given format
func ApplyFilters(format translator.Format, body []byte) ([]byte, error) {
	filters, ok := registry[format]
	if !ok {
		return body, nil
	}

	result := body
	for _, f := range filters {
		if !f.Applies(format) {
			continue
		}
		newBody, changed, err := f.Apply(result)
		if err != nil {
			return result, err
		}
		if changed {
			result = newBody
		}
	}
	return result, nil
}
