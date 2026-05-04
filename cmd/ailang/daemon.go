package main

import (
	"context"
	"flag"
	"fmt"
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

	cfg, project, prefix, err := daemon.ConfigForEnv(*envFlag, fc)
	if err != nil {
		return err
	}

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

	d := daemon.New(
		cfg,
		pubsubAdapter{sub: pubsub.NewSubscriber(psClient)},
		storeFetcher{store: backends.Messaging},
		notify.Notify,
	)

	fmt.Printf("ailang daemon: env=%s project=%s events=%s messages=%s dry_run=%t\n",
		envOrDefault(*envFlag, fc.Env, "prod"), project, cfg.EventsSub, cfg.MessagesSub, cfg.DryRun)

	return d.Run(ctx)
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
