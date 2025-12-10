package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/messaging"
)

// GitHub integration for messages: import-github, syncMessageToGitHub

func runMessagesImportGitHub(args []string) {
	fs := flag.NewFlagSet("messages import-github", flag.ExitOnError)
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides config default")
	labels := fs.String("labels", "", "Comma-separated labels to filter issues")
	inbox := fs.String("inbox", "user", "Target inbox for imported messages")
	dryRun := fs.Bool("dry-run", false, "Show what would be imported without importing")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Load GitHub config
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check auto_import setting if not explicitly called
	if config != nil && !config.IsAutoImportEnabled() && len(args) == 0 {
		// Auto-import disabled and no explicit args - skip silently
		return
	}

	// Create GitHub client
	client := messaging.NewGitHubClient(config)

	// Parse labels
	var labelList []string
	if *labels != "" {
		labelList = strings.Split(*labels, ",")
		for i := range labelList {
			labelList[i] = strings.TrimSpace(labelList[i])
		}
	}

	// Get repo from flag or config
	repoName := *repo
	if repoName == "" && config != nil {
		repoName = config.DefaultRepo
	}

	// List issues from GitHub
	issues, err := client.ListIssuesByLabel(repoName, labelList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Println("No matching GitHub issues found.")
		return
	}

	// Open store to check for duplicates and insert
	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	imported := 0
	skipped := 0

	for _, issue := range issues {
		// Check if already imported
		exists, err := store.InboxMessageExistsByGitHub(repoName, issue.Number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s checking issue #%d: %v\n", yellow("⚠"), issue.Number, err)
			continue
		}

		if exists {
			skipped++
			continue
		}

		if *dryRun {
			fmt.Printf("  Would import: #%d %s\n", issue.Number, issue.Title)
			imported++
			continue
		}

		// Determine category from labels
		category := ""
		for _, label := range issue.Labels {
			switch label {
			case "bug":
				category = messaging.CategoryBug
			case "feature", "enhancement":
				category = messaging.CategoryFeature
			}
		}

		// Parse from agent from title prefix [agent-name]
		fromAgent := "github"
		title := issue.Title
		if strings.HasPrefix(title, "[") {
			if idx := strings.Index(title, "]"); idx > 0 {
				fromAgent = title[1:idx]
				title = strings.TrimSpace(title[idx+1:])
			}
		}

		// Create inbox message
		msg := &messaging.InboxMessage{
			FromAgent:   fromAgent,
			ToInbox:     *inbox,
			MessageType: messaging.InboxTypeNotification,
			Title:       title,
			Payload:     issue.Body,
			Category:    category,
			GitHubIssue: &issue.Number,
			GitHubRepo:  repoName,
		}

		if err := store.InsertInboxMessage(msg); err != nil {
			fmt.Fprintf(os.Stderr, "%s importing issue #%d: %v\n", yellow("⚠"), issue.Number, err)
			continue
		}

		imported++
	}

	if *dryRun {
		fmt.Printf("\nDry run: would import %d issue(s), skip %d existing\n", imported, skipped)
	} else if imported > 0 {
		fmt.Printf("%s Imported %d new issue(s) from GitHub (%d already existed)\n", green("✓"), imported, skipped)
	} else {
		fmt.Printf("No new issues to import (%d already existed)\n", skipped)
	}
}

// syncMessageToGitHub creates a GitHub issue for the message.
// Returns the issue number on success.
func syncMessageToGitHub(msg *messaging.InboxMessage, repoOverride string) (int, error) {
	// Load GitHub config
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		return 0, fmt.Errorf("failed to load GitHub config: %w", err)
	}

	// Create GitHub client
	client := messaging.NewGitHubClient(config)

	// Determine repo
	repo := repoOverride
	if repo == "" && config != nil {
		repo = config.DefaultRepo
	}

	// Create the issue
	input := messaging.CreateIssueInput{
		Title:     msg.Title,
		Body:      msg.Payload,
		FromAgent: msg.FromAgent,
		Category:  msg.Category,
		Repo:      repo,
	}

	return client.CreateIssue(input)
}
