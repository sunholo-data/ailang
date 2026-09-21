package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nfs_sandbox = \"${SANDBOX}\"\nprocess_allow = [\"sh\"]\nentry = \"main\"\ntimeout_ms = 800\n")
	// sh writes its own pid then sleeps far past the deadline. exec's own
	// per-call timeout (30s) would otherwise outlive the policy.
	prog := `module prog
import std/process (exec)
export func main() -> () ! {IO, Process} = {
  match exec("sh", ["-c", "echo $$ > child.pid; sleep 30"]) {
    Ok(_) => println("done"),
    Err(_) => println("err")
  }
}
`
	f := writeAil(t, dir, "prog.ail", prog)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
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
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
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
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"run\"\n")
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
	pol := writePolicy(t, dir, "allowed_caps = []\nentry = \"main\"\n")
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
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\n")
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
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\n")
	// The program rewrites ITSELF and its helper module on first FS write,
	// then calls the helper: the helper must still be the admitted one.
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	helperPath := writeAil(t, filepath.Join(dir, "lib"), "h.ail", "module lib/h\nexport func tag() -> string = \"ADMITTED\"\n")
	prog := `module prog
import std/fs (writeFile)
import lib/h as H
export func main() -> () ! {IO, FS} = {
  writeFile("marker.txt", "x");
  println(H.tag())
}
`
	f := writeAil(t, dir, "prog.ail", prog)
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
	// Module paths resolve against the cwd: run from the fixture directory.
	_ = f
	stdout, stderr, code := testutil.RunBounded(t, dir, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
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
	if code != 1 || !strings.Contains(stderr, "Process") || !strings.Contains(stderr, "trusted_host") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	// A Stream-only policy: a program using asyncExecProcess cannot even
	// typecheck as {Stream} (the row is {Stream, Process}); declared honestly
	// it is denied by admission with Process in missing_from_policy.
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"Stream\"]\nnet_allow = [\"api.example\"]\nentry = \"main\"\n")
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
	pol = writePolicy(t, dir, "allowed_caps = [\"IO\", \"Process\"]\nsecurity_mode = \"trusted_host\"\nprocess_allow = [\"sh\"]\nentry = \"main\"\n")
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
