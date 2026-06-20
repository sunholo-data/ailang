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

---

### 2026-06-17 — ROOT CAUSE PROVEN via transcripts (targeted diagnostic, 6 benchmarks ×2 trials)

- **Method:** lock-respecting targeted run (the new rig-lock waited out the rotation, then
  acquired — no contention) of the 6 failing AILANG benchmarks ×2 trials with the
  transcript-retaining binary. 12 runs; transcripts attached.
- **Result — 100% correlation between `WriteFile` and pass:**

  | outcome | n | turns | WriteFile? | what the model did |
  |---|---|---|---|---|
  | PASS | 8 | 3–21 | **YES** (1–7×) | wrote solution, iterated → passed |
  | FAIL (logic_error) | 1 | 3 | **no** | `BashExec(mkdir)` + `ReadFile(stub)`, then stopped — never wrote |
  | api_error | 3 | None | **no** | 0 tool calls — pure prose, non-agentic gate rejects |

- **Root cause (PROVEN, not inferred):** qwen3.6 under motoko **non-deterministically fails
  to write the solution file.** Two failure shapes, same cause — under-engagement:
  (A) explores (mkdir/read) but never `WriteFile` then stops (`finish=stop`, leaves the
  seeded placeholder → logic_error); (B) emits 0 tool calls / prose (→ harness non-agentic
  gate → api_error). The *same* benchmarks pass on other trials with 3–21 turns and
  WriteFile. So it is NOT wrong-path/isolation (passing runs use identical temp paths and
  write fine), NOT an AILANG capture bug, NOT model-capability (when it writes, logic is
  correct). It is **reliability of agentic engagement** — pi drives the same model to
  write+iterate reliably (88%); motoko does not (79%).
- **Lever classification:** **PROMPT** (primary) — motoko's system/loop must compel "use
  tools; WriteFile your solution before finishing; iterate against the expected output."
  Secondary harness angle: motoko could refuse to finalize while `solution.ail` is still
  the placeholder (a loop "definition-of-done" check). DP7 won't help (an empty/stub module
  passes `ailang check`).
- **Routing:** lands as a **draft PR to `arniwesth/motoko_agent`** (`.ail` core / SYSTEM
  prompt / loop) via the `sunholo-voight-kampff` fork — verified locally first.
- **Discipline win:** this is the same conclusion ("prompt/loop") I reached two cycles ago
  by a turn-count *hunch* and the user rightly rejected — now established by the
  WriteFile↔pass correlation in the transcripts. The deciding evidence was the artifact, not
  the metric.
- **Prior-action status:** closes mission item #1 (observability → diagnosis). Promotes
  mission item #2 (motoko prompt/loop PR) to active with a concrete, evidence-backed target.

---

### 2026-06-17 — M-MOTOKO-COMPEL-WRITE A/B → REVERTED (guard non-functional; pi has no loop magic)

- **Built 3 changes** (loop guard in `agent_loop_v2.ail` + WriteFile tool-desc + AILANG-side
  `MOTOKO_REQUIRE_WRITE` env & directive imperative), `.ail` type-checked, smoke-passed
  graph_bfs. A/B: 6 failing benchmarks ×3 trials, before vs after, lock-respecting.
- **A/B result:** before pass **9/18 (50%)** / write 10/18 → after pass **12/18 (66%)** /
  write 12/18. Net +16pp — BUT:
  - **The loop guard fired 0/18 times.** The lift came entirely from the prompt/tool-desc
    "coax", not the guard ("compel").
  - **`config_file_parser` REGRESSED 3/3 → 0/3**, all `api_error` "1 turn, 0 tool calls"
    (pure prose). The verbose directive plausibly pushed qwen into reasoning-prose with no
    tool call on every trial.
  - The guard *should* have fired on those exact 0-tool-call cases and didn't — so it is
    **non-functional** (env not reaching the `.ail` loop via `bun→ailang`, OR the prose path
    exits before the `NoDecision` branch I guarded). Unvalidated + a regression → **reverted
    to known-good** (both repos restored, binary reinstalled, `MOTOKO_REQUIRE_WRITE` gone).
- **pi reverse-engineering (the real finding):** pi's `agent-loop.js` ALSO ends on a
  no-tool-call turn (the follow-up queue is empty in non-interactive `-p` mode) — there is
  **no "turns don't end" mechanism**. pi's request (`openai-completions.js:buildParams`) is a
  vanilla `/v1` streaming chat-completions with `tools`, **no forced `tool_choice`**, no magic
  temperature. So pi's edge is purely that its qwen *engages* (keeps calling tools), not the
  harness. The loop guard was therefore NOT "what pi does" — confirmed.
- **Two concrete leads for the engagement gap (the right direction):**
  1. **Sampling.** qwen3.6's ollama default is **`temperature 1.0`** (+ top_p 0.95,
     presence_penalty 1.5) — high variance, matching the non-deterministic 0-tool-call/prose
     failures. Our `/v1` path doesn't lower it. **Lowering temperature (~0.2–0.3) for the
     agentic path** is the cleanest first experiment.
  2. **Thinking mode.** pi explicitly gates qwen thinking (`enable_thinking` /
     `chat_template_kwargs`); our path does NOT manage it at all. qwen reasoning-in-prose
     would emit no tool call.
- **Lever classification:** AILANG-INTEGRATION (request params: temperature/thinking) — NOT a
  loop guard. The guard is parked (would need env-propagation + correct insertion + to be
  re-justified vs the param fix).
- **Next action (morning, user to steer coax-vs-compel):** A/B **temperature 0.2–0.3** on the
  agentic ollama path (a few-line, pi-faithful change) on the same 6 benchmarks; if it lifts
  engagement, that's the upstream-clean fix. Then revisit prompt nudges (drop the verbose
  directive that regressed config_file_parser).
- **Prior-action status:** M-MOTOKO-COMPEL-WRITE reverted; mission item #2 re-scoped from
  "loop guard" to "request-param engagement (temperature/thinking), pi-faithful."

---

### 2026-06-17 — M-OLLAMA-TEMPERATURE-KNOB landed (code; A/B is the GPU follow-up)

- **Action:** added an opt-in `AILANG_OLLAMA_TEMPERATURE` knob to `internal/ai/ollama`
  (`resolveOllamaTemperature`): applies a temperature on BOTH the `/v1` delegation and the
  native `/api/chat` path. Precedence req.Temperature>0 > env > unset. **Off by default** —
  env unset ⇒ byte-for-byte today's request (no rotation behaviour change on commit).
- **Why:** the pi-faithful lever from the prior diagnosis — qwen3.6's ollama default is
  `temperature 1.0` (high variance), a likely cause of the non-deterministic 0-tool-call/prose
  failures; pi sends a vanilla /v1 body so its edge is the model engaging. Lowering temperature
  is the cleanest first experiment, far less invasive than the reverted loop guard.
- **Verified (no GPU):** `go test ./internal/ai/...` green incl. new tests
  (`TestResolveOllamaTemperature` precedence; `TestStep_ToolsViaOpenAICompat_TemperatureEnv`
  — env 0.2 ⇒ `"temperature":0.2` in the /v1 body; unset ⇒ absent). Build + vet clean.
- **Lever classification:** AILANG-INTEGRATION (request params).
- **Next action (GPU, separate run):** A/B the 6 flaky benchmarks with
  `AILANG_OLLAMA_TEMPERATURE` unset (control) vs `0.2`/`0.3` (treatment, wired via the motoko
  executor env), lock-respecting. If engagement/pass lifts, consider making a low temperature
  the default for the agentic path; else investigate thinking-mode (lead #2).
- **Prior-action status:** delivers the code half of mission item #2; the empirical A/B is the
  only remaining step (deliberately deferred — this was a no-GPU downtime cycle).

---

### 2026-06-18 — Native-path hang fixed; request-capture tooling added; temperature ruled OUT as the pi-gap cause

- **Recurring hang (FIXED):** a motoko run hung **~7h** with ollama idle (rig wasted the night;
  cleared by killing the subprocess tree). Root cause: the native `/api/chat` path streams via
  `c.client.Chat` with **no client timeout** — the `/v1` http timeout (63fc63e0) only guards the
  `/v1` delegation. Fix (`772704cb`): `ollamaCallContext()` adds a per-call context deadline
  (`AILANG_OLLAMA_HTTP_TIMEOUT_SEC`, default 300s) at Step + Generate; plus an explicit
  `ctx.Err()` check after `Chat` (the ollama client can swallow a ctx cancel mid-stream and
  return nil). Test `TestStep_NativePath_TimesOut`. **Caveat:** the hung process held motoko's
  bun **env-server** socket (:8080) — if hangs persist, the next suspect is the bun↔ailang RPC
  (motoko-side), not the ollama provider.
- **Temperature RULED OUT as the pi-vs-motoko differentiator:** pi-coding-agent sets **no**
  default temperature, so pi runs qwen at the SAME ollama default (1.0) we do. The
  M-OLLAMA-TEMPERATURE-KNOB A/B is still worth running (lowering temp may reduce motoko's
  flakiness) but it is NOT why pi is more reliable. The pi gap's cause remains **unisolated** —
  narrowed to the only remaining unequal variables: **system prompt + tool schemas + message
  formatting** (and possibly thinking-mode).
- **Observability gap closed for the diff:** we did NOT capture the exact outbound request from
  any harness (`internal/ai/ollama` logs none; OTEL keeps only a 100-char prompt preview; the
  motoko JSONL logs responses not requests; pi/opencode go direct). Added **`tools/ollama-tap`**
  — a transparent logging reverse-proxy that records every request body (system prompt +
  messages + tool schemas + params), tagged by `?harness=`, for ALL harnesses uniformly
  (they all POST to ollama's HTTP API). Smoke-verified. **This is the enabler for the real
  "why": run motoko vs pi on the same benchmark through the tap, then diff the captured requests.**
- **Lever classification:** HARNESS-ERROR (the hang) + AILANG-INTEGRATION (timeout) +
  observability tooling (the tap).
- **Next action:** (GPU) capture motoko vs pi requests for one benchmark via `ollama-tap` and
  diff the system prompt + tool schemas — the last unequal variable, and the actual answer to
  "why does motoko vary vs pi on the same model."

---

### 2026-06-18 — REQUEST DIFF: the pi-vs-motoko gap is the SYSTEM PROMPT (strong evidence)

- **Method:** captured pi's exact `/v1/chat/completions` request via `ollama-tap` on json_parse
  (`eval_results/pi-request-json_parse.jsonl`). motoko's tap capture failed (its executor doesn't
  propagate `OLLAMA_HOST`/config to the bun→ailang subprocess — went direct to :11434), so
  motoko's side is read from source (config.ail).
- **The diff:**
  - **pi sends a directive agentic SYSTEM prompt** (2460 chars, system role): *"You are an
    expert coding assistant operating inside pi… You help users by reading files, executing
    commands, editing code, and **writing new files**. Available tools: read/bash/edit/write.
    Guidelines: …"* — then the AILANG teaching prompt as the USER message. 4 tools
    (read/bash/edit/write).
  - **motoko sends NO system prompt** — `config.ail:164` `system_prompt: ""` (default); the
    AILANG teaching prompt is folded into the user message (the executor passes
    `SystemPrompt+"\n\n"+Directive` as one positional task arg). 6 tools (ReadFile/WriteFile/
    EditFile/BashExec/RunTests/Search).
  - Both: qwen3.6, stream, temperature unset (=1.0), no tool_choice — sampling confirmed equal.
- **SMOKING GUN:** the shared AILANG teaching prompt ("v0.16.2 … with Output Discipline") literally
  says **"Output raw AILANG code only — no markdown fences, no prose, no JSON wrappers. The first
  line must be `module ...`"** — a **0-shot** instruction that tells the model to EMIT CODE AS
  TEXT, not use tools. In AGENT mode this actively discourages tool use.
- **Mechanism (highly likely root cause):** motoko gives qwen the "output raw code, no prose"
  instruction with NO agentic system prompt to override it → qwen often complies by emitting code
  as prose → **0 tool calls → no WriteFile → no solution → fail** (exactly the observed failure
  mode). pi prepends its agentic "you are a coding assistant; use read/bash/edit/write to write
  files" SYSTEM prompt, which overrides the 0-shot framing → qwen uses the write tool reliably
  (88%). This also explains: the non-determinism (qwen sometimes ignores "output raw code"); why
  the verbose write-imperative I appended REGRESSED config_file_parser (it piled more conflicting
  instruction into the USER message instead of establishing agentic SYSTEM framing); and why
  temperature isn't the cause (equal).
- **Lever classification:** **PROMPT** — (a) the AILANG AGENT-mode prompt should NOT carry the
  0-shot "output raw code only / no prose" instruction (it belongs to standard mode); (b) motoko
  needs a pi-style agentic system prompt. Both AILANG-side (the prompt is AILANG-generated; the
  motoko system prompt can be set via the eval/executor).
- **Confidence:** strong but not yet airtight — confirmed pi's request live + motoko's missing
  system prompt from source; NOT yet captured motoko's live request+response (env propagation
  blocked the tap). Confirmation step: log motoko's outbound request (small `internal/ai/ollama`
  debug knob, or make the executor propagate OLLAMA_HOST) and verify the response is prose.
- **Next action (the actual fix, pi-faithful):** A/B an **agent-mode prompt that says "use the
  write tool to write your solution" instead of "output raw code only"** + a concise agentic
  system prompt for motoko. Expect engagement (WriteFile rate) to rise toward pi's.
- **Prior-action status:** answers the standing question "why does motoko vary vs pi on the same
  model + benchmarks." Supersedes the temperature lead (ruled equal) and the reverted loop guard.

---

### 2026-06-18 — "Copy pi's system prompt" tested → system prompt is NOT the dominant factor

- **Built motoko-request observability** (`internal/ai/ollama` request-dump, env OR `$HOME`-sentinel
  gated — committed `0bac6116`) because the external tap couldn't capture motoko (its bun→ailang
  chain doesn't propagate our custom env; `HOME` does). CONFIRMED motoko sends an **empty system
  message** vs pi's agentic one (request saved: `eval_results/motoko-request-json_parse.jsonl`).
- **Untangled motoko's system-prompt delivery** (two dead ends, both verified via the dump):
  `--system-prompt` is **ignored in headless** (motoko reads `SYSTEM_MD` env, `config.ts`), and
  `SYSTEM_MD` must point **inside the workspace** (`index.ts::systemPromptForWorkspace` rejects
  `..`/abs-outside). Correct delivery = file in workspace + `SYSTEM_MD`. Verified the dump then
  showed motoko's system message = pi's adapted prompt (756 chars), tools intact, smoke PASS.
- **A/B (6 flaky benchmarks ×3): empty system 10/18 (55%) → pi-adapted system 11/18 (61%).**
  Marginal (+1/18, one regression config_file_parser), NOT pi's ~88%. **Hypothesis "copy pi's
  system prompt → similar performance" is FALSE.** The system prompt is the biggest *request-level*
  difference but NOT the dominant *performance* factor.
- **Conclusion / redirect:** the gap is the **agent loop / iteration depth**, not the prompt — pi
  takes 9–41 turns self-correcting on these benchmarks; motoko engages but stops sooner (often
  compiling-but-wrong). That "run → see failure → fix → re-run until it passes" loop quality is
  the higher-leverage target, now measurable with the request/turn captures.
- **Landed/kept:** request-dump observability (`0bac6116`). **Reverted:** the SYSTEM_MD executor
  injection (marginal + hacky). **Earlier this session, kept:** agent-mode output-delivery override
  (+11pp, `2cbaf85a`). **Filed:** motoko draft PR (arniwesth/motoko_agent#45) documenting the
  empty-system-prompt + SYSTEM_MD/headless friction + the "system prompt ≠ the gap" finding.
- **Lever classification:** PROMPT confirmed sub-dominant; next lever = **HARNESS (agent loop /
  iteration)** — likely a motoko-side change (loop persistence / verify-and-retry), or an
  AILANG-side observation of pi's loop to mirror.
- **Next action:** capture motoko's FULL turn sequence vs pi's on a shared failing benchmark
  (now possible) and characterize the iteration-depth difference; that's the real gap.

---

### 2026-06-18 — ITERATION-DEPTH GAP QUANTIFIED (rolling data, no GPU) + motoko 83%

- **Fresh rotation analysis** (existing rolling data, turn distributions per harness on AILANG):

  | harness | AILANG | turns on PASS (median/max) | turns on FAIL (median/max) |
  |---|---|---|---|
  | **motoko** | **83%** | 4 / 49 | **2 / 9** |
  | **pi** | 92% | 7 / 121 | **33 / 164** |
  | opencode | 75% | 6 / 49 | 21 / 92 |

- **The gap is iteration depth, now measured:** on FAILURES, **motoko gives up at a median of 2
  turns (never past 9); pi grinds a median of 33 (up to 164).** Same model. motoko *can* iterate
  (passes reach 49 turns) — so it is NOT a hard step-budget/timeout cap (a cap would bound passes
  too); on hard problems motoko's model **stops emitting tool calls after ~2 turns** and motoko
  finalizes, whereas pi's model keeps trying. Recent failing-run transcripts confirm the shape:
  1–2 exploratory calls (often no WriteFile), then stop → stub/logic_error.
- **Also: motoko AILANG is now 83%** (was 76% pre-output-delivery) — the agent-mode output-delivery
  override (`2cbaf85a`) is helping at scale. pi 92%, opencode 75%; motoko now between them.
- **Lever classification:** **HARNESS (agent loop / iteration persistence)** — confirmed. NOT
  prompt (sub-dominant, tested), NOT temperature (equal), NOT a harness timeout cap.
- **BLOCKED ON GPU (honest):** both remaining steps need the rig — (1) root cause: capture a fresh
  failing motoko run's FULL turn-by-turn (request-dump per Step is built; needs one GPU run) to see
  *why* the model disengages at turn 2; (2) the fix: a motoko-side loop-persistence change
  (keep iterating / re-prompt instead of finalizing on an early apparent-stop) — a motoko PR that
  must be A/B-validated on the rig. Observability (transcript + request-dump) and the measurement
  (`avgTurnsFailure` per executor/lang, export_json_executors.go) are already in place.
- **Prior-action status:** advances the 2026-06-18 redirect (gap = the loop) from a hypothesis to a
  quantified fact. No new code this cycle (no-GPU): the frontier's next steps are GPU-bound and the
  enabling observability/metrics already exist — recording + backlog refresh per the mission's
  "blocked → note and stop" rule rather than a manufactured change.

---

### 2026-06-18 — Two prompt experiments: system-prompt override FAILED; agent-mode output-delivery LANDED

- **Experiment 1 — pi-style agentic system prompt via `--system-prompt` (FAILED, reverted).**
  Hypothesis: motoko has no agentic system prompt (inferred from `config.ail` `system_prompt:""`).
  Passed a pi-style prompt via the executor's `--system-prompt`. A/B (6×3): **55%→0%** pass/
  WriteFile — every after-run "1 turn, 0 tool calls". The override REPLACED motoko's default
  system prompt (built by `dispatch_build_system_prompt` extensions, rpc.ail) with an inferior
  one → my source-inference ("motoko sends no system prompt") was **wrong**; the extensions build
  a working one. Reverted (uncommitted, gated off — rotation never affected). Lesson: stop
  guessing at motoko's prompt blind — I still cannot capture motoko's live request (the tap
  missed it; `BuildEnvironment` DOES pass `os.Environ()`/`OLLAMA_HOST`, so it's a deeper routing
  quirk, not env propagation). Motoko-request observability is the real blocker for prompt work.
- **Experiment 2 — agent-mode output-delivery override (LANDED).** Fix at the correct layer
  (user's direction): the AILANG teaching prompt's 0-shot "Output raw code only — no prose" line
  is wrong for AGENT mode. `GenerateAgentPromptsWithSystemPrompt` is agent-only, so append an
  override clarifying that "raw code, no prose" = FILE CONTENTS, not the chat reply; use the
  file-write tool. A/B (6×3): **44%→55%** pass/WriteFile (+11pp), **no regression** (json_transform
  0→1, lambda_calc 1→2; rest unchanged; graph_bfs still 0/3). Modest + noisy (2/18) but principled
  and non-harmful → **ON by default** for agent mode (opt out `AILANG_AGENT_OUTPUT_DELIVERY=0`).
  Doesn't touch motoko's machinery (just adds text to the AILANG-generated prompt all harnesses
  get) so it can't break motoko's tool setup. Lands on `dev`; rotation validates at scale.
- **Lever classification:** PROMPT (agent-mode output discipline). Honest scope: this is a real
  but partial fix — the non-determinism persists (graph_bfs 0/3) and the prompt conflict is only
  one factor. Closing the full gap to pi likely needs (a) motoko-request observability, then (b)
  understanding motoko's default system prompt vs pi's, and possibly the fuller prompt refactor
  (generic teaching reference + separate standard/agent output-discipline preambles).
- **Next action:** (1) build motoko-request observability — make the executor route motoko's
  ollama calls through a tap, or add an `internal/ai/ollama` request-dump, so prompt changes are
  diagnosable not guessed; (2) re-measure the output-delivery override on the rotation.
- **Prior-action status:** acts on the request-diff finding; the system-prompt-override path is
  closed (made it worse); the output-delivery prompt fix is the landed, principled improvement.

## 2026-06-18 — 3-way request capture: motoko vs pi vs opencode (json_parse/ailang, qwen3.6)

Captured the actual outbound request each harness sends to the SAME model on the SAME
benchmark — motoko via the request-dump (`AILANG_OLLAMA_LOG_REQUESTS`), pi + opencode via
`tools/ollama-tap` (:11435→:11434, since they bypass `internal/ai/ollama` and drive `/v1`
directly). Picked each harness's main coding request (most tools + largest payload).

| harness | system role | AILANG teaching lives in | tools | threading |
|---|---|---|---|---|
| **motoko** (system-role ON) | **89,754 ch — the AILANG teaching prompt** | system role | **6** (ReadFile/WriteFile/EditFile/BashExec/RunTests/Search) | system→user→(asst/tool)×N |
| **pi** | 2,458 ch — generic "expert coding assistant in pi" | user msg (98,319 ch) | **4** (read/bash/edit/write) | identical |
| **opencode** | 10,050 ch — generic "you are opencode" | user msg (94,854 ch) | **33** (23 `ailang-docs_*` MCP + bash/edit/glob/grep/read/skill/task/todowrite/webfetch/write) | identical |

**Findings (back up our decisions):**
- **Chat-history threading is IDENTICAL across all three** — system → user(task) → alternating
  `assistant(tool_calls)`/`tool(result)` with matching ids. The motoko↔pi↔opencode gap is **NOT**
  history handling. (Resolves the "I suspect chat history may not be working" hypothesis: it works,
  and equivalently, in all three.)
- **All three deliver the AILANG teaching content** — the difference is ROLE. motoko (with
  M-MOTOKO-SYSTEM-ROLE) is the only one that puts it in the SYSTEM role; pi/opencode bury the
  equivalent ~95–98 KB in the first USER message. NOTE: the *historical* motoko default (legacy
  fold) ALSO put teaching in the user message — so system-role is a NEW improvement on top, not the
  reason motoko historically beat opencode.
- **Tool surface is the standout differentiator.** opencode exposes **33** tools (23 of them the
  `ailang-docs` MCP server configured in `~/.config/opencode/opencode.jsonc`) vs motoko's **6** and
  pi's **4**. opencode's transcript is full of tiny `tool(24)`/`tool(26)` results (todowrite, empty
  MCP lookups) — low-value churn. A 35B local model with a 33-tool schema spends turns *looking
  things up / bookkeeping* instead of writing code. This is the most defensible explanation for
  "opencode grinds 21 turns but still trails motoko 83% vs 75%": **lean, task-focused toolset >
  broad tool surface for a small local model.** pi is leanest (4) but lacks AILANG-native execution.
- **Single-trial outcomes this capture:** motoko PASS (29 turns), opencode PASS (20 turns), pi alarm
  at 37 turns (compile_error — still grinding at the 200s cap, consistent with pi's high-iteration
  profile).

**Lever classification:** HARNESS-DESIGN (tool surface) + PROMPT (system-role placement). Decisions
backed: (1) keep motoko's toolset lean — do NOT bolt the ailang-docs MCP onto motoko; (2) the
M-MOTOKO-SYSTEM-ROLE change (teaching → system role) is principled and matches no-harness-does-better
evidence; verify + A/B before default-on.

**Prior-action status:** M-MOTOKO-SYSTEM-ROLE delivered + verified (system msg 0→89,754 ch in the
system role, user 94,168→4,406, json_parse PASS @ 29 turns). Capture confirms it lands as a real
system-role message, not folded. Follow-up: motoko-side `--system-prompt` flag PR so external
harnesses inject the system prompt cleanly (removes AILANG's `.motoko_system.md` workaround).

## 2026-06-18 — A/B: AILANG_MOTOKO_SYSTEM_ROLE off vs on (6 flaky ×3) → KEEP GATED, not default-on

Tested whether delivering motoko's system prompt in the **system role** (`=1`) vs the
legacy **fold-into-user** default (unset) lifts pass rate. motoko-local-qwen3.6, single
lock hold, `--parallel 1 --trials 3` on json_parse, json_transform, config_file_parser,
lambda_calc, graph_bfs, cli_args.

| benchmark | off (fold) | on (system-role) | off turns | on turns |
|---|---|---|---|---|
| cli_args | 3/3 | 3/3 | 12.3 | 16.0 |
| config_file_parser | 0/3 | **1/3** | 6.0 | **17.5** |
| graph_bfs | 0/3 | **1/3** | 0.0 | 3.0 |
| json_parse | 3/3 | **2/3** | 8.7 | 10.0 |
| json_transform | 2/3 | **1/3** | 5.3 | 10.0 |
| lambda_calc | 2/3 | **0/3** | 3.7 | 2.0 |
| **TOTAL** | **10/18 (56%)** | **8/18 (44%)** | | |

**Verdict: do NOT default-on; keep `AILANG_MOTOKO_SYSTEM_ROLE` gated (opt-in).** Net **−2/18,
well within n=3 noise** — no basis to flip the default.

**But two signals worth keeping:**
- **System-role helps the FLOOR**: the two benchmarks that never passed under fold
  (config_file_parser, graph_bfs) both went **0→1/3**. It hurt the mid-tier
  (json_parse 3→2, json_transform 2→1, lambda_calc 2→0).
- **System-role lifts iteration depth**: turn counts rose across the board (config_file_parser
  6.0→17.5, json_parse 8.7→10.0, json_transform 5.3→10.0). The model ENGAGES MORE when the
  teaching is in the system role — directionally aligned with backlog #1 (iteration persistence).
  The extra grinding didn't convert to net passes here, but the harder benchmarks (which need
  more turns) benefited. This suggests system-role may pay off **combined with** a loop-persistence
  change rather than alone.

**Lever classification:** PROMPT (role placement). Ruled out as a standalone default; retained as
a gated lever and a co-factor to re-test alongside iteration-persistence (backlog #1). The flag and
the motoko `--system-prompt` PR (arniwesth/motoko_agent#46) stay — they're the clean injection
mechanism regardless of the default.

**Prior-action status:** closes the M-MOTOKO-SYSTEM-ROLE A/B. Decision: stays opt-in. Next mission
cycle returns to backlog #1 (iteration-persistence root cause), now with the added hypothesis that
system-role + persistence should be tested together (system-role already raises engagement).

## 2026-06-18 — Backlog #1 ROOT CAUSE (proven from A/B transcripts) + fix design

Mined the failing runs from the system-role A/B (transcripts retained via M-MOTOKO-OBS-TRANSCRIPT).
The disengagement has **two shapes**, both = "model stops emitting tool calls early, motoko finalizes":

- **Mode A — `1 turn, 0 tool calls`** (`api_error` "non-agentic result"): the model answers in
  PROSE/code in the chat reply and never calls WriteFile. motoko finalizes immediately. Dominant on
  graph_bfs (off 3/3 trials). The AILANG executor's non-agentic guard catches it post-hoc.
- **Mode B — `2 turns, 1 tool call`** (`logic_error`): the model makes ONE inspect call then stops
  WITHOUT writing. Proven transcript (lambda_calc on trial2): a single
  `BashExec {"cmd":"cat …/solution.ail || echo FILE_NOT_FOUND"}` — it checks for a solution, sees
  none, and **stops**. motoko finalizes. No WriteFile ever happens.

**Where motoko finalizes** (`src/core/agent_loop_v2.ail`, ~L1042–1110): when
`finish_reason != "tool_calls"`, the loop tries (1) hybrid-bash extraction (only fires on a fenced
shell block), then (2) `dispatch_solver_candidate`, which has three outcomes —
`Accept`→done, **`ContinueWithFeedback(fb)`→inject user msg + recurse**, `NoDecision`→done. The
persistence mechanism **already exists** (`ContinueWithFeedback`), but in the **ollama eval profile
no extension returns it**, so prose-without-write falls to `NoDecision → emit done` → premature
finalize. THAT is the gap (not a missing mechanism — a missing trigger).

**Fix design (backlog #1b — motoko_agent PR):** a bounded, built-in persistence nudge at the
`NoDecision` boundary (flag-gated, e.g. `MOTOKO_PERSIST_RETRIES=N`, default 0 = off so it's opt-in
for A/B). When the model stops with `finish_reason != tool_calls` AND no solution has been written
this session AND retries remain: inject a user-role nudge ("You stopped without writing a solution —
use WriteFile to save your AILANG implementation to <path>, then continue.") and recurse, decrementing
the retry budget. Requires threading two new bits through `loop_v2`: a `wrote_solution: bool` (set
when a WriteFile tool dispatch succeeds) and a `persist_left: int`. Distinct from the reverted
M-MOTOKO-COMPEL-WRITE (a hard one-shot guard that fired 0/18) — this keeps the loop ALIVE with a
nudge, bounded. A/B-validate on the same 6 flaky benchmarks; the system-role-A/B signal (system-role
already raises turn counts) suggests testing persistence WITH `AILANG_MOTOKO_SYSTEM_ROLE=1`.

**Prior-action status:** advances #1a (root cause) from "quantified" to "proven with transcripts +
exact finalize site identified". Scopes #1b precisely. Next: implement the bounded persistence nudge
on a motoko_agent branch, build, A/B (GPU).

## 2026-06-18 — Backlog #1b IMPLEMENTED + A/B: persistence nudge = +3/18 (61%→78%), PR-worthy

Implemented M-MOTOKO-PERSIST-NUDGE (motoko_agent): a bounded built-in nudge at the
`agent_loop_v2.ail` NoDecision boundary. When the model stops (finish_reason != tool_calls),
no extension claims the prose, NO WriteFile has been attempted this run, and nudge budget
remains → inject a user-role nudge ("use WriteFile, then keep going until it compiles") and
recurse instead of finalizing. State (writes-attempted, nudges-used) is read from the
already-threaded `msgs` history — **no new params across loop_v2's 16 recursive call sites.**
Gated by `MOTOKO_PERSIST_RETRIES` (default 0 = off); plumbed through the RuntimeProcess env
allowlist (runtime-process.ts) so it reaches the ailang runtime.

**Plumbing proven (direct motoko run):** with `MOTOKO_PERSIST_RETRIES=3` and a prose-only
task, `persist_nudge` fired once and converted a 0-tool-call disengagement into **17 tool
calls** — then `any_writefile_attempt` blocked further nudges (bounded). Confirms env
propagation + firing + the bound.

**A/B (6 flaky ×3, off vs on=3), motoko-local-qwen3.6:**

| benchmark | off | on | off turns | on turns |
|---|---|---|---|---|
| cli_args | 2/3 | 3/3 | 17.0 | 7.3 |
| config_file_parser | 1/3 | 2/3 | 4.5 | **21.0** |
| json_transform | 2/3 | 3/3 | 12.0 | 14.7 |
| lambda_calc | 2/3 | 3/3 | 6.7 | 5.0 |
| json_parse | 3/3 | 3/3 | 8.0 | 11.3 |
| graph_bfs | 1/3 | 0/3 | 5.0 | 4.3 |
| **TOTAL** | **11/18 (61%)** | **14/18 (78%)** | | |

**Net +3/18.** 4 of 6 improved, 1 regressed (graph_bfs 1→0 — the perennial 0/1 floor, noise
at n=3). config_file_parser is the clearest causal win: the nudge drove it 4.5→21 turns and it
converted. Directionally clear and the OPPOSITE of system-role (−2): persistence earns a PR.

**Lever classification:** HARNESS-LOOP (iteration persistence) — the mission's flagship #1b.
**Decision:** land the mechanism, keep **gated (default off)** given n=3, recommend
`MOTOKO_PERSIST_RETRIES=3` and validate at scale on the rotation before any default-on. Draft
PR to arniwesth/motoko_agent. The system-role A/B's hint (system-role raises engagement) is the
natural next combined test: persistence + `AILANG_MOTOKO_SYSTEM_ROLE=1` together.

**Prior-action status:** closes backlog #1 (a: root cause proven; b: fix implemented, proven,
A/B net-positive). Next cycle: PR review + rotation-scale A/B + the persistence×system-role combo.

## 2026-06-18 — SOURCE-GROUNDED CORRECTION: pi has NO persistence; the gap is context-engagement

Mined pi's actual loop (`@mariozechner/pi-agent-core/dist/agent-loop.js`, `runLoop`) instead of
inferring from behavior. **pi STOPS the instant the model emits zero tool calls** — the inner loop
is `while (hasMoreToolCalls || pendingMessages.length > 0)`; a turn with no toolCall sets
`hasMoreToolCalls=false`, and in headless eval there are no steering/follow-up messages → it ends.
**No re-prompt, no nudge, no persistence. Identical to motoko's stop condition.**

**This overturns the working hypothesis.** pi's ~33-turn grind is NOT a loop feature — the qwen3.6
model *under pi's context* naturally keeps emitting tool calls; motoko's ~2-turn stop is the same
model *under motoko's context* disengaging early. **The differentiator is context-driven engagement,
not loop persistence.**

Consequence: **M-MOTOKO-PERSIST-NUDGE (#47) is a DIVERGENT band-aid, not a pi port** — it adds a
force-continue mechanism pi does not have. It tested +3/18 (real, keep it, default-off), but it is
NOT "how pi wins" and should not be sold as such.

Source-level rule-outs (pi system prompt + tool schemas, from captures + source):
- pi system prompt is **lean (~2.5 KB), concise, tool-focused**; NO "persist/keep going" directive;
  notably says **"Be concise in your responses"** (discourages the prose-dump that causes motoko's
  Mode-A 0-tool-call disengagement).
- pi tools = 4 lean (read/bash/edit/write); motoko = 6 (adds RunTests, Search). Descriptions
  comparable — not a glaring differentiator.

**Reframed investigation (pi-faithful, context-engagement):**
1. **Prompt leanness / "be concise"**: motoko sends a huge AILANG teaching prompt (89 KB system or
   folded); pi sends 2.5 KB. Hypothesis: dense teaching pushes a 35B model toward "emit a final
   answer" (prose) rather than iterating with tools. Test a lean, action-oriented, "be concise"
   motoko system framing.
2. **Tool-RESULT / error feedback format** (NOT yet compared): what the model sees after each tool
   call is the loop's fuel. Compare pi's vs motoko's result/error envelopes — noisy/confusing
   results would cause early disengagement. Next capture target.
3. Sampling note: motoko request showed `temperature: 0`; re-confirm pi's and whether it matters.

**Prior-action status:** corrects the mission's mental model via source (the user's "are our
optimizations based off pi source?" — answer was NO; now grounded). Persistence nudge stays as a
divergent local aid; the real lever is context/prompt engagement. Next cycle: lean-prompt A/B +
tool-result feedback capture, both pi-faithful.

## 2026-06-18 — M-MOTOKO-AGENT-SYSTEM-PROMPT: proper A/B = NULL at scale (+1/52). Smoke did not generalize.

Design doc: design_docs/planned/v0_25_0/m-motoko-agent-system-prompt.md. Hypothesis: motoko's
EMPTY system role (vs pi/opencode's lean agentic prompt) causes the Mode-A prose disengagement;
giving motoko a lean agentic system prompt (teaching stays in user) should lift pass rate.
Delivery: AILANG_MOTOKO_AGENT_SYSTEM_FILE knob (motoko.go, c90f4a2a).

**Exploratory smoke (6 flaky ×2):** 9/12 (75%) vs empty-base 11/18 (61%), tool calls 6–17. Looked
strong → justified the proper A/B.

**Proper A/B (core tier 26, n=2, BOTH arms fresh, one run):**
- empty 39/52 (75%) vs agentsys 40/52 (77%) = **+1/52 — NOISE.**
- improved: explicit_dataflow_ssa, higher_order_functions, json_transform (all disengagers on empty)
- regressed: audit_chain_replay (2/2→1/2), merge_sort (2/2→1/2) — both were passing
- **The 6-flaky smoke did NOT generalize.** It was a biased subset (the hardest, most-disengaged
  benchmarks with the most headroom). On the broader core tier the lean prompt is net-neutral.

**Failure-mode segmentation (empty baseline, the useful by-product):** 8 of 10 failing benchmarks
are DISENGAGE (0–2.5 tool calls): state_machine_vending, higher_order_functions, csv_to_json_converter,
explicit_dataflow_ssa, graph_bfs, json_transform, ast_patch_roundtrip, contract_roman_numeral. Only
2 are grind-but-wrong: prompt_injection (8 TC), config_file_parser (18 TC). So disengagement IS the
dominant failure mode even on core — but the lean system prompt only PARTIALLY fixes it (converts
some disengagers, misses others, and adds variance to passers). Net wash.

**Decision (per the design doc's success criteria — NOT met):** do NOT productionize the agentic
prompt as a win. Keep the AILANG_MOTOKO_AGENT_SYSTEM_FILE knob (harmless experimental tool). Record
NULL. Note: core-tier empty baseline (75%) is much higher than the rotation AILANG-agent aggregate
(63%) — the motoko↔pi gap concentrates on HARDER tiers (stretch/vision), not core. A fresh pi-on-core
number was not captured (head-to-head pi arm was killed); measuring it is a gap.

**Lever classification:** PROMPT (system-role framing) — REAL but PARTIAL & NOISY; not the clean
mission lever. **Honest correction of my own smoke-driven overclaim** (caught by running the full
controlled set per the discipline rule [[motoko-investigation-discipline]]).

**Next (per design doc Non-Goals → "move to the next diff"):** (a) the 2 regressions suggest the
prompt wording disrupts passers (e.g. "keep going until it compiles" → over-editing) — a refined
prompt MIGHT net positive, but that risks a prompt-tuning rabbit hole; (b) the structurally cleaner
diff to examine next is the **tool-RESULT / error feedback format** (pi vs motoko — not yet
compared), since disengagement persists even with an agentic prompt; (c) measure pi-on-core for the
true gap, and segment the gap by tier (the real gap is on harder benchmarks). USER to steer.

## 2026-06-18 (cycle 2, NON-GPU) — Failure-mode observability: the motoko↔pi gap IS disengagement

Mission cycle on fresh rotation data (`eval_results/rotation/os-rolling`, summary updated 20:53).
The top backlog items are GPU-bound (blocked this downtime run), so did the prescribed UNBLOCKING
observability item: built `tools/eval_failure_modes.py` (tested, `--self-test` passes) to segment
agent failures by MODE — existing tooling only groups by `error_category`, which hides the split
that matters:
- **DISENGAGE** — fail with ≤2 tool calls (prose / one inspect call; no real solution attempt).
- **GRIND_WRONG** — fail with >2 tool calls (engaged, iterated, but incorrect).

**Finding (qwen3.6, AILANG agent, rotation):**

| harness | N | pass | disengage | grind_wrong |
|---|---|---|---|---|
| motoko | 117 | 69% | **29%** | 1% |
| pi | 113 | 95% | **3%** | 0% |
| opencode | 114 | 80% | 18% | 1% |

**motoko↔pi gap = +26pp pass, and it is ENTIRELY disengagement (+26pp disengage, +1pp
grind_wrong).** This confirms the diagnosis at full rotation scale (not the biased 6-flaky subset)
and quantifies it: motoko's problem is NOT correctness (grind_wrong ~1% for both) — it is that
qwen3.6 under motoko answers in prose / stops after one tool call 29% of the time, vs 3% under pi.

**Always-disengage benchmarks (3/3 fail, 0 pass, ≤2 tool calls)** — the precise targets for the
next fix cycle: csv_to_json_converter, log_file_analyzer, graph_bfs, polymorphic_ord_defaulting,
run_length_encode, symbolic_diff, config_file_parser.

**Why the prior two fixes under-delivered (now explained by this number):**
- M-MOTOKO-AGENT-SYSTEM-PROMPT A/B was run on the CORE tier (motoko already 75% there) — but the
  29% disengagement is spread across the FULL set incl. harder benchmarks; core under-samples it.
- M-MOTOKO-PERSIST-NUDGE forces continuation AFTER disengagement; but the cleaner win is to stop
  the model disengaging in the FIRST place. Both touched the symptom, not the 29%.

**Lever classification:** OBSERVABILITY (unblocks). **Tool:** `tools/eval_failure_modes.py`
(reusable; building block toward the future `motoko-analyzer` skill).

**Prior-action status / next:** gap is now precisely located = disengagement (29% vs 3%). Next
(GPU) cycle: target the 7 always-disengage benchmarks; capture their first-turn requests via the
request-dump to see WHY qwen emits 0 tool calls there (tool schema not seen? task framing? result
format?), then a targeted fix + A/B measured by the disengage-rate delta (not just pass rate).

## 2026-06-19 — Why model I/O isn't in `ailang chains` (real gap) + a captured temp=0 vs 1.0 diff

**User question: should the model in/out be in the chains CLI — is its absence a gap? YES.**
Investigated observatory.db (`~/.ailang/state/observatory.db`):
- `ailang chains` is architected for the **COORDINATOR agent pipeline** — `chain_stages` rows are
  high-level stages (design-doc-creator → sprint-planner → execute) with agent_id/provider/status/
  approval/cost. There is **no column for per-turn model request/response text**. Eval runs DO create
  a chain (`eval_suite:…/agent`) but it shows STAGES=0 / $0.00 — the conversation isn't recorded.
- Per-model-call detail is meant to live in OTEL **spans**. AILANG's providers ARE instrumented
  (`internal/ai/ollama/client.go` emits `ollama.generate`; all providers similar). BUT the `spans`
  table is **EMPTY (0 rows)** — those spans don't reach observatory.db in the eval context.
- **Why empty:** (a) agent model calls run in the harness **SUBPROCESS** (motoko/pi/codex/opencode);
  subprocess spans don't export back to the parent's observatory.db (the documented TRACEPARENT/
  subprocess boundary). (b) external CLIs (pi/codex/opencode) never touch AILANG's instrumented
  path at all → invisible by design; only a network tap sees them.
- **That's exactly why the file-based request-dump exists.** This run FIXED it to also log the
  RESPONSE (text + tool_calls + finish_reason) — and fixed a follow-up bug where the response log
  was on the native path while motoko always delegates to /v1 (`5d6fa7f8b`). Dump now = full IN+OUT.
- **Proper fix (backlog):** persist model-call spans (with req/resp) to observatory.db tagged with
  the eval chain_id, at least for the in-process + motoko paths → `ailang chains view <id>` would
  show the conversation instead of the file-dump hack.

**Captured IN diff (graph_bfs, the smoking-gun comparison the user asked for):**
- **motoko**: system **0 chars** (empty), 6 tools, **temperature 0**.
- **pi** (engaged: pass, 3 tool calls, 1530-char solution): system **2458 chars** (agentic), 4 tools,
  **temperature None → ollama default 1.0**.
Two concrete input differences: empty-vs-agentic system (system-prompt A/B was null, so not the
whole story) AND **temperature 0 vs 1.0**. This CONTRADICTS the stale backlog #2 assumption ("pi runs
the same 1.0 default") — the request data shows motoko runs qwen GREEDY (temp 0), pi at 1.0. Greedy
decoding plausibly drives the confident-prose / 0-tool-call disengagement. Untested as the cause;
warrants a temperature A/B measured by **disengage-rate delta** (now that we have the tool).

**Prior-action status / next:** dump now captures OUT (re-run a graph_bfs capture to see motoko's
actual prose). Two live leads for the next GPU cycle, both from the captured request: temp 0→default,
and (re-confirm) system framing. Record the chains→spans persistence as an observability backlog item.

## 2026-06-19 — THE DELTA (source-grounded): better harnesses read qwen's `reasoning` field; AILANG drops it

Wire capture (new HTTP-wire logger, c1f87275e) proved qwen3.6 over ollama /v1 returns a `reasoning`
field (10k+ chars/turn) that AILANG's `ParseChatStepResponse` discards (reads only content +
tool_calls). Cross-checked against the three reference harnesses + Qwen's own:

| harness | reasoning-field handling |
|---|---|
| pi (`pi-ai/openai-completions`) | GENERIC: tries `reasoning_content`/`reasoning`/`reasoning_text`, first non-empty |
| qwen-code / Qwen-Agent | GENERIC: `reasoning_content or reasoning` (fixed in Qwen-Agent#789) |
| AILANG/motoko | NONE — dropped |

**Qwen-Agent #789 IS our exact bug:** "Ollama streaming chunks use `reasoning` field not
`reasoning_content` — thinking content silently lost with Qwen3." Fix is a 2-line GENERIC fallback,
not a qwen branch — confirms the fix belongs in AILANG's generic openai-compat core, not per-model
branches/extensions. AILANG already avoids the streaming XML-tool-call breakage (qwen-code#176,
lmstudio#1071) by using NON-streaming /v1 — the wire capture confirmed it gets proper native
`tool_calls` when qwen tool-calls. So AILANG's gap is specifically the dropped `reasoning` field.

**Causation status (honest):** the captured graph_bfs run ENGAGED + PASSED (6 tool calls; reasoning
dropped but the tool call came through the native field) — so the rotation's "graph_bfs always
disengages" is likely STALE vs the current binary, and dropping reasoning does NOT always disengage.
The dropped-reasoning is a CONFIRMED latent gap; not yet proven to CAUSE the 0-tool-call cases.
Next: capture disengaging benchmarks on the wire to check whether the tool call / answer is stuck in
`reasoning` (-> AILANG sees 0 tool_calls). Then implement the generic reasoning-field read + A/B.

**Sources:** pi-ai openai-completions provider (local); Qwen-Agent#789; qwen-code#176, #2402;
lmstudio#1071; ollama#15288; vLLM#39056; pi-mono#1205.

**TODO (user, push priority): add `qwen-code` (QwenLM/qwen-code CLI) as an eval-suite harness arm**
— qwen's own coding agent, OpenAI-compat/ollama capable, CLI like opencode/pi (fits the executor
contract: download, wire `agent_cli: "qwen-code"`). Gives a qwen-tuned reference data point next to
pi/opencode/motoko on the SAME benchmarks — directly measures the motoko↔well-tuned-harness delta.

## 2026-06-19 — ROOT CAUSE FOUND (wire-proven): disengagement = TRUNCATION (max_tokens 4096 vs reasoning ~4k tokens)

The HTTP-wire logger (c1f87275e) + reasoning capture (79714e3d5) finally made the disengaged
turns visible — and the cause is NOT prompt/temperature/persistence/reasoning-parsing/hermes. It is:

**qwen3.6 is a heavy reasoner; its thinking blows the max_tokens budget, truncating BEFORE the
tool call.** Signal run (7 disengaging benchmarks ×2, with the fix + wire log):
- 14 disengaged (0-native-tool-call) turns: **11 = `finish_reason=length` (TRUNCATED)**, 3 = stop.
- **Of 14 native-0-toolcall turns, 0 had a `<tool_call>` in reasoning** → the hermes-recovery fix
  is a no-op here; the reasoning-parsing hypothesis is REFUTED for motoko's disengagement.
- Disengaged-turn reasoning length: **median 13,872 / max 16,155 chars (~4k+ tokens of thinking)**.
- The model writes the whole solution INSIDE its reasoning ("Now let me write this to the file"
  at the END of 14k chars) then gets cut off.

**The delta, wire-proven:** motoko sends **`max_tokens=4096`**; pi sends **`max_completion_tokens=16384`**.
qwen's reasoning alone exceeds 4096 → `finish=length` → no tool call → "disengaged". pi gives room
(16384) AND reads the reasoning. AILANG default is `internal/ai/handler.go:95 maxTokens:4096`.

**Second broken link:** motoko's ollama profile sets `ai_options_json:{"enable_thinking":false}` to
disable thinking, but the wire request has only `[max_tokens,messages,model,tools]` — **enable_thinking
is DROPPED, never forwarded to /v1.** So motoko's thinking-disable attempt is a no-op; qwen thinks freely.

**Fix (pi-faithful, high confidence):** raise the agent/ollama `max_tokens` to ≥16384 (matches pi;
give qwen room to think AND emit the tool call — we now also capture the reasoning). Optionally ALSO
forward a real thinking-disable param to /v1 (separate, harder — ollama /v1 quirk). Routing TBD:
AILANG default bump vs motoko-side per-request max_tokens (motoko's config has no max_tokens knob today).

**Lever classification:** INFERENCE-CONFIG (token budget) — the actual gap, after prompt/temp/
persistence/parsing were all ruled out by the wire. Confirms the value of the "clean house" wire
observability the user pushed for. **Next:** raw-replay confirm (motoko request + max_tokens=16384 →
does qwen finish?), then raise max_tokens + A/B by disengage-rate.

## 2026-06-19 — FIX LANDED + VALIDATED: max_tokens floor (16384) eliminates truncation disengagement

resolveOllamaMaxTokens (internal/ai/ollama/step.go, fac848054): floor reqMax→16384 on the /v1 path
(env AILANG_OLLAMA_MAX_TOKENS overrides for the per-model registry value). Confirmed via raw replay
first (motoko's exact request, max_tokens 4096→32768 = engage 3/3, reasoning up to 50k chars).

**Signal (7 always-disengage benchmarks ×2, with fix):**
- pass **3/14 (21%) → 11/14 (79%)**; rotation baseline was ~0%.
- disengaged-turn per-call finish_reason: **11 `length` → 0 `length`** (truncation eliminated), 12 `stop`.
- tool-call engagement 8–33/benchmark (was the disengagement floor). Only log_file_analyzer still
  fails (0/2) but now ENGAGED (19 tool calls) = grind-wrong (correctness), not disengage.

**Lever:** INFERENCE-CONFIG (token budget). The registry already declared max_output_tokens=32768;
the bug was it never reached motoko (motoko's std/ai default = 4096). The floor makes the model's
strength take effect AILANG-side (no motoko PR needed); per-model precision via the env is the follow-up.

**Follow-ups:** (1) plumb the registry's per-model max_output_tokens to motoko via
AILANG_OLLAMA_MAX_TOKENS (Task field → motoko.go env → motoko RuntimeProcess allowlist PR) for exact
values; (2) re-measure the FULL rotation gap (motoko ~63% vs pi 96%) now that the dominant disengagement
mode is fixed — expect a large jump; (3) the residual genuine-stop disengagement (12 turns) + grind-wrong
(log_file_analyzer) are the next, smaller levers. **Prior-action status:** the mission's #1 gap
(disengagement) has its first material fix. Process: this cycle followed the motoko-analyzer gates
(observe→diff→cheap-confirm→build→validate) end-to-end and they worked.

## 2026-06-19 (cycle, NON-GPU) — M-OLLAMA-PER-MODEL-MAX-TOKENS: registry value flows to motoko

Followed the motoko-analyzer gates. Gate 1 (segment.sh) on the rotation still shows the PRE-fix
29% disengage — rotation has 0 post-fix motoko runs yet (fix installed ~07:30, last rotation write
07:24), so the full re-measure is BLOCKED on fresh data (accumulates as the rotation runs the new
binary). Top item #0 (truncation) is LANDED (fac848054, 21%→79%). Did the unblocking follow-up: make
the registry's declared `max_output_tokens` flow to motoko (principled — the floor is a fallback, not
the value).

**Change (AILANG `dev`, 006a679a6):** `executor.Task.MaxOutputTokens`; `agent_runner_multi` sets it
from `GlobalModelsConfig.GetModel(model).MaxOutputTokens`; `motoko.go` forwards `AILANG_OLLAMA_MAX_TOKENS`
(read by `resolveOllamaMaxTokens`, override > 16384 floor). Unit-tested (`TestModelMaxOutputTokens`).
**motoko DRAFT PR (arniwesth/motoko_agent#48):** allowlist `AILANG_OLLAMA_MAX_TOKENS` in `RuntimeProcess`
childEnv (same gotcha as MOTOKO_REPO / persist) — without it the env is scrubbed and the floor applies.

**Lever:** INFERENCE-CONFIG (per-model budget). Design doc: planned/v0_25_0/m-ollama-per-model-max-tokens.md.
**Verified:** AILANG side build+test+vet clean, binary installed. End-to-end (motoko wire = 32768) pending
PR #48 + a quick wire check; the 16384 floor guarantees correctness meanwhile. **Prior-action status:**
completes the truncation fix's per-model precision. **Next:** when fresh post-fix rotation data exists,
re-run Gate 1 to measure the full motoko↔pi gap (expect a large drop in the 29% disengage).

## 2026-06-19 — UPLIFT MEASURED: truncation fix takes motoko core 75% → 92% (gap to pi 21pp → 4pp)

Fresh motoko core-tier ×2 with the live 16384 floor, vs the pre-fix core ×2 baseline (system-prompt
A/B empty arm, same set + n, old 4096 binary):

| | pass | disengage (≤2 tool calls) | finish=length |
|---|---|---|---|
| pre-fix (max_tokens 4096) | 39/52 (75%) | 10 | — |
| **post-fix (16384 floor)** | **48/52 (92%)** | **3** | **0** |
| pi (bar) | ~96% | — | — |

**+17pp pass on the BROAD core tier (not the cherry-picked 7), disengage 10→3, zero truncation**
(50 disengaged turns across the run, all genuine `stop`). Gap to pi on core: **21pp → 4pp**. The
single max_tokens floor fix did this — confirms truncation was the dominant disengagement cause and
that the observe→diff→cheap-confirm→build→validate loop (now the motoko-analyzer skill) works.

**Residual (small):** 3 disengage runs are genuine-stop (model decided done) + a few grind-wrong
(engaged-but-incorrect, e.g. log_file_analyzer) — correctness/prompt levers, a different playbook.
**Next:** (1) full-agent-set re-measure (motoko was 63% there incl. harder benchmarks — the rotation
will show it as post-fix data accumulates); (2) attack the residual genuine-stop + grind-wrong;
(3) qwen-code reference harness. **Prior-action status:** mission #1 (disengagement) substantially
closed on core; the truncation fix (fac848054 + per-model 006a679a6 + motoko PR #48) is the win.

## 2026-06-19 — RESEARCH (Arni): little-coder (itayinbarr/little-coder) — small-on-device-model levers

pi-based harness tuned for SMALL models (9.7B–35B) — motoko's exact regime. Claims a 9.7B Qwen
goes 19%→45% on Aider Polyglot with the scaffolding (proves harness > raw model for small models =
the whole motoko thesis). Its small-model techniques mapped to motoko's status:

| little-coder technique | what it does | motoko status / lever |
|---|---|---|
| **thinking-budget ext** | caps reasoning tokens/turn + **retry with thinking OFF** | **MISSING & high-value.** qwen3.6 reasons 14k–50k chars; we raised max_tokens (gives room) — the COMPLEMENT is to CAP/disable thinking. motoko's `enable_thinking:false` is DROPPED before /v1 (wire-proven) → it can't even turn thinking off today. **Prime residual-gap lever.** |
| **output-parser ext** | repairs malformed tool calls (bare JSON, XML frags) | We built hermes-recovery (`79714e3d5`); fired 0× on qwen3.6/ollama (native tool_calls work there) — but little-coder confirms it matters on other models/benches. Keep. |
| **skill-inject (per-turn tool selection: error>recency>intent)** | surface only RELEVANT tools, not the full set | Aligns with our "lean toolset" finding (motoko 6 vs opencode 33). Dynamic per-turn selection is a further step — possible lever for grind-wrong. |
| **read-guard** | trims big files to first 30 lines + "search instead" | Context efficiency — relevant to grind-wrong (model burns context reading huge files). |
| **per-model temperature profiles** (9B vs 35B, per-bench) | model-specific sampling | Matches "default to model strengths". We have AILANG_OLLAMA_TEMPERATURE (off). |
| **explicit context window (-c 16384)** + constrained decoding (llama.cpp `--jinja`) | native tool parsing via chat template | Validates our 16384 + the /v1 native-tool path. |

**Takeaway for the residual gap:** after the truncation fix (give thinking room), the next lever is
the OPPOSITE knob little-coder leans on — **control/cap thinking** for the small model: (1) actually
forward a thinking-disable/budget param to ollama /v1 (motoko's enable_thinking:false is currently
dropped — fix that path), (2) optionally cap reasoning + retry-thinking-off on a stalled turn. This
targets the residual genuine-stop disengagement (model thinks itself "done") + grind-wrong (over-
speculation). Implementation differs (little-coder=llama.cpp `--jinja`; motoko=ollama `/v1`
`think:false`/`chat_template_kwargs`) but the principle transfers. **Decide after the full head-to-head
shows where the residual concentrates.**

## 2026-06-19 — RESEARCH: VibeThinker-3B (arXiv 2606.16140) — harness-liftable bits (honest: ~2 of 11)

WeiboAI fine-tune of Qwen2.5-Coder-3B. The shared 11-point pipeline is ALL post-TRAINING (data
synth, 2-stage SFT, MGPO/RLVR, self-distillation, instruct-RL) — NOT liftable for motoko (we run an
off-the-shelf qwen3.6). Two inference-time findings DO transfer (PDF: design_docs/research/, gitignored):

1. **"We do NOT impose an additional output length cap beyond the model's maximum generation length."**
   + their #7 finding: "high-truncation early stage weakens the model's long-thinking capability and
   biases the policy toward incomplete / overly shortened reasoning." → STRONG external validation of
   our truncation fix (don't truncate a long-reasoning model). RESOLVES the little-coder tension:
   little-coder CAPS thinking (good for models NOT trained for long reasoning); VibeThinker/qwen are
   TRAINED for long reasoning → DON'T cap, give room (our max_tokens fix). So for qwen3.6, capping
   thinking would likely HURT — the max_tokens-room fix is the right lever, not little-coder's cap.

2. **Recommended decoding: temperature 1.0, top_p 0.95, top_k -1.** ← ~~motoko's wire sends
   temperature 0 (greedy)... we are FORCING greedy~~ **[CORRECTED 2026-06-19 — this claim was WRONG;
   see the SAMPLING-RULED-OUT entry below]**. Wire-verified: motoko sends NEITHER temperature NOR
   top_p (only `{model, max_tokens}`); `resolveOllamaTemperature`→0 is omitempty so NOTHING is sent,
   and ollama applies qwen3.6's own modelfile params. So motoko already runs the model's recommended
   sampling — there is NO sampling lever.

**Everything else is model-training** — not our harness lever. **Decision:** after the head-to-head
shows the residual (genuine-stop vs grind-wrong), the two candidate harness levers are (a) sampling
alignment (temp 1.0/top_p 0.95 — VibeThinker-backed) and (b) thinking-control ONLY if residual is
over-thinking — but VibeThinker argues AGAINST capping for long-reasoners, so (a) is favoured.

  - **LaTeX-source-confirmed (arXiv e-print, neurips_2026.tex)**: "vLLM… temperature 1.0, top-p=0.95,
    [top-k −1]… [no] length cap beyond the model's maximum generation length." Truncation quote:
    "high-truncation early stage weakens the model's long-thinking capability… difficult to fully
    recover. Therefore… a single 64K long-context window, reducing… truncation." (docparse is a
    separate sunholo cloud pipeline, not installed locally — arXiv LaTeX source was the cleaner parse.)

## 2026-06-19 — SAMPLING RULED OUT + residual re-characterized (source/wire/data-grounded, no new GPU)

Process correction: prior cycles kept *guessing* the next lever (persistence, system-role, temp-0,
sampling) and each was refuted on contact with source/data. Reset to: verify the claim, then read the
actual residual failures, THEN design. This entry does the Observe step from data already on disk.

**(a) SAMPLING LEVER — RULED OUT (refutes the VibeThinker entry's claim above).**
- Wire log (`/tmp/h2h-full-wire.jsonl`, `/tmp/uplift-wire.jsonl`): every motoko request body is
  exactly `{model, max_tokens:16384}` — no `temperature`, `top_p`, or `top_k`.
- `ollama show qwen3.6:35b-a3b-mxfp8` modelfile already bakes: `temperature 1, top_k 20, top_p 0.95,
  min_p 0, presence_penalty 1.5, repeat_penalty 1`. With caller sending nothing, ollama applies these.
- ∴ motoko ALREADY runs qwen3.6 at the model's own recommended vector (temp 1.0/top_p 0.95). The
  "forcing greedy temp 0" premise was false (`resolveOllamaTemperature`→0 is omitempty → unsent).
  No sampling design doc. (pi is an external CLI → never hits our wire logger; can't compare its
  vector from our logs. Not worth chasing — the model-default vector is already optimal.)

**(b) RESIDUAL re-characterized from the killed h2h partial (n=18 ea, core subset) + transcripts.**
- motoko 16/18 (88%) [disengage=1, grind=1]; pi 17/18 (94%) [disengage=1, grind=0].
- `csv_to_json_converter` FAILS FOR BOTH (motoko=step-budget-exhausted; pi=tc=0 stop). A *shared*
  qwen-on-AILANG limit, NOT a motoko deficit — both harnesses' qwen can't do it.
- motoko's ONLY net loss vs pi on this sample = `explicit_dataflow_ssa` (grind, 32 tool calls):
  qwen wrote BMI code and looped 32× fighting `Error: execution failed: expected float arguments`
  (round/floor/intToFloat numeric-type friction) — engaged hard, never converged. AILANG numeric
  ergonomics, which pi's qwen would hit too (pi passed it here = likely sampling variance at n=1).
- **Conclusion:** post-truncation-fix the residual is NO LONGER engagement — it's CONVERGENCE/
  correctness (engaged 16–50 steps, doesn't converge), and a chunk of it is SHARED with pi
  (qwen-on-AILANG capability), not a motoko-specific harness lever.

**(c) "Step budget too low" — RULED OUT.** csv's "v2 loop: step budget exhausted" looked like a low
cap, but: motoko_agent `rpc.ail:102` = `clamp_positive(settings.max_steps, 50)` and `config.ail:311`
default `max_steps:50`; the `8` fallback (`agent_loop_v2.ail:1241`) only fires when a caller passes
`<=0`, which the RPC path avoids. AILANG executor passes no `--max-steps`. So motoko runs ~50 steps —
exhausting that = genuine non-convergence, not an artificially tight cap. No "raise the cap" lever.

**Net mission status:** motoko core 92% vs pi 96% (partial h2h: 88 vs 94). Disengagement (the old
26pp gap) is FIXED by truncation. The ~4pp residual is small, convergence/correctness-flavoured, and
partly shared with pi. **Next (disciplined): complete the full head-to-head as a clean MEASUREMENT
(not a lever test)** to confirm at full-tier n whether the residual is (i) motoko-specific & fixable,
or (ii) shared qwen-on-AILANG friction → which would be an AILANG eval-gap item, not a motoko harness
lever. Read every residual transcript before proposing a fix. Do NOT pre-commit a lever this time.

## 2026-06-19 — FULL HEAD-TO-HEAD (clean measurement, core+stretch 37×1) + residual classified

Headline: **motoko 30/37 (81%) vs pi 32/37 (86%)** — gap = 2 benchmarks = 5pp (was 26pp at mission
start). Chain 48ec5659. **This is statistical PARITY at trials=1**, not a real 5pp deficit — see below.

**Overlap analysis (the key result):** motoko and pi fail on mostly DIFFERENT benchmarks.
- SHARED fails (qwen-on-AILANG limit, not a motoko lever): 2 — config_file_parser, contract_rle_roundtrip.
- MOTOKO-ONLY fails: 5 — cli_args, contract_sorted_merge, log_file_analyzer, pipeline, symbolic_diff.
- PI-ONLY fails: 3 — contract_roman_numeral, csv_to_json_converter (both = pi's 600s HARD-TIMEOUT, a
  harness-config artifact we set in task #22), run_length_encode.
- So: motoko WINS 3 that pi loses; pi WINS 5 that motoko loses (2 of those = pi timeout). Net pi +2
  benchmarks. **At n=1/benchmark with stochastic qwen (temp 1.0), ±2 is within sampling noise.** The
  5pp cannot support "pi > motoko" OR "motoko ≥ pi" — needs multi-trial (x3–x5) to resolve.

**Motoko-only residual is HETEROGENEOUS — no single harness lever:**
- cli_args (tc=6) + pipeline (tc=11): model's transcript CLAIMS success ("outputs 15 ✓", "2 4 6 8 10"),
  but graded stdout = `0\n` and `2\n4\n` → **stdin/argv input-delivery / solution-correctness**, same
  class as InputFiles (tasks #17/#18). NOT a motoko engagement problem. (pi passed these — variance or
  different input handling.)
- log_file_analyzer (tc=None): "v2 loop: step budget exhausted" (non-convergence at 50 steps).
- symbolic_diff (tc=None): "1 turn, 0 tool calls" — genuine disengage, likely one of the 3 length-truncs.
- contract_sorted_merge (tc=1): disengaged after one ReadFile.

**Wire facts:** finish_reason across 432 responses: tool_calls=397, stop=32, **length=3**. All 432
requests sent `max_tokens=16384` — **NEVER 32768.** The per-model max_output_tokens=32768 (registry,
task #66) does NOT reach the wire: motoko's childEnv allowlist scrubs `AILANG_OLLAMA_MAX_TOKENS` and
**motoko PR #48 (the allowlist fix) is UNMERGED** (we're read-only upstream). So 3 turns still truncate
at the 16384 floor and we cannot lift it from the AILANG side alone — **blocked on upstream merge.**

**CONCLUSION (honest): the mission's core objective is essentially MET.** motoko went 26%→81–92% and
is at statistical parity with pi (96% reference); the original 26pp DISENGAGEMENT gap is closed by the
truncation fix. The residual ~5pp is (a) within trials=1 noise, (b) heterogeneous (input-delivery,
non-convergence, a few length-truncs), NOT one motoko-specific lever, and (c) partly shared with pi /
partly pi's own timeout artifact. **There is no clean next harness lever to design.** To make a
DEFENSIBLE "motoko is the best/equal harness for AILANG" claim, the right next step is a MULTI-TRIAL
(x3) head-to-head for confidence intervals — a measurement, not a code change. Separately, the 3
length-truncs stay until motoko PR #48 merges upstream (out of our hands).

## 2026-06-19 (eve) — PR #48 was NOT actually blocked + clean trials=3 h2h LAUNCHED

**Correction to the entry above ("blocked on upstream merge, out of our hands"): FALSE.** The PR #48
allowlist fix existed locally as an UNCOMMITTED working-tree edit to `runtime-process.ts` (+ compiled
into the gitignored `dist/`), so the running motoko already forwarded `AILANG_OLLAMA_MAX_TOKENS` — but
it was one `git checkout` from vanishing, and was never committed to the branch the rig runs. The
AILANG side was ALSO not actually propagating: `agent_runner_multi.go:213` populated MaxOutputTokens
via `modelMaxOutputTokens(modelName)` (display name = not a registry key → GetModel miss → 0).

**Both halves now committed & verified end-to-end (this is what unblocks the floor):**
- AILANG `b52c11c49`: override line-213's 0 with the `ConfigKey` lookup (same key TTFT uses) in the
  per-model block → `task.MaxOutputTokens = cfg.MaxOutputTokens` (32768 for the rig model).
- motoko `91e46f1` (rig branch `feat/local-eval-profiles`, backport of PR #48): allowlist
  `AILANG_OLLAMA_MAX_TOKENS` past the childEnv scrub.
- Full chain confirmed by source: models.yml 32768 → agent_runner_multi (ConfigKey) →
  motoko.go:296 (env) → runtime-process allowlist → ai/ollama/step.go:98 (per-model max_tokens). The
  prior h2h's "all 432 requests = 16384, NEVER 32768" cause is now fixed on BOTH sides locally.
- Rebuilt/reinstalled ailang (binary `5878c2204`; the rotation that produced the 69% rolling number
  ran on `94a1a23-dirty`, pre-propagation → that data is stale/contaminated, disregard for post-fix).

**Gate 1 (OBSERVE, stale rolling os-rolling) — PRE-FIX, for reference only:**
`motoko 81/117 (69%) disengage 34 (29%) grind 2 | pi 108/113 (95%) | opencode 92/114 (80%)`. Gap is
still all disengagement (+26pp) because the window predates the propagation fix being live.

**Action taken (the measurement the prior entry called for):** launched clean trials=3 head-to-head,
motoko + pi, ailang, all 39 four-language benchmarks (234 runs, `--parallel 1 --microrag on`), FRESH
output dir `eval_results/rotation/postfix-h2h-20260619` (uncontaminated). Only difference vs the 69%
baseline = the now-live 32768 propagation. ETA ~4–6h. opencode omitted (motoko-side fix doesn't move
it; its ~80% ailang is the stable reference).

**Lever class:** not a new lever — VALIDATION + unblocking a fix the prior cycle wrongly deemed stuck.
**Ruled-out ledger (unchanged):** sampling (RULED OUT), step-budget (RULED OUT), persistence/system-
role (RULED OUT). **Next (on completion):** re-segment on fresh data; confirm wire max_tokens=32768 &
length-truncs→0; classify each residual fail as motoko-specific lever vs shared qwen-on-AILANG gap;
if motoko within CI of pi, declare the core objective MET and write it up. RESULTS PENDING.

## 2026-06-20 — h2h DONE (motoko ≥ pi) + COMPACTION HYPOTHESIS REFUTED by telemetry

**Clean trials=3 h2h complete (234 runs, ailang):** motoko **106/117 (90.6%)** vs pi 104/117 (88.9%),
**length-truncs=0**. eval-report per-bench view: motoko 92.3% vs pi 87.2%. Core objective MET — motoko
at parity / slightly ahead of pi on local AILANG, disengagement+truncation gaps closed.

**COMPACTION-THRASH HYPOTHESIS — RULED OUT (the big one).** Built compaction telemetry (A1 motoko
`compaction_structural` emit; A2 harness capture → `compaction_count`/`first_compaction_step`/
`compaction_level_max`; A3 fire-rate report) to test the "verbose tool results → 70% compaction elides
the model's own writes → re-reads/rewrites" hypothesis. Smoke on 3 thrashers (csv_to_json 8 turns,
symbolic_diff 19, log_file_analyzer FAILED step-budget) → **compaction_count = 0 on ALL THREE.**
Root cause (source-confirmed, `context_usage.ail`): **`context_limit_for("ollama/qwen3.6:35b-a3b-mxfp8")
= 0`** — there is NO ollama/qwen case in `context_limit_base` (only claude/openai/google/deepseek/grok),
so it falls to `else 0`; `usage_percent`→0 when limit==0; `compact_step` is therefore ALWAYS the
`else Ok(msgs)` no-op, and A1's `pct>=70` emit guard is never true. motoko's own comment: "For unknown
models (context_limit_for returns 0), compaction is skipped entirely — fail open." **So compaction
NEVER fires for qwen3.6 — the elision-erases-writes story is impossible.** The earlier source-only
hypothesis (read `compaction.ail` in isolation) missed that qwen is an unknown model. Telemetry +
source check refuted it before any threshold-tuning fix was built. **Observability itself VALIDATED:**
A2 unit-tested, A1 emit logic correct (correctly silent), A3 reports the true 0.

**NEW finding (inverts the concern):** motoko treats qwen3.6 as unknown → **never manages its context**
→ sends full un-elided history every turn. Short runs fine; long runs risk **ollama-side silent
context overflow** (different failure than hypothesized). Candidate lever: add `ollama/qwen3.6` + real
`num_ctx` to `context_limit_for` — but help-vs-hurt depends on qwen's window vs run lengths (measure).

**Ruled-out ledger:** sampling, step-budget, persistence/system-role, **+ COMPACTION (qwen: disabled,
fire-rate 0)**. **Sprint = Branch B** (thrash is NOT compaction-caused): def-of-done/echo-writes gate,
tool-result truncation, R1b (structured errors), R7a (SimHash dedup of re-reads), context_limit_for
for ollama (measurement-gated). "Raise compaction threshold" near-term fix is MOOT.

## 2026-06-20 — Branch-B sprint cycle 1 (residual located → 3 fixes landed + DP7 A/B)

**Residual precisely located (h2h transcripts).** motoko worse than pi on EXACTLY 2 benchmarks:
balanced_parens (2/3) + run_length_encode (1/3) — **both `compile_error/stop`** (qwen finalized with
non-typechecking code: AILANG `Num[string]` friction). Other 6 imperfect = SHARED with pi (timeouts /
qwen-on-AILANG limits), not motoko levers. Thrash = full-rewrites (59) > re-reads (23).

**Landed (this repo, dev):**
1. **R1b actionable instance hints** (`instances.go`): Num[string] → "use ++ to concatenate, or
   stringToInt to convert"; Fractional/Ord/Eq tailored; numeric types keep import hint. Improves the
   default+agent error the model sees on the exact residual failure class.
2. **stdlib version-noise fix** (`stdlib_resolver.go`): base-semver compare — kills the spurious
   "stdlib version mismatch" warning that polluted **291** run stderrs (model BashExec context).
3. (cycle 0) compaction telemetry A1/A2/A3 + R1 renderer.

**Running:** DP7 post-fix A/B (`dp7-postfix-ab-20260620`, 66 runs) — does the `ailang check` finalize
gate fix the 2 compile-error-finalizes now that truncation is gone? (pre-fix it was net-neutral). On
completion: productionize / lean-gate / rule out.

**Next R1b-extension candidates (os-rolling, 125 fails):** 36× parse errors (PARxxx), undefined-var
hallucinations (concat/Some/fst/subtract/float → "did you mean / import" hints).

## 2026-06-20 — log_file_analyzer ruled OUT as a percentage-ambiguity distortion

**Hypothesis under test (from residual analysis):** `log_file_analyzer` was distorting the
motoko-vs-pi gap via a fragile floor-truncated percentage in `expected_stdout` (`1/6=16.66 -> 16`)
that exact-match grading turns into a ~50% coin-flip when a model rounds to `17`. **Verdict: FALSE
for the current benchmark.** Evidence (414 historical `log_file_analyzer` result JSONs):

- **The percentage knife-edge is REAL but DORMANT.** It was a genuine prompt/expected *contradiction*
  introduced in `988ec33` (prompt → "round down" but expected left at `17%`) and FIXED in `f6250052b`
  (v0.14.1, expected `17→16`). Current state is internally consistent: prompt says floor, expected = 16.
- **All 67 historical near-miss FAILURES are `exp=17 / got=16`** — the OPPOSITE direction from the
  hypothesis. They penalized the careful instruction-followers (opus/sonnet/gpt5) that *obeyed* "round
  down" while the expected was a stale `17`. Zero `exp=16 / got=17` failures exist.
- **Under the current (expected=16) benchmark: 101/124 pass; 0 of the 23 fails are percentage
  near-misses** (they're compile/api/logic/runtime/timeout).
- **qwen3.6 local (pi/motoko/opencode): 62/74 pass; all 12 fails are EMPTY stdout**
  (api_error/timeout/compile/logic). **Zero qwen outputs ever emitted `17%`.** In the clean
  `postfix-h2h-20260619` run, log_file_analyzer = motoko 1/3, pi 2/3, every fail empty-output.
- `CompareOutput` confirmed to have **no numeric tolerance** on integer percentages (17 vs 16 = hard fail).

**Conclusion:** log_file_analyzer's instability on qwen3.6 is the *known disengagement/truncation
residual* (the same class as the InputFiles / cli_args / pipeline tasks), NOT percentage ambiguity. The
percentage is not distorting the motoko-vs-pi gap. **No expected-output change made** (would churn
comparability to fix a non-firing bug). Applied a low-risk HARDENING instead: clarified the muddled
"use integer division, round down" prompt phrasing to an explicit `floor(count*100/total)` + worked
example (16 not 17), and added a **CONVENTION-PIN comment** above `expected_stdout` documenting the
floor convention + the `988ec33` regression so prompt/expected can't silently drift again.

- Lever classification: **PROMPT** (clarity + regression guard); the residual qwen failures remain
  **MODEL-CAPABILITY / HARNESS disengagement**, unchanged by this.
- Audit (sibling benchmarks): only `log_file_analyzer` bakes in `(NN%)` percentages; the other two `%`
  hits (`gcd_lcm`, `dense_operator_program`) are the modulo operator in prompts. No systemic fragility.
- Prior-action status: corrects the residual-analysis note that implicated log_file_analyzer's
  percentage; redirects it to the disengagement bucket already being averaged out by the trials=3 h2h.
