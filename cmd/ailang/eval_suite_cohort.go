package main

// Cohort plumbing for `ailang eval-suite` (M-EVAL-CHAINS + M-COST-PER-SUCCESS-KPI M4a).
//
// This file owns the WRITE side of the cost-per-verified-success cohort: the
// execution-chain creation that stamps `chains.source_ref`, and (M4a) the
// `--baseline` freeze mechanism plus the recorded cohort manifest.
//
// It was extracted from eval_suite.go, which sat at 790/800 lines against the
// hard `make check-file-sizes` CI gate — adding the flag inline would have
// broken CI.

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// evalChainParams groups the inputs needed to open the eval execution chain.
type evalChainParams struct {
	taskID     string
	evalMode   string // "standard" | "agent"
	conditions []string
}

// evalModeName maps the --agent flag to the canonical eval-mode label used in
// source_ref, the banked assessment, and the cohort manifest.
func evalModeName(agent bool) string {
	if agent {
		return "agent"
	}
	return "standard"
}

// conditionRef renders the single-condition suffix used in source_ref
// ("/full", "/baseline", …). Empty for the legacy no-condition run and for
// multi-condition runs (which fan out into one chain covering all conditions).
func conditionRef(conditions []string) string {
	if len(conditions) == 1 && conditions[0] != "" {
		return "/" + conditions[0]
	}
	return ""
}

// createEvalChain opens the M-EVAL-CHAINS execution chain for an eval-suite run.
// Each benchmark × model × language × condition later becomes a chain stage.
//
// Returns nil (with a printed warning) when the observatory is unavailable or
// the chain cannot be created — eval results still land as JSON files, so this
// is an observability degradation, not a data-integrity failure.
func createEvalChain(ctx context.Context, p evalChainParams) *EvalChainContext {
	obsStore, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Warning: Could not open observatory database: %v\n", yellow("⚠️"), err)
		fmt.Fprintf(os.Stderr, "   Eval results will be stored as JSON files only\n")
		return nil
	}

	cwd, _ := os.Getwd()
	chain, err := obsStore.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType:    observatory.ChainSourceEvalSuite,
		SourceRef:     fmt.Sprintf("%s/%s%s", p.taskID, p.evalMode, conditionRef(p.conditions)),
		WorkspacePath: cwd,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Warning: Could not create eval chain: %v\n", yellow("⚠️"), err)
		return nil
	}

	fmt.Printf("  Chain ID: %s\n", chain.ID[:8])
	return &EvalChainContext{
		Store:   obsStore,
		ChainID: chain.ID,
	}
}
