package iteration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor/proctree"
)

// MaxCheckOutputBytes bounds the total retained output across a stage. JSON
// escaping can expand it sixfold; the cap reserves room for acceptance metadata.
const MaxCheckOutputBytes = 1024 * 1024

type CheckReceipt struct {
	ID             string   `json:"id"`
	Argv           []string `json:"argv"`
	Cwd            string   `json:"cwd"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	ExitCode       int      `json:"exit_code"`
	Output         string   `json:"output"`
	OutputSHA256   string   `json:"output_sha256"`
	Truncated      bool     `json:"truncated"`
	DurationMillis int64    `json:"duration_millis"`
	Tree           string   `json:"tree"`
	Workspace      string   `json:"workspace"`
}

func verifyChecks(ctx context.Context, repo, root string, checks []Verification, e *Evidence) error {
	repoAbs, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	// Resolve the existing ancestor before making any directory, preventing an
	// apparently external symlink from placing verification inside the author root.
	resolved, err := resolvePlacement(rootAbs)
	if err != nil {
		return err
	}
	if within(repoAbs, resolved) || within(resolved, repoAbs) {
		return fmt.Errorf("verification root must be outside and separate from author workspace")
	}
	if len(checks) == 0 {
		return nil
	}
	if err := os.MkdirAll(resolved, 0700); err != nil {
		return err
	}
	work, err := os.MkdirTemp(resolved, "check-")
	if err != nil {
		return err
	}
	if _, err := Git(ctx, repo, "worktree", "add", "--detach", work, e.OutputRevision); err != nil {
		return err
	}
	// Normal removal refuses dirty content. A changed check workspace is retained
	// with its receipt for inspection, never force-cleaned.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = Git(cleanup, repo, "worktree", "remove", work)
	}()
	remainingOutput := MaxCheckOutputBytes
	for _, check := range checks {
		receipt := CheckReceipt{ID: check.ID, Argv: append([]string(nil), check.Argv...), Cwd: check.Cwd, TimeoutSeconds: check.TimeoutSeconds, ExitCode: -1, Tree: e.OutputTree, Workspace: work}
		cwd, err := filepath.EvalSymlinks(filepath.Join(work, check.Cwd))
		if err != nil {
			return err
		}
		if !within(work, cwd) {
			return fmt.Errorf("verification cwd resolves outside candidate workspace")
		}
		checkCtx, cancel := context.WithTimeout(ctx, time.Duration(check.TimeoutSeconds)*time.Second)
		cmd := exec.CommandContext(checkCtx, check.Argv[0], check.Argv[1:]...)
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(), "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0")
		proctree.Configure(cmd)
		out := &boundedOutput{limit: remainingOutput}
		cmd.Stdout = out
		cmd.Stderr = out
		started := time.Now()
		runErr := cmd.Run()
		proctree.Kill(cmd)
		receipt.DurationMillis = time.Since(started).Milliseconds()
		if cmd.ProcessState != nil {
			receipt.ExitCode = cmd.ProcessState.ExitCode()
		}
		receipt.Output = out.String()
		remainingOutput -= len(receipt.Output)
		receipt.OutputSHA256 = shaBytes([]byte(receipt.Output))
		receipt.Truncated = out.truncated
		e.Checks = append(e.Checks, receipt)
		ctxErr := checkCtx.Err()
		cancel()
		if ctxErr != nil {
			return fmt.Errorf("verification %s deadline/cancellation: %w", check.ID, ctxErr)
		}
		if runErr != nil {
			return fmt.Errorf("verification %s failed (exit %d): %w", check.ID, receipt.ExitCode, runErr)
		}
		head, err := Git(ctx, work, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if strings.TrimSpace(head) != e.OutputRevision {
			return fmt.Errorf("verification changed candidate commit")
		}
		status, err := Git(ctx, work, "status", "--porcelain=v1", "-z", "--untracked-files=all")
		if err != nil {
			return err
		}
		if status != "" {
			return fmt.Errorf("verification changed candidate workspace: %q", status)
		}
	}
	return nil
}
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func resolvePlacement(p string) (string, error) {
	if _, err := os.Lstat(p); err == nil {
		return filepath.EvalSymlinks(p)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(p)
	if parent == p {
		return "", fmt.Errorf("cannot resolve placement %s", p)
	}
	resolved, err := resolvePlacement(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(p)), nil
}
