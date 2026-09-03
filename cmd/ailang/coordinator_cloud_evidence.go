package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// Approval evidence produced by the executor (M-COMPLETION-PATH-PARITY M3).
//
// The coordinator cannot compute this. It is a Cloud Run service with no clone
// and no worktree — it passes "" for worktreePath — so `git diff` is unavailable
// to it at any price. The executor is the only component holding a git tree, and
// it already discovers changed FILES here; a diffstat and a patch are separate
// invocations, so this is new work rather than an extension of an existing call.
//
// Why two commit SHAs rather than base..branch: a branch can move or be deleted
// between attempts, so a branch name is not a deterministic input and the same
// approval could render differently on a replay. Two immutable SHAs give the same
// answer every time.
//
// Why it matters that a missing diff stays explicit: #921. An approval card that
// rendered a confident "Files (0)" was approved blind, twice. Absence must read
// as absence.

// gitEvidence is what an executor run leaves behind for the approval card.
type gitEvidence struct {
	ChangedFiles []string
	BaseCommit   string // the clone point — immutable
	HeadCommit   string // captured after a successful push — immutable
	DiffStat     string
	Diff         string // capped; see maxCloudDiffBytes
}

// maxCloudDiffBytes caps the patch carried on the completion payload. A Pub/Sub
// message has a hard size limit, and a diff that overflows it would fail the
// whole completion — losing the run's status to make its card prettier.
const maxCloudDiffBytes = 256 * 1024

// collectGitEvidence gathers the approval evidence for a finished run.
//
// Each piece degrades independently: a failure to read the diffstat must not cost
// us the changed-file list, and none of it may fail the task, because the work is
// already pushed. Every gap is logged rather than swallowed.
func collectGitEvidence(ctx context.Context, workDir, clonePoint string) gitEvidence {
	ev := gitEvidence{
		BaseCommit:   clonePoint,
		ChangedFiles: discoverChangedFilesFromCommit(workDir, clonePoint),
	}

	head, err := gitEvidenceOutput(ctx, workDir, "rev-parse", "HEAD")
	if err != nil {
		// Without a head SHA the diff is not replay-stable, so M3's contract
		// cannot be met for this task and the card must say so.
		fmt.Fprintf(os.Stderr, "execute-job: warning: could not read HEAD, approval diff will be unavailable: %v\n", err)
		return ev
	}
	ev.HeadCommit = head

	if clonePoint == "" {
		fmt.Fprintln(os.Stderr, "execute-job: warning: no clone point recorded, approval diff will be unavailable")
		return ev
	}

	rng := clonePoint + ".." + ev.HeadCommit
	if stat, err := gitEvidenceOutput(ctx, workDir, "diff", "--stat", rng); err == nil {
		ev.DiffStat = stat
	} else {
		fmt.Fprintf(os.Stderr, "execute-job: warning: diff --stat failed: %v\n", err)
	}

	if patch, err := gitEvidenceOutput(ctx, workDir, "diff", rng); err == nil {
		if len(patch) > maxCloudDiffBytes {
			patch = patch[:maxCloudDiffBytes] + "\n\n[diff truncated — see the PR for the full change]"
		}
		ev.Diff = patch
	} else {
		fmt.Fprintf(os.Stderr, "execute-job: warning: diff failed: %v\n", err)
	}

	return ev
}

func gitEvidenceOutput(ctx context.Context, workDir string, args ...string) (string, error) {
	full := append([]string{"-C", workDir}, args...)
	out, err := gitexec.CommandContext(ctx, full...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// diffResultFromCompletion is the coordinator side: read what the executor sent.
// Kept next to the producer so the two halves of the contract stay visible
// together.
func diffResultFromEvidence(ev gitEvidence) (coordinator.DiffResult, error) {
	if ev.HeadCommit == "" || ev.BaseCommit == "" {
		return coordinator.DiffResult{}, coordinator.ErrNoDiffSource
	}
	return coordinator.DiffResult{
		Stat:         ev.DiffStat,
		ChangedFiles: ev.ChangedFiles,
		Patch:        ev.Diff,
	}, nil
}
