package modelreg

import (
	"fmt"
	"sort"
	"strings"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M4 (decision D1(a)).
//
// D1(a) was ratified with one cost accepted explicitly: PRICING rides the same
// file as role policy, so a bad publish can now reach observatory cost
// accounting without a rebuild. Before M4 that was impossible — the embedded
// copy was authoritative and a release gated every change.
//
// Validate is the mitigation, and it deliberately covers pricing rather than
// only `roles:`. A validator that checked role wiring and waved pricing through
// would guard the new feature while leaving the newly-exposed surface open,
// which is the more expensive of the two failures: a broken role fails loudly at
// resolution, a wrong price is silently wrong on every run until an invoice.

// Validate reports every problem in a registry, not just the first, so one
// publish attempt surfaces one complete fix list.
func (c *ModelsConfig) Validate() error {
	var problems []string

	if len(c.Models) == 0 {
		problems = append(problems, "registry declares no models at all")
	}

	names := make([]string, 0, len(c.Models))
	for n := range c.Models {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		m := c.Models[name]
		if strings.TrimSpace(m.APIName) == "" {
			problems = append(problems, fmt.Sprintf("model %q has no api_name", name))
		}
		if strings.TrimSpace(m.Provider) == "" {
			problems = append(problems, fmt.Sprintf("model %q has no provider", name))
		}
		// Pricing: negative is always wrong. Zero is legitimate (local ollama
		// rows genuinely cost nothing), so it is NOT an error — flagging it
		// would train publishers to ignore this validator.
		if m.Pricing.InputPer1K < 0 || m.Pricing.OutputPer1K < 0 {
			problems = append(problems, fmt.Sprintf(
				"model %q has negative pricing (input %g, output %g) — cost accounting reads this",
				name, m.Pricing.InputPer1K, m.Pricing.OutputPer1K))
		}
	}

	// Roles must name models the registry actually has. A chain pointing at a
	// deleted row is exactly the publish this validator exists to stop: it
	// resolves to nothing at the moment an agent needs a model.
	roles := make([]string, 0, len(c.Roles))
	for r := range c.Roles {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	for _, role := range roles {
		chain := c.Roles[role]
		if len(chain) == 0 {
			problems = append(problems, fmt.Sprintf("role %q has an empty chain", role))
			continue
		}
		for _, friendly := range chain {
			if _, err := c.GetModel(friendly); err != nil {
				problems = append(problems, fmt.Sprintf(
					"role %q names %q, which is not a model in this registry", role, friendly))
				continue
			}
			if _, _, err := c.GetExecutorForModel(friendly); err != nil {
				problems = append(problems, fmt.Sprintf(
					"role %q names %q, which cannot run as an agent: %v", role, friendly, err))
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("registry failed validation (%d problems):\n  - %s",
			len(problems), strings.Join(problems, "\n  - "))
	}
	return nil
}
