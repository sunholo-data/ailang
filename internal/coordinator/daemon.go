package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// Config holds daemon configuration
type Config struct {
	PollInterval time.Duration // How often to check for new messages
	MaxWorktrees int           // Maximum concurrent worktrees
	LogFile      string        // Path to log file
	PIDFile      string        // Path to PID file
	StateDir     string        // Directory for state files
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".ailang", "state")
	logsDir := filepath.Join(homeDir, ".ailang", "logs")

	return &Config{
		PollInterval: 30 * time.Second,
		MaxWorktrees: 3,
		LogFile:      filepath.Join(logsDir, "coordinator.log"),
		PIDFile:      filepath.Join(stateDir, "coordinator.pid"),
		StateDir:     stateDir,
	}
}

// Status represents the daemon's current state
type Status struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
	TasksRun  int       `json:"tasks_run"`
	// Extended stats from database
	PendingTasks    int     `json:"pending_tasks,omitempty"`
	RunningTasks    int     `json:"running_tasks,omitempty"`
	PendingApprovals int    `json:"pending_approvals,omitempty"` // Tasks awaiting human approval
	FailedTasks     int     `json:"failed_tasks,omitempty"`
	TotalCost       float64 `json:"total_cost,omitempty"`
	TotalTokens     int     `json:"total_tokens,omitempty"`
}

// Daemon is the coordinator daemon
type Daemon struct {
	config    *Config
	logger    *log.Logger
	logFile   *os.File // Underlying log file handle (for cleanup on Windows)
	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time
	tasksRun  int

	// Task processing components
	msgStore        *messaging.Store
	msgAdapter      *InboxMessageAdapter        // Legacy: single adapter (for backwards compat)
	inboxAdapters   map[string]*InboxMessageAdapter // Key: inbox name
	worktreeManagers map[string]*WorktreeManager     // Key: agent ID
	analyzer        *TaskAnalyzer
	worktreeMgr     *WorktreeManager              // Legacy: default worktree manager
	taskStore       Store
	executor        *TaskExecutor

	// Agent configuration
	agentRegistry *AgentRegistry
	coordConfig   *CoordinatorConfig

	// Approval workflow
	approvalCheckpoint *ApprovalCheckpoint

	// Event broadcasting for real-time updates
	eventBroadcaster EventBroadcaster

	// Resource tracking for running tasks
	resourceRegistry *ResourceTrackerRegistry

	// History tracking
	instanceID string
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) (*Daemon, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(config.LogFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(config.PIDFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, "[coordinator] ", log.LstdFlags|log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		config:           config,
		logger:           logger,
		logFile:          logFile,
		ctx:              ctx,
		cancel:           cancel,
		resourceRegistry: NewResourceTrackerRegistry(),
	}, nil
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Check if already running
	if status, _ := d.Status(); status.Running {
		return fmt.Errorf("daemon already running with PID %d", status.PID)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.startedAt = time.Now()
	d.logger.Printf("Daemon starting (PID: %d)", os.Getpid())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		d.logger.Printf("Received signal: %v, shutting down...", sig)
		d.cancel()
	}()

	// Run main loop
	return d.Run()
}

// Run is the main daemon loop
func (d *Daemon) Run() error {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	// Initialize task processing components
	if err := d.initTaskProcessing(); err != nil {
		d.logger.Printf("Warning: Failed to initialize task processing: %v", err)
		d.logger.Println("Daemon running in standby mode (no message processing)")
	} else {
		d.logger.Println("Daemon running, polling for tasks...")
	}

	// Initialize HTTP broadcaster for real-time streaming to dashboard
	if err := d.initHTTPBroadcaster(); err != nil {
		d.logger.Printf("Warning: HTTP broadcaster not available: %v", err)
		d.logger.Println("Task streaming to dashboard disabled - events will be logged only")
	} else {
		d.logger.Println("HTTP broadcaster connected to Collaboration Hub")
	}

	// Register as an agent in the collaboration hub
	if err := d.registerAgent(); err != nil {
		d.logger.Printf("Warning: Failed to register agent: %v", err)
	}

	// Start GitHub sync if enabled
	if d.coordConfig != nil && d.coordConfig.GitHubSync != nil && d.coordConfig.GitHubSync.Enabled {
		go d.runGitHubSync()
	}

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Daemon shutting down gracefully")
			d.cleanup()
			return nil
		case <-ticker.C:
			d.logger.Println("Checking for new tasks...")
			if d.msgAdapter != nil {
				if err := d.pollAndProcessTasks(); err != nil {
					d.logger.Printf("Error processing tasks: %v", err)
				}
			}

			// Execute pending tasks
			if d.executor != nil {
				if err := d.executeTaskQueue(); err != nil {
					d.logger.Printf("Error executing tasks: %v", err)
				}
			}

			// Sync worktree manager state with disk (handles CLI-initiated cleanups)
			d.syncWorktreeState()
		}
	}
}
