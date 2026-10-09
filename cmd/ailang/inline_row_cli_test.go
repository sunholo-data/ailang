package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInlineRowDiagnostics(t *testing.T) {
	bin := cliTestBin(t)
	for _, tc := range []struct{ file, code string }{
		{"inline_rows_binary.ail", "TST002"}, {"inline_rows_unsupported.ail", "TST001"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("..", "..", "internal", "testing", "testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{
				{"check", "--relax-modules", "--json", path},
				{"ai-check", "--relax-modules", path},
				{"test", "--json", path},
			} {
				res := runCLIIsolated(t, bin, args...)
				if res.exitCode != 1 || !strings.Contains(res.stdout+res.stderr, tc.code) {
					t.Fatalf("%v: %+v", args, res)
				}
				if args[0] == "test" {
					var report struct {
						Total  int `json:"total_tests"`
						Failed int `json:"failed_tests"`
						Tests  []struct {
							Name     string `json:"name"`
							Error    string `json:"error"`
							Location string `json:"location"`
						} `json:"tests"`
					}
					if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
						t.Fatal(err, res.stdout)
					}
					if report.Total == 0 || report.Failed == 0 || len(report.Tests) == 0 {
						t.Fatalf("missing row report: %s", res.stdout)
					}
					for _, row := range report.Tests {
						if !strings.Contains(row.Error, tc.code) || row.Location == "" || !strings.Contains(row.Name, "_test_") {
							t.Fatalf("invalid row report: %+v", row)
						}
					}
				}
				if args[0] == "ai-check" {
					var report struct {
						Check struct {
							Errors []checkJSONError `json:"errors"`
						} `json:"check"`
					}
					if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
						t.Fatal(err, res.stdout)
					}
					if len(report.Check.Errors) == 0 || report.Check.Errors[0].Code != tc.code || report.Check.Errors[0].Line == 0 {
						t.Fatalf("missing structured check error: %s", res.stdout)
					}
				}
			}
		})
	}
	path, err := filepath.Abs(filepath.Join("..", "..", "internal", "testing", "testdata", "inline_expected_values.ail"))
	if err != nil {
		t.Fatal(err)
	}
	res := runCLIIsolated(t, bin, "test", path)
	if res.exitCode != 0 {
		t.Fatalf("composites: %+v", res)
	}
}

func TestCLIInlineRowCapabilityGating(t *testing.T) {
	bin := cliTestBin(t)
	path := filepath.Join(t.TempDir(), "effect_row.ail")
	const src = `module effect_row
import std/io (println)
export func echo(x: int) -> int ! {IO}
tests [ (2, 2) ]
{ println("hello"); x }
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	denied := runCLIIsolated(t, bin, "test", path)
	if denied.exitCode != 1 {
		t.Fatalf("without caps: %+v", denied)
	}
	if !strings.Contains(denied.stdout+denied.stderr, "requires capability") {
		t.Fatalf("capability gate changed: %+v", denied)
	}
}
