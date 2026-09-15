package eval_analysis

// IsValid and InvalidReason are promoted from the embedded eval_harness.RunMetrics
// (a NIL Validity means VALID — see eval_harness/validity.go for why).

// FilterValidResults drops rows that are not measurements.
//
// Callers get this by DEFAULT via LoadResults/LoadResultsFromDirs. That default
// is the point of the measurement contract: if excluding non-measurements
// required opting in, the next analysis written in a hurry would silently
// include the garbage again — which is exactly how a 0/84 harness artefact
// spent a week inside the microRAG trend line.
func FilterValidResults(results []*BenchmarkResult) []*BenchmarkResult {
	valid := make([]*BenchmarkResult, 0, len(results))
	for _, r := range results {
		if r.IsValid() {
			valid = append(valid, r)
		}
	}
	return valid
}

// CountInvalid reports how many rows were excluded and why, so callers can
// surface the exclusion rather than silently shrinking the dataset.
func CountInvalid(results []*BenchmarkResult) map[string]int {
	counts := map[string]int{}
	for _, r := range results {
		if !r.IsValid() {
			counts[r.InvalidReason()]++
		}
	}
	return counts
}
