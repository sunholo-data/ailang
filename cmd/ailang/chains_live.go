// Package main: M-EVAL-LOCAL-OBSERVABILITY M3 — `ailang chains live <id>` subcommand.
//
// Single-page refreshing live view of an in-flight execution chain.
// Joins chain_stages + spans + Ollama runtime state to give per-benchmark
// live progress with stuck-vs-thinking detection.
//
// Usage:
//
//	ailang chains live <chain-id>                  # default 3s refresh
//	ailang chains live <chain-id> --interval 5     # custom refresh interval
//	ailang chains live <chain-id> --once           # render once and exit (testable)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

const (
	// stuckThreshold is how long since the last span before we flag a stage as
	// possibly stuck. Local thinking models can think for minutes between
	// spans, so this is generous.
	stuckThreshold = 300 * time.Second

	// liveDefaultRefresh is the default refresh interval for the live view.
	liveDefaultRefresh = 3 * time.Second
)

// chainsLiveCommand implements `ailang chains live <id>`.
func chainsLiveCommand() {
	fs := flag.NewFlagSet("chains live", flag.ExitOnError)
	interval := fs.Int("interval", 3, "Refresh interval in seconds")
	once := fs.Bool("once", false, "Render once and exit (for snapshot testing)")
	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains live <chain-id> [--interval N] [--once]")
		fmt.Println()
		fmt.Println("Shows live progress of an in-flight execution chain, refreshing every N seconds.")
		fmt.Println("Distinguishes 'model is thinking hard' from 'model is genuinely stuck' using span age.")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)
	refresh := time.Duration(*interval) * time.Second
	if refresh < time.Second {
		refresh = liveDefaultRefresh
	}

	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Single-shot mode: useful for snapshot testing and one-off inspection.
	if *once {
		if err := renderLiveChain(os.Stdout, backend, ctx, chainID, time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Refresh loop: handle Ctrl-C gracefully.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	startedAt := time.Now()
	for {
		clearScreen()
		if err := renderLiveChain(os.Stdout, backend, ctx, chainID, startedAt); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Printf("\nPress Ctrl-C to exit. Refreshes every %ds.\n", *interval)

		// Exit when chain completes
		if done, _ := chainIsTerminal(backend, ctx, chainID); done {
			fmt.Println("\nChain reached terminal state. Exiting.")
			return
		}

		select {
		case <-sigCh:
			fmt.Println("\n\nInterrupted. Goodbye.")
			return
		case <-ticker.C:
			// continue loop
		}
	}
}

// chainIsTerminal returns true if the chain has finished (completed/failed/cancelled).
func chainIsTerminal(backend observatory.Backend, ctx context.Context, chainID string) (bool, error) {
	chain, err := backend.GetChain(ctx, chainID, observatory.ChainReadOptions{})
	if err != nil || chain == nil {
		return false, err
	}
	switch string(chain.Status) {
	case "completed", "failed", "cancelled":
		return true, nil
	}
	return false, nil
}

// renderLiveChain writes one snapshot of the chain's live state to w.
func renderLiveChain(w io.Writer, backend observatory.Backend, ctx context.Context, chainID string, startedAt time.Time) error {
	chain, err := backend.GetChain(ctx, chainID, observatory.ChainReadOptions{IncludeStages: true})
	if err != nil {
		return fmt.Errorf("get chain: %w", err)
	}
	if chain == nil {
		return fmt.Errorf("chain not found: %s", chainID)
	}
	stages, err := backend.GetChainStages(ctx, chainID, observatory.ChainReadOptions{IncludeStages: true})
	if err == nil {
		chain.Stages = stages
	}

	now := time.Now()
	elapsed := now.Sub(startedAt).Truncate(time.Second)

	// Header: chain id, source, elapsed, GPU/Ollama state.
	ollamaName, ollamaVRAM := ollamaCurrent()
	fmt.Fprintf(w, "Chain: %s  Source: %s  Status: %s  Elapsed: %s\n",
		shortID(chain.ID), chain.SourceType, chain.Status, elapsed)
	if ollamaName != "" {
		fmt.Fprintf(w, "Ollama: %s  (VRAM %.1f GB)\n", ollamaName, ollamaVRAM)
	}
	fmt.Fprintln(w, strings.Repeat("─", 96))
	fmt.Fprintf(w, "%-4s %-32s %-12s %-7s %-9s %-22s\n",
		"#", "Benchmark / Agent", "Status", "Turns", "Tokens", "Last span")
	fmt.Fprintln(w, strings.Repeat("─", 96))

	for _, stage := range chain.Stages {
		lastSpan := lastSpanForStage(backend, ctx, chainID, stage.ID)
		var lastSpanCol string
		if lastSpan.IsZero() {
			lastSpanCol = "(no spans yet)"
		} else {
			age := now.Sub(lastSpan).Truncate(time.Second)
			lastSpanCol = age.String() + " ago"
			if age > stuckThreshold && string(stage.Status) == "running" {
				lastSpanCol = "⚠ " + lastSpanCol + " (stuck?)"
			}
		}
		tokens := int64(stage.TokensIn) + int64(stage.TokensOut)
		fmt.Fprintf(w, "%-4d %-32s %-12s %-7d %-9d %-22s\n",
			stage.StageNumber,
			truncateLive(stage.AgentID, 32),
			truncateLive(string(stage.Status), 12),
			stage.Turns,
			tokens,
			lastSpanCol,
		)
	}
	fmt.Fprintln(w, strings.Repeat("─", 96))
	return nil
}

// lastSpanForStage returns the timestamp of the most recent span for a stage,
// or zero time if none exists.
//
// Implementation note: MAX() over a TIMESTAMP column returns the raw string
// value (the SQLite driver doesn't apply timestamp conversion to aggregate
// output), so we Scan into a *string and parse manually. NULL → empty string
// → zero time (rendered as "(no spans yet)").
func lastSpanForStage(backend observatory.Backend, ctx context.Context, chainID, stageID string) time.Time {
	if stageID == "" {
		return time.Time{}
	}
	sqliteBackend, ok := backend.(*observatory.SQLiteBackend)
	if !ok {
		return time.Time{}
	}
	db := sqliteBackend.Store().DB()
	queryMaxTime := func(query string, arg string) time.Time {
		var raw sql.NullString
		if err := db.QueryRowContext(ctx, query, arg).Scan(&raw); err != nil {
			return time.Time{}
		}
		if !raw.Valid || raw.String == "" {
			return time.Time{}
		}
		// SQLite stores timestamps as RFC3339Nano with timezone offset.
		// Try a couple of common formats — the M-EVAL-LOCAL-OBSERVABILITY
		// pipeline writes "2026-05-22 21:25:31.927687417+02:00".
		for _, layout := range []string{
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999Z07:00",
			time.RFC3339Nano,
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, raw.String); err == nil {
				return t
			}
		}
		return time.Time{}
	}

	if t := queryMaxTime(`SELECT MAX(COALESCE(end_time, start_time)) FROM spans WHERE stage_id = ?`, stageID); !t.IsZero() {
		return t
	}
	// Fallback: spans may not have stage_id populated (pre-FOLLOWUP chains).
	return queryMaxTime(`SELECT MAX(COALESCE(end_time, start_time)) FROM spans WHERE chain_id = ?`, chainID)
}

// ollamaCurrent returns the name + VRAM (GB) of the currently-loaded Ollama
// model, or empty string if Ollama is not running.
func ollamaCurrent() (string, float64) {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://localhost:11434/api/ps")
	if err != nil {
		return "", 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0
	}
	var payload struct {
		Models []struct {
			Name     string `json:"name"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", 0
	}
	if len(payload.Models) == 0 {
		return "", 0
	}
	m := payload.Models[0]
	return m.Name, float64(m.SizeVRAM) / 1e9
}

// clearScreen emits ANSI escape codes to clear the terminal and home the cursor.
// Cheap and works on every modern terminal we care about (iTerm2, Terminal.app,
// xterm-compatible).
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// shortID returns the first 8 chars of an id, or the full id if shorter.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// truncate truncates s to n chars (rune-aware) with an ellipsis.
func truncateLive(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
