package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/browser/auth"
)

// browserProfileEnv resolves the on-disk locations the profile commands use.
// AILANG_BROWSER_PROFILE_DIR overrides the root so tests and alternate
// deployments do not touch the operator's real profiles.
type browserProfileEnv struct {
	registry  *auth.FileRegistry
	broker    *auth.Broker
	auditPath string
	root      string
}

func browserProfileRoot() (string, error) {
	if override := os.Getenv("AILANG_BROWSER_PROFILE_DIR"); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".ailang", "browser-profiles"), nil
}

func openBrowserProfileEnv() (*browserProfileEnv, error) {
	root, err := browserProfileRoot()
	if err != nil {
		return nil, err
	}
	registry, err := auth.NewFileRegistry(filepath.Join(root, "profiles"))
	if err != nil {
		return nil, err
	}
	// The local file protector is development/single-host only; a KMS-backed
	// protector satisfies the same interface for shared deployments. Which one a
	// deployment uses is a human decision at deployment review, so this default
	// is deliberately the loud, local one.
	protector, err := auth.NewLocalFileKeyProtector(filepath.Join(root, "keys", "local.key"))
	if err != nil {
		return nil, err
	}
	auditPath := filepath.Join(root, "audit.jsonl")
	sink, err := auth.NewFileAuditSink(auditPath)
	if err != nil {
		return nil, err
	}
	broker, err := auth.NewBroker(auth.BrokerOptions{
		Registry:    registry,
		Leases:      auth.NewLeaseManager(auth.DefaultLeaseTTL),
		Protector:   protector,
		Audit:       sink,
		SessionRoot: filepath.Join(root, "materializations"),
	})
	if err != nil {
		return nil, err
	}
	return &browserProfileEnv{registry: registry, broker: broker, auditPath: auditPath, root: root}, nil
}

func runBrowserProfile(args []string) {
	if len(args) == 0 {
		printBrowserProfileHelp()
		os.Exit(1)
	}
	sub, rest := args[0], args[1:]

	var err error
	switch sub {
	case "bootstrap":
		err = browserProfileBootstrap(rest)
	case "inspect", "list":
		err = browserProfileInspect(rest)
	case "revoke":
		err = browserProfileRevoke(rest)
	case "purge":
		err = browserProfilePurge(rest)
	case "audit":
		err = browserProfileAudit(rest)
	case "gc":
		err = browserProfileGC(rest)
	case "refresh":
		err = browserProfileRefresh(rest)
	case "help", "-h", "--help":
		printBrowserProfileHelp()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown browser-profile subcommand %q\n\n", sub)
		printBrowserProfileHelp()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printBrowserProfileHelp() {
	fmt.Println(`ailang browser-profile — manage persistent authenticated browser identities

The AI model never receives a password, a canonical profile, or the authority to
change one. These commands are the trusted control plane that sits outside the
model/MCP loop.

SUBCOMMANDS
  bootstrap <alias>          Publish the first version from an operator-captured state
  inspect [alias]            Show safe metadata (never material)
  revoke <alias@version>     Mark a version permanently unusable
  purge <alias@version>      Delete a REVOKED version's material from disk
  audit [--limit N]          Show the profile audit trail
  gc [--max-age 1h]          Sweep orphaned decrypted materializations
  refresh <alias@version>    Publish a new version from a fresh login

CAPTURING A STATE FOR BOOTSTRAP
  Run a headful browser YOURSELF, sign in to a dedicated low-privilege
  automation account, and save the storage state. Your password stays between
  you and your password manager — AILANG never sees it:

    npx -y @playwright/mcp@0.0.79 --isolated --save-storage ./crm-state.json

  Recording must be off, and this session is not an eval.

EXAMPLE
  ailang browser-profile bootstrap crm-readonly-eu \
    --state-file ./crm-state.json \
    --provider local-playwright \
    --origins https://crm.example.com \
    --account-class readonly \
    --egress-ack

  ailang eval-suite --agent --benchmarks crm_readonly_fixture \
    --browser-provider local-playwright \
    --browser-profile crm-readonly-eu@latest

ENVIRONMENT
  AILANG_BROWSER_PROFILE_DIR   Override the profile root (default ~/.ailang/browser-profiles)`)
}

func browserProfileBootstrap(args []string) error {
	alias, args := splitPositional(args)
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	stateFile := fs.String("state-file", "", "Path to the operator-captured Playwright storage-state JSON (required)")
	provider := fs.String("provider", "local-playwright", "Browser provider: local-playwright or browserbase")
	contextID := fs.String("context-id", "", "Hosted provider context reference (browserbase only, instead of --state-file)")
	origins := fs.String("origins", "", "Comma-separated exact origins this profile may visit (required)")
	accountClass := fs.String("account-class", "readonly", "readonly, mutable, or privileged")
	version := fs.String("version", "v1", "Version to publish")
	maxConcurrent := fs.Int("max-concurrent", 1, "Maximum concurrent read leases")
	allowArtifacts := fs.String("allow-artifacts", "", "Comma-separated artifact classes to permit (default: none)")
	allowTakeover := fs.Bool("allow-human-takeover", false, "Permit human takeover of sessions using this profile")
	egressAck := fs.Bool("egress-ack", false, "Acknowledge that no egress boundary is enforced yet (required to run)")
	expires := fs.String("expires", "", "RFC3339 expiry, after which the profile fails closed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if alias == "" {
		return fmt.Errorf("bootstrap requires an alias, e.g. `ailang browser-profile bootstrap crm-readonly-eu ...`")
	}

	if *origins == "" {
		return fmt.Errorf("--origins is required: a profile with no allowed origins can never navigate")
	}
	if !*egressAck {
		return fmt.Errorf("--egress-ack is required.\n" +
			"  M-BROWSER-EGRESS-BOUNDARY has not shipped, so destination policy is NOT enforced below the browser.\n" +
			"  Passing this flag records an explicit, audited operator decision to proceed anyway.")
	}

	policy := auth.AuthProfilePolicy{
		AllowedOrigins:     splitList(*origins),
		AccountClass:       auth.AccountClass(*accountClass),
		MaxConcurrent:      *maxConcurrent,
		AllowArtifacts:     splitList(*allowArtifacts),
		AllowHumanTakeover: *allowTakeover,
		EgressBoundary:     auth.EgressOperatorAcknowledged,
	}
	// A nil artifact list means "no decision"; bootstrap always makes one.
	if policy.AllowArtifacts == nil {
		policy.AllowArtifacts = []string{}
	}
	if *expires != "" {
		parsed, err := time.Parse(time.RFC3339, *expires)
		if err != nil {
			return fmt.Errorf("--expires must be RFC3339: %w", err)
		}
		policy.ExpiresAt = parsed
	}

	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	ctx := context.Background()
	identity := auth.RunIdentity{RunID: "bootstrap-" + alias, Principal: operatorPrincipal()}

	var published auth.SafeProfile
	switch {
	case *contextID != "":
		published, err = env.registry.Publish(ctx, auth.Record{
			Alias: alias, Version: *version, Provider: *provider,
			Policy: policy, Material: auth.NewProviderContextMaterial(*contextID),
		})
	case *stateFile != "":
		state, readErr := os.ReadFile(*stateFile)
		if readErr != nil {
			return fmt.Errorf("read state file: %w", readErr)
		}
		published, err = env.broker.Bootstrap(ctx, auth.BootstrapRequest{
			Alias: alias, Version: *version, Provider: *provider,
			Policy: policy, State: state, Run: identity, CapturedAt: time.Now(),
		})
	default:
		return fmt.Errorf("bootstrap needs --state-file (local) or --context-id (hosted)")
	}
	if err != nil {
		return err
	}

	fmt.Printf("Published %s@%s\n", published.Alias, published.Version)
	fmt.Printf("  profile hash : %s\n", published.ProfileHash)
	fmt.Printf("  provider     : %s\n", published.Provider)
	fmt.Printf("  origins      : %s\n", strings.Join(published.Policy.AllowedOrigins, ", "))
	fmt.Printf("  account class: %s\n", published.Policy.AccountClass)
	fmt.Printf("  artifacts    : %s\n", artifactSummary(published.Policy))
	fmt.Printf("  egress       : %s (NOT enforced)\n", published.Policy.EgressBoundary)
	if *stateFile != "" {
		fmt.Printf("\nThe captured state is now sealed in the registry. Delete your copy:\n  rm %s\n", *stateFile)
	}
	return nil
}

func browserProfileInspect(args []string) error {
	only, args := splitPositional(args)
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	ctx := context.Background()

	aliases := []string{}
	if only != "" {
		aliases = append(aliases, only)
	} else {
		aliases, err = env.registry.Aliases()
		if err != nil {
			return err
		}
	}
	if len(aliases) == 0 {
		fmt.Printf("No browser auth profiles in %s\n", env.registry.Root())
		return nil
	}

	all := map[string][]auth.SafeProfile{}
	for _, alias := range aliases {
		profiles, err := env.registry.List(ctx, alias)
		if err != nil {
			return err
		}
		all[alias] = profiles
	}

	if *asJSON {
		// SafeProfile carries no material by construction, so this is safe to
		// pipe anywhere.
		return json.NewEncoder(os.Stdout).Encode(all)
	}

	for _, alias := range aliases {
		fmt.Printf("%s\n", alias)
		for _, profile := range all[alias] {
			state := "live"
			switch {
			case profile.Revoked():
				state = "REVOKED: " + profile.RevocationReason
			case profile.Retired():
				state = "retired"
			case profile.Expired(time.Now()):
				state = "EXPIRED"
			}
			fmt.Printf("  %-8s seq=%-3d %-10s %s  %s\n",
				profile.Version, profile.Sequence, profile.Provider, profile.ProfileHash, state)
			fmt.Printf("           origins=%s class=%s max-concurrent=%d artifacts=%s\n",
				strings.Join(profile.Policy.AllowedOrigins, ","), profile.Policy.AccountClass,
				profile.Policy.MaxConcurrent, artifactSummary(profile.Policy))
		}
	}
	return nil
}

func browserProfileRevoke(args []string) error {
	target, args := splitPositional(args)
	fs := flag.NewFlagSet("revoke", flag.ExitOnError)
	reason := fs.String("reason", "", "Why the version is being revoked (recorded in the audit trail)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("revoke requires alias@version")
	}
	ref, err := auth.ParseRef(target)
	if err != nil {
		return err
	}
	if ref.IsLatest() {
		return fmt.Errorf("revoke requires a concrete version, not %q", auth.VersionLatest)
	}
	if *reason == "" {
		return fmt.Errorf("--reason is required: a revocation without a recorded cause is not auditable")
	}
	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	if err := env.broker.Revoke(context.Background(), ref, *reason); err != nil {
		return err
	}
	fmt.Printf("Revoked %s\n", ref)
	fmt.Println("\nRevocation stops AILANG from using this version. It does NOT end sessions the")
	fmt.Println("website already issued — revoke those in the site's own security settings, and")
	fmt.Println("rotate the account password if the state may have been exposed.")
	return nil
}

func browserProfilePurge(args []string) error {
	target, args := splitPositional(args)
	fs := flag.NewFlagSet("purge", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("purge requires alias@version")
	}
	ref, err := auth.ParseRef(target)
	if err != nil {
		return err
	}
	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	if err := env.registry.Purge(context.Background(), ref); err != nil {
		return err
	}
	fmt.Printf("Purged material for %s\n", ref)
	return nil
}

func browserProfileAudit(args []string) error {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	limit := fs.Int("limit", 50, "Show at most this many of the most recent events")
	asJSON := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	events, err := auth.ReadAuditLog(env.auditPath)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		fmt.Printf("No audit events in %s\n", env.auditPath)
		return nil
	}
	if *limit > 0 && len(events) > *limit {
		events = events[len(events)-*limit:]
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(events)
	}
	for _, event := range events {
		fmt.Printf("%s  %-12s %-10s %s@%s  run=%s principal=%s",
			event.At.Format(time.RFC3339), event.Op, event.Decision,
			event.Alias, event.Version, event.Run.RunID, event.Run.Principal)
		if event.FailureCategory != "" {
			fmt.Printf("  %s", event.FailureCategory)
		}
		if event.Reason != "" {
			fmt.Printf("  (%s)", event.Reason)
		}
		fmt.Println()
	}
	return nil
}

func browserProfileGC(args []string) error {
	fs := flag.NewFlagSet("gc", flag.ExitOnError)
	maxAge := fs.Duration("max-age", time.Hour, "Remove materializations older than this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, err := openBrowserProfileEnv()
	if err != nil {
		return err
	}
	sink, err := auth.NewFileAuditSink(env.auditPath)
	if err != nil {
		return err
	}
	result, err := auth.AuditOrphanMaterializations(context.Background(),
		filepath.Join(env.root, "materializations"),
		auth.OrphanAuditOptions{
			MinAge: *maxAge,
			Sink:   sink,
			Run:    auth.RunIdentity{RunID: "gc", Principal: operatorPrincipal()},
			Where:  "cli",
		})
	if err != nil {
		return err
	}
	fmt.Printf("Scanned %d materialization(s); removed %d, failed %d\n",
		result.Scanned, result.Removed, result.Failed)
	if result.Failed > 0 {
		return fmt.Errorf("%d materialization(s) could not be removed; decrypted state may remain on disk (see %s)",
			result.Failed, env.auditPath)
	}
	return nil
}

func browserProfileRefresh(args []string) error {
	target, args := splitPositional(args)
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	publishVersion := fs.String("publish-version", "", "Version to publish after a successful login (required)")
	reason := fs.String("reason", "", "Why the profile is being refreshed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("refresh requires alias@version")
	}
	if *publishVersion == "" {
		return fmt.Errorf("--publish-version is required: refresh publishes a NEW version, it never overwrites")
	}
	_ = reason

	// Automated refresh needs a reviewed, site-specific login adapter. None
	// ships, and a generic model-driven password filler is explicitly out of
	// scope, so this fails loudly rather than pretending.
	return fmt.Errorf("no site login adapter is registered for %s.\n"+
		"  Automated refresh requires reviewed, site-specific control-plane code implementing\n"+
		"  auth.LoginAdapter. AILANG deliberately ships no generic model-driven login step.\n"+
		"  Until an adapter exists, re-run bootstrap with a freshly captured state:\n"+
		"    ailang browser-profile bootstrap %s --version %s --state-file ./state.json ...",
		target, strings.SplitN(target, "@", 2)[0], *publishVersion)
}

// splitPositional pulls a leading positional argument out before flag parsing.
//
// Go's flag package stops at the first non-flag argument, so
// `browser-profile bootstrap crm --origins ...` would silently parse ZERO flags
// and fall back to every default. That failure mode is quiet and dangerous here
// — it would publish a profile with default policy — so the positional is
// removed up front rather than relying on operators to put flags first.
func splitPositional(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func artifactSummary(policy auth.AuthProfilePolicy) string {
	if !policy.ArtifactPolicyPresent() {
		return "UNSET (denies)"
	}
	if len(policy.AllowArtifacts) == 0 {
		return "none"
	}
	return strings.Join(policy.AllowArtifacts, ",")
}

func operatorPrincipal() string {
	if user := os.Getenv("USER"); user != "" {
		return "operator:" + user
	}
	return "operator"
}
