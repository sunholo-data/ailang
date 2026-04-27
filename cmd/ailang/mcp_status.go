// `ailang mcp` — small subcommand surface for the remote MCP server (M-AGENT-MCP M5).
//
// `ailang mcp status` reports:
//   - The CLI's compile-time AILANG version (what gets passed as for_version)
//   - The configured MCP endpoint (env: AILANG_MCP_URL, default prod)
//   - Reachability + reported served version (does the server know our version?)
//   - Drift: is the deployed prompt SHA different from the embedded copy's SHA?
//
// This lets users (and agents) answer "should I pull fresh content?" without
// reading the source. Designed to be machine-parseable too — pass --json.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/mcp_client"
	"github.com/sunholo-data/ailang/internal/prompt"
	versionpkg "github.com/sunholo-data/ailang/internal/version"
)

func runMCPCommand() {
	args := flag.Args()
	if len(args) < 2 {
		printMCPHelp()
		return
	}
	switch args[1] {
	case "status":
		runMCPStatus(args[2:])
	case "help", "-h", "--help":
		printMCPHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown mcp subcommand %q. Try `ailang mcp help`.\n", red("Error"), args[1])
		os.Exit(2)
	}
}

func printMCPHelp() {
	fmt.Print(`Usage: ailang mcp <subcommand>

Subcommands:
  status      Show MCP endpoint reachability + drift vs embedded prompt
  help        Show this help

Environment:
  AILANG_MCP_URL   Override the MCP endpoint (default: https://mcp.ailang.sunholo.com/mcp/)
  AILANG_MCP_QUIET Suppress source-provenance stderr line on prompt commands
`)
}

type mcpStatusReport struct {
	CLIVersion     string `json:"cli_version"`
	MCPEndpoint    string `json:"mcp_endpoint"`
	Reachable      bool   `json:"reachable"`
	ServerKnowsVer bool   `json:"server_knows_version"`
	ServedFor      string `json:"served_for,omitempty"`
	DeployedSHA    string `json:"deployed_sha,omitempty"`
	EmbeddedSHA    string `json:"embedded_sha,omitempty"`
	Drift          bool   `json:"drift"`
	Error          string `json:"error,omitempty"`
}

func runMCPStatus(rawArgs []string) {
	fs := flag.NewFlagSet("mcp status", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Emit machine-parseable JSON instead of human text")
	_ = fs.Parse(rawArgs)

	report := mcpStatusReport{
		CLIVersion:  versionpkg.Version,
		MCPEndpoint: os.Getenv("AILANG_MCP_URL"),
		EmbeddedSHA: prompt.EmbeddedSHA256(),
	}
	if report.MCPEndpoint == "" {
		report.MCPEndpoint = mcp_client.DefaultURL
	}

	res, err := prompt.LoadPromptFresh(context.Background(), prompt.FreshOptions{
		Source: prompt.SourceMCP,
		MCPURL: os.Getenv("AILANG_MCP_URL"),
	}, versionpkg.Version)

	switch {
	case err == nil:
		report.Reachable = true
		report.ServerKnowsVer = true
		report.ServedFor = res.Version
		report.DeployedSHA = res.SHA256
		report.Drift = report.DeployedSHA != report.EmbeddedSHA
	default:
		report.Error = err.Error()
		// Best-effort classification: if the error mentions a tool-side
		// version-related response, the server WAS reachable, it just
		// doesn't have content for our CLI version.
		if errIs(err, "unknown_version") || errIs(err, "no prompt content for AILANG") || errIs(err, "no content for AILANG") {
			report.Reachable = true
			report.ServerKnowsVer = false
		}
	}

	if *jsonOut {
		body, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(body))
		return
	}
	printMCPStatusHuman(report)
}

func printMCPStatusHuman(r mcpStatusReport) {
	fmt.Printf("%s %s\n", bold("CLI version:"), r.CLIVersion)
	fmt.Printf("%s %s\n", bold("MCP endpoint:"), r.MCPEndpoint)

	switch {
	case !r.Reachable:
		fmt.Printf("%s unreachable — %s\n", red("Status:"), r.Error)
		fmt.Printf("\n  %s `ailang prompt --source auto` will silently use the embedded copy.\n", yellow("→"))
		return
	case !r.ServerKnowsVer:
		fmt.Printf("%s reachable, but server has no content for %s\n", yellow("Status:"), withVPrefix(r.CLIVersion))
		fmt.Printf("  Embedded SHA: %s\n", shortSHA(r.EmbeddedSHA))
		fmt.Printf("\n  %s `ailang prompt --source auto` will silently use the embedded copy.\n", yellow("→"))
		fmt.Printf("  %s This usually means your CLI is newer than the deployed snapshot — wait for the next release deploy.\n", yellow("→"))
		return
	case r.Drift:
		fmt.Printf("%s reachable, content for %s available\n", green("Status:"), withVPrefix(r.CLIVersion))
		fmt.Printf("  Embedded SHA:  %s\n", shortSHA(r.EmbeddedSHA))
		fmt.Printf("  Deployed SHA:  %s  %s\n", shortSHA(r.DeployedSHA), green("← drift"))
		fmt.Printf("\n  %s Server has fresher prompt content for your version. `ailang prompt --source auto` will use it.\n", green("→"))
	default:
		fmt.Printf("%s reachable, in sync\n", green("Status:"))
		fmt.Printf("  Embedded SHA:  %s\n", shortSHA(r.EmbeddedSHA))
		fmt.Printf("  Deployed SHA:  %s\n", shortSHA(r.DeployedSHA))
		fmt.Printf("\n  %s Embedded and deployed prompts match. Nothing to refresh.\n", green("→"))
	}
}

// withVPrefix ensures a single leading "v" on a version string so output
// reads "v0.14.2" not "vv0.14.2" when the source already includes the prefix.
func withVPrefix(s string) string {
	if len(s) > 0 && s[0] == 'v' {
		return s
	}
	return "v" + s
}

func errIs(err error, substr string) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), substr)
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
