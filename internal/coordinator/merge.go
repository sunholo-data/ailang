// Package coordinator provides git merge operations for approved worktrees.
package coordinator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// MergeResult contains the result of a merge operation.
type MergeResult struct {
	Success       bool     `json:"success"`
	MergedFiles   []string `json:"merged_files,omitempty"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
	Error         string   `json:"error,omitempty"`
	CommitHash    string   `json:"commit_hash,omitempty"`
}

// MergeWorktree merges changes from a worktree into the main branch.
// It performs the following steps:
// 1. Get the list of changed files
// 2. Attempt to merge the worktree branch into main
// 3. If conflicts occur, report them without forcing
// 4. On success, return the merge commit hash
func MergeWorktree(ctx context.Context, worktreePath, mainBranch string) (*MergeResult, error) {
	result := &MergeResult{}

	// Validate worktree exists
	if worktreePath == "" {
		return nil, fmt.Errorf("worktree path is empty")
	}

	// Get the worktree branch name
	branchName, err := getWorktreeBranch(ctx, worktreePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get worktree branch: %v", err)
		return result, nil
	}

	// Get list of changed files in worktree (compare against target merge branch)
	changedFiles, err := getChangedFiles(ctx, worktreePath, mainBranch)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get changed files: %v", err)
		return result, nil
	}
	result.MergedFiles = changedFiles

	// Get the main repo path (parent of worktrees directory)
	mainRepoPath := getMainRepoPath(worktreePath)
	if mainRepoPath == "" {
		result.Error = "failed to determine main repo path"
		return result, nil
	}

	// Checkout main branch
	if err := gitCheckout(ctx, mainRepoPath, mainBranch); err != nil {
		result.Error = fmt.Sprintf("failed to checkout %s: %v", mainBranch, err)
		return result, nil
	}

	// Attempt merge
	mergeOutput, err := gitMerge(ctx, mainRepoPath, branchName)
	if err != nil {
		// Check if it's a conflict
		if strings.Contains(mergeOutput, "CONFLICT") || strings.Contains(mergeOutput, "Automatic merge failed") {
			result.ConflictFiles = parseConflictFiles(mergeOutput)
			result.Error = "merge conflicts detected"
			// Abort the merge to leave repo in clean state
			_ = gitMergeAbort(ctx, mainRepoPath)
			return result, nil
		}
		result.Error = fmt.Sprintf("merge failed: %v\n%s", err, mergeOutput)
		return result, nil
	}

	// Get the merge commit hash
	commitHash, err := getHeadCommit(ctx, mainRepoPath)
	if err != nil {
		result.Error = fmt.Sprintf("merge succeeded but failed to get commit hash: %v", err)
		result.Success = true // Merge did succeed
		return result, nil
	}

	result.Success = true
	result.CommitHash = commitHash
	return result, nil
}

// GetWorktreeDiff returns the git diff for a worktree against its base branch.
// baseBranch is the branch the worktree was created from (e.g., "dev").
// If baseBranch is empty, it queries git for the remote's default branch.
func GetWorktreeDiff(ctx context.Context, worktreePath, baseBranch string) (string, error) {
	if worktreePath == "" {
		return "", fmt.Errorf("worktree path is empty")
	}

	// If base branch not provided, query git for the default branch
	if baseBranch == "" {
		baseBranch = GetDefaultBranch(worktreePath)
	}

	// Compare against the base branch to show all changes made by this task
	cmd := exec.CommandContext(ctx, "git", "diff", baseBranch+"..HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff against %s failed: %w\n%s", baseBranch, err, output)
	}
	return string(output), nil
}

// getWorktreeBranch returns the current branch name in the worktree.
func getWorktreeBranch(ctx context.Context, worktreePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getChangedFiles returns a list of files changed in the worktree.
// baseBranch is the branch the worktree was created from (e.g., "dev").
// If baseBranch is empty, it queries git for the remote's default branch.
func getChangedFiles(ctx context.Context, worktreePath, baseBranch string) ([]string, error) {
	// If base branch not provided, query git for the default branch
	if baseBranch == "" {
		baseBranch = GetDefaultBranch(worktreePath)
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseBranch+"..HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff against %s failed: %w\n%s", baseBranch, err, output)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// getMainRepoPath determines the main repo path from a worktree path.
// Worktrees are typically at ~/.ailang/state/worktrees/coordinator/<task-id>/
// and the main repo is the current working directory.
func getMainRepoPath(worktreePath string) string {
	// Return empty string immediately if worktreePath is empty to prevent
	// running git commands in the current directory
	if worktreePath == "" {
		return ""
	}

	// Get the git directory of the worktree
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	gitDir := strings.TrimSpace(string(output))
	// Worktree git dirs look like: /main/repo/.git/worktrees/<name>
	// We need to extract /main/repo from that
	if strings.Contains(gitDir, ".git/worktrees") {
		// Go up from .git/worktrees/<name> to the repo root
		parts := strings.Split(gitDir, ".git/worktrees")
		if len(parts) > 0 {
			return filepath.Clean(parts[0])
		}
	}
	return ""
}

// gitCheckout checks out a branch in the given repo.
func gitCheckout(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

// gitMerge merges a branch into the current branch.
func gitMerge(ctx context.Context, repoPath, branch string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "merge", "--no-edit", branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// gitMergeAbort aborts an in-progress merge.
func gitMergeAbort(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "merge", "--abort")
	cmd.Dir = repoPath
	_, err := cmd.CombinedOutput()
	return err
}

// getHeadCommit returns the HEAD commit hash.
func getHeadCommit(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// parseConflictFiles extracts conflicting file names from git merge output.
func parseConflictFiles(output string) []string {
	var conflicts []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CONFLICT") {
			// Parse lines like: CONFLICT (content): Merge conflict in path/to/file.go
			if idx := strings.Index(line, "Merge conflict in "); idx >= 0 {
				file := strings.TrimSpace(line[idx+len("Merge conflict in "):])
				conflicts = append(conflicts, file)
			}
		}
	}
	return conflicts
}
