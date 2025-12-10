package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/messaging"
)

// Send and reply operations for messages

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
