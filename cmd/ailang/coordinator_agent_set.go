package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// `ailang coordinator agent-set <agent-id> <field>=<value>` — edit one field on
// one agent, safely, and commit it. It does NOT deploy.
//
// It used to drive the dev -> test -> prod ladder too, and that half is gone
// (Mark, 2026-09-13: "this is not an improvement"). It was worth removing on its
// own record: its wait matched any build for a trigger rather than the one for
// this commit, so it reported three SUCCESSes in a second and pushed prod before
// test had built; and it could not tell a rung with nothing to deploy from a
// rung that failed to fire. Meanwhile M-AGENT-CONFIG-FAST-PATH cut each rung
// from 75s to 32s, so the three pushes are now cheap and, unlike this command,
// they cannot lie about what happened.
//
// What remains is the part with no substitute: a safe edit and the local checks
// that catch real mistakes in a second, before a commit exists.
//
// Two deliberate properties:
//
//   - The edit is LINE-ORIENTED, not a YAML round-trip. Re-serialising a
//     1300-line config would reflow it and drop the comments that carry every
//     ruling in this file ("Mark, attended 2026-09-11", the auto-merge scope
//     warning). One line changes; every other byte is preserved.
//   - Validation runs BEFORE the commit, not after the deploy. A field the
//     registry does not read is refused up front — that is the dead-key class
//     (`push_branch`) caught the hard way on 2026-09-11.

type agentSetOpts struct {
	agentID    string
	field      string
	value      string
	repoConfig string
	dryRun     bool
}

func coordinatorAgentSet(args []string) error {
	opts := agentSetOpts{repoConfig: os.Getenv("AILANG_AGENT_CHECK_REPO_CONFIG")}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-config":
			if i+1 < len(args) {
				opts.repoConfig = args[i+1]
				i++
			}
		case "--dry-run":
			opts.dryRun = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				positional = append(positional, args[i])
			}
		}
	}
	if len(positional) < 2 {
		return fmt.Errorf("usage: ailang coordinator agent-set <agent-id> <field>=<value> [--repo-config <config.cloud.yaml>] [--dry-run]")
	}
	opts.agentID = positional[0]
	field, value, ok := strings.Cut(positional[1], "=")
	if !ok || strings.TrimSpace(field) == "" || strings.TrimSpace(value) == "" {
		return fmt.Errorf("expected <field>=<value>, got %q", positional[1])
	}
	opts.field, opts.value = strings.TrimSpace(field), strings.TrimSpace(value)

	if opts.repoConfig == "" {
		return fmt.Errorf("--repo-config is required: the repo is the source of truth, and the bucket is a cache a deploy regenerates from it")
	}
	return runAgentSet(context.Background(), opts)
}

func runAgentSet(ctx context.Context, o agentSetOpts) error {
	repoDir, err := repoDirOf(o.repoConfig)
	if err != nil {
		return err
	}

	original, err := os.ReadFile(o.repoConfig)
	if err != nil {
		return fmt.Errorf("reading %s: %w", o.repoConfig, err)
	}

	// Refuse to build on someone else's uncommitted edit: the commit below would
	// carry it, unreviewed and unattributed.
	if dirty, derr := fileIsDirty(ctx, repoDir, o.repoConfig); derr == nil && dirty {
		return fmt.Errorf("%s has uncommitted changes — commit or revert them first, or this change would carry them", o.repoConfig)
	}

	edited, before, after, err := setAgentField(string(original), o.agentID, o.field, o.value)
	if err != nil {
		return err
	}
	if before == after {
		fmt.Printf("agent-set: %s.%s is already %q — nothing to do\n", o.agentID, o.field, after)
		return nil
	}

	fmt.Printf("agent-set: %s\n\n  %s: %s -> %s\n\n", o.agentID, o.field, orNone(before), after)

	// Validate against the EDITED bytes, before anything is committed.
	if err := validateEditedConfig(edited, o.agentID, o.field); err != nil {
		return err
	}
	fmt.Println("  ✓ strict parse: every key in the config is read by the registry")
	fmt.Println("  ✓ registry loads and still contains the agent")

	if o.dryRun {
		fmt.Println("\n--dry-run: nothing written, committed or pushed")
		return nil
	}

	if err := os.WriteFile(o.repoConfig, []byte(edited), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", o.repoConfig, err)
	}
	msg := fmt.Sprintf("config(%s): %s = %s\n\nSet with `ailang coordinator agent-set`.", o.agentID, o.field, o.value)
	if err := commitFile(ctx, repoDir, o.repoConfig, msg); err != nil {
		return err
	}
	fmt.Println("  ✓ committed")

	fmt.Printf(`
Committed, NOT deployed — the live plane still runs the old value.

Each rung is a push; the agents fast path makes them ~32s each:

  cd %s
  git push origin dev:dev            # then wait for ailang-multivac-agents-dev
  git push origin origin/dev:test    #      "        ailang-multivac-agents-test
  git push origin origin/test:prod   #      "        ailang-multivac-agents-prod

  gcloud builds list --project=ailang-multivac-deploy --region=europe-west3 --limit=3
  ailang coordinator agent-check %s --repo-config %s
`, repoDir, o.agentID, o.repoConfig)
	return nil
}

// validateEditedConfig runs, on the edited bytes, the checks that actually catch
// agent mistakes — before a commit exists.
func validateEditedConfig(edited, agentID, field string) error {
	unknown, err := coordinator.UnknownConfigKeys([]byte(edited))
	if err != nil {
		return fmt.Errorf("the edited config does not parse: %w", err)
	}
	for _, k := range unknown {
		// The field just set is the one worth naming precisely: a key the
		// registry does not read looks decisive and does nothing (push_branch,
		// 2026-09-11).
		if strings.HasSuffix(k, field) {
			return fmt.Errorf("%q is not a field the agent registry reads — setting it would look decisive and do nothing", field)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("the config contains keys nothing reads: %s", strings.Join(unknown, ", "))
	}

	tmp, err := os.CreateTemp("", "agent-set-*.yaml")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(edited); err != nil {
		return err
	}
	_ = tmp.Close()

	reg, err := coordinator.LoadAgentRegistryFrom(tmp.Name())
	if err != nil {
		return fmt.Errorf("the edited config does not load as a registry: %w", err)
	}
	if reg.GetAgentByID(agentID) == nil {
		return fmt.Errorf("after the edit the registry no longer contains %q", agentID)
	}
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}

func repoDirOf(configPath string) (string, error) {
	// Through gitexec like every other git call: check-git-exec refuses bare-name
	// git exec sites outside that package, and the baseline is a ratchet, not a
	// place to record an exception.
	out, err := gitexec.CommandContext(context.Background(), "-C", dirOf(configPath), "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("%s is not inside a git repo: %w", configPath, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func dirOf(p string) string {
	if i := strings.LastIndex(p, "/"); i > 0 {
		return p[:i]
	}
	return "."
}

func fileIsDirty(ctx context.Context, repoDir, path string) (bool, error) {
	out, err := gitexec.CommandContext(ctx, "-C", repoDir, "status", "--porcelain", "--", path).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func commitFile(ctx context.Context, repoDir, path, message string) error {
	if err := gitexec.CommandContext(ctx, "-C", repoDir, "add", path).Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	cmd := gitexec.CommandContext(ctx, "-C", repoDir, "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
