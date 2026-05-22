// M-EVAL-LOCAL-OBSERVABILITY M3: snapshot test for `ailang chains live`.
//
// Verifies that renderLiveChain produces an output with the expected
// columns and headers. Uses an in-memory observatory backend seeded
// with a small chain + stages so the test is hermetic.
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// TestRenderLiveChain_FormatBasic seeds a chain with two stages (one
// running fizzbuzz, one running adt_option) and verifies the rendered
// table contains the expected header columns and stage rows.
func TestRenderLiveChain_FormatBasic(t *testing.T) {
	backend, err := observatory.NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("create backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	// Use the proper CreateChain API so all columns get sensible defaults
	// (sql.NullString etc), avoiding NULL->string scan errors downstream.
	chain, err := backend.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType: observatory.ChainSourceType("eval_suite"),
		SourceRef:  "test-run",
	})
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	chainID := chain.ID

	// Seed two stages with benchmark-labeled agent_id (post-M2 format)
	db := backend.Store().DB()
	_, err = db.ExecContext(ctx, `
		INSERT INTO chain_stages (id, chain_id, stage_number, agent_id, status, turns, tokens_in, tokens_out)
		VALUES (?, ?, 1, 'eval-agent:fizzbuzz', 'running', 12, 45000, 5000),
		       (?, ?, 2, 'eval-agent:adt_option', 'running', 8, 28000, 3000)
	`, "stage-1", chainID, "stage-2", chainID)
	if err != nil {
		t.Fatalf("insert stages: %v", err)
	}

	var buf bytes.Buffer
	if err := renderLiveChain(&buf, backend, ctx, chainID, time.Now()); err != nil {
		t.Fatalf("renderLiveChain: %v", err)
	}

	out := buf.String()

	// Header line should mention the chain id (shortened to first 8 chars) and source.
	wantShortID := chainID[:8]
	if !strings.Contains(out, wantShortID) {
		t.Errorf("expected output to contain shortened chain id %q, got: %s", wantShortID, out)
	}
	if !strings.Contains(out, "eval_suite") {
		t.Errorf("expected output to contain source 'eval_suite', got: %s", out)
	}

	// Table headers must be present.
	for _, header := range []string{"#", "Benchmark / Agent", "Status", "Turns", "Tokens", "Last span"} {
		if !strings.Contains(out, header) {
			t.Errorf("expected output to contain column header %q, got: %s", header, out)
		}
	}

	// Per-stage rows: benchmark labels (from M2) flow through.
	for _, expected := range []string{
		"eval-agent:fizzbuzz",
		"eval-agent:adt_option",
		"running",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected output to contain %q, got: %s", expected, out)
		}
	}

	// Token sums (45000+5000=50000 and 28000+3000=31000) should appear.
	if !strings.Contains(out, "50000") || !strings.Contains(out, "31000") {
		t.Errorf("expected token sums 50000 and 31000 in output, got: %s", out)
	}
}

// TestChainIsTerminal verifies that the chain-completion detection
// returns true for terminal statuses and false otherwise.
func TestChainIsTerminal(t *testing.T) {
	backend, err := observatory.NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()
	db := backend.Store().DB()

	cases := []struct {
		status string
		want   bool
	}{
		{"active", false},
		{"running", false},
		{"completed", true},
		{"failed", true},
		{"cancelled", true},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			chain, err := backend.CreateChain(ctx, &observatory.ChainCreateRequest{
				SourceType: observatory.ChainSourceType("test"),
				SourceRef:  "term-" + tc.status,
			})
			if err != nil {
				t.Fatalf("create chain: %v", err)
			}
			// Bump status to the target state for this test case.
			_, err = db.ExecContext(ctx, `UPDATE execution_chains SET status = ? WHERE id = ?`, tc.status, chain.ID)
			if err != nil {
				t.Fatalf("update chain status: %v", err)
			}
			got, _ := chainIsTerminal(backend, ctx, chain.ID)
			if got != tc.want {
				t.Errorf("chainIsTerminal(status=%s) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}
