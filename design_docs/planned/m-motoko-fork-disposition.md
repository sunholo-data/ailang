# Motoko fork disposition — iteration 1

Measured on 2026-08-12 by reading each non-merge commit in fork `9e8d647f18eda6cb32ee9aa82afe8d16f157e537` and comparing its mechanism with `origin/main_dst` at `303d8697caa50cf6ce0552560f26c4c92f4e56be`, using path-scoped content searches and the likely upstream host modules. The range contains 52 commits, but `ed61097` is a merge carrying no unique content: `git -C /Users/voightkampff/dev/mk-ast diff --quiet ed61097^2 ed61097; echo $?` returned `0`; therefore 51 rows is correct.

| R# | sha | date | subject (trimmed) | VERDICT | Evidence (command + result) | Confidence |
|---:|:---|:---|:---|:---|:---|:---|
| R1 | `6dfc5b1bf` | 2026-06-10 | Reenabled local models | SUPERSEDED | `git show 6dfc5b1` adds `openaiBaseUrl`/`openai/` routing. Scoped `git grep -il openaiBaseUrl ... -- src/tui/src` => 4/69 files; upstream `session.ail` also strips the `openai/` prefix. | high |
| R2 | `d24846614` | 2026-06-16 | local eval A/B profiles | DROP | `git show d248466` => three one-off `ollama_docs`, `ollama_dp7`, `ollama_microrag` profiles. `ls-tree ... .motoko/config` => 44 current files with first two profiles absent and current `microrag`/`ollama` profiles present. Historical experiment inputs, not a mechanism to migrate. | high |
| R3 | `b2f5dcc85` | 2026-06-19 | allowlist OLLAMA max tokens | SUPERSEDED | `git show b2f5dcc` adds child env forwarding. Scoped grep for `AILANG_OLLAMA_MAX_TOKENS` in `src/tui/src` => 1/69, at upstream `runtime-process.ts:342`. | high |
| R4 | `cb58a801f` | 2026-06-20 | compaction structural telemetry | SUPERSEDED | `git show cb58a80` adds `compaction_structural`. Scoped grep in `src/core` => 8/68; upstream `session.ail` emits phased compaction counts plus pinned-prefix and token-window telemetry. | high |
| R5 | `05f60d8da` | 2026-06-23 | continue after length truncation | UNRESOLVED | `git show 05f60d8` adds `MOTOKO_CONTINUE_ON_TRUNCATION`; scoped grep `continue_on_trunc` in `src/core` => 0/68. `step_machine.ail` enumerates terminal reasons but the inspected output did not establish provider `finish_reason=length` behavior. | low |
| R6 | `ce938f239` | 2026-06-23 | DP7 type-check gate default | DROP | `git show ce938f2` installs `ailang check .`; R7 makes it conditional and R29 replaces it with `ailang ai-check`. This version is superseded by our own later commits. | high |
| R7 | `5c74b5dcd` | 2026-06-23 | conditional DP7 type-check | DROP | `git show 5c74b5d` adds `workspace_has_ail` and conditional `ailang check`; R29 replaces the command and closes the stated verification hole. | high |
| R8 | `96542f898` | 2026-06-24 | reserve 75k output headroom | **PORT** *(settled 2026-08-14, iteration 4)* | Settled by running the sharpened instrument (see *Residuals*), `tools/motoko/r8_headroom_band.ail` against `main_dst@6c06b08`, replicating the live `session.ail:2534-2561` wiring. Arm A (`docx_lambda`-class history: one large **user** message, which the ladder can never elide — `elide_walk` only touches `role=="tool"`): the ladder removed **2,061 of 208,980** tokens (1%), hit its last branch and returned `structural: tier=floor keep_last=1` at **79%** — above its own 70% target. The seal then saw 207,239 tok = **79% < 95%** and returned **`Ok`**, i.e. it SENT, leaving headroom **54,905 < the 65,536 output cap**. Per this row's own decision rule that is **PORT**. Controls both fire: arm B (small history) → `PassThrough`, `Ok`, headroom 259,642; arm C (158%) → **`Err(SealExhausted)`**, so the seal genuinely refuses and arm A's `Ok` is a real permission, not a dead gate. | high |
| R9 | `1d9d8453d` | 2026-06-24 | register EditDecl/ReadInterface | UNRESOLVED | `git show 1d9d845` adds schemas and SYSTEM guidance; scoped grep `EditDecl` in `src/core` => 0/68. The new dispatch architecture exists, but whether these two tools remain desired was not established. | low |
| R10 | `087e68e0e` | 2026-06-24 | retry empty response | UNRESOLVED | `git show 087e68e` adds bounded `MOTOKO_RETRY_EMPTY`; scoped grep `retry_empty` => 0/68. Upstream has `recovery.ail`, but its inspected policy covers retryable stream errors and persist nudges, not demonstrably empty successful responses. | low |
| R11 | `21331d636` | 2026-06-24 | count tool-call args in tokens | PORT | `git show 21331d6` adds `tool_calls_chars`; scoped grep for that helper => 0/68. Upstream `compaction.ail:estimate_tokens_messages` visibly folds only `length(m.content)`, so port argument sizing into the phased estimator (`compaction.ail`/`context_usage.ail`). | high |
| R12 | `869fefadf` | 2026-06-24 | relative lock paths | DROP | `git show 869fefa` changes two developer-local package paths to relative paths; subsequent dependency/lock commits rewrite these entries. Obsolete lockfile churn, not unique behavior. | high |
| R13 | `49b6b58ee` | 2026-06-23 | gated AST autoread | PORT | Controller scoped measurement: `autoread` ours=4 files, upstream=0, with 68 upstream `src/core` files as firing control. Port the read-routing mechanism to `src/core/tool_runtime.ail` and new dispatch/envelope layers. | high |
| R14 | `96382d54e` | 2026-06-23 | forward autoread env | PORT | `git show 96382d5` adds both AST env keys to child env; controller found upstream `autoread`=0 with firing controls. Port configuration/forwarding through upstream `runtime-process.ts` and core policy plumbing. | high |
| R15 | `5fea09054` | 2026-06-24 | per-edit AILANG check | PORT | `git show 5fea090` adds `ail_typecheck_after_edit` and result feedback; scoped grep `per_edit` => 0/68. Port as a tool-result enrichment in `tool_runtime.ail`/`tool_phase.ail`, not the old loop. | med |
| R16 | `dad0992b2` | 2026-06-24 | retry recoverable stream errors | **UNRESOLVED** *(was SUPERSEDED/high — downgraded by the sprint-evaluator, controller-reproduced)* | Upstream `recovery.ail:17` does export `should_retry_stream_error`, but reading the body rather than the name refutes equivalence: it is `retryable && retry_enabled && remaining_step_budget > 1`, and its own comment says a non-retryable error should "fail fast instead". Ours (`dad0992`) retried an `AIError code=Internal` from a Go-side XML parse failure **unconditionally** — that was the point of the commit. Control-verified contradicting evidence in the very file the original row cited: `session.ail:2379` constructs `Err({ code: "Internal", message: msg, retryable: false })`. A name match was read as a behavioural equivalence. See Residuals. | — |
| R17 | `18cb6ac35` | 2026-06-24 | compaction from provider tokens | DROP | `git show 18cb6ac` introduces previous-call `last_input_tokens` gating; R19 and then R26 explicitly replace this unstable scheme. Superseded by our own later calibration. | high |
| R18 | `623c144ef` | 2026-06-24 | chars/7 calibration | DROP | `git show 623c144` changes the ratio; R19 replaces ratio-driven elision and R26 documents this iteration as inadequate. | high |
| R19 | `b7dc43d7d` | 2026-06-25 | actual-token elision | DROP | `git show b7dc43d` adds `compact_step_actual`; R26 explicitly replaces stale post-compaction gating with current-history affine calibration. | high |
| R20 | `bc785210f` | 2026-06-25 | wire compaction_ai | SUPERSEDED | `git show bc78521` adds compaction-ai config. Scoped grep `compaction_ai` => 10/68 core files; upstream profiles list it and `session.ail` identifies/counts the extension stage. | high |
| R21 | `159125e29` | 2026-06-26 | pin system message | SUPERSEDED | `git show 159125e` partitions system messages. Upstream `session.ail` imports `split_for_compaction`, records `system_prefix` count/chars/digest, and passes the pinned prefix separately. | high |
| R22 | `f9f722e47` | 2026-06-26 | env-authoritative system prompt | SUPERSEDED | `git show f9f722e` adds `SYSTEM_MD` selection and telemetry. Scoped grep `SYSTEM_MD` in `src/tui/src` => 5/69; upstream has `system-prompt.ts`, config tests and RPC handling. | high |
| R23 | `ec90f2263` | 2026-06-26 | forward SYSTEM_MD | SUPERSEDED | `git show ec90f22` forwards the variable. Scoped grep `SYSTEM_MD` => 5/69 and upstream `system-prompt.ts` is the dedicated mechanism. | high |
| R24 | `c5b09240a` | 2026-06-26 | derive env forwarding maps | SUPERSEDED | `git show c5b0924` introduces map-derived forwarding. Upstream `runtime-process.ts` has centralized `buildChildEnv` with explicit profile/runtime inputs; the phase rewrite removes old core env-gated feature plumbing. | med |
| R25 | `b58b0d327` | 2026-06-26 | materialize SYSTEM_MD | SUPERSEDED | `git show b58b0d3` materializes an out-of-workspace prompt. Scoped grep `materialize` in `src/tui/src` => 3/69, in upstream `system-prompt.ts`, `index.ts`, and its harness test. | high |
| R26 | `46e9eae43` | 2026-06-28 | calibrated history estimate | SUPERSEDED | `git show 46e9eae` introduces calibrated current-history sizing. Scoped grep `calibrated` => 4/68; upstream `compaction.ail` implements `affine_calibrate` and anchored calibrated usage, consumed by `session.ail`/`step_machine.ail`. | high |
| R27 | `52d3d24eb` | 2026-06-28 | behavioral done gate | PORT | `git show 52d3d24` adds `MOTOKO_REQUIRE_TEST`; scoped grep for it in `src/core` => 0/68. Port the behavioral policy into the upstream terminal/finalization phase. | high |
| R28 | `4b752ad6d` | 2026-06-28 | forward REQUIRE_TEST | PORT | `git show 4b752ad` adds env forwarding; scoped grep `MOTOKO_REQUIRE_TEST` => 0/68. Port via upstream typed profile/policy loading and launcher env only as needed. | high |
| R29 | `9da17835b` | 2026-06-29 | done gate uses ai-check | PORT | Controller measurement: `ai-check` is in 13 whole-tree files but 0 under upstream `src/core` (68-file firing control), versus ours=1. Port to upstream finalization/terminal phase. | high |
| R30 | `a1230dc05` | 2026-06-29 | direct autoread env reads | PORT | `git show a1230dc` replaces shell probing with `getEnvOr`; controller found AST autoread absent upstream with firing controls. Preserve the fix when porting R13 into typed policy/env plumbing. | high |
| R31 | `c3b0e69dc` | 2026-06-29 | default max_steps 120 | SUPERSEDED | `git show c3b0e69` changes a loader default. Scoped grep `max_steps` => 6/68 and profile grep lists current values across upstream configs; upstream `step_machine.ail` has an explicit step budget policy. | high |
| R32 | `8c2dc5d55` | 2026-06-29 | unified env forwarding | SUPERSEDED | `git show 8c2dc5d` removes shell env probes and auto-forwards names. Upstream phase-core uses typed policy plus `env_client.ail`/ports and centralized `buildChildEnv`; old scattered gate reads no longer host core policy. | med |
| R33 | `e3878dc4c` | 2026-06-29 | autoread default on | PORT | `git show e3878dc` changes autoread to opt-out default; controller found the whole autoread mechanism absent upstream. Carry the default in the new typed tool policy. | high |
| R34 | `9f013e917` | 2026-06-29 | broadcast resolved gates | **PORT** *(was SUPERSEDED/med — downgraded by the sprint-evaluator, controller-reproduced)* | The original row named `phase_vocab.ail` as the superseding mechanism but that module is types + pure helpers, not an emitted audit event. Controller re-measured, scoped to upstream `src/core`: `runtime_config_resolved`, `config_resolved`, `policy_resolved`, `resolved_config` are **all 0**, same-path control `session` = **29 files** firing; ours = 1. No upstream resolved-config broadcast exists. This matters more than a lone row: several toggles this event audits are themselves PORT (autoread, require_test) or UNRESOLVED (retry_stream_error, retry_empty), so dropping the visibility mechanism would re-hide exactly the gates we are carrying over — the "a silently-off feature is a step-0 log line, not an hours-later discovery" property. | high |
| R35 | `ba683eb32` | 2026-06-29 | whitespace-tolerant EditFile | PORT | Controller scoped measurement: `ws_window_eq`, `ws_trim_each`, `ws_drop_leading_empty` each 0 upstream; control=126 upstream `src` files containing `func`. Target `src/core/tool_runtime.ail` exists. | high |
| R36 | `ce4a3e59c` | 2026-06-30 | per-benchmark max-step cap | PORT | Controller measurement: `MOTOKO_MAX_STEPS` ours=1 file, upstream=0 with firing control. Port as a typed policy override feeding `step_machine.ail`'s step budget. | high |
| R37 | `ec8af0974` | 2026-06-30 | realpath path validation | DROP | `git show ec8af09` adds `realpath_both`; R38 is its exact revert. No net desired fork content. | high |
| R38 | `de479acb9` | 2026-06-30 | revert realpath validation | DROP | `git show de479ac` removes R37 exactly. This bookkeeping revert carries no feature to port. | high |
| R39 | `8a14fe059` | 2026-06-30 | relative iface path | PORT | `git show 8a14fe0` fixes the R13 subprocess path/module name. Scoped grep `iface` => 1/68 but only in `ai_compat.ail`, not tool runtime; include relative-path invocation in the autoread port. | high |
| R40 | `6910983f1` | 2026-07-20 | add Message images fields | SUPERSEDED | `git show 6910983` adds `images: []`. Scoped grep `images:` => 10/68 across upstream live core modules, showing the v0.30 Message shape is already adopted. | high |
| R41 | `5f22e1255` | 2026-07-20 | compaction test model change | DROP | `git show 5f22e12` adjusts a test solely for the old fixed-75k implementation. Upstream uses deployment context limits and phased compaction tests; the old test premise is obsolete. | high |
| R42 | `7861c64d7` | 2026-07-20 | dependency bumps for images | DROP | `git show 7861c64` is registry/lock version churn supporting R40. Upstream already uses the new Message shape and has a different dependency graph. | high |
| R43 | `e8719bbaa` | 2026-07-23 | wire fmt extension/profile | PORT | `git show e8719bb` adds registration, package pin and `ollama_fmt` profile. Scoped grep `motoko_ext_fmt` => 0/68; port registration/profile into upstream generated extension registry and profile model, using final R51 version. | med |
| R44 | `13bc08511` | 2026-07-28 | effect-row soundness boot fix | SUPERSEDED | `git show 13bc085` rewrites old core/package effects. Scoped grep `effect row` => 11/68 across upstream phase/DST modules, whose rewritten signatures already compile against the sound effect system. | high |
| R45 | `e2dbff86b` | 2026-07-28 | cloud profile | PORT | `git show e2dbff8` adds a cloud profile and extension configs. Scoped grep `cloud` in `.motoko/config` => 0/44; port as an upstream profile directory, updated to its split-config schema. | med |
| R46 | `f7bbe8dca` | 2026-07-29 | broadcast resolved profile | UNRESOLVED | `git show f7bbe8d` emits `MOTOKO_CONFIG`; scoped grep `resolved_profile` => 0/68. Upstream has typed profile state, but inspected output did not establish that the live trace reports the selected profile value. | low |
| R47 | `60da7d030` | 2026-07-29 | broadcast loaded extension ids | UNRESOLVED | `git show 60da7d0` emits joined live hook IDs. Scoped grep `extension_ids` => 2/68, both DST driver files; no observed live-session payload proved equivalent IDs. | low |
| R48 | `674bff979` | 2026-07-29 | fmt 0.1.1 repin | DROP | `git show 674bff9` is only a version repin later replaced by R49–R51. | high |
| R49 | `6a1eef26f` | 2026-07-30 | fmt 0.4.0 repin | DROP | `git show 6a1eef2` is only an intermediate dependency/lock repin, replaced by R50 and R51. | high |
| R50 | `c8e47283e` | 2026-07-30 | fmt 0.4.1 repin | DROP | `git show c8e4728` is only an intermediate repin, replaced by R51. | high |
| R51 | `9e8d647f1` | 2026-07-30 | fmt 0.4.2 fail-soft write | PORT | `git show 9e8d647` pins final `motoko_ext_fmt` 0.4.2; scoped grep `motoko_ext_fmt` => 0/68. Use this final version when porting R43 registration/profile. | high |

## Counts

Post-review (see *Review history* below). The parenthesised figure is the executor's first pass.

- SUPERSEDED: **14** (was 16)
- PORT: **17** (was 15 at first pass, 16 post-review)
- DROP: **14** (unchanged)
- UNRESOLVED: **6** (was 6 at first pass, 7 post-review)
- Total: **51**

The UNRESOLVED count is **6**. Recounted independently from the VERDICT column, not carried over
from the executor's own tally.

**Settled since review:** **R8 → PORT** at mission iteration 4 (2026-08-14), by measurement rather
than by re-reading — see its row and the *Residuals* entry. That is the first of the seven
post-review UNRESOLVED rows to close, and it closed in the direction the row's own decision rule
named, not in the direction the sharpening guessed at.

## Review history

The first pass scored **65/100 — FAIL** from an independent sprint-evaluator (sonnet; the executor
was `codex:gpt-5.6-sol`, so generator ≠ judge). Two rows were downgraded and the controller
reproduced both findings first-party before applying them:

- **R16** SUPERSEDED → UNRESOLVED. The strongest single result in this review: the cited upstream
  function `should_retry_stream_error` exists, and reading its *body* (`retryable && retry_enabled
  && remaining_step_budget > 1`) refutes the equivalence its *name* implies. Directly contradicting
  evidence — `session.ail:2379`, `retryable: false` on a `code: "Internal"` error — was in the same
  file the row cited.
- **R34** SUPERSEDED → PORT. The named superseding mechanism (`phase_vocab.ail`) is a types module,
  not an emitted event; four candidate tokens measure 0 upstream against a firing 29-file control.

Both were the failure mode the directive named as the primary attack surface — **a token match read
as a behavioural equivalence** — and both were in SUPERSEDED, the one verdict whose error is
irreversible. 2 of 16 SUPERSEDED rows (12.5%) were wrong in the direction that silently loses work.
Treat the surviving 14 accordingly: they are the best available reading, not a settled fact.

One MEDIUM finding was recorded and deliberately **not** re-verdicted: **R44**'s evidence cell rests
on a grep for the English phrase "effect row" rather than a named mechanism, and the row is arguably
DROP ("fixed a bug in code that no longer exists in that form") rather than SUPERSEDED. The
evaluator judged the end effect identical — nothing to port either way — so this is an evidentiary
defect, not a work-loss risk. It is flagged here rather than silently corrected.

## Method and its limits

Each row began with `git show <sha>` (the distinctive added behavior is summarized in the evidence cell), then used `git grep -il -- <token> origin/main_dst -- <scope>` against the module family where the mechanism would live. Every zero relied upon in a verdict was paired with an observed nonzero scope control: 68 files under `src/core`, 69 under `src/tui/src`, or 44 under `.motoko/config`; the initially malformed multi-path controls that returned zero files were discarded and rerun one scope at a time. Likely host modules were read at the upstream revision, especially `session.ail`, `step_machine.ail`, `recovery.ail`, `compaction.ail`, `context_usage.ail`, and `runtime-process.ts`. No verdict rests on a whole-tree grep: the only whole-tree figure mentioned is the controller's 13-file `ai-check` warning, and the verdict uses its scoped `src/core` result. This instrument sees committed text and static architecture; it cannot prove runtime equivalence, extension ordering, provider-specific terminal behavior, or telemetry payloads without executing focused scenarios. A positive token match is not by itself treated as supersession, and a scoped zero is not by itself treated as proof that a feature is wanted.

## Residuals

- **R16 — stream/parse-error retry** *(added at review)*: establish how `e.retryable` is actually
  set for stream/parse-class errors originating from `stepWithStream`. That assignment lives in
  AILANG's `std/ai` Go runtime, **outside this repo**, which is why static analysis of `main_dst`
  cannot settle it. If an `Internal`-coded XML/parse failure arrives `retryable: false`, upstream
  does **not** supersede `dad0992` and the row becomes PORT; if it arrives `retryable: true`,
  upstream's gate covers it and the row becomes SUPERSEDED.
- **R5 — length truncation:** run a scripted provider scenario through upstream `session.run_v2_from_messages` whose first successful response has `finish_reason="length"` and no tool calls, then assert whether a second provider step occurs within budget and history contains the continuation instruction.
- **R8 — output headroom:** ~~run the same long-context transcript/model limit through fork and upstream with a provider that attempts a large completion; record pre-call input tokens, allowed output tokens, compaction tier, and whether either request exceeds total context.~~ **SHARPENED 2026-08-13 (mission iteration 3) — the old wording asks for a whole rig A/B, and a static read has already settled most of it** (migration doc V27–V29, measured against `main_dst@6c06b08`). What is now *known*: the live ladder targets **70%** (`result_target_pct`), which on a 262144 window leaves ≥78,644 — more than both the 65,536 output cap and our own 75k reserve; and the live refusal is the phase core's `seal_compacted_payload` at `exhaustion_pct() = 95`, whose predicate is **input-only**. So the residual is a single, much narrower question, and it is the one the settling measurement should target: **is the band between the ladder's 70% target and the seal's 95% permission reachable in practice?** The ladder's last branch (`compaction_structural.ail:191`) sends unconditionally when even `keep_last=1` misses 70%, so the band is reachable *by construction*; what is unmeasured is whether a real `docx_lambda`-class transcript gets there. **Cheapest sufficient instrument** (and it is not a rig run): drive `compact_for_pre_step` directly at `limit=262144` with a scripted history whose floor form still exceeds 183,500 estimated tokens, and assert the returned note is `tier=floor`; then feed that payload to `seal_compacted_payload` and read whether it returns `Ok` (⇒ over-target input is SENT with <65,536 of window left ⇒ **PORT**, as a phase-core seal argument, not an extension reserve) or `Err(SealExhausted …)` (⇒ the hard stop catches it ⇒ **SUPERSEDED**, and the residual is a *starved completion*, not an overflow). Both functions are `pure`, so this is a unit-level assertion, not a provider A/B. **Note the instrument must not use `compact_step_with_limit`** — V28: zero production callers, and its 95% `Err` is the one the migration doc wrongly credited.
  **SETTLED 2026-08-14 (mission iteration 4) → PORT.** The instrument was built and run:
  `tools/motoko/r8_headroom_band.ail`, against `main_dst@6c06b08`, replicating the live
  `session.ail:2534-2561` wiring rather than calling the two functions in isolation — `split_for_compaction`,
  `ext_limit = context_limit - estimate_tokens_messages(split.pinned)`, `compact_for_pre_step` over the
  segment, then `seal_compacted_payload(split, chain, model, context_limit, false)`. Measured, three arms:
  **A (the question)** pinned=320, ext_limit=261,824; segment 208,980 → 206,919 tokens (the ladder removed
  **1%**), decision `structural: tier=floor keep_last=1` at **79%**, seal saw 207,239 tok = **79%**, returned
  **`Ok`**, headroom **54,905 < 65,536**. **B (negative control)** small history → `PassThrough`, `Ok`,
  headroom 259,642. **C (over-band control)** 158% → **`Err(SealExhausted)`**.
  So the band is reachable *in practice*, not merely by construction, and the seal permits it — the
  `Ok` branch, which is the PORT half of this row's own rule.
  **The mechanism is worth more than the verdict, and it is not the one the sharpening assumed.** The
  sharpening pictured a transcript that grinds down the ladder; what actually happens is that the ladder
  has **no lever at all**. `elide_walk` only rewrites `role=="tool"` messages, so a large **user** message
  — a pasted document, the `docx_lambda` shape — is invisible to every tier. All four tiers therefore
  produce near-identical output, the unconditional floor branch fires, and 79% is sent unchanged. Raising
  keep-last aggressiveness would not move this case by one token; only a reserve at the seal does. That is
  why the ask is `session.ail:2561` and not the extension, and it is the same move upstream already makes
  one line up at `:2534` (`context_limit - pinned_tokens`) — now confirmed first-party by reading the call
  site, not inherited from the row.
- **R9 — EditDecl/ReadInterface:** obtain the migration product decision for whether these public tools remain supported, then run upstream tool-catalog/dispatch tests for both schemas and calls. Desired plus absent means PORT; deliberately removed means DROP.
- **R10 — empty response:** scripted provider test: first successful `stop` response has empty/whitespace content and no tool calls, second response is nonempty. Observe whether upstream retries once, terminates, or injects only the persist nudge.
- **R46 — resolved profile telemetry:** launch two upstream profiles and inspect the first live session event/trace payload. A field containing the resolved loaded profile (not merely requested CLI text) settles SUPERSEDED; absence settles PORT to `phase_vocab.ail`/`session.ail`.
- **R47 — loaded extension IDs telemetry:** launch a profile with two known extensions and inspect the first live session event. Exact loaded IDs after registry construction settle SUPERSEDED; only counts/requested IDs or no field settles PORT to the extension-runtime/session trace boundary.
