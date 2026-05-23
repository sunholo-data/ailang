// Package observatory: adaptive token-budget baselines.
//
// M-EVAL-OS-LONGITUDINAL Phase 2 (v0.23.0). Per-(model, benchmark) rolling
// mean+stddev of total tokens on PASS outcomes, maintained via Welford's
// online algorithm so we don't have to keep individual sample arrays on disk.
// The eval-suite reads the baseline before each benchmark and uses
// `mean + 2*stddev` as the adaptive thrash-abort threshold once N>=5
// passing samples have accumulated; falls back to the fixed
// --max-tokens-per-bench flag value during bootstrap.
//
// Storage: eval_baselines table in observatory.db (created in migrate.go).
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// BootstrapPassesRequired is the minimum N of passing trials before
// the adaptive (mean+Nσ) threshold becomes meaningful. Until then,
// callers should fall back to the fixed --max-tokens-per-bench flag.
const BootstrapPassesRequired = 5

// AdaptiveThresholdSigmas controls how aggressively to abort. 2.0 ≈ 95%
// of passing runs stay under the threshold (Gaussian assumption). Tuned
// to err on the side of letting borderline-thrashing runs complete —
// we'd rather burn extra tokens than abort a legitimate slow run.
const AdaptiveThresholdSigmas = 2.0

// EvalBaseline is one row of the eval_baselines table.
type EvalBaseline struct {
	ModelID      string
	BenchmarkID  string
	NPassTrials  int
	MeanTokens   float64
	StddevTokens float64
	M2Tokens     float64 // Welford accumulator: sum-of-squared-deviations
	LastUpdated  time.Time
}

// GetEvalBaseline fetches the baseline row for (model, benchmark). Returns
// (nil, nil) when the row doesn't exist yet (bootstrap case — caller falls
// back to fixed threshold).
func GetEvalBaseline(ctx context.Context, db *sql.DB, modelID, benchmarkID string) (*EvalBaseline, error) {
	row := db.QueryRowContext(ctx, `
		SELECT n_pass_trials, mean_tokens, stddev_tokens, m2_tokens, last_updated
		FROM eval_baselines
		WHERE model_id = ? AND benchmark_id = ?
	`, modelID, benchmarkID)

	var b EvalBaseline
	b.ModelID = modelID
	b.BenchmarkID = benchmarkID
	err := row.Scan(&b.NPassTrials, &b.MeanTokens, &b.StddevTokens, &b.M2Tokens, &b.LastUpdated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get eval baseline (%s, %s): %w", modelID, benchmarkID, err)
	}
	return &b, nil
}

// UpdatePassedTrial extends the rolling baseline with a new passing trial's
// token count, using Welford's online algorithm. Pure additive — never
// removes data. UPSERT semantics: creates a row if none exists.
//
// Algorithm:
//
//	delta  = newTokens - oldMean
//	nNew   = nOld + 1
//	meanNew = oldMean + delta / nNew
//	delta2 = newTokens - meanNew
//	m2New  = oldM2 + delta * delta2
//	stddev = sqrt(m2New / (nNew - 1))   if nNew >= 2, else 0
//
// All math in float64; tokens cast to float64 for delta/mean/stddev.
func UpdatePassedTrial(ctx context.Context, db *sql.DB, modelID, benchmarkID string, tokens int) error {
	now := time.Now()
	tokensF := float64(tokens)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin baseline update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Load current row inside the txn (so concurrent updates don't race).
	var (
		nOld     int
		meanOld  float64
		m2Old    float64
		_stddev  float64 // unused (recomputed below), but Scan needs the col
		_updated time.Time
	)
	err = tx.QueryRowContext(ctx, `
		SELECT n_pass_trials, mean_tokens, stddev_tokens, m2_tokens, last_updated
		FROM eval_baselines
		WHERE model_id = ? AND benchmark_id = ?
	`, modelID, benchmarkID).Scan(&nOld, &meanOld, &_stddev, &m2Old, &_updated)
	freshRow := err == sql.ErrNoRows
	if err != nil && !freshRow {
		return fmt.Errorf("load baseline for update: %w", err)
	}

	delta := tokensF - meanOld
	nNew := nOld + 1
	meanNew := meanOld + delta/float64(nNew)
	delta2 := tokensF - meanNew
	m2New := m2Old + delta*delta2
	stddevNew := 0.0
	if nNew >= 2 {
		stddevNew = math.Sqrt(m2New / float64(nNew-1))
	}

	if freshRow {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO eval_baselines (model_id, benchmark_id, n_pass_trials, mean_tokens, stddev_tokens, m2_tokens, last_updated)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, modelID, benchmarkID, nNew, meanNew, stddevNew, m2New, now)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE eval_baselines
			SET n_pass_trials = ?, mean_tokens = ?, stddev_tokens = ?, m2_tokens = ?, last_updated = ?
			WHERE model_id = ? AND benchmark_id = ?
		`, nNew, meanNew, stddevNew, m2New, now, modelID, benchmarkID)
	}
	if err != nil {
		return fmt.Errorf("write baseline row: %w", err)
	}
	return tx.Commit()
}

// ComputeAdaptiveThreshold returns the token-abort threshold for a (model,
// benchmark) pair. When the baseline has at least BootstrapPassesRequired
// passing samples, returns ceil(mean + sigmas * stddev). Otherwise returns
// the caller-supplied fixedFallback (the --max-tokens-per-bench flag value).
//
// fixedFallback of 0 means "no ceiling" — the function returns 0 in that
// case during bootstrap, signaling no abort enforcement.
func ComputeAdaptiveThreshold(baseline *EvalBaseline, sigmas float64, fixedFallback int) int {
	if baseline == nil || baseline.NPassTrials < BootstrapPassesRequired {
		return fixedFallback
	}
	t := baseline.MeanTokens + sigmas*baseline.StddevTokens
	if t < 0 {
		return fixedFallback
	}
	return int(math.Ceil(t))
}
