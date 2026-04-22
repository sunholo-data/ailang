package microrag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubResolver returns canned BuiltinSpecJSONs by name.
type stubResolver struct {
	specs map[string]*BuiltinSpecJSON
	err   error
	calls map[string]int
}

func (s *stubResolver) Resolve(name string) (*BuiltinSpecJSON, error) {
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[name]++
	if s.err != nil {
		return nil, s.err
	}
	if spec, ok := s.specs[name]; ok {
		return spec, nil
	}
	return nil, nil // mimic CLIBuiltinResolver "not a builtin"
}

func newConcatSpec() *BuiltinSpecJSON {
	return &BuiltinSpecJSON{
		Name:      "concat_String",
		Module:    "std/string",
		Signature: "concat_String: (string, string) -> string",
		IsPure:    true,
		Metadata: &struct {
			Description string `json:"description,omitempty"`
			Examples    []struct {
				Code        string `json:"code"`
				Description string `json:"description,omitempty"`
			} `json:"examples,omitempty"`
			Since string `json:"since,omitempty"`
		}{
			Description: "Concatenate two strings",
			Examples: []struct {
				Code        string `json:"code"`
				Description string `json:"description,omitempty"`
			}{{Code: `concat_String("hello", " world")`}},
		},
	}
}

func newLinter(t *testing.T, specs map[string]*BuiltinSpecJSON) (*Linter, *stubResolver, string) {
	t.Helper()
	res := &stubResolver{specs: specs}
	dir := t.TempDir()
	return &Linter{Resolver: res, SessionDir: dir}, res, dir
}

func TestLint_FirstUseEmitsNudge(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	t.Setenv(EnvDryrun, "")
	linter, _, _ := newLinter(t, map[string]*BuiltinSpecJSON{
		"concat_String": newConcatSpec(),
	})
	res, err := linter.Lint(LintRequest{
		FilePath: "foo.ail",
		Code:     `let s = concat_String("a", "b") in s`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Nudges) != 1 {
		t.Fatalf("expected 1 nudge, got %d (%+v)", len(res.Nudges), res)
	}
	if res.Nudges[0].Name != "concat_String" {
		t.Errorf("name: got %q want concat_String", res.Nudges[0].Name)
	}
	if !strings.Contains(res.Nudges[0].InjectionText, "concat_String") {
		t.Errorf("injection text missing name: %q", res.Nudges[0].InjectionText)
	}
	if res.Nudges[0].Tokens > 80 {
		t.Errorf("nudge over 80 tokens: %d", res.Nudges[0].Tokens)
	}
}

func TestLint_SecondCallSilent(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	linter, _, _ := newLinter(t, map[string]*BuiltinSpecJSON{
		"concat_String": newConcatSpec(),
	})
	code := `concat_String("a","b")`
	first, _ := linter.Lint(LintRequest{Code: code})
	if len(first.Nudges) != 1 {
		t.Fatalf("first call: expected 1 nudge, got %d", len(first.Nudges))
	}
	second, _ := linter.Lint(LintRequest{Code: code})
	if len(second.Nudges) != 0 {
		t.Errorf("second call must be silent, got %d nudges", len(second.Nudges))
	}
	if second.Reason != "no_first_use" {
		t.Errorf("reason: got %q want no_first_use", second.Reason)
	}
}

func TestLint_NonBuiltinIdentifiersIgnored(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	linter, res, _ := newLinter(t, map[string]*BuiltinSpecJSON{}) // resolver returns nil for everything
	out, _ := linter.Lint(LintRequest{Code: `let x = my_local_func(1) in foo(x)`})
	if len(out.Nudges) != 0 {
		t.Errorf("non-builtins must produce no nudges, got %d", len(out.Nudges))
	}
	// Both should be resolved exactly once and then cached.
	if res.calls["my_local_func"] != 1 || res.calls["foo"] != 1 {
		t.Errorf("expected one resolve per identifier, got %v", res.calls)
	}
}

func TestLint_KeywordsSkipped(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	linter, res, _ := newLinter(t, map[string]*BuiltinSpecJSON{})
	_, _ = linter.Lint(LintRequest{Code: `if (x) then foo() else bar()`})
	// "if" is a keyword — should never reach the resolver.
	if _, called := res.calls["if"]; called {
		t.Error("'if' keyword should not be resolved")
	}
}

func TestLint_HardCapMaxNudges(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	specs := map[string]*BuiltinSpecJSON{
		"a": newConcatSpec(), "b": newConcatSpec(), "c": newConcatSpec(), "d": newConcatSpec(),
	}
	linter, _, _ := newLinter(t, specs)
	out, _ := linter.Lint(LintRequest{Code: `a(); b(); c(); d()`})
	if len(out.Nudges) > 2 {
		t.Errorf("MaxNudges default 2 exceeded: %d", len(out.Nudges))
	}
}

func TestLint_DryrunEmitsNoNudge(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	t.Setenv(EnvDryrun, "1")
	linter, _, dir := newLinter(t, map[string]*BuiltinSpecJSON{
		"concat_String": newConcatSpec(),
	})
	res, _ := linter.Lint(LintRequest{Code: `concat_String("a","b")`})
	if len(res.Nudges) != 0 {
		t.Error("dryrun must suppress nudges")
	}
	if res.State != "dryrun" {
		t.Errorf("state: got %q want dryrun", res.State)
	}
	// Seen ledger must still be updated so the next non-dryrun call dedups.
	data, _ := os.ReadFile(filepath.Join(dir, "builtins_seen.txt"))
	if !strings.Contains(string(data), "concat_String") {
		t.Error("dryrun must still update builtins_seen.txt")
	}
}

func TestLint_DisabledByEnv(t *testing.T) {
	t.Setenv(EnvEnabled, "0")
	linter, res, _ := newLinter(t, map[string]*BuiltinSpecJSON{"concat_String": newConcatSpec()})
	out, _ := linter.Lint(LintRequest{Code: `concat_String("a","b")`})
	if out.State != "disabled" {
		t.Errorf("state: got %q want disabled", out.State)
	}
	if len(res.calls) != 0 {
		t.Errorf("disabled lint must not resolve, got %v", res.calls)
	}
}

func TestLint_ResolverErrorSkipsCandidate(t *testing.T) {
	t.Setenv(EnvEnabled, "1")
	linter, _, _ := newLinter(t, nil)
	linter.Resolver = &stubResolver{err: errors.New("boom")}
	out, err := linter.Lint(LintRequest{Code: `concat_String("a","b")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Nudges) != 0 {
		t.Errorf("resolver error must drop candidate, got %d nudges", len(out.Nudges))
	}
}

func TestExtractCandidates_Order(t *testing.T) {
	got := extractCandidates(`b(1); a(2); b(3); c(4)`)
	want := []string{"b", "a", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestExtractCandidates_UnderscorePrefix(t *testing.T) {
	got := extractCandidates(`_str_slice("hi", 0, 1)`)
	if len(got) != 1 || got[0] != "_str_slice" {
		t.Errorf("expected [_str_slice], got %v", got)
	}
}
