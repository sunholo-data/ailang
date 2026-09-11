package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// verifyDeployKey is the pre-flight smoke test.
//
// Run BEFORE any task work, because a key that cannot push turns into a
// mysterious failure at the end of an expensive run instead of a clear one at
// the start.
//
// Note the write check is `! grep -q denied`, NOT `grep -qv denied`. The latter
// succeeds whenever ANY line lacks the word, and a denial prints several lines
// of which only the first contains it — so the inverted form reports success on
// exactly the failure it is meant to catch. Confirmed against POSIX grep with a
// real denial transcript (2026-09-11); note it behaves differently under ugrep,
// which is why reasoning about it on a dev laptop is not enough.
func verifyDeployKey(ctx context.Context, alias, ownerRepo string) error {
	remote := fmt.Sprintf("git@%s:%s.git", alias, ownerRepo)

	lsCmd := exec.CommandContext(ctx, "git", "ls-remote", remote, "HEAD")
	out, err := lsCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("deploy key cannot READ %s: %v: %s", remote, err, strings.TrimSpace(string(out)))
	}

	// --dry-run so nothing is created; the point is the permission answer.
	pushCmd := exec.CommandContext(ctx, "git", "push", "--dry-run", remote, "HEAD:refs/heads/ailang-deploy-key-smoke")
	pushOut, pushErr := pushCmd.CombinedOutput()
	combined := string(pushOut)
	if strings.Contains(combined, "denied") || strings.Contains(combined, "read-only") {
		return fmt.Errorf("deploy key is READ-ONLY on %s — it needs --allow-write: %s", ownerRepo, strings.TrimSpace(combined))
	}
	if pushErr != nil {
		return fmt.Errorf("deploy key push check failed on %s: %v: %s", ownerRepo, pushErr, strings.TrimSpace(combined))
	}
	fmt.Printf("execute-job: deploy key verified read+write on %s\n", ownerRepo)
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
