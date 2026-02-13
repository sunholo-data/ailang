package coordinator

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/observatory"
)

// inboxMessage pairs a message with the inbox and agent it came from.
type inboxMessage struct {
	inbox   string
	agentID string
	msg     *Message // coordinator.Message type
}

// pollAndProcessTasks polls for new messages and queues them as tasks
func (d *Daemon) pollAndProcessTasks() error {
	// Collect messages from all inbox adapters
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
