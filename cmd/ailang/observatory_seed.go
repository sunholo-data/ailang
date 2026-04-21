package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func observatorySeedCommand() {
	fs := flag.NewFlagSet("observatory seed", flag.ExitOnError)
	minimal := fs.Bool("minimal", false, "Generate minimal test data (1 workspace, 2 tasks, 5 spans)")
	stress := fs.Bool("stress", false, "Generate stress test data (10 workspaces, 100 tasks, 1000+ spans)")
	clean := fs.Bool("clean", false, "Delete all existing data before seeding")
	dbPath := fs.String("db", "", "Custom database path (default: ~/.ailang/state/observatory.db)")
	verbose := fs.Bool("verbose", false, "Show detailed output during seeding")

	// Skip "ailang observatory seed" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get database path
	path := *dbPath
	if path == "" {
		path = observatory.DefaultDatabasePath()
	}

	// Clean database if requested
	if *clean {
		if *verbose {
			fmt.Printf("Cleaning database at %s...\n", path)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: could not remove database: %v\n", err)
		}
	}

	backend, err := observatory.NewSQLiteBackendFromPath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	// Select configuration
	var cfg observatory.SeedConfig
	switch {
	case *minimal:
		cfg = observatory.MinimalSeedConfig()
	case *stress:
		cfg = observatory.StressSeedConfig()
	default:
		cfg = observatory.DefaultSeedConfig()
	}
	cfg.Verbose = *verbose
	cfg.CleanFirst = *clean

	// Run seeding
	fmt.Println("Seeding observatory database...")
	if *verbose {
		fmt.Printf("  Database: %s\n", path)
		fmt.Printf("  Workspaces: %d\n", cfg.NumWorkspaces)
		fmt.Printf("  Tasks per workspace: %d-%d\n", cfg.TasksPerWorkspace.Min, cfg.TasksPerWorkspace.Max)
		fmt.Printf("  Spans per task: %d-%d\n", cfg.SpansPerTask.Min, cfg.SpansPerTask.Max)
	}

	result, err := observatory.SeedDatabase(ctx, backend, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error seeding database: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	fmt.Println("=== Seed Complete ===")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Workspaces created:   %d\n", result.WorkspacesCreated)
	fmt.Printf("Tasks created:        %d\n", result.TasksCreated)
	fmt.Printf("Assignments created:  %d\n", result.AssignmentsCreated)
	fmt.Printf("Spans created:        %d\n", result.SpansCreated)
	fmt.Printf("Events created:       %d\n", result.EventsCreated)
	fmt.Printf("Messages created:     %d\n", result.MessagesCreated)
	fmt.Println()
	fmt.Println("View the data at: http://localhost:1957 (run 'ailang serve' first)")
}

// observatoryHeatmapCommand outputs activity heatmap data.
