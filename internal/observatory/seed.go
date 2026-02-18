// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// SeedConfig configures the seed data generator.
type SeedConfig struct {
	NumWorkspaces     int
	TasksPerWorkspace Range
	SpansPerTask      Range
	IncludeMessages   bool
	CleanFirst        bool
	Verbose           bool
}

// Range represents a min/max range for random generation.
type Range struct {
	Min int
	Max int
}

// DefaultSeedConfig returns a realistic default configuration.
func DefaultSeedConfig() SeedConfig {
	return SeedConfig{
		NumWorkspaces:     3,
		TasksPerWorkspace: Range{Min: 5, Max: 10},
		SpansPerTask:      Range{Min: 10, Max: 50},
		IncludeMessages:   true,
		CleanFirst:        false,
		Verbose:           false,
	}
}

// MinimalSeedConfig returns a minimal configuration for quick testing.
func MinimalSeedConfig() SeedConfig {
	return SeedConfig{
		NumWorkspaces:     1,
		TasksPerWorkspace: Range{Min: 2, Max: 2},
		SpansPerTask:      Range{Min: 5, Max: 5},
		IncludeMessages:   false,
		CleanFirst:        false,
		Verbose:           false,
	}
}

// StressSeedConfig returns a configuration for load testing.
func StressSeedConfig() SeedConfig {
	return SeedConfig{
		NumWorkspaces:     10,
		TasksPerWorkspace: Range{Min: 10, Max: 10},
		SpansPerTask:      Range{Min: 100, Max: 100},
		IncludeMessages:   true,
		CleanFirst:        false,
		Verbose:           false,
	}
}

// SeedResult contains statistics from the seed operation.
type SeedResult struct {
	WorkspacesCreated  int
	TasksCreated       int
	AssignmentsCreated int
	SpansCreated       int
	EventsCreated      int
	MessagesCreated    int
}

// SeedDatabase generates test data in the database.
func SeedDatabase(ctx context.Context, backend Backend, cfg SeedConfig) (*SeedResult, error) {
	result := &SeedResult{}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Predefined realistic data
	workspaceNames := []string{"ailang-dev", "ailang-staging", "ailang-prod", "web-app", "api-service", "data-pipeline"}
	agentIDs := []string{"design-doc-creator", "sprint-planner", "sprint-executor", "code-reviewer", "bug-fixer"}
	providers := []Provider{ProviderClaude, ProviderGemini, ProviderClaude}
	models := map[Provider][]string{
		ProviderClaude: {"claude-sonnet-4-6", "claude-haiku-4-5", "claude-opus-4-6"},
		ProviderGemini: {"gemini-2-5-pro", "gemini-2-5-flash"},
		ProviderOllama: {"llama3", "codellama"},
	}
	spanNames := []string{
		"ailang.exec", "ailang.check", "ailang run: main.ail",
		"claude.api.call", "claude.tool.read", "claude.tool.write", "claude.tool.bash",
		"gemini.generate", "gemini.tool.call",
		"api.request", "db.query", "cache.lookup",
	}
	taskTitles := []string{
		"Fix null pointer in parser",
		"Add dark mode toggle",
		"Implement user authentication",
		"Refactor database queries",
		"Update documentation",
		"Fix CI pipeline",
		"Add unit tests for evaluator",
		"Optimize memory usage",
		"Implement caching layer",
		"Fix type inference bug",
	}

	// Create workspaces
	for i := 0; i < cfg.NumWorkspaces; i++ {
		wsName := workspaceNames[i%len(workspaceNames)]
		if i >= len(workspaceNames) {
			wsName = fmt.Sprintf("%s-%d", wsName, i)
		}

		workspace := &Workspace{
			ID:        fmt.Sprintf("ws-%s", uuid.New().String()[:8]),
			Name:      wsName,
			Path:      fmt.Sprintf("/Users/dev/projects/%s", wsName),
			GitRemote: fmt.Sprintf("https://github.com/example/%s", wsName),
			CreatedAt: time.Now().Add(-time.Duration(rng.Intn(30)) * 24 * time.Hour),
			UpdatedAt: time.Now(),
		}

		if err := backend.CreateWorkspace(ctx, workspace); err != nil {
			return result, fmt.Errorf("create workspace: %w", err)
		}
		result.WorkspacesCreated++

		if cfg.Verbose {
			fmt.Printf("Created workspace: %s (%s)\n", workspace.Name, workspace.ID)
		}

		// Create tasks for this workspace
		numTasks := randomInRange(rng, cfg.TasksPerWorkspace)
		for j := 0; j < numTasks; j++ {
			taskCreatedAt := time.Now().Add(-time.Duration(rng.Intn(7*24)) * time.Hour)
			taskStartedAt := taskCreatedAt.Add(time.Duration(rng.Intn(60)) * time.Minute)
			taskCompletedAt := taskStartedAt.Add(time.Duration(rng.Intn(120)+30) * time.Minute)

			status := randomTaskStatus(rng)
			var startedPtr, completedPtr *time.Time
			if status != TaskStatusPending {
				startedPtr = &taskStartedAt
			}
			if status == TaskStatusCompleted || status == TaskStatusFailed {
				completedPtr = &taskCompletedAt
			}

			task := &Task{
				ID:          fmt.Sprintf("task-%s", uuid.New().String()[:8]),
				WorkspaceID: workspace.ID,
				Title:       taskTitles[rng.Intn(len(taskTitles))],
				Description: fmt.Sprintf("Detailed description for task %d in workspace %s", j+1, wsName),
				SourceType:  randomSourceType(rng),
				Status:      status,
				Priority:    randomPriority(rng),
				CreatedAt:   taskCreatedAt,
				StartedAt:   startedPtr,
				CompletedAt: completedPtr,
			}

			if err := backend.CreateTask(ctx, task); err != nil {
				return result, fmt.Errorf("create task: %w", err)
			}
			result.TasksCreated++

			if cfg.Verbose {
				fmt.Printf("  Created task: %s (%s)\n", task.Title[:min(30, len(task.Title))], task.ID)
			}

			// Create agent assignments for this task
			numAssignments := rng.Intn(3) + 1
			for k := 0; k < numAssignments; k++ {
				provider := providers[rng.Intn(len(providers))]
				assignedAt := taskStartedAt.Add(time.Duration(k*30) * time.Minute)
				assignmentStartedAt := assignedAt.Add(time.Duration(rng.Intn(5)) * time.Minute)
				assignmentCompletedAt := assignmentStartedAt.Add(time.Duration(rng.Intn(60)+10) * time.Minute)

				assignmentStatus := AgentStatusCompleted
				if task.Status == TaskStatusRunning && k == numAssignments-1 {
					assignmentStatus = AgentStatusRunning
				} else if task.Status == TaskStatusFailed && k == numAssignments-1 {
					assignmentStatus = AgentStatusFailed
				}

				var assignStartPtr, assignCompletePtr *time.Time
				if assignmentStatus != AgentStatusPending {
					assignStartPtr = &assignmentStartedAt
				}
				if assignmentStatus == AgentStatusCompleted || assignmentStatus == AgentStatusFailed {
					assignCompletePtr = &assignmentCompletedAt
				}

				assignment := &AgentAssignment{
					ID:          fmt.Sprintf("assign-%s", uuid.New().String()[:8]),
					TaskID:      task.ID,
					AgentID:     agentIDs[rng.Intn(len(agentIDs))],
					Provider:    provider,
					Status:      assignmentStatus,
					AssignedAt:  assignedAt,
					StartedAt:   assignStartPtr,
					CompletedAt: assignCompletePtr,
				}

				if err := backend.CreateAgentAssignment(ctx, assignment); err != nil {
					return result, fmt.Errorf("create assignment: %w", err)
				}
				result.AssignmentsCreated++

				// Create spans for this assignment
				numSpans := randomInRange(rng, cfg.SpansPerTask)
				traceID := uuid.New().String()
				var rootSpanID string

				for s := 0; s < numSpans; s++ {
					spanStartTime := assignmentStartedAt.Add(time.Duration(s*100) * time.Millisecond)
					durationMs := int64(rng.Intn(5000) + 100)
					spanEndTime := spanStartTime.Add(time.Duration(durationMs) * time.Millisecond)

					// Determine parent span (first span is root)
					var parentSpanID string
					if s > 0 && rng.Float32() < 0.7 {
						// 70% chance to be child of root
						parentSpanID = rootSpanID
					}

					spanID := uuid.New().String()
					if s == 0 {
						rootSpanID = spanID
					}

					providerModels := models[provider]
					model := providerModels[rng.Intn(len(providerModels))]
					tokensIn := int64(rng.Intn(4000) + 100)
					tokensOut := int64(rng.Intn(2000) + 50)

					// Calculate realistic cost based on model
					costUSD := calculateSeedCost(model, tokensIn, tokensOut)

					span := &Span{
						ID:                spanID,
						TraceID:           traceID,
						ParentSpanID:      parentSpanID,
						TaskID:            task.ID,
						AgentAssignmentID: assignment.ID,
						Name:              spanNames[rng.Intn(len(spanNames))],
						Kind:              SpanKindInternal,
						Status:            randomSpanStatus(rng),
						StartTime:         spanStartTime,
						EndTime:           &spanEndTime,
						DurationMs:        durationMs,
						TokensIn:          tokensIn,
						TokensOut:         tokensOut,
						CostUSD:           costUSD,
						Model:             model,
						Provider:          provider,
						Attributes: map[string]any{
							"exec.task_id":  task.ID,
							"exec.provider": string(provider),
						},
						ResourceAttributes: map[string]any{
							"service.name": fmt.Sprintf("ailang-%s", provider),
							"process.cwd":  workspace.Path,
						},
						CreatedAt: spanStartTime,
					}

					if err := backend.CreateSpan(ctx, span); err != nil {
						return result, fmt.Errorf("create span: %w", err)
					}
					result.SpansCreated++

					// Create span events for some spans
					if rng.Float32() < 0.3 {
						numEvents := rng.Intn(3) + 1
						for e := 0; e < numEvents; e++ {
							event := &SpanEvent{
								SpanID:    spanID,
								Name:      randomEventName(rng),
								Timestamp: spanStartTime.Add(time.Duration(e*50) * time.Millisecond),
								EventType: randomEventType(rng),
								Attributes: map[string]any{
									"detail": fmt.Sprintf("Event %d detail", e+1),
								},
							}

							if err := backend.CreateSpanEvent(ctx, event); err != nil {
								// Non-fatal: continue if event creation fails
								continue
							}
							result.EventsCreated++
						}
					}
				}
			}
		}
	}

	// Create messages if configured
	if cfg.IncludeMessages {
		inboxes := []string{"user", "design-doc-creator", "sprint-planner", "sprint-executor", "coordinator"}
		messageTypes := []string{"bug", "feature", "question", "feedback", "handoff"}

		for i := 0; i < 10; i++ {
			createdAt := time.Now().Add(-time.Duration(rng.Intn(7*24)) * time.Hour)
			status := MessageStatusUnread
			if rng.Float32() < 0.5 {
				status = MessageStatusRead
			}

			message := &Message{
				ID:          fmt.Sprintf("msg-%s", uuid.New().String()[:8]),
				Inbox:       inboxes[rng.Intn(len(inboxes))],
				FromAgent:   agentIDs[rng.Intn(len(agentIDs))],
				Title:       fmt.Sprintf("Test message %d", i+1),
				Content:     fmt.Sprintf("This is test message content for message %d. It contains detailed information about the task or request.", i+1),
				MessageType: messageTypes[rng.Intn(len(messageTypes))],
				Status:      status,
				Priority:    randomPriority(rng),
				CreatedAt:   createdAt,
			}

			if err := backend.CreateMessage(ctx, message); err != nil {
				// Non-fatal: continue if message creation fails
				continue
			}
			result.MessagesCreated++

			if cfg.Verbose {
				fmt.Printf("Created message: %s (%s)\n", message.Title, message.ID)
			}
		}
	}

	return result, nil
}

// Helper functions

func randomInRange(rng *rand.Rand, r Range) int {
	if r.Max <= r.Min {
		return r.Min
	}
	return rng.Intn(r.Max-r.Min+1) + r.Min
}

func randomTaskStatus(rng *rand.Rand) TaskStatus {
	statuses := []TaskStatus{TaskStatusPending, TaskStatusRunning, TaskStatusCompleted, TaskStatusCompleted, TaskStatusFailed}
	return statuses[rng.Intn(len(statuses))]
}

func randomSourceType(rng *rand.Rand) TaskSourceType {
	sources := []TaskSourceType{TaskSourceGitHub, TaskSourceMessage, TaskSourceManual}
	return sources[rng.Intn(len(sources))]
}

func randomPriority(rng *rand.Rand) string {
	priorities := []string{"low", "medium", "high", "critical"}
	return priorities[rng.Intn(len(priorities))]
}

func randomSpanStatus(rng *rand.Rand) SpanStatus {
	// Weight towards OK status
	statuses := []SpanStatus{SpanStatusOK, SpanStatusOK, SpanStatusOK, SpanStatusUnset, SpanStatusError}
	return statuses[rng.Intn(len(statuses))]
}

func randomEventType(rng *rand.Rand) EventType {
	types := []EventType{EventTypeTool, EventTypeApproval, EventTypeError, EventTypeCustom}
	return types[rng.Intn(len(types))]
}

func randomEventName(rng *rand.Rand) string {
	names := []string{"tool.invoke", "approval.requested", "error.occurred", "checkpoint", "progress.update"}
	return names[rng.Intn(len(names))]
}

func calculateSeedCost(model string, tokensIn, tokensOut int64) float64 {
	// Simplified cost calculation for seed data
	// Real costs would come from models.yml
	rates := map[string]struct{ in, out float64 }{
		"claude-sonnet-4-6": {3.0, 15.0},
		"claude-sonnet-4-5": {3.0, 15.0},
		"claude-haiku-4-5":  {0.25, 1.25},
		"claude-opus-4-6":   {5.0, 25.0},
		"claude-opus-4":     {15.0, 75.0},
		"gemini-2-5-pro":    {1.25, 5.0},
		"gemini-2-5-flash":  {0.075, 0.3},
		"llama3":            {0.0, 0.0},
		"codellama":         {0.0, 0.0},
	}

	rate, ok := rates[model]
	if !ok {
		return 0.0
	}

	inCost := float64(tokensIn) / 1_000_000.0 * rate.in
	outCost := float64(tokensOut) / 1_000_000.0 * rate.out
	return inCost + outCost
}
