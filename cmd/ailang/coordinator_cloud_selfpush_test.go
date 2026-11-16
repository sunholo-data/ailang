package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// D-FLEET-23 (3), measured at fleet iteration 49; arm A09 of
// .claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh is the shell
// reproduction. The agent pushes its own branch from the job's single-branch
// --depth 1 clone, which creates no refs/remotes/origin/<branch>, and then
// commits again. Step 5b read the tracking ref, concluded "already on the
// remote", and never published the later commit.
func TestAgentSelfPushThenCommitIsStillPublished(t *testing.T) {
	f := newShadowJobFixture(t)
	t.Setenv(config.EnvTaskShadow, "false")
	const branch = "coordinator/task-sp"

	var workDir, selfPushed string
	runExecutorFn = func(ctx context.Context, wd, _, _, _, _, _, _, _ string) (*executor.Result, error) {
		workDir = wd
		for i, name := range []string{"milestone-1", "milestone-2"} {
			if err := os.WriteFile(filepath.Join(wd, "work.txt"), []byte(name+"\n"), 0o644); err != nil {
				return nil, err
			}
			runGit(t, wd, "add", "work.txt")
			runGit(t, wd, "commit", "-m", name)
			if i == 0 {
				runGit(t, wd, "push", "origin", "HEAD:refs/heads/"+branch)
				selfPushed = strings.TrimSpace(runGit(t, wd, "rev-parse", "HEAD"))
			}
		}
		return &executor.Result{NumTurns: 1}, nil
	}

	if _, _, _, err := executeCloudTask(context.Background(), "task-sp", "pkg-x", f.repoURL, "dev", "do it", "pi", "", "m", "1m"); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	// The precondition that made the old check lie: no tracking ref.
	if err := exec.Command("git", "-C", workDir, "rev-parse", "--verify", "-q", "refs/remotes/origin/"+branch).Run(); err == nil {
		t.Fatal("fixture no longer reproduces: the clone has a tracking ref for the task branch")
	}
	head := strings.TrimSpace(runGit(t, workDir, "rev-parse", "HEAD"))
	remote := strings.TrimSpace(runGit(t, f.remote, "rev-parse", "refs/heads/"+branch))
	if remote == selfPushed {
		t.Fatalf("origin/%s stayed at the agent's own push %s; the later commit %s was never published", branch, selfPushed, head)
	}
	if remote != head {
		t.Fatalf("origin/%s = %s, want HEAD %s", branch, remote, head)
	}
}
