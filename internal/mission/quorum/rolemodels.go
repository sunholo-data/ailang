package quorum

// The mission-role OpenAI model policy (D-FLEET-12). It lives in quorum, which
// cmd/ailang already links, rather than in its own package: a new package would
// change the binary's package closure that ARCHITECTURE.md records.

import (
	"regexp"
	"strings"
)

// AllowedOpenAIModels implements D-FLEET-12 item 1 (Mark, attended 2026-10-02: Sol 6.1
// is the OpenAI rung; gpt-6-astra stays registered but dormant). gpt-6-luna is
// here because a LATER attended ruling uses it: the triage charter, ratified
// 2026-10-15, routes its read-only verifiers to codex:gpt-6-luna (the docs
// mission pinned it 2026-09-30). The conflict is D-FLEET-27, for Mark to confirm.
// Registration does not authorize a mission role.
// 2026-11-17 (Mark, attended: "upgrade our defaults to these"): the Sol rung moves to
// gpt-6.2-sol. gpt-6.1-sol stays allowed while the cloud codex images (codex-cli 0.159.2,
// which 400s gpt-6.2-sol) still run it; drop it once the 0.169.0 image is deployed.
var AllowedOpenAIModels = []string{"gpt-6.2-sol", "gpt-6.1-sol", "gpt-6-luna"}

// SpawnOverrideKey hides native child-model overrides so children inherit the parent.
const SpawnOverrideKey = "features.multi_agent_v2.expose_spawn_agent_model_overrides"

var modelProviderPrefix = regexp.MustCompile(`^\w+:`)
var modelKeySeparators = strings.NewReplacer("-", "", ".", "", "_", "")

// CanonicalModelKey unifies registry, provider-prefixed and Codex model spellings.
func CanonicalModelKey(id string) string {
	id = modelProviderPrefix.ReplaceAllString(strings.ToLower(id), "")
	return modelKeySeparators.Replace(strings.TrimPrefix(id, "openai/"))
}

// IsOpenAIModel recognizes an OpenAI model id: canonical key beginning "gpt", or
// "astra" anywhere, or a "codex:" lane prefix. Narrower than VendorOf, which
// matches substrings, so a prefixed id such as "or-gpt-6-sol" is not recognized.
func IsOpenAIModel(id string) bool {
	raw := strings.ToLower(id)
	return strings.HasPrefix(CanonicalModelKey(id), "gpt") || strings.Contains(raw, "astra") || strings.HasPrefix(raw, "codex:")
}

// ModelAllowed leaves other vendors unchanged and admits only allowlisted OpenAI ids.
func ModelAllowed(id string) bool {
	if !IsOpenAIModel(id) {
		return true
	}
	return openAIListed(id, AllowedOpenAIModels)
}

// QuorumOpenAIModels are the OpenAI models a design-quorum seat may run: the
// mission-role allowlist plus gpt-6.2-astra, admitted for review only
// (Mark, attended 2026-11-25: quorum pool "codex astra, gemini carbon, fable,
// glm 5-3, kimi"). Reviewing a doc is not a mission role, so this widens nothing
// that `mission codex-exec` runs. The retired gpt-6-astra stays refused.
var QuorumOpenAIModels = append(append([]string{}, AllowedOpenAIModels...), "gpt-6.2-astra")

// QuorumModelAllowed is ModelAllowed for quorum seating.
func QuorumModelAllowed(id string) bool {
	if !IsOpenAIModel(id) {
		return true
	}
	return openAIListed(id, QuorumOpenAIModels)
}

func openAIListed(id string, list []string) bool {
	for _, allowed := range list {
		if CanonicalModelKey(id) == CanonicalModelKey(allowed) {
			return true
		}
	}
	return false
}
