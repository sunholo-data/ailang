package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
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

// =============================================================================
// M-AI-EFFECT-MODES M2: tests for the declared-mode-aware safety-gate
// relaxation. The CLI safety-gate runs after typecheck so it can read the
// entry function's elaborated effect row. A program declared
// !{AI[mode=routeable]} attests routing intent at the type level — the
// runtime --allow-routing flag is no longer required. Bare !{AI} (which
// desugars to mode=fixed) still requires explicit --allow-routing.
// =============================================================================

// mkIfaceWithEntry constructs an interface with a single export "main" whose
// type is `() -> () !{AI[mode=<mode>]}`. Used to build the typechecker
// fixture for safety-gate tests without spinning up the full pipeline.
//
// Pass mode="" to construct an entry with no AI in its effect row.
func mkIfaceWithEntry(t *testing.T, mode string) *iface.Iface {
	t.Helper()
	var effectRow *types.Row
	if mode != "" {
		var err error
		// Empty user params + Name=AI causes the elaborator to apply the
		// default ("fixed"). We need explicit params to override.
		if mode == "fixed" {
			// Bare !{AI} desugars to mode=fixed.
			effectRow, err = types.ElaborateEffectRow([]string{"AI"})
		} else {
			// We can't import internal/ast here without making the test heavier;
			// build the row manually with explicit params.
			effectRow = &types.Row{
				Kind:   types.EffectRow,
				Labels: map[string]types.Type{"AI": types.Unit()},
				Params: map[string]map[string]string{"AI": {"mode": mode}},
			}
			err = nil
		}
		if err != nil {
			t.Fatalf("ElaborateEffectRow(AI) failed: %v", err)
		}
	}
	scheme := &types.Scheme{
		Type: &types.TFunc2{
			Params:    nil,
			EffectRow: effectRow,
			Return:    types.Unit(),
		},
	}
	i := iface.NewIface("test")
	i.AddExport("main", scheme, false)
	return i
}

// TestDetermineDeclaredAIMode covers the four read-paths the safety-gate
// relaxation depends on.
func TestDetermineDeclaredAIMode(t *testing.T) {
	t.Run("nil interface returns empty", func(t *testing.T) {
		got := DetermineDeclaredAIMode(nil, "main")
		if got != "" {
			t.Errorf("DetermineDeclaredAIMode(nil) = %q, want \"\"", got)
		}
	})

	t.Run("missing entry returns empty", func(t *testing.T) {
		got := DetermineDeclaredAIMode(mkIfaceWithEntry(t, "routeable"), "nonexistent")
		if got != "" {
			t.Errorf("DetermineDeclaredAIMode(missing) = %q, want \"\"", got)
		}
	})

	t.Run("bare AI desugars to fixed", func(t *testing.T) {
		got := DetermineDeclaredAIMode(mkIfaceWithEntry(t, "fixed"), "main")
		if got != "fixed" {
			t.Errorf("DetermineDeclaredAIMode(bare AI) = %q, want \"fixed\"", got)
		}
	})

	t.Run("explicit routeable returns routeable", func(t *testing.T) {
		got := DetermineDeclaredAIMode(mkIfaceWithEntry(t, "routeable"), "main")
		if got != "routeable" {
			t.Errorf("DetermineDeclaredAIMode(routeable) = %q, want \"routeable\"", got)
		}
	})

	t.Run("explicit replay-only returns replay-only", func(t *testing.T) {
		got := DetermineDeclaredAIMode(mkIfaceWithEntry(t, "replay-only"), "main")
		if got != "replay-only" {
			t.Errorf("DetermineDeclaredAIMode(replay-only) = %q, want \"replay-only\"", got)
		}
	})

	t.Run("no AI in row returns empty", func(t *testing.T) {
		got := DetermineDeclaredAIMode(mkIfaceWithEntry(t, ""), "main")
		if got != "" {
			t.Errorf("DetermineDeclaredAIMode(no AI) = %q, want \"\"", got)
		}
	})
}

// TestResolveRoutingPolicy_ImplicitAllowFromRouteableMode is the load-bearing
// M-AI-EFFECT-MODES M2 acceptance test: a program declared
// !{AI[mode=routeable]} bypasses the runtime --allow-routing requirement.
func TestResolveRoutingPolicy_ImplicitAllowFromRouteableMode(t *testing.T) {
	values := routingFlagValues{
		fallback:     "anthropic,openai",
		allowRouting: false, // not passed
	}
	policy, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, "routeable"), "main")
	if err != nil {
		t.Fatalf("expected nil error (mode=routeable should bypass gate), got: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	if len(policy.Order) != 2 || policy.Order[0] != "anthropic" || policy.Order[1] != "openai" {
		t.Errorf("policy.Order = %v, want [anthropic openai]", policy.Order)
	}
}

// TestResolveRoutingPolicy_ImplicitAllowFromReplayOnlyMode covers the
// replay-only mode path.
func TestResolveRoutingPolicy_ImplicitAllowFromReplayOnlyMode(t *testing.T) {
	values := routingFlagValues{
		require:      "structured_outputs",
		allowRouting: false,
	}
	policy, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, "replay-only"), "main")
	if err != nil {
		t.Fatalf("expected nil error (mode=replay-only should bypass gate), got: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
}

// TestResolveRoutingPolicy_FixedModeStillRequiresAllowRouting verifies the
// gate is unchanged for bare !{AI} (which desugars to mode=fixed): the user
// must explicitly opt in via --allow-routing.
func TestResolveRoutingPolicy_FixedModeStillRequiresAllowRouting(t *testing.T) {
	values := routingFlagValues{
		fallback:     "anthropic",
		allowRouting: false,
	}
	_, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, "fixed"), "main")
	if err == nil {
		t.Fatal("expected safety-gate error for mode=fixed without --allow-routing")
	}
	if !strings.Contains(err.Error(), "--allow-routing") {
		t.Errorf("error should mention --allow-routing, got: %v", err)
	}
}

// TestResolveRoutingPolicy_FixedModeWithAllowRoutingPermits verifies the
// existing back-compat path: bare !{AI} + --allow-routing still works.
func TestResolveRoutingPolicy_FixedModeWithAllowRoutingPermits(t *testing.T) {
	values := routingFlagValues{
		fallback:     "anthropic",
		allowRouting: true,
	}
	policy, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, "fixed"), "main")
	if err != nil {
		t.Fatalf("expected nil error (--allow-routing passed), got: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
}

// TestResolveRoutingPolicy_NoAIInRowFallsThroughGate verifies that programs
// without AI in their entry effect row still go through the standard
// safety-gate (no relaxation). This is also the path for non-AILANG callers
// of resolveRoutingPolicy with no interface info.
func TestResolveRoutingPolicy_NoAIInRowFallsThroughGate(t *testing.T) {
	values := routingFlagValues{
		fallback:     "anthropic",
		allowRouting: false,
	}
	_, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, ""), "main")
	if err == nil {
		t.Fatal("expected safety-gate error for entry with no AI effect")
	}
	if !strings.Contains(err.Error(), "--allow-routing") {
		t.Errorf("error should mention --allow-routing, got: %v", err)
	}

	// Same with nil interface (e.g., non-module file).
	_, err = resolveRoutingPolicy(values, nil, "main")
	if err == nil {
		t.Fatal("expected safety-gate error for nil interface")
	}
}

// TestResolveRoutingPolicy_NoFlagsReturnsNilPolicy verifies the no-routing
// no-op path: when no --routing-* flags are set, returns (nil, nil)
// regardless of declared mode.
func TestResolveRoutingPolicy_NoFlagsReturnsNilPolicy(t *testing.T) {
	values := routingFlagValues{} // all empty
	for _, mode := range []string{"fixed", "routeable", "replay-only", ""} {
		t.Run(mode, func(t *testing.T) {
			policy, err := resolveRoutingPolicy(values, mkIfaceWithEntry(t, mode), "main")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if policy != nil {
				t.Errorf("expected nil policy when no flags set, got: %+v", policy)
			}
		})
	}
}
