package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-EXECUTOR-POLICY-HARDENING M5 (AC1) — the audited counterexamples, end to
// end through `ailang run --policy` on the checkout binary: F1 (relative
// traversal), F2 (in-root symlink out) and N1 (redirect to a non-allowlisted
// host). Each program tries the escape and prints what it got; the assertion
// is that the sentinel never reaches stdout and the outside tree is intact.

type containmentE2E struct {
	dir, sandbox, sentinel string
}

func newContainmentE2E(t *testing.T) *containmentE2E {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows; restricted mode is refused there (D6)")
	}
	dir := t.TempDir()
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	c := &containmentE2E{dir: dir, sandbox: filepath.Join(dir, "sandbox"), sentinel: "E2E-SENTINEL-" + filepath.Base(dir)}
	if err := os.MkdirAll(c.sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "marker.txt"), []byte(c.sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../marker.txt", filepath.Join(c.sandbox, "link.txt")); err != nil {
		t.Fatal(err)
	}
	return c
}

func (c *containmentE2E) policy(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(c.dir, "policy.toml")
	body = strings.ReplaceAll(body, "${SANDBOX}", filepath.ToSlash(c.sandbox))
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// fsProbe reads a path with readFileResult and prints the outcome.
const fsProbe = `module prog
import std/fs (readFileResult, writeFileResult)
export func main() -> () ! {IO, FS} = {
  match readFileResult(PATH) {
    Ok(s) => println("READ:${s}"),
    Err(e) => println("DENIED:${e}")
  };
  match writeFileResult(WPATH, "clobbered") {
    Ok(_) => println("WROTE"),
    Err(e) => println("WDENIED:${e}")
  }
}
`

func TestRunPolicyE2E_FSTraversalAndSymlinkDenied(t *testing.T) {
	bin := buildAilang(t)
	for _, tc := range []struct{ name, read, write string }{
		{"traversal", "../marker.txt", "../leak.txt"},
		{"symlink", "link.txt", "link.txt"},
		{"absolute", "", ""}, // filled below with the absolute outside path
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newContainmentE2E(t)
			if tc.read == "" {
				tc.read = filepath.Join(c.dir, "marker.txt")
				tc.write = filepath.Join(c.dir, "leak.txt")
			}
			pol := c.policy(t, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n")
			src := strings.ReplaceAll(strings.ReplaceAll(fsProbe, "WPATH", `"`+tc.write+`"`), "PATH", `"`+tc.read+`"`)
			writeAil(t, c.sandbox, "prog.ail", src)
			stdout, stderr, code := testutil.RunBounded(t, c.sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
			if code != 0 {
				t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
			}
			if strings.Contains(stdout, c.sentinel) || strings.Contains(stdout, "READ:") {
				t.Fatalf("CONTAINMENT FAILURE (%s): %s", tc.name, stdout)
			}
			if !strings.Contains(stdout, "DENIED:") || !strings.Contains(stdout, "WDENIED:") {
				t.Fatalf("both probes must be denied: %s", stdout)
			}
			if b, err := os.ReadFile(filepath.Join(c.dir, "marker.txt")); err != nil || string(b) != c.sentinel {
				t.Fatalf("outside marker changed: %q %v", b, err)
			}
			if _, err := os.Stat(filepath.Join(c.dir, "leak.txt")); err == nil {
				t.Fatal("a file was created outside the sandbox")
			}
		})
	}
}

// Positive control: the same program on an in-root file reads and writes.
func TestRunPolicyE2E_FSInsideRootWorks(t *testing.T) {
	bin := buildAilang(t)
	c := newContainmentE2E(t)
	if err := os.WriteFile(filepath.Join(c.sandbox, "data.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	pol := c.policy(t, "allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = \"${SANDBOX}\"\nentry = \"main\"\ntimeout_ms = 30000\n")
	src := strings.ReplaceAll(strings.ReplaceAll(fsProbe, "WPATH", `"out.txt"`), "PATH", `"data.txt"`)
	writeAil(t, c.sandbox, "prog.ail", src)
	stdout, stderr, code := testutil.RunBounded(t, c.sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 0 || !strings.Contains(stdout, "READ:inside") || !strings.Contains(stdout, "WROTE") {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
}

// N1 end to end. A loopback listener is reachable only under trusted_host
// (an explicit loopback entry in net_allow is that grant; restricted mode
// has none), which is fine: the property under test is the per-hop domain
// check, identical in both modes.
func TestRunPolicyE2E_RedirectToNonAllowlistedHostDenied(t *testing.T) {
	bin := buildAilang(t)
	c := newContainmentE2E(t)
	var finalHits atomic.Int32
	var port string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "http://localhost:"+port+"/final", http.StatusFound)
		case "/same":
			http.Redirect(w, r, "http://127.0.0.1:"+port+"/final", http.StatusFound)
		case "/final":
			finalHits.Add(1)
			_, _ = w.Write([]byte(c.sentinel))
		}
	}))
	defer srv.Close()
	_, port, _ = net.SplitHostPort(srv.Listener.Addr().String())

	pol := c.policy(t, "allowed_caps = [\"IO\", \"Net\"]\nsecurity_mode = \"trusted_host\"\nnet_allow = [\"127.0.0.1\"]\nnet_allow_http = true\nentry = \"main\"\ntimeout_ms = 30000\n")
	prog := `module prog
import std/net (httpRequest)
export func fetch(u: string) -> string ! {Net} = match httpRequest("GET", u, [], "") {
  Ok(resp) => "BODY:${resp.body}",
  Err(_) => "DENIED"
}
export func main() -> () ! {IO, Net} = {
  println(fetch("http://127.0.0.1:` + port + `/start"));
  println(fetch("http://127.0.0.1:` + port + `/same"))
}
`
	writeAil(t, c.sandbox, "prog.ail", prog)
	stdout, stderr, code := testutil.RunBounded(t, c.sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	lines := programLines(stdout)
	if len(lines) < 2 || lines[0] != "DENIED" {
		t.Fatalf("ALLOWLIST FAILURE: redirect to localhost must be denied, got %q\n%s", stdout, stderr)
	}
	if lines[1] != "BODY:"+c.sentinel {
		t.Fatalf("same-host redirect is the positive control, got %q", lines[1])
	}
	if finalHits.Load() != 1 {
		t.Fatalf("/final hit %d times, want exactly 1 (the allowlisted hop)", finalHits.Load())
	}
}

// Restricted mode refuses the loopback grant outright: the listener is never
// reached, with the refusal naming the reason (no permissive fallback, AC11).
func TestRunPolicyE2E_RestrictedHasNoLoopbackGrant(t *testing.T) {
	bin := buildAilang(t)
	c := newContainmentE2E(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	pol := c.policy(t, "allowed_caps = [\"IO\", \"Net\"]\nnet_allow = [\"127.0.0.1\"]\nnet_allow_http = true\nentry = \"main\"\ntimeout_ms = 30000\n")
	prog := `module prog
import std/net (httpRequest)
export func main() -> () ! {IO, Net} = match httpRequest("GET", "http://127.0.0.1:` + port + `/x", [], "") {
  Ok(resp) => println("BODY:${resp.body}"),
  Err(_) => println("DENIED")
}
`
	writeAil(t, c.sandbox, "prog.ail", prog)
	stdout, _, _ := testutil.RunBounded(t, c.sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if !strings.Contains(stdout, "DENIED") || hits.Load() != 0 {
		t.Fatalf("restricted mode must not reach loopback: %q, hits=%d", stdout, hits.Load())
	}
}

// programLines drops the runner's status lines ("→ Type checking…",
// "✓ Running…") and returns what the program itself printed.
func programLines(stdout string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if strings.HasPrefix(l, "→") || strings.HasPrefix(l, "✓") || strings.TrimSpace(l) == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

// M6 end to end, with the deployed lane policy's exact shape
// (allowed_caps IO/FS/Process, process_allow git:status/diff/log, restricted):
// read-only git works inside the clone; a planted repo config does not
// execute; an outside-reaching flag is NotAllowed; the program cannot write
// .git/.
func TestRunPolicyE2E_ConfinedGitInRestrictedMode(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	bin := buildAilang(t)
	c := newContainmentE2E(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = c.sandbox
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(c.sandbox, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	pwned := filepath.Join(c.dir, "pwned")
	cfg := filepath.Join(c.sandbox, ".git", "config")
	b, _ := os.ReadFile(cfg)
	if err := os.WriteFile(cfg, append(b, []byte("[core]\n\tfsmonitor = \"touch "+pwned+"\"\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	pol := c.policy(t, "allowed_caps = [\"IO\", \"FS\", \"Process\"]\nfs_sandbox = \"${SANDBOX}\"\nprocess_allow = [\"git:status\", \"git:diff\", \"git:log\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	prog := `module prog
import std/process (exec)
import std/bytes (toString)
import std/fs (writeFileResult)
export func git(args: [string]) -> string ! {Process} = match exec("git", args) {
  Ok(r) => "OK:${toString(r.stdout)}",
  Err(e) => "ERR:${show(e)}"
}
export func main() -> () ! {IO, FS, Process} = {
  println(git(["log", "--oneline", "-n", "1"]));
  println(git(["status", "--porcelain"]));
  println(git(["diff", "--no-index", "/etc/passwd", "/dev/null"]));
  println(git(["push"]));
  match writeFileResult(".git/config", "[core]\n") {
    Ok(_) => println("WROTE-GIT-CONFIG"),
    Err(e) => println("PROTECTED:${e}")
  }
}
`
	writeAil(t, c.sandbox, "prog.ail", prog)
	stdout, stderr, code := testutil.RunBounded(t, c.sandbox, 60*time.Second, bin, "run", "--policy", pol, "prog.ail")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	// status prints several lines; assert by marker, not by position.
	if !strings.Contains(stdout, "OK:") || !strings.Contains(stdout, "first") {
		t.Errorf("git log must work: %q", stdout)
	}
	if !strings.Contains(stdout, "?? prog.ail") {
		t.Errorf("git status must work (the program file is untracked): %q", stdout)
	}
	if strings.Count(stdout, "ERR:NotAllowed") != 2 || !strings.Contains(stdout, "--no-index") || !strings.Contains(stdout, "git push") {
		t.Errorf("--no-index and push must both be NotAllowed: %q", stdout)
	}
	if !strings.Contains(stdout, "PROTECTED:") || strings.Contains(stdout, "WROTE-GIT-CONFIG") {
		t.Errorf(".git/config must be read-only to the program: %q", stdout)
	}
	if _, err := os.Stat(pwned); err == nil {
		t.Fatal("CONFINEMENT FAILURE: the planted core.fsmonitor command ran")
	}
	if !strings.Contains(stderr, `"process_allow":["git:status","git:diff","git:log"]`) {
		t.Errorf("admission line must carry the grant: %s", stderr)
	}
}
