package main

import (
	"strings"
	"testing"
)

// TestBuildRoutingPolicy_SafetyGate_RoutingFlagsRequireAllowFlag verifies
// that any routing flag set without --allow-routing returns a typed error
// before submitting the request. This is the runtime equivalent of the
// design doc's AI[Routeable] type-level marker — explicit opt-in for
// dynamic provider selection.
func TestBuildRoutingPolicy_SafetyGate_RoutingFlagsRequireAllowFlag(t *testing.T) {
	tests := []struct {
		name     string
		fallback string
		require  string
		prefer   string
		maxPrice string
	}{
		{name: "fallback only", fallback: "anthropic,openai"},
		{name: "require only", require: "structured_outputs"},
		{name: "prefer only", prefer: "cheapest"},
		{name: "maxprice only", maxPrice: "1.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := buildRoutingPolicy("openrouter", tt.fallback, tt.require, tt.prefer, tt.maxPrice, false)
			if err == nil {
				t.Fatal("expected safety-gate error, got nil")
			}
			if policy != nil {
				t.Error("policy should be nil when safety gate trips")
			}
			if !strings.Contains(err.Error(), "--allow-routing") {
				t.Errorf("error should mention --allow-routing, got: %v", err)
			}
		})
	}
}

// TestBuildRoutingPolicy_SafetyGate_AllowRoutingPermits verifies that
// passing --allow-routing alongside routing flags returns a non-nil
// policy without error.
func TestBuildRoutingPolicy_SafetyGate_AllowRoutingPermits(t *testing.T) {
	policy, err := buildRoutingPolicy("openrouter", "anthropic,openai", "", "cheapest", "", true)
	if err != nil {
		t.Fatalf("buildRoutingPolicy() error = %v, want nil with --allow-routing", err)
	}
	if policy == nil {
		t.Fatal("policy is nil, want populated")
	}
	if len(policy.Order) != 2 {
		t.Errorf("policy.Order = %v, want [anthropic openai]", policy.Order)
	}
}

// TestBuildRoutingPolicy_NoFlags_ReturnsNilPolicy verifies that calling
// without any routing flags returns (nil, nil) regardless of allowRouting.
// This is the "no routing requested" no-op path.
func TestBuildRoutingPolicy_NoFlags_ReturnsNilPolicy(t *testing.T) {
	for _, allow := range []bool{false, true} {
		policy, err := buildRoutingPolicy("openai", "", "", "", "", allow)
		if err != nil {
			t.Fatalf("allow=%v: unexpected error = %v", allow, err)
		}
		if policy != nil {
			t.Errorf("allow=%v: policy = %+v, want nil", allow, policy)
		}
	}
}

// TestBuildRoutingPolicy_NonOpenRouterRejected verifies that the
// pre-existing provider check still fires once the safety gate passes.
func TestBuildRoutingPolicy_NonOpenRouterRejected(t *testing.T) {
	_, err := buildRoutingPolicy("anthropic", "openai", "", "", "", true)
	if err == nil {
		t.Fatal("expected error for non-openrouter provider with routing flags")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error should mention openrouter, got: %v", err)
	}
}
