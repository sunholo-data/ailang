package managed_agents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

const testKey = "test-key-not-real"

// devAPIStub routes Developer API requests by method + path and records them.
type devAPIStub struct {
	mu       sync.Mutex
	sse      []byte
	files    map[string][]byte // sandbox path → content; absent = 404
	requests []*http.Request
	bodies   [][]byte
}

func (s *devAPIStub) Do(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b []byte
	if req.Body != nil {
		b, _ = io.ReadAll(req.Body)
	}
	s.requests = append(s.requests, req)
	s.bodies = append(s.bodies, b)
	resp := func(code int, body []byte) *http.Response {
		return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}}
	}
	p := req.URL.Path
	switch {
	case req.Method == http.MethodPost && strings.HasSuffix(p, "/interactions"):
		return resp(200, s.sse), nil
	case req.Method == http.MethodDelete:
		return resp(200, []byte("{}")), nil
	case req.Method == http.MethodGet && strings.Contains(p, "/files/"):
		path := "/" + p[strings.Index(p, "/files/")+len("/files/"):]
		if c, ok := s.files[path]; ok {
			return resp(200, c), nil
		}
		return resp(404, []byte(`{"error":{"message":"not found"}}`)), nil
	}
	return resp(400, []byte(`{"error":{"message":"unexpected request"}}`)), nil
}

func newDevAPITestExecutor(t *testing.T, stub *devAPIStub) *Executor {
	t.Helper()
	e, err := NewDevAPI(&executor.Config{})
	if err != nil {
		t.Fatal(err)
	}
	e.httpClient = stub
	e.apiKey = func() string { return testKey }
	return e
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepoWithRemote builds a repo whose origin is a GitHub https URL and whose
// origin/dev points at the first commit, then adds an UNPUSHED commit, an
// uncommitted edit and an untracked file on top.
func newRepoWithRemote(t *testing.T) (dir, base string) {
	t.Helper()
	dir = t.TempDir()
	git(t, dir, "init", "-q", "-b", "dev")
	// The assertions compare exact bytes. A Windows runner's global
	// core.autocrlf=true would check files out (and `git apply` them) with
	// CRLF, so pin the repo to LF regardless of the host's git config.
	git(t, dir, "config", "core.autocrlf", "false")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644)
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")
	base = git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "remote", "add", "origin", "https://github.com/example/repo.git")
	git(t, dir, "update-ref", "refs/remotes/origin/dev", base)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("unpushed\n"), 0o644)
	git(t, dir, "add", "b.txt")
	git(t, dir, "commit", "-q", "-m", "unpushed")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\nedited\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("untracked\n"), 0o644)
	return dir, base
}

func TestDevAPI_Registration(t *testing.T) {
	ex, err := executor.GlobalFactory().GetExecutor(DevAPIExecutorName)
	if err != nil {
		t.Fatalf("GetExecutor(%s): %v", DevAPIExecutorName, err)
	}
	if ex.Name() != DevAPIExecutorName {
		t.Errorf("Name()=%q", ex.Name())
	}
	for _, c := range ex.Capabilities() {
		if c == executor.CapRemoteSandbox {
			t.Error("gemini_agents must NOT advertise CapRemoteSandbox — its edits land locally")
		}
	}
}

func TestDevAPI_HealthCheckNamesMissingKey(t *testing.T) {
	e, _ := NewDevAPI(&executor.Config{})
	e.apiKey = func() string { return "" }
	err := e.HealthCheck(context.Background())
	if err == nil || !strings.Contains(err.Error(), DevAPIKeyEnv) {
		t.Fatalf("want error naming %s, got %v", DevAPIKeyEnv, err)
	}
}

func TestSplitAgentModel(t *testing.T) {
	for in, want := range map[string][2]string{
		"antigravity-preview-09-2026":                {"antigravity-preview-09-2026", ""},
		"antigravity-preview-09-2026/gemini-4-argon": {"antigravity-preview-09-2026", "gemini-4-argon"},
		"": {"", ""},
	} {
		a, m := splitAgentModel(in)
		if a != want[0] || m != want[1] {
			t.Errorf("splitAgentModel(%q)=(%q,%q), want %v", in, a, m, want)
		}
	}
}

func TestHTTPSCloneURL(t *testing.T) {
	if got := httpsCloneURL("git@github.com:sunholo-data/ailang.git"); got != "https://github.com/sunholo-data/ailang.git" {
		t.Errorf("ssh form: %q", got)
	}
	if got := httpsCloneURL("https://github.com/a/b\n"); got != "https://github.com/a/b" {
		t.Errorf("https form: %q", got)
	}
}

// The local delta must carry the unpushed commit, the uncommitted edit AND the
// untracked file, and building it must not touch the caller's real index.
func TestPlanWorkspace_LocalDeltaCoversEverythingAndSparesIndex(t *testing.T) {
	dir, base := newRepoWithRemote(t)
	indexBefore := git(t, dir, "status", "--porcelain")

	plan, err := planWorkspace(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.BaseSHA != base {
		t.Errorf("BaseSHA=%s, want merge-base %s", plan.BaseSHA, base)
	}
	if plan.RepoURL != "https://github.com/example/repo.git" {
		t.Errorf("RepoURL=%q", plan.RepoURL)
	}
	for _, want := range []string{"b/b.txt", "+edited", "b/c.txt"} {
		if !strings.Contains(plan.LocalPatch, want) {
			t.Errorf("local patch missing %q:\n%s", want, plan.LocalPatch)
		}
	}
	if after := git(t, dir, "status", "--porcelain"); after != indexBefore {
		t.Errorf("real index changed:\nbefore %q\nafter  %q", indexBefore, after)
	}

	// The patch must reproduce the working tree on a clean checkout of base.
	clone := t.TempDir()
	git(t, clone, "clone", "-q", "-c", "core.autocrlf=false", dir, ".")
	git(t, clone, "checkout", "-q", "--detach", base)
	cmd := exec.Command("git", "apply", "--binary", "-")
	cmd.Dir = clone
	cmd.Stdin = strings.NewReader(plan.LocalPatch)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("local patch does not apply on base: %v: %s", err, out)
	}
	for f, want := range map[string]string{"a.txt": "one\nedited\n", "b.txt": "unpushed\n", "c.txt": "untracked\n"} {
		got, _ := os.ReadFile(filepath.Join(clone, f))
		if string(got) != want {
			t.Errorf("%s=%q, want %q", f, got, want)
		}
	}
}

func TestPlanWorkspace_NoRemoteRefIsNamedError(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644)
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "x")
	git(t, dir, "remote", "add", "origin", "https://github.com/example/repo.git")
	if _, err := planWorkspace(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "fetch first") {
		t.Fatalf("want named no-remote-ref error, got %v", err)
	}
}

func TestPlanWorkspace_NonGitGoesInline(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "solution.ail"), []byte("module x\n"), 0o644)
	plan, err := planWorkspace(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan.RepoURL != "" || len(plan.Inline) != 1 || plan.Inline[0].Target != sandboxRepo+"/solution.ail" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if !strings.Contains(plan.setupScript(), "git init") {
		t.Error("non-git setup must git-init the sandbox copy so the export diff works")
	}
}

// End to end against a stub API, replaying the real 2026-10-13 SSE capture:
// the key goes in x-goog-api-key, the repository source + local patch go up,
// the exported patch is applied locally, and the sandbox is deleted.
func TestDevAPI_ExecuteRoundTrip(t *testing.T) {
	sse, err := os.ReadFile("testdata/sse_devapi_e2e.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir, base := newRepoWithRemote(t)
	out := []byte("diff --git a/new.txt b/new.txt\nnew file mode 100644\nindex 0000000..3e75765\n--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1 @@\n+from agent\n")
	stub := &devAPIStub{sse: sse, files: map[string][]byte{startSHAPath: []byte(base + "\n"), outPatchPath: out}}
	e := newDevAPITestExecutor(t, stub)

	res, err := e.Execute(context.Background(), &executor.Task{
		Directive: "add new.txt", Workspace: dir, Model: "antigravity-preview-09-2026/gemini-4-argon",
	})
	if err != nil || !res.Success {
		t.Fatalf("Execute: err=%v res=%+v", err, res)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "new.txt")); string(got) != "from agent\n" {
		t.Errorf("agent patch not applied locally: new.txt=%q", got)
	}

	post := stub.requests[0]
	if post.Header.Get(devAPIKeyHeader) != testKey || post.Header.Get("Authorization") != "" {
		t.Error("Developer API auth must be x-goog-api-key, never a bearer token")
	}
	var body map[string]any
	if err := json.Unmarshal(stub.bodies[0], &body); err != nil {
		t.Fatal(err)
	}
	if body["agent"] != "antigravity-preview-09-2026" {
		t.Errorf("agent=%v", body["agent"])
	}
	if ac, _ := body["agent_config"].(map[string]any); ac["model"] != "gemini-4-argon" || ac["type"] != "antigravity" {
		t.Errorf("agent_config=%v", body["agent_config"])
	}
	envJSON, _ := json.Marshal(body["environment"])
	for _, want := range []string{`"repository"`, `github.com/example/repo.git`, localPatchPath, `"domain":"*"`} {
		if !strings.Contains(string(envJSON), want) {
			t.Errorf("environment missing %s: %s", want, envJSON)
		}
	}
	if res.ProviderData["managed_agents_requested_model"] != "gemini-4-argon" {
		t.Errorf("requested model not recorded: %v", res.ProviderData)
	}
	last := stub.requests[len(stub.requests)-1]
	if last.Method != http.MethodDelete || !strings.Contains(last.URL.Path, "af55d52646cae2f8ef0b156bcf3f017a") {
		t.Errorf("sandbox not deleted; last request %s %s", last.Method, last.URL.Path)
	}
}

// A completed interaction whose setup never ran is a FAILED task, not a
// success with no edits.
func TestDevAPI_MissingSetupMarkerFails(t *testing.T) {
	sse, _ := os.ReadFile("testdata/sse_devapi_e2e.txt")
	dir, _ := newRepoWithRemote(t)
	e := newDevAPITestExecutor(t, &devAPIStub{sse: sse, files: map[string][]byte{}})
	res, _ := e.Execute(context.Background(), &executor.Task{Directive: "x", Workspace: dir})
	if res.Success || !strings.Contains(res.Error, "setup command never ran") {
		t.Fatalf("want setup failure, got success=%v err=%q", res.Success, res.Error)
	}
}

// A patch that does not apply fails the task and keeps the patch on disk.
func TestDevAPI_UnappliablePatchFailsAndIsKept(t *testing.T) {
	sse, _ := os.ReadFile("testdata/sse_devapi_e2e.txt")
	dir, base := newRepoWithRemote(t)
	bad := []byte("diff --git a/a.txt b/a.txt\n--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-nonexistent line\n+x\n")
	e := newDevAPITestExecutor(t, &devAPIStub{sse: sse, files: map[string][]byte{startSHAPath: []byte(base), outPatchPath: bad}})
	res, _ := e.Execute(context.Background(), &executor.Task{Directive: "x", Workspace: dir})
	if res.Success {
		t.Fatal("unappliable patch must fail the task")
	}
	kept, _ := res.ProviderData["gemini_agents_unapplied_patch"].(string)
	if kept == "" {
		t.Fatal("unapplied patch path not recorded")
	}
	defer os.Remove(kept)
	if b, _ := os.ReadFile(kept); !bytes.Equal(b, bad) {
		t.Error("kept patch differs from the downloaded one")
	}
}
