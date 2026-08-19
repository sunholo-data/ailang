# Sprint plan — M-MOTOKO-FMT-REMEASUREMENT-INSTRUMENT (D1 + D1b + D2 + smoke-bank)

**Design doc (authority):** `design_docs/planned/m-motoko-fmt-remeasurement-instrument.md`
The design doc WINS over this plan wherever they disagree. This plan adds only *how* and *in what
order*, plus the measurements that were run to choose among the options the doc left open.

**Created:** 2026-08-20 (motoko mission iteration 14, Gate 3)
**Branch / worktree:** `sprint/motoko-iter14-fmt-d1` @ `/Users/voightkampff/.ailang-driver-pin/.wt-motoko-iter14-fmt-d1`, branched from `origin/dev` at `44aa3cab4`
**Scope, from the doc's §6 "Sequencing":** D1 + D1b + D2 + smoke-bank wiring
**Estimated:** 5 milestones, ~1,515 LOC (≈475 impl + ≈1,040 test/harness), **≈3.0 days** — fits the doc's ≤3–4 day discipline. **Nothing is deferred to a follow-up sprint.**
**Platform of every measurement in this plan:** **darwin/arm64** (`uname -sm` → `Darwin arm64`, macOS 26.5.2). The windows and ubuntu CI legs are **unrun locally**; every "BASE" result below is a darwin/arm64 result only. The one place this is load-bearing is `runHealthCheck`'s exec-bit branch, which is guarded `runtime.GOOS != "windows"` — its mutation row (T-B3) is therefore darwin-only and its windows behaviour is asserted by construction, not by execution.

---

## 0. The finding that re-shapes D1 — the doc's preferred option is UNSOUND

The design doc's §12.2 lists three options and leans to **option (1)** ("set `cfg.MotokoModel` from
`models.yml` at construction — smallest true fix"). **Option (1) cannot work.** Two independent
measurements, either one fatal:

### 0.1 There is no per-model construction, and no singular model to set

`executor.GlobalFactory()` builds **one** factory, **once**, from `DefaultConfig()`:

```
internal/executor/factory.go:178-182   GlobalFactory() → globalFactoryOnce.Do(func(){ globalFactory = NewFactory(nil) })
internal/executor/factory.go:79-87     NewFactory(nil) → cfg = DefaultConfig()
internal/executor/factory.go:96-122    GetExecutor(name) caches by EXECUTOR NAME: f.executors[name]
```

`SetGlobalFactory` has **zero non-test callers** (measured: `grep -rn 'SetGlobalFactory\|NewFactory(' --include='*.go' . | grep -v _test.go` → 4 hits, all inside `factory.go` itself: the declaration, the `GlobalFactory` body, and the doc comment). So `cfg` is a process-global singleton and `cfg.MotokoModel` is a **single string**.

But `agent_cli: "motoko"` names **17 distinct lanes** in `internal/eval_harness/models.yml`, split across two providers:

| provider | lanes | example `agent_model_name` |
|---|---|---|
| OpenRouter-routed | **10** | `openrouter/anthropic/claude-haiku-4-5`, `openrouter/z-ai/glm-5.1`, … |
| ollama (local) | **7** | `ollama/qwen3.6:35b-a3b-mxfp8` (`motoko-local-qwen3-6-fmt`, profile `ollama_fmt` — the treatment arm) |

`GetExecutor("motoko")` hands **the same cached `*MotokoExecutor`** to all 17. There is no
construction event per model, so "set `cfg.MotokoModel` from models.yml" has no well-defined
value: 10 lanes need `OPENROUTER_API_KEY` and 7 need nothing, and the field can hold one string.

### 0.2 Even granted a value, `HealthCheck` is `sync.Once`-cached — a per-model verdict frozen at the first model

```
internal/executor/motoko/motoko.go:131-133   healthCheckOnce sync.Once; healthCheckErr error
internal/executor/motoko/healthcheck.go:31-43 HealthCheck → e.healthCheckOnce.Do(...); return e.healthCheckErr
```

The canary loop calls `GetExecutor` **per model** (`internal/eval_harness/canary_filter.go:62`) and
gets the same instance, so the first model canaried decides for every later one. A model-dependent
condition evaluated inside a once-cached method is a category error: on a night that canaries both
`motoko-local-qwen3-6-fmt` (ollama) and any of the 10 OpenRouter motoko lanes, one lane's provider
answer would be applied to the other. Option (1) would convert today's *loud, uniform* refusal into
a *silent, order-dependent* one — strictly worse under CLAUDE.md rule 2.

### 0.3 Chosen: **option (2), refined** — the check moves to the per-task choke point

> **D1 = remove the unconditional key refusal from `runHealthCheck`; add a resolved-provider
> credential check at the top of `MotokoExecutor.ExecuteStreaming`, keyed on `e.getModel(task)`.**

Option (3) (widen the `HealthCheck` signature) is rejected without further measurement: it is a
change to `executor.Executor` (`internal/executor/executor.go:31`) with 6 implementors, and the
trace gives no reason to prefer it.

**Why `ExecuteStreaming` and not `Execute` or `CanaryCheck`:** `Execute` delegates
(`motoko.go:166-168 → ExecuteStreaming`), so `ExecuteStreaming` is the single choke point through
which *all* motoko work passes. Placing it in `CanaryCheck` would miss
`agent_runner_multi.go`'s path; placing it in `Execute` would miss direct `ExecuteStreaming`
callers.

**Why this satisfies §12.3's ordering requirement for free.** The doc requires that the
resolved-provider check must NOT run ahead of `MOTOKO_REPO` / repo discovery, or the profile
degrades to `extensions.order=[]` and drops the `fmt` extension that IS the treatment. Measured:
`e.motokoRepo` is populated by `HealthCheck`'s `motoko --version` query
(`healthcheck.go:70-77`), and the code comment at `motoko.go:364-365` states it directly — *"e.motokoRepo
is discovered by HealthCheck from `motoko --version`, which the eval path and the canary both run
before Execute"* — confirmed at both call sites (`internal/eval_harness/agent_runner_multi.go:107`,
`internal/executor/motoko/canary.go:67`). A check inside `ExecuteStreaming` is therefore **strictly
downstream** of repo discovery in both production paths. This is an ordering property to be
*asserted*, not assumed — see test row **T-ORDER**.

**Cost of the doc's stated downside, measured rather than accepted.** §12.2 says option (2) "loses
fail-fast-before-workspace-setup". What is actually lost:
- canary path (`canary.go:95-103`): one `os.MkdirTemp` + one 3-byte `os.WriteFile` before `Execute`;
- eval path (`agent_runner_multi.go:112-122`): one `os.MkdirAll` + a stub `.git` dir.

Both are local filesystem operations of microsecond order. Everything expensive (bun startup, the
LLM call) happens *inside* `ExecuteStreaming`, after the check. Attribution is preserved end to end:
an `Execute` error propagates `canary.go:118 evaluateCanaryResult → "canary task failed: %w" →
executor.CanaryError → eval_harness.CanarySkip`, i.e. the model is still dropped from the run matrix
with a named reason, exactly as today.

### 0.4 The condition is expressed on the RESOLVED PROVIDER, never on a literal env-var name

```go
// internal/executor/motoko/provider_preflight.go (new)
provider := ai.GuessProvider(model)          // internal/ai/config.go:45-101
envVar  := ai.EnvVarForProvider(provider)    // internal/ai/config.go:104-119
```

One check covers ollama / OpenAI / Anthropic / Google / OpenRouter. The literal string
`OPENROUTER_API_KEY` must not appear in the new predicate (it may appear only in a test's
*expectation table*, which is the mechanism's own value set — see §3).

**Import-boundary check (controller fact 4, re-verified this session, plus the missing half).**
`internal/executor/motoko` imports exactly two in-module packages today:

```
$ grep -rho '"github.com/sunholo-data/ailang/internal/[a-z_/]*"' --include='*.go' internal/executor/motoko/ | sort -u
"github.com/sunholo-data/ailang/internal/executor"
"github.com/sunholo-data/ailang/internal/telemetry"
```

Non-empty, so the instrument fires. `internal/ai` does **not** import `internal/executor` (same
anchored grep over `internal/ai/`), so no cycle. `scripts/check_boundaries.sh` polices only
`CORE_PKGS=(parser types eval core elaborate effects builtins lexer ast pipeline runtime link iface)`
and `DASHBOARD_PKGS=(server coordinator observatory messaging)` — neither `executor` nor `ai` is in
either set, and five files already import `internal/ai` (`internal/feedbackgate/classifier.go`,
`internal/coordinator/provider_gemini.go`, `internal/eval_harness/ai_provider.go`, + tests), so the
grep sees this class of hit. §12.2's caveat *"confirm the boundary allows it"* is **ANSWERED: it
does.** Gate command and BASE result in §4.

> ⚠ **Instrument warning inherited from the controller and re-confirmed.** The module path is
> `github.com/sunholo-data/ailang`, **not** `github.com/sunholo/ailang`. An import grep anchored on
> the wrong path returns a confident, silent **0**. Every import grep in this sprint must be
> anchored on the `go.mod` path and paired with a known-positive control.

---

## 1. Milestones

Each milestone is independently committable, independently bisectable, and leaves the tree green on
the §4 gates. Order is the doc's §6 order: **D1 first** — it is the only item that unblocks anything
else, and it un-breaks the current tree's Wednesday A/B lane regardless of the rest.

| # | milestone | impl LOC | test LOC | days | depends on |
|---|---|---|---|---|---|
| M1 | **D1** — provider-conditional preflight | 70 | 220 | 0.75 | — |
| M2 | **AC-D1-live** — live provider-connection proof | 130 | 30 | 0.25 (+~15 min rig) | M1 |
| M3 | **D1b** — counterbalanced Wednesday block | 55 | 120 | 0.25 | M1 |
| M4 | **D2** — censored-pair analyzer | 380 | 330 | 1.25 | — (parallel-safe with M1–M3) |
| M5 | **smoke-bank wiring** (§5.2) | 110 | 80 | 0.5 | M3 |
| | **total** | **745** | **780** | **3.0** | |

---

### M1 — D1: provider-conditional preflight

**Files**
- `internal/executor/motoko/healthcheck.go` — DELETE the unconditional refusal at :64-66. Keep the binary/dir/exec-bit checks and, crucially, keep the `motoko --version` query that populates `e.motokoRepo`.
- `internal/executor/motoko/provider_preflight.go` — NEW. `func requireProviderCredential(model string) error`.
- `internal/executor/motoko/motoko.go` — call it at the top of `ExecuteStreaming`, before the span/workspace/subprocess work, with `e.getModel(task)`.
- `internal/executor/motoko/motoko_test.go` — **DELETE** `TestHealthCheck_MissingAPIKey` (:141-154). Per `.claude/rules/coding-standards.md` ("ALWAYS remove out-of-date tests. No backward compatibility."), it asserts the exact behaviour this milestone removes. Its replacement is T-B5 below. Also drop the now-false `t.Setenv("OPENROUTER_API_KEY", …)` scaffolding and stale comment at :130-132.
- `internal/executor/motoko/execute_test.go` — see the migration hazard below.

**Measured migration hazard (do not discover this at CI time).** Eight of the nine `Execute` tests
build the executor as `New(&executor.Config{MotokoPath: mockMotoko})`
(`execute_test.go:131,204,285,360,402,445,497,547`), leaving `MotokoModel` empty, which `New`
defaults to `"openrouter/anthropic/claude-haiku-4-5"` (`motoko.go:145-148`). Under M1 those tasks
resolve to provider `openrouter` and will be **refused** wherever `OPENROUTER_API_KEY` is unset —
i.e. CI. They pass on this machine today only because the ambient env has the key set
(`${#OPENROUTER_API_KEY}` = 73). The fix is per-test and deliberate: give each mock task an
`ollama/…` model (matching what the mock actually simulates), or `t.Setenv` the key where the test
is genuinely about an OpenRouter lane. The regression gate for exactly this is **AC-M1-3**.

**Acceptance criteria**
- **AC-M1-1** `go build ./internal/... ./cmd/ailang/...` exits 0. *(BASE: PASS — see §4 for why `go build ./...` is NOT the gate.)*
- **AC-M1-2** `go test ./internal/executor/... ./internal/ai/...` exits 0. *(BASE: PASS, 14/14 packages ok.)*
- **AC-M1-3** `env -u OPENROUTER_API_KEY go test ./internal/executor/motoko/` exits 0. *(BASE: PASS, `ok … 20.422s`.)* This is the gate that catches the 8-test hazard above; it must stay green.
- **AC-M1-4** `bash scripts/check_boundaries.sh` exits 0 after the new `internal/ai` import. *(BASE: PASS, "OK: no architecture boundary violations".)*
- **AC-M1-5** `grep -n 'OPENROUTER_API_KEY' internal/executor/motoko/*.go | grep -v _test.go` returns **zero** lines. Known-positive control in the same sweep: the same grep over `internal/ai/config.go` returns lines 115 and 143, so the grep sees this class of hit. *(BASE: 1 hit at `healthcheck.go:64`; control: 2 hits — so the instrument discriminates.)*
- **AC-M1-6** The full refusal-branch mutation matrix (§3, rows T-B1…T-B6 + T-ORDER) passes, and each listed neutering mutation makes exactly its own row fail.

---

### M2 — AC-D1-live: the live provider-connection proof

**The design doc's criterion, carried VERBATIM (§12.4):**

> **AC-D1-live.** With the fix in place, one fmt-lane run reaches `localhost:11434` and makes **zero**
> connections to `openrouter.ai`. Assert on the connection, not on the absence of an error — an
> absence is satisfied equally by "no OpenRouter call" and "the run never started" (the observable
> must be unique to the mechanism). Pair it with a known-positive control: an OpenRouter-lane run in
> the same sweep must show the connection, or the instrument proves nothing.

**This criterion may NOT be weakened to "no error occurred".** The doc forbids it explicitly, for
the stated reason. It is satisfied only by an *observed connection*, in both directions.

**Files**
- `tools/eval/motoko_connection_probe.sh` — NEW. Runs one motoko canary for a named models.yml lane, samples the connection table of the motoko process tree once per second for the run's duration, and writes the union of ESTABLISHED peers to a JSON artifact.

**Instrument (darwin/arm64 only; no root required).**
1. Resolve the target set once, up front: `dig +short openrouter.ai` → `OR_IPS`. Record it in the artifact — a criterion computed from an unrecorded resolution is unauditable.
2. Launch `ailang eval-suite --agent --models <lane> --benchmarks <one> --trials 1 --dry-run=false` (or the canary path directly), capture the driver PID.
3. Poll `lsof -nP -iTCP -sTCP:ESTABLISHED -a -p "$(pgrep -d, -P <pid> ; echo <pid>)"` each second; union all `NAME` peer endpoints.
4. Classify each peer as loopback / `OR_IPS` member / other. **Emit the full peer set**, always — the artifact is the evidence, the verdict is a function of it.

**Acceptance criteria (both must hold, in the same sweep, same day)**
- **AC-M2-treatment** For lane `motoko-local-qwen3-6-fmt`: the peer set **contains** `127.0.0.1:11434`, and **contains no member of `OR_IPS`**. (The probe additionally reports any non-loopback peer for human review; a non-loopback peer that is not in `OR_IPS` does not fail the criterion but must be recorded.)
- **AC-M2-control** For an OpenRouter lane (`motoko-claude-haiku-4-5`, cheapest of the 10) with `OPENROUTER_API_KEY` set: the peer set **contains at least one member of `OR_IPS`**. If this control does not fire, **AC-M2-treatment is void** — the probe proved nothing.
- **AC-M2-3** The artifact records `OR_IPS`, both peer sets, both lane names, the probe start/end timestamps, and `uname -sm`.

**Why this is not a weakened absence.** The treatment leg's verdict rests on a *positive*
observation (`127.0.0.1:11434` present) conjoined with a negative one, and the negative one is only
admissible because the control leg demonstrates, on the same instrument and the same day, that the
probe can see a remote connection when there is one. "The run never started" fails
AC-M2-treatment's positive half.

**Dependencies / cost.** Needs the rig (or any host with ollama serving qwen3.6 and the motoko
binary present) and, for the control leg only, `OPENROUTER_API_KEY` and a few cents of metered
spend on one trivial canary task. ~15 min wall-clock. This is the one milestone whose completion is
not purely local.

---

### M3 — D1b: counterbalanced Wednesday block (doc §5.3)

**File:** `tools/launchd/nightly-eval.sh`, the fmt block. V22 cites lines **492-507**; re-verified
this session at the same lines — the whole-arm `for arm in on off; do … "$BIN" eval-suite --agent
--models "$m" --benchmarks "$FMT_BENCH_LIST" … --trials "$FMT_TRIALS" … done` is one whole-suite
call per arm.

**Replacement, per §5.3 exactly:** iterate the ELO-selected benchmarks *in selection order*; for
benchmark index `i` (0-based) run both arms back-to-back — **ON then OFF when `i` is even, OFF then
ON when `i` is odd** — each arm as one `eval-suite --benchmarks <b> --trials "$FMT_TRIALS"`
invocation into the same per-arm `--output` dir.

**Two mechanics confirmed by measurement, so this is expressible:**
- `select_ab_benchmarks` (`nightly-eval.sh:225-232`) emits a **comma-separated, space-stripped** list (`… | grep '^Benchmarks:' | sed 's/^Benchmarks:[[:space:]]*//' | tr -d ' '`), so `IFS=',' read -ra FMT_BENCHES <<< "$FMT_BENCH_LIST"` splits it safely.
- Repeated invocations into one output dir do not collide: bank filenames embed the bank time (`<id>[_trialN]_<lang>_<model>_<timestamp>.json`, `internal/eval_harness/metrics.go`, V23).

Everything downstream of the loop (the `eval-paired` call, `count_passes`, `FMT_VALID`, the
`fmt_ab.jsonl` bank) reads the two output dirs and is **untouched**.

**Acceptance criteria**
- **AC-M3-1** `bash -n tools/launchd/nightly-eval.sh` exits 0. *(BASE: PASS.)*
- **AC-M3-2** `shellcheck -S warning tools/launchd/nightly-eval.sh` exits 0 with 0 findings. *(BASE: PASS, exit 0, 0 findings.)*
- **AC-M3-3** Order test T-D1B (§3) passes: with a stub `$BIN` on `PATH` that appends `"$*"` to a log and exits 0, sourcing the fmt block over a 6-benchmark list produces **12** `eval-suite` invocations whose (benchmark, arm) sequence is `b0:on,b0:off, b1:off,b1:on, b2:on,b2:off, b3:off,b3:on, b4:on,b4:off, b5:off,b5:on`, and each invocation names exactly one benchmark.
- **AC-M3-4** The same log satisfies the §5.3 gate's own predicate: contiguous same-(benchmark,arm) blocks, each benchmark's two blocks adjacent, `|#ON-lead − #OFF-lead| ≤ 1`. Asserted by calling **M4's** order-integrity checker on synthesised rows, so the driver and the analyzer are proven to agree rather than each being right about a different schedule.

---

### M4 — D2: the censored-pair analyzer

**Decision: a SIBLING command, not an extension of `eval-paired`.** Justified by measurement, not
taste: `eval-paired`'s stdout JSON is a live contract — `nightly-eval.sh:512` pipes it through
`paired_summary` and it is banked, and the **Monday microRAG** block calls the same command. Changing
its shape would silently re-point an unrelated experiment's bank. `cmd/ailang/eval_paired.go` is 73
lines and stays byte-identical.

**Files**
- `internal/eval_analysis/types.go` — ADD two read-side fields to `BenchmarkResult` (additive, `omitempty`, JSON tags matching the already-banked keys at `internal/eval_harness/metrics.go:138,143`): `FmtHookState string \`json:"fmt_hook_state,omitempty"\`` and `FmtHookEvents []eval_harness.FmtHookEvent \`json:"fmt_hook_events,omitempty"\``. Measured gap: `BenchmarkResult` today carries `MicroRAGState` but **no** fmt field, while `Validity` and `Timestamp` and `Trial` and the token fields are already present.
- `internal/eval_analysis/censored.go` — NEW. The §2 verdict, the §5 void rules, the §7 decision rule.
- `cmd/ailang/eval_censored.go` — NEW. `ailang eval-censored-pairs [flags] <on-dir> <off-dir>`.
- `cmd/ailang/main.go` (+1 case), `cmd/ailang/help.go` (+1 line), `cmd/ailang/eval_remote_read.go` (+1 map entry, matching how `eval-paired` is registered).

**What it implements — the doc's own words, not a paraphrase**
- §2 verdict per (benchmark, trial) pair: one-passes → passing arm **wins**; both pass → fewer total tokens (input cache-inclusive + output, as banked) wins with `|log ratio| ≤ 0.10` a **tie**; both fail → **tie**.
- Primary: one-sided exact sign test on non-tied pairs, α = 0.05, H₁ = ON wins more often. (`internal/eval_analysis/paired.go:205-231` already has `exactBinomialTwoSided` + `binomialPMF` to build on.)
- Secondaries, reported never deciding: mean paired log token ratio on both-pass pairs with 95% CI; and the existing `PairArms` McNemar + headroom as a pass-rate guardrail.
- §5 **treatment integrity**: refuse quarantined input. A pair whose ON row is quarantined is dropped **and counted**. `> 20%` of ON rows quarantined, or **any** OFF-row contamination → **VOID, no numbers reported at all**, void reason is the output.
- §5.3 **order integrity**: sort a slot's rows by `timestamp`; require (a) contiguous same-(benchmark, arm) blocks, (b) each benchmark's two blocks adjacent, (c) lead arm alternates (`|#ON-lead − #OFF-lead| ≤ 1`). Any failure → **VOID, no numbers reported**.
- §7 decision rule verdicts: `VOID` / `KEEP` / `RETIRE` / `INCONCLUSIVE`, with the exact thresholds (`n_eff < 24`; KEEP iff sign test rejects at α=0.05 **and** both-pass median token ratio ≤ 0.90 **and** McNemar shows no significant ON pass-rate loss; RETIRE iff opposite-direction rejection **or** KEEP fails with `n_eff ≥ 40`).

**Explicitly out of scope for this sprint:** the "void run pages the mission via `ailang messages
send`" wiring from §5.1. The analyzer must *emit* the void verdict and reason on stdout and exit
non-zero; who pages is driver policy and belongs in the nightly block, where M5 already adds a
failure path.

**Acceptance criteria**
- **AC-M4-1** `go test ./internal/eval_analysis/... ./cmd/ailang/...` exits 0. *(BASE: to be recorded by the executor as its first action — this plan baselined `./internal/executor/... ./internal/ai/...` and the scoped lint, not this pair. Treat a red base here as a repo finding, not a sprint failure.)*
- **AC-M4-2** `golangci-lint run ./internal/eval_analysis/... ./cmd/ailang/... ./internal/executor/motoko/...` exits 0 with 0 issues. *(BASE: PASS, "0 issues".)*
- **AC-M4-3** `ailang eval-paired --with-pairs=true <on> <off>` produces **byte-identical** stdout before and after M4, on the banked AC5 arms `eval_results/ab2_fmt_on` / `eval_results/ab2_fmt_off`. Proves the sibling command did not disturb the live contract.
- **AC-M4-4** Fixture-driven verdict matrix (§3, rows T-D2-*) passes, including the two void rules and each of the four §7 verdicts.
- **AC-M4-5** Run against the real banked AC5 arms, the analyzer reports **VOID with an order-integrity reason** (V19 established those 12 rows are a perfect arm block). This is the strongest available end-to-end check: a real, independently-measured dataset whose correct verdict is known in advance and is *not* the happy path.

---

### M5 — smoke-bank wiring (doc §5.2)

**File:** `tools/launchd/nightly-eval.sh` (fmt block, immediately before M3's interleave loop) plus a
small checker.

**What §5.2 asks for:** before the first measured slot, one smoke bank of **1 benchmark × 1 trial,
ON arm** verifying (a) the ported extension still writes `<workspace>/.claude/fmt_hook_events.jsonl`
with the `{status,file}` schema, and (b) the new-tree step-0 broadcast still names the extension
`fmt` in `resolved_extensions`. Cost ~5 min rig.

**Design.** Reuse the machinery that already exists rather than re-deriving it: check (b) is exactly
`eval_harness.ResolveFmtArm(flagMode, resolvedExtensions) == "on"` (`internal/eval_harness/fmt_treatment.go:28-47`, exact `fmt#N` name match), and check (a) is exactly
`AssertFmtTreatmentIntegrity("on", events) == nil` (`:60-92`, which already demands ≥1 event **and**
≥1 with `status=formatted` — "firing is not delivering"). So the smoke check is: bank one ON row,
then assert the banked row's `validity` is absent/valid and its `fmt_hook_state` is `on`. **No new
assertion logic is written**; the smoke bank is a *driver* change that runs the existing gate early
and cheaply.

**Failure behaviour, per §5.2's "failure direction is loud either way":** the fmt block logs the
specific failing contract, sends `ailang messages send controlplane`, sets `RUN_AB_FMT=0`, and does
**not** enter the measurement loop. Modelled on the existing `select_ab_benchmarks`-returned-nothing
path (`nightly-eval.sh:480-487`), which is the repo's own precedent for "no silent fallback: skip
loudly rather than measure something uninterpretable".

**Acceptance criteria**
- **AC-M5-1** `bash -n` + `shellcheck -S warning` on `nightly-eval.sh` both exit 0 with 0 findings. *(BASE: PASS/PASS.)*
- **AC-M5-2** Stub-`$BIN` test T-D2-SMOKE: when the smoke bank produces a row with `fmt_hook_state=off`, the driver logs the contract failure, sets `RUN_AB_FMT=0`, and emits **zero** `eval-suite --trials 5` invocations. Killed mutation: neutering the smoke gate (`if false && <smoke_failed>`) yields 12 measurement invocations instead of 0. Observable: the stub's invocation log (value set = the set of invocations, not a boolean).
- **AC-M5-3** The happy path — a stub smoke row with `fmt_hook_state=on` and a `status=formatted` event — proceeds into M3's interleave and emits the full 12 invocations. Without this row, AC-M5-2 is satisfiable by a driver that always skips.

---

## 2. Deployment precondition (doc §6) — a checkable step, not a note

Merging D1 + D1b to `origin/dev` does **not** put them on the rig. The installed plist executes
`nightly-eval.sh` **in place from V1's checkout**, and that checkout is not this mission's
(open issue **#558**, the stale-checkout defect).

**Re-verified this session, with a known-negative control:**

```
$ /usr/libexec/PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist
Array {
    /bin/bash
    /Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/nightly-eval.sh
}                                                                          # exit 0
$ /usr/libexec/PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.NOT-REAL.plist
Print: Entry, ":ProgramArguments", Does Not Exist                          # exit 1  ← control: absence is a measurement
```

**AC-DEPLOY (blocks the first measured slot, not the sprint's merge):**
1. `SCRIPT=$(/usr/libexec/PlistBuddy -c "Print :ProgramArguments" ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist | sed -n '3p' | tr -d ' ')` — read the path **from the installed plist**, never from a working-tree path.
2. `grep -c 'requireProviderCredential\|fmt smoke' "$SCRIPT"` is not the check (the Go fix isn't in the script). The two reads are:
   - the script at `$SCRIPT` contains the D1b per-benchmark interleave (no `for arm in on off` wrapping a whole-suite call);
   - the `ailang` binary that `$SCRIPT` invokes as `$BIN` reports a build containing D1 — assert by running `"$BIN" --version` and comparing its git rev to a rev that contains M1's commit.
3. Until **both** reads pass, **no Wednesday slot counts as a measurement attempt** (doc §6, verbatim intent).

This step is deliberately *not* an acceptance criterion of M1 or M3: the sprint cannot make V1's
clone pull. It is a gate on the measurement, owned by whoever fires the first slot, and it is
recorded here so it cannot be forgotten.

---

## 3. Test plan — one mutation per refusal branch, with its observable

**Rules applied to every row below.** (i) The row names the **mutation** it kills. (ii) Mutations are
**neutering, never deleting**: `if false && <cond>` keeps the mutant compiling, so a "test passed"
result cannot be an artifact of a broken build. (iii) The **observable** is named, and is checked to
be *downstream* of the mechanism (not adjacent to it) with a value set in **bijection with the
mechanism's branch set** — a row asserting only "an error was returned" / "no error was returned"
has a 2-valued observable against a 6-valued mechanism and is decorative; such rows are excluded.

The preflight is a **REFUSAL**, so its branches are enumerated exhaustively.

### 3.1 Refusal branches of `runHealthCheck` after M1

| id | branch | mutation | observable | value set |
|---|---|---|---|---|
| **T-B1** | `os.Stat` error → binary missing | `if false && err != nil` | error string contains `motoko CLI not found at %q` **and** the quoted path equals the configured `MotokoPath` | the path — arbitrary strings, ⊇ the branch |
| **T-B2** | `info.IsDir()` | `if false && info.IsDir()` | error contains `is a directory, expected an executable` and names the same path | as above |
| **T-B3** | non-windows, `perm&0111 == 0` | `if false && (runtime.GOOS != "windows" && …)` | error contains `is not executable (chmod +x)` and names the same path. **darwin/arm64 only**; the windows leg is unrun locally | as above |
| **T-B4** | *(the deleted `OPENROUTER_API_KEY` branch)* | n/a — asserted by **absence**, which is only admissible because AC-M1-5 pairs it with a known-positive control grep | `HealthCheck` on a mock binary with `OPENROUTER_API_KEY` unset returns **nil** (this is the behaviour change), and `e.motokoRepo`/version parsing still runs | nil vs error — admissible here only because the *positive* half (T-ORDER) proves the version query fired |

### 3.2 Refusal branches of `requireProviderCredential`

The mechanism's value set is `ai.EnvVarForProvider`'s 5-way switch (`internal/ai/config.go:104-119`).
The test is therefore a **table over models → expected env-var name**, which is exactly that value
set — not a boolean.

| id | branch | mutation | observable | why the observable is downstream, not adjacent |
|---|---|---|---|---|
| **T-B5** | provider needs a key and it is unset → refuse | `if false && os.Getenv(envVar) == ""` | the returned error names **both** the resolved env var **and** the model string. Table (each row with the key `t.Setenv("", …)`-cleared): `ollama/qwen3.6:35b-a3b-mxfp8` → **no error**; `ollama:qwen3.5` → **no error**; `openrouter/anthropic/claude-haiku-4-5` → error naming `OPENROUTER_API_KEY`; `anthropic/claude-sonnet-4-6` → error naming `OPENROUTER_API_KEY` (vendor/model routes to OpenRouter, V26); `claude-3-5-sonnet` → error naming `ANTHROPIC_API_KEY`; `gpt-5-codex` → error naming `OPENAI_API_KEY`; `gemini-3-1-pro` → error naming `GOOGLE_API_KEY` | the env-var **name** is produced *by* the resolution chain `GuessProvider → EnvVarForProvider`; nothing else in the function can produce it. A hardcoded `OPENROUTER_API_KEY` string would fail the `claude-3-5-sonnet` / `gpt-5-codex` / `gemini-3-1-pro` rows |
| **T-B5b** | key present → proceed | `if false && …` (same site, inverted arm) | for each keyed model above, with the correct env var **set**, `requireProviderCredential` returns nil; with a *different* provider's var set instead, it still refuses. Kills a mutant that checks "any API key is set" | the pairing (model, which var was set) is 2-dimensional; a single-var check collapses it |
| **T-B6** | `GuessProvider` returns `""` (unrecognised model string) | `if false && provider == ""` | **DECISION, flagged for the executor:** `EnvVarForProvider("")` returns `""` (`config.go:117 default:`), so the naive implementation would **pass** an unrecognised model silently. Per CLAUDE.md rule 2 (no silent fallbacks where data integrity is at stake — this decides whether a run is admitted), the plan requires a **loud refusal** naming the model and the fact that its provider could not be resolved. Observable: error contains the model string and the substring `could not resolve provider` | the message is produced only on the `provider == ""` path; the *model string* in it distinguishes this branch from T-B5's, whose message names an env var instead |

**Risk note attached to T-B6:** a loud refusal on unknown providers could refuse a lane whose
`agent_model_name` is a bare local alias. Measured mitigation: all 17 motoko lanes' `agent_model_name`
values were enumerated this session and every one begins with `ollama/` or `openrouter/`, both of
which `GuessProvider` resolves (V25/V26). Coverage gate: **T-B6b** asserts, as a table, that
`GuessProvider` returns non-empty for **all 17** `agent_cli: "motoko"` lanes read out of
`internal/eval_harness/models.yml` — so the day a new motoko lane is added with an unresolvable
model string, the test fails at add-time rather than at rig-time.

### 3.3 Ordering (doc §12.3)

| id | claim | mutation | observable |
|---|---|---|---|
| **T-ORDER** | the resolved-provider refusal never runs ahead of `MOTOKO_REPO` discovery | move `requireProviderCredential` from `ExecuteStreaming` into `runHealthCheck` **before** the version query (the exact defect §12.3 warns about) | with a mock `motoko` stub that prints `motoko_repo=/tmp/fake-repo` for `--version`: after `HealthCheck` + `Execute` on an ollama-model task, the child process env captured by the stub contains `MOTOKO_REPO=/tmp/fake-repo` (**non-empty**). Under the mutation with the key unset, the process is never spawned, so no env is captured at all — the row fails. Value set = the observed `MOTOKO_REPO` value (path strings), not a boolean |

### 3.4 D1b schedule (M3)

| id | mutation | observable |
|---|---|---|
| **T-D1B-1** | revert to `for arm in on off` around a whole-suite call | the stub-`$BIN` invocation log: 2 invocations each naming 6 benchmarks, instead of 12 each naming 1 |
| **T-D1B-2** | fix the lead arm (`ON` always first) instead of alternating on `i%2` | the (benchmark, arm) sequence: `#ON-lead = 6, #OFF-lead = 0`, violating `≤ 1`. Observable is the **sequence**, whose value set is the 2^6 lead assignments |
| **T-D1B-3** | pass `$FMT_BENCH_LIST` (all 6) instead of `${FMT_BENCHES[i]}` inside the loop | each logged invocation's `--benchmarks` argument has 6 comma-separated names instead of 1 |

### 3.5 D2 verdicts (M4) — fixture-driven

| id | mutation | observable |
|---|---|---|
| **T-D2-VOID-ORDER** | neuter the order-integrity gate | on a fixture that is a perfect arm block: verdict changes from `VOID`/`order_integrity` to a numeric verdict. Observable = the `verdict` + `void_reason` pair (4 verdicts × N reasons), not "did it error" |
| **T-D2-VOID-TREAT-ON** | neuter the `>20% ON quarantined` rule | fixture with 2/6 ON rows `validity.valid=false`: verdict changes from `VOID`/`treatment_unproven_rate` |
| **T-D2-VOID-TREAT-OFF** | neuter the OFF-contamination rule | fixture with 1 OFF row carrying `fmt_hook_events`: verdict changes from `VOID`/`control_contaminated` |
| **T-D2-CENSOR** | neuter rule 1 (one-passes-wins) so both-fail and one-passes both tie | fixture with 10 one-arm-passes pairs: `n_eff` drops 10→0 and verdict flips `KEEP`→`RETIRE` (n_eff<24 after slots). Observable = `(n_eff, W, verdict)` triple |
| **T-D2-MARGIN** | neuter the `\|log ratio\| ≤ 0.10` tie margin | fixture of both-pass pairs all within ±5%: `n_eff` 0→12 and a spurious `W` |
| **T-D2-KEEP-3** | neuter each of the three KEEP conjuncts in turn (sign test / median ratio ≤ 0.90 / McNemar guardrail) | three fixtures, each failing exactly one conjunct: verdict must be non-`KEEP` in all three; a mutant dropping any conjunct returns `KEEP` on its fixture |
| **T-D2-REAL** | *(no mutation — an end-to-end oracle)* | the real banked AC5 arms `eval_results/ab2_fmt_{on,off}` → `VOID` with an order-integrity reason. Known in advance from V19, independently measured, and **not** the happy path |

---

## 4. Baselines — every acceptance command run on the UNMODIFIED tree first

Run 2026-08-20 on **darwin/arm64** (macOS 26.5.2) in
`/Users/voightkampff/.ailang-driver-pin/.wt-motoko-iter14-fmt-d1` at `44aa3cab4`, clean tree.

| command | BASE result | note |
|---|---|---|
| `go build ./...` | ❌ **exit 1** | `# github.com/sunholo-data/ailang/cmd/wasm` / `runtime.main_main·f: function main is undeclared in the main package`. **`go build ./...` is RED at base** (the wasm package builds only under `GOOS=js`), so it cannot be an acceptance gate — it would measure the repo, not the change |
| `go build ./internal/... ./cmd/ailang/...` | ✅ exit 0 | **this is the build gate** |
| `go vet ./internal/executor/... ./internal/ai/...` | ✅ exit 0 | |
| `go test ./internal/executor/... ./internal/ai/...` | ✅ exit 0 | 14/14 packages `ok` (motoko 27.1 s) |
| `env -u OPENROUTER_API_KEY go test ./internal/executor/motoko/` | ✅ exit 0, `ok … 20.422s` | the M1 regression gate; green at base, must stay green |
| `bash scripts/check_boundaries.sh` | ✅ exit 0, "OK: no architecture boundary violations" | |
| `golangci-lint run ./internal/executor/motoko/... ./internal/eval_analysis/... ./cmd/ailang/...` | ✅ exit 0, "0 issues" | |
| `bash -n tools/launchd/nightly-eval.sh` | ✅ exit 0 | |
| `shellcheck -S warning tools/launchd/nightly-eval.sh` | ✅ exit 0, 0 findings | shellcheck present at `/opt/homebrew/bin/shellcheck` |
| `grep -n OPENROUTER_API_KEY internal/executor/motoko/*.go \| grep -v _test.go` | 1 hit (`healthcheck.go:64`) | control: same grep over `internal/ai/config.go` → 2 hits (`:115`, `:143`) — instrument discriminates |
| `PlistBuddy … dev.ailang.nightly-eval.plist` | ✅ exit 0, path = V1's checkout | control: nonexistent plist → exit 1 |
| `jq -e . .ailang/state/sprints/sprint_m-motoko-fmt-remeasurement-instrument.json` | — | produced by this sprint plan; validated on write |

**Not baselined here** (the executor must baseline them before its first edit, and record the
result): `go test ./internal/eval_analysis/... ./cmd/ailang/...` (AC-M4-1). A red base there is a
repo finding, not a sprint failure.

---

## 5. Risks

| # | risk | mitigation |
|---|---|---|
| R1 | The 8 mock-binary `Execute` tests break in CI where `OPENROUTER_API_KEY` is unset (they pass locally only because the ambient env has it) | Measured and named in M1; **AC-M1-3** (`env -u OPENROUTER_API_KEY go test ./internal/executor/motoko/`) is the gate, green at base |
| R2 | T-B6's loud refusal on an unresolvable provider refuses a legitimate lane | **T-B6b** asserts `GuessProvider` resolves all 17 motoko lanes read from `models.yml`, so a future unresolvable lane fails at add-time |
| R3 | AC-M2-control needs `OPENROUTER_API_KEY` + a few cents of metered spend, and the rig | Flagged as M2's only non-local dependency; the control leg is one trivial canary on the cheapest OpenRouter motoko lane. If the control does not fire, AC-M2-treatment is **void** — the doc says so and this plan does not soften it |
| R4 | D2's shape drifts from the doc's §2/§5/§7 during implementation | Every D2 rule is quoted from the doc in M4, and **AC-M4-5** pins the analyzer against a real, independently-measured dataset (AC5) whose correct verdict (`VOID`, order integrity) was established by V19 before the analyzer existed |
| R5 | The fix merges but never reaches the rig (issue #558) | **AC-DEPLOY** (§2) reads the script at the path named in the *installed plist*, plus the `$BIN` git rev. Until both pass, no slot counts |
| R6 | M4's new `BenchmarkResult` fields collide with the existing loader's strictness | Fields are additive with `omitempty` and JSON tags copied from the already-banked keys (`metrics.go:138,143`); **AC-M4-3** asserts `eval-paired`'s output is byte-identical before and after |

---

## 6. Documentation obligations (`.claude/rules/coding-standards.md`)

- `CHANGELOG.md` — one entry per milestone, grouped.
- `docs/docs/guides/evaluation/` — document `ailang eval-censored-pairs` (M4).
- `cmd/ailang/help.go` — `eval-censored-pairs` line (M4), per the `cli-doc-maintainer` convention that help is the source of truth.
- The design doc's §12.5 Verification Log gains rows for whatever the sprint measures live (AC-M2 especially): the doc's §12.4 explicitly moved the live arm into acceptance, so the result is owed back to the doc.
- No `examples/*.ail` obligation: **no AILANG language feature is added** — mission default bias holds, this is entirely harness lane.

---

## 7. Handoff

Sprint JSON: `.ailang/state/sprints/sprint_m-motoko-fmt-remeasurement-instrument.json`
Execute milestones **in order M1 → M2 → M3 → M5**, with **M4 parallel-safe** (it touches
`internal/eval_analysis` + `cmd/ailang` only, disjoint from M1–M3, M5). M3 must land before M5
(M5 inserts into the block M3 restructures).
