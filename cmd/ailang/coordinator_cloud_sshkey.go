package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/gitexec"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// A per-agent SSH deploy key, scoped to exactly one repository.
//
// The fleet token (`ailang-github-token`, identity sunholo-voight-kampff) is
// deliberately READ-ONLY on some repos. Measured 2026-09-11 against
// sunholo-data/daneel-memory: `push=false, pull=true`, with sunholo-data/ailang
// (`push=true`) as the control proving the field is meaningful. A `GET` on the
// repo returns 200 either way, so read access is NOT evidence of push access —
// checking the wrong verb is how this was nearly missed.
//
// The design is Daneel's, and the good part is that the bound is MECHANICAL
// rather than policy:
//
//   - the key is reachable only through a host ALIAS (`Host <alias>` →
//     HostName github.com), so a push to `git@github.com:...` has no identity
//     and simply fails;
//   - `IdentitiesOnly yes` stops ssh offering every other key the process can
//     see — without it a fleet key that happens to be present is offered first
//     and succeeds against other repos, which is precisely the boundary this
//     exists to draw;
//   - `known_hosts` is PINNED, so there is no trust-on-first-use window. The
//     pinned key was verified against api.github.com/meta rather than pasted on
//     trust.
//
// A deploy key can PUSH and cannot use the REST/GraphQL API, so an agent using
// one cannot open a pull request or enable auto-merge. That is why this pairs
// with push_branch (direct push) rather than the PR flow.

const (
	// githubEd25519HostKey is GitHub's published ed25519 host key, verified
	// against https://api.github.com/meta on 2026-09-11. Pinned rather than
	// accepted on first use: TOFU against a host you already know is a
	// man-in-the-middle window for no benefit.
	githubEd25519HostKey = "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

	sshKeySecretEnv = "AILANG_SSH_KEY_SECRET" // Secret Manager secret NAME, never the key itself
	sshHostAliasEnv = "AILANG_SSH_HOST_ALIAS"
)

// sshDeployKeyRequested reports whether this task is configured for one.
func sshDeployKeyRequested() bool {
	return strings.TrimSpace(os.Getenv(sshKeySecretEnv)) != ""
}

// configureSSHDeployKey installs the agent's deploy key and returns the host
// alias to clone through.
//
// The secret NAME travels in the environment; the key material is fetched here,
// by the job's own service account. Passing the value as an env override would
// write the private key into the Cloud Run execution spec, where anyone with
// project viewer can read it — a secret is only as scoped as its least careful
// hop.
func configureSSHDeployKey(ctx context.Context, project string) (string, error) {
	// The ssh BINARY, before the key. Debian's git only Recommends
	// openssh-client and every executor image is built with
	// --no-install-recommends, so agent-base shipped without it (found in
	// production 2026-09-12, first real dispatch of a deploy-key agent). git
	// then reports "cannot run ssh: No such file or directory" from inside a
	// clone, which reads as a key or permission problem and sends you to the
	// wrong place entirely. Say what is actually missing.
	if _, err := exec.LookPath("ssh"); err != nil {
		return "", errors.New("no ssh binary in this executor image, so the deploy key cannot be used: " +
			"install openssh-client in docker/Dockerfile.agent-base (git Recommends it, and the images build with --no-install-recommends)")
	}

	secretName := strings.TrimSpace(os.Getenv(sshKeySecretEnv))
	alias := strings.TrimSpace(os.Getenv(sshHostAliasEnv))
	if alias == "" {
		alias = "agent-repo"
	}

	key, err := fetchSecret(ctx, project, secretName)
	if err != nil {
		return "", fmt.Errorf("fetching ssh key secret %q: %w", secretName, err)
	}
	if len(strings.TrimSpace(key)) == 0 {
		return "", fmt.Errorf("ssh key secret %q is empty", secretName)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "/root"
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("creating %s: %w", sshDir, err)
	}

	// 0600: ssh REFUSES a key with looser permissions, and the refusal reads as
	// an auth failure rather than a permission problem.
	keyPath := filepath.Join(sshDir, "agent_deploy_key")
	if !strings.HasSuffix(key, "\n") {
		key += "\n" // ssh rejects a key without a trailing newline
	}
	if err := os.WriteFile(keyPath, []byte(key), 0o600); err != nil {
		return "", fmt.Errorf("writing the deploy key: %w", err)
	}

	knownHosts := filepath.Join(sshDir, "known_hosts")
	if err := os.WriteFile(knownHosts, []byte(githubEd25519HostKey+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("writing known_hosts: %w", err)
	}

	cfg := fmt.Sprintf(`Host %s
  HostName github.com
  User git
  IdentityFile %s
  IdentitiesOnly yes
  StrictHostKeyChecking yes
  UserKnownHostsFile %s
`, alias, keyPath, knownHosts)
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(cfg), 0o600); err != nil {
		return "", fmt.Errorf("writing ssh config: %w", err)
	}

	fmt.Printf("execute-job: ssh deploy key installed for host alias %q (identity is scoped to that alias only)\n", alias)
	return alias, nil
}

// fetchSecret reads the latest version of a Secret Manager secret.
func fetchSecret(ctx context.Context, project, name string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("secret manager client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Accept a bare name or a fully-qualified resource path.
	resource := name
	if !strings.HasPrefix(name, "projects/") {
		resource = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, name)
	}
	resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: resource})
	if err != nil {
		return "", err
	}
	return string(resp.GetPayload().GetData()), nil
}

// sshCloneURL rewrites an HTTPS GitHub URL to go through the host alias.
//
// Rewriting rather than configuring the agent to use a different URL: the repo
// coordinate comes from the registry as owner/repo, and the alias is the ONLY
// place the key is reachable, so this is what makes the scoping bite.
func sshCloneURL(repoURL, alias string) string {
	s := strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")
	for _, p := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(s, p) {
			return fmt.Sprintf("git@%s:%s.git", alias, strings.TrimPrefix(s, p))
		}
	}
	return repoURL // not a GitHub URL; leave it alone rather than mangle it
}

// The deploy-key pre-flight, in two halves, because they cannot run in the same
// place.
//
// Measured 2026-09-12 on task-98301715 (daneel-writer, attempt 3): the whole
// check ran BEFORE the clone and failed with
//
//	deploy key push check failed on sunholo-data/daneel-memory: exit status 128:
//	fatal: not a git repository (or any of the parent directories): .git
//
// `git push --dry-run <remote> HEAD:refs/...` resolves HEAD, which is a LOCAL
// ref, so it needs a repository; `git ls-remote <url>` does not. The key was
// never exercised, and the failure read as a key problem when it was an
// ordering problem — the third time in two days that a broken check has been
// mistaken for the thing it was checking.
//
// So: READ before the clone (cheap, and a read failure would only turn into a
// confusing "git clone failed" a moment later), WRITE straight after it, from
// inside the clone. Both still land before the agent does any work, which is
// the entire point of a pre-flight.

// verifyDeployKeyRead proves the alias resolves and the key can read the repo.
// Safe to call outside a git repository.
func verifyDeployKeyRead(ctx context.Context, alias, ownerRepo string) error {
	remote := fmt.Sprintf("git@%s:%s.git", alias, ownerRepo)
	out, err := gitexec.CommandContext(ctx, "ls-remote", remote, "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("deploy key cannot READ %s: %v: %s", remote, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// verifyDeployKeyWrite proves the key can push. MUST run inside the clone.
//
// --dry-run so nothing is created; the point is the permission answer.
func verifyDeployKeyWrite(ctx context.Context, workDir, ownerRepo string) error {
	cmd := gitexec.CommandContext(ctx, "-C", workDir, "push", "--dry-run", "origin", "HEAD:refs/heads/ailang-deploy-key-smoke")
	out, err := cmd.CombinedOutput()
	if verdictErr := pushProbeVerdict(ownerRepo, string(out), err); verdictErr != nil {
		return verdictErr
	}
	fmt.Printf("execute-job: deploy key verified read+write on %s\n", ownerRepo)
	return nil
}

// pushProbeVerdict reads the outcome of the push probe.
//
// Split out and tested because the two ways to be wrong here are symmetrical:
// calling a broken check a bad key (what happened on task-98301715), and calling
// a bad key a broken check. A probe that could not run says so, in those words,
// and never renders a verdict on the key.
//
// Note the denial test is a substring search, NOT `grep -qv denied`: the
// inverted form succeeds whenever ANY line lacks the word, and a denial prints
// several lines of which only the first contains it.
func pushProbeVerdict(ownerRepo, combined string, err error) error {
	trimmed := strings.TrimSpace(combined)

	// The probe itself could not run. Not a statement about the key.
	if strings.Contains(combined, "not a git repository") {
		return fmt.Errorf("deploy key push check could not RUN for %s (it needs to execute inside the clone, and did not): %s",
			ownerRepo, trimmed)
	}
	// Both spellings. GitHub's own wording is "has been marked as read only"
	// (a SPACE), and matching only the hyphen let a genuinely read-only key fall
	// through to the generic arm — caught by the test, not by reading the code.
	lower := strings.ToLower(combined)
	if strings.Contains(lower, "denied") || strings.Contains(lower, "read only") || strings.Contains(lower, "read-only") {
		return fmt.Errorf("deploy key is READ-ONLY on %s — it needs --allow-write: %s", ownerRepo, trimmed)
	}
	if err != nil {
		return fmt.Errorf("deploy key push check failed on %s: %v: %s", ownerRepo, err, trimmed)
	}
	return nil
}

// gitHubOwnerRepoFromURL extracts owner/repo from a GitHub URL, or "".
func gitHubOwnerRepoFromURL(repoURL string) string {
	s := strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")
	for _, p := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(s, p) {
			return strings.TrimPrefix(s, p)
		}
	}
	// A bare owner/repo coordinate, which is how the registry states it.
	if strings.Count(s, "/") == 1 && !strings.Contains(s, ":") {
		return s
	}
	return ""
}
