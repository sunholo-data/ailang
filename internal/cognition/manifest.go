// Package cognition implements the Cognitive OS substrate primitives:
// capability manifest generation (this file), the cognitive event log (M2),
// Lamport clocks (M2), the deterministic scheduler (M3), and the transport
// trait (M2/M-COG-MESH).
//
// This subtree is platform-neutral Go. JavaScript bridges live in
// cmd/wasm/ and (later) docs/static/wasm/cognitive-runtime/.
//
// Design doc: design_docs/planned/v0_21_0/m-cog-runtime.md
package cognition

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ============================================================================
// CapabilityManifest — JSON sidecar emitted alongside a Cognitive OS module
// ============================================================================
//
// The manifest declares what authority an agent module requires, what
// budgets cap that authority, and what transports the module can use for
// inter-agent messaging. The host (cmd/wasm/effects.go or a CLI runner) is
// responsible for enforcing the manifest at runtime — refusing to grant
// effects that aren't declared, capping budgets, and rejecting transports
// the agent didn't ask for.
//
// M1 ships the generator + JSON schema. M2 wires it through cmd/wasm so
// that `ailang compile --target wasm-reflective` emits this alongside the
// .wasm bundle. M-COG-MESH extends with capability tokens for cross-origin
// auth (TrustedPeers field).

// CapabilityManifest is the schema for the JSON sidecar.
//
// Field naming uses snake_case in JSON to match existing AILANG manifests
// (ailang.toml, motoko-extension manifests, etc.). Go-side fields keep
// idiomatic CamelCase via the json tags.
type CapabilityManifest struct {
	Module     string         `json:"module"`
	Effects    []string       `json:"effects"`
	Budgets    map[string]int `json:"budgets,omitempty"`
	Transports []string       `json:"transports,omitempty"`
}

// NewManifest constructs a manifest with deterministic field ordering.
//
// Effects and Transports lists are sorted lexicographically for replay
// determinism — same inputs → byte-identical JSON output across runs.
// Budgets are stored as a map; map iteration is non-deterministic but JSON
// marshal sorts keys alphabetically (encoding/json contract), so the wire
// output is still byte-stable.
func NewManifest(module string, effects []string, budgets map[string]int, transports []string) *CapabilityManifest {
	sortedEffects := append([]string(nil), effects...)
	sort.Strings(sortedEffects)

	sortedTransports := append([]string(nil), transports...)
	sort.Strings(sortedTransports)

	// Defensive copy of budgets so the caller can mutate the input map
	// without aliasing the manifest's internal state.
	var budgetsCopy map[string]int
	if len(budgets) > 0 {
		budgetsCopy = make(map[string]int, len(budgets))
		for k, v := range budgets {
			budgetsCopy[k] = v
		}
	}

	return &CapabilityManifest{
		Module:     module,
		Effects:    sortedEffects,
		Budgets:    budgetsCopy,
		Transports: sortedTransports,
	}
}

// MarshalJSON emits the manifest with stable key ordering.
//
// Uses encoding/json's default key sort within the Budgets map; the rest
// of the schema has fixed field order via the struct definition. Indent is
// two spaces, matching the convention used by other AILANG sidecar files.
func (m *CapabilityManifest) MarshalJSONIndent() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// Validate checks the manifest for shape errors before emission.
//
// Returns the first error encountered; callers can rely on the error
// message naming the field. Validation is intentionally strict — capability
// manifests are security-critical and the cost of an over-broad declaration
// is much higher than the cost of an explicit rejection.
func (m *CapabilityManifest) Validate() error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}
	if m.Module == "" {
		return fmt.Errorf("manifest.module: must not be empty")
	}
	if len(m.Effects) == 0 {
		return fmt.Errorf("manifest.effects: must declare at least one effect (pure modules don't need a manifest)")
	}
	for _, e := range m.Effects {
		if e == "" {
			return fmt.Errorf("manifest.effects: contains empty string")
		}
	}
	for k, v := range m.Budgets {
		if k == "" {
			return fmt.Errorf("manifest.budgets: contains empty key")
		}
		if v < 0 {
			return fmt.Errorf("manifest.budgets[%q]: negative budget (%d) — use 0 for forbidden, or omit for unbounded", k, v)
		}
	}
	for _, t := range m.Transports {
		if t == "" {
			return fmt.Errorf("manifest.transports: contains empty string")
		}
	}
	return nil
}

// ============================================================================
// Known effect labels and transports — for validation against the canonical set
// ============================================================================

// KnownTransports is the canonical set of transport names this version
// supports. M-COG-RUNTIME ships LocalWorker + BroadcastChannel only;
// M-COG-MESH extends with WebSocket / FirestoreRelay / WebRTC / etc.
//
// Lookup is case-sensitive: transport names match the Go-side trait impl
// names (transport.go in M2).
var KnownTransports = map[string]bool{
	"LocalWorker":      true,
	"BroadcastChannel": true,
	// Future (M-COG-MESH):
	// "WebSocket":       true,
	// "FirestoreRelay":  true,
	// "WebRTC":          true,
	// "A2A":             true,
	// "MCP":             true,
}

// ValidateAgainstKnown checks the manifest's transport list against
// KnownTransports. Effects are NOT validated here (they're validated by
// internal/types/effects.go:IsKnownEffect during type-checking).
//
// Returns the first unknown transport encountered or nil if all known.
// Callers may run this in addition to Validate() to catch typos before
// emission.
func (m *CapabilityManifest) ValidateAgainstKnown() error {
	if err := m.Validate(); err != nil {
		return err
	}
	for _, t := range m.Transports {
		if !KnownTransports[t] {
			return fmt.Errorf("manifest.transports: %q is not a known transport (M-COG-RUNTIME supports: LocalWorker, BroadcastChannel)", t)
		}
	}
	return nil
}
