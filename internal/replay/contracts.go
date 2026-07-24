// Package replay defines the replay-contract taxonomy for parameterised
// effects (M-EFFECT-REPLAY-CONTRACTS, v1.0.0 — Rand pilot).
//
// This package answers a DIFFERENT question from internal/types' effectSchema:
//
//   - effectSchema (internal/types) is a VALIDATION table: it answers
//     "is this (effect, mode) LEGAL?" and is the single source of truth for
//     which mode-parameter values exist. It gates elaboration.
//
//   - This package is a TAXONOMY table: it answers "given a legal (effect,
//     mode), what REPLAY SEMANTICS does it have?" — mapping each pair to a
//     Contract label ∈ {Deterministic, ReSampleable, Opaque}. It is consumed
//     by trace emission now (so a moded effect event carries its contract),
//     and by replay harnesses later (which dispatch pin/redraw/substitute on
//     the label, not the raw effect token).
//
// The two tables are kept from drifting by TestReplayContractsAreLegalModes,
// which asserts every key here is legal per effectSchema (via the exported
// types.IsLegalEffectMode). effectSchema stays the single source of *which
// modes exist*; this package only *labels* the ones that already exist.
package replay

// Contract is the replay-semantics label for a legal (effect, mode) pair.
type Contract string

const (
	// Deterministic: replay MUST pin this value — re-running with the same
	// seed/config reproduces the exact same draw (e.g. Rand[mode=seeded],
	// AI[mode=fixed]).
	Deterministic Contract = "deterministic"

	// ReSampleable: replay MAY redraw — the value is non-deterministic but a
	// fresh sample is an acceptable substitute (e.g. Rand[mode=os],
	// AI[mode=routeable]).
	ReSampleable Contract = "re-sampleable"

	// Opaque: replay MUST substitute from the harness — the value comes from a
	// source that cannot (or must not) be reproduced or resampled in-process
	// (e.g. Rand[mode=crypto] CSPRNG draws, AI[mode=replay-only]).
	Opaque Contract = "opaque"
)

// contractKey identifies a (effect, mode) pair in the taxonomy table.
type contractKey struct {
	effect string
	mode   string
}

// contracts is the replay-contract taxonomy. Every key here MUST be a legal
// (effect, mode) pair under internal/types' effectSchema — enforced by
// TestReplayContractsAreLegalModes.
//
// Rand (dispatch pilot — the three modes have three runtime behaviours as of
// this sprint):
//   - seeded → deterministic (dedicated per-context PRNG from an explicit seed)
//   - os     → re-sampleable (global OS-entropy-seeded PRNG; may be redrawn)
//   - crypto → opaque        (crypto/rand CSPRNG draws; never reproducible)
//
// AI (labels only — dispatch already exists via M-AI-EFFECT-MODES; this sprint
// does NOT add AI dispatch, only its replay classification for trace tooling):
//   - fixed       → deterministic
//   - routeable   → re-sampleable
//   - replay-only → opaque
//
// Clock/Net/FS rows are registered by their own port sprint
// (m-effect-clock-net-fs-modes, Phase 5) once effectSchema unlocks their modes.
var contracts = map[contractKey]Contract{
	{effect: "Rand", mode: "seeded"}: Deterministic,
	{effect: "Rand", mode: "os"}:     ReSampleable,
	{effect: "Rand", mode: "crypto"}: Opaque,

	{effect: "AI", mode: "fixed"}:       Deterministic,
	{effect: "AI", mode: "routeable"}:   ReSampleable,
	{effect: "AI", mode: "replay-only"}: Opaque,
}

// ContractFor returns the replay contract for a legal (effect, mode) pair.
// The second return is false when the pair has no registered contract (e.g. an
// effect/mode whose port sprint has not landed yet). Callers MUST NOT invent a
// fallback label — an unlisted pair is genuinely unknown to replay tooling
// (no-silent-fallbacks: a wrong replay label corrupts replay decisions).
func ContractFor(effect, mode string) (Contract, bool) {
	c, ok := contracts[contractKey{effect: effect, mode: mode}]
	return c, ok
}

// registeredPairs returns all (effect, mode) pairs in the taxonomy table, for
// the drift-guard test. Order is unspecified (map iteration); callers that need
// determinism must sort.
func registeredPairs() [][2]string {
	pairs := make([][2]string, 0, len(contracts))
	for k := range contracts {
		pairs = append(pairs, [2]string{k.effect, k.mode})
	}
	return pairs
}
