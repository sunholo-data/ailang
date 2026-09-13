package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// `ailang coordinator agents [<id>]` — what is ACTUALLY registered on the live
// plane, and for one agent, every field including the ones it never declared.
//
// There was no way to ask this. `coordinator list` lists tasks; `messages
// inboxes` needs a local registry file and answers a routing question; reading
// config.cloud.yaml answers what the REPO says, which is not the same claim —
// the two disagreed for two days in September 2026 and six agents vanished in
// the gap.
//
// The detail view exists because most of an agent's behaviour is not in its
// entry. Timeout, idle timeout, invoke, output markers, artifact patterns and
// approval all fall back to per-agent defaults compiled into the registry, so a
// field absent from the YAML is not a field that is unset — and `**/*` as an
// undeclared artifact-pattern default is how an auto-merge guard came to bound
// nothing. Declared and effective are printed as separate columns for exactly
// that reason.

func coordinatorAgents(args []string) error {
	asJSON := false
	var want string
	for _, a := range args {
		switch {
		case a == "--json":
			asJSON = true
		case !strings.HasPrefix(a, "-"):
			want = a
		}
	}

	reg, source, err := loadCloudInboxRegistry()
	if err != nil {
		return fmt.Errorf("cannot read the live registry: %w", err)
	}
	agents := reg.ListAgents()
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })

	if want != "" {
		for _, a := range agents {
			if a.ID == want {
				if asJSON {
					return printJSON(agentDetail(a))
				}
				printAgentDetail(a, source)
				return nil
			}
		}
		return fmt.Errorf("no agent %q on the live plane (%d registered) — `ailang coordinator agents` lists them", want, len(agents))
	}

	if asJSON {
		rows := make([]map[string]any, 0, len(agents))
		for _, a := range agents {
			rows = append(rows, agentDetail(a))
		}
		return printJSON(rows)
	}

	fmt.Printf("live registry: %s — %d agents\n\n", source, len(agents))
	fmt.Printf("%-34s %-32s %-8s %s\n", "AGENT", "INBOX", "PROVIDER", "MODEL")
	for _, a := range agents {
		fmt.Printf("%-34s %-32s %-8s %s\n", a.ID, a.Inbox, a.Provider, orDash(a.Model))
	}
	fmt.Printf("\n`ailang coordinator agents <id>` shows every field, declared and defaulted.\n")
	return nil
}

func printAgentDetail(a *coordinator.AgentConfig, source string) {
	fmt.Printf("%s   (live registry: %s)\n\n", a.ID, source)

	// Declared: what the YAML says.
	fmt.Println("DECLARED")
	for _, kv := range [][2]string{
		{"label", a.Label},
		{"inbox", a.Inbox},
		{"workspace", a.Workspace},
		{"merge_branch", a.MergeBranch},
		{"provider", a.Provider},
		{"executor_variant", a.ExecutorVariant},
		{"role", a.Role},
		{"model", a.Model},
		{"timeout", a.Timeout},
		{"idle_timeout", a.IdleTimeout},
		{"ssh_key_secret", a.SSHKeySecret},
		{"ssh_host_alias", a.SSHHostAlias},
	} {
		if kv[1] != "" {
			fmt.Printf("  %-20s %s\n", kv[0], kv[1])
		}
	}
	fmt.Printf("  %-20s %v\n", "auto_merge", a.AutoMerge)
	fmt.Printf("  %-20s %v\n", "skip_approval", a.SkipApproval)
	fmt.Printf("  %-20s %v\n", "auto_approve_handoffs", a.AutoApproveHandoffs)
	if len(a.Capabilities) > 0 {
		fmt.Printf("  %-20s %s\n", "capabilities", strings.Join(a.Capabilities, ", "))
	}
	if len(a.TriggerOnComplete) > 0 {
		fmt.Printf("  %-20s %s\n", "trigger_on_complete", strings.Join(a.TriggerOnComplete, ", "))
	}

	// Effective: what actually runs. A field absent above is not unset.
	fmt.Println("\nEFFECTIVE (defaults filled in — this is what runs)")
	fmt.Printf("  %-20s %s%s\n", "timeout", a.GetEffectiveTimeout(), defaulted(a.Timeout == ""))
	fmt.Printf("  %-20s %s%s\n", "idle_timeout", a.GetEffectiveIdleTimeout(), defaulted(a.IdleTimeout == ""))

	if inv := a.GetEffectiveInvokeConfig(); inv != nil {
		fmt.Printf("  %-20s type=%s name=%s%s\n", "invoke", inv.Type, inv.Name, defaulted(a.Invoke == nil))
	} else {
		fmt.Printf("  %-20s (none — an unknown agent with no invoke config runs the raw task content)\n", "invoke")
	}
	fmt.Printf("  %-20s %s%s\n", "output_markers",
		orDash(strings.Join(a.GetEffectiveOutputMarkers(), ", ")), defaulted(len(a.OutputMarkers) == 0))

	// The one that has already caused harm: an undeclared default of `**/*`
	// gives an auto-merge guard nothing to bound.
	eff := a.GetEffectiveArtifactPatterns()
	fmt.Printf("  %-20s %s%s\n", "artifact_patterns", orDash(strings.Join(eff, ", ")), defaulted(len(a.ArtifactPatterns) == 0))
	if len(a.ArtifactPatterns) == 0 && a.AutoMerge {
		fmt.Printf("  %-20s auto_merge is ON with no DECLARED artifact_patterns — the scope half of the merge guard reads the declared list, so it refuses\n", "⚠")
	}
	if ap := a.GetEffectiveApprovalConfig(); ap != nil {
		fmt.Printf("  %-20s needs=%s approved=%s%s\n", "approval", ap.NeedsLabel, ap.ApprovedLabel, defaulted(a.Approval == nil))
	}
}

func agentDetail(a *coordinator.AgentConfig) map[string]any {
	m := map[string]any{
		"id": a.ID, "inbox": a.Inbox, "workspace": a.Workspace, "provider": a.Provider,
		"model": a.Model, "role": a.Role, "merge_branch": a.MergeBranch,
		"auto_merge": a.AutoMerge, "skip_approval": a.SkipApproval,
		"declared": map[string]any{
			"timeout": a.Timeout, "idle_timeout": a.IdleTimeout,
			"output_markers": a.OutputMarkers, "artifact_patterns": a.ArtifactPatterns,
		},
		"effective": map[string]any{
			"timeout": a.GetEffectiveTimeout().String(), "idle_timeout": a.GetEffectiveIdleTimeout().String(),
			"output_markers": a.GetEffectiveOutputMarkers(), "artifact_patterns": a.GetEffectiveArtifactPatterns(),
		},
	}
	return m
}

func defaulted(isDefault bool) string {
	if isDefault {
		return "   (default — not in the config)"
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
