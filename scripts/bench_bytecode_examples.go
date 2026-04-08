//go:build ignore
// +build ignore

// bench_bytecode_examples.go: wall-clock speedup comparison for the
// runnable corpus. For each parity-clean example, runs both
// `ailang run` and `ailang run --bytecode` N times, takes the
// best-of-N wall clock, and reports a speedup ratio.
//
// Caveat: process startup (~50-100ms on darwin/arm64) dominates short
// examples. Cases where eval_ms < StartupFloor are flagged as
// startup-dominated so you don't mistake them for "the VM isn't helping".
//
// Usage:
//
//	go run ./scripts/bench_bytecode_examples.go                 # full corpus
//	go run ./scripts/bench_bytecode_examples.go --iters 5       # more samples
//	go run ./scripts/bench_bytecode_examples.go --only fib      # filter by name
//	go run ./scripts/bench_bytecode_examples.go --min-ms 100    # drop quick examples
//	go run ./scripts/bench_bytecode_examples.go --markdown      # markdown table
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StartupFloor is the approximate wall-clock cost of an `ailang run`
// invocation with no user code (process start + pipeline cold-start).
// Examples that run under this threshold on the evaluator side are
// almost entirely startup, so their "speedup" is noise.
const StartupFloor = 80 * time.Millisecond

type Result struct {
	File       string
	Caps       []string
	Entry      string
	EvalMs     float64
	VMMs       float64
	Speedup    float64
	StartupDom bool
	Note       string
}

func main() {
	iters := flag.Int("iters", 3, "Samples per backend (best-of-N is reported)")
	only := flag.String("only", "", "Only run files whose basename contains this substring")
	minMs := flag.Float64("min-ms", 0, "Drop examples whose evaluator best-of-N is below this (ms)")
	timeout := flag.Duration("timeout", 60*time.Second, "Per-sample timeout")
	mdOut := flag.Bool("markdown", false, "Emit markdown table to stdout")
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

	// Known-bad examples: EVAL_SKIP set + the uuid non-determinism case.
	// For a speedup run we want both paths to complete cleanly; anything
	// that EVAL_SKIPs in verify_bytecode_parity is also eval_ms == -1 here.
	skip := map[string]string{
		"ai_effect.ail":            "EVAL_SKIP (needs API key)",
		"ai_image_generation.ail":  "EVAL_SKIP (needs API key)",
		"game_npc_dialogue.ail":    "EVAL_SKIP (needs API key)",
		"structured_ai_basic.ail":  "EVAL_SKIP (needs API key)",
		"structured_ai_schema.ail": "EVAL_SKIP (needs API key)",
		"exit_code.ail":            "intentional exit(42)",
		"uuid.ail":                 "non-deterministic output",
	}

	bin := ailangBinary()

	var results []Result
	for _, f := range files {
		base := filepath.Base(f)
		if reason, bad := skip[base]; bad {
			results = append(results, Result{File: f, Note: "skipped: " + reason})
			continue
		}
		r := benchOne(bin, f, *iters, *timeout)
		if *minMs > 0 && r.EvalMs > 0 && r.EvalMs < *minMs {
			continue
		}
		results = append(results, r)
	}

	// Sort by speedup descending, skipped/failed at the end.
	sort.SliceStable(results, func(i, j int) bool {
		ai, aj := results[i].Speedup, results[j].Speedup
		if ai == 0 && aj == 0 {
			return results[i].File < results[j].File
		}
		if ai == 0 {
			return false
		}
		if aj == 0 {
			return true
		}
		return ai > aj
	})

	if *mdOut {
		emitMarkdown(results)
	} else {
		emitText(results)
	}
}

func benchOne(bin, file string, iters int, timeout time.Duration) Result {
	src, err := os.ReadFile(file)
	if err != nil {
		return Result{File: file, Note: "read error: " + err.Error()}
	}
	caps := detectCaps(string(src))
	entry := detectEntry(string(src))

	evalBest := bestOfN(bin, file, caps, entry, false, iters, timeout)
	if evalBest < 0 {
		return Result{File: file, Caps: caps, Entry: entry, Note: "evaluator failed"}
	}
	vmBest := bestOfN(bin, file, caps, entry, true, iters, timeout)
	if vmBest < 0 {
		return Result{File: file, Caps: caps, Entry: entry, EvalMs: ms(evalBest), Note: "vm failed"}
	}

	r := Result{
		File:    file,
		Caps:    caps,
		Entry:   entry,
		EvalMs:  ms(evalBest),
		VMMs:    ms(vmBest),
		Speedup: float64(evalBest) / float64(vmBest),
	}
	if evalBest < StartupFloor {
		r.StartupDom = true
	}
	return r
}

func bestOfN(bin, file string, caps []string, entry string, bytecodeMode bool, iters int, timeout time.Duration) time.Duration {
	best := time.Duration(-1)
	for i := 0; i < iters; i++ {
		d := runOnce(bin, file, caps, entry, bytecodeMode, timeout)
		if d < 0 {
			return -1
		}
		if best < 0 || d < best {
			best = d
		}
	}
	return best
}

func runOnce(bin, file string, caps []string, entry string, bytecodeMode bool, timeout time.Duration) time.Duration {
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
	args = append(args, "--relax-modules", "--quiet", "--no-print", file)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	if err != nil {
		return -1
	}
	return elapsed
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func ailangBinary() string {
	for _, p := range []string{"bin/ailang", "./ailang"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	if p, err := exec.LookPath("ailang"); err == nil {
		return p
	}
	return "go run ./cmd/ailang"
}

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
			if strings.HasPrefix(rest[i:], "()") {
				return name
			}
		}
	}
	return "main"
}

func emitText(rs []Result) {
	var meaningful, startup, skipped, failed int
	for _, r := range rs {
		switch {
		case r.Note != "":
			if strings.HasPrefix(r.Note, "skipped") {
				skipped++
			} else {
				failed++
			}
		case r.StartupDom:
			startup++
		default:
			meaningful++
		}
	}

	fmt.Printf("Bytecode VM vs Evaluator — wall-clock on %d examples\n", len(rs))
	fmt.Printf("  %d meaningful, %d startup-dominated, %d skipped, %d failed\n", meaningful, startup, skipped, failed)
	fmt.Printf("  startup floor: %s (ratios below this are process-start noise)\n\n", StartupFloor)

	fmt.Printf("%-40s  %10s  %10s  %10s\n", "Example", "eval_ms", "vm_ms", "speedup")
	fmt.Printf("%s\n", strings.Repeat("-", 76))
	for _, r := range rs {
		base := filepath.Base(r.File)
		if r.Note != "" {
			fmt.Printf("%-40s  %s\n", base, r.Note)
			continue
		}
		mark := ""
		if r.StartupDom {
			mark = " †"
		}
		fmt.Printf("%-40s  %10.1f  %10.1f  %9.2fx%s\n", base, r.EvalMs, r.VMMs, r.Speedup, mark)
	}
	if startup > 0 {
		fmt.Printf("\n† startup-dominated (eval_ms < %s); VM speedup is noise here.\n", StartupFloor)
	}
}

func emitMarkdown(rs []Result) {
	fmt.Println("| Example | eval_ms | vm_ms | Speedup | Note |")
	fmt.Println("|---|---:|---:|---:|---|")
	for _, r := range rs {
		base := filepath.Base(r.File)
		if r.Note != "" {
			fmt.Printf("| %s | — | — | — | %s |\n", base, r.Note)
			continue
		}
		note := ""
		if r.StartupDom {
			note = "startup-dominated"
		}
		fmt.Printf("| %s | %.1f | %.1f | %.2fx | %s |\n", base, r.EvalMs, r.VMMs, r.Speedup, note)
	}
}
