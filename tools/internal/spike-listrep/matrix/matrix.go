// Package matrix defines the full LC-1 measurement matrix and its non-vacuous verdict.
package matrix

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/protocol"
)

var Arms = []string{"C0", "C1", "C2K8", "C2K32"}
var Candidates = []string{"C1", "C2K8", "C2K32"}

type Cell struct {
	Name       string              `json:"name"`
	Invocation protocol.Invocation `json:"invocation"`
	Trials     []protocol.Trial    `json:"trials,omitempty"`
}
type Encapsulation string

const (
	Unknown Encapsulation = "unknown"
	Pass    Encapsulation = "pass"
	Fail    Encapsulation = "fail"
)

type Clause struct {
	Name       string              `json:"name"`
	Analyses   []protocol.Analysis `json:"analyses"`
	Pass       bool                `json:"pass"`
	Arithmetic string              `json:"arithmetic"`
}
type CandidateVerdict struct {
	Arm           string            `json:"arm"`
	Clauses       map[string]Clause `json:"clauses"`
	Encapsulation Encapsulation     `json:"encapsulation"`
	Pass          bool              `json:"pass"`
}
type VerdictKind string

const (
	Go   VerdictKind = "GO"
	Stop VerdictKind = "STOP"
)

type Overall struct {
	Kind   VerdictKind `json:"kind"`
	Chosen string      `json:"chosen,omitempty"`
}
type Metadata struct {
	GoVersion, GOOS, GOARCH, Machine string
	Started, Ended                   time.Time
	Elapsed                          string
}
type Report struct {
	Metadata   Metadata           `json:"metadata"`
	Partial    bool               `json:"partial"`
	AC1        []Cell             `json:"ac1_cells"`
	BLen       []Cell             `json:"b_len_cells"`
	Candidates []CandidateVerdict `json:"candidates,omitempty"`
	Overall    *Overall           `json:"overall,omitempty"`
	Refusal    string             `json:"refusal,omitempty"`
}
type Collector func(context.Context, protocol.Invocation, int, int) []protocol.Trial

func Sets() (ac1, bLen []Cell) {
	for _, name := range protocol.BenchmarkCatalog() {
		c := Cell{Name: name, Invocation: protocol.Invocation{Kind: "benchmark", Selector: "^" + name + "$", Metric: "ns/op"}}
		if strings.Contains(name, "BLEN_") {
			bLen = append(bLen, c)
		} else {
			ac1 = append(ac1, c)
		}
	}
	for _, arm := range Arms {
		ac1 = append(ac1, Cell{Name: "B6/arm=" + arm, Invocation: protocol.Invocation{Kind: "retained", Selector: arm, Metric: "bytes_per_element"}}, Cell{Name: "B8/arm=" + arm, Invocation: protocol.Invocation{Kind: "gcshape", Selector: arm, Metric: "num_gc_delta"}})
	}
	return ac1, bLen
}
func Collect(ctx context.Context, cells []Cell, g Collector, progress func(int, int, time.Duration)) {
	start := time.Now()
	for i := range cells {
		cells[i].Trials = g(ctx, cells[i].Invocation, 1, protocol.InitialTrials)
		if progress != nil {
			progress(i+1, len(cells), time.Since(start))
		}
	}
}

// byName indexes cells BY POINTER so that a tie/spread rerun writes its extra
// trials back into the caller's slice — the Report therefore records all ten
// trials for any cell that was extended, which AC-1 requires.
func byName(cellSets ...[]Cell) map[string]*Cell {
	m := map[string]*Cell{}
	for _, cells := range cellSets {
		for i := range cells {
			m[cells[i].Name] = &cells[i]
		}
	}
	return m
}

// firstN returns the first n trials of a cell, or nil if it has fewer. Clauses
// pair operands BY ORDINAL, so both arms of a comparison must be cut to the
// same length: a cell extended to ten by one clause must still be read at five
// by a clause whose own partner was never extended.
func firstN(c *Cell, n int) []protocol.Trial {
	if c == nil || len(c.Trials) < n {
		return nil
	}
	return c.Trials[:n]
}

// extend tops a cell up to `want` trials, collecting ONLY the missing ones and
// numbering them contiguously after the existing batch. Collection is cached in
// the cell itself, so a second clause needing the same extension reuses it and
// no cell is ever measured twice (V2).
func extend(ctx context.Context, c *Cell, want int, g Collector) {
	if c == nil || g == nil || len(c.Trials) >= want {
		return
	}
	c.Trials = append(c.Trials, g(ctx, c.Invocation, len(c.Trials)+1, want-len(c.Trials))...)
}
func checkTrials(cells []Cell) error {
	if len(cells) == 0 {
		return errors.New("empty_result_set")
	}
	for _, c := range cells {
		n := 0
		for _, t := range c.Trials {
			if t.Valid {
				n++
			}
		}
		if n < protocol.InitialTrials {
			return fmt.Errorf("insufficient_valid_trials: %s has %d", c.Name, n)
		}
	}
	return nil
}
func bench(prefix, arm, suffix string) string { return prefix + "/arm=" + arm + suffix }

// analyse runs the doc's five-trial protocol for ONE comparison and, only on the
// predeclared tie/spread condition, performs the mandatory rerun: five further
// fresh-process trials per arm, after which the median of all ten is final.
//
// The rerun is NOT optional and NOT cosmetic. protocol.Paired with allowRerun=true
// forces Pass=false whenever the flag fires, so a driver that sets the flag and
// never collects the extra trials banks a straddling clause as a permanent FAIL —
// and for the C0 control leg it aborts the whole verdict with control_leg_failed.
// Either outcome is a STOP produced by a rerun that was never run, on a gate that
// cancels ~16 person-days. Hence: no collector + a fired rerun = a LOUD refusal,
// never a silent failure.
func analyse(ctx context.Context, m map[string]*Cell, g Collector, cand, ctrl string, t float64, op string) protocol.Analysis {
	a := protocol.Paired(firstN(m[cand], protocol.InitialTrials), firstN(m[ctrl], protocol.InitialTrials), t, op, true)
	if !a.Valid || !a.Rerun {
		return a
	}
	if g == nil {
		a.Valid = false
		a.Error = "rerun_required_but_no_collector: " + cand + " vs " + ctrl
		return a
	}
	total := protocol.InitialTrials + protocol.RerunTrials
	extend(ctx, m[cand], total, g)
	extend(ctx, m[ctrl], total, g)
	// allowRerun=false: a rerun may never cascade. The median of all ten is final.
	return protocol.Paired(firstN(m[cand], total), firstN(m[ctrl], total), t, op, false)
}
func clause(name string, aa []protocol.Analysis) Clause {
	c := Clause{Name: name, Analyses: aa, Pass: true}
	var p []string
	for _, a := range aa {
		c.Pass = c.Pass && a.Valid && a.Pass
		p = append(p, a.MedianArithmetic)
	}
	c.Arithmetic = strings.Join(p, "; ")
	return c
}

// Verdict evaluates the matrix with no collector: any clause that fires the
// tie/spread rerun refuses loudly rather than adjudicating on five trials.
func Verdict(ac1, bLen []Cell, enc map[string]Encapsulation, partial bool) (Report, error) {
	return VerdictWith(context.Background(), ac1, bLen, enc, partial, nil)
}

// VerdictWith evaluates within-arm (a,d), cross-arm (b,c), and caller-supplied (e).
// g supplies the extra trials for the protocol's predeclared tie/spread rerun; it
// may be nil, in which case a fired rerun is a refusal.
func VerdictWith(ctx context.Context, ac1, bLen []Cell, enc map[string]Encapsulation, partial bool, g Collector) (Report, error) {
	r := Report{Partial: partial, AC1: ac1, BLen: bLen}
	if partial {
		r.Refusal = "partial_run"
		return r, errors.New(r.Refusal)
	}
	if e := checkTrials(append(append([]Cell{}, ac1...), bLen...)); e != nil {
		r.Refusal = e.Error()
		return r, e
	}
	m := byName(ac1, bLen)
	for _, mp := range []int{1024, 4096} {
		a := analyse(ctx, m, g, bench("BenchmarkListRep_B1_Branching", "C0", fmt.Sprintf("/m=%d/L=16384", mp)), bench("BenchmarkListRep_B1_Branching", "C0", fmt.Sprintf("/m=%d/L=1024", mp)), 8, ">=")
		if !a.Valid {
			r.Refusal = "zero_control_operand"
			if strings.HasPrefix(a.Error, "rerun_required_but_no_collector") {
				r.Refusal = a.Error
			}
			return r, errors.New(r.Refusal)
		}
		if !a.Pass {
			r.Refusal = "control_leg_failed"
			return r, errors.New(r.Refusal)
		}
	}
	for _, arm := range Candidates {
		a := []protocol.Analysis{}
		for _, mp := range []int{1024, 4096} {
			a = append(a, analyse(ctx, m, g, bench("BenchmarkListRep_B1_Branching", arm, fmt.Sprintf("/m=%d/L=16384", mp)), bench("BenchmarkListRep_B1_Branching", arm, fmt.Sprintf("/m=%d/L=1024", mp)), 1.5, "<="))
		}
		b := []protocol.Analysis{}
		for _, n := range []int{4096, 65536} {
			b = append(b, analyse(ctx, m, g, bench("BenchmarkListRep_B3_Iteration", arm, fmt.Sprintf("/n=%d", n)), bench("BenchmarkListRep_B3_Iteration", "C0", fmt.Sprintf("/n=%d", n)), 2, "<="))
		}
		c := []protocol.Analysis{analyse(ctx, m, g, "B6/arm="+arm, "B6/arm=C0", 2.5, "<=")}
		d := []protocol.Analysis{analyse(ctx, m, g, bench("BenchmarkListRep_BLEN_Length", arm, "/n=65536"), bench("BenchmarkListRep_BLEN_Length", arm, "/n=4096"), 1.2, "<=")}
		// A candidate clause that could not be adjudicated for want of a rerun is
		// NOT a failed clause: refuse rather than record a STOP nobody measured.
		for _, aa := range [][]protocol.Analysis{a, b, c, d} {
			for _, an := range aa {
				if !an.Valid && strings.HasPrefix(an.Error, "rerun_required_but_no_collector") {
					r.Refusal = an.Error
					return r, errors.New(r.Refusal)
				}
			}
		}
		cv := CandidateVerdict{Arm: arm, Clauses: map[string]Clause{"a": clause("a", a), "b": clause("b", b), "c": clause("c", c), "d": clause("d", d)}, Encapsulation: enc[arm]}
		numeric := cv.Clauses["a"].Pass && cv.Clauses["b"].Pass && cv.Clauses["c"].Pass && cv.Clauses["d"].Pass
		if numeric && cv.Encapsulation == Unknown {
			r.Refusal = "encapsulation_unknown:" + arm
			return r, errors.New(r.Refusal)
		}
		cv.Pass = numeric && cv.Encapsulation == Pass
		r.Candidates = append(r.Candidates, cv)
	}
	var winners []CandidateVerdict
	for _, c := range r.Candidates {
		if c.Pass {
			winners = append(winners, c)
		}
	}
	if len(winners) == 0 {
		r.Overall = &Overall{Kind: Stop}
		return r, nil
	}
	sort.SliceStable(winners, func(i, j int) bool {
		ci, cj := winners[i].Clauses["c"].Analyses[0].Median, winners[j].Clauses["c"].Analyses[0].Median
		if ci != cj {
			return ci < cj
		}
		bi, bj := winners[i].Clauses["b"].Analyses[1].Median, winners[j].Clauses["b"].Analyses[1].Median
		return bi < bj
	})
	r.Overall = &Overall{Kind: Go, Chosen: winners[0].Arm}
	return r, nil
}

func Markdown(r Report) string {
	var b strings.Builder
	b.WriteString("| candidate | (a) | (b) | (c) | (d) | (e) | overall |\n|---|---|---|---|---|---|---|\n")
	for _, c := range r.Candidates {
		fmt.Fprintf(&b, "| %s | %t %s | %t %s | %t %s | %t %s | %s | %t |\n", c.Arm, c.Clauses["a"].Pass, c.Clauses["a"].Arithmetic, c.Clauses["b"].Pass, c.Clauses["b"].Arithmetic, c.Clauses["c"].Pass, c.Clauses["c"].Arithmetic, c.Clauses["d"].Pass, c.Clauses["d"].Arithmetic, c.Encapsulation, c.Pass)
	}
	return b.String()
}
