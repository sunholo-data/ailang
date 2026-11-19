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
	"github.com/sunholo-data/ailang/internal/types"
)

func TestWarningToCheckJSON(t *testing.T) {
	cases := []struct {
		name string
		w    elaborate.Warning
		file string
		want checkJSONError
	}{
		{
			name: "TC_MATCH_003 names its file, position, message and fix",
			w:    &types.MatchWarning{Pos: ast.Pos{File: "lib/m.ail", Line: 7, Column: 39}, Arm: 3, Pattern: "Idle"},
			file: "main.ail",
			want: checkJSONError{Code: "TC_MATCH_003", File: "lib/m.ail", Line: 7, Column: 39,
				Message:    "arm 3 (Idle) of this match can never be reached: earlier arms match every value it matches",
				Suggestion: "remove the arm, or move it above the arm that already covers it"},
		},
		{
			name: "coded, positioned warning",
			w:    &pipeline.TakeAfterFlatMapWarning{Location: ast.Pos{File: "m.ail", Line: 7, Column: 3}},
			file: "m.ail",
			want: checkJSONError{Code: "LIST_TAKE_AFTER_FLATMAP", File: "m.ail", Line: 7, Column: 3,
				Message: "at m.ail:7:3 take(n, flatMap(f, xs)) materializes the entire flatMap result before take runs. Fix: use takeFlatMap(n, f, xs) to stop producing values once the prefix is full."},
		},
		{
			name: "TC_MATCH_005 names its own file, message and suggestion",
			w:    &pipeline.JSONIntArmWarning{Location: ast.Pos{File: "lib/json_use.ail", Line: 4, Column: 2}, Module: "lib/json_use"},
			file: "main.ail",
			want: checkJSONError{Code: "TC_MATCH_005", File: "lib/json_use.ail", Line: 4, Column: 2,
				Message:    (&pipeline.JSONIntArmWarning{}).Message(),
				Suggestion: (&pipeline.JSONIntArmWarning{}).Suggestion()},
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

// End to end: a passing check with a TC_MATCH_003 warning carries it in --json
// and in --format agent — on a cold compile and on a warm one, where the
// module is served from the compile cache and the type checker that finds the
// warning does not run (the warning is cached with the module).
func TestCheckMachineOutputsCarryWarnings(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "matchfail", "redundant_arm.ail")
	file := filepath.ToSlash(src)

	for _, run := range []string{"cold", "warm"} {
		stdout, stderr, code := runWithStdin(t, bin, "", "check", "--json", src)
		if code != 0 {
			t.Fatalf("%s: check --json exit %d\nstdout: %s\nstderr: %s", run, code, stdout, stderr)
		}
		var out checkJSONOutput
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("%s: bad JSON %q: %v", run, stdout, err)
		}
		if !out.Passed || len(out.Errors) != 0 {
			t.Fatalf("%s: want a pass with no errors, got %+v", run, out)
		}
		if len(out.Warnings) != 1 {
			t.Fatalf("%s: want 1 warning, got %+v", run, out.Warnings)
		}
		w := out.Warnings[0]
		if w.Code != "TC_MATCH_003" || filepath.ToSlash(w.File) != file || w.Line != 7 || w.Column != 39 || !strings.Contains(w.Message, "arm 3 (Idle)") {
			t.Errorf("%s: warning = %+v, want TC_MATCH_003 at %s:7:39 naming arm 3 (Idle)", run, w, file)
		}
	}

	stdout, stderr, code := runWithStdin(t, bin, "", "check", "--format", "agent", src)
	if code != 0 {
		t.Fatalf("check --format agent exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	// Paths print with the OS separator; compare slash-normalised.
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(stdout, "\r\n", "\n")), "\n")
	want := "warn: " + file + ":7:39 TC_MATCH_003: arm 3 (Idle) of this match can never be reached"
	if len(lines) != 2 || filepath.ToSlash(lines[0]) != "ok: "+file || !strings.HasPrefix(filepath.ToSlash(lines[1]), want) {
		t.Errorf("agent output:\n%s\nwant `ok: %s` then a line starting %q", stdout, file, want)
	}
}
