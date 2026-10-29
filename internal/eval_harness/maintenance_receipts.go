package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/sunholo-data/ailang/internal/executor"
)

// These are wire records for the native package, not Go metric implementations.
type MaintenanceUsageIdentity struct {
	Source         string  `json:"source"`
	Version        string  `json:"version"`
	Digest         string  `json:"digest"`
	AttemptID      string  `json:"attemptId"`
	RequestID      *string `json:"requestId"`
	Provider       *string `json:"provider"`
	Model          *string `json:"model"`
	Harness        string  `json:"harness"`
	SettingsDigest string  `json:"settingsDigest"`
}
type MaintenanceTokenCounters struct {
	Contract          string `json:"contract"`
	Input             *int   `json:"input"`
	CacheRead         *int   `json:"cacheRead"`
	CacheWrite        *int   `json:"cacheWrite"`
	CacheReported     bool   `json:"cacheReported"`
	Output            *int   `json:"output"`
	Reasoning         *int   `json:"reasoning"`
	ReasoningReported bool   `json:"reasoningReported"`
	OutputScheme      string `json:"outputScheme"`
}
type MaintenanceCostEvidence struct {
	Amount       *string `json:"amount"`
	Currency     *string `json:"currency"`
	Basis        string  `json:"basis"`
	PriceVersion *string `json:"priceVersion"`
}
type MaintenanceUsageReceipt struct {
	ID               string                   `json:"id"`
	Identity         MaintenanceUsageIdentity `json:"identity"`
	Tokens           MaintenanceTokenCounters `json:"tokens"`
	Cost             MaintenanceCostEvidence  `json:"cost"`
	Role             string                   `json:"role"`
	ParentID         *string                  `json:"parentId"`
	ChildrenComplete bool                     `json:"childrenComplete"`
	Purpose          string                   `json:"purpose"`
	Failed           bool                     `json:"failed"`
}
type MaintenanceUsageBundle struct {
	Schema   string                    `json:"schema"`
	Receipts []MaintenanceUsageReceipt `json:"receipts"`
}

func maintenanceUsageBase(attempt, settings, model string, result *executor.Result) MaintenanceUsageReceipt {
	d := sha256.Sum256([]byte("unavailable:" + attempt))
	harness := "unmeasured"
	if result != nil && result.ExecutorVersion != "" {
		harness = result.ExecutorVersion
	}
	return MaintenanceUsageReceipt{
		ID: attempt + "-summary", Role: "summary", Purpose: "maintenance",
		Identity: MaintenanceUsageIdentity{Source: "executor-unavailable", Version: "usage-evidence/1", Digest: hex.EncodeToString(d[:]), AttemptID: attempt, Model: &model, Harness: harness, SettingsDigest: settings},
		Tokens:   MaintenanceTokenCounters{Contract: "exclusive/1", OutputScheme: "unknown"}, Cost: MaintenanceCostEvidence{Basis: "unknown"},
	}
}

// MaintenanceUsageReceipts exports settled parser observations and an explicit
// coverage parent. Result's aggregate zeroes and floating USD are never receipts.
// Even without a leaf, the unchecked parent prevents a fake free completed run.
func MaintenanceUsageReceipts(attempt, settings, model string, result *executor.Result) ([]MaintenanceUsageReceipt, error) {
	if attempt == "" || settings == "" || model == "" {
		return nil, fmt.Errorf("missing maintenance usage identity")
	}
	parent := maintenanceUsageBase(attempt, settings, model, result)
	rows := []MaintenanceUsageReceipt{parent}
	if result == nil || result.UsageEvidence == nil {
		return rows, nil
	}
	x := result.UsageEvidence
	rows[0].Identity.Source = x.Source
	rows[0].Identity.Version = x.Version
	rows[0].Identity.Digest = x.StreamDigest
	rows[0].ChildrenComplete = x.CoverageComplete && len(x.Observations) > 0
	rows[0].Failed = !result.Success
	for _, ob := range x.Observations {
		for _, n := range []*int{ob.Input, ob.Output, ob.Reasoning, ob.CacheRead, ob.CacheWrite, ob.ReportedTotal} {
			if n != nil && (*n < 0 || int64(*n) > 9007199254740991) {
				return nil, fmt.Errorf("invalid reported usage in %s", ob.ImportID)
			}
		}
		r := parent
		r.ID = attempt + "-" + ob.ImportID
		r.Role = "leaf"
		r.ParentID = &parent.ID
		r.Failed = ob.Failed
		r.Identity.Source = x.Source
		r.Identity.Version = x.Version
		r.Identity.Digest = ob.EventDigest
		r.Identity.RequestID = ob.RequestID
		r.Identity.Provider = ob.Provider
		if ob.Model != nil {
			r.Identity.Model = ob.Model
		}
		r.Tokens = MaintenanceTokenCounters{Contract: ob.Contract, Input: ob.Input, Output: ob.Output, Reasoning: ob.Reasoning, ReasoningReported: ob.Reasoning != nil, OutputScheme: ob.OutputScheme}
		// A partial cache split cannot be asserted to the strict native schema.
		// Raw reported fields remain in result.UsageEvidence for investigation.
		if ob.CacheRead != nil && ob.CacheWrite != nil {
			r.Tokens.CacheReported = true
			r.Tokens.CacheRead = ob.CacheRead
			r.Tokens.CacheWrite = ob.CacheWrite
		} else if ob.CacheRead != nil || ob.CacheWrite != nil {
			rows[0].ChildrenComplete = false
		}
		r.Cost = MaintenanceCostEvidence{Basis: ob.CostBasis, Amount: ob.CostAmount}
		if ob.CostAmount != nil {
			usd := "USD"
			r.Cost.Currency = &usd
		}
		rows = append(rows, r)
	}
	return rows, nil
}
