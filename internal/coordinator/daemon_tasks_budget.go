package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
)

// checkBudgetBeforeExecution checks if the task can proceed within budget limits.
// Returns (blocked, error) where blocked=true means task should wait for approval.
func (d *Daemon) checkBudgetBeforeExecution(ctx context.Context, task *TaskRecord, agentConfig *AgentConfig) (bool, error) {
	// Load budget configuration
	budgetsCfg, err := LoadBudgetsConfig()
	if err != nil || budgetsCfg == nil {
		// No budget config = no enforcement
		d.logger.Printf("[DEBUG] No budget config found, skipping budget check")
		return false, nil
	}

	// Determine provider
	provider := "claude" // default
	if agentConfig != nil && agentConfig.Provider != "" {
		provider = agentConfig.Provider
	} else if d.coordConfig != nil && d.coordConfig.DefaultProvider != "" {
		provider = d.coordConfig.DefaultProvider
	}

	// Get provider-specific limits
	var dailyLimit, taskMaxLimit float64
	var hardLimit bool

	if budgetsCfg.Providers != nil {
		if providerCfg, ok := budgetsCfg.Providers[provider]; ok && providerCfg != nil {
			dailyLimit = providerCfg.DailyBudget
			taskMaxLimit = providerCfg.TaskMaxCost
			hardLimit = providerCfg.HardLimit
		}
	}

	// Fall back to global limits
	if dailyLimit == 0 && budgetsCfg.Global != nil {
		dailyLimit = budgetsCfg.Global.DailyBudget
	}
	if taskMaxLimit == 0 && budgetsCfg.Global != nil {
		taskMaxLimit = budgetsCfg.Global.TaskMaxCost
	}

	// If no limits configured, allow the task
	if dailyLimit == 0 && taskMaxLimit == 0 {
		return false, nil
	}

	// Get current spend by provider
	costByProvider, err := d.taskStore.GetCostByProvider()
	if err != nil {
		d.logger.Printf("Warning: Failed to get cost by provider: %v", err)
		return false, nil // Allow task if we can't check budget
	}

	currentSpend := costByProvider[provider]
	d.logger.Printf("[BUDGET] Provider %s: current spend $%.2f, daily limit $%.2f, hard=%v",
		provider, currentSpend, dailyLimit, hardLimit)

	// Check if already over budget
	if dailyLimit > 0 && currentSpend >= dailyLimit {
		d.logger.Printf("[BUDGET] Provider %s: daily budget exceeded ($%.2f >= $%.2f)",
			provider, currentSpend, dailyLimit)

		if hardLimit {
			// Create cost approval request
			return d.createBudgetApproval(ctx, task, provider, currentSpend, dailyLimit)
		}
		// Soft limit - log warning and continue
		d.logger.Printf("[BUDGET] WARNING: Provider %s over budget but soft limit, continuing", provider)
	}

	return false, nil
}

// createBudgetApproval creates an ApprovalTypeCost approval request for budget-blocked tasks.
func (d *Daemon) createBudgetApproval(ctx context.Context, task *TaskRecord, provider string, currentSpend, limit float64) (bool, error) {
	d.logger.Printf("[BUDGET] Creating cost approval for task %s (provider %s)", task.ID, provider)

	// Mark task as pending approval
	if err := d.taskStore.MarkTaskPendingApproval(ctx, task.ID, "", "", "", "", nil); err != nil {
		d.logger.Printf("Warning: Failed to mark task as pending approval: %v", err)
	}

	// Create approval request with type "cost"
	approvalReq := &ApprovalRequestRecord{
		ID:        fmt.Sprintf("apr-cost-%s", task.ID),
		TaskID:    task.ID,
		Type:      string(ApprovalTypeCost),
		Status:    "pending",
		CreatedAt: time.Now(),
		ContextJSON: fmt.Sprintf(`{
			"provider": %q,
			"current_spend": %.2f,
			"daily_limit": %.2f,
			"reason": "Daily budget limit exceeded for provider %s"
		}`, provider, currentSpend, limit, provider),
	}

	if err := d.taskStore.CreateApprovalRequest(ctx, approvalReq); err != nil {
		d.logger.Printf("Warning: Failed to create cost approval request: %v", err)
		return false, err
	}

	// Post status update
	d.postTaskStatus(task, "budget_blocked",
		fmt.Sprintf("Task blocked: %s budget limit exceeded ($%.2f/$%.2f). Waiting for approval.",
			provider, currentSpend, limit))

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     task.ID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "budget_blocked",
			Text: fmt.Sprintf("Budget limit exceeded for %s ($%.2f/$%.2f)",
				provider, currentSpend, limit),
		})
	}

	return true, nil // Task blocked
}
