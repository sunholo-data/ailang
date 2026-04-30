// Package policy implements the operator-pinned execution policy used by the
// safe-runner pattern (M-AGENT-SAFE-RUNNER, planned v0.16.0).
//
// A Policy is loaded from a TOML file authored by an operator and pinned at
// deploy time. It declares which capabilities a submitted .ail program is
// allowed to use. The policy admission check is a row-subset comparison
// against the program's declared effect row — see [Check].
//
// SPIKE STATUS: This is the M1 spike for the design doc at
// design_docs/planned/v0_16_0/m-agent-safe-runner.md. It implements only
// monomorphic admission control; parametric (open) entry-function effect
// rows are rejected with a clear error, which is acceptable for v1 because
// `main` is conventionally monomorphic.
package policy

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Policy is the operator-pinned execution policy for AI-authored programs.
//
// Field semantics:
//   - AllowedCaps: closed set of effect labels permitted for the entry
//     function's declared effect row. Empty list = deny-all (the default).
//   - FSSandbox: filesystem root the runner will export as AILANG_FS_SANDBOX.
//   - NetAllow: hostname allowlist passed to the Net effect handler.
//   - Budgets: per-effect operation caps.
//   - TimeoutMs: hard timeout for the program.
//   - MaxSourceBytes: cap on the submitted source size.
//   - AIProvider: "stub" or a model name; controls the AI effect handler.
//   - Entry: name of the exported function to invoke.
type Policy struct {
	AllowedCaps    []string       `toml:"allowed_caps"`
	FSSandbox      string         `toml:"fs_sandbox"`
	NetAllow       []string       `toml:"net_allow"`
	Budgets        map[string]int `toml:"budgets"`
	TimeoutMs      int            `toml:"timeout_ms"`
	MaxSourceBytes int            `toml:"max_source_bytes"`
	AIProvider     string         `toml:"ai_provider"`
	Entry          string         `toml:"entry"`
}

// DefaultPolicy returns a deny-all policy. This is what an empty file decodes
// to (TOML zero value) — the default is intentionally restrictive.
func DefaultPolicy() *Policy {
	return &Policy{
		AllowedCaps: []string{},
		Entry:       "main",
		TimeoutMs:   5000,
	}
}

// Load reads and parses a policy file from disk.
//
// Errors fall into two buckets:
//   - I/O errors (missing file, unreadable) — returned wrapped
//   - TOML errors (bad syntax, type mismatch) — returned wrapped with line info
//     where BurntSushi provides it
//
// Unknown fields are surfaced as a separate error so a typo in the policy
// file fails loudly instead of silently being ignored — A4 (explicit authority).
func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy: cannot read %s: %w", path, err)
	}

	p := DefaultPolicy()
	meta, err := toml.Decode(string(data), p)
	if err != nil {
		return nil, fmt.Errorf("policy: %s: %w", path, err)
	}

	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		// Convert MetaData keys to strings for the error
		names := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			names = append(names, key.String())
		}
		return nil, fmt.Errorf("policy: %s: unknown fields: %v", path, names)
	}

	if p.Entry == "" {
		p.Entry = "main"
	}

	return p, nil
}

// AllowedSet returns the AllowedCaps slice as a set for O(1) membership tests.
func (p *Policy) AllowedSet() map[string]bool {
	s := make(map[string]bool, len(p.AllowedCaps))
	for _, c := range p.AllowedCaps {
		s[c] = true
	}
	return s
}
