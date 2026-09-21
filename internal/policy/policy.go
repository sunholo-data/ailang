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
	AllowedCaps []string `toml:"allowed_caps"`
	FSSandbox   string   `toml:"fs_sandbox"`
	NetAllow    []string `toml:"net_allow"`
	// NetAllowHTTP permits http:// (default https only). ProcessAllow is the
	// Process allowlist in `ailang run --process-allowlist` syntax: `git`,
	// `git:pull`, `gh:pr:list`, `git:*` — a binary narrowed to its subcommands.
	// Both are meaningful only with the matching cap in AllowedCaps.
	NetAllowHTTP bool     `toml:"net_allow_http"`
	ProcessAllow []string `toml:"process_allow"`
	// CLIAllow is the `ailang` subcommand allowlist for an agent whose only
	// route to the binary is the ailang_cli tool (the ailang_only lane): `iface`,
	// `docs:search` — a subcommand narrowed to its own subcommand, the
	// process_allow syntax. It is read by the pi extension, not by `ailang run`
	// (which admits programs, not commands); it lives here so one policy file
	// states everything the agent may do. Absent means the tool's documented
	// read-only default set; `run`/`test`/`exec`/`repl` are refused whatever
	// the list says — execution only ever goes through the gate.
	CLIAllow       []string       `toml:"cli_allow"`
	Budgets        map[string]int `toml:"budgets"`
	TimeoutMs      int            `toml:"timeout_ms"`
	MaxSourceBytes int            `toml:"max_source_bytes"`
	AIProvider     string         `toml:"ai_provider"`
	Entry          string         `toml:"entry"`

	// SecurityMode is "restricted" (absent = restricted) or "trusted_host"
	// (M-EXECUTOR-POLICY-HARDENING D3). Restricted admits only effects with
	// a confined adapter and refuses proxies and host integrations; see
	// Resolve for the rules and the migration message.
	SecurityMode string `toml:"security_mode"`
	// Byte ceilings (D5). 0 = the restricted default in restricted mode,
	// unbounded in trusted_host.
	MaxModuleGraphBytes int `toml:"max_module_graph_bytes"`
	MaxOutputBytes      int `toml:"max_output_bytes"`
	MaxFSTransferBytes  int `toml:"max_fs_transfer_bytes"`
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
	return decode(path, data)
}

// AllowedSet returns the AllowedCaps slice as a set for O(1) membership tests.
func (p *Policy) AllowedSet() map[string]bool {
	s := make(map[string]bool, len(p.AllowedCaps))
	for _, c := range p.AllowedCaps {
		s[c] = true
	}
	return s
}
