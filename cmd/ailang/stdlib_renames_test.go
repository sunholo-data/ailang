package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/importhint"
)

// M-V1-STDLIB-SURFACE-POLISH M1/M2: every deprecated std spelling still runs,
// gives exactly the replacement's result on both engines, and warns once per
// use on stderr only — stdout stays the program's.
//
// aliases.ail uses all 16 names; it runs on the evaluator and the default
// bytecode VM (the strict VM does not yet run the FS and Rand builtins the
// rand/fs aliases forward to — pre-existing, BC001 "not yet wired"). The pure
// aliases also run on the STRICT VM in aliases_pure.ail.
func TestStdlibRenameAliases(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	all := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "aliases.ail")
	pure := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "aliases_pure.ail")
	type engineRun struct {
		args     []string
		warnings int
		pairs    int
		stderr   string
	}
	fixtures := map[string]map[string]engineRun{
		"all": {
			"evaluator": {[]string{"run", "--quiet", "--caps", "IO,FS,Rand", "--entry", "main", all}, 16, 14, "printErr|eprint|"},
			"vm":        {[]string{"run", "--quiet", "--bytecode", "--caps", "IO,FS,Rand", "--entry", "main", all}, 16, 14, "printErr|eprint|"},
		},
		"pure": {
			"evaluator": {[]string{"run", "--quiet", "--caps", "IO", "--entry", "main", pure}, 10, 10, ""},
			"strict-vm": {[]string{"run", "--quiet", "--bytecode", "--strict-bytecode", "--caps", "IO", "--entry", "main", pure}, 10, 10, ""},
		},
	}
	lf := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
	for _, fixture := range []string{"all", "pure"} {
		outs := map[string]string{}
		for engine, er := range fixtures[fixture] {
			for _, run := range []string{"cold", "warm"} {
				tag := fixture + "/" + engine + "/" + run
				stdout, stderr, code := runWithStdin(t, bin, "", er.args...)
				if code != 0 {
					t.Fatalf("%s: exit %d\nstderr: %s", tag, code, stderr)
				}
				stdout, stderr = lf(stdout), lf(stderr)
				outs[engine] = stdout
				if strings.Contains(stdout, "TC_DEPRECATED_001") {
					t.Errorf("%s: a deprecation warning reached stdout", tag)
				}
				if n := strings.Count(stderr, "TC_DEPRECATED_001 at "); n != er.warnings {
					t.Errorf("%s: %d deprecation warnings on stderr, want %d\n%s", tag, n, er.warnings, stderr)
				}
				if er.stderr != "" && !strings.Contains(stderr, er.stderr) {
					t.Errorf("%s: printErr and eprint must both write stderr without a newline: %q", tag, stderr)
				}
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				if len(lines) != er.pairs+1 || lines[len(lines)-1] != "done" {
					t.Fatalf("%s: transcript has %d lines, want %d pairs + done:\n%s", tag, len(lines), er.pairs, stdout)
				}
				for _, l := range lines[:len(lines)-1] {
					name, results, ok := strings.Cut(l, ": ")
					old, new, ok2 := strings.Cut(results, " | ")
					if !ok || !ok2 || old != new || old == "" {
						t.Errorf("%s: %s: old and new spellings disagree: %q vs %q", tag, name, old, new)
					}
				}
			}
		}
		var first string
		for engine, out := range outs {
			if first == "" {
				first = out
			} else if out != first {
				t.Errorf("%s: engine %s disagrees:\n%s", fixture, engine, lineDiff(first, out))
			}
		}
	}
}

// check --json carries one TC_DEPRECATED_001 per use, with the replacement in
// the message and the fix, on a cold and a warm cache; --format agent has a
// warn: line for each; and the human output names the replacement.
func TestDeprecatedWarningCheckOutputs(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	src := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "aliases.ail")
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runWithStdin(t, bin, "", "check", src)
		if code != 0 || strings.Count(stderr, "TC_DEPRECATED_001 at ") != 16 ||
			!strings.Contains(stderr, "std/list.head_or is deprecated; use headOr") {
			t.Errorf("%s: human check: exit %d, stderr:\n%s", run, code, stderr)
		}

		stdout, _, code := runWithStdin(t, bin, "", "check", "--json", src)
		var out struct {
			Passed   bool `json:"passed"`
			Warnings []struct {
				Code       string `json:"code"`
				File       string `json:"file"`
				Line       int    `json:"line"`
				Message    string `json:"message"`
				Suggestion string `json:"suggestion"`
			} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("%s: check --json: %v\n%s", run, err, stdout)
		}
		var named []string
		for _, w := range out.Warnings {
			if w.Code != "TC_DEPRECATED_001" {
				continue
			}
			if w.Line == 0 || !strings.HasSuffix(filepath.ToSlash(w.File), "stdrenames/aliases.ail") ||
				!strings.Contains(w.Message, " is deprecated; use ") || w.Suggestion == "" {
				t.Errorf("%s: malformed warning %+v", run, w)
			}
			named = append(named, w.Message)
		}
		if code != 0 || !out.Passed || len(named) != 16 {
			t.Errorf("%s: check --json: exit %d passed %v, %d TC_DEPRECATED_001 warnings, want 16\n%s", run, code, out.Passed, len(named), stdout)
		}
		// Each deprecated name exactly once.
		for _, r := range importhint.StdlibRenames {
			want := r.Module + "." + r.Old + " is deprecated; use " + r.New
			n := 0
			for _, m := range named {
				if m == want {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%s: %q appears %d times, want 1", run, want, n)
			}
		}

		stdout, _, _ = runWithStdin(t, bin, "", "check", "--format", "agent", src)
		if strings.Count(stdout, "TC_DEPRECATED_001") != 16 {
			t.Errorf("%s: --format agent: want 16 TC_DEPRECATED_001 lines:\n%s", run, stdout)
		}
	}
}

// `ailang iface std/<m>` carries tier and replaced_by on a cold AND a warm
// cache (the cache's iface.json codec used to drop both), and the compact view
// marks deprecated and experimental functions.
func TestIfaceTierSurvivesWarmCache(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	for _, run := range []string{"cold", "warm"} {
		stdout, stderr, code := runWithStdin(t, bin, "", "iface", "std/list")
		if code != 0 {
			t.Fatalf("%s: iface exit %d: %s", run, code, stderr)
		}
		var g struct {
			Funcs []struct {
				Name       string `json:"name"`
				Tier       string `json:"tier"`
				ReplacedBy string `json:"replaced_by"`
			} `json:"funcs"`
		}
		if err := json.Unmarshal([]byte(stdout), &g); err != nil {
			t.Fatal(err)
		}
		marks := map[string]string{}
		for _, f := range g.Funcs {
			if f.Tier == "" {
				t.Errorf("%s: std/list.%s has no tier", run, f.Name)
			}
			marks[f.Name] = f.Tier + "/" + f.ReplacedBy
		}
		if marks["head_or"] != "stable/headOr" || marks["headOr"] != "stable/" || marks["takeMap"] != "experimental/" {
			t.Errorf("%s: marks head_or=%q headOr=%q takeMap=%q", run, marks["head_or"], marks["headOr"], marks["takeMap"])
		}
		compact, _, _ := runWithStdin(t, bin, "", "iface", "--compact", "std/list")
		if !strings.Contains(compact, "head_or : (list[a],a)->a  -- deprecated: use headOr") ||
			!strings.Contains(compact, "takeMap : (int,(a)->b,list[a])->list[b]  -- experimental") ||
			!strings.Contains(compact, "headOr : (list[a],a)->a\n") {
			t.Errorf("%s: compact iface lacks the marks:\n%s", run, compact)
		}
	}
}

// `ailang docs` (a line scraper plus the parser) and the goldens (the compiler)
// agree on every std function's tier and replacement — two readers of one
// annotation (§6.1 item 3).
func TestDocsTiersMatchGoldens(t *testing.T) {
	stdDir := filepath.Join("..", "..", "std")
	goldenDir := filepath.Join("..", "..", ".stdlib-golden")
	goldens, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	if err != nil || len(goldens) == 0 {
		t.Fatalf("no goldens: %v", err)
	}
	sort.Strings(goldens)
	checked := 0
	for _, gp := range goldens {
		mod := strings.TrimSuffix(filepath.Base(gp), ".json")
		data, err := os.ReadFile(gp)
		if err != nil {
			t.Fatal(err)
		}
		var g struct {
			Funcs []struct {
				Name            string `json:"name"`
				Tier            string `json:"tier"`
				ReplacedBy      string `json:"replaced_by"`
				DeprecationNote string `json:"deprecation_note"`
			} `json:"funcs"`
		}
		if err := json.Unmarshal(data, &g); err != nil {
			t.Fatal(err)
		}
		marks, _, err := parseExportMarks(os.DirFS(stdDir), mod+".ail")
		if err != nil {
			t.Fatal(err)
		}
		if len(marks) != len(g.Funcs) {
			t.Errorf("%s: docs sees %d exports, golden %d", mod, len(marks), len(g.Funcs))
		}
		for _, f := range g.Funcs {
			m, ok := marks[f.Name]
			if !ok {
				t.Errorf("%s.%s: in the golden, not in docs", mod, f.Name)
				continue
			}
			if m.Tier != f.Tier || m.ReplacedBy != f.ReplacedBy || m.Note != f.DeprecationNote {
				t.Errorf("%s.%s: docs %+v, golden tier=%q replaced_by=%q note=%q", mod, f.Name, m, f.Tier, f.ReplacedBy, f.DeprecationNote)
			}
			checked++
		}
	}
	if checked < 400 {
		t.Errorf("only %d functions compared; the golden set looks truncated", checked)
	}
}

// The docs line scraper keeps an export's doc comment across an annotation
// line, and the module page shows the marks.
func TestDocsShowsTierMarks(t *testing.T) {
	bin := buildAilang(t)
	stdout, stderr, code := runWithStdin(t, bin, "", "docs", "std/list")
	if code != 0 {
		t.Fatalf("docs exit %d: %s", code, stderr)
	}
	for _, want := range []string{
		"Tier: stable",
		"head_or[a](list[a], a) -> a  [deprecated → headOr]",
		"member[a](a, list[a]) -> bool  [deprecated → contains]",
		"takeMap[a, b](int, (a -> b), list[a]) -> list[b]  [experimental]",
		// takeMap's doc comment sits above `@experimental`; it must survive.
		"Fused take + map",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("ailang docs std/list lacks %q", want)
		}
	}
}

// `check --package` (the publish boundary, and how downstream packages are
// checked) reports TC_DEPRECATED_001 too: package mode used to drop every
// pipeline warning. Two files share one use through a common module; the
// warning appears once, the check still passes, and `errors` is unchanged.
func TestCheckPackageReportsDeprecations(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	files := map[string]string{
		"ailang.toml": "[package]\nname = \"depvendor/deppkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[exports]\nmodules = [\"depvendor/deppkg/a\", \"depvendor/deppkg/b\"]\n",
		"a.ail":       "module depvendor/deppkg/a\nimport std/list (head_or)\n\nexport pure func first(xs: [int]) -> int = head_or(xs, 0)\n",
		"b.ail":       "module depvendor/deppkg/b\nimport depvendor/deppkg/a (first)\n\nexport pure func firstPlusOne(xs: [int]) -> int = first(xs) + 1\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runAilangBin(t, bin, "check", "--package", dir)
		if code != 0 || strings.Count(stderr, "TC_DEPRECATED_001 at ") != 1 || !strings.Contains(stderr, "std/list.head_or is deprecated; use headOr") {
			t.Errorf("%s: human: exit %d, want 1 TC_DEPRECATED_001 on stderr:\n%s", run, code, stderr)
		}
		stdout, _, code := runAilangBin(t, bin, "check", "--package", "--json", dir)
		var out struct {
			Passed   bool                    `json:"passed"`
			Errors   []struct{ Code string } `json:"errors"`
			Warnings []struct {
				Code string `json:"code"`
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("%s: %v\n%s", run, err, stdout)
		}
		if code != 0 || !out.Passed || len(out.Warnings) != 1 || out.Warnings[0].Code != "TC_DEPRECATED_001" ||
			!strings.HasSuffix(filepath.ToSlash(out.Warnings[0].File), "a.ail") || out.Warnings[0].Line != 4 {
			t.Errorf("%s: --json: exit %d %+v", run, code, out)
		}
		for _, e := range out.Errors {
			if e.Code == "TC_DEPRECATED_001" {
				t.Errorf("%s: a deprecation reached errors[]: %+v", run, out.Errors)
			}
		}
	}
}

// Deprecated names used only in test code — named test blocks and a property
// block, which `check` never elaborates — warn from check (human and --json)
// and from `ailang test` (stderr; stdout and the verdict unchanged), once per
// use, cold and warm. Before the fix all three were silent.
func TestDeprecatedWarningsInTestCode(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	src := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "testblock.ail")
	wantLines := []int{16, 20, 24}
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runWithStdin(t, bin, "", "check", src)
		if code != 0 || strings.Count(stderr, "TC_DEPRECATED_001 at ") != len(wantLines) {
			t.Errorf("%s: check: exit %d, want %d warnings:\n%s", run, code, len(wantLines), stderr)
		}
		stdout, _, code := runWithStdin(t, bin, "", "check", "--json", src)
		var out struct {
			Passed   bool `json:"passed"`
			Warnings []struct {
				Code string `json:"code"`
				Line int    `json:"line"`
			} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("%s: %v\n%s", run, err, stdout)
		}
		var lines []int
		for _, w := range out.Warnings {
			if w.Code == "TC_DEPRECATED_001" {
				lines = append(lines, w.Line)
			}
		}
		sort.Ints(lines)
		if code != 0 || !out.Passed || !reflect.DeepEqual(lines, wantLines) {
			t.Errorf("%s: check --json: exit %d passed %v, warning lines %v, want %v", run, code, out.Passed, lines, wantLines)
		}

		stdout, stderr, code = runWithStdin(t, bin, "", "test", src)
		if code != 0 || strings.Count(stderr, "TC_DEPRECATED_001 at ") != len(wantLines) || strings.Contains(stdout, "TC_DEPRECATED_001") {
			t.Errorf("%s: ailang test: exit %d, want %d warnings on stderr only\nstdout:\n%s\nstderr:\n%s", run, code, len(wantLines), stdout, stderr)
		}
		stdout, _, code = runWithStdin(t, bin, "", "test", "--json", src)
		var tr struct {
			Passed int `json:"passed_tests"`
		}
		if err := json.Unmarshal([]byte(stdout), &tr); err != nil || code != 0 || tr.Passed != 4 {
			t.Errorf("%s: ailang test --json must stay clean JSON with 4 passes: exit %d err %v\n%s", run, code, err, stdout)
		}
	}
}

// A deprecated name that is imported and never used warns once, at the import
// (it fails with IMP010 when the alias is removed); a used one warns at its use
// only. Every channel, cold and warm.
func TestDeprecatedUnusedImport(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	src := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "unused_import.ail")
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runWithStdin(t, bin, "", "check", src)
		if code != 0 || strings.Count(stderr, "TC_DEPRECATED_001 at ") != 2 ||
			!strings.Contains(stderr, "unused_import.ail:6:27") || !strings.Contains(stderr, "std/list.nth_or is deprecated; use nthOr (it is imported here and never used)") ||
			!strings.Contains(stderr, "unused_import.ail:8:") {
			t.Errorf("%s: check:\n%s", run, stderr)
		}
		stdout, _, _ := runWithStdin(t, bin, "", "check", "--json", src)
		var out struct {
			Warnings []struct {
				Code    string `json:"code"`
				Line    int    `json:"line"`
				Message string `json:"message"`
			} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatal(err)
		}
		got := map[int]string{}
		for _, w := range out.Warnings {
			if w.Code == "TC_DEPRECATED_001" {
				got[w.Line] = w.Message
			}
		}
		if len(got) != 2 || !strings.Contains(got[6], "nth_or") || !strings.Contains(got[8], "head_or") {
			t.Errorf("%s: --json warnings by line: %v", run, got)
		}
		stdout, _, _ = runWithStdin(t, bin, "", "check", "--format", "agent", src)
		if strings.Count(stdout, "TC_DEPRECATED_001") != 2 {
			t.Errorf("%s: --format agent:\n%s", run, stdout)
		}
		_, stderr, _ = runWithStdin(t, bin, "", "test", src)
		if strings.Count(stderr, "TC_DEPRECATED_001 at ") != 2 {
			t.Errorf("%s: ailang test stderr:\n%s", run, stderr)
		}
	}
}

// `ailang test a.ail b.ail` (and a directory) prints a shared module's
// deprecation once per run, in one path spelling — not once per importing
// file, in relative and absolute forms.
func TestTestCommandDeduplicatesDeprecations(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "multi")
	for name, args := range map[string][]string{
		"two files": {"test", filepath.Join(dir, "a.ail"), filepath.Join(dir, "b.ail")},
		"directory": {"test", dir},
	} {
		_, stderr, _ := runWithStdin(t, bin, "", args...)
		if n := strings.Count(stderr, "TC_DEPRECATED_001 at "); n != 1 {
			t.Errorf("%s: %d TC_DEPRECATED_001 lines, want 1:\n%s", name, n, stderr)
		}
		want := "TC_DEPRECATED_001 at " + filepath.Join(dir, "shared.ail") + ":7:44"
		if !strings.Contains(filepath.ToSlash(stderr), filepath.ToSlash(want)) {
			t.Errorf("%s: want the relative spelling %q:\n%s", name, want, stderr)
		}
	}
}

// The de-duplication is per use site, not per position: two files with a
// deprecated use at the same line:column are both reported, in a multi-file
// and in a directory run (sub-directories included).
func TestTestCommandReportsSamePositionInTwoFiles(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "samepos")
	b, c := filepath.Join(dir, "b.ail"), filepath.Join(dir, "sub", "c.ail")
	for name, args := range map[string][]string{
		"two files": {"test", b, c},
		"directory": {"test", dir},
	} {
		_, stderr, _ := runWithStdin(t, bin, "", args...)
		got := filepath.ToSlash(stderr)
		if n := strings.Count(got, "TC_DEPRECATED_001 at "); n != 2 {
			t.Errorf("%s: %d TC_DEPRECATED_001 lines, want 2:\n%s", name, n, stderr)
		}
		for _, f := range []string{b, c} {
			if want := "TC_DEPRECATED_001 at " + filepath.ToSlash(f) + ":4:33"; !strings.Contains(got, want) {
				t.Errorf("%s: want %q:\n%s", name, want, stderr)
			}
		}
	}
}

// `L.nthOrx` through a module alias is IMP010 naming the member, with the
// did-you-mean of the import-list path, in human and --json output. It used to
// say "undefined variable: L".
func TestQualifiedMissingMemberCLI(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "stdrenames", "qualified_missing.ail")
	_, stderr, code := runWithStdin(t, bin, "", "check", src)
	if code == 0 || !strings.Contains(stderr, "IMP010: symbol 'nthOrx' not exported by 'std/list' (qualified access L.nthOrx)") ||
		!strings.Contains(stderr, "did you mean 'nthOr'") || strings.Contains(stderr, "undefined variable: L") {
		t.Errorf("human: exit %d\n%s", code, stderr)
	}
	stdout, _, _ := runWithStdin(t, bin, "", "check", "--json", src)
	var out struct {
		Errors []struct {
			Code string `json:"code"`
			Line int    `json:"line"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) != 1 || out.Errors[0].Code != "IMP010" || out.Errors[0].Line != 6 {
		t.Errorf("--json: %+v", out.Errors)
	}
}
