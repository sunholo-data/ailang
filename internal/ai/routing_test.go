package ai

import (
	"errors"
	"testing"
)

func TestAIRoutingPolicy_IsZero(t *testing.T) {
	tests := []struct {
		name string
		p    *AIRoutingPolicy
		want bool
	}{
		{"nil pointer", nil, true},
		{"all-empty struct", &AIRoutingPolicy{}, true},
		{"order set", &AIRoutingPolicy{Order: []string{"anthropic"}}, false},
		{"allow fallback set", &AIRoutingPolicy{AllowFallback: true}, false},
		{"require set", &AIRoutingPolicy{Require: []AICapability{CapToolCalling}}, false},
		{"max price set", &AIRoutingPolicy{MaxPricePerMTok: "0.005"}, false},
		{"prefer set", &AIRoutingPolicy{Prefer: PreferCheapest}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIRoutingPolicy_HasRouting(t *testing.T) {
	tests := []struct {
		name string
		p    *AIRoutingPolicy
		want bool
	}{
		{"nil pointer", nil, false},
		{"all-empty struct", &AIRoutingPolicy{}, false},
		{"order set", &AIRoutingPolicy{Order: []string{"anthropic"}}, true},
		{"allow fallback set", &AIRoutingPolicy{AllowFallback: true}, true},
		{"require only", &AIRoutingPolicy{Require: []AICapability{CapToolCalling}}, false},
		{"prefer only", &AIRoutingPolicy{Prefer: PreferCheapest}, false},
		{"max price only", &AIRoutingPolicy{MaxPricePerMTok: "0.005"}, false},
		{"order + prefer", &AIRoutingPolicy{Order: []string{"openai"}, Prefer: PreferFastest}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.HasRouting(); got != tt.want {
				t.Errorf("HasRouting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrRoutingNotSupported_IsSentinel(t *testing.T) {
	// Provider error wrapping ErrRoutingNotSupported must be detectable via errors.Is.
	wrapped := NewProviderError("openai", 0, "no routing here", ErrRoutingNotSupported)
	if !errors.Is(wrapped, ErrRoutingNotSupported) {
		t.Fatalf("errors.Is should find ErrRoutingNotSupported in wrapped ProviderError")
	}

	// Unwrap chain works for direct comparison too.
	if !errors.Is(ErrRoutingNotSupported, ErrRoutingNotSupported) {
		t.Fatalf("ErrRoutingNotSupported is not equal to itself via errors.Is")
	}

	// A different sentinel must NOT match.
	other := errors.New("other")
	wrappedOther := NewProviderError("openai", 0, "x", other)
	if errors.Is(wrappedOther, ErrRoutingNotSupported) {
		t.Fatalf("errors.Is must not match an unrelated wrapped error")
	}
}

func TestRoutePreference_Constants(t *testing.T) {
	// Stable wire identifiers — guard against accidental rename.
	cases := map[RoutePreference]string{
		PreferUnspecified:  "",
		PreferCheapest:     "cheapest",
		PreferFastest:      "fastest",
		PreferMostReliable: "most_reliable",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("RoutePreference %q stringifies to %q, want %q", got, string(got), want)
		}
	}
}

func TestAICapability_Constants(t *testing.T) {
	cases := map[AICapability]string{
		CapStructuredOutputs: "structured_outputs",
		CapToolCalling:       "tool_calling",
		CapVision:            "vision",
		CapJSONMode:          "json_mode",
		CapStreaming:         "streaming",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("AICapability %q stringifies to %q, want %q", got, string(got), want)
		}
	}
}
