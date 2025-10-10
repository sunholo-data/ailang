package eval_analysis

import (
	"fmt"
)

// Compare compares two sets of benchmark results and produces a detailed report
func Compare(baseline, new []*BenchmarkResult, baselineLabel, newLabel string) (*ComparisonReport, error) {
	if len(baseline) == 0 {
		return nil, fmt.Errorf("baseline results are empty")
	}
	if len(new) == 0 {
		return nil, fmt.Errorf("new results are empty")
	}

	report := &ComparisonReport{
		BaselineLabel: baselineLabel,
		NewLabel:      newLabel,
	}

	// Build maps for quick lookup: key = benchmark_id + lang + model
	baselineMap := buildResultMap(baseline)
	newMap := buildResultMap(new)

	// Find changes
	for key, baselineResult := range baselineMap {
		newResult, exists := newMap[key]

		if !exists {
			// Benchmark removed
			report.Removed = append(report.Removed, baselineResult)
			continue
		}

		// Compare status
		baselineSuccess := baselineResult.StdoutOk
		newSuccess := newResult.StdoutOk

		if !baselineSuccess && newSuccess {
			// Fixed!
			report.Fixed = append(report.Fixed, &BenchmarkChange{
				ID:             baselineResult.ID,
				Lang:           baselineResult.Lang,
				Model:          baselineResult.Model,
				BaselineStatus: false,
				NewStatus:      true,
				BaselineError:  baselineResult.ErrorCategory,
				NewError:       "",
			})
		} else if baselineSuccess && !newSuccess {
			// Broken!
			report.Broken = append(report.Broken, &BenchmarkChange{
				ID:             baselineResult.ID,
				Lang:           baselineResult.Lang,
				Model:          baselineResult.Model,
				BaselineStatus: true,
				NewStatus:      false,
				BaselineError:  "",
				NewError:       newResult.ErrorCategory,
			})
		} else if baselineSuccess && newSuccess {
			// Still passing
			report.StillPassing = append(report.StillPassing, newResult)
		} else {
			// Still failing
			report.StillFailing = append(report.StillFailing, newResult)
		}
	}

	// Find new benchmarks
	for key, newResult := range newMap {
		if _, exists := baselineMap[key]; !exists {
			report.NewBenchmarks = append(report.NewBenchmarks, newResult)
		}
	}

	// Calculate aggregates
	report.TotalBaselineBench = len(baseline)
	report.TotalNewBench = len(new)

	baselineSuccess := countSuccesses(baseline)
	newSuccess := countSuccesses(new)

	report.BaselineSuccessRate = safeDiv(float64(baselineSuccess), float64(len(baseline)))
	report.NewSuccessRate = safeDiv(float64(newSuccess), float64(len(new)))
	report.SuccessRateDelta = report.NewSuccessRate - report.BaselineSuccessRate

	return report, nil
}

// CompareBaselines compares two baselines (with metadata)
func CompareBaselines(baseline, new *Baseline) (*ComparisonReport, error) {
	report, err := Compare(baseline.Results, new.Results, baseline.Version, new.Version)
	if err != nil {
		return nil, err
	}

	// Attach baseline metadata
	report.Baseline = baseline
	report.New = new

	return report, nil
}

// FindRegressions returns only benchmarks that broke
func FindRegressions(baseline, new []*BenchmarkResult) ([]*BenchmarkChange, error) {
	report, err := Compare(baseline, new, "baseline", "new")
	if err != nil {
		return nil, err
	}

	return report.Broken, nil
}

// FindImprovements returns only benchmarks that were fixed
func FindImprovements(baseline, new []*BenchmarkResult) ([]*BenchmarkChange, error) {
	report, err := Compare(baseline, new, "baseline", "new")
	if err != nil {
		return nil, err
	}

	return report.Fixed, nil
}

// HasRegressions checks if there are any regressions
func (r *ComparisonReport) HasRegressions() bool {
	return len(r.Broken) > 0
}

// HasImprovements checks if there are any improvements
func (r *ComparisonReport) HasImprovements() bool {
	return len(r.Fixed) > 0
}

// NetChange returns the net change in passing benchmarks
func (r *ComparisonReport) NetChange() int {
	return len(r.Fixed) - len(r.Broken)
}

// ImprovementPercent returns the improvement as a percentage
func (r *ComparisonReport) ImprovementPercent() float64 {
	return r.SuccessRateDelta * 100
}

// Summary returns a one-line summary of the comparison
func (r *ComparisonReport) Summary() string {
	delta := r.ImprovementPercent()
	if delta > 0 {
		return fmt.Sprintf("✓ Improved by %.1f%% (%d fixed, %d broken)",
			delta, len(r.Fixed), len(r.Broken))
	} else if delta < 0 {
		return fmt.Sprintf("✗ Regressed by %.1f%% (%d fixed, %d broken)",
			-delta, len(r.Fixed), len(r.Broken))
	} else {
		return fmt.Sprintf("→ No change in success rate (%d fixed, %d broken)",
			len(r.Fixed), len(r.Broken))
	}
}

// Helper functions

// buildResultMap creates a map keyed by benchmark_id + lang + model
func buildResultMap(results []*BenchmarkResult) map[string]*BenchmarkResult {
	m := make(map[string]*BenchmarkResult)
	for _, r := range results {
		key := fmt.Sprintf("%s|%s|%s", r.ID, r.Lang, r.Model)
		// If multiple results for same key, keep the newest
		if existing, exists := m[key]; !exists || r.Timestamp.After(existing.Timestamp) {
			m[key] = r
		}
	}
	return m
}

// countSuccesses counts how many results are successful
func countSuccesses(results []*BenchmarkResult) int {
	count := 0
	for _, r := range results {
		if r.StdoutOk {
			count++
		}
	}
	return count
}

// safeDiv performs division with zero-check
func safeDiv(num, denom float64) float64 {
	if denom == 0 {
		return 0.0
	}
	return num / denom
}
