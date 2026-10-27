// Package verification contains dependency-free evidence shared by the static
// verifier and its package/CLI consumers. It does not decide proof or gate status.
package verification

import (
	"fmt"
	"sort"
	"strings"
)

// ReasonCode refines an existing outcome without changing its status or counter.
type ReasonCode string

const (
	MissingPostcondition  ReasonCode = "missing_postcondition"
	UnsupportedEncoding   ReasonCode = "unsupported_encoding"
	MissingBody           ReasonCode = "missing_body"
	SolverUnknown         ReasonCode = "solver_unknown"
	SolverTimeout         ReasonCode = "solver_timeout"
	VerifierError         ReasonCode = "verifier_error"
	Counterexample        ReasonCode = "counterexample"
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
}

// Evidence is optional on older producers. A missing model means unavailable,
// never a default claim about their proof scope or runtime assertions.
type Evidence struct {
	EvidenceSchema   string        `json:"evidence_schema,omitempty"`
	NumericModel     *NumericModel `json:"numeric_model,omitempty"`
	RuntimeContracts string        `json:"runtime_contracts,omitempty"`
}

func EncodedEvidence() Evidence {
	return Evidence{EvidenceSchema: "ailang.verification-evidence/v1", NumericModel: &NumericModel{FloatSort: "Real", IntSort: "Int", RuntimeEquivalence: "not_established"}, RuntimeContracts: "not_measured"}
}

// Summary is shared by all human proof summaries; it never upgrades absent data.
func (e Evidence) Summary() string {
	if e.NumericModel == nil {
		return "numeric model unavailable; runtime contracts not measured"
	}
	return fmt.Sprintf("encoded model: float=%s, int=%s; runtime equivalence %s (not an IEEE/machine-arithmetic proof); absent recursion depth is not an inductive proof; runtime contracts %s", e.NumericModel.FloatSort, e.NumericModel.IntSort, e.NumericModel.RuntimeEquivalence, orUnavailable(e.RuntimeContracts))
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
