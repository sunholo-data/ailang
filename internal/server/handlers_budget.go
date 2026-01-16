package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/observatory"
)

// BudgetStatusResponse provides budget status for the dashboard
type BudgetStatusResponse struct {
	Config      BudgetConfig              `json:"config"`
	Status      BudgetStatus              `json:"status"`
	Usage       BudgetUsage               `json:"usage"`
	BurnRate    BurnRateInfo              `json:"burnRate"`             // Burn rate and forecast
	ByProvider  map[string]*ProviderUsage `json:"byProvider,omitempty"` // Per-provider breakdown
	UsingAILANG bool                      `json:"usingAilang"`          // True if AILANG bridge is active
}

// BudgetUsage represents current budget usage
type BudgetUsage struct {
	WorkspaceSpend float64 `json:"workspaceSpend"`
	DailySpend     float64 `json:"dailySpend"`
	UsagePercent   float64 `json:"usagePercent"`
}

// BurnRateInfo represents burn rate and exhaustion forecast
type BurnRateInfo struct {
	CostPerHour          float64 `json:"costPerHour"`          // $/hour based on recent activity
	HoursUntilExhaustion int     `json:"hoursUntilExhaustion"` // -1 if no burn rate
	WindowHours          int     `json:"windowHours"`          // Time window used for calculation
}

// ProviderUsage represents budget usage for a specific provider
type ProviderUsage struct {
	Spend        float64 `json:"spend"`
	Budget       float64 `json:"budget"`
	UsagePercent float64 `json:"usagePercent"`
	WarningLevel string  `json:"warningLevel"`
	HardLimit    bool    `json:"hardLimit"`
}

// BudgetCheckRequest is the request body for checking task budget
type BudgetCheckRequest struct {
	EstimatedCost float64 `json:"estimatedCost"`
	Provider      string  `json:"provider,omitempty"` // Optional: specific provider to check against
}

// LoadBudgetConfig loads budget configuration from ~/.ailang/config.yaml
// and converts it to the server's BudgetConfig format.
func LoadBudgetConfig() BudgetConfig {
	cfg, err := coordinator.LoadBudgetsConfig()
	if err != nil {
		log.Printf("Failed to load budget config, using defaults: %v", err)
		return DefaultBudgetConfig()
	}

	config := BudgetConfig{
		WorkspaceBudget:  cfg.Global.WorkspaceBudget,
		DailyBudget:      cfg.Global.DailyBudget,
		TaskMaxCost:      cfg.Global.TaskMaxCost,
		WarningThreshold: cfg.Global.WarningThreshold,
	}

	// Convert provider budgets
	if len(cfg.Providers) > 0 {
		config.ProviderBudgets = make(map[string]*ProviderBudget)
		for name, limit := range cfg.Providers {
			config.ProviderBudgets[name] = &ProviderBudget{
				DailyBudget:      limit.DailyBudget,
				TaskMaxCost:      limit.TaskMaxCost,
				HardLimit:        limit.HardLimit,
				WarningThreshold: limit.WarningThreshold,
			}
		}
	}

	return config
}

// DefaultBudgetConfig returns a sensible default budget configuration
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		WorkspaceBudget:  100.0, // $100 workspace budget
		DailyBudget:      50.0,  // $50 daily budget
		TaskMaxCost:      25.0,  // $25 max per task
		WarningThreshold: 0.8,   // Warn at 80% usage
		ProviderBudgets: map[string]*ProviderBudget{
			"claude": {
				DailyBudget: 30.0,
				TaskMaxCost: 15.0,
				HardLimit:   true,
			},
			"gemini": {
				DailyBudget: 20.0,
				TaskMaxCost: 10.0,
				HardLimit:   false,
			},
		},
	}
}

// GET /api/budget/status - Get current budget status
// Demonstrates: AILANG dogfooding with contracts
func (s *Server) handleBudgetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current spend from observatory spans (same source as aggregations panel)
	// This ensures budget shows the same costs as the Control Plane aggregations
	var workspaceSpend, dailySpend float64
	costByProvider := make(map[string]float64)

	// Type assert to SQLiteBackend which has breakdown methods
	if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
		ctx := r.Context()

		// Get daily costs (today only)
		todayStr := time.Now().Format("2006-01-02")
		dailyFilter := &observatory.ControlPlaneFilter{
			StartDate: todayStr,
			EndDate:   todayStr,
		}
		if breakdown, err := sqliteBackend.GetFilteredBreakdownByProvider(ctx, dailyFilter); err == nil {
			for _, item := range breakdown {
				costByProvider[item.ID] = item.CostUSD
				dailySpend += item.CostUSD
			}
		}

		// Get workspace total (all time) - no time filter
		if breakdown, err := sqliteBackend.GetBreakdownByProvider(ctx); err == nil {
			for _, item := range breakdown {
				workspaceSpend += item.CostUSD
			}
		}
	} else if s.coordStore != nil {
		// Fallback to coordinator store if observatory not available
		stats, err := s.coordStore.GetCoordinatorStats()
		if err == nil && stats != nil {
			workspaceSpend = stats.TotalCost
			dailySpend = stats.TotalCost
		}
		costByProvider, _ = s.coordStore.GetCostByProvider()
	}

	config := LoadBudgetConfig()
	bridge := GetAILANGBridge()

	// Check budget status using AILANG (with Go fallback)
	status := bridge.CheckTaskBudget(config, 0, workspaceSpend, dailySpend)

	response := BudgetStatusResponse{
		Config:      config,
		Status:      status,
		UsingAILANG: bridge.IsEnabled(),
	}
	response.Usage.WorkspaceSpend = workspaceSpend
	response.Usage.DailySpend = dailySpend

	// Calculate usage percent (using simple Go calculation)
	if config.DailyBudget > 0 {
		response.Usage.UsagePercent = (dailySpend / config.DailyBudget) * 100
	}

	// Calculate burn rate from recent tasks (last 4 hours)
	windowHours := 4
	response.BurnRate = s.calculateBurnRate(bridge, config, windowHours)

	// Build per-provider breakdown
	if len(costByProvider) > 0 || len(config.ProviderBudgets) > 0 {
		response.ByProvider = make(map[string]*ProviderUsage)

		// Include all configured providers (even if spend is 0)
		for name, pb := range config.ProviderBudgets {
			spend := costByProvider[name]
			budget := pb.DailyBudget
			if budget == 0 {
				budget = config.DailyBudget
			}
			usagePercent := 0.0
			if budget > 0 {
				usagePercent = (spend / budget) * 100
			}
			warningThreshold := pb.WarningThreshold
			if warningThreshold == 0 {
				warningThreshold = config.WarningThreshold
			}
			level := "ok"
			if usagePercent >= 100 {
				level = "exceeded"
			} else if usagePercent/100 > 0.9 {
				level = "critical"
			} else if usagePercent/100 > warningThreshold {
				level = "warning"
			}

			response.ByProvider[name] = &ProviderUsage{
				Spend:        spend,
				Budget:       budget,
				UsagePercent: usagePercent,
				WarningLevel: level,
				HardLimit:    pb.HardLimit,
			}
		}

		// Also include providers with costs but no config (use global limits)
		// Skip empty provider names (legacy data)
		for name, spend := range costByProvider {
			if name == "" {
				continue // Skip empty provider names
			}
			if _, exists := response.ByProvider[name]; !exists {
				budget := config.DailyBudget
				usagePercent := 0.0
				if budget > 0 {
					usagePercent = (spend / budget) * 100
				}
				level := "ok"
				if usagePercent >= 100 {
					level = "exceeded"
				} else if usagePercent/100 > 0.9 {
					level = "critical"
				} else if usagePercent/100 > config.WarningThreshold {
					level = "warning"
				}
				response.ByProvider[name] = &ProviderUsage{
					Spend:        spend,
					Budget:       budget,
					UsagePercent: usagePercent,
					WarningLevel: level,
					HardLimit:    false,
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode budget status: %v", err)
	}
}

// POST /api/budget/check - Check if a task can proceed within budget
// Demonstrates: AILANG contracts (requires clause) with Go fallback
func (s *Server) handleBudgetCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BudgetCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current spend from observatory spans (same source as aggregations panel)
	var workspaceSpend, dailySpend, providerSpend float64

	// Type assert to SQLiteBackend which has breakdown methods
	if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
		ctx := r.Context()

		// Get daily costs (today only)
		todayStr := time.Now().Format("2006-01-02")
		dailyFilter := &observatory.ControlPlaneFilter{
			StartDate: todayStr,
			EndDate:   todayStr,
		}
		if breakdown, err := sqliteBackend.GetFilteredBreakdownByProvider(ctx, dailyFilter); err == nil {
			for _, item := range breakdown {
				dailySpend += item.CostUSD
				// Get provider-specific spend if provider is specified
				if req.Provider != "" && item.ID == req.Provider {
					providerSpend = item.CostUSD
				}
			}
		}

		// Get workspace total (all time)
		if breakdown, err := sqliteBackend.GetBreakdownByProvider(ctx); err == nil {
			for _, item := range breakdown {
				workspaceSpend += item.CostUSD
			}
		}
	} else if s.coordStore != nil {
		// Fallback to coordinator store if observatory not available
		stats, err := s.coordStore.GetCoordinatorStats()
		if err == nil && stats != nil {
			workspaceSpend = stats.TotalCost
			dailySpend = stats.TotalCost
		}
		if req.Provider != "" {
			costByProvider, _ := s.coordStore.GetCostByProvider()
			providerSpend = costByProvider[req.Provider]
		}
	}

	config := LoadBudgetConfig()
	bridge := GetAILANGBridge()

	// If provider specified, check against provider-specific limits
	if req.Provider != "" {
		pb := config.GetProviderBudget(req.Provider)
		// Create a modified config with provider-specific limits
		providerConfig := BudgetConfig{
			WorkspaceBudget:  config.WorkspaceBudget,
			DailyBudget:      pb.DailyBudget,
			TaskMaxCost:      pb.TaskMaxCost,
			WarningThreshold: pb.WarningThreshold,
		}
		status := bridge.CheckTaskBudget(providerConfig, req.EstimatedCost, workspaceSpend, providerSpend)

		// Add provider context to message
		if pb.HardLimit && !status.Allowed {
			status.Message = "[" + req.Provider + " hard limit] " + status.Message
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Printf("Failed to encode budget check: %v", err)
		}
		return
	}

	// No provider specified - use global limits
	status := bridge.CheckTaskBudget(config, req.EstimatedCost, workspaceSpend, dailySpend)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode budget check: %v", err)
	}
}

// calculateBurnRate calculates the cost per hour from recent tasks
func (s *Server) calculateBurnRate(bridge *AILANGBridge, config BudgetConfig, windowHours int) BurnRateInfo {
	info := BurnRateInfo{
		WindowHours:          windowHours,
		HoursUntilExhaustion: -1, // Default: no forecast
	}

	if s.coordStoreRaw == nil {
		return info
	}

	// Get recent tasks from the last N hours
	ctx := context.Background()
	windowStart := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	filter := &coordinator.TaskFilter{
		Since: &windowStart,
	}

	tasks, err := s.coordStoreRaw.ListTasks(ctx, filter)
	if err != nil {
		log.Printf("Failed to get recent tasks for burn rate: %v", err)
		return info
	}

	// Build cost records from tasks
	var costs []CostRecord
	for _, t := range tasks {
		if t.Cost > 0 {
			costs = append(costs, CostRecord{
				Timestamp: t.CreatedAt.UnixMilli(),
				Cost:      t.Cost,
			})
		}
	}

	if len(costs) == 0 {
		return info
	}

	// Calculate burn rate using AILANG bridge (with Go fallback)
	windowMillis := int64(windowHours) * 3600000
	info.CostPerHour = bridge.CalculateBurnRate(costs, windowMillis)

	// Calculate hours until exhaustion
	if info.CostPerHour > 0 {
		remainingBudget := config.DailyBudget - info.CostPerHour*float64(windowHours)
		if remainingBudget < 0 {
			remainingBudget = 0
		}
		info.HoursUntilExhaustion = bridge.ForecastExhaustion(remainingBudget, info.CostPerHour)
	}

	return info
}
