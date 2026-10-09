package main

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	// Import to trigger init() registration — same as local coordinator (provider_executor.go)
	_ "github.com/sunholo-data/ailang/internal/executor/claude"
	_ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// coordinatorExecuteJob is the entry point for Cloud Run Jobs.
// It reads task configuration from environment variables, executes the task
// using an AI executor (Claude — Gemini CLI was retired in v0.22.0
// per M-MANAGED-AGENTS), and publishes completion to Pub/Sub.
//
// RELIABILITY: This function guarantees a completion message is always published.
// A defer guard with recover() catches panics and early exits. For failures before
// Pub/Sub is initialized, structured logs go to stderr for Cloud Logging.
//
// The executor uses the same infrastructure as the local coordinator (via
// executor.GlobalFactory) for full parity: stream-JSON parsing, token extraction,
// OTEL spans, session tracking, idle/hard timeouts, and cost calculation.
//
// Required environment variables:
//
//	AILANG_TASK_ID      - Task ID to execute
//	AILANG_AGENT_ID     - Agent ID (e.g., "sprint-executor")
//	AILANG_WORKSPACE    - Workspace identifier
//	AILANG_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT - GCP project for Pub/Sub
//
// Optional:
//
//	AILANG_PROVIDER      - Executor provider (default: "claude")
//	AILANG_REPO_URL      - Git repository URL to clone
//	AILANG_BRANCH        - Base branch to clone (default: "dev")
//	AILANG_PUSH_BRANCH   - Push directly to this branch (skip coordinator/ branch creation)
//	AILANG_DIRECTIVE     - Task directive/prompt
//	AILANG_TASK_TITLE    - Human task title, used for PR/commit subjects
//	AILANG_TOPIC_PREFIX  - Topic prefix (default: "ailang")
//	AILANG_PLUGIN_REPO   - Git URL for shared skills plugin (cloned as --plugin-dir)
//	AILANG_MAX_COST_USD  - Per-task cost budget in USD (0 = unlimited) from budget config
//	AILANG_MODEL         - AI model override (e.g., "sonnet", "opus") from agent config
//
// executeJobWorkspace resolves the workspace a job's completion is published
// under. A silent "default" filed a mis-dispatched job's result where nothing
// listened, so it is a deprecated default under D3 (M-V1-SIMPLIFY-S4 M1):
// served with one stderr warning, refused under AILANG_STRICT_CONFIG=1.
func executeJobWorkspace() (string, error) {
	if ws := config.Workspace(); ws != "" {
		return ws, nil
	}
	return config.DeprecatedDefault(coordinator.EnvWorkspace, coordinator.DeprecatedWorkspaceDefault)
}

func coordinatorExecuteJob(args []string) (returnErr error) {
	// Parse flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printExecuteJobHelp()
			return nil
		}
	}

	// Read ALL environment variables upfront (before any early returns).
	taskID := config.TaskID()
	agentID := config.AgentID()
	// Refused before Pub/Sub exists, because there is no workspace to publish
	// a failure to.
	workspace, wsErr := executeJobWorkspace()
	if wsErr != nil {
		fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|error=%v\n", taskID, agentID, wsErr)
		return wsErr
	}
	// The container's project: the one resolver, which on Cloud Run also has
	// the metadata server. Unresolvable is reported below, once
	// publishCompletion exists — it is "" here so the guard can still say so.
	projectID, projErr := config.CloudProject(context.Background())
	// Resolved (and verified) below, once publishCompletion exists to report a
	// bad answer. Deliberately not defaulted here — see resolveContainerProvider.
	requestedProvider := config.Provider()
	imageProvider := config.ImageProvider()
	var provider string
	repoURL := config.RepoURL()
	branch := config.Branch()
	// The directive is resolved below, once publishCompletion exists to report
	// a failure to read it (resolveJobDirective).
	var directive string
	// AILANG_TASK_TITLE is read where it is used (taskSubject); named here so
	// the env contract above stays the complete list.
	prefix := pubsub.TopicPrefixFromEnv()

	// Initialize Pub/Sub client as early as possible so the defer guard can use it.
	// If Pub/Sub init itself fails, we fall back to stderr logging.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	var publisher *pubsub.Publisher
	var completionSent atomic.Bool

	if projectID != "" {
		client, err := pubsub.NewClient(ctx, projectID, prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|error=pubsub_init: %v\n",
				taskID, agentID, err)
		} else {
			defer client.Close()
			publisher = pubsub.NewPublisher(client)
			defer publisher.Stop()
		}
	}

	// publishCompletion is a closure that publishes (or logs to stderr as fallback).
	// The optional execResult carries metrics from the executor for parity with local.
	// changedFiles lists files created/modified by the agent (discovered via git diff).
	// artifactPath is the GCS path prefix where raw artifacts were uploaded (may be empty).
	publishCompletion := func(status, errMsg, branchName string, execResult *executor.Result, ev gitEvidence, artifactPath string) {
		if completionSent.Swap(true) {
			return // Already sent — prevent double-publish.
		}
		status, errMsg = contractCompletionStatus(status, errMsg)
		completion := buildCloudCompletion(taskID, agentID, status, errMsg, branchName, execResult, ev, artifactPath)
		if publisher != nil {
			pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if pubErr := publisher.PublishCompletion(pubCtx, completion, workspace); pubErr != nil {
				fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|status=%s|error=publish: %v\n",
					taskID, agentID, status, pubErr)
			} else {
				fmt.Printf("execute-job: completion published for task %s (status=%s, turns=%d, tokens=%d+%d, cost=$%.4f)\n",
					taskID, status, completion.NumTurns, completion.InputTokens, completion.OutputTokens, completion.CostUSD)
			}
		} else {
			// No Pub/Sub available — structured stderr log for Cloud Logging.
			fmt.Fprintf(os.Stderr, "COMPLETION_FAILED|task=%s|agent=%s|status=%s|error=%s\n",
				taskID, agentID, status, errMsg)
		}
	}

	// Defer guard: catches panics and any exit path that forgot to publish.
	defer func() {
		if r := recover(); r != nil {
			returnErr = fmt.Errorf("panic: %v", r)
			publishCompletion("failed", returnErr.Error(), "", nil, gitEvidence{}, "")
		} else if !completionSent.Load() {
			// Should not happen — means we returned without publishing.
			publishCompletion("failed", "unknown: exited without publishing completion", "", nil, gitEvidence{}, "")
		}
	}()

	// Validate required env vars (after Pub/Sub init so failures are reported).
	if taskID == "" {
		publishCompletion("failed", "AILANG_TASK_ID environment variable is required", "", nil, gitEvidence{}, "")
		return fmt.Errorf("AILANG_TASK_ID environment variable is required")
	}
	if agentID == "" {
		publishCompletion("failed", "AILANG_AGENT_ID environment variable is required", "", nil, gitEvidence{}, "")
		return fmt.Errorf("AILANG_AGENT_ID environment variable is required")
	}
	if projErr != nil {
		publishCompletion("failed", projErr.Error(), "", nil, gitEvidence{}, "")
		return projErr
	}
	var dirErr error
	directive, dirErr = resolveJobDirective(ctx, config.DirectiveSource(), taskID, openDirectiveStore(projectID))
	if dirErr != nil {
		publishCompletion("failed", dirErr.Error(), "", nil, gitEvidence{}, "")
		return dirErr
	}

	// Settle which executor runs, and prove it can, BEFORE cloning the repo.
	// Three prod runs on 2026-08-27/28 cloned 24,032 files and only then failed
	// on a binary the image never installed.
	var provErr error
	if provider, provErr = resolveContainerProvider(requestedProvider, imageProvider); provErr != nil {
		publishCompletion("failed", provErr.Error(), "", nil, gitEvidence{}, "")
		return provErr
	}
	codexCred, credErr := installCodexCredential(ctx, provider, projectID)
	if credErr != nil {
		publishCompletion("failed", credErr.Error(), "", nil, gitEvidence{}, "")
		return credErr
	}
	// Write refreshed subscription tokens back however the task ends; a container
	// is discarded after one task, so an unpersisted refresh is lost.
	if codexCred != nil {
		ctx = codexCred.ctx
		defer func() {
			if r := recover(); r != nil {
				codexCred.quarantine()
				panic(r)
			}
			if err := codexCred.finish(); err != nil {
				fmt.Fprintln(os.Stderr, "execute-job: credential finalization failed:", err)
				if returnErr == nil {
					returnErr = err
				}
			}
		}()
	}
	if err := preflightExecutor(ctx, provider); err != nil {
		publishCompletion("failed", err.Error(), "", nil, gitEvidence{}, "")
		return err
	}

	// Read plugin repo for shared skills (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	pluginRepo := config.PluginRepo()

	// Read model override from agent config (passed via AILANG_MODEL env var).
	// The executor has NO default model (M-MODEL-REGISTRY-SINGLE-SOURCE M6,
	// D2(a)): an empty value fails at the point of use rather than silently
	// running "haiku", which is too weak for coding tasks.
	model := config.Model()

	// Read timeout from agent config (passed via AILANG_TIMEOUT env var, M-CLOUD-OAUTH).
	// Without this, the executor defaults to 5m which is too short for complex tasks.
	timeoutStr := config.Timeout()
	if timeoutStr == "" {
		// M-COORDINATOR-EXECUTION-TRUST M8: was 30m. Cloud Run Jobs allow 24h and
		// were chosen for that; idle_timeout is the liveness guard, so a generous
		// wall-clock costs nothing while an agent is making progress.
		timeoutStr = coordinator.DefaultTaskTimeout.String()
	}

	// idle_timeout is printed because its ABSENCE was invisible for months: the
	// start line named the wall-clock ceiling only, so a 3m idle kill under a
	// declared 5m read as the model stalling rather than as config not arriving.
	idleStr := config.IdleTimeout()
	if idleStr == "" {
		idleStr = "executor default"
	}
	fmt.Printf("execute-job: starting task %s (agent=%s, workspace=%s, model=%s, timeout=%s, idle_timeout=%s)\n", taskID, agentID, workspace, model, timeoutStr, idleStr)

	// Execute the task
	branchName, execResult, evidence, execErr := executeCloudTask(ctx, taskID, agentID, repoURL, branch, directive, provider, pluginRepo, model, timeoutStr)

	// Finalize rotating credentials before reporting success. Failed executor
	// tasks may also have refreshed; persist those before releasing ownership.
	execErr = finalizeCodexCredential(codexCred, execErr)

	// Write artifact files to the GCS-mounted directory (/artifacts/tasks/{taskID}/).
	// The artifact bucket is mounted read-write at /artifacts via Cloud Run volume mount.
	// Files written here land directly in GCS — no upload client needed.
	artifactPath := writeTaskArtifacts(taskID, execResult)

	// Publish completion with executor metrics (success or failure)
	if execErr != nil {
		publishCompletion("failed", execErr.Error(), branchName, execResult, evidence, artifactPath)
		fmt.Printf("execute-job: task %s failed: %v\n", taskID, execErr)
	} else {
		// M-COORDINATOR-EXECUTION-TRUST M2: "the executor exited 0" is not
		// "work landed". A run that was expected to produce a diff and produced
		// none reports no_changes — a terminal status that an old consumer reads
		// as not-success, which is the safe direction.
		//
		// expectChanges is trusted dispatch metadata (AILANG_EXPECT_CHANGES,
		// from the agent registry), NOT the content-derived task type: that
		// classifier is a substring match over sender-controlled message text
		// (design doc V18), and M2's first draft inherited exactly the hole M1a
		// was rewritten to remove.
		//
		// A pushed branch implies commits, and commits imply discovered files,
		// so changedFiles is the load-bearing signal here; branchPushed is
		// passed for callers that can distinguish the two.
		// Only an explicit "true" declares acknowledge-only. Unset, empty or
		// malformed all mean "changes were expected", so a misconfigured or
		// older dispatcher fails LOUD rather than silently lenient.
		expectChanges := !config.AcknowledgeOnly()
		status := coordinator.ClassifyCompletionStatus(evidence.ChangedFiles, false, expectChanges)

		// The agent's own word on the outcome, which it has never had.
		//
		// A BLOCKED: marker means it did not attempt the work — a precondition
		// was unmet — and that is a different fact from `no_changes`, which also
		// means "I did the work and nothing needed changing". One of those needs
		// a human and the other needs nobody, and they were indistinguishable.
		//
		// Only honoured when the run produced NOTHING. An agent that declared a
		// blocker and then changed files worked around it, and the files are the
		// stronger evidence — trusting the marker over them would discard real
		// work on the strength of a sentence.
		blockedMsg := ""
		if len(evidence.ChangedFiles) == 0 && execResult != nil {
			if report, ok := coordinator.ParseBlockedMarker(execResult.Transcript); ok {
				status = coordinator.TaskStatusBlocked
				blockedMsg = report.Reason
				if report.On != "" {
					blockedMsg += " (blocked on: " + report.On + ")"
				}
				fmt.Printf("execute-job: task %s BLOCKED: %s\n", taskID, blockedMsg)
			}
		}
		publishCompletion(string(status), blockedMsg, branchName, execResult, evidence, artifactPath)
		fmt.Printf("execute-job: task %s %s (branch=%s, files=%d)\n", taskID, status, branchName, len(evidence.ChangedFiles))
	}

	return execErr
}
