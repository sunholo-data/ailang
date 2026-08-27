package eval_harness

import "fmt"

// FrozenMarker records that a prompt version's bytes are immutable because the version
// has been used in at least one banked eval baseline. Decision D-41(c).
// ABSENT (nil) means never-banked, i.e. mutable.
type FrozenMarker struct {
	At              string `json:"at"`
	Reason          string `json:"reason"`
	EvidenceCount   int    `json:"evidence_count"`
	EvidenceExample string `json:"evidence_example"`
}

// IsHexSHA256 reports whether s is exactly 64 lowercase hex characters.
func IsHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func FrozenHashMismatchError(versionID string, v PromptVersion, actualHash string) error {
	return fmt.Errorf("prompt version %q is FROZEN: it is cited by %d banked baseline files under eval_results/baselines/ (e.g. %s; frozen %s, reason: %s — decision D-41c). Its bytes are immutable: editing %s in place would silently change what those baselines measured.\nTo change the teaching prompt, create a NEW version instead:\n  .claude/skills/prompt-manager/scripts/create_prompt_version.sh <new-id> %s \"<why>\"\n(expected sha256 %s, got %s)", versionID, v.Frozen.EvidenceCount, v.Frozen.EvidenceExample, v.Frozen.At, v.Frozen.Reason, v.File, versionID, v.Hash, actualHash)
}

func MutableHashMismatchError(versionID string, v PromptVersion, actualHash string) error {
	expectedPreview, actualPreview := v.Hash, actualHash
	if len(expectedPreview) > 16 {
		expectedPreview = expectedPreview[:16] + "..."
	}
	if len(actualPreview) > 16 {
		actualPreview = actualPreview[:16] + "..."
	}
	return fmt.Errorf("hash mismatch for %q: expected %s, got %s. This version is not yet banked, so in-place editing is allowed (D-41c) — regenerate the change-detector hash:\n  shasum -a 256 %s   # then update the \"hash\" field in prompts/versions.json", versionID, expectedPreview, actualPreview, v.File)
}

func FrozenUnenforceableHashError(versionID string, v PromptVersion) error {
	return fmt.Errorf("prompt version %q is FROZEN but its recorded hash %q is not a 64-hex sha256: refuse to load — a frozen version with an unenforceable hash is a freeze with no invariant (D-41c, Q6). Restore the recorded hash from git history, or bump to a new version.", versionID, v.Hash)
}
