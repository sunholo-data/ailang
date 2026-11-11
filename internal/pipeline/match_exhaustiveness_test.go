package pipeline

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-MATCH-EXHAUSTIVENESS M2 through the module pipeline: the universe of a
// constructor's type is that type's FULL constructor list, keyed by its
// nominal identity (defining module, name) — never the constructors a module
// happens to import, and never a bare name two modules share.

// checkMatchModules writes files (path -> source) under a fresh directory,
// checks entry and returns the match errors (nil when the check passed) and
// the TC_MATCH_003 warnings.
func checkMatchModules(t *testing.T, files map[string]string, entry string) ([]*types.MatchError, []*types.MatchWarning, error) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	for path, src := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: entry}, productionCacheDependencies())
	var warns []*types.MatchWarning
	for _, w := range res.Warnings {
		if mw, ok := w.(*types.MatchWarning); ok {
			warns = append(warns, mw)
		}
	}
	if err == nil {
		return nil, warns, nil
	}
	var many types.MatchErrors
	if errors.As(err, &many) {
		return many, warns, err
	}
	var one *types.MatchError
	if errors.As(err, &one) {
		return []*types.MatchError{one}, warns, err
	}
	return nil, warns, err
}

// missingOf renders each error as "line:CODE:message-tail".
func missingOf(errs []*types.MatchError) []string {
	out := make([]string, len(errs))
	for i, e := range errs {
		msg := e.Message
		if j := strings.Index(msg, "no arm matches: "); j >= 0 {
			msg = msg[j+len("no arm matches: "):]
		}
		out[i] = strings.Join([]string{itoa(e.Pos.Line), e.Code, msg}, ":")
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

const msgModule = "module lib/a\n\nexport type Msg = Message(string) | Ping\nexport type T = A | B | C\n"
const otherT = "module lib/b\n\nexport type T = A | Z\n"

func TestMatchExhaustivenessAcrossModules(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		want  []string // "line:CODE:missing"; empty = the check passes
		other error
	}{
		{
			// A by-name pattern constrains nothing: here only the application
			// decides the lambda parameter's type, after the patterns were
			// checked. The recorded type must follow the substitution.
			name: "by-name constructors, type decided after the patterns",
			src: "module main\n\nimport std/list (nth)\n\n" +
				"export pure func f(xs: [int]) -> int =\n  (\\o. match o { None => 0, Some(v) => v })(nth(xs, 0))\n",
		},
		{
			// Msg is loaded only through lib/hold. Nothing fixes x's type, so
			// g(42) would fall through both arms: never call it exhaustive.
			name: "by-name constructors on a value nothing types",
			src: "module main\n\nimport lib/hold (one)\n\n" +
				"export pure func g[a](x: a) -> int = match x {\n  Ping => 1,\n  Message(_) => 2\n}\n\n" +
				"export pure func h[a](x: a) -> int = match x {\n  Ping => 1,\n  _ => 2\n}\n",
			want: []string{"5:TC_MATCH_002:this match cannot be checked for exhaustiveness: constructor Ping is matched by name and nothing fixes the type of the value it is matched against (a); import the constructor's type, or annotate that value"},
		},
		{
			// The design's in-scope registry trap: importing ONE constructor
			// must not shrink the type to that constructor.
			name: "single constructor imported",
			src: "module main\n\nimport lib/a (Msg, Message)\n\n" +
				"export pure func f(m: Msg) -> int = match m {\n  Message(_) => 1\n}\n",
			want: []string{"5:TC_MATCH_002:Ping"},
		},
		{
			name: "single constructor imported, other matched by name, covered",
			src: "module main\n\nimport lib/a (Msg, Message)\n\n" +
				"export pure func f(m: Msg) -> int = match m {\n  Message(_) => 1,\n  Ping => 2\n}\n",
		},
		{
			// Only the type is imported; its constructors are matched by name
			// (#323) and take their identity from the scrutinee's type.
			name: "constructors matched by name",
			src: "module main\n\nimport lib/a (Msg)\n\n" +
				"export pure func f(m: Msg) -> int = match m {\n  Ping => 2\n}\n",
			want: []string{"5:TC_MATCH_002:Message(_)"},
		},
		{
			// Two modules each declare a type T and a constructor A.
			name: "same-named types and constructors in two modules",
			src: "module main\n\nimport lib/a (T as AT, A as AA, B, C)\nimport lib/b (T, A, Z)\n\n" +
				"export pure func fa(t: AT) -> int = match t {\n  AA => 1,\n  C => 3\n}\n\n" +
				"export pure func fb(t: T) -> int = match t {\n  A => 1\n}\n\n" +
				"export pure func ok(t: T) -> int = match t {\n  A => 1,\n  Z => 2\n}\n",
			want: []string{"6:TC_MATCH_002:B", "11:TC_MATCH_002:Z"},
		},
		{
			name: "prelude Option through std/list without importing Option",
			src: "module main\n\nimport std/list (nth)\n\n" +
				"export pure func f(xs: [int]) -> int = match nth(xs, 0) {\n  Some(v) => v\n}\n\n" +
				"export pure func g(xs: [int]) -> int = match nth(xs, 0) {\n  None => 0,\n  Some(v) => v\n}\n",
			want: []string{"5:TC_MATCH_002:None"},
		},
		{
			name: "nested prelude types",
			src: "module main\n\nimport std/option (Option, Some, None)\nimport std/result (Result, Ok, Err)\n\n" +
				"export pure func f(o: Option[Result[int, string]]) -> int = match o {\n  Some(Ok(n)) => n,\n  None => 0\n}\n",
			want: []string{"6:TC_MATCH_002:Some(Err(_))"},
		},
		{
			name: "contract body",
			src: "module main\n\nimport lib/a (T, A, B, C)\n\n" +
				"export pure func f(t: T) -> int\nensures { match t { A => result == 1, B => result == 2 } } {\n  match t {\n    A => 1,\n    B => 2,\n    C => 3\n  }\n}\n",
			want: []string{"6:TC_MATCH_002:C"},
		},
		{
			// #2243: the evaluator never matches a wrong-arity pattern, the VM
			// binds a prefix of the fields; coverage is undefined, so it is an
			// error.
			name: "constructor pattern with the wrong arity",
			src: "module main\n\nimport lib/a (Msg, Message, Ping)\n\n" +
				"export pure func f(m: Msg) -> int = match m {\n  Message(a, b) => 1,\n  _ => 0\n}\n",
			want: []string{"6:TC_MATCH_006:constructor pattern Message has 2 argument(s), but Message takes 1"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs, _, err := checkMatchModules(t, map[string]string{
				"lib/a.ail": msgModule, "lib/b.ail": otherT, "main.ail": c.src,
				"lib/hold.ail": "module lib/hold\n\nimport lib/a (T)\n\nexport pure func one() -> int = 1\n",
			}, "main.ail")
			if err != nil && errs == nil {
				t.Fatalf("check failed with a non-match error: %v", err)
			}
			got := missingOf(errs)
			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Errorf("got  %q\nwant %q\n(error: %v)", got, c.want, err)
			}
		})
	}
}

// TC_MATCH_003 survives the compile cache: the second check is served from
// the cache and must report the same warning.
func TestMatchRedundantArmWarningIsCached(t *testing.T) {
	src := "module main\n\ntype P = Idle | Running(int) | Done\n\n" +
		"export pure func f(p: P) -> int = match p {\n  Running(n) => n,\n  other => 0,\n  Idle => 1\n}\n"
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	if err := os.WriteFile("main.ail", []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, run := range []string{"cold", "warm"} {
		res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err != nil {
			t.Fatalf("%s: %v", run, err)
		}
		var got []string
		for _, w := range res.Warnings {
			if mw, ok := w.(*types.MatchWarning); ok {
				got = append(got, mw.String())
			}
		}
		want := "warning: TC_MATCH_003 at main.ail:5:35: arm 3 (Idle) of this match can never be reached: earlier arms match every value it matches"
		if len(got) != 1 || got[0] != want {
			t.Errorf("%s: warnings %q, want [%q]", run, got, want)
		}
	}
}

// The single-file path (pipeline_single.go: embedded Eval, IsREPL) gives the
// same verdict on the declaration it type-checks.
func TestMatchExhaustivenessSingleExpressionPath(t *testing.T) {
	// Source with no file name routes to runSingleWithContext, which
	// type-checks the first declaration.
	for _, c := range []struct {
		code, want string
	}{
		{"match 3 { 0 => \"z\", 1 => \"o\" }", "no arm matches: _ (any int other than 0, 1)"},
		{"type C = R | G\n\\c. match c { R => 1 }", "no arm matches: G"},
		{"match [1] { [] => 0, [h, ...t] => h }", ""},
	} {
		_, err := Run(Config{Mode: ModeCheck}, Source{Code: c.code})
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", c.code, err)
			}
			continue
		}
		var me *types.MatchError
		if !errors.As(err, &me) || me.Code != types.TC_MATCH_002 || !strings.HasSuffix(me.Message, c.want) {
			t.Errorf("%s: want TC_MATCH_002 ending %q, got %v", c.code, c.want, err)
		}
	}
}

// TestExpectedFail_MatchNonExhaustive pins examples/expected_fail/match_nonexhaustive_bool.ail
// (rejected) and its twin examples/runnable/match_exhaustiveness.ail (accepted).
func TestExpectedFail_MatchNonExhaustive(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository go.mod not found")
		}
		root = parent
	}
	t.Chdir(root)
	_, err = Run(Config{Mode: ModeCheck}, Source{Filename: "examples/expected_fail/match_nonexhaustive_bool.ail"})
	var me *types.MatchError
	if !errors.As(err, &me) || me.Code != types.TC_MATCH_002 || !strings.HasSuffix(me.Message, "no arm matches: false") {
		t.Fatalf("want TC_MATCH_002 naming false, got %v", err)
	}
	if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "examples/runnable/match_exhaustiveness.ail"}); err != nil {
		t.Fatalf("the exhaustive twin was rejected: %v", err)
	}
}
