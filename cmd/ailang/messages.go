package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/messaging"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"github.com/sunholo-data/ailang/internal/storage"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
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
// The store is selected by messagesTarget: the messaging store of the ONE
// storage plane (AILANG_STORAGE, overridable for this store alone with
// AILANG_STORAGE_MESSAGING=gcp), so the canonical cloud inbox or this
// machine's local collaboration.db. Both are also readable by the
// Collaboration Hub dashboard.
func messagesCommand() {
	// The plane must resolve before any subcommand touches a store: a retired
	// selector (AILANG_MESSAGES_STORE) or an unknown value is a hard error
	// here, not a silent local read.
	if _, err := resolveMessagesTarget(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", red("Error:"), err)
		os.Exit(2)
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := otelplatform.Init(ctx, "ailang-messages")
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
	case "github-sync":
		runMessagesGitHubSync(args)
	case "reply":
		runMessagesReply(args)
	case "health":
		runMessagesHealth(args)
	case "inboxes":
		if err := messagesInboxesCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "%s %v\n", red("Error:"), err)
			os.Exit(1)
		}
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
// The messaging store follows the ONE plane switch: AILANG_STORAGE for every
// store, or AILANG_STORAGE_MESSAGING=local|gcp for this one alone — which is
// how a machine keeps its eval banking and coordinator state local while its
// inbox is the shared cloud store (M-V1-SIMPLIFY-S3 M3; the scoped
// AILANG_MESSAGES_STORE this replaced is a hard error naming it).
//
// Resolution (first non-empty wins):
//
//	store:   AILANG_STORAGE_MESSAGING > AILANG_STORAGE > local
//	project: AILANG_MESSAGES_PROJECT > config.CloudProject
//
// Only gcp reaches Firestore; hybrid keeps messaging in SQLite, matching
// storage.NewHybridBackends. The plane value is returned (not the per-store
// mode) so callers that print "hybrid" keep doing so.
//
// It cannot fail after messagesCommand's up-front check; any other entry
// point that reaches it with an unresolvable plane exits with the error
// rather than reading a store the operator did not name.
func messagesTarget() (storage.Mode, string) {
	t, err := resolveMessagesTarget()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", red("Error:"), err)
		os.Exit(2)
	}
	return t.mode, t.project
}

// messagesTargetResolution is what resolveMessagesTarget returns.
type messagesTargetResolution struct {
	mode    storage.Mode // the plane value the messaging store follows
	store   config.StoreSelection
	project string
}

func resolveMessagesTarget() (messagesTargetResolution, error) {
	sel, err := config.StoragePlane()
	if err != nil {
		return messagesTargetResolution{}, err
	}
	project := os.Getenv("AILANG_MESSAGES_PROJECT")
	if project == "" && sel.Messaging.Mode == config.StoreGCP {
		// The messaging pin wins; otherwise the one cloud-project resolver.
		// An unresolvable project is "" here and openStore names what to set.
		project, _ = config.CloudProject(context.Background())
	}
	mode := sel.Plane
	if sel.Messaging.Source != sel.PlaneSource {
		// A per-store override: report the store's own mode, not the plane's.
		mode = storage.Mode(sel.Messaging.Mode)
	}
	return messagesTargetResolution{mode: mode, store: sel.Messaging, project: project}, nil
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
	t, err := resolveMessagesTarget()
	if err != nil {
		return nil, fmt.Errorf("openStore: %w", err)
	}
	switch t.store.Mode {
	case config.StoreGCP:
		if t.project == "" {
			return nil, fmt.Errorf("openStore: AILANG_MESSAGES_PROJECT or AILANG_CLOUD_PROJECT must be set for the gcp message store (%s via %s)", t.store.Mode, t.store.Source)
		}
		client, err := fsstore.NewClientForProject(context.Background(), t.project)
		if err != nil {
			return nil, fmt.Errorf("openStore: %w", err)
		}
		return fsstore.NewMessagingStore(client), nil
	default:
		// local and hybrid both keep messaging in SQLite.
		return messaging.OpenStore(messaging.GetDefaultDatabasePath())
	}
}

// describeMessageStore returns a one-line description of the store being read, or
// "" for the local default. Printed above listings so a session can never mistake
// a stale dev graveyard for the canonical inbox — the failure that made prod
// feedback invisible was indistinguishable from an empty inbox.
func describeMessageStore() string {
	t, err := resolveMessagesTarget()
	if err != nil {
		return ""
	}
	if t.store.Mode == config.StoreGCP {
		return fmt.Sprintf("store: %s (Firestore, project %s, via %s)", t.store.Mode, t.project, t.store.Source)
	}
	return ""
}
