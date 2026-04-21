package coordinator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// inboxMessage pairs a message with the inbox and agent it came from.
type inboxMessage struct {
	inbox   string
	agentID string
	msg     *Message // coordinator.Message type
}

// pollAndProcessTasks polls for new messages and queues them as tasks.
// M-CLOUD-E2E: In cloud mode, pulls from PubSub adapter and routes by message Inbox attribute.
func (d *Daemon) pollAndProcessTasks() error {
	// Cloud mode: pull from single PubSub adapter, route by Inbox attribute
	if d.cloudInboxAdapter != nil {
		return d.pollAndProcessTasksCloud()
	}

	// Local mode: poll per-inbox SQLite adapters
	var allMessages []inboxMessage

	for inbox, adapter := range d.inboxAdapters {
		messages, err := adapter.ListUnread()
		if err != nil {
			d.logger.Printf("Warning: Failed to list messages from inbox %q: %v", inbox, err)
			continue
		}
		agent := d.agentRegistry.GetAgentForInbox(inbox)
		agentID := ""
		if agent != nil {
			agentID = agent.ID
		}
		for _, msg := range messages {
			allMessages = append(allMessages, inboxMessage{
				inbox:   inbox,
				agentID: agentID,
				msg:     msg,
			})
		}
	}

	if len(allMessages) == 0 {
		return nil
	}

	d.logger.Printf("Found %d unread messages across %d inboxes", len(allMessages), len(d.inboxAdapters))

	for _, im := range allMessages {
		msg := im.msg
		agentID := im.agentID

		// Create a Task for the analyzer
		taskID := fmt.Sprintf("task-%s", msg.ID[:8])

		// Determine kind - use message kind if set, otherwise infer from type
		kind := msg.Kind
		if kind == "" {
			if msg.Type == "question" || msg.Type == "research" {
				kind = "question"
			} else {
				kind = "directive"
			}
		}

		taskInput := &Task{
			ID:        taskID,
			Title:     msg.Title,
			Content:   msg.Content,
			Kind:      kind,
			MessageID: msg.ID,
			CreatedAt: msg.CreatedAt,
		}

		// Analyze the message to classify it
		analyzed := d.analyzer.Analyze(taskInput)

		// Create a task record
		// Get the agent's workspace path from the registry
		workspace := ""
		if agent := d.agentRegistry.GetAgentByID(agentID); agent != nil && agent.Workspace != "" {
			workspace = agent.Workspace
		} else {
			// Fallback to current directory - warn as skills may not be available
			workspace, _ = os.Getwd()
			d.logger.Printf("WARNING: Agent %q has no workspace configured, using current directory: %s (skills may not be available)", agentID, workspace)
		}

		// Set iteration: use message iteration if provided, otherwise default to 1
		iteration := msg.Iteration
		if iteration == 0 {
			iteration = 1 // First run is iteration 1
		}

		// M-HARNESS-COMMIT-CONTRACT: Extract siteSlug and briefId from message content.
		// Messages from website-builder may contain JSON with these fields.
		var siteSlug, briefID string
		var payloadFields struct {
			SiteSlug string `json:"siteSlug"`
			BriefID  string `json:"briefId"`
		}
		if json.Unmarshal([]byte(msg.Content), &payloadFields) == nil {
			siteSlug = payloadFields.SiteSlug
			briefID = payloadFields.BriefID
		}

		task := &TaskRecord{
			ID:            taskID,
			MessageID:     msg.ID,
			AgentID:       agentID,          // M-COORD-ARTIFACT-DISCOVERY: Set AgentID from inbox
			ParentTaskID:  msg.ParentTaskID, // M-TASK-HIERARCHY: Link to parent task for handoff chains
			Iteration:     iteration,        // M-TASK-HIERARCHY: Iteration number for feedback loops
			Title:         msg.Title,
			Content:       msg.Content,
			Type:          analyzed.Type,
			Kind:          kind,
			Priority:      CalculatePriority(analyzed),
			Status:        TaskStatusPending,
			Workspace:     workspace,
			GithubIssue:   msg.GithubIssue, // M-COORD-GITHUB-AUTO-ROUTING
			GithubRepo:    msg.GithubRepo,  // M-COORD-GITHUB-CLOSE-ON-MERGE
			CreatedAt:     msg.CreatedAt,
			Capabilities:  analyzed.Capabilities,  // M-DEPRECATE-AILANG-AGENT
			ImpactLevel:   analyzed.ImpactLevel,   // M-DEPRECATE-AILANG-AGENT
			EstimatedCost: analyzed.EstimatedCost, // M-DEPRECATE-AILANG-AGENT
			SiteSlug:      siteSlug,               // M-HARNESS-COMMIT-CONTRACT
			BriefID:       briefID,                // M-HARNESS-COMMIT-CONTRACT
		}

		// Check for duplicates
		fingerprint := analyzed.Fingerprint
		if fingerprint != 0 {
			if dup, _ := d.taskStore.FindDuplicateTask(d.ctx, fingerprint, 0.9); dup != nil {
				d.logger.Printf("Skipping duplicate task for message %s (similar to task %s)", msg.ID, dup.ID)
				// Mark message as read since we're skipping it
				if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
					_ = adapter.MarkAsRead(msg.ID)
				}
				d.publishDedupCompletion(taskID, agentID, dup.ID)
				continue
			}
		}

		// Get or create a thread in collaboration.db for dashboard visibility
		// Using GetOrCreate prevents duplicate threads when message isn't marked read properly
		targetAgent := agentID
		if targetAgent == "" {
			targetAgent = "coordinator"
		}
		thread, created, err := d.msgStore.GetOrCreateThreadWithWorkspace(
			msg.Title,         // title
			"ailang_instance", // createdByType (constraint: 'human' or 'ailang_instance')
			"coordinator",     // createdByID
			targetAgent,       // targetAgent - the agent that will handle this task
			workspace,         // workspace - source project/agent
		)
		if err != nil {
			d.logger.Printf("Failed to get/create thread for task %s: %v", taskID, err)
			// Continue anyway - thread is for visibility, not required for task
		} else {
			task.ThreadID = thread.ID
			if created {
				d.logger.Printf("Created thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
			} else {
				d.logger.Printf("Reusing existing thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
			}
		}

		// Store the task
		if err := d.taskStore.CreateTask(d.ctx, task); err != nil {
			d.logger.Printf("Failed to create task for message %s: %v", msg.ID, err)
			// Still mark message as read even if task creation fails (e.g., duplicate task)
			// This prevents infinite loops when the same message keeps being processed
			if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
				if markErr := adapter.MarkAsRead(msg.ID); markErr != nil {
					d.logger.Printf("Failed to mark message as read after task error: %v", markErr)
				} else {
					d.logger.Printf("Marked message %s as read (task already existed)", msg.ID)
				}
			}
			continue
		}

		// Set fingerprint for deduplication
		if fingerprint != 0 {
			_ = d.taskStore.SetTaskFingerprint(d.ctx, task.ID, fingerprint)
		}

		// M-CHAINS-SIMPLIFY: Create execution chain and stage for unified hierarchy tracking
		if d.obsBackend != nil {
			chainID := msg.ChainID // May be set from handoff
			var stageID string

			// Determine chain ID: from message (handoff), from parent task, or create new
			if chainID == "" && task.ParentTaskID != "" {
				// Look up parent task to get its chain
				if parentTask, err := d.taskStore.GetTask(d.ctx, task.ParentTaskID); err == nil && parentTask != nil {
					chainID = parentTask.ChainID
				}
			}

			// If still no chain, create a new one
			if chainID == "" {
				sourceType := observatory.ChainSourceMessage
				if task.GithubIssue > 0 {
					sourceType = observatory.ChainSourceGitHubIssue
				}
				chain, err := d.obsBackend.CreateChain(d.ctx, &observatory.ChainCreateRequest{
					SourceType:        sourceType,
					SourceRef:         msg.ID,
					GitHubRepo:        task.GithubRepo,
					GitHubIssueNumber: task.GithubIssue,
				})
				if err != nil {
					d.logger.Printf("Warning: Failed to create execution chain for task %s: %v", task.ID, err)
				} else {
					chainID = chain.ID
					d.logger.Printf("Created execution chain %s for task %s", chainID, task.ID)
				}
			}

			// Create a stage for this agent
			if chainID != "" {
				stage, err := d.obsBackend.CreateStage(d.ctx, &observatory.StageCreateRequest{
					ChainID:   chainID,
					AgentID:   agentID,
					MessageID: msg.ID,
					TaskID:    task.ID,
				})
				if err != nil {
					d.logger.Printf("Warning: Failed to create chain stage for task %s: %v", task.ID, err)
				} else {
					stageID = stage.ID
					d.logger.Printf("Created chain stage %s (agent: %s) in chain %s", stageID, agentID, chainID)
				}
			}

			// Store chain context in task
			if chainID != "" || stageID != "" {
				task.ChainID = chainID
				task.StageID = stageID
				if err := d.taskStore.UpdateTaskChainInfo(d.ctx, task.ID, chainID, stageID); err != nil {
					d.logger.Printf("Warning: Failed to update chain info for task %s: %v", task.ID, err)
				}
			}
		}

		// M-COORD-GITHUB-AUTO-ROUTING: Initialize GitHub-linked tasks
		if task.GithubIssue > 0 && d.taskChain != nil {
			// Start the task chain (posts "working" comment to GitHub)
			if err := d.taskChain.StartTask(d.ctx, task.ID, task.GithubIssue); err != nil {
				d.logger.Printf("Warning: Failed to start task chain for issue #%d: %v", task.GithubIssue, err)
			} else {
				d.logger.Printf("Started GitHub pipeline for task %s (issue #%d)", task.ID, task.GithubIssue)
			}

			// Start watching for approval labels
			if d.approvalWatcher != nil {
				d.approvalWatcher.WatchIssue(task.GithubIssue, task.ID)
			}
		}

		// Log capability detection results
		capStr := "none"
		if len(analyzed.Capabilities) > 0 {
			capTypes := make([]string, len(analyzed.Capabilities))
			for i, cap := range analyzed.Capabilities {
				capTypes[i] = string(cap.Type)
			}
			capStr = fmt.Sprintf("%v", capTypes)
		}
		d.logger.Printf("Created task %s (type: %s, priority: %d, impact: %s, caps: %s, est_cost: $%.2f, agent: %s, issue: #%d) from message %s",
			task.ID, task.Type, task.Priority, task.ImpactLevel, capStr, task.EstimatedCost, agentID, task.GithubIssue, msg.ID)

		// Mark message as read using the correct inbox adapter
		if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
			if err := adapter.MarkAsRead(msg.ID); err != nil {
				d.logger.Printf("Failed to mark message as read: %v", err)
			}
		}

		d.tasksRun++
	}

	return nil
}

// pollAndProcessTasksCloud pulls messages from the single PubSub inbox adapter
// and routes each message to the correct agent based on the Inbox attribute.
// M-CLOUD-E2E: This replaces per-inbox SQLite polling in cloud mode.
func (d *Daemon) pollAndProcessTasksCloud() error {
	messages, err := d.cloudInboxAdapter.ListUnread()
	if err != nil {
		return fmt.Errorf("cloud inbox adapter: %w", err)
	}
	if len(messages) == 0 {
		return nil
	}

	d.logger.Printf("Cloud mode: received %d message(s) from Pub/Sub", len(messages))

	for _, msg := range messages {
		inbox := msg.Inbox
		if inbox == "" {
			inbox = "coordinator" // Fallback if Inbox attribute not set
		}

		// Look up agent for this inbox
		agent := d.agentRegistry.GetAgentForInbox(inbox)
		agentID := ""
		if agent != nil {
			agentID = agent.ID
		} else {
			d.logger.Printf("Warning: No agent registered for inbox %q (message %s)", inbox, msg.ID)
		}

		im := inboxMessage{
			inbox:   inbox,
			agentID: agentID,
			msg:     msg,
		}

		// Reuse the existing task creation logic from local mode.
		// Build task ID
		msgIDPrefix := msg.ID
		if len(msgIDPrefix) > 8 {
			msgIDPrefix = msgIDPrefix[:8]
		}
		taskID := fmt.Sprintf("task-%s", msgIDPrefix)

		// Determine kind
		kind := msg.Kind
		if kind == "" {
			if msg.Type == "question" || msg.Type == "research" {
				kind = "question"
			} else {
				kind = "directive"
			}
		}

		taskInput := &Task{
			ID:        taskID,
			Title:     msg.Title,
			Content:   msg.Content,
			Kind:      kind,
			MessageID: msg.ID,
			CreatedAt: msg.CreatedAt,
		}

		analyzed := d.analyzer.Analyze(taskInput)

		workspace := ""
		if agent != nil && agent.Workspace != "" {
			workspace = agent.Workspace
		} else {
			workspace, _ = os.Getwd()
			d.logger.Printf("WARNING: Agent %q has no workspace configured, using: %s", agentID, workspace)
		}

		iteration := msg.Iteration
		if iteration == 0 {
			iteration = 1
		}

		// M-HARNESS-COMMIT-CONTRACT: Extract siteSlug and briefId from message content.
		// Messages from website-builder may contain JSON with these fields.
		var siteSlug, briefID string
		var payloadFields struct {
			SiteSlug string `json:"siteSlug"`
			BriefID  string `json:"briefId"`
		}
		if json.Unmarshal([]byte(msg.Content), &payloadFields) == nil {
			siteSlug = payloadFields.SiteSlug
			briefID = payloadFields.BriefID
		}

		task := &TaskRecord{
			ID:            taskID,
			MessageID:     msg.ID,
			AgentID:       agentID,
			ParentTaskID:  msg.ParentTaskID,
			Iteration:     iteration,
			Title:         msg.Title,
			Content:       msg.Content,
			Type:          analyzed.Type,
			Kind:          kind,
			Priority:      CalculatePriority(analyzed),
			Status:        TaskStatusPending,
			Workspace:     workspace,
			GithubIssue:   msg.GithubIssue,
			GithubRepo:    msg.GithubRepo,
			CreatedAt:     msg.CreatedAt,
			Capabilities:  analyzed.Capabilities,
			ImpactLevel:   analyzed.ImpactLevel,
			EstimatedCost: analyzed.EstimatedCost,
			SiteSlug:      siteSlug, // M-HARNESS-COMMIT-CONTRACT
			BriefID:       briefID,  // M-HARNESS-COMMIT-CONTRACT
		}

		// Check for duplicates
		fingerprint := analyzed.Fingerprint
		if fingerprint != 0 {
			if dup, _ := d.taskStore.FindDuplicateTask(d.ctx, fingerprint, 0.9); dup != nil {
				d.logger.Printf("Skipping duplicate task for message %s (similar to task %s)", msg.ID, dup.ID)
				d.publishDedupCompletion(taskID, agentID, dup.ID)
				continue
			}
		}

		// Get or create thread for dashboard visibility
		targetAgent := agentID
		if targetAgent == "" {
			targetAgent = "coordinator"
		}
		thread, created, threadErr := d.msgStore.GetOrCreateThreadWithWorkspace(
			msg.Title, "ailang_instance", "coordinator", targetAgent, workspace,
		)
		if threadErr != nil {
			d.logger.Printf("Failed to get/create thread for task %s: %v", taskID, threadErr)
		} else {
			task.ThreadID = thread.ID
			if created {
				d.logger.Printf("Created thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
			}
		}

		// Store the task
		if err := d.taskStore.CreateTask(d.ctx, task); err != nil {
			d.logger.Printf("Failed to create task for message %s: %v", msg.ID, err)
			continue
		}

		if fingerprint != 0 {
			_ = d.taskStore.SetTaskFingerprint(d.ctx, task.ID, fingerprint)
		}

		// M-CHAINS-SIMPLIFY: Create execution chain and stage
		if d.obsBackend != nil {
			chainID := msg.ChainID
			var stageID string

			if chainID == "" && task.ParentTaskID != "" {
				if parentTask, err := d.taskStore.GetTask(d.ctx, task.ParentTaskID); err == nil && parentTask != nil {
					chainID = parentTask.ChainID
				}
			}
			if chainID == "" {
				sourceType := observatory.ChainSourceMessage
				if task.GithubIssue > 0 {
					sourceType = observatory.ChainSourceGitHubIssue
				}
				chain, err := d.obsBackend.CreateChain(d.ctx, &observatory.ChainCreateRequest{
					SourceType:        sourceType,
					SourceRef:         msg.ID,
					GitHubRepo:        task.GithubRepo,
					GitHubIssueNumber: task.GithubIssue,
				})
				if err != nil {
					d.logger.Printf("Warning: Failed to create chain for task %s: %v", task.ID, err)
				} else {
					chainID = chain.ID
				}
			}
			if chainID != "" {
				stage, err := d.obsBackend.CreateStage(d.ctx, &observatory.StageCreateRequest{
					ChainID: chainID, AgentID: agentID, MessageID: msg.ID, TaskID: task.ID,
				})
				if err != nil {
					d.logger.Printf("Warning: Failed to create stage for task %s: %v", task.ID, err)
				} else {
					stageID = stage.ID
				}
			}
			if chainID != "" || stageID != "" {
				task.ChainID = chainID
				task.StageID = stageID
				_ = d.taskStore.UpdateTaskChainInfo(d.ctx, task.ID, chainID, stageID)
			}
		}

		// GitHub pipeline integration
		if task.GithubIssue > 0 && d.taskChain != nil {
			if err := d.taskChain.StartTask(d.ctx, task.ID, task.GithubIssue); err != nil {
				d.logger.Printf("Warning: Failed to start task chain for issue #%d: %v", task.GithubIssue, err)
			}
			if d.approvalWatcher != nil {
				d.approvalWatcher.WatchIssue(task.GithubIssue, task.ID)
			}
		}

		d.logger.Printf("Created task %s (type: %s, agent: %s, inbox: %s) from cloud message %s",
			task.ID, task.Type, agentID, im.inbox, msg.ID)

		// PubSub messages are acked on receipt — MarkAsRead is a no-op
		d.cloudInboxAdapter.MarkAsRead(msg.ID)
		d.tasksRun++
	}

	return nil
}

// publishDedupCompletion posts a completion notification when a task is skipped
// due to deduplication. This ensures external clients (portal, sidecar) receive
// a response instead of hanging indefinitely waiting for a build that never starts.
func (d *Daemon) publishDedupCompletion(taskID, agentID, originalTaskID string) {
	if d.msgStore == nil {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"task_id":          taskID,
		"agent_id":         agentID,
		"status":           "deduplicated",
		"error_msg":        fmt.Sprintf("Skipped: similar to recent task %s", originalTaskID),
		"original_task_id": originalTaskID,
	})

	msg := &messaging.InboxMessage{
		FromAgent:   agentID,
		ToInbox:     agentID,
		MessageType: "completion",
		Title:       fmt.Sprintf("Task %s: deduplicated", taskID),
		Payload:     string(payload),
	}

	if err := d.msgStore.InsertInboxMessage(msg); err != nil {
		d.logger.Printf("Failed to post dedup completion for task %s: %v", taskID, err)
	} else {
		d.logger.Printf("Posted dedup completion for task %s (original: %s)", taskID, originalTaskID)
	}
}
