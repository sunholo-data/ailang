package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// createEvalTask creates a Task in Observatory for this eval run.
// This enables eval suites to appear in the task hierarchy alongside
// coordinator tasks. The task ID is used as ailang.task_id resource
// attribute so all child spans (benchmarks, API calls) are linked.
func createEvalTask(taskID, assignmentID string, models, benchmarks, langs []string, totalRuns int, agentMode bool) {
	// Try Observatory API endpoint (default server at localhost:1957)
	endpoint := os.Getenv("OBSERVATORY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1957"
	}

	// Get working directory for workspace lookup
	cwd, _ := os.Getwd()

	// Find workspace ID by looking up existing workspaces
	workspaceID := lookupWorkspaceID(endpoint, cwd)
	if workspaceID == "" {
		// No matching workspace found - skip task creation
		// (Task creation would fail due to foreign key constraint)
		return
	}

	// Build task title
	title := fmt.Sprintf("Eval: %d benchmarks × %d models", len(benchmarks), len(models))
	if agentMode {
		title += " (agent)"
	}

	// Build description
	description := fmt.Sprintf("Models: %s\nBenchmarks: %s\nLanguages: %s\nTotal runs: %d",
		strings.Join(models, ", "),
		strings.Join(benchmarks, ", "),
		strings.Join(langs, ", "),
		totalRuns,
	)

	// Create task object
	now := time.Now()
	task := observatory.Task{
		ID:          taskID,
		WorkspaceID: workspaceID,
		Title:       title,
		Description: description,
		SourceType:  observatory.TaskSourceManual,
		SourceRef:   "eval-suite",
		Status:      observatory.TaskStatusRunning,
		Priority:    "normal",
		CreatedAt:   now,
		StartedAt:   &now,
	}

	// POST to Observatory API
	taskJSON, err := json.Marshal(task)
	if err != nil {
		// Non-fatal: task creation failure shouldn't stop eval
		return
	}

	resp, err := http.Post(
		endpoint+"/api/observatory/tasks",
		"application/json",
		bytes.NewReader(taskJSON),
	)
	if err != nil {
		// Non-fatal: Observatory might not be running
		return
	}
	defer resp.Body.Close()

	// Log success if created
	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("%s Task created in Observatory: %s\n", cyan("📊"), taskID)

		// Also create an agent assignment so spans show up in UI
		// The UI groups spans under agent assignments
		createEvalAgentAssignment(endpoint, taskID, assignmentID, models, agentMode)
	}
}

// createEvalAgentAssignment creates an agent assignment for eval runs.
// This is needed because the UI groups spans under agent assignments.
// The assignmentID is passed from the caller to ensure it matches what was
// set in OTEL_RESOURCE_ATTRIBUTES (so spans link to this assignment).
func createEvalAgentAssignment(endpoint, taskID, assignmentID string, models []string, agentMode bool) {
	now := time.Now()

	// Determine provider from models
	provider := observatory.ProviderClaude // default
	for _, model := range models {
		if strings.Contains(model, "gemini") {
			provider = observatory.ProviderGemini
			break
		}
	}

	agentID := "eval-harness"
	if agentMode {
		agentID = "eval-agent"
	}

	assignment := observatory.AgentAssignment{
		ID:         assignmentID,
		TaskID:     taskID,
		AgentID:    agentID,
		Provider:   provider,
		Status:     observatory.AgentStatusRunning,
		AssignedAt: now,
		StartedAt:  &now,
	}

	assignmentJSON, err := json.Marshal(assignment)
	if err != nil {
		return
	}

	resp, err := http.Post(
		endpoint+"/api/observatory/agents",
		"application/json",
		bytes.NewReader(assignmentJSON),
	)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// completeEvalTask updates the task status to completed when the eval finishes.
func completeEvalTask(taskID string, success bool) {
	endpoint := os.Getenv("OBSERVATORY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1957"
	}

	status := observatory.TaskStatusCompleted
	if !success {
		status = observatory.TaskStatusFailed
	}

	now := time.Now()
	update := map[string]any{
		"status":       status,
		"completed_at": now.Format(time.RFC3339),
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPut, endpoint+"/api/observatory/tasks/"+taskID, bytes.NewReader(updateJSON))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// lookupWorkspaceID finds the workspace ID for a given path by querying the Observatory API.
// Returns empty string if no workspace found or API unavailable.
func lookupWorkspaceID(endpoint, path string) string {
	resp, err := http.Get(endpoint + "/api/observatory/workspaces")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var workspaces []observatory.Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return ""
	}

	// Find workspace matching our path
	for _, ws := range workspaces {
		if ws.Path == path {
			return ws.ID
		}
	}

	return ""
}
