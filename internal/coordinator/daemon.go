package coordinator

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/pubsub"
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
	msgStore         messaging.MessageStore
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

	// Pub/Sub cloud mode (M-PUBSUB)
	pubsubClient    *pubsub.Client
	pubsubPublisher *pubsub.Publisher

	// Cloud mode adapters (M-CLOUD-E2E / M-CLOUD-PUSH):
	// In push mode, HTTP handlers call HandleNotification/HandleCompletion directly.
	// In pull mode, Start() runs background goroutines.
	cloudInboxAdapter *PubSubInboxAdapter
	completionHandler *CompletionHandler

	// Cloud dispatcher for triggering remote task execution (M-CLOUD-DISPATCH)
	cloudDispatcher CloudDispatcher

	// API key cache for external user auth (M-CLOUD-DUAL-AUTH)
	apiKeyCache  *APIKeyCache
	kmsEncrypter *KMSEncrypter
}

// SetStores pre-sets the task store, messaging store, and observatory backend.
// When set, initTaskProcessing() will use these instead of opening local SQLite databases.
// Call this after NewDaemon() but before Start() to use cloud backends.
func (d *Daemon) SetStores(taskStore Store, msgStore messaging.MessageStore, obsBackend observatory.Backend) {
	d.taskStore = taskStore
	d.msgStore = msgStore
	d.obsBackend = obsBackend
}

// SetCloudDispatcher sets the cloud task dispatcher.
// Call this after NewDaemon() but before Start() to enable Cloud Run Job dispatch.
// The dispatcher is created in the CLI entry point to avoid circular imports.
func (d *Daemon) SetCloudDispatcher(dispatcher CloudDispatcher) {
	d.cloudDispatcher = dispatcher
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

	// M-CLOUD-DISPATCH: In cloud mode, log to both file and stderr.
	// Cloud Run only ingests stdout/stderr into Cloud Logging.
	var writer io.Writer = logFile
	if os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud {
		writer = io.MultiWriter(logFile, os.Stderr)
	}
	logger := log.New(writer, "[coordinator] ", log.LstdFlags|log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		config:           config,
		logger:           logger,
		logFile:          logFile,
		ctx:              ctx,
		cancel:           cancel,
		resourceRegistry: NewResourceTrackerRegistry(),
		apiKeyCache:      NewAPIKeyCache(10 * time.Minute),
		kmsEncrypter:     NewKMSEncrypter(), // nil if AILANG_KMS_KEY not set (local dev)
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
	// M-CLOUD-WEBHOOK: In cloud mode, use a longer poll interval (safety net only).
	// Primary work arrives via push handlers and webhooks, not polling.
	pollInterval := d.config.PollInterval
	if os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud {
		pollInterval = 5 * time.Minute
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Initialize task processing components
	if err := d.initTaskProcessing(); err != nil {
		d.logger.Printf("Warning: Failed to initialize task processing: %v", err)
		d.logger.Println("Daemon running in standby mode (no message processing)")
	} else {
		d.logger.Println("Daemon running, polling for tasks...")
	}

	// Start HTTP health server if PORT env var is set (Cloud Run convention)
	if port := os.Getenv("PORT"); port != "" {
		go d.startHealthServer(port)
	}

	// Initialize event broadcaster for real-time streaming
	// COORDINATOR_MODE=cloud uses Pub/Sub, local (default) uses HTTP
	if err := d.initEventBroadcaster(); err != nil {
		d.logger.Printf("Warning: Event broadcaster not available: %v", err)
		d.logger.Println("Task streaming disabled - events will be logged only")
	} else {
		d.logger.Println("Event broadcaster initialized")
	}

	// M-CLOUD-PUSH: Create completion handler in cloud mode.
	// In push mode, the HTTP handler at /pubsub/completions calls HandleCompletion directly.
	// In pull mode (fallback), Start() would run a background goroutine.
	if d.cloudInboxAdapter != nil && d.pubsubClient != nil && d.taskStore != nil {
		subscriber := pubsub.NewSubscriber(d.pubsubClient)
		d.completionHandler = NewCompletionHandler(subscriber, d.taskStore, d.msgStore, d.agentRegistry, d.logger)
		d.logger.Println("Cloud mode: completion handler ready (push delivery via /pubsub/completions)")
	}

	// M-CLOUD-JOB-RELIABILITY: Start stale task detector in cloud mode.
	// Periodically marks queued/running tasks as failed if they exceed their timeout.
	// Catches container failures, Pub/Sub delivery failures, and missed completions.
	if IsCloudMode() && d.taskStore != nil {
		detector := NewStaleTaskDetector(d.taskStore, d.agentRegistry, d.msgStore, d.logger)
		go detector.Run(d.ctx)
		d.logger.Println("Cloud mode: stale task detector started (interval=2m)")
	}

	// Register as an agent in the collaboration hub
	if err := d.registerAgent(); err != nil {
		d.logger.Printf("Warning: Failed to register agent: %v", err)
	}

	// M-CLOUD-WEBHOOK: In cloud mode, GitHub webhooks replace polling goroutines.
	// Local mode keeps polling as before.
	mode := os.Getenv("COORDINATOR_MODE")
	if mode != CoordinatorModeCloud {
		// Local mode: start polling-based GitHub sync and approval watcher
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
	} else {
		d.logger.Println("Cloud mode: GitHub polling disabled (using webhook delivery)")
		// Still load watched issues map for webhook label lookup
		if d.approvalWatcher != nil {
			if err := d.approvalWatcher.LoadWatchedIssuesFromStore(d.ctx); err != nil {
				d.logger.Printf("Warning: Failed to load watched issues: %v", err)
			} else {
				d.logger.Printf("Cloud mode: loaded %d watched issues for webhook lookup",
					d.approvalWatcher.WatchedIssueCount())
			}
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
