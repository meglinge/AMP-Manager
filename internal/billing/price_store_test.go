package billing

import (
	"encoding/json"
	"testing"
)

func TestLiteLLMPricingAcceptsWholeNumberFloats(t *testing.T) {
	raw := []byte(`{"max_input_tokens":2000000.0,"max_output_tokens":8192}`)

	var pricing LiteLLMPricing
	if err := json.Unmarshal(raw, &pricing); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if pricing.MaxInputTokens == nil || *pricing.MaxInputTokens != wholeNumber(2000000) {
		t.Fatalf("unexpected max_input_tokens: %#v", pricing.MaxInputTokens)
	}
	if pricing.MaxOutputTokens == nil || *pricing.MaxOutputTokens != wholeNumber(8192) {
		t.Fatalf("unexpected max_output_tokens: %#v", pricing.MaxOutputTokens)
	}
}

func TestLiteLLMPricingRejectsFractionalTokenCounts(t *testing.T) {
	raw := []byte(`{"max_input_tokens":123.45}`)

	var pricing LiteLLMPricing
	if err := json.Unmarshal(raw, &pricing); err == nil {
		t.Fatal("expected unmarshal error for fractional token count")
	}
}
