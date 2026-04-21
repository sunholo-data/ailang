package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/agentprompt"
)

// runAgentPrompt handles the 'ailang agent-prompt' command
// Displays minimal AILANG coding guide for iterative agentic coders
func runAgentPrompt() {
	fs := flag.NewFlagSet("agent-prompt", flag.ExitOnError)
	versionFlag := fs.String("version", "", "Prompt version to display (e.g., v0.8.2, or 'latest')")
	listFlag := fs.Bool("list", false, "List all available agent prompt versions")
	infoFlag := fs.Bool("info", false, "Show metadata for specified version")
	helpFlag := fs.Bool("help", false, "Show help for agent-prompt command")

	_ = fs.Parse(flag.Args()[1:])

	if *helpFlag {
		printAgentPromptHelp()
		return
	}

	if *listFlag {
		listAgentPromptVersions()
		return
	}

	if *infoFlag {
		version := *versionFlag
		if version == "" {
			version = "latest"
		}
		showAgentPromptInfo(version)
		return
	}

	// Default: display prompt content
	version := *versionFlag
	if version == "" {
		version = "latest"
	}

	content, err := agentprompt.LoadPrompt(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Write to stdout (pipe-friendly)
	fmt.Print(content)
}

// printAgentPromptHelp shows help for the agent-prompt command
func printAgentPromptHelp() {
	fmt.Print(`Usage: ailang agent-prompt [OPTIONS]

Display minimal AILANG coding guide for iterative agentic coders.

This is a compact reference for agents that write AILANG code iteratively
(write → check → run → fix). For full language syntax use 'ailang prompt'.
For toolchain reference use 'ailang devtools-prompt'.

OPTIONS:
  --version VERSION    Display specific version (e.g., v0.8.2)
                      Default: latest/active version
  --list              List all available agent prompt versions
  --info              Show metadata for specified version (requires --version)
  --help              Show this help message

EXAMPLES:
  # Display current agent coding guide
  ailang agent-prompt

  # List all available versions
  ailang agent-prompt --list

  # Show metadata for a version
  ailang agent-prompt --version v0.8.2 --info

  # Pipe to file for AI context injection
  ailang agent-prompt > agent_context.md

THREE PROMPT TYPES:
  ailang prompt           Full language syntax reference (~1600 lines)
  ailang devtools-prompt  Toolchain/CLI reference (~600 lines)
  ailang agent-prompt     Minimal agent coding guide (~180 lines)
`)
}

// listAgentPromptVersions lists all available agent prompt versions
func listAgentPromptVersions() {
	versions, err := agentprompt.ListVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, err := agentprompt.GetActiveVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get active version: %v\n", yellow("Warning"), err)
		activeVersion = ""
	}

	fmt.Println("Available agent prompt versions:")
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

// showAgentPromptInfo displays metadata for a specific agent prompt version
func showAgentPromptInfo(version string) {
	metadata, err := agentprompt.GetVersionMetadata(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	activeVersion, _ := agentprompt.GetActiveVersion()
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
