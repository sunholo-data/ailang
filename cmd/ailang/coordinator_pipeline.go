package main

// `ailang coordinator pipeline` — did the chain run, and where did it stop?
//
// The question "is the pipeline working?" had no answer short of reading a
// message list and inferring. That inference was wrong repeatedly on
// 2026-09-14: a handoff that dispatched correctly looked identical to one that
// did not, because the failure was a `deduplicated` completion one second later
// and nothing put those two facts side by side.
//
// So this puts them side by side. One row per stage, in chain order, with the
// first stage that did not advance named explicitly. A stage that received work
// and produced nothing is a different fault from one that never received any,
// and the difference is the whole diagnosis.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// pipelineStage is one row of the report.
type pipelineStage struct {
	Agent    string   `json:"agent"`
	Inbox    string   `json:"inbox"`
	Inbound  int      `json:"inbound"`
	Outcomes []string `json:"outcomes"`
	Stalled  string   `json:"stalled,omitempty"`
}

// inboundMessageTypes are the types that ASK for work. Deliberately the
// complement of the result types health buckets separately: an agent's own
// completion arriving in its own inbox is not a stage receiving input, and
// counting it as one is how a stalled stage reads as a busy one.
var inboundMessageTypes = map[string]bool{
	messaging.InboxTypeRequest:      true,
	messaging.InboxTypeNotification: true,
	messaging.InboxTypeHandoff:      true,
	messaging.InboxTypeFeedback:     true,
}

func coordinatorPipeline(args []string) error {
	since := time.Now().Add(-24 * time.Hour)
	asJSON := false
	var chain []string
	var registryPath string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--json":
			asJSON = true
		case args[i] == "--since" && i+1 < len(args):
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return fmt.Errorf("--since %q: %w", args[i], err)
			}
			since = time.Now().Add(-d)
		case args[i] == "--chain" && i+1 < len(args):
			i++
			chain = strings.Split(args[i], ",")
		case args[i] == "--registry" && i+1 < len(args):
			i++
			registryPath = args[i]
		}
	}

	reg, source, err := resolveInboxRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("cannot load the registry: %w", err)
	}
	if len(chain) == 0 {
		chain = defaultPipelineChain(reg)
	}
	if len(chain) == 0 {
		return fmt.Errorf("no chain found in the registry — pass --chain a,b,c")
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{Limit: 500})
	if err != nil {
		return fmt.Errorf("listing messages: %w", err)
	}

	stages := make([]pipelineStage, 0, len(chain))
	for _, agentID := range chain {
		agent := reg.GetAgentByID(agentID)
		inbox := agentID
		if agent != nil && agent.Inbox != "" {
			inbox = agent.Inbox
		}
		st := pipelineStage{Agent: agentID, Inbox: inbox}
		for _, m := range msgs {
			if m.ToInbox != inbox || m.CreatedAt.Before(since) {
				continue
			}
			if inboundMessageTypes[m.MessageType] {
				st.Inbound++
			}
			if m.MessageType == messaging.InboxTypeCompletion {
				st.Outcomes = append(st.Outcomes, completionStatus(m.Payload))
			}
		}
		stages = append(stages, st)
	}

	// The first stage that did not advance. Ordered checks, because "received
	// nothing" and "received work and produced nothing" are different faults.
	for i := range stages {
		s := &stages[i]
		switch {
		case s.Inbound == 0 && len(s.Outcomes) == 0:
			s.Stalled = "never received work"
		case len(s.Outcomes) == 0:
			s.Stalled = "received work, produced no completion"
		case allBlocked(s.Outcomes):
			s.Stalled = "every completion was " + s.Outcomes[0]
		}
		if s.Stalled != "" {
			break // downstream silence is a consequence, not a separate fault
		}
	}

	if asJSON {
		return printJSON(stages)
	}
	fmt.Printf("\npipeline (registry: %s, since %s)\n\n", source, since.Format("15:04 02 Jan"))
	fmt.Printf("  %-26s %-6s %s\n", "STAGE", "IN", "OUTCOMES")
	fmt.Println("  " + strings.Repeat("-", 70))
	for _, s := range stages {
		fmt.Printf("  %-26s %-6d %s\n", s.Agent, s.Inbound, orDash(summarizeOutcomes(s.Outcomes)))
	}
	fmt.Println()
	stopped := false
	for _, s := range stages {
		if s.Stalled != "" {
			fmt.Printf("  %s %s: %s\n", red("STOPPED"), s.Agent, s.Stalled)
			stopped = true
			break
		}
	}
	if !stopped {
		fmt.Printf("  %s every stage received work and completed.\n", green("OK"))
	}
	fmt.Println()
	return nil
}

// defaultPipelineChain walks trigger_on_complete from the agent nothing points
// at — the registry already declares the topology, so asking the operator to
// retype it would be a second source of truth.
func defaultPipelineChain(reg *coordinator.AgentRegistry) []string {
	agents := reg.ListAgents()
	targeted := map[string]bool{}
	edges := map[string][]string{}
	for _, a := range agents {
		edges[a.ID] = a.TriggerOnComplete
		for _, t := range a.TriggerOnComplete {
			targeted[t] = true
		}
	}
	var heads []string
	for _, a := range agents {
		if len(a.TriggerOnComplete) > 0 && !targeted[a.ID] {
			heads = append(heads, a.ID)
		}
	}
	sort.Strings(heads)
	if len(heads) == 0 {
		return nil
	}
	// Longest chain from a head, so a two-stage branch does not hide a four.
	var best []string
	for _, h := range heads {
		var walk func(id string, seen map[string]bool) []string
		walk = func(id string, seen map[string]bool) []string {
			if seen[id] {
				return []string{id} // a cycle: stop, `coordinator lint` reports it
			}
			seen[id] = true
			longest := []string{id}
			for _, t := range edges[id] {
				if c := walk(t, seen); len(c)+1 > len(longest) {
					longest = append([]string{id}, c...)
				}
			}
			return longest
		}
		if c := walk(h, map[string]bool{}); len(c) > len(best) {
			best = c
		}
	}
	return best
}

// summarizeOutcomes collapses a run of statuses into counts.
//
// A stage that ran forty times printed forty words, and the one that mattered —
// a single `deduplicated` — was the last of them. Counts in a fixed order make
// the shape of a stage readable at a glance: "completed×14 failed×15" is a
// flaky stage, "deduplicated×1" is a blocked one.
func summarizeOutcomes(outcomes []string) string {
	if len(outcomes) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, o := range outcomes {
		counts[o]++
	}
	// Fixed order, not map order: the same run must render identically twice.
	order := []string{"completed", "no_changes", "failed", "deduplicated"}
	seen := map[string]bool{}
	var parts []string
	for _, k := range order {
		if n := counts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s×%d", k, n))
			seen[k] = true
		}
	}
	var rest []string
	for k, n := range counts {
		if !seen[k] {
			rest = append(rest, fmt.Sprintf("%s×%d", k, n))
		}
	}
	sort.Strings(rest)
	return strings.Join(append(parts, rest...), "  ")
}

func completionStatus(payload string) string {
	var o struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(payload), &o); err != nil || o.Status == "" {
		return "?"
	}
	return o.Status
}

// allBlocked reports whether every completion was a non-advancing outcome.
// A single success among failures means the stage DID work.
func allBlocked(outcomes []string) bool {
	for _, o := range outcomes {
		if o != "deduplicated" && o != "failed" {
			return false
		}
	}
	return len(outcomes) > 0
}
