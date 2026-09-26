// Package gitutil contains small helpers for interrogating Git repositories.
package gitutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubOwnerRepo parses the origin remote in workDir and returns its GitHub
// owner and repository. Non-GitHub remotes return empty strings and no error.
func GitHubOwnerRepo(ctx context.Context, workDir string) (string, string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", workDir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote: %w", err)
	}
	url := strings.TrimSuffix(strings.TrimSpace(string(out)), ".git")
	for _, prefix := range []string{"https://github.com/", "git@github.com:"} {
		if strings.HasPrefix(url, prefix) {
			parts := strings.SplitN(strings.TrimPrefix(url, prefix), "/", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
		}
	}
	return "", "", nil
}
