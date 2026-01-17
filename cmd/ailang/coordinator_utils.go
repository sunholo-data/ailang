package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sunholo/ailang/internal/coordinator"
)

// wrapText wraps text at the specified width
//
//nolint:unused // Scaffolded for future coordinator UI improvements
func wrapText(text string, width int) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		// Handle each line
		if len(line) <= width {
			result.WriteString(line)
			continue
		}
		// Wrap long lines
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine + "\n")
				currentLine = word
			}
		}
		if currentLine != "" {
			result.WriteString(currentLine)
		}
	}
	return result.String()
}

// showRawLogs shows the raw event stream (original format)
func showRawLogs(events []*coordinator.TaskEventRecord) {
	fmt.Println()
	fmt.Println(bold("Raw Event Stream"))
	fmt.Println(strings.Repeat("─", 60))

	for _, event := range events {
		timestamp := event.CreatedAt.Format("15:04:05")

		switch event.StreamType {
		case "turn_start":
			fmt.Printf("%s %s Turn %d started\n", dim(timestamp), blue("◆"), event.TurnNum)
		case "turn_end":
			fmt.Printf("%s %s Turn %d ended\n", dim(timestamp), blue("◇"), event.TurnNum)
		case "text":
			text := event.Text
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			text = strings.ReplaceAll(text, "\n", " ")
			fmt.Printf("%s %s\n", dim(timestamp), text)
		case "tool_use":
			fmt.Printf("%s %s %s\n", dim(timestamp), cyan("🔧"), event.ToolName)
		case "tool_result":
			output := event.ToolOutput
			if len(output) > 60 {
				output = output[:60] + "..."
			}
			fmt.Printf("%s %s %s\n", dim(timestamp), green("→"), output)
		case "error":
			fmt.Printf("%s %s %s\n", dim(timestamp), red("✗"), event.ErrorMsg)
		case "status":
			fmt.Printf("%s %s %s\n", dim(timestamp), yellow("●"), event.Status)
		default:
			if event.Text != "" {
				fmt.Printf("%s %s\n", dim(timestamp), event.Text)
			}
		}
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// fileExists checks if a file/directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// openInFinder opens a directory in the system file manager (Finder on macOS)
func openInFinder(path string) {
	fmt.Println(cyan("→"), "Opening in file manager:", path)
	var cmd *exec.Cmd

	// Use platform-appropriate command
	switch {
	case fileExists("/usr/bin/open"): // macOS
		cmd = exec.Command("open", path)
	case fileExists("/usr/bin/xdg-open"): // Linux
		cmd = exec.Command("xdg-open", path)
	case fileExists("/usr/bin/explorer"): // Windows (unlikely via CLI)
		cmd = exec.Command("explorer", path)
	default:
		fmt.Println(yellow("!"), "No file manager command found")
		fmt.Println("  Path:", path)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(red("✗"), "Failed to open:", err)
	} else {
		fmt.Println(green("✓"), "Opened in file manager")
	}
}

// autoCommitWorktreeChanges checks if there are uncommitted changes in the worktree
// and commits them automatically. This handles the case where the agent creates
// files but doesn't commit them.
//
//nolint:unused // Scaffolded for auto-commit feature
func autoCommitWorktreeChanges(worktreePath, taskTitle string) error {
	// Check for uncommitted changes (untracked or modified)
	statusCmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, nothing to do
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return nil
	}

	fmt.Println(cyan("→"), "Auto-committing uncommitted changes in worktree...")

	// Add all changes
	addCmd := exec.Command("git", "-C", worktreePath, "add", "-A")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %w\n%s", err, output)
	}

	// Commit with a descriptive message
	commitMsg := fmt.Sprintf("Changes for task: %s\n\nAuto-committed by coordinator on approval.", taskTitle)
	commitCmd := exec.Command("git", "-C", worktreePath, "commit", "-m", commitMsg)
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		// Check if it's just "nothing to commit"
		if strings.Contains(string(commitOutput), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("failed to commit: %w\n%s", err, commitOutput)
	}

	fmt.Println(green("✓"), "Auto-committed changes")
	return nil
}
