package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CorpusSummary is one corpus's line in the summary table.
type CorpusSummary struct {
	Name        string         `json:"name"`
	Level       string         `json:"level"` // blocking | advisory in this tier
	Files       int            `json:"files"`
	Packages    int            `json:"packages"`
	States      map[string]int `json:"states"`
	BaseFail    int            `json:"baseline_failing"`
	Unchecked   int            `json:"baseline_unchecked_deps"` // baseline failures that are only DEP/*
	BaseSecs    float64        `json:"baseline_cpu_secs"`
	CandSecs    float64        `json:"candidate_cpu_secs"`
	NotChecked  string         `json:"not_checked,omitempty"`
	EntryRules  map[string]int `json:"entry_rules"`
	CwdStdlibAt []string       `json:"cwd_stdlib_dirs,omitempty"`
}

// Report is report.json; report.md and pins.md are rendered from it.
type Report struct {
	Started     time.Time          `json:"started"`
	WallSecs    float64            `json:"wall_secs"`
	ArmWall     map[string]float64 `json:"arm_wall_secs"`
	Tier        string             `json:"tier"`
	RefMode     string             `json:"ref_mode"`
	Arms        []*Arm             `json:"arms"`
	NullControl bool               `json:"null_control"` // both arms are the same binary
	Failures    []string           `json:"instrument_failures"`
	Sources     []Source           `json:"sources"`
	Corpora     []CorpusSummary    `json:"corpora"`
	Buckets     []Bucket           `json:"buckets"`
	Expected    []ExpectedStatus   `json:"expected_breaks"`
	Uncoded     [2]int             `json:"uncoded_codes_of_candidate_failures"` // [uncoded, total]
	Audit       []AuditFinding     `json:"write_audit"`
	Pins        []PinHit           `json:"pins"`
	ExitCode    int                `json:"exit_code"`
	Verdict     string             `json:"verdict"`
	Rows        []Row              `json:"rows"` // every entry that is not pass_both
}

// PinHit is one pin line found in a snapshot (pins.md).
type PinHit struct {
	Corpus string `json:"corpus"`
	Where  string `json:"where"`
	Text   string `json:"text"`
}

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func cell(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ") }

// WriteReports writes report.json, report.md, pins.md and rows.jsonl into dir.
func (r *Report) WriteReports(dir string, all []Row) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), data, 0o644); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "rows.jsonl"))
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, row := range all {
		if err := enc.Encode(row); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	var md strings.Builder
	r.renderMarkdown(&md)
	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(md.String()), 0o644); err != nil {
		return err
	}
	var pins strings.Builder
	r.renderPins(&pins)
	return os.WriteFile(filepath.Join(dir, "pins.md"), []byte(pins.String()), 0o644)
}

func (r *Report) renderMarkdown(w io.Writer) {
	p := func(format string, a ...interface{}) { fmt.Fprintf(w, format, a...) }
	p("# Downstream compatibility report\n\n")
	p("**Verdict: %s** (exit %d) · tier `%s` · ref mode `%s` · started %s · wall %.0f s\n\n",
		r.Verdict, r.ExitCode, r.Tier, r.RefMode, r.Started.Format(time.RFC3339), r.WallSecs)
	if r.NullControl {
		p("> Null control: both arms are the same binary, so any difference below is instrument noise.\n\n")
	}
	p("## Arms\n\n| arm | origin | version | sha256 | wall s |\n|---|---|---|---|---|\n")
	for _, a := range r.Arms {
		p("| %s | %s | %s | %s | %.0f |\n", a.Name, cell(a.Origin), cell(a.Version), shortSHA(a.SHA256), r.ArmWall[a.Name])
	}
	if len(r.Failures) > 0 {
		p("\n## INSTRUMENT FAILURES (exit 3)\n\n")
		for _, f := range r.Failures {
			p("- %s\n", f)
		}
	}
	p("\n## Sources\n\n| corpus | repo | mode | ref | sha | state | local checkout | freshness |\n|---|---|---|---|---|---|---|---|\n")
	for _, s := range r.Sources {
		loc := "—"
		if s.Local != nil {
			loc = fmt.Sprintf("%s @%s, %d uncommitted", s.Local.Branch, shortSHA(s.Local.HEAD), s.Local.Porcelain)
		}
		p("| %s | %s | %s | %s | %s | %s | %s | %s |\n", s.Corpus, s.Repo, s.Mode, cell(s.Ref), shortSHA(s.SHA), cell(s.State), cell(loc), cell(s.Freshest))
	}
	p("\n## Summary\n\n| corpus | level | files | pkgs | pass_both | **NEWLY_FAILING** | fail_changed | timeout_cand | newly_passing | fail_same | expected_fail | timeout_base | baseline fail | of which deps unresolved | base cpu s | cand cpu s |\n")
	p("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, c := range r.Corpora {
		if c.NotChecked != "" {
			p("| %s | %s | **NOT CHECKED**: %s |||||||||||||||\n", c.Name, c.Level, cell(c.NotChecked))
			continue
		}
		s := c.States
		p("| %s | %s | %d | %d | %d | **%d** | %d | %d | %d | %d | %d | %d | %d | %d | %.0f | %.0f |\n", c.Name, c.Level, c.Files, c.Packages,
			s[statePassBoth], s[stateNewlyFailing], s[stateFailChanged], s[stateTimeoutCand], s[stateNewlyPassing], s[stateFailSame],
			s[stateExpectedFail], s[stateTimeoutBase], c.BaseFail, c.Unchecked, c.BaseSecs, c.CandSecs)
	}
	p("\n## Gate buckets (corpus × normalised code)\n\n")
	if len(r.Buckets) == 0 {
		p("No newly failing entries.\n")
	} else {
		p("| corpus | code | count | acknowledged max | status | acknowledged by | examples |\n|---|---|---|---|---|---|---|\n")
		for _, b := range r.Buckets {
			p("| %s | `%s` | %d | %d | **%s** | %s | %s |\n", b.Corpus, b.Code, b.Count, b.Max, b.Status,
				cell(strings.Join(b.Changes, "; ")), cell(strings.Join(b.Examples, " · ")))
		}
	}
	p("\n## expected-breaks.yaml entries\n\n")
	if len(r.Expected) == 0 {
		p("None.\n")
	} else {
		p("| change | corpus | code | max | count | status | stale (past expires) |\n|---|---|---|---|---|---|---|\n")
		for _, e := range r.Expected {
			p("| %s | %s | `%s` | %d | %d | %s | %v |\n", cell(e.Change), e.Corpus, e.Code, e.Max, e.Count, e.Status, e.Stale)
		}
	}
	p("\n## Newly passing (reported, never gated)\n\n")
	n := 0
	for _, row := range r.Rows {
		if row.State == stateNewlyPassing {
			if n < 50 {
				p("- %s `%s` (baseline: %s)\n", row.Corpus, row.Rel, strings.Join(row.Base.Codes, ","))
			}
			n++
		}
	}
	if n == 0 {
		p("None.\n")
	} else if n > 50 {
		p("- … and %d more (report.json)\n", n-50)
	}
	p("\n## Instrument quality\n\n- UNCODED share of candidate failure codes: %d of %d\n", r.Uncoded[0], r.Uncoded[1])
	for _, c := range r.Corpora {
		if len(c.CwdStdlibAt) > 0 {
			p("- WARNING %s: a `std/` stdlib sits in the working dir of %d entries; those checks used it instead of each binary's own\n", c.Name, len(c.CwdStdlibAt))
		}
	}
	p("\n## Write audit\n\n")
	if len(r.Audit) == 0 {
		p("Nothing was written in any materialised corpus or read local checkout during the run.\n")
	} else {
		for _, a := range r.Audit {
			p("- %s: `%s`\n", a.Scope, a.Path)
		}
	}
}

func (r *Report) renderPins(w io.Writer) {
	fmt.Fprintf(w, "# Downstream AILANG pins\n\nRead from each corpus's materialised snapshot (ref mode `%s`).\n\n", r.RefMode)
	byCorpus := map[string][]PinHit{}
	for _, h := range r.Pins {
		byCorpus[h.Corpus] = append(byCorpus[h.Corpus], h)
	}
	for _, s := range r.Sources {
		fmt.Fprintf(w, "## %s\n\n- source: %s %s @%s (%s)\n", s.Corpus, s.Repo, cell(s.Ref), shortSHA(s.SHA), cell(s.State))
		if s.Local != nil {
			fmt.Fprintf(w, "- local checkout: %s @%s, %d uncommitted (%s)\n", s.Local.Branch, shortSHA(s.Local.HEAD), s.Local.Porcelain, cell(s.Freshest))
		}
		hits := byCorpus[s.Corpus]
		sort.Slice(hits, func(i, j int) bool { return hits[i].Where < hits[j].Where })
		if len(hits) == 0 {
			fmt.Fprintf(w, "- no pin lines found\n")
		}
		for _, h := range hits {
			fmt.Fprintf(w, "- `%s`: `%s`\n", h.Where, cell(h.Text))
		}
		fmt.Fprintln(w)
	}
}
