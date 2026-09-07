package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func missionAttemptCommand(args []string) error {
	ctx, c := context.WithTimeout(context.Background(), 15*time.Second)
	defer c()
	return runMissionAttempt(ctx, args, os.Stdout)
}
func runMissionAttempt(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang mission attempt <status|reconcile|cancel> --state-db FILE [--mission ID --work-item ID --stage ID] [--version N]")
	}
	action := args[0]
	if action != "status" && action != "reconcile" && action != "cancel" {
		return fmt.Errorf("unknown mission attempt action %q", action)
	}
	fs := flag.NewFlagSet("mission attempt "+action, flag.ContinueOnError)
	fs.SetOutput(out)
	db := fs.String("state-db", "", "existing coordinator SQLite database (required)")
	mission := fs.String("mission", "", "mission ID")
	work := fs.String("work-item", "", "work item ID")
	stage := fs.String("stage", "", "stage ID")
	version := fs.Int64("version", 0, "observed state version required for cancel")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *db == "" || fs.NArg() != 0 {
		return fmt.Errorf("explicit --state-db and no positional arguments required")
	}
	if action == "reconcile" && (*mission != "" || *work != "" || *stage != "" || *version != 0) {
		return fmt.Errorf("reconcile sweeps expired running attempts in the selected DB; key/version flags are not accepted")
	}
	if action != "reconcile" && (*mission == "" || *work == "" || *stage == "") {
		return fmt.Errorf("--mission, --work-item and --stage are required")
	}
	if action == "cancel" && *version < 1 {
		return fmt.Errorf("cancel requires the observed --version")
	}
	if action == "status" && *version != 0 {
		return fmt.Errorf("status does not accept --version")
	}
	if _, err := os.Stat(*db); err != nil {
		return err
	} // A typo must not initialize an empty authority.
	store, err := coordinator.NewSQLiteStore(*db)
	if err != nil {
		return err
	}
	defer store.Close()
	key := coordinator.MissionAttemptKey{MissionID: *mission, WorkItemID: *work, StageID: *stage}
	if action == "reconcile" {
		n, err := store.ReconcileMissionAttempts(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(map[string]int64{"needs_reconciliation": n})
	}
	if action == "cancel" {
		if err := store.CancelMissionAttempt(ctx, key, *version); err != nil {
			return err
		}
	}
	record, err := store.GetMissionAttempt(ctx, key)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(record)
}
