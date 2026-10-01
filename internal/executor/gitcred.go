package executor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Per-task git credentials for an agent child (M-EXECUTOR-ENV-HARDENING D2).
//
// In a cloud job the PARENT clones, pushes and opens the PR; the child CLI
// never needs the fleet token. Some agents still push their own branch
// mid-run or fetch a private task repo, so the parent may hand the child a
// credential-store file scoped by URL to the task's repository only:
//
//   - the file lives in a fresh 0700 directory under the OS temp dir, outside
//     the workspace the agent edits and commits, and is 0600;
//   - the child's git reaches it through command-scope config
//     (GIT_CONFIG_COUNT/KEY/VALUE) keyed on the repository URL, so a clone or
//     push of any other repository gets nothing from it;
//   - the caller removes it when the task ends.
//
// It is still the fleet token on disk, readable by a same-UID process: this
// removes it from `printenv`, ~/.gitconfig and every other repo's git
// operations, it does not make it unreadable (that is audit H-6).

// GitCredentialScopes returns the URLs a credential for repoURL should answer
// for — with and without ".git", because git matches credential.<url>.* by
// path prefix at a "/" boundary — or nil when repoURL is not a plain https
// repository URL (an SSH deploy-key clone, a bare coordinate).
func GitCredentialScopes(repoURL string) []string {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil
	}
	p := strings.Trim(u.Path, "/")
	if p == "" || strings.Contains(p, "..") {
		return nil
	}
	base := "https://" + u.Host + "/" + strings.TrimSuffix(p, ".git")
	return []string{base, base + ".git"}
}

// WriteGitCredentialFile writes a git credential-store file holding token for
// the host of scopes[0] and returns its path and a cleanup that removes it.
func WriteGitCredentialFile(scopes []string, token string) (string, func(), error) {
	token = strings.TrimSpace(token)
	if len(scopes) == 0 || token == "" {
		return "", noCleanup, fmt.Errorf("git credential: need a scope and a token")
	}
	u, err := url.Parse(scopes[0])
	if err != nil || u.Host == "" {
		return "", noCleanup, fmt.Errorf("git credential: bad scope %q", scopes[0])
	}
	dir, err := os.MkdirTemp("", "ailang-gitcred-")
	if err != nil {
		return "", noCleanup, fmt.Errorf("git credential: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return "", noCleanup, fmt.Errorf("git credential: %w", err)
	}
	path := filepath.Join(dir, "credentials")
	line := (&url.URL{Scheme: "https", User: url.UserPassword("x-access-token", token), Host: u.Host}).String() + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		cleanup()
		return "", noCleanup, fmt.Errorf("git credential: %w", err)
	}
	return path, cleanup, nil
}

// GitCredentialEnv renders the command-scope git config that points the
// child's git at the credential file for exactly the given scopes.
func GitCredentialEnv(file string, scopes []string) []string {
	if file == "" || len(scopes) == 0 {
		return nil
	}
	env := make([]string, 0, 1+2*len(scopes))
	env = append(env, "GIT_CONFIG_COUNT="+strconv.Itoa(len(scopes)))
	for i, scope := range scopes {
		n := strconv.Itoa(i)
		env = append(env,
			"GIT_CONFIG_KEY_"+n+"=credential."+scope+".helper",
			"GIT_CONFIG_VALUE_"+n+"=store --file="+shellQuote(file),
		)
	}
	return env
}

// shellQuote single-quotes s for the shell git runs a credential helper
// through, so a temp dir with a space (a Windows profile) stays one word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// noCleanup is the cleanup for a call that created nothing.
func noCleanup() {
	// Nothing was written, so there is nothing to remove.
}
