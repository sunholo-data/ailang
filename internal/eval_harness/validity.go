package eval_harness

import "strings"

// Validity records whether a banked datapoint is a MEASUREMENT at all.
//
// # THE DISTINCTION THIS EXISTS TO MAKE
//
// Before v0.31.0 the eval pipeline could not tell "the subject was measured and
// did badly" apart from "we failed to measure the subject". Both produced a low
// number and both were banked as findings. Concretely, in six weeks:
//
//   - motoko died before step 0 for six days; 72 phantom "benchmark failures"
//     were banked and indistinguishable from a model that could not solve them.
//   - a shell bug produced an empty pass count which `${PASS_ON:-0}` coerced to
//     0, banking `on_pass: 0, delta_pp: -73.8` as a real 0% measurement.
//   - cloud models silently ran the wrong profile, so rows advertised
//     capabilities they never loaded.
//
// A row carrying Validity{Valid:false} is retained (never deleted — it is
// evidence of the bug) but excluded from aggregates by default.
type Validity struct {
	// Valid reports whether this row is a real measurement.
	Valid bool `json:"valid"`
	// Reason explains an invalid row. Always set when Valid is false.
	// This is the MACHINE-readable enum that analysis filters on — keep the
	// vocabulary small and stable.
	Reason string `json:"reason,omitempty"`
	// Detail carries the human-readable specifics for this particular row
	// (e.g. which two profiles disagreed). Optional, never filtered on: it
	// exists so an operator reading a quarantined row can see what happened
	// without re-deriving it from two other files.
	Detail string `json:"detail,omitempty"`
}

// Invalid-reason constants. Keep these stable: they are written to disk and
// downstream analysis filters on them.
const (
	// ReasonCanaryFailed: the subject failed its pre-flight canary, so nothing
	// it produced reflects model capability.
	ReasonCanaryFailed = "canary_failed"
	// ReasonZeroFiles: an arm produced no result files at all.
	ReasonZeroFiles = "zero_files"
	// ReasonZeroPassAll: an arm passed nothing — with a non-trivial benchmark
	// set this is a harness failure, not a result. (The 2026-07-20 shape.)
	ReasonZeroPassAll = "zero_pass_all"
	// ReasonConfigMismatch: the resolved runtime config contradicts what
	// models.yml claimed, so the row does not measure what it says it does.
	ReasonConfigMismatch = "config_mismatch"
	// ReasonHarnessError: a harness-side failure with no more specific reason.
	ReasonHarnessError = "harness_error"
	// ReasonInfraOutage: the nightly classifier could not measure the suite.
	ReasonInfraOutage = "infra_outage"
	// ReasonTreatmentUnproven: an experiment arm cannot demonstrate that its
	// treatment actually applied (or a control shows it leaked in). The run may
	// be perfectly healthy — it simply does not measure what it claims, so a
	// null result from it would be meaningless. Void by preregistration.
	ReasonTreatmentUnproven = "treatment_unproven"
)

// MarkValid returns an explicitly-valid marker.
//
// Rows may equally leave Validity nil; both mean valid. The explicit form is
// useful when a stage has actively checked something and wants to say so.
func MarkValid() *Validity {
	return &Validity{Valid: true}
}

// MarkInvalid returns an invalid marker carrying why.
//
// An empty reason falls back to ReasonHarnessError rather than being allowed
// through: an invalid row without a reason tells you a number is wrong but not
// what to fix, which is how the 2026-07-20 artefact sat unexamined for a week.
func MarkInvalid(reason string) *Validity {
	if reason == "" {
		reason = ReasonHarnessError
	}
	return &Validity{Valid: false, Reason: reason}
}

// IsValid reports whether this row should count toward aggregates.
//
// A NIL Validity means VALID. This is load-bearing: every row banked before
// v0.31.0 has no `validity` field, and if absent decoded to the zero value
// (Valid:false) the entire pre-v0.31.0 history would silently disappear from
// every trend — a far worse data loss than the bug being fixed. That is also
// why the field is a POINTER: a plain struct would make absent and
// explicitly-invalid indistinguishable.
func (m *RunMetrics) IsValid() bool {
	return m.Validity == nil || m.Validity.Valid
}

// InvalidReason returns the reason this row is invalid, or "" if it is valid.
func (m *RunMetrics) InvalidReason() string {
	if m.IsValid() {
		return ""
	}
	return m.Validity.Reason
}

// applyValidityBackstop marks a row invalid when it failed for a reason the
// harness could not identify. Called from MetricsLogger.Log, the single point
// every banked row passes through.
//
// # WHY THIS EXISTS
//
// The validity framework shipped with its CONSUMERS wired and its PRODUCER
// missing. MarkInvalid was called from exactly two places — fmt_treatment.go
// and config_assert.go — so the dominant failure mode on the local rig, a
// motoko crash banked as error_category=api_error, was recorded as a VALID
// model failure. Four things silently depended on that being right:
//
//  1. A/B arms counted harness crashes against the treatment. The 2026-07-30
//     fmt run had 2 of its first 3 rows die in the harness, both banked as
//     model failures.
//  2. --skip-existing only retries INVALID rows, so it never retried anything:
//     every crash looked like a completed measurement and the gap became
//     permanent.
//  3. ELO fits benchmark difficulty from pass rates, so crash-prone benchmarks
//     were rated "hard" when they were merely broken — and set selection
//     built on that then picked the broken ones on purpose.
//  4. Published pass rates carried the crash rate inside them.
//
// api_error is safe to treat this way because the taxonomy defines it as the
// fallback "when no more specific cause is known": every identifiable MODEL
// failure has its own category (compile_error, runtime_error, logic_error,
// refused, step_exhausted, thrash_aborted, resource_limit, timeout, ...). So
// reaching api_error means the harness genuinely does not know what happened,
// which is the definition of a failure to measure rather than a measurement.
//
// The direction is deliberately conservative: an unknown failure is charged to
// us and retried, never to the model.
func (m *RunMetrics) applyValidityBackstop() {
	// An earlier stage that actively ruled on this row wins — including a stage
	// that marked it explicitly VALID.
	if m.Validity != nil {
		return
	}
	if m.ErrorCategory != ErrorCategoryAPI {
		return
	}
	v := MarkInvalid(ReasonHarnessError)
	v.Detail = validityDetailFromStderr(m.Stderr)
	m.Validity = v
}

// validityDetailFromStderr picks the last non-empty stderr line as the
// human-readable specifics for a quarantined row, so an operator can see WHICH
// harness failure this was without opening the transcript.
func validityDetailFromStderr(stderr string) string {
	const maxDetail = 300
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if len(line) > maxDetail {
			line = line[:maxDetail] + "…"
		}
		return line
	}
	return "no stderr captured"
}

// FilterValid returns only the rows that count toward aggregates.
//
// Analysis paths should call this by default and offer an explicit
// --include-invalid escape hatch, so that excluding bad data is the behaviour
// you get without thinking about it.
func FilterValid(rows []RunMetrics) []RunMetrics {
	valid := make([]RunMetrics, 0, len(rows))
	for _, r := range rows {
		if r.IsValid() {
			valid = append(valid, r)
		}
	}
	return valid
}
