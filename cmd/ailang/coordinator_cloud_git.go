package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// assertRemoteMatchesHead proves the branch we published carries the commit we
// built. A push can exit 0 having sent something other than the agent's work —
// measured 2026-09-13, when it sent a branch ref still sitting at the clone
// point while HEAD held two design documents.
//
// This is deliberately an assertion and not a log line: a completion that names
// changed_files on a branch that does not contain them is worse than a failure,
// because everything downstream believes it.
func assertRemoteMatchesHead(ctx context.Context, workDir, branchName string) error {
	headOut, err := gitexec.CommandContext(ctx, "-C", workDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("cannot read local HEAD to verify the push: %w", err)
	}
	head := strings.TrimSpace(string(headOut))

	lsOut, err := gitexec.CommandContext(ctx, "-C", workDir, "ls-remote", "origin", "refs/heads/"+branchName).Output()
	if err != nil {
		return fmt.Errorf("cannot read origin/%s to verify the push: %w", branchName, err)
	}
	fields := strings.Fields(string(lsOut))
	if len(fields) == 0 {
		return fmt.Errorf("push reported success but origin/%s does not exist — the work is at %s locally and was NOT published", branchName, head)
	}
	if remote := fields[0]; remote != head {
		return fmt.Errorf("push reported success but origin/%s is %s, not the HEAD we built (%s) — the work was NOT published to that branch",
			branchName, remote, head)
	}
	fmt.Printf("execute-job: verified origin/%s is at %s\n", branchName, head)
	return nil
}

// configureGitAuthor sets the commit author for this task's checkout.
//
// Both halves or neither: git resolves user.name and user.email independently,
// so setting one leaves the other on the container default and produces a commit
// half-attributed to each identity — worse than either alone, and hard to spot.
func configureGitAuthor(ctx context.Context, workDir string) {
	name, email := config.GitAuthor()
	if name == "" || email == "" {
		if name != "" || email != "" {
			fmt.Fprintf(os.Stderr, "warning: git identity is half-configured (name=%q email=%q) — using the container default for BOTH rather than mixing identities\n", name, email)
		}
		return
	}
	for _, kv := range [][2]string{{"user.name", name}, {"user.email", email}} {
		cmd := gitexec.CommandContext(ctx, "-C", workDir, "config", kv[0], kv[1])
		if err := cmd.Run(); err != nil {
			// Loud: the commit will still be made, but by somebody else, and an
			// author line nobody checked is how a record stops being evidence.
			fmt.Fprintf(os.Stderr, "warning: could not set git %s=%q: %v — commits will carry the container's identity\n", kv[0], kv[1], err)
			return
		}
	}
	fmt.Printf("execute-job: commits authored as %s <%s>\n", name, email)
}
