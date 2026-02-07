package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/observatory"
)

func chainsDiffCommand() {
	fs := flag.NewFlagSet("chains diff", flag.ExitOnError)
	statOnly := fs.Bool("stat", false, "Show diffstat only")
	jsonOutput := fs.Bool("json", false, "Output as JSON (stage metadata)")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains diff <chain-id> [--stat] [--json]")
		fmt.Println()
		fmt.Println("Show the combined git diff across all stages in a chain.")
		fmt.Println("Cross-references observatory (chain stages) and coordinator (task worktrees).")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	// Connect to observatory database for chain/stage data
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve chain ID prefix
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get chain stages
	stages, err := backend.GetChainStages(ctx, chainID, observatory.ChainReadOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain stages: %v\n", err)
		os.Exit(1)
	}

	if len(stages) == 0 {
		fmt.Printf("Chain %s has no stages.\n", truncateChainID(chainID))
		return
	}

	// Connect to coordinator database for task worktree data
	cfg := coordinator.DefaultConfig()
	coordDBPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(coordDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open coordinator database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	fmt.Printf("Chain: %s (%d stages)\n", truncateChainID(chainID), len(stages))
	fmt.Println("═══════════════════════════════════════════")

	diffFound := false
	for i, stage := range stages {
		if stage.TaskID == "" {
			continue
		}

		// Look up task record for worktree info
		task, err := store.GetTask(ctx, stage.TaskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Stage %d (%s): task %s not found in coordinator\n", i+1, stage.AgentID, stage.TaskID)
			continue
		}

		if task.WorktreePath == "" {
			fmt.Printf("\n  Stage %d (%s): no worktree (task %s)\n", i+1, stage.AgentID, stage.TaskID)
			continue
		}

		// Check worktree exists
		if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
			fmt.Printf("\n  Stage %d (%s): worktree removed (%s)\n", i+1, stage.AgentID, task.WorktreePath)
			continue
		}

		// Print stage header
		fmt.Printf("\n%s Stage %d: %s [%s]\n", bold("───"), i+1, stage.AgentID, colorizeStatus(string(stage.Status)))
		fmt.Printf("    Task: %s\n", stage.TaskID)
		fmt.Printf("    Worktree: %s\n", task.WorktreePath)
		if stage.Cost > 0 {
			fmt.Printf("    Cost: $%.4f (%d tokens)\n", stage.Cost, stage.TokensIn+stage.TokensOut)
		}
		fmt.Println()

		if *jsonOutput {
			continue // JSON mode just shows metadata
		}

		// Run git diff
		diffRef := "HEAD~1"
		if task.BaseCommit != "" {
			diffRef = task.BaseCommit
		}

		var cmd *exec.Cmd
		if *statOnly {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--stat", diffRef+"..HEAD")
		} else {
			cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--color=always", diffRef+"..HEAD")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			// Fallback: diff against origin/dev
			if *statOnly {
				cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--stat", "origin/dev", "HEAD")
			} else {
				cmd = exec.Command("git", "-C", task.WorktreePath, "diff", "--color=always", "origin/dev", "HEAD")
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: git diff failed: %v\n", err)
				continue
			}
		}
		diffFound = true
	}

	if !diffFound && !*jsonOutput {
		fmt.Println("\nNo diffs available (worktrees may have been cleaned up).")
	}
}
