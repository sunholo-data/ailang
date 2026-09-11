// Package server provides the HTTP server for the Collaboration Hub.
// ailang_bridge.go provides AILANG integration for dashboard transforms.
package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// timeNow is a function variable for time.Now, allowing tests to mock time.
var timeNow = time.Now

// AILANGBridge runs the dashboard's transforms, which are written in AILANG.
//
// It used to be a SWITCH between two implementations: AILANG when
// AILANG_DASHBOARD=1, a Go copy otherwise, plus a silent fall back to that copy
// whenever an AILANG call errored. Removed 2026-09-08 on Mark's instruction,
// because a second implementation of a rule is not a safety net:
//
//   - the copies were free to drift, and nothing compared them;
//   - the substitution was visible only in a log line nobody reads, so a broken
//     AILANG path looked exactly like a working one;
//   - `budget_checker` decides SPENDING. CLAUDE.md Principle 2 puts business
//     logic explicitly outside what may be quietly defaulted;
//   - and the flag was off in production while the deployed image shipped no
//     `.ail` files at all — so the AILANG path had never run there, and the
//     "fallback" was in fact the only thing that had ever executed.
//
// Now there is one implementation. Every method returns an error the caller must
// handle, and a transform that cannot run says so.
type AILANGBridge struct {
	engine *embed.Engine
	err    error
	mu     sync.RWMutex
}

var (
	ailangBridge     *AILANGBridge
	ailangBridgeOnce sync.Once
)

// GetAILANGBridge returns the singleton AILANG bridge instance.
func GetAILANGBridge() *AILANGBridge {
	ailangBridgeOnce.Do(func() {
		// Resolve against a module we actually call, so a tree that is present
		// but missing the transforms fails here rather than on first request.
		eng, err := embed.NewForModule(moduleEventFormatter)
		ailangBridge = &AILANGBridge{engine: eng, err: err}
		if err != nil {
			log.Printf("[AILANG] dashboard transforms UNAVAILABLE: %v", err)
			return
		}
		for _, m := range []string{moduleHeatmap, moduleBudgetChecker} {
			if lErr := eng.Load(m); lErr != nil {
				ailangBridge.err = fmt.Errorf("loading %s: %w", m, lErr)
				log.Printf("[AILANG] dashboard transforms UNAVAILABLE: %v", ailangBridge.err)
				return
			}
		}
		log.Printf("[AILANG] dashboard transforms ready")
	})
	return ailangBridge
}

// AILANG module paths, named once so a typo is a compile error rather than a
// runtime miss that used to be absorbed by a fallback.
const (
	moduleEventFormatter = "internal/dashboard_transforms/event_formatter"
	moduleHeatmap        = "internal/dashboard_transforms/heatmap"
	moduleBudgetChecker  = "internal/dashboard_transforms/budget_checker"
)

// Ready reports whether the transforms can run, and why not if they cannot.
//
// Replaces IsEnabled. The old name asked whether a FEATURE was switched on; the
// question that matters now is whether the only implementation is reachable.
func (b *AILANGBridge) Ready() error {
	if b == nil {
		return fmt.Errorf("AILANG bridge not initialised")
	}
	if b.err != nil {
		return b.err
	}
	if b.engine == nil {
		return fmt.Errorf("AILANG engine not loaded")
	}
	return nil
}

// SummarizeEvents calls the AILANG summarizeEvents function.
func (b *AILANGBridge) SummarizeEvents(events []*coordinator.TaskEventRecord) (string, error) {
	if err := b.Ready(); err != nil {
		return "", err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "summarizeEvents", convertEventsForAILANG(events))
	if err != nil {
		return "", fmt.Errorf("summarizeEvents: %w", err)
	}
	str, err := embed.ToString(result)
	if err != nil {
		return "", fmt.Errorf("summarizeEvents result: %w", err)
	}
	return str, nil
}

// CountTurns calls the AILANG countTurns function.
func (b *AILANGBridge) CountTurns(events []*coordinator.TaskEventRecord) (int, error) {
	if err := b.Ready(); err != nil {
		return 0, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "countTurns", convertEventsForAILANG(events))
	if err != nil {
		return 0, fmt.Errorf("countTurns: %w", err)
	}
	count, err := embed.ToInt(result)
	if err != nil {
		return 0, fmt.Errorf("countTurns result: %w", err)
	}
	return count, nil
}

// Truncate calls the AILANG truncate function.
func (b *AILANGBridge) Truncate(text string, maxLen int) (string, error) {
	if err := b.Ready(); err != nil {
		return "", err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.Call(moduleEventFormatter, "truncate", text, maxLen)
	if err != nil {
		return "", fmt.Errorf("truncate: %w", err)
	}
	str, err := embed.ToString(result)
	if err != nil {
		return "", fmt.Errorf("truncate result: %w", err)
	}
	return str, nil
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
func (b *AILANGBridge) BuildHeatmapGrid(cells []HeatmapCell, totalTasks int, totalCost float64, days int) (HeatmapGridResponse, error) {
	if err := b.Ready(); err != nil {
		return HeatmapGridResponse{}, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.CallPreserveFloats(moduleHeatmap, "buildHeatmapGridAt",
		convertHeatmapCellsForAILANG(cells), totalTasks, totalCost, days, timeNow().UnixMilli())
	if err != nil {
		return HeatmapGridResponse{}, fmt.Errorf("buildHeatmapGridAt: %w", err)
	}
	grid, err := convertHeatmapResultFromAILANG(result)
	if err != nil {
		return HeatmapGridResponse{}, fmt.Errorf("buildHeatmapGridAt result: %w", err)
	}
	return grid, nil
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
//
// This one decides SPENDING, which is why it may not quietly default. A caller
// that cannot evaluate the budget must refuse the spend, not invent an answer.
func (b *AILANGBridge) CheckTaskBudget(config BudgetConfig, estimatedCost, workspaceSpend, dailySpend float64) (BudgetStatus, error) {
	if err := b.Ready(); err != nil {
		return BudgetStatus{}, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.CallPreserveFloats(moduleBudgetChecker, "checkTaskBudget",
		config, estimatedCost, workspaceSpend, dailySpend)
	if err != nil {
		return BudgetStatus{}, fmt.Errorf("checkTaskBudget: %w", err)
	}
	status, err := convertBudgetStatusFromAILANG(result)
	if err != nil {
		return BudgetStatus{}, fmt.Errorf("checkTaskBudget result: %w", err)
	}
	return status, nil
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
func (b *AILANGBridge) CalculateBurnRate(costs []CostRecord, windowMillis int64) (float64, error) {
	if err := b.Ready(); err != nil {
		return 0, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	ailangCosts := make([]map[string]interface{}, len(costs))
	for i, c := range costs {
		ailangCosts[i] = map[string]interface{}{"timestamp": c.Timestamp, "cost": c.Cost}
	}

	result, err := b.engine.CallPreserveFloats(moduleBudgetChecker, "calculateBurnRate", ailangCosts, windowMillis)
	if err != nil {
		return 0, fmt.Errorf("calculateBurnRate: %w", err)
	}
	rate, err := embed.ToFloat(result)
	if err != nil {
		return 0, fmt.Errorf("calculateBurnRate result: %w", err)
	}
	return rate, nil
}

// ForecastExhaustion calls the AILANG forecastExhaustion function.
// Returns estimated hours until budget exhaustion, or -1 for None (no burn).
func (b *AILANGBridge) ForecastExhaustion(remainingBudget, burnRate float64) (int, error) {
	if err := b.Ready(); err != nil {
		return 0, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	result, err := b.engine.CallPreserveFloats(moduleBudgetChecker, "forecastExhaustion", remainingBudget, burnRate)
	if err != nil {
		return 0, fmt.Errorf("forecastExhaustion: %w", err)
	}
	goResult, err := embed.ToGo(result)
	if err != nil {
		return 0, fmt.Errorf("forecastExhaustion result: %w", err)
	}

	// Option[int]: Some carries "value"; None is the -1 sentinel this API uses.
	if resultMap, ok := goResult.(map[string]interface{}); ok {
		if _, exists := resultMap["value"]; exists {
			return getInt(resultMap, "value"), nil
		}
		if tag, exists := resultMap["_tag"]; exists && tag == "Some" {
			return getInt(resultMap, "value"), nil
		}
	}
	return -1, nil
}

// Close shuts down the AILANG engine.
func (b *AILANGBridge) Close() error {
	if b.engine != nil {
		return b.engine.Close()
	}
	return nil
}
