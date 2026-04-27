package microrag

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// upStubSearcher returns canned per-namespace results.
type upStubSearcher struct {
	calls   int
	byNS    map[string][]SearchHit
	err     error
	queries []string // captured queries for inspection
	nss     []string // captured namespaces
}

func (s *upStubSearcher) Search(query, ns string, _ int) ([]SearchHit, error) {
	s.calls++
	s.queries = append(s.queries, query)
	s.nss = append(s.nss, ns)
	if s.err != nil {
		return nil, s.err
	}
	return s.byNS[ns], nil
}

func newTestEngineUP(t *testing.T, byNS map[string][]SearchHit) (*Engine, *upStubSearcher, string) {
	t.Helper()
	cfg := (&Config{Enabled: true}).applyDefaults()
	stub := &upStubSearcher{byNS: byNS}
	dir := t.TempDir()
	return &Engine{
		Cfg:        cfg,
		Searcher:   stub,
		SessionDir: dir,
		Now:        func() time.Time { return time.Unix(1000, 0) },
	}, stub, dir
}

func TestUserPrompt_EnvDisabledShortCircuits(t *testing.T) {
	t.Setenv(EnvEnabled, "0")
	eng, stub, _ := newTestEngineUP(t, map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.9, Content: "x", Namespace: "ailang-syntax", Key: "k"}},
	})
	res, err := eng.UserPrompt(UserPromptRequest{Prompt: "how do I concat strings in AILANG?"})
	if err != nil {
		t.Fatal(err)
	}
	if res.State != "disabled" {
		t.Errorf("state: got %q want disabled", res.State)
	}
	if stub.calls != 0 {
		t.Errorf("expected 0 search calls when disabled, got %d", stub.calls)
	}
}

func TestUserPrompt_TooShortSkipped(t *testing.T) {
	eng, stub, _ := newTestEngineUP(t, map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.9, Content: "x"}},
	})
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "hi"})
	if res.Reason != "prompt_too_short" {
		t.Errorf("reason: got %q want prompt_too_short", res.Reason)
	}
	if stub.calls != 0 {
		t.Errorf("short prompt must short-circuit, got %d calls", stub.calls)
	}
}

func TestUserPrompt_InjectsTopHitAcrossNamespaces(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax":   {{Score: 0.55, Content: "syntax-chunk", Namespace: "ailang-syntax", Key: "k1"}},
		"ailang-builtins": {{Score: 0.80, Content: "builtin-chunk", Namespace: "ailang-builtins", Key: "k2"}},
	}
	eng, stub, dir := newTestEngineUP(t, hits)
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "how do I concat strings in AILANG?"})
	if res.Injection == nil {
		t.Fatalf("expected injection; reason=%q", res.Reason)
	}
	if res.Injection.Namespace != "ailang-builtins" {
		t.Errorf("expected top hit from ailang-builtins (score 0.80); got %q", res.Injection.Namespace)
	}
	if !strings.Contains(res.Injection.InjectionText, "builtin-chunk") {
		t.Errorf("injection text missing builtin chunk: %q", res.Injection.InjectionText)
	}
	if stub.calls != 2 {
		t.Errorf("expected 2 namespace queries, got %d", stub.calls)
	}
	// Ledger entry written.
	ledger := readLedger(t, dir)
	if len(ledger) != 1 || ledger[0].SnippetID != res.Injection.SnippetID {
		t.Errorf("ledger missing or mismatched: %+v", ledger)
	}
	if !strings.HasPrefix(ledger[0].FilePath, "user-prompt://") {
		t.Errorf("ledger FilePath should be user-prompt://… got %q", ledger[0].FilePath)
	}
}

func TestUserPrompt_BelowFloorSuppresses(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax":   {{Score: 0.20, Content: "weak", Namespace: "ailang-syntax", Key: "a"}},
		"ailang-builtins": {{Score: 0.25, Content: "weak2", Namespace: "ailang-builtins", Key: "b"}},
	}
	eng, _, _ := newTestEngineUP(t, hits)
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "what's the weather like today?"})
	if res.Reason != "below_floor" {
		t.Errorf("reason: got %q want below_floor", res.Reason)
	}
	if res.Injection != nil {
		t.Error("expected no injection below floor")
	}
}

func TestUserPrompt_NoHitsReturnsCleanly(t *testing.T) {
	eng, _, _ := newTestEngineUP(t, map[string][]SearchHit{}) // empty
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "explain the AILANG type system in detail"})
	if res.Reason != "no_hits" {
		t.Errorf("reason: got %q want no_hits", res.Reason)
	}
}

func TestUserPrompt_DedupSuppressesRepeat(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.55, Content: "same", Namespace: "ailang-syntax", Key: "k"}},
	}
	eng, _, _ := newTestEngineUP(t, hits)
	first, _ := eng.UserPrompt(UserPromptRequest{Prompt: "how do I do string concat in AILANG?"})
	if first.Injection == nil {
		t.Fatalf("first call must inject; reason=%q", first.Reason)
	}
	second, _ := eng.UserPrompt(UserPromptRequest{Prompt: "and what about list concat in AILANG?"})
	if second.Reason != "dedup_suppressed" {
		t.Errorf("second call reason: got %q want dedup_suppressed", second.Reason)
	}
}

func TestUserPrompt_RelevanceBypassOverridesDedup(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.95, Content: "very-relevant", Namespace: "ailang-syntax", Key: "k"}},
	}
	eng, _, _ := newTestEngineUP(t, hits)
	// Default ailang-syntax bypass is 0.70; a 0.95 score easily exceeds it.
	eng.Cfg.Dedup.RelevanceBypass["ailang-syntax"] = 0.50
	first, _ := eng.UserPrompt(UserPromptRequest{Prompt: "the canonical AILANG string concat docs please"})
	if first.Injection == nil {
		t.Fatal("first call must inject")
	}
	second, _ := eng.UserPrompt(UserPromptRequest{Prompt: "another question about string concat in AILANG"})
	if second.Reason != "injected_bypass" {
		t.Errorf("reason: got %q want injected_bypass", second.Reason)
	}
}

func TestUserPrompt_SessionBudgetExhausts(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.55, Content: "x", Namespace: "ailang-syntax", Key: "k"}},
	}
	eng, _, dir := newTestEngineUP(t, hits)
	// Pre-fill ledger with entries that exceed the budget.
	pre := []LedgerEntry{
		{Tokens: eng.Cfg.SessionBudget, SnippetID: "warm", Namespace: "ailang-syntax", State: "on"},
	}
	writeLedger(t, dir, pre)
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "anything about AILANG syntax please"})
	if res.Reason != "session_budget_exhausted" {
		t.Errorf("reason: got %q want session_budget_exhausted", res.Reason)
	}
}

func TestUserPrompt_DryrunLogsButDoesNotInject(t *testing.T) {
	t.Setenv(EnvDryrun, "1")
	hits := map[string][]SearchHit{
		"ailang-syntax": {{Score: 0.55, Content: "x", Namespace: "ailang-syntax", Key: "k"}},
	}
	eng, _, dir := newTestEngineUP(t, hits)
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "how do I concat strings in AILANG?"})
	if res.Injection != nil {
		t.Error("dryrun must not emit injection")
	}
	if res.State != "dryrun" {
		t.Errorf("state: got %q want dryrun", res.State)
	}
	entries := readLedger(t, dir)
	if len(entries) != 1 || entries[0].State != "dryrun" {
		t.Errorf("dryrun must log to ledger, got %+v", entries)
	}
}

func TestUserPrompt_AllowlistFiltersNamespaces(t *testing.T) {
	t.Setenv(EnvRoutes, "ailang-builtins")
	hits := map[string][]SearchHit{
		"ailang-syntax":   {{Score: 0.95, Content: "syntax-only", Namespace: "ailang-syntax", Key: "a"}},
		"ailang-builtins": {{Score: 0.55, Content: "builtin-allowed", Namespace: "ailang-builtins", Key: "b"}},
	}
	eng, stub, _ := newTestEngineUP(t, hits)
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "how do I list operations in AILANG?"})
	if res.Injection == nil {
		t.Fatalf("expected injection from allowlisted namespace; reason=%q", res.Reason)
	}
	if res.Injection.Namespace != "ailang-builtins" {
		t.Errorf("expected ailang-builtins (only allowlisted ns); got %q", res.Injection.Namespace)
	}
	// Only the allowlisted namespace should have been queried.
	for _, ns := range stub.nss {
		if ns != "ailang-builtins" {
			t.Errorf("allowlist must skip non-listed namespaces; queried %q", ns)
		}
	}
}

func TestUserPrompt_SearchErrorDegrades(t *testing.T) {
	eng, stub, _ := newTestEngineUP(t, map[string][]SearchHit{})
	stub.err = errors.New("boom")
	res, _ := eng.UserPrompt(UserPromptRequest{Prompt: "anything about AILANG syntax please"})
	if res.Reason != "no_hits" {
		t.Errorf("reason: got %q want no_hits (search error must degrade gracefully)", res.Reason)
	}
}

func TestUserPrompt_CacheReusesAcrossCalls(t *testing.T) {
	hits := map[string][]SearchHit{
		"ailang-syntax":   {{Score: 0.55, Content: "x", Namespace: "ailang-syntax", Key: "k"}},
		"ailang-builtins": {{Score: 0.40, Content: "y", Namespace: "ailang-builtins", Key: "k"}},
	}
	eng, stub, _ := newTestEngineUP(t, hits)
	// First call: 2 underlying searches (one per ns).
	_, _ = eng.UserPrompt(UserPromptRequest{Prompt: "how do I concat strings in AILANG?"})
	first := stub.calls
	// Second call with same prompt: cache hits, no new search calls.
	_, _ = eng.UserPrompt(UserPromptRequest{Prompt: "how do I concat strings in AILANG?"})
	if stub.calls != first {
		t.Errorf("expected cached searches, got %d new calls", stub.calls-first)
	}
}

func TestUserPrompt_CustomNamespaces(t *testing.T) {
	hits := map[string][]SearchHit{
		"custom-ns": {{Score: 0.55, Content: "custom", Namespace: "custom-ns", Key: "k"}},
	}
	eng, stub, _ := newTestEngineUP(t, hits)
	res, _ := eng.UserPrompt(UserPromptRequest{
		Prompt:     "how do I do something in this domain?",
		Namespaces: []string{"custom-ns"},
	})
	if res.Injection == nil {
		t.Fatalf("expected injection from custom ns; reason=%q", res.Reason)
	}
	if stub.calls != 1 {
		t.Errorf("expected 1 search call (custom ns only), got %d", stub.calls)
	}
	if stub.nss[0] != "custom-ns" {
		t.Errorf("expected custom-ns query; got %q", stub.nss[0])
	}
}
