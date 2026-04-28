package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// deriveRepoURL converts a workspace identifier into a Git repo URL.
// In cloud mode, workspace is a GitHub org/repo (e.g., "sunholo-data/ailang").
// Falls back to AILANG_REPO_URL env var, then returns empty string.
func deriveRepoURL(workspace string) string {
	// If workspace looks like a GitHub org/repo path (contains exactly one slash,
	// no path separators like /Users/ or C:\), treat it as a GitHub repo.
	if workspace != "" && strings.Count(workspace, "/") == 1 && !strings.HasPrefix(workspace, "/") {
		return fmt.Sprintf("https://github.com/%s.git", workspace)
	}
	// Fall back to env var for backwards compatibility
	return os.Getenv("AILANG_REPO_URL")
}

// coordinatorTracer returns the tracer for coordinator instrumentation.
var coordinatorTracer = telemetry.Tracer("coordinator")

// executeTaskQueue picks up pending tasks and executes them.
// M-CLOUD-E2E: In cloud mode, dispatches tasks via Pub/Sub to Cloud Run Jobs
// instead of executing locally.
func (d *Daemon) executeTaskQueue() error {
	// Cloud mode: dispatch via Pub/Sub (Eventarc triggers Cloud Run Job)
	if d.pubsubPublisher != nil && d.cloudInboxAdapter != nil {
		return d.dispatchTasksCloud()
	}

	// Local mode: execute tasks directly
	if d.executor == nil {
		return nil // No executor available
	}

	// Get pending tasks, ordered by priority (highest first)
	filter := &TaskFilter{
		Status:    []TaskStatus{TaskStatusPending},
		OrderBy:   "priority",
		OrderDesc: true, // Higher priority first
		Limit:     1,    // Process one at a time for now
	}

	tasks, err := d.taskStore.ListTasks(d.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if err := d.executeTask(task); err != nil {
			d.logger.Printf("Failed to execute task %s: %v", task.ID, err)
			// Mark task as failed
			_ = d.taskStore.MarkTaskFailed(d.ctx, task.ID, err)
			// Post failure message to thread
			d.postTaskResult(task, nil, err)
		}
	}

	return nil
}

// dispatchTasksCloud publishes pending tasks to the Pub/Sub tasks topic
// so that Eventarc can trigger Cloud Run Jobs for execution.
// M-CLOUD-E2E: Replaces local execution in cloud mode.
func (d *Daemon) dispatchTasksCloud() error {
	filter := &TaskFilter{
		Status:    []TaskStatus{TaskStatusPending},
		OrderBy:   "priority",
		OrderDesc: true,
		Limit:     5, // Batch dispatch up to 5 tasks
	}

	tasks, err := d.taskStore.ListTasks(d.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		// Mark as queued before publishing
		if err := d.taskStore.MarkTaskQueued(d.ctx, task.ID); err != nil {
			d.logger.Printf("Failed to mark task %s as queued: %v", task.ID, err)
			continue
		}

		// Determine provider from coordinator config
		provider := "claude"
		if d.coordConfig != nil && d.coordConfig.DefaultProvider != "" {
			provider = d.coordConfig.DefaultProvider
		}

		// Publish task to Pub/Sub for audit trail / event streaming.
		if err := d.pubsubPublisher.PublishTask(d.ctx, task.ID, task.AgentID, task.Workspace, provider); err != nil {
			d.logger.Printf("Failed to publish task %s to Pub/Sub: %v", task.ID, err)
			_ = d.taskStore.ResetTaskToPending(d.ctx, task.ID)
			continue
		}

		// M-CLOUD-DISPATCH: Trigger Cloud Run Job execution via dispatcher.
		// Pub/Sub publish above is for audit trail only — the dispatcher actually starts the job.
		if d.cloudDispatcher != nil {
			// M-PKG-FEEDBACK-LOOP M2: Apply template_file / template_by_message_type
			// before dispatch. Local mode does this in executeTask via
			// BuildDirectiveFromConfig; cloud mode used to send task.Content raw,
			// which silently bypassed every pkg-* agent's template_file. Look up
			// the agent config here so the cloud agent receives the same fully
			// templated prompt the local executor would build.
			directive := task.Content
			if d.agentRegistry != nil {
				if agent := d.agentRegistry.GetAgentByID(task.AgentID); agent != nil {
					directive = BuildDirectiveFromConfig(task, agent)
				}
			}
			params := DispatchParams{
				TaskID:    task.ID,
				AgentID:   task.AgentID,
				Workspace: task.Workspace,
				Provider:  provider,
				Directive: directive,
				RepoURL:   deriveRepoURL(task.Workspace),
				Branch:    task.BaseBranch, // From task record, defaults handled by job
			}
			// Pass plugin repo from coordinator config (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
			if d.coordConfig != nil && d.coordConfig.PluginRepo != "" {
				params.PluginRepo = d.coordConfig.PluginRepo
			}
			// Use agent config for branch resolution and skip_approval push mode.
			// The agent's MergeBranch is the correct clone branch for repos that
			// don't use "dev" as default (e.g., sunholo-websites uses "main").
			if agent := d.agentRegistry.GetAgentByID(task.AgentID); agent != nil {
				if agent.MergeBranch != "" && params.Branch == "" {
					params.Branch = agent.MergeBranch
				}
				if agent.SkipApproval && agent.MergeBranch != "" {
					params.PushBranch = agent.MergeBranch
				}
				if agent.Model != "" {
					params.Model = agent.Model
				}
				if agent.Timeout != "" {
					params.Timeout = agent.Timeout
				}
				// M-CLOUD-DUAL-AUTH: Per-agent default auth mode.
				if agent.AuthMode != "" {
					params.AuthMode = agent.AuthMode
				}
				// M-GIT-GUARDRAILS: Per-agent git mode for PreToolUse hook enforcement.
				if agent.GitMode != "" {
					params.GitMode = agent.GitMode
				}
				// M-EXECUTOR-VARIANTS: Per-agent Docker image variant selection.
				if agent.ExecutorVariant != "" {
					params.ExecutorVariant = agent.ExecutorVariant
				}
				// M-PKG-AUTONOMOUS-UPDATES: Pass subdirectory for monorepo package agents.
				if agent.Subdirectory != "" {
					params.Subdirectory = agent.Subdirectory
				}
			}
			// M-HARNESS-COMMIT-CONTRACT: Pass site metadata for structured commit messages.
			if task.SiteSlug != "" {
				params.SiteSlug = task.SiteSlug
			}
			if task.BriefID != "" {
				params.BriefID = task.BriefID
			}
			// M-CLOUD-PROGRESS-TRACKING: Pass per-task cost budget for mid-execution enforcement.
			if budgetsCfg, budgetErr := LoadBudgetsConfig(); budgetErr == nil && budgetsCfg != nil {
				var taskMaxCost float64
				if budgetsCfg.Providers != nil {
					if provCfg, ok := budgetsCfg.Providers[provider]; ok && provCfg != nil && provCfg.TaskMaxCost > 0 {
						taskMaxCost = provCfg.TaskMaxCost
					}
				}
				if taskMaxCost == 0 && budgetsCfg.Global != nil && budgetsCfg.Global.TaskMaxCost > 0 {
					taskMaxCost = budgetsCfg.Global.TaskMaxCost
				}
				if taskMaxCost > 0 {
					params.MaxCostUSD = taskMaxCost
				}
			}
			// M-CLOUD-DUAL-AUTH: Check if the originating message had a user-provided API key.
			// The cache is keyed by message ID — if a key exists, use apikey mode.
			// This overrides per-agent defaults (user-provided key takes precedence).
			if d.apiKeyCache != nil && task.MessageID != "" {
				if apiKey, ok := d.apiKeyCache.Retrieve(task.MessageID); ok {
					params.AuthMode = "apikey"
					params.APIKey = apiKey
				}
			}
			if err := d.cloudDispatcher.Dispatch(d.ctx, params); err != nil {
				d.logger.Printf("Failed to dispatch task %s to Cloud Run Job: %v", task.ID, err)
				_ = d.taskStore.ResetTaskToPending(d.ctx, task.ID)
				continue
			}
			d.logger.Printf("Cloud dispatch: task %s → Cloud Run Job (agent: %s, provider: %s)", task.ID, task.AgentID, provider)
		} else {
			d.logger.Printf("Cloud dispatch: published task %s to Pub/Sub only (no dispatcher, agent: %s, provider: %s)", task.ID, task.AgentID, provider)
		}
	}

	return nil
}

// executeTask runs a single task through the executor
func (d *Daemon) executeTask(task *TaskRecord) error {
	// Start OTEL span for task execution with a NEW trace root.
	// CRITICAL: Use context.Background() instead of d.ctx to avoid trace contamination
	// between tasks. Each task gets its own trace_id instead of all tasks sharing
	// the daemon's trace context (which caused Phase 16 trace mixing bug).
	taskCtx, span := telemetry.StartSpan(context.Background(), coordinatorTracer, "coordinator.task.execute",
		trace.WithNewRoot(), // Force new trace, don't inherit from previous tasks
		trace.WithAttributes(
			attribute.String("task.id", task.ID),
			attribute.String("task.type", string(task.Type)),
			attribute.String("task.stage", string(task.Stage)),
			attribute.String("task.title", task.Title),
			attribute.Int("task.iteration", task.Iteration), // M-TRANSCRIPT: feedback loop iteration
		),
	)
	defer span.End()

	// NOTE: Intentionally NOT mutating d.ctx to avoid trace contamination.
	// Use taskCtx for all operations within this task execution.

	d.logger.Printf("Starting execution of task %s (type: %s)", task.ID, task.Type)

	// Mark task as running
	if err := d.taskStore.MarkTaskRunning(taskCtx, task.ID, "", ""); err != nil {
		return fmt.Errorf("failed to mark task as running: %w", err)
	}
	// M-CHAINS-SIMPLIFY: Update stage status
	d.updateChainStageStatus(taskCtx, task, observatory.StageStatusRunning)
	d.updateChainStatus(taskCtx, task, observatory.ChainStatusActive)

	// Post "starting" message to thread
	d.postTaskStatus(task, "running", "Starting task execution...")

	// Determine which agent should handle this task
	// Look up the thread to get the target agent
	targetAgent := "coordinator" // default
	if task.ThreadID != "" && d.msgStore != nil {
		if thread, err := d.msgStore.GetThread(task.ThreadID); err == nil && thread != nil && thread.TargetAgent != "" {
			targetAgent = thread.TargetAgent
		}
	}

	// Create analyzed task for executor
	// Use config-driven directive for GitHub-linked tasks (M-COORD-GENERIC-WORKFLOWS)
	// Look up agent config by targetAgent (from thread) or fall back to stage-based lookup
	var agentConfig *AgentConfig
	if d.agentRegistry != nil {
		// First try targetAgent (from thread - for direct messages to inboxes)
		agentConfig = d.agentRegistry.GetAgentByID(targetAgent)
		if agentConfig == nil {
			// Fall back to stage-based lookup (for workflow stages)
			agentID := stageToAgentID(task.Stage)
			d.logger.Printf("[DEBUG] Task stage: %s -> agentID: %s", task.Stage, agentID)
			if agentID != "" {
				agentConfig = d.agentRegistry.GetAgentByID(agentID)
			}
		}
		if agentConfig != nil {
			d.logger.Printf("[DEBUG] Found agent config: ID=%s, Invoke=%+v", agentConfig.ID, agentConfig.Invoke)
		} else {
			d.logger.Printf("[DEBUG] No agent config found for agent: %s", targetAgent)
		}
	}

	// M-PKG-AUTONOMOUS-UPDATES: Adjust autonomy based on package message change class.
	// Must happen before budget/approval checks since it modifies SkipApproval/AutoMerge.
	if agentConfig != nil && d.msgStore != nil && task.MessageID != "" {
		if msg, err := d.msgStore.GetInboxMessage(task.MessageID); err == nil && msg != nil {
			adjusted := AdjustAutonomyForChangeClass(agentConfig, msg)
			if adjusted != agentConfig {
				d.logger.Printf("Autonomy router: task %s adjusted (SkipApproval=%v, AutoMerge=%v, AutoApproveHandoffs=%v)",
					task.ID, adjusted.SkipApproval, adjusted.AutoMerge, adjusted.AutoApproveHandoffs)
				agentConfig = adjusted
			}
		}
	}

	// Budget enforcement check (M-PER-PROVIDER-BUDGETS)
	// Check budget before executing task. If hard limit exceeded, create approval request.
	if budgetBlocked, budgetErr := d.checkBudgetBeforeExecution(taskCtx, task, agentConfig); budgetBlocked {
		if budgetErr != nil {
			d.logger.Printf("Budget check failed for task %s: %v", task.ID, budgetErr)
		}
		return nil // Task blocked, waiting for budget approval
	}

	directive := BuildDirectiveFromConfig(task, agentConfig)
	d.logger.Printf("[DEBUG] Built directive (first 500 chars): %s", truncateString(directive, 500))
	analyzed := &AnalyzedTask{
		Task: &Task{
			ID:           task.ID,
			Title:        task.Title,
			Content:      directive, // Stage-aware: design/sprint/implementation prompts
			Kind:         task.Kind,
			MessageID:    task.MessageID,
			ParentTaskID: task.ParentTaskID, // M-TASK-HIERARCHY
			CreatedAt:    task.CreatedAt,
			Iteration:    task.Iteration, // M-TRANSCRIPT: feedback loop iteration
			SessionID:    task.SessionID, // M-TRANSCRIPT: for Claude --resume
		},
		Type: task.Type,
	}

	// Check if this is a script agent (doesn't need worktree isolation)
	isScriptAgent := agentConfig != nil && agentConfig.Invoke != nil && agentConfig.Invoke.Type == "script"

	// Get the correct worktree manager for this agent
	worktreeMgr := d.worktreeManagers[targetAgent]
	if worktreeMgr == nil {
		worktreeMgr = d.worktreeMgr // Fallback to default
	}

	// Create worktree for task isolation (AI agents only)
	// Script agents run deterministic commands and don't modify files, so they don't need isolation
	var worktree *Worktree
	var workspacePath string

	if isScriptAgent {
		// Script agents use workspace directly - no worktree needed
		workspacePath = task.Workspace
		if workspacePath == "" && agentConfig != nil {
			workspacePath = agentConfig.Workspace
		}
		d.logger.Printf("Script agent %s using workspace directly: %s", targetAgent, workspacePath)
	} else if worktreeMgr != nil {
		var wtErr error
		// Get merge branch from agent config (inherits from global if not set)
		mergeBranch := ""
		if agentConfig != nil {
			mergeBranch = agentConfig.MergeBranch
		}
		worktree, wtErr = worktreeMgr.CreateWorktree(task.ID, mergeBranch)
		if wtErr != nil {
			// Check if this is a recoverable "limit reached" error
			if errors.Is(wtErr, ErrWorktreeLimitReached) {
				// Reset task back to pending - it will be retried when a worktree frees up
				// CRITICAL: Task was marked "running" at line 462, must reset or it's stuck forever!
				d.logger.Printf("WARN: Task %s worktree limit reached, resetting to pending for retry", task.ID)
				if resetErr := d.taskStore.ResetTaskToPending(taskCtx, task.ID); resetErr != nil {
					d.logger.Printf("ERROR: Failed to reset task %s to pending: %v", task.ID, resetErr)
				}
				return wtErr // Return error, task will be picked up on next poll
			}

			// CRITICAL: Do NOT continue without worktree!
			// Agent would modify main repo directly, causing data loss.
			d.logger.Printf("ERROR: Failed to create worktree for task %s (agent: %s): %v", task.ID, targetAgent, wtErr)
			failErr := fmt.Errorf("worktree creation failed: %w (check 'ailang coordinator cleanup' to free slots)", wtErr)
			if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, failErr); err != nil {
				d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
			}
			return failErr // Don't execute without worktree isolation
		}
		workspacePath = worktree.Path
	} else {
		// No worktree manager means we can't ensure isolation
		d.logger.Printf("ERROR: No worktree manager available for task %s (agent: %s)", task.ID, targetAgent)
		failErr := fmt.Errorf("no worktree manager available for agent %s", targetAgent)
		if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, failErr); err != nil {
			d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
		}
		return failErr
	}

	// Apply subdirectory for monorepo package agents (M-PKG-AUTONOMOUS-UPDATES, v0.10.0).
	// When set, the agent works within a subdirectory of the worktree (e.g., "packages/auth").
	if agentConfig != nil && agentConfig.Subdirectory != "" && workspacePath != "" {
		subdirPath := filepath.Join(workspacePath, agentConfig.Subdirectory)
		if info, err := os.Stat(subdirPath); err == nil && info.IsDir() {
			d.logger.Printf("Task %s scoped to subdirectory: %s (within %s)", task.ID, agentConfig.Subdirectory, workspacePath)
			workspacePath = subdirPath
		} else {
			d.logger.Printf("WARN: Subdirectory %q does not exist in workspace %s, using workspace root", agentConfig.Subdirectory, workspacePath)
		}
	}

	// Sync task to Observatory for trace linking (M-TASK-HIERARCHY)
	var obsContext *ObservatoryContext
	if d.observatorySync != nil {
		// Sync task entity
		if err := d.observatorySync.SyncTask(taskCtx, task); err != nil {
			d.logger.Printf("Warning: Failed to sync task to Observatory: %v", err)
		}

		// Create agent assignment and get ID for context propagation
		providerName := "claude" // fallback
		if agentConfig != nil && agentConfig.Provider != "" {
			providerName = agentConfig.Provider
		} else if d.coordConfig != nil && d.coordConfig.DefaultProvider != "" {
			providerName = d.coordConfig.DefaultProvider
		}
		assignmentID, err := d.observatorySync.SyncAgentAssignment(taskCtx, task.ID, targetAgent, providerName)
		if err != nil {
			d.logger.Printf("Warning: Failed to sync agent assignment: %v", err)
		}

		// Get workspace ID from sync cache
		workspaceID := d.observatorySync.GetWorkspaceID(task.Workspace)

		obsContext = &ObservatoryContext{
			TaskID:       task.ID,
			AgentID:      targetAgent,
			AssignmentID: assignmentID,
			WorkspaceID:  workspaceID,
			// Chain context for unified hierarchy tracking (M-CHAINS-SIMPLIFY)
			ChainID:   task.ChainID,
			StageID:   task.StageID,
			MessageID: task.MessageID,
		}
		d.logger.Printf("Observatory context: task=%s agent=%s assignment=%s workspace=%s",
			task.ID, targetAgent, assignmentID, workspaceID)
	}

	opts := &ExecuteOptions{
		Timeout:            agentConfig.GetEffectiveTimeout(),     // Hard ceiling (v0.8.1), default 60m
		IdleTimeout:        agentConfig.GetEffectiveIdleTimeout(), // Idle kill (v0.8.1), default 3m
		Workspace:          workspacePath,                         // Worktree path for AI agents, direct workspace for script agents
		ObservatoryContext: obsContext,
		AgentConfig:        agentConfig, // For system prompt construction (v0.8.0+)
	}

	// Pass per-agent model override (v0.8.0+)
	// If agent config specifies a model, use it instead of the executor default
	if agentConfig != nil && agentConfig.Model != "" {
		opts.Model = agentConfig.Model
		d.logger.Printf("Task %s using agent-configured model: %s", task.ID, agentConfig.Model)
	}

	// Pass per-agent effort level (Claude Code 2.1.47+)
	if agentConfig != nil && agentConfig.Effort != "" {
		opts.Effort = agentConfig.Effort
	}

	// Pass plugin directories (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	if agentConfig != nil && len(agentConfig.PluginDirs) > 0 {
		opts.PluginDirs = agentConfig.PluginDirs
		d.logger.Printf("Task %s using %d plugin dir(s)", task.ID, len(agentConfig.PluginDirs))
	}

	// Pass third-party plugins config (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	if agentConfig != nil && agentConfig.Plugins != nil {
		opts.Plugins = agentConfig.Plugins
		d.logger.Printf("Task %s using plugins config: %d marketplaces, %d installs",
			task.ID, len(agentConfig.Plugins.Marketplaces), len(agentConfig.Plugins.Install))
	}

	// Pass InvokeConfig for script execution (v0.6.4+)
	// This allows the TaskExecutor to route to ScriptProvider
	if agentConfig != nil && agentConfig.Invoke != nil {
		opts.InvokeConfig = agentConfig.Invoke
		if agentConfig.Invoke.Type == "script" {
			d.logger.Printf("Task %s will use script execution: %s", task.ID, agentConfig.Invoke.Command)
		}
	}

	// Create streaming event handler if broadcaster is available
	var eventHandler *CoordinatorEventHandler
	if d.eventBroadcaster != nil {
		eventHandler = NewCoordinatorEventHandler(task.ID, task.ThreadID, d.eventBroadcaster)
		opts.EventHandler = eventHandler

		// Set task context for event enrichment (workspace, directive, agent info)
		eventHandler.SetTaskContext(&TaskEventContext{
			Workspace:  task.Workspace,
			Directive:  task.Content,
			AgentID:    task.AgentID,
			SourceType: "coordinator",
		})

		// Set up event storage for historical replay
		if sqliteStore, ok := d.taskStore.(*SQLiteStore); ok {
			eventHandler.SetEventStorer(func(record *TaskEventRecord) error {
				return sqliteStore.StoreTaskEvent(taskCtx, record)
			})
		}

		// Emit starting status
		eventHandler.EmitStatus("running")
	}

	// Create resource tracker for monitoring
	// Use current PID as initial tracker - executor will spawn subprocess
	resourceTracker := NewResourceTracker(task.ID, task.ThreadID, 0)
	d.resourceRegistry.Register(task.ID, resourceTracker)

	// Set up update callback for WebSocket broadcasting
	if d.eventBroadcaster != nil {
		resourceTracker.SetUpdateCallback(func(metrics *ResourceMetrics) {
			// Broadcast metrics update via event handler status
			if eventHandler != nil {
				eventHandler.UpdateMetrics(metrics.TokensIn, metrics.TokensOut, metrics.Cost)
			}
		})
	}

	// Start resource tracking
	resourceTracker.Start(taskCtx)

	// Execute the task
	result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)

	// Stop resource tracking and get final metrics
	resourceTracker.Stop()
	finalMetrics := resourceTracker.GetMetrics()
	d.resourceRegistry.Unregister(task.ID)

	// Emit completion status via event handler
	if eventHandler != nil {
		if err != nil {
			eventHandler.OnError(err)
			eventHandler.EmitStatus("failed")
		} else if result != nil {
			eventHandler.UpdateMetrics(result.InputTokens, result.OutputTokens, result.Cost)
			if result.Success {
				eventHandler.EmitStatus("completed")
			} else {
				eventHandler.EmitStatus("failed")
			}
		}
	}

	// Store peak resource metrics in task record
	if finalMetrics != nil {
		_ = d.taskStore.UpdateTaskMetrics(taskCtx, task.ID, finalMetrics.PeakCPU, finalMetrics.PeakMemory)
	}

	if err != nil {
		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		// Cleanup worktree on error (no work to preserve)
		if worktree != nil && d.worktreeMgr != nil {
			if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
				d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
			}
		}
		return fmt.Errorf("executor error: %w", err)
	}

	// Record result metrics in span
	if result != nil {
		span.SetAttributes(
			attribute.Int64("task.tokens_in", int64(result.InputTokens)),
			attribute.Int64("task.tokens_out", int64(result.OutputTokens)),
			attribute.Float64("task.cost_usd", result.Cost),
			attribute.Bool("task.success", result.Success),
		)
	}

	// Record resource metrics in span (peak CPU/memory)
	if finalMetrics != nil {
		span.SetAttributes(
			attribute.Float64("task.peak_cpu_percent", finalMetrics.PeakCPU),
			attribute.Float64("task.peak_memory_mb", finalMetrics.PeakMemory),
			attribute.Float64("task.cpu_percent", finalMetrics.CPUPercent),
			attribute.Float64("task.memory_mb", finalMetrics.MemoryMB),
			attribute.Int("task.duration_sec", finalMetrics.DurationSec),
		)
	}

	// Update task status based on result
	if result.Success {
		// Check if this agent skips approval (e.g., script agents)
		skipApproval := agentConfig != nil && agentConfig.SkipApproval

		// PRESERVE WORKTREE - mark as pending approval, not completed
		// Human must approve/reject before worktree is cleaned up
		worktreePath := ""
		worktreeBranch := ""
		baseBranch := ""
		baseCommit := ""
		if worktree != nil {
			worktreePath = worktree.Path
			worktreeBranch = worktree.Branch
			baseBranch = worktree.BaseBranch
			baseCommit = worktree.BaseCommit

			// AUTO-COMMIT: Deterministically commit any uncommitted changes
			// Agents may forget to commit, so we do it automatically before approval
			if err := autoCommitWorktree(worktreePath, task.Title, d.logger); err != nil {
				d.logger.Printf("Warning: Auto-commit failed for task %s: %v", task.ID, err)
				// Continue anyway - some tasks may not produce file changes
			}
		}

		// M-PKG-AUTONOMOUS-UPDATES: Deterministic version bump + publish for package agents.
		// Don't rely on the AI for mechanical steps — bump version based on change class
		// and run `ailang publish` deterministically after the agent finishes.
		if agentConfig != nil && agentConfig.Subdirectory != "" && worktreePath != "" {
			publishDir := filepath.Join(worktreePath, agentConfig.Subdirectory)

			// Determine bump type from change class (re-read message to classify)
			bumpType := "patch" // Default for Class A
			if task.MessageID != "" && d.msgStore != nil {
				if msg, err := d.msgStore.GetInboxMessage(task.MessageID); err == nil && msg != nil {
					env, _ := messaging.ExtractPackageEnvelope(msg)
					if env != nil {
						switch ClassifyChange(env) {
						case ChangeClassB:
							bumpType = "minor"
						case ChangeClassC:
							bumpType = "major"
						}
					}
				}
			}

			// Deterministic version bump in ailang.toml
			tomlPath := filepath.Join(publishDir, "ailang.toml")
			if tomlData, err := os.ReadFile(tomlPath); err == nil {
				manifest, _ := pkg.LoadManifest(publishDir)
				if manifest != nil {
					newVersion, bumpErr := pkg.BumpSemver(manifest.Package.Version, bumpType)
					if bumpErr == nil {
						// Replace version in ailang.toml
						oldLine := fmt.Sprintf(`version = "%s"`, manifest.Package.Version)
						newLine := fmt.Sprintf(`version = "%s"`, newVersion)
						updated := strings.Replace(string(tomlData), oldLine, newLine, 1)
						if err := os.WriteFile(tomlPath, []byte(updated), 0644); err == nil {
							d.logger.Printf("Deterministic version bump: %s → %s (%s) for %s",
								manifest.Package.Version, newVersion, bumpType, agentConfig.ID)
							// Commit the version bump
							commitCmd := exec.Command("git", "-C", worktreePath, "add", "-A")
							commitCmd.Run()
							commitMsg := fmt.Sprintf("chore: bump %s %s → %s (%s update)", manifest.Package.Name, manifest.Package.Version, newVersion, bumpType)
							exec.Command("git", "-C", worktreePath, "commit", "-m", commitMsg).Run()
						}
					} else {
						d.logger.Printf("Warning: Version bump failed for %s: %v", agentConfig.ID, bumpErr)
					}
				}
			}

			// Deterministic publish
			d.logger.Printf("Running deterministic publish for package agent %s in %s", agentConfig.ID, publishDir)
			publishCmd := exec.Command("ailang", "publish")
			publishCmd.Dir = publishDir
			publishOutput, publishErr := publishCmd.CombinedOutput()
			if publishErr != nil {
				d.logger.Printf("Warning: Deterministic publish failed for %s: %s\n%s", agentConfig.ID, publishErr, string(publishOutput))
			} else {
				d.logger.Printf("Deterministic publish succeeded for %s: %s", agentConfig.ID, strings.TrimSpace(string(publishOutput)))
			}
		}

		// M-SEMANTIC-ENVELOPE: Compute resolution envelope from git diff
		// Best-effort — failures don't affect task completion
		if worktreePath != "" && task.MessageID != "" && d.msgStore != nil {
			d.enrichResolutionEnvelope(taskCtx, task, worktreePath)
		}

		// Update task with worktree and agent info (needed for git diff artifact discovery)
		task.WorktreePath = worktreePath
		task.AgentID = targetAgent

		// Sync thread target_agent to match task agent_id (keeps hierarchy API in sync)
		if task.ThreadID != "" && d.msgStore != nil {
			if err := d.msgStore.SetThreadTargetAgent(task.ThreadID, targetAgent); err != nil {
				d.logger.Printf("Warning: Failed to sync thread target_agent: %v", err)
			}
		}

		if skipApproval {
			// Script agents or agents with skip_approval: mark as completed directly
			if err := d.taskStore.MarkTaskCompleted(taskCtx, task.ID, result); err != nil {
				d.logger.Printf("Warning: Failed to mark task completed: %v", err)
			}
			// M-CHAINS-SIMPLIFY: Update stage and chain status
			d.updateChainStageStatus(taskCtx, task, observatory.StageStatusCompleted)
			d.updateChainStatus(taskCtx, task, observatory.ChainStatusCompleted)
			d.updateStageSession(taskCtx, task, result.SessionID)
			// M-CHAINS-SOURCE-OF-TRUTH: Record cost/token metrics on stage and chain
			d.updateStageMetrics(taskCtx, task, result)
			d.updateChainMetrics(taskCtx, task, result)
			d.logger.Printf("Task %s completed (skip_approval=true, cost: $%.4f, tokens: %d)",
				task.ID, result.Cost, result.TokensUsed)
		} else {
			// Normal agents: mark as pending approval
			if err := d.taskStore.MarkTaskPendingApproval(taskCtx, task.ID, worktreePath, worktreeBranch, baseBranch, baseCommit, result); err != nil {
				d.logger.Printf("Warning: Failed to mark task pending approval: %v", err)
			}
			// M-CHAINS-SIMPLIFY: Update stage status to awaiting approval
			d.updateChainStageStatus(taskCtx, task, observatory.StageStatusAwaitingApproval)
			d.updateChainStatus(taskCtx, task, observatory.ChainStatusPendingApproval)
			d.updateStageSession(taskCtx, task, result.SessionID)
			// M-CHAINS-SOURCE-OF-TRUTH: Record cost/token metrics on stage and chain
			d.updateStageMetrics(taskCtx, task, result)
			d.updateChainMetrics(taskCtx, task, result)

			// M-COORD-GITHUB-AUTO-ROUTING: Process stage completion for GitHub-linked tasks
			// This posts the summary to GitHub and adds the appropriate approval label
			if err := d.ProcessStageCompletion(taskCtx, task, result); err != nil {
				d.logger.Printf("Warning: Failed to process stage completion: %v", err)
			}

			// Create approval request record for the CLI/dashboard to show
			approvalID := fmt.Sprintf("apr-%s", task.ID[5:]) // apr-<hash> from task-<hash>

			// Check if handoffs are pending for this agent (will be embedded in approval)
			var contextJSON string
			var handoffTargets []string
			if d.agentRegistry != nil && task.AgentID != "" {
				agent := d.agentRegistry.GetAgentByID(task.AgentID)
				if agent != nil && len(agent.TriggerOnComplete) > 0 && !agent.AutoApproveHandoffs {
					handoffTargets = agent.TriggerOnComplete
					// Build context with embedded handoff data
					handoffContext := map[string]interface{}{
						"handoff_targets": handoffTargets,
						"session_id":      result.SessionID,
						"source_agent":    task.AgentID,
					}
					if contextBytes, err := json.Marshal(handoffContext); err == nil {
						contextJSON = string(contextBytes)
					}
					d.logger.Printf("Embedding handoff to %v in merge approval for task %s", handoffTargets, task.ID)
				}
			}

			// Determine approval type based on whether handoff is embedded
			approvalType := string(ApprovalTypeMerge)
			description := fmt.Sprintf("Agent completed work on: %s", task.Title)
			if len(handoffTargets) > 0 {
				approvalType = "merge_handoff" // Combined type for UI clarity
				description = fmt.Sprintf("Agent completed work on: %s (will handoff to: %v)", task.Title, handoffTargets)
			}

			approvalReq := &ApprovalRequestRecord{
				ID:          approvalID,
				TaskID:      task.ID,
				Type:        approvalType,
				Description: description,
				ContextJSON: contextJSON,
				Status:      "pending",
				CreatedAt:   time.Now(),
			}
			if err := d.taskStore.CreateApprovalRequest(taskCtx, approvalReq); err != nil {
				d.logger.Printf("Warning: Failed to create approval request: %v", err)
			}

			// M-DASHBOARD-APPROVAL-INTEGRATION: Create inbox message for Event Queue visibility
			// This allows approvals to show up in the dashboard's event queue with pulsing animation
			if d.msgStore != nil {
				approvalMsg := &messaging.InboxMessage{
					FromAgent:     "coordinator",
					ToInbox:       "approvals",
					MessageType:   "approval_request",
					Title:         fmt.Sprintf("Approval Required: %s", task.Title),
					Payload:       fmt.Sprintf("**Approval Required**\n\nTask: %s\nAgent: %s\n\nReview and approve via dashboard or CLI:\n`ailang coordinator approve %s`", task.Title, task.AgentID, approvalID),
					CorrelationID: task.ID,
					Status:        messaging.InboxStatusUnread,
				}
				if msgErr := d.msgStore.InsertInboxMessage(approvalMsg); msgErr != nil {
					d.logger.Printf("Warning: Failed to create approval inbox message: %v", msgErr)
				}
			}

			d.logger.Printf("Task %s awaiting approval (cost: $%.4f, tokens: %d, worktree: %s)",
				task.ID, result.Cost, result.TokensUsed, worktreePath)
		}

		// Check for agent-to-agent handoffs
		// Find the agent that handled this task and check for trigger_on_complete
		if err := d.handleAgentHandoffs(task, result); err != nil {
			d.logger.Printf("Warning: Failed to process agent handoffs: %v", err)
		}
	} else {
		// Failed tasks: cleanup worktree (no useful work to preserve)
		if worktree != nil && d.worktreeMgr != nil {
			if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
				d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
			}
		}
		if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, fmt.Errorf("%s", result.Error)); err != nil {
			d.logger.Printf("Warning: Failed to mark task failed: %v", err)
		}
		// M-CHAINS-SIMPLIFY: Update stage and chain status
		d.updateChainStageStatus(taskCtx, task, observatory.StageStatusFailed)
		d.updateChainStatus(taskCtx, task, observatory.ChainStatusFailed)
		// Capture partial metrics, session, and error even on failure (v0.8.1)
		d.updateStageSession(taskCtx, task, result.SessionID)
		d.updateStageMetrics(taskCtx, task, result)
		d.updateChainMetrics(taskCtx, task, result)
		d.updateStageError(taskCtx, task, result.Error)
		d.logger.Printf("Task %s failed: %s", task.ID, result.Error)
		span.SetStatus(codes.Error, result.Error)
	}

	// Set span status for successful tasks
	if result.Success {
		span.SetStatus(codes.Ok, "task completed successfully")
	}

	// Post result to thread
	d.postTaskResult(task, result, nil)

	return nil
}

// enrichResolutionEnvelope computes a resolution embedding from the git diff
// and updates the original message's envelope. Best-effort: failures are logged
// but don't affect task completion.
func (d *Daemon) enrichResolutionEnvelope(ctx context.Context, task *TaskRecord, worktreePath string) {
	// Get git diff (HEAD~1..HEAD)
	diffCmd := exec.Command("git", "diff", "HEAD~1..HEAD", "--stat")
	diffCmd.Dir = worktreePath
	diffOutput, err := diffCmd.Output()
	if err != nil {
		d.logger.Printf("[envelope] No git diff available for task %s: %v", task.ID, err)
		return
	}

	// Get commit message
	logCmd := exec.Command("git", "log", "-1", "--pretty=%B")
	logCmd.Dir = worktreePath
	commitMsg, err := logCmd.Output()
	if err != nil {
		d.logger.Printf("[envelope] No commit message available for task %s: %v", task.ID, err)
		return
	}

	commitMsgStr := strings.TrimSpace(string(commitMsg))
	diffStr := strings.TrimSpace(string(diffOutput))
	if commitMsgStr == "" && diffStr == "" {
		return
	}

	// Create embedder from config
	cfg := messaging.LoadEmbedConfigFromEnv()
	embedder, err := messaging.NewEmbedderFromConfig(cfg)
	if err != nil || embedder == nil {
		d.logger.Printf("[envelope] No embedder available for resolution: %v", err)
		return
	}

	// Build resolution envelope
	env, err := messaging.NewEnvelopeBuilder(embedder).
		WithResolution(commitMsgStr, diffStr).
		Build(&messaging.InboxMessage{Title: task.Title, Payload: commitMsgStr})
	if err != nil {
		d.logger.Printf("[envelope] Failed to build resolution envelope for task %s: %v", task.ID, err)
		return
	}

	// Extract just the resolution slot to update
	resOnly := messaging.NewEnvelope()
	if resVec := env.Get(messaging.SlotResolution); resVec != nil {
		resOnly.Slots[messaging.SlotResolution] = resVec
	}

	// Update the original message's envelope (non-overwrite to preserve existing slots)
	if err := d.msgStore.UpdateMessageEnvelope(task.MessageID, resOnly, false); err != nil {
		d.logger.Printf("[envelope] Failed to update message envelope for task %s: %v", task.ID, err)
		return
	}

	d.logger.Printf("[envelope] Resolution envelope added to message %s for task %s", task.MessageID, task.ID)
}
