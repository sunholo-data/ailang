package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sunholo-data/ailang/internal/effects"
)

// M-EXECUTOR-POLICY-HARDENING M3 (D3, D5) — the typed, immutable policy.
//
// A Policy is what the TOML says; a Resolved is what the run is BOUND by:
// every consistency rule has been applied once, the mode is explicit, the
// restricted-mode caps have their defaults, and the digest is over exactly
// the bytes that were decoded. Nothing downstream re-reads the file or
// re-derives authority from a mutable path.

// Security modes (D3). Absent means restricted.
const (
	// ModeRestricted: only effects with a tested confined adapter are
	// admitted; proxies, local/private destinations and host integrations
	// are refused with a named reason. The mode the ailang_only lane runs in.
	ModeRestricted = "restricted"
	// ModeTrustedHost: operator-approved host integrations (Process, AI,
	// Env, …) with conspicuous provenance and NO confinement claim.
	ModeTrustedHost = "trusted_host"
)

// RestrictedEffects are the labels restricted mode may admit — each has an
// adapter with containment tests (FS: M1; Net/Stream: M2). Anything else
// registered in the effect registry is refused until an explicit
// constrained adapter exists, so a future registry addition defaults to
// unsupported (AC7).
var RestrictedEffects = map[string]bool{
	"IO": true, "FS": true, "Net": true, "Clock": true, "Rand": true, "Stream": true,
}

// Proposed restricted-mode defaults (D5). Overridable per policy field.
const (
	DefaultRestrictedMaxSourceBytes      = 1 << 20  // 1 MiB entry file
	DefaultRestrictedMaxModuleGraphBytes = 16 << 20 // 16 MiB across every loaded module
	DefaultRestrictedMaxOutputBytes      = 8 << 20  // 8 MiB combined stdout+stderr
	DefaultRestrictedMaxFSTransferBytes  = 8 << 20  // 8 MiB per FS read or write
	maxTimeout                           = 24 * time.Hour
)

// Resolved is the immutable per-run policy. Slices are copies; callers must
// treat the value as read-only.
type Resolved struct {
	Mode     string
	Digest   string
	Entry    string
	Root     string   // fs_sandbox, "" when FS is not admitted
	Effects  []string // sorted admitted labels
	NetAllow []string
	// NetAllowHTTP permits http:// (and ws://) destinations.
	NetAllowHTTP bool
	ProcessAllow []string
	// CLIAllow is nil when the policy leaves the tool's default set in force.
	CLIAllow []string
	// Budgets are the operator ceilings: present with 0 = zero operations;
	// absent = unlimited for that admitted label.
	Budgets    map[string]int
	Timeout    time.Duration
	AIProvider string

	// Limits. 0 = unbounded (trusted_host with the field unset).
	MaxSourceBytes      int64
	MaxModuleGraphBytes int64
	MaxOutputBytes      int64
	MaxFSTransferBytes  int64
}

// Restricted reports whether the run is in restricted mode.
func (r *Resolved) Restricted() bool { return r.Mode == ModeRestricted }

// Admits reports whether label is an admitted effect.
func (r *Resolved) Admits(label string) bool {
	for _, e := range r.Effects {
		if e == label {
			return true
		}
	}
	return false
}

// DigestBytes is the sha256 hex of the accepted policy bytes — the same
// value executor.PolicyDigest banks, computed from bytes not from a path.
func DigestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// LoadResolved reads the policy file ONCE, decodes strictly, resolves it and
// returns the accepted bytes alongside so the caller can bank exactly what
// was decoded.
func LoadResolved(path string) (*Resolved, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("policy: cannot read %s: %w", path, err)
	}
	p, err := decode(path, data)
	if err != nil {
		return nil, nil, err
	}
	r, err := Resolve(p, DigestBytes(data))
	if err != nil {
		return nil, nil, fmt.Errorf("policy: %s: %w", path, err)
	}
	return r, data, nil
}

// Decode is the strict TOML decode of policy CONTENT (unknown keys are
// errors) for callers that ship the policy by value rather than by path.
func Decode(data []byte) (*Policy, error) { return decode("<content>", data) }

// decode is the one strict TOML decode (unknown keys are errors).
func decode(path string, data []byte) (*Policy, error) {
	p := DefaultPolicy()
	meta, err := toml.Decode(string(data), p)
	if err != nil {
		return nil, fmt.Errorf("policy: %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
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

// Resolve applies every consistency rule to p and returns the immutable
// run policy. Errors name the field and, where one exists, the migration.
func Resolve(p *Policy, digest string) (*Resolved, error) {
	if p == nil {
		return nil, fmt.Errorf("policy: nil policy")
	}
	mode := strings.TrimSpace(p.SecurityMode)
	switch mode {
	case "":
		mode = ModeRestricted
	case ModeRestricted, ModeTrustedHost:
	default:
		return nil, fmt.Errorf("security_mode %q is not one of %q, %q", p.SecurityMode, ModeRestricted, ModeTrustedHost)
	}

	caps := append([]string{}, p.AllowedCaps...)
	sort.Strings(caps)
	has := func(name string) bool {
		for _, c := range caps {
			if c == name {
				return true
			}
		}
		return false
	}
	for i, c := range caps {
		if _, known := effects.Registry[c]; !known {
			return nil, fmt.Errorf("allowed_caps names unknown capability %q (known: %s)", c, knownEffectNames())
		}
		if i > 0 && caps[i-1] == c {
			return nil, fmt.Errorf("allowed_caps lists %q twice", c)
		}
		if mode == ModeRestricted && !RestrictedEffects[c] {
			return nil, fmt.Errorf("allowed_caps admits %s, which restricted mode has no confined adapter for — set security_mode = %q to keep it as an operator-approved host integration, or drop it (restricted admits: %s)",
				c, ModeTrustedHost, sortedKeys(RestrictedEffects))
		}
	}

	// Fine-grained caps: a coarse grant with no narrowing is the dangerous
	// default, so each is refused unless the policy names what it allows;
	// and a narrowing without its cap is a policy that says two things.
	if has("FS") && p.FSSandbox == "" {
		return nil, fmt.Errorf("policy admits FS but sets no fs_sandbox — refusing to run FS unsandboxed")
	}
	if has("Net") && len(p.NetAllow) == 0 {
		return nil, fmt.Errorf("policy admits Net but sets no net_allow — name the hosts or drop Net")
	}
	if len(p.NetAllow) > 0 && !has("Net") && !has("Stream") {
		return nil, fmt.Errorf("net_allow is set but neither Net nor Stream is in allowed_caps")
	}
	if has("Stream") && len(p.NetAllow) == 0 {
		return nil, fmt.Errorf("policy admits Stream but sets no net_allow — Stream destinations are the net_allow hosts")
	}
	if has("Process") && len(p.ProcessAllow) == 0 {
		return nil, fmt.Errorf("policy admits Process but sets no process_allow — name the commands (e.g. [\"git:pull\", \"git:status\"]) or drop Process")
	}
	if len(p.ProcessAllow) > 0 && !has("Process") {
		return nil, fmt.Errorf("process_allow is set but Process is not in allowed_caps")
	}
	for _, entry := range p.ProcessAllow {
		if entry == "" || strings.HasPrefix(entry, ":") || strings.HasSuffix(entry, ":") || strings.Contains(entry, "::") {
			return nil, fmt.Errorf("process_allow entry %q is malformed (want cmd, cmd:sub or cmd:*)", entry)
		}
	}
	if has("AI") && p.AIProvider == "" {
		return nil, fmt.Errorf("policy admits AI but sets no ai_provider — name the model (or \"stub\") or drop AI")
	}
	if p.AIProvider != "" && !has("AI") {
		return nil, fmt.Errorf("ai_provider is set but AI is not in allowed_caps")
	}

	// Limits: timeout_ms is positive and bounded (D5); byte caps non-negative.
	if p.TimeoutMs <= 0 {
		return nil, fmt.Errorf("timeout_ms must be positive, got %d", p.TimeoutMs)
	}
	timeout := time.Duration(p.TimeoutMs) * time.Millisecond
	if timeout > maxTimeout {
		return nil, fmt.Errorf("timeout_ms %d exceeds the %s ceiling", p.TimeoutMs, maxTimeout)
	}
	for name, v := range map[string]int{
		"max_source_bytes": p.MaxSourceBytes, "max_module_graph_bytes": p.MaxModuleGraphBytes,
		"max_output_bytes": p.MaxOutputBytes, "max_fs_transfer_bytes": p.MaxFSTransferBytes,
	} {
		if v < 0 {
			return nil, fmt.Errorf("%s must not be negative, got %d", name, v)
		}
	}

	// Budgets: an explicit zero is zero operations; a budget for an effect
	// the policy does not admit is a contradiction, not a no-op.
	budgets := make(map[string]int, len(p.Budgets))
	for label, n := range p.Budgets {
		if n < 0 {
			return nil, fmt.Errorf("budgets.%s must not be negative, got %d", label, n)
		}
		if !has(label) {
			return nil, fmt.Errorf("budgets names %s, which is not in allowed_caps", label)
		}
		budgets[label] = n
	}

	r := &Resolved{
		Mode:                mode,
		Digest:              digest,
		Entry:               p.Entry,
		Effects:             caps,
		NetAllow:            append([]string{}, p.NetAllow...),
		NetAllowHTTP:        p.NetAllowHTTP,
		ProcessAllow:        append([]string{}, p.ProcessAllow...),
		Budgets:             budgets,
		Timeout:             timeout,
		AIProvider:          p.AIProvider,
		MaxSourceBytes:      int64(p.MaxSourceBytes),
		MaxModuleGraphBytes: int64(p.MaxModuleGraphBytes),
		MaxOutputBytes:      int64(p.MaxOutputBytes),
		MaxFSTransferBytes:  int64(p.MaxFSTransferBytes),
	}
	if r.Entry == "" {
		r.Entry = "main"
	}
	if has("FS") {
		r.Root = p.FSSandbox
	}
	if p.CLIAllow != nil {
		r.CLIAllow = append([]string{}, p.CLIAllow...)
	}
	if mode == ModeRestricted {
		if r.MaxSourceBytes == 0 {
			r.MaxSourceBytes = DefaultRestrictedMaxSourceBytes
		}
		if r.MaxModuleGraphBytes == 0 {
			r.MaxModuleGraphBytes = DefaultRestrictedMaxModuleGraphBytes
		}
		if r.MaxOutputBytes == 0 {
			r.MaxOutputBytes = DefaultRestrictedMaxOutputBytes
		}
		if r.MaxFSTransferBytes == 0 {
			r.MaxFSTransferBytes = DefaultRestrictedMaxFSTransferBytes
		}
	}
	return r, nil
}

func knownEffectNames() string {
	names := make([]string, 0, len(effects.Registry))
	for k := range effects.Registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func sortedKeys(m map[string]bool) string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// AsPolicy is the admission view of a Resolved: the labels Check/CheckScheme
// compare the entry's row against. Nothing else in a Policy is consulted by
// admission.
func (r *Resolved) AsPolicy() *Policy {
	return &Policy{AllowedCaps: append([]string{}, r.Effects...), Entry: r.Entry}
}
