package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/devtoolsprompt"
)

// runDevtoolsPrompt handles the 'ailang devtools-prompt' command
// Displays AILANG dev tools reference for AI agents
func runDevtoolsPrompt() {
	fs := flag.NewFlagSet("devtools-prompt", flag.ExitOnError)
	versionFlag := fs.String("version", "", "Prompt version to display (e.g., v0.8.0, or 'latest')")
	listFlag := fs.Bool("list", false, "List all available devtools prompt versions")
	infoFlag := fs.Bool("info", false, "Show metadata for specified version")
	compactFlag := fs.Bool("compact", false, "Use token-efficient compact version (~8KB vs ~27KB)")
	helpFlag := fs.Bool("help", false, "Show help for devtools-prompt command")

	_ = fs.Parse(flag.Args()[1:])

	if *helpFlag {
		printDevtoolsPromptHelp()
		return
	}

	if *listFlag {
		listDevtoolsPromptVersions()
		return
	}

	if *infoFlag {
		version := *versionFlag
		if version == "" {
			version = "latest"
		}
		showDevtoolsPromptInfo(version)
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
			activeVer, err := devtoolsprompt.GetActiveVersion()
			if err == nil && activeVer != "" {
				version = activeVer + "-compact"
			} else {
				version = version + "-compact"
			}
		} else {
			version = version + "-compact"
		}
	}

	content, err := devtoolsprompt.LoadPrompt(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Write to stdout (pipe-friendly)
	fmt.Print(content)
}

// printDevtoolsPromptHelp shows help for the devtools-prompt command
func printDevtoolsPromptHelp() {
	fmt.Print(`Usage: ailang devtools-prompt [OPTIONS]

Display AILANG dev tools reference for AI agents.

This complements 'ailang prompt' (language syntax) with toolchain documentation:
how to debug, test, trace, replay, monitor agents, and use the eval harness.

OPTIONS:
  --version VERSION    Display specific version (e.g., v0.8.0)
                      Default: latest/active version
  --compact           Use token-efficient compact version (~8KB vs ~27KB)
  --list              List all available devtools prompt versions
  --info              Show metadata for specified version (requires --version)
  --help              Show this help message

EXAMPLES:
  # Display current dev tools reference
  ailang devtools-prompt

  # Display compact version (for small context windows)
  ailang devtools-prompt --compact

  # List all available versions
  ailang devtools-prompt --list

  # Show metadata for a version
  ailang devtools-prompt --version v0.8.0 --info

  # Pipe to file for AI context injection
  ailang devtools-prompt > devtools_context.md

  # Combine with syntax prompt for full AI context
  cat <(ailang prompt) <(ailang devtools-prompt) > full_context.md

DESCRIPTION:
  The devtools-prompt command provides access to version-locked AILANG dev tools
  references. These references teach AI agents how to use the AILANG toolchain
  for debugging, testing, tracing, agent coordination, and evaluation.

  Organized by workflow (what the AI is trying to accomplish), not by command.

  Complements 'ailang prompt' which covers language syntax only.
  Devtools prompts are versioned and tracked in prompts/devtools/versions.json.
`)
}

// listDevtoolsPromptVersions lists all available devtools prompt versions
func listDevtoolsPromptVersions() {
	versions, err := devtoolsprompt.ListVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, err := devtoolsprompt.GetActiveVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get active version: %v\n", yellow("Warning"), err)
		activeVersion = ""
	}

	fmt.Println("Available devtools prompt versions:")
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

// showDevtoolsPromptInfo displays metadata for a specific devtools prompt version
func showDevtoolsPromptInfo(version string) {
	metadata, err := devtoolsprompt.GetVersionMetadata(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, _ := devtoolsprompt.GetActiveVersion()
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
