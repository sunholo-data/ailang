package coordinator

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The shadow report (M-AILANG-LANE-EXECUTOR-SHADOW M7): per-harness outcomes,
// every disagreement with what is needed to look at it, and the verdict tally.
// It feeds Mark's fleet-switch decision ("motoko writes better AILANG than
// codex/pi on real work over time"); it decides nothing itself.

// ShadowHarnessStats aggregates one harness's runs, whichever side it ran on.
type ShadowHarnessStats struct {
	Harness    string  `json:"harness"`
	Runs       int     `json:"runs"`
	Graded     int     `json:"graded"`      // runs the mechanical layer could grade
	Passed     int     `json:"passed"`      // of those, passed
	NoOutput   int     `json:"no_output"`   // produced no diff
	CostUSD    float64 `json:"cost_usd"`    // as each completion reported it
	DurationMS int64   `json:"duration_ms"` // total wall time
	Turns      int     `json:"turns"`
}

// PassRate is passed/graded, or 0 with nothing graded.
func (h ShadowHarnessStats) PassRate() float64 {
	if h.Graded == 0 {
		return 0
	}
	return float64(h.Passed) / float64(h.Graded)
}

// ShadowDisagreement is one task where the verdict names a winner.
type ShadowDisagreement struct {
	ParentID        string        `json:"parent_id"`
	Verdict         ShadowVerdict `json:"verdict"`
	Reason          string        `json:"reason,omitempty"`
	PrimaryHarness  string        `json:"primary_harness"`
	ShadowHarness   string        `json:"shadow_harness"`
	ShadowTaskID    string        `json:"shadow_task_id"`
	PrimarySession  string        `json:"primary_session,omitempty"`
	ShadowSession   string        `json:"shadow_session,omitempty"`
	PrimaryArtifact string        `json:"primary_artifact,omitempty"`
	ShadowArtifact  string        `json:"shadow_artifact,omitempty"`
	PrimaryDiffStat string        `json:"primary_diffstat,omitempty"`
	ShadowDiffStat  string        `json:"shadow_diffstat,omitempty"`
	Merged          string        `json:"merged,omitempty"` // "#12 merged (circular)" etc.
	ReviewNote      string        `json:"review_note,omitempty"`
	BaseDrift       bool          `json:"base_drift,omitempty"`
}

// ShadowReport is the whole report.
type ShadowReport struct {
	Since         time.Time                      `json:"since"`
	Until         time.Time                      `json:"until"`
	Comparisons   int                            `json:"comparisons"`
	Done          int                            `json:"done"`
	InProgress    int                            `json:"in_progress"`
	BaseDrift     int                            `json:"base_drift"`
	Verdicts      map[ShadowVerdict]int          `json:"verdicts"`
	Ungraded      map[string]int                 `json:"ungraded_reasons,omitempty"`
	Harnesses     []ShadowHarnessStats           `json:"harnesses"`
	Disagreements []ShadowDisagreement           `json:"disagreements"`
	Wins          map[string]int                 `json:"wins"` // harness -> verdicts naming it the winner
	harnessIdx    map[string]*ShadowHarnessStats // build-time only
}

// BuildShadowReport summarises comparisons for the window [since, until].
func BuildShadowReport(cs []*ShadowComparison, since, until time.Time) *ShadowReport {
	r := &ShadowReport{Since: since, Until: until, Verdicts: map[ShadowVerdict]int{}, Ungraded: map[string]int{},
		Wins: map[string]int{}, harnessIdx: map[string]*ShadowHarnessStats{}}
	for _, c := range cs {
		r.Comparisons++
		if c.BaseDrift {
			r.BaseDrift++
		}
		r.addRun(c.Primary, c.Mechanical, true)
		r.addRun(c.Shadow, c.Mechanical, false)
		if c.GradeState != ShadowGradeDone {
			r.InProgress++
			continue
		}
		r.Done++
		r.Verdicts[c.Verdict]++
		switch c.Verdict {
		case ShadowVerdictUngraded:
			r.Ungraded[ungradedKey(c.UngradedReason)]++
		case ShadowVerdictPrimaryBetter, ShadowVerdictShadowBetter:
			d := disagreementOf(c)
			r.Disagreements = append(r.Disagreements, d)
			if c.Verdict == ShadowVerdictPrimaryBetter {
				r.Wins[d.PrimaryHarness]++
			} else {
				r.Wins[d.ShadowHarness]++
			}
		}
	}
	for _, h := range r.harnessIdx {
		r.Harnesses = append(r.Harnesses, *h)
	}
	sort.Slice(r.Harnesses, func(i, j int) bool { return r.Harnesses[i].Harness < r.Harnesses[j].Harness })
	return r
}

func (r *ShadowReport) addRun(run *ShadowRunResult, m *ShadowMechanicalGrade, primary bool) {
	if run == nil {
		return
	}
	name := harnessName(run)
	h := r.harnessIdx[name]
	if h == nil {
		h = &ShadowHarnessStats{Harness: name}
		r.harnessIdx[name] = h
	}
	h.Runs++
	h.CostUSD += run.CostUSD
	h.DurationMS += run.DurationMS
	h.Turns += run.NumTurns
	if strings.TrimSpace(run.Diff) == "" {
		h.NoOutput++
	}
	if m == nil {
		return
	}
	g := m.Shadow
	if primary {
		g = m.Primary
	}
	if g.Ungraded == "" {
		h.Graded++
		if g.Pass {
			h.Passed++
		}
	}
}

func harnessName(run *ShadowRunResult) string {
	if run == nil || run.Provider == "" {
		return "unknown"
	}
	return run.Provider
}

// ungradedKey reduces a reason to its cause: the side prefix goes ("primary:
// apply_failed" counts as apply_failed) and so does the free-text tail
// ("reviewer_unavailable: 401 ..."), so the tally counts causes, not messages.
func ungradedKey(reason string) string {
	for _, side := range []string{ShadowSidePrimary + ": ", ShadowSideShadow + ": "} {
		reason = strings.TrimPrefix(reason, side)
	}
	if i := strings.IndexByte(reason, ':'); i > 0 {
		return reason[:i]
	}
	if reason == "" {
		return "unspecified"
	}
	return reason
}

func disagreementOf(c *ShadowComparison) ShadowDisagreement {
	d := ShadowDisagreement{ParentID: c.ParentID, Verdict: c.Verdict, Reason: c.UngradedReason,
		PrimaryHarness: harnessName(c.Primary), ShadowHarness: harnessName(c.Shadow),
		ShadowTaskID: shadowTaskID(c.ParentID), BaseDrift: c.BaseDrift}
	if p := c.Primary; p != nil {
		d.PrimarySession, d.PrimaryArtifact, d.PrimaryDiffStat = p.SessionID, p.ArtifactGCSPath, p.DiffStat
	}
	if s := c.Shadow; s != nil {
		d.ShadowSession, d.ShadowArtifact, d.ShadowDiffStat = s.SessionID, s.ArtifactGCSPath, s.DiffStat
	}
	if m := c.Merged; m != nil {
		d.Merged = m.State
		if m.Number > 0 {
			d.Merged = fmt.Sprintf("#%d %s", m.Number, m.State)
		}
		if m.Circular {
			d.Merged += " (circular)"
		}
	}
	if rv := c.Review; rv != nil && rv.Ungraded == "" && rv.Winner != "" {
		d.ReviewNote = fmt.Sprintf("%s (confidence %.2f): %s", rv.Winner, rv.Confidence, rv.StrongestObjection)
	}
	return d
}

// Markdown renders the report for a person (and the weekly inbox message).
func (r *ShadowReport) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Shadow report %s – %s\n\n", r.Since.Format("2006-01-02"), r.Until.Format("2006-01-02"))
	fmt.Fprintf(&b, "%d comparisons: %d graded to the end, %d still in progress", r.Comparisons, r.Done, r.InProgress)
	if r.BaseDrift > 0 {
		fmt.Fprintf(&b, ", %d with base drift", r.BaseDrift)
	}
	b.WriteString(".\n\n")
	if r.Comparisons == 0 {
		b.WriteString("No shadowed tasks in this window.\n")
		return b.String()
	}
	b.WriteString("## Harnesses\n\n| Harness | Runs | Pass (graded) | No output | Cost | Wall time | Turns | Wins |\n|---|---|---|---|---|---|---|---|\n")
	for _, h := range r.Harnesses {
		fmt.Fprintf(&b, "| %s | %d | %d/%d (%.0f%%) | %d | $%.2f | %s | %d | %d |\n", h.Harness, h.Runs, h.Passed, h.Graded,
			100*h.PassRate(), h.NoOutput, h.CostUSD, (time.Duration(h.DurationMS) * time.Millisecond).Round(time.Second), h.Turns, r.Wins[h.Harness])
	}
	b.WriteString("\n## Verdicts\n\n")
	for _, v := range []ShadowVerdict{ShadowVerdictPrimaryBetter, ShadowVerdictShadowBetter, ShadowVerdictEquivalent, ShadowVerdictBothFailed, ShadowVerdictUngraded} {
		fmt.Fprintf(&b, "- %s: %d\n", v, r.Verdicts[v])
	}
	if len(r.Ungraded) > 0 {
		keys := make([]string, 0, len(r.Ungraded))
		for k := range r.Ungraded {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s %d", k, r.Ungraded[k]))
		}
		fmt.Fprintf(&b, "  - ungraded because: %s\n", strings.Join(parts, ", "))
	}
	fmt.Fprintf(&b, "\n## Disagreements (%d)\n\n", len(r.Disagreements))
	if len(r.Disagreements) == 0 {
		b.WriteString("None: no task where one harness passed and the other did not.\n")
	}
	for _, d := range r.Disagreements {
		winner := d.PrimaryHarness + " (primary)"
		if d.Verdict == ShadowVerdictShadowBetter {
			winner = d.ShadowHarness + " (shadow)"
		}
		fmt.Fprintf(&b, "### %s — %s better\n\n", d.ParentID, winner)
		fmt.Fprintf(&b, "- primary %s: `ailang chains find %s` then `ailang chains chat <chain-id>`", d.PrimaryHarness, d.ParentID)
		if d.PrimaryDiffStat != "" {
			fmt.Fprintf(&b, "; diff: %s", oneLine(d.PrimaryDiffStat))
		}
		fmt.Fprintf(&b, "\n- shadow %s: task `%s`, session `%s`, artifacts `%s`", d.ShadowHarness, d.ShadowTaskID, orDash(d.ShadowSession), orDash(d.ShadowArtifact))
		if d.ShadowDiffStat != "" {
			fmt.Fprintf(&b, "; diff: %s", oneLine(d.ShadowDiffStat))
		}
		b.WriteString("\n")
		if d.Reason != "" {
			fmt.Fprintf(&b, "- decided by: %s\n", d.Reason)
		}
		if d.Merged != "" {
			fmt.Fprintf(&b, "- primary PR: %s\n", d.Merged)
		}
		if d.ReviewNote != "" {
			fmt.Fprintf(&b, "- reviewer: %s\n", d.ReviewNote)
		}
		if d.BaseDrift {
			b.WriteString("- base drift: the two runs started from different commits\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func oneLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1]) // git's diffstat summary line
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
