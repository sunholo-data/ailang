//go:build ignore
// +build ignore

// verify_bytecode_parity.go is the M-BYTECODE-2D M6 acceptance harness: it
// runs every file in examples/runnable/ under both the tree-walking evaluator
// (the existing `ailang run`) and the bytecode VM path (`ailang run
// --bytecode`) and reports per-file parity (exit code + stdout byte-equal).
//
// The evaluator is treated as ground truth. Any divergence is categorized:
//
//   - MATCH         both backends produced identical stdout + exit code
//   - EVAL_SKIP     evaluator itself fails the file; we can't compare
//   - VM_BRIDGE     VM path failed due to an explicit bridge-scope error
//     ("not yet supported", "closure", "ADT", etc.)
//   - VM_COMPILE    VM path failed during compile/lower (not bridgeable)
//   - VM_RUNTIME    VM path raised a runtime error the evaluator didn't
//   - DIVERGE       both paths exited 0 but stdout differs
//
// Usage:
//
//	go run ./scripts/verify_bytecode_parity.go               # text summary + stats
//	go run ./scripts/verify_bytecode_parity.go --json        # JSON report to stdout
//	go run ./scripts/verify_bytecode_parity.go --markdown    # markdown table to stdout
//	go run ./scripts/verify_bytecode_parity.go --only <glob> # filter by filename substring
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusMatch     Status = "MATCH"
	StatusEvalSkip  Status = "EVAL_SKIP"
	StatusVMBridge  Status = "VM_BRIDGE"
	StatusVMCompile Status = "VM_COMPILE"
	StatusVMRuntime Status = "VM_RUNTIME"
	StatusDiverge   Status = "DIVERGE"
)

type Result struct {
	File       string        `json:"file"`
	Status     Status        `json:"status"`
	Caps       []string      `json:"caps,omitempty"`
	Entrypoint string        `json:"entry,omitempty"`
	EvalExit   int           `json:"eval_exit"`
	VMExit     int           `json:"vm_exit"`
	EvalStdout string        `json:"eval_stdout,omitempty"`
	VMStdout   string        `json:"vm_stdout,omitempty"`
	VMError    string        `json:"vm_error,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	Duration   time.Duration `json:"-"`
}

func main() {
	jsonOut := flag.Bool("json", false, "Emit JSON report to stdout")
	mdOut := flag.Bool("markdown", false, "Emit markdown table to stdout")
	only := flag.String("only", "", "Only run files whose basename contains this substring")
	parallel := flag.Int("parallel", 6, "Max concurrent example runs")
	timeout := flag.Duration("timeout", 60*time.Second, "Per-example timeout")
	flag.Parse()

	files, err := filepath.Glob("examples/runnable/*.ail")
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no examples found under examples/runnable/\n")
		os.Exit(1)
	}
	sort.Strings(files)

	if *only != "" {
		filtered := files[:0]
		for _, f := range files {
			if strings.Contains(filepath.Base(f), *only) {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	bin := ailangBinary()

	// Concurrent driver with a bounded semaphore.
	sem := make(chan struct{}, *parallel)
	results := make([]Result, len(files))
	var wg sync.WaitGroup
	for i, f := range files {
		i, f := i, f
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = verifyOne(bin, f, *timeout)
		}()
	}
	wg.Wait()

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return
	}
	if *mdOut {
		emitMarkdown(results)
		return
	}
	emitText(results)
}

func ailangBinary() string {
	// Prefer a pre-built binary for speed (~30-50x faster than go run).
	for _, p := range []string{"bin/ailang", "./ailang"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	// Fallback: use ailang from PATH.
	if p, err := exec.LookPath("ailang"); err == nil {
		return p
	}
	return "go run ./cmd/ailang"
}

// verifyOne runs one file under both backends and categorizes the outcome.
func verifyOne(bin, file string, timeout time.Duration) Result {
	start := time.Now()
	res := Result{File: file}

	content, err := os.ReadFile(file)
	if err != nil {
		res.Status = StatusEvalSkip
		res.Reason = fmt.Sprintf("read file: %v", err)
		return res
	}
	src := string(content)
	res.Caps = detectCaps(src)
	res.Entrypoint = detectEntry(src)

	// Evaluator run — ground truth.
	evalStdout, _, evalExit := runOne(bin, file, res.Caps, res.Entrypoint, false, timeout)
	res.EvalExit = evalExit
	res.EvalStdout = evalStdout

	if evalExit != 0 {
		res.Status = StatusEvalSkip
		res.Reason = fmt.Sprintf("evaluator itself failed (exit %d)", evalExit)
		res.Duration = time.Since(start)
		return res
	}

	// Bytecode VM run.
	vmStdout, vmStderr, vmExit := runOne(bin, file, res.Caps, res.Entrypoint, true, timeout)
	res.VMExit = vmExit
	res.VMStdout = vmStdout

	if vmExit != 0 {
		res.VMError = oneLine(vmStderr)
		switch {
		case strings.Contains(vmStderr, "not yet supported"),
			strings.Contains(vmStderr, "M-BYTECODE-2E"),
			strings.Contains(vmStderr, "closure"),
			strings.Contains(vmStderr, "TaggedValue"):
			res.Status = StatusVMBridge
			res.Reason = "bridge scope (M-BYTECODE-2E)"
		case strings.Contains(vmStderr, "compile"),
			strings.Contains(vmStderr, "lower"):
			res.Status = StatusVMCompile
			res.Reason = "compile/lower failure"
		default:
			res.Status = StatusVMRuntime
			res.Reason = "VM runtime error"
		}
		res.Duration = time.Since(start)
		return res
	}

	if vmStdout == evalStdout {
		res.Status = StatusMatch
	} else {
		res.Status = StatusDiverge
		res.Reason = "stdout differs"
	}
	res.Duration = time.Since(start)
	return res
}

// runOne invokes ailang on `file` with the given caps and entrypoint.
// When bytecodeMode is true, adds --bytecode. --quiet is always set to keep
// stderr status lines out of the comparison.
func runOne(bin, file string, caps []string, entry string, bytecodeMode bool, timeout time.Duration) (stdout, stderr string, exit int) {
	args := []string{"run"}
	if bytecodeMode {
		args = append(args, "--bytecode")
	}
	if len(caps) > 0 {
		args = append(args, "--caps", strings.Join(caps, ","))
	}
	if entry != "" && entry != "main" {
		args = append(args, "--entry", entry)
	}
	args = append(args, "--relax-modules", "--quiet", file)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se

	err := cmd.Run()
	stdout = so.String()
	stderr = se.String()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			exit = -1
		}
		return
	}
	return
}

// detectCaps mirrors scripts/verify_examples.go's capability sniffer but
// returns only the subset we care about for parity (no AI — bridge eval
// calls won't happen in strict parity for obvious reasons, but we still
// pass the cap through so the example doesn't crash on a missing handler).
func detectCaps(src string) []string {
	caps := []string{}
	want := []struct {
		name string
		hint []string
	}{
		{"IO", []string{"import std/io", "! {IO", ", IO"}},
		{"FS", []string{"import std/fs", "! {FS", ", FS"}},
		{"Net", []string{"import std/net", "! {Net", ", Net"}},
		{"Clock", []string{"import std/clock", "! {Clock", ", Clock"}},
		{"Rand", []string{"import std/rand", "! {Rand", ", Rand"}},
		{"Env", []string{"import std/env", "! {Env", ", Env"}},
		{"Debug", []string{"import std/debug", "! {Debug", ", Debug"}},
		{"Process", []string{"import std/process", "! {Process", ", Process"}},
		{"Stream", []string{"import std/stream", "! {Stream", ", Stream"}},
	}
	for _, w := range want {
		for _, h := range w.hint {
			if strings.Contains(src, h) {
				caps = append(caps, w.name)
				break
			}
		}
	}
	return caps
}

// detectEntry picks an exported nullary function if main is absent.
func detectEntry(src string) string {
	if strings.Contains(src, "export func main") {
		return "main"
	}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "export func ") {
			continue
		}
		rest := strings.TrimPrefix(line, "export func ")
		if i := strings.Index(rest, "("); i > 0 {
			name := rest[:i]
			// Zero-arg?
			if strings.HasPrefix(rest[i:], "()") {
				return name
			}
		}
	}
	return "main"
}

func oneLine(s string) string {
	// Skip known noise lines (telemetry warnings, MOD010 relaxed path notices,
	// etc.) and return the first line that looks like an actual error.
	noise := []string{
		"OTLP endpoint",
		"relaxed module path",
		"MOD010",
		"⚠",
		"stdlib version mismatch",
		"Warning:",
	}
	var pick string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		skip := false
		for _, n := range noise {
			if strings.Contains(trim, n) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		pick = trim
		break
	}
	if pick == "" {
		pick = strings.TrimSpace(s)
	}
	if len(pick) > 200 {
		pick = pick[:200] + "..."
	}
	return pick
}

func emitText(rs []Result) {
	// Counters.
	tally := map[Status]int{}
	for _, r := range rs {
		tally[r.Status]++
	}
	total := len(rs)

	fmt.Printf("M-BYTECODE-2D M6 — bytecode parity report (%d examples)\n\n", total)
	// Ordered summary line.
	order := []Status{StatusMatch, StatusDiverge, StatusVMBridge, StatusVMCompile, StatusVMRuntime, StatusEvalSkip}
	for _, s := range order {
		if tally[s] > 0 {
			fmt.Printf("  %-10s %3d  (%.1f%%)\n", s, tally[s], pct(tally[s], total))
		}
	}
	fmt.Println()

	// Non-match details.
	anyBad := false
	for _, s := range order {
		if s == StatusMatch {
			continue
		}
		var bucket []Result
		for _, r := range rs {
			if r.Status == s {
				bucket = append(bucket, r)
			}
		}
		if len(bucket) == 0 {
			continue
		}
		anyBad = true
		fmt.Printf("── %s ──\n", s)
		for _, r := range bucket {
			fmt.Printf("  %s", filepath.Base(r.File))
			if r.Reason != "" {
				fmt.Printf("  — %s", r.Reason)
			}
			fmt.Println()
			if r.VMError != "" {
				fmt.Printf("      %s\n", r.VMError)
			}
			if r.Status == StatusDiverge {
				fmt.Printf("      eval: %q\n", headN(r.EvalStdout, 120))
				fmt.Printf("      vm:   %q\n", headN(r.VMStdout, 120))
			}
		}
		fmt.Println()
	}
	if !anyBad {
		fmt.Println("  all matched — full parity.")
	}
}

func emitMarkdown(rs []Result) {
	tally := map[Status]int{}
	for _, r := range rs {
		tally[r.Status]++
	}
	total := len(rs)
	match := tally[StatusMatch]
	fmt.Printf("# M-BYTECODE-2D M6 Parity Report\n\n")
	fmt.Printf("**Parity rate**: %d/%d (%.1f%%) of examples produce byte-identical stdout under `--bytecode` and the evaluator.\n\n", match, total, pct(match, total))

	fmt.Println("| Status | Count | % |")
	fmt.Println("|---|---:|---:|")
	order := []Status{StatusMatch, StatusDiverge, StatusVMBridge, StatusVMCompile, StatusVMRuntime, StatusEvalSkip}
	for _, s := range order {
		if tally[s] == 0 {
			continue
		}
		fmt.Printf("| %s | %d | %.1f%% |\n", s, tally[s], pct(tally[s], total))
	}
	fmt.Println()

	// Non-match table.
	hasBad := false
	fmt.Println("## Non-matching examples")
	fmt.Println()
	fmt.Println("| Example | Status | Reason / Error |")
	fmt.Println("|---|---|---|")
	for _, s := range []Status{StatusDiverge, StatusVMBridge, StatusVMCompile, StatusVMRuntime, StatusEvalSkip} {
		for _, r := range rs {
			if r.Status != s {
				continue
			}
			hasBad = true
			detail := r.Reason
			if r.VMError != "" {
				detail = r.VMError
			}
			fmt.Printf("| %s | %s | %s |\n", filepath.Base(r.File), r.Status, mdEscape(detail))
		}
	}
	if !hasBad {
		fmt.Println("| _(all examples matched)_ | - | - |")
	}
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) * 100.0 / float64(d)
}

func headN(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
