package eval_analysis

// Headline-KPI publish path (M-COST-PER-SUCCESS-KPI, M2).
//
// This snapshots the canonical observatory cost-per-verified-success result into
// latest.json under an ADDITIVE `headlineKpis.costPerVerifiedSuccess` object. It
// does NOT recompute anything: the object is the very same struct the CLI
// (--json) and the HTTP handler serialize, round-tripped through JSON so the
// nested keys/values are field-for-field identical (the sprint's core contract).
//
// No baseline cohort is materialized in this sprint (that is M4, Mark-gated);
// this code path is exercised via fixture so the wiring is ready the moment a
// frozen cohort exists.

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// HeadlineKpiCostPerVerifiedSuccess is the map key under headlineKpis.
const HeadlineKpiCostPerVerifiedSuccess = "costPerVerifiedSuccess"

// headlineKpiObject marshals the canonical KPI result into the additive object
// stored in latest.json. It round-trips the struct through JSON so the published
// snapshot is byte-for-byte the same shape the CLI/HTTP surfaces emit — there is
// exactly one source of truth for the field names and values.
func headlineKpiObject(res *observatory.CostPerVerifiedSuccessResult) (map[string]interface{}, error) {
	if res == nil {
		return nil, fmt.Errorf("cost-per-verified-success result is nil")
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KPI result: %w", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("failed to decode KPI result: %w", err)
	}
	return obj, nil
}

// AttachHeadlineKpi snapshots the canonical cost-per-verified-success result into
// the dashboard's additive headlineKpis map. Existing dashboard fields are left
// untouched. Returns an error rather than silently publishing a partial/absent
// KPI (NO SILENT FALLBACKS).
//
// Note: this attaches whatever the observatory computed, INCLUDING an
// unavailable result (available=false). The dashboard renders the Incomplete
// state from that flag; we never fabricate an available number here.
func AttachHeadlineKpi(dashboard *DashboardJSON, res *observatory.CostPerVerifiedSuccessResult) error {
	if dashboard == nil {
		return fmt.Errorf("dashboard is nil")
	}
	obj, err := headlineKpiObject(res)
	if err != nil {
		return err
	}
	if dashboard.HeadlineKpis == nil {
		dashboard.HeadlineKpis = make(map[string]interface{})
	}
	dashboard.HeadlineKpis[HeadlineKpiCostPerVerifiedSuccess] = obj
	return nil
}
