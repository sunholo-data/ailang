package pkg

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func confine(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	return home
}

func assertHomeUntouched(t *testing.T, home string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed under confinement: %v", err)
	}
}

// AC9: one canonical text, pinned. The registry wording is unchanged from #1704.
func TestRequireMessagePinned(t *testing.T) {
	confine(t)
	err := Require(AuthNetwork, "resolving git dependency test/gitlib", GitRemedy)
	want := "confined by AILANG_AGENT_POLICY: resolving git dependency test/gitlib is operator authority — run 'ailang lock' outside the sandbox or provision AILANG_PACKAGE_ROOT"
	if err == nil || err.Error() != want {
		t.Fatalf("Require text:\n got %v\nwant %s", err, want)
	}
	var ce *ConfinedError
	if !errors.As(err, &ce) || ce.Authority != AuthNetwork {
		t.Fatalf("Require must return *ConfinedError carrying the authority, got %T", err)
	}
	if !IsConfinedError(err) {
		t.Fatal("IsConfinedError(Require(...)) = false")
	}
	reg := "confined by AILANG_AGENT_POLICY: writing the registry cache is operator authority — provision AILANG_PACKAGE_ROOT or run 'ailang install' outside the sandbox"
	if err := Require(AuthHomeWrite, "writing the registry cache", ""); err == nil || err.Error() != reg {
		t.Fatalf("default remedy text changed: %v", err)
	}
	t.Setenv(config.EnvAgentPolicy, "")
	if err := Require(AuthNetwork, "x", ""); err != nil {
		t.Fatalf("unconfined Require refused: %v", err)
	}
}

// The transport is the backstop: a request built at a call site that forgot
// the pre-check is still refused, before any byte leaves the process.
func TestHTTPClientTransportGate(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	home := confine(t)
	client := HTTPClient("querying the registry API", 5*time.Second)
	resp, err := client.Get(srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "confined by AILANG_AGENT_POLICY: querying the registry API") {
		t.Fatalf("confined request not refused: %v", err)
	}
	if !IsConfinedError(err) {
		t.Fatalf("transport refusal lost its type through net/http: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("confined client reached the server %d times", n)
	}
	assertHomeUntouched(t, home)

	t.Setenv(config.EnvAgentPolicy, "")
	resp, err = client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unconfined request failed: %v", err)
	}
	resp.Body.Close()
	if n := hits.Load(); n != 1 {
		t.Fatalf("instrument blind: unconfined hits = %d, want 1", n)
	}
}

func TestGitClassification(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		network bool
		readOnl bool
	}{
		{[]string{"clone", "--", "u", "d"}, true, false},
		{[]string{"fetch", "origin"}, true, false},
		{[]string{"ls-remote", "--", "u"}, true, false},
		{[]string{"pull"}, true, false},
		{[]string{"push"}, true, false},
		{[]string{"submodule", "update"}, true, false},
		{[]string{"remote", "update"}, true, false},
		{[]string{"checkout", "v1"}, false, false},
		{[]string{"rev-parse", "HEAD"}, false, true},
		{[]string{"cat-file", "-t", "HEAD"}, false, true},
	} {
		if got := gitNeedsNetwork(tc.args); got != tc.network {
			t.Errorf("gitNeedsNetwork(%v) = %v, want %v", tc.args, got, tc.network)
		}
		if got := gitReadOnly(tc.args); got != tc.readOnl {
			t.Errorf("gitReadOnly(%v) = %v, want %v", tc.args, got, tc.readOnl)
		}
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func TestGitConfinedRefusals(t *testing.T) {
	requireGit(t)
	home := confine(t)
	// Network subcommand: refused wherever it runs.
	if _, err := Git("", "ls-remote", "--", "file:///nonexistent"); !IsConfinedError(err) {
		t.Fatalf("confined ls-remote not refused: %v", err)
	}
	// A write inside ~/.ailang: refused. The dir does not exist, so the only
	// way to get a ConfinedError (not an exec error) is the gate.
	inCache := filepath.Join(home, ".ailang", "cache", "git", "abc")
	if _, err := Git(inCache, "checkout", "v1"); !IsConfinedError(err) {
		t.Fatalf("confined checkout in ~/.ailang not refused: %v", err)
	}
	assertHomeUntouched(t, home)

	// Read-only in ~/.ailang and any local op elsewhere are not authority.
	repo := t.TempDir()
	if _, err := Git(repo, "init", "-q"); err != nil {
		t.Fatalf("confined git init outside HOME refused: %v", err)
	}
	if _, err := Git(repo, "rev-parse", "--git-dir"); err != nil {
		t.Fatalf("rev-parse refused: %v", err)
	}
}

func TestGitHardenedEnv(t *testing.T) {
	cmd := gitCommand("", "status")
	if cmd.Args[1] != "-c" || cmd.Args[2] != "protocol.ext.allow=never" {
		t.Fatalf("missing hardened prefix: %v", cmd.Args)
	}
	found := false
	for _, kv := range cmd.Env {
		if kv == "GIT_TERMINAL_PROMPT=0" {
			found = true
		}
	}
	if !found {
		t.Fatal("GIT_TERMINAL_PROMPT=0 not set")
	}
}

func TestUserDirPurityAndEnsure(t *testing.T) {
	home := confine(t)
	dir, err := UserDir("cache/git")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".ailang", "cache", "git"); dir != want {
		t.Fatalf("UserDir = %s, want %s", dir, want)
	}
	assertHomeUntouched(t, home)
	if _, err := EnsureUserDir("cache/git"); !IsConfinedError(err) {
		t.Fatalf("confined EnsureUserDir not refused: %v", err)
	}
	assertHomeUntouched(t, home)

	t.Setenv(config.EnvAgentPolicy, "")
	got, err := EnsureUserDir("cache/git")
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Fatalf("unconfined EnsureUserDir did not create %s: %v", got, err)
	}
}
