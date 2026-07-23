# M-EVAL-FMT-WEAKMODEL-AB — M5 (hard-set re-run) results

**Status**: COMPLETE — NULL (again), for a diagnosable reason. Pairs the frozen
[`…-M5-hardset-prereg.md`](m-eval-fmt-weakmodel-ab-M5-hardset-prereg.md).
**Date**: 2026-07-23.
**Runs**: 90 (45 OFF + 45 ON), `claude-haiku-4-5` agent mode, cloud (no rig contention).

## Verdict

> **NEUTRAL — true null, treatment delivered — BUT the drift regime the re-run targeted did not
> materialise on the current stack. The cheap-cloud path to testing fmt is exhausted.**

| Metric | OFF | ON | Δ (ON−OFF) |
|---|---|---|---|
| Pass rate | 43/45 = **95.6%** | 42/45 = **93.3%** | **−2.2pp**, 95% CI [−11.7, +7.2] |
| Compile-stuck (`compile_ok=false`) | 1/45 | 0/45 | ≈0 in both arms |

- **Treatment integrity: PASSED** — OFF banked **0** fmt events; ON banked **70** across **45/45** runs
  (100% delivery). The void clause does not fire; this is a genuine null, not an unevaluable run.
- **Primary null**: delta CI includes 0 and |point| < the frozen +0.10 threshold → NEUTRAL.

## Why null — a selection error, honestly recorded

The M5 drift-set was chosen from **historical** banked haiku pass rates (17–73%). On the **current**
stack (v0.30.0 + prompt v0.16.3), haiku **near-aces the same set**: 8 of 9 benchmarks at 5/5, 95.6%
overall. The historical low rates were stale/mixed-version aggregates; they no longer hold. So the
re-run landed **back near ceiling** — the exact M4 failure mode — because the set was selected on stale
data rather than current pass rates. This is a methodological miss on the selection step, not a
property of the treatment.

The one benchmark with movement, `csv_to_json_converter` (4/5 → 2/5), went the **wrong** way and is
noise at n=5.

## The informative finding (why this still concludes something)

**Compile-stuck ≈ 0 in BOTH arms (1/90).** fmt's mechanism is "canonical formatting removes syntax
drift → fewer compile-stuck spirals." On the current stack, **`claude-haiku-4-5` essentially does not
spiral on AILANG** — on any benchmark set cheaply selectable through the `claude` executor. There is no
Claude-executable drift regime left for fmt to act in. M4 and M5 now agree from two different angles:
the treatment is delivered cleanly and is inert **where there is no drift**, and haiku produces no drift.

## Conclusion → the only path that can test the hypothesis

The genuine syntax-drift regime is the **weak on-device / small models** (qwen spirals with
`compile_error`s and timeouts in the live rotation). Those models cannot run through the `claude`
executor, and the fmt hook is Claude-Code-only (`.claude/settings.json` + `--settings`). Therefore:

1. **No cheap cloud A/B can settle this.** Every drift-prone model runs through `opencode`/`pi`/`motoko`,
   not `claude`, so the hook never reaches it. Confirmed twice over now.
2. **The next step is fmt DELIVERY for the `opencode` executor** — a `postToolUse` plugin that runs
   `ailang fmt --write` on Edit/Write of `.ail` files, mirroring the existing opencode **microRAG**
   plugin (same `postToolUse` interface, TypeScript). This single build unlocks the A/B for **any**
   opencode-driven model:
   - **local qwen** (free, slow, rig contention) — the on-device thesis, OR
   - **a cheap drift-prone OpenRouter model** (`opencode-or-*`, metered but cheap + fast, no rig
     contention) — provided it is first verified to actually drift on AILANG (M4/M5's lesson: verify
     current drift before selecting).
3. `pi` and `motoko` would each need their own delivery (pi hook mechanism; motoko a loop step) if the
   A/B is to span all three local harnesses.

## Cost paid / saved

M5 cost ~90 fast cloud runs and produced a null — but it **de-risked the local build correctly**: it
proved the treatment delivers and is inert without drift, and that no cheaper vehicle exists. Building
local fmt delivery is now justified as the only way forward, not a guess.

## Local fmt-delivery feasibility (verified 2026-07-23 — read before scoping)

The fmt hook is **Claude-Code-only**: `FmtHookMode.Apply` writes `<ws>/.claude/settings.json` and the
`claude` CLI loads it via `--settings`. All three local harnesses are **separate agent subprocesses**
whose edit loop is internal — the hook never reaches them. There is **no cheap, zero-risk path**; each
needs its own delivery mechanism, and each touches infra the **live rotation depends on**:

| Harness | Delivery mechanism | Cost / risk |
|---|---|---|
| **opencode** | Install a `postToolUse` plugin (microRAG template exists in-repo but is **NOT installed** — no `plugins` field in `~/.config/opencode/opencode.jsonc`, unverified in the eval subprocess) + register it in the **global** opencode.jsonc | First-time integration; the global config is **shared with the running rotation** — a plugin error could break rotation opencode runs |
| **motoko** | Modify our fork `/Users/voightkampff/dev/arniwesth/motoko_agent` (TS/bun) to run `ailang fmt --write` post-edit + write the `fmt_hook_events` sink; rebuild + redeploy the `motoko` binary | Separate repo + build/deploy cycle; the rotation **runs that binary** — a bad build breaks it |
| **pi** | Unverified — pi's post-tool hook surface not investigated | Unknown |

Common to all: gate on an env var (mirror `MicroragMode.ApplyToEnv` → a new `FmtHookMode.ApplyToEnv`
setting `AILANG_FMT_HOOK_ENABLED`), default off so the rotation is untouched; write events to the
existing `<ws>/.claude/fmt_hook_events.jsonl` sink so `ReadFmtHookSink` banks identical
`fmt_hook_events` and the M-void-clause integrity check works.

**Recommendation**: this is a real per-harness feature, not a one-turn build, and it touches the live
rig. Scope it as its own sprint (M6) — pick ONE harness (motoko is the flagship local model and our own
fork; opencode unlocks OpenRouter too), build the delivery + env gating + sink logging behind a default-off
flag, verify treatment integrity on a drift-verified model, THEN run the A/B. Do NOT bolt it onto the
live rotation's shared config/binary without an isolated verification first.
