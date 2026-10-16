package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSourceCheckFile(t *testing.T, dir, name, source string) {
	t.Helper()
	file := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sourceCheckPackage(t *testing.T, withTests bool) string {
	t.Helper()
	dir := t.TempDir()
	writeSourceCheckFile(t, dir, "ailang.toml", `[package]
name = "probe/p"
version = "0.1.0"
edition = "1"
[exports]
modules = ["probe/p/good"]
`)
	writeSourceCheckFile(t, dir, "good.ail", "module probe/p/good\nexport pure func one() -> int = 1\n")
	if withTests {
		writeSourceCheckFile(t, dir, "good_test.ail", "module probe/p/good_test\nimport probe/p/good (one)\ntest \"one is one\" { one() == 1 }\n")
	}
	return dir
}

func TestPackageSourceCheckFailFast(t *testing.T) {
	bin := buildAilang(t)
	for _, tests := range []bool{false, true} {
		dir := sourceCheckPackage(t, tests)
		manifest, err := os.ReadFile(filepath.Join(dir, "ailang.toml"))
		if err != nil {
			t.Fatal(err)
		}
		writeSourceCheckFile(t, dir, "ailang.toml", strings.Replace(string(manifest), `["probe/p/good"]`, `["probe/p/good", "probe/p/bad"]`, 1))
		writeSourceCheckFile(t, dir, "bad.ail", "module probe/p/bad\nexport pure func oops() -> int = \"not an int\"\n")
		out, stderr, exit := runAilangBin(t, bin, "test", "--package", dir, "--allow-skips", "--no-color")
		if exit != 1 || !strings.Contains(out+stderr, "bad.ail") || !strings.Contains(out, "Package source check failed") || strings.Contains(out, "All tests passed") {
			t.Fatalf("tests=%v exit=%d\n%s\n%s", tests, exit, out, stderr)
		}
		if tests {
			out, stderr, exit = runAilangBin(t, bin, "test", "--package", dir, "--no-source-check", "--json")
			var doc map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatalf("%v: %s %s", err, out, stderr)
			}
			if exit != 0 || string(doc["passed_tests"]) != "1" || doc["source_check"] != nil {
				t.Fatalf("opt-out: exit=%d %s %s", exit, out, stderr)
			}
		}
	}
}

func TestPackageSourceCheckJSON(t *testing.T) {
	bin := buildAilang(t)
	for _, format := range [][]string{{"--json"}, {"--format", "json"}} {
		for _, tests := range []bool{false, true} {
			for _, broken := range []bool{false, true} {
				dir := sourceCheckPackage(t, tests)
				wantPassed := 1
				if tests {
					wantPassed++
				}
				if broken {
					writeSourceCheckFile(t, dir, "bad.ail", "module probe/p/bad\nexport pure func oops() -> int = \"bad\"\n")
				}
				args := append([]string{"test", "--package", dir}, format...)
				out, stderr, exit := runAilangBin(t, bin, args...)
				var doc struct {
					Success bool `json:"success"`
					Total   int  `json:"total_tests"`
					Source  *struct {
						Passed   int      `json:"passed"`
						Failed   int      `json:"failed"`
						Errors   []string `json:"errors"`
						Warnings []string `json:"warnings"`
					} `json:"source_check"`
				}
				if err := json.Unmarshal([]byte(out), &doc); err != nil {
					t.Fatalf("not one JSON document: %v\n%s\n%s", err, out, stderr)
				}
				wantExit, wantTests, wantFailed := 0, 0, 0
				if tests && !broken {
					wantTests = 1
				}
				if broken {
					wantExit, wantFailed = 1, 1
				}
				if exit != wantExit || doc.Total != wantTests || doc.Success != (wantTests > 0) || doc.Source == nil {
					t.Fatalf("exit=%d report=%+v\n%s\n%s", exit, doc, out, stderr)
				}
				if doc.Source.Passed != wantPassed || doc.Source.Failed != wantFailed || (len(doc.Source.Errors) > 0) != broken || doc.Source.Errors == nil || doc.Source.Warnings == nil {
					t.Fatalf("source report: %s", out)
				}
				if !tests && !broken && !strings.Contains(stderr, "No *_test.ail") {
					t.Fatal("missing no-tests warning")
				}
			}
		}
	}
}
