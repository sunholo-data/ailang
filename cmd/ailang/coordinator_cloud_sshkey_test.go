package main

import (
	"context"
	"strings"
	"testing"
)

// The alias IS the security bound, so the URL rewrite is the thing that makes
// scoping bite. If it silently leaves a github.com URL in place, the agent
// pushes with whatever other credential is around — the exact fallback the
// deploy key exists to prevent.

func TestSSHCloneURL_RoutesThroughTheAlias(t *testing.T) {
	const alias = "daneel-memory"
	for _, tt := range []struct{ in, want string }{
		{"https://github.com/sunholo-data/daneel-memory.git", "git@daneel-memory:sunholo-data/daneel-memory.git"},
		{"https://github.com/sunholo-data/daneel-memory", "git@daneel-memory:sunholo-data/daneel-memory.git"},
		{"git@github.com:sunholo-data/daneel-memory.git", "git@daneel-memory:sunholo-data/daneel-memory.git"},
		{"ssh://git@github.com/sunholo-data/daneel-memory.git", "git@daneel-memory:sunholo-data/daneel-memory.git"},
		// Not GitHub: leave it rather than mangle it into something that
		// resolves somewhere unintended.
		{"https://gitlab.com/x/y.git", "https://gitlab.com/x/y.git"},
	} {
		if got := sshCloneURL(tt.in, alias); got != tt.want {
			t.Errorf("sshCloneURL(%q)\n got  %q\n want %q", tt.in, got, tt.want)
		}
	}
}

func TestGitHubOwnerRepoFromURL(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"https://github.com/sunholo-data/daneel-memory.git", "sunholo-data/daneel-memory"},
		{"git@github.com:sunholo-data/ailang.git", "sunholo-data/ailang"},
		// The registry states the workspace as a bare coordinate.
		{"sunholo-data/daneel-memory", "sunholo-data/daneel-memory"},
		{"https://gitlab.com/x/y.git", ""},
		{"", ""},
	} {
		if got := gitHubOwnerRepoFromURL(tt.in); got != tt.want {
			t.Errorf("gitHubOwnerRepoFromURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSSHDeployKeyRequested_OnlyWhenNamed(t *testing.T) {
	t.Setenv("AILANG_SSH_KEY_SECRET", "")
	if sshDeployKeyRequested() {
		t.Error("no secret named must mean no deploy key path — the fleet token stays in use")
	}
	t.Setenv("AILANG_SSH_KEY_SECRET", "   ")
	if sshDeployKeyRequested() {
		t.Error("whitespace is not a secret name")
	}
	t.Setenv("AILANG_SSH_KEY_SECRET", "DANEEL_WRITER_SSH_KEY")
	if !sshDeployKeyRequested() {
		t.Error("a named secret must select the deploy key path")
	}
}

func TestPinnedHostKey_IsGitHubs(t *testing.T) {
	// Verified against https://api.github.com/meta on 2026-09-11. Pinned in a
	// test too, so a careless edit to the constant is caught here rather than by
	// a man-in-the-middle.
	const want = "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
	if githubEd25519HostKey != want {
		t.Errorf("pinned host key changed:\n got  %q\n want %q\nVerify against api.github.com/meta before accepting.", githubEd25519HostKey, want)
	}
}

// Found in production 2026-09-12, on the first real dispatch of a deploy-key
// agent: agent-base had no ssh binary. Debian's git only *Recommends*
// openssh-client and every executor image builds with --no-install-recommends,
// so the key, the config and the known_hosts were all written correctly onto an
// image that could not run `ssh` — and git reported it from inside a clone as
// "cannot run ssh: No such file or directory", which reads as a key problem.
//
// The assertion is about ORDER as much as message: the binary check must come
// before the Secret Manager fetch, or the first thing an operator sees is a
// secrets error on an image whose real defect is a missing package.
func TestConfigureSSHDeployKey_NamesTheMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv(sshKeySecretEnv, "some-secret-that-must-never-be-fetched")

	_, err := configureSSHDeployKey(context.Background(), "ailang-multivac")
	if err == nil {
		t.Fatal("with no ssh on PATH this must fail, not proceed to fetch a key it cannot use")
	}
	for _, want := range []string{"no ssh binary", "openssh-client", "Dockerfile.agent-base"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error must name %q so the fix is obvious; got: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("the binary check must run BEFORE the secret fetch; got: %v", err)
	}
}
