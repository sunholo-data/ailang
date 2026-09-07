package dispatch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

type Candidate struct {
	Model      string `json:"model"`
	Executor   string `json:"executor"`
	WireModel  string `json:"wire_model"`
	Transport  string `json:"transport"`
	Vendor     string `json:"vendor"`
	SkipReason string `json:"skip_reason,omitempty"`
	config     *modelreg.ModelConfig
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
	for _, name := range r.AuthorModels {
		v, err := models.DispatchOriginVendor(name)
		if err != nil {
			return nil, fmt.Errorf("author identity: %w", err)
		}
		authors[v] = true
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
		if r.Role == "evaluator" && authors[vendor] {
			c.SkipReason = "evaluator origin vendor matches a declared author"
		}
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
	return p, nil
}
