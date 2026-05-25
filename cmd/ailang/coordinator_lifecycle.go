package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/dispatch/cloudrun"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/storage"
	"github.com/sunholo-data/ailang/internal/telemetry"
)

func coordinatorStart(args []string) error {
	cfg := coordinator.DefaultConfig()

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--poll-interval":
			if i+1 < len(args) {
				duration, err := time.ParseDuration(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid poll-interval: %w", err)
				}
				cfg.PollInterval = duration
				i++
			}
		case "--max-worktrees":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid max-worktrees: %w", err)
				}
				cfg.MaxWorktrees = n
				i++
			}
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--log-file":
			if i+1 < len(args) {
				cfg.LogFile = args[i+1]
				i++
			}
		case "--help", "-h":
			printCoordinatorStartHelp()
			return nil
		}
	}

	// Initialize OpenTelemetry (if configured via environment variables)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-coordinator")
	if err != nil {
		fmt.Printf("  %s Warning: Failed to initialize OpenTelemetry: %v\n", yellow("!"), err)
	} else if telemetry.IsDualExportEnabled() {
		fmt.Printf("  %s Dual telemetry export enabled:\n", green("✓"))
		fmt.Printf("      → Google Cloud Trace (project: %s)\n", telemetry.GoogleCloudProject())
		fmt.Printf("      → OTLP endpoint: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	} else if telemetry.IsGoogleCloudEnabled() {
		fmt.Printf("  %s Google Cloud Trace enabled (project: %s)\n", green("✓"), telemetry.GoogleCloudProject())
	} else if telemetry.IsEnabled() {
		fmt.Printf("  %s OpenTelemetry OTLP export enabled\n", green("✓"))
	}
	// Note: shutdownTelemetry will be called when daemon stops via defer in daemon.Run()
	_ = shutdownTelemetry // We don't call it here since daemon runs indefinitely

	// Create daemon
	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Check if already running
	status, _ := daemon.Status()
	if status.Running {
		fmt.Println(yellow("⚠"), "Coordinator is already running")
		fmt.Printf("  PID: %d\n", status.PID)
		return nil
	}

	// M-COORD-MULTI-HOST-WORKERS (v0.24.0): install a file-backed HeartbeatStore
	// at ~/.ailang/state/worker_heartbeats.json so the daemon's heartbeats are
	// visible to a separate `workers list` CLI process on the same host.
	// Cross-host visibility (Firestore-backed) is the v0.25 roadmap item —
	// drops into the same HeartbeatStore interface without changing this wiring.
	daemon.SetHeartbeatStore(coordinator.NewFileHeartbeatStore(
		coordinator.DefaultHeartbeatPath(cfg.StateDir),
	))

	// Pre-set cloud backends if configured (AILANG_STORAGE=gcp|hybrid)
	storageMode := storage.GetMode()
	if storageMode != storage.ModeLocal {
		backends, err := storage.NewBackends(ctx)
		if err != nil {
			return fmt.Errorf("failed to create %s backends: %w", storageMode, err)
		}
		daemon.SetStores(backends.Coordinator, backends.Messaging, backends.Observatory)
		fmt.Printf("  Storage: %s\n", storageMode)
	}

	// M-CLOUD-DISPATCH: Create Cloud Run Jobs dispatcher in cloud mode.
	// Created here (not in coordinator package) to avoid circular imports.
	if os.Getenv("COORDINATOR_MODE") == "cloud" {
		projectID := os.Getenv("AILANG_CLOUD_PROJECT")
		region := os.Getenv("AILANG_CLOUD_REGION")
		if region == "" {
			region = "europe-west1"
		}
		prefix := os.Getenv("AILANG_TOPIC_PREFIX")
		if prefix == "" {
			prefix = pubsub.DefaultTopicPrefix
		}
		dispatcher, dispErr := cloudrun.NewDispatcher(ctx, projectID, region, prefix)
		if dispErr != nil {
			fmt.Printf("  %s Cloud Run Jobs dispatcher: %v\n", yellow("⚠"), dispErr)
		} else {
			daemon.SetCloudDispatcher(dispatcher)
			fmt.Printf("  %s Cloud Run Jobs dispatcher (region: %s)\n", green("✓"), region)
		}
	}

	// Start daemon
	fmt.Println(cyan("→"), "Starting coordinator daemon...")
	fmt.Printf("  PID file: %s\n", cfg.PIDFile)
	fmt.Printf("  Log file: %s\n", cfg.LogFile)
	fmt.Printf("  Poll interval: %s\n", cfg.PollInterval)
	fmt.Printf("  Max worktrees: %d\n", cfg.MaxWorktrees)

	if err := daemon.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Println(green("✓"), "Coordinator daemon started")
	return nil
}

func coordinatorStop(args []string) error {
	cfg := coordinator.DefaultConfig()
	forceCleanup := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--force", "-f":
			forceCleanup = true
		case "--help", "-h":
			printCoordinatorStopHelp()
			return nil
		}
	}

	// Force cleanup mode: remove lock files and PID file without checking status
	// This is the escape hatch when processes are stuck in uninterruptible sleep
	if forceCleanup {
		fmt.Println(cyan("→"), "Force cleanup mode...")
		cleaned := cleanupCoordinatorLockFiles(cfg.StateDir)
		if cleaned {
			fmt.Println(green("✓"), "Lock files cleaned up")
		} else {
			fmt.Println(yellow("⚠"), "No lock files found to clean")
		}
		return nil
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Check if running (with timeout protection now)
	status, _ := daemon.Status()
	if !status.Running {
		fmt.Println(yellow("⚠"), "Coordinator is not running")
		return nil
	}

	// Stop daemon
	fmt.Println(cyan("→"), "Stopping coordinator daemon...")
	if err := daemon.Stop(); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println(green("✓"), "Coordinator daemon stopped")
	return nil
}

// cleanupCoordinatorLockFiles removes SQLite lock files and PID file
// This is the recovery mechanism when processes are stuck in uninterruptible sleep
func cleanupCoordinatorLockFiles(stateDir string) bool {
	cleaned := false

	// Files to clean up
	lockFiles := []string{
		filepath.Join(stateDir, "coordinator.db-shm"),
		filepath.Join(stateDir, "coordinator.db-wal"),
		filepath.Join(stateDir, "collaboration.db-shm"),
		filepath.Join(stateDir, "collaboration.db-wal"),
		filepath.Join(stateDir, "coordinator.pid"),
	}

	for _, f := range lockFiles {
		if _, err := os.Stat(f); err == nil {
			if err := os.Remove(f); err != nil {
				fmt.Printf("  Warning: could not remove %s: %v\n", filepath.Base(f), err)
			} else {
				fmt.Printf("  Removed: %s\n", filepath.Base(f))
				cleaned = true
			}
		}
	}

	// Try to checkpoint the databases to flush WAL
	dbFiles := []string{
		filepath.Join(stateDir, "coordinator.db"),
		filepath.Join(stateDir, "collaboration.db"),
	}

	for _, dbPath := range dbFiles {
		if _, err := os.Stat(dbPath); err == nil {
			// Try to checkpoint using sqlite3 CLI
			cmd := exec.Command("sqlite3", dbPath, "PRAGMA wal_checkpoint(TRUNCATE);")
			if err := cmd.Run(); err == nil {
				fmt.Printf("  Checkpointed: %s\n", filepath.Base(dbPath))
				cleaned = true
			}
		}
	}

	return cleaned
}

func coordinatorStatus(args []string) error {
	cfg := coordinator.DefaultConfig()
	jsonOutput := false
	watchMode := false

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--json":
			jsonOutput = true
		case "--watch", "-w":
			watchMode = true
		case "--help", "-h":
			printCoordinatorStatusHelp()
			return nil
		}
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	if watchMode {
		return coordinatorStatusWatch(daemon)
	}

	status, err := daemon.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	printCoordinatorStatusOutput(status)
	return nil
}

// coordinatorStatusWatch shows live status updates
func coordinatorStatusWatch(daemon *coordinator.Daemon) error {
	fmt.Println(bold("Coordinator Status (live, press Ctrl+C to exit)"))
	fmt.Println()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Print initial status
	status, _ := daemon.Status()
	printCoordinatorStatusOutput(status)

	for range ticker.C {
		// Clear screen and move cursor to top
		fmt.Print("\033[2J\033[H")

		fmt.Println(bold("Coordinator Status (live, press Ctrl+C to exit)"))
		fmt.Println()

		status, err := daemon.Status()
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		printCoordinatorStatusOutput(status)
	}

	return nil
}

// printCoordinatorStatusOutput prints the status in human-readable format
func printCoordinatorStatusOutput(status *coordinator.Status) {
	if status.Running {
		fmt.Printf("  State:      %s %s\n", "▶", green("running"))
		fmt.Printf("  PID:        %d\n", status.PID)
		if !status.StartedAt.IsZero() {
			fmt.Printf("  Started:    %s\n", status.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if status.Uptime != "" {
			fmt.Printf("  Uptime:     %s\n", status.Uptime)
		}
	} else {
		fmt.Printf("  State:      %s %s\n", "⏹", red("stopped"))
	}

	fmt.Println()
	fmt.Println(bold("Task Statistics"))
	fmt.Printf("  Completed:  %s\n", green(fmt.Sprintf("%d", status.TasksRun)))
	if status.PendingTasks > 0 {
		fmt.Printf("  Pending:    %s\n", yellow(fmt.Sprintf("%d", status.PendingTasks)))
	}
	if status.RunningTasks > 0 {
		fmt.Printf("  Running:    %s\n", cyan(fmt.Sprintf("%d", status.RunningTasks)))
	}
	if status.PendingApprovals > 0 {
		fmt.Printf("  Approvals:  %s %s\n", yellow(fmt.Sprintf("%d", status.PendingApprovals)), "(use 'ailang coordinator pending' to review)")
	}
	if status.FailedTasks > 0 {
		fmt.Printf("  Failed:     %s\n", red(fmt.Sprintf("%d", status.FailedTasks)))
	}
	if status.TotalCost > 0 {
		fmt.Printf("  Total Cost: $%.4f\n", status.TotalCost)
	}
	if status.TotalTokens > 0 {
		fmt.Printf("  Tokens:     %d\n", status.TotalTokens)
	}
}

func printCoordinatorStartHelp() {
	fmt.Println("Usage: ailang coordinator start [options]")
	fmt.Println("")
	fmt.Println("Start the coordinator daemon")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --poll-interval DURATION   How often to check for new messages (default: 30s)")
	fmt.Println("  --max-worktrees N          Maximum concurrent worktrees (default: 3)")
	fmt.Println("  --state-dir DIR            State directory (default: ~/.ailang/state)")
	fmt.Println("  --log-file PATH            Log file path (default: ~/.ailang/logs/coordinator.log)")
	fmt.Println("  --help, -h                 Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator start")
	fmt.Println("  ailang coordinator start --poll-interval 60s")
	fmt.Println("  ailang coordinator start --max-worktrees 5")
}

func printCoordinatorStopHelp() {
	fmt.Println("Usage: ailang coordinator stop [options]")
	fmt.Println("")
	fmt.Println("Stop the coordinator daemon")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --force, -f       Force cleanup: remove lock files without checking status")
	fmt.Println("                    Use this if normal stop hangs (SQLite lock contention)")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator stop              # Normal graceful stop")
	fmt.Println("  ailang coordinator stop --force      # Emergency cleanup if hung")
}

func printCoordinatorStatusHelp() {
	fmt.Println("Usage: ailang coordinator status [options]")
	fmt.Println("")
	fmt.Println("Show coordinator daemon status")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --json            Output status as JSON")
	fmt.Println("  --watch, -w       Watch mode: continuously update status")
	fmt.Println("  --help, -h        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator status")
	fmt.Println("  ailang coordinator status --json")
	fmt.Println("  ailang coordinator status --watch")
}
