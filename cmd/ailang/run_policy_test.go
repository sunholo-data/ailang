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

// M-EXECUTOR-POLICY-HARDENING M4: [budgets] are the operator ceiling and ARE
// enforced (they used to be refused as unenforceable). An explicit IO = 0
// denies the very first println before its side effect; the admission line
// banks the budgets.
func TestRunPolicy_BudgetsEnforcedAsOperatorCeiling(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyFixture(t, dir, `"IO"`)
	if err := os.WriteFile(pol, []byte("allowed_caps = [\"IO\"]\nentry = \"main\"\n[budgets]\nIO = 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := writeAil(t, dir, "prog.ail", ioProgram)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code == 0 || strings.Contains(stdout, "admitted-and-ran") {
		t.Fatalf("IO = 0 must deny the first println: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "E_BUDGET_OPERATOR") {
		t.Fatalf("denial must be the operator-budget error: %s", stderr)
	}
	if !strings.Contains(stderr, `"budgets":{"IO":0}`) {
		t.Fatalf("admission line must bank the budgets: %s", stderr)
	}
}

func TestRunPolicy_FineGrainedCapsNarrowOrRefuse(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	// Process is a host integration: restricted mode refuses it BY NAME with
	// the trusted_host migration (M-EXECUTOR-POLICY-HARDENING D3) …
	pol := writePolicyFixture(t, dir, `"IO", "Process"`)
	f := writeAil(t, dir, "prog.ail", ioProgram)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, "Process") || !strings.Contains(stderr, "trusted_host") {
		t.Fatalf("Process under restricted mode must be refused naming the migration: exit %d\n%s", code, stderr)
	}
	// … and in trusted_host, Process without process_allow is refused by name.
	if err := os.WriteFile(pol, []byte("allowed_caps = [\"IO\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nentry = \"main\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, "process_allow") {
		t.Fatalf("Process with no process_allow must be refused: exit %d\n%s", code, stderr)
	}
	// process_allow narrows: the admission line carries it, and the run's
	// Process effect handler receives it as --process-allowlist (the program
	// here uses IO only, so admission and the flag plumbing are what is tested).
	if err := os.WriteFile(pol, []byte("allowed_caps = [\"IO\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nprocess_allow = [\"git:pull\", \"git:status\"]\nentry = \"main\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 || !strings.Contains(stdout, "admitted-and-ran") {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, `"process_allow":["git:pull","git:status"]`) {
		t.Fatalf("admission line must name the process allowlist: %s", stderr)
	}
	// Net without net_allow: refused.
	if err := os.WriteFile(pol, []byte("allowed_caps = [\"IO\", \"Net\"]\nentry = \"main\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f); code != 1 || !strings.Contains(stderr, "net_allow") {
		t.Fatalf("Net with no net_allow must be refused: exit %d\n%s", code, stderr)
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

// M-DANEEL-AILANG-EXECUTOR M1: the policy pins the AI provider. Under --policy
// the model an AI-cap program talks to is the operator's decision — the
// lending boundary — so --ai/--ai-stub are widening flags, an AI grant without
// ai_provider is refused, and ai_provider without the AI cap is refused.
func writePolicyWithAI(t *testing.T, dir, caps, aiProvider string) string {
	t.Helper()
	p := writePolicyFixture(t, dir, caps)
	// AI is a host integration: only trusted_host admits it (D3), so these
	// fixtures test the provider rules in that mode.
	extra := "security_mode = \"trusted_host\"\n"
	if aiProvider != "" {
		extra += "ai_provider = \"" + aiProvider + "\"\n"
	}
	b, _ := os.ReadFile(p)
	if err := os.WriteFile(p, append(b, []byte(extra)...), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const aiProgram = `module prog
import std/ai (call)
export func main() -> () ! {IO, AI} = println(call("say hi"))
`

func TestRunPolicy_AIProvider_RefusesAIFlagsUnderPolicy(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyWithAI(t, dir, `"IO", "AI"`, "stub")
	f := writeAil(t, dir, "prog.ail", aiProgram)
	for _, extra := range [][]string{{"--ai", "gpt5-mini"}, {"--ai-stub"}} {
		args := append([]string{"run", "--policy", pol}, extra...)
		args = append(args, f)
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1 (refused)\n%s%s", extra, code, stdout, stderr)
		}
		if !strings.Contains(stderr, extra[0]) || !strings.Contains(stderr, "ai_provider") {
			t.Errorf("%v: refusal must name the flag and ai_provider: %s", extra, stderr)
		}
	}
}

func TestRunPolicy_AIProvider_CapAndProviderMustAgree(t *testing.T) {
	bin := buildAilang(t)
	for _, tc := range []struct{ name, caps, provider, want string }{
		{"AI cap without ai_provider", `"IO", "AI"`, "", "ai_provider"},
		{"ai_provider without AI cap", `"IO"`, "stub", "AI is not in allowed_caps"},
	} {
		dir := t.TempDir()
		pol := writePolicyWithAI(t, dir, tc.caps, tc.provider)
		f := writeAil(t, dir, "prog.ail", ioProgram)
		stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
		if code != 1 {
			t.Errorf("%s: exit %d, want 1\n%s%s", tc.name, code, stdout, stderr)
		}
		if !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: refusal must say %q: %s", tc.name, tc.want, stderr)
		}
		if strings.Contains(stdout, "admitted-and-ran") {
			t.Errorf("%s: program ran despite the refusal", tc.name)
		}
	}
}

func TestRunPolicy_AIProvider_StubRunsWithoutAIFlag(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicyWithAI(t, dir, `"IO", "AI"`, "stub")
	f := writeAil(t, dir, "prog.ail", aiProgram)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 {
		t.Fatalf("stub-admitted AI program: exit %d\n%s%s", code, stdout, stderr)
	}
	// the admission line records the pinned provider
	if !strings.Contains(stderr, `"ai_provider":"stub"`) {
		t.Errorf("admission line must carry ai_provider: %s", stderr)
	}
}
