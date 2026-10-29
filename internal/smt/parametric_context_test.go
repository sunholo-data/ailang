package smt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestParametricCheckedNodeContext(t *testing.T) {
	source := `module probe
import std/result (Result, Ok, Err)
export type AudioError = ResourceLimit(string)
export pure func checked(n: int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(n) }
`
	f := filepath.Join(t.TempDir(), "probe.ail")
	if err := os.WriteFile(f, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true}, pipeline.Source{Filename: f, Code: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Errors) > 0 {
		t.Fatal(r.Errors)
	}
	modules := make(map[string]VerifyModule)
	for path, mod := range r.Modules {
		if mod != nil {
			ti, _ := mod.CoreTI.(types.CoreTypeInfo)
			modules[path] = VerifyModule{File: mod.File, Core: mod.Core, CoreTI: ti}
		}
	}
	if !Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	report := Verify(r.Artifacts.Core, r.Artifacts.AST, modules, VerifyOptions{CoreTI: r.Artifacts.CoreTI, Timeout: time.Second, Verbose: true})
	if report.Verified != 1 || report.Errors != 0 || report.Skipped != 0 {
		t.Fatalf("typed guard not verified: %+v", report)
	}
	script := report.Results[0].SMTLib
	for _, want := range []string{"(Result Int AudioError)", "(par (a e)", "(Err_0 e)", "(ResourceLimit_0 String)", "(as Err (Result Int AudioError))"} {
		if !strings.Contains(script, want) {
			t.Fatalf("missing %s:\n%s", want, script)
		}
	}
	// The same checked node types are necessary: no map must skip, not fall
	// back to the outer return sort or choose Int for the phantom parameter.
	missing := Verify(r.Artifacts.Core, r.Artifacts.AST, modules, VerifyOptions{Timeout: time.Second})
	if missing.Skipped != 1 || missing.Verified != 0 || missing.Errors != 0 {
		t.Fatalf("missing constructor types must skip: %+v", missing)
	}
}

func TestParametricNestedPatternShadowingRefutes(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	source := `module shadow_probe
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)
export pure func shadow(n: int) -> bool
requires { n >= 0 }
ensures { result }
{
 let x: Result[Option[int], string] = Ok(Some(-1));
 match x { Ok(Some(n)) => n >= 0, Ok(None) => true, Err(_) => true }
}
`
	f := filepath.Join(t.TempDir(), "shadow_probe.ail")
	if err := os.WriteFile(f, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true}, pipeline.Source{Filename: f, Code: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Errors) > 0 {
		t.Fatal(r.Errors)
	}
	modules := make(map[string]VerifyModule)
	for path, mod := range r.Modules {
		if mod != nil {
			ti, _ := mod.CoreTI.(types.CoreTypeInfo)
			modules[path] = VerifyModule{File: mod.File, Core: mod.Core, CoreTI: ti}
		}
	}
	report := Verify(r.Artifacts.Core, r.Artifacts.AST, modules, VerifyOptions{CoreTI: r.Artifacts.CoreTI, Timeout: time.Second, Verbose: true})
	if report.Counterexample != 1 || report.Verified != 0 || report.Errors != 0 || report.Skipped != 0 {
		t.Fatalf("pattern binder must shadow param: %+v", report)
	}
}
