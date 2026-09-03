package coordinator

import (
	"fmt"
	"sort"
)

// Startup audit of provider/executor_variant pairs
// (found 2026-09-03; see dispatch_provider.go for the defect it guards).
//
// The dispatcher already refuses a job whose executor binary cannot exist in its
// image. That guard was added 2026-08-31 after an opencode/codex mismatch died
// mid-run, and it works — but it only ever speaks one agent at a time, and only
// when something happens to dispatch to that agent. Two pipeline agents were
// undispatchable for an unknown period and nothing said so until a task was
// aimed at one of them, at which point it retried every five minutes.
//
// A control that reports a fleet-wide misconfiguration one instance at a time,
// on demand, is not an inventory. This is: it runs once at startup and names
// every agent that cannot dispatch, before anyone waits on one.

// ProviderMismatch is an agent whose declared executor CLI cannot exist in the
// image its variant selects.
type ProviderMismatch struct {
	AgentID  string
	Provider string
	Variant  string
	Allowed  []string
}

func (m ProviderMismatch) String() string {
	return fmt.Sprintf("%s (provider=%q, executor_variant=%q, image has %v)",
		m.AgentID, m.Provider, m.Variant, m.Allowed)
}

// variantProviders mirrors the dispatcher's table. It is duplicated rather than
// imported because internal/dispatch/cloudrun imports the coordinator's types,
// and the reverse edge would be a cycle. AuditProviderMismatches is covered by a
// drift test that fails if the two ever disagree.
var variantProviders = map[string][]string{
	"":          {"claude"},
	"default":   {"claude"},
	"go":        {"claude"},
	"codex":     {"codex"},
	"codex-go":  {"codex"},
	"gemini":    {"gemini"},
	"gemini-go": {"gemini"},
	"opencode":  {"opencode"},
	"pi":        {"pi"},
	"pi-go":     {"pi"},
	"motoko":    {"motoko"},
	"eval":      nil, // agent-eval carries every CLI
	"eval-go":   nil, //   ditto (FROM agent-eval)
}

// AuditProviderMismatches returns every registered agent whose effective
// provider cannot run in the image its executor_variant selects.
//
// It resolves the provider exactly as dispatch does — agent first, then the
// plane default — so it reports what would actually happen, not what the config
// appears to say. That distinction is the whole point: the defect this exists to
// catch was an agent declaring "codex" and being dispatched as "pi".
func AuditProviderMismatches(registry *AgentRegistry, planeDefault string) []ProviderMismatch {
	if registry == nil {
		return nil
	}

	var out []ProviderMismatch
	for _, agent := range registry.ListAgents() {
		if agent == nil {
			continue
		}
		provider := ResolveDispatchProvider(agent, planeDefault)
		if provider == "" {
			continue
		}
		allowed, known := variantProviders[agent.ExecutorVariant]
		if !known || allowed == nil {
			// An unknown variant is refused elsewhere; a nil list means the
			// image carries every CLI.
			continue
		}
		ok := false
		for _, p := range allowed {
			if p == provider {
				ok = true
				break
			}
		}
		if !ok {
			out = append(out, ProviderMismatch{
				AgentID:  agent.ID,
				Provider: provider,
				Variant:  agent.ExecutorVariant,
				Allowed:  allowed,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AgentID < out[j].AgentID })
	return out
}

// VariantProviders exposes this package's copy of the variant/provider table so
// the dispatcher's own test can prove the two have not drifted. The comparison
// lives on that side because internal/dispatch/cloudrun already imports this
// package; the reverse edge would be a cycle, which is also why the copy exists.
func VariantProviders() map[string][]string {
	out := make(map[string][]string, len(variantProviders))
	for k, v := range variantProviders {
		if v == nil {
			out[k] = nil
			continue
		}
		out[k] = append([]string(nil), v...)
	}
	return out
}
