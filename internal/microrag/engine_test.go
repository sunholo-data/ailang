package microrag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubSearcher returns canned results without touching the brain.
type stubSearcher struct {
	calls   int
	results []SearchHit
	err     error
}

func (s *stubSearcher) Search(_, _ string, _ int) ([]SearchHit, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func newTestEngine(t *testing.T, hits []SearchHit) (*Engine, *stubSearcher, string) {
	t.Helper()
	cfg := (&Config{
		Enabled: true,
		Routes:  []Route{{Glob: "**/*.ail", KB: "ailang-syntax", MaxTokensPerInjection: 150, RelevanceFloor: 0.30}},
	}).applyDefaults()
	stub := &stubSearcher{results: hits}
	dir := t.TempDir()
	return &Engine{
		Cfg:        cfg,
		Searcher:   stub,
		SessionDir: dir,
		Now:        func() time.Time { return time.Unix(1000, 0) },
	}, stub, dir
}

func TestContext_EnvDisabledShortCircuits(t *testing.T) {
	t.Setenv(EnvEnabled, "0")
	eng, stub, _ := newTestEngine(t, []SearchHit{{Score: 0.9, Content: "x"}})
	res, err := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "a"})
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

func TestContext_RoutesAllowlistFilters(t *testing.T) {
	t.Setenv(EnvRoutes, "other-ns")
	eng, stub, _ := newTestEngine(t, []SearchHit{{Score: 0.9, Content: "x"}})
	res, _ := eng.Context(Request{ToolName: "Edit", FilePath: "foo.ail", Content: "a"})
	if res.Reason != "kb_not_in_allowlist" {
		t.Errorf("reason: got %q want kb_not_in_allowlist", res.Reason)
	}
	if stub.calls != 0 {
		t.Errorf("allowlist filter must short-circuit before search, got %d calls", stub.calls)
	}
}

func TestContext_NoRouteSkipsSearch(t *testing.T) {
	eng, stub, _ := newTestEngine(t, []SearchHit{{Score: 0.9}})
	res, _ := eng.Context(Request{ToolName: "Edit", FilePath: "README.md"})
	if res.Reason != "no_route" {
		t.Errorf("reason: got %q want no_route", res.Reason)
	}
	if stub.calls != 0 {
		t.Errorf("no-route must short-circuit, got %d calls", stub.calls)
	}
}

func TestContext_KBSkipShortCircuits(t *testing.T) {
	eng, stub, _ := newTestEngine(t, []SearchHit{{Score: 0.9}})
	eng.Cfg.Routes = []Route{{Glob: "**/CLAUDE.md", KB: "skip"}}
	res, _ := eng.Context(Request{ToolName: "Read", FilePath: "CLAUDE.md"})
	if res.Reason != "kb_skip" {
		t.Errorf("reason: got %q", res.Reason)
	}
	if stub.calls != 0 {
		t.Errorf("kb:skip must short-circuit, got %d calls", stub.calls)
	}
}

func TestContext_BelowRelevanceFloorSuppresses(t *testing.T) {
	eng, _, _ := newTestEngine(t, []SearchHit{{Score: 0.20, Content: "low-score"}})
	res, _ := eng.Context(Request{FilePath: "foo.ail"})
	if res.Reason != "below_floor" {
		t.Errorf("reason: got %q", res.Reason)
	}
	if res.Injection != nil {
		t.Error("expected no injection below floor")
	}
}

func TestContext_InjectsTopHit(t *testing.T) {
	hits := []SearchHit{
		{Score: 0.50, Content: "second", Namespace: "ailang-syntax", Key: "k2"},
		{Score: 0.80, Content: "winner", Namespace: "ailang-syntax", Key: "k1"},
	}
	eng, _, dir := newTestEngine(t, hits)
	res, _ := eng.Context(Request{FilePath: "foo.ail"})
	if res.Injection == nil {
		t.Fatalf("expected injection; reason=%q", res.Reason)
	}
	if res.Injection.SnippetID == "" {
		t.Error("snippet_id must be non-empty")
	}
	// Ledger must contain one entry.
	ledger := readLedger(t, dir)
	if len(ledger) != 1 {
		t.Errorf("ledger entries: got %d want 1", len(ledger))
	}
	if ledger[0].SnippetID != res.Injection.SnippetID {
		t.Error("ledger snippet_id mismatch")
	}
}

func TestContext_DedupSuppressesRepeatWithinWindow(t *testing.T) {
	hits := []SearchHit{{Score: 0.50, Content: "same", Namespace: "ailang-syntax", Key: "k"}}
	eng, _, _ := newTestEngine(t, hits)
	first, _ := eng.Context(Request{FilePath: "foo.ail"})
	if first.Injection == nil {
		t.Fatalf("first call must inject; reason=%q", first.Reason)
	}
	second, _ := eng.Context(Request{FilePath: "foo.ail"})
	if second.Reason != "dedup_suppressed" {
		t.Errorf("second call reason: got %q want dedup_suppressed", second.Reason)
	}
}

func TestContext_RelevanceBypassOverridesDedup(t *testing.T) {
	hits := []SearchHit{{Score: 0.95, Content: "very-relevant", Namespace: "ailang-syntax", Key: "k"}}
	eng, _, _ := newTestEngine(t, hits)
	// Configure a low bypass for ailang-syntax so 0.95 trivially exceeds it.
	eng.Cfg.Dedup.RelevanceBypass["ailang-syntax"] = 0.50
	first, _ := eng.Context(Request{FilePath: "foo.ail"})
	if first.Injection == nil {
		t.Fatal("first call must inject")
	}
	second, _ := eng.Context(Request{FilePath: "foo.ail"})
	if second.Injection == nil {
		t.Errorf("bypass must override dedup; reason=%q", second.Reason)
	}
	if second.Reason != "injected_bypass" {
		t.Errorf("reason: got %q want injected_bypass", second.Reason)
	}
}

func TestContext_DryrunLogsButDoesNotInject(t *testing.T) {
	t.Setenv(EnvDryrun, "1")
	hits := []SearchHit{{Score: 0.50, Content: "x", Namespace: "ailang-syntax", Key: "k"}}
	eng, _, dir := newTestEngine(t, hits)
	res, _ := eng.Context(Request{FilePath: "foo.ail"})
	if res.Injection != nil {
		t.Error("dryrun must not emit injection")
	}
	if res.State != "dryrun" {
		t.Errorf("state: got %q want dryrun", res.State)
	}
	// Ledger should still have the entry tagged dryrun.
	entries := readLedger(t, dir)
	if len(entries) != 1 || entries[0].State != "dryrun" {
		t.Errorf("dryrun must log to ledger, got %+v", entries)
	}
}

func TestContext_SearchCacheReusesResults(t *testing.T) {
	hits := []SearchHit{{Score: 0.50, Content: "x", Namespace: "ailang-syntax", Key: "k"}}
	eng, stub, _ := newTestEngine(t, hits)
	_, _ = eng.Context(Request{FilePath: "foo.ail", Content: "stable"})
	_, _ = eng.Context(Request{FilePath: "foo.ail", Content: "stable"})
	if stub.calls != 1 {
		t.Errorf("expected 1 underlying search (second call from cache), got %d", stub.calls)
	}
}

func TestContext_SessionBudgetExhausts(t *testing.T) {
	hits := []SearchHit{{Score: 0.50, Content: "x", Namespace: "ailang-syntax", Key: "k"}}
	eng, _, dir := newTestEngine(t, hits)
	// Pre-fill ledger with entries that exceed the budget.
	pre := []LedgerEntry{
		{Tokens: eng.Cfg.SessionBudget, SnippetID: "warm", Namespace: "ailang-syntax", State: "on"},
	}
	writeLedger(t, dir, pre)
	res, _ := eng.Context(Request{FilePath: "foo.ail"})
	if res.Reason != "session_budget_exhausted" {
		t.Errorf("reason: got %q want session_budget_exhausted", res.Reason)
	}
}

func TestContext_CorruptLedgerIgnored(t *testing.T) {
	hits := []SearchHit{{Score: 0.50, Content: "x", Namespace: "ailang-syntax", Key: "k"}}
	eng, _, dir := newTestEngine(t, hits)
	// Write garbage to the ledger.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "injections.jsonl"), []byte("{not json\n{bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _ := eng.Context(Request{FilePath: "foo.ail"})
	if res.Injection == nil {
		t.Errorf("corrupt ledger must not block injection; reason=%q", res.Reason)
	}
}

func TestApproxTokens(t *testing.T) {
	if approxTokens("") != 0 {
		t.Error("empty string should be 0 tokens")
	}
	if approxTokens("a") != 1 {
		t.Error("single char should be at least 1 token")
	}
	// 400 chars ~ 100 tokens
	got := approxTokens(string(make([]byte, 400)))
	if got < 80 || got > 120 {
		t.Errorf("approxTokens(400 chars) = %d, expected ~100", got)
	}
}

func TestEnabledFromEnvDefaults(t *testing.T) {
	t.Setenv(EnvEnabled, "")
	if !EnabledFromEnv() {
		t.Error("default should be enabled")
	}
	t.Setenv(EnvEnabled, "invalid")
	if !EnabledFromEnv() {
		t.Error("invalid value must default to enabled (don't fail closed)")
	}
	t.Setenv(EnvEnabled, "0")
	if EnabledFromEnv() {
		t.Error("'0' must disable")
	}
	t.Setenv(EnvEnabled, "false")
	if EnabledFromEnv() {
		t.Error("'false' must disable")
	}
}

// --- helpers --------------------------------------------------------------

func readLedger(t *testing.T, dir string) []LedgerEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "injections.jsonl"))
	if err != nil {
		return nil
	}
	var out []LedgerEntry
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var e LedgerEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			out = append(out, e)
		}
	}
	return out
}

func writeLedger(t *testing.T, dir string, entries []LedgerEntry) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf []byte
	for _, e := range entries {
		b, _ := json.Marshal(e)
		buf = append(buf, b...)
		buf = append(buf, '\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "injections.jsonl"), buf, 0o644); err != nil {
		t.Fatal(err)
	}
}

func splitLines(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// --- Anti-pattern hint extraction ----------------------------------------

func TestExtractAntiPatternHints_ModuleHyphen(t *testing.T) {
	content := "module mcp-tools/lib\n\nexport func main() = ()\n"
	hints := extractAntiPatternHints(content)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint for module hyphen, got none")
	}
	if !strings.Contains(strings.Join(hints, " "), "underscores") {
		t.Errorf("expected 'underscores' hint for module-hyphen pattern, got: %v", hints)
	}
}

func TestExtractAntiPatternHints_StringPlusPlus(t *testing.T) {
	cases := []string{
		`let s = "hello" ++ "world"`,
		`let s = "hello" ++ name`,
		`let s = name ++ "!"`,
	}
	for _, c := range cases {
		hints := extractAntiPatternHints(c)
		if len(hints) == 0 {
			t.Errorf("expected hint for string ++ in %q, got none", c)
			continue
		}
		joined := strings.Join(hints, " ")
		if !strings.Contains(joined, "list-only") && !strings.Contains(joined, "interpolation") {
			t.Errorf("expected list-only/interpolation hint for %q, got: %v", c, hints)
		}
	}
}

func TestExtractAntiPatternHints_PythonSyntax(t *testing.T) {
	cases := []string{
		"def hello():\n  pass\n",
		"class Foo:\n  pass\n",
		"for i in range(10):\n  print(i)\n",
		"while x < 10:\n  x = x+1\n",
	}
	for _, c := range cases {
		hints := extractAntiPatternHints(c)
		if len(hints) == 0 {
			t.Errorf("expected python-syntax hint for %q, got none", c)
		}
	}
}

func TestExtractAntiPatternHints_MarkdownFence(t *testing.T) {
	content := "```ailang\nmodule foo/bar\n```"
	hints := extractAntiPatternHints(content)
	if len(hints) == 0 {
		t.Fatal("expected markdown-fence hint")
	}
	if !strings.Contains(strings.Join(hints, " "), "fences") {
		t.Errorf("expected 'fences' hint, got: %v", hints)
	}
}

func TestExtractAntiPatternHints_NoMatch(t *testing.T) {
	content := "module foo/bar\n\nexport func main() = ()\n"
	hints := extractAntiPatternHints(content)
	if len(hints) != 0 {
		t.Errorf("expected no hints for clean code, got: %v", hints)
	}
}

func TestExtractAntiPatternHints_DedupSamePattern(t *testing.T) {
	// Two string-++ occurrences should yield only one hint, not duplicates.
	content := `let a = "x" ++ "y";` + "\n" + `let b = "p" ++ "q"`
	hints := extractAntiPatternHints(content)
	if len(hints) != 1 {
		t.Errorf("expected exactly one deduped hint, got %d: %v", len(hints), hints)
	}
}

func TestBuildQuery_EnrichesWithHints(t *testing.T) {
	req := Request{
		ToolName: "Write",
		FilePath: "/tmp/lib.ail",
		Content:  "module mcp-tools/lib\n",
	}
	q := buildQuery(req)
	if !strings.Contains(q, "underscores") {
		t.Errorf("expected query to contain hint terms; got: %q", q)
	}
	if !strings.Contains(q, "/tmp/lib.ail") {
		t.Errorf("expected query to still contain file path; got: %q", q)
	}
}

func TestBuildQuery_NoHintsPreservesOldBehaviour(t *testing.T) {
	req := Request{
		ToolName: "Read",
		FilePath: "/tmp/clean.ail",
		Content:  "module foo/bar\n",
	}
	q := buildQuery(req)
	expected := "/tmp/clean.ail\nmodule foo/bar\n"
	if q != expected {
		t.Errorf("expected unchanged query for clean content;\n  got: %q\n want: %q", q, expected)
	}
}
