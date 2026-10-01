package coordinator

import "fmt"

// The variant -> executor-CLI table.
//
// This is the coordinator's copy of the dispatcher's providersForVariant.
// ProviderForVariant reads it to derive an agent's executor from its image, so
// the two must not drift; a drift arm on the cloudrun side compares them.
//
// The copy exists because internal/dispatch/cloudrun already imports this
// package's types, and the reverse edge would be a cycle.
//
// A runtime audit of provider/variant mismatches lived here briefly. It was
// deleted when the provider became derived: a mismatch is no longer
// constructible, so the audit could never fire, and a control that cannot trip
// is worse than none — it reads as coverage.

var variantProviders = map[string][]string{
	"":         {"claude"},
	"default":  {"claude"},
	"go":       {"claude"},
	"codex":    {"codex"},
	"codex-go": {"codex"},
	"opencode": {"opencode"},
	"pi":       {"pi"},
	"pi-go":    {"pi"},
	"motoko":   {"motoko"},
	"eval":     nil, // agent-eval carries every CLI
	"eval-go":  nil, //   ditto (FROM agent-eval)
}

// retiredVariants are executor_variant values whose image and Cloud Run Job no
// longer exist. They are named rather than left to the generic "unknown
// variant" refusal so a stale config is told where its agents went.
//
// gemini / gemini-go: the Gemini CLI executor was retired in v0.22.0
// (M-MANAGED-AGENTS) but the agent-gemini images and jobs lingered until
// 2026-10-01, and every execution on them failed inside the container with
// `unknown executor: gemini` after cloning the repo.
var retiredVariants = map[string]string{
	"gemini":    "the Gemini CLI executor was retired in v0.22.0 and the agent-gemini image and job were removed",
	"gemini-go": "the Gemini CLI executor was retired in v0.22.0 and the agent-gemini-go image and job were removed",
}

// RetiredVariantError returns a non-nil error naming the replacement when
// variant is a retired executor_variant, and nil otherwise.
func RetiredVariantError(variant string) error {
	why, retired := retiredVariants[variant]
	if !retired {
		return nil
	}
	return fmt.Errorf("executor_variant %q is retired: %s. Run Gemini models with provider: managed_agents "+
		"(the binaryless Vertex Managed Agents executor) on a multi-CLI image such as executor_variant: eval, "+
		"and fix the agent in config.cloud.yaml", variant, why)
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
