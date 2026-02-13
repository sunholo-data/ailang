package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/observatory"
)

// TaskHierarchyNode represents a task with its relationships for cross-task visualization.
type TaskHierarchyNode struct {
	ID             string               `json:"id"`
	Title          string               `json:"title"`
	AgentID        string               `json:"agent_id,omitempty"`
	ParentTaskID   string               `json:"parent_task_id,omitempty"`
	SessionID      string               `json:"session_id,omitempty"`
	Status         string               `json:"status"`
	ApprovalStatus string               `json:"approval_status,omitempty"` // "pending", "approved", "rejected", ""
	ApprovalType   string               `json:"approval_type,omitempty"`   // "merge", "merge_handoff", etc.
	Iteration      int                  `json:"iteration,omitempty"`
	Cost           float64              `json:"cost"`
	TokensIn       int                  `json:"tokens_in"`
	TokensOut      int                  `json:"tokens_out"`
	Turns          int                  `json:"turns,omitempty"`
	DurationMs     int64                `json:"duration_ms"`
	CreatedAt      time.Time            `json:"created_at"`
	Provider       string               `json:"provider,omitempty"`
	Workspace      string               `json:"workspace,omitempty"`
	Children       []*TaskHierarchyNode `json:"children,omitempty"` // Child tasks (via parent_task_id)
	// Execution spans nested within this task (from observatory.db)
	Spans []*TaskSpanNode `json:"spans,omitempty"`
	// Turn-grouped hierarchy (when group_by=turns is requested)
	TurnGrouped *observatory.TurnGroupedHierarchy `json:"turn_grouped,omitempty"`
}

// TaskSpanNode represents a span within a task for the unified task hierarchy view.
// Simplified version of SpanHierarchyNode for API response.
type TaskSpanNode struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	NodeType   string          `json:"node_type"` // coordinator, executor, turn, tool, other
	DurationMs int64           `json:"duration_ms"`
	TokensIn   int64           `json:"tokens_in,omitempty"`
	TokensOut  int64           `json:"tokens_out,omitempty"`
	CostUSD    float64         `json:"cost_usd,omitempty"`
	TurnNumber int             `json:"turn_number,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	Status     string          `json:"status"`
	Children   []*TaskSpanNode `json:"children,omitempty"`
}

// TaskHierarchyEdge represents a relationship between tasks.
type TaskHierarchyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "handoff" (parent_task_id) or "session" (shared session_id)
}

// TaskHierarchyResult contains the full cross-task hierarchy.
type TaskHierarchyResult struct {
	Tasks []*TaskHierarchyNode `json:"tasks"`
	Edges []TaskHierarchyEdge  `json:"edges"`
	Stats struct {
		TotalTasks       int     `json:"total_tasks"`
		TotalSpans       int     `json:"total_spans"`
		PendingApprovals int     `json:"pending_approvals"`
		TotalCost        float64 `json:"total_cost"`
	} `json:"stats"`
}

// GET /api/controlplane/task-hierarchy - Get cross-task hierarchy with relationships
// Returns tasks with parent_task_id chains, session continuity, and approval status.
// Query params:
//   - limit: Maximum number of tasks (default: 50)
//   - status: Filter by status (optional, comma-separated)
//   - workspace: Filter by workspace path (optional)
//   - provider: Filter by provider (optional)
//   - task_id: Filter to specific task and its handoff chain (optional)
//   - trace_id: Filter to tasks with spans in this trace (optional)
//   - group_by: "turns" to group spans by conversation turn (Session -> Turn 1 -> Turn 2 -> ...)
func (s *Server) handleTaskHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get coordinator store
	coordStore := s.getCoordStoreForControlPlane()
	if coordStore == nil {
		http.Error(w, "Coordinator store not available", http.StatusServiceUnavailable)
		return
	}

	// Parse limit parameter
	limit := 50
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	// Parse status filter
	var statusFilter []coordinator.TaskStatus
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		for _, s := range strings.Split(statusParam, ",") {
			statusFilter = append(statusFilter, coordinator.TaskStatus(strings.TrimSpace(s)))
		}
	}

	// Parse workspace filter
	workspace := r.URL.Query().Get("workspace")

	// Parse provider filter
	provider := r.URL.Query().Get("provider")

	// Parse task_id filter (filter to specific task and its chain)
	filterTaskID := r.URL.Query().Get("task_id")

	// Parse task_ids filter (filter to multiple specific task IDs, comma-separated)
	var filterTaskIDs []string
	if taskIDsParam := r.URL.Query().Get("task_ids"); taskIDsParam != "" {
		for _, id := range strings.Split(taskIDsParam, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				filterTaskIDs = append(filterTaskIDs, trimmed)
			}
		}
	}

	// Parse trace_id filter (filter to tasks with spans in this trace)
	filterTraceID := r.URL.Query().Get("trace_id")

	// Parse group_by parameter (turns = group spans by conversation turn)
	groupBy := r.URL.Query().Get("group_by")

	// If trace_id filter is provided, find task_ids from spans first
	var traceTaskIDs map[string]bool
	if filterTraceID != "" && s.obsBackend != nil {
		spans, err := s.obsBackend.ListSpans(ctx, observatory.SpanListOptions{
			TraceID: filterTraceID,
			Limit:   1000,
		})
		if err != nil {
			log.Printf("Failed to query spans for trace %s: %v", filterTraceID, err)
		} else {
			traceTaskIDs = make(map[string]bool)
			for _, span := range spans {
				if span.TaskID != "" {
					traceTaskIDs[span.TaskID] = true
				}
			}
		}
	}

	// Fetch tasks
	filter := &coordinator.TaskFilter{
		Limit:     limit,
		OrderBy:   "created_at",
		OrderDesc: true,
		Workspace: workspace,
		Provider:  provider,
	}
	if len(statusFilter) > 0 {
		filter.Status = statusFilter
	}

	tasks, err := coordStore.ListTasks(ctx, filter)
	if err != nil {
		log.Printf("Failed to list tasks for hierarchy: %v", err)
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	// Filter to specific task and its handoff chain if task_id is provided
	if filterTaskID != "" {
		tasks = filterTaskChain(tasks, filterTaskID)
	}

	// Filter by trace_id if provided (tasks with spans in that trace)
	if traceTaskIDs != nil {
		var filteredTasks []*coordinator.TaskRecord
		for _, t := range tasks {
			if traceTaskIDs[t.ID] {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasks = filteredTasks
	}

	// Filter by task_ids if provided (multiple specific task IDs)
	if len(filterTaskIDs) > 0 {
		taskIDSet := make(map[string]bool)
		for _, id := range filterTaskIDs {
			taskIDSet[id] = true
		}
		var filteredTasks []*coordinator.TaskRecord
		for _, t := range tasks {
			if taskIDSet[t.ID] {
				filteredTasks = append(filteredTasks, t)
			}
		}
		tasks = filteredTasks
	}

	// Fetch pending approvals
	pendingApprovals, err := coordStore.ListPendingApprovals(ctx)
	if err != nil {
		log.Printf("Failed to list pending approvals: %v", err)
		// Continue without approvals
		pendingApprovals = nil
	}

	// Build approval lookup map: task_id -> approval
	approvalMap := make(map[string]*coordinator.ApprovalRequestRecord)
	for _, apr := range pendingApprovals {
		approvalMap[apr.TaskID] = apr
	}

	// Build result (initialize slices to avoid null in JSON)
	result := TaskHierarchyResult{
		Tasks: []*TaskHierarchyNode{},
		Edges: []TaskHierarchyEdge{},
	}
	taskMap := make(map[string]*TaskHierarchyNode)
	sessionTasks := make(map[string][]string) // session_id -> task_ids

	for _, task := range tasks {
		node := &TaskHierarchyNode{
			ID:           task.ID,
			Title:        task.Title,
			AgentID:      task.AgentID,
			ParentTaskID: task.ParentTaskID,
			SessionID:    task.SessionID,
			Status:       string(task.Status),
			Cost:         task.Cost,
			TokensIn:     task.InputTokens,
			TokensOut:    task.OutputTokens,
			CreatedAt:    task.CreatedAt,
			Provider:     task.Provider,
			Workspace:    task.Workspace,
			Iteration:    task.Iteration,
		}

		// Calculate duration
		if task.StartedAt != nil && task.CompletedAt != nil {
			node.DurationMs = task.CompletedAt.Sub(*task.StartedAt).Milliseconds()
		} else if task.Duration > 0 {
			node.DurationMs = task.Duration.Milliseconds()
		}

		// Add approval status if exists
		if apr, ok := approvalMap[task.ID]; ok {
			node.ApprovalStatus = apr.Status
			node.ApprovalType = apr.Type
		} else if task.Status == coordinator.TaskStatusPendingApproval {
			node.ApprovalStatus = "pending"
		}

		taskMap[task.ID] = node
		result.Tasks = append(result.Tasks, node)

		// Track session relationships
		if task.SessionID != "" {
			sessionTasks[task.SessionID] = append(sessionTasks[task.SessionID], task.ID)
		}

		// Update stats
		result.Stats.TotalTasks++
		result.Stats.TotalCost += task.Cost
		if node.ApprovalStatus == "pending" {
			result.Stats.PendingApprovals++
		}
	}

	// Fetch spans from observatory.db for each task
	if s.obsBackend != nil {
		for _, node := range result.Tasks {
			spans, err := s.obsBackend.ListSpans(ctx, observatory.SpanListOptions{
				TaskID: node.ID,
				Limit:  500, // Limit spans per task
			})
			if err != nil {
				log.Printf("Failed to fetch spans for task %s: %v", node.ID, err)
				continue
			}
			if len(spans) > 0 {
				// Build span hierarchy from flat list
				node.Spans = buildSpanHierarchyForTask(spans)
				result.Stats.TotalSpans += len(spans)

				// Apply turn grouping if requested
				if groupBy == "turns" {
					spanNodes := buildSpanNodeTreeFromFlat(spans)
					node.TurnGrouped = observatory.GroupSpansByTurn(spanNodes)
				}
			}
		}
	}

	// Build edges (only include edges where BOTH source and target exist in result)
	for _, task := range result.Tasks {
		// Handoff edges (parent_task_id)
		if task.ParentTaskID != "" {
			// Only add edge if parent task exists in filtered result
			if _, parentExists := taskMap[task.ParentTaskID]; parentExists {
				result.Edges = append(result.Edges, TaskHierarchyEdge{
					Source: task.ParentTaskID,
					Target: task.ID,
					Type:   "handoff",
				})
			}
		}
	}

	// Session edges (shared session_id - only add between tasks that exist)
	for sessionID, taskIDs := range sessionTasks {
		if len(taskIDs) > 1 && sessionID != "" {
			// Both tasks must exist in the result
			if _, ok1 := taskMap[taskIDs[0]]; ok1 {
				if _, ok2 := taskMap[taskIDs[1]]; ok2 {
					result.Edges = append(result.Edges, TaskHierarchyEdge{
						Source: taskIDs[0],
						Target: taskIDs[1],
						Type:   "session",
					})
				}
			}
		}
	}

	// Build children arrays for tree structure
	for _, task := range result.Tasks {
		if task.ParentTaskID != "" {
			if parent, ok := taskMap[task.ParentTaskID]; ok {
				parent.Children = append(parent.Children, task)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode task hierarchy response: %v", err)
	}
}

// filterTaskChain filters tasks to include only the specified task and its
// parent/child handoff chain.
func filterTaskChain(tasks []*coordinator.TaskRecord, filterTaskID string) []*coordinator.TaskRecord {
	// Build task lookup for filtering
	allTasksMap := make(map[string]*coordinator.TaskRecord)
	for _, t := range tasks {
		allTasksMap[t.ID] = t
	}

	// Collect task IDs that should be included (the task + its chain)
	includeIDs := make(map[string]bool)

	// Include the requested task
	includeIDs[filterTaskID] = true

	// Walk up parent chain
	currentID := filterTaskID
	for {
		if t, ok := allTasksMap[currentID]; ok && t.ParentTaskID != "" {
			includeIDs[t.ParentTaskID] = true
			currentID = t.ParentTaskID
		} else {
			break
		}
	}

	// Include child tasks (tasks where parent_task_id = any included task)
	changed := true
	for changed {
		changed = false
		for _, t := range tasks {
			if t.ParentTaskID != "" && includeIDs[t.ParentTaskID] && !includeIDs[t.ID] {
				includeIDs[t.ID] = true
				changed = true
			}
		}
	}

	// Filter tasks list
	var filteredTasks []*coordinator.TaskRecord
	for _, t := range tasks {
		if includeIDs[t.ID] {
			filteredTasks = append(filteredTasks, t)
		}
	}
	return filteredTasks
}

// buildSpanHierarchyForTask converts a flat list of spans into a tree structure
// using parent_span_id relationships. Returns root spans with nested children.
func buildSpanHierarchyForTask(spans []*observatory.Span) []*TaskSpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Convert spans to TaskSpanNode and build lookup map
	nodeMap := make(map[string]*TaskSpanNode)
	for _, span := range spans {
		node := &TaskSpanNode{
			ID:         span.ID,
			Name:       span.Name,
			NodeType:   classifySpanNodeType(span.Name),
			DurationMs: span.DurationMs,
			TokensIn:   span.TokensIn,
			TokensOut:  span.TokensOut,
			CostUSD:    span.CostUSD,
			Status:     string(span.Status),
		}

		// Extract turn number from attributes if present
		if span.Attributes != nil {
			if turnNum, ok := span.Attributes["turn.number"]; ok {
				if tn, ok := turnNum.(float64); ok {
					node.TurnNumber = int(tn)
				}
			}
			if toolName, ok := span.Attributes["tool.name"]; ok {
				if tn, ok := toolName.(string); ok {
					node.ToolName = tn
				}
			}
		}

		nodeMap[span.ID] = node
	}

	// Build parent-child relationships
	var roots []*TaskSpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID != "" {
			if parent, ok := nodeMap[span.ParentSpanID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not in this task's spans - treat as root
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	return roots
}

// classifySpanNodeType determines the node type based on span name.
func classifySpanNodeType(name string) string {
	switch {
	case strings.HasPrefix(name, "coordinator."):
		return "coordinator"
	case strings.HasPrefix(name, "claude.") || strings.HasPrefix(name, "gemini.") ||
		strings.HasPrefix(name, "openai.") || name == "ailang.exec":
		return "executor"
	case strings.HasPrefix(name, "exec.turn") || strings.HasPrefix(name, "turn."):
		return "turn"
	case strings.HasPrefix(name, "exec.tool_use") || strings.HasPrefix(name, "tool."):
		return "tool"
	default:
		return "other"
	}
}

// buildSpanNodeTreeFromFlat converts a flat list of spans into observatory.SpanNode tree.
// This is used for turn grouping which requires the SpanNode tree structure.
func buildSpanNodeTreeFromFlat(spans []*observatory.Span) []*observatory.SpanNode {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*observatory.SpanNode)
	for _, span := range spans {
		nodeMap[span.ID] = &observatory.SpanNode{Span: span}
	}

	// Build parent-child relationships
	var roots []*observatory.SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[span.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}
