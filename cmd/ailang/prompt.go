package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/prompt"
)

// runPrompt handles the 'ailang prompt' command
// Displays AILANG teaching prompt for AI code generation
func runPrompt() {
	// Parse prompt subcommand flags
	promptFS := flag.NewFlagSet("prompt", flag.ExitOnError)
	versionFlag := promptFS.String("version", "", "Prompt version to display (e.g., v0.3.24, v0.4.2, or 'latest')")
	listFlag := promptFS.Bool("list", false, "List all available prompt versions")
	infoFlag := promptFS.Bool("info", false, "Show metadata for specified version")
	helpFlag := promptFS.Bool("help", false, "Show help for prompt command")

	_ = promptFS.Parse(flag.Args()[1:])

	if *helpFlag {
		printPromptHelp()
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

	content, err := prompt.LoadPrompt(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Write to stdout (pipe-friendly)
	fmt.Print(content)
}

// printPromptHelp shows help for the prompt command
func printPromptHelp() {
	fmt.Print(`Usage: ailang prompt [OPTIONS]

Display AILANG teaching prompt for AI code generation.

OPTIONS:
  --version VERSION    Display specific version (e.g., v0.3.24, v0.4.2)
                      Default: latest/active version
  --list              List all available prompt versions
  --info              Show metadata for specified version (requires --version)
  --help              Show this help message

EXAMPLES:
  # Display current/latest prompt
  ailang prompt

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
