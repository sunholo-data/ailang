// Package protocol implements the ratified list-representation measurement protocol.
package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const InitialTrials = 5
const RerunTrials = 5
const SubprocessDeadline = 12 * time.Minute

type Trial struct {
	Ordinal     int      `json:"ordinal"`
	Value       float64  `json:"value"`
	Valid       bool     `json:"valid"`
	Error       string   `json:"error,omitempty"`
	Command     []string `json:"command,omitempty"`
	MaxRSSBytes int64    `json:"max_rss_bytes,omitempty"`
	RawOutput   string   `json:"raw_output,omitempty"`
}
type Analysis struct {
	Valid            bool      `json:"valid"`
	Error            string    `json:"error,omitempty"`
	Candidate        []Trial   `json:"candidate_operands"`
	Control          []Trial   `json:"control_operands"`
	Ratios           []float64 `json:"paired_ratios,omitempty"`
	SortedRatios     []float64 `json:"sorted_ratios,omitempty"`
	Median           float64   `json:"median,omitempty"`
	MedianArithmetic string    `json:"median_arithmetic,omitempty"`
	Rerun            bool      `json:"rerun_required"`
	Pass             bool      `json:"pass"`
}
type Invocation struct{ Kind, Selector, Metric string }
type Collector func(context.Context, Invocation, int, int) []Trial

func Paired(candidate, control []Trial, threshold float64, op string, allowRerun bool) Analysis {
	a := Analysis{Candidate: candidate, Control: control}
	if len(candidate) != len(control) || len(candidate) == 0 {
		a.Error = "operand vectors must have the same non-zero length"
		return a
	}
	for i := range candidate {
		if !candidate[i].Valid || !control[i].Valid {
			a.Error = fmt.Sprintf("invalid trial at ordinal %d; sample retained at %d", i+1, len(candidate))
			return a
		}
		if candidate[i].Ordinal != control[i].Ordinal {
			a.Error = fmt.Sprintf("ordinal mismatch at index %d", i)
			return a
		}
		if control[i].Value == 0 {
			a.Error = fmt.Sprintf("zero control operand at ordinal %d", candidate[i].Ordinal)
			return a
		}
		a.Ratios = append(a.Ratios, candidate[i].Value/control[i].Value)
	}
	a.SortedRatios = append([]float64(nil), a.Ratios...)
	sort.Float64s(a.SortedRatios)
	a.Median, a.MedianArithmetic = median(a.SortedRatios)
	a.Valid = true
	if allowRerun {
		a.Rerun = a.Median == threshold || touchesBothSides(a.Ratios, threshold)
	}
	a.Pass = compare(a.Median, threshold, op) && !a.Rerun
	return a
}
func median(v []float64) (float64, string) {
	n := len(v)
	if n%2 == 1 {
		m := v[n/2]
		return m, fmt.Sprintf("sorted[%d]=%g", n/2, m)
	}
	l, r := v[n/2-1], v[n/2]
	return (l + r) / 2, fmt.Sprintf("(%g+%g)/2=%g", l, r, (l+r)/2)
}
func touchesBothSides(v []float64, t float64) bool {
	lo, hi := false, false
	for _, x := range v {
		lo = lo || x <= t
		hi = hi || x >= t
	}
	return lo && hi
}
func compare(v, t float64, op string) bool {
	if op == "<=" {
		return v <= t
	}
	if op == ">=" {
		return v >= t
	}
	return false
}

func BenchmarkCatalog() []string {
	arms := []string{"C0", "C1", "C2K8", "C2K32"}
	var c []string
	for _, a := range arms {
		for _, m := range []int{1024, 4096} {
			for _, l := range []int{1024, 4096, 16384} {
				c = append(c, fmt.Sprintf("BenchmarkListRep_B1_Branching/arm=%s/m=%d/L=%d", a, m, l))
			}
		}
		for _, n := range []int{1600, 3200, 6400, 12800} {
			c = append(c, fmt.Sprintf("BenchmarkListRep_B2_LinearBuild/arm=%s/n=%d", a, n))
		}
		for _, n := range []int{4096, 65536} {
			c = append(c, fmt.Sprintf("BenchmarkListRep_B3_Iteration/arm=%s/n=%d", a, n))
		}
		for _, i := range []int{0, 1024, 2048, 4095} {
			c = append(c, fmt.Sprintf("BenchmarkListRep_B4_Nth/arm=%s/n=4096/i=%d", a, i))
		}
		c = append(c, fmt.Sprintf("BenchmarkListRep_B5_Materialize/arm=%s/n=4096", a))
		for _, n := range []int{4096, 65536} {
			c = append(c, fmt.Sprintf("BenchmarkListRep_BLEN_Length/arm=%s/n=%d", a, n))
		}
	}
	return c
}
func ValidateCell(p string) (string, error) {
	if !strings.HasPrefix(p, "^") || !strings.HasSuffix(p, "$") {
		return "", errors.New("cell regex must be anchored with ^ and $")
	}
	re, e := regexp.Compile(p)
	if e != nil {
		return "", e
	}
	var m []string
	for _, c := range BenchmarkCatalog() {
		if re.MatchString(c) {
			m = append(m, c)
		}
	}
	if len(m) != 1 {
		return "", fmt.Errorf("cell regex matched %d benchmarks, want exactly 1", len(m))
	}
	return m[0], nil
}

func RunTrial(ctx context.Context, ordinal int, in Invocation) Trial {
	tc, cancel := context.WithTimeout(ctx, SubprocessDeadline)
	defer cancel()
	args := []string{"-l", "go"}
	switch in.Kind {
	case "benchmark":
		args = append(args, "test", "./tools/internal/spike-listrep", "-run=^$", "-count=1", "-timeout=10m", "-benchmem", "-benchtime=1s", "-bench="+in.Selector)
	case "retained", "gcshape":
		args = append(args, "run", "./tools/internal/spike-listrep/cmd/"+in.Kind, "-arm="+in.Selector)
	default:
		return Trial{Ordinal: ordinal, Error: "unknown invocation kind"}
	}
	cmd := exec.CommandContext(tc, "/usr/bin/time", args...)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	r := Trial{Ordinal: ordinal, Command: append([]string{"/usr/bin/time"}, args...), MaxRSSBytes: parseMaxRSS(errout.String())}
	if tc.Err() == context.DeadlineExceeded {
		r.Error = "deadline killed subprocess"
		return r
	}
	if err != nil {
		r.Error = err.Error() + ": " + strings.TrimSpace(errout.String())
		return r
	}
	v, raw, e := parseMetric(in, out.Bytes())
	if e != nil {
		r.Error = e.Error()
		return r
	}
	r.Value = v
	r.RawOutput = raw
	r.Valid = true
	return r
}
func parseMaxRSS(s string) int64 {
	m := regexp.MustCompile(`(?m)^\s*(\d+)\s+maximum resident set size`).FindStringSubmatch(s)
	if len(m) != 2 {
		return 0
	}
	v, _ := strconv.ParseInt(m[1], 10, 64)
	return v
}
func parseMetric(in Invocation, out []byte) (float64, string, error) {
	if in.Kind == "retained" {
		var r struct {
			BytesPerElement float64 `json:"bytes_per_element"`
		}
		if e := json.Unmarshal(out, &r); e != nil {
			return 0, "", e
		}
		return r.BytesPerElement, strings.TrimSpace(string(out)), nil
	}
	if in.Kind == "gcshape" {
		// RawMessage, not float64: the gcshape report carries a STRING `arm`
		// alongside its numeric counters, so map[string]float64 fails to
		// unmarshal the whole document and the metric is never reached. That
		// made -kind=gcshape unmeasurable from the day it was written — every
		// B8 trial returned `cannot unmarshal string into Go value of type
		// float64` — and nothing caught it because no kill clause reads B8, so
		// only AC-1's completeness floor ever asks for it.
		var raw map[string]json.RawMessage
		if e := json.Unmarshal(out, &raw); e != nil {
			return 0, "", e
		}
		r := map[string]float64{}
		for k, v := range raw {
			var f float64
			if json.Unmarshal(v, &f) == nil {
				r[k] = f
			}
		}
		v, ok := r[in.Metric]
		if !ok {
			return 0, "", fmt.Errorf("metric %q absent", in.Metric)
		}
		return v, strings.TrimSpace(string(out)), nil
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 4 || !strings.HasPrefix(f[0], strings.TrimSuffix(strings.TrimPrefix(in.Selector, "^"), "$")) {
			continue
		}
		for i := 2; i+1 < len(f); i++ {
			if f[i+1] == in.Metric {
				v, err := strconv.ParseFloat(f[i], 64)
				return v, s.Text(), err
			}
		}
	}
	return 0, "", fmt.Errorf("metric %q not found", in.Metric)
}
func Collect(ctx context.Context, in Invocation, start, count int) []Trial {
	r := make([]Trial, 0, count)
	for i := range count {
		r = append(r, RunTrial(ctx, start+i, in))
	}
	return r
}
func Adjudicate(ctx context.Context, g Collector, ci, co Invocation, t float64, op string) Analysis {
	c := g(ctx, ci, 1, InitialTrials)
	d := g(ctx, co, 1, InitialTrials)
	r := Paired(c, d, t, op, true)
	if r.Valid && r.Rerun {
		c = append(c, g(ctx, ci, InitialTrials+1, RerunTrials)...)
		d = append(d, g(ctx, co, InitialTrials+1, RerunTrials)...)
		r = Paired(c, d, t, op, false)
	}
	return r
}
