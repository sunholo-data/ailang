package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// `ailang coordinator approvals` — review and resolve approvals on ANY plane.
//
// An approval releases work: approving a merge_handoff dispatches the next agent
// in the chain. So the card has to carry enough to decide on, and the decision
// has to reach the store that actually holds the approval.
//
// Neither was true before. The approve path was hardcoded to local SQLite, and
// nothing printed which plane it was acting on.

func coordinatorApprovalsCommand(args []string) error {
	fs := flag.NewFlagSet("coordinator approvals", flag.ExitOnError)
	remote := fs.String("remote", "", "plane: local|gcp (default $AILANG_COORDINATOR_REMOTE, then $AILANG_STORAGE)")
	stateDir := fs.String("state-dir", "", "local state dir (local mode only)")
	full := fs.Bool("full", false, "print the whole diff rather than a summary")
	_ = fs.Parse(args)

	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, *remote, *stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()

	// Always say which plane. "approved" against the wrong store looks exactly
	// like success.
	fmt.Printf("store: %s\n\n", bundle.Mode)

	pending, err := bundle.Store.ListPendingApprovals(ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending approvals: %w", err)
	}
	if len(pending) == 0 {
		fmt.Println("No pending approvals.")
		return nil
	}

	for _, req := range pending {
		printApprovalCard(req, *full)
	}
	fmt.Printf("\n%d pending. Approve with:\n", len(pending))
	fmt.Printf("  ailang coordinator approve <task-id> --remote %s\n", firstWord(bundle.Mode))
	return nil
}

// printApprovalCard shows what the approval actually releases.
//
// The four things a reviewer needs and could not previously see: what changed,
// what it was asked to do, what approving will START, and whether the diff is
// real or absent. A card that renders a confident "Files (0)" gets approved
// blind — measured twice (#921) — so an absent diff is stated as absent.
func printApprovalCard(req *coordinator.ApprovalRequestRecord, full bool) {
	fmt.Printf("── %s ──\n", req.ID)
	fmt.Printf("  task:    %s\n", req.TaskID)
	fmt.Printf("  type:    %s\n", req.Type)
	fmt.Printf("  created: %s\n", req.CreatedAt.Format(time.RFC3339))
	if req.Description != "" {
		fmt.Printf("  what:    %s\n", req.Description)
	}

	if req.ContextJSON == "" {
		fmt.Println("  ⚠ no context recorded — nothing to review")
		fmt.Println()
		return
	}

	var ctxData struct {
		HandoffTargets  []string `json:"handoff_targets"`
		SourceAgent     string   `json:"source_agent"`
		ChangedFiles    []string `json:"changed_files"`
		DiffStat        string   `json:"diff_stat"`
		Diff            string   `json:"diff"`
		DiffUnavailable string   `json:"diff_unavailable"`
	}
	if err := json.Unmarshal([]byte(req.ContextJSON), &ctxData); err != nil {
		fmt.Printf("  ⚠ context could not be parsed: %v\n\n", err)
		return
	}

	// What approving STARTS. This is the part that makes an approval more than a
	// merge: on a merge_handoff it dispatches the next agent.
	if len(ctxData.HandoffTargets) > 0 {
		fmt.Printf("  ⚠ APPROVING DISPATCHES: %s\n", strings.Join(ctxData.HandoffTargets, ", "))
		if ctxData.SourceAgent != "" {
			fmt.Printf("    (handoff from %s)\n", ctxData.SourceAgent)
		}
	}

	switch {
	case ctxData.DiffUnavailable != "":
		fmt.Printf("  ⚠ DIFF UNAVAILABLE: %s\n", ctxData.DiffUnavailable)
		fmt.Println("    Approving this means approving a change you cannot see.")
	case len(ctxData.ChangedFiles) == 0:
		fmt.Println("  ⚠ no changed files recorded")
	default:
		fmt.Printf("  files:   %d\n", len(ctxData.ChangedFiles))
		for i, f := range ctxData.ChangedFiles {
			if i == 12 && !full {
				fmt.Printf("           … and %d more (--full)\n", len(ctxData.ChangedFiles)-12)
				break
			}
			fmt.Printf("           %s\n", f)
		}
		if ctxData.DiffStat != "" {
			fmt.Printf("  stat:    %s\n", strings.TrimSpace(ctxData.DiffStat))
		}
	}

	if full && ctxData.Diff != "" {
		fmt.Println("\n--- diff ---")
		fmt.Println(ctxData.Diff)
	}
	fmt.Println()
}

// coordinatorResolveRemote approves or rejects on the resolved plane, using the
// SAME processor the dashboard uses — so handoffs fire immediately rather than
// waiting for the coordinator's next startup sweep.
func coordinatorResolveRemote(args []string, action string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator %s <task-id|approval-id> --remote gcp", action)
	}
	taskID := args[0]

	fs := flag.NewFlagSet("coordinator "+action, flag.ExitOnError)
	remote := fs.String("remote", "", "plane: local|gcp")
	stateDir := fs.String("state-dir", "", "local state dir (local mode only)")
	by := fs.String("by", "", "who is approving (defaults to $USER)")
	feedback := fs.String("feedback", "", "feedback text (rejections)")
	yes := fs.Bool("yes", false, "skip the confirmation prompt")
	_ = fs.Parse(args[1:])

	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, *remote, *stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()

	fmt.Printf("store: %s\n", bundle.Mode)

	// Show the card before acting on it. An approval that dispatches the next
	// agent should never be a blind yes.
	if req, err := bundle.Store.GetApprovalRequestByTaskAnyStatus(ctx, strings.Replace(taskID, "apr-", "task-", 1)); err == nil && req != nil {
		printApprovalCard(req, false)
		if req.Status != "pending" {
			return fmt.Errorf("approval %s is already %q — nothing to do", req.ID, req.Status)
		}
	}

	if !*yes {
		fmt.Printf("%s %s? [y/N] ", strings.ToUpper(action), taskID)
		var answer string
		_, _ = fmt.Scanln(&answer)
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	who := *by
	if who == "" {
		who = os.Getenv("USER")
	}
	if who == "" {
		who = "cli-user"
	}

	agentRegistry, _ := coordinator.LoadAgentRegistry()
	result, err := coordinator.ProcessApprovalRequest(ctx, &coordinator.ApprovalParams{
		TaskID:        taskID,
		Action:        action,
		ApprovedBy:    who,
		Channel:       "cli",
		Feedback:      *feedback,
		Store:         bundle.Store,
		MsgStore:      bundle.MsgStore,
		ObsBackend:    bundle.ObsBackend,
		AgentRegistry: agentRegistry,
		// A cloud task has no worktree to merge or clean up; the branch is already
		// pushed and its PR already open.
		SkipMerge: bundle.MsgStore != nil,
	})
	if err != nil {
		return fmt.Errorf("failed to %s: %w", action, err)
	}

	fmt.Printf("\n✓ %sd %s (by %s)\n", action, taskID, who)
	if result != nil && result.Message != "" {
		fmt.Printf("  %s\n", result.Message)
	}
	return nil
}

func firstWord(s string) string {
	if i := strings.IndexByte(s, ' '); i > 0 {
		return s[:i]
	}
	return s
}
