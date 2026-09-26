package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

func missionConfirmStoppedCommand(args []string) error {
	return runMissionConfirmStopped(context.Background(), args, os.Stdout, missionIterationDeps{})
}

func runMissionConfirmStopped(ctx context.Context, args []string, out io.Writer, deps missionIterationDeps) error {
	var name, work, attestation string
	var version int64
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("mission confirm-stopped NAME", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&work, "work-item", "", "stable work item ID")
	fs.Int64Var(&version, "version", 0, "expected current status version (required)")
	fs.StringVar(&attestation, "attestation", "", "operator assertion that exact child processes and descendants are stopped; no credentials (required)")
	seen := map[string]bool{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			key := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
			if seen[key] {
				return iterationExit(2, fmt.Errorf("duplicate flag %q", key))
			}
			seen[key] = true
		}
	}
	if err := fs.Parse(args); errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil {
		return iterationExit(2, err)
	}
	if name == "" || work == "" || version < 1 || strings.TrimSpace(attestation) == "" || len(attestation) > 2048 || fs.NArg() != 0 {
		return iterationExit(2, fmt.Errorf("usage: ailang mission confirm-stopped NAME --work-item ID --version N --attestation TEXT (1..2048 bytes)"))
	}
	path := deps.BindingPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return iterationExit(2, err)
		}
		path = filepath.Join(home, ".config", "ailang", "mission-runtime.toml")
	}
	binding, err := iteration.LoadBinding(path)
	if err != nil {
		return iterationExit(2, err)
	}
	key := coordinator.MissionWorkItemKey{MissionID: name, WorkItemID: work}
	// Prove existing state read-only before acquiring writable access. The second
	// open also uses mode=rw, so removal between opens cannot create a database.
	ro, err := coordinator.OpenMissionReadOnlyStore(binding.StateDB)
	if err != nil {
		return iterationExit(2, err)
	}
	item, err := ro.GetMissionWorkItem(ctx, key)
	err = errors.Join(err, ro.Close())
	if err != nil {
		return iterationExit(2, err)
	}
	if item.Version != version || item.State != "needs_reconciliation" || (item.ReasonCode != "operator_cancelled" && item.ReasonCode != "deadline_exceeded") {
		return iterationExit(3, coordinator.ErrMissionAttemptConflict)
	}
	store, err := coordinator.OpenMissionExistingWriteStore(binding.StateDB)
	if err != nil {
		return iterationExit(2, err)
	}
	defer store.Close()
	if err = store.ConfirmMissionWorkItemStoppedAttested(ctx, key, version, attestation); err != nil {
		return iterationExit(3, err)
	}
	evidence, err := store.GetMissionStopAttestation(ctx, key, version)
	if err != nil {
		return iterationExit(5, err)
	}
	_, err = fmt.Fprintf(out, "Confirmed stopped: %s/%s, version %d; attestation retained for %d child attempts. No execution dispatched.\n", name, work, version, len(evidence.Children))
	return err
}
