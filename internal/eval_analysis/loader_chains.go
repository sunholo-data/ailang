package eval_analysis

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// LoadResultsFromChain loads all benchmark results from a chain's stages,
// converting EvalAssessment data into BenchmarkResult format.
// This allows the entire downstream pipeline (GenerateMatrix, ExportBenchmarkJSON,
// FormatComparison) to work unchanged.
func LoadResultsFromChain(chainID string) ([]*BenchmarkResult, error) {
	store, err := observatory.OpenDefaultStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open observatory: %w", err)
	}

	ctx := context.Background()
	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID: chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query eval results: %w", err)
	}

	if len(stages) == 0 {
		return nil, fmt.Errorf("no eval stages found in chain %s", chainID)
	}

	var results []*BenchmarkResult
	for _, stage := range stages {
		if stage.EvalAssessment == nil {
			continue
		}
		results = append(results, stageToResult(stage))
	}

	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// LoadResultsFromLatestEvalChain finds the most recent eval_suite chain
// and loads its results.
func LoadResultsFromLatestEvalChain() ([]*BenchmarkResult, string, error) {
	store, err := observatory.OpenDefaultStore()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open observatory: %w", err)
	}

	ctx := context.Background()
	chains, err := store.ListEvalChains(ctx, 1)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list eval chains: %w", err)
	}
	if len(chains) == 0 {
		return nil, "", fmt.Errorf("no eval chains found")
	}

	chain := chains[0]
	results, err := LoadResultsFromChain(chain.ID)
	if err != nil {
		return nil, "", err
	}

	return results, chain.ID, nil
}

// LoadBaselineFromChain creates a Baseline from a chain's stages.
func LoadBaselineFromChain(chainID string) (*Baseline, error) {
	results, err := LoadResultsFromChain(chainID)
	if err != nil {
		return nil, err
	}

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open observatory: %w", err)
	}

	ctx := context.Background()
	chain, err := store.GetChain(ctx, chainID, observatory.ChainReadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get chain: %w", err)
	}

	baseline := &Baseline{
		Version:   chain.SourceRef,
		Timestamp: chain.CreatedAt,
		Results:   results,
	}

	// Calculate stats
	baseline.TotalBenchmarks = len(results)
	for _, r := range results {
		if r.StdoutOk {
			baseline.SuccessCount++
		} else {
			baseline.FailCount++
		}
	}

	// Extract model and language info
	models := map[string]bool{}
	langs := map[string]bool{}
	for _, r := range results {
		models[r.Model] = true
		langs[r.Lang] = true
	}
	if len(models) == 1 {
		for m := range models {
			baseline.Model = m
		}
	} else {
		baseline.Model = fmt.Sprintf("%d models", len(models))
	}
	langList := []string{}
	for l := range langs {
		langList = append(langList, l)
	}
	sort.Strings(langList)
	baseline.Languages = fmt.Sprintf("%v", langList)

	return baseline, nil
}

// stageToResult converts a ChainStage with EvalAssessment to a BenchmarkResult.
func stageToResult(stage *observatory.ChainStage) *BenchmarkResult {
	a := stage.EvalAssessment

	timestamp := time.Now()
	if stage.CompletedAt != nil {
		timestamp = *stage.CompletedAt
	} else if stage.StartedAt != nil {
		timestamp = *stage.StartedAt
	}

	return &BenchmarkResult{
		ID:             a.BenchmarkID,
		Lang:           a.Language,
		Model:          a.Model,
		Executor:       a.Executor,
		Seed:           a.Seed,
		CompileOk:      a.CompileOk,
		RuntimeOk:      a.RuntimeOk,
		StdoutOk:       a.StdoutOk,
		ErrorCategory:  a.ErrorCategory,
		FirstAttemptOk: a.FirstAttemptOk,
		RepairUsed:     a.RepairUsed,
		RepairOk:       a.RepairOk,
		ErrCode:        a.ErrCode,
		PromptVersion:  a.PromptVersion,
		Code:           a.Code,
		Stderr:         a.Stderr,
		Timestamp:      timestamp,

		// Metrics from chain stage (denormalized)
		CostUSD:      stage.Cost,
		InputTokens:  stage.TokensIn,
		OutputTokens: stage.TokensOut,
		TotalTokens:  stage.TokensIn + stage.TokensOut,
		DurationMs:   stage.DurationMs,

		// Agent metrics
		EvalMode:   a.EvalMode,
		AgentTurns: stage.Turns,
	}
}
