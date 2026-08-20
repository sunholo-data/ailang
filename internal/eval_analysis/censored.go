package eval_analysis

import (
	"fmt"
	"math"
	"sort"
)

const (
	CensoredVerdictVoid         = "VOID"
	CensoredVerdictKeep         = "KEEP"
	CensoredVerdictRetire       = "RETIRE"
	CensoredVerdictInconclusive = "INCONCLUSIVE"
)

// CensoredPairResult reports the pre-registered fmt decision instrument.
// Numeric evidence is omitted for VOID results because an integrity failure
// makes the estimand undefined, rather than merely unfavorable.
type CensoredPairResult struct {
	Verdict    string `json:"verdict"`
	Reason     string `json:"reason"`
	VoidReason string `json:"void_reason,omitempty"`

	NEff    int `json:"n_eff,omitempty"`
	OnWins  int `json:"on_wins,omitempty"`
	OffWins int `json:"off_wins,omitempty"`
	Ties    int `json:"ties,omitempty"`

	SignPValue         float64       `json:"sign_p_value,omitempty"`
	OppositeSignPValue float64       `json:"opposite_sign_p_value,omitempty"`
	BothPassPairs      int           `json:"both_pass_pairs,omitempty"`
	MeanLogTokenRatio  float64       `json:"mean_log_token_ratio,omitempty"`
	MeanLogRatioCILow  float64       `json:"mean_log_ratio_ci_low,omitempty"`
	MeanLogRatioCIHigh float64       `json:"mean_log_ratio_ci_high,omitempty"`
	MedianTokenRatio   float64       `json:"median_token_ratio,omitempty"`
	OnQuarantined      int           `json:"on_quarantined,omitempty"`
	OnRows             int           `json:"on_rows,omitempty"`
	OffQuarantined     int           `json:"off_quarantined,omitempty"`
	OffRows            int           `json:"off_rows,omitempty"`
	PassRateLoss       bool          `json:"significant_on_pass_rate_loss,omitempty"`
	McNemar            McNemarResult `json:"mcnemar"`
}

// AnalyzeCensoredPairs applies the treatment and order gates before computing
// any statistics, then applies the decision rule from design doc section 7.
func AnalyzeCensoredPairs(on, off []*BenchmarkResult) CensoredPairResult {
	if reason := treatmentIntegrityReason(on, off); reason != "" {
		return voidCensoredResult(reason)
	}
	if reason := CheckFmtOrderIntegrity(on, off); reason != "" {
		return voidCensoredResult(reason)
	}

	// Both arms are filtered before ANY statistic is computed. The gates above
	// deliberately receive the RAW slices — the executed order is a fact about
	// the run, and the ">20% quarantined" rate is defined over the banked set —
	// but a row that is not a MEASUREMENT must never reach a win/tie tally, a
	// token ratio, or the McNemar guardrail. Filtering only ON was a real
	// defect: an OFF row invalid for any reason other than contamination
	// (harness_error, config_mismatch, ...) still scored a full OFF win.
	validOn, onQuarantined := partitionMeasurements(on)
	validOff, offQuarantined := partitionMeasurements(off)

	paired := PairArms(validOn, validOff)
	result := CensoredPairResult{
		OnQuarantined:  onQuarantined,
		OnRows:         len(on),
		OffQuarantined: offQuarantined,
		OffRows:        len(off),
		McNemar:        paired.McNemar,
	}
	offByKey := make(map[pairKey]*BenchmarkResult, len(validOff))
	for _, row := range validOff {
		offByKey[pairKey{row.ID, row.Lang, row.Trial}] = row
	}

	var logRatios []float64
	var tokenRatios []float64
	for _, onRow := range validOn {
		offRow, ok := offByKey[pairKey{onRow.ID, onRow.Lang, onRow.Trial}]
		if !ok {
			continue
		}
		onPass, offPass := passed(onRow), passed(offRow)
		switch {
		case onPass && !offPass:
			result.OnWins++
		case !onPass && offPass:
			result.OffWins++
		case !onPass && !offPass:
			result.Ties++
		default:
			ratio := tokenRatio(onRow, offRow)
			if ratio <= 0 {
				result.Ties++
				continue
			}
			logRatio := math.Log(ratio)
			logRatios = append(logRatios, logRatio)
			tokenRatios = append(tokenRatios, ratio)
			if math.Abs(logRatio) <= 0.10 {
				result.Ties++
			} else if logRatio < 0 {
				result.OnWins++
			} else {
				result.OffWins++
			}
		}
	}

	result.NEff = result.OnWins + result.OffWins
	result.SignPValue = exactBinomialUpperTail(result.OnWins, result.NEff)
	result.OppositeSignPValue = exactBinomialUpperTail(result.OffWins, result.NEff)
	result.BothPassPairs = len(logRatios)
	result.MeanLogTokenRatio, result.MeanLogRatioCILow, result.MeanLogRatioCIHigh = mean95CI(logRatios)
	result.MedianTokenRatio = medianFloat(tokenRatios)
	result.PassRateLoss = paired.McNemar.Reportable && paired.McNemar.PValue <= 0.05 && paired.OnlyOffPassed > paired.OnlyOnPassed
	applyCensoredDecision(&result)
	return result
}

// partitionMeasurements splits rows into those that ARE measurements and a
// count of those that are not. Design doc section 5: a quarantined row is
// dropped AND counted — never silently discarded, and never counted twice.
func partitionMeasurements(rows []*BenchmarkResult) (valid []*BenchmarkResult, quarantined int) {
	valid = make([]*BenchmarkResult, 0, len(rows))
	for _, row := range rows {
		if row.Validity != nil && !row.Validity.Valid {
			quarantined++
			continue
		}
		valid = append(valid, row)
	}
	return valid, quarantined
}

func treatmentIntegrityReason(on, off []*BenchmarkResult) string {
	quarantined := 0
	for _, row := range on {
		if row.Validity != nil && !row.Validity.Valid {
			quarantined++
		}
	}
	if len(on) > 0 && float64(quarantined)/float64(len(on)) > 0.20 {
		return "treatment_unproven_rate"
	}
	for _, row := range off {
		if len(row.FmtHookEvents) > 0 {
			return "control_contaminated"
		}
	}
	return ""
}

func voidCensoredResult(reason string) CensoredPairResult {
	return CensoredPairResult{Verdict: CensoredVerdictVoid, Reason: reason, VoidReason: reason}
}

type orderedFmtRow struct {
	row *BenchmarkResult
	arm string
}

// CheckFmtOrderIntegrity returns a stable machine-readable refusal reason, or
// an empty string when the rows implement section 5.3's counterbalanced block.
func CheckFmtOrderIntegrity(on, off []*BenchmarkResult) string {
	rows := make([]orderedFmtRow, 0, len(on)+len(off))
	for _, row := range on {
		rows = append(rows, orderedFmtRow{row: row, arm: "on"})
	}
	for _, row := range off {
		rows = append(rows, orderedFmtRow{row: row, arm: "off"})
	}
	if len(rows) == 0 {
		return "order_integrity_empty"
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].row.Timestamp.Before(rows[j].row.Timestamp) })
	for _, item := range rows {
		if item.row.Timestamp.IsZero() {
			return "order_integrity_timestamp"
		}
	}

	type block struct{ benchmark, arm string }
	blocks := make([]block, 0)
	seenBlock := make(map[block]bool)
	for _, item := range rows {
		key := block{benchmark: item.row.ID, arm: item.arm}
		if len(blocks) > 0 && blocks[len(blocks)-1] == key {
			continue
		}
		if seenBlock[key] {
			return "order_integrity_noncontiguous_block"
		}
		seenBlock[key] = true
		blocks = append(blocks, key)
	}
	if len(blocks)%2 != 0 {
		return "order_integrity_unpaired_block"
	}

	previousLead := ""
	seenBenchmark := make(map[string]bool)
	for i := 0; i < len(blocks); i += 2 {
		first, second := blocks[i], blocks[i+1]
		if first.benchmark != second.benchmark || first.arm == second.arm {
			return "order_integrity_nonadjacent_arms"
		}
		// DECLARED UNREACHABLE, defensively retained. Blocks are deduplicated by
		// (benchmark, arm) and a non-contiguous repeat of either key returns
		// order_integrity_noncontiguous_block above; a pair holding only one of a
		// benchmark's blocks returns order_integrity_nonadjacent_arms. With two
		// arms a benchmark owns at most two block keys, so no input reaches here.
		// Pinned by TestD2OrderRefusalRepeatedBenchmarkIsUnreachable, which fails
		// loudly if that ever stops being true.
		if seenBenchmark[first.benchmark] {
			return "order_integrity_repeated_benchmark"
		}
		seenBenchmark[first.benchmark] = true
		if previousLead == first.arm {
			return "order_integrity_lead_not_alternating"
		}
		previousLead = first.arm
	}
	// Exact alternation above entails |ON-lead - OFF-lead| <= 1.
	return ""
}

func tokenRatio(on, off *BenchmarkResult) float64 {
	if on.TotalTokens <= 0 || off.TotalTokens <= 0 {
		return 0
	}
	return float64(on.TotalTokens) / float64(off.TotalTokens)
}

func exactBinomialUpperTail(wins, n int) float64 {
	if n == 0 {
		return 1
	}
	var p float64
	for i := wins; i <= n; i++ {
		p += binomialPMF(i, n)
	}
	return math.Min(1, p)
}

func applyCensoredDecision(result *CensoredPairResult) {
	if result.NEff < 24 {
		// A directory analyzer cannot know whether two slots plus the authorized
		// third have run. Slot-budget retirement belongs to the driver/human.
		result.Verdict = CensoredVerdictInconclusive
		result.Reason = "insufficient-neff"
		return
	}
	if result.OppositeSignPValue <= 0.05 {
		result.Verdict = CensoredVerdictRetire
		result.Reason = "opposite-direction-rejection"
		return
	}
	keep := result.SignPValue <= 0.05 && result.MedianTokenRatio > 0 && result.MedianTokenRatio <= 0.90 && !result.PassRateLoss
	if keep {
		result.Verdict = CensoredVerdictKeep
		result.Reason = "keep-criteria-met"
		return
	}
	if result.NEff >= 40 {
		result.Verdict = CensoredVerdictRetire
		result.Reason = "keep-failed-at-neff"
		return
	}
	result.Verdict = CensoredVerdictInconclusive
	result.Reason = "additional-slot-required"
}

func mean95CI(values []float64) (mean, low, high float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	if len(values) == 1 {
		return mean, mean, mean
	}
	var sumSquares float64
	for _, value := range values {
		delta := value - mean
		sumSquares += delta * delta
	}
	standardError := math.Sqrt(sumSquares/float64(len(values)-1)) / math.Sqrt(float64(len(values)))
	margin := 1.96 * standardError
	return mean, mean - margin, mean + margin
}

func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	mid := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[mid]
	}
	return (ordered[mid-1] + ordered[mid]) / 2
}

func (r CensoredPairResult) String() string {
	return fmt.Sprintf("%s (%s)", r.Verdict, r.Reason)
}
