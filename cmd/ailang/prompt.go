package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/prompt"
	versionpkg "github.com/sunholo-data/ailang/internal/version"
)

// runPrompt handles the 'ailang prompt' command
// Displays AILANG teaching prompt for AI code generation
func runPrompt() {
	// Parse prompt subcommand flags
	promptFS := flag.NewFlagSet("prompt", flag.ExitOnError)
	versionFlag := promptFS.String("version", "", "Prompt version to display (e.g., v0.3.24, v0.4.2, or 'latest')")
	listFlag := promptFS.Bool("list", false, "List all available prompt versions")
	infoFlag := promptFS.Bool("info", false, "Show metadata for specified version")
	compactFlag := promptFS.Bool("compact", false, "Use token-efficient compact version (~15KB vs ~49KB)")
	versionActiveFlag := promptFS.Bool("version-active", false, "Print the active prompt version (machine-parseable)")
	sourceFlag := promptFS.String("source", "auto", "Where to load the prompt from: auto|mcp|embedded (default auto). Use embedded for reproducible eval runs.")
	helpFlag := promptFS.Bool("help", false, "Show help for prompt command")

	_ = promptFS.Parse(flag.Args()[1:])

	if *helpFlag {
		printPromptHelp()
		return
	}

	if *versionActiveFlag {
		v, err := prompt.GetActiveVersion()
		if err != nil || v == "" {
			fmt.Fprintf(os.Stderr, "%s: no active prompt version resolvable (run 'ailang prompt --list')\n", red("Error"))
			os.Exit(1)
		}
		fmt.Println(v)
		return
	}

	// Handle --list flag
	if *listFlag {
		listPromptVersions()
		return
	}

	// Handle --info flag
	if *infoFlag {
		version := *versionFlag
		if version == "" {
			version = "latest"
		}
		showPromptInfo(version)
		return
	}

	// Default: display prompt content
	version := *versionFlag
	if version == "" {
		version = "latest"
	}

	// --compact appends "-compact" to the resolved version
	if *compactFlag {
		if version == "latest" {
			// Resolve latest to actual version first, then append -compact
			activeVer, err := prompt.GetActiveVersion()
			if err == nil && activeVer != "" {
				version = activeVer + "-compact"
			} else {
				version = version + "-compact"
			}
		} else {
			version = version + "-compact"
		}
	}

	// New: --source auto|mcp|embedded routes through LoadPromptFresh, which
	// prefers the MCP-served (version-locked) copy when reachable AND the
	// served_for matches the binary's compile-time version. Falls back to
	// embedded silently when MCP is unreachable or version-mismatched.
	src := prompt.Source(*sourceFlag)
	if src != prompt.SourceAuto && src != prompt.SourceMCP && src != prompt.SourceEmbedded {
		fmt.Fprintf(os.Stderr, "%s: --source must be auto, mcp, or embedded\n", red("Error"))
		os.Exit(2)
	}

	res, err := prompt.LoadPromptFresh(context.Background(), prompt.FreshOptions{
		Source:  src,
		Version: version,
		MCPURL:  os.Getenv("AILANG_MCP_URL"),
	}, versionpkg.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	// Suppress the info note when stdout is piped — pipe mode is for scripts,
	// stderr should only carry errors/warnings. Explicit AILANG_MCP_QUIET=1 also
	// suppresses; AILANG_MCP_VERBOSE=1 forces the note even in pipe mode for
	// when an operator wants to see it.
	if res.MCPNote != "" && os.Getenv("AILANG_MCP_QUIET") == "" && (isStdoutTerminal() || os.Getenv("AILANG_MCP_VERBOSE") != "") {
		fmt.Fprintf(os.Stderr, "%s prompt source=%s version=%s sha=%s (%s)\n",
			yellow("note:"), res.Source, res.Version, shortSHA(res.SHA256), res.MCPNote)
	}

	// Write to stdout (pipe-friendly)
	fmt.Print(res.Content)
}

// shortSHA returns the first 7 chars of a hex sha (git-style).
func shortSHA(s string) string {
	if len(s) < 7 {
		return s
	}
	return s[:7]
}

// isStdoutTerminal reports whether stdout is a TTY. Used to suppress
// informational stderr notes (e.g. "cache hit") when the command is piped
// — scripts capturing stdout shouldn't have to filter benign info out of
// stderr too.
func isStdoutTerminal() bool {
	st, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// printPromptHelp shows help for the prompt command
func printPromptHelp() {
	fmt.Print(`Usage: ailang prompt [OPTIONS]

Display AILANG teaching prompt for AI code generation.

OPTIONS:
  --version VERSION    Display specific version (e.g., v0.3.24, v0.4.2)
                      Default: latest/active version
  --compact           Use token-efficient compact version (~15KB vs ~49KB)
  --list              List all available prompt versions
  --info              Show metadata for specified version (requires --version)
  --help              Show this help message

EXAMPLES:
  # Display current/latest prompt
  ailang prompt

  # Display compact version (for small context windows)
  ailang prompt --compact

  # Display specific version
  ailang prompt --version v0.3.24

  # List all available versions
  ailang prompt --list

  # Show metadata for a version
  ailang prompt --version v0.4.2 --info

  # Pipe to file
  ailang prompt > syntax.md

  # Pipe to pager
  ailang prompt | less

DESCRIPTION:
  The prompt command provides access to version-locked AILANG teaching prompts.
  These prompts are used by AI models to generate AILANG code and are validated
  through multi-model evaluation benchmarks.

  Each prompt version corresponds to a specific AILANG language version and
  includes comprehensive syntax reference, examples, and limitations.

  The active version (v0.16.0) covers the IFC label system: T<label>
  source annotations, T{not LABEL} sink refinements, and the Declassify
  effect for prompt-injection-resistant agent code. Use --version v0.12.1
  for the prior version without IFC labels.

  Prompts are versioned and tracked in prompts/versions.json.
`)
}

// listPromptVersions lists all available prompt versions
func listPromptVersions() {
	versions, err := prompt.ListVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, err := prompt.GetActiveVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get active version: %v\n", yellow("Warning"), err)
		activeVersion = ""
	}

	fmt.Println("Available prompt versions:")
	fmt.Println()

	for _, version := range versions {
		if version == activeVersion {
			fmt.Printf("  %s %s\n", green("*"), bold(version))
		} else {
			fmt.Printf("    %s\n", version)
		}
	}

	fmt.Println()
	if activeVersion != "" {
		fmt.Printf("* = active version (%s)\n", activeVersion)
	}
}

// showPromptInfo displays metadata for a specific prompt version
func showPromptInfo(version string) {
	metadata, err := prompt.GetVersionMetadata(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, _ := prompt.GetActiveVersion()
	isActive := false
	if version == "latest" || version == "" {
		version = activeVersion
		isActive = true
	} else if version == activeVersion {
		isActive = true
	}

	fmt.Printf("%s: %s", bold("Version"), version)
	if isActive {
		fmt.Printf(" %s", green("(active)"))
	}
	fmt.Println()

	fmt.Printf("%s: %s\n", bold("File"), metadata.File)
	fmt.Printf("%s: %s\n", bold("Created"), metadata.Created)
	fmt.Printf("%s: %s\n", bold("Description"), metadata.Description)

	if len(metadata.Tags) > 0 {
		fmt.Printf("%s: ", bold("Tags"))
		for i, tag := range metadata.Tags {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(tag)
		}
		fmt.Println()
	}

	if metadata.Notes != "" {
		fmt.Printf("%s:\n  %s\n", bold("Notes"), metadata.Notes)
	}

	if metadata.Hash != "" && metadata.Hash != "PLACEHOLDER" {
		hashDisplay := metadata.Hash
		if len(hashDisplay) > 16 {
			hashDisplay = hashDisplay[:16] + "..."
		}
		fmt.Printf("%s: %s\n", bold("Hash"), hashDisplay)
	}
}
