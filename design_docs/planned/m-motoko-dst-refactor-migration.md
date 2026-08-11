# M-MOTOKO-DST-MIGRATION: Adopt Arni's phase-core/DST refactor and re-prove our fork's improvements

**Status**: Planned (2026-08-11)
**Target**: rolling — operational; no AILANG core changes
**Priority**: P0 — once [#154](https://github.com/arniwesth/motoko_agent/pull/154) lands on `main`, our fork is 805 commits behind a tree whose ABI it cannot build against. Every motoko eval depends on closing this.
**Estimated**: 3 phases; Phase 1 is ~12 mechanical package ports, Phase 3 is the open-ended one
**Owner**: sunholo (we co-develop motoko with @arniwesth)
**Dependencies**: #154 merged to `origin/main`; Arni's extension republish (he states the ABI "is still subject to change" — Phase 1 must not start before it settles)

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
   of rig time (see the extension-fix baseline shift (rig memory)). The DST framework —
   seeded generation, virtual clock, exact-program strict replay, invariants asserted over a
   `LedgerTrace` rather than over model prose — is a much cheaper instrument for exactly the
   questions we keep spending rig hours on.

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
5. At least one of our historical A/B questions is answered via DST instead of a rig run,
   establishing the cheaper instrument works.

## High-Impact Decisions

| Decision | Options | Recommendation | Who decides | Cost to change later |
|---|---|---|---|---|
| Adopt or fork permanently | adopt `main_dst` / stay on ours | **Adopt** (ratified by Mark 2026-08-11) | Mark — **decided** | Very high |
| Extension source of truth | our registry / his vendored `{path=...}` | **Registry**, and reconcile his vendored copies back to it | Mark + Arni | High — this is the drift that cost an audit once ([MOTOKO.md](../../MOTOKO.md) §3) |
| When to start Phase 1 | now / after ABI settles | **After** — he says the ABI is "still subject to change" | us | Medium — starting early means porting twice |
| `motoko_ext_fmt` disposition | port / upstream / drop | **Port first, propose upstream once re-measured** | us | Low |
| Profiles | restore ours / adopt his | **Restore ours** — they encode our A/B design, not his | us | Low |

### Design Freeze

Check off before sprint-executor starts:

- [ ] #154 merged to `origin/main`
- [ ] Arni's extension republish landed and ABI 5.0 declared stable
- [ ] Registry-vs-vendored reconciliation agreed with Arni (see Risks)
- [ ] Phase 1 port target confirmed as ABI 5.0 (not a later 6.x)

## Solution Design

### Overview

Three sequential phases behind a gate. Phase 0 is a wait, not work. Phase 1 is mechanical and
sizeable. Phase 3 is the one with genuine unknowns, and is deliberately last so it runs on a green
tree.

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

**Phase 0 — Gate (no work, blocking).** Wait for #154 on `main` and the extension republish.
Track via `pr-monitor`. Starting Phase 1 against a moving ABI means porting twice.

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
isolates ABI mechanics from behavioural complexity. Do `compaction-ai` **last**: it is the one
package whose `on_pre_step` genuinely needs all ten effects via `ai_step`, per the ABI's own
commentary.

**Phase 2 — Restore profiles and re-point the harness.** `main_dst` carries `ollama` but not
`cloud`, `ollama_docs`, `ollama_dp7`, `ollama_fmt`, or `ollama_microrag`. 14 of our 18
`motoko_profile:` entries reference the missing ones. Recreate them under `.motoko/config/`, then
verify each resolves. These encode our A/B design (one variable per arm) and are ours to maintain.

**Phase 3 — Disposition all 52 commits and re-prove what remains.** Classify each:

- **Superseded — drop.** Already confirmed: the compaction calibration, the elision ladder, and
  system-message pinning (see below); the empty-response retry (`087e68e`), which `main_dst`
  reimplements as `motoko-ext-empty-stop-guard` — a pure budgeted `on_solver_candidate` returning
  `ContinueWithFeedback`, cleaner than our loop-level retry.
- **Port — carry forward.** `motoko_ext_fmt`, the eval profiles, `MOTOKO_MAX_STEPS`, AST autoread,
  whitespace-tolerant `EditFile`, the DP7 done-gate via `ailang ai-check`.
- **Re-prove.** Anything whose value was measured on the old tree gets re-measured on the new one
  *before* we argue to keep it. `motoko_ext_fmt` is the headline case.

### Compaction: what survives, what doesn't

Our [PR #97](https://github.com/arniwesth/motoko_agent/pull/97) had four legs. Verified against
`main_dst`:

| Leg | Disposition | Evidence |
|---|---|---|
| Calibrated estimate (anti-oscillation) | **Superseded, by a better mechanism** | His `affine_calibrate` is an affine fit with a fixed delta-density slope (1235‰), floored at raw. Anchor-size-*insensitive* where our ratio calibration was not — which was the actual bug. Pinned by the test `affine_stable_across_anchor_size`. |
| Tiered elision ladder | **Superseded and extended** | `motoko-ext-compaction-structural`, tiers 70/85/95, `keep_last` 10/5/3→1, plus `cap_oversized_tool_results` at 30% which we never had. |
| System-message pinning | **Superseded, structurally stronger** | Ours pins at runtime (`159125e`). His makes it a type-level property: `PinnedSplit`, `CompactableSegment` as a newtype, `system_is_head_prefix` enforced at seed *and* resume, `seal_compacted_payload` able to require a non-empty prefix. |
| **75k output headroom reserve** | **Still open — the one thing to carry** | No output reserve in his compaction path. Tiers run off raw `ctx.context_limit`. |

The output-headroom case, stated precisely: on a 262144-token window his 95% emergency ceiling
admits ~249k of *input*, leaving ~13k against a 65536 output cap — and input and output share the
window. This is the exact failure `96542f8` was written for (`docx_lambda`, `finish=error` at step
84, input 263259 > 262144). Two mitigations exist upstream that we should credit: `affine_calibrate`
scales *up* (implicit but unbounded margin), and `try_emergency_compaction_with_limit` fails loudly
with `compaction_exhausted` rather than sending. So this is a **hard-stop** risk, not silent
corruption — but it is an output-side concern that input-side calibration does not address.

Raised on PR #97 ([comment](https://github.com/arniwesth/motoko_agent/pull/97#issuecomment-5257958760)).
If Arni agrees, this becomes a small upstream PR against the new architecture rather than a
carried patch.

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
| V9 | No output-headroom reserve upstream | grep `headroom\|reserve\|max_tokens\|output_budget\|effective_window` over `src/core/*.ail` + compaction packages | **Negative existence** confirmed — `headroom` appears only in `dst_corpus.ail`; structural compactor tiers off raw `ctx.context_limit` |
| V10 | System-message pinning is superseded | read `phase_vocab.ail` — `PinnedSplit`, `take_system_prefix`, `system_is_head_prefix`, `seal_compacted_payload` | Confirmed. **This corrects an earlier session claim that it was unverified.** |
| V11 | `empty_stop_guard` supersedes `087e68e` | read `packages/motoko-ext-empty-stop-guard/empty_stop_guard.ail` | Confirmed — budgeted `ContinueWithFeedback` on `on_solver_candidate` |
| V12 | `on_tool_handle` gains `Rand`, retiring the a2a deferral | V4 diff + `a2a@0.2.2` present in `main_dst` `[extensions]` | Confirmed |
| V13 | His vendored extension copies diverge from our published ones | compare `compaction_ai` `ailang.toml` version + file sizes | Confirmed: his "0.3.0" is 33,851 bytes; our published 0.3.2 is 9,454. Same name, lower version, different content |

**Not verified — carried as open questions, not premises:**
- Whether `motoko-ext-progress-contract-guard` supersedes any of our commits (it has no obvious
  counterpart in our 52; likely new work of his).
- Whether the remaining ~40 of our 52 commits have upstream counterparts. Phase 3 dispositions
  these one at a time; this doc does not assume.

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
- [ ] All 52 fork commits dispositioned in a table in this doc (superseded / ported / dropped)
- [ ] `motoko_ext_fmt` re-measured on the new tree; kept or dropped **on evidence**
- [ ] Output-headroom question resolved upstream or carried as a named patch
- [ ] [MOTOKO.md](../../MOTOKO.md) rewritten to describe the new tree
- [ ] One historical A/B question answered via DST rather than a rig run

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
- Whether the output-headroom fix lands as an upstream PR or a carried patch — decide on Arni's
  response to the #97 comment.

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

- **Use DST as our primary A/B instrument.** If a seeded DST run can answer questions that
  currently cost 7–14h of rig time, that changes the economics of the whole self-improvement loop
  — arguably a bigger win than the refactor itself.
- **Propose `motoko_ext_fmt` upstream** once re-measured, with the token evidence attached.
- **Revisit `motoko_ext_a2a`** now that `Rand` is in the `on_tool_handle` row.
- **Reconcile the registry as single source of truth** with Arni, so vendored `{path=...}` copies
  stop diverging from published versions.
