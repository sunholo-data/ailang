// Package rolemodels defines the shared mission-role model policy.
package rolemodels

import (
	"regexp"
	"strings"
)

// AllowedOpenAI implements D-FLEET-12 item 1 (Mark, attended 2026-10-02: Sol 6.1
// is the OpenAI rung; gpt-6-astra stays registered but dormant). gpt-6-luna is
// here because a LATER attended ruling uses it: the triage charter, ratified
// 2026-10-15, routes its read-only verifiers to codex:gpt-6-luna (the docs
// mission pinned it 2026-09-30). The conflict is D-FLEET-27, for Mark to confirm.
// Registration does not authorize a mission role.
var AllowedOpenAI = []string{"gpt-6.1-sol", "gpt-6-luna"}

// SpawnOverrideKey hides native child-model overrides so children inherit the parent.
const SpawnOverrideKey = "features.multi_agent_v2.expose_spawn_agent_model_overrides"

var providerPrefix = regexp.MustCompile(`^\w+:`)
var separators = strings.NewReplacer("-", "", ".", "", "_", "")

// CanonicalKey unifies registry, provider-prefixed and Codex model spellings.
func CanonicalKey(id string) string {
	id = providerPrefix.ReplaceAllString(strings.ToLower(id), "")
	return separators.Replace(strings.TrimPrefix(id, "openai/"))
}

// IsOpenAI duplicates quorum's gpt/astra/codex vendor-recognition keys.
// Keep those semantics here without importing quorum: this must remain a leaf.
func IsOpenAI(id string) bool {
	raw := strings.ToLower(id)
	return strings.HasPrefix(CanonicalKey(id), "gpt") || strings.Contains(raw, "astra") || strings.HasPrefix(raw, "codex:")
}

// Allowed leaves other vendors unchanged and admits only allowlisted OpenAI ids.
func Allowed(id string) bool {
	if !IsOpenAI(id) {
		return true
	}
	for _, allowed := range AllowedOpenAI {
		if CanonicalKey(id) == CanonicalKey(allowed) {
			return true
		}
	}
	return false
}
