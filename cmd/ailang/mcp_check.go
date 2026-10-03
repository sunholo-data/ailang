package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/mcpcheck"
)

// runMCPCheck implements `ailang mcp check <url>`: verify a live MCP endpoint
// against the Anthropic / OpenAI directory requirements before submitting
// (M-SERVEAPI-DIRECTORY-READY L3). Exit 0 = no FAIL, 1 = at least one FAIL,
// 2 = could not check (usage, unreachable, not MCP).
func runMCPCheck(rawArgs []string) {
	fs := flag.NewFlagSet("mcp check", flag.ExitOnError)
	target := fs.String("target", "anthropic", "Directory to check against: anthropic, openai or both")
	asJSON := fs.Bool("json", false, "Emit findings as JSON")
	_ = fs.Parse(hoistFlagsWith(rawArgs, map[string]bool{"--target": true, "-target": true}))
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang mcp check <mcp-url> [--target anthropic|openai|both] [--json]")
		os.Exit(2)
	}
	switch *target {
	case "anthropic", "openai", "both":
	default:
		fmt.Fprintf(os.Stderr, "%s: --target must be anthropic, openai or both (got %q)\n", red("Error"), *target)
		os.Exit(2)
	}

	findings, err := mcpcheck.Run(context.Background(), mcpcheck.Options{URL: fs.Arg(0), Target: *target})
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
