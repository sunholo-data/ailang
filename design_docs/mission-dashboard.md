# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-10 ~20:30 local (iteration 171)

## Now
- **v0.33.0** · `origin/dev` `eecb4d011` · Standing SonarCloud red (`#615`) unchanged — non-required,
  inherited across all 5 analysed commits.
- ✅ **`#618` routed** (ollama `/v1` streaming idle-timeout, P0 — 80 motoko runs and ~74.6 GPU-h lost
  over 43 days, accelerating). Doc quorum-cleared, sprint plan + state JSON landed. **Not executed
  this slot, deliberately** — the planner's refutations reshaped it minutes before.
- 🔴 **Sharpest find: the design would have SHIPPED INERT.** `ollamaCallContext` applies a 300s
  `context.WithTimeout` at `step.go:266` — 17 lines *above* the `/v1` branch — so setting
  `Client.Timeout: 0` fixes nothing. Three designer passes and four quorum reviews read the branch;
  the deadline is applied above it. Found by the **planner**, not the quorum.
- 🔴 **Second find: a would-have-shipped regression.** The streamed path has no `Reasoning` and no
  Hermes tool-call recovery (controls 6 / 8), i.e. it would have traded a timeout bug for the
  **disengagement** failure mode. New M3; estimate 1–2d → 3d.
- ✅ **Feasibility settled empirically** — rig probe under the GPU lock: ollama 0.32.1 `/v1` streams
  tool calls fine on qwen3.6. The 13,077-B capture is committed as a fixture, so M3's capture task
  was done before the sprint began.

## In flight / queued
- **`#618` M1** (idle/TTFT/deadline `ReadCloser` + watchdog + RoundTripper, 380 LOC, no GPU) → M2
  (owns S8 + the inert-deadline fix) → M3 (parity) → M4 (rig, **GPU**).
- **Lane B1 M4** (`#517`) — ADTs, recursion/size budgets, `TypeApp`; slipped one slot, resumes after.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2`.

## Next
**`#618` M1**, then M2 — M2 is where the two planner refutations actually get closed, so it is the
milestone to review hardest.

## Loop + routing
Controller **opus** · designer **`claude:claude-fable-5`** (3 bounded passes; pointer advances →
next `codex:gpt-5.6-sol`) · planner **opus** (`derive-planner-lane.sh` → `opus
fail-closed:planner-lane-field-missing`) · executor/evaluator **not fired** (no code milestone).
Metered **$0.165** (quorum R1 $0.074 + R2 $0.091) against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.

Full record: charter `## STATUS … ITERATION 171` + `v1-mission-log.md`.
