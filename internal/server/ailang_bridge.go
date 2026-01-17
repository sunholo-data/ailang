// Package server provides the HTTP server for the Collaboration Hub.
// ailang_bridge.go provides AILANG integration for dashboard transforms.
package server

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/embed"
	"github.com/sunholo/ailang/internal/eval"
)

// timeNow is a function variable for time.Now, allowing tests to mock time.
var timeNow = time.Now

// AILANGBridge provides AILANG-based event formatting as an alternative to Go.
// Enable with AILANG_DASHBOARD=1 environment variable.
type AILANGBridge struct {
	engine  *embed.Engine
	enabled bool
	mu      sync.RWMutex
}

var (
	ailangBridge     *AILANGBridge
	ailangBridgeOnce sync.Once
)

// GetAILANGBridge returns the singleton AILANG bridge instance.
func GetAILANGBridge() *AILANGBridge {
	ailangBridgeOnce.Do(func() {
		enabled := os.Getenv("AILANG_DASHBOARD") == "1"
		ailangBridge = &AILANGBridge{
			enabled: enabled,
		}
		if enabled {
			// Find AILANG project root (assumes server runs from project root or near it)
			basePath := os.Getenv("AILANG_PROJECT_ROOT")
			if basePath == "" {
				// Default to current working directory
				basePath, _ = os.Getwd()
			}
			ailangBridge.engine = embed.New(basePath)
			log.Printf("[AILANG] Dashboard bridge enabled (basePath=%s)", basePath)
		}
	})
	return ailangBridge
}

// IsEnabled returns true if the AILANG bridge is enabled.
func (b *AILANGBridge) IsEnabled() bool {
	if b == nil {
		return false
	}
	return b.enabled
}

// SummarizeEvents calls the AILANG summarizeEvents function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) SummarizeEvents(events []*coordinator.TaskEventRecord) string {
	if !b.IsEnabled() || b.engine == nil {
		return coordinator.SummarizeEvents(events)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert events to AILANG-compatible format
	ailangEvents := convertEventsForAILANG(events)

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"summarizeEvents",
		ailangEvents,
	)
	if err != nil {
		log.Printf("[AILANG] SummarizeEvents failed, falling back to Go: %v", err)
		return coordinator.SummarizeEvents(events)
	}

	str, err := embed.ToString(result)
	if err != nil {
		log.Printf("[AILANG] SummarizeEvents result conversion failed: %v", err)
		return coordinator.SummarizeEvents(events)
	}

	return str
}

// CountTurns calls the AILANG countTurns function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) CountTurns(events []*coordinator.TaskEventRecord) int {
	if !b.IsEnabled() || b.engine == nil {
		return coordinator.CountTurns(events)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	ailangEvents := convertEventsForAILANG(events)

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"countTurns",
		ailangEvents,
	)
	if err != nil {
		log.Printf("[AILANG] CountTurns failed, falling back to Go: %v", err)
		return coordinator.CountTurns(events)
	}

	count, err := embed.ToInt(result)
	if err != nil {
		log.Printf("[AILANG] CountTurns result conversion failed: %v", err)
		return coordinator.CountTurns(events)
	}

	return count
}

// Truncate calls the AILANG truncate function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) Truncate(text string, maxLen int) string {
	if !b.IsEnabled() || b.engine == nil {
		return goTruncate(text, maxLen)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(
		"internal/dashboard_transforms/event_formatter",
		"truncate",
		text,
		maxLen,
	)
	if err != nil {
		log.Printf("[AILANG] Truncate failed, falling back to Go: %v", err)
		return goTruncate(text, maxLen)
	}

	str, err := embed.ToString(result)
	if err != nil {
		log.Printf("[AILANG] Truncate result conversion failed: %v", err)
		return goTruncate(text, maxLen)
	}

	return str
}

// goTruncate is the Go fallback for truncate.
func goTruncate(text string, maxLen int) string {
	if maxLen == 0 || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// ailangEvent is the struct format expected by the AILANG event_formatter module.
type ailangEvent struct {
	TurnNum    int    `json:"turnNum"`
	StreamType string `json:"streamType"`
	Text       string `json:"text"`
}

// convertEventsForAILANG converts coordinator events to AILANG-compatible format.
func convertEventsForAILANG(events []*coordinator.TaskEventRecord) []ailangEvent {
	result := make([]ailangEvent, len(events))
	for i, e := range events {
		result[i] = ailangEvent{
			TurnNum:    e.TurnNum,
			StreamType: e.StreamType,
			Text:       e.Text,
		}
	}
	return result
}

// BuildHeatmapGrid calls the AILANG buildHeatmapGridAt function.
// Falls back to Go implementation on error.
func (b *AILANGBridge) BuildHeatmapGrid(cells []HeatmapCell, totalTasks int, totalCost float64, days int) HeatmapGridResponse {
	if !b.IsEnabled() || b.engine == nil {
		return buildHeatmapGrid(cells, totalTasks, totalCost, days)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert cells to AILANG-compatible format
	ailangCells := convertHeatmapCellsForAILANG(cells)

	// Use current time as Unix milliseconds for AILANG
	endTs := timeNow().UnixMilli()

	result, err := b.engine.Call(
		"internal/dashboard_transforms/heatmap",
		"buildHeatmapGridAt",
		ailangCells,
		totalTasks,
		totalCost,
		days,
		endTs,
	)
	if err != nil {
		log.Printf("[AILANG] BuildHeatmapGrid failed, falling back to Go: %v", err)
		return buildHeatmapGrid(cells, totalTasks, totalCost, days)
	}

	// Convert result back to Go types
	goResult, err := convertHeatmapResultFromAILANG(result)
	if err != nil {
		log.Printf("[AILANG] BuildHeatmapGrid result conversion failed: %v", err)
		return buildHeatmapGrid(cells, totalTasks, totalCost, days)
	}

	return goResult
}

// ailangHeatmapCell is the struct format expected by the AILANG heatmap module.
type ailangHeatmapCell struct {
	Date        string  `json:"date"`
	TaskCount   int     `json:"taskCount"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"`
}

// convertHeatmapCellsForAILANG converts server cells to AILANG-compatible format.
func convertHeatmapCellsForAILANG(cells []HeatmapCell) []ailangHeatmapCell {
	result := make([]ailangHeatmapCell, len(cells))
	for i, c := range cells {
		result[i] = ailangHeatmapCell(c)
	}
	return result
}

// convertHeatmapResultFromAILANG converts AILANG result to Go HeatmapGridResponse.
func convertHeatmapResultFromAILANG(result eval.Value) (HeatmapGridResponse, error) {
	goResult, err := embed.ToGo(result)
	if err != nil {
		return HeatmapGridResponse{}, err
	}

	resultMap, ok := goResult.(map[string]interface{})
	if !ok {
		return HeatmapGridResponse{}, fmt.Errorf("expected map result, got %T", goResult)
	}

	var response HeatmapGridResponse

	// Extract weeks
	if weeksRaw, ok := resultMap["weeks"]; ok {
		if weeksSlice, ok := weeksRaw.([]interface{}); ok {
			response.Weeks = make([][]HeatmapGridCell, len(weeksSlice))
			for i, weekRaw := range weeksSlice {
				if weekSlice, ok := weekRaw.([]interface{}); ok {
					response.Weeks[i] = make([]HeatmapGridCell, len(weekSlice))
					for j, cellRaw := range weekSlice {
						if cellMap, ok := cellRaw.(map[string]interface{}); ok {
							response.Weeks[i][j] = extractGridCell(cellMap)
						}
					}
				}
			}
		}
	}

	// Extract monthLabels
	if labelsRaw, ok := resultMap["monthLabels"]; ok {
		if labelsSlice, ok := labelsRaw.([]interface{}); ok {
			response.MonthLabels = make([]HeatmapMonthLabel, len(labelsSlice))
			for i, labelRaw := range labelsSlice {
				if labelMap, ok := labelRaw.(map[string]interface{}); ok {
					response.MonthLabels[i] = HeatmapMonthLabel{
						Name:      getString(labelMap, "name"),
						WeekIndex: getInt(labelMap, "weekIndex"),
					}
				}
			}
		}
	}

	// Extract totals
	response.Totals.Tasks = getInt(resultMap, "totalTasks")
	response.Totals.Cost = getFloat(resultMap, "totalCost")

	// Extract date range
	response.DateRange.Start = getString(resultMap, "startDate")
	response.DateRange.End = getString(resultMap, "endDate")

	return response, nil
}

func extractGridCell(m map[string]interface{}) HeatmapGridCell {
	return HeatmapGridCell{
		Date:        getString(m, "date"),
		TaskCount:   getInt(m, "count"),
		Cost:        getFloat(m, "cost"),
		SuccessRate: getFloat(m, "successRate"),
		Intensity:   getFloat(m, "intensity"),
		DayOfWeek:   getInt(m, "dayOfWeek"),
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0.0
}

// BudgetConfig represents budget configuration for task execution.
type BudgetConfig struct {
	WorkspaceBudget  float64                    `json:"workspaceBudget"`
	DailyBudget      float64                    `json:"dailyBudget"`
	TaskMaxCost      float64                    `json:"taskMaxCost"`
	WarningThreshold float64                    `json:"warningThreshold"`
	ProviderBudgets  map[string]*ProviderBudget `json:"providerBudgets,omitempty"` // Per-provider overrides
}

// ProviderBudget defines budget limits for a specific AI provider.
type ProviderBudget struct {
	DailyBudget      float64 `json:"dailyBudget" yaml:"daily_budget"`           // Per-provider daily limit
	TaskMaxCost      float64 `json:"taskMaxCost" yaml:"task_max_cost"`          // Per-provider task limit
	HardLimit        bool    `json:"hardLimit" yaml:"hard_limit"`               // Block if exceeded (vs warn only)
	WarningThreshold float64 `json:"warningThreshold" yaml:"warning_threshold"` // Override global threshold
}

// GetProviderBudget returns the budget for a specific provider, falling back to global limits.
func (c *BudgetConfig) GetProviderBudget(provider string) *ProviderBudget {
	if c.ProviderBudgets != nil {
		if pb, ok := c.ProviderBudgets[provider]; ok && pb != nil {
			// Fill in defaults from global config if not set
			result := &ProviderBudget{
				DailyBudget:      pb.DailyBudget,
				TaskMaxCost:      pb.TaskMaxCost,
				HardLimit:        pb.HardLimit,
				WarningThreshold: pb.WarningThreshold,
			}
			if result.DailyBudget == 0 {
				result.DailyBudget = c.DailyBudget
			}
			if result.TaskMaxCost == 0 {
				result.TaskMaxCost = c.TaskMaxCost
			}
			if result.WarningThreshold == 0 {
				result.WarningThreshold = c.WarningThreshold
			}
			return result
		}
	}
	// Return global limits as provider budget
	return &ProviderBudget{
		DailyBudget:      c.DailyBudget,
		TaskMaxCost:      c.TaskMaxCost,
		HardLimit:        false,
		WarningThreshold: c.WarningThreshold,
	}
}

// BudgetStatus represents the result of a budget check.
type BudgetStatus struct {
	Allowed            bool    `json:"allowed"`
	RemainingWorkspace float64 `json:"remainingWorkspace"`
	RemainingDaily     float64 `json:"remainingDaily"`
	WarningLevel       string  `json:"warningLevel"`
	Message            string  `json:"message"`
}

// CheckTaskBudget calls the AILANG checkTaskBudget function with contracts.
// Falls back to Go implementation on error.
// Demonstrates: requires contracts for input validation
func (b *AILANGBridge) CheckTaskBudget(config BudgetConfig, estimatedCost, workspaceSpend, dailySpend float64) BudgetStatus {
	if !b.IsEnabled() || b.engine == nil {
		return goCheckTaskBudget(config, estimatedCost, workspaceSpend, dailySpend)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"checkTaskBudget",
		config,
		estimatedCost,
		workspaceSpend,
		dailySpend,
	)
	if err != nil {
		log.Printf("[AILANG] CheckTaskBudget failed, falling back to Go: %v", err)
		return goCheckTaskBudget(config, estimatedCost, workspaceSpend, dailySpend)
	}

	status, err := convertBudgetStatusFromAILANG(result)
	if err != nil {
		log.Printf("[AILANG] CheckTaskBudget result conversion failed: %v", err)
		return goCheckTaskBudget(config, estimatedCost, workspaceSpend, dailySpend)
	}

	return status
}

// goCheckTaskBudget is the Go fallback for budget checking.
func goCheckTaskBudget(config BudgetConfig, estimatedCost, workspaceSpend, dailySpend float64) BudgetStatus {
	remainingWorkspace := config.WorkspaceBudget - workspaceSpend
	remainingDaily := config.DailyBudget - dailySpend
	minRemaining := remainingWorkspace
	if remainingDaily < minRemaining {
		minRemaining = remainingDaily
	}

	if estimatedCost > minRemaining {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task cost exceeds remaining budget",
		}
	}

	if estimatedCost > config.TaskMaxCost {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task exceeds maximum single-task cost",
		}
	}

	usageRatio := (dailySpend + estimatedCost) / config.DailyBudget
	level := "ok"
	if usageRatio > 0.9 {
		level = "critical"
	} else if usageRatio > config.WarningThreshold {
		level = "warning"
	}

	return BudgetStatus{
		Allowed:            true,
		RemainingWorkspace: remainingWorkspace - estimatedCost,
		RemainingDaily:     remainingDaily - estimatedCost,
		WarningLevel:       level,
		Message:            "Task approved",
	}
}

// convertBudgetStatusFromAILANG converts AILANG result to Go BudgetStatus.
func convertBudgetStatusFromAILANG(result eval.Value) (BudgetStatus, error) {
	goResult, err := embed.ToGo(result)
	if err != nil {
		return BudgetStatus{}, err
	}

	resultMap, ok := goResult.(map[string]interface{})
	if !ok {
		return BudgetStatus{}, fmt.Errorf("expected map result, got %T", goResult)
	}

	return BudgetStatus{
		Allowed:            getBool(resultMap, "allowed"),
		RemainingWorkspace: getFloat(resultMap, "remainingWorkspace"),
		RemainingDaily:     getFloat(resultMap, "remainingDaily"),
		WarningLevel:       getString(resultMap, "warningLevel"),
		Message:            getString(resultMap, "message"),
	}, nil
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// CostRecord represents a historical cost entry for burn rate calculation.
type CostRecord struct {
	Timestamp int64   `json:"timestamp"` // Unix milliseconds
	Cost      float64 `json:"cost"`
}

// CalculateBurnRate calls the AILANG calculateBurnRate function.
// Returns cost per hour based on recent spending within the time window.
// Falls back to Go implementation on error.
func (b *AILANGBridge) CalculateBurnRate(costs []CostRecord, windowMillis int64) float64 {
	if !b.IsEnabled() || b.engine == nil {
		return goCalculateBurnRate(costs, windowMillis)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert to AILANG-compatible format
	ailangCosts := make([]map[string]interface{}, len(costs))
	for i, c := range costs {
		ailangCosts[i] = map[string]interface{}{
			"timestamp": c.Timestamp,
			"cost":      c.Cost,
		}
	}

	result, err := b.engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"calculateBurnRate",
		ailangCosts,
		windowMillis,
	)
	if err != nil {
		log.Printf("[AILANG] CalculateBurnRate failed, falling back to Go: %v", err)
		return goCalculateBurnRate(costs, windowMillis)
	}

	rate, err := embed.ToFloat(result)
	if err != nil {
		log.Printf("[AILANG] CalculateBurnRate result conversion failed: %v", err)
		return goCalculateBurnRate(costs, windowMillis)
	}

	return rate
}

// goCalculateBurnRate is the Go fallback for burn rate calculation.
func goCalculateBurnRate(costs []CostRecord, windowMillis int64) float64 {
	if len(costs) == 0 || windowMillis <= 0 {
		return 0.0
	}
	var totalCost float64
	for _, c := range costs {
		totalCost += c.Cost
	}
	windowHours := float64(windowMillis) / 3600000.0
	return totalCost / windowHours
}

// ForecastExhaustion calls the AILANG forecastExhaustion function.
// Returns estimated hours until budget exhaustion, or -1 if burn rate is zero.
// Falls back to Go implementation on error.
func (b *AILANGBridge) ForecastExhaustion(remainingBudget, burnRate float64) int {
	if !b.IsEnabled() || b.engine == nil {
		return goForecastExhaustion(remainingBudget, burnRate)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"forecastExhaustion",
		remainingBudget,
		burnRate,
	)
	if err != nil {
		log.Printf("[AILANG] ForecastExhaustion failed, falling back to Go: %v", err)
		return goForecastExhaustion(remainingBudget, burnRate)
	}

	// Result is Option[int] - extract value or return -1 for None
	goResult, err := embed.ToGo(result)
	if err != nil {
		log.Printf("[AILANG] ForecastExhaustion result conversion failed: %v", err)
		return goForecastExhaustion(remainingBudget, burnRate)
	}

	// Handle Option type (Some has "value" key, None is empty or has no value)
	if resultMap, ok := goResult.(map[string]interface{}); ok {
		if _, exists := resultMap["value"]; exists {
			return getInt(resultMap, "value")
		}
		// Check for tag-based ADT representation
		if tag, exists := resultMap["_tag"]; exists {
			if tag == "Some" {
				return getInt(resultMap, "value")
			}
		}
	}

	return -1 // None case
}

// goForecastExhaustion is the Go fallback for exhaustion forecasting.
func goForecastExhaustion(remainingBudget, burnRate float64) int {
	if burnRate <= 0 {
		return -1 // Infinite / no burn
	}
	return int(remainingBudget / burnRate)
}

// Close shuts down the AILANG engine.
func (b *AILANGBridge) Close() error {
	if b.engine != nil {
		return b.engine.Close()
	}
	return nil
}
