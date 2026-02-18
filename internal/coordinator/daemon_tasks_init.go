package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/websocket"
)

// stageToAgentID maps task stages to agent IDs for config lookup.
// Returns empty string for unknown stages (fallback to legacy behavior).
func stageToAgentID(stage TaskStage) string {
	switch stage {
	case TaskStageDesign:
		return "design-doc-creator"
	case TaskStageSprint:
		return "sprint-planner"
	case TaskStageImplementation:
		return "sprint-executor"
	default:
		return ""
	}
}

// initTaskProcessing initializes the message adapter, analyzer, and store
func (d *Daemon) initTaskProcessing() error {
	// Load agent configuration from ~/.ailang/config.yaml
	coordConfig, err := LoadCoordinatorConfig()
	if err != nil {
		return fmt.Errorf("failed to load coordinator config: %w", err)
	}
	d.coordConfig = coordConfig

	// Build agent registry
	d.agentRegistry = NewAgentRegistry()
	for _, agent := range coordConfig.Agents {
		if regErr := d.agentRegistry.Register(agent); regErr != nil {
			d.logger.Printf("Warning: Failed to register agent %q: %v", agent.ID, regErr)
		}
	}
	d.logger.Printf("Agent registry initialized with %d agent(s): %v",
		d.agentRegistry.Count(), d.agentRegistry.ListInboxes())

	// Validate agent references
	if issues := d.agentRegistry.Validate(); len(issues) > 0 {
		for _, issue := range issues {
			d.logger.Printf("Warning: Agent config issue: %s", issue)
		}
	}

	// Initialize inbox adapters for each configured agent
	d.inboxAdapters = make(map[string]*InboxMessageAdapter)
	d.worktreeManagers = make(map[string]*WorktreeManager)

	// Open shared message store (skip if already set via SetStores)
	if d.msgStore == nil {
		adapter, store, err := OpenDefaultInboxAdapter("coordinator")
		if err != nil {
			return fmt.Errorf("failed to open inbox adapter: %w", err)
		}
		d.msgAdapter = adapter // Keep for backwards compatibility
		d.msgStore = store
	} else {
		// Wrap pre-set store in adapter for backwards compatibility
		d.msgAdapter = NewInboxMessageAdapter(d.msgStore, "coordinator")
	}

	// Create adapters and worktree managers for each agent
	for _, agent := range coordConfig.Agents {
		// Create inbox adapter
		inboxAdapter := &InboxMessageAdapter{
			store: d.msgStore,
			inbox: agent.Inbox,
		}
		d.inboxAdapters[agent.Inbox] = inboxAdapter
		d.logger.Printf("Watching inbox %q for agent %q", agent.Inbox, agent.ID)

		// Create worktree manager with agent's workspace
		workspace := agent.Workspace
		if workspace == "" || workspace == "." {
			// Use current working directory
			workspace, _ = os.Getwd()
		}
		worktreeBase := filepath.Join(d.config.StateDir, "worktrees", agent.ID)
		worktreeMgr, wmErr := NewWorktreeManager(workspace, worktreeBase, d.config.MaxWorktrees)
		if wmErr != nil {
			d.logger.Printf("Warning: Failed to create worktree manager for agent %q: %v", agent.ID, wmErr)
			continue
		}
		d.worktreeManagers[agent.ID] = worktreeMgr
		d.logger.Printf("Worktree manager ready for agent %q (base: %s)", agent.ID, worktreeBase)
	}

	// Initialize analyzer
	d.analyzer = NewTaskAnalyzer(0.8)

	// Initialize task store (skip if already set via SetStores)
	if d.taskStore == nil {
		dbPath := filepath.Join(d.config.StateDir, "coordinator.db")
		taskStore, err := NewSQLiteStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open task store: %w", err)
		}
		d.taskStore = taskStore
	}

	// Legacy worktree manager (for coordinator agent fallback)
	d.worktreeMgr = d.worktreeManagers["coordinator"]
	if d.worktreeMgr == nil {
		// Create default if coordinator wasn't configured
		worktreeMgr, wmErr := NewWorktreeManager("", filepath.Join(d.config.StateDir, "worktrees", "coordinator"), d.config.MaxWorktrees)
		if wmErr != nil {
			return fmt.Errorf("failed to create default worktree manager: %w", wmErr)
		}
		d.worktreeMgr = worktreeMgr
	}

	// Initialize task executor with available providers
	executor, err := DefaultTaskExecutor()
	if err != nil {
		d.logger.Printf("Warning: Failed to create task executor: %v", err)
		d.logger.Println("Task execution disabled - will queue tasks but not execute them")
	} else {
		d.executor = executor
		d.logger.Printf("Task executor initialized with providers: %v", executor.ListProviders())
	}

	// Initialize approval checkpoint for human-in-the-loop workflow
	d.approvalCheckpoint = NewApprovalCheckpoint(24 * time.Hour) // 24h default timeout
	d.approvalCheckpoint.SetCallback(func(request *ApprovalRequest) {
		d.logger.Printf("Approval request %s resolved: %s (task: %s)", request.ID, request.Status, request.TaskID)
	})
	d.logger.Println("Approval checkpoint initialized")

	// Initialize GitHub-driven approval workflow (M-COORD-GITHUB-AUTO-ROUTING)
	githubPoster, err := NewGitHubPoster()
	if err != nil {
		d.logger.Printf("Warning: GitHub poster not available: %v", err)
		d.logger.Println("GitHub-based approval workflow disabled")
	} else {
		d.githubPoster = githubPoster

		// Create approval watcher with poll interval from config (default 60s)
		pollInterval := d.config.ApprovalPollInterval
		if pollInterval == 0 {
			pollInterval = 60 * time.Second
		}
		d.approvalWatcher = NewApprovalWatcher(githubPoster, d.taskStore, pollInterval)

		// Create task chain and register approval handlers
		d.taskChain = NewTaskChain(githubPoster, d.taskStore, d.approvalWatcher)

		// Register approval event handlers
		d.approvalWatcher.RegisterHandler(ApprovalEventDesign, d.taskChain.OnDesignApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventSprint, d.taskChain.OnSprintApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventMerge, d.taskChain.OnMergeApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventRevision, d.taskChain.OnNeedsRevision)

		// Load existing watched issues from store
		if err := d.approvalWatcher.LoadWatchedIssuesFromStore(d.ctx); err != nil {
			d.logger.Printf("Warning: Failed to load watched issues: %v", err)
		} else {
			d.logger.Printf("GitHub approval watcher initialized (watching %d issue(s), poll interval: %v)",
				d.approvalWatcher.WatchedIssueCount(), pollInterval)
		}
	}

	// Initialize Observatory sync for trace linking (M-TASK-HIERARCHY)
	if d.obsBackend == nil {
		// Open local SQLite observatory (skip if already set via SetStores)
		obsDBPath := filepath.Join(d.config.StateDir, "observatory.db")
		obsBackend, err := observatory.NewSQLiteBackendFromPath(obsDBPath)
		if err != nil {
			d.logger.Printf("Warning: Failed to initialize Observatory backend: %v", err)
			d.logger.Println("Observatory trace linking disabled")
		} else {
			d.obsBackend = obsBackend
			d.logger.Printf("Observatory sync initialized (db: %s)", obsDBPath)
		}
	}
	if d.obsBackend != nil && d.observatorySync == nil {
		d.observatorySync = NewObservatorySync(d.obsBackend, d.logger)
	}

	// Recover stale tasks from previous daemon runs
	// Tasks that were running/queued when daemon crashed are marked cancelled
	if d.taskStore != nil {
		staleThreshold := 5 * time.Minute // Tasks idle for >5 min are considered stale
		recovered, err := d.taskStore.RecoverStaleTasks(d.ctx, staleThreshold)
		if err != nil {
			d.logger.Printf("Warning: Failed to recover stale tasks: %v", err)
		} else if recovered > 0 {
			d.logger.Printf("Recovered %d stale task(s) from previous daemon run", recovered)
		}
	}

	// Clean up worktrees for tasks in terminal states (cancelled, completed, failed, rejected)
	// This handles worktrees left behind by RecoverStaleTasks and any other cleanup gaps
	d.cleanupWorktreesForTerminalTasks()

	return nil
}

// initHTTPBroadcaster initializes the HTTP broadcaster for streaming events to dashboard
func (d *Daemon) initHTTPBroadcaster() error {
	serverURL := DefaultServerURL()

	broadcaster := NewHTTPBroadcaster(serverURL, d.logger)

	// Check if server is reachable
	if !broadcaster.CheckServerAvailable() {
		return fmt.Errorf("collaboration hub server not available at %s", serverURL)
	}

	// Set the broadcaster
	d.SetEventBroadcaster(broadcaster.BroadcastFunc())
	d.logger.Printf("HTTP broadcaster initialized, streaming to %s", serverURL)

	return nil
}

// SetEventBroadcaster sets the event broadcaster for real-time task updates.
// When set, task execution events will be streamed via the broadcaster.
func (d *Daemon) SetEventBroadcaster(broadcaster EventBroadcaster) {
	d.eventBroadcaster = broadcaster
}

// CreateWebSocketBroadcaster creates an EventBroadcaster that broadcasts to a WebSocket server.
// This is used when the coordinator is connected to the collaboration hub server.
func CreateWebSocketBroadcaster(wsServer interface {
	BroadcastTaskEvent(stream *websocket.TaskStreamEvent)
}) EventBroadcaster {
	return func(event *websocket.TaskStreamEvent) {
		wsServer.BroadcastTaskEvent(event)
	}
}
