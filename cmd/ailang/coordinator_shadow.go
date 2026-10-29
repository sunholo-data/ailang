package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// coordinatorShadow is `ailang coordinator shadow report` (M-AILANG-LANE-EXECUTOR-SHADOW M7).
func coordinatorShadow(args []string) error {
	if len(args) == 0 || args[0] != "report" {
		fmt.Println("Usage: ailang coordinator shadow report [--since 7d|30d|all] [--json] [--post] [--remote gcp]")
		fmt.Println()
		fmt.Println("  Per-harness outcomes of shadowed ailang_only tasks, every disagreement, and the verdict tally.")
		fmt.Println("  --post sends the markdown to the user inbox (the coordinator also posts it weekly, Mondays).")
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("unknown shadow subcommand %q", args[0])
	}
	fs := flag.NewFlagSet("coordinator shadow report", flag.ContinueOnError)
	remote := fs.String("remote", "", "plane: local|gcp (default $AILANG_STORAGE_COORDINATOR, then $AILANG_STORAGE)")
	stateDir := fs.String("state-dir", "", "local state dir (local mode only)")
	sinceFlag := fs.String("since", "7d", "window: Nd (days) or all")
	asJSON := fs.Bool("json", false, "emit the report as JSON")
	post := fs.Bool("post", false, "send the report to the user inbox")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	now := time.Now().UTC()
	since, err := parseShadowSince(*sinceFlag, now)
	if err != nil {
		return err
	}
	ctx := context.Background()
	bundle, err := openCoordinatorStore(ctx, *remote, *stateDir)
	if err != nil {
		return err
	}
	defer bundle.Close()
	store, ok := bundle.Store.(coordinator.ShadowComparisonStore)
	if !ok {
		return fmt.Errorf("the %s store holds no shadow comparisons", bundle.Mode)
	}
	cs, err := store.ListShadowComparisonsSince(ctx, since)
	if err != nil {
		return fmt.Errorf("listing shadow comparisons: %w", err)
	}
	report := coordinator.BuildShadowReport(cs, since, now)
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Printf("store: %s\n\n%s", bundle.Mode, report.Markdown())
	if *post {
		msgs, err := openStore()
		if err != nil {
			return err
		}
		msg := coordinator.ShadowReportMessage(cs, now)
		msg.ID = "" // an on-demand post is not the weekly one
		if err := msgs.InsertInboxMessage(msg); err != nil {
			return fmt.Errorf("posting the report: %w", err)
		}
		fmt.Println("\nposted to the user inbox")
	}
	return nil
}

// parseShadowSince reads "7d", "30d" or "all".
func parseShadowSince(s string, now time.Time) (time.Time, error) {
	if s == "all" {
		return time.Time{}, nil
	}
	days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	if err != nil || !strings.HasSuffix(s, "d") || days <= 0 {
		return time.Time{}, fmt.Errorf("--since %q: want Nd (e.g. 7d) or all", s)
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour), nil
}
