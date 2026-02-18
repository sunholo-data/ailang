package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/storage"
	"github.com/sunholo/ailang/internal/storage/migrate"
)

func storageCommand(args []string) error {
	if len(args) == 0 {
		printStorageHelp()
		return nil
	}

	switch args[0] {
	case "migrate":
		return storageMigrate(args[1:])
	case "verify":
		return storageVerify(args[1:])
	case "status":
		return storageStatus()
	case "--help", "-h", "help":
		printStorageHelp()
		return nil
	default:
		return fmt.Errorf("unknown storage subcommand: %s", args[0])
	}
}

func storageMigrate(args []string) error {
	opts := migrate.Options{
		BatchSize: 100,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--verbose", "-v":
			opts.Verbose = true
		case "--batch-size":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.BatchSize)
				i++
			}
		case "--collection":
			if i+1 < len(args) {
				opts.Collection = args[i+1]
				i++
			}
		case "--help", "-h":
			printStorageMigrateHelp()
			return nil
		}
	}

	ctx := context.Background()

	// Source: local SQLite
	srcBackends, err := storage.NewSQLiteBackends()
	if err != nil {
		return fmt.Errorf("failed to open local backends: %w", err)
	}
	defer srcBackends.Close()

	// Destination: GCP Firestore
	dstBackends, err := storage.NewGCPBackends(ctx)
	if err != nil {
		return fmt.Errorf("failed to open GCP backends: %w", err)
	}
	defer dstBackends.Close()

	src := &migrate.Sources{
		Coordinator: srcBackends.Coordinator,
		Messaging:   srcBackends.Messaging,
		Observatory: srcBackends.Observatory,
	}
	dst := &migrate.Destinations{
		Coordinator: dstBackends.Coordinator,
		Messaging:   dstBackends.Messaging,
		Observatory: dstBackends.Observatory,
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	migrator := migrate.NewMigrator(src, dst, opts, logger)

	if opts.DryRun {
		fmt.Println(cyan("→"), "Running migration in dry-run mode (no writes)")
	} else {
		fmt.Println(cyan("→"), "Migrating local SQLite → GCP Firestore...")
	}

	stats, err := migrator.Run(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println()
	fmt.Println(bold("Migration Results"))
	fmt.Printf("  Duration: %s\n", stats.Duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println(bold("Coordinator"))
	fmt.Printf("  Tasks:     %d\n", stats.CoordinatorTasks)
	fmt.Printf("  Approvals: %d\n", stats.CoordinatorApprovals)
	fmt.Printf("  Events:    %d\n", stats.CoordinatorEvents)
	fmt.Println()
	fmt.Println(bold("Messaging"))
	fmt.Printf("  Threads:   %d\n", stats.MessagingThreads)
	fmt.Printf("  Inbox:     %d\n", stats.MessagingInbox)
	fmt.Printf("  Agents:    %d\n", stats.MessagingAgents)
	fmt.Println()
	fmt.Println(bold("Observatory"))
	fmt.Printf("  Workspaces: %d\n", stats.ObservatoryWorkspaces)
	fmt.Printf("  Spans:      %d\n", stats.ObservatorySpans)

	if len(stats.Errors) > 0 {
		fmt.Println()
		fmt.Printf(yellow("⚠")+" %d error(s) during migration:\n", len(stats.Errors))
		for _, e := range stats.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if opts.DryRun {
		fmt.Println()
		fmt.Println(green("✓"), "Dry run complete. Run without --dry-run to execute.")
	} else {
		fmt.Println()
		fmt.Println(green("✓"), "Migration complete")
	}

	return nil
}

func storageVerify(args []string) error {
	ctx := context.Background()

	// Source: local SQLite
	srcBackends, err := storage.NewSQLiteBackends()
	if err != nil {
		return fmt.Errorf("failed to open local backends: %w", err)
	}
	defer srcBackends.Close()

	// Destination: GCP Firestore
	dstBackends, err := storage.NewGCPBackends(ctx)
	if err != nil {
		return fmt.Errorf("failed to open GCP backends: %w", err)
	}
	defer dstBackends.Close()

	src := &migrate.Sources{
		Coordinator: srcBackends.Coordinator,
		Messaging:   srcBackends.Messaging,
		Observatory: srcBackends.Observatory,
	}
	dst := &migrate.Destinations{
		Coordinator: dstBackends.Coordinator,
		Messaging:   dstBackends.Messaging,
		Observatory: dstBackends.Observatory,
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	migrator := migrate.NewMigrator(src, dst, migrate.Options{}, logger)

	fmt.Println(cyan("→"), "Verifying local vs GCP record counts...")

	issues, err := migrator.Verify(ctx)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println(green("✓"), "All counts match")
	} else {
		fmt.Printf(yellow("⚠")+" %d mismatch(es) found\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	return nil
}

func storageStatus() error {
	mode := storage.GetMode()
	fmt.Println(bold("Storage Configuration"))
	fmt.Printf("  Mode: %s\n", mode)
	fmt.Printf("  Env:  AILANG_STORAGE=%s\n", os.Getenv("AILANG_STORAGE"))

	switch mode {
	case storage.ModeLocal:
		fmt.Println()
		fmt.Println("  Using local SQLite databases in ~/.ailang/state/")
	case storage.ModeGCP:
		fmt.Println()
		fmt.Printf("  GCP Project: %s\n", os.Getenv("GOOGLE_CLOUD_PROJECT"))
		fmt.Println("  All databases stored in Firestore")
	case storage.ModeHybrid:
		fmt.Println()
		fmt.Printf("  GCP Project: %s\n", os.Getenv("GOOGLE_CLOUD_PROJECT"))
		fmt.Println("  Coordinator/Messaging: local SQLite")
		fmt.Println("  Observatory: GCP Firestore")
	}
	return nil
}

func printStorageHelp() {
	fmt.Println("Usage: ailang storage <command> [options]")
	fmt.Println("")
	fmt.Println("Manage storage backends and data migration")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  migrate    Migrate data from local SQLite to GCP Firestore")
	fmt.Println("  verify     Verify record counts match between local and GCP")
	fmt.Println("  status     Show current storage configuration")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang storage status")
	fmt.Println("  ailang storage migrate --dry-run")
	fmt.Println("  ailang storage migrate")
	fmt.Println("  ailang storage migrate --collection coordinator")
	fmt.Println("  ailang storage verify")
}

func printStorageMigrateHelp() {
	fmt.Println("Usage: ailang storage migrate [options]")
	fmt.Println("")
	fmt.Println("Migrate data from local SQLite to GCP Firestore")
	fmt.Println("")
	fmt.Println("Requires GOOGLE_CLOUD_PROJECT to be set.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --dry-run              Show what would be migrated without writing")
	fmt.Println("  --verbose, -v          Print detailed progress")
	fmt.Println("  --batch-size N         Records per batch (default: 100)")
	fmt.Println("  --collection NAME      Migrate specific collection only")
	fmt.Println("                         (coordinator, messaging, observatory)")
	fmt.Println("  --help, -h             Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang storage migrate --dry-run")
	fmt.Println("  ailang storage migrate --collection coordinator --verbose")
	fmt.Println("  ailang storage migrate --batch-size 500")
}
