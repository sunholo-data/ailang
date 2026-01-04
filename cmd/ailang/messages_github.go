package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// GitHub integration for messages: import-github, syncMessageToGitHub

func runMessagesImportGitHub(args []string) {
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-messages")
	if err != nil {
		// Non-fatal: continue without telemetry
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Start span for GitHub sync operation
	tracer := otel.Tracer("ailang.messaging")
	_, span := tracer.Start(ctx, "messages.github_sync")
	defer span.End()

	fs := flag.NewFlagSet("messages import-github", flag.ExitOnError)
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides config default")
	labels := fs.String("labels", "", "Comma-separated labels to filter issues")
	inbox := fs.String("inbox", "user", "Target inbox for imported messages")
	dryRun := fs.Bool("dry-run", false, "Show what would be imported without importing")
	githubUser := fs.String("github-user", "", "Override expected GitHub user (bypass config.expected_user)")
	routeByLabel := fs.Bool("route-by-label", true, "Route issues with coordinator:* labels to coordinator inbox")

	if err := fs.Parse(args); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Load GitHub config
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check auto_import setting if not explicitly called
	if config != nil && !config.IsAutoImportEnabled() && len(args) == 0 {
		// Auto-import disabled and no explicit args - skip silently
		span.SetStatus(codes.Ok, "auto-import disabled")
		return
	}

	// Create GitHub client
	client := messaging.NewGitHubClient(config)

	// Set override user if provided
	if *githubUser != "" {
		client.SetOverrideUser(*githubUser)
	}

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

	// Add repo attribute to span
	span.SetAttributes(
		attribute.String("github.repo", repoName),
		attribute.Bool("sync.dry_run", *dryRun),
	)

	// List issues from GitHub
	issues, err := client.ListIssuesByLabel(repoName, labelList)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Check for account mismatch - show prominently with fix options
		if errors.Is(err, messaging.ErrAccountMismatch) {
			fmt.Fprintf(os.Stderr, "\n%s %v\n", red("ERROR:"), err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	span.SetAttributes(attribute.Int("github.issues_found", len(issues)))

	if len(issues) == 0 {
		span.SetStatus(codes.Ok, "no matching issues")
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

		// Determine category and target inbox from labels (before dry-run check)
		category := ""
		targetInbox := *inbox

		for _, label := range issue.Labels {
			switch label {
			case "bug":
				category = messaging.CategoryBug
			case "feature", "enhancement":
				category = messaging.CategoryFeature
			}

			// Label-based routing: coordinator:* labels route to coordinator inbox
			if *routeByLabel && strings.HasPrefix(label, "coordinator:") {
				targetInbox = "coordinator"
				// Extract task type from label (e.g., "coordinator:bug" -> "bug")
				coordinatorTaskType := strings.TrimPrefix(label, "coordinator:")
				// Override category based on coordinator task type
				switch coordinatorTaskType {
				case "bug":
					category = messaging.CategoryBug
				case "feature":
					category = messaging.CategoryFeature
				case "docs":
					category = messaging.CategoryDocs
				case "research":
					category = messaging.CategoryResearch
				case "refactor":
					category = messaging.CategoryRefactor
				case "test":
					category = messaging.CategoryTest
				}
			}
		}

		if *dryRun {
			routeInfo := ""
			if targetInbox == "coordinator" {
				routeInfo = fmt.Sprintf(" [auto-routed to %s]", targetInbox)
			}
			fmt.Printf("  Would import: #%d %s%s\n", issue.Number, issue.Title, routeInfo)
			imported++
			continue
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
			ToInbox:     targetInbox,
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

	// Record final counts on span
	span.SetAttributes(
		attribute.Int("sync.imported", imported),
		attribute.Int("sync.skipped", skipped),
	)
	span.SetStatus(codes.Ok, "sync complete")

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
// If githubUserOverride is non-empty, bypasses expected_user validation if that user is active.
func syncMessageToGitHub(msg *messaging.InboxMessage, repoOverride string, githubUserOverride string) (int, error) {
	// Load GitHub config
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		return 0, fmt.Errorf("failed to load GitHub config: %w", err)
	}

	// Create GitHub client
	client := messaging.NewGitHubClient(config)

	// Set override user if provided
	if githubUserOverride != "" {
		client.SetOverrideUser(githubUserOverride)
	}

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
