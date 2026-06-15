# M-MOTOKO-OLLAMA-LOOP-CONVERGENCE: Make the Motoko Agent Loop Terminate Against Local Ollama

**Status**: Implemented (v0.25.x) — see **RESOLUTION** below
**Target**: v0.24.x
**Priority**: P1 — the config wiring shipped ([M-MOTOKO-LOCAL-OLLAMA](../../implemented/v0_24_0/m-motoko-local-ollama.md)) but motoko still produces **zero completions** on the rig; this is the blocker that doc's premise missed
**Estimated**: 1.5–2 days (diagnosis-first; fix scope depends on Phase 2 finding)
**Dependencies**: [M-MOTOKO-LOCAL-OLLAMA](../../implemented/v0_24_0/m-motoko-local-ollama.md) (config + models.yml + executor wiring — all shipped), [M-EVAL-LOCAL-OLLAMA](./m-eval-local-ollama.md) (rig operational)

## Supersedes a False Premise

[M-MOTOKO-LOCAL-OLLAMA](../../implemented/v0_24_0/m-motoko-local-ollama.md) shipped on the assumption that *"the gap is purely configuration."* It was not. The config is in place and motoko **does** run against local Ollama — but the agent loop never terminates:

- `session_nested_records_*.md` is **3,242 lines** for a benchmark whose correct answer is ~15 lines of AILANG.
- The model **generated a valid solution** (it sits at the tail of the session log) — yet the loop kept going.
- Multiple sessions end in **`step budget`** exhaustion, not a clean "task complete".
- Across the session's runs: **0/N completions**, with orphaned `bun` subprocesses left behind after each killed run.

That predecessor's **"Key Risk: Tool-Call Compatibility"** section called this exact failure mode in advance (thinking-token bleed, Ollama tool-call format, agent loop not converging). This doc promotes that risk from a footnote to the primary problem and turns it into a diagnosis-then-fix plan.

## RESOLUTION (2026-06-15)

**Fixed and validated end-to-end.** The fizzbuzz benchmark now passes 1/1 in
agent mode against `motoko-local-qwen3-5-35b-a3b-mxfp8` (local Ollama, $0),
**14 turns / 13 tool calls**, stdout exact-match, clean idiomatic AILANG output
(`export func main() -> () ! {IO}`, `letrec`, `show`/`println`). The loop
converges; no step-budget hang.

### The ranked hypotheses below were WRONG

The loop did not hang on a *completion-signal mismatch* (H1), thinking-token
bleed (H2), tool-call malformation (H3), or edit livelock (H4). The model
"generated a valid solution but kept looping" because **its tool calls were
never executed at all** — so its file-writes never landed, the verifier never
saw a passing solution, and the loop correctly kept retrying until the step
budget. The fix was not in motoko's loop logic; it was a chain of provider-side
and infra blockers, fixed in this order (each unblocked the next):

1. **Executor: `--headless` missing** → motoko launched the interactive TUI
   and hung. Fixed in `internal/executor/motoko/motoko.go` (commit 46884c46).
2. **Executor: `ENV_PORT=0` ephemeral vs static `cfg.url :8080`** → backend
   unreachable. Pinned `ENV_PORT=8080` (same commit). Orphaned env-servers on
   :8080 must be reaped before each run (operational note, not a code fix).
3. **Provider routing: `ollama/` prefix unrecognized** → `GuessProvider`
   matched `ollama:` but not `ollama/`, so the slash form fell through to the
   generic `vendor/model` → OpenRouter check and then "cannot determine
   provider". Fixed: `GuessProvider` now claims BOTH prefixes
   (`internal/ai/config.go`). motoko's *shipped* config (`ollama/qwen3.5:…`)
   works unchanged.
4. **Tool calling hard-rejected** → `internal/ai/ollama/step.go` still carried
   the M-AI-TOOL-LOOP (v0.17.0) "tool calling not supported" stub. Replaced
   with a full native implementation (advertise `req.Tools` → `ollamaapi.Tools`,
   native `tool` role + `tool_call_id`, thread assistant `ToolCalls`, parse
   `resp.Message.ToolCalls`, `FinishReason="tool_calls"`). **This is the fix
   that actually closed the loop** — once tools executed, motoko converged.
5. **`400 invalid model name`** → the routing prefix wasn't stripped before the
   Ollama API call. Added `bareModel()` (strips `ollama:`/`ollama/`, preserves
   the model's own `:tag`), applied to the Step and Generate paths.

### Key correction to the architecture model

The integration fix is **purely AILANG-side** (`internal/ai/ollama/` +
`internal/ai/config.go`). The `motoko` binary is a thin shim →
`scripts/run-agent.sh` → the **system `ailang` binary on PATH**; there is no
vendored ailang build inside motoko_agent. So a clean motoko install gets the
working integration the moment its `ailang` is ≥ the release carrying this fix —
**no motoko_agent source change is required for the ollama-native path.** PR #39
(`reenable_local_models`) on the fork is an *orthogonal* contribution for the
OpenAI-compatible local-endpoint path (it only strips `openrouter/`/`openai/`
prefixes and adds `OPENAI_BASE_URL` precedence); it does not touch the
`ollama/` path and is not needed for ollama-native convergence.

### Methodology note

The hang was diagnosed by running `motoko --headless` **directly** (not through
the eval harness, which buried every failure as a generic "terminated without
emitting run_summary"). Each direct run surfaced the next concrete stderr error.
The harness should surface motoko stderr on non-convergence — tracked separately.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval determinism unchanged (harness loop only) |
| A2: Replayability | +1 | A terminating loop is replayable; a step-budget-hang is not |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | Single-GPU serial |
| A7: Machines First | +2 | Unlocks zero-cost local **agent**-mode evals + cross-harness (opencode vs motoko) comparison the rig currently can't produce |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Local motoko evals at $0 once they terminate; today they cost GPU-hours for nothing |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +2 | Replaces silent step-budget hangs + orphaned `bun` PIDs with clean termination and subprocess reaping |
| A12: System Boundary | 0 | Boundary already made explicit by the predecessor doc |

**Net Score: +6** → **Decision: Proceed** (no −1 on A1/A3/A4/A7)

## Problem Statement

Motoko's agent loop, against Qwen 3.5 35B-A3B served by local Ollama, **generates correct code but never recognizes it is done**. It loops on its own output until it hits `max_steps` and is killed by the step budget. Two observable symptoms:

1. **Non-convergence** — the loop emits valid AILANG early, then keeps iterating (3k+ line session logs for trivial tasks). The completion signal motoko's loop is waiting for is never produced — or is produced but not recognized.
2. **Orphaned subprocesses** — a killed/hung run leaves `bun` (and `motoko_agent`/`opencode`) child processes running. `pkill eval-suite` alone does not reap them, so the rig accumulates zombie inference clients that contend for the GPU.

Both block local motoko evals entirely. opencode and pi converge against the same Ollama model + same benchmarks, so this is **motoko-harness-specific**, not a model or Ollama-server defect.

## Root-Cause Hypotheses (ranked)

The predecessor's Key Risk section enumerates these; ranked here by likelihood given the "generates a valid solution but keeps looping" evidence:

1. **Completion-signal mismatch (most likely).** Motoko's loop terminates on a specific tool call / verifier result ("done"). Qwen-via-Ollama emits its final answer in a *shape* motoko doesn't recognize as terminal — so the loop re-prompts indefinitely. The vLLM-served `local` profile may format the final tool call differently than Ollama's `/v1/chat/completions` proxy.
2. **Thinking-token bleed.** `<think>...</think>` blocks leak into the tool-call channel despite `enable_thinking:false` (which Ollama's proxy may silently no-op, exactly as the predecessor warned for `chat_template_kwargs`), corrupting the stop-condition parser so it never matches "done".
3. **Tool-call JSON malformation.** Qwen's Ollama Modelfile chat template emits tool calls motoko's AILANG-side parser rejects; motoko treats the rejection as "not done, retry" and loops.
4. **Edit-retry livelock.** `edit_mode: hashline` line-addressed diffs fail to apply against the model's output, and the loop retries the same failing edit until the step budget ends.

These are not mutually exclusive. Phase 1 distinguishes them empirically rather than guessing.

## Solution Design: Diagnose, then Fix

### Phase 1 — Instrument & Diff (the load-bearing step)

Capture one full motoko session JSONL for the **simplest** benchmark (`fizzbuzz`) against `motoko-local-qwen3-5-35b-a3b-mxfp8`, and an opencode run of the **same model + same benchmark**. opencode converges; motoko does not. The delta between the two transcripts localizes the failure.

```bash
# motoko (hangs) — capture the full session log, kill at first step-budget hit
ailang eval-suite --models motoko-local-qwen3-5-35b-a3b-mxfp8 \
  --benchmarks fizzbuzz --langs ailang --output /tmp/motoko_diag --parallel 1
# opencode (converges) — the control
ailang eval-suite --agent --models opencode-qwen3-5-35b-a3b-mxfp8 \
  --benchmarks fizzbuzz --langs ailang --output /tmp/opencode_diag --parallel 1
```

Inspect the motoko session JSONL for: (a) does a "done"/verifier tool call ever appear? (b) is there `<think>` text in any tool-call payload? (c) do edit operations succeed or repeatedly fail? Answering these picks the hypothesis.

### Phase 2 — Characterize

Classify the non-convergence into exactly one of: **(a)** never emits a terminal tool call, **(b)** emits a terminal call motoko's parser rejects, **(c)** thinking-token bleed corrupts the stop check, **(d)** edit-retry livelock. The fix in Phase 3 is selected by this classification — we do not fix speculatively.

### Phase 3 — Targeted Fix (scoped by Phase 2)

| Phase 2 finding | Fix |
|-----------------|-----|
| (a) never emits terminal call | Relax motoko's stop-condition to also terminate on "tests pass + no further edits proposed", or inject an explicit `finish` tool the model is prompted to call |
| (b) terminal call rejected | Add a tool-call-format adapter normalizing Ollama's output to motoko's expected schema |
| (c) thinking bleed | Strip `<think>` blocks server-side before the stop check; confirm whether `enable_thinking:false` actually reaches the Modelfile template |
| (d) edit livelock | Fall back to `edit_mode: hashline` → full-file-replace when N consecutive hashline edits fail to apply |

### Phase 4 — Subprocess Reaping (operational, independent of root cause)

Regardless of the convergence fix, hung runs must not orphan `bun`. Add a process-group kill (kill the motoko subprocess **group**, not just the PID) so `pkill eval-suite` / a watchdog reaps `bun`, `motoko_agent`, and `opencode` children together. This is the `A11: Structured Failure` win and prevents GPU-contending zombies on the rig.

## Files to Modify

| File | Change | Est. LOC |
|------|--------|---------:|
| `internal/executor/motoko/motoko.go` | Process-group spawn + reap (Phase 4); stop-condition hook (Phase 3, if a/c) | 40–80 |
| `motoko_agent/.motoko/config/ollama/config.json` | `enable_thinking`/tool-format tuning per Phase 2 | 5–15 |
| `internal/eval_harness/` | Diagnostic transcript capture flag (if not already emitted) | 10–20 |
| (motoko_agent repo) | Tool-call adapter or stop-condition change (Phase 3, if b/d) — **cross-repo** | TBD |

**Cross-repo caveat:** like M-COORD-TAG-ROUTING-LASTMILE, part of the fix may live in the separate `motoko_agent` repo. The AILANG-side half (process-group reaping, diagnostic capture, config) ships here; any motoko-internal loop change is a tracked follow-up PR against that repo.

## Acceptance Criteria

1. `ailang eval-suite --models motoko-local-qwen3-5-35b-a3b-mxfp8 --benchmarks fizzbuzz --langs ailang` **terminates cleanly** (loop exits on completion, not step-budget exhaustion).
2. ≥1 smoke benchmark passes end-to-end with a clean session JSONL showing a terminal tool call (not a 3k-line step-budget hang).
3. A killed/hung motoko run leaves **zero orphaned** `bun`/`motoko_agent`/`opencode` subprocesses (verified via `pgrep` after `pkill eval-suite`).
4. The Phase 1 motoko-vs-opencode transcript diff is captured in the implementation report so the root cause is documented, not just patched.

## Out of Scope

- Reaching opencode-parity pass rates — this doc only requires *termination*; tuning motoko's prompt/verifier for local-inference accuracy is a follow-on.
- Adding motoko-local to the nightly rotation — gated on Criterion 1 + 3 holding for a full smoke run.
- Other Ollama models through motoko — the fix is harness-level and reusable; add model entries in a follow-on.

## Related Documents

- [M-MOTOKO-LOCAL-OLLAMA](../../implemented/v0_24_0/m-motoko-local-ollama.md) — predecessor: shipped the config/wiring this doc depends on; its "purely configuration" premise is corrected here.
- [M-EVAL-LOCAL-OLLAMA](./m-eval-local-ollama.md) — the local rig + opencode harness this uses as the convergence control.
