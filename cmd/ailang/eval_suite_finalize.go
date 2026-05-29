package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// suiteSummaryParams groups the parameters needed for finalizeSuiteRun.
type suiteSummaryParams struct {
	ctx         context.Context
	results     []SuiteResult
	outputDir   string
	totalRuns   int
	trialsToRun int
	taskID      string
	evalStore   messaging.MessageStore
	evalChain   *EvalChainContext
	suiteSpan   oteltrace.Span
	duration    time.Duration
}

// finalizeSuiteRun reports results, emits completion events, updates Observatory
// chain metrics, and marks the eval task as complete. It is called after
// runBenchmarksParallel returns in the direct (non-queue) execution path.
func finalizeSuiteRun(p suiteSummaryParams) {
	successCount := 0
	failCount := 0
	for _, r := range p.results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	// Record suite results on span
	p.suiteSpan.SetAttributes(
		attribute.Int("eval.success_count", successCount),
		attribute.Int("eval.fail_count", failCount),
		attribute.Int64("eval.duration_ms", p.duration.Milliseconds()),
		attribute.Float64("eval.success_rate", float64(successCount)/float64(p.totalRuns)*100),
	)
	if failCount > 0 {
		p.suiteSpan.SetStatus(codes.Error, fmt.Sprintf("%d/%d benchmarks failed", failCount, p.totalRuns))
	} else {
		p.suiteSpan.SetStatus(codes.Ok, "all benchmarks passed")
	}

	fmt.Println()
	fmt.Printf("%s Benchmark suite complete!\n", green("✓"))
	fmt.Printf("Duration: %s\n", p.duration.Round(time.Second))
	fmt.Printf("Success: %d/%d (%.1f%%)\n", successCount, p.totalRuns, float64(successCount)/float64(p.totalRuns)*100)
	fmt.Printf("Failed:  %d/%d\n", failCount, p.totalRuns)
	fmt.Println()

	// M-EVAL-OS-LONGITUDINAL Phase 3: write summary.json that aggregates
	// per-(benchmark, model, lang, condition) pass rate and token distribution
	// across trials. Required for Phase 4 candidates command + Phase 5
	// publication. Best-effort — log on error but don't fail the suite.
	if rs, err := eval_harness.SummarizeRotation(p.outputDir); err != nil {
		fmt.Printf("Note: failed to write summary.json: %v\n", err)
	} else if p.trialsToRun > 1 {
		fmt.Printf("Summary: aggregated %d result files into %s/summary.json\n",
			rs.TotalResultFiles, p.outputDir)
	}

	fmt.Println("Results:")
	fmt.Printf("  - JSON: %s/*.json\n", p.outputDir)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  ailang eval-summary %s\n", p.outputDir)
	fmt.Printf("  ailang eval-matrix %s v0.3.0\n", p.outputDir)

	// Create "Suite Completed" message for event queue visibility
	if p.evalStore != nil {
		status := "completed"
		if failCount > 0 {
			status = "partial"
		}

		completePayload := map[string]interface{}{
			"task_id":      p.taskID,
			"success":      successCount,
			"failed":       failCount,
			"total":        p.totalRuns,
			"duration_sec": p.duration.Seconds(),
			"success_rate": float64(successCount) / float64(p.totalRuns) * 100,
		}
		payloadBytes, _ := json.Marshal(completePayload)

		completeMsg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       "controlplane",
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Eval Suite %s: %d/%d passed (%.1f%%)", status, successCount, p.totalRuns, float64(successCount)/float64(p.totalRuns)*100),
			Payload:       string(payloadBytes),
			Category:      "eval",
			CorrelationID: p.taskID,
		}
		if err := p.evalStore.InsertInboxMessageWithContext(p.ctx, completeMsg); err == nil {
			// Broadcast to dashboard via HTTP (non-blocking)
			go broadcastEvalEvent(completeMsg)
			// Emit span so event appears in ExecHierarchy (Milestone 12)
			emitEventSpan(p.ctx, "suite_completed", p.taskID, completeMsg)
		}
	}

	// Update task status to completed
	completeEvalTask(p.taskID, failCount == 0)

	// M-EVAL-CHAINS: Finalize chain status and roll up metrics from stages
	if p.evalChain != nil {
		// Use "partial" status if mixed results, "completed" if all pass, "failed" if all fail
		chainStatus := observatory.ChainStatusCompleted
		if failCount > 0 && successCount > 0 {
			chainStatus = observatory.ChainStatusCompleted // Mixed — still "completed" (assessment has details)
		} else if failCount > 0 {
			chainStatus = observatory.ChainStatusFailed
		}
		if err := p.evalChain.Store.UpdateChainStatus(p.ctx, p.evalChain.ChainID, chainStatus); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update eval chain status: %v\n", err)
		}

		// Roll up cost/tokens/turns from stages to chain
		stages, stageErr := p.evalChain.Store.GetChainStages(p.ctx, p.evalChain.ChainID, observatory.ChainReadOptions{})
		if stageErr == nil {
			var totalCost float64
			var totalTokens, totalTurns int
			for _, st := range stages {
				totalCost += st.Cost
				totalTokens += st.TokensIn + st.TokensOut
				totalTurns += st.Turns
			}
			_ = p.evalChain.Store.UpdateChainMetrics(p.ctx, p.evalChain.ChainID, totalCost, totalTokens, totalTurns)
		}

		fmt.Printf("  Chain: ailang chains view %s\n", p.evalChain.ChainID[:8])
	}
}
