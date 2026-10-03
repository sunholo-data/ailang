package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/policytool"
)

// policyToolCommand is the local tool endpoint of the ailang_only lane
// (M-EXECUTOR-POLICY-HARDENING M4, D4):
//
//	ailang policy-tool [--policy <agent-policy.toml>] < request.json > response.json
//	ailang policy-tool [--policy <agent-policy.toml>] --request-file request.json > response.json
//
// One JSON request on stdin (or in --request-file, for a caller that cannot
// pipe stdin: an AILANG program's std/process.exec takes no input), one JSON response on stdout, exit 0 whenever a
// response was produced (a refusal is a response, not an exit code). The
// policy path comes from --policy or AILANG_AGENT_POLICY — the launcher's
// handle — and never from the request. Exit 1 only when no response could
// be formed (no policy, unreadable request).
func policyToolCommand() {
	fs := flag.NewFlagSet("policy-tool", flag.ExitOnError)
	policyPath := fs.String("policy", "", "Path to the operator policy (default: $AILANG_AGENT_POLICY)")
	requestFile := fs.String("request-file", "", "Read the JSON request from this file instead of stdin (for callers that cannot pipe stdin)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	if *policyPath == "" {
		*policyPath = config.AgentPolicy()
	}
	if *policyPath == "" {
		fmt.Fprintln(os.Stderr, "policy-tool: no policy — pass --policy or export AILANG_AGENT_POLICY (the launcher's handle); the tool refuses by default")
		os.Exit(1)
	}
	var in io.Reader = os.Stdin
	if *requestFile != "" {
		f, ferr := os.Open(*requestFile)
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "policy-tool: opening --request-file: %v\n", ferr)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}
	raw, err := io.ReadAll(io.LimitReader(in, 64<<20))
	if err != nil {
		fmt.Fprintf(os.Stderr, "policy-tool: reading the request: %v\n", err)
		os.Exit(1)
	}
	var req policytool.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(os.Stderr, "policy-tool: request is not JSON: %v\n", err)
		os.Exit(1)
	}
	host, err := policytool.Open(*policyPath)
	if err != nil {
		// A policy that does not resolve is a refusal the model can read.
		emitJSON(policytool.Response{OK: false, Refused: err.Error()})
		return
	}
	defer host.Close()
	if exe, err := os.Executable(); err == nil {
		host.SetBinary(exe)
	}
	emitJSON(host.Dispatch(req))
}
