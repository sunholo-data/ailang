package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/observatory"
)

// traceHierarchyCommand shows span hierarchy from the local observatory database
// Supports filtering by --workspace and --task-id for unified task+span view
func traceHierarchyCommand() {
	fs := flag.NewFlagSet("trace hierarchy", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Maximum number of tasks/spans to show")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	workspace := fs.String("workspace", "", "Filter by workspace path (shows tasks with their spans)")
	taskID := fs.String("task-id", "", "Filter to specific task and its child tasks")

	// Skip "ailang trace hierarchy" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// If --workspace or --task-id specified, use unified task+span view
	if *workspace != "" || *taskID != "" {
		traceTaskHierarchyCommand(*workspace, *taskID, *limit, *jsonOutput)
		return
	}

	// Original behavior: show span hierarchy without task grouping
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening observatory database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", dbPath)
		os.Exit(1)
	}
	defer backend.Close()
	store := backend.Store()

	// Get hierarchy
	result, err := store.GetSpanHierarchy(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting span hierarchy: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Pretty print
	if len(result.Roots) == 0 {
		fmt.Println("No span hierarchy found in local database")
		fmt.Printf("Database path: %s\n", dbPath)
		return
	}

	fmt.Printf("Span Hierarchy (local database)\n")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Total spans: %d | Cost: $%.4f | Tokens: %d in / %d out",
		result.Stats.TotalSpans, result.Stats.TotalCost,
		result.Stats.TotalTokens.In, result.Stats.TotalTokens.Out)
	if result.Stats.TotalTokens.CacheRead > 0 || result.Stats.TotalTokens.CacheCreation > 0 {
		fmt.Printf(" | Cache: %d read / %d create",
			result.Stats.TotalTokens.CacheRead, result.Stats.TotalTokens.CacheCreation)
	}
	fmt.Println()
	if len(result.Sessions) > 0 {
		fmt.Printf("Sessions: %d\n", len(result.Sessions))
	}
	fmt.Println(strings.Repeat("─", 80))

	for _, root := range result.Roots {
		printSpanNode(root, "")
	}
}

// unifiedTaskSpan is used for the unified task + span hierarchy view
type unifiedTaskSpan struct {
	Task     *coordinator.TaskRecord          `json:"task"`
	Spans    []*observatory.SpanHierarchyNode `json:"spans,omitempty"`
	Children []*unifiedTaskSpan               `json:"children,omitempty"`
}

// unifiedResult is the full result for the task hierarchy command
type unifiedResult struct {
	Tasks []*unifiedTaskSpan `json:"tasks"`
	Stats struct {
		TotalTasks       int     `json:"total_tasks"`
		TotalSpans       int     `json:"total_spans"`
		TotalCost        float64 `json:"total_cost"`
		PendingApprovals int     `json:"pending_approvals"`
	} `json:"stats"`
}

// traceTaskHierarchyCommand shows unified task + span hierarchy
// This combines data from coordinator.db (tasks) and observatory.db (spans)
func traceTaskHierarchyCommand(workspace, taskID string, limit int, jsonOutput bool) {
	ctx := context.Background()

	// Open coordinator database for tasks
	homeDir, _ := os.UserHomeDir()
	coordDBPath := filepath.Join(homeDir, ".ailang", "state", "coordinator.db")
	coordStore, err := coordinator.NewSQLiteStore(coordDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening coordinator database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", coordDBPath)
		os.Exit(1)
	}
	defer coordStore.Close()

	// Open observatory database for spans
	obsDBPath := observatory.DefaultDatabasePath()
	obsBackend, err := observatory.NewSQLiteBackendFromPath(obsDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening observatory database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", obsDBPath)
		os.Exit(1)
	}
	defer obsBackend.Close()

	result := &unifiedResult{}

	// Fetch tasks based on filters
	var tasks []*coordinator.TaskRecord
	if taskID != "" {
		// Get specific task and its children
		task, err := coordStore.GetTask(ctx, taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching task %s: %v\n", taskID, err)
			os.Exit(1)
		}
		if task == nil {
			fmt.Fprintf(os.Stderr, "Task not found: %s\n", taskID)
			os.Exit(1)
		}
		tasks = append(tasks, task)

		// Fetch child tasks (handoff chain)
		childFilter := &coordinator.TaskFilter{
			Limit:     100,
			OrderBy:   "created_at",
			OrderDesc: false,
		}
		allTasks, err := coordStore.ListTasks(ctx, childFilter)
		if err == nil {
			for _, t := range allTasks {
				if t.ParentTaskID == taskID {
					tasks = append(tasks, t)
				}
			}
		}
	} else if workspace != "" {
		// Filter by workspace
		filter := &coordinator.TaskFilter{
			Workspace: workspace,
			Limit:     limit,
			OrderBy:   "created_at",
			OrderDesc: true,
		}
		tasks, err = coordStore.ListTasks(ctx, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tasks: %v\n", err)
			os.Exit(1)
		}
	}

	// Build task map for hierarchy building
	taskMap := make(map[string]*unifiedTaskSpan)
	for _, task := range tasks {
		taskMap[task.ID] = &unifiedTaskSpan{Task: task}
	}

	// For each task, fetch its spans from observatory
	obsStore := obsBackend.Store()
	for _, uts := range taskMap {
		spans, err := obsStore.ListSpans(observatory.SpanListOptions{
			TaskID: uts.Task.ID,
			Limit:  500,
		})
		if err != nil {
			continue // Skip on error
		}

		// Build span hierarchy for this task
		spanNodes := buildSpanHierarchyFromList(spans)
		uts.Spans = spanNodes

		// Update stats
		result.Stats.TotalTasks++
		result.Stats.TotalSpans += len(spans)
		result.Stats.TotalCost += uts.Task.Cost
		if uts.Task.Status == coordinator.TaskStatusPendingApproval {
			result.Stats.PendingApprovals++
		}
	}

	// Build parent-child relationships between tasks
	for _, uts := range taskMap {
		if uts.Task.ParentTaskID != "" {
			if parent, ok := taskMap[uts.Task.ParentTaskID]; ok {
				parent.Children = append(parent.Children, uts)
			}
		}
	}

	// Collect root tasks (no parent in our set)
	for _, uts := range taskMap {
		if uts.Task.ParentTaskID == "" || taskMap[uts.Task.ParentTaskID] == nil {
			result.Tasks = append(result.Tasks, uts)
		}
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Pretty print
	if len(result.Tasks) == 0 {
		fmt.Println("No tasks found matching filters")
		if workspace != "" {
			fmt.Printf("Workspace filter: %s\n", workspace)
		}
		if taskID != "" {
			fmt.Printf("Task ID filter: %s\n", taskID)
		}
		return
	}

	fmt.Printf("Task + Span Hierarchy\n")
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Tasks: %d | Spans: %d | Cost: $%.4f | Pending: %d\n",
		result.Stats.TotalTasks, result.Stats.TotalSpans,
		result.Stats.TotalCost, result.Stats.PendingApprovals)
	fmt.Println(strings.Repeat("═", 80))

	for _, task := range result.Tasks {
		printUnifiedTask(task, "")
	}
}

// buildSpanHierarchyFromList builds a hierarchy from a flat list of spans
func buildSpanHierarchyFromList(spans []*observatory.Span) []*observatory.SpanHierarchyNode {
	if len(spans) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*observatory.SpanHierarchyNode)
	for _, span := range spans {
		nodeMap[span.ID] = &observatory.SpanHierarchyNode{
			ID:         span.ID,
			ParentID:   span.ParentSpanID,
			Name:       span.Name,
			DurationMs: span.DurationMs,
			CostUSD:    span.CostUSD,
			TokensIn:   span.TokensIn,
			TokensOut:  span.TokensOut,
			Status:     span.Status,
			Provider:   span.Provider,
			NodeType:   classifySpanType(span.Name),
			Children:   []*observatory.SpanHierarchyNode{},
		}
	}

	// Build parent-child relationships
	var roots []*observatory.SpanHierarchyNode
	for _, node := range nodeMap {
		if node.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in our set, treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

// classifySpanType determines the node type from span name
func classifySpanType(name string) observatory.SpanHierarchyNodeType {
	switch {
	case strings.Contains(name, "coordinator"):
		return observatory.NodeTypeCoordinator
	case strings.Contains(name, "execute") || strings.Contains(name, "claude") || strings.Contains(name, "gemini"):
		return observatory.NodeTypeExecutor
	case strings.Contains(name, "turn"):
		return observatory.NodeTypeTurn
	case strings.Contains(name, "tool"):
		return observatory.NodeTypeTool
	default:
		return observatory.NodeTypeOther
	}
}

// printUnifiedTask prints a task with its spans in tree format
func printUnifiedTask(u *unifiedTaskSpan, prefix string) {
	if u == nil {
		return
	}
	task := u.Task

	// Approval status badge
	statusBadge := ""
	switch task.Status {
	case coordinator.TaskStatusPendingApproval:
		statusBadge = " [pending_approval]"
	case coordinator.TaskStatusCompleted:
		statusBadge = " [completed]"
	case coordinator.TaskStatusFailed:
		statusBadge = " [failed]"
	case coordinator.TaskStatusRunning:
		statusBadge = " [running]"
	}

	// Agent info
	agentInfo := ""
	if task.AgentID != "" {
		agentInfo = fmt.Sprintf(" (%s)", task.AgentID)
	}

	// Duration
	durationStr := ""
	if task.Duration > 0 {
		if task.Duration.Seconds() >= 1 {
			durationStr = fmt.Sprintf(" %.1fs", task.Duration.Seconds())
		} else {
			durationStr = fmt.Sprintf(" %dms", task.Duration.Milliseconds())
		}
	}

	// Cost
	costStr := ""
	if task.Cost > 0 {
		costStr = fmt.Sprintf(" $%.4f", task.Cost)
	}

	fmt.Printf("%s⬢ Task: %s%s%s%s%s\n", prefix, task.ID, agentInfo, statusBadge, durationStr, costStr)

	// Print title if different from ID
	if task.Title != "" && task.Title != task.ID {
		fmt.Printf("%s  Title: %s\n", prefix, truncateString(task.Title, 60))
	}

	// Print spans
	spanCount := len(u.Spans)
	for i, span := range u.Spans {
		isLastSpan := i == spanCount-1 && len(u.Children) == 0
		if isLastSpan {
			fmt.Printf("%s└─ ", prefix)
			printSpanHierarchyNode(span, prefix+"   ")
		} else {
			fmt.Printf("%s├─ ", prefix)
			printSpanHierarchyNode(span, prefix+"│  ")
		}
	}

	// Print child tasks (handoff chain)
	childCount := len(u.Children)
	for i, child := range u.Children {
		isLast := i == childCount-1
		if isLast {
			fmt.Printf("%s└─ Handoff → ", prefix)
			printUnifiedTask(child, prefix+"   ")
		} else {
			fmt.Printf("%s├─ Handoff → ", prefix)
			printUnifiedTask(child, prefix+"│  ")
		}
	}
}

// printSpanHierarchyNode prints a span node in tree format
func printSpanHierarchyNode(node *observatory.SpanHierarchyNode, prefix string) {
	// Icon by type
	icon := "•"
	switch node.NodeType {
	case observatory.NodeTypeCoordinator:
		icon = "⬢"
	case observatory.NodeTypeExecutor:
		icon = "●"
	case observatory.NodeTypeTurn:
		icon = "◉"
	case observatory.NodeTypeTool:
		icon = "▸"
	}

	// Build name
	name := node.Name
	if node.ToolName != "" {
		name = node.ToolName
	}
	if node.TurnNumber > 0 {
		name = fmt.Sprintf("Turn #%d", node.TurnNumber)
	}

	// Metrics
	metrics := ""
	if node.DurationMs > 0 {
		if node.DurationMs >= 1000 {
			metrics += fmt.Sprintf(" %.1fs", float64(node.DurationMs)/1000)
		} else {
			metrics += fmt.Sprintf(" %dms", node.DurationMs)
		}
	}
	if node.CostUSD > 0 {
		metrics += fmt.Sprintf(" $%.4f", node.CostUSD)
	}
	if node.TokensIn > 0 || node.TokensOut > 0 {
		metrics += fmt.Sprintf(" [%d→%d]", node.TokensIn, node.TokensOut)
	}

	fmt.Printf("%s %s%s\n", icon, name, metrics)

	// Print children
	childCount := len(node.Children)
	for i, child := range node.Children {
		isLast := i == childCount-1
		if isLast {
			fmt.Printf("%s└─ ", prefix)
			printSpanHierarchyNode(child, prefix+"   ")
		} else {
			fmt.Printf("%s├─ ", prefix)
			printSpanHierarchyNode(child, prefix+"│  ")
		}
	}
}

// printSpanNode recursively prints a span node with tree formatting
func printSpanNode(node *observatory.SpanHierarchyNode, prefix string) {
	// Build display line
	name := node.Name
	if node.ToolName != "" {
		name = fmt.Sprintf("%s: %s", node.Name, node.ToolName)
	}

	// Add metrics
	metrics := ""
	if node.DurationMs > 0 {
		if node.DurationMs >= 1000 {
			metrics += fmt.Sprintf(" %.1fs", float64(node.DurationMs)/1000)
		} else {
			metrics += fmt.Sprintf(" %dms", node.DurationMs)
		}
	}
	if node.CostUSD > 0 {
		metrics += fmt.Sprintf(" $%.4f", node.CostUSD)
	}
	if node.TokensIn > 0 || node.TokensOut > 0 {
		metrics += fmt.Sprintf(" [%d→%d]", node.TokensIn, node.TokensOut)
	}
	if node.CacheReadTokens > 0 || node.CacheCreationTokens > 0 {
		metrics += fmt.Sprintf(" 📦%d/%d", node.CacheReadTokens, node.CacheCreationTokens)
	}

	// Add turn number for turn spans
	if node.TurnNumber > 0 {
		name = fmt.Sprintf("Turn #%d", node.TurnNumber)
	}

	// Color/icon by type
	icon := "•"
	switch node.NodeType {
	case observatory.NodeTypeCoordinator:
		icon = "⬢"
	case observatory.NodeTypeExecutor:
		icon = "●"
	case observatory.NodeTypeTurn:
		icon = "◉"
	case observatory.NodeTypeTool:
		icon = "▸"
	}

	fmt.Printf("%s%s %s%s\n", prefix, icon, name, metrics)

	// Print children
	childCount := len(node.Children)
	for i, child := range node.Children {
		isLast := i == childCount-1
		var childPrefix string
		if isLast {
			fmt.Printf("%s└─ ", prefix)
			childPrefix = prefix + "   "
		} else {
			fmt.Printf("%s├─ ", prefix)
			childPrefix = prefix + "│  "
		}
		printSpanNodeChild(child, childPrefix)
	}
}

// printSpanNodeChild prints a child node (already has prefix from parent)
func printSpanNodeChild(node *observatory.SpanHierarchyNode, prefix string) {
	// Build display line
	name := node.Name
	if node.ToolName != "" {
		name = node.ToolName
	}

	// Add metrics
	metrics := ""
	if node.DurationMs > 0 {
		if node.DurationMs >= 1000 {
			metrics += fmt.Sprintf(" %.1fs", float64(node.DurationMs)/1000)
		} else {
			metrics += fmt.Sprintf(" %dms", node.DurationMs)
		}
	}
	if node.CostUSD > 0 {
		metrics += fmt.Sprintf(" $%.4f", node.CostUSD)
	}
	if node.CacheReadTokens > 0 || node.CacheCreationTokens > 0 {
		metrics += fmt.Sprintf(" 📦%d/%d", node.CacheReadTokens, node.CacheCreationTokens)
	}

	// Add turn number for turn spans
	if node.TurnNumber > 0 {
		name = fmt.Sprintf("Turn #%d", node.TurnNumber)
		if len(node.Children) > 0 {
			name += fmt.Sprintf(" (%d tools)", len(node.Children))
		}
	}

	fmt.Printf("%s%s\n", name, metrics)

	// Print children
	childCount := len(node.Children)
	for i, child := range node.Children {
		isLast := i == childCount-1
		var childPrefix string
		if isLast {
			fmt.Printf("%s└─ ", prefix)
			childPrefix = prefix + "   "
		} else {
			fmt.Printf("%s├─ ", prefix)
			childPrefix = prefix + "│  "
		}
		printSpanNodeChild(child, childPrefix)
	}
}
