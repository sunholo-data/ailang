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
var AllowedOpenAIModels = []string{"gpt-6.1-sol", "gpt-6-luna"}

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

// Allowed leaves other vendors unchanged and admits only allowlisted OpenAI ids.
func ModelAllowed(id string) bool {
	if !IsOpenAIModel(id) {
		return true
	}
	for _, allowed := range AllowedOpenAIModels {
		if CanonicalModelKey(id) == CanonicalModelKey(allowed) {
			return true
		}
	}
	return false
}
