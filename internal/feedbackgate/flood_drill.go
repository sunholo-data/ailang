package feedbackgate

import (
	"context"
	"fmt"
	"io"
	"sort"
)

// FloodDrillResult is the outcome of a simulated flood run.
type FloodDrillResult struct {
	Total          int
	Dispatched     int
	Filed          int
	Rejected       int
	ReasonHist     map[string]int
	SimulatedSpend float64 // estimated USD that WOULD be spent on dispatched agents
	BaselineSpend  float64 // estimated USD if every message had dispatched (no gate)
}

// RunFloodDrill feeds n synthetic submissions from `contacts` distinct contacts
// through a fully-assembled gate (in-memory cooldown; optional fake classifier
// via cfg.Classifier) and reports the verdict histogram + simulated spend. It
// is deterministic and OFFLINE — no cloud, no live Anthropic, no Ollama. The
// caller supplies cfg (with an in-memory Cooldown and, optionally, a fake
// Classifier); this function never constructs a real provider.
//
// This is the engine behind scripts/security/feedback_flood_drill.sh.
func RunFloodDrill(ctx context.Context, cfg FeedbackGateConfig, n, contacts int) (FloodDrillResult, error) {
	cfg = cfg.normalized()
	res := FloodDrillResult{Total: n, ReasonHist: map[string]int{}}
	if contacts < 1 {
		contacts = 1
	}

	for i := 0; i < n; i++ {
		c := i % contacts
		in := Input{
			ID:       fmt.Sprintf("flood-%d", i),
			Category: "auto:bug",
			Body:     fmt.Sprintf("synthetic flood from contact %d\ncontact: flood%d@example.com\n", c, c),
			From:     "mcp-public",
			Inbox:    "pkg:flood/target",
			Source:   "public",
		}
		v, err := Decide(ctx, in, cfg)
		if err != nil {
			return res, fmt.Errorf("flood message %d: %w", i, err)
		}
		res.ReasonHist[v.Reason]++
		switch v.Action {
		case ActionDispatch:
			res.Dispatched++
			res.SimulatedSpend += v.Cost
		case ActionFile:
			res.Filed++
		case ActionReject:
			res.Rejected++
		}
		res.BaselineSpend += estimatedDispatchCostUSD
	}
	return res, nil
}

// WriteReport prints a human-readable histogram + spend summary to w.
func (r FloodDrillResult) WriteReport(w io.Writer) {
	fmt.Fprintf(w, "Feedback flood drill (OFFLINE — no cloud, no live API)\n")
	fmt.Fprintf(w, "  total submissions : %d\n", r.Total)
	fmt.Fprintf(w, "  dispatched        : %d\n", r.Dispatched)
	fmt.Fprintf(w, "  filed             : %d\n", r.Filed)
	fmt.Fprintf(w, "  rejected          : %d\n", r.Rejected)
	fmt.Fprintf(w, "  verdict reasons:\n")
	reasons := make([]string, 0, len(r.ReasonHist))
	for k := range r.ReasonHist {
		reasons = append(reasons, k)
	}
	sort.Strings(reasons)
	for _, k := range reasons {
		fmt.Fprintf(w, "    %-32s %d\n", k, r.ReasonHist[k])
	}
	fmt.Fprintf(w, "  simulated spend   : $%.4f (with gate)\n", r.SimulatedSpend)
	fmt.Fprintf(w, "  baseline spend    : $%.4f (no gate — every msg dispatches)\n", r.BaselineSpend)
	saved := r.BaselineSpend - r.SimulatedSpend
	fmt.Fprintf(w, "  spend avoided     : $%.4f\n", saved)
}
