package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// `ailang coordinator agent-set <agent-id> <field>=<value>` — change one field
// on one agent and take it all the way to prod.
//
// Why this exists: a one-line model pin cost six interactions and about ten
// minutes (edit, commit, push dev, wait, push test, wait, push prod, wait,
// verify). None of that ceremony validates the agent entry — the checks that
// catch real mistakes are local and take a second. The ladder is kept exactly
// as it is, because config and terraform share a trigger and the rungs are the
// deploy; what changes is that the operator performs ONE step instead of six.
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
	noDeploy   bool
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
		case "--no-deploy":
			opts.noDeploy = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				positional = append(positional, args[i])
			}
		}
	}
	if len(positional) < 2 {
		return fmt.Errorf("usage: ailang coordinator agent-set <agent-id> <field>=<value> [--repo-config <config.cloud.yaml>] [--dry-run] [--no-deploy]")
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

	if o.noDeploy {
		fmt.Println("\n--no-deploy: committed but NOT pushed. The live plane still runs the old value.")
		return nil
	}
	return deployConfigLadder(ctx, repoDir, o.agentID, o.repoConfig)
}

// deployConfigLadder walks dev -> test -> prod, waiting for each build.
//
// The rungs are not decoration: a push to each branch is what regenerates that
// environment's config bucket and rolls the coordinator. Skipping one leaves the
// environment behind, and the next config build there silently reverts this
// change — that is how six agents were deleted on 2026-09-10.
func deployConfigLadder(ctx context.Context, repoDir, agentID, repoConfig string) error {
	steps := []struct{ from, to, trigger string }{
		{"dev", "dev", "ailang-multivac-config-dev"},
		{"origin/dev", "test", "ailang-multivac-config-test"},
		{"origin/test", "prod", "ailang-multivac-config-prod"},
	}
	for _, s := range steps {
		fmt.Printf("\n  → pushing %s to %s\n", s.from, s.to)
		if err := gitPushRef(ctx, repoDir, s.from, s.to); err != nil {
			return fmt.Errorf("push %s -> %s: %w", s.from, s.to, err)
		}
		status, err := waitForTrigger(ctx, s.trigger, 12*time.Minute)
		if err != nil {
			return fmt.Errorf("waiting for %s: %w", s.trigger, err)
		}
		if status != "SUCCESS" {
			return fmt.Errorf("%s finished %s — stopping the ladder here rather than pushing a broken config onward", s.trigger, status)
		}
		fmt.Printf("    %s: SUCCESS\n", s.trigger)
	}

	fmt.Printf("\n  → verifying against the live plane\n\n")
	return coordinatorAgentCheck([]string{agentID, "--repo-config", repoConfig})
}

// waitForTrigger polls the most recent build for a trigger until it settles.
func waitForTrigger(ctx context.Context, trigger string, budget time.Duration) (string, error) {
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		out, err := exec.CommandContext(ctx, "gcloud", "builds", "list",
			"--project=ailang-multivac-deploy", "--region=europe-west3",
			"--limit=10", "--sort-by=~createTime",
			"--format=value(status,substitutions.TRIGGER_NAME)").Output()
		if err != nil {
			return "", fmt.Errorf("gcloud builds list: %w", err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[1] == trigger {
				switch fields[0] {
				case "SUCCESS", "FAILURE", "CANCELLED", "TIMEOUT", "INTERNAL_ERROR":
					return fields[0], nil
				}
				break // most recent build for this trigger is still running
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(20 * time.Second):
		}
	}
	// A timeout is not a pass. The build may still be running, and reporting
	// success here would be the "unverifiable is not verified" mistake.
	return "", fmt.Errorf("%s did not finish within %s — check `gcloud builds list --region=europe-west3`", trigger, budget)
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
	out, err := exec.Command("git", "-C", dirOf(configPath), "rev-parse", "--show-toplevel").Output()
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

func gitPushRef(ctx context.Context, repoDir, from, to string) error {
	cmd := gitexec.CommandContext(ctx, "-C", repoDir, "push", "origin", from+":"+to)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
