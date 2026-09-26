package executor

import "testing"

// The conformance property: equivalent work must produce a comparable canonical total
// on every harness, and the OLD guard quantity must not.
//
// The two rows are real measurements, not constructed numbers — `ailang mission role-run`
// 2026-09-14, two requests identical except the model, same executor role, same
// workspace, same instructions (read five named files one call each, then reply DONE).
// Their divergence under Input+Output is the defect this definition exists to close.
func TestTokensProcessed_ComparableAcrossHarnesses(t *testing.T) {
	measured := []struct {
		harness                            string
		input, cacheCreate, cacheRead, out int
	}{
		{"pi/minimax-m3", 35992, 0, 1024, 311},
		{"claude/haiku-4-5", 50, 44841, 172565, 648},
	}

	type row struct {
		harness         string
		oldGuard, canon int
	}
	var rows []row
	for _, m := range measured {
		r := &Result{
			InputTokens: m.input, CacheCreationInputTokens: m.cacheCreate,
			CacheReadInputTokens: m.cacheRead, OutputTokens: m.out,
		}
		rows = append(rows, row{m.harness, m.input + m.out, r.TokensProcessed()})
	}

	// The defect, pinned: the old quantity puts these two 52x apart.
	oldRatio := float64(rows[0].oldGuard) / float64(rows[1].oldGuard)
	if oldRatio < 40 {
		t.Errorf("Input+Output ratio between harnesses = %.0fx, expected the measured ~52x; "+
			"if this shrank, the measurement or a harness's normalisation changed and the "+
			"comment block in tokens_processed.go is now stale", oldRatio)
	}

	// The fix: the canonical quantity puts them within 2x. Not 1x — claude took 6 turns
	// to pi's 2 and carries its own system prompt — but the same order of magnitude,
	// which is what makes one cap mean one thing.
	canonRatio := float64(rows[1].canon) / float64(rows[0].canon)
	if canonRatio < 0.5 || canonRatio > 2.0 {
		t.Errorf("canonical ratio = %.2fx, want within [0.5, 2.0]; got %+v", canonRatio, rows)
	}

	// And it must be strictly larger than the old quantity wherever caching happened,
	// or the guard has not actually started seeing the cached prompt.
	if rows[1].canon <= rows[1].oldGuard {
		t.Errorf("claude: canonical %d not greater than old %d — cache-creation still invisible",
			rows[1].canon, rows[1].oldGuard)
	}
	t.Logf("pi:     old=%d canonical=%d", rows[0].oldGuard, rows[0].canon)
	t.Logf("claude: old=%d canonical=%d", rows[1].oldGuard, rows[1].canon)
}

// Cache READS are excluded on purpose. Including them would measure conversation
// length: the real 23-turn stage below processed 101,542 but re-read 679,040 from cache,
// so a guard counting reads would have fired at 6.7x its actual work.
func TestTokensProcessed_ExcludesCacheReads(t *testing.T) {
	r := &Result{InputTokens: 99822, OutputTokens: 1720, CacheReadInputTokens: 679040}
	if got, want := r.TokensProcessed(), 101542; got != want {
		t.Fatalf("TokensProcessed() = %d, want %d (cache reads must not count)", got, want)
	}
}

// Reasoning tokens count: they are billed at the output rate and are real work, and a
// model that thinks in circles is exactly what a thrash guard is for. They are DISJOINT
// from OutputTokens by the Result contract, so adding them cannot double-count.
func TestTokensProcessed_CountsReasoning(t *testing.T) {
	plain := &Result{InputTokens: 100, OutputTokens: 50}
	thinking := &Result{InputTokens: 100, OutputTokens: 50, ReasonTokens: 9000}
	if thinking.TokensProcessed() <= plain.TokensProcessed() {
		t.Fatal("hidden reasoning is invisible to the guard — a model can think in circles for free")
	}
	if got, want := thinking.TokensProcessed(), 9150; got != want {
		t.Fatalf("got %d, want %d", got, want)
	}
}

// A malformed stream must never be able to shrink the total and buy more budget.
func TestTokensProcessed_ClampsNegatives(t *testing.T) {
	r := &Result{InputTokens: 1000, CacheCreationInputTokens: -5000, OutputTokens: 20, ReasonTokens: -1}
	if got, want := r.TokensProcessed(), 1020; got != want {
		t.Fatalf("got %d, want %d — negatives must clamp, not subtract", got, want)
	}
	if (*Result)(nil).TokensProcessed() != 0 {
		t.Fatal("nil Result must be 0, not a panic")
	}
}

// The in-flight helper must agree with the Result method, since the guards accumulate
// counters before any Result exists. If these ever diverge the guard and the banked
// row disagree about the same run.
func TestTokensProcessedFrom_AgreesWithResult(t *testing.T) {
	for _, c := range [][4]int{{35992, 0, 311, 0}, {50, 44841, 648, 0}, {1, 2, 3, 4}, {-1, -2, -3, -4}} {
		r := &Result{InputTokens: c[0], CacheCreationInputTokens: c[1], OutputTokens: c[2], ReasonTokens: c[3]}
		if got, want := TokensProcessedFrom(c[0], c[1], c[2], c[3]), r.TokensProcessed(); got != want {
			t.Errorf("TokensProcessedFrom%v = %d, Result.TokensProcessed() = %d", c, got, want)
		}
	}
}
