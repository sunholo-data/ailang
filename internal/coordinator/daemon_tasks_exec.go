package coordinator

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
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
	return config.RepoURL()
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

	// The daemon may be shutting down: Close() releases the task store and sets
	// the field to nil (daemon_lifecycle.go), and a tick already in flight would
	// otherwise dereference it and take the process down with a nil-pointer
	// panic. Reported as an error rather than a quiet `return nil`, because
	// "the store is gone" and "there is nothing to do" are different facts and
	// only one of them is normal.
	if d.taskStore == nil {
		return fmt.Errorf("task store unavailable (daemon shutting down?)")
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
		// A shared task store holds EVERY coordinator's tasks. This worker must
		// only execute tasks for agents it actually serves — measured 2026-08-27:
		// the rig claimed a pending pkg-sunholo-ailang-parse task (a cloud-lane
		// agent absent from its registry), fell through to the default
		// "coordinator" worktree manager and the legacy provider path, and ran a
		// user's feedback task under codex in the wrong repo. This is the
		// queue-side mirror of the cloud dispatcher's local-lane skip (M3) and
		// the inbox refusal (resolveInboxAgent): work you cannot own, you do not
		// touch. Empty-agent tasks keep legacy default handling — they exist
		// only in single-machine setups.
		if task.AgentID != "" && d.agentRegistry != nil && d.agentRegistry.GetAgentByID(task.AgentID) == nil {
			d.logger.Printf("Skipping task %s: agent %q is not served by this coordinator (left pending for its owner)", task.ID, task.AgentID)
			continue
		}
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
		// CLAIM the task. This is the only thing standing between a task and
		// two Cloud Run executions of it: the listing above is not exclusive, so
		// concurrent dispatchers (two instances, or two ticks over a slow list)
		// see the same pending task and all reach here.
		if err := d.taskStore.MarkTaskQueued(d.ctx, task.ID); err != nil {
			if errors.Is(err, ErrTaskNotClaimable) {
				// Normal under concurrency: another dispatcher owns it. Logged
				// because "we raced and lost" is a fact worth seeing in the log
				// when duplicate work is being investigated — silence here is
				// what made the 2026-09-17 triple dispatch hard to attribute.
				d.logger.Printf("Dispatch: task %s already claimed by another dispatcher, skipping", task.ID)
			} else {
				d.logger.Printf("Failed to mark task %s as queued: %v", task.ID, err)
			}
			continue
		}

		// Determine the provider: the AGENT's own declaration wins over the
		// coordinator's default.
		//
		// This used to read the global default only, which made every agent's
		// `provider:` field decorative. Measured on the dev plane 2026-09-03:
		// sprint-executor declares provider "codex" with executor_variant
		// "codex-go", the coordinator handed the dispatcher "pi" (the plane
		// default), and checkVariantProviderAgreement correctly refused —
		// "executor_variant \"codex-go\" runs image agent-codex-go, which has
		// [codex] on $PATH, but provider is \"pi\"". The task retried every five
		// minutes and never ran.
		//
		// Three of 34 agents declare a provider the plane default contradicts:
		// sprint-planner (codex), sprint-executor (codex) and motoko (motoko).
		// Two of those are the middle stages of the four-stage pipeline, so the
		// pipeline could not have completed even once, whatever else was fixed.
		provider := resolveDispatchProvider(d.agentRegistry, task.AgentID, d.coordConfig)

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
			var agentCfg *AgentConfig
			if d.agentRegistry != nil {
				if agent := d.agentRegistry.GetAgentByID(task.AgentID); agent != nil {
					agentCfg = agent
					directive = BuildDirectiveFromConfig(task, agent)
				}
			}

			// M-MESSAGE-PLANE-FAIL-LOUD M3 (D3): a LOCAL-lane agent must never be
			// cloud-dispatched. Measured 2026-08-26: 10 consecutive Cloud Run jobs
			// died on arrival for agent=eval-rig because the job received a Mac
			// Studio filesystem path as its clone target — and even with a valid
			// coordinate the lane is wrong, since the rig's whole purpose is local
			// GPU and ollama models a Cloud Run job does not have.
			if agentCfg.ResolveLane() == LaneLocal {
				d.logger.Printf("Task %s: agent %q is execution_lane=local; leaving it for its bare-metal worker instead of dispatching to Cloud Run", task.ID, task.AgentID)
				continue
			}

			// Prefer the agent's resolved coordinate over the task's workspace
			// string, so `workspace` stops doubling as a repo coordinate.
			repoURL := deriveRepoURL(task.Workspace)
			if repo := agentCfg.ResolveRepo(); repo != "" {
				repoURL = deriveRepoURL(repo)
			}
			params := DispatchParams{
				TaskID:    task.ID,
				AgentID:   task.AgentID,
				Workspace: task.Workspace,
				Provider:  provider,
				Directive: directive,
				TaskTitle: task.Title,
				RepoURL:   repoURL,
				Branch:    task.BaseBranch, // From task record, defaults handled by job
				// M-PKG-CASCADE-DETERMINISTIC-FIRST: propagate cascade envelope so the
				// Cloud Run Job wrapper can decide deterministic-bump vs AI-escalation.
				RootPackage:       task.RootPackage,
				RootChangeClass:   task.RootChangeClass,
				FromVersion:       task.FromVersion,
				ToVersion:         task.ToVersion,
				FromInterfaceHash: task.FromInterfaceHash,
				ToInterfaceHash:   task.ToInterfaceHash,
				EffectsWidened:    task.EffectsWidened,
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
				// M-COORDINATOR-EXECUTION-TRUST M1a: resolve the permission tier
				// from the TRUSTED registry entry, after PushBranch is final —
				// ResolveWorkTier refuses tier 1 on a direct-push dispatch, which
				// has no PR containment (design doc V24).
				params.WorkTier = string(ResolveWorkTier(agent, params.PushBranch))
				params.AcknowledgeOnly = agent.AcknowledgeOnly
				// M5: resolve through the shared routing table (pin > role >
				// default); a missing role is loud and skips the dispatch.
				agentModel, mErr := ResolveModel(agent)
				if mErr != nil {
					d.logger.Printf("ERROR: task %s not dispatched: %v", task.ID, mErr)
					continue
				}
				if agentModel != "" {
					params.Model = agentModel
				}
				if agent.Timeout != "" {
					params.Timeout = agent.Timeout
				}
				// Without this the job falls back to the executor default (3m),
				// which reads as the agent stalling rather than as config that
				// never arrived. Deliberative agents notice first: they emit few,
				// long turns, where a tool-heavy agent resets the idle timer
				// constantly. sprint-evaluator died on "pi idle for 3m0.004s"
				// six times on 2026-09-22 while declaring idle_timeout: 5m.
				if agent.IdleTimeout != "" {
					params.IdleTimeout = agent.IdleTimeout
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
				// Whether GitHub may merge this agent's PR on green. Registry
				// metadata, so a message cannot ask for its own auto-merge.
				params.AutoMerge = agent.AutoMerge
				// DECLARED patterns, not GetEffectiveArtifactPatterns(): that
				// falls back to `**/*` for any unrecognised agent
				// (DefaultArtifactPatterns), which is the opposite of a bound. The
				// auto-merge guard treats an empty list as "nothing constrains
				// this, refuse" — so an agent that never declared its artifacts
				// gets no auto-merge instead of unlimited scope.
				params.ArtifactPatterns = agent.ArtifactPatterns
				params.SSHKeySecret = agent.SSHKeySecret
				params.SSHHostAlias = agent.SSHHostAlias
				params.ToolPolicy = agent.GetEffectiveToolPolicy()
				if agent.PolicyPath != "" {
					toml, rerr := os.ReadFile(agent.PolicyPath)
					if rerr != nil {
						// Loud, and skip THIS dispatch: an ailang_only agent sent
						// without its policy would run with ailang_run refusing
						// everything and read as a model failure.
						d.logger.Printf("ERROR: task %s not dispatched: agent %s policy_path %s: %v", task.ID, agent.ID, agent.PolicyPath, rerr)
						continue
					}
					params.PolicyTOML = string(toml)
				}
				// Same refusal when the policy was never CONFIGURED. An unreadable
				// path already fails closed; an absent one used to dispatch a job
				// whose ailang_run denies every program by default — the model then
				// spends its turns discovering it cannot execute anything, and the
				// run banks as a model failure.
				//
				// This is load-bearing now that a `pkg:` inbox DEFAULTS to the lane
				// (GetEffectiveToolPolicy): a package agent added without a
				// policy_path would inherit the restriction and not the authority.
				if params.ToolPolicy != executor.ToolProfileFull && params.PolicyTOML == "" {
					d.logger.Printf("ERROR: task %s not dispatched: agent %s runs the %q lane with no policy_path, so ailang_run would refuse every program. Add policy_path (the package lane uses /etc/ailang-config/policies/pkg-ailang-only.toml), or declare tool_policy: full if this agent really needs a shell.",
						task.ID, agent.ID, params.ToolPolicy)
					continue
				}
				// The lane's boundary claim needs a restricted policy underneath
				// it (M-EXECUTOR-POLICY-HARDENING D3); a policy that does not
				// resolve is refused here, before a job is paid for.
				if lerr := executor.CheckLanePolicy(params.ToolPolicy, []byte(params.PolicyTOML)); lerr != nil {
					d.logger.Printf("ERROR: task %s not dispatched: agent %s: %v", task.ID, agent.ID, lerr)
					continue
				}
				if agent.GitIdentity != nil {
					params.GitAuthorName = agent.GitIdentity.Name
					params.GitAuthorEmail = agent.GitIdentity.Email
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
