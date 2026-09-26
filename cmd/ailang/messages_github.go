package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// GitHub integration for messages: import-github, syncMessageToGitHub

func runMessagesImportGitHub(args []string) {
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := otelplatform.Init(ctx, "ailang-messages")
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

		// SECURITY (2026-08-10): the repo is PUBLIC and the issue templates auto-apply
		// `bug`/`enhancement` with no write access needed, so the label filter above
		// says nothing about WHO is speaking. Everything that gives an imported issue
		// directive weight — category (which feeds the auto-triage router), inbox
		// routing, and the spoofable `[agent-name]` sender prefix — is gated on the
		// author here. Untrusted issues still import, because dropping public feedback
		// would be a worse failure; they import INERT.
		trusted := config.IsTrustedAuthor(issue.Author)
		if !trusted {
			category = ""
			targetInbox = *inbox
		}

		if *dryRun {
			routeInfo := ""
			if targetInbox == "coordinator" {
				routeInfo = fmt.Sprintf(" [auto-routed to %s]", targetInbox)
			}
			if !trusted {
				routeInfo = fmt.Sprintf(" [UNTRUSTED author %s — inert]", issue.Author)
			}
			fmt.Printf("  Would import: #%d %s%s\n", issue.Number, issue.Title, routeInfo)
			imported++
			continue
		}

		// Parse from agent from title prefix [agent-name].
		// Only for trusted authors: otherwise an outsider titling an issue
		// "[mission-world] ..." would import as that sibling mission's message.
		fromAgent := "github"
		title := issue.Title
		if trusted && strings.HasPrefix(title, "[") {
			if idx := strings.Index(title, "]"); idx > 0 {
				fromAgent = title[1:idx]
				title = strings.TrimSpace(title[idx+1:])
			}
		}
		if !trusted {
			// Encode the real principal, and make it unmistakable downstream that
			// this is public feedback rather than an instruction.
			fromAgent = "github-untrusted:" + issue.Author
			title = "[untrusted] " + title
		}

		// Extract and cache images from issue body (before JWT tokens expire)
		modifiedBody, imagePaths, err := messaging.ExtractAndCacheImages(ctx, issue.Body, issue.Number, repoName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s extracting images from issue #%d: %v\n", yellow("⚠"), issue.Number, err)
			modifiedBody = issue.Body // Fall back to original
		} else if len(imagePaths) > 0 {
			fmt.Printf("  Cached %d image(s) for issue #%d\n", len(imagePaths), issue.Number)
		}

		// Create inbox message
		msg := &messaging.InboxMessage{
			FromAgent:   fromAgent,
			ToInbox:     targetInbox,
			MessageType: messaging.InboxTypeNotification,
			Title:       title,
			Payload:     modifiedBody, // Use modified body with local image paths
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

	// Determine repo: CLI flag > inbox-specific mapping > default
	repo := repoOverride
	if repo == "" && config != nil {
		repo = config.RepoForInbox(msg.ToInbox)
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

// runMessagesGitHubSync retries the GitHub sync for a locally-stored message
// whose send-time sync failed (#754). The send path always saves the message
// to the store BEFORE syncing, and on a GitHub 5xx it prints this subcommand's
// name as the recovery hint — but the subcommand itself did not exist, so a
// 5xx during issue creation left the message stranded locally with a hint
// naming nothing.
func runMessagesGitHubSync(args []string) {
	fs := flag.NewFlagSet("messages github-sync", flag.ExitOnError)
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides message's repo")
	githubUser := fs.String("github-user", "", "Override expected GitHub user (bypass config.expected_user)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "%s: github-sync requires exactly one message ID\n", red("Error"))
		fmt.Fprintln(os.Stderr, "  Usage: ailang messages github-sync MSG_ID [--repo owner/repo]")
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message %q not found\n", red("Error"), msgID)
		os.Exit(1)
	}

	// Idempotent by default: a message that already carries an issue number is
	// not re-posted — re-running the hint after a partial success must not
	// create a duplicate issue.
	if msg.GitHubIssue != nil {
		fmt.Printf("%s Message %s is already synced to #%d\n", yellow("⚠"), msg.MessageID, *msg.GitHubIssue)
		return
	}

	issueNum, err := syncMessageToGitHub(msg, *repo, *githubUser)
	if err != nil {
		if errors.Is(err, messaging.ErrAccountMismatch) {
			fmt.Fprintf(os.Stderr, "\n%s %v\n\n", red("ERROR:"), err)
		} else {
			fmt.Fprintf(os.Stderr, "%s GitHub sync failed: %v\n", yellow("⚠"), err)
		}
		os.Exit(1)
	}

	if err := store.UpdateInboxMessageGitHub(msg.MessageID, issueNum, *repo); err != nil {
		fmt.Fprintf(os.Stderr, "%s Could not save issue number: %v\n", yellow("⚠"), err)
	}
	fmt.Printf("%s GitHub issue #%d created for %s\n", green("✓"), issueNum, msg.MessageID)
}
