package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/coordinator"
)

// showWorktreeDiff shows the git diff for a worktree
func showWorktreeDiff(worktreePath string, statOnly bool) {
	hasCommittedChanges := false
	hasUncommittedChanges := false

	// 1. Show committed changes (origin/dev to HEAD)
	var cmd *exec.Cmd
	if statOnly {
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--stat", "origin/dev", "HEAD")
	} else {
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--color=always", "origin/dev", "HEAD")
	}
	output, err := cmd.Output()
	if err != nil {
		// If origin/dev doesn't exist, try dev
		if statOnly {
			cmd = exec.Command("git", "-C", worktreePath, "diff", "--stat", "dev", "HEAD")
		} else {
			cmd = exec.Command("git", "-C", worktreePath, "diff", "--color=always", "dev", "HEAD")
		}
		output, _ = cmd.Output()
	}
	if len(output) > 0 {
		hasCommittedChanges = true
		fmt.Println(bold("Committed changes (origin/dev → HEAD):"))
		fmt.Println(strings.Repeat("─", 50))
		fmt.Print(string(output))
		fmt.Println()
	}

	// 2. Check for uncommitted changes (untracked + modified files)
	statusCmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	statusOutput, _ := statusCmd.Output()
	if len(statusOutput) > 0 {
		hasUncommittedChanges = true
		fmt.Println(bold("Uncommitted changes (will be auto-committed on approve):"))
		fmt.Println(strings.Repeat("─", 50))

		// Show status with nice formatting
		lines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
		for _, line := range lines {
			if len(line) < 3 {
				continue
			}
			status := line[:2]
			file := strings.TrimSpace(line[3:])
			switch {
			case strings.HasPrefix(status, "??"):
				fmt.Println("  " + green("+") + " " + file + " " + dim("(new file)"))
			case strings.HasPrefix(status, "M") || strings.HasPrefix(status, " M"):
				fmt.Println("  " + yellow("~") + " " + file + " " + dim("(modified)"))
			case strings.HasPrefix(status, "D") || strings.HasPrefix(status, " D"):
				fmt.Println("  " + red("-") + " " + file + " " + dim("(deleted)"))
			case strings.HasPrefix(status, "A"):
				fmt.Println("  " + green("+") + " " + file + " " + dim("(staged)"))
			default:
				fmt.Println("  " + status + " " + file)
			}
		}
		fmt.Println()

		// If not stat-only, show actual diff of uncommitted changes
		if !statOnly {
			// Show diff of modified files (not untracked)
			diffCmd := exec.Command("git", "-C", worktreePath, "diff", "--color=always")
			diffCmd.Stdout = os.Stdout
			diffCmd.Stderr = os.Stderr
			diffCmd.Run()
		}
	}

	if !hasCommittedChanges && !hasUncommittedChanges {
		fmt.Println(yellow("!"), "No changes found")
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// browseChangedFiles shows a list of changed files and lets user view them
func browseChangedFiles(worktreePath string) {
	// Parse files with their status
	type changedFile struct {
		status      string
		path        string
		uncommitted bool
	}
	var files []changedFile
	seenPaths := make(map[string]bool)

	// 1. Get committed changes (compare origin/dev to HEAD)
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--name-status", "origin/dev", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		// If origin/dev doesn't exist, try dev
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--name-status", "dev", "HEAD")
		output, _ = cmd.Output()
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			files = append(files, changedFile{
				status:      parts[0],
				path:        parts[1],
				uncommitted: false,
			})
			seenPaths[parts[1]] = true
		}
	}

	// 2. Get uncommitted changes (from git status)
	statusCmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	statusOutput, _ := statusCmd.Output()
	statusLines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
	for _, line := range statusLines {
		if len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		// Skip if we already have this file from committed changes
		if seenPaths[path] {
			continue
		}
		// Convert git status codes to display format
		var displayStatus string
		switch {
		case status == "??":
			displayStatus = "+"
		case strings.Contains(status, "M"):
			displayStatus = "M"
		case strings.Contains(status, "D"):
			displayStatus = "D"
		case strings.Contains(status, "A"):
			displayStatus = "A"
		default:
			displayStatus = status
		}
		files = append(files, changedFile{
			status:      displayStatus,
			path:        path,
			uncommitted: true,
		})
	}

	if len(files) == 0 {
		fmt.Println(yellow("!"), "No changed files found")
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	for {
		fmt.Println()
		fmt.Println(bold("Changed Files:"))
		fmt.Println()
		for i, f := range files {
			var statusStr string
			switch f.status {
			case "A", "+":
				statusStr = green(f.status)
			case "M":
				statusStr = yellow(f.status)
			case "D":
				statusStr = red(f.status)
			default:
				statusStr = dim(f.status)
			}
			uncommittedTag := ""
			if f.uncommitted {
				uncommittedTag = " " + dim("(uncommitted)")
			}
			fmt.Printf("  [%d] %s %s%s\n", i+1, statusStr, f.path, uncommittedTag)
		}
		fmt.Println()
		fmt.Println("  [q] Back")
		fmt.Println()
		fmt.Print("Select file to view (or q to go back): ")

		var input string
		fmt.Scanln(&input)

		if input == "q" || input == "Q" || input == "" {
			return
		}

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(files) {
			fmt.Println("Invalid selection")
			continue
		}

		selectedFile := files[num-1]
		if selectedFile.uncommitted && selectedFile.status == "+" {
			// For new untracked files, just show the file contents
			showNewFileContents(worktreePath, selectedFile.path)
		} else {
			showFileDiff(worktreePath, selectedFile.path)
		}
	}
}

// showFileDiff shows the diff for a specific file
func showFileDiff(worktreePath, filePath string) {
	fmt.Println()
	fmt.Println(bold("Diff for:"), filePath)
	fmt.Println(strings.Repeat("─", 60))

	// Compare origin/dev to HEAD for the specific file (committed changes)
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--color=always", "origin/dev", "HEAD", "--", filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// If origin/dev doesn't exist, try dev
		cmd = exec.Command("git", "-C", worktreePath, "diff", "--color=always", "dev", "HEAD", "--", filePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// showNewFileContents shows the contents of a new (untracked) file
func showNewFileContents(worktreePath, filePath string) {
	fullPath := filepath.Join(worktreePath, filePath)
	fmt.Println()
	fmt.Println(bold("New file:"), filePath)
	fmt.Println(strings.Repeat("─", 60))

	content, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Println(red("✗"), "Failed to read file:", err)
	} else {
		// Show file contents with line numbers
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			fmt.Printf("%s%4d%s │ %s\n", dim(""), i+1, dim(""), green(line))
		}
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// browseWorktreeDirectory lets user browse the worktree directory
func browseWorktreeDirectory(worktreePath, subPath string) {
	currentPath := filepath.Join(worktreePath, subPath)

	for {
		// List directory contents
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			fmt.Println(red("✗"), "Failed to read directory:", err)
			return
		}

		fmt.Println()
		if subPath == "" {
			fmt.Println(bold("Worktree Root:"), worktreePath)
		} else {
			fmt.Println(bold("Directory:"), subPath)
		}
		fmt.Println(strings.Repeat("─", 60))

		// Separate dirs and files
		var dirs, filelist []os.DirEntry
		for _, e := range entries {
			if e.Name() == ".git" {
				continue // Skip .git
			}
			if e.IsDir() {
				dirs = append(dirs, e)
			} else {
				filelist = append(filelist, e)
			}
		}

		// Show directories first
		idx := 1
		entryMap := make(map[int]os.DirEntry)
		for _, d := range dirs {
			fmt.Printf("  [%d] %s/\n", idx, cyan(d.Name()))
			entryMap[idx] = d
			idx++
		}
		// Then files
		for _, f := range filelist {
			info, _ := f.Info()
			size := ""
			if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			fmt.Printf("  [%d] %s%s\n", idx, f.Name(), dim(size))
			entryMap[idx] = f
			idx++
		}

		fmt.Println()
		if subPath != "" {
			fmt.Println("  [u] Go up")
		}
		fmt.Println("  [q] Back to task")
		fmt.Println()
		fmt.Print("Select entry (or q to go back): ")

		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "q", "":
			return
		case "u":
			if subPath != "" {
				subPath = filepath.Dir(subPath)
				if subPath == "." {
					subPath = ""
				}
				currentPath = filepath.Join(worktreePath, subPath)
			}
		default:
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num >= idx {
				fmt.Println("Invalid selection")
				continue
			}

			entry := entryMap[num]
			entryPath := filepath.Join(subPath, entry.Name())

			if entry.IsDir() {
				subPath = entryPath
				currentPath = filepath.Join(worktreePath, subPath)
			} else {
				// Show file contents
				showFileContents(worktreePath, entryPath)
			}
		}
	}
}

// showFileContents displays the contents of a file
func showFileContents(worktreePath, filePath string) {
	fullPath := filepath.Join(worktreePath, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Println(red("✗"), "Failed to read file:", err)
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	fmt.Println()
	fmt.Println(bold("File:"), filePath)
	fmt.Println(strings.Repeat("─", 60))

	// Show file contents (limit to reasonable size)
	lines := strings.Split(string(content), "\n")
	maxLines := 50
	if len(lines) > maxLines {
		for i, line := range lines[:maxLines] {
			fmt.Printf("%4d│ %s\n", i+1, line)
		}
		fmt.Printf("\n... and %d more lines (file truncated)\n", len(lines)-maxLines)
	} else {
		for i, line := range lines {
			fmt.Printf("%4d│ %s\n", i+1, line)
		}
	}

	fmt.Println()
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// showTaskLogs shows the execution logs for a task with grouped conversation view
func showTaskLogs(ctx context.Context, store *coordinator.SQLiteStore, task *coordinator.TaskRecord) {
	events, err := store.GetTaskEvents(ctx, task.ID, 500) // Get more events for full conversation
	if err != nil {
		fmt.Println(red("✗"), "Failed to get logs:", err)
		fmt.Print("Press Enter to continue...")
		fmt.Scanln()
		return
	}

	for {
		fmt.Println()
		fmt.Println(bold("Execution Logs - Conversation View"))
		fmt.Println(strings.Repeat("─", 70))

		if len(events) == 0 {
			fmt.Println(yellow("No events recorded for this task."))
		} else {
			// Group events by turn and aggregate text
			type TurnContent struct {
				TurnNum   int
				StartTime string
				EndTime   string
				Text      strings.Builder
				Tools     []string
			}

			turns := make(map[int]*TurnContent)
			var statusEvents []string
			var errorEvents []string

			for _, event := range events {
				timestamp := event.CreatedAt.Format("15:04:05")

				switch event.StreamType {
				case "turn_start":
					if _, exists := turns[event.TurnNum]; !exists {
						turns[event.TurnNum] = &TurnContent{
							TurnNum:   event.TurnNum,
							StartTime: timestamp,
						}
					}
				case "turn_end":
					if turn, exists := turns[event.TurnNum]; exists {
						turn.EndTime = timestamp
					}
				case "text":
					if event.TurnNum > 0 {
						if _, exists := turns[event.TurnNum]; !exists {
							turns[event.TurnNum] = &TurnContent{TurnNum: event.TurnNum}
						}
						turns[event.TurnNum].Text.WriteString(event.Text)
					}
				case "tool_use":
					if event.TurnNum > 0 {
						if _, exists := turns[event.TurnNum]; !exists {
							turns[event.TurnNum] = &TurnContent{TurnNum: event.TurnNum}
						}
						turns[event.TurnNum].Tools = append(turns[event.TurnNum].Tools, event.ToolName)
					}
				case "status":
					statusEvents = append(statusEvents, fmt.Sprintf("%s %s", timestamp, event.Status))
				case "error":
					errorEvents = append(errorEvents, fmt.Sprintf("%s %s", timestamp, event.ErrorMsg))
				}
			}

			// Show status events
			if len(statusEvents) > 0 {
				fmt.Println(dim("Status:"))
				for _, s := range statusEvents {
					fmt.Printf("  %s %s\n", yellow("●"), s)
				}
				fmt.Println()
			}

			// Show turns in order
			turnNums := make([]int, 0, len(turns))
			for num := range turns {
				turnNums = append(turnNums, num)
			}
			sort.Ints(turnNums)

			for _, num := range turnNums {
				turn := turns[num]
				// Turn header
				timeRange := turn.StartTime
				if turn.EndTime != "" && turn.EndTime != turn.StartTime {
					timeRange = turn.StartTime + " - " + turn.EndTime
				}
				fmt.Printf("%s %s\n", blue(fmt.Sprintf("◆ Turn %d", num)), dim("["+timeRange+"]"))

				// Tools used
				if len(turn.Tools) > 0 {
					fmt.Printf("  %s ", cyan("Tools:"))
					for i, tool := range turn.Tools {
						if i > 0 {
							fmt.Print(", ")
						}
						fmt.Print(tool)
					}
					fmt.Println()
				}

				// Text content - wrap nicely
				text := strings.TrimSpace(turn.Text.String())
				if text != "" {
					// Wrap text at ~70 chars with indent
					wrapped := wrapText(text, 66)
					for _, line := range strings.Split(wrapped, "\n") {
						fmt.Printf("    %s\n", line)
					}
				}
				fmt.Println()
			}

			// Show errors if any
			if len(errorEvents) > 0 {
				fmt.Println(red("Errors:"))
				for _, e := range errorEvents {
					fmt.Printf("  %s %s\n", red("✗"), e)
				}
			}
		}

		// Interactive menu
		hasWorktree := task.WorktreePath != "" && fileExists(task.WorktreePath)

		fmt.Println(strings.Repeat("─", 70))
		fmt.Print("Options: ")
		if hasWorktree {
			fmt.Print("[f] Browse files  [d] View diff  ")
		}
		fmt.Println("[r] Raw logs  [q] Back")
		fmt.Print("> ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "f":
			if hasWorktree {
				browseChangedFiles(task.WorktreePath)
			} else {
				fmt.Println(red("✗"), "No worktree available")
			}
		case "d":
			if hasWorktree {
				showWorktreeDiff(task.WorktreePath, false)
			} else {
				fmt.Println(red("✗"), "No worktree available")
			}
		case "r":
			showRawLogs(events)
		case "q", "":
			return
		}
	}
}
