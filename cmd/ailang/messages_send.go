package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/storage"
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

	// M-COORD-TAG-ROUTING-LASTMILE (v0.23.0): --requires N1,N2 routes the
	// message through the daemon's HTTP /api/messages endpoint, which carries
	// the tags to Pub/Sub attributes for tag-subset filtering by workers.
	// Requires the local daemon to be running with its HTTP listener bound
	// (the M1 plist+installer changes default PORT=8765). Without --requires,
	// `messages send` keeps its existing SQLite-only behavior, unchanged from
	// v0.22.0.
	requires := fs.String("requires", "", "Comma-separated worker tags this message requires (e.g., agent:motoko,ollama:gemma4). Routes via HTTP /api/messages → Pub/Sub.")

	// Normalize args: move flags before positional arguments
	// Go's flag package requires flags to come first, but users often put them at the end
	args, normErr := normalizeArgsForFlags(args, fs)
	if normErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), normErr)
		os.Exit(1)
	}

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

	// M-COORD-TAG-ROUTING-LASTMILE: --requires routes via HTTP /api/messages
	// so the daemon can attach Pub/Sub attributes for tag-subset filtering.
	// This short-circuits the SQLite-only path because the HTTP endpoint
	// stores the message + publishes the notification in one step.
	if strings.TrimSpace(*requires) != "" {
		tags := splitAndTrim(*requires, ",")
		if err := sendViaHTTP(inbox, msgTitle, payload, *from, category, *repo, tags); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Message sent to '%s' with requires=%v (via HTTP /api/messages)\n", green("✓"), inbox, tags)
		return
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
	// The config error is REPORTED, not discarded: an unreadable config is the
	// difference between "will be worked on" and "will sit forever", and the
	// caller cannot tell those apart from a bare success line.
	cfg, cfgErr := messaging.LoadConfig()
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "%s messaging config unreadable (%v) — cannot determine whether this message will be dispatched\n", yellow("!"), cfgErr)
	}
	notified := false
	if cfg != nil && cfg.PubSub != nil && cfg.PubSub.Enabled {
		notifier, notifyErr := messaging.NewPubSubNotifier(notifyConfigForStore(cfg.PubSub))
		if notifyErr != nil {
			fmt.Fprintf(os.Stderr, "%s Pub/Sub notify failed: %v\n", yellow("!"), notifyErr)
		} else if notifier != nil {
			defer notifier.Close()
			if notifyErr := notifier.Notify(context.Background(), msg); notifyErr != nil {
				fmt.Fprintf(os.Stderr, "%s Pub/Sub notify failed: %v\n", yellow("!"), notifyErr)
			} else {
				notified = true
				fmt.Printf("%s Pub/Sub notification published\n", green("✓"))
			}
		}
	}
	warnIfFiledButUndispatchable(inbox, notified)

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

// sendViaHTTP POSTs a tag-routed message to the local coordinator daemon's
// /api/messages endpoint. The daemon stores the message and publishes a
// Pub/Sub notification with the requires tags as attributes, enabling
// tag-subset filtering by workers (M-COORD-MULTI-HOST-WORKERS v0.22.0).
//
// Returns an error with a clear next-step hint if the daemon HTTP listener
// isn't reachable — common cause: the launchd plist was installed before
// M-COORD-TAG-ROUTING-LASTMILE shipped, so PORT isn't set. The fix is
// `make coord-install --port 8765`.
func sendViaHTTP(inbox, title, content, from, category, repo string, requires []string) error {
	port := discoverCoordinatorHTTPPort()
	if port == "" {
		return fmt.Errorf("--requires needs the daemon's HTTP listener but no PORT is configured.\n  Fix: make coord-install   (or set AILANG_COORD_HTTP_PORT)")
	}
	if !probeCoordinatorHTTP(port) {
		return fmt.Errorf("--requires needs the daemon's HTTP listener but http://127.0.0.1:%s/health is unreachable.\n  Check: ailang coordinator status, then `make coord-install` or `launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.ailang.coordinator.plist`", port)
	}

	body := map[string]interface{}{
		"inbox":   inbox,
		"title":   title,
		"content": content,
		"from":    from,
	}
	if category != "" {
		body["category"] = category
	}
	if repo != "" {
		body["github_repo"] = repo
	}
	if len(requires) > 0 {
		body["requires"] = requires
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding HTTP body: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%s/api/messages", port)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("COORDINATOR_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s returned %d: %s", url, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// splitAndTrim splits s by sep and trims whitespace from each element,
// dropping empty elements. Used for parsing comma-separated CLI flag values
// like --requires "agent:motoko, ollama:gemma4".
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
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

// normalizeArgsForFlags moves flags to the front of args so Go's flag package
// can parse them. Go stops parsing at the first non-flag argument, but users
// naturally write `send inbox message --title foo` rather than
// `send --title foo inbox message`.
//
// The flag set is read from fs itself rather than a hand-maintained list of
// names. The old signature took []string, which meant every caller repeated the
// names — and a list that must be kept in step with a FlagSet is a list that
// drifts. It also could not distinguish a boolean flag from one taking a value,
// which was the second half of the bug below.
//
// M-COORDINATOR-EXECUTION-TRUST M5 (design doc V29). The previous version
// decided whether the NEXT token was a value with `!strings.HasPrefix(next, "-")`.
// The intent was to avoid swallowing a following flag; the effect was to reject
// any legitimate value that begins with a dash, drop it into the positional
// list, and shift every later token. Measured 2026-09-02:
//
//	ailang messages send diag-argparse "body" \
//	  --title "--help is inconsistent" --from "diag-sender"
//
// delivered the message to inbox "diag-sender", set from_agent to the "cli"
// default, set the title to the literal string "--from" — and printed
// "✓ Message sent". A misrouted message that reports success is exactly the
// failure class this milestone exists to remove, one layer earlier than the rest.
//
// Two rules now:
//
//  1. Whether the next token is this flag's value is decided by the FLAG, not by
//     the token's first character. A value flag always takes the next token; a
//     boolean flag never does. Dash-leading values therefore survive, and a bool
//     flag no longer swallows the positional after it.
//  2. A value flag with nothing left to consume is an ERROR. Silently shifting is
//     how routing became a side effect of a parse that half-failed.
func normalizeArgsForFlags(args []string, fs *flag.FlagSet) ([]string, error) {
	if fs == nil {
		return args, nil
	}

	// takesValue[name] reports whether that flag consumes the following token.
	// Booleans are identified the way the flag package itself does it.
	takesValue := make(map[string]bool)
	fs.VisitAll(func(f *flag.Flag) {
		isBool := false
		if bv, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bv.IsBoolFlag() {
			isBool = true
		}
		takesValue[f.Name] = !isBool
	})

	// flagNameOf returns the bare flag name for "-x"/"--x", and whether it is one
	// this set knows.
	flagNameOf := func(arg string) (string, bool) {
		if !strings.HasPrefix(arg, "-") {
			return "", false
		}
		name := strings.TrimLeft(arg, "-")
		if name == "" {
			return "", false
		}
		_, known := takesValue[name]
		return name, known
	}

	var flags, positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// "--name=value" carries its own value; nothing to consume.
		if eq := strings.IndexByte(arg, '='); eq > 0 {
			if _, known := flagNameOf(arg[:eq]); known {
				flags = append(flags, arg)
				continue
			}
		}

		name, known := flagNameOf(arg)
		if !known {
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)
		if !takesValue[name] {
			continue // boolean: must not eat the next token
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("flag --%s needs a value", name)
		}
		i++
		flags = append(flags, args[i])
	}

	return append(flags, positional...), nil
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

// warnIfFiledButUndispatchable says so when a message has been FILED but will
// never be DISPATCHED.
//
// The cloud coordinator's intake is Pub/Sub ONLY: pollAndProcessTasksCloud reads
// the Pub/Sub adapter (internal/coordinator/daemon_tasks_polling.go) and never
// queries Firestore. So a message written to the cloud store whose notification
// did not publish is invisible to it — permanently, not slowly.
//
// Printing a bare "✓ Message sent" in that case is exactly the failure this
// guards against: the write genuinely succeeded, the caller reasonably believed
// work had been queued, and nothing ever ran. Measured 2026-08-31 — three
// pkg:sunholo/ailang_parse reports sat unread with no task and no job, because
// this machine's config carries no pubsub section at all.
//
// Local and hybrid stores are silent here on purpose: that daemon polls the
// store directly, so no notification is required for work to start.
func warnIfFiledButUndispatchable(inbox string, notified bool) {
	if notified {
		return
	}
	mode, project := messagesTarget()
	if mode != storage.ModeGCP {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s FILED, NOT DISPATCHED — no Pub/Sub notification was published.\n", yellow("!"))
	fmt.Fprintf(os.Stderr, "  The message IS in Firestore (project %s, inbox %q) and is readable with\n", project, inbox)
	fmt.Fprintf(os.Stderr, "    ailang messages list --inbox %q\n", inbox)
	fmt.Fprintf(os.Stderr, "  but the cloud coordinator takes work from Pub/Sub only, so nothing will pick it up.\n")
	fmt.Fprintf(os.Stderr, "  Fix — add to %s:\n", messaging.GetConfigPath())
	fmt.Fprintf(os.Stderr, "    pubsub:\n      enabled: true\n      project_id: %s\n", project)
}

// notifyConfigForStore makes the notification follow the store the message was
// actually written to.
//
// NewPubSubNotifier resolves its project from config.project_id, then
// AILANG_CLOUD_PROJECT, then GOOGLE_CLOUD_PROJECT — none of which know which
// store the write went to. So a message written to one project could be
// announced in another, reaching a coordinator that cannot see it. The write
// succeeds, the publish succeeds, both report ok, and the work is invisible to
// the only process that could do it.
//
// Measured 2026-08-31: a probe written to ailang-multivac-dev published its
// notification to ailang-multivac, because the pubsub block pinned project_id
// to prod. The dev coordinator was never told and the task never ran.
//
// Notifying a project you did not write to is never correct, so the store wins
// and a pinned mismatch is reported rather than honoured.
func notifyConfigForStore(pc *messaging.PubSubConfig) *messaging.PubSubConfig {
	mode, storeProject := messagesTarget()
	if mode != storage.ModeGCP || storeProject == "" || pc == nil {
		return pc
	}
	if pc.ProjectID != "" && pc.ProjectID != storeProject {
		fmt.Fprintf(os.Stderr, "%s pubsub.project_id is %q but this message was written to %q; notifying %q so the right coordinator hears it\n",
			yellow("!"), pc.ProjectID, storeProject, storeProject)
	}
	clone := *pc
	clone.ProjectID = storeProject
	// The topic prefix is per-environment infrastructure, not a user preference:
	// terraform sets AILANG_TOPIC_PREFIX = var.prefix, giving ailang / ailang-dev
	// / ailang-test alongside ailang-multivac{,-dev,-test}. Carrying prod's
	// prefix into a dev store publishes to a topic that does not exist there —
	// measured 2026-08-31, the probe failed on ailang-messages in a project whose
	// topic is ailang-dev-messages. Derive it with the project so the pair always
	// agrees; an unrecognised project keeps whatever was configured, and the
	// FILED, NOT DISPATCHED warning still fires if that turns out to be wrong.
	if derived, ok := topicPrefixForProject(storeProject); ok {
		clone.TopicPrefix = derived
	}
	return &clone
}

// topicPrefixForProject maps ailang-multivac{,-dev,-test} to the topic prefix
// terraform provisions for it. Returns false for anything it does not recognise,
// so an unknown project is never silently given a guessed prefix.
func topicPrefixForProject(project string) (string, bool) {
	switch project {
	case "ailang-multivac":
		return "ailang", true
	case "ailang-multivac-dev":
		return "ailang-dev", true
	case "ailang-multivac-test":
		return "ailang-test", true
	}
	return "", false
}
