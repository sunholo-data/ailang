package main

import (
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// buildCloudCompletion preserves input evidence on success and failure, including
// failures without an executor result. Delivery status never erases provenance.
func buildCloudCompletion(taskID, agentID, status, errMsg, branchName string, execResult *executor.Result, ev gitEvidence, artifactPath string) pubsub.TaskCompletion {
	completion := pubsub.TaskCompletion{
		TaskID:       taskID,
		AgentID:      agentID,
		Status:       status,
		ErrorMsg:     errMsg,
		BranchName:   branchName,
		ChangedFiles: ev.ChangedFiles,
		// Approval evidence (M3). Two immutable SHAs, so the card renders
		// identically however many times this completion is delivered.
		BaseCommit:      ev.BaseCommit,
		HeadCommit:      ev.HeadCommit,
		DiffStat:        ev.DiffStat,
		Diff:            ev.Diff,
		ArtifactGCSPath: artifactPath,
	}
	// Populate executor metrics when available (same data as local coordinator)
	if execResult != nil {
		completion.Summary = completionSummary(execResult.Transcript)
		completion.SessionID = execResult.SessionID
		completion.NumTurns = execResult.NumTurns
		completion.ToolCallCount = execResult.ToolCallCount
		completion.InputTokens = execResult.InputTokens
		completion.OutputTokens = execResult.OutputTokens
		completion.CostUSD = execResult.CostUSD
		completion.DurationMS = execResult.DurationMS
		completion.CacheReadTokens = execResult.CacheReadInputTokens
		completion.CacheCreationTokens = execResult.CacheCreationInputTokens
	}
	completion.Summary = appendInputReport(completion.Summary, ev.Inputs)
	return completion
}
