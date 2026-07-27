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
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// flagWasSet reports whether a flag was EXPLICITLY passed, distinguishing
// `--baseline ""` from an absent `--baseline`. Needed because the zero value of
// the cohort flag is also its "no freeze" sentinel, and silently treating an
// explicit empty value as "no freeze" would be exactly the kind of fallback
// CLAUDE.md §2 forbids on data that decides a published number.
func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// baselineIDPattern is the ONLY accepted shape for a cohort baseline id.
//
// This is not cosmetic — it is the BF-2 fix. The baseline id flows into an
// UNESCAPED SQL LIKE pattern: internal/observatory/store_chains_eval.go builds
// `c.source_ref LIKE ?` with SourceRefPrefix+"%" and NO ESCAPE clause. SQLite's
// LIKE treats '_' as a single-character wildcard and '%' as any-sequence, so a
// baseline id like "v1_0" would silently ALSO match a "v1x0/…" cohort and widen
// the KPI denominator without any error. Restricting the charset to
// alphanumerics, '.' and '-' guarantees a frozen id is always a LITERAL prefix.
//
// '/' is excluded because it is the source_ref path separator: an id containing
// '/' could forge a cohort boundary. A leading '.' or '-' is excluded so ids
// sort and read predictably.
const baselineIDPattern = `^[A-Za-z0-9][A-Za-z0-9.-]*$`

var baselineIDRe = regexp.MustCompile(baselineIDPattern)

// validateBaselineID fails loudly on any baseline id that would corrupt the
// LIKE prefix match (CLAUDE.md §2 — no silent fallbacks on data that decides a
// published number). It is deliberately shared by BOTH sides of the cohort:
//   - write side: cmd/ailang/eval_suite.go `--baseline` (before any spend)
//   - read  side: cmd/ailang/chains_stats_cvs.go `--baseline`
//
// One validator, both sides, so a frozen id is always queryable and a queryable
// id could always have been frozen.
func validateBaselineID(id string) error {
	if id == "" {
		return fmt.Errorf("baseline id must not be empty (expected %s, e.g. v1.0)", baselineIDPattern)
	}
	if !baselineIDRe.MatchString(id) {
		return fmt.Errorf("invalid baseline id %q: must match %s (letters, digits, '.' and '-'; "+
			"'_' and '%%' are SQL LIKE wildcards that would silently widen the cohort, and '/' is the source_ref separator)",
			id, baselineIDPattern)
	}
	return nil
}

// validateCohortFreeze is the pre-flight gate for `--baseline`. It runs before
// the rig lock, before the API-key check, and before any benchmark executes,
// because both failures it catches would otherwise only surface AFTER an
// expensive metered run had already been paid for.
//
// baselineSet distinguishes "flag absent" from `--baseline ""`; the latter is an
// explicit error rather than a silent "no freeze" (CLAUDE.md §2).
func validateCohortFreeze(baselineSet bool, baselineID string, verify bool) error {
	if !baselineSet {
		return nil // default path — unchanged behaviour
	}
	if err := validateBaselineID(baselineID); err != nil {
		return err
	}
	if !verify {
		return fmt.Errorf("--baseline %s requires --verify: a frozen cohort with no verification "+
			"evidence can only ever produce `cost-per-verified-success: zero_denominator` (the "+
			"verified-success predicate requires verify_verified > 0). Refusing to spend on a run "+
			"that cannot yield the KPI it is being frozen for", baselineID)
	}
	return nil
}

// cohortSourceRef composes the chains.source_ref for an eval-suite run.
//
//	baseline == ""  ->  "<taskID>/<mode><condRef>"             (today's exact string)
//	baseline != ""  ->  "<baseline>/<taskID>/<mode><condRef>"  (frozen cohort)
//
// The taskID is RETAINED inside the frozen form on purpose: it is also the
// Observatory Task.ID (createEvalTask) and the OTEL correlation id, so trace
// linking and per-run disambiguation survive the freeze. The baseline is a
// CHAIN COHORT LABEL, not a correlation id — which is exactly why this is a
// flag and not an OTEL_RESOURCE_ATTRIBUTES ailang.task_id=v1.0 hack (that would
// collide the task registry row across every freeze run).
//
// The prefix produced here is matched by cohortSourceRefPrefix() on the read
// side; TestCohortSourceRef_RoundTripsWithReaderPrefix proves they agree.
func cohortSourceRef(baseline, taskID, evalMode, condRef string) string {
	base := fmt.Sprintf("%s/%s%s", taskID, evalMode, condRef)
	if baseline == "" {
		return base
	}
	return baseline + "/" + base
}

// evalChainParams groups the inputs needed to open the eval execution chain.
type evalChainParams struct {
	baselineID string // "" = no cohort freeze (default)
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

// chainID returns the chain id, or "" when the observatory was unavailable
// (createEvalChain returns nil in that case). Nil-safe so the cohort manifest
// can record "no chain" explicitly instead of crashing a frozen run.
func (c *EvalChainContext) chainID() string {
	if c == nil {
		return ""
	}
	return c.ChainID
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
		SourceRef:     cohortSourceRef(p.baselineID, p.taskID, p.evalMode, conditionRef(p.conditions)),
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
