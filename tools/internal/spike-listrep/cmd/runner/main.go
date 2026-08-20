package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	initialTrials      = 5
	subprocessDeadline = 12 * time.Minute
)

type trial struct {
	Ordinal     int      `json:"ordinal"`
	Value       float64  `json:"value"`
	Valid       bool     `json:"valid"`
	Error       string   `json:"error,omitempty"`
	Command     []string `json:"command,omitempty"`
	MaxRSSBytes int64    `json:"max_rss_bytes,omitempty"`
	RawOutput   string   `json:"raw_output,omitempty"`
}

type analysis struct {
	Valid            bool      `json:"valid"`
	Error            string    `json:"error,omitempty"`
	Candidate        []trial   `json:"candidate_operands"`
	Control          []trial   `json:"control_operands"`
	Ratios           []float64 `json:"paired_ratios,omitempty"`
	SortedRatios     []float64 `json:"sorted_ratios,omitempty"`
	Median           float64   `json:"median,omitempty"`
	MedianArithmetic string    `json:"median_arithmetic,omitempty"`
	Rerun            bool      `json:"rerun_required"`
	Pass             bool      `json:"pass"`
}

func paired(candidate, control []trial, threshold float64, op string, allowRerun bool) analysis {
	a := analysis{Candidate: candidate, Control: control}
	if len(candidate) != len(control) || len(candidate) == 0 {
		a.Error = "operand vectors must have the same non-zero length"
		return a
	}
	for i := range candidate {
		if !candidate[i].Valid || !control[i].Valid {
			a.Error = fmt.Sprintf("invalid trial at ordinal %d; sample retained at %d", i+1, len(candidate))
			return a
		}
		if control[i].Value == 0 {
			a.Error = fmt.Sprintf("zero control operand at ordinal %d", i+1)
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

func median(sortedValues []float64) (float64, string) {
	n := len(sortedValues)
	if n%2 == 1 {
		m := sortedValues[n/2]
		return m, fmt.Sprintf("sorted[%d]=%g", n/2, m)
	}
	left, right := sortedValues[n/2-1], sortedValues[n/2]
	return (left + right) / 2, fmt.Sprintf("(%g+%g)/2=%g", left, right, (left+right)/2)
}

func touchesBothSides(values []float64, threshold float64) bool {
	low, high := false, false
	for _, value := range values {
		low = low || value <= threshold
		high = high || value >= threshold
	}
	return low && high
}

func compare(value, threshold float64, op string) bool {
	switch op {
	case "<=":
		return value <= threshold
	case ">=":
		return value >= threshold
	default:
		return false
	}
}

func benchmarkCatalog() []string {
	arms := []string{"C0", "C1", "C2K8", "C2K32"}
	var cells []string
	for _, arm := range arms {
		for _, m := range []int{1024, 4096} {
			for _, l := range []int{1024, 4096, 16384} {
				cells = append(cells, fmt.Sprintf("BenchmarkListRep_B1_Branching/arm=%s/m=%d/L=%d", arm, m, l))
			}
		}
		for _, n := range []int{1600, 3200, 6400, 12800} {
			cells = append(cells, fmt.Sprintf("BenchmarkListRep_B2_LinearBuild/arm=%s/n=%d", arm, n))
		}
		for _, n := range []int{4096, 65536} {
			cells = append(cells, fmt.Sprintf("BenchmarkListRep_B3_Iteration/arm=%s/n=%d", arm, n))
		}
		for _, i := range []int{0, 1024, 2048, 4095} {
			cells = append(cells, fmt.Sprintf("BenchmarkListRep_B4_Nth/arm=%s/n=4096/i=%d", arm, i))
		}
		cells = append(cells, fmt.Sprintf("BenchmarkListRep_B5_Materialize/arm=%s/n=4096", arm))
		for _, n := range []int{4096, 65536} {
			cells = append(cells, fmt.Sprintf("BenchmarkListRep_BLEN_Length/arm=%s/n=%d", arm, n))
		}
	}
	return cells
}

func validateCell(pattern string) (string, error) {
	if !strings.HasPrefix(pattern, "^") || !strings.HasSuffix(pattern, "$") {
		return "", errors.New("cell regex must be anchored with ^ and $")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, cell := range benchmarkCatalog() {
		if re.MatchString(cell) {
			matches = append(matches, cell)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("cell regex matched %d benchmarks, want exactly 1", len(matches))
	}
	return matches[0], nil
}

type invocation struct{ Kind, Selector, Metric string }

func runTrial(ctx context.Context, ordinal int, in invocation) trial {
	trialCtx, cancel := context.WithTimeout(ctx, subprocessDeadline)
	defer cancel()
	args := []string{"-l", "go"}
	switch in.Kind {
	case "benchmark":
		args = append(args, "test", "./tools/internal/spike-listrep", "-run=^$", "-count=1", "-timeout=10m", "-benchmem", "-benchtime=1s", "-bench="+in.Selector)
	case "retained", "gcshape":
		args = append(args, "run", "./tools/internal/spike-listrep/cmd/"+in.Kind, "-arm="+in.Selector)
	default:
		return trial{Ordinal: ordinal, Error: "unknown invocation kind"}
	}
	cmd := exec.CommandContext(trialCtx, "/usr/bin/time", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := trial{Ordinal: ordinal, Command: append([]string{"/usr/bin/time"}, args...)}
	result.MaxRSSBytes = parseMaxRSS(stderr.String())
	result.RawOutput = strings.TrimSpace(stdout.String())
	if trialCtx.Err() == context.DeadlineExceeded {
		result.Error = "deadline killed subprocess"
		return result
	}
	if err != nil {
		result.Error = err.Error() + ": " + strings.TrimSpace(stderr.String())
		return result
	}
	value, err := parseMetric(in, stdout.Bytes())
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Value, result.Valid = value, true
	return result
}

func parseMaxRSS(stderr string) int64 {
	re := regexp.MustCompile(`(?m)^\s*(\d+)\s+maximum resident set size`)
	m := re.FindStringSubmatch(stderr)
	if len(m) != 2 {
		return 0
	}
	v, _ := strconv.ParseInt(m[1], 10, 64)
	return v
}

func parseMetric(in invocation, output []byte) (float64, error) {
	if in.Kind == "retained" {
		var r struct {
			BytesPerElement float64 `json:"bytes_per_element"`
		}
		if err := json.Unmarshal(output, &r); err != nil {
			return 0, err
		}
		return r.BytesPerElement, nil
	}
	if in.Kind == "gcshape" {
		var r map[string]float64
		if err := json.Unmarshal(output, &r); err != nil {
			return 0, err
		}
		v, ok := r[in.Metric]
		if !ok {
			return 0, fmt.Errorf("metric %q absent", in.Metric)
		}
		return v, nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], strings.TrimSuffix(strings.TrimPrefix(in.Selector, "^"), "$")) {
			continue
		}
		for i := 2; i+1 < len(fields); i++ {
			if fields[i+1] == in.Metric {
				return strconv.ParseFloat(fields[i], 64)
			}
		}
	}
	return 0, fmt.Errorf("metric %q not found", in.Metric)
}

func collect(ctx context.Context, in invocation, start, count int) []trial {
	results := make([]trial, 0, count)
	for i := range count {
		results = append(results, runTrial(ctx, start+i, in))
	}
	return results
}

func main() {
	kind := flag.String("kind", "benchmark", "benchmark, retained, or gcshape")
	cell := flag.String("cell", "", "one anchored benchmark regex, or arm for a main program")
	control := flag.String("control", "", "anchored control regex, or control arm")
	metric := flag.String("metric", "ns/op", "reported metric")
	threshold := flag.Float64("threshold", math.NaN(), "comparison threshold")
	op := flag.String("op", "<=", "<= or >=")
	flag.Parse()
	if *cell == "" || *control == "" || math.IsNaN(*threshold) || (*op != "<=" && *op != ">=") {
		flag.Usage()
		os.Exit(2)
	}
	if *kind == "benchmark" {
		if _, err := validateCell(*cell); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if _, err := validateCell(*control); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	ctx := context.Background()
	candidateIn, controlIn := invocation{*kind, *cell, *metric}, invocation{*kind, *control, *metric}
	candidate := collect(ctx, candidateIn, 1, initialTrials)
	controlTrials := collect(ctx, controlIn, 1, initialTrials)
	result := paired(candidate, controlTrials, *threshold, *op, true)
	if result.Valid && result.Rerun {
		candidate = append(candidate, collect(ctx, candidateIn, 6, 5)...)
		controlTrials = append(controlTrials, collect(ctx, controlIn, 6, 5)...)
		result = paired(candidate, controlTrials, *threshold, *op, false)
	}
	_ = json.NewEncoder(os.Stdout).Encode(result)
	if !result.Valid {
		os.Exit(1)
	}
}
