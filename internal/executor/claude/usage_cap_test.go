package claude

import "testing"

// The claude harness's accounting contract, pinned.
//
// claude-code emits CUMULATIVE usage in message_delta, so claude.go ASSIGNS the latest
// value; pi and opencode emit deltas and SUM. Nothing asserted which was which before,
// so a provider changing stream shape would have silently corrupted every cap and cost
// on this lane rather than failing. These tests pin the pieces that are reachable; see
// TestUsageSemantics_DocumentedPerHarness in the executor package for the shape table.
func TestCapExceeded_CountsCachedPromptCreation(t *testing.T) {
	// Real measurement, 2026-09-14: a five-file read on claude-haiku-4-5.
	const input, cacheCreation, output = 50, 44841, 648

	// The defect: Input+Output is 698, so any cap above that passed a run that actually
	// processed 45,539.
	if tp, over := capExceeded(1000, input, cacheCreation, output); !over {
		t.Errorf("processed %d must breach a 1000 cap (Input+Output alone reads %d)", tp, input+output)
	}
	if tp, _ := capExceeded(100000, input, cacheCreation, output); tp != 45539 {
		t.Errorf("processed = %d, want 45539", tp)
	}
	// A cap that genuinely fits must not fire.
	if _, over := capExceeded(100000, input, cacheCreation, output); over {
		t.Error("45,539 must not breach a 100,000 cap")
	}
	// maxTokens <= 0 means no cap, matching every other guard in the tree.
	for _, noCap := range []int{0, -1} {
		if _, over := capExceeded(noCap, input, cacheCreation, output); over {
			t.Errorf("maxTokens=%d must mean no cap", noCap)
		}
	}
}

// A negative residual would REFUND tokens into the cost budget, buying an over-spending
// run more room. message_delta values can arrive out of order or duplicated, so this is
// a real input, not a hypothetical.
func TestResidual_NeverRefunds(t *testing.T) {
	for _, c := range []struct{ final, running, want int }{
		{100, 40, 60}, // normal forward progress
		{100, 100, 0}, // already reconciled
		{40, 100, 0},  // out-of-order or duplicated event: clamp, never negative
		{0, 500, 0},   // missing final usage must not refund 500 tokens
	} {
		if got := residual(c.final, c.running); got != c.want {
			t.Errorf("residual(%d, %d) = %d, want %d", c.final, c.running, got, c.want)
		}
	}
}
