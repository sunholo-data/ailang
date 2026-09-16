package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-AGENT-AILANG-ONLY-EXECUTION M2: `ailang run --policy` is the ONE admission
// gate. The caller (an agent) is not trusted: caps, net allowlist and FS
// sandbox come from the policy, and the flags an agent could use to widen
// them are refused outright.

func writePolicyFixture(t *testing.T, dir, caps string) string {
	t.Helper()
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "policy.toml")
	body := "allowed_caps = [" + caps + "]\nfs_sandbox = \"" + filepath.ToSlash(sandbox) + "\"\nentry = \"main\"\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeAil(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const ioProgram = `module prog
export func main() -> () ! {IO} = println("admitted-and-ran")
`

func decisionFrom(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not the decision JSON: %v\n%s", err, stdout)
	}
	d, _ := out["decision"].(map[string]any)
	if d == nil {
		t.Fatalf("no decision in %s", stdout)
	}
	return d
}

func TestRunPolicy_AdmittedProgramRuns(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO"`)
	f := writeAil(t, dir, "prog.ail", ioProgram)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "admitted-and-ran") {
		t.Fatalf("program output missing from stdout: %s", stdout)
	}
	// The admission decision goes to stderr as one line, so stdout stays the program's.
	if !strings.Contains(stderr, `"ok":true`) || !strings.Contains(stderr, "policy_digest") {
		t.Fatalf("admission line missing from stderr: %s", stderr)
	}
}

func TestRunPolicy_DefaultDenyRefusesIO(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, ``)
	f := writeAil(t, dir, "prog.ail", ioProgram)
	stdout, _, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 2 {
		t.Fatalf("exit %d, want 2 (denied)\n%s", code, stdout)
	}
	d := decisionFrom(t, stdout)
	if d["error_kind"] != "policy_violation" {
		t.Fatalf("error_kind = %v, want policy_violation", d["error_kind"])
	}
	if strings.Contains(stdout, "admitted-and-ran") {
		t.Fatal("a DENIED program must not execute")
	}
}

func TestRunPolicy_LyingEntryIsTypecheckFailed(t *testing.T) {
	// A clean-row main that calls an imported ! {Net} function: the
	// typechecker rejects it before the policy is consulted (V14 refuted —
	// transitive effects are enforced by construction). Pinned here so a
	// future DryLink change that stops checking imports fails loudly.
	bin := buildAilang(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeAil(t, filepath.Join(dir, "lib"), "netty.ail", "module lib/netty\nimport std/net as N\nexport func fetch(u: string) -> string ! {Net} = N.httpGet(u)\n")
	f := writeAil(t, dir, "lying.ail", "module lying\nimport lib/netty as L\nexport func main() -> string ! {} = L.fetch(\"http://x\")\n")
	pol := writePolicyFixture(t, dir, `"IO"`)
	stdout, _, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 2 {
		t.Fatalf("exit %d, want 2\n%s", code, stdout)
	}
	if d := decisionFrom(t, stdout); d["error_kind"] != "typecheck_failed" {
		t.Fatalf("error_kind = %v, want typecheck_failed", d["error_kind"])
	}
}

func TestRunPolicy_RefusesWideningFlags(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO"`)
	f := writeAil(t, dir, "prog.ail", ioProgram)
	for _, extra := range [][]string{
		{"--caps", "Net"},
		{"--no-budgets"},
		{"--allow-env", "HOME"},
	} {
		args := append([]string{"run", "--policy", pol}, extra...)
		args = append(args, f)
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1 (refused)\n%s%s", extra, code, stdout, stderr)
		}
		if !strings.Contains(stderr, extra[0]) {
			t.Errorf("%v: refusal must name the flag: %s", extra, stderr)
		}
		if strings.Contains(stdout, "admitted-and-ran") {
			t.Errorf("%v: program ran despite the refusal", extra)
		}
	}
}

func TestRunPolicy_RefusesUnenforceableBudgets(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO"`)
	if err := os.WriteFile(pol, []byte("allowed_caps = [\"IO\"]\nentry = \"main\"\n[budgets]\nIO = 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := writeAil(t, dir, "prog.ail", ioProgram)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, "budgets") {
		t.Fatalf("a policy with [budgets] must be refused (no run-time hook enforces them yet): exit %d\n%s", code, stderr)
	}
}

func TestRunPolicy_UnknownCapInPolicyIsLoud(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO", "Nope"`)
	f := writeAil(t, dir, "prog.ail", ioProgram)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, "Nope") {
		t.Fatalf("an unknown capability in the policy must be refused by name: exit %d\n%s", code, stderr)
	}
}
