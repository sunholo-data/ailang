package coordinator

import (
	"context"

	"github.com/sunholo/ailang/internal/observatory"
)

// updateChainStageStatus updates the stage status in observatory (M-CHAINS-SIMPLIFY)
func (d *Daemon) updateChainStageStatus(ctx context.Context, task *TaskRecord, status observatory.ChainStageStatus) {
	if d.obsBackend == nil || task.StageID == "" {
		return
	}
	if err := d.obsBackend.UpdateStageStatus(ctx, task.StageID, status); err != nil {
		d.logger.Printf("Warning: Failed to update chain stage %s status to %s: %v", task.StageID, status, err)
	} else {
		d.logger.Printf("Updated chain stage %s status to %s", task.StageID, status)
	}
}

// updateChainStatus updates the chain status in observatory (M-CHAINS-SIMPLIFY)
func (d *Daemon) updateChainStatus(ctx context.Context, task *TaskRecord, status observatory.ChainStatus) {
	if d.obsBackend == nil || task.ChainID == "" {
		return
	}
	if err := d.obsBackend.UpdateChainStatus(ctx, task.ChainID, status); err != nil {
		d.logger.Printf("Warning: Failed to update chain %s status to %s: %v", task.ChainID, status, err)
	} else {
		d.logger.Printf("Updated chain %s status to %s", task.ChainID, status)
	}
}

// updateStageSession links a chain stage to its Claude/Gemini session ID in observatory
func (d *Daemon) updateStageSession(ctx context.Context, task *TaskRecord, sessionID string) {
	if d.obsBackend == nil || task.StageID == "" || sessionID == "" {
		return
	}
	if err := d.obsBackend.UpdateStageSession(ctx, task.StageID, sessionID); err != nil {
		d.logger.Printf("Warning: Failed to link stage %s to session %s: %v", task.StageID, sessionID, err)
	} else {
		d.logger.Printf("Linked stage %s to session %s", task.StageID, sessionID)
	}
}

// updateStageMetrics records cost/token/turn metrics on a chain stage (M-CHAINS-SOURCE-OF-TRUTH)
func (d *Daemon) updateStageMetrics(ctx context.Context, task *TaskRecord, result *ExecuteResult) {
	if d.obsBackend == nil || task.StageID == "" || result == nil {
		return
	}
	durationMs := result.Duration.Milliseconds()
	if err := d.obsBackend.UpdateStageMetrics(ctx, task.StageID, result.Cost, result.InputTokens, result.OutputTokens, result.NumTurns, result.ToolCallCount, durationMs); err != nil {
		d.logger.Printf("Warning: Failed to update stage %s metrics: %v", task.StageID, err)
	}
}

// updateStageError records the error message on a failed stage.
func (d *Daemon) updateStageError(ctx context.Context, task *TaskRecord, errorMsg string) {
	if d.obsBackend == nil || task.StageID == "" || errorMsg == "" {
		return
	}
	if err := d.obsBackend.UpdateStageError(ctx, task.StageID, errorMsg); err != nil {
		d.logger.Printf("Warning: Failed to update stage %s error: %v", task.StageID, err)
	}
}

// updateChainMetrics rolls up cost/token metrics to the chain level (M-CHAINS-SOURCE-OF-TRUTH)
func (d *Daemon) updateChainMetrics(ctx context.Context, task *TaskRecord, result *ExecuteResult) {
	if d.obsBackend == nil || task.ChainID == "" || result == nil {
		return
	}
	if err := d.obsBackend.UpdateChainMetrics(ctx, task.ChainID, result.Cost, result.TokensUsed, 0); err != nil {
		d.logger.Printf("Warning: Failed to update chain %s metrics: %v", task.ChainID, err)
	}
}
