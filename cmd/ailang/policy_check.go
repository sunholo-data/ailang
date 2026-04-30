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

	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read source: %v\n", err)
		os.Exit(1)
	}

	if pol.MaxSourceBytes > 0 && len(source) > pol.MaxSourceBytes {
		emitJSON(policyCheckOutput{
			File:           filename,
			Policy:         *policyPath,
			SourceTooLarge: true,
			Decision: policy.Decision{
				OK:        false,
				ErrorKind: "source_too_large",
				Message:   fmt.Sprintf("source size %d exceeds max_source_bytes=%d", len(source), pol.MaxSourceBytes),
			},
		})
		os.Exit(2)
	}

	// Suppress non-JSON warnings — the policy gate must speak only JSON.
	os.Setenv("AILANG_QUIET_WARNINGS", "1")

	cfg := pipeline.Config{DryLink: true}
	src := pipeline.Source{Code: string(source), Filename: filename, IsREPL: false}

	result, perr := pipeline.Run(cfg, src)
	if perr != nil {
		emitJSON(policyCheckOutput{
			File:   filename,
			Policy: *policyPath,
			Decision: policy.Decision{
				OK:        false,
				ErrorKind: "typecheck_failed",
				Message:   perr.Error(),
			},
		})
		os.Exit(2)
	}
	if len(result.Errors) > 0 {
		emitJSON(policyCheckOutput{
			File:   filename,
			Policy: *policyPath,
			Decision: policy.Decision{
				OK:        false,
				ErrorKind: "typecheck_failed",
				Message:   result.Errors[0].Error(),
			},
		})
		os.Exit(2)
	}

	if result.Interface == nil {
		emitJSON(policyCheckOutput{
			File:   filename,
			Policy: *policyPath,
			Decision: policy.Decision{
				OK:        false,
				ErrorKind: "missing_entry",
				Message:   "no module interface produced — is the file a module?",
			},
		})
		os.Exit(2)
	}

	item, ok := result.Interface.GetExport(pol.Entry)
	if !ok {
		emitJSON(policyCheckOutput{
			File:   filename,
			Policy: *policyPath,
			Module: result.Interface.Module,
			Decision: policy.Decision{
				OK:        false,
				ErrorKind: policy.KindMissingEntry,
				Message:   fmt.Sprintf("entry %q not exported by module %q", pol.Entry, result.Interface.Module),
				Function:  pol.Entry,
			},
		})
		os.Exit(2)
	}

	decision := policy.CheckScheme(pol, pol.Entry, item.Type)

	out := policyCheckOutput{
		File:     filename,
		Policy:   *policyPath,
		Module:   result.Interface.Module,
		Decision: decision,
	}
	emitJSON(out)
	if !decision.OK {
		os.Exit(2)
	}
}

func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
