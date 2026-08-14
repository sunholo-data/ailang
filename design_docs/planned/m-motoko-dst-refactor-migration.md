# M-MOTOKO-DST-MIGRATION: Adopt Arni's phase-core/DST refactor and re-prove our fork's improvements

**Status**: Planned (2026-08-11)
**Target**: rolling — operational; no AILANG core changes
**Priority**: P0 — once [#154](https://github.com/arniwesth/motoko_agent/pull/154) lands on `main`, our fork is 805 commits behind a tree whose ABI it cannot build against. Every motoko eval depends on closing this.
**Estimated**: 3 phases; Phase 1 is ~12 mechanical package ports; Phase 0 and Phase 3 each carry a
fixed timebox (28 mission fires ≈ 14 days at the 12h interval) with a fail-closed expiry action —
neither is open-ended
**Owner**: sunholo (we co-develop motoko with @arniwesth)
**Dependencies**: #154 merged to `origin/main`; Arni's extension republish (he states the ABI "is still subject to change" — Phase 1 must not start before it settles; bounded by the Phase 0 gate below)

## TL;DR

Arni has rewritten motoko's core as a functional core with an imperative shell, and built a
deterministic simulation testing (DST) framework on top. **We are adopting it.** The refactor is
strictly better architecture than our fork carries, and three of the four legs of our compaction
work are superseded by better mechanisms.

The migration is not free: the extension ABI goes **2.2.0 → 5.0** (breaking), **12 of our
extension packages** need porting, and **5 of our 6 eval profiles** plus the `motoko_ext_fmt`
extension do not exist upstream. This doc sequences that work and — critically — separates
*adopting his architecture* from *re-proving our improvements*, so we carry forward only what
survives measurement.

## Problem Statement

Our eval-canonical fork (`~/dev/mk-ast`, `sunholo/eval-canonical`, HEAD `9e8d647`, 2026-07-30)
sits **52 commits ahead / 805 commits behind** `origin/main_dst`. Three concrete pressures:

1. **The ABI break is total.** All 12 of our extension packages pin `motoko_ext_abi = "2.2.0"`.
   ABI 5.0 changes four of the five effectful hook slots — both their return *types* and their
   effect *rows*. Nothing we own type-checks against it.

2. **Our fork's architecture is the thing being deleted.** Our 52 commits are concentrated in
   `agent_loop_v2.ail` (95,868 bytes in our tree; a 4,005-byte compatibility facade in his) and
   `compaction.ail` (17,543 → 5,804). Staying on our fork means maintaining a 1,727-line loop
   that has no deterministic test path, against an upstream that has one.

3. **We cannot cheaply prove our own improvements.** Resolving a paired A/B currently costs 7–14h
   of rig time (see the extension-fix baseline shift (rig memory)). The DST framework — seeded
   generation, virtual clock, exact-program strict replay, invariants over a `LedgerTrace` rather
   than over model prose — is a much cheaper instrument **for the core loop**.
   **CORRECTED 2026-08-12 — it does not reach our extensions, and that is where our value is.**
   Measured from Arni's own closing note: extension coverage is **1 of ~40 covered hooks
   substantively world-mediated**, across 15 extensions; a whole profile's 32 covered hooks are
   *"entirely of no-ops"*. What the framework gives extensions is a **contract layer**
   (`declared_vs_performed`, `conformance`, `hook_guard`, `ext_call_inventory`) — valuable for the
   ABI port, useless for "does fmt save tokens". See the charter's *DST scope* section.

The risk of a naive merge is that our measured wins vanish silently. `motoko_ext_fmt` (a −74%
tokens-to-pass result, the fmt A/B result (rig memory)) does not exist upstream, and
14 of our 18 `motoko_profile:` eval entries point at profiles that do not exist in `main_dst`.

## Goals

**Primary goal:** Rebase our eval harness onto the phase-core/DST architecture without losing any
improvement that still measures positive — and without carrying forward any that doesn't.

**Success metrics:**

1. `make check_core && make verify_extensions` green on a `main_dst`-based branch with our 12
   extensions ported to ABI 5.0.
2. All 18 `motoko_profile:` entries in [models.yml](../../internal/eval_harness/models.yml) resolve to a
   profile that exists.
3. Every one of our 52 fork commits is dispositioned: **superseded**, **ported**, or
   **dropped-with-evidence**. No commit silently lost.
4. `motoko_ext_fmt`'s token win is re-measured on the new tree — kept if it holds, dropped if not.
5. The 12-package ABI port is gated by motoko's **contract** instruments (`make
   declared_vs_performed`, `conformance`, `hook_guard`) rather than by hand-reading effect rows —
   the mistake that port most invites. **This metric originally read "at least one historical A/B
   answered via DST instead of a rig run"; that over-read what DST covers (see Problem Statement 3)
   and is withdrawn.**

## High-Impact Decisions

| Decision | Options | Recommendation | Who decides | Cost to change later |
|---|---|---|---|---|
| Adopt or fork permanently | adopt `main_dst` / stay on ours | **Adopt** (ratified by Mark 2026-08-11) | Mark — **decided** | Very high |
| Extension source of truth | our registry / his vendored `{path=...}` | **Registry**, and reconcile his vendored copies back to it | Mark + Arni | High — this is the drift that cost an audit once ([MOTOKO.md](../../MOTOKO.md) §3) |
| When to start Phase 1 | now / after ABI settles | **After** — he says the ABI is "still subject to change"; bounded by the Phase 0 gate | us | Medium — starting early means porting twice |
| `motoko_ext_fmt` disposition | port / upstream / drop | **Port first, propose upstream once re-measured** | us | Low |
| Profiles | restore ours / adopt his | **Restore ours** — they encode our A/B design, not his | us | Low |

### Design Freeze

**Check off before PHASE 1 starts** (reworded 2026-08-12 R2 per `gemini-3-1-pro`'s verbatim fix —
the previous "before sprint-executor starts" was a **deadlock**: Phase 0 is the thing the executor
runs to *evaluate* G1–G4, so gating the executor's start on G1–G4 permanently blocks the doc and
silently restores the unbounded manual wait R1 rejected):

- [ ] Phase 0 gate predicates all TRUE in one evaluation — G1 (#154 MERGED), G2 (`packages/` on
      motoko_agent's `origin/main`), G3 (registry exposes 5.x at a pinned digest), G4 (compile
      probe green). These are **evaluated by Phase 0, not checked by hand** — the boxes record
      Phase 0's verdict, they do not precede it
- [ ] Registry-vs-vendored reconciliation agreed with Arni (see Risks) — **queue item 11**
- [ ] Phase 1 port target confirmed as ABI 5.0 (not a later 6.x)
- [ ] **OPEN — parked for Mark (R2, `gpt5-6-sol`)**: is Arni's explicit ABI-settled acknowledgement
      a *gate predicate* (G5) or an accepted risk? The ratified charter guardrail says the port does
      not start until "#154 is merged **AND** Arni has declared the ABI stable", so the charter reads
      as G5-required — but this doc's *Declared residual* currently starts Phase 1 on G1–G4 alone.
      Until that is resolved the two documents disagree and **Phase 1 must not start**. See the
      Quorum revision log

## Solution Design

### Overview

Three sequential phases behind a gate. Phase 0 is a bounded gate, not work. Phase 1 is mechanical
and sizeable. Phase 3 is the one with genuine unknowns, and is deliberately last so it runs on a
green tree; it is timeboxed, with unresolved items promoted to named blockers rather than left open.

### Architecture: what actually changed

| Concern | Our fork | `main_dst` |
|---|---|---|
| Agent loop | `agent_loop_v2.ail`, 1,727 lines, owns decisions + effects + transcript + telemetry | `session.ail` (173k) sole loop; `agent_loop_v2.ail` is a 37-line facade |
| Effects | performed inline | injected `ExtPorts` (~270 lines): `ai_step`, clock, process, file read/mutation, path stat, dir listing |
| Provider messages | constructed in the loop | `phase_vocab.ail` is the *only* producer |
| Decision logic | interleaved with effects | `step_machine.ail` — pure `decide` |
| Testing | live-provider only | ~19 `dst_*.ail` + 52 files under `scripts/dst/`; invariants over a `LedgerTrace` |

**The ABI break, precisely** (verified by diffing `motoko_ext_abi` 2.2.0 against `main_dst`):

```
on_budget_plan:         BudgetPatch ! {Env, FS}          →  BudgetPatch                     (row removed)
on_pre_step:            PreStepDecision ! {9 effects}    →  PreStepOutcome ! {AI, IO, Trace}
on_tool_handle:         ToolHandleDecision ! {9}         →  ToolHandleOutcome ! {9 + Rand}
on_response_intercept:  ResponseInterceptDecision ! {9}  →  ResponseInterceptOutcome ! {IO, Process, FS, Clock}
on_solver_candidate:    FinalizeDecision ! {9}           →  FinalizeOutcome ! {Process}
```

Two independent changes land together:

1. **World threading.** The `Decision` payloads are unchanged — `PreStepDecision = PassThrough |
   Compacted(...)` still exists. But hooks now return `Outcome = { decision, next_state: ExtWorld }`.
   This is mechanical per hook.
2. **Effect-row narrowing.** Extensions no longer perform effects directly; they call ports. Any
   hook body that does real IO must be rewritten to route through `ctx.ports`. This is *not*
   mechanical and is where the Phase 1 estimate can slip.

`on_tool_handle` **gaining `Rand`** retires the deferral recorded in [MOTOKO.md](../../MOTOKO.md) §5 —
`motoko_ext_a2a` is back in `main_dst`'s extension list and can be compiled in again.

### Implementation Plan

**Phase 0 — Bounded, fail-closed gate (no port work).** Rewritten 2026-08-12 after the design
quorum correctly rejected the original as an unbounded wait. The exit condition is a
**conjunction of four machine-checkable predicates**, evaluated once per mission fire, under a
fixed evaluation budget with a defined action on expiry. All four must be TRUE in the same
evaluation for Phase 1 to start. Starting Phase 1 against a moving ABI means porting twice.

| # | Predicate | Command | Observed 2026-08-12 |
|---|---|---|---|
| G1 | #154 is MERGED | `gh pr view 154 --repo arniwesth/motoko_agent --json state,mergedAt` → `state == "MERGED"`, `mergedAt` non-null | `state=OPEN`, merged `-` → **FALSE** (V17) |
| G2 | The DST tree is reachable from **motoko_agent's** `origin/main`: `packages/motoko-ext-abi/ailang.toml` resolves there | `U=/Users/voightkampff/dev/arniwesth/motoko_agent; git -C "$U" fetch origin && git -C "$U" cat-file -e origin/main:packages/motoko-ext-abi/ailang.toml` exits 0 — **and the mandatory control** `git -C "$U" cat-file -e origin/main:README.md` also exits 0 | Predicate `rc=128` (*"path … does not exist in 'origin/main'"*), control `rc=0` → **FALSE, for the right reason** (V20). `packages/` arrives *with* the merge, so this is "#154 merged" as seen from the tree itself, not a duplicate of the PR API's word for it |
| G3 | The registry exposes ABI 5.x at a pinnable digest | `curl -s https://registry.ailang.sunholo.com/api/packages/sunholo/motoko_ext_abi \| jq -r '.index.versions'` contains a `5.` version | `latest=2.2.0`, `versions=1.0.0,2.0.0,2.1.0,2.2.0` → **FALSE** (V19) |
| G4 | Compile probe: the pilot port compiles against the *published* ABI | port `motoko_ext_test_dummy`'s trivial hooks to the `Outcome` shape in a scratch worktree, repin its `motoko_ext_abi` dep to the registry 5.x, `ailang check` exits 0 | unrunnable while G3 is FALSE → **FALSE** |

**Digest pinning (G3 detail).** The registry publishes a per-version `content_hash` (V19: 2.2.0 is
`sha256:60d6ec4684d6bf80d1ec800efda7aa598ccbebcbf529dd6c760fd10c5c101956`). On the first evaluation
where a 5.x version appears, record its `content_hash` in this section; every subsequent evaluation
requires that exact digest. A digest change under the same version number flips G3 back to FALSE —
that is direct evidence the ABI is still moving, which is precisely what this gate exists to detect.

**Explicitly NOT a predicate: `[stability] level`.** The obvious machine-readable form of "ABI
declared stable" is vacuous, and was measured so before being excluded (V18): `origin/main_dst`'s
ABI 5.0 `ailang.toml` already carries `[stability] level = "stable"`, and the registry reports
`stability: stable` for the 2.2.0 line we are pinned to *and call unstable*. A gate on that field
passes immediately and falsely. Future readers: do not reach for it.

**Cadence and timebox.** Predicates are evaluated at most once per mission fire — the charter's
launchd `StartInterval=43200` (12h); no dedicated poller is invented. Budget: **28 evaluations
(~14 days), counted from the first fire after this revision lands.** Each evaluation ends exactly
one of three ways: all four TRUE → Phase 1 starts; any FALSE with budget remaining → idle
iteration (a correct outcome per the mission guardrail — the queue works other items meanwhile);
budget exhausted → the expiry action below. There is no fourth outcome; in particular, expiry does
**not** start Phase 1.

**Action on expiry (fail-closed).** Emit a structured BLOCKED result: a mission-log entry plus a
post to issue #663 (the mission's existing human-facing channel) recording each predicate's last
observed value, **escalated to Mark**. The mission queue continues with other backlog items; this
doc stays Planned but stops consuming iterations. Re-arming the gate for another 28-evaluation
window is a human decision by Mark, recorded in this section — never automatic.

**Declared residual — what the predicates do and do not cover.** G1–G4 prove: the PR is merged,
the packages are on `main`, a 5.x ABI is published at a fixed digest, and our pilot package
type-checks against it. They do **not** prove that Arni considers the ABI settled — "declared
stable" is a human judgement, and V18 shows the machine-readable field that appears to encode it
is vacuous. No predicate substitutes for his word, and over-claiming the gate would be worse than
naming the gap. Phase 1 starts on G1–G4 alone, with the "ABI changes again mid-port" risk row
still live; an explicit ack from Arni (PR comment or message), recorded here when it arrives, is
what retires that risk — not this gate.

**Phase 1 — Port 12 extensions to ABI 5.0.** For each of `ailang-docs`, `a2a`,
`decision-framework`, `context-mode`, `compaction-ai`, `compose`, `microrag`, `mcp`, `fmt`,
`exa-search`, `test-dummy`, `omnigraph`:
1. Repin `motoko_ext_abi` to 5.0.
2. Wrap the four effectful hook returns in their `Outcome` record, threading `next_state`.
3. Route any direct effect through `ctx.ports`; narrow the declared row to match.
4. `ailang publish` (server compiles stricter than `--dry-run` — trust the server).
5. Repin in `ailang.toml`, `ailang lock`, `ailang generate-extension-registry`.
6. `make check_core && make verify_extensions`.

Do `test-dummy` **first** as the pilot — it is the smallest and its hooks are trivial, so it
isolates ABI mechanics from behavioural complexity. Do `compaction-ai` **last** — but for
**behavioural complexity, not for its effect row**. ~~it is the one package whose `on_pre_step`
genuinely needs all ten effects via `ai_step`, per the ABI's own commentary.~~ **Struck 2026-08-12
(R2, V26).** That sentence cited, as authority, a passage upstream has since *retracted*:
`on_pre_step` is `! {AI, IO, Trace}` — **three** effects — and the ABI's own commentary says the
ten-effect reading "was taken as given … over-declared by SEVEN". The claim also contradicted this
doc's own ABI-break table, where `on_pre_step` is 9→3 and only `on_tool_handle` reaches ten
(9 + `Rand`). The ordering still holds; its stated reason did not.

**Phase 2 — Restore profiles and re-point the harness.** `main_dst` carries `ollama` but not
`cloud`, `ollama_docs`, `ollama_dp7`, `ollama_fmt`, or `ollama_microrag`. 14 of our 18
`motoko_profile:` entries reference the missing ones. Recreate them under `.motoko/config/`, then
verify each resolves. These encode our A/B design (one variable per arm) and are ours to maintain.

**Phase 3 — Settle the disposition residual and re-prove what remains (timeboxed).** The
disposition itself already exists: all **51** non-merge fork commits (V14) are classified in
[m-motoko-fork-disposition.md](m-motoko-fork-disposition.md) — 14 SUPERSEDED / 17 PORT / 14 DROP /
**6 UNRESOLVED** (R8 settled → PORT at iteration 4), each UNRESOLVED row naming the measurement that settles it. That ledger is the
working surface for this phase; its rows are not duplicated here. Classifications:

- **Superseded — drop.** Already confirmed: the compaction calibration, the elision ladder, and
  system-message pinning (see below); the empty-response retry (`087e68e`), which `main_dst`
  reimplements as `motoko-ext-empty-stop-guard` — a pure budgeted `on_solver_candidate` returning
  `ContinueWithFeedback`, cleaner than our loop-level retry.
- **Port — carry forward.** `motoko_ext_fmt` (V6), the eval profiles (V7), `MOTOKO_MAX_STEPS`
  (V21), AST autoread (V22), whitespace-tolerant `EditFile` (V23), the DP7 done-gate via
  `ailang ai-check` (V24). The last four were asserted without upstream verification in the first
  draft — the quorum caught it; each now has a Verification Log row establishing upstream absence
  with a same-scope control, and they stay on this list because they are substantiated, not
  because the draft said so.
- **Re-prove.** Anything whose value was measured on the old tree gets re-measured on the new one
  *before* we argue to keep it. `motoko_ext_fmt` is the headline case.

**Timebox.** Phase 3 gets a fixed budget of **28 mission fires (~14 days at the 12h interval),
counted from the first fire after Phase 2's success metric is green.** At expiry, every still-open
item — an unsettled UNRESOLVED row, or a re-proof not yet run — is emitted as an explicit blocker
(mission-log entry plus a #663 post naming the commit and its settling measurement, escalated to
Mark), and the phase **closes with named blockers** rather than remaining open. A promoted blocker
is a correct outcome; an open-ended phase is not.

### Compaction: what survives, what doesn't

Our [PR #97](https://github.com/arniwesth/motoko_agent/pull/97) had four legs. Verified against
`main_dst`:

| Leg | Disposition | Evidence |
|---|---|---|
| Calibrated estimate (anti-oscillation) | **Superseded, by a better mechanism** | His `affine_calibrate` is an affine fit with a fixed delta-density slope (1235‰), floored at raw. Anchor-size-*insensitive* where our ratio calibration was not — which was the actual bug. Pinned by the test `affine_stable_across_anchor_size`. |
| Tiered elision ladder | **Superseded and extended** | `motoko-ext-compaction-structural`, `keep_last` 10/5/3→1, plus `cap_oversized_tool_results` at 30% which we never had. **Corrected 2026-08-13**: "tiers 70/85/95" describes the *constants*, not the live selection. `compact_for_pre_step` selects the **gentlest tier whose RESULT fits under 70%** (`candidate_fits`/`result_target_pct`), and never reads `elide_hard_tier_pct()`/`emergency_pct()` at all — those are only read by the off-path `compact_step_with_limit`. Upstream's own `.agent/issues/ephemeral-compaction-and-ai-noop-thrash.md:55` proposed exactly this change ("select the structural tier by *result*, not by uncompacted pct") and it has landed. |
| System-message pinning | **Superseded, structurally stronger** | Ours pins at runtime (`159125e`). His makes it a type-level property: `PinnedSplit`, `CompactableSegment` as a newtype, `system_is_head_prefix` enforced at seed *and* resume, `seal_compacted_payload` able to require a non-empty prefix. |
| **75k output headroom reserve** | **Still open — but NOT where this row said, and the ask is one argument** | **RE-MEASURED 2026-08-13 (iteration 3) against `main_dst@6c06b08`; the previous wording is struck below.** The gap is real and it lives in the **phase core**, not in the compaction extension: `seal_compacted_payload` (`src/core/phase_vocab.ail:145-155`) is handed the **raw** `context_limit` by `session.ail:2561` and refuses on `usage_percent_with_limit(payload_msgs, limit) >= exhaustion_pct()` = **95** (`src/core/compaction.ail:30`). That predicate counts messages only — there is no output term anywhere between `policy.step.compaction.context_limit` (`session.ail:2530`) and the provider call. Rows V27–V29. |

~~The output-headroom case, stated precisely: on a 262144-token window his 95% emergency ceiling
admits ~249k of *input*, leaving ~13k against a 65536 output cap … Two mitigations exist upstream
that we should credit: `affine_calibrate` scales *up*, and `try_emergency_compaction_with_limit`
fails loudly with `compaction_exhausted` rather than sending.~~ **Struck 2026-08-13 (iteration 3).
The arithmetic is right, the two attributions are wrong, and the ask is smaller and better than
this paragraph made it look. Corrected below.**

**The output-headroom case, stated precisely (re-measured 2026-08-13, `main_dst@6c06b08`).** Three
numbers govern the live path and they are in three different files:

1. **The extension ladder targets 70%, not 95%.** `compact_for_pre_step`
   (`packages/motoko-ext-compaction-structural/compaction_structural.ail:170`) — the hook actually
   registered as `on_pre_step` (`register.ail:24`) — gates *every* branch on
   `candidate_fits` → `calibrated_ctx_usage(ctx, msgs) < result_target_pct()`, and
   `result_target_pct() = elide_tier_pct() = 70` (`:30`, `:16`). It picks the **gentlest tier whose
   result fits**, not a tier keyed off the uncompacted percentage. `emergency_pct() = 95` is
   referenced **only** at `:126`, `:130`, `:137` — all inside `compact_step_with_limit` /
   `try_emergency_compaction_with_limit`, which have **zero production callers** (V28). At 70% of a
   262144 window that is ≤183,500 input, leaving **≥78,644** — *more* than the 65,536 output cap
   and more than the 75k reserve `96542f8` adds. **When the ladder reaches its target, upstream is
   already better than our patch.**
2. **The ladder's last branch is unconditional.** When even `keep_last=1` cannot reach 70%,
   `compact_for_pre_step` returns `Compacted(floor, structural_note("floor", …))` rather than an
   error (`:191-193`). It does not refuse; it hands the over-target payload upward.
3. **The seal is the refusal, and it is input-only.** `seal_compacted_payload` then admits anything
   **< 95%** of the **raw** `context_limit` — ~249,036 input on a 262144 window, leaving **~13,108**
   against a 65,536 output cap. Over that, it is genuinely loud and terminal:
   `CompactionExhausted` ledger event, `code: "ContextExhausted"`, `retryable: false`,
   `c2_finalize(… TermCompactionExhausted …)` (`session.ail:2572-2578`).

So: **"hard-stop risk, not silent corruption" is CONFIRMED** — but by the phase core's seal, not by
`try_emergency_compaction_with_limit`, which nothing on the live path calls. And the residual is
**the band between the ladder's 70% aspiration and the seal's 95% permission**, which is exactly
where `docx_lambda` died (`finish=error` at step 84, input 263259 > 262144).

**Why this makes the upstream ask smaller and more acceptable.** We are not asking for a reserve
inside a compaction extension, and not asking anyone to adopt `OUTPUT_HEADROOM`/`75000`. We are
asking that the seal's threshold become output-aware — one argument at `session.ail:2561`
(`context_limit - reserved_output`) or one term inside `seal_compacted_payload`. One function, one
constant, in the phase core. Note the seal *already* demonstrates the pattern one line up: the
extension is handed `ext_context_limit = context_limit - pinned_tokens` (`session.ail:2534`), i.e.
upstream already subtracts a reserve before delegating — for the pinned system prefix. Ours is the
same move for the output budget.

Raised on PR #97 ([comment](https://github.com/arniwesth/motoko_agent/pull/97#issuecomment-5257958760)).
**As of 2026-08-13 that comment has no reply** — zero `arniwesth` events across issue comments,
review comments, reviews and the timeline, with a firing control (34 issues/PRs in that repo carry
his comments). See the charter's queue item 5 for the bounded decision rule that replaced
"if Arni's #97 reply invites it".

### Files to Modify/Create

- `ailang-packages/packages/motoko-ext-*/` (12 packages) — repin ABI 5.0, wrap hook returns in
  `Outcome`, route effects through `ctx.ports`. ~50–150 LOC each depending on how much direct IO
  the hooks do.
- `mk-ast/ailang.toml` — repin all 12 to republished registry versions; restore the registry-only
  policy if Arni's vendored `{path=...}` copies are reconciled.
- `mk-ast/.motoko/config/` — recreate `cloud`, `ollama_docs`, `ollama_dp7`, `ollama_fmt`,
  `ollama_microrag`.
- `internal/eval_harness/models.yml` — verify all 18 `motoko_profile:` entries resolve; no edits
  expected if Phase 2 restores the profile names verbatim.
- `MOTOKO.md` — rewrite §3 (packaging), §4 (profiles), §5 (a2a deferral retired), §7 (upstream
  delta). Roughly half the file describes a tree that will no longer exist.
- `design_docs/planned/m-motoko-compaction-quality.md` — update: its "Current architecture" table
  names `agent_loop_v2.ail` and `src/core/compaction.ail`, both of which move.

## Verification Log

Every claim above was checked against the actual trees, not inferred. This repo's docs have been
wrong before by asserting rather than checking (ground conclusions in data, not assumptions).

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | Canonical eval checkout is `mk-ast` @ `sunholo/eval-canonical` | `git worktree list`; read `~/go/bin/motoko` shim | Confirmed, HEAD `9e8d647` |
| V2 | 52 ours-only / 805 theirs-only commits | `git log --oneline origin/main_dst..HEAD` and reverse | Confirmed |
| V3 | ABI is 2.2.0 vs 5.0 | read both `ailang.toml` files | Confirmed |
| V4 | Four hook slots change type and row | `awk` the `ExtensionHooks` block from both `types.ail`, strip comments, diff | Confirmed — see table |
| V5 | 12 of our packages pin ABI 2.2.0 | `grep -l '"sunholo/motoko_ext_abi" = "2.2.0"' */ailang.toml` | Confirmed, 12 |
| V6 | `motoko_ext_fmt` absent from `main_dst` | `git grep -il 'motoko_ext_fmt\|ext-fmt' origin/main_dst` | Confirmed — **negative existence**, zero hits |
| V7 | 5 of 6 profiles absent | `git ls-tree -d origin/main_dst .motoko/config/` vs local `ls` | Confirmed: `ollama` survives; `cloud`, `ollama_docs`, `ollama_dp7`, `ollama_fmt`, `ollama_microrag` absent |
| V8 | 14 of 18 eval entries point at missing profiles | `grep -oE 'motoko_profile: "(cloud\|ollama_docs\|ollama_dp7\|ollama_fmt\|ollama_microrag)"' models.yml \| wc -l` | Confirmed 14 of 18 |
| V9 | No output-headroom reserve upstream | grep `headroom\|reserve\|max_tokens\|output_budget\|effective_window` over `src/core/*.ail` + compaction packages | **Negative existence** confirmed — `headroom` appears only in `dst_corpus.ail`. ⚠ **The row is true as measured and the CONCLUSION drawn from it was not** (see V27): a grep for *our* solution's vocabulary cannot see whether the other tree has the *property*. The trailing clause "structural compactor tiers off raw `ctx.context_limit`" is a **behavioural** claim no token grep can support, and it is wrong for the live path — the ladder gates on a 70% result target. Superseded by V27–V29 |
| V10 | System-message pinning is superseded | read `phase_vocab.ail` — `PinnedSplit`, `take_system_prefix`, `system_is_head_prefix`, `seal_compacted_payload` | Confirmed. **This corrects an earlier session claim that it was unverified.** |
| V11 | `empty_stop_guard` supersedes `087e68e` | read `packages/motoko-ext-empty-stop-guard/empty_stop_guard.ail` | Confirmed — budgeted `ContinueWithFeedback` on `on_solver_candidate` |
| V12 | `on_tool_handle` gains `Rand`, retiring the a2a deferral | V4 diff + `a2a@0.2.2` present in `main_dst` `[extensions]` | Confirmed |
| V13 | His vendored extension copies diverge from our published ones | compare `compaction_ai` `ailang.toml` version + file sizes | Confirmed: his "0.3.0" is 33,851 bytes; our published 0.3.2 is 9,454. Same name, lower version, different content |
| V14 | **The fork delta is 52 commits but only 51 are dispositionable** | `git rev-list --count origin/main_dst..HEAD` → **52**; `--no-merges` → **51**; `git log --merges` names the one, `ed61097` | Confirmed 2026-08-12 (iteration 1, re-measured first-party against `main_dst@303d869`). V2 counted correctly; the Success Criterion built on it was off-by-one *in kind*, asking for a row that cannot exist |
| V15 | That merge carries no unique content, so dropping it loses nothing | `git diff ed61097^2 ed61097` → **empty** (the merge is content-equivalent to its second parent); `git diff ed61097^1 ed61097` → 2 files, i.e. it is a real merge, not a no-op commit | Confirmed. Both arms measured: the second-parent diff being empty is the finding, the first-parent diff being non-empty is the control proving the instrument reads this merge at all |
| V16 | **Path-existence is NOT a usable supersession signal for this fork** — the headline files survive upstream as facades | for each path our 51 commits touch, `git cat-file -e origin/main_dst:<path>`: **80 of 94** rows come back `UPSTREAM_HAS`. But `agent_loop_v2.ail` is **4,005 B** upstream vs **95,868 B** ours, `compaction.ail` **5,804** vs **17,543**. Controls both fire: `.motoko/config/cloud/config.json` → GONE (matches V7), `README.md` → present | Confirmed — and it **retires an instrument before it was used**. A naive does-the-file-still-exist test would label ~85% of our commits "upstream still has this surface" and under-detect supersession almost completely. Disposition must be judged on **content**, not on path survival |
| V17 | #154 still OPEN on 2026-08-12 — the Phase 0 gate is genuinely closed, not already-passed | `gh pr view 154 --repo arniwesth/motoko_agent --json state,mergedAt,baseRefName` | `state=OPEN`, merged `-`, base `main`, updated 2026-08-11T19:37:56Z. **Control**: `gh pr list --repo arniwesth/motoko_agent --state merged --limit 3` → #152, #151, #150 all MERGED 2026-08-11 — the same instrument can see a MERGED state |
| V18 | **`[stability] level = "stable"` is VACUOUS as a Phase 0 predicate** | `git -C /Users/voightkampff/dev/arniwesth/motoko_agent show origin/main_dst:packages/motoko-ext-abi/ailang.toml \| grep -A1 '^\[stability\]'` (the 5.0 tree); `curl -s https://registry.ailang.sunholo.com/api/packages/sunholo/motoko_ext_abi \| jq -r .index.stability` (the 2.2.0 line we pin) | Both read `stable` — including the 2.2.0 line we are pinned to **and call unstable**. A gate on this field passes immediately and falsely; excluded from Phase 0 by design, and recorded here so a future reader does not reach for the obvious answer |
| V19 | Registry does NOT yet expose ABI 5.0 — a non-vacuous, discriminating predicate | `curl -s https://registry.ailang.sunholo.com/api/packages/sunholo/motoko_ext_abi \| jq -r '"latest=\(.index.latest) versions=\(.index.versions\|join(","))"'` | `latest=2.2.0 versions=1.0.0,2.0.0,2.1.0,2.2.0` — currently FALSE, flips TRUE only on Arni's republish. **Control**: the version list is non-empty (4 entries), so "5.0 absent" is a measurement, not an endpoint outage. The registry publishes a per-version `content_hash` (2.2.0 → `sha256:60d6ec4684d6bf80d1ec800efda7aa598ccbebcbf529dd6c760fd10c5c101956`), so a 5.x digest pins the same way |
| V20 | `packages/` does not exist on `origin/main` at all — #154 merging IS the vendored packages landing | `U=/Users/voightkampff/dev/arniwesth/motoko_agent; git -C "$U" show origin/main:packages/motoko-ext-abi/ailang.toml` | `fatal: path 'packages/motoko-ext-abi/ailang.toml' does not exist in 'origin/main'`. **Controls**: same path on `origin/main_dst` resolves and is 5.0; `origin/main:README.md` resolves (`rc=0`), so the ref itself is fine and only the path is absent. Upstream `main_dst` HEAD at measurement: `303d869` |
| V25 | **`origin/main` is AMBIGUOUS across this mission's three repos, and the ambiguous form of G2 is vacuously FALSE forever** — caught at iteration 2 when the controller ran the predicate table verbatim instead of reading it | the draft's `git cat-file -e origin/main:packages/…` with no `-C`, run from the mission checkout (`sunholo-data/ailang`) | `fatal: invalid object name 'origin/main'`, **rc=128** — this repo's default branch is `dev` and it has no `origin/main` (control: `git rev-parse --verify origin/dev` resolves). The wrong-repo error and the genuine path-absent answer **both return 128**, so an `exits 0` test cannot tell them apart, and G2 would have read FALSE forever — including after #154 merges. Fixed by pinning the repo with `-C` and pairing the predicate with a `README.md` control that proves the ref resolves. General form: **a predicate evaluated in a multi-repo mission must name its repo, and "it failed" is not "it is false"** |
| V21 | `MOTOKO_MAX_STEPS` env-var override absent upstream | grep `MOTOKO_MAX_STEPS` over `origin/main_dst` vs `mk-ast` | Upstream **0** files / mk-ast **1**. Nuance recorded: the generic token `max_steps` matches **70** upstream files — the upstream *concept* exists; what is ours is the ENV-VAR OVERRIDE, and this row claims only that. **Control (same scope)**: `MOTOKO_` → 612 upstream files — the instrument fires in that tree |
| V22 | AST autoread absent upstream | grep `autoread` and `auto_read` over both trees | `autoread`: upstream **0** / mk-ast **2** files; `auto_read`: 0 both sides. **Control (same scope)**: `agent_loop` → 116 upstream / 21 mk-ast — the instrument fires in both trees |
| V23 | Whitespace-tolerant `EditFile` absent upstream | grep the `ws_*` helper names over both trees | Upstream **0**; ours live at mk-ast `src/core/tool_runtime.ail:847` (`apply_edit_ws_tolerant`), test at `:860`. **Control (same scope)**: `src` matches 819 upstream files; `EditFile` → 51 upstream / 8 mk-ast. **Instrument warning**: the bare word `whitespace` matches 26 upstream files — prose and unrelated trimming, the WRONG instrument; use the helper names |
| V24 | DP7 done-gate (`ailang ai-check`) has no upstream *implementation* | grep `done_gate` and `ailang ai-check` over both trees, then classify the hits | `done_gate`: 0 both sides. `ailang ai-check`: **10** upstream files — ALL TEN are `.agent/` prose (plans, PR write-ups), not implementation; mk-ast **23**. **Control (same scope)** fires as in V21–V23. Scoping recorded deliberately: an unscoped grep reads as "already upstream" and is wrong — this repo's own disposition work retired token-matching as a behavioural-equivalence test for exactly this reason (see the mission log's R16/R34) |

| V26 | **`on_pre_step` takes THREE effects, not ten — and the "ten" the first draft cited is a reading upstream has explicitly RETRACTED** | `U=/Users/voightkampff/dev/arniwesth/motoko_agent; git -C "$U" show origin/main_dst:packages/motoko-ext-abi/types.ail` — read the `ExtensionHooks` signature and the commentary above it | Signature: `on_pre_step: (ExtCtx, [Msg]) -> PreStepOutcome ! {AI, IO, Trace}` — **3**. The commentary: *"WI-D8 NARROWED THIS ROW FROM TEN EFFECTS TO THREE, AND NOTHING HAD EVER MEASURED IT … WI-D7's central conclusion about `on_pre_step` rested on that row being what the port performs — 'whose own port row is exactly those ten'. It was taken as given. **It was over-declared by SEVEN.**"* **Controls (same scope)**: the file resolves at **971** lines; `on_tool_handle` in the same record reads `! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream, Rand}` — ten — so the instrument can see a ten-effect row where one exists. Raised by `gemini-3-1-pro` at R2 against this doc's own ABI-break table (9→3 for `on_pre_step`; only `on_tool_handle` reaches ten); measured first-party rather than forwarded, and the measurement made it **worse** than filed — not merely inconsistent with V4, but sourced from a passage whose own author records it as an unmeasured assumption |

| V27 | **The live compaction ladder targets 70%, not the 95% this doc's compaction section asserted — and V9's grep could never have seen that** | `U=/Users/voightkampff/dev/arniwesth/motoko_agent`; `git -C "$U" show 6c06b08:packages/motoko-ext-compaction-structural/register.ail` (which func is bound to `on_pre_step`), then `… \| grep -n 'emergency_pct()'` and `… \| grep -n 'result_target_pct()'` over `compaction_structural.ail` | `register.ail:24` binds **`compact_for_pre_step`**. In that function every branch gates on `candidate_fits` → `< result_target_pct()`, and `result_target_pct() = elide_tier_pct() = 70` (`:30`, `:16`). `emergency_pct() = 95` occurs at exactly **3** sites — `:126`, `:130`, `:137` — **all** inside `try_emergency_compaction_with_limit` / `compact_step_with_limit`. **Control (same scope, same call)**: the `result_target_pct()` grep returns 2 hits (`:30`, `:167`) in the same file, so the instrument fires on both patterns. **Arithmetic** (262144 window, 65536 output cap): 95% ⇒ 249,036 in / 13,108 left (**deficit**); 70% ⇒ ≤183,500 in / **≥78,644** left (**exceeds both the output cap and our own 75k reserve**). **The generalisable error in V9**: a negative-existence grep for the vocabulary of *our* fix (`headroom`, `reserve`, `output_budget`) establishes only that upstream does not spell it our way — never that it lacks the property. Upstream's protection is spelled `result_target_pct`, a word no such grep contains |
| V28 | **`try_emergency_compaction_with_limit` — the mitigation this doc credited — has ZERO production callers** | `git -C "$U" grep -n 'compact_step_with_limit' 6c06b08 -- '*.ail'` and the same for `compact_for_pre_step`, then classify every importer | Non-test importers of `compact_step_with_limit`: **none**. All importers are `scripts/smoke_catalog_compaction.ail`, `scripts/smoke_v2_compaction_tiers.ail`, `src/core/test/integration_tests.ail`, plus its own inline unit test. **Control (same scope)**: `compact_for_pre_step` is imported by `register.ail:17` — the live registration — so the instrument distinguishes a live importer from a test one. Corroborated by upstream's own ADR-002 (`.agent/projects/001_DST/…:153`): *"Target the **live** compactor (`compact_for_pre_step`), not the off-path `compact_step_with_limit`"*. ⚠ **That same ADR (`:344`) calls it "the only `compaction_exhausted` `Err`", which is FALSE at `6c06b08`** — `src/core/phase_vocab.ail:152` emits one on the live path. Upstream's doc is stale and this doc had inherited its claim; see V29 |
| V29 | **The live refusal exists and is loud — it is the phase core's seal, and its threshold is input-only** | `git -C "$U" show 6c06b08:src/core/phase_vocab.ail \| sed -n '145,155p'`; `git -C "$U" grep -n 'exhaustion_pct' 6c06b08 -- '*.ail'`; `git -C "$U" show 6c06b08:src/core/session.ail \| sed -n '2530,2580p'` | `seal_compacted_payload` (`phase_vocab.ail:145`) tests `usage_percent_with_limit(payload_msgs, limit) >= exhaustion_pct()`, and `exhaustion_pct() = 95` (`src/core/compaction.ail:30`). It is called on the live path at `session.ail:2561` with the **raw** `context_limit` (`:2530`), and its `Err(SealExhausted …)` arm is terminal and loud: `CompactionExhausted` ledger event, `code: "ContextExhausted"`, `retryable: false`, `TermCompactionExhausted` (`:2572-2578`). So **"hard-stop, not silent corruption" is CONFIRMED** — at a different function than this doc named. The predicate counts messages only; there is no output term between `policy.step.compaction.context_limit` and the provider call. **Control, and the reason the ask is one argument**: one line up, `session.ail:2534` hands the extension `ext_context_limit = context_limit - pinned_tokens` — upstream *already* subtracts a reserve before delegating, for the pinned system prefix. **Self-correction recorded**: this controller's first reading was "the live path has no refusal at all", from the unconditional `floor` branch at `compaction_structural.ail:191`. That was FALSE, and only chasing the second `compaction_exhausted` site instead of trusting the ADR's "the only" caught it |

Rows V17–V26 were measured first-party by mission-control on 2026-08-12 (iteration 2), commands
and outputs as recorded; they answer the 2026-08-12 quorum block (see the Quorum revision log below).
Rows **V27–V29** were measured first-party on 2026-08-13 (iteration 3) against
`arniwesth/motoko_agent@6c06b08` (`origin/main_dst`, which advanced from the `303d869` V14/V20 were
taken at). They correct this doc's compaction section — see the struck paragraph above.

**Not verified — carried as open questions, not premises:**
- Whether `motoko-ext-progress-contract-guard` supersedes any of our commits (it has no obvious
  counterpart in our 52; likely new work of his).
- ~~Whether the remaining ~40 of our 52 commits have upstream counterparts.~~ Superseded since the
  first draft: the four Port-list features the quorum flagged now have rows (V21–V24), and the
  full 51-commit disposition lives in [m-motoko-fork-disposition.md](m-motoko-fork-disposition.md),
  whose **6 UNRESOLVED** rows — each naming its settling measurement — are the remaining open
  questions. Phase 3 settles them under its timebox.

### Quorum revision log

**2026-08-12 — BLOCKED by `ailang design-quorum`** (artifact:
`.ailang/state/mission-quorum/m-motoko-dst-refactor-migration-2026-08-12T08-41-12Z.json`, both
reviewers present, `absent_reviewers` empty, $0.064). Neither reviewer disputed the design
direction (adopt the phase-core/DST refactor, re-prove our improvements); both objections were
measured first-party by mission-control (iteration 2) before this revision, which answers them:

- **gpt5-6-sol (reject)** — *"Phase 0 is an explicitly unbounded wait on external events … neither
  a deadline nor a machine-verifiable stability condition."* **Upheld.** Phase 0 is rewritten as a
  bounded, fail-closed gate: four conjunctive predicates G1–G4, each with its evaluating command
  and current observed value (V17, V19, V20, plus the compile probe), a 28-evaluation timebox at
  the mission's 12h fire interval, a structured BLOCKED expiry escalating to Mark, and a declared
  residual naming what the predicates cannot prove. One part of the proposed fix did **not**
  survive contact with the data: a machine check on "ABI declared stable" is vacuous (V18) — the
  discriminating replacement is "registry exposes 5.x at a pinned digest" (V19). The objection's
  second sentence (open-ended Phase 3) is answered with a fixed Phase 3 timebox that promotes
  unresolved commits to named blockers.
- **gemini-3-1-pro (reject)** — *the four "Port — carry forward" features were asserted without
  verifying their absence in `main_dst`.* **Procedurally upheld** — the premise-verification rule
  was violated; **substantively refuted 4/4** — all four features really are absent upstream.
  Resolved via the first branch of the reviewer's proposed fix: Verification Log rows V21–V24
  added, each with a same-scope known-positive control, and the Port list now cites them. The
  features remain on the list because they are now substantiated, not deferred to Phase 3.

**2026-08-12 R2 — BLOCKED again, on NEW objections** (artifact
`…m-motoko-dst-refactor-migration-2026-08-12T09-50-35Z.json`, both reviewers present,
`absent_reviewers` empty, $0.096). R1's two objections are **not** re-raised — the bounded gate and
the V21–V24 rows answered them. Both new objections are internal-consistency defects in the R1
revision itself, and both were measured first-party before any action:

- **`gemini-3-1-pro` (reject)** — two parts, **both UPHELD, both fixed in this R2 pass**.
  (i) The Design Freeze said "check off before *sprint-executor* starts", creating a **deadlock**:
  Phase 0 is what the executor runs to evaluate G1–G4. Reworded to "before **Phase 1** starts" —
  the reviewer's verbatim fix. (ii) The Phase 1 ordering rationale asserted `compaction-ai`'s
  `on_pre_step` "genuinely needs all ten effects". **Refuted, and worse than filed** (V26): the row
  is `! {AI, IO, Trace}` — three — and the ABI commentary the sentence cited as authority is one
  upstream *retracted* ("over-declared by SEVEN"). Struck; the ordering keeps a different reason.
- **`gpt5-6-sol` (reject) — UPHELD and PARKED for Mark, not resolved in-loop.** The gate authorises
  Phase 1 on G1–G4 while the doc's own Dependencies + Design Freeze also demand Arni's confirmation
  and the reconciliation. It is right that the doc contradicts itself — and the contradiction reaches
  **outside** the doc: the *ratified charter guardrail* requires "#154 merged **AND** Arni has
  declared the ABI stable", which the R1 *Declared residual* silently relaxed to G1–G4. Its fix
  offers two branches (add G5/G6/G7 vs. drop the human condition and have the decision owner accept
  the risk). **Choosing is Mark's call, not the controller's** — and branch A's G6 would make
  queue item 11 (itself Phase-0 gated) a Phase-0 predicate, a dependency loop worth his eyes. The
  loop's one-revision-one-requorum budget is spent, so this doc is **`needs-human-review`** and
  **Phase 1 must not start** until he answers. Nothing is lost by waiting: Phase 0 is measurably
  CLOSED today (G1 FALSE, G2 FALSE, G3 FALSE — V17/V19/V20/V25), so no sprint was available to
  block.

## Axiom Compliance

Scored for the *adoption*, since that is the decision this doc makes. This is harness work, so
several axioms are scored on what the architecture does for our ability to reason about the agent.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The entire point: pure `decide`, virtual clock, seeded generation, exact-program strict replay |
| A2: Replayability | +1 | Append-only event ledger doubles as the DST trace; `dst_replay.ail` replays exact programs |
| A3: Effect Legibility | +1 | `ExtPorts` makes every extension effect explicit and injectable; hook rows narrowed to what is actually performed |
| A4: Explicit Authority | +1 | Ports are the only authority path; `buildChildEnv()` replaces inline env setup with an explicit allowlist |
| A5: Bounded Verification | +1 | Invariants asserted over a `LedgerTrace`, never over model prose — locally checkable |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Deterministic replay is a machine-analysis affordance; the DST corpus is machine-generated |
| A8: Minimal Syntax | 0 | No AILANG syntax change |
| A9: Cost Visibility | +1 | `cost_phase.ail` isolates cost; DST replaces rig hours with deterministic runs |
| A10: Composability | 0 | ABI break is a one-time cost against a cleaner long-run composition story |
| A11: Structured Failure | +1 | Typed exit codes reach the caller; `compaction_exhausted` fails loudly rather than sending |
| A12: System Boundary | +1 | Harness→runtime boundary is now a testable `buildChildEnv()` function |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): improves it — no implicit nondeterminism introduced
- [x] A3 (Effects): improves it — ports make effects explicit
- [x] A4 (Authority): improves it — no ambient access; explicit allowlist
- [x] A7 (Machines First): improves it

## Success Criteria

- [ ] `make check_core && make verify_extensions` green on the migrated branch
- [ ] All 12 extensions published at ABI 5.0 and repinned from the registry
- [ ] All 18 `motoko_profile:` entries resolve
- [x] All **51 non-merge** fork commits dispositioned, in
      [m-motoko-fork-disposition.md](m-motoko-fork-disposition.md) (2026-08-12, iteration 1):
      **14 SUPERSEDED / 16 PORT / 14 DROP / 7 UNRESOLVED** as first landed; **14/17/14/6** after R8
      settled at iteration 4. The range holds **52** commits; exactly
      one (`ed61097`) is a merge carrying no unique content, so 51 is the number of rows this
      criterion can ever have — see V14/V15. Split into its own file rather than inlined here: it
      is 114 lines of evidence and this doc is the *decision*, not the ledger. **The 6 remaining UNRESOLVED
      rows are the honest residual and each names the measurement that settles it** — Phase 3 is
      not complete until they are settled, and a forced verdict there would have been worse than
      the gap.
- [ ] `motoko_ext_fmt` re-measured on the new tree; kept or dropped **on evidence**
- [ ] Output-headroom question resolved upstream or carried as a named patch
- [ ] [MOTOKO.md](../../MOTOKO.md) rewritten to describe the new tree
- [ ] ~~One historical A/B question answered via DST rather than a rig run~~ **WITHDRAWN** — this is
      the metric Goals §5 already retracts as a DST over-read. It survived here as a checkbox after
      the retraction landed in Goals, i.e. this criterion could never be met by design. Removed
      rather than reworded: the honest replacement is Goals §5's contract-instrument metric, which
      is the line above it.

## Testing Strategy

The migration's own test is `make check_core && make verify_extensions`, per package and then
tree-wide. Beyond that, Phase 3 should use his instruments rather than inventing ours:

- `make declared_vs_performed` — asserts hook rows match measured behaviour. This is the gate that
  will catch a lazily-widened row during our port.
- `make conformance` — the conformance kit (`motoko_ext_conformance`) over our ported extensions.
- `make corpus_pr` — the blocking fixed-seed corpus, as a regression gate.
- A DST scenario reproducing the `docx_lambda` overflow, if we carry the output-headroom patch.
  A deterministic reproduction is worth more than the original rig observation.

## Non-Goals

- Rewriting our extensions to *use* the DST framework beyond what conformance requires. Adopt
  first, exploit later.
- Upstreaming our eval profiles. They encode our A/B design; there is no reason Arni should carry them.
- Re-litigating the compaction mechanism. Three legs are superseded; we take his.
- Fixing the registry-vs-vendored divergence unilaterally. That needs Arni's agreement (see Risks).

## Deferred Decisions

Latitude granted to the implementing agent:

- Port order within Phase 1, except: `test-dummy` first, `compaction-ai` last.
- Whether to port `motoko_ext_a2a` at all now that `Rand` is available — nothing depends on it today.
- Exact profile file layout, as long as the five names resolve.
- Whether the output-headroom fix lands as an upstream PR or a carried patch. ~~Decide on Arni's
  response to the #97 comment.~~ **Struck 2026-08-13 (iteration 3): that is an unbounded wait on a
  third party, the same defect `gpt5-6-sol` blocked Phase 0 for at iteration 1.** Bounded rule, and
  it is the charter's queue item 5: the #97 comment is the ask; if there is no `arniwesth` response
  on #97 by **2026-08-27** (14 days from the re-measurement), file the corrected case as a standalone
  upstream **issue** citing V27–V29, and carry the patch locally against `main_dst` regardless. A
  reply may still redirect it to a PR at any point; what expires is the *waiting*, not the offer.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| ABI changes again mid-port | High — port twice | Phase 0 gate. Arni explicitly says the ABI "is still subject to change and the current version number is somewhat arbitrary" |
| **Registry/vendored version collision** | **High** — his `compaction_ai` "0.3.0" is different, larger code than our published 0.3.2. A naive `ailang lock` could resolve to either | Reconcile with Arni before Phase 1. `main_dst` moved every dep back to `{path=...}` — the exact drift [MOTOKO.md](../../MOTOKO.md) §3 warns about. Do not paper over it locally |
| Effect-row narrowing is not mechanical | Medium — Phase 1 slips | Pilot on `test-dummy` first to separate ABI mechanics from behavioural rewrites; `compaction-ai` last |
| Our wins don't reproduce on the new tree | Medium | That is the *point* of Phase 3 — "if proven". Drop with evidence rather than carrying dead weight |
| Eval baselines break across the migration | Medium | Migration is a baseline discontinuity by definition. Bank a pre- and post-migration baseline on the same benchmark set; do not compare across it (the extension-fix baseline shift (rig memory)) |
| We lose local work to a git operation | High | Fork work lives in worktrees of one clone. `git status` first, never `checkout`/`reset --hard` with uncommitted changes ([CLAUDE.md](../../CLAUDE.md) §0) |

## Related Documents

- [[m-motoko-self-improvement-loop]] — the parent initiative; this migration is the tree it runs on
- [[m-motoko-compaction-quality]] — summary *content* quality; distinct from this doc's *mechanism*
  work, but its architecture table needs updating (both files it names move)
- [m-motoko-v021-effect-row-migration](../implemented/v0_22_0/m-motoko-v021-effect-row-migration.md) —
  **the direct precedent.** Same class of work (effect-row migration across extensions + core) at
  smaller scale. Read it before Phase 1; its failure modes will recur
- [m-motoko-ailang-reconcile](../implemented/v0_22_0/m-motoko-ailang-reconcile.md) — the parent of that
  migration; the 12-extension bump + republish cycle is the same shape as Phase 1
- [MOTOKO.md](../../MOTOKO.md) — the checkout map; roughly half becomes false on merge

## References

- [PR #154 — main dst](https://github.com/arniwesth/motoko_agent/pull/154): +217,701/−3,061 across
  892 files, 773 commits, 65 merged PRs (#79–#152). ~120k insertions are `.agent/` record; ~60k is
  runtime and test code
- [PR #97 — our compaction fix](https://github.com/arniwesth/motoko_agent/pull/97) and the
  [disposition comment](https://github.com/arniwesth/motoko_agent/pull/97#issuecomment-5257958760)
- PRs [#96](https://github.com/arniwesth/motoko_agent/pull/96) /
  [#98](https://github.com/arniwesth/motoko_agent/pull/98) — closed by Arni as superseded by #154;
  correct, since `main_dst` requires `ailang >= 0.33.0` and the v0.30 images fix is moot there
- `main_dst` ADRs: `.agent/projects/004_phase_core_refactor/ADR-001` (phase-oriented core),
  `.agent/projects/009_motoko_dst_execution/ADR-001` (DST, Accepted 2026-08-02)

## Future Work

- **Extension-level DST is an OPEN RESEARCH PROBLEM, and worth watching rather than assuming.**
  Arni: *"Doing proper DST of extensions turned out to be exquisitely complex. That is basically an
  open research project."* If it is ever solved, the economics of our whole self-improvement loop
  change — a paired A/B currently costs 7–14h of rig time. Until then, extension A/Bs stay rig
  questions and should be priced as such.
- **Propose `motoko_ext_fmt` upstream** once re-measured, with the token evidence attached.
- **Revisit `motoko_ext_a2a`** now that `Rand` is in the `on_tool_handle` row.
- **Reconcile the registry as single source of truth** with Arni, so vendored `{path=...}` copies
  stop diverging from published versions.
