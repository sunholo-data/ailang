package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/storage"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
	"github.com/sunholo-data/ailang/internal/telemetry"
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
// The store is selected by messagesTarget: the canonical cloud store when
// AILANG_MESSAGES_STORE=gcp, otherwise this machine's local collaboration.db.
// Both are also readable by the Collaboration Hub dashboard.
func messagesCommand() {
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-messages")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
	} else {
		defer shutdownTelemetry(ctx)
	}

	if len(os.Args) < 3 {
		// Check if stdin is a terminal (interactive)
		if isTerminal() {
			runMessagesInteractive()
		} else {
			runMessagesList([]string{})
		}
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "list", "ls":
		runMessagesList(args)
	case "search":
		runMessagesSearch(args)
	case "dedupe":
		runMessagesDedupe(args)
	case "ack":
		runMessagesAck(args)
	case "unack":
		runMessagesUnack(args)
	case "send":
		runMessagesSend(args)
	case "read":
		runMessagesRead(args)
	case "forward", "fwd":
		runMessagesForward(args)
	case "watch":
		runMessagesWatch(args)
	case "cleanup":
		runMessagesCleanup(args)
	case "import-github":
		runMessagesImportGitHub(args)
	case "reply":
		runMessagesReply(args)
	case "health":
		runMessagesHealth(args)
	case "activity":
		runMessagesActivity(args)
	case "triage":
		runMessagesTriage(args)
	case "--help", "-h", "help":
		printMessagesHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subCmd)
		printMessagesHelp()
		os.Exit(1)
	}
}

// messagesTarget resolves WHICH message store the inbox commands talk to, and in
// which project, WITHOUT moving this process's coordinator or observatory backends.
//
// AILANG_STORAGE is a process-wide switch over all three backends, so a machine that
// wants its inbox on the shared cloud store but its eval banking and coordinator state
// local could not express that with AILANG_STORAGE alone — the only way to reach the
// canonical inbox was to move everything, which is why the docs said "never export it".
// AILANG_MESSAGES_STORE is the scoped selector for messaging, the same shape
// AILANG_CHAINS_READ already provides for the observatory (see openChainsReadBackend).
//
// Resolution order (first non-empty wins):
//
//	mode:    AILANG_MESSAGES_STORE > AILANG_STORAGE > "local"
//	project: AILANG_MESSAGES_PROJECT > AILANG_CLOUD_PROJECT
//
// Only "gcp" reaches Firestore; "hybrid" keeps messaging in SQLite, matching
// storage.NewHybridBackends.
func messagesTarget() (storage.Mode, string) {
	mode := os.Getenv("AILANG_MESSAGES_STORE")
	if mode == "" {
		mode = os.Getenv("AILANG_STORAGE")
	}
	project := os.Getenv("AILANG_MESSAGES_PROJECT")
	if project == "" {
		project = os.Getenv("AILANG_CLOUD_PROJECT")
	}
	switch storage.Mode(mode) {
	case storage.ModeGCP, storage.ModeHybrid:
		return storage.Mode(mode), project
	case storage.ModeLocal, "":
		return storage.ModeLocal, project
	default:
		// Unknown value: refuse rather than silently reading the wrong store.
		// openStore turns this into an error naming the offending value.
		return storage.Mode(mode), project
	}
}

// openStore opens the message store selected by messagesTarget.
//
// Without this, `ailang messages list --inbox public-feedback` always returned
// "No messages found" because openStore unconditionally opened the local SQLite
// database — invisible to the cloud-side public-feedback inbox.
//
// In gcp mode this builds ONLY the messaging store, not the full Backends struct:
// storage.NewGCPBackends also constructs a coordinator store and starts its
// background cost-sync goroutine, which a one-shot `messages list` has no use for.
func openStore() (messaging.MessageStore, error) {
	mode, project := messagesTarget()

	switch mode {
	case storage.ModeLocal, storage.ModeHybrid:
		// Both keep messaging in SQLite.
		return messaging.OpenStore(messaging.GetDefaultDatabasePath())
	case storage.ModeGCP:
		if project == "" {
			return nil, fmt.Errorf("openStore: AILANG_MESSAGES_PROJECT or AILANG_CLOUD_PROJECT must be set for gcp message store")
		}
		client, err := fsstore.NewClientForProject(context.Background(), project)
		if err != nil {
			return nil, fmt.Errorf("openStore: %w", err)
		}
		return fsstore.NewMessagingStore(client), nil
	default:
		return nil, fmt.Errorf("openStore: unknown message store mode %q (valid: local, gcp, hybrid)", string(mode))
	}
}

// describeMessageStore returns a one-line description of the store being read, or
// "" for the local default. Printed above listings so a session can never mistake
// a stale dev graveyard for the canonical inbox — the failure that made prod
// feedback invisible was indistinguishable from an empty inbox.
func describeMessageStore() string {
	mode, project := messagesTarget()
	if mode == storage.ModeGCP {
		return fmt.Sprintf("store: %s (Firestore, project %s)", mode, project)
	}
	return ""
}
