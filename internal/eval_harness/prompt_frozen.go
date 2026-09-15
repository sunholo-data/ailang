package eval_harness

// prompt_frozen.go — the freeze marker and the hash-shape predicate that
// cmd/ailang/prompt_freeze_core.go writes into prompts/versions.json. The
// three error constructors that once lived here (Frozen/Mutable hash mismatch,
// unenforceable frozen hash) are dead since M-V1-SIMPLIFY-S3 M4 moved the
// teaching text into prompt.verifyHash (internal/prompt/loader.go).

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
