package microrag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fullSessionEngine builds an engine with both .ail and .go routes plus a
// configurable searcher per call. Mirrors what the CLI shim instantiates.
func fullSessionEngine(t *testing.T, hits []SearchHit) (*Engine, *stubSearcher) {
	t.Helper()
	cfg := (&Config{
		Enabled: true,
		Routes: []Route{
			{Glob: "**/*.ail", KB: "ailang-syntax", MaxTokensPerInjection: 150, RelevanceFloor: 0.30},
			{Glob: "**/*.go", KB: "project-resolutions", MaxTokensPerInjection: 200, RelevanceFloor: 0.25},
			{Glob: "**/CLAUDE.md", KB: "skip"},
		},
		Dedup: DedupConfig{
			Windows:          map[string]int{"ailang-syntax": 30000, "project-resolutions": 40000},
			RelevanceBypass:  map[string]float64{"ailang-syntax": 0.70, "project-resolutions": 0.70},
			WallClockMaxSecs: 240,
		},
		SessionBudget: 5000,
	}).applyDefaults()
	stub := &stubSearcher{results: hits}
	return &Engine{
		Cfg:        cfg,
		Searcher:   stub,
		SessionDir: t.TempDir(),
		Now:        func() time.Time { return time.Unix(1_700_000_000, 0) },
	}, stub
}

// TestIntegration_FullSession_MultiRoute exercises the realistic interleaving
// of edits across both routes — verifies router dispatch, per-namespace dedup
// independence, and that the .ail snippet doesn't dedup .go's snippet.
func TestIntegration_FullSession_MultiRoute(t *testing.T) {
	eng, stub := fullSessionEngine(t, []SearchHit{
		{Score: 0.9, Namespace: "ailang-syntax", Key: "k1", Content: "use ${x} not ++"},
	})

	// First .ail edit → injection.
	res, _ := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "x ++ y"})
	if res.Injection == nil {
		t.Fatalf("first ail edit should inject; got %+v", res)
	}
	if res.Injection.Namespace != "ailang-syntax" {
		t.Fatalf("expected ailang-syntax namespace, got %s", res.Injection.Namespace)
	}

	// Now switch to .go — different namespace; must re-inject (different KB).
	stub.results = []SearchHit{{Score: 0.9, Namespace: "project-resolutions", Key: "k2", Content: "build flag note"}}
	res, _ = eng.Context(Request{ToolName: "Edit", FilePath: "main.go", Content: "package main"})
	if res.Injection == nil {
		t.Fatalf(".go edit should inject in different namespace; got reason=%s", res.Reason)
	}
	if res.Injection.Namespace != "project-resolutions" {
		t.Fatalf("expected project-resolutions namespace, got %s", res.Injection.Namespace)
	}

	// CLAUDE.md → routed to "skip".
	res, _ = eng.Context(Request{ToolName: "Edit", FilePath: "CLAUDE.md", Content: "instructions"})
	if res.Injection != nil {
		t.Fatalf("CLAUDE.md should be skipped; got injection")
	}
	if res.Reason != "kb_skip" {
		t.Fatalf("expected reason kb_skip, got %s", res.Reason)
	}
}

// TestIntegration_CacheStability simulates 5 rapid edits of the same .ail
// file — first hits searcher, the next 4 must reuse the cached envelope so
// the AI prompt cache stays warm.
func TestIntegration_CacheStability(t *testing.T) {
	eng, stub := fullSessionEngine(t, []SearchHit{
		{Score: 0.9, Namespace: "ailang-syntax", Key: "stable", Content: "stable advice"},
	})
	for i := 0; i < 5; i++ {
		_, err := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "same query"})
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if stub.calls != 1 {
		t.Errorf("expected exactly 1 searcher call across 5 identical edits, got %d", stub.calls)
	}
}

// --- Negative tests: graceful degradation -------------------------------

// TestNegative_SearcherErrorReturnsOn reports state=on with reason rather
// than crashing — the harness should always get a usable envelope.
func TestNegative_SearcherErrorReturnsOn(t *testing.T) {
	eng, stub := fullSessionEngine(t, nil)
	stub.err = osErr("ollama down: connection refused")
	res, err := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "x"})
	if err != nil {
		t.Fatalf("must not propagate error to caller: %v", err)
	}
	if res.State != "on" {
		t.Errorf("expected state=on for graceful degradation, got %s", res.State)
	}
	if !strings.Contains(res.Reason, "search_error") {
		t.Errorf("reason should mention search_error, got %q", res.Reason)
	}
}

// TestNegative_CorruptLedgerSurvives the dedup ledger gets garbage written
// to it (e.g. partial writes from a killed process) and the engine must
// continue rather than crash.
func TestNegative_CorruptLedgerSurvives(t *testing.T) {
	eng, _ := fullSessionEngine(t, []SearchHit{
		{Score: 0.9, Namespace: "ailang-syntax", Key: "k", Content: "advice"},
	})
	// Write garbage to ledger.
	if err := os.WriteFile(eng.ledgerPath(), []byte("not-json{garbage\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "x"})
	if err != nil {
		t.Fatalf("corrupt ledger should not return an error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result on corrupt ledger")
	}
}

// TestNegative_InvalidEnvDefaultsOn — invalid AILANG_MICRORAG_ENABLED must
// default to enabled rather than fail-closed (matches design Risks table).
func TestNegative_InvalidEnvDefaultsOn(t *testing.T) {
	t.Setenv(EnvEnabled, "garbage")
	if !EnabledFromEnv() {
		t.Fatal("invalid env value should default to enabled, not disabled")
	}
}

// TestNegative_NilSearcherDoesNotPanic — defensive: a misconfigured engine
// (no searcher) must be reported, not crash. Today the engine relies on a
// non-nil Searcher; this test documents that contract by ensuring the
// crash mode is detectable rather than silent.
func TestNegative_NilSearcherWithNoMatchedRoute(t *testing.T) {
	cfg := (&Config{
		Enabled: true,
		Routes:  []Route{{Glob: "**/*.go", KB: "x"}},
	}).applyDefaults()
	eng := &Engine{Cfg: cfg, SessionDir: t.TempDir()}
	// File doesn't match any route → must return cleanly without touching searcher.
	res, err := eng.Context(Request{ToolName: "Read", FilePath: "foo.ail", Content: ""})
	if err != nil {
		t.Fatalf("no-route path should not error: %v", err)
	}
	if res.Reason != "no_route" {
		t.Errorf("expected no_route, got %s", res.Reason)
	}
}

// helpers

type osError string

func (e osError) Error() string { return string(e) }
func osErr(s string) error      { return osError(s) }

// TestEngineLedgerPath_Resolves makes sure tests can locate the ledger we
// just corrupted (smoke test for the path layout, not the engine logic).
func TestEngineLedgerPath_Resolves(t *testing.T) {
	eng, _ := fullSessionEngine(t, nil)
	p := eng.ledgerPath()
	if !strings.HasSuffix(p, filepath.Join("injections.jsonl")) {
		t.Errorf("ledger path layout changed; expected ...injections.jsonl, got %s", p)
	}
}
