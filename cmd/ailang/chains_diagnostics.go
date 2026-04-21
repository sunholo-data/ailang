package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// chainsDiagnoseCommand provides a quick health report for a specific chain
// showing what went wrong, where it's stuck, and any data quality issues.
func chainsDiagnoseCommand() {
	fs := flag.NewFlagSet("chains diagnose", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains diagnose <chain-id>")
		fmt.Println()
		fmt.Println("Quick health report for a chain showing issues and stuck stages.")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve short ID prefix to full ID
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain: %v\n", err)
		os.Exit(1)
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "Error: chain not found: %s\n", chainID)
		os.Exit(1)
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	// Run diagnostics
	diag := runChainDiagnostics(ctx, backend, chain)

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(diag)
		return
	}

	// Print human-readable report
	printChainDiagnostics(diag)
}

// ChainDiagnostics contains diagnostic information about a chain
type ChainDiagnostics struct {
	ChainID      string             `json:"chain_id"`
	Status       string             `json:"status"`
	Duration     string             `json:"duration"`
	TotalCost    float64            `json:"total_cost"`
	TotalTurns   int                `json:"total_turns"`
	StageCount   int                `json:"stage_count"`
	Stages       []StageDiagnostics `json:"stages"`
	Issues       []string           `json:"issues"`
	IssueCount   int                `json:"issue_count"`
	HealthStatus string             `json:"health_status"` // healthy, warning, critical
}

// StageDiagnostics contains diagnostic information about a stage
type StageDiagnostics struct {
	StageNum       int      `json:"stage_num"`
	AgentID        string   `json:"agent_id"`
	Status         string   `json:"status"`
	SessionID      string   `json:"session_id,omitempty"`
	TaskID         string   `json:"task_id,omitempty"`
	Turns          int      `json:"turns"`
	ChatMessages   int      `json:"chat_messages"`
	ApprovalStatus string   `json:"approval_status,omitempty"`
	HandoffTo      string   `json:"handoff_to,omitempty"`
	WaitingTime    string   `json:"waiting_time,omitempty"`
	Issues         []string `json:"issues,omitempty"`
}

func runChainDiagnostics(ctx context.Context, backend *observatory.SQLiteBackend, chain *observatory.ExecutionChain) ChainDiagnostics {
	diag := ChainDiagnostics{
		ChainID:    chain.ID,
		Status:     string(chain.Status),
		TotalCost:  chain.TotalCost,
		TotalTurns: chain.TotalTurns,
		StageCount: len(chain.Stages),
		Issues:     []string{},
		Stages:     []StageDiagnostics{},
	}

	// Calculate duration
	if !chain.CreatedAt.IsZero() {
		var endTime time.Time
		if chain.CompletedAt != nil && !chain.CompletedAt.IsZero() {
			endTime = *chain.CompletedAt
		} else {
			endTime = time.Now()
		}
		diag.Duration = formatDurationHuman(endTime.Sub(chain.CreatedAt))
	}

	// Check each stage
	for i, stage := range chain.Stages {
		sd := StageDiagnostics{
			StageNum:       i + 1,
			AgentID:        stage.AgentID,
			Status:         string(stage.Status),
			SessionID:      stage.SessionID,
			TaskID:         stage.TaskID,
			Turns:          stage.Turns,
			ApprovalStatus: string(stage.ApprovalStatus),
			HandoffTo:      stage.HandoffTo,
			Issues:         []string{},
		}

		// Try to get chat messages for this stage
		if stage.TaskID != "" {
			messages := getChatMessagesForTask(stage.TaskID)
			sd.ChatMessages = len(messages)

			// Check for missing chat data
			if len(messages) == 0 && stage.Status != "pending" {
				issue := fmt.Sprintf("Stage %d: No chat messages linked (task_id: %s)", i+1, stage.TaskID)
				sd.Issues = append(sd.Issues, issue)
				diag.Issues = append(diag.Issues, issue)
			}
		} else if stage.SessionID != "" {
			// Fallback to session query
			messages := getChatMessages(stage.SessionID)
			sd.ChatMessages = len(messages)
		}

		// Check for stuck approvals
		if stage.Status == "awaiting_approval" || stage.ApprovalStatus == "pending" {
			if stage.CompletedAt != nil && !stage.CompletedAt.IsZero() {
				waitTime := time.Since(*stage.CompletedAt)
				sd.WaitingTime = formatDurationHuman(waitTime)
				if waitTime > time.Hour {
					issue := fmt.Sprintf("Stage %d: Approval pending for %s", i+1, sd.WaitingTime)
					sd.Issues = append(sd.Issues, issue)
					diag.Issues = append(diag.Issues, issue)
				}
			} else if stage.StartedAt != nil && !stage.StartedAt.IsZero() {
				waitTime := time.Since(*stage.StartedAt)
				sd.WaitingTime = formatDurationHuman(waitTime)
				if waitTime > 4*time.Hour {
					issue := fmt.Sprintf("Stage %d: Running for %s (may be stuck)", i+1, sd.WaitingTime)
					sd.Issues = append(sd.Issues, issue)
					diag.Issues = append(diag.Issues, issue)
				}
			}
		}

		// Check for failed stages
		if stage.Status == "failed" {
			issue := fmt.Sprintf("Stage %d: Failed (%s)", i+1, stage.AgentID)
			sd.Issues = append(sd.Issues, issue)
			diag.Issues = append(diag.Issues, issue)
		}

		// Check for missing session
		if stage.SessionID == "" && stage.Status != "pending" && stage.Status != "failed" {
			issue := fmt.Sprintf("Stage %d: No session ID recorded", i+1)
			sd.Issues = append(sd.Issues, issue)
			diag.Issues = append(diag.Issues, issue)
		}

		// Check for expected handoff that didn't happen
		if stage.HandoffTo != "" && stage.Status == "completed" {
			// Check if next stage exists
			nextStageExists := false
			for _, s := range chain.Stages {
				if s.AgentID == stage.HandoffTo && s.StageNumber > stage.StageNumber {
					nextStageExists = true
					break
				}
			}
			if !nextStageExists {
				issue := fmt.Sprintf("Stage %d: Handoff to %s expected but next stage not created", i+1, stage.HandoffTo)
				sd.Issues = append(sd.Issues, issue)
				diag.Issues = append(diag.Issues, issue)
			}
		}

		diag.Stages = append(diag.Stages, sd)
	}

	// Determine overall health
	diag.IssueCount = len(diag.Issues)
	if diag.IssueCount == 0 {
		diag.HealthStatus = "healthy"
	} else if diag.IssueCount <= 2 {
		diag.HealthStatus = "warning"
	} else {
		diag.HealthStatus = "critical"
	}

	return diag
}

func printChainDiagnostics(diag ChainDiagnostics) {
	// Header
	fmt.Printf("Chain: %s [%s]\n", truncateChainID(diag.ChainID), colorizeStatus(diag.Status))
	fmt.Printf("Duration: %s | Cost: $%.2f | Turns: %d\n", diag.Duration, diag.TotalCost, diag.TotalTurns)
	fmt.Println()

	// Stage details
	for _, stage := range diag.Stages {
		statusIcon := getStatusIcon(stage.Status)
		statusStr := stage.Status
		if stage.ApprovalStatus == "pending" {
			statusStr += " " + yellow("[awaiting approval]")
		}
		if len(stage.Issues) > 0 {
			statusStr += " " + red("ISSUES")
		}

		fmt.Printf("%s Stage %d: %s [%s]\n", statusIcon, stage.StageNum, stage.AgentID, statusStr)

		// Details line
		details := []string{}
		if stage.SessionID != "" {
			details = append(details, fmt.Sprintf("Session: %s", truncateChainID(stage.SessionID)))
		}
		if stage.Turns > 0 {
			details = append(details, fmt.Sprintf("Turns: %d", stage.Turns))
		}
		details = append(details, fmt.Sprintf("Chat: %d messages", stage.ChatMessages))
		if stage.WaitingTime != "" {
			details = append(details, fmt.Sprintf("Waiting: %s", stage.WaitingTime))
		}
		fmt.Printf("  %s\n", strings.Join(details, " | "))

		// Show handoff
		if stage.HandoffTo != "" {
			fmt.Printf("  → %s\n", stage.HandoffTo)
		}
		fmt.Println()
	}

	// Issues summary
	if len(diag.Issues) > 0 {
		fmt.Println(red("Issues Found:"))
		for _, issue := range diag.Issues {
			fmt.Printf("  • %s\n", issue)
		}
	} else {
		fmt.Println(green("✓ No issues detected"))
	}
}

// chainsHealthCommand provides system-wide data capture validation
func chainsHealthCommand() {
	fs := flag.NewFlagSet("chains health", flag.ExitOnError)
	hours := fs.Int("hours", 24, "Time window in hours")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Collect health metrics
	health := runSystemHealthCheck(ctx, backend, *hours)

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(health)
		return
	}

	// Print human-readable report
	printSystemHealth(health)
}

// SystemHealth contains system-wide health metrics
type SystemHealth struct {
	TimeWindow  string `json:"time_window"`
	GeneratedAt string `json:"generated_at"`

	// Chain metrics
	TotalChains     int `json:"total_chains"`
	CompletedChains int `json:"completed_chains"`
	ActiveChains    int `json:"active_chains"`
	FailedChains    int `json:"failed_chains"`
	PendingApproval int `json:"pending_approval"`

	// Stage metrics
	TotalStages       int     `json:"total_stages"`
	StagesWithSession int     `json:"stages_with_session"`
	SessionLinkRate   float64 `json:"session_link_rate"`

	// Chat message metrics
	TotalChatMessages  int     `json:"total_chat_messages"`
	MessagesWithTaskID int     `json:"messages_with_task_id"`
	TaskIDLinkRate     float64 `json:"task_id_link_rate"`

	// Session metrics
	TotalSessions     int     `json:"total_sessions"`
	SessionsWithChain int     `json:"sessions_with_chain"`
	ChainLinkRate     float64 `json:"chain_link_rate"`

	// Issues
	Issues       []string `json:"issues"`
	IssueCount   int      `json:"issue_count"`
	HealthStatus string   `json:"health_status"`

	// Recent activity
	LastChainCreated string `json:"last_chain_created,omitempty"`
	LastSessionStart string `json:"last_session_start,omitempty"`
	LastChatSync     string `json:"last_chat_sync,omitempty"`
}

func runSystemHealthCheck(ctx context.Context, backend *observatory.SQLiteBackend, hours int) SystemHealth {
	health := SystemHealth{
		TimeWindow:  fmt.Sprintf("last %d hours", hours),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Issues:      []string{},
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	// Get chains in time window
	chains, err := backend.ListChains(ctx, observatory.ChainListOptions{
		Limit: 1000,
	})
	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to list chains: %v", err))
	} else {
		for _, chain := range chains {
			if chain.CreatedAt.After(since) {
				health.TotalChains++
				switch chain.Status {
				case observatory.ChainStatusCompleted:
					health.CompletedChains++
				case observatory.ChainStatusActive:
					health.ActiveChains++
				case observatory.ChainStatusFailed:
					health.FailedChains++
				case observatory.ChainStatusPendingApproval:
					health.PendingApproval++
				}

				// Check stages
				stages, _ := backend.GetChainStages(ctx, chain.ID, observatory.ChainReadOptions{})
				for _, stage := range stages {
					health.TotalStages++
					if stage.SessionID != "" {
						health.StagesWithSession++
					}
				}
			}
		}

		// Update last chain created
		if len(chains) > 0 {
			health.LastChainCreated = chains[0].CreatedAt.Format(time.RFC3339)
		}
	}

	// Calculate session link rate
	if health.TotalStages > 0 {
		health.SessionLinkRate = float64(health.StagesWithSession) / float64(health.TotalStages) * 100
	}

	// Get chat message stats from database directly
	dbPath := filepath.Join(filepath.Dir(observatory.DefaultDatabasePath()), "observatory.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err == nil {
		defer db.Close()

		// Count chat messages and task_id linkage
		var totalMsgs, msgsWithTaskID int
		row := db.QueryRowContext(ctx, `
			SELECT
				COUNT(*),
				SUM(CASE WHEN task_id IS NOT NULL AND task_id != '' THEN 1 ELSE 0 END)
			FROM chat_messages
			WHERE created_at > ?
		`, since)
		row.Scan(&totalMsgs, &msgsWithTaskID)
		health.TotalChatMessages = totalMsgs
		health.MessagesWithTaskID = msgsWithTaskID
		if totalMsgs > 0 {
			health.TaskIDLinkRate = float64(msgsWithTaskID) / float64(totalMsgs) * 100
		}

		// Count sessions and chain linkage
		var totalSessions, sessionsWithChain int
		row = db.QueryRowContext(ctx, `
			SELECT
				COUNT(*),
				SUM(CASE WHEN chain_id IS NOT NULL AND chain_id != '' THEN 1 ELSE 0 END)
			FROM sessions
			WHERE created_at > ?
		`, since)
		row.Scan(&totalSessions, &sessionsWithChain)
		health.TotalSessions = totalSessions
		health.SessionsWithChain = sessionsWithChain
		if totalSessions > 0 {
			health.ChainLinkRate = float64(sessionsWithChain) / float64(totalSessions) * 100
		}

		// Get last session start
		var lastSession string
		db.QueryRowContext(ctx, `
			SELECT created_at FROM sessions ORDER BY created_at DESC LIMIT 1
		`).Scan(&lastSession)
		health.LastSessionStart = lastSession
	}

	// Identify issues
	if health.SessionLinkRate < 80 && health.TotalStages > 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("Low session link rate: %.1f%% (%d/%d stages)",
			health.SessionLinkRate, health.StagesWithSession, health.TotalStages))
	}

	if health.TaskIDLinkRate < 50 && health.TotalChatMessages > 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("Low task_id link rate: %.1f%% (%d/%d messages)",
			health.TaskIDLinkRate, health.MessagesWithTaskID, health.TotalChatMessages))
	}

	if health.FailedChains > 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("%d chains failed", health.FailedChains))
	}

	if health.PendingApproval > 3 {
		health.Issues = append(health.Issues, fmt.Sprintf("%d chains awaiting approval (backlog)", health.PendingApproval))
	}

	if health.TotalChains == 0 && hours >= 24 {
		health.Issues = append(health.Issues, "No chains created in time window")
	}

	// Determine overall health
	health.IssueCount = len(health.Issues)
	if health.IssueCount == 0 {
		health.HealthStatus = "healthy"
	} else if health.IssueCount <= 2 {
		health.HealthStatus = "warning"
	} else {
		health.HealthStatus = "critical"
	}

	return health
}

func printSystemHealth(health SystemHealth) {
	fmt.Printf("Chains Health Report (%s)\n", health.TimeWindow)
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	// Chain stats
	fmt.Println("Chain Status:")
	fmt.Printf("  Total:            %d\n", health.TotalChains)
	fmt.Printf("  Completed:        %s\n", green(fmt.Sprintf("%d", health.CompletedChains)))
	fmt.Printf("  Active:           %s\n", cyan(fmt.Sprintf("%d", health.ActiveChains)))
	if health.PendingApproval > 0 {
		fmt.Printf("  Pending Approval: %s\n", yellow(fmt.Sprintf("%d", health.PendingApproval)))
	}
	if health.FailedChains > 0 {
		fmt.Printf("  Failed:           %s\n", red(fmt.Sprintf("%d", health.FailedChains)))
	}
	fmt.Println()

	// Data capture stats
	fmt.Println("Data Capture:")
	fmt.Printf("  Stages:           %d total, %d with sessions (%.1f%%)\n",
		health.TotalStages, health.StagesWithSession, health.SessionLinkRate)
	fmt.Printf("  Chat Messages:    %d total, %d with task_id (%.1f%%)\n",
		health.TotalChatMessages, health.MessagesWithTaskID, health.TaskIDLinkRate)
	fmt.Printf("  Sessions:         %d total, %d with chain_id (%.1f%%)\n",
		health.TotalSessions, health.SessionsWithChain, health.ChainLinkRate)
	fmt.Println()

	// Recent activity
	fmt.Println("Recent Activity:")
	if health.LastChainCreated != "" {
		t, _ := time.Parse(time.RFC3339, health.LastChainCreated)
		fmt.Printf("  Last Chain:       %s ago\n", formatDurationHuman(time.Since(t)))
	}
	if health.LastSessionStart != "" {
		t, _ := time.Parse(time.RFC3339, health.LastSessionStart)
		fmt.Printf("  Last Session:     %s ago\n", formatDurationHuman(time.Since(t)))
	}
	fmt.Println()

	// Issues
	if len(health.Issues) > 0 {
		fmt.Println(red("Issues:"))
		for _, issue := range health.Issues {
			fmt.Printf("  ⚠ %s\n", issue)
		}
		// Historical context for low link rates
		if health.SessionLinkRate < 80 || health.TaskIDLinkRate < 50 || health.ChainLinkRate < 50 {
			fmt.Println()
			fmt.Println(dim("Note: Sessions before 2026-01-29 pre-date correlation ID tracking (M-DETERMINISTIC-CHAT-LINKING)."))
			fmt.Println(dim("      Only coordinator-executed sessions from that date onwards have task/chain/stage IDs."))
		}
	} else {
		fmt.Println(green("✓ All systems healthy"))
	}
}
