package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Aliases of primitives keep their instances (#2211 with #2285): ==, <, + on
// lowercase and uppercase int aliases and / on float aliases, declared locally
// and imported, check and print the same on the evaluator and the strict VM,
// cold and warm.
func TestPrimitiveAliasOperators(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "typenames", "alias_ops.ail")
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	want := "true\ntrue\n42\n1.5\ntrue\n0.25\n"
	for _, run := range []string{"cold", "warm"} {
		if _, stderr, code := runWithStdin(t, bin, "", "check", src); code != 0 {
			t.Fatalf("%s: check: exit %d\n%s", run, code, stderr)
		}
		for _, backend := range [][]string{{"run"}, {"run", "--strict-bytecode"}} {
			args := append(append([]string{}, backend...), "--caps", "IO", "--entry", "main", src)
			stdout, stderr, code := runWithStdin(t, bin, "", args...)
			if code != 0 || stdout != want {
				t.Errorf("%s %v: exit %d\nstdout:\n%s\nstderr:\n%s", run, backend, code, stdout, stderr)
			}
		}
	}
}

// A parse error on the module-load path reaches `check --json` with its own
// PAR_ code and position, not "ERROR" (loader.ParseErrors unwraps to each
// *parser.ParserError). Covered for three codes from different parser paths.
func TestCheckJSON_ParseErrorCodes(t *testing.T) {
	bin := buildAilang(t)
	for _, c := range []struct{ code, body string }{
		{"PAR_TYPE_DECL_BUILTIN", "type float = float\n"},
		{"PAR_TYPE_PARAM_APPLIED", "pure func same[m](x: m[int]) -> int = 0\n"},
		{"PAR_TYPE_PARAM_BUILTIN", "type Res[int] = Ok(int) | Err(string)\n"},
	} {
		t.Run(c.code, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "pe.ail")
			src := "module pe\n" + c.body + "export func main() -> () ! {IO} = println(\"x\")\n"
			if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, _, code := runWithStdin(t, bin, "", "check", "--json", file)
			var out struct {
				Passed bool `json:"passed"`
				Errors []struct {
					Code    string `json:"code"`
					Line    int    `json:"line"`
					Message string `json:"message"`
				} `json:"errors"`
			}
			if err := json.Unmarshal([]byte(stdout), &out); err != nil {
				t.Fatalf("check --json: %v\n%s", err, stdout)
			}
			if code == 0 || out.Passed || len(out.Errors) == 0 || out.Errors[0].Code != c.code || out.Errors[0].Line != 2 ||
				!strings.Contains(out.Errors[0].Message, c.code) {
				t.Errorf("want %s at line 2, got exit %d:\n%s", c.code, code, stdout)
			}
		})
	}
}

// TC_UNBOUND_TYPE_002 (#2211) reaches every output channel of `ailang check`,
// on a cold and on a warm compile cache, and the program still checks.
func TestTypeNameWarningCheckOutputs(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "typenames", "warn_implicit.ail")
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runWithStdin(t, bin, "", "check", src)
		if code != 0 || !strings.Contains(stderr, "TC_UNBOUND_TYPE_002 at ") || !strings.Contains(stderr, "warn_implicit.ail:6:") {
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
		n := 0
		for _, w := range out.Warnings {
			if w.Code == "TC_UNBOUND_TYPE_002" {
				n++
				if w.Line != 6 || !strings.HasSuffix(filepath.ToSlash(w.File), "warn_implicit.ail") ||
					!strings.Contains(w.Suggestion, "import std/sem (namespace)") || !strings.Contains(w.Message, "'namespace'") {
					t.Errorf("%s: check --json warning malformed: %+v", run, w)
				}
			}
		}
		// One warning per (declaration, name), though `namespace` is written twice.
		if code != 0 || !out.Passed || n != 1 {
			t.Errorf("%s: check --json: exit %d passed %v, %d TC_UNBOUND_TYPE_002 warnings\n%s", run, code, out.Passed, n, stdout)
		}
	}
}
