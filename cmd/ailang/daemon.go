package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sunholo-data/ailang/internal/daemon"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/notify"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/storage"
	firestore "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// daemonCommand is the entry point for `ailang daemon ...`. Subcommands:
//
//	run        — foreground mode (default; used by launchd ProgramArguments)
//	install    — install launchd plist (M3)
//	uninstall  — remove launchd plist (M3)
//	status     — show launchctl status + recent log lines (M3)
func daemonCommand() {
	args := flag.Args()[1:]
	if len(args) == 0 {
		args = []string{"run"}
	}
	switch args[0] {
	case "run":
		if err := daemonRun(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
	case "install":
		if err := daemonInstall(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
	case "uninstall":
		if err := daemon.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Println("ailang daemon: uninstalled")
	case "status":
		out, err := daemon.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Println(out)
		// Show last 10 log lines if the log exists.
		if data, err := os.ReadFile("/tmp/ailang-daemon.log"); err == nil {
			lines := splitLastN(string(data), 10)
			if len(lines) > 0 {
				fmt.Println("\nrecent log:")
				for _, l := range lines {
					fmt.Println("  " + l)
				}
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown daemon subcommand '%s' (want run|install|uninstall|status)\n", red("Error"), args[0])
		os.Exit(1)
	}
}

func daemonRun(args []string) error {
	fs := flag.NewFlagSet("daemon run", flag.ExitOnError)
	envFlag := fs.String("env", "", "Cloud environment to subscribe to (dev|test|prod). Default: from daemon.yaml or 'prod'.")
	dryRun := fs.Bool("dry-run", false, "Log notifications instead of firing them. Useful for tests.")
	var alsoSubscribe multiFlag
	fs.Var(&alsoSubscribe, "also-subscribe", "ADDITIONAL cloud env whose inbox messages to also watch (dev|test|prod). Repeatable. Appends to daemon.yaml extra_message_envs. Example: --env dev --also-subscribe prod.")
	extraMessagesSub := fs.String("extra-messages-sub", "", "Base subscription name for the EXTRA message sources (default messages-laptop). Give each device its own (e.g. messages-rig) — shared subscriptions work-steal, so two daemons on one sub each see only some messages. Overrides daemon.yaml extra_messages_sub. Does NOT rename the PRIMARY subscription — use --messages-sub for that.")
	// --messages-sub names the PRIMARY subscription, which nothing could do
	// before. Measured on the rig 2026-09-11: the plist passed
	// `--extra-messages-sub messages-rig` intending "this device pulls
	// messages-rig", but that flag only names the subscription for EXTRA envs,
	// and there were none — so the daemon reported `extra_message_sources=[]`
	// and quietly pulled `messages-laptop` instead. Two daemons then shared one
	// subscription, which work-steals: each sees only some messages, and
	// messages appear to vanish. The operator's intent was not expressible.
	messagesSub := fs.String("messages-sub", "", "Subscription for the PRIMARY env's inbox messages (default messages-laptop). Give each device its own — two daemons on one subscription work-steal and each sees only part of the traffic.")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fc, err := daemon.LoadFileConfig()
	if err != nil {
		return fmt.Errorf("load daemon config: %w", err)
	}
	if *dryRun {
		fc.DryRun = true
	}
	// CLI --also-subscribe appends to the yaml extra_message_envs.
	fc.ExtraMessageEnvs = append(fc.ExtraMessageEnvs, alsoSubscribe...)
	if *extraMessagesSub != "" {
		fc.ExtraMessagesSub = *extraMessagesSub
	}

	cfg, project, prefix, err := daemon.ConfigForEnv(*envFlag, fc)
	if err != nil {
		return err
	}
	if *messagesSub != "" {
		cfg.MessagesSub = *messagesSub
	}

	if err := validateSubscriptionFlags(*extraMessagesSub, fc.ExtraMessageEnvs); err != nil {
		return err
	}
	primaryEnv := envOrDefault(*envFlag, fc.Env, "prod")

	ctx, cancel := signalContext()
	defer cancel()

	psClient, err := pubsub.NewClient(ctx, project, prefix)
	if err != nil {
		return fmt.Errorf("pubsub client: %w", err)
	}
	defer func() { _ = psClient.Close() }()

	// Force GCP storage so we read the cloud-side InboxMessage docs (not local SQLite).
	if os.Getenv("AILANG_STORAGE") == "" {
		_ = os.Setenv("AILANG_STORAGE", "gcp")
	}
	if os.Getenv("AILANG_CLOUD_PROJECT") == "" {
		_ = os.Setenv("AILANG_CLOUD_PROJECT", project)
	}
	backends, err := storage.NewBackends(ctx)
	if err != nil {
		return fmt.Errorf("storage backends: %w", err)
	}

	// Build the channel registry: macOS desktop (local best-effort) plus any
	// env-gated remote channels (Discord if AILANG_DISCORD_WEBHOOK_URL is set).
	// The daemon fans out over all of them; remote channels are authoritative
	// for ack, the local one is best-effort. With no remote channel configured,
	// this degrades to today's macOS-only behaviour.
	reg := notify.NewRegistry()
	_ = reg.Register(notify.MacOSChannel{})
	notify.RegisterChannels(reg, log.Default())

	d := daemon.New(
		cfg,
		pubsubAdapter{sub: pubsub.NewSubscriber(psClient)},
		storeFetcher{store: backends.Messaging},
		reg.FanOut(log.Default()),
	)

	// Additional inbox-message sources (e.g. prod), each scoped to its OWN
	// project's Firestore WITHOUT mutating the shared AILANG_CLOUD_PROJECT env
	// (env mutation would make the dev/prod fetchers collide). We build a
	// project-explicit Firestore messaging store per extra source.
	extras, err := daemon.ResolveExtraMessageSourcesWithSub(primaryEnv, fc.ExtraMessageEnvs, fc.ExtraMessagesSub)
	if err != nil {
		return err
	}
	var extraClosers []func()
	defer func() {
		for _, c := range extraClosers {
			c()
		}
	}()
	for _, ex := range extras {
		exPS, err := pubsub.NewClient(ctx, ex.Project, ex.Prefix)
		if err != nil {
			return fmt.Errorf("pubsub client (%s): %w", ex.Env, err)
		}
		extraClosers = append(extraClosers, func() { _ = exPS.Close() })

		fsClient, err := firestore.NewClientForProject(ctx, ex.Project)
		if err != nil {
			return fmt.Errorf("firestore client (%s): %w", ex.Env, err)
		}
		exStore := firestore.NewMessagingStore(fsClient)
		extraClosers = append(extraClosers, func() { _ = exStore.Close() })

		d.AddMessageSource(daemon.MessageSource{
			Sub:     pubsubAdapter{sub: pubsub.NewSubscriber(exPS)},
			Fetcher: storeFetcher{store: exStore},
			SubName: ex.MessagesSub,
			Label:   ex.Env,
		})
	}

	extraLabels := make([]string, 0, len(extras))
	for _, ex := range extras {
		extraLabels = append(extraLabels, ex.Env+"("+ex.Project+")")
	}
	fmt.Printf("ailang daemon: env=%s project=%s events=%s messages=%s extra_message_sources=%v dry_run=%t channels=%v\n",
		primaryEnv, project, cfg.EventsSub, cfg.MessagesSub, extraLabels, cfg.DryRun,
		reg.Names())

	return d.Run(ctx)
}

// multiFlag collects a repeatable string flag (e.g. --also-subscribe prod
// --also-subscribe test).
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func daemonInstall(args []string) error {
	fs := flag.NewFlagSet("daemon install", flag.ExitOnError)
	envFlag := fs.String("env", "prod", "Cloud environment to subscribe to (dev|test|prod).")
	binPath := fs.String("binary", "", "Absolute path to the ailang binary. Default: result of `which ailang`.")
	force := fs.Bool("force", false, "Overwrite existing plist.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	bin := *binPath
	if bin == "" {
		resolved, err := exec.LookPath("ailang")
		if err != nil {
			return fmt.Errorf("could not locate ailang binary: %w (pass --binary)", err)
		}
		bin = resolved
	}
	if err := daemon.Install(daemon.InstallOpts{Env: *envFlag, BinaryPath: bin, Force: *force}); err != nil {
		return err
	}
	fmt.Printf("ailang daemon: installed (env=%s, binary=%s)\n", *envFlag, bin)
	fmt.Println("            log: /tmp/ailang-daemon.log")
	fmt.Println("           stop: ailang daemon uninstall")
	return nil
}

func splitLastN(s string, n int) []string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// signalContext returns a context cancelled on SIGINT/SIGTERM so the daemon
// can shut down cleanly under launchd's stop signal.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}

func envOrDefault(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ── Adapters ──────────────────────────────────────────────────────────────

// pubsubAdapter wraps *pubsub.Subscriber so the daemon can be unit-tested
// against an in-memory fake without depending on the gpubsub package.
type pubsubAdapter struct {
	sub *pubsub.Subscriber
}

func (a pubsubAdapter) Subscribe(ctx context.Context, subName string, handler daemon.MessageHandler) error {
	return a.sub.Subscribe(ctx, subName, pubsub.MessageHandler(handler))
}

// storeFetcher resolves a Pub/Sub MessageNotification to the full InboxMessage
// via the storage backend (Firestore in cloud mode).
type storeFetcher struct {
	store messaging.MessageStore
}

func (f storeFetcher) Fetch(_ context.Context, messageID string) (*messaging.InboxMessage, error) {
	return f.store.GetInboxMessage(messageID)
}

// validateSubscriptionFlags refuses a subscription name that would be silently
// ignored.
//
// --extra-messages-sub renames the subscription for EXTRA message sources only.
// Given without any extra envs it names nothing, and the daemon starts happily
// on a different subscription than the one the operator asked for.
//
// Measured on the rig 2026-09-11: the plist passed
// `--extra-messages-sub messages-rig`, meaning "this device pulls messages-rig".
// There were no extra envs, so the daemon printed `extra_message_sources=[]` and
// pulled `messages-laptop` — the same subscription as another machine. Pub/Sub
// work-steals across consumers of one subscription, so each daemon saw only part
// of the traffic and messages appeared to vanish. Nothing was broken enough to
// report itself; the flag was simply accepted and ignored.
func validateSubscriptionFlags(extraMessagesSub string, extraEnvs []string) error {
	if extraMessagesSub == "" || len(extraEnvs) > 0 {
		return nil
	}
	return fmt.Errorf(
		"--extra-messages-sub=%q was given but no extra message envs are configured, so it names nothing and NOTHING would subscribe to it.\n"+
			"  It renames the subscription for --also-subscribe / extra_message_envs sources only.\n"+
			"  To make THIS daemon pull %q, use: --messages-sub %s",
		extraMessagesSub, extraMessagesSub, extraMessagesSub)
}
