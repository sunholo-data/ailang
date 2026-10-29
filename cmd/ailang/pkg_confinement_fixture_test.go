package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// confinementFixture is the #1720 AC1 world: an operator-made git package and
// lock, a loopback recorder standing in for the registry and validator, a
// PATH-first git shim that logs every invocation, and per-run fresh HOME,
// sandbox, outside canary and cache dirs. Nothing reaches the real HOME or
// the network.
type confinementFixture struct {
	bin      string
	base     string
	upstream string // git repo holding test/gitlib (v0.1.0)
	template string // sandbox template: manifest + operator-made ailang.lock
	opHome   string // HOME of the operator lock that made the template lock
	shimDir  string
	recorder *httptest.Server

	mu       sync.Mutex
	requests []string
}

// confinementRun is one execution's world and what the instruments saw.
type confinementRun struct {
	home, sandbox, canary, cache, shimLog string
	stdout, stderr                        string
	exit                                  int
	requests                              []string
	netGit                                []string
}

const gitlibCore = "module test/gitlib/core\nexport pure func answer() -> int = 42\n"

func newConfinementFixture(t *testing.T) *confinementFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the git shim is a POSIX shell script")
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	f := &confinementFixture{bin: buildAilang(t), base: t.TempDir()}
	f.recorder = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.mu.Unlock()
		http.NotFound(w, r)
	}))
	t.Cleanup(f.recorder.Close)

	f.shimDir = filepath.Join(f.base, "shim")
	mustWrite(t, filepath.Join(f.shimDir, "git"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GIT_SHIM_LOG\"\nexec "+realGit+" \"$@\"\n", 0o755)

	f.upstream = filepath.Join(f.base, "upstream")
	mustWrite(t, filepath.Join(f.upstream, "ailang.toml"), "[package]\nname = \"test/gitlib\"\nversion = \"0.1.0\"\nedition = \"1\"\n", 0o644)
	mustWrite(t, filepath.Join(f.upstream, "core.ail"), gitlibCore, 0o644)
	fixtureGit(t, f.upstream, "init", "-q")
	fixtureGit(t, f.upstream, "add", "-A")
	fixtureGit(t, f.upstream, "commit", "-qm", "v1")
	fixtureGit(t, f.upstream, "tag", "v0.1.0")

	f.template = filepath.Join(f.base, "template")
	mustWrite(t, filepath.Join(f.template, "ailang.toml"), fmt.Sprintf("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/gitlib\" = { git = %q, tag = \"v0.1.0\" }\n", "file://"+f.upstream), 0o644)
	mustWrite(t, filepath.Join(f.template, "main.ail"), "module test/app/main\nimport pkg/test/gitlib/core (answer)\nexport pure func main() -> int = answer()\n", 0o644)

	// The operator locks once, unconfined, in its own HOME: the committed
	// lock every confined row starts from (and the clone AC2c provisions).
	f.opHome = filepath.Join(f.base, "ophome")
	op := f.run(t, false, f.template, f.opHome, nil, "lock")
	if op.exit != 0 || len(op.netGit) == 0 {
		t.Fatalf("operator lock failed (exit %d, git %v): %s%s", op.exit, op.netGit, op.stdout, op.stderr)
	}
	copyTree(t, op.sandbox, f.template) // carry the generated ailang.lock into the template
	return f
}

// run executes the binary once in a fresh world. home == "" makes a fresh,
// empty HOME. prep may adjust the fresh sandbox before the run.
func (f *confinementFixture) run(t *testing.T, confined bool, template, home string, extraEnv map[string]string, args ...string) *confinementRun {
	t.Helper()
	r := &confinementRun{home: home}
	dir := t.TempDir()
	if r.home == "" {
		r.home = filepath.Join(dir, "home")
	}
	r.sandbox = filepath.Join(dir, "sandbox")
	r.canary = filepath.Join(dir, "outside")
	r.cache = filepath.Join(dir, "cache")
	r.shimLog = filepath.Join(dir, "git.log")
	for _, d := range []string{r.home, r.cache} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	copyTree(t, template, r.sandbox)
	mustWrite(t, filepath.Join(r.canary, "canary.txt"), "outside the sandbox\n", 0o644)

	env := []string{
		"PATH=" + f.shimDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + r.home, "USERPROFILE=" + r.home,
		"TMPDIR=" + os.TempDir(),
		"AILANG_REGISTRY=" + f.recorder.URL,
		"AILANG_REGISTRY_VALIDATOR=" + f.recorder.URL,
		"AILANG_CACHE_DIR=" + r.cache,
		"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_SHIM_LOG=" + r.shimLog,
		"NO_COLOR=1",
	}
	if confined {
		policy := filepath.Join(dir, "policy.toml")
		mustWrite(t, policy, "allowed_caps = [\"IO\"]\n", 0o644)
		env = append(env, "AILANG_AGENT_POLICY="+policy)
	}
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}

	f.mu.Lock()
	f.requests = nil
	f.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, f.bin, args...)
	cmd.Dir, cmd.Env, cmd.WaitDelay = r.sandbox, env, 5*time.Second
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	r.stdout, r.stderr = out.String(), errb.String()
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &ee):
		r.exit = ee.ExitCode()
	default:
		t.Fatalf("running ailang %v: %v", args, err)
	}
	f.mu.Lock()
	r.requests = append([]string(nil), f.requests...)
	f.mu.Unlock()
	r.netGit = networkGitCalls(t, r.shimLog)
	return r
}

// networkGitCalls returns the logged git invocations that use a transport.
func networkGitCalls(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		for i := 0; i < len(fields); i++ {
			if fields[i] == "-c" {
				i++
				continue
			}
			switch fields[i] {
			case "clone", "fetch", "ls-remote", "pull", "submodule":
				out = append(out, line)
			}
			break
		}
	}
	return out
}

// treeSnapshot is path → mode+size+sha256 for every entry under root.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			snap[rel] = "dir " + info.Mode().String()
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		snap[rel] = fmt.Sprintf("%s %d %s", info.Mode(), info.Size(), hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return snap
}

// snapshotDiff lists paths that differ, ignoring the allowed ledger.
func snapshotDiff(before, after map[string]string, ledger []string) []string {
	allowed := map[string]bool{}
	for _, p := range ledger {
		allowed[p] = true
	}
	var diff []string
	for p, v := range after {
		if before[p] != v && !allowed[p] {
			diff = append(diff, "+"+p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok && !allowed[p] {
			diff = append(diff, "-"+p)
		}
	}
	sort.Strings(diff)
	return diff
}

func fixtureGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func mustWrite(t *testing.T, path, data string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
}

// copyTree copies regular files and dirs from src into dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}
