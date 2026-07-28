package eval_analysis

// IsValid reports whether this row should count toward aggregates.
//
// A NIL Validity means VALID: every row banked before v0.31.0 lacks the field,
// and treating absent as invalid would erase all of that history.
func (r *BenchmarkResult) IsValid() bool {
	return r.Validity == nil || r.Validity.Valid
}

// InvalidReason returns why this row is invalid, or "" if it is valid.
func (r *BenchmarkResult) InvalidReason() string {
	if r.IsValid() {
		return ""
	}
	return r.Validity.Reason
}

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
