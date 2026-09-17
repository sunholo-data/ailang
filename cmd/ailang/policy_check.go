package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/policy"
)

// policyCheckOutput is the stable JSON shape of `ailang policy-check`.
// Field names are part of the runner contract — do not rename without
// bumping the message-schema version (see m-agent-safe-runner design doc).
type policyCheckOutput struct {
	File     string          `json:"file"`
	Policy   string          `json:"policy"`
	Module   string          `json:"module,omitempty"`
	Decision policy.Decision `json:"decision"`
	// SourceTooLarge is set when the source exceeds policy.MaxSourceBytes
	// before any typecheck runs. Distinct from a typecheck error.
	SourceTooLarge bool `json:"source_too_large,omitempty"`
}

// policyCheckCommand is the M1 spike of M-AGENT-SAFE-RUNNER.
//
// Usage:
//
//	ailang policy-check --policy <agent-policy.toml> <file.ail>
//
// Exit codes:
//
//	0 — admitted
//	2 — admission denied (with structured JSON)
//	1 — internal error (bad policy file, I/O, etc.)
func policyCheckCommand() {
	fs := flag.NewFlagSet("policy-check", flag.ExitOnError)
	policyPath := fs.String("policy", "", "Path to agent-policy.toml (required)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *policyPath == "" || fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang policy-check --policy <agent-policy.toml> <file.ail>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Statically gates an AILANG program against an operator-pinned policy.")
		fmt.Fprintln(os.Stderr, "Output is JSON. Exits 0 on admission, 2 on denial, 1 on internal error.")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	pol, err := policy.Load(*policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "policy load error: %v\n", err)
		os.Exit(1)
	}

	out, code := admitProgram(pol, *policyPath, filename)
	emitJSON(out)
	os.Exit(code)
}

// admitProgram is the ONE admission path (M-AGENT-AILANG-ONLY-EXECUTION D1):
// size cap → typecheck (DryLink; imports resolve, so a lying entry that calls
// an imported effectful function fails HERE, before the policy is consulted)
// → entry export → static cap-subset check. Returns the stable JSON shape and
// the exit code the caller should use: 0 admitted, 2 denied, 1 internal.
// Shared by `policy-check` and `run --policy` so the two can never disagree
// about the same program.
func admitProgram(pol *policy.Policy, policyPath, filename string) (policyCheckOutput, int) {
	deny := func(kind policy.ErrorKind, msg string, extra func(*policyCheckOutput)) (policyCheckOutput, int) {
		out := policyCheckOutput{File: filename, Policy: policyPath, Decision: policy.Decision{OK: false, ErrorKind: kind, Message: msg}}
		if extra != nil {
			extra(&out)
		}
		return out, 2
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		return policyCheckOutput{File: filename, Policy: policyPath, Decision: policy.Decision{OK: false, ErrorKind: "read_failed", Message: err.Error()}}, 1
	}

	if pol.MaxSourceBytes > 0 && len(source) > pol.MaxSourceBytes {
		return deny("source_too_large", fmt.Sprintf("source size %d exceeds max_source_bytes=%d", len(source), pol.MaxSourceBytes),
			func(o *policyCheckOutput) { o.SourceTooLarge = true })
	}

	// Suppress non-JSON warnings — the policy gate must speak only JSON.
	os.Setenv("AILANG_QUIET_WARNINGS", "1")

	cfg := pipeline.Config{DryLink: true}
	src := pipeline.Source{Code: string(source), Filename: filename, IsREPL: false}

	result, perr := pipeline.Run(cfg, src)
	if perr != nil {
		return deny("typecheck_failed", perr.Error(), nil)
	}

	if result.Interface == nil {
		return deny("missing_entry", "no module interface produced — is the file a module?", nil)
	}

	item, ok := result.Interface.GetExport(pol.Entry)
	if !ok {
		return deny(policy.KindMissingEntry, fmt.Sprintf("entry %q not exported by module %q", pol.Entry, result.Interface.Module),
			func(o *policyCheckOutput) { o.Module = result.Interface.Module; o.Decision.Function = pol.Entry })
	}

	decision := policy.CheckScheme(pol, pol.Entry, item.Type)
	out := policyCheckOutput{File: filename, Policy: policyPath, Module: result.Interface.Module, Decision: decision}
	if !decision.OK {
		return out, 2
	}
	return out, 0
}

func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
