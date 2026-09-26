package main

// `ailang coordinator prs --landed` — pending cards whose PR already merged.
//
// `coordinator prs` resolves PRs a decision left behind (card → PR). This is the
// reverse: decisions a merge already made (PR → card). Nothing ran it, so a card
// whose PR merged first — by hand, by a repo's own docs-only auto-merge, by
// another session — stayed pending forever. Measured 2026-09-23 on prod: 57 of
// 77 pending cards had a merged coordinator PR.
//
// DRY RUN BY DEFAULT, like its sibling. --apply resolves each landed card as
// approved (ApprovedBy names the merge, not an operator) and completes the task.
// It does NOT fire the card's handoffs unless --fire-handoffs is given: the
// first backlog this ran against was entirely superseded — docs landed and
// later reverted, or already planned — and dispatching sprint-planner on them
// would have started sprints nobody wanted. Firing is a decision; recording a
// merge is a fact.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

type landedRow struct {
	task     *coordinator.TaskRecord
	pr       *coordinator.LandedPR
	ok       bool
	reason   string
	handoffs []string
}

func coordinatorPRsLanded(ctx context.Context, bundle *coordinatorStoreBundle, reg *coordinator.AgentRegistry, registrySource, repoFlag string, apply, fireHandoffs bool) error {
	if fireHandoffs {
		if auth := resolveApprovalAuthority(); !auth.Granted {
			return fmt.Errorf("--fire-handoffs dispatches agents, and that is an approval decision: %s", auth.Reason)
		}
	}

	tasks, err := bundle.Store.ListTasks(ctx, &coordinator.TaskFilter{
		Status: []coordinator.TaskStatus{coordinator.TaskStatusPendingApproval}, Limit: 500,
	})
	if err != nil {
		return fmt.Errorf("listing pending tasks: %w", err)
	}

	ghCfg, err := messaging.LoadGitHubConfig()
	if err != nil {
		return fmt.Errorf("cannot load the GitHub config: %w", err)
	}
	gh := messaging.NewGitHubClient(ghCfg)

	// One listing per repository, not one lookup per task.
	merged := map[string]map[string]*coordinator.LandedPR{} // repo -> branch -> PR
	var rows []landedRow
	for _, t := range tasks {
		repo := repoForTask(repoFlag, t, reg)
		if repo == "" {
			rows = append(rows, landedRow{task: t, reason: "no repository resolvable for this task"})
			continue
		}
		byBranch, seen := merged[repo]
		if !seen {
			prs, err := gh.ListMergedPRsWithPrefix(repo, "coordinator/")
			if err != nil {
				// Loud: a repo we cannot read is not a repo with nothing merged.
				fmt.Fprintf(os.Stderr, "  %s cannot list merged PRs in %s: %v\n", yellow("!"), repo, err)
			}
			byBranch = map[string]*coordinator.LandedPR{}
			for _, p := range prs {
				byBranch[p.HeadRefName] = &coordinator.LandedPR{
					Number: p.Number, State: p.State, HeadRefName: p.HeadRefName,
					MergedAt: p.MergedAt, MergedBy: p.MergedBy,
				}
			}
			merged[repo] = byBranch
		}
		pr := byBranch[coordinator.BranchForTask(t.ID)]
		if pr == nil {
			continue // not landed: the ordinary pending card
		}
		apr, err := bundle.Store.GetApprovalRequestByTaskAnyStatus(ctx, t.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s cannot read the approval for %s: %v\n", yellow("!"), t.ID, err)
			continue
		}
		ok, reason := coordinator.DecideLandedCard(t, apr, pr)
		rows = append(rows, landedRow{t, pr, ok, reason, coordinator.ApprovalHandoffTargets(reg, t)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].task.CreatedAt.Before(rows[j].task.CreatedAt) })

	fmt.Printf("store: %s\n", bundle.Mode)
	if registrySource != "" {
		fmt.Printf("handoff topology from: %s\n", registrySource)
	}
	if !apply {
		fmt.Printf("%s dry run — no card will be resolved. Add --apply to act.\n", yellow("!"))
	}
	fmt.Println()

	landed, withHandoffs := 0, 0
	for _, r := range rows {
		label := dim("skip  ")
		if r.ok {
			label = green("LANDED")
			landed++
		}
		fmt.Printf("  %s %-14s %-22s %s\n", label, r.task.ID, r.task.AgentID, dim(r.reason))
		if r.ok && len(r.handoffs) > 0 {
			withHandoffs++
			verb := "would NOT fire"
			if fireHandoffs {
				verb = "WILL fire"
			}
			fmt.Printf("         %s → %s\n", verb, strings.Join(r.handoffs, ", "))
		}
	}
	fmt.Printf("\n  %d of %d pending card(s) already landed; %d of those carry a handoff.\n\n", landed, len(tasks), withHandoffs)

	if !apply {
		if landed > 0 {
			fmt.Println("  Re-run with --apply to resolve them (add --fire-handoffs to also dispatch).")
		}
		return nil
	}

	var failures int
	for _, r := range rows {
		if !r.ok {
			continue
		}
		res, err := coordinator.ProcessApprovalRequest(ctx, &coordinator.ApprovalParams{
			TaskID:        r.task.ID,
			Action:        "approve",
			ApprovedBy:    coordinator.LandedApprover(r.pr),
			Channel:       "pr-merged",
			Store:         bundle.Store,
			MsgStore:      bundle.MsgStore,
			ObsBackend:    bundle.ObsBackend,
			AgentRegistry: reg,
			SkipMerge:     true, // the merge already happened, on GitHub
			SkipHandoffs:  !fireHandoffs,
		})
		if err != nil {
			failures++
			fmt.Fprintf(os.Stderr, "  %s %s: %v\n", red("FAILED"), r.task.ID, err)
			continue
		}
		fmt.Printf("  %s %s  %s\n", green("✓"), r.task.ID, res.Message)
	}
	if failures > 0 {
		return fmt.Errorf("%d card(s) could not be resolved", failures)
	}
	return nil
}
