# M-EVAL-LOCAL-OBSERVABILITY — Sprint Plan

**Sprint ID**: M-EVAL-LOCAL-OBSERVABILITY
**Design Doc**: [m-eval-local-observability.md](./m-eval-local-observability.md)
**Target Version**: v0.22.0
**Estimated Duration**: 2.5 days (20 hours)
**Total LOC**: ~350
**Risk Level**: Low–Medium

## Goal

Close the monitoring blind spot for the 24/7 local Ollama eval rotation. Today opencode emits OTEL spans, they reach the local OTLP receiver, and **the receiver drops them due to a SQLite FOREIGN KEY constraint failure**. Fix that bug, surface the spans through the existing `ailang chains` CLI, and add a live-monitoring subcommand so we can tell "model is thinking hard" from "model is genuinely stuck" within 30 seconds.

## Why Now

Direct dependency of the M-EVAL-LOCAL-OLLAMA rotation. Without this work, 24/7 unattended runs are not trustworthy — multiple times during the 2026-05-22 investigation we couldn't tell whether a slow run was making progress or hung. The fix is small (~350 LOC) and unblocks the rotation entirely.

## Current Velocity (as of 2026-05-22)

- Recent 5 commits: 1549 insertions across 14 files (very healthy pace)
- M-TRANSITIVE-ALIAS-ENV-IMPORT just shipped: 94/110 quality score
- M-PROMPT-STDLIB-COVERAGE shipped v0.16.1 teaching prompt
- M-BYTES-TOINTS-BYTEAT added byteAt stdlib function
- Typical milestone duration: 1–2 days for ≤200 LOC

This sprint fits comfortably in 2–3 days at current pace.

## Milestones

### M1 — FK-tolerant span insert (P0 blocker)

**Estimate**: 4 hours, ~50 LOC (15 impl + 35 test)

The bug that drops everything. Without this, the other milestones are unobservable. Highest leverage in the sprint.

**Files**:
- `internal/observatory/backend_sqlite.go` (+15 LOC) — modify `createSpanWithAggregation` to NULL out `task_id`/`agent_assignment_id` when the referenced row doesn't exist
- `internal/observatory/store.go` (+5 LOC) — add `taskExists(ctx, id)` helper if one doesn't exist (or reuse existing query)
- `internal/observatory/backend_sqlite_test.go` (+30 LOC) — regression test: insert with non-existent task_id → succeeds with NULL, no error

**Acceptance criteria**:
- Test added: `TestCreateSpan_MissingTaskID_StoresWithNull` passes
- Manual verification: `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 make eval-smoke MODELS=opencode-gemma4-26b ...` produces a non-zero count in `SELECT COUNT(*) FROM spans WHERE task_id IS NULL`
- `~/.ailang/logs/server.log` no longer logs `FOREIGN KEY constraint failed` for eval-suite-emitted spans

**Dependencies**: None
**Risk**: Low — surgical change to a single function

---

### M2 — Chain-stage benchmark labels (P1)

**Estimate**: 2 hours, ~20 LOC

When 4 stages are running concurrently, `ailang chains view` shows "Stage 1: eval-agent [running]" × 4. We need "Stage 1: eval-agent fizzbuzz [running]" so we can tell them apart.

**Files**:
- `internal/eval_harness/agent_runner_multi.go` (+5 LOC) — record `benchmark_id` as a stage attribute when registering the eval-agent stage
- `cmd/ailang/chains_view.go` (+10 LOC) — surface the benchmark_id attribute in the stage line
- Existing tests cover; spot-check by running the smoke tier and inspecting `ailang chains view <id>`

**Acceptance criteria**:
- `ailang chains view <id>` shows each running eval-agent stage with its benchmark name
- `ailang chains diagnose <id>` no longer flags missing session ID as an issue (or is fixed at the same time)

**Dependencies**: None (independent of M1)
**Risk**: Low — additive change

---

### M3 — `ailang chains live <id>` subcommand (P1)

**Estimate**: 8 hours, ~180 LOC (150 impl + 30 test)

The new live-monitoring view. Single-page output that refreshes every 3 seconds, joining chain stages + spans + opencode session DB + Ollama runtime state.

**Files**:
- `cmd/ailang/chains_live.go` (+150 LOC, new) — subcommand implementation
- `cmd/ailang/main.go` (+3 LOC) — register subcommand
- `cmd/ailang/chains_live_test.go` (+30 LOC) — basic snapshot test of formatting

**Format (single-page text, refreshes every 3s)**:

```
Chain: 17a81788  Source: eval_suite/agent  Elapsed: 23m 18s  GPU: 27% (gemma4:26b)
─────────────────────────────────────────────────────────────────────────────────
Stage   Benchmark              Status    Turns  Tokens   Last span (s ago)
1       fizzbuzz               running     12   47K       3
2       adt_option             running      8   31K      12
3       balanced_parens        running      0    0       ⚠ 540 (TTFT — possibly stuck)
4       recursion_fibonacci    running     14   52K       1
─────────────────────────────────────────────────────────────────────────────────
Press Ctrl-C to exit. Refreshes every 3s.
```

**Key queries it runs each refresh**:
- `chain_stages` for the active stages and their benchmark_id attribute (from M2)
- `spans` for max(end_time) per stage_id and aggregate `tokens_in+tokens_out`
- `opencode.db` `message` for live turn count
- `/api/ps` on Ollama for current model + last activity

**Acceptance criteria**:
- `ailang chains live <id>` runs without error
- Output matches the layout above
- Refreshes every 3 seconds (configurable via `--interval`)
- Exits cleanly when chain transitions to completed
- Shows "⚠ stuck" indicator when last_span_age > 300s

**Dependencies**: M1 (spans must actually land) + M2 (benchmark labels must exist)
**Risk**: Medium — terminal UI work, depends on M1 + M2

---

### M4 — launchd plist + docs + integration verification (P1)

**Estimate**: 6 hours, ~150 LOC (50 plist + 80 docs + 20 integration test/cleanup)

Stand up the eval-server at boot so the OTLP receiver is always live, and document the local-Ollama eval workflow end-to-end.

**Files**:
- `~/Library/LaunchAgents/dev.ailang.server.plist` (+50 lines, new) — RunAtLoad + KeepAlive + log paths
- `docs/docs/guides/evaluation/local-ollama.md` (+80 LOC, new) — full user-facing guide: env setup, ollama config, opencode config, recommended `-parallel`, monitoring with `ailang chains live`, troubleshooting
- `tools/eval-rotation.sh` (+20 LOC, optional helper) — wraps the canonical `make eval-smoke` command from the M-EVAL-LOCAL-OLLAMA design doc

**Acceptance criteria**:
- `launchctl load ~/Library/LaunchAgents/dev.ailang.server.plist` brings up the server
- After reboot, `make services-status` shows server running
- `docs/docs/guides/evaluation/local-ollama.md` exists and renders correctly on docusaurus build
- `ailang chains live <id>` end-to-end works against a fresh smoke-tier run

**Dependencies**: M1, M2, M3 all complete
**Risk**: Low–Medium — launchd has macOS-version quirks

---

## Day-by-Day Plan

| Day | Hours | Work |
|-----|-------|------|
| **Day 1 AM** | 4 hrs | M1: FK-tolerant span insert + regression test |
| **Day 1 PM** | 4 hrs | M1 manual verification (run smoke tier with OTLP set, confirm spans land) + M2: chain-stage labels |
| **Day 2 AM** | 4 hrs | M3 part 1: scaffolding for `chains live` subcommand + per-stage span query |
| **Day 2 PM** | 4 hrs | M3 part 2: refresh loop, opencode.db join, Ollama state join, stuck-detection indicator |
| **Day 3 AM** | 4 hrs | M4: launchd plist + docs + integration verification |
| **Day 3 PM** | 2 hrs | Polish + CHANGELOG + final commit + ship |

**Total: 22 hours over 3 days.** Buffer included (sprint plan's 20-hour estimate is the cleanroom number).

## Success Metrics

| Metric | Target | How measured |
|---|---|---|
| Span count after a 17-benchmark smoke tier | ≥100 | `SELECT COUNT(*) FROM spans` post-run |
| FK constraint failures in server.log | 0 | grep server.log for "FOREIGN KEY constraint failed" |
| `ailang chains view` shows benchmark per stage | Yes | Visual inspection during active run |
| `ailang chains live` refreshes in ≤5s | Yes | Stopwatch on terminal |
| Stuck-vs-thinking distinguishable in 30s | Yes | Time from "is it stuck?" → answer |
| `make eval-smoke` 24h rotation lands clean | Yes | Run for a full day; check observatory.db growth |
| Doc page renders on docusaurus | Yes | `make docs-build` |
| Test coverage on new code | ≥70% | `go test -cover ./internal/observatory/... ./cmd/ailang/...` |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| `chains live` terminal UI is fiddly (cursor positioning, clearing) | Medium | Start with simple newline-clear approach; upgrade to ANSI escape codes only if needed |
| launchd plist quirks (macOS Sonoma vs Sequoia vs Tahoe) | Medium | Test on the actual rig (M4 Max, macOS 26.2 Tahoe); document version requirements |
| OTLP exporter batching delays make live view feel slow | Low | Configure `WithBatcher` interval down to 2s during eval runs |
| FK fix breaks existing coordinator-driven span inserts | Low | Regression test specifically against coordinator-task span insertion |

## Dependencies & Open Questions

- **Decided (no need to escalate)**:
  - Fix shape: option B (NULL out missing FK) — eval-suite tasks are ephemeral
  - launchd at boot: yes
  - Auto-set OTLP env var in eval-suite if local receiver is running: yes (added at the eval-suite entry point)
  - `chains live` UI: auto-refreshing text (not full TUI) — simplest, works everywhere

- **Deferred to follow-up sprint**:
  - `ailang trace list --local` flag (lets local query work without GCP creds)
  - Retention policy for old spans (auto-delete after 30 days)
  - Whether to register `eval_runs` as a separate parent table

## Acceptance / Done Definition

This sprint is DONE when:

1. All 4 milestones merged
2. Manual verification: full smoke tier (17 benchmarks) at p=4 produces ≥100 spans in observatory.db
3. `ailang chains live <chain-id>` shows live progress for 4 concurrent stages with benchmark names
4. launchd plist installed; server survives a reboot
5. Local-ollama eval guide rendered on docusaurus
6. CHANGELOG entry written
7. Move design doc + sprint plan from `planned/v0_22_0/` to `implemented/v0_22_0/`

## Handoff

After approval, sprint-executor takes over with the JSON progress file at `.ailang/state/sprints/sprint_M-EVAL-LOCAL-OBSERVABILITY.json`.
