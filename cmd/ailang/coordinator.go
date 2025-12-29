package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

func coordinatorCommand(args []string) error {
	if len(args) == 0 {
		printCoordinatorHelp()
		return nil
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "start":
		return coordinatorStart(subargs)
	case "stop":
		return coordinatorStop(subargs)
	case "status":
		return coordinatorStatus(subargs)
	case "help", "--help", "-h":
		printCoordinatorHelp()
		return nil
	default:
		return fmt.Errorf("unknown coordinator subcommand: %s", subcommand)
	}
}

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

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state-dir":
			if i+1 < len(args) {
				cfg.StateDir = args[i+1]
				cfg.PIDFile = filepath.Join(args[i+1], "coordinator.pid")
				i++
			}
		case "--help", "-h":
			printCoordinatorStopHelp()
			return nil
		}
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Check if running
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

func coordinatorStatus(args []string) error {
	cfg := coordinator.DefaultConfig()
	jsonOutput := false

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
		case "--help", "-h":
			printCoordinatorStatusHelp()
			return nil
		}
	}

	daemon, err := coordinator.NewDaemon(cfg)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
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

	// Human-readable output
	fmt.Println(bold("Coordinator Status"))
	fmt.Println()

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
	fmt.Printf("  Tasks Run:  %d\n", status.TasksRun)

	return nil
}

func printCoordinatorHelp() {
	fmt.Println("Usage: ailang coordinator <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start     Start the coordinator daemon")
	fmt.Println("  stop      Stop the coordinator daemon")
	fmt.Println("  status    Show coordinator status")
	fmt.Println("  help      Show this help message")
	fmt.Println("")
	fmt.Println("The coordinator daemon watches for incoming messages and executes tasks")
	fmt.Println("using Claude Code or Gemini in isolated git worktrees.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator start")
	fmt.Println("  ailang coordinator start --poll-interval 60s --max-worktrees 2")
	fmt.Println("  ailang coordinator status --json")
	fmt.Println("  ailang coordinator stop")
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
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --help, -h        Show this help message")
}

func printCoordinatorStatusHelp() {
	fmt.Println("Usage: ailang coordinator status [options]")
	fmt.Println("")
	fmt.Println("Show coordinator daemon status")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --state-dir DIR   State directory (default: ~/.ailang/state)")
	fmt.Println("  --json            Output status as JSON")
	fmt.Println("  --help, -h        Show this help message")
}
