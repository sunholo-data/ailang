package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// humanDuration supports human-friendly duration parsing including "d" for days.
// Examples: "7d", "30d", "24h", "1h30m", "168h"
type humanDuration time.Duration

func (d *humanDuration) String() string {
	return time.Duration(*d).String()
}

func (d *humanDuration) Set(s string) error {
	// Check for day suffix (e.g., "7d", "30d")
	dayRegex := regexp.MustCompile(`^(\d+)d$`)
	if matches := dayRegex.FindStringSubmatch(s); len(matches) == 2 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return err
		}
		*d = humanDuration(time.Duration(days) * 24 * time.Hour)
		return nil
	}

	// Fall back to standard Go duration parsing
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = humanDuration(parsed)
	return nil
}

// messagesCommand handles the 'messages' (alias: 'msg') subcommand.
// This uses the unified collaboration.db for both CLI and dashboard access.
func messagesCommand() {
	if len(os.Args) < 3 {
		runMessagesList([]string{})
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "list", "ls":
		runMessagesList(args)
	case "ack":
		runMessagesAck(args)
	case "unack":
		runMessagesUnack(args)
	case "send":
		runMessagesSend(args)
	case "read":
		runMessagesRead(args)
	case "watch":
		runMessagesWatch(args)
	case "cleanup":
		runMessagesCleanup(args)
	case "import-github":
		runMessagesImportGitHub(args)
	case "reply":
		runMessagesReply(args)
	case "--help", "-h", "help":
		printMessagesHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subCmd)
		printMessagesHelp()
		os.Exit(1)
	}
}

// openStore opens the unified collaboration database
func openStore() (*messaging.Store, error) {
	dbPath := messaging.GetDefaultDatabasePath()
	return messaging.OpenStore(dbPath)
}

func runMessagesList(args []string) {
	fs := flag.NewFlagSet("messages list", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox (user, claude-code, etc.)")
	unread := fs.Bool("unread", false, "Show only unread messages")
	from := fs.String("from", "", "Filter by sender agent")
	limit := fs.Int("limit", 20, "Maximum messages to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	opts := messaging.InboxListOptions{
		Inbox:      *inbox,
		FromAgent:  *from,
		Limit:      *limit,
		UnreadOnly: *unread,
	}

	messages, err := store.ListInboxMessages(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(messages, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(messages) == 0 {
		fmt.Println("No messages found.")
		return
	}

	// Get counts for summary
	counts, _ := store.CountInboxMessagesByStatus(*inbox)

	// Print summary
	fmt.Printf("\n%s\n\n", bold("Messages"))
	if counts[messaging.InboxStatusUnread] > 0 {
		fmt.Printf("  Unread: %s\n", yellow(fmt.Sprintf("%d", counts[messaging.InboxStatusUnread])))
	}
	if counts[messaging.InboxStatusRead] > 0 {
		fmt.Printf("  Read: %d\n", counts[messaging.InboxStatusRead])
	}
	fmt.Println()

	// Print messages
	for _, msg := range messages {
		printInboxMessage(msg, false)
	}
}

func runMessagesAck(args []string) {
	fs := flag.NewFlagSet("messages ack", flag.ExitOnError)
	all := fs.Bool("all", false, "Acknowledge all unread messages")
	inbox := fs.String("inbox", "", "Filter by inbox when using --all")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	if *all {
		count, err := store.MarkAllInboxMessagesRead(*inbox)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s %d message(s) marked as read.\n", green("✓"), count)
		return
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required (or use --all)\n", red("Error"))
		os.Exit(1)
	}

	msgID := fs.Arg(0)
	if err := store.MarkInboxMessageRead(msgID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Message marked as read.\n", green("✓"))
}

func runMessagesUnack(args []string) {
	fs := flag.NewFlagSet("messages unack", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID := fs.Arg(0)
	if err := store.MarkInboxMessageUnread(msgID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Message marked as unread.\n", green("✓"))
}

func runMessagesSend(args []string) {
	fs := flag.NewFlagSet("messages send", flag.ExitOnError)
	// Note: --payload is preferred over --json to avoid confusion with --json output flag
	payloadFlag := fs.String("payload", "", "Send structured payload (alternative to positional message)")
	title := fs.String("title", "", "Message title")
	from := fs.String("from", "cli", "Sender agent name")
	correlationID := fs.String("correlation", "", "Correlation ID for grouping messages")

	// GitHub sync flags
	github := fs.Bool("github", false, "Also create a GitHub issue")
	msgType := fs.String("type", "", "Message type: bug, feature, general (implies --github)")
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides config default")

	// Normalize args: move flags before positional arguments
	// Go's flag package requires flags to come first, but users often put them at the end
	args = normalizeArgsForFlags(args, []string{"payload", "title", "from", "correlation", "github", "type", "repo"})

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: inbox required\n", red("Error"))
		fmt.Println("Usage: ailang messages send <inbox> [message]")
		fmt.Println("       ailang messages send <inbox> --payload '{...}'")
		os.Exit(1)
	}

	inbox := fs.Arg(0)
	var payload string

	if *payloadFlag != "" {
		payload = *payloadFlag
	} else if fs.NArg() >= 2 {
		payload = strings.Join(fs.Args()[1:], " ")
	} else {
		fmt.Fprintf(os.Stderr, "%s: message content required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Determine message title
	msgTitle := *title
	if msgTitle == "" {
		msgTitle = truncateString(payload, 50)
	}

	// Determine category from --type flag
	var category string
	if *msgType != "" {
		switch *msgType {
		case "bug":
			category = messaging.CategoryBug
		case "feature":
			category = messaging.CategoryFeature
		case "general":
			category = messaging.CategoryGeneral
		default:
			fmt.Fprintf(os.Stderr, "%s: invalid --type value '%s' (must be bug, feature, or general)\n", red("Error"), *msgType)
			os.Exit(1)
		}
	}

	// If --type is specified, imply --github
	syncToGitHub := *github || *msgType != ""

	msg := &messaging.InboxMessage{
		FromAgent:     *from,
		ToInbox:       inbox,
		MessageType:   messaging.InboxTypeNotification,
		Title:         msgTitle,
		Payload:       payload,
		CorrelationID: *correlationID,
		Category:      category,
		GitHubRepo:    *repo,
	}

	// ALWAYS save to SQLite first
	if err := store.InsertInboxMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message sent to '%s' (ID: %s)\n", green("✓"), inbox, msg.MessageID)

	// Optionally sync to GitHub
	if syncToGitHub {
		issueNum, err := syncMessageToGitHub(msg, *repo)
		if err != nil {
			// Message saved locally, but GitHub sync failed
			fmt.Fprintf(os.Stderr, "%s GitHub sync failed: %v\n", yellow("⚠"), err)
			fmt.Println("  Message saved locally. Retry GitHub sync with:")
			fmt.Printf("  ailang messages github-sync %s\n", msg.MessageID)
		} else {
			// Update message with issue number
			if err := store.UpdateInboxMessageGitHub(msg.MessageID, issueNum, *repo); err != nil {
				fmt.Fprintf(os.Stderr, "%s Could not save issue number: %v\n", yellow("⚠"), err)
			}
			fmt.Printf("%s GitHub issue #%d created\n", green("✓"), issueNum)
		}
	}
}

func runMessagesRead(args []string) {
	fs := flag.NewFlagSet("messages read", flag.ExitOnError)
	peek := fs.Bool("peek", false, "Show without marking as read")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID := fs.Arg(0)
	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message not found\n", red("Error"))
		os.Exit(1)
	}

	// Mark as read unless peeking
	if !*peek && msg.Status == messaging.InboxStatusUnread {
		_ = store.MarkInboxMessageRead(msgID) // Ignore error for auto-mark
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(data))
		return
	}

	printInboxMessage(*msg, true)
}

func runMessagesWatch(args []string) {
	fs := flag.NewFlagSet("messages watch", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	interval := fs.Duration("interval", time.Second, "Poll interval")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	fmt.Println("Watching for new messages... (Ctrl+C to stop)")
	fmt.Println()

	seen := make(map[string]bool)

	// First, mark all existing messages as seen
	existing, _ := store.ListInboxMessages(messaging.InboxListOptions{
		Inbox:      *inbox,
		UnreadOnly: true,
	})
	for _, msg := range existing {
		seen[msg.ID] = true
	}

	for {
		messages, err := store.ListInboxMessages(messaging.InboxListOptions{
			Inbox:      *inbox,
			UnreadOnly: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			time.Sleep(*interval)
			continue
		}

		for _, msg := range messages {
			if !seen[msg.ID] {
				seen[msg.ID] = true
				fmt.Printf("%s New message:\n", green("→"))
				printInboxMessage(msg, false)
			}
		}

		time.Sleep(*interval)
	}
}

func runMessagesCleanup(args []string) {
	fs := flag.NewFlagSet("messages cleanup", flag.ExitOnError)
	olderThan := humanDuration(7 * 24 * time.Hour) // Default: 7 days
	fs.Var(&olderThan, "older-than", "Remove messages older than this (e.g., 7d, 30d, 168h)")
	expired := fs.Bool("expired", false, "Remove only expired messages")
	dryRun := fs.Bool("dry-run", false, "Show what would be deleted without deleting")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	if *dryRun {
		// Just show counts
		counts, _ := store.CountInboxMessagesByStatus("")
		fmt.Printf("Would clean up messages older than %v:\n", time.Duration(olderThan))
		fmt.Printf("  Deleted: %d\n", counts[messaging.InboxStatusDeleted])
		fmt.Printf("  (Dry run - no changes made)\n")
		return
	}

	count, err := store.CleanupInboxMessages(time.Duration(olderThan), *expired)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Cleaned up %d message(s).\n", green("✓"), count)
}

func runMessagesReply(args []string) {
	fs := flag.NewFlagSet("messages reply", flag.ExitOnError)
	from := fs.String("from", "cli", "Sender agent name")
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides message's repo")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "%s: reply requires MSG_ID and reply text\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Usage: ailang messages reply MSG_ID \"Your reply\" [--from agent]")
		os.Exit(1)
	}

	msgID := fs.Arg(0)
	replyText := fs.Arg(1)

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Look up the original message
	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: message not found: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check if the message has a linked GitHub issue
	if msg.GitHubIssue == nil {
		fmt.Fprintf(os.Stderr, "%s: message has no linked GitHub issue\n", red("Error"))
		fmt.Fprintln(os.Stderr, "This message was not created with --github flag.")
		fmt.Fprintln(os.Stderr, "Use 'ailang messages send' to create a new message instead.")
		os.Exit(1)
	}

	// Load GitHub config and create client
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		// Config is optional, but we need it for the default repo
		config = nil
	}
	ghClient := messaging.NewGitHubClient(config)

	// Determine the repo: CLI flag > message repo > config default
	targetRepo := msg.GitHubRepo
	if *repo != "" {
		targetRepo = *repo
	}
	if targetRepo == "" {
		// Try to get default from config
		cfg := ghClient.GetConfig()
		if cfg != nil && cfg.DefaultRepo != "" {
			targetRepo = cfg.DefaultRepo
		}
	}
	if targetRepo == "" {
		fmt.Fprintf(os.Stderr, "%s: no repository specified\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Use --repo flag or configure default_repo in ~/.ailang/config.yaml")
		os.Exit(1)
	}

	// Format the comment with attribution
	commentBody := replyText
	if *from != "" {
		commentBody = fmt.Sprintf("%s\n\n---\n_Reply by: %s via ailang messages_", replyText, *from)
	}

	if err := ghClient.AddComment(targetRepo, *msg.GitHubIssue, commentBody); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to add comment: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Reply added to GitHub issue #%d in %s\n", green("✓"), *msg.GitHubIssue, targetRepo)
}

// printInboxMessage formats and prints a message to stdout.
func printInboxMessage(msg messaging.InboxMessage, full bool) {
	statusIcon := "○"
	if msg.Status == messaging.InboxStatusUnread {
		statusIcon = yellow("●")
	} else if msg.Status == messaging.InboxStatusRead {
		statusIcon = green("○")
	}

	age := formatAge(msg.CreatedAt)

	fmt.Printf("  %s [%s] %s • %s\n",
		statusIcon,
		cyan(msg.ToInbox),
		msg.FromAgent,
		age,
	)

	if msg.Title != "" {
		fmt.Printf("    %s\n", bold(msg.Title))
	}

	// Truncate payload for list view
	payload := msg.Payload
	if !full && len(payload) > 100 {
		payload = payload[:97] + "..."
	}

	// Try to pretty-print JSON
	if strings.HasPrefix(strings.TrimSpace(payload), "{") {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &parsed); err == nil {
			if full {
				formatted, _ := json.MarshalIndent(parsed, "    ", "  ")
				fmt.Printf("    %s\n", string(formatted))
			} else {
				// Compact summary
				if content, ok := parsed["content"].(string); ok {
					if len(content) > 100 {
						content = content[:97] + "..."
					}
					fmt.Printf("    %s\n", content)
				} else {
					fmt.Printf("    %s\n", payload)
				}
			}
		} else {
			fmt.Printf("    %s\n", payload)
		}
	} else {
		fmt.Printf("    %s\n", payload)
	}

	if full {
		fmt.Printf("\n    ID: %s\n", msg.MessageID)
		if msg.CorrelationID != "" {
			fmt.Printf("    Correlation: %s\n", msg.CorrelationID)
		}
		fmt.Printf("    Status: %s\n", msg.Status)
		fmt.Printf("    Created: %s\n", msg.CreatedAt.Format(time.RFC3339))
		if msg.ReadAt != nil {
			fmt.Printf("    Read: %s\n", msg.ReadAt.Format(time.RFC3339))
		}
	}
	fmt.Println()
}

// formatAge returns a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	} else if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

// truncateString truncates a string to maxLen and adds "..." if needed.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// normalizeArgsForFlags moves flags to the front of args so Go's flag package can parse them.
// Go's flag package stops parsing when it sees a non-flag argument, but users often put
// flags at the end (e.g., "send inbox message --title foo" instead of "send --title foo inbox message").
func normalizeArgsForFlags(args []string, flagNames []string) []string {
	// Build a set of known flag names (with -- prefix)
	knownFlags := make(map[string]bool)
	for _, name := range flagNames {
		knownFlags["--"+name] = true
		knownFlags["-"+name] = true
	}

	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Check if this is a known flag
		isFlag := false
		for flagName := range knownFlags {
			if arg == flagName {
				isFlag = true
				// Flag with separate value
				flags = append(flags, arg)
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					flags = append(flags, args[i])
				}
				break
			}
			if strings.HasPrefix(arg, flagName+"=") {
				isFlag = true
				// Flag with = value
				flags = append(flags, arg)
				break
			}
		}
		if !isFlag {
			positional = append(positional, arg)
		}
	}

	// Flags first, then positional arguments
	return append(flags, positional...)
}

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

func printMessagesHelp() {
	fmt.Println("Usage: ailang messages <subcommand> [options]")
	fmt.Println()
	fmt.Println("Unified messaging system for agent-to-agent and human-agent communication.")
	fmt.Println("Messages are accessible from both CLI and Collaboration Hub dashboard.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Printf("  %s                     List messages (default)\n", cyan("list"))
	fmt.Printf("  %s <id>                Mark message as read\n", cyan("ack"))
	fmt.Printf("  %s <id>              Mark message as unread\n", cyan("unack"))
	fmt.Printf("  %s <inbox> <msg>      Send a message\n", cyan("send"))
	fmt.Printf("  %s <id> <text>       Reply to GitHub issue thread\n", cyan("reply"))
	fmt.Printf("  %s <id>               Show full message content\n", cyan("read"))
	fmt.Printf("  %s                    Watch for new messages\n", cyan("watch"))
	fmt.Printf("  %s                  Clean up old messages\n", cyan("cleanup"))
	fmt.Printf("  %s            Import GitHub issues as messages\n", cyan("import-github"))
	fmt.Println()
	fmt.Println("List Flags:")
	fmt.Println("  --inbox <name>       Filter by inbox (user, claude-code, etc.)")
	fmt.Println("  --unread             Show only unread messages")
	fmt.Println("  --from <agent>       Filter by sender")
	fmt.Println("  --limit <n>          Maximum messages to show (default: 20)")
	fmt.Println("  --json               Output as JSON")
	fmt.Println()
	fmt.Println("Ack Flags:")
	fmt.Println("  --all                Acknowledge all unread messages")
	fmt.Println("  --inbox <name>       Filter by inbox when using --all")
	fmt.Println()
	fmt.Println("Send Flags:")
	fmt.Println("  --payload <data>     Send payload via flag (alternative to positional arg)")
	fmt.Println("  --title <text>       Message title")
	fmt.Println("  --from <agent>       Sender name (default: cli)")
	fmt.Println("  --correlation <id>   Correlation ID for grouping")
	fmt.Println()
	fmt.Println("GitHub Sync Flags (send):")
	fmt.Println("  --github             Also create a GitHub issue")
	fmt.Println("  --type <type>        Message type: bug, feature, general (implies --github)")
	fmt.Println("  --repo <owner/repo>  GitHub repo (overrides config default)")
	fmt.Println()
	fmt.Println("Reply Flags:")
	fmt.Println("  --from <agent>       Sender name for attribution (default: cli)")
	fmt.Println("  --repo <owner/repo>  Override repo from original message")
	fmt.Println()
	fmt.Println("Import GitHub Flags:")
	fmt.Println("  --repo <owner/repo>  GitHub repo to import from")
	fmt.Println("  --labels <list>      Comma-separated labels to filter issues")
	fmt.Println("  --inbox <name>       Target inbox for imported messages (default: user)")
	fmt.Println("  --dry-run            Show what would be imported without importing")
	fmt.Println()
	fmt.Println("Note: Flags can appear before or after positional arguments.")
	fmt.Println()
	fmt.Println("Aliases: msg, messages")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                         # List all messages\n", cyan("ailang messages list"))
	fmt.Printf("  %s               # List unread only\n", cyan("ailang messages list --unread"))
	fmt.Printf("  %s          # Filter by inbox\n", cyan("ailang messages list --inbox user"))
	fmt.Printf("  %s              # Mark as read\n", cyan("ailang messages ack MSG_ID"))
	fmt.Printf("  %s                     # Ack all unread\n", cyan("ailang messages ack --all"))
	fmt.Printf("  %s   # Send message\n", cyan("ailang messages send user \"Hello\""))
	fmt.Printf("  %s    # Watch for new\n", cyan("ailang messages watch --inbox user"))
	fmt.Println()
	fmt.Println("GitHub Sync Examples:")
	fmt.Printf("  %s\n", cyan("ailang messages send ailang-core \"Parser crash\" --type bug --github"))
	fmt.Printf("  %s\n", cyan("ailang messages send user \"Add dark mode\" --type feature"))
	fmt.Printf("  %s\n", cyan("ailang messages send user \"Hello\" --github --repo owner/repo"))
	fmt.Printf("  %s\n", cyan("ailang messages reply MSG_ID \"Fixed in v0.5.10\" --from claude-code"))
	fmt.Println()
	fmt.Println("Import GitHub Examples:")
	fmt.Printf("  %s\n", cyan("ailang messages import-github"))
	fmt.Printf("  %s\n", cyan("ailang messages import-github --labels bug,feature"))
	fmt.Printf("  %s\n", cyan("ailang messages import-github --dry-run"))
}
