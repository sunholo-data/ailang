# Sprint Plan — M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT

**Design doc**: [../m-ollama-v1-streaming-idle-timeout.md](../m-ollama-v1-streaming-idle-timeout.md)
**Sprint ID**: `M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT`
**Tracking**: ailang#618 · target v0.34.0 · **P0**
**Planned at**: HEAD `3e1f63f7a` (branch `dev`), 2026-08-10
**Baseline for every "BASE:" row below**: `3e1f63f7a`, working tree clean except
`.claude/fmt_hook_events.jsonl`, `docs/static/benchmarks/os/*.json` and five untracked
`docs/docs/prompts/v0.16.*.md` — none of which touch a Go package in scope.
**Duration**: **3 days, 4 milestones (M1–M4)** — the design doc says 1–2 days / 3 milestones.
**This plan REFUTES that estimate.** See §0 and §6.
**Risk**: **High** — two refutations below show the doc's central "just swap `Step`→`StreamStep`,
everything downstream is unchanged" premise is false in two independent ways, one of which makes
the feature inert in the exact configuration the rollout creates.
**Executor lane**: opus-required.

---

## 0. Reader's summary — what a planner changed, and why

This plan does not re-open the design. The design doc is the reviewed artifact and wins on any
disagreement (mission rule vii). Two findings below are **rule-vii escalations**: they are not
preferences, they are places where the doc as written cannot produce a working feature. Both are
flagged for the controller in §7 rather than silently "fixed".

**Five things the executor must know before writing a line of code:**

1. **`go build ./...` is rc=1 on unmodified `dev`.** Re-confirmed first-party (§2.1) — `cmd/wasm`
   and `gen/main` have no native `main`. It can never be an acceptance gate here. The green-at-base
   substitute `go build ./internal/... ./cmd/ailang` (rc=0) is used everywhere instead.
   *Methodology note for the executor:* I initially measured this through `go build ./... | tail -5`
   and read rc=0, because `$?` after a pipe is `tail`'s status. **Never measure a build/test rc
   through a pipe.** Redirect to a file, then read `$?`.

2. **REFUTATION #1 — the `/v1` path is ALREADY bounded by an outer context deadline that the design
   doc never mentions, and it will make the feature inert on the rig.** `Step` wraps the *entire*
   call in `ollamaCallContext(ctx)` at **step.go:266** — `context.WithTimeout(ctx, ollamaV1Timeout())`,
   **default 300s** — *before* the `/v1` branch is reached at :283. The doc's Solution Design §2 says
   the streaming client "supplies `Timeout: 0` (V12)". **That is necessary but not sufficient**: the
   300s *context* deadline survives and kills the stream anyway. Worse, the doc's own rollout makes
   this the default rig state — M4 *mandatorily removes* the `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800`
   plist pin, leaving the var **unset**, at which point `ollamaCallContext` = 300s while the new hard
   deadline = 3600s, so the effective bound is **300s** and the rig reproduces today's
   `took=4m59.97` failure signature exactly, with the flag on. **Handled in M2** (capture the
   pre-`ollamaCallContext` ctx; the streaming branch derives from it). Gated by **AC-M2.4**, which is
   discriminating and sub-second — see the note there, because a naive test of this *cannot* bite.

3. **REFUTATION #2 — `StreamStep` is NOT response-equivalent to `Step`, and the difference is the
   exact failure mode this whole ollama line of work exists to fight.** The doc asserts (Solution
   Design §3) that "`StreamStep` returns the same `*ai.Response` shape as `Step` … so
   `logOllamaResponse` and everything downstream is unchanged". Same *shape*, different *semantics*:
   - `ParseChatStepSSEStream` **never sets `out.Reasoning`** (grep `Reasoning` in
     `streamstep.go`: hits at :63,151-173,272-284 only — all callback/struct-tag, no `out.Reasoning =`).
     The buffered `openai.Step` **does** (`step.go:587-588` → `ai.Response{Reasoning: reasoning}` at :633).
   - Far more serious: buffered `openai.Step` runs a **Hermes tool-call recovery** at
     **step.go:610-616** — `if len(toolCalls) == 0 { extractHermesToolCalls(text + "\n" + reasoning) }`
     — whose own comment says it exists because *"Ollama's `/v1` sometimes fails to lift [a
     `<tool_call>` block] into the tool_calls field for **Qwen3 thinking models**"*, and calls the
     thing it prevents *"the '0 tool calls / disengagement' failure mode"*. `ParseChatStepSSEStream`
     has **no** such recovery, and it discards the reasoning text where qwen3 puts the block.
   So the doc's swap silently deletes an ollama-qwen3-specific mitigation for motoko disengagement.
   The doc's **S6 as written would not catch it** ("the same response content served once buffered
   and once as SSE") — an implementer using a native-`tool_calls` response passes S6 green while the
   regression ships. **Handled in a new M3**, which the doc does not have, and S6 is strengthened
   into a three-case table. This is the main reason the estimate moves from 1–2 days to 3.

4. **The doc's M3 "capture a fixture" task is genuinely already DONE** — confirmed:
   `internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse`, 104 lines, 13,077 B, one-chunk
   tool call, `finish_reason:"tool_calls"`, usage `294/80/374`, one `[DONE]`. Re-scoped; not planned again.

5. **This planning workspace is NOT socket-restricted** (§2.3): `net.Listen("tcp","127.0.0.1:0")`
   succeeded, and `go test ./internal/ai/ollama/...` ran **23 `--- PASS`** tests, twelve of which
   already use `httptest`. That last fact refines the controller's sandbox note: if the executor
   sandbox denies loopback binds, then the **pre-existing** ollama suite is uninformative under
   sandbox too, at base — the label is a property of the package, not of the new tests. Every
   socket-bearing gate below therefore carries an **UNINFORMATIVE UNDER SANDBOX** escape hatch, and
   records what informative output looks like so a denial is distinguishable from a failure.

---

## 1. Milestone → design-doc success-criterion correspondence (rule vii)

The doc's Success Criteria S1–S9 are the contract. Every one is owned by exactly one milestone.

| Milestone | Doc criteria closed | Doc lines | GPU? |
|---|---|---|---|
| **M1** — Idle/TTFT/deadline `io.ReadCloser` + watchdog + RoundTripper (no call sites) | S9; S2/S3 *mechanics at reader level* | 479–487 | No |
| **M2** — Flag-gated streaming branch + **mandatory hard deadline** | **S8**, **S5**, S1, S2, S3, S4 | 489–497 | No |
| **M3** — Response parity: `Reasoning` + Hermes recovery (**NEW — not in the doc**) | **S6** (strengthened) | — (see §7 E-2) | No |
| **M4** — Fixture replay + rig validation + stopgap removal + docs | **S7** | 499–509 | **YES** |

- **S8 (mandatory backstop) is owned by M2** — the milestone that introduces the deadline, per the
  controller's requirement. It is not deferred.
- **S5 (default-off byte-identical) is owned by M2** and has its own AC with a named mutation
  (AC-M2.1). It is not a footnote of another criterion.
- **S6 moves out of M2 into M3** because §0.3 shows it is not a one-line assertion; it is the gate on
  a whole milestone of parity work the doc does not budget.
- **Only M4 touches the GPU** (AC-M4.2, the `docx_reimplement` run). Everything else is unit-testable
  with `httptest` + `io.Pipe` + the committed fixture. **The controller must take the rig lock around
  M4 AC-M4.2 only.**

---

## 2. Pristine-tree baselines (rule 3e — a gate that is red at base measures the repo, not the change)

### 2.1 Build

| Command | rc at `3e1f63f7a` | Verdict |
|---|---|---|
| `go build ./...` | **1** (`cmd/wasm`, `gen/main`: "function main is undeclared") | **STRUCTURALLY RED. NEVER an acceptance gate.** |
| `go build ./internal/... ./cmd/ailang` | **0** | **ADOPTED** as the build gate everywhere below. |

### 2.2 Tests, vet, format, repo gates — all green at base

| Command | Base result |
|---|---|
| `go test ./internal/ai/ollama/...` | rc=0. Control: `-count=1 -v` → **23** `--- PASS` lines, so the instrument reads. |
| `go test ./internal/ai/openai/...` | rc=0, `ok … 0.392s`. |
| `go vet ./internal/ai/...` | rc=0 |
| `gofmt -l internal/ai` | 0 files |
| `make check-file-sizes` | rc=0 (`step.go` = **447** lines; the 800 gate has ~350 lines of headroom, M2 adds ~180) |
| `make check-boundaries` | rc=0 |
| `make check-changelog` | rc=0 (`## [Unreleased]` exists at `changelogs/v0.18-current.md:5`) |

**Vacuous-greenness warning.** Every `-run <NewTestName>` command in this plan returns **rc=0 with
zero `=== RUN` lines** at base (`[no tests to run]`). An AC written as "rc=0" would pass after a
revert that *deletes the tests*. **Every named-test AC below therefore carries a `=== RUN` count
assertion in addition to rc=0.**

### 2.3 Sandbox posture

`net.Listen("tcp","127.0.0.1:0")` → **BIND OK**. All socket-bearing baselines here ran informative.
The executor may run somewhere stricter; see §0.5. Do **not** rewrite tests to dodge sandbox policy —
report UNINFORMATIVE and let the controller re-run outside.

### 2.4 Rollout-target baselines (used by M4)

| Command | Base |
|---|---|
| `grep -c AILANG_OLLAMA_HTTP_TIMEOUT_SEC tools/launchd/dev.ailang.{os-rotation-filler,nightly-eval}.plist` | **1** each — **correctly red**, must reach **0** |
| `grep -c AILANG_OLLAMA tools/launchd/dev.ailang.{os-rotation-filler,nightly-eval}.plist` | **1** each — anti-vacuity **control fires** |
| `grep -rc AILANG_OLLAMA_V1_STREAM tools/launchd/*.plist` | **0** everywhere — must reach ≥1 in both |
| `launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC` | **`1800`** — **SECOND DELIVERY SITE, missed by this table's first draft.** See below |
| `launchctl getenv AILANG_OLLAMA_V1_STREAM` · `launchctl getenv AILANG_NOT_A_REAL_VAR` | **empty** each — the anti-vacuity control pair: the instrument discriminates set from unset, so the `1800` above is a measurement |
| `test -L ~/Library/LaunchAgents/dev.ailang.{nightly-eval,os-rotation-filler}.plist` | **not symlinks** — both are regular files. The repo copies are *source*, not the running config |
| `diff -q ~/Library/LaunchAgents/dev.ailang.*.plist` vs `git show HEAD:tools/launchd/…` | **identical** — the installed files match repo HEAD, i.e. pre-M4. Install is a manual `cp` + `launchctl load` (`tools/launchd/nightly-eval.sh:19-21`), so **a repo edit reaches the rig only when a human copies it** |

**The stopgap has TWO delivery sites and the three rows above this one only measure ONE.**
`AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800` is also a launchd **user-domain global**, set by
`launchctl setenv`, which **no plist edit touches**. The stopgap commit says so in its own body —
`git log -1 b67d415cd`: *"pinned in both rig plists (**also set live via launchctl setenv**)"* —
and this table's first draft measured only the plists. Consequence: the grep criterion can go fully
GREEN while the hazard stays live, and it did. Measured on the controller's in-flight GPU run with
the flag on and both plists already edited: every streamed request logged
`hard_deadline_sec = 1800` **and** `effective_deadline_sec = 1800`, not the 3600 default — i.e. the
inherited global became the hard deadline, which is the exact ~2941s-worst-case hazard AC-M4.3
exists to remove, one level up from where the criterion was looking. AC-M4.3 gains a fourth arm.

### 2.5 Load-bearing code facts, re-verified first-party at `3e1f63f7a`

| Fact | Where | Status |
|---|---|---|
| `ollamaV1Timeout()` returns **0 (= no timeout)** for `<= 0` | `ollama/step.go:29-39` | CONFIRMED |
| **`Step` wraps everything in `ollamaCallContext` (300s default) at :266, before the `/v1` branch at :283** | `ollama/step.go:266` | **NEW — not in the doc's V-log. See §0.2.** |
| `ollamaCallContext` has a second caller | `ollama/client.go:93` (`Generate`) | CONFIRMED — do not change its semantics |
| `/v1` client build + call | `ollama/step.go:293-296`, `v1.Step(ctx,&r2)` at **:309** | CONFIRMED |
| `StreamStep` at `:42`; `ParseChatStepSSEStream(body io.Reader,…)` at `:217`; `bufio.NewScanner(body)` at `:218` | `openai/streamstep.go` | CONFIRMED |
| `onChunk` fires at exactly **4** sites — `:255`,`:269`,`:281`,`:284` — **never** for tool-call deltas | `openai/streamstep.go` | CONFIRMED |
| `StreamStep` `defer httpResp.Body.Close()` | `openai/streamstep.go:100` | CONFIRMED — the wrapper's `Close` **is** called exactly once on the happy path (this is what makes S9 testable) |
| Only non-test `StreamStep` caller, passes `context.Background()` | `ai/handler.go:378` | CONFIRMED — **out of scope, must not change** |
| `ParseChatStepSSEStream` **never sets `out.Reasoning`** | `openai/streamstep.go` | **NEW — REFUTATION #2a** |
| Buffered `Step` sets `Reasoning` **and** runs `extractHermesToolCalls(text+"\n"+reasoning)` when `len(toolCalls)==0` | `openai/step.go:587-588, 610-616, 633` | **NEW — REFUTATION #2b** |
| `extractHermesToolCalls` is **unexported**, defined at `openai/step.go:657`, tested at `openai/hermes_test.go:38` | | CONFIRMED — M3 must export it |
| **`goleak` is in `go.sum` v1.3.0 but NOT in `go.mod`, and is imported nowhere** | | **CORRECTION to doc S9** — see AC-M1.4 |
| Fixture: 104 lines, 13,077 B, one-chunk tool call `get_weather{"city":"Paris"}`, usage `294/80/374`, one `[DONE]` | `ollama/testdata/ollama_v1_stream_toolcall.sse` | CONFIRMED |
| Existing tests that pin today's semantics and **must stay green unmodified** | `TestOllamaV1Timeout` (`step_test.go:179`, asserts `"0"` → `0` = disabled), `TestStep_ToolsViaOpenAICompat_TimesOut` (:133), `TestStep_ToolsViaOpenAICompat` (:72) | CONFIRMED — this is why the hard deadline needs its **own** resolver, not a change to `ollamaV1Timeout()` |

---

## 3. Refusal-branch mutation register

The new code's contract is largely *refusal*. Per the controller's requirement, **every refusal
branch gets exactly one neutering mutation of the form `if false && <cond>`** — never a deleted
block, because a mutant that fails to compile reds for the wrong reason.

| # | Refusal branch | Neutering mutation | Gated by |
|---|---|---|---|
| R1 | TTFT window expiry (first `Read`) | `if false && r.firstRead && elapsed > r.ttft {` | AC-M1.2, AC-M2.7 |
| R2 | Idle window expiry (subsequent `Read`s) | `if false && elapsed > r.idle {` | AC-M1.1, AC-M2.6 |
| R3 | Deadline reset on `n>0` | `if false && n > 0 { r.reset() }` | AC-M2.5 (S1 starvation) |
| R4 | Hard deadline applied | `if false && hard > 0 { ctx, cancel = context.WithTimeout(outer, hard) }` | **AC-M2.2 (S8)** |
| R5 | `AILANG_OLLAMA_HTTP_TIMEOUT_SEC <= 0` rejected in streaming mode | `if false && n <= 0 { return 0, errStreamDeadlineInvalid }` | **AC-M2.3** |
| R6 | Watchdog shutdown in `Close()` | `if false && r.stopped.CompareAndSwap(false, true) {` | AC-M1.4 (S9) |
| R7 | Streaming branch selected only when flag set | `if false && os.Getenv("AILANG_OLLAMA_V1_STREAM") == "1" {` | AC-M2.1 (S5, inverted: flag-ON case) |
| R8 | Hermes recovery on zero native tool calls | `if false && len(resp.ToolCalls) == 0 {` | **AC-M3.1** |
| R9 | Reasoning accumulation from thinking deltas | `if false && d.Text != "" { rb.WriteString(d.Text) }` | AC-M3.2 |
| R10 | Tool-call argument fragment concatenation | `if false && tcFrag.Function.Arguments != "" {` (`streamstep.go:`**`298`**) | AC-M4.1 |
| R11 | Usage mapping from the final chunk | `if false && chunk.Usage.PromptTokens > 0 {` (`streamstep.go:248`) | AC-M4.1 (controller-added) |
| R12b | **Fixture** tool-call arguments: line 97 `Paris` → `Berlin` | not an `if false &&` — the mutation target is the captured wire bytes, not a branch | AC-M4.1 (controller-added) |

R10 is a mutation of *existing* code, used only to prove the M4 replay test actually reads the
assembly path. It is a verification step, not a change to ship.

**R10's line number was wrong in this register's first draft** — it said `streamstep.go:299`, but
`:299` is the `WriteString` body; the predicate is at **`:298`**. Measured independently by the
executor and the controller.

**Finding (2026-08-11): R10 is NOT discriminating for AC-M4.1; R12b is.** Executed results:

| Mutation | Landed (sha256 differs) | Builds | Arm A — target test alone | Arm B — package `-skip` target |
|---|---|---|---|---|
| R10 | yes | rc=0 | **rc=1**, `Arguments = "{}"` | **rc=1** — pre-existing `TestStreamStep_ParsesToolCallFragments` also dies |
| R11 | yes | rc=0 | **rc=1**, `InputTokens = 0, want 294` | **rc=1** — pre-existing `TestStreamStep_ParsesContentAndUsage` also dies |
| R12b | yes, *and asserted to have landed on the mechanism* | n/a (data) | **rc=1**, `=== RUN` 1, `Arguments = "{\"city\":\"Berlin\"}"` | **rc=0** — **UNIQUE killer** |

**Why arm B failing is expected rather than a defect in the M4 test.** No mutation of
`openai/streamstep.go` can be unique to the replay test, because the package's existing tests
already cover parse, assembly and usage on hand-written SSE. AC-M4.1's object is the **captured
wire shape** — that today's real ollama emission still parses to `get_weather{"city":"Paris"}` with
usage `294/80/374` — so the discriminating mutation is one on the **fixture**, which is what R12b
is. The earlier instruction "arm B must stay rc=0 under R10" was an over-specification: this plan
only ever claimed R10 proves the test reaches the assembly path, which it does.

**Methodology note carried by R12b, worth more than the result.** The controller's first R12b
attempt edited line 17 — thinking text, not the tool call — and the test correctly stayed green.
A mutation must be asserted to have landed **on the mechanism under test**, not merely to have
changed the file's bytes; a sha256 diff proves the edit happened, not that it was aimed correctly.

---

## 4. Milestones

### M1 — Idle/TTFT/deadline reader + watchdog + RoundTripper (no call sites, no behaviour change)

**Est. LOC: 380** (impl ~170, tests ~210). **No GPU.**

**Files**: `internal/ai/ollama/idlereader.go` (new) · `internal/ai/ollama/idlereader_test.go` (new)

**Build**:
- An `io.ReadCloser` wrapper. **Must be a ReadCloser, not a Reader** — `StreamStep`'s
  `defer httpResp.Body.Close()` (`streamstep.go:100`) is what stops the watchdog; a bare Reader loses
  the underlying `Close` and leaves the timer armed.
- Watchdog: first-`Read` deadline = TTFT window; every `Read` returning `n>0` resets to the idle
  window. A blocked `Read` cannot interrupt itself, so the watchdog calls a supplied
  `context.CancelFunc`; the transport then fails the pending `Read`.
- **Which-window-fired is recorded on the wrapper via an atomic CAS (first writer wins)**, so the
  surfaced error is typed `ttft-timeout` / `idle-timeout`, not a generic `context canceled`.
- `Close()` stops the timer/goroutine **then** closes the underlying body; idempotent.
- Typed sentinel errors distinguishable by `errors.Is`: `ErrTTFTTimeout`, `ErrIdleTimeout`,
  `ErrStreamDeadlineExceeded`, `ErrStreamDeadlineInvalid` (config).
- A `RoundTripper` that wraps `resp.Body` in the reader.
- The three window resolvers: `AILANG_OLLAMA_IDLE_TIMEOUT_SEC` (default 120),
  `AILANG_OLLAMA_TTFT_TIMEOUT_SEC` (default 600), and a **separate**
  `ollamaStreamHardDeadline()` reading `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` with **default 3600 and
  `<= 0` REJECTED**. `ollamaV1Timeout()` is **NOT TOUCHED** — `TestOllamaV1Timeout`
  (`step_test.go:188-191`) pins `"0"` → disabled for the legacy path and must stay green.

**Transport-construction constraint (planner finding, not in the doc).** The RoundTripper needs a
base `*http.Transport` carrying `ResponseHeaderTimeout` = TTFT window. Do **NOT** mutate
`http.DefaultTransport` (process-global) and do **NOT** build a fresh `&http.Transport{}` per call:
a bare `http.Transport{}` has `IdleConnTimeout: 0` = *no limit*, and the `/v1` client is constructed
per request (`step.go:293`), so a per-call transport accumulates idle connections forever across
thousands of rig requests. Build the base transport **once** (package-level `sync.Once`, cloned from
`http.DefaultTransport` so `IdleConnTimeout: 90s` and proxy settings are inherited) and give each
request only the cheap per-request wrapper struct that delegates to it.

**Acceptance criteria** (each names its mutation; each is falsifiable):

- **AC-M1.1 (R2, doc S2 mechanics).** `go test -count=1 -v ./internal/ai/ollama -run 'TestIdleReader'`
  — an `io.Pipe`-driven case writes 3 chunks then goes silent; `Read` must return an error satisfying
  `errors.Is(err, ErrIdleTimeout)` within idle+slack (test windows ≤ 200ms via direct struct
  construction, not env).
  **BASE**: rc=0, `=== RUN` count **0**, `[no tests to run]` — VACUOUSLY GREEN. Pass requires rc=0
  **AND** `=== RUN` ≥ 6 **AND** a `--- PASS` per named subtest.
  **RED UNDER**: mutation **R2**. The watchdog never fires; the test blocks to the Go test timeout.

- **AC-M1.2 (R1, doc S3 mechanics).** Pre-first-byte silence longer than the idle window but shorter
  than the TTFT window, then data → the read **completes**; and a pre-first-byte silence longer than
  the TTFT window → `errors.Is(err, ErrTTFTTimeout)`.
  **BASE**: new by construction.
  **RED UNDER**: mutation **R1** (TTFT never fires → the second half returns no error), and
  separately under collapsing to one window (`r.ttft = r.idle`) → the first half false-trips.

- **AC-M1.3 (which-window-fired is preserved).** Three cases assert three *distinct* sentinels:
  `ErrTTFTTimeout`, `ErrIdleTimeout`, `ErrStreamDeadlineExceeded`. Assert with `errors.Is`, and
  assert pairwise **non**-equality (`!errors.Is(idleErr, ErrTTFTTimeout)`), so returning one generic
  error for all three cannot pass.
  **BASE**: new by construction.
  **RED UNDER**: returning a single generic error — the non-equality assertions fail.

- **AC-M1.4 (R6, doc S9 — leak-free `Close`).** After a fully-consumed happy-path stream,
  `Close()` (a) closes the watchdog's done-channel within 100ms, (b) calls the underlying body's
  `Close` **exactly once** (counting `io.ReadCloser` stub), and (c) a second `Close()` is a no-op
  that does **not** increment the count.
  **CORRECTION TO THE DOC**: the doc offers `go.uber.org/goleak` "already in `go.sum` at v1.3.0".
  It **is** in `go.sum` but **is NOT in `go.mod`** and is imported nowhere in the repo — adding the
  import is a dependency change requiring `go mod tidy` and a `go.mod` edit, which drags in
  Dependabot/CI surface for a single test. **Use the watchdog done-channel probe** (the doc's own
  stated alternative). Do not add goleak.
  **BASE**: new by construction.
  **RED UNDER**: mutation **R6** — the done-channel never closes, (a) fails.

- **AC-M1.5 (R5, config rejection at the resolver level).** Table test on
  `ollamaStreamHardDeadline()`: unset → 3600s; `"7200"` → 7200s; `"0"` → `ErrStreamDeadlineInvalid`;
  `"-1"` → `ErrStreamDeadlineInvalid`. **Plus the non-regression half**: `TestOllamaV1Timeout` passes
  **unmodified**, still asserting `"0"` → `0` for the legacy resolver.
  **BASE**: `TestOllamaV1Timeout` green (23-PASS control); the new table is new by construction.
  **RED UNDER**: mutation **R5** (the `"0"`/`"-1"` rows get 0 or 3600 instead of an error); and, in
  the other direction, changing `ollamaV1Timeout()` to reject `0` reds `TestOllamaV1Timeout`.

**Milestone boundary** (all rc=0 at base, must remain rc=0):
`go build ./internal/... ./cmd/ailang` · `go test -count=1 ./internal/ai/ollama` ·
`go vet ./internal/ai/...` · `gofmt -l internal/ai` → 0 · `make check-file-sizes`.
**Zero call sites are added in M1, so runtime behaviour is unchanged by construction.**

---

### M2 — Flag-gated streaming branch + MANDATORY hard deadline (owns S8 and S5)

**Est. LOC: 510** (impl ~180, tests ~330). **No GPU.**

**Files**: `internal/ai/ollama/step.go` · `internal/ai/ollama/step_test.go`

**Build**:
1. **Capture the pre-`ollamaCallContext` context.** At `step.go:265`, before
   `ctx, cancel := ollamaCallContext(ctx)`, add `outerCtx := ctx`. The streaming branch derives from
   `outerCtx`; **the flag-off branch and the native path keep using the 300s-bounded `ctx`, byte-for-byte
   as today.** This is REFUTATION #1's fix (§0.2). It is a two-line change plus one substitution and
   keeps the flag-off diff indentation-only.
2. Streaming branch under `os.Getenv("AILANG_OLLAMA_V1_STREAM") == "1"`:
   `streamCtx, cancel, cfgErr := streamCallContext(outerCtx)` — resolves the hard deadline
   (`ollamaStreamHardDeadline()`), returns `cfgErr` **before any HTTP request is made** on `<= 0`,
   otherwise `context.WithTimeout(outerCtx, hard)`.
3. `openai.NewClient(… WithHTTPClient(&http.Client{Timeout: 0, Transport: idleTransport(streamCancel)}))`
   — `Timeout: 0` per V12, the bound is the context.
4. `v1.StreamStep(streamCtx, &r2, onChunk)`. **`onChunk` is NOT nil** — see M3; in M2 pass a
   collector that only counts bytes/deltas for the debug log, and M3 gives it the parity job.
5. Error mapping with **explicit precedence**: (a) if the wrapper recorded a fired window (atomic CAS,
   first-writer-wins), surface that typed error; (b) else if `streamCtx.Err() == context.DeadlineExceeded`,
   surface `ErrStreamDeadlineExceeded`; (c) else classify normally. The precedence must be explicit
   because when the hard deadline fires, the idle timer may be racing.
6. **Per-request debug logging** (the falsifier data for Design Freeze #1/#2/#3): max inter-chunk gap,
   TTFT, total duration, **and the *effective* hard deadline read back from `streamCtx.Deadline()`**
   — read back, never the configured value. This last field is what makes AC-M2.4 possible.

**Acceptance criteria**:

- **AC-M2.1 (doc S5 — DEFAULT-OFF IS BYTE-IDENTICAL).** With `AILANG_OLLAMA_V1_STREAM` **unset**, a
  fake `/v1` server captures the raw request body and headers and asserts: **no** `"stream":true`,
  **no** `"stream_options"`, and `Accept` is **not** `text/event-stream`. Additionally,
  `go test -count=1 -v ./internal/ai/ollama -run 'TestStep_ToolsViaOpenAICompat$|TestStep_ToolsViaOpenAICompat_TimesOut|TestOllamaV1Timeout'`
  passes **with the test file's existing bodies unmodified** — rc=0 AND `=== RUN` ≥ 3 AND three
  `--- PASS` lines.
  **BASE**: those three tests are green today (part of the 23-PASS control); the new byte-identity
  test is new by construction.
  **RED UNDER**: mutation **R7 inverted** — defaulting the flag on (e.g. `!= "0"` instead of `== "1"`)
  puts `"stream":true` on the wire and reds the body assertion. Also red under any drive-by edit to
  the non-flag branch (the three unmodified tests).
  *Socket-bearing → UNINFORMATIVE UNDER SANDBOX if the listener is denied.*

- **AC-M2.2 (doc S8 — MANDATORY KEEP-ALIVE-FOREVER BOUND).** A fake SSE server writes
  `: keep-alive\n` every **20ms forever** (bytes flowing, never a parseable chunk — the V15/V16 shape).
  Test config: idle window **500ms**, hard deadline **2s** — emission interval well under the idle
  window (so the idle timer is genuinely reset and provably never fires) and hard deadline well above
  it (so the terminating error provably comes from the deadline). The call must return
  `errors.Is(err, ErrStreamDeadlineExceeded)` within 2s + 1s slack, and the test asserts
  `!errors.Is(err, ErrIdleTimeout)`.
  **BASE**: new by construction; `=== RUN` ≥ 1 required.
  **RED UNDER**: mutation **R4** — with no `context.WithTimeout`, keep-alive bytes reset the idle
  timer indefinitely and the test blocks to the Go test timeout.
  *This is the highest-risk requirement in the sprint and it is owned by the milestone that
  introduces the deadline, not a later one.*

- **AC-M2.3 (doc S8 second half, R5 — a configured `0` is REJECTED, not "disabled").** With the flag
  **on** and `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=0`, `Step` returns an error satisfying
  `errors.Is(err, ErrStreamDeadlineInvalid)` **and the fake server's request counter is exactly 0** —
  i.e. the refusal happens at client construction, before any request leaves.
  Repeat for `"-1"`. Then the mirror case: with the flag **off** and the same `"0"`,
  `ollamaV1Timeout()` still returns `0` and the legacy path still works (the documented semantics split).
  **BASE**: new by construction.
  **RED UNDER**: mutation **R5** — the client is built unbounded, the request goes out, the counter
  hits 1, and no typed error is returned.
  *The request-counter assertion is what stops a "returns an error eventually" implementation passing.*

- **AC-M2.4 (REFUTATION #1 gate — the outer 300s deadline must not survive).** With the flag **on**
  and `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` **UNSET**, a trivially-completing fake SSE server, and the
  per-request debug log enabled: the logged **effective** hard deadline (read back from
  `streamCtx.Deadline()`) must be **≥ 3500s**.
  **Why this shape and not a wall-clock test:** the legacy `ollamaCallContext` bound and the new hard
  deadline read the *same* env var (Design Freeze #3 fixes the knob name), so whenever the var is
  **set** both clocks are equal and the bug is unobservable at any timescale. The clocks differ
  **only via their defaults** — 300s vs 3600s — so the only fast, discriminating observation is the
  deadline value itself. A drip-and-wait test **cannot** bite here; do not write one and call it a gate.
  **BASE**: new by construction.
  **RED UNDER**: deriving the streaming branch from the `ollamaCallContext`-bounded `ctx` instead of
  `outerCtx` — the logged effective deadline reads **~300s** and the assertion fails. (This is the
  literal state the doc as written produces.)

- **AC-M2.5 (doc S1 — tool-call-only starvation gate; R3).** A fake SSE server emits **only**
  `tool_calls` fragments — no `content`, no `reasoning` — one fragment per **300ms** for ~2s, with the
  idle window set to **1s**, then `finish_reason:"tool_calls"` and `[DONE]`. The call must **COMPLETE**
  with correctly assembled `ToolCalls` (name + concatenated arguments).
  *Honest framing, per the doc's V-C3 narrowing:* this is a **hypothetical** shape (the probed
  qwen3.6 thinking turn fed `onChunk` 49/51 times), pinning the property that makes read-level
  placement correct **by construction** (V10: no tool-call fire site; V23: `content:""` fires nothing).
  **BASE**: new by construction.
  **RED UNDER**: mutation **R3** — the idle deadline stops resetting on bytes, so a ~2s
  tool-call-only stream trips the 1s idle window and returns `ErrIdleTimeout` instead of completing.
  Equivalently red under moving the reset to the `onChunk` callback: no callback ever fires for
  tool-call deltas, so the timer starves.

- **AC-M2.6 (doc S2 at branch level; R2).** Fake server sends 3 content chunks then goes silent
  forever; idle window 500ms; call returns `errors.Is(err, ErrIdleTimeout)` within 500ms + 1s slack.
  **RED UNDER**: mutation **R2** → blocks to the Go test timeout.

- **AC-M2.7 (doc S3 at branch level; R1).** Fake server sends headers immediately, delays the first
  **body** byte by 900ms (> idle window 500ms, < TTFT window 3s), then streams normally → **COMPLETES**.
  **RED UNDER**: collapsing to a single window (idle armed from t=0) → false-trips with
  `ErrIdleTimeout`. And the TTFT half is red under mutation **R1**.

- **AC-M2.8 (doc S4 — long-but-progressing beats the old client timeout).** With
  `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=4` and the flag on, a fake server drips one content chunk every
  200ms for **2.5s** then finishes → **COMPLETES**, with `Response.Text` equal to the concatenation.
  **RED UNDER**: leaving `Timeout: ollamaV1Timeout()` (= 4s here) *and* shortening the drip config —
  the honest, always-biting mutation is `Timeout: 1 * time.Second` on the streaming branch's
  `http.Client`, which is the V12 trap in its pure form: the client timer interrupts the body read
  mid-stream and the test fails.
  *Note: this AC does **not** cover REFUTATION #1 — AC-M2.4 does. Keep them separate.*

**Milestone boundary**: `go build ./internal/... ./cmd/ailang` · `go test -count=1 ./internal/ai/...`
(both ollama and openai) · `go vet ./internal/ai/...` · `gofmt -l internal/ai` → 0 ·
`make check-file-sizes` (step.go 447 + ~180 ≈ 630, still under 800 — **if it crosses 800, split the
streaming branch into `internal/ai/ollama/streamstep.go` rather than trimming a gate**).
**Flag-off diff must be reviewable as indentation-only plus the two-line `outerCtx` capture.**

---

### M3 — Response parity: `Reasoning` + Hermes tool-call recovery (NEW — REFUTATION #2)

**Est. LOC: 280** (impl ~80, tests ~200). **No GPU.** **Depends on M2.**

**This milestone does not exist in the design doc.** It exists because §0.3 shows the doc's
"everything downstream is unchanged" premise is false, and the thing it silently deletes is the
ollama-qwen3 mitigation for motoko's dominant disengagement mode. See the rule-vii escalation in §7.

**Files**: `internal/ai/openai/step.go` (export one function) ·
`internal/ai/ollama/step.go` (accumulate + recover) · `internal/ai/ollama/step_test.go`

**Build**:
1. Export `extractHermesToolCalls` → `openai.ExtractHermesToolCalls(s string) []ai.ToolCall`
   (rename at `openai/step.go:657`, update the two internal references at `:616` and
   `hermes_test.go:38`). **This touches `internal/ai/openai/step.go`, NOT `streamstep.go`** — the
   doc's non-goal names `streamstep.go` specifically, so this respects its letter; flagged in §7 anyway.
2. In the ollama streaming branch, pass an `onChunk` that accumulates `ai.StreamThinkingDelta.Text`
   into a `strings.Builder`. **`onChunk` DOES fire for reasoning deltas** (`streamstep.go:281,284`) —
   this is the seam that makes parity achievable **without touching `streamstep.go` at all**.
   *This contradicts the doc's "`onChunk: nil`" (High-Impact Decisions row 2 / Solution Design §3).
   `onChunk` is nil-safe (V14), so nil is merely permitted, not required — but the divergence is
   deliberate and is escalated in §7.*
3. After `StreamStep` returns: set `resp.Reasoning` from the accumulator, then
   `if len(resp.ToolCalls) == 0 { if rec := openai.ExtractHermesToolCalls(resp.Text + "\n" + resp.Reasoning); len(rec) > 0 { resp.ToolCalls = rec; resp.FinishReason = "tool_calls" } }`
   — mirroring `openai/step.go:610-616` exactly.

**Acceptance criteria**:

- **AC-M3.1 (Hermes recovery parity — R8).** A fake SSE server streams a turn with **zero** native
  `tool_calls` and a `<tool_call>{"name":"write_file","arguments":{"path":"a.ail"}}</tool_call>`
  block delivered across **`delta.reasoning`** chunks. Flag on. Assert `len(resp.ToolCalls) == 1`,
  `Name == "write_file"`, arguments parse to the expected object, `FinishReason == "tool_calls"`.
  **Anti-vacuity control in the same test**: the identical logical response served through the
  **flag-off buffered** path must also yield exactly 1 tool call — proving the criterion measures a
  *parity gap* and not a property neither path has.
  **BASE**: the streamed half cannot run (no branch); the buffered control is green today.
  **RED UNDER**: mutation **R8** — the streamed half returns 0 tool calls and `FinishReason` stays
  `"stop"`, while the buffered control still returns 1. That divergence is exactly the shipped
  regression this milestone prevents.

- **AC-M3.2 (`Reasoning` parity — R9).** Same content buffered vs streamed: `resp.Reasoning` is
  **non-empty and byte-identical** across both.
  **RED UNDER**: mutation **R9** — streamed `Reasoning` is `""` while buffered is not.

- **AC-M3.3 (doc S6, STRENGTHENED — streamed ≡ buffered).** A **three-case** table, each case served
  once buffered and once as SSE, asserting `Text`, `ToolCalls` (**order, IDs, assembled arguments**),
  `FinishReason` **and `Reasoning`** are identical:
  (a) native single `tool_calls` chunk (the committed-fixture shape);
  (b) **`tool_calls` fragmented across 3 chunks** (mirrors `streamstep_test.go:109`);
  (c) **Hermes block in `reasoning`, zero native tool calls** (case (c) is what the doc's S6 as
  written would have missed — see §0.3).
  **BASE**: new by construction; `=== RUN` ≥ 3 subtests required.
  **RED UNDER**: mutations **R8** (case c), **R9** (`Reasoning` column, all cases), or **R10** on
  `streamstep.go:299` (case b's assembled arguments).

**Milestone boundary**: `go build ./internal/... ./cmd/ailang` ·
`go test -count=1 ./internal/ai/ollama ./internal/ai/openai` (the export rename must leave
`hermes_test.go` green) · `go vet ./internal/ai/...` · `gofmt -l internal/ai` → 0 ·
`make check-boundaries`.

**Hard dependency for the rollout**: **M4's rig opt-in MUST NOT happen before M3 lands.** Between M2
and M3 the flag-on path has a known parity gap. M2 is still independently landable because the flag
is default-off and no default behaviour changes — but turning it on early would ship the regression.

---

### M4 — Fixture replay + rig validation + stopgap removal + docs (**GPU**)

**Est. LOC: 160** (replay test ~90, plists ~10, docs/changelog ~60) + wall-clock rig time.
**Depends on M2 and M3.**

**Files**: `internal/ai/openai/streamstep_test.go` (coverage only, **no source change**) ·
`tools/launchd/dev.ailang.os-rotation-filler.plist` · `tools/launchd/dev.ailang.nightly-eval.plist` ·
`.claude/rules/dev-workflow.md` · `docs/docs/guides/debugging.md` · `changelogs/v0.18-current.md`

**Acceptance criteria**:

- **AC-M4.1 (fixture replay — regression pin on today's ollama wire shape).**
  `go test -count=1 -v ./internal/ai/openai -run TestParseChatStepSSEStream_OllamaV1Fixture` reads
  `../ollama/testdata/ollama_v1_stream_toolcall.sse` (or a copied path — do **not** move the
  committed fixture) through `ParseChatStepSSEStream` and asserts, from the fixture's measured
  contents: exactly **1** tool call, `Name == "get_weather"`, `Arguments == {"city":"Paris"}`
  (JSON-equal), `FinishReason` maps from `"tool_calls"`, and usage `InputTokens 294 /
  OutputTokens 80 / TotalTokens 374`.
  **BASE**: rc=0, `=== RUN` **0**, `[no tests to run]` — VACUOUSLY GREEN. Pass requires rc=0 AND
  `=== RUN` ≥ 1 AND `--- PASS`.
  **RED UNDER**: mutation **R10** at `streamstep.go:299` — arguments collapse to `{}` and the
  assertion fails, proving the test actually exercises the assembly path rather than just parsing.
  *(R10 is a verification-only mutation; revert it.)*

- **AC-M4.2 (doc S7 — FIELD VALIDATION). ⚠️ GPU — THE CONTROLLER MUST TAKE THE RIG LOCK AROUND THIS
  STEP AND ONLY THIS STEP.** With `AILANG_OLLAMA_V1_STREAM=1`, `docx_reimplement` produces an
  end-to-end **X/17 grade** (a number, not a `context deadline exceeded` abort), and the per-request
  debug log records, **for every request**: max inter-chunk gap, TTFT, total duration, and the
  effective hard deadline. Record the observed TTFT and max-gap distributions against the freeze
  defaults (120s / 600s / 3600s) in the PR description — these are the named falsifiers for Design
  Freeze #1/#2/#3.
  **RED UNDER**: any idle/TTFT false trip on a progressing run, or a run lost to
  `context deadline exceeded` at a step boundary. **Note**: if the run dies at ~300s with the env
  unset, that is REFUTATION #1 resurfacing — check AC-M2.4's logged effective deadline first, before
  concluding anything about the model.

- **AC-M4.3 (stopgap removal is mandatory, not tidy-up).**
  `grep -c AILANG_OLLAMA_HTTP_TIMEOUT_SEC` on both plists → **0** (base: **1** each — correctly red
  at base), and `grep -c AILANG_OLLAMA_V1_STREAM` on both → **≥ 1** (base: **0** everywhere).
  Anti-vacuity control in the same call: `grep -c AILANG_OLLAMA` on both plists → **≥ 1**
  (base: 1 each — the control fires, so a zero above is a measurement and not a broken path).
  **Why mandatory**: flag-on, a leftover `1800` pin *becomes the hard deadline* and sits **below** the
  doc's computed worst-case legitimate request (~2941s) — it would re-create the `4m59.97` failure at
  a larger scale.
  **RED UNDER**: leaving either pin in place.

  **FOURTH ARM (added 2026-08-11, after the first three passed while the hazard stayed live).**
  The stopgap has **two** delivery sites and the three arms above measure only one. The launchd
  user-domain global is set by `launchctl setenv`, survives every plist edit, and is invisible to
  any grep over the repo; §2.4 records the commit that says so and the in-flight rig measurement
  (`effective_deadline_sec = 1800`) that caught it. A criterion that greens on the plists alone is
  measuring the *file*, not the process environment the rig job actually inherits.

  > ### ⚠️ THIS ARM IS RIG-STATE ROLLOUT — **NOT** PART OF M4's REPO DELIVERABLE
  >
  > **M4's repo work is complete when the files change.** The plists in `tools/launchd/` are
  > *source*; the rig runs `~/Library/LaunchAgents/`. Measured 2026-08-11: both installed files are
  > **regular files, not symlinks**, and were **byte-identical to repo HEAD** — installation is a
  > manual `cp` + `launchctl load`, documented at `tools/launchd/nightly-eval.sh:19-21`. Editing
  > this repo therefore changes **nothing** on the rig. Steps 1–2 below are a separate,
  > human-sequenced action with its own ordering hazard, and no executor should perform them as
  > "finishing the milestone".

  **The rollout is ORDERED. Flag ON first, clear the global SECOND.**

  1. **Install the flag-on, pin-free plists**, then reload them:
     `cp tools/launchd/dev.ailang.{nightly-eval,os-rotation-filler}.plist ~/Library/LaunchAgents/`
     followed by `launchctl unload && launchctl load` on each. Verify with
     `grep -c AILANG_OLLAMA_V1_STREAM ~/Library/LaunchAgents/dev.ailang.*.plist` → ≥1 each
     (base: **0** each, so the check discriminates).
  2. **Only then** `launchctl unsetenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC`, asserted by
     `launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC` returning **empty**, paired in the same call
     with a **firing control** — a known-set variable returning non-empty — so an empty result is a
     measurement and not a broken call. (At base the target itself reads `1800` while
     `launchctl getenv AILANG_OLLAMA_V1_STREAM` and `launchctl getenv AILANG_NOT_A_REAL_VAR` read
     empty, which is the same discrimination in the other direction.)

  **HAZARD OF THE REVERSE ORDER — do not "tidy up" the global first.** Until step 1 is installed,
  `AILANG_OLLAMA_V1_STREAM` is unset **everywhere installed**, so the streaming branch is OFF and
  the buffered path is live. On the buffered path `ollamaV1Timeout()` (`step.go:28-39`) falls back
  to `defaultOllamaV1TimeoutSec = 300` (`step.go:24`), and the domain global is the **only** thing
  raising it for any invocation not covered by a plist's own `EnvironmentVariables`. Clearing the
  global before the flag is on therefore drops those invocations straight back to 300s and
  **reintroduces the exact defect #618 exists to fix** — the one that cost 895 retries /
  ~74.6 GPU-hours over 43 days, 80 runs lost (`b67d415cd`). **The stopgap is load-bearing until the
  flag is on.** Step 2 before step 1 is not a cosmetic mistake; it is a regression to the original
  bug at full cost.

  **RED UNDER**: editing the plists (or even installing them) without clearing the domain global —
  the three grep arms pass and step 2 fails, which is precisely the state that shipped. Also red,
  in the opposite and more expensive direction, under performing step 2 before step 1.

  **Known limitation of arms 1–3, stated so the next person does not read it as an oversight**:
  the instrument is a raw `grep -c` over the **whole file**, not a plist-key query, so the criterion
  forbids naming `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` anywhere in either file — including in a comment
  explaining why the pin was removed. The M4 executor hit this: an explanatory removal comment held
  the count at 1 and read red. The comments therefore describe it as "the 1800s total-timeout pin"
  without the literal token, and the variable name lives in git history. Do not "fix" this by
  matching a partial token; that games the instrument. If a future revision wants the name back in
  prose, change the arm to a key-level query (e.g. `plutil -extract EnvironmentVariables …`) first.

**Milestone boundary (not acceptance criteria — these gate the repo, not the feature):**
`go build ./internal/... ./cmd/ailang` · `go test -count=1 ./internal/ai/...` ·
`go vet ./internal/ai/...` · `gofmt -l internal/ai` → 0 · `make check-file-sizes` ·
`make check-boundaries` · `make check-changelog`.
**Docs are boundary work, deliberately NOT an acceptance criterion** — per the controller's rule,
"docs updated" passes identically whether or not the feature works. Required at the boundary:
the three env knobs (`AILANG_OLLAMA_V1_STREAM`, `AILANG_OLLAMA_IDLE_TIMEOUT_SEC`,
`AILANG_OLLAMA_TTFT_TIMEOUT_SEC`) plus the changed `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` semantics added
to the debug-flags table in `.claude/rules/dev-workflow.md` and `docs/docs/guides/debugging.md`, and
a `## [Unreleased]` entry in `changelogs/v0.18-current.md` (the section exists at line 5).

---

## 5. Explicitly out of scope — do not touch

- **`internal/ai/handler.go:378`** — the pre-existing unbounded `StreamStep` consumer with
  `context.Background()`. The doc names it deliberately out of scope. The backstop added here lives in
  the **ollama package's transport**, so it does not and must not cover `handler.go`. **Do not
  silently change it**; if the executor wants it bounded, file a follow-up issue.
- **`internal/ai/openai/streamstep.go` source** — no changes. The two-consumer blast radius (V11) is
  the reason. Adding tests to `streamstep_test.go` is coverage-widening and is fine.
  *(M3's export of `ExtractHermesToolCalls` touches `openai/step.go`, a different file — see §7 E-2.)*
- **`ollamaV1Timeout()` / `ollamaCallContext()` semantics** — the native `/api/chat` path and
  `Generate` (`client.go:93`) depend on them, and `TestOllamaV1Timeout` pins `"0"` → disabled. The
  new hard deadline gets its **own** resolver.
- **The native `/api/chat` path** — already streams.
- **Re-capturing the wire fixture** — already committed; the doc's M3 capture task is DONE.
- **Adding `go.uber.org/goleak` to `go.mod`** — see AC-M1.4.

---

## 6. Estimate — this plan REFUTES the design doc's 1–2 days

| Milestone | Impl | Tests | Total |
|---|---|---|---|
| M1 — idle/TTFT reader + watchdog + transport | 170 | 210 | **380** |
| M2 — flag-gated branch + hard deadline (S8, S5) | 180 | 330 | **510** |
| M3 — response parity (**not in the doc**) | 80 | 200 | **280** |
| M4 — fixture replay + rig + plists + docs | 100 | 60 | **160** |
| **Total** | **530** | **800** | **1,330** |

**Velocity basis** (measured at `3e1f63f7a`, last 10 days, non-merge feat/fix/test commits):
`59b74e06d` 1300+39 · `48cf25cff` 733+55 · `632024121` 537+40 · `825c37ee9` 238+23 ·
`7d8db911f` 204+5 · `90b27d3d8` 2893+139 (multi-milestone). Observed single-milestone band
≈ **200–750** lines changed. All four milestones here sit inside that band.

**Why 3 days, not the doc's 1–2:**
- The doc's estimate is downstream of the premise that the swap is semantically free. **M3 (280 LOC)
  is work the doc does not know exists** (§0.3), and it cannot be dropped without shipping a
  disengagement regression on the exact path this sprint is meant to save.
- The doc budgets step.go at "~40–60 LOC". With the mandatory hard deadline + typed config rejection
  + the `outerCtx` restructure + explicit error precedence + four debug-log fields, **180** is honest.
  The doc's own note that "the tests are the bulk" is correct and is why M2 is 510 total.
- M4 is wall-clock-dominated by the GPU run, not LOC-dominated. **Do not compress M4 by dropping
  AC-M4.2** — it is the only field falsifier for all three freeze defaults.

**Day shape**: Day 1 = M1 + start M2 · Day 2 = finish M2 + M3 · Day 3 = M4 (rig-lock window + docs).
The doc's "Day 1 AM: M1, Day 1 PM: M2" is achievable **only** if S8's fake-server harness turns out
cheaper than budgeted; plan for it not to.

---

## 7. Rule-vii escalations for the controller (do not resolve unilaterally)

**E-1 — The doc's `onChunk: nil` decision must be reversed to make parity achievable.**
High-Impact Decisions row 2 and Solution Design §3 both specify `v1.StreamStep(ctx, &r2, nil)`.
M3 requires a **non-nil** `onChunk` to accumulate `StreamThinkingDelta` text, because that is the
**only** way to recover `Response.Reasoning` (and therefore the Hermes block) without modifying
`internal/ai/openai/streamstep.go`, which the doc forbids. The doc's own V14 establishes `onChunk` is
nil-**safe**, i.e. nil is *permitted*, not *required*. **Recommendation: adopt non-nil.** The
alternative — accumulating reasoning inside `ParseChatStepSSEStream` — is arguably the better fix for
both consumers, but it changes shared-parser source with a two-consumer blast radius and the doc
rules it out.

**E-2 — M3 renames an unexported function in `internal/ai/openai/step.go`.**
The doc's non-goal is "*any change to `internal/ai/openai/streamstep.go` source*". Exporting
`extractHermesToolCalls` (`step.go:657`) is a different file and is mechanically safe
(2 call sites, `hermes_test.go` green at the boundary). **Recommendation: allow**, since the only
alternative is duplicating the Hermes regex in the ollama package. Flagged so it is a decision, not a
drive-by.

**E-3 — The Design Freeze #3 knob choice makes REFUTATION #1 untestable by wall clock.**
Because `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` drives both the legacy `ollamaCallContext` bound and the new
hard deadline, the two clocks differ *only* in their defaults (300s vs 3600s). AC-M2.4 works around
this by asserting the **read-back effective deadline** rather than elapsed time. This is sound but it
is a weaker instrument than a behavioural test. **If the controller would accept a second knob**
(e.g. `AILANG_OLLAMA_STREAM_DEADLINE_SEC`, falling back to `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`), the gate
becomes a straightforward wall-clock test. **Recommendation: keep the doc's single knob** (rule vii;
the doc is explicit) and accept AC-M2.4's shape — but the controller should know the trade was made
consciously.

**E-4 — Not a refutation, a warning.** The doc's Rollout says v0.35.0 flips the default on "only if
the M3 rig validation held". With M3 renumbered, that sentence now points at parity work rather than
rig validation. The gate the doc means is **AC-M4.2** in this plan.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| **REFUTATION #1 ships unnoticed** and the feature is inert on the rig at 300s | AC-M2.4 (read-back effective deadline ≥ 3500s) + the explicit check-first note in AC-M4.2 |
| **REFUTATION #2 ships unnoticed** — motoko disengagement regression on the flag-on path | M3 exists; AC-M3.1's buffered-path anti-vacuity control makes the parity gap visible as a divergence, not an absence |
| Per-call `&http.Transport{}` leaks idle connections (`IdleConnTimeout: 0` = no limit) across thousands of rig requests | M1 builds the base transport once via `sync.Once`, cloned from `http.DefaultTransport`; per-request wrapper only |
| `http.DefaultTransport` mutated process-wide by setting `ResponseHeaderTimeout` on it | Explicitly forbidden in M1; clone, never mutate |
| Race between hard-deadline expiry and the idle timer produces the wrong typed error | M2 step 5: explicit precedence + atomic CAS first-writer-wins in the wrapper; AC-M2.2 asserts `!errors.Is(err, ErrIdleTimeout)` |
| A gate is written as bare `rc=0` and passes after the tests are deleted | Every named-test AC carries a `=== RUN` count assertion (§2.2) |
| Executor sandbox denies loopback binds | Report **UNINFORMATIVE UNDER SANDBOX**; controller re-runs outside. Note the *pre-existing* 12 `httptest` uses in `step_test.go` mean the whole package is affected, not just new tests. **Do not rewrite tests to dodge policy.** |
| `step.go` crosses the 800-line gate | Split the streaming branch into `internal/ai/ollama/streamstep.go`; never trim a gate to fit |
| Future ollama version changes the emission shape | AC-M4.1 pins today's shape from real wire bytes; the opt-in flag keeps the old path default for one release |

---

## 9. Commit policy

One bisectable commit per milestone; `go test -count=1 ./internal/ai/...` green at every milestone
boundary. **The controller commits and reviews — this planner created no branches, worktrees or
commits, and did not modify the design doc.**

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_0_0/m-ollama-v1-streaming-idle-timeout-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT.json`

## 10. Controller rulings (iteration 171, 2026-08-10)

All four §7 escalations are resolved here so the executor has no open questions. Every planner
refutation was re-verified first-party at HEAD before ruling — the commands and outputs are below,
because a ruling that inherits its evidence is the laundering this mission's Gate 2 forbids.

### Refutations verified (all three CONFIRMED)

| # | Claim | Controller's own measurement | Verdict |
|---|---|---|---|
| R1 | An outer 300s context deadline already wraps the `/v1` branch, so `Client.Timeout: 0` is necessary but not sufficient | `grep -n ollamaCallContext internal/ai/ollama/step.go` → definition `:63`, **call `:266`**; `sed -n '262,270p'` shows `ctx, cancel := ollamaCallContext(ctx)` with the comment *"Covers the /v1 delegation…"*; the `/v1` branch is at **`:283`**, i.e. **17 lines later**. `ollamaCallContext` is `context.WithTimeout(ctx, ollamaV1Timeout())` — the same 300s default | **CONFIRMED — and decisive.** The design as written ships **inert**: flag on, plist pin removed (which M4 mandates), both clocks fall to defaults, effective bound = min(300s outer, 3600s hard) = **300s**, reproducing the exact `took=4m59.97x` signature the sprint exists to remove |
| R2 | The streamed path is not response-equivalent: no `Reasoning`, and no Hermes tool-call recovery | `grep -c "out.Reasoning" internal/ai/openai/streamstep.go` → **0** (control: `grep -c "\.Reasoning" step.go` → **6**). `grep -c ermes streamstep.go` → **0** (control: `step.go` → **8**). The recovery at `step.go:610-616` carries its own comment: *"Ollama's /v1 sometimes fails to lift [a Hermes `<tool_call>` block] into the tool_calls field for Qwen3 thinking models … the '0 tool calls / disengagement' failure mode"* | **CONFIRMED, and it is a would-have-shipped regression.** Switching motoko's path to streaming without M3 trades a timeout bug for a **disengagement** bug — this mission's single most-studied failure mode. M3 is required, not optional |
| R3 | goleak is in `go.sum` but not `go.mod` | `grep -c goleak go.mod` → **0** (control: `go.sum` → **2**) | **CONFIRMED.** Use the done-channel probe; adding goleak is a dependency change, out of scope |

**A note the controller owes the record:** R2 was findable from this iteration's own rig probe and
was missed. The probe measured **48 `reasoning` deltas** before the tool call — precisely the text
where qwen3 puts the Hermes block — and the controller read that column as *"the callback gets fed,
so the frequency argument is weaker"* rather than as *"the reasoning text is load-bearing content
the streamed path discards."* Same bytes, two readings, and only one of them was looked for. The
planner found it by asking a question the controller never asked: not *"does streaming work?"* but
*"is streaming EQUIVALENT?"*

### Rulings

- **E-1 — ADOPT non-nil `onChunk`.** The doc's `v1.StreamStep(ctx, &r2, nil)` is reversed. It is the
  only seam that recovers `Response.Reasoning` without editing `streamstep.go`, which the doc's
  non-goals forbid. This does **not** disturb the quorum-approved direction: the idle deadline stays
  at the **read** level, which is exactly why read-level placement was the right call — it does not
  depend on the callback for anything.
- **E-2 — ALLOW exporting `extractHermesToolCalls`** from `internal/ai/openai/step.go`. The doc's
  non-goal names `streamstep.go` specifically; this is a different file, 2 call sites, and the only
  alternative is duplicating the Hermes regex in the ollama package.
- **E-3 — KEEP the doc's single knob, and STRENGTHEN AC-M2.4 rather than accept the weaker
  instrument.** The planner is right that the two clocks differ only in their *defaults*, so a
  wall-clock test on defaults cannot discriminate. But the test controls the environment: with
  `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` set small (~2s) and the flag ON, the **error type** discriminates
  — if M2 correctly captures the pre-`:266` context, the typed hard-deadline error fires; if the
  outer `ollamaCallContext` wrap leaks through, the legacy untyped context error fires instead. So
  AC-M2.4 keeps its `streamCtx.Deadline()` read-back **and** gains this behavioural arm, whose
  named mutation is "do not capture `outerCtx` before `:266`" — the exact defect R1 describes.
  A second env var is declined: this path already carries three, and the reviewer's verbatim
  Freeze-#3 requirement is knob-agnostic.
- **E-4 — the v0.35.0 default flip requires BOTH M3 parity AND M4 rig validation to have held.**
  The doc's rollout sentence predates the renumbering; parity alone is not sufficient evidence to
  flip a default on the rig.

### Standing instructions for the executor

- `go build ./...` is **rc=1 at base** (`cmd/wasm`, `gen/main` have no native `main`). It is not an
  acceptance gate. Use `go build ./internal/... ./cmd/ailang`.
- Every refusal branch gets **one** neutering mutation of the form `if false && <cond>` — never a
  deleted block, because a mutant that fails to compile reds for the wrong reason.
- Only **M4 AC-M4.2** touches the GPU; the controller takes the rig lock around that step alone.
