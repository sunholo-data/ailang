package replay

import (
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// TestContractFor_Rand covers the three Rand pilot modes: seeded is
// deterministic (replay pins), os is re-sampleable (replay may redraw), crypto
// is opaque (replay substitutes from the harness).
func TestContractFor_Rand(t *testing.T) {
	cases := []struct {
		mode string
		want Contract
	}{
		{"seeded", Deterministic},
		{"os", ReSampleable},
		{"crypto", Opaque},
	}
	for _, tc := range cases {
		got, ok := ContractFor("Rand", tc.mode)
		if !ok {
			t.Fatalf("ContractFor(Rand, %q): expected a registered contract, got none", tc.mode)
		}
		if got != tc.want {
			t.Errorf("ContractFor(Rand, %q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// TestContractFor_AI covers the three AI label rows. AI dispatch already exists
// (M-AI-EFFECT-MODES); this sprint only registers the replay classification.
func TestContractFor_AI(t *testing.T) {
	cases := []struct {
		mode string
		want Contract
	}{
		{"fixed", Deterministic},
		{"routeable", ReSampleable},
		{"replay-only", Opaque},
	}
	for _, tc := range cases {
		got, ok := ContractFor("AI", tc.mode)
		if !ok {
			t.Fatalf("ContractFor(AI, %q): expected a registered contract, got none", tc.mode)
		}
		if got != tc.want {
			t.Errorf("ContractFor(AI, %q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// TestContractFor_Unregistered proves ContractFor returns ok=false (no silent
// fallback) for pairs with no registered contract: an unknown mode, an
// unknown effect, and a Clock mode whose port sprint has not landed.
func TestContractFor_Unregistered(t *testing.T) {
	cases := [][2]string{
		{"Rand", "banana"}, // legal effect, illegal mode
		{"Bogus", "os"},    // unknown effect
		{"Clock", "sim"},   // port sprint not landed
		{"AI", "byok"},     // scope value, not a mode
	}
	for _, tc := range cases {
		if c, ok := ContractFor(tc[0], tc[1]); ok {
			t.Errorf("ContractFor(%q, %q) = (%q, true), want ok=false", tc[0], tc[1], c)
		}
	}
}

// TestContractFor_RowCounts pins the registry shape: 3 Rand rows + 3 AI rows.
func TestContractFor_RowCounts(t *testing.T) {
	rand, ai := 0, 0
	for _, p := range registeredPairs() {
		switch p[0] {
		case "Rand":
			rand++
		case "AI":
			ai++
		}
	}
	if rand != 3 {
		t.Errorf("Rand rows = %d, want 3", rand)
	}
	if ai != 3 {
		t.Errorf("AI rows = %d, want 3", ai)
	}
	if total := len(registeredPairs()); total != 6 {
		t.Errorf("total registered pairs = %d, want 6", total)
	}
}

// TestReplayContractsAreLegalModes is the SPIKE-2 drift guard: every key in the
// replay taxonomy MUST be a legal (effect, mode) pair under internal/types'
// effectSchema (the single source of truth for legal modes). If someone adds a
// replay label for a mode that isn't legal — or a port sprint removes a mode —
// this fails, keeping the two tables in lockstep without duplicating the schema.
func TestReplayContractsAreLegalModes(t *testing.T) {
	pairs := registeredPairs()
	// Deterministic reporting order.
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}
		return pairs[i][1] < pairs[j][1]
	})
	for _, p := range pairs {
		effect, mode := p[0], p[1]
		if !types.IsLegalEffectMode(effect, mode) {
			t.Errorf("replay contract key (%q, %q) is NOT a legal mode per types.effectSchema — "+
				"the replay taxonomy has drifted from the validation table", effect, mode)
		}
	}
}
