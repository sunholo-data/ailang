package main

// `ailang coordinator prs` — what happened to the pull requests the agents left?
//
// The coordinator has created PRs since cloud execution existed and never had
// any way to resolve one. Measured 2026-09-14: 24 open `coordinator/*` PRs, 22
// belonging to tasks already rejected or failed, and one belonging to a task an
// operator had APPROVED twenty minutes earlier — approval sets SkipMerge for a
// cloud task (correctly: there is no local worktree) and that read as "do
// nothing", so approved and abandoned looked identical from GitHub.
//
// DRY RUN BY DEFAULT. Merging is the one irreversible action in this path, and
// a tool that acts before you have read what it plans to do is not one you can
// leave running.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

func coordinatorPRs(args []string) error {
	apply := false
	remote, stateDir, repo := "", "", ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--apply":
			apply = true
		case args[i] == "--remote" && i+1 < len(args):
			i++
			remote = args[i]
		case args[i] == "--repo" && i+1 < len(args):
			i++
			repo = args[i]
		case args[i] == "--state-dir" && i+1 < len(args):
			i++
			stateDir = args[i]
		}
	}

	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, remote, stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()

	reg, _, err := resolveInboxRegistry("")
	if err != nil {
		return fmt.Errorf("cannot load the registry — the merge scope check needs it: %w", err)
	}

	ghCfg, err := messaging.LoadGitHubConfig()
	if err != nil {
		return fmt.Errorf("cannot load the GitHub config: %w", err)
	}
	gh := messaging.NewGitHubClient(ghCfg)

	// Open PRs first, tasks second: the PR set is small and bounded, the task
	// set is not.
	if repo == "" {
		repo = defaultCoordinatorRepo(reg)
	}
	prs, err := gh.ListOpenPRsWithPrefix(repo, "coordinator/")
	if err != nil {
		return fmt.Errorf("listing coordinator PRs: %w", err)
	}
	tasks, err := bundle.Store.ListTasks(ctx, &coordinator.TaskFilter{Limit: 500, OrderBy: "created_at", OrderDesc: true})
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}
	byID := map[string]*coordinator.TaskRecord{}
	for _, t := range tasks {
		byID[t.ID] = t
	}

	fmt.Printf("store: %s\n", bundle.Mode)
	if !apply {
		fmt.Printf("%s dry run — nothing will be merged or closed. Add --apply to act.\n", yellow("!"))
	}
	fmt.Println()

	type row struct {
		pr   *messaging.PullRequest
		task *coordinator.TaskRecord
		dec  coordinator.PRDecision
	}
	var rows []row
	for i := range prs {
		pr := &prs[i]
		taskID := strings.TrimPrefix(pr.HeadRefName, "coordinator/")
		t := byID[taskID]
		if t == nil {
			// A PR whose task this store has never heard of. Never guessed at:
			// it may belong to another plane, and closing someone else's work
			// is not recoverable by rerunning this command.
			rows = append(rows, row{pr, nil, coordinator.PRDecision{
				Verdict: coordinator.PRLeave,
				Reason:  "no task " + taskID + " in this store — possibly another plane",
			}})
			continue
		}
		// The card the approval was decided on, so a merge cannot land a change
		// the operator never saw. Unreadable is not the same as absent: say so
		// rather than checking nothing in silence.
		cardFiles, cardErr := approvalCardFiles(ctx, bundle.Store, taskID)
		if cardErr != nil {
			fmt.Fprintf(os.Stderr, "  %s could not read the approval card for %s (%v) — the branch/card check is NOT running for it\n", yellow("!"), taskID, cardErr)
		}
		rows = append(rows, row{pr, t, coordinator.DecidePR(t, reg.GetAgentByID(t.AgentID), pr.Files, cardFiles)})
	}
	if len(rows) == 0 {
		fmt.Printf("%s no open coordinator PRs.\n\n", green("✓"))
		return nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].pr.Number < rows[j].pr.Number })

	counts := map[coordinator.PRVerdict]int{}
	for _, r := range rows {
		counts[r.dec.Verdict]++
		status, id := "-", strings.TrimPrefix(r.pr.HeadRefName, "coordinator/")
		if r.task != nil {
			status, id = string(r.task.Status), r.task.ID
		}
		fmt.Printf("  %-7s #%-5d %-18s %s\n", verdictLabel(r.dec.Verdict), r.pr.Number, status, id)
		fmt.Printf("          %s\n", dim(r.dec.Reason))
	}
	fmt.Printf("\n  %d PR(s): merge %d, close %d, leave %d, refuse %d\n\n",
		len(rows), counts[coordinator.PRMerge], counts[coordinator.PRClose],
		counts[coordinator.PRLeave], counts[coordinator.PRRefuse])

	if !apply {
		fmt.Println("  Re-run with --apply to act on the merge/close rows.")
		return nil
	}

	var failures int
	for _, r := range rows {
		if r.task == nil {
			continue
		}
		target := repoForTask(repo, r.task, reg)
		var actErr error
		switch r.dec.Verdict {
		case coordinator.PRClose:
			actErr = gh.ClosePR(target, r.pr.Number, fmt.Sprintf(
				"Closing: coordinator task `%s` is **%s**, so this branch will never merge.\n\n%s",
				r.task.ID, r.task.Status, r.dec.Reason))
		case coordinator.PRMerge:
			actErr = gh.MergePR(target, r.pr.Number,
				strings.TrimPrefix(r.pr.Title, "[agent] "),
				fmt.Sprintf("Approved coordinator task `%s`.\n\n%s", r.task.ID, r.dec.Reason))
		default:
			continue
		}
		if actErr != nil {
			failures++
			fmt.Fprintf(os.Stderr, "  %s #%d: %v\n", red("FAILED"), r.pr.Number, actErr)
			continue
		}
		fmt.Printf("  %s #%d %s\n", green("✓"), r.pr.Number, r.dec.Verdict)
	}
	if failures > 0 {
		return fmt.Errorf("%d PR action(s) failed", failures)
	}
	return nil
}

// defaultCoordinatorRepo is the workspace the pipeline agents share.
func defaultCoordinatorRepo(reg *coordinator.AgentRegistry) string {
	for _, id := range []string{"design-doc-creator", "sprint-planner", "sprint-executor"} {
		if a := reg.GetAgentByID(id); a != nil {
			if r := a.ResolveRepo(); r != "" {
				return r
			}
			if a.Workspace != "" {
				return a.Workspace
			}
		}
	}
	return ""
}

// reconcileTaskPR resolves the pull request for ONE task, right after its
// approval decision.
//
// The reconciler above is the sweep; this is the hot path, so the decision
// lands with the decision rather than waiting for someone to run a command.
// It is best-effort ON PURPOSE and says so: the approval itself already
// succeeded and is durable, so a GitHub hiccup must never turn a recorded
// decision into an error. What it must not do is stay quiet — an approved task
// whose PR is still open is the exact state this whole path exists to end.
func reconcileTaskPR(taskID string, store coordinator.Store, reg *coordinator.AgentRegistry) {
	if store == nil || reg == nil {
		return
	}
	ctx := context.Background()
	task, err := store.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return
	}
	agent := reg.GetAgentByID(task.AgentID)
	repo := repoForTask("", task, reg)
	if repo == "" {
		return
	}
	ghCfg, err := messaging.LoadGitHubConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s PR not reconciled (no GitHub config): %v\n", yellow("!"), err)
		return
	}
	gh := messaging.NewGitHubClient(ghCfg)
	pr, err := gh.FindPRForBranch(repo, coordinator.BranchForTask(taskID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s could not look up the PR for %s: %v\n", yellow("!"), taskID, err)
		return
	}
	if pr == nil {
		return // no PR: a no-changes task, or a direct-push agent
	}
	cardFiles, cardErr := approvalCardFiles(ctx, store, taskID)
	if cardErr != nil {
		fmt.Fprintf(os.Stderr, "  %s could not read the approval card for %s (%v) — the branch/card check is NOT running\n", yellow("!"), taskID, cardErr)
	}
	dec := coordinator.DecidePR(task, agent, pr.Files, cardFiles)
	switch dec.Verdict {
	case coordinator.PRClose:
		if err := gh.ClosePR(repo, pr.Number, fmt.Sprintf(
			"Closing: coordinator task `%s` is **%s**.\n\n%s", taskID, task.Status, dec.Reason)); err != nil {
			fmt.Fprintf(os.Stderr, "  %s PR #%d left OPEN: %v\n", yellow("!"), pr.Number, err)
			return
		}
		fmt.Printf("  closed PR #%d (%s)\n", pr.Number, task.Status)
	case coordinator.PRMerge:
		if err := gh.MergePR(repo, pr.Number, strings.TrimPrefix(pr.Title, "[agent] "),
			fmt.Sprintf("Approved coordinator task `%s`.\n\n%s", taskID, dec.Reason)); err != nil {
			fmt.Fprintf(os.Stderr, "  %s PR #%d left OPEN: %v\n", yellow("!"), pr.Number, err)
			return
		}
		fmt.Printf("  merged PR #%d\n", pr.Number)
	case coordinator.PRRefuse:
		// Loud: the operator approved, and something mechanical disagreed.
		fmt.Printf("  %s PR #%d NOT merged: %s\n", yellow("!"), pr.Number, dec.Reason)
	}
}

// repoForTask resolves which repository a task's PR lives in: the flag, then the
// task's own record, then the agent's configured repo. Never a global default —
// acting on the wrong repository is the one mistake this cannot take back.
func repoForTask(flagRepo string, t *coordinator.TaskRecord, reg *coordinator.AgentRegistry) string {
	if flagRepo != "" {
		return flagRepo
	}
	if t != nil && t.GithubRepo != "" {
		return t.GithubRepo
	}
	if t != nil && reg != nil {
		if a := reg.GetAgentByID(t.AgentID); a != nil {
			if r := a.ResolveRepo(); r != "" {
				return r
			}
			return a.Workspace
		}
	}
	return ""
}

func verdictLabel(v coordinator.PRVerdict) string {
	switch v {
	case coordinator.PRMerge:
		return green("MERGE")
	case coordinator.PRClose:
		return yellow("CLOSE")
	case coordinator.PRRefuse:
		return red("REFUSE")
	default:
		return dim("leave")
	}
}

// approvalCardFiles is the file list the approval card showed, read back from
// the stored decision.
//
// A task with no approval row at all is not an error — a skip_approval agent
// never creates one — and reads as "no card to check against".
func approvalCardFiles(ctx context.Context, store coordinator.Store, taskID string) ([]string, error) {
	if store == nil {
		return nil, nil
	}
	apr, err := store.GetApprovalRequestByTaskAnyStatus(ctx, taskID)
	if err != nil || apr == nil {
		if err != nil && strings.Contains(err.Error(), "no approval") {
			return nil, nil
		}
		return nil, err
	}
	return coordinator.CardFilesFromContext(apr.ContextJSON), nil
}
