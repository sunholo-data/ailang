package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// coordinatorCancel cancels tasks that have not started.
//
// There was no way to do this from a terminal. On 2026-09-30 an operator
// asked for a prod task (task-68771ff3) to be dropped because the request had
// been handled elsewhere; the store had a MarkTaskCancelled that nothing
// called, so the only route was a hand-written Firestore PATCH. That PATCH was
// also unsafe: the write was unconditional, and a dispatch attempt failing a
// second later reset the task to pending again. Both store methods are now
// compare-and-set (MarkTaskCancelled from pending only; ResetTaskToPending
// from queued/running only), so a cancel sticks.
//
// Only pending tasks: a running task's executor would still finish and write
// over the cancel, and a pending_approval task is decided with reject.
func coordinatorCancel(args []string) error {
	fs := flag.NewFlagSet("coordinator cancel", flag.ContinueOnError)
	remote := fs.String("remote", "", "plane: local|gcp (default $AILANG_STORAGE_COORDINATOR, then $AILANG_STORAGE)")
	stateDir := fs.String("state-dir", "", "local state dir (local mode only)")
	yes := fs.Bool("yes", false, "skip the confirmation prompt")
	fs.Usage = func() {
		fmt.Println("Usage: ailang coordinator cancel <task-id>... [--remote gcp] [--yes]")
		fmt.Println()
		fmt.Println("Cancel tasks that have not started (status pending). A task awaiting")
		fmt.Println("approval is decided with `reject`; a running task cannot be cancelled.")
		fmt.Println()
		fs.PrintDefaults()
	}

	// Task ids may come before or after the flags.
	var ids []string
	for len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		ids = append(ids, args[0])
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	ids = append(ids, fs.Args()...)
	if len(ids) == 0 {
		fs.Usage()
		return fmt.Errorf("no task id given")
	}

	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, *remote, *stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()
	fmt.Printf("store: %s\n", bundle.Mode)

	failed := 0
	for _, id := range ids {
		if err := cancelOneTask(ctx, bundle.Store, id, *yes); err != nil {
			fmt.Printf("  %s %s: %v\n", red("✗"), id, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d task(s) not cancelled", failed, len(ids))
	}
	return nil
}

// cancelOneTask shows the task, confirms unless yes, and cancels it. The
// status check here is for a readable refusal; the store's compare-and-set is
// what makes it safe against a dispatcher acting at the same moment.
func cancelOneTask(ctx context.Context, store coordinator.Store, id string, yes bool) error {
	task, err := store.GetTask(ctx, id)
	if err != nil {
		return fmt.Errorf("cannot read task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("no such task")
	}
	fmt.Printf("\n  %s  %s  agent=%s  created=%s\n  %s\n", task.ID, task.Status, task.AgentID,
		task.CreatedAt.Format("2006-01-02 15:04"), truncateTitle(task.Title, 100))
	switch task.Status {
	case coordinator.TaskStatusPending:
	case coordinator.TaskStatusPendingApproval:
		return fmt.Errorf("awaiting approval — use `ailang coordinator reject %s`, which also resolves its card", id)
	default:
		return fmt.Errorf("%w: it is %s", coordinator.ErrTaskNotCancellable, task.Status)
	}
	if !yes {
		fmt.Printf("  CANCEL %s? [y/N] ", id)
		var answer string
		_, _ = fmt.Scanln(&answer)
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			return fmt.Errorf("aborted")
		}
	}
	if err := store.MarkTaskCancelled(ctx, id); err != nil {
		return err
	}
	fmt.Printf("  %s cancelled %s\n", green("✓"), id)
	return nil
}

func truncateTitle(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
