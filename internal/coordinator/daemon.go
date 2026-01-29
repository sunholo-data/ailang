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
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/telemetry"
	traceAttribute "go.opentelemetry.io/otel/attribute"
)

// Config holds daemon configuration
type Config struct {
	PollInterval         time.Duration // How often to check for new messages
	MaxWorktrees         int           // Maximum concurrent worktrees
	LogFile              string        // Path to log file
	PIDFile              string        // Path to PID file
	StateDir             string        // Directory for state files
	ApprovalPollInterval time.Duration // How often to poll GitHub for approval labels (M-COORD-GITHUB-AUTO-ROUTING)
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
	PendingTasks     int     `json:"pending_tasks,omitempty"`
	RunningTasks     int     `json:"running_tasks,omitempty"`
	PendingApprovals int     `json:"pending_approvals,omitempty"` // Tasks awaiting human approval
	FailedTasks      int     `json:"failed_tasks,omitempty"`
	TotalCost        float64 `json:"total_cost,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
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
	msgStore         *messaging.Store
	msgAdapter       *InboxMessageAdapter            // Legacy: single adapter (for backwards compat)
	inboxAdapters    map[string]*InboxMessageAdapter // Key: inbox name
	worktreeManagers map[string]*WorktreeManager     // Key: agent ID
	analyzer         *TaskAnalyzer
	worktreeMgr      *WorktreeManager // Legacy: default worktree manager
	taskStore        Store
	executor         *TaskExecutor

	// Agent configuration
	agentRegistry *AgentRegistry
	coordConfig   *CoordinatorConfig

	// Approval workflow
	approvalCheckpoint *ApprovalCheckpoint

	// GitHub-driven approval workflow (M-COORD-GITHUB-AUTO-ROUTING)
	githubPoster    *GitHubPoster
	approvalWatcher *ApprovalWatcher
	taskChain       *TaskChain

	// Event broadcasting for real-time updates
	eventBroadcaster EventBroadcaster

	// Resource tracking for running tasks
	resourceRegistry *ResourceTrackerRegistry

	// History tracking
	instanceID string

	// Observatory integration for trace linking (M-TASK-HIERARCHY)
	observatorySync *ObservatorySync

	// Observatory backend for chain operations (M-CHAINS-SIMPLIFY)
	obsBackend observatory.Backend
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

		// Start label resync if enabled (M-MSG-ROUTING)
		if d.coordConfig.GitHubSync.ResyncLabels {
			go d.runLabelResync()
		}
	}

	// Start GitHub approval watcher (M-COORD-GITHUB-AUTO-ROUTING)
	if d.approvalWatcher != nil {
		if err := d.approvalWatcher.Start(d.ctx); err != nil {
			d.logger.Printf("Warning: Failed to start approval watcher: %v", err)
		} else {
			d.logger.Println("GitHub approval watcher started")
		}
	}

	// Catch-up: trigger any missed handoffs from approvals processed before this code was deployed
	if count, err := d.triggerMissedHandoffs(); err != nil {
		d.logger.Printf("Warning: Failed to trigger missed handoffs: %v", err)
	} else if count > 0 {
		d.logger.Printf("Triggered %d missed handoff(s) from previous approvals", count)
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

// triggerMissedHandoffs finds approved merge_handoff approvals that never had their handoffs triggered
// (due to approvals being processed before the handoff code was deployed) and triggers them now.
// Returns the count of handoffs triggered.
func (d *Daemon) triggerMissedHandoffs() (int, error) {
	if d.taskStore == nil || d.msgStore == nil || d.agentRegistry == nil {
		return 0, nil
	}

	// Find approved merge_handoff approvals where handoffs_triggered is still 0
	missedApprovals, err := d.taskStore.ListApprovedMergeHandoffsWithoutTrigger(d.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list missed handoffs: %w", err)
	}

	if len(missedApprovals) == 0 {
		return 0, nil
	}

	d.logger.Printf("Found %d approved merge_handoff approval(s) without triggered handoffs", len(missedApprovals))

	triggered := 0
	for _, approval := range missedApprovals {
		// Get the task for context
		task, err := d.taskStore.GetTask(d.ctx, approval.TaskID)
		if err != nil || task == nil {
			d.logger.Printf("Warning: could not get task %s for missed handoff: %v", approval.TaskID, err)
			continue
		}

		// Create ApprovalParams to reuse the existing handoff triggering logic
		params := &ApprovalParams{
			TaskID:        approval.TaskID,
			Store:         d.taskStore,
			MsgStore:      d.msgStore,
			AgentRegistry: d.agentRegistry,
		}

		// Use a background context span for catch-up
		ctx, span := telemetry.StartSpan(d.ctx, approvalProcessorTracer, "approval.catchup_handoff")
		span.SetAttributes(
			traceAttribute.String("task.id", approval.TaskID),
			traceAttribute.String("approval.type", approval.Type),
		)

		handoffTriggered, err := triggerHandoffsFromApprovalRecord(ctx, span, params, task, approval.TaskID, approval)
		if err != nil {
			d.logger.Printf("Warning: failed to trigger missed handoff for %s: %v", approval.TaskID, err)
			span.End()
			continue
		}

		if handoffTriggered {
			d.logger.Printf("Triggered missed handoff for task %s (title: %s)", approval.TaskID, task.Title)
			triggered++
		}
		span.End()
	}

	return triggered, nil
}
