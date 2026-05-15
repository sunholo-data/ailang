package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Send and reply operations for messages

func runMessagesSend(args []string) {
	fs := flag.NewFlagSet("messages send", flag.ExitOnError)
	// Note: --payload is preferred over --json to avoid confusion with --json output flag
	payloadFlag := fs.String("payload", "", "Send structured payload (alternative to positional message)")
	title := fs.String("title", "", "Message title")
	from := fs.String("from", "cli", "Sender agent name")
	correlationID := fs.String("correlation", "", "Correlation ID for grouping messages")
	force := fs.Bool("force", false, "Force send even if duplicate exists")

	// Task hierarchy flags (M-UNIFIED-AI-CONTROL-PLANE)
	parentTaskID := fs.String("parent-task", "", "Parent task ID for hierarchical execution")

	// Envelope flags (M-SEMANTIC-ENVELOPE)
	envelopeCode := fs.String("envelope-code", "", "File paths for code envelope slot (comma-separated, or 'auto' for git-detected)")
	envelopeContext := fs.String("envelope-context", "", "Context description for context envelope slot")
	noEnvelope := fs.Bool("no-envelope", false, "Skip all automatic envelope computation")

	// GitHub sync flags
	github := fs.Bool("github", false, "Also create a GitHub issue")
	msgType := fs.String("type", "", "Message category (any string; bug/feature imply --github)")
	repo := fs.String("repo", "", "GitHub repo (owner/repo) - overrides config default")
	githubUser := fs.String("github-user", "", "Override expected GitHub user (bypass config.expected_user)")

	// Normalize args: move flags before positional arguments
	// Go's flag package requires flags to come first, but users often put them at the end
	args = normalizeArgsForFlags(args, []string{"payload", "title", "from", "correlation", "force", "parent-task", "envelope-code", "envelope-context", "no-envelope", "github", "type", "repo", "github-user"})

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

	// Determine category from --type flag (any string allowed)
	category := *msgType

	// Special behavior for known types: bug/feature imply --github
	knownGitHubTypes := map[string]bool{
		messaging.CategoryBug:     true,
		messaging.CategoryFeature: true,
	}
	syncToGitHub := *github || knownGitHubTypes[category]

	// For bug reports, auto-append binary version and MD5 for reproducibility
	if category == messaging.CategoryBug {
		payload = appendBinaryInfo(payload)
	}

	msg := &messaging.InboxMessage{
		FromAgent:     *from,
		ToInbox:       inbox,
		MessageType:   messaging.InboxTypeNotification,
		Title:         msgTitle,
		Payload:       payload,
		CorrelationID: *correlationID,
		ParentTaskID:  *parentTaskID,
		Category:      category,
		GitHubRepo:    *repo,
	}

	// Check for duplicate messages (same title in same inbox)
	if !*force {
		existingID, err := store.InboxMessageExistsByTitle(inbox, msgTitle)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: checking duplicates: %v\n", yellow("⚠"), err)
			// Continue anyway - dedup check is best-effort
		} else if existingID != "" {
			fmt.Fprintf(os.Stderr, "%s: duplicate message exists (ID: %s)\n", yellow("⚠"), existingID[:8])
			fmt.Fprintf(os.Stderr, "  Use --force to send anyway, or use a different title\n")
			os.Exit(1)
		}
	}

	// ALWAYS save to SQLite first
	if err := store.InsertInboxMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message sent to '%s' (ID: %s)\n", green("✓"), inbox, msg.MessageID)

	// Dual-write: publish notification to Pub/Sub if enabled (M-PUBSUB)
	// This is non-fatal — message is already safely in SQLite/Firestore.
	cfg, _ := messaging.LoadConfig()
	if cfg != nil && cfg.PubSub != nil && cfg.PubSub.Enabled {
		notifier, notifyErr := messaging.NewPubSubNotifier(cfg.PubSub)
		if notifyErr != nil {
			fmt.Fprintf(os.Stderr, "%s Pub/Sub notify failed: %v\n", yellow("!"), notifyErr)
		} else if notifier != nil {
			defer notifier.Close()
			if notifyErr := notifier.Notify(context.Background(), msg); notifyErr != nil {
				fmt.Fprintf(os.Stderr, "%s Pub/Sub notify failed: %v\n", yellow("!"), notifyErr)
			} else {
				fmt.Printf("%s Pub/Sub notification published\n", green("✓"))
			}
		}
	}

	// Compute envelope (M-SEMANTIC-ENVELOPE)
	// Auto-detects git context unless --no-envelope is set
	if !*noEnvelope {
		// Resolve code files: explicit flag, "auto", or auto-detect from git
		codeFiles := resolveEnvelopeCodeFiles(*envelopeCode)

		if len(codeFiles) > 0 || *envelopeContext != "" {
			cfg := messaging.LoadEmbedConfigFromEnv()
			embedder, embErr := messaging.NewEmbedderFromConfig(cfg)
			if embErr != nil || embedder == nil {
				// Silent: no embedder configured is fine for most users
			} else {
				builder := messaging.NewEnvelopeBuilder(embedder)
				if len(codeFiles) > 0 {
					builder = builder.WithCodeContext(codeFiles, nil)
				}
				if *envelopeContext != "" {
					builder = builder.WithSessionContext(nil, nil, []string{*envelopeContext})
				}
				env, buildErr := builder.Build(msg)
				if buildErr != nil {
					fmt.Fprintf(os.Stderr, "%s Envelope computation failed: %v\n", yellow("!"), buildErr)
				} else if !env.IsEmpty() {
					if updateErr := store.UpdateMessageEnvelope(msg.MessageID, env, false); updateErr != nil {
						fmt.Fprintf(os.Stderr, "%s Envelope save failed: %v\n", yellow("!"), updateErr)
					} else {
						slotCount := 0
						for _, v := range env.Slots {
							if v != nil && len(v.Vector) > 0 {
								slotCount++
							}
						}
						source := "auto"
						if *envelopeCode != "" {
							source = "explicit"
						}
						fmt.Printf("%s Envelope computed (%d slots, code: %s)\n", green("✓"), slotCount, source)
					}
				}
			}
		}
	}

	// Optionally sync to GitHub
	if syncToGitHub {
		issueNum, err := syncMessageToGitHub(msg, *repo, *githubUser)
		if err != nil {
			// Check for account mismatch - show prominently in red
			if errors.Is(err, messaging.ErrAccountMismatch) {
				fmt.Fprintf(os.Stderr, "\n%s %v\n\n", red("ERROR:"), err)
				fmt.Println("  Message saved locally but GitHub sync BLOCKED.")
			} else {
				// Other errors get a warning
				fmt.Fprintf(os.Stderr, "%s GitHub sync failed: %v\n", yellow("⚠"), err)
				fmt.Println("  Message saved locally. Retry GitHub sync with:")
				fmt.Printf("  ailang messages github-sync %s\n", msg.MessageID)
			}
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
	githubUser := fs.String("github-user", "", "Override expected GitHub user (bypass config.expected_user)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "%s: reply requires MSG_ID and reply text\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Usage: ailang messages reply MSG_ID \"Your reply\" [--from agent]")
		os.Exit(1)
	}

	msgIDArg := fs.Arg(0)
	replyText := fs.Arg(1)

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID, err := resolveMessageID(store, msgIDArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Look up the original message
	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message %q not found\n", red("Error"), msgIDArg)
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

	// Set override user if provided
	if *githubUser != "" {
		ghClient.SetOverrideUser(*githubUser)
	}

	// Determine the repo: CLI flag > message repo > inbox mapping > config default
	targetRepo := msg.GitHubRepo
	if *repo != "" {
		targetRepo = *repo
	}
	if targetRepo == "" {
		// Try inbox-specific mapping, then default
		cfg := ghClient.GetConfig()
		if cfg != nil {
			targetRepo = cfg.RepoForInbox(msg.ToInbox)
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

// resolveEnvelopeCodeFiles determines which files to use for the code envelope slot.
// Priority: explicit --envelope-code flag > auto-detect from git.
// Returns nil if no files found (envelope will only have intent slot).
func resolveEnvelopeCodeFiles(flagValue string) []string {
	if flagValue != "" && flagValue != "auto" {
		// Explicit file list
		return strings.Split(flagValue, ",")
	}

	// Auto-detect from git context
	files := detectGitModifiedFiles()
	if len(files) == 0 {
		return nil
	}

	// Cap at 10 files to keep embedding input reasonable
	if len(files) > 10 {
		files = files[:10]
	}

	return files
}

// detectGitModifiedFiles returns files with uncommitted changes (staged + unstaged).
// Falls back to files from the most recent commit if working tree is clean.
// Only includes source files (Go, AILANG, etc.), not generated/binary files.
func detectGitModifiedFiles() []string {
	// Try uncommitted changes first (most relevant to current work)
	out, err := exec.Command("git", "diff", "--name-only", "HEAD").Output()
	if err != nil {
		return nil
	}

	files := filterSourceFiles(strings.TrimSpace(string(out)))
	if len(files) > 0 {
		return files
	}

	// Working tree is clean — try staged files
	out, err = exec.Command("git", "diff", "--staged", "--name-only").Output()
	if err != nil {
		return nil
	}

	files = filterSourceFiles(strings.TrimSpace(string(out)))
	if len(files) > 0 {
		return files
	}

	// Nothing staged — try last commit
	out, err = exec.Command("git", "diff", "--name-only", "HEAD~1", "HEAD").Output()
	if err != nil {
		return nil
	}

	return filterSourceFiles(strings.TrimSpace(string(out)))
}

// filterSourceFiles keeps only source code files from a newline-separated list.
func filterSourceFiles(output string) []string {
	if output == "" {
		return nil
	}

	var result []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Include source code files, exclude generated/binary/config
		if isSourceFile(line) {
			result = append(result, line)
		}
	}
	return result
}

// appendBinaryInfo appends ailang binary version and MD5 hash to a bug report payload.
// This ensures every bug report includes the exact binary used, preventing
// "works on my machine" debugging sessions.
func appendBinaryInfo(payload string) string {
	var info []string

	// Get version
	info = append(info, fmt.Sprintf("ailang version: %s", Version))

	// Get binary MD5
	exe, err := os.Executable()
	if err == nil {
		md5out, err := exec.Command("md5", "-q", exe).Output()
		if err != nil {
			// Try md5sum (Linux)
			md5out, err = exec.Command("md5sum", exe).Output()
		}
		if err == nil {
			hash := strings.TrimSpace(string(md5out))
			// md5sum outputs "hash  filename", take just the hash
			if parts := strings.Fields(hash); len(parts) > 0 {
				hash = parts[0]
			}
			info = append(info, fmt.Sprintf("binary md5: %s", hash))
			info = append(info, fmt.Sprintf("binary path: %s", exe))
		}
	}

	// Get git commit if available
	if Commit != "" {
		info = append(info, fmt.Sprintf("git commit: %s", Commit))
	}

	if len(info) > 0 {
		return payload + "\n\n---\nBinary info (auto-attached):\n" + strings.Join(info, "\n")
	}
	return payload
}

// isSourceFile returns true for files that are meaningful for code context embedding.
func isSourceFile(path string) bool {
	// Include common source extensions
	sourceExts := []string{
		".go", ".ail", ".py", ".js", ".ts", ".tsx",
		".rs", ".java", ".c", ".cpp", ".h",
		".sh", ".bash",
	}
	for _, ext := range sourceExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
