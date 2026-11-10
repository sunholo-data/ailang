package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

func TestWarningToCheckJSON(t *testing.T) {
	cases := []struct {
		name string
		w    elaborate.Warning
		file string
		want checkJSONError
	}{
		{
			name: "exhaustiveness warning in the checked file",
			w:    &elaborate.ExhaustivenessWarning{Location: "lit1:3:36", MissingPattern: []string{"_"}},
			file: "lit1.ail",
			want: checkJSONError{Code: "ELB004", Message: "non-exhaustive match — missing pattern: _", File: "lit1.ail", Line: 3, Column: 36,
				Suggestion: "add an arm for the missing pattern, or end with `_ => …`"},
		},
		{
			name: "exhaustiveness warning, module path matches a nested file",
			w:    &elaborate.ExhaustivenessWarning{Location: "pkg/a/lit:5:2", MissingPattern: []string{"_"}},
			file: "pkg/a/lit.ail",
			want: checkJSONError{Code: "ELB004", Message: "non-exhaustive match — missing pattern: _", File: "pkg/a/lit.ail", Line: 5, Column: 2,
				Suggestion: "add an arm for the missing pattern, or end with `_ => …`"},
		},
		{
			name: "exhaustiveness warning from an imported module names that module's file",
			w:    &elaborate.ExhaustivenessWarning{Location: "lib/other:9:4", MissingPattern: []string{"_"}},
			file: "main.ail",
			want: checkJSONError{Code: "ELB004", Message: "non-exhaustive match — missing pattern: _", File: "lib/other.ail", Line: 9, Column: 4,
				Suggestion: "add an arm for the missing pattern, or end with `_ => …`"},
		},
		{
			name: "coded, positioned warning",
			w:    &pipeline.TakeAfterFlatMapWarning{Location: ast.Pos{File: "m.ail", Line: 7, Column: 3}},
			file: "m.ail",
			want: checkJSONError{Code: "LIST_TAKE_AFTER_FLATMAP", File: "m.ail", Line: 7, Column: 3,
				Message: "at m.ail:7:3 take(n, flatMap(f, xs)) materializes the entire flatMap result before take runs. Fix: use takeFlatMap(n, f, xs) to stop producing values once the prefix is full."},
		},
		{
			name: "uncoded warning keeps the generic code and drops the prefix",
			w:    &pipeline.ArgOrderWarning{Location: ast.Pos{File: "s.ail", Line: 2, Column: 9}, FuncName: "split", Delimiter: ",", Arg1String: "s"},
			file: "s.ail",
			want: checkJSONError{Code: "WARNING", File: "s.ail", Line: 2, Column: 9},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := warningToCheckJSON(tc.w, tc.file)
			if tc.want.Message == "" {
				// Only the shape matters for the uncoded case.
				if strings.Contains(got.Message, "\n") || strings.HasPrefix(got.Message, "warning:") {
					t.Errorf("message not one line / still prefixed: %q", got.Message)
				}
				got.Message = ""
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
	if checkWarningsJSON(nil, "x.ail") != nil {
		t.Error("no warnings must give nil, so the JSON field is omitted")
	}
}

// End to end: a passing check with a warning carries it in --json and in
// --format agent. The compile cache is off: a warm cache drops elaborator
// warnings from every output today (pre-existing; see the PR).
func TestCheckMachineOutputsCarryWarnings(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "matchfail", "lit_warn.ail")
	file := filepath.ToSlash(src)

	stdout, stderr, code := runWithStdin(t, bin, "", "check", "--json", src)
	if code != 0 {
		t.Fatalf("check --json exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var out checkJSONOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("bad JSON %q: %v", stdout, err)
	}
	if !out.Passed || len(out.Errors) != 0 {
		t.Fatalf("want a pass with no errors, got %+v", out)
	}
	if len(out.Warnings) != 1 {
		t.Fatalf("want 1 warning, got %+v", out.Warnings)
	}
	w := out.Warnings[0]
	if w.Code != "ELB004" || filepath.ToSlash(w.File) != file || w.Line != 5 || w.Column != 36 || !strings.Contains(w.Message, "missing pattern: _") {
		t.Errorf("warning = %+v, want ELB004 at %s:5:36 naming the missing pattern", w, file)
	}

	stdout, stderr, code = runWithStdin(t, bin, "", "check", "--format", "agent", src)
	if code != 0 {
		t.Fatalf("check --format agent exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	// Paths print with the OS separator; compare slash-normalised.
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(stdout, "\r\n", "\n")), "\n")
	want := "warn: " + file + ":5:36 ELB004: non-exhaustive match — missing pattern: _"
	if len(lines) != 2 || filepath.ToSlash(lines[0]) != "ok: "+file || !strings.HasPrefix(filepath.ToSlash(lines[1]), want) {
		t.Errorf("agent output:\n%s\nwant `ok: %s` then a line starting %q", stdout, file, want)
	}
}
