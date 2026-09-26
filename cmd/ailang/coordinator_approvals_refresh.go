package main

// Repairing a pending approval card that no longer describes its branch.
//
// The card and the branch came apart whenever a task re-ran while its approval
// was still pending: CreateApprovalIfAbsent said "already exists" and the row
// kept the FIRST execution's diff. Finalisation now refreshes a pending card
// (CollisionStaleCard), and DecidePR refuses a merge whose card disagrees with
// the branch — but neither of those repairs a row that is already stale, and on
// 2026-09-15 that was 11 approvals out of 11.
//
// So this exists: check every pending card against the branch, and rewrite the
// stale ones FROM the branch. The branch is the authority — it is what a merge
// would land — and the rewrite records that it happened, because silently
// editing the evidence behind a decision is the thing this whole path is
// supposed to prevent.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// refreshApprovalCards reports, and optionally repairs, pending approvals whose
// recorded diff disagrees with the branch.
func refreshApprovalCards(ctx context.Context, bundle *coordinatorStoreBundle, reg *coordinator.AgentRegistry, apply bool) error {
	if reg == nil {
		return fmt.Errorf("no agent registry loaded: the repo a task's branch lives in comes from the agent, so nothing can be checked")
	}
	pending, err := bundle.Store.ListPendingApprovals(ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending approvals: %w", err)
	}
	ghCfg, err := messaging.LoadGitHubConfig()
	if err != nil {
		return fmt.Errorf("no GitHub config, so the branch cannot be read: %w", err)
	}
	gh := messaging.NewGitHubClient(ghCfg)

	if !apply {
		fmt.Printf("%s dry run — no card will be rewritten. Add --apply to repair.\n\n", yellow("!"))
	}
	checked, stale, repaired, skipped := 0, 0, 0, 0
	for _, apr := range pending {
		task, tErr := bundle.Store.GetTask(ctx, apr.TaskID)
		if tErr != nil || task == nil {
			skipped++
			continue
		}
		repo := repoForTask("", task, reg)
		if repo == "" {
			skipped++
			continue
		}
		pr, pErr := gh.FindPRForBranch(repo, coordinator.BranchForTask(apr.TaskID))
		if pErr != nil {
			fmt.Fprintf(os.Stderr, "  %s %s: could not read the branch: %v\n", yellow("!"), apr.TaskID, pErr)
			skipped++
			continue
		}
		if pr == nil || len(pr.Files) == 0 {
			// No PR, or a PR GitHub reports no files for. Either way there is no
			// authority to rewrite from, and inventing one is the original bug.
			skipped++
			continue
		}
		checked++
		cardFiles := coordinator.CardFilesFromContext(apr.ContextJSON)
		missing, extra := coordinator.CardBranchDisagreement(cardFiles, pr.Files)
		if len(cardFiles) > 0 && len(missing) == 0 && len(extra) == 0 {
			continue
		}
		stale++
		fmt.Printf("  %s %s (PR #%d)\n", yellow("STALE"), apr.TaskID, pr.Number)
		fmt.Printf("      card  : %s\n", fileListOrNone(cardFiles))
		fmt.Printf("      branch: %s\n", fileListOrNone(pr.Files))
		if !apply {
			continue
		}
		newCtx, cErr := rewriteCardFromBranch(apr.ContextJSON, pr.Files, pr.Number, bundle.Mode)
		if cErr != nil {
			fmt.Fprintf(os.Stderr, "      %s not repaired: %v\n", yellow("!"), cErr)
			continue
		}
		ok, rErr := bundle.Store.RefreshPendingApproval(ctx, apr.TaskID, apr.Description, newCtx)
		if rErr != nil {
			fmt.Fprintf(os.Stderr, "      %s not repaired: %v\n", yellow("!"), rErr)
			continue
		}
		if !ok {
			// It left 'pending' between the list and the write — someone decided
			// it. Their decision stands; never overwrite it.
			fmt.Printf("      %s resolved while this ran — left alone\n", yellow("!"))
			continue
		}
		repaired++
		fmt.Printf("      %s card now describes the branch\n", green("✓"))
	}

	fmt.Printf("\n  %d pending approval(s), %d checked against a branch, %d stale", len(pending), checked, stale)
	if apply {
		fmt.Printf(", %d repaired", repaired)
	}
	fmt.Printf(" (%d skipped: no task, no repo, or no PR)\n", skipped)
	if stale > 0 && !apply {
		fmt.Println("  Re-run with --apply to rewrite the stale cards from their branches.")
	}
	return nil
}

// rewriteCardFromBranch replaces the recorded diff with the branch's file list.
//
// The stat is re-derived rather than invented: GitHub's PR listing gives the
// files, not a diffstat, and writing a fabricated "N insertions" would be a
// worse lie than the stale card. The provenance goes in the card so a reader
// can see that this evidence was repaired and from where.
func rewriteCardFromBranch(contextJSON string, files []string, prNumber int, plane string) (string, error) {
	obj := map[string]interface{}{}
	if strings.TrimSpace(contextJSON) != "" {
		if err := json.Unmarshal([]byte(contextJSON), &obj); err != nil {
			return "", fmt.Errorf("existing card is not readable JSON: %w", err)
		}
	}
	stat := fmt.Sprintf("%d file(s) on the branch (re-derived from PR #%d; line counts unavailable from the PR listing)", len(files), prNumber)
	obj["changed_files"] = files
	obj["diff_stat"] = stat
	// The patch belonged to the superseded run. Leaving it would put the old
	// change back on the card the moment anyone asked for --full.
	delete(obj, "diff")
	obj["work_id"] = coordinator.WorkIDForApproval(files, stat)
	obj["card_source"] = fmt.Sprintf("re-derived from PR #%d on %s at %s", prNumber, plane, time.Now().UTC().Format(time.RFC3339))
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func fileListOrNone(files []string) string {
	if len(files) == 0 {
		return "(none recorded)"
	}
	return strings.Join(files, ", ")
}
