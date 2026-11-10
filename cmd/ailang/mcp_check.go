package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/sunholo-data/ailang/internal/mcpcheck"
)

// runMCPCheck implements `ailang mcp check <url>`: verify a live MCP endpoint
// against the Anthropic / OpenAI directory requirements before submitting
// (M-SERVEAPI-DIRECTORY-READY L3). Exit 0 = no FAIL, 1 = at least one FAIL,
// 2 = could not check (usage, unreachable, not MCP).
func runMCPCheck(rawArgs []string) {
	fs := flag.NewFlagSet("mcp check", flag.ExitOnError)
	target := fs.String("target", "anthropic", "Directory to check against: anthropic, openai, both (anthropic + openai), copilot, or all")
	asJSON := fs.Bool("json", false, "Emit findings as JSON")
	probeID := fs.String("probe-client-id", "", "Pre-registered client_id for the consent-page framing probe, for an AS without CIMD (needs --probe-redirect-uri)")
	probeRedirect := fs.String("probe-redirect-uri", "", "Registered redirect_uri of --probe-client-id; the probe never follows a redirect to it")
	_ = fs.Parse(hoistFlagsWith(rawArgs, map[string]bool{
		"--target": true, "-target": true,
		"--probe-client-id": true, "-probe-client-id": true,
		"--probe-redirect-uri": true, "-probe-redirect-uri": true,
	}))
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang mcp check <mcp-url> [--target anthropic|openai|both|copilot|all] [--json]")
		fmt.Fprintln(os.Stderr, "                        [--probe-client-id ID --probe-redirect-uri URI]")
		fmt.Fprintln(os.Stderr, "The consent-page framing probe presents "+mcpcheck.ProbeClientURL)
		fmt.Fprintln(os.Stderr, "to a CIMD authorization server, or the --probe-client-id client otherwise (else it is skipped).")
		fmt.Fprintln(os.Stderr, "Side effect: one pending authorization request on the target, which expires with the AS's request TTL.")
		os.Exit(2)
	}
	if (*probeID == "") != (*probeRedirect == "") {
		fmt.Fprintf(os.Stderr, "%s: --probe-client-id and --probe-redirect-uri go together\n", red("Error"))
		os.Exit(2)
	}
	if *probeRedirect != "" {
		if u, err := url.Parse(*probeRedirect); err != nil || u.Scheme == "" || u.Host == "" {
			fmt.Fprintf(os.Stderr, "%s: --probe-redirect-uri must be an absolute URL (got %q)\n", red("Error"), *probeRedirect)
			os.Exit(2)
		}
	}
	switch *target {
	case "anthropic", "openai", "both", "copilot", "all":
	default:
		fmt.Fprintf(os.Stderr, "%s: --target must be anthropic, openai, both, copilot or all (got %q)\n", red("Error"), *target)
		os.Exit(2)
	}

	findings, err := mcpcheck.Run(context.Background(), mcpcheck.Options{URL: fs.Arg(0), Target: *target,
		ProbeClientID: *probeID, ProbeRedirectURI: *probeRedirect})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot check %s: %v\n", red("Error"), fs.Arg(0), err)
		os.Exit(2)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"url": fs.Arg(0), "target": *target, "failed": mcpcheck.Failed(findings), "findings": findings})
	} else {
		printMCPFindings(fs.Arg(0), *target, findings)
	}
	if mcpcheck.Failed(findings) {
		os.Exit(1)
	}
}

func printMCPFindings(url, target string, findings []mcpcheck.Finding) {
	fmt.Printf("MCP directory check: %s (target: %s)\n\n", url, target)
	for _, f := range findings {
		label := f.Status
		switch f.Status {
		case mcpcheck.Pass:
			label = green(f.Status)
		case mcpcheck.Fail:
			label = red(f.Status)
		case mcpcheck.Warn:
			label = yellow(f.Status)
		}
		subject := f.Check
		if f.Tool != "" {
			subject += " " + f.Tool
		}
		src := ""
		if f.Source != "" {
			src = " [" + f.Source + "]"
		}
		fmt.Printf("  %-4s  %-28s %s%s\n", label, subject, f.Message, src)
	}
	fmt.Println()
	if mcpcheck.Failed(findings) {
		fmt.Println("Not ready to submit. Sources: design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md")
	} else {
		fmt.Println("No failures. Re-fetch the vendor requirements before submitting; they change.")
	}
}
