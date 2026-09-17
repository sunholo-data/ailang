package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// ScratchDir is the one directory an agent may use for probe/scratch programs
// that the wrapper never commits, in any repo. The ailang_only lane tells the
// model to put probes there: it has no delete tool, and the first package run
// (ailang-packages#64, 2026-09-16) shipped its scratch/ package, lock and two
// probes inside an otherwise correct PR. artifact_patterns cannot do this —
// the wrapper reads them only to decide auto-mergeability, never to bound the
// commit (branchIsAutoMergeable) — so the exclusion is a pathspec here.
// It is coordinator.ScratchDirName so the commit pathspec and the changed-file
// list cannot drift: artifact discovery drops the same directory, which is what
// keeps the approval card describing the branch.
const ScratchDir = coordinator.ScratchDirName

// stageForCommit stages every change under workDir except ScratchDir and
// reports whether anything is actually staged. A tree whose only changes are
// scratch files stages nothing, so the caller must not attempt a commit.
func stageForCommit(ctx context.Context, workDir string) (bool, error) {
	status, err := gitexec.CommandContext(ctx, "-C", workDir, "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	if strings.TrimSpace(string(status)) == "" {
		return false, nil
	}
	add := gitexec.CommandContext(ctx, "-C", workDir, "add", "-A", "--", ".",
		":(exclude)"+ScratchDir, ":(exclude)**/"+ScratchDir+"/**")
	if err := add.Run(); err != nil {
		return false, fmt.Errorf("git add failed: %w", err)
	}
	// exit 1 = differences staged; 0 = nothing staged (only scratch changed)
	if err := gitexec.CommandContext(ctx, "-C", workDir, "diff", "--cached", "--quiet").Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return true, nil
		}
		return false, fmt.Errorf("git diff --cached failed: %w", err)
	}
	return false, nil
}
