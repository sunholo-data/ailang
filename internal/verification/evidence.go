// Package verification contains dependency-free evidence shared by the static
// verifier and its package/CLI consumers, and the ONE exit contract
// (outcome.go) every verification command applies to what the verifier
// reported. It never decides whether an individual obligation is proved.
package verification

import (
	"fmt"
	"sort"
	"strings"
)

// ReasonCode refines an existing outcome without changing its status or counter.
type ReasonCode string

const (
	MissingPostcondition ReasonCode = "missing_postcondition"
	UnsupportedEncoding  ReasonCode = "unsupported_encoding"
	MissingBody          ReasonCode = "missing_body"
	SolverUnknown        ReasonCode = "solver_unknown"
	SolverTimeout        ReasonCode = "solver_timeout"
	VerifierError        ReasonCode = "verifier_error"
	Counterexample       ReasonCode = "counterexample"
	// RequiresViolation marks a counterexample in which a call breaks its
	// callee's requires (decision D1(a), M-VERIFY-CALL-SITE-REQUIRES).
	RequiresViolation ReasonCode = "requires_violation"
	// IntOverflow: an int + - * may leave the int64 range (--int-model=int64).
	IntOverflow ReasonCode = "int_overflow"
	// FloatNaNOrInf: a float operation may produce NaN or ±Infinity.
	FloatNaNOrInf ReasonCode = "float_nan_or_inf"
	// BoundedInconclusive marks a sat found under bounded recursion unrolling:
	// the model may rest on an invented value for the uninterpreted frontier
	// stub, so the verifier can neither prove nor refute the obligation.
	BoundedInconclusive   ReasonCode = "bounded_unrolling_inconclusive"
	Uncontracted          ReasonCode = "uncontracted"
	ModuleBudgetExhausted ReasonCode = "module_budget_exhausted"
	MissingSource         ReasonCode = "missing_source"
	CompileError          ReasonCode = "compile_error"
	ReadError             ReasonCode = "read_error"
)

// NumericModel identifies encoder sorts, not equivalence with runtime arithmetic.
type NumericModel struct {
	FloatSort          string `json:"float_sort"`
	IntSort            string `json:"int_sort"`
	RuntimeEquivalence string `json:"runtime_equivalence"`
	// IntOverflow is how int64 overflow of + - * is handled: "reported" (a
	// caveat on the result, the default) or "checked" (a violation, under
	// --int-model=int64). FloatNaN "operations_checked": the operations that
	// make NaN or ±Inf from ordinary floats are checked (M4).
	IntOverflow string `json:"int_overflow,omitempty"`
	FloatNaN    string `json:"float_nan,omitempty"`
}

// Int models accepted by --int-model.
const (
	IntModelUnbounded = "unbounded" // default: overflow reported as a caveat
	IntModelInt64     = "int64"     // overflow is a violation
)

// Evidence is optional on older producers. A missing model means unavailable,
// never a default claim about their proof scope or runtime assertions.
type Evidence struct {
	EvidenceSchema   string        `json:"evidence_schema,omitempty"`
	NumericModel     *NumericModel `json:"numeric_model,omitempty"`
	RuntimeContracts string        `json:"runtime_contracts,omitempty"`
}

func EncodedEvidence() Evidence {
	return EncodedEvidenceFor(IntModelUnbounded)
}

// EncodedEvidenceFor is the evidence under an --int-model setting.
func EncodedEvidenceFor(intModel string) Evidence {
	overflow := "reported"
	if intModel == IntModelInt64 {
		overflow = "checked"
	}
	return Evidence{EvidenceSchema: "ailang.verification-evidence/v1", NumericModel: &NumericModel{FloatSort: "Real", IntSort: "Int", RuntimeEquivalence: "not_established", IntOverflow: overflow, FloatNaN: "operations_checked"}, RuntimeContracts: "not_measured"}
}

// Summary is shared by all human proof summaries; it never upgrades absent data.
func (e Evidence) Summary() string {
	if e.NumericModel == nil {
		return "numeric model unavailable; runtime contracts not measured"
	}
	numeric := ""
	if e.NumericModel.IntOverflow != "" {
		numeric += "; int64 overflow " + e.NumericModel.IntOverflow
	}
	if e.NumericModel.FloatNaN != "" {
		numeric += "; NaN/Inf-producing float operations checked, float inputs assumed non-NaN"
	}
	return fmt.Sprintf("encoded model: float=%s, int=%s%s; runtime equivalence %s (not an IEEE/machine-arithmetic proof); absent recursion depth is not an inductive proof; runtime contracts %s", e.NumericModel.FloatSort, e.NumericModel.IntSort, numeric, e.NumericModel.RuntimeEquivalence, orUnavailable(e.RuntimeContracts))
}
func orUnavailable(s string) string {
	if s == "" {
		return "not measured"
	}
	return s
}

// Rejection preserves machine code and human explanation across report projection.
type Rejection struct {
	Location string `json:"location,omitempty"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

// ModuleIssue is explicitly module-scoped: legacy sentinel counts are not counts
// of individual functions the verifier inspected.
type ModuleIssue struct {
	Scope      string     `json:"scope"`
	Module     string     `json:"module,omitempty"`
	ReasonCode ReasonCode `json:"reason_code"`
	Reason     string     `json:"reason"`
}

// ReasonSummary names only measured details. nil marks an older/unavailable
// producer; an empty non-nil map records that no reason-bearing functions ran.
func ReasonSummary(counts map[ReasonCode]int) string {
	if counts == nil {
		return "reason detail unavailable"
	}
	if len(counts) == 0 {
		return "no reason-bearing function outcomes"
	}
	keys := make([]string, 0, len(counts))
	for code := range counts {
		keys = append(keys, string(code))
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[ReasonCode(key)]))
	}
	return strings.Join(parts, ", ")
}
