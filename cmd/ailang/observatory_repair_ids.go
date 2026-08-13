package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// `ailang observatory repair-ids` — repair trace/span IDs corrupted by the
// OTLP/JSON decode defect fixed in v0.33.1.
//
// This is a CLOUD-side repair, deliberately separate from the SQLite migration.
// The deployed observatories run AILANG_STORAGE=gcp, so their spans are in
// Firestore, and internal/observatory migrations only run on the SQLite paths —
// migrate_v18 silently does nothing to dev or prod.
//
// Why an explicit command rather than an automatic startup migration: a
// migration that runs on every boot against a live production datastore is a far
// worse failure mode than one a human invokes deliberately, and the corruption is
// bounded and historical — it stopped accruing the moment v0.33.1 deployed.
func observatoryRepairIDsCommand() {
	fs := flag.NewFlagSet("observatory repair-ids", flag.ExitOnError)
	apply := fs.Bool("apply", false, "Actually write the repairs. WITHOUT this flag the command only reports what it would do.")
	project := fs.String("project", "", "GCP project holding the observatory (default: AILANG_CLOUD_PROJECT)")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	proj := *project
	if proj == "" {
		proj = os.Getenv("AILANG_CLOUD_PROJECT")
	}
	if proj == "" {
		fmt.Fprintln(os.Stderr, "Error: no project. Pass --project or set AILANG_CLOUD_PROJECT.")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := fsstore.NewClientForProject(ctx, proj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to Firestore in %s: %v\n", proj, err)
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	store := fsstore.NewObservatoryStore(client)

	mode := "DRY RUN (no writes)"
	if *apply {
		mode = "APPLY (writing)"
	}
	fmt.Printf("Observatory ID repair — project %s — %s\n\n", proj, mode)

	report, err := store.RepairCorruptedSpanIDs(ctx, !*apply)
	if err != nil {
		// Report partial progress: with chunked batches, some repairs may already have
		// landed before the failure, and the operator needs to know that.
		fmt.Fprintf(os.Stderr, "\nError during repair: %v\n", err)
		fmt.Fprintf(os.Stderr, "Documents written before the failure: %d\n", report.DocsWritten)
		fmt.Fprintln(os.Stderr, "The repair is idempotent — re-running finishes the job.")
		os.Exit(1)
	}

	fmt.Printf("  scanned            %d spans\n", report.Scanned)
	fmt.Printf("  trace_id  repairs  %d\n", report.TraceIDsFixed)
	fmt.Printf("  span_id   repairs  %d\n", report.SpanIDsFixed)
	fmt.Printf("  parent_id repairs  %d\n", report.ParentIDsFixed)
	if report.Skipped > 0 {
		fmt.Printf("  SKIPPED            %d (repaired id already exists — the same span under both encodings; they keep their corrupted ids)\n", report.Skipped)
	}
	fmt.Printf("  documents written  %d\n", report.DocsWritten)

	if len(report.Samples) > 0 {
		fmt.Println("\n  samples:")
		for _, s := range report.Samples {
			fmt.Printf("    %s\n", s)
		}
	}

	if !*apply {
		needed := report.TraceIDsFixed + report.SpanIDsFixed + report.ParentIDsFixed
		fmt.Println()
		if needed == 0 {
			fmt.Println("Nothing to repair.")
			return
		}
		fmt.Println("Dry run — nothing was written. Re-run with --apply to perform the repair.")
	}
}
