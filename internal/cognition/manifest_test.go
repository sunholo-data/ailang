package cognition

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewManifest_SortsEffectsAndTransports(t *testing.T) {
	m := NewManifest(
		"agent_planner",
		[]string{"Msg", "DOM", "Trace"}, // intentionally unsorted
		map[string]int{"DOM.patches": 20, "Msg.sends": 100},
		[]string{"BroadcastChannel", "LocalWorker"}, // intentionally unsorted
	)

	wantEffects := []string{"DOM", "Msg", "Trace"}
	for i, e := range m.Effects {
		if e != wantEffects[i] {
			t.Errorf("Effects[%d]: got %q, want %q (manifest must be lex-sorted for replay determinism)", i, e, wantEffects[i])
		}
	}

	wantTransports := []string{"BroadcastChannel", "LocalWorker"}
	for i, tname := range m.Transports {
		if tname != wantTransports[i] {
			t.Errorf("Transports[%d]: got %q, want %q", i, tname, wantTransports[i])
		}
	}
}

func TestNewManifest_DefensiveBudgetsCopy(t *testing.T) {
	input := map[string]int{"DOM.patches": 20}
	m := NewManifest("a", []string{"DOM"}, input, nil)

	// Mutate the caller's map — the manifest should be unaffected.
	input["DOM.patches"] = 999

	if got := m.Budgets["DOM.patches"]; got != 20 {
		t.Errorf("manifest budget should be insulated from caller mutation, got %d", got)
	}
}

func TestNewManifest_NilBudgets_OmitsField(t *testing.T) {
	m := NewManifest("a", []string{"DOM"}, nil, nil)
	if m.Budgets != nil {
		t.Errorf("nil budgets input should produce nil Budgets, got %v", m.Budgets)
	}
}

func TestMarshalJSONIndent_DeterministicOutput(t *testing.T) {
	m := NewManifest(
		"agent_a",
		[]string{"DOM", "Msg"},
		map[string]int{"DOM.patches": 20, "Msg.sends": 100},
		[]string{"LocalWorker"},
	)

	out1, err := m.MarshalJSONIndent()
	if err != nil {
		t.Fatalf("MarshalJSONIndent: %v", err)
	}
	out2, err := m.MarshalJSONIndent()
	if err != nil {
		t.Fatalf("MarshalJSONIndent: %v", err)
	}
	if string(out1) != string(out2) {
		t.Errorf("MarshalJSONIndent must be byte-deterministic across calls — got differences")
	}

	// Round-trip
	var parsed CapabilityManifest
	if err := json.Unmarshal(out1, &parsed); err != nil {
		t.Fatalf("round-trip unmarshal failed: %v", err)
	}
	if parsed.Module != "agent_a" {
		t.Errorf("round-trip lost module: %q", parsed.Module)
	}
	if len(parsed.Effects) != 2 {
		t.Errorf("round-trip lost effects: %v", parsed.Effects)
	}
	if parsed.Budgets["DOM.patches"] != 20 {
		t.Errorf("round-trip lost budget value: %v", parsed.Budgets)
	}
}

func TestValidate_RejectsEmpty(t *testing.T) {
	cases := []struct {
		name string
		m    *CapabilityManifest
		want string // substring expected in error message
	}{
		{"nil", nil, "nil"},
		{"empty_module", &CapabilityManifest{Module: "", Effects: []string{"DOM"}}, "module"},
		{"empty_effects", &CapabilityManifest{Module: "m", Effects: nil}, "effects"},
		{"empty_effect_string", &CapabilityManifest{Module: "m", Effects: []string{"DOM", ""}}, "effects"},
		{"empty_budget_key", &CapabilityManifest{Module: "m", Effects: []string{"DOM"}, Budgets: map[string]int{"": 10}}, "budgets"},
		{"negative_budget", &CapabilityManifest{Module: "m", Effects: []string{"DOM"}, Budgets: map[string]int{"DOM.patches": -1}}, "negative"},
		{"empty_transport", &CapabilityManifest{Module: "m", Effects: []string{"DOM"}, Transports: []string{""}}, "transports"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.m.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error message %q should contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidate_AcceptsZeroBudget(t *testing.T) {
	// 0 means "forbidden" — should pass validation (distinct from negative)
	m := &CapabilityManifest{
		Module:  "m",
		Effects: []string{"DOM"},
		Budgets: map[string]int{"DOM.patches": 0},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("zero budget should be valid (means forbidden), got: %v", err)
	}
}

func TestValidateAgainstKnown_AcceptsKnownTransports(t *testing.T) {
	m := NewManifest("m", []string{"DOM"}, nil, []string{"LocalWorker", "BroadcastChannel"})
	if err := m.ValidateAgainstKnown(); err != nil {
		t.Errorf("known transports should validate, got: %v", err)
	}
}

func TestValidateAgainstKnown_RejectsUnknownTransport(t *testing.T) {
	m := NewManifest("m", []string{"DOM"}, nil, []string{"WebSocket"})
	err := m.ValidateAgainstKnown()
	if err == nil {
		t.Fatal("expected error for unknown transport (WebSocket is deferred to M-COG-MESH)")
	}
	if !strings.Contains(err.Error(), "WebSocket") {
		t.Errorf("error should name the unknown transport: %v", err)
	}
}
