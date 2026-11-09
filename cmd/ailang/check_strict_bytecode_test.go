package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// M-BYTECODE-STRICT-256-REG-CEILING Phase A: a function the bytecode compiler
// tags evaluator-only used to surface only as a mid-run `vm: CALL` trap under
// --strict-bytecode, and not at all on a non-strict --bytecode run.
//
// wide.ail holds two such functions — a 300-field record literal cannot be
// built by one MAKE_RECORD (8-bit operand) — one reached from main, one not.
func strictBytecodeFixtures(t *testing.T) (wide, clean string) {
	t.Helper()
	dir := t.TempDir()
	fields := make([]string, 300)
	for i := range fields {
		fields[i] = fmt.Sprintf("f%d: %d", i, i)
	}
	rec := strings.Join(fields, ", ")
	wide = filepath.Join(dir, "wide.ail")
	writeFile(t, wide, fmt.Sprintf(`module wide

pure func wideRec() -> int {
  let r = {%s};
  r.f7
}

pure func unused() -> int {
  let r = {%s};
  r.f9
}

export func main() -> () ! {IO} {
  println("start");
  println(show(wideRec()))
}
`, rec, rec))

	// clean.ail is the two Phase B repro classes (601-element literal,
	// 60-arm else-if chain over a 16-field record) plus an std/list import
	// whose image carries evaluator-only helpers (takeMap, takeFlatMap) that
	// nothing here reaches: none of it may be reported.
	xs, f16, init16, arms := make([]string, 601), make([]string, 16), make([]string, 16), make([]string, 60)
	for i := range xs {
		xs[i] = fmt.Sprintf("%d.5", i)
	}
	for i := range f16 {
		f16[i], init16[i] = fmt.Sprintf("f%d: int", i), fmt.Sprintf("f%d: 0", i)
	}
	for i := range arms {
		arms[i] = fmt.Sprintf("if k == %d then {r | f%d: %d}", i, i%16, i)
	}
	clean = filepath.Join(dir, "clean.ail")
	writeFile(t, clean, fmt.Sprintf(`module clean
import std/list (length)

type R = {%s}

pure func lut() -> [float] = [%s]

pure func pick(r: R, k: int) -> R = %s else r

export func main() -> () ! {IO} {
  println(show(length(lut())));
  println(show(pick({%s}, 59).f11))
}
`, strings.Join(f16, ", "), strings.Join(xs, ", "), strings.Join(arms, " else "), strings.Join(init16, ", ")))
	return wide, clean
}

func TestCheckStrictBytecodeReportsBC001(t *testing.T) {
	bin := buildAilang(t)
	wide, clean := strictBytecodeFixtures(t)
	check := func(args ...string) (string, string, int) {
		return runAilangBin(t, bin, append([]string{"check", "--relax-modules"}, args...)...)
	}

	// Plain check is unchanged: the program is valid AILANG.
	if out, errOut, code := check(wide); code != 0 || strings.Contains(out+errOut, "BC001:") {
		t.Fatalf("plain check must pass without BC001: exit=%d\n%s%s", code, out, errOut)
	}

	// Human: one BC001 per evaluator-only function the module declares,
	// called or not, sorted by name, positioned at the declaration. (Module
	// names of temp-dir files are path-derived, so names are matched by
	// suffix.)
	_, errOut, code := check("--strict-bytecode", wide)
	if code == 0 {
		t.Fatalf("check --strict-bytecode passed a module with evaluator-only functions:\n%s", errOut)
	}
	if n := strings.Count(errOut, "BC001: function "); n != 2 {
		t.Errorf("want 2 BC001 diagnostics, got %d:\n%s", n, errOut)
	}
	unusedAt, wideAt := strings.Index(errOut, "wide.unused is evaluator-only"), strings.Index(errOut, "wide.wideRec is evaluator-only")
	if unusedAt < 0 || wideAt < 0 || unusedAt > wideAt {
		t.Errorf("BC001 must name wide.unused then wide.wideRec:\n%s", errOut)
	}
	if !strings.Contains(errOut, "contiguous block of 300 would exceed 256) at ") || !strings.Contains(errOut, "wide.ail:3") {
		t.Errorf("BC001 lacks the allocator reason or file:line:\n%s", errOut)
	}

	// --json: structured code, file and line.
	out, _, code := check("--strict-bytecode", "--json", wide)
	var rep checkJSONOutput
	if err := json.Unmarshal([]byte(out), &rep); err != nil || code == 0 {
		t.Fatalf("--json: exit=%d err=%v\n%s", code, err, out)
	}
	if rep.Passed || rep.ErrorCount != 2 || len(rep.Errors) != 2 {
		t.Fatalf("--json report: %+v", rep)
	}
	for i, wantLine := range []int{8, 3} {
		e := rep.Errors[i]
		if e.Code != "BC001" || e.Line != wantLine || !strings.HasSuffix(e.File, "wide.ail") {
			t.Errorf("--json error %d = %+v, want BC001 at wide.ail:%d", i, e, wantLine)
		}
	}

	// --format agent: one dense line per diagnostic.
	out, _, code = check("--strict-bytecode", "--format", "agent", wide)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code == 0 || len(lines) != 2 || !strings.Contains(lines[0], "wide.ail:8:0 BC001: function ") || !strings.Contains(lines[0], "wide.unused is evaluator-only") {
		t.Errorf("--format agent: exit=%d\n%s", code, out)
	}

	// Both Phase B repro classes compile; unreached std helpers are silent.
	if out, errOut, code := check("--strict-bytecode", clean); code != 0 || strings.Contains(out+errOut, "BC001:") {
		t.Errorf("check --strict-bytecode on the fixed repros: exit=%d\n%s%s", code, out, errOut)
	}

	// Combinations the gate does not cover are refused, not ignored.
	for _, args := range [][]string{{"--strict-bytecode", "--package", filepath.Dir(wide)}, {"--strict-bytecode", filepath.Dir(wide)}} {
		if _, errOut, code := check(args...); code == 0 || !strings.Contains(errOut, "--strict-bytecode") {
			t.Errorf("%v: want a refusal naming --strict-bytecode, got exit=%d\n%s", args, code, errOut)
		}
	}
}

func TestRunBytecodeReportsEvalOnlyBeforeDispatch(t *testing.T) {
	bin := buildAilang(t)
	wide, _ := strictBytecodeFixtures(t)
	run := func(engine ...string) (string, string, int) {
		args := append([]string{"run", "--relax-modules", "--caps", "IO", "--entry", "main"}, engine...)
		return runAilangBin(t, bin, append(args, wide)...)
	}

	// Strict: rejected before main runs (no "start"), naming only what main
	// reaches — wide.unused cannot trap, so it is not part of a run's report.
	out, errOut, code := run("--bytecode", "--strict-bytecode")
	if code == 0 || strings.Contains(out, "start") {
		t.Fatalf("strict run must fail before dispatch: exit=%d stdout=%q\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "BC001: function ") || !strings.Contains(errOut, "wide.wideRec is evaluator-only") || strings.Contains(errOut, "wide.unused") || strings.Contains(errOut, "vm: CALL") {
		t.Errorf("strict run must report BC001 for wide.wideRec only, not trap mid-run:\n%s", errOut)
	}

	// Non-strict: same stdout as the evaluator, plus a stderr warning.
	evOut, evErr, evCode := run()
	bcOut, bcErr, bcCode := run("--bytecode")
	if evCode != 0 || bcCode != 0 || evOut != bcOut {
		t.Fatalf("evaluator vs --bytecode: exit %d/%d\nstdout:\n%s---\n%s\nstderr:\n%s\n%s", evCode, bcCode, evOut, bcOut, evErr, bcErr)
	}
	if !strings.Contains(bcErr, "1 evaluator-only function(s) reachable from main") || !strings.Contains(bcErr, "wide.wideRec at ") {
		t.Errorf("non-strict --bytecode must warn about the bridged function:\n%s", bcErr)
	}
	if strings.Contains(evErr, "evaluator-only") {
		t.Errorf("the evaluator path printed a bytecode warning:\n%s", evErr)
	}
}

// Strict `ailang test` names an evaluator-only helper a body reaches before
// running that body, instead of trapping mid-run.
func TestStrictTestReportsEvalOnlyHelper(t *testing.T) {
	bin := buildAilang(t)
	wide, _ := strictBytecodeFixtures(t)
	file := filepath.Join(filepath.Dir(wide), "widetest.ail")
	src := strings.Replace(readFile(t, wide), "module wide\n", "module widetest\n", 1) +
		"\ntest \"helper\" { wideRec() == 7 }\n\ntest \"plain\" { 1 + 1 == 2 }\n"
	writeFile(t, file, src)
	out, errOut, code := runAilangBin(t, bin, "test", "--strict-bytecode", "--no-color", file)
	if code == 0 {
		t.Fatalf("strict test passed with an evaluator-only helper:\n%s\n%s", out, errOut)
	}
	all := out + errOut
	if !strings.Contains(all, "reachable from the test body") || !strings.Contains(all, "widetest.wideRec is evaluator-only") || strings.Contains(all, "vm: CALL") {
		t.Errorf("strict test must report the helper as BC001:\n%s", all)
	}
	if !strings.Contains(all, "1 passed, 1 failed") {
		t.Errorf("only the helper-reaching body may fail:\n%s", all)
	}
}
