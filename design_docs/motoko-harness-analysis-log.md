# Motoko Harness Analysis Log

An append-only log of motoko-harness eval failure analyses. The goal is iterative
self-improvement: each analysis characterizes *why* motoko underperforms vs other
coding harnesses (pi, opencode) on the same model/benchmarks, classifies the fix
lever, and records what observability we have vs need. When we have ~3+ entries and
a repeatable method, codify into a `motoko-analyzer` skill.

## Analysis template (use for every entry)

```
### YYYY-MM-DD — <data source>
- Headline: motoko AILANG X% (+ per-lang); vs pi Y%, opencode Z%. Coverage: N/total.
- Failure taxonomy (count): <category> = n ...
- Root cause: <the dominant failure + hypothesis>
- Lever classification (pick): HARNESS-ERROR | AILANG-INTEGRATION | PROMPT |
  AILANG-LANGUAGE | NEW-PACKAGE-EXTENSION | MODEL-CAPABILITY
- Observability: HAVE <…> | MISSING <…>
- Next action: <…>
- Prior-action status: <closes which earlier item>
```

**Lever definitions:**
- **HARNESS-ERROR** — motoko/eval plumbing bug (PATH, env, step budget, crash). Fix the harness.
- **AILANG-INTEGRATION** — the AILANG↔model glue (e.g. `internal/ai/ollama` tool-calling, provider routing). Fix AILANG's integration code.
- **PROMPT** — motoko's system/agent prompt doesn't elicit the right behaviour (e.g. doesn't compel tool use on small models). Fix the prompt (a `motoko_ext_*` package or SYSTEM.md).
- **AILANG-LANGUAGE** — the model writes plausible code but AILANG's compiler/semantics reject it where a fix to the *language* (or docs) would help. Fix AILANG.
- **NEW-PACKAGE-EXTENSION** — a missing AILANG-aware capability (docs, μRAG, verification) the agent needs. Build a `motoko_ext_*`.
- **MODEL-CAPABILITY** — the model just can't (logic errors on correct-compiling code). Not fixable by us short of a better model.

---

### 2026-06-17 — overnight OS rotation (os-rolling, qwen3.6, 3 trials)

- **Headline:** motoko AILANG **26%** (Py 70% / JS 89% / Go 79%) vs **pi 96%** / **opencode 79%** on AILANG. Coverage: motoko ~6/39 benchmarks (late start — failing until the ~22:20 PATH/key fix), pi/opencode ~32/39. Numbers are early/partial but directionally clear.
- **Failure taxonomy** (14 AILANG failures / 19 runs):
  - `api_error` **12** — of which **10 = "non-agentic result: 1 turn, 0 tool calls, 0 code"**, 1 = `step budget exhausted`, 1 = `motoko CLI not found` (pre-fix residual).
  - `logic_error` **2** (model wrote compiling-but-wrong AILANG).
- **Root cause:** motoko's ollama tool-calling is **unreliable** — qwen3.6 frequently produces **0 tool calls** (one prose turn, no `WriteFile`/`RunTests`), so no solution is written (`code_len=0`, `turns=None`). This is **language-agnostic** (motoko is below pi/opencode on Python/JS/Go too), and motoko fails benchmarks pi/opencode pass (symbolic_diff, type_unify, recursion_fibonacci) → it's the *harness*, not the language or benchmark difficulty.
- **Lever classification:** **AILANG-INTEGRATION (primary)** + **PROMPT (secondary)**. NOT AILANG-language, NOT a new extension, NOT (mostly) model-capability. opencode/pi don't hit this because they use their own battle-tested ollama tool-calling; motoko routes through AILANG's `internal/ai/ollama` (tool-calling only enabled this session) which qwen3.6 doesn't reliably trigger.
- **Observability: HAVE** — per-run `error_category`, `stderr` (carries the "0 tool calls" signature), `code`, `compile_ok/runtime_ok/stdout_ok`, cross-harness pass rates. **MISSING** — for *failing* runs, `agent_turns`/`finish_reason` are null and `code` is empty; we do **not** retain the motoko **session JSONL** (the raw model responses), so we can't yet see *why* tool calls are 0 (qwen emitting Hermes/XML `<function>` tool blocks the native parser misses? vs pure prose? vs tools not advertised?). **This is the #1 tooling gap to close before we can fix it.**
- **Next action:**
  1. **Capture observability** — retain the motoko `session_*.jsonl` (or at least the raw first model response) on eval failure, so we can inspect qwen's actual tool-call output.
  2. From that, root-cause the 0-tool-calls: (a) malformed/Hermes-XML tool calls → add tolerant parsing in `internal/ai/ollama/step.go`; (b) tools not advertised / prompt too weak → strengthen tool advertisement + motoko's prompt to compel tool use on small models.
  3. Re-measure on the rotation; expect AILANG to jump if 0-tool-calls is fixed (10/14 failures).
- **Prior-action status:** closes the "is the integration working?" question (it runs, but tool-calling is unreliable) and the PATH/key harness fix (done; 1 residual CLI-not-found failure pre-fix). Supersedes the assumption that motoko's low AILANG = model can't write AILANG (it mostly never gets to write any).

#### Root cause CONFIRMED via pi's source (the reference harness)
pi = `@mariozechner/pi-coding-agent` → `@mariozechner/pi-ai`. pi-ai has **no native
ollama provider** — it drives local ollama through the **OpenAI-compatible
`/v1/chat/completions`** endpoint (`openai-completions` provider, baseURL
`localhost:11434/v1`), using OpenAI-style `tools`/`tool_calls`. qwen3.6 emits tool
calls reliably over `/v1` (ollama's compat layer normalizes the model's tool-call
format) → pi 96%.

motoko routes through AILANG `internal/ai/ollama`, which uses ollama's **native
`/api/chat`** Tools API (`github.com/ollama/ollama/api`). qwen3.6 does not reliably
emit *native* tool calls over `/api/chat` → 0 tool calls → 26%. (Ironic: motoko's
own `.motoko/config/ollama/config.json` already sets `openai_base_url:
localhost:11434/v1`, but the AILANG ollama provider ignores it and uses native.)

**Concrete fix (highest-value next work item):** make AILANG drive ollama tool-calling
over the OpenAI-compat `/v1/chat/completions` endpoint (like pi/opencode), instead of
native `/api/chat`. Either (a) add a `/v1` tool path to `internal/ai/ollama`, or
(b) route `ollama/…` models through AILANG's OpenAI provider with baseURL
`localhost:11434/v1`. This is the first design-doc→sprint→execute item for the mission.

---

### 2026-06-17 — FIX LANDED: M-OLLAMA-V1-TOOLCALLING (mission item #2)

- **Action:** implemented approach (b) — in `internal/ai/ollama/step.go`, when `req.Tools`
  is present, delegate to an `openai.Client` pointed at `<ollama-host>/v1` (dummy key;
  ollama ignores auth), reusing AILANG's battle-tested OpenAI tool path. Gated by
  `AILANG_OLLAMA_NATIVE_TOOLS=1` (opt-in fallback to the old native `/api/chat` path for
  A/B). ~6 LOC + 2 tests (`TestStep_ToolsViaOpenAICompat` default path,
  `TestStep_ToolsAdvertisedAndParsed_Native` opt-in path). No import cycle.
- **Live result (chain `c6409fd7`):** motoko-local-qwen3-6 on fizzbuzz/AILANG →
  **4 turns, 3 tool calls**, compile ✓ runtime ✓ stdout ✓, **1/1 pass**. Directly
  reverses the baseline failure signature ("1 turn, 0 tool calls, 0 code"). The 0-tool-calls
  root cause is closed: qwen3.6 emits tool calls reliably over `/v1`.
- **Lever classification (confirmed):** AILANG-INTEGRATION — fixed in AILANG glue, no
  language/prompt/model change needed to recover agentic behaviour.
- **Prior-action status:** closes the #1 fix item from the 2026-06-17 entry (route ollama
  tool-calling over `/v1`). Expect the rotation AILANG number to climb from 26% as
  coverage fills — **re-measure on the next OS rotation** to quantify the lift across the
  full benchmark set (single-benchmark proof here; rotation gives the aggregate).
- **Next action:** observe the next rotation's motoko AILANG %; if still gapped vs pi 96%,
  the residual is mission items #3 (tolerant parse — likely now unnecessary) / #4 (prompt)
  / #5 (convergence). Lands as: **commit to `dev`** (AILANG glue, per routing rule).

---

### 2026-06-17 — POST-FIX rotation (os-rolling, qwen3.6) — AGGREGATE LIFT CONFIRMED

- **Data source:** 20 fresh motoko runs in `os-rolling` after the /v1 fix (mtime > 07:45),
  vs the archived pre-fix baseline in `_prefix-baseline-motoko-20260617/` (79 runs).
- **Headline:** motoko AILANG **5/5 = 100%** post-fix (vs **5/19 = 26%** pre-fix). Other
  langs: Go 6/6, JS 5/6, Py 3/3. Partial sample (rotation still filling) but the AILANG
  jump is unambiguous and matches the single-benchmark proof.
- **Failure taxonomy (post-fix, 1 failure / 20):** the lone failure is JS `api_call_json`
  → `runtime_error` "JavaScript execution timed out" at **6 turns** (model wrote a hanging
  http server). This is **MODEL-CAPABILITY / logic**, NOT the harness 0-tool-calls bug.
  AILANG runs now show healthy multi-turn loops (3–8 turns, finish_reason=stop, code
  written, compile/runtime/stdout ✓).
- **Root cause status:** the AILANG-INTEGRATION 0-tool-calls failure mode (10/14 of the
  original failures) is **eliminated**. The failure class has shifted from "never wrote
  code" to "wrote code that's logically wrong" — a fundamentally healthier regime, and the
  one pi/opencode also live in.
- **Lever classification:** prior dominant lever (AILANG-INTEGRATION) **closed**. New
  residual lever = **MODEL-CAPABILITY** (qwen3.6 logic errors) — not fixable by us short of
  a stronger model or better teaching/prompt.
- **Observability gap found (this cycle's pick):** the per-run JSON shows
  `agent_tool_calls = None` even though the agent made tool calls — the chain DB has the
  count (`chains view`: "3 tool calls") but `RunMetrics` has **no field** to receive
  `executor.Result.ToolCallCount`, so it's dropped at `cmd/ailang/eval_benchmark.go:278`.
  Tool-call count is THE signature metric of this whole investigation ("0 tool calls") yet
  it's invisible in the rolling data. → backlog item #1 (observability), scoped below.
- **Prior-action status:** **closes mission item #2** (/v1 tool-calling) — aggregate lift
  confirmed, not just the single fizzbuzz. Pre-fix baseline preserved for the before/after.
- **Next action:** surface `agent_tool_calls` end-to-end (M-MOTOKO-OBS-TOOLCALLS) so the
  next analysis can quantify tool-use directly from rolling JSON without chain queries.

---

### 2026-06-17 — FIX LANDED: M-MOTOKO-OBS-TOOLCALLS (mission item #1, observability)

- **Action:** wired `executor.Result.ToolCallCount` → `RunMetrics.AgentToolCalls`
  (`json:"agent_tool_calls,omitempty"`) end-to-end: `internal/eval_harness/metrics.go`
  (new field) + `cmd/ailang/eval_benchmark.go:279` (file writer) +
  `internal/eval_analysis/types.go` (analysis `RunMetrics` + `ResultJSON` export struct +
  converter) + `internal/eval_analysis/loader_chains.go` (chain loader maps `stage.ToolCalls`).
  ~6 field/assignment lines across 4 files. Value was already captured at every hop except
  the per-run JSON, where it was silently dropped. Standard-mode rows unaffected (omitempty).
- **Tests:** `TestRunMetrics_AgentToolCalls` (serialize/round-trip/omitempty) +
  extended `TestStageToResult` (asserts `stage.ToolCalls:7 → AgentToolCalls==7`, the real
  chain-loader path). `go test ./internal/eval_harness/... ./internal/eval_analysis/...` green.
- **Live file-writer confirmation:** deferred to the next OS rotation tick (which uses the
  freshly-installed binary) to avoid starving the *currently-running* rotation's ollama —
  manual motoko runs while the rotation is mid-flight produced GPU-contention failures (see
  edge case below). The file-writer hop is a 1-line assignment adjacent to the proven-working
  `AgentTurns` mapping, reading the verified-populated `result.ToolCallCount`.
- **Observed edge case (logged, NOT this item):** under concurrent ollama load, motoko can
  exit with `finish_reason=tool_calls and no run_summary` (api_error, turns/code null) — the
  model's final action was a tool call but motoko terminated without emitting a run_summary.
  This is a motoko **convergence/robustness** gap (mission backlog: convergence / parsing),
  surfaces only under contention, and is distinct from the closed 0-tool-calls bug. Flag for
  a future cycle if the next rotation shows it outside contention.
- **Lever classification:** AILANG-INTEGRATION (eval-rig observability plumbing).
- **Prior-action status:** **closes mission item #1** (retain motoko tool-call observability).
  The literal "session JSONL on failure" concern is largely moot post-/v1 (failing runs now
  retain code/turns/finish_reason/stderr); the one genuinely-dropped signal — the tool-call
  **count** — is now surfaced per-run. Backlog reprioritized: convergence robustness rises.

---

### 2026-06-17 — INCIDENT + FIX: /v1 contention hang & rig-lock enforcement

- **Incident:** the OS rotation wedged for ~2h. A motoko run hung 1h54m with the bun
  subprocess + `ailang run --ai ollama` alive but **ollama idle (0 models loaded)**. Killing
  the subprocess tree unstuck the parent rotation.
- **Root cause (two faults, confirmed):**
  1. **The `/v1` switch made it possible.** The hung run used the new OpenAI-compat `/v1`
     default (no `AILANG_OLLAMA_NATIVE_TOOLS=1` anywhere). The `/v1` delegation built its
     client with `http.DefaultClient` (**Timeout: 0**) and uses the **non-streaming** Step
     path (`io.ReadAll`). On the single shared GPU, a concurrent request triggered an ollama
     model reload that stalled the open connection → the read blocked **forever**. The native
     `/api/chat` path streams chunk-by-chunk, so it surfaces a dropped stream quickly — which
     is why the user recalled "lots of hangs before we moved to ollama endpoints." Their
     hypothesis was correct.
  2. **The contention was unguarded.** The shell `rig-lock.sh` existed but was opt-in; an
     ad-hoc `ailang eval-suite` (mine, verifying the obs change) took no lock and collided
     with the rotation.
- **Fixes (both AILANG → `dev`):**
  - **`/v1` HTTP timeout** (commit `63fc63e0`): bound the delegation client with
    `http.Client{Timeout: ollamaV1Timeout()}` (default 300s, `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`
    override; `0` disables). A stalled stream now fails fast. Tests:
    `TestStep_ToolsViaOpenAICompat_TimesOut`, `TestOllamaV1Timeout`.
  - **Native rig-lock in `eval-suite`** (M-RIG-LOCK-ENFORCE): new `internal/riglock` (Go
    port of `rig-lock.sh`, same dir/staleness); `eval-suite` acquires NoWait and **fails fast
    with the holder identity** if the rig is busy. `--dry-run`/`--no-rig-lock` bypass;
    `rig-lock.sh` exports `AILANG_RIG_LOCK_HELD=1` so launchd wrappers don't double-lock.
    Verified live: held lock → `exit 1` "rig is busy — PID … holds the rig lock".
- **Lever classification:** AILANG-INTEGRATION (ollama HTTP robustness) + HARNESS-ERROR
  (missing mutual exclusion on the shared GPU).
- **Lesson enshrined:** the safety must live **in the command we already run**, not in a
  script to remember — the user's explicit requirement. Manual rig runs now surface a
  conflict immediately; the timeout is defense-in-depth if contention ever slips through.
- **Prior-action status:** hardens mission item #2 (/v1) which introduced the hang surface;
  no revert needed (the /v1 win — AILANG 26%→100% — stands, now bounded + serialized).

---

### 2026-06-17 — STUB-FAILURE DIAGNOSIS + OBSERVABILITY FIX (M-MOTOKO-OBS-TRANSCRIPT)

- **Data:** 209 new-binary motoko runs. AILANG 38/48 (79%) — now **beats opencode (72%)**,
  approaching pi (88%). Go 98% / JS 96% / Py 98%. **Zero 0-tool-call runs** (the /v1 fix's
  failure mode stays closed).
- **Investigated the residual gap (user challenge: "is it definitely the prompt?"). It is
  NOT.** Initial read ("motoko under-iterates, fix the prompt") was a turn-count correlation,
  not proof. Digging into the actual submitted code:
  - **9 of 10 AILANG failures submit the byte-identical 112-char placeholder** (the seeded
    `solution.ail` stub, `// TODO: Add your solution code below`), all at turns=2/tools=1/
    finish=stop. 5 *different* problems can't yield the identical stub by chance.
  - Mechanism: the eval seeds `${ws}/benchmark/solution.ail` with that placeholder
    (`agent_runner_multi.go:138-168`) and reads it back (`:330`). `code`=placeholder ⟹
    motoko made 1 tool call and **never wrote a valid solution**, then stopped. Passing runs
    DO capture real code (avg 5.1 turns) → not a blanket capture bug; these runs truly
    produced nothing. pi passes the *same* benchmarks by grinding **9–41 turns**.
  - **Could not determine the precise cause** (wrong write path? non-write call then quit?
    silent write fail?) because the executor parsed the session JSONL, counted tool calls,
    and **discarded their content**; the workspace JSONL is deleted post-run.
    `agent_transcript=None` for motoko AND pi.
- **Fix (mission item #1, observability — M-MOTOKO-OBS-TRANSCRIPT):** `parser.go` now
  accumulates a compact transcript (tool name + truncated write path/content per call + the
  `done` output) into `res.Transcript` → flows to `agent_transcript` in the result JSON. The
  data was already decoded; it was just thrown away. Verified output:
  `tool_call: WriteFile {"path":"f.ail","content":"export func answer()…"}`. Bounded
  (400-char arg cap + 1 MB field cap); test `TestParseSessionJSONL_Success` asserts content.
- **Lever classification:** HARNESS-ERROR (observability gap). Root cause of the *failures*
  still TBD — that's the point: we now retain the data to determine it next cycle.
- **Lesson enshrined:** "we need enough information always" — do NOT attribute a root cause
  from a metric correlation (turn counts) when the deciding evidence (the transcript /
  submitted artifact) is unexamined. The submitted `code` being the placeholder, not the
  turn count, was the real signal.
- **Prior-action status:** closes the *observability* half of mission item #1 for the
  stub-failure class. Next cycle (after the rotation produces transcripts): read what the
  failing runs' single tool call actually did, then fix the real root cause (path/isolation
  vs loop/prompt) on the correct side per the routing rule.
