package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/policy"
)

// M-AGENT-AILANG-ONLY-EXECUTION M2 — `ailang run --policy <agent-policy.toml>`.
//
// The caller is an AGENT, and an agent that can pass --caps has no policy.
// So with --policy the authority comes from the file and nowhere else: caps,
// the Net allowlist and the FS sandbox are DERIVED from it, and every flag
// that could widen them is refused outright rather than merged. Admission
// itself is admitProgram — the same path `policy-check` runs, so the two can
// never disagree.
//
// Output contract: on DENIAL the decision JSON goes to stdout and the exit
// code is 2 — nothing executes. On ADMISSION one JSON line goes to STDERR
// (prefixed "policy: ") so stdout stays the program's own; the pi tool
// (ailang-exec.ts) reads that line for the decision and the policy digest.

// runPolicyWidening names the flags a caller must not combine with --policy.
type runPolicyWidening struct {
	caps      string
	noBudgets bool
	allowEnv  string
}

// runPolicyResolved is what the policy DECIDES for the run.
type runPolicyResolved struct {
	caps       string
	netDomains string
	sandbox    string
	digest     string
}

// applyRunPolicy loads the policy, refuses widening flags, validates the
// policy against the effect registry, admits the program, and returns the
// derived run settings. It exits the process on refusal (1) or denial (2).
func applyRunPolicy(policyPath, filename string, w runPolicyWidening) runPolicyResolved {
	refuse := func(format string, a ...any) {
		fmt.Fprintf(os.Stderr, "%s: --policy: %s\n", red("Error"), fmt.Sprintf(format, a...))
		os.Exit(1)
	}

	// Widening flags: refused by NAME, so the agent's transcript says which one.
	switch {
	case w.caps != "":
		refuse("--caps is not allowed with --policy — capabilities come from the policy's allowed_caps")
	case w.noBudgets:
		refuse("--no-budgets is not allowed with --policy")
	case w.allowEnv != "":
		refuse("--allow-env is not allowed with --policy — environment access is not a policy field")
	}

	pol, err := policy.Load(policyPath)
	if err != nil {
		refuse("%v", err)
	}

	// A policy that declares budgets would be silently unenforced: run-time
	// budgets come from source annotations today. Refuse rather than pretend.
	if len(pol.Budgets) > 0 {
		refuse("policy declares [budgets] (%s), which `run --policy` cannot enforce yet — remove them or enforce via source annotations", joinKeys(pol.Budgets))
	}

	// Every allowed cap must be a real effect: a typo would silently narrow
	// the policy AND make the run's --caps fail later with a less useful error.
	for _, c := range pol.AllowedCaps {
		if _, known := effects.Registry[c]; !known {
			refuse("allowed_caps names unknown capability %q (known: %s)", c, knownEffects())
		}
	}

	if hasCap(pol, "FS") && pol.FSSandbox == "" {
		refuse("policy admits FS but sets no fs_sandbox — refusing to run FS unsandboxed")
	}

	out, code := admitProgram(pol, policyPath, filename)
	if code != 0 {
		emitJSON(out)
		os.Exit(code)
	}

	resolved := runPolicyResolved{
		caps:       strings.Join(pol.AllowedCaps, ","),
		netDomains: strings.Join(pol.NetAllow, ","),
		sandbox:    pol.FSSandbox,
		digest:     executor.PolicyDigest(policyPath),
	}
	if resolved.sandbox != "" {
		// The effects context reads the sandbox from the environment
		// (internal/config.FSSandbox). Setting it here — after the policy has
		// been read and before the runtime starts — is what makes the policy,
		// not the caller's environment, the authority. os.Setenv is how
		// check.go and ai_check.go already export per-run task variables.
		os.Setenv(config.EnvFSSandbox, resolved.sandbox)
	}

	line, _ := json.Marshal(map[string]any{
		"ok":            true,
		"policy":        policyPath,
		"policy_digest": resolved.digest,
		"caps":          pol.AllowedCaps,
		"fs_sandbox":    resolved.sandbox,
		"net_allow":     pol.NetAllow,
		"decision":      out.Decision,
	})
	fmt.Fprintf(os.Stderr, "policy: %s\n", line)
	return resolved
}

func hasCap(p *policy.Policy, name string) bool {
	for _, c := range p.AllowedCaps {
		if c == name {
			return true
		}
	}
	return false
}

func joinKeys(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func knownEffects() string {
	names := make([]string, 0, len(effects.Registry))
	for k := range effects.Registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
