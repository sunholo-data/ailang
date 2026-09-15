package coordinator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/websocket"
)

// Environment variables the budget gate and provider resolver read
// (M-V1-SIMPLIFY-S4 M1). Both are documented in docs/docs/guides/debugging.md.
const (
	// EnvDefaultProvider names the provider a task is attributed to when
	// neither the agent's config nor the coordinator section's
	// default_provider says. Before M1 the literal "claude" was assumed at two
	// sites, which mislabelled provider and cost in the observatory and keyed
	// the wrong per-provider spend cap for every agent dispatched without a
	// declaration.
	EnvDefaultProvider = "AILANG_DEFAULT_PROVIDER"
	// EnvBudgetUnlimited, set to 1, acknowledges that tasks may run with NO
	// spend cap. Without it, every path where the cap disappears (no budgets
	// section, an unreadable config, a spend lookup that fails, or limits
	// that resolve to zero) warns once per process and, under
	// AILANG_STRICT_CONFIG=1, refuses the task instead of running it
	// unlimited. Before M1 each of those paths silently allowed the task.
	EnvBudgetUnlimited = "AILANG_BUDGET_UNLIMITED"
)

// taskProvider resolves the provider a task runs under: the agent's own
// declaration, then the coordinator config's default_provider, then
// AILANG_DEFAULT_PROVIDER, then the deprecated "claude" default through
// config.DeprecatedDefault — one stderr warning per process, and an error
// wrapping config.ErrDeprecatedDefault under AILANG_STRICT_CONFIG=1.
func (d *Daemon) taskProvider(agentConfig *AgentConfig) (string, error) {
	if agentConfig != nil && agentConfig.Provider != "" {
		return agentConfig.Provider, nil
	}
	if d.coordConfig != nil && d.coordConfig.DefaultProvider != "" {
		return d.coordConfig.DefaultProvider, nil
	}
	if v := strings.TrimSpace(os.Getenv(EnvDefaultProvider)); v != "" {
		return v, nil
	}
	return config.DeprecatedDefault(EnvDefaultProvider, "claude")
}

// checkBudgetBeforeExecution checks if the task can proceed within budget limits.
// Returns (blocked, error) where blocked=true means the task must not run now:
// either it waits for a cost approval, or (under AILANG_STRICT_CONFIG=1) it was
// refused because it would have run with no spend cap at all.
func (d *Daemon) checkBudgetBeforeExecution(ctx context.Context, task *TaskRecord, agentConfig *AgentConfig) (bool, error) {
	// Determine provider — the per-provider cap is keyed on it, so a guessed
	// provider is a guessed cap.
	provider, err := d.taskProvider(agentConfig)
	if err != nil {
		return d.refuseTask(ctx, task, err)
	}

	// Load budget configuration. An unreadable file used to mean "no
	// enforcement" — a parse error erased the cap.
	budgetsCfg, err := LoadBudgetsConfig()
	if err != nil || budgetsCfg == nil {
		return d.uncapped(ctx, task, provider, fmt.Sprintf("budget config unavailable: %v", err))
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

	// No limits configured at either level: the cap is gone.
	if dailyLimit == 0 && taskMaxLimit == 0 {
		return d.uncapped(ctx, task, provider, "no daily_budget or task_max_cost for the provider or globally")
	}

	// Get current spend by provider. A failed lookup used to allow the task —
	// a cap that cannot be checked is a cap that has disappeared.
	costByProvider, err := d.taskStore.GetCostByProvider()
	if err != nil {
		return d.uncapped(ctx, task, provider, fmt.Sprintf("spend lookup failed: %v", err))
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

// uncapped is every path on which a task would run with no spend cap. With
// AILANG_BUDGET_UNLIMITED=1 that is acknowledged and logged; otherwise it is
// the deprecated default — served with one stderr warning per process, and
// refused under AILANG_STRICT_CONFIG=1 (config.ErrDeprecatedDefault).
func (d *Daemon) uncapped(ctx context.Context, task *TaskRecord, provider, reason string) (bool, error) {
	d.logger.Printf("[BUDGET] no spend cap for provider %s on task %s: %s — set budgets.providers.%s.daily_budget "+
		"(or budgets.global.daily_budget) in %s, or %s=1 to acknowledge unlimited spend",
		provider, task.ID, reason, provider, config.FilePath(), EnvBudgetUnlimited)
	if os.Getenv(EnvBudgetUnlimited) == "1" {
		return false, nil
	}
	if _, err := config.DeprecatedDefault(EnvBudgetUnlimited, "1 (no spend cap)"); err != nil {
		return d.refuseTask(ctx, task, fmt.Errorf("task %s would run with no spend cap (%s): %w", task.ID, reason, err))
	}
	return false, nil
}

// refuseTask marks the task failed with err and reports it as blocked so the
// caller does not execute it. This is the strict-mode outcome: loud, terminal,
// and attributable to the unset variable in err.
func (d *Daemon) refuseTask(ctx context.Context, task *TaskRecord, err error) (bool, error) {
	d.logger.Printf("[BUDGET] REFUSED task %s: %v", task.ID, err)
	if mErr := d.taskStore.MarkTaskFailed(ctx, task.ID, err); mErr != nil {
		d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, mErr)
	}
	return true, err
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
