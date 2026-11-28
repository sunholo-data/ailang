package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/prompt"
	versionpkg "github.com/sunholo-data/ailang/internal/version"
)

// runPrompt handles the 'ailang prompt' command
// Displays AILANG teaching prompt for AI code generation
func runPrompt() {
	sub := flag.Args()[1:]
	if len(sub) > 0 && sub[0] == "freeze" {
		runPromptFreeze(sub[1:])
		return
	}
	// Parse prompt subcommand flags
	promptFS := flag.NewFlagSet("prompt", flag.ExitOnError)
	versionFlag := promptFS.String("version", "", "Prompt version to display (e.g., v0.3.24, v0.4.2, or 'latest')")
	listFlag := promptFS.Bool("list", false, "List all available prompt versions")
	infoFlag := promptFS.Bool("info", false, "Show metadata for specified version")
	compactFlag := promptFS.Bool("compact", false, "Use token-efficient compact prompt version (for small context windows)")
	versionActiveFlag := promptFS.Bool("version-active", false, "Print the active prompt version (machine-parseable)")
	sourceFlag := promptFS.String("source", "embedded", "Where to load the prompt from: auto|mcp|embedded (default embedded — current via build, reproducible). auto/mcp fetch the remote MCP copy, which can lag the embedded active version.")
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

	// --compact resolves to <version>-compact, or for default/latest resolves
	// to <active>-compact if present, otherwise the latest available *-compact
	// version in prompts/versions.json.
	if *compactFlag {
		resolved, err := resolveCompactPromptVersion(version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		version = resolved
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
		MCPURL:  config.MCPURL(),
	}, versionpkg.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	// Suppress the info note when stdout is piped — pipe mode is for scripts,
	// stderr should only carry errors/warnings. Explicit AILANG_MCP_QUIET=1 also
	// suppresses; AILANG_MCP_VERBOSE=1 forces the note even in pipe mode for
	// when an operator wants to see it.
	if res.MCPNote != "" && !config.MCPQuiet() && (isStdoutTerminal() || config.MCPVerbose()) {
		fmt.Fprintf(os.Stderr, "%s prompt source=%s version=%s sha=%s (%s)\n",
			yellow("note:"), res.Source, res.Version, shortSHA(res.SHA256), res.MCPNote)
	}

	// Write to stdout (pipe-friendly)
	fmt.Print(res.Content)
}

// resolveCompactPromptVersion resolves --compact against available prompt
// versions. Explicit versions map to "<version>-compact" (unless already
// suffixed). Default/"latest" prefers "<active>-compact" if it exists, and
// otherwise selects the highest-versioned "*-compact" entry in the manifest.
func resolveCompactPromptVersion(requested string) (string, error) {
	if requested != "" && requested != "latest" {
		if strings.HasSuffix(requested, "-compact") {
			return requested, nil
		}
		return requested + "-compact", nil
	}
	versions, err := prompt.ListVersions()
	if err != nil {
		return "", fmt.Errorf("failed to list prompt versions: %w", err)
	}
	activeVer, _ := prompt.GetActiveVersion()
	if activeVer != "" {
		want := activeVer + "-compact"
		for _, v := range versions {
			if v == want {
				return want, nil
			}
		}
	}
	var compacts []string
	for _, v := range versions {
		if strings.HasSuffix(v, "-compact") {
			compacts = append(compacts, v)
		}
	}
	if len(compacts) == 0 {
		return "", fmt.Errorf("no compact prompt version is available in prompts/versions.json")
	}
	return latestCompactVersion(compacts), nil
}

// latestCompactVersion returns the highest "vMAJOR.MINOR.PATCH-compact" entry
// by parsed semver. ASCII order is wrong here ("v0.7.4-compact" sorts above
// "v0.16.21-compact" because '7' > '1'). Names that do not parse sort below
// every parsed version and tie-break lexically, so the result is deterministic.
func latestCompactVersion(compacts []string) string {
	best, bestKey, bestOK := "", [3]int{}, false
	for _, v := range compacts {
		key, ok := parseCompactSemver(v)
		better := best == ""
		switch {
		case better:
		case ok != bestOK:
			better = ok
		case ok && key != bestKey:
			better = key[0] > bestKey[0] ||
				(key[0] == bestKey[0] && (key[1] > bestKey[1] || (key[1] == bestKey[1] && key[2] > bestKey[2])))
		default:
			better = v > best
		}
		if better {
			best, bestKey, bestOK = v, key, ok
		}
	}
	return best
}

// parseCompactSemver parses "v0.16.21-compact" into (0, 16, 21).
func parseCompactSemver(v string) ([3]int, bool) {
	var key [3]int
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(v, "v"), "-compact"), ".")
	if len(parts) != 3 {
		return key, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return key, false
		}
		key[i] = n
	}
	return key, true
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
  --compact           Use token-efficient compact prompt version (for small context windows)
  --list              List all available prompt versions
  --info              Show metadata for specified version (requires --version)
  --help              Show this help message

FREEZE:
  ailang prompt freeze <version>  Freeze a corpus-evidenced prompt version
  ailang prompt freeze --migrate  Record the initial banked/legacy split
  ailang prompt freeze --check    Check derivation, hash, immutability, and mirror invariants

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

  Each prompt version is a curated teaching document — syntax reference,
  examples, and limitations — revised when the teaching needs to change, not
  once per language release.

  Prompts are versioned and tracked in prompts/versions.json.

  NOTE: prompt versions are their own series. They do NOT track the binary's
  release version — the active prompt is numbered independently of
  'ailang --version', so a prompt numbered below the binary is expected, not
  drift. Cross-check the stdlib surface with 'ailang docs', which is generated
  from std/ at build time and is always current.
`)
	if active, err := prompt.GetActiveVersion(); err == nil && active != "" {
		fmt.Printf("\n  Active prompt version: %s (binary: %s)\n", active, versionpkg.Version)
	}
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
