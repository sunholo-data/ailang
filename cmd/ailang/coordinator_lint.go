package main

// `ailang coordinator lint` — validate the WHOLE agent registry, not one agent.
//
// Why this is separate from `agent-check`. That command answers "is THIS agent
// deployed and coherent", one agent at a time, against the live plane. Every
// fault it can see is local to a single entry. But the faults that have actually
// cost us time are RELATIONAL — an edge pointing at an agent that does not
// exist, a handoff that no code path can ever release, a cycle — and none of
// them are visible from inside one entry. There was no command that could see
// them, so nothing did.
//
// Every rule here is a bug that already happened. A lint rule invented from
// first principles tends to be a rule nobody believes; a rule with a date and a
// task id attached gets fixed instead of suppressed.
//
// Run against a FILE (`--registry config/config.cloud.yaml`) so it gates a
// config BEFORE deploy. `cloudbuild-agents-only.yaml` runs it as verify-registry
// for exactly that reason: a linter that only exists as a command someone could
// run is a linter nobody runs — this repo already has seven abandoned `diag-*`
// probe inboxes proving the point.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

type lintFinding struct {
	Agent string
	Rule  string
	Msg   string
	Fix   string
}

func coordinatorLint(args []string) error {
	var registryPath string
	quiet := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--registry" && i+1 < len(args):
			i++
			registryPath = args[i]
		case strings.HasPrefix(a, "--registry="):
			registryPath = strings.TrimPrefix(a, "--registry=")
		case a == "--quiet":
			quiet = true
		}
	}

	reg, source, err := resolveInboxRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("cannot load the registry to lint: %w", err)
	}
	agents := reg.ListAgents()
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })

	findings := lintRegistry(agents)

	if !quiet || len(findings) > 0 {
		fmt.Printf("coordinator lint: %s — %d agents, %d rule(s)\n\n", source, len(agents), len(lintRuleNames))
	}
	if len(findings) == 0 {
		if !quiet {
			fmt.Printf("%s no findings.\n", green("✓"))
		}
		return nil
	}
	for _, f := range findings {
		fmt.Printf("  %s %-28s %s\n", red("✗"), f.Agent, f.Msg)
		fmt.Printf("      %-28s %s\n", cyan("rule: "+f.Rule), f.Fix)
	}
	fmt.Println()
	return fmt.Errorf("%d registry finding(s)", len(findings))
}

var lintRuleNames = []string{
	"edge-target-exists",
	"handoff-can-fire",
	"no-chain-cycle",
	"automerge-is-bounded",
	"deploy-key-coherent",
	"inbox-not-near-duplicate",
}

// lintRegistry is pure over the agent list so every rule is testable without a
// plane, a bucket or a network.
func lintRegistry(agents []*coordinator.AgentConfig) []lintFinding {
	byID := map[string]*coordinator.AgentConfig{}
	for _, a := range agents {
		byID[a.ID] = a
	}
	var out []lintFinding
	for _, a := range agents {
		out = append(out, lintEdges(a, byID)...)
		out = append(out, lintCoherence(a)...)
	}
	out = append(out, lintCycles(agents, byID)...)
	out = append(out, lintNearDuplicateInboxes(agents)...)
	return out
}

// lintEdges: an edge must point somewhere, and must be releasable by something.
func lintEdges(a *coordinator.AgentConfig, byID map[string]*coordinator.AgentConfig) []lintFinding {
	var out []lintFinding
	for _, tgt := range a.TriggerOnComplete {
		if byID[tgt] == nil {
			out = append(out, lintFinding{
				Agent: a.ID, Rule: "edge-target-exists",
				Msg: fmt.Sprintf("trigger_on_complete names %q, which is not an agent in this registry", tgt),
				Fix: "the handoff resolves the TARGET in the registry and gives up silently when it is absent — fix the name or remove the edge",
			})
			continue
		}
		// THE LANDMINE. An edge is released by exactly one of two paths:
		//   auto  -> dispatched at completion by finalizer.autoHandoffTargets
		//   gated -> embedded in the approval record, released on approval
		// applyApproval only runs when `completed && !SkipApproval`. So an agent
		// that skips approval and has a NON-auto edge has neither path: nothing
		// embeds the target and nothing dispatches it. The edge is present in
		// config, reads as configured, and fires never — with no log line.
		//
		// Latent, not live, when this rule was written (2026-09-14). It is here
		// because the same shape has already shipped twice: a "Triggering next
		// stage: X" GitHub comment nothing acted on, and an EvaluationRound
		// counter the template promises and no code assigns.
		if a.SkipApproval && !a.AutoApprovesHandoffTo(tgt) {
			out = append(out, lintFinding{
				Agent: a.ID, Rule: "handoff-can-fire",
				Msg: fmt.Sprintf("handoff to %q can never fire: skip_approval is true, so no approval record is created to release it, and the edge is not auto", tgt),
				Fix: fmt.Sprintf("add %q to auto_approve_handoff_to, or drop skip_approval", tgt),
			})
		}
	}
	return out
}

// lintCoherence mirrors agent-check's per-agent rules so a registry-wide run
// catches them too. Deliberately duplicated in EFFECT, not in code path: this
// one is pure and runs over a file in CI, agent-check probes the live plane.
func lintCoherence(a *coordinator.AgentConfig) []lintFinding {
	var out []lintFinding
	// The scope guard reads the DECLARED list; the effective default is `**/*`,
	// which bounds nothing, so the wrapper refuses instead. Reads as
	// "auto_merge silently does nothing".
	if a.AutoMerge && len(a.ArtifactPatterns) == 0 {
		out = append(out, lintFinding{
			Agent: a.ID, Rule: "automerge-is-bounded",
			Msg: "auto_merge is on with no DECLARED artifact_patterns, so the scope guard bounds nothing and refuses",
			Fix: "declare artifact_patterns, or turn auto_merge off",
		})
	}
	// A deploy key is SSH-only: it cannot reach the GitHub API, so it cannot
	// open a PR or set auto-merge.
	if a.SSHKeySecret != "" && a.AutoMerge {
		out = append(out, lintFinding{
			Agent: a.ID, Rule: "deploy-key-coherent",
			Msg: "auto_merge with an ssh deploy key: a deploy key cannot use the GitHub API, so auto-merge can never fire",
			Fix: "drop auto_merge, or move the agent to token auth",
		})
	}
	if a.SSHKeySecret != "" && a.SSHHostAlias == "" {
		out = append(out, lintFinding{
			Agent: a.ID, Rule: "deploy-key-coherent",
			Msg: "ssh_key_secret without ssh_host_alias: the alias is the bound that stops the key reaching other repos",
			Fix: "set ssh_host_alias",
		})
	}
	return out
}

// lintCycles: a chain that returns to itself never terminates.
//
// Not hypothetical. The evaluator is the natural place to want a loop back to
// the executor ("re-run it if the verdict fails"), and wiring that as a plain
// trigger_on_complete edge would re-dispatch on PASS as well as FAIL — an
// unbounded loop of real executor runs. A retry needs a ROUND COUNTER, which
// this codebase does not have: templates.go promises "another evaluation round
// will follow automatically (round N/3)" and nothing anywhere assigns
// EvaluationRound. Until that exists, a cycle in the registry is a bug.
func lintCycles(agents []*coordinator.AgentConfig, byID map[string]*coordinator.AgentConfig) []lintFinding {
	var out []lintFinding
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := map[string]int{}
	var path []string
	var visit func(id string) []string
	visit = func(id string) []string {
		a := byID[id]
		if a == nil {
			return nil
		}
		colour[id] = grey
		path = append(path, id)
		for _, tgt := range a.TriggerOnComplete {
			switch colour[tgt] {
			case grey:
				// Found the back edge. Report the cycle from where it closes.
				for i, p := range path {
					if p == tgt {
						cyc := append(append([]string{}, path[i:]...), tgt)
						colour[id] = black
						path = path[:len(path)-1]
						return cyc
					}
				}
			case white:
				if cyc := visit(tgt); cyc != nil {
					colour[id] = black
					path = path[:len(path)-1]
					return cyc
				}
			}
		}
		colour[id] = black
		path = path[:len(path)-1]
		return nil
	}
	seen := map[string]bool{}
	for _, a := range agents {
		if colour[a.ID] != white {
			continue
		}
		if cyc := visit(a.ID); cyc != nil {
			key := strings.Join(cyc, ">")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, lintFinding{
				Agent: cyc[0], Rule: "no-chain-cycle",
				Msg: "trigger_on_complete forms a cycle: " + strings.Join(cyc, " -> "),
				Fix: "a retry loop needs a bounded round counter; there is none (EvaluationRound is declared in templates.go and assigned nowhere) — break the cycle",
			})
		}
	}
	return out
}

// lintNearDuplicateInboxes: two inboxes that differ only by separator or case
// are a typo waiting to happen, and a typo'd inbox is ACCEPTED, filed, and never
// acted on.
//
// Measured 2026-09-13: five design requests went to `daneel-design-ailang`
// instead of `daneel-design`. Each was filed, each bounced, and each bounce was
// itself addressed to an unregistered inbox. No work was lost only because a
// human noticed twenty minutes later.
func lintNearDuplicateInboxes(agents []*coordinator.AgentConfig) []lintFinding {
	norm := func(s string) string {
		r := strings.NewReplacer("-", "", "_", "", ":", "", "/", "", ".", "")
		return strings.ToLower(r.Replace(s))
	}
	byNorm := map[string][]string{}
	for _, a := range agents {
		if a.Inbox == "" {
			continue
		}
		n := norm(a.Inbox)
		byNorm[n] = append(byNorm[n], a.Inbox)
	}
	var out []lintFinding
	keys := make([]string, 0, len(byNorm))
	for k := range byNorm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := byNorm[k]
		if len(v) < 2 {
			continue
		}
		sort.Strings(v)
		out = append(out, lintFinding{
			Agent: v[0], Rule: "inbox-not-near-duplicate",
			Msg: "inboxes differ only by separator or case: " + strings.Join(v, ", "),
			Fix: "a send to the wrong one is filed and never dispatched — pick one spelling",
		})
	}
	return out
}
