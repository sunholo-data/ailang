package smt_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/smt"
)

func verifySource(t *testing.T, filename, source string) (*smt.VerifyReport, pipeline.Result) {
	t.Helper()
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable")
	}
	result, err := pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true}, pipeline.Source{Code: source, Filename: filename})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, result.Errors)
	}
	modules := make(map[string]smt.VerifyModule)
	for name, mod := range result.Modules {
		modules[name] = smt.VerifyModule{File: mod.File, Core: mod.Core}
	}
	before := make(map[*core.DeclMeta]bool)
	for _, meta := range result.Artifacts.Core.Meta {
		before[meta] = meta.IsPure
	}
	for _, mod := range result.Modules {
		if mod.Core != nil {
			for _, meta := range mod.Core.Meta {
				before[meta] = meta.IsPure
			}
		}
	}
	report := smt.Verify(result.Artifacts.Core, result.Artifacts.AST, modules, smt.VerifyOptions{Timeout: 5 * time.Second})
	for meta, pure := range before {
		if meta.IsPure != pure {
			t.Fatalf("verification mutated purity of %s", meta.Name)
		}
	}
	return report, result
}

func TestPromptInjectionReferenceContracts(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	for _, tc := range []struct {
		name                 string
		verified, violations int
	}{{"safe", 3, 0}, {"injected", 2, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("benchmarks", "prompt_injection", "expected_ailang_"+tc.name+".ail")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			report, _ := verifySource(t, path, string(source))
			if report.Verified != tc.verified || report.Counterexample != tc.violations || report.Errors != 0 || report.Skipped != 0 {
				t.Fatalf("unexpected report: %+v", report)
			}
			if tc.violations > 0 {
				for _, r := range report.Results {
					if r.Status == "counterexample" && r.Function != "injectedForward" {
						t.Fatalf("wrong violation: %+v", r)
					}
				}
			}
		})
	}
}

func TestDeclassifyCrossModuleContracts(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	helper := filepath.Join(dir, "helper.ail")
	if err := os.WriteFile(helper, []byte(`module helper
export func sanitizeBody(x: string) -> string ! {Declassify[label=x]} { "clean" }
`), 0600); err != nil {
		t.Fatal(err)
	}
	source := `module caller
import helper (sanitizeBody)
export func forward(x: string) -> string ! {Declassify[label=x]}
ensures { result == "clean" }
{ sanitizeBody(x) }
export pure func noRow(x: int) -> int
ensures { result == x }
{ x }
`
	if err := os.WriteFile("caller.ail", []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	report, result := verifySource(t, "caller.ail", source)
	if report.Verified != 2 || report.Counterexample != 0 || report.Errors != 0 || report.Skipped != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, mod := range result.Modules {
		if mod.Core != nil {
			if meta := mod.Core.Meta["sanitizeBody"]; meta != nil && meta.IsPure {
				t.Fatal("imported Declassify gained IsPure")
			}
		}
	}
}
