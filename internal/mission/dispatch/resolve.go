package dispatch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

type Candidate struct {
	Model      string `json:"model"`
	Executor   string `json:"executor"`
	WireModel  string `json:"wire_model"`
	Transport  string `json:"transport"`
	Vendor     string `json:"vendor"`
	SkipReason string `json:"skip_reason,omitempty"`
	// SameVendorAsAuthor records that generator != judge held only at MODEL level for this
	// candidate. Recorded, not fatal: see the evaluator branch in Resolve.
	SameVendorAsAuthor bool `json:"same_vendor_as_author,omitempty"`
	config             *modelreg.ModelConfig
}

type Plan struct {
	RegistryDigest     string      `json:"registry_digest"`
	Version            int         `json:"version"`
	RequestDigest      string      `json:"request_digest"`
	ContractVersion    string      `json:"contract_version"`
	Candidates         []Candidate `json:"candidates"`
	IdentityProvenance string      `json:"identity_provenance"`
}

// Resolve does no executor construction, network probing, or state writes.
func Resolve(r Request, models *modelreg.ModelsConfig) (*Plan, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if models == nil {
		return nil, fmt.Errorf("model registry is required")
	}
	authors := map[string]bool{}
	authorModels := map[string]bool{}
	for _, name := range r.AuthorModels {
		v, err := models.DispatchOriginVendor(name)
		if err != nil {
			return nil, fmt.Errorf("author identity: %w", err)
		}
		authors[v] = true
		authorModels[name] = true
	}
	registryJSON, err := json.Marshal(models)
	if err != nil {
		return nil, fmt.Errorf("encode model registry: %w", err)
	}
	p := &Plan{RegistryDigest: fmt.Sprintf("sha256:%x", sha256.Sum256(registryJSON)), Version: 1, RequestDigest: r.Digest(), ContractVersion: "mission-role-v1", IdentityProvenance: "registry-and-dispatch"}
	for _, name := range r.Models {
		cli, wire, err := models.GetExecutorForModel(name)
		if err != nil {
			return nil, err
		}
		m, err := models.GetModel(name)
		if err != nil {
			return nil, err
		}
		vendor, err := models.OriginVendor(name)
		if err != nil {
			return nil, err
		}
		if wire == "" {
			return nil, fmt.Errorf("model %q has no wire model", name)
		}
		c := Candidate{Model: name, Executor: cli, WireModel: wire, Transport: m.Provider, Vendor: vendor, config: m}
		switch cli {
		case "claude", "codex", "pi":
		default:
			c.SkipReason = "executor budget contract is not admitted by role-run"
		}
		if models.UsesLocalGPU(name) {
			c.SkipReason = "local GPU admission is not supported by role-run"
		}
		// generator != judge: PREFERRED, not required (Mark, attended 2026-09-08 — "its a nice
		// to have, not a blocker").
		//
		// The same MODEL judging its own output is still refused: that is self-review and it
		// is worthless. A different model from the same VENDOR is now ALLOWED and flagged,
		// because insisting on cross-vendor made the judge unroutable outright — the only
		// harness that loads the sprint-evaluator skill is claude, so a claude-authored work
		// item had no valid evaluator at all. A same-vendor judge WITH its methodology beats a
		// cross-vendor judge without one; the canary proved the second option 3/3.
		//
		// SameVendorAsAuthor is set so the choice is visible in the receipt rather than
		// silently taken. Cross-vendor candidates sort first, so the property still holds
		// whenever it can be had.
		if r.Role == "evaluator" && authorModels[name] {
			c.SkipReason = "evaluator model is itself a declared author (self-review)"
		}
		c.SameVendorAsAuthor = r.Role == "evaluator" && authors[vendor]
		if c.SkipReason == "" {
			if _, err := models.DispatchOriginVendor(name); err != nil {
				return nil, err
			}
			for _, price := range []float64{m.Pricing.InputPer1K, m.Pricing.OutputPer1K} {
				if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
					return nil, fmt.Errorf("model %q requires finite positive input/output pricing for cost guard", name)
				}
			}
		}
		p.Candidates = append(p.Candidates, c)
	}
	// Cross-vendor judges first, so generator != judge still holds whenever it CAN. The
	// same-vendor rungs remain available behind them rather than being refused, which is the
	// difference between a preference and a blocker. Stable within each group: the caller's
	// declared model order is a routing decision and must survive this sort.
	if r.Role == "evaluator" {
		sort.SliceStable(p.Candidates, func(i, j int) bool {
			return !p.Candidates[i].SameVendorAsAuthor && p.Candidates[j].SameVendorAsAuthor
		})
	}
	return p, nil
}
