package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/policy"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-EXECUTOR-POLICY-HARDENING M3 — the supervised policy run.

// writePolicy writes a policy file with the given body plus a sandbox.
func writePolicy(t *testing.T, dir, body string) string {
	t.Helper()
	skipWithoutRestrictedMode(t)
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "policy.toml")
	body = strings.ReplaceAll(body, "${SANDBOX}", filepath.ToSlash(sandbox))
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A blocked-I/O case (sleep never checks a Go context) — the evaluator has
// no TCO, so a CPU spin hits the recursion ceiling before any deadline.
const loopProgram = `module prog
import std/clock (sleep)
export func main() -> () ! {IO, Clock} = { sleep(30000); println("survived") }
`

// AC9: timeout_ms terminates a run that never checks a context, from the
// supervisor, with the envelope naming the reason.
func TestRunPolicy_TimeoutMsEnforced(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"Clock\"]\nentry = \"main\"\ntimeout_ms = 300\n")
	f := writeAil(t, dir, "prog.ail", loopProgram)
	start := time.Now()
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("timeout_ms = 300 took %s to terminate", elapsed)
	}
	if code != 3 {
		t.Fatalf("exit %d, want 3 (policy limit)\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, `"reason":"timeout"`) || !strings.Contains(stderr, "policy-result:") {
		t.Fatalf("envelope must name the timeout: %s", stderr)
	}
	if strings.Contains(stdout, "survived") {
		t.Fatal("the program ran to completion past the deadline")
	}
}

// AC9: a trusted_host Process grandchild does not survive the timeout — the
// supervisor kills the worker's whole process group.
func TestRunPolicy_TimeoutKillsDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("descendant termination is not claimed on Windows (D6)")
	}
	bin := buildAilang(t)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "sandbox", "child.pid")
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nfs_sandbox = \"${SANDBOX}\"\nprocess_allow = [\"sh\"]\nentry = \"main\"\ntimeout_ms = 15000\nmax_output_bytes = 0\n")
	// sh writes its own pid then sleeps far past the deadline; exec's own
	// per-call timeout (30s) is raised below so the POLICY is what fires.
	// 15s leaves room for the binary's startup under a parallel `make test`
	// (measured: 4s was not enough on a loaded rig); the property under test
	// is descendant death, not the exact deadline.
	prog := `module prog
import std/process (exec)
export func main() -> () ! {IO, Process} = {
  match exec("sh", ["-c", "echo $$ > child.pid; sleep 120"]) {
    Ok(_) => println("done"),
    Err(_) => println("err")
  }
}
`
	writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", prog) // inside fs_sandbox (M7)
	_, stderr, code := testutil.RunBounded(t, filepath.Join(dir, "sandbox"), 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 3 || !strings.Contains(stderr, `"reason":"timeout"`) {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the grandchild never wrote its pid (fixture broken): %v\n%s", err, stderr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if pid <= 0 {
		t.Fatalf("bad pid %q", raw)
	}
	// Bounded ESRCH poll: the grandchild must be gone within 2s of return.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if processGone(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	killProcess(pid)
	t.Fatalf("grandchild %d survived the supervisor's group kill", pid)
}

// Every flag that could widen or redirect authority is refused BY NAME.
func TestRunPolicy_RefusesEveryWideningFlag(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\n")
	f := writeAil(t, dir, "prog.ail", ioProgram)
	cases := [][]string{
		{"--caps", "Net"}, {"--no-budgets"}, {"--allow-env", "HOME"}, {"--allow-env-file", "x"},
		{"--env", "A=1"}, {"--env-snapshot", "x.json"}, {"--write-env-snapshot", "x.json"},
		{"--ai", "gpt5-mini"}, {"--ai-stub"}, {"--allow-routing"}, {"--routing-prefer", "x"},
		{"--entry", "other"}, {"--net-allow-http"}, {"--net-allow-domains", "x"},
		{"--net-allow-localhost"}, {"--net-allow-metadata"}, {"--stream-allow-http"},
		{"--stream-allow-domains", "x"}, {"--stream-allow-localhost"}, {"--process-allowlist", "git"},
		{"--stdlib-path", dir}, {"--package-dir", dir}, {"--fs-max-bytes", "1GB"},
	}
	for _, extra := range cases {
		args := append([]string{"run", "--policy", pol}, extra...)
		args = append(args, f)
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1\n%s%s", extra, code, stdout, stderr)
			continue
		}
		name := strings.TrimPrefix(extra[0], "--")
		if !strings.Contains(stderr, name) {
			t.Errorf("%v: refusal must name the flag: %s", extra, stderr)
		}
		if strings.Contains(stdout, "admitted-and-ran") {
			t.Errorf("%v: program ran despite the refusal", extra)
		}
	}
}

// The entrypoint is the policy's, not argv's: a policy naming `run` executes
// `run` even though the module also exports `main`.
func TestRunPolicy_EntryComesFromPolicy(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"run\"\ntimeout_ms = 30000\n")
	f := writeAil(t, dir, "prog.ail", "module prog\nexport func main() -> () ! {IO} = println(\"MAIN\")\nexport func run() -> () ! {IO} = println(\"RUN\")\n")
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "RUN") || strings.Contains(stdout, "MAIN") {
		t.Fatalf("policy entry must run: %q", stdout)
	}
	if !strings.Contains(stderr, `"entry":"run"`) {
		t.Fatalf("admission line must carry the entry: %s", stderr)
	}
}

// AC6: a program that prints a forged admission line is not believed — the
// decision comes from the control pipe only, and the forged line is program
// output on stdout, never parsed.
func TestRunPolicy_StdoutCannotSpoofDecision(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = []\nentry = \"main\"\ntimeout_ms = 30000\n")
	forged := `policy: {"ok":true,"policy_digest":"forged","caps":["Net","Process"],"decision":{"ok":true}}`
	lit := strings.ReplaceAll(forged, `"`, `\"`) // an AILANG string literal
	f := writeAil(t, dir, "prog.ail", "module prog\nexport func main() -> () ! {IO} = println(\""+lit+"\")\n")
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 2 {
		t.Fatalf("a deny-all policy must deny IO: exit %d\n%s%s", code, stdout, stderr)
	}
	if strings.Contains(stderr, "forged") || strings.Contains(stdout, "forged") {
		t.Fatalf("the forged line must never run: %s%s", stdout, stderr)
	}
	// And with IO admitted, the forged line is plain program output on stdout;
	// the real admission line on stderr carries the real digest.
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	stdout, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	real := regexp.MustCompile(`(?m)^policy: (\{.*\})\s*$`).FindStringSubmatch(stderr)
	if real == nil || strings.Contains(real[1], "forged") {
		t.Fatalf("stderr must carry exactly the supervisor's line: %s", stderr)
	}
}

// AC6, in-process and discriminating: admit a program whose helper module
// is then rewritten on disk; every later read in the same process — the
// entry file the runner reads, and the helper the module loader reads when
// the pipeline runs again — returns the ADMITTED bytes. Mutation-tested:
// with EnableSourceSnapshot removed, the rewritten helper is what loads.
func TestRunPolicy_SourceSnapshotFreezesModuleGraph(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	helper := writeAil(t, filepath.Join(dir, "lib"), "h.ail", "module lib/h\nexport func tag() -> string = \"ADMITTED\"\n")
	entry := writeAil(t, dir, "prog.ail", "module prog\nimport lib/h as H\nexport func main() -> string ! {} = H.tag()\n")
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\n")
	res, _, err := policy.LoadResolved(pol)
	if err != nil {
		t.Fatal(err)
	}
	// The module loader resolves imports against the cwd.
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	loader.EnableSourceSnapshot(res.MaxSourceBytes, res.MaxModuleGraphBytes)
	t.Cleanup(loader.ResetSourceSnapshotForTest)
	out, code := admitProgram(res, pol, "prog.ail")
	if code != 0 {
		t.Fatalf("admission failed: %+v", out)
	}
	// Rewrite BOTH files after admission: the helper now needs Process and
	// the entry declares it.
	if err := os.WriteFile(helper, []byte("module lib/h\nimport std/process (exec)\nexport func tag() -> string ! {Process} = { exec(\"echo\", []); \"REWRITTEN\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte("module prog\nimport lib/h as H\nexport func main() -> string ! {Process} = H.tag()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// What the runner would read for the entry:
	got, err := loader.ReadSourceFile(entry)
	if err != nil || strings.Contains(string(got), "Process") {
		t.Fatalf("entry read after admission must be the admitted bytes: %q %v", got, err)
	}
	// What the module loader reads for the helper when the pipeline runs
	// again (the execution phase): the same typecheck must still pass with
	// the pure helper, and the module graph digest must not move.
	d1, n1, _ := loader.SourceSnapshotDigest()
	out2, code2 := admitProgram(res, pol, "prog.ail")
	if code2 != 0 {
		t.Fatalf("re-admission saw the rewritten graph: %+v", out2)
	}
	d2, n2, _ := loader.SourceSnapshotDigest()
	if d1 != d2 || n1 != n2 || n1 < 2 {
		t.Fatalf("module graph digest moved or is incomplete: %s/%d vs %s/%d", d1, n1, d2, n2)
	}
}

// AC6 positive control, end to end: the admission line banks the module
// graph digest, and a rewrite of the helper during execution is inert.
func TestRunPolicy_SourceSnapshotAcrossAdmitAndRun(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n")
	// Program and helper live INSIDE the sandbox (M7 rule); the helper is
	// rewritten during execution and must still be the admitted one.
	sb := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(filepath.Join(sb, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	helperPath := writeAil(t, filepath.Join(sb, "lib"), "h.ail", "module lib/h\nexport func tag() -> string = \"ADMITTED\"\n")
	prog := `module prog
import std/fs (writeFile)
import lib/h as H
export func main() -> () ! {IO, FS} = {
  writeFile("marker.txt", "x");
  println(H.tag())
}
`
	f := writeAil(t, sb, "prog.ail", prog)
	// Race the rewrite against the run: overwrite the helper as soon as the
	// sandbox marker appears (i.e. after admission, during execution).
	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(dir, "sandbox", "marker.txt")); err == nil {
				_ = os.WriteFile(helperPath, []byte("module lib/h\nexport func tag() -> string = \"REWRITTEN\"\n"), 0o644)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()
	// Module paths resolve against the cwd: run from the sandbox.
	_ = f
	stdout, stderr, code := testutil.RunBounded(t, sb, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	<-done
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "ADMITTED") || strings.Contains(stdout, "REWRITTEN") {
		t.Fatalf("executed the rewritten helper: %q", stdout)
	}
	if !strings.Contains(stderr, `"module_graph":{"digest":"`) {
		t.Fatalf("admission line must bank the module-graph digest: %s", stderr)
	}
}

// AC7/AC11: restricted mode refuses Process at startup with the migration
// named; Stream's process source needs Process, which restricted never has.
func TestRunPolicy_RestrictedRefusesProcessAndStreamProcessSource(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"Process\"]\nprocess_allow = [\"echo\"]\nentry = \"main\"\n")
	f := writeAil(t, dir, "prog.ail", ioProgram)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, `"echo"`) || !strings.Contains(stderr, "trusted_host") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	// A Stream-only policy: a program using asyncExecProcess cannot even
	// typecheck as {Stream} (the row is {Stream, Process}); declared honestly
	// it is denied by admission with Process in missing_from_policy.
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"Stream\"]\nnet_allow = [\"api.example\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	f = writeAil(t, dir, "sp.ail", "module sp\nimport std/stream (asyncExecProcess, StreamSource)\nexport func main() -> StreamSource ! {Stream, Process} = asyncExecProcess(\"echo\", [\"x\"], \"s\", 1, 64)\n")
	stdout, _, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 2 {
		t.Fatalf("exit %d, want 2\n%s", code, stdout)
	}
	d := decisionFrom(t, stdout)
	if d["error_kind"] != "policy_violation" || !strings.Contains(stdout, `"Process"`) {
		t.Fatalf("Stream's process source must be denied for Process: %s", stdout)
	}
}

// AC9: output stops at the cap; the worker is killed, not buffered.
func TestRunPolicy_OutputCapStopsProduction(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\nmax_output_bytes = 4096\ntimeout_ms = 10000\n")
	prog := `module prog
export func spam(n: int) -> () ! {IO} = if n == 0 then () else { println("0123456789012345678901234567890123456789"); spam(n - 1) }
export func main() -> () ! {IO} = spam(100000)
`
	f := writeAil(t, dir, "prog.ail", prog)
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 3 || !strings.Contains(stderr, `"reason":"output_limit"`) {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	if len(stdout) > 4096+64*1024 {
		t.Fatalf("stdout kept flowing past the cap: %d bytes", len(stdout))
	}
}

// AC11: the restricted worker sees only the allowlisted environment.
func TestRunPolicy_RestrictedWorkerEnvIsAllowlisted(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"Env\"]\nentry = \"main\"\n")
	f := writeAil(t, dir, "prog.ail", ioProgram)
	// Env is refused in restricted mode outright; the allowlist is proven
	// through the worker's own process environment instead.
	if _, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f); code != 1 || !strings.Contains(stderr, "Env") {
		t.Fatalf("Env must be refused in restricted mode: %d %s", code, stderr)
	}
	t.Setenv("OPENROUTER_API_KEY", "leak-me")
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nprocess_allow = [\"sh\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	prog := `module prog
import std/process (exec)
import std/bytes (toString)
export func main() -> () ! {IO, Process} = match exec("sh", ["-c", "echo KEY=$OPENROUTER_API_KEY"]) { Ok(r) => println(toString(r.stdout)), Err(_) => println("err") }
`
	f = writeAil(t, dir, "env.ail", prog)
	stdout, _, _ := runAilangBin(t, bin, "run", "--policy", pol, f)
	if !strings.Contains(stdout, "KEY=leak-me") {
		t.Fatalf("trusted_host passes the operator's environment through: %q", stdout)
	}
}

// M7: with an fs_sandbox, the program file must be inside it — an agent
// could otherwise execute (and read) any .ail on the host under the policy.
func TestRunPolicy_EntryOutsideSandboxRefused(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n")
	outside := writeAil(t, dir, "elsewhere.ail", ioProgram) // dir, not dir/sandbox
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, outside)
	if code != 1 || !strings.Contains(stderr, "outside fs_sandbox") || strings.Contains(stdout, "admitted-and-ran") {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	inside := writeAil(t, filepath.Join(dir, "sandbox"), "prog.ail", ioProgram)
	stdout, stderr, code = testutil.RunBounded(t, filepath.Join(dir, "sandbox"), 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 0 || !strings.Contains(stdout, "admitted-and-ran") {
		t.Fatalf("inside must run: exit %d\n%s%s", code, stdout, stderr)
	}
	_ = inside
}

// M7: AI runs in restricted mode with a pinned provider and a budget; the
// budget is the ceiling.
func TestRunPolicy_RestrictedAIWithBudget(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	prog := "module prog\nimport std/ai (call)\nexport func main() -> () ! {IO, AI} = { println(call(\"one\")); println(call(\"two\")) }\n"
	f := writeAil(t, dir, "prog.ail", prog)
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"AI\"]\nai_provider = \"stub\"\nentry = \"main\"\ntimeout_ms = 30000\n[budgets]\nAI = 2\n")
	stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 0 {
		t.Fatalf("restricted AI with provider + budget must run: exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, `"security_mode":"restricted"`) || !strings.Contains(stderr, `"ai_provider":"stub"`) || !strings.Contains(stderr, `"budgets":{"AI":2}`) {
		t.Fatalf("admission line must bank mode, provider and budget: %s", stderr)
	}
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"AI\"]\nai_provider = \"stub\"\nentry = \"main\"\ntimeout_ms = 30000\n[budgets]\nAI = 1\n")
	stdout, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, f)
	if code == 0 || !strings.Contains(stderr, "E_BUDGET_OPERATOR") || !strings.Contains(stderr, "'AI'") {
		t.Fatalf("the second call must exceed AI = 1: exit %d\n%s%s", code, stdout, stderr)
	}
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"AI\"]\nai_provider = \"stub\"\nentry = \"main\"\n")
	_, stderr, code = runAilangBin(t, bin, "run", "--policy", pol, f)
	if code != 1 || !strings.Contains(stderr, "[budgets] AI") {
		t.Fatalf("restricted AI without a budget must be refused by name: exit %d\n%s", code, stderr)
	}
}

// M7: fs_deny_write, end to end.
func TestRunPolicy_FSDenyWriteE2E(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	sandbox := filepath.Join(dir, "sandbox")
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nfs_deny_write = [\".github/**\", \"Makefile\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	if err := os.MkdirAll(filepath.Join(sandbox, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	prog := `module prog
import std/fs (writeFileResult)
export func try(p: string) -> () ! {IO, FS} = match writeFileResult(p, "x") {
  Ok(_) => println("WROTE:${p}"),
  Err(e) => println("PROTECTED:${p}")
}
export func main() -> () ! {IO, FS} = { try(".github/workflows/ci.yml"); try("Makefile"); try("src.ail.txt") }
`
	writeAil(t, sandbox, "prog.ail", prog)
	stdout, stderr, code := testutil.RunBounded(t, sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "PROTECTED:.github/workflows/ci.yml") || !strings.Contains(stdout, "PROTECTED:Makefile") || !strings.Contains(stdout, "WROTE:src.ail.txt") {
		t.Fatalf("%q", stdout)
	}
	if _, err := os.Stat(filepath.Join(sandbox, "Makefile")); err == nil {
		t.Fatal("Makefile was written")
	}
}

// M7: only the pinned provider's credentials reach a restricted worker.
func TestWorkerEnv_ProviderScopedCredentials(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "or-secret")
	t.Setenv("GOOGLE_API_KEY", "g-secret")
	t.Setenv("GEMINI_API_KEY", "gem-secret")
	t.Setenv("ANTHROPIC_API_KEY", "a-secret")
	has := func(env []string, name string) bool {
		for _, kv := range env {
			if strings.HasPrefix(kv, name+"=") {
				return true
			}
		}
		return false
	}
	noAI := &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"IO"}}
	env := workerEnv(noAI)
	for _, k := range []string{"OPENROUTER_API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY", "ANTHROPIC_API_KEY"} {
		if has(env, k) {
			t.Errorf("no-AI worker must not see %s", k)
		}
	}
	gem := &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"AI", "IO"}, AIProvider: "gemini-3-5-flash-lite"}
	env = workerEnv(gem)
	if !has(env, "GOOGLE_API_KEY") || !has(env, "GEMINI_API_KEY") {
		t.Errorf("gemini worker must see the Google credentials: %v", env)
	}
	if has(env, "OPENROUTER_API_KEY") || has(env, "ANTHROPIC_API_KEY") {
		t.Errorf("gemini worker must not see other providers' keys")
	}
	stub := &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"AI", "IO"}, AIProvider: "stub"}
	if env := workerEnv(stub); has(env, "GOOGLE_API_KEY") || has(env, "OPENROUTER_API_KEY") {
		t.Errorf("stub needs no credentials")
	}
	trusted := &policy.Resolved{Mode: policy.ModeTrustedHost, Effects: []string{"IO"}}
	if env := workerEnv(trusted); !has(env, "OPENROUTER_API_KEY") {
		t.Errorf("trusted_host gets the operator's full environment")
	}
}

// std/web reads OLLAMA_API_KEY in Go, so a restricted worker granted Net to
// the backend host needs it whatever the AI cap says (task-386cc079: 16/16
// webFetch/webSearch failed with the key unset). Any other net_allow, or a
// Net-less policy, gets no key.
func TestWorkerEnv_WebBackendCredential(t *testing.T) {
	t.Setenv("OLLAMA_API_KEY", "ol-secret")
	t.Setenv("OPENROUTER_API_KEY", "or-secret")
	has := func(env []string, name string) bool {
		for _, kv := range env {
			if strings.HasPrefix(kv, name+"=") {
				return true
			}
		}
		return false
	}
	cases := []struct {
		name string
		res  *policy.Resolved
		want bool
	}{
		{"net ollama.com, no AI", &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"IO", "Net"}, NetAllow: []string{"ollama.com"}}, true},
		{"net ollama.com, AI on gemini", &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"AI", "IO", "Net"}, AIProvider: "gemini-3-5-flash-lite", NetAllow: []string{"example.org", "ollama.com"}}, true},
		{"net without ollama.com", &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"IO", "Net"}, NetAllow: []string{"example.org"}}, false},
		{"lookalike host", &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"IO", "Net"}, NetAllow: []string{"evil-ollama.com", "*.ollama.com"}}, false},
		{"net_allow without Net cap", &policy.Resolved{Mode: policy.ModeRestricted, Effects: []string{"IO"}, NetAllow: []string{"ollama.com"}}, false},
	}
	for _, tc := range cases {
		env := workerEnv(tc.res)
		if got := has(env, "OLLAMA_API_KEY"); got != tc.want {
			t.Errorf("%s: OLLAMA_API_KEY present=%v, want %v", tc.name, got, tc.want)
		}
		if has(env, "OPENROUTER_API_KEY") {
			t.Errorf("%s: an unrelated key leaked", tc.name)
		}
	}
}
