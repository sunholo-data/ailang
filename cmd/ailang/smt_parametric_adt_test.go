package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// The broken success arm is a necessary negative control: a design that skips
// refutation but proves the good body has not repaired the verifier's boundary.
func TestSMTParametricADT_ImportedErrorProofAndRefutationCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprintf("broken_%v", broken), func(t *testing.T) {
			body := "n"
			if broken {
				body = "-1"
			}
			source := `export pure func checkSize(n: int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(` + body + `) }
`
			dir, file := writeSMTParametricPackage(t, source)
			out, stderr, exit := runAilangBin(t, bin, "check", "--relax-modules", file)
			if exit != 0 {
				t.Fatalf("repro failed to type-check: %d\n%s\n%s", exit, out, stderr)
			}
			out, stderr, exit = runAilangBin(t, bin, "verify", "--verbose", "--json", "--relax-modules", file)
			report := readSMTParametricFileReport(t, out, stderr)
			assertSMTParametricDecision(t, report, broken, exit, out+stderr)
			if len(report.Results) != 1 {
				t.Fatalf("expected exactly the checked function, got %d results:\n%s", len(report.Results), out)
			}
			if broken && (report.Results[0].Status != "counterexample" || len(report.Results[0].Model) == 0) {
				t.Fatalf("broken guard lacks a concrete SAT model:\n%s", out)
			}
			if report.NumericModel == nil || report.NumericModel.RuntimeEquivalence != "not_established" {
				t.Fatalf("numeric-model qualification lost:\n%s", out)
			}
			requireSMTParametricResultScript(t, report.Results[0].SMTLib)
			out, stderr, exit = runAilangBin(t, bin, "verify", "--package", dir, "--json")
			packageReport, err := pkg.DecodePackageVerifyReport([]byte(out))
			if err != nil {
				t.Fatalf("package decode: %v\n%s\n%s", err, out, stderr)
			}
			assertSMTParametricDecision(t, smt.VerifyReport{Verified: packageReport.Verified, Counterexample: packageReport.Counterexample, Skipped: packageReport.Skipped, Errors: packageReport.Errors}, broken, exit, out+stderr)
			out, stderr, exit = runAilangBin(t, bin, "ai-check", "--relax-modules", file)
			var ai aiCheckOutput
			if err := json.Unmarshal([]byte(out), &ai); err != nil {
				t.Fatalf("ai-check JSON: %v\n%s\n%s", err, out, stderr)
			}
			if !ai.Check.Passed || !ai.Verify.Available {
				t.Fatalf("ai-check did not reach verifier: %+v\n%s", ai, stderr)
			}
			assertSMTParametricDecision(t, smt.VerifyReport{Verified: ai.Verify.Verified, Counterexample: ai.Verify.Counterexample, Skipped: ai.Verify.Skipped, Errors: ai.Verify.Errors}, broken, exit, out+stderr)
		})
	}
}

func writeSMTParametricPackage(t *testing.T, source string) (dir, file string) {
	t.Helper()
	dir = t.TempDir()
	files := map[string]string{
		"ailang.toml": `[package]
name = "resource_probe/audio"
version = "0.1.0"
edition = "1"
[exports]
modules = ["resource_probe/audio/safety", "resource_probe/audio/main"]
[effects]
max = []
[stability]
level = "experimental"
[release]
kind = "feature"
`,
		"safety.ail": `module resource_probe/audio/safety
export type AudioError = ResourceLimit(string) | InvalidFormat(string)
`,
		"main.ail": `module resource_probe/audio/main
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)
import ./safety (AudioError, ResourceLimit, InvalidFormat)
` + source,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir, filepath.Join(dir, "main.ail")
}

func readSMTParametricFileReport(t *testing.T, out, stderr string) smt.VerifyReport {
	t.Helper()
	var report smt.VerifyReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("verify JSON: %v\n%s\n%s", err, out, stderr)
	}
	return report
}

func assertSMTParametricDecision(t *testing.T, r smt.VerifyReport, broken bool, exit int, output string) {
	t.Helper()
	wantVerified, wantCounter, wantExit := 1, 0, 0
	if broken {
		wantVerified, wantCounter, wantExit = 0, 1, 1
	}
	if r.Verified != wantVerified || r.Counterexample != wantCounter || r.Skipped != 0 || r.Errors != 0 || exit != wantExit {
		t.Fatalf("verified/counterexample/skipped/errors/exit = %d/%d/%d/%d/%d, want %d/%d/0/0/%d\n%s", r.Verified, r.Counterexample, r.Skipped, r.Errors, exit, wantVerified, wantCounter, wantExit, output)
	}
}

func requireSMTParametricResultScript(t *testing.T, script string) {
	t.Helper()
	s := strings.Join(strings.Fields(script), " ")
	for _, required := range []string{"AudioError", "ResourceLimit", "(Result Int AudioError)", "(par ("} {
		if !strings.Contains(s, required) {
			t.Fatalf("typed SMT lacks %q:\n%s", required, script)
		}
	}
	// Z3 accepts both the singular native par declaration and the explicit
	// arity form used for grouped datatypes. Neither erases Result's binders.
	if !strings.Contains(s, "(declare-datatype Result (par (") && !strings.Contains(s, "(Result 2)") {
		t.Fatalf("Result lacks a native parametric datatype declaration:\n%s", script)
	}
	errField := regexp.MustCompile(`\(Err_0\s+([^\s()]+)\)`).FindStringSubmatch(s)
	if len(errField) != 2 || errField[1] == "Int" {
		t.Fatalf("Err payload erased or declaration absent: %q\n%s", errField, script)
	}
	if !strings.Contains(s, "(ResourceLimit_0 String)") {
		t.Fatalf("imported error payload was not retained:\n%s", script)
	}
}

func TestSMTParametricADT_NestedAndContractRootsCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	cases := []struct {
		name, source string
		sorts        []string
	}{
		{"nested_option_result", `export pure func nested(n: int) -> Result[Option[int], AudioError]
ensures { match result { Ok(Some(x)) => x >= 0, Ok(None) => true, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(Some(n)) }
`, []string{"(Result (Option Int) AudioError)", "(Option Int)"}},
		{"two_option_instances", `export pure func twoInstances(n: int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{
 let label: Option[string] = Some("ok");
 let size: Option[int] = Some(if n < 0 then 0 else n);
 match label {
   None => Err(ResourceLimit("label")),
   Some(s) => match size {
     None => Err(ResourceLimit("size")),
     Some(x) => if s == "ok" then Ok(x) else Err(InvalidFormat(s))
   }
 }
}
`, []string{"(Option Int)", "(Option String)", "(Result Int AudioError)"}},
		{"nested_result_instances", `export pure func nestedResult(n: int) -> Result[Result[int, AudioError], AudioError]
ensures { match result { Ok(Ok(x)) => x >= 0, Ok(Err(_)) => true, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(Ok(n)) }
`, []string{"(Result (Result Int AudioError) AudioError)", "(Result Int AudioError)"}},
		{"postcondition_only_constructor", `export pure func postconditionRoot(n: int) -> int
ensures { match ResourceLimit("contract-only") { ResourceLimit(_) => result >= 0, InvalidFormat(_) => false } }
{ if n < 0 then 0 else n }
`, []string{"AudioError", "ResourceLimit_0 String"}},
		{"precondition_only_constructor", `export pure func preconditionRoot(n: int) -> int
requires { match ResourceLimit("contract-only") { ResourceLimit(_) => true, InvalidFormat(_) => false } }
ensures { result >= 0 }
{ if n < 0 then 0 else n }
`, []string{"AudioError", "ResourceLimit_0 String"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, file := writeSMTParametricPackage(t, c.source)
			out, stderr, rc := runAilangBin(t, bin, "check", "--relax-modules", file)
			if rc != 0 {
				t.Fatalf("valid fixture failed check: %d\n%s\n%s", rc, out, stderr)
			}
			out, stderr, rc = runAilangBin(t, bin, "verify", "--verbose", "--json", "--relax-modules", file)
			report := readSMTParametricFileReport(t, out, stderr)
			assertSMTParametricDecision(t, report, false, rc, out+stderr)
			if len(report.Results) != 1 {
				t.Fatalf("missing function result:\n%s", out)
			}
			script := strings.Join(strings.Fields(report.Results[0].SMTLib), " ")
			for _, sort := range c.sorts {
				if !strings.Contains(script, sort) {
					t.Errorf("actual script missing %q:\n%s", sort, report.Results[0].SMTLib)
				}
			}
			out, stderr, rc = runAilangBin(t, bin, "ai-check", "--relax-modules", file)
			var ai aiCheckOutput
			if err := json.Unmarshal([]byte(out), &ai); err != nil {
				t.Fatalf("ai-check decode: %v\n%s\n%s", err, out, stderr)
			}
			if !ai.Check.Passed {
				t.Fatalf("ai-check fixture rejected:\n%s", out)
			}
			assertSMTParametricDecision(t, smt.VerifyReport{Verified: ai.Verify.Verified, Counterexample: ai.Verify.Counterexample, Skipped: ai.Verify.Skipped, Errors: ai.Verify.Errors}, false, rc, out+stderr)
		})
	}
}

func TestSMTParametricADT_UnsupportedShapesRemainUnprovedCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	cases := []struct {
		name, source string
		aiError      bool
	}{
		{"unit_result", `export pure func unitResult(n: int) -> Result[(), AudioError]
ensures { match result { Ok(_) => true, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(()) }
`, false},
		{"higher_order_parameter", `export pure func higherOrder(f: int -> int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{ Ok(f(0)) }
`, false},
		{"open_record_parameter", `export pure func openRecord(x: { value: int | r }) -> Result[int, AudioError]
ensures { match result { Ok(n) => n >= 0, Err(_) => true } }
{ Ok(x.value) }
`, false},
		// This pre-existing special policy is deliberately a separate control:
		// ai-check treats the encoder's unsupported Core construct as an error.
		{"existing_list_pattern_policy", `export pure func listPattern(xs: [int]) -> int
ensures { result >= 0 }
{ match xs { [] => 0, x :: rest => x } }
`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, file := writeSMTParametricPackage(t, c.source)
			out, stderr, rc := runAilangBin(t, bin, "check", "--relax-modules", file)
			if rc != 0 {
				t.Fatalf("unsupported verifier fixture is not valid AILANG: %d\n%s\n%s", rc, out, stderr)
			}
			out, stderr, rc = runAilangBin(t, bin, "verify", "--json", "--relax-modules", file)
			report := readSMTParametricFileReport(t, out, stderr)
			if rc != 0 || report.Verified != 0 || report.Counterexample != 0 || report.Skipped != 1 || report.Errors != 0 || len(report.Results) != 1 {
				t.Fatalf("unsupported shape must skip honestly (not prove/error): exit=%d\n%s\n%s", rc, out, stderr)
			}
			r := report.Results[0]
			if r.Status != "skipped" || r.ReasonCode != verification.UnsupportedEncoding || r.Reason == "" {
				t.Fatalf("skip lacks typed actionable evidence: %+v", r)
			}
			out, stderr, rc = runAilangBin(t, bin, "ai-check", "--relax-modules", file)
			var ai aiCheckOutput
			if err := json.Unmarshal([]byte(out), &ai); err != nil {
				t.Fatalf("ai-check JSON: %v\n%s\n%s", err, out, stderr)
			}
			wantSkipped, wantErrors, wantExit := 1, 0, 0
			if c.aiError {
				wantSkipped, wantErrors, wantExit = 0, 1, 1
			}
			if !ai.Check.Passed || ai.Verify.Verified != 0 || ai.Verify.Counterexample != 0 || ai.Verify.Skipped != wantSkipped || ai.Verify.Errors != wantErrors || rc != wantExit {
				t.Fatalf("ai-check unsupported policy changed: exit=%d expected skipped/errors/exit=%d/%d/%d\n%s\n%s", rc, wantSkipped, wantErrors, wantExit, out, stderr)
			}
		})
	}
}

func TestSMTParametricADT_NestedRefutationsCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	cases := []struct{ name, source string }{
		{"option_inside_result", `export pure func brokenNested(n: int) -> Result[Option[int], AudioError]
ensures { match result { Ok(Some(x)) => x >= 0, Ok(None) => true, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(Some(-1)) }
`},
		{"result_inside_result", `export pure func brokenNestedResult(n: int) -> Result[Result[int, AudioError], AudioError]
ensures { match result { Ok(Ok(x)) => x >= 0, Ok(Err(_)) => true, Err(_) => true } }
{ if n < 0 then Err(ResourceLimit("negative")) else Ok(Ok(-1)) }
`},
		{"two_option_instances", `export pure func brokenTwoInstances(n: int) -> Result[int, AudioError]
ensures { match result { Ok(x) => x >= 0, Err(_) => true } }
{
 let label: Option[string] = Some("ok");
 let size: Option[int] = Some(-1);
 match label {
   None => Err(ResourceLimit("label")),
   Some(s) => match size {
     None => Err(ResourceLimit("size")),
     Some(x) => if s == "ok" then Ok(x) else Err(InvalidFormat(s))
   }
 }
}
`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, file := writeSMTParametricPackage(t, c.source)
			out, stderr, rc := runAilangBin(t, bin, "check", "--relax-modules", file)
			if rc != 0 {
				t.Fatalf("broken contract fixture must still typecheck: %d\n%s\n%s", rc, out, stderr)
			}
			out, stderr, rc = runAilangBin(t, bin, "verify", "--verbose", "--json", "--relax-modules", file)
			r := readSMTParametricFileReport(t, out, stderr)
			assertSMTParametricDecision(t, r, true, rc, out+stderr)
			if len(r.Results) != 1 || r.Results[0].Status != "counterexample" || len(r.Results[0].Model) == 0 {
				t.Fatalf("nested refutation lost SAT model:\n%s", out)
			}
			out, stderr, rc = runAilangBin(t, bin, "ai-check", "--relax-modules", file)
			var ai aiCheckOutput
			if err := json.Unmarshal([]byte(out), &ai); err != nil {
				t.Fatalf("ai-check decode: %v\n%s\n%s", err, out, stderr)
			}
			assertSMTParametricDecision(t, smt.VerifyReport{Verified: ai.Verify.Verified, Counterexample: ai.Verify.Counterexample, Skipped: ai.Verify.Skipped, Errors: ai.Verify.Errors}, true, rc, out+stderr)
		})
	}
}

func TestSMTParametricADT_AmbiguousImportedShortNameIsIsolatedCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	dir := t.TempDir()
	files := map[string]string{
		"ailang.toml": `[package]
name = "review/collision"
version = "0.1.0"
edition = "1"
[exports]
modules = ["review/collision/sameleft", "review/collision/sameright", "review/collision/samemain"]
[effects]
max = []
[stability]
level = "experimental"
[release]
kind = "feature"
`,
		"sameleft.ail": `module review/collision/sameleft
export type Error = LeftError(int)
`,
		"sameright.ail": `module review/collision/sameright
export type Error = RightError(string)
`,
		"samemain.ail": `module review/collision/samemain
import ./sameleft (Error as LeftType, LeftError)
import ./sameright (Error as RightType, RightError)
export pure func probe(n: int) -> bool
ensures { result } {
 let a = LeftError(n);
 let b = RightError("x");
 let x = match a { LeftError(v) => v == n };
 let y = match b { RightError(s) => s == "x" };
 x && y
}
export pure func plain(n: int) -> int
ensures { result == n } { n }
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(dir, "samemain.ail")
	out, stderr, rc := runAilangBin(t, bin, "check", "--relax-modules", file)
	if rc != 0 {
		t.Fatalf("qualified type aliases must check: %d\n%s\n%s", rc, out, stderr)
	}
	// Core's function-map report order is not the determinism contract. Compare
	// each function's classified evidence and actual script, not raw JSON bytes.
	var oldPlainScript, oldProbeReason string
	for i := 0; i < 3; i++ {
		out, stderr, rc = runAilangBin(t, bin, "verify", "--verbose", "--json", "--relax-modules", file)
		r := readSMTParametricFileReport(t, out, stderr)
		if rc != 0 || r.Verified != 1 || r.Skipped != 1 || r.Errors != 0 || r.Counterexample != 0 {
			t.Fatalf("ambiguous Error ownership must skip only probe: exit=%d\n%s\n%s", rc, out, stderr)
		}
		byName := map[string]smt.VerifyResult{}
		for _, item := range r.Results {
			byName[item.Function] = item
		}
		probe, ok := byName["probe"]
		if !ok || probe.Status != "skipped" || probe.ReasonCode != verification.UnsupportedEncoding || !strings.Contains(strings.ToLower(probe.Reason), "ambig") {
			t.Fatalf("ambiguous import lacks honest typed diagnostic: %+v", probe)
		}
		plain, ok := byName["plain"]
		if !ok || plain.Status != "verified" || plain.SMTLib == "" {
			t.Fatalf("unrelated primitive proof polluted: %+v", plain)
		}
		if i > 0 && (plain.SMTLib != oldPlainScript || probe.Reason != oldProbeReason) {
			t.Fatal("map iteration changed the declaration script or ambiguity decision")
		}
		oldPlainScript, oldProbeReason = plain.SMTLib, probe.Reason
	}
}

// A nested pattern's n shadows the parameter. Renaming only the outer pattern
// leaks the precondition's nonnegative parameter into the branch/guard, proving
// a runtime-false property. These must all be refutable under n >= 0.
func TestSMTParametricADT_NestedPatternShadowingRefutesCLI(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI has no Z3)")
	}
	bin := buildAilang(t)
	cases := []struct {
		name, source  string
		booleanOutput bool
	}{
		{"body", `export pure func shadow(n: int) -> bool
requires { n >= 0 }
ensures { result } {
 let x: Result[Option[int], string] = Ok(Some(-1));
 match x { Ok(Some(n)) => n >= 0, Ok(None) => true, Err(_) => true }
}
`, true},
		{"contract", `export pure func shadow(n: int) -> Result[Option[int], string]
requires { n >= 0 }
ensures { match result { Ok(Some(n)) => n >= 0, Ok(None) => true, Err(_) => true } }
{ Ok(Some(-1)) }
`, false},
		{"guard", `export pure func shadow(n: int) -> bool
requires { n >= 0 }
ensures { result } {
 let x: Result[Option[int], string] = Ok(Some(-1));
 match x {
  Ok(Some(n)) if n >= 0 => true,
  Ok(Some(_)) => false,
  Ok(None) => true,
  Err(_) => true
 }
}
`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, file := writeSMTParametricPackage(t, c.source)
			out, stderr, rc := runAilangBin(t, bin, "check", "--relax-modules", file)
			if rc != 0 {
				t.Fatalf("valid shadowing source rejected: %d\n%s\n%s", rc, out, stderr)
			}
			if c.booleanOutput {
				for _, vm := range []bool{false, true} {
					args := []string{"run", "--entry", "shadow", "--args-json", "0"}
					if vm {
						args = append(args, "--strict-bytecode")
					}
					args = append(args, file)
					out, stderr, rc = runAilangBin(t, bin, args...)
					if rc != 0 || strings.TrimSpace(out) != "false" {
						t.Fatalf("runtime control vm=%v: expected false at n=0, exit=%d\n%s\n%s", vm, rc, out, stderr)
					}
				}
			}
			out, stderr, rc = runAilangBin(t, bin, "verify", "--verbose", "--json", "--relax-modules", file)
			r := readSMTParametricFileReport(t, out, stderr)
			assertSMTParametricDecision(t, r, true, rc, out+stderr)
			if len(r.Results) != 1 || r.Results[0].Status != "counterexample" || len(r.Results[0].Model) == 0 {
				t.Fatalf("nested shadowing lacks genuine SAT counterexample:\n%s", out)
			}
			out, stderr, rc = runAilangBin(t, bin, "ai-check", "--relax-modules", file)
			var ai aiCheckOutput
			if err := json.Unmarshal([]byte(out), &ai); err != nil {
				t.Fatalf("ai-check JSON: %v\n%s\n%s", err, out, stderr)
			}
			if !ai.Check.Passed {
				t.Fatalf("ai-check shadowing source rejected:\n%s", out)
			}
			assertSMTParametricDecision(t, smt.VerifyReport{Verified: ai.Verify.Verified, Counterexample: ai.Verify.Counterexample, Skipped: ai.Verify.Skipped, Errors: ai.Verify.Errors}, true, rc, out+stderr)
		})
	}
}
