package pkg

import (
	"encoding/json"
	"fmt"
)

// PackageVerifyReport is the JSON `ailang verify --package --json` emits and
// the registry validator decodes (M-PKG-QUALITY-LADDER M1). It lives here, not
// in cmd/ailang, so the producer and every consumer share one struct — the
// previous validator decoded a bare array while verify printed an object, and
// no contract was ever counted (0/373 published versions, measured 2026-09-17).
type PackageVerifyReport struct {
	Schema         string               `json:"schema"`
	Package        string               `json:"package"`
	Version        string               `json:"version"`
	Verified       int                  `json:"verified"`
	Total          int                  `json:"total"`
	Counterexample int                  `json:"counterexample"`
	Skipped        int                  `json:"skipped"`
	Errors         int                  `json:"errors"`
	Uncontracted   int                  `json:"uncontracted"`
	WallSeconds    float64              `json:"wall_seconds"`
	WallCapHit     bool                 `json:"wall_cap_hit,omitempty"`
	Modules        []ModuleVerifyReport `json:"modules"`
}

// ModuleVerifyReport is one exported module's verification outcome.
type ModuleVerifyReport struct {
	Module         string               `json:"module"`
	File           string               `json:"file"`
	Verified       int                  `json:"verified"`
	Counterexample int                  `json:"counterexample"`
	Skipped        int                  `json:"skipped"`
	Errors         int                  `json:"errors"`
	Uncontracted   int                  `json:"uncontracted"`
	CompileError   string               `json:"compile_error,omitempty"`
	Results        []FunctionVerifyItem `json:"results,omitempty"`
}

// FunctionVerifyItem is the per-function status the verifier reports.
type FunctionVerifyItem struct {
	Function string `json:"function"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

// PackageVerifySchema is the schema tag on PackageVerifyReport.
const PackageVerifySchema = "ailang.package-verify/v1"

// DecodePackageVerifyReport parses the JSON produced by
// `ailang verify --package --json`, refusing anything that is not that shape
// so a producer/consumer drift fails loudly instead of counting zero.
func DecodePackageVerifyReport(data []byte) (*PackageVerifyReport, error) {
	var r PackageVerifyReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode package verify report: %w", err)
	}
	if r.Schema != PackageVerifySchema {
		return nil, fmt.Errorf("decode package verify report: schema %q, want %q", r.Schema, PackageVerifySchema)
	}
	return &r, nil
}

// Add folds one module's counts into the package totals.
func (r *PackageVerifyReport) Add(m ModuleVerifyReport) {
	r.Modules = append(r.Modules, m)
	r.Verified += m.Verified
	r.Counterexample += m.Counterexample
	r.Skipped += m.Skipped
	r.Errors += m.Errors
	r.Uncontracted += m.Uncontracted
	r.Total = r.Verified + r.Counterexample + r.Skipped + r.Errors
}
