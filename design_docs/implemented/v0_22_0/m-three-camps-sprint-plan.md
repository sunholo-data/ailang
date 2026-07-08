# M-THREE-CAMPS Sprint Plan (Gap-Analysis-First)

**Design doc**: [m-three-camps-language-survey.md](m-three-camps-language-survey.md)
**Sprint ID**: M-THREE-CAMPS
**Start**: 2026-05-20 (Wed)
**Hard deadline**: 2026-05-25 (Mon) — talk
**Duration**: 5 days
**Total estimate**: ~38h (~7.5h/day)
**Risk level**: Medium

---

## Sprint Goal

Ship a publicly-visible "three camps" comparison page **and** a gap-driven expansion of the eval suite that probes each camp's hypothesis empirically. Talk story: not "AILANG vs the field" but "here's the empirical map of what each camp's claims look like on a shared benchmark grid, plus an honest self-audit of where AILANG falls short."

## Scope: This Sprint vs Post-Talk

**In scope this sprint** (Phase 1 + Phase 2 + partial Phase 3):
- Public comparison page
- 14 gap benchmarks added and runnable
- AILANG self-audit on all 14 gaps
- MoonBit wired up as primary peer (apples-to-apples FP comparison)
- Vera + Aver wired up on contract tier (verification-camp head-to-head) — toolchain-gated

**Out of scope** (Phase 4, post-talk):
- Audit memos with build/eval/shelve decisions for borrowable ideas
- Zero, Pact, and remaining 10 peer languages
- Building any new AILANG features surfaced by gap analysis (file follow-up issues)

---

## Milestones

### M1: Comparison page draft (Day 1, Wed — ~5h)

Build `docs/docs/guides/three-camps-comparison.md` from the Camp Matrix + Gap Analysis sections of the design doc. Public-audience tone.

- **LOC**: ~400 of markdown
- **Files**:
  - `docs/docs/guides/three-camps-comparison.md` (new)
  - `docs/sidebars.ts` (nav entry)
- **Acceptance**:
  - [ ] Page renders via `cd docs && npm start`
  - [ ] All 16 peer languages have a row in their camp matrix
  - [ ] AILANG's placement clearly explained (verification full member; orchestration strong; syntactic non-bet)
  - [ ] Cites Negroni post
  - [ ] Linked from sidebar under Guides
- **Risk**: low

### M2: Gap benchmarks — Syntactic + Verification camps (Day 1–2, Wed–Thu — ~8h)

Author 7 of the 14 gap benchmarks: `ast_patch_roundtrip`, `dense_operator_program`, `explicit_dataflow_ssa`, `shadowing_heavy_contract`, `decision_block_capture`, `intent_annotated_solver`, `canonical_convergence`.

- **LOC**: 7 × ~50 LOC YAML = ~350 LOC
- **Files**: `benchmarks/<name>.yml` × 7
- **Acceptance**:
  - [ ] All 7 YAML files exist with `id`, `description`, `languages: ["ailang"]`, `task_prompt`, `expected_stdout`, `tier`, `tags`
  - [ ] Each runs without harness error under `ailang eval --lang ailang --benchmark <name>` (pass/fail TBD by M4)
  - [ ] Each cites the source language and the hypothesis it tests
- **Risk**: low — well-scoped per benchmark

### M3: Gap benchmarks — Orchestration + AILANG-strength (Day 2, Thu — ~7h)

Author the remaining 7 gap benchmarks: `multi_agent_handoff`, `typed_stream_pipeline`, `parallel_independent_subtasks`, `audit_chain_replay`, `ai_effect_summarize`, `ai_effect_json_schema`, `unauthorized_fs_refused`, `parallel_map_reduce`. (Note: 8 listed but 1 is alternative — pick 7 that surface clearest signal.)

- **LOC**: 7 × ~50 LOC YAML = ~350 LOC
- **Files**: `benchmarks/<name>.yml` × 7
- **Acceptance**:
  - [ ] All 7 YAML files exist (same schema as M2)
  - [ ] `ai_effect_*` benchmarks gated behind a `--with-ai-effect` flag so smoke runs don't blow real API costs
  - [ ] Each runs without harness error under `ailang eval --lang ailang --benchmark <name>`
- **Risk**: medium — `audit_chain_replay` and `multi_agent_handoff` may need harness additions

### M4: AILANG self-audit + report (Day 3, Fri — ~6h)

Run AILANG against all 14 gap benchmarks; document results honestly. This is the talk centerpiece.

- **LOC**: ~250 of markdown report + ~50 of CSV/JSON results
- **Files**:
  - `docs/docs/guides/three-camps-self-audit.md` (new)
  - `eval_results/three-camps-self-audit/` (raw data)
- **Acceptance**:
  - [ ] Each of the 14 benchmarks has a result: pass / fail / harness-incompatible
  - [ ] For each failure, a one-paragraph diagnosis (what does AILANG need to do better?)
  - [ ] Page embedded in comparison page sidebar / cross-linked
  - [ ] Identifies at least 3 follow-up issues (filed as GitHub issues or noted in audit memos)
- **Risk**: medium — failures are expected and load-bearing, not a problem

### M5: MoonBit bring-up (Day 3 PM – Day 4 AM, Fri–Sat — ~6h)

Wire MoonBit into the harness; run on smoke tier + applicable gap benchmarks.

- **LOC**: ~60 langreg + ~150 runner + ~2k token teaching prompt + benchmark opt-ins
- **Files**:
  - `internal/eval_harness/langreg/moonbit.go` (new)
  - `internal/eval_harness/templates/agent_task_moonbit.txt` (new)
  - `prompts/moonbit.md` (new — from moonbitlang.com docs)
  - Selected `benchmarks/*.yml` updated with `"moonbit"` in `languages:`
- **Acceptance**:
  - [ ] `ailang eval --lang moonbit --benchmark fizzbuzz` produces a result row
  - [ ] At least 5 smoke + 3 gap benchmarks runnable
  - [ ] Pass rate captured
- **Risk**: medium — `moon` CLI install must succeed

### M6: Vera bring-up — verification-camp peer (Day 4, Sat — ~6h)

Wire Vera; run on contract tier + `shadowing_heavy_contract`. Direct Z3-vs-Z3 peer.

- **LOC**: ~60 langreg + ~150 runner + ~2k token prompt + YAML
- **Files**:
  - `internal/eval_harness/langreg/vera.go` (new)
  - `internal/eval_harness/templates/agent_task_vera.txt` (new)
  - `prompts/vera.md` (new — from veralang.dev docs)
  - Contract-tier benchmarks updated
- **Acceptance**:
  - [ ] `ailang eval --lang vera --benchmark contract_bst_validate` works
  - [ ] At least 3 contract benchmarks + `shadowing_heavy_contract` runnable
  - [ ] Pass rate captured
- **Risk**: high — Vera is research-stage; toolchain unclear

### M7: Aver bring-up — Lean-variant verification peer (Day 5 AM, Sun — ~5h)

Same pattern as M6. Tests `decision_block_capture` natively.

- **LOC**: ~60 langreg + ~150 runner + ~2k token prompt + YAML
- **Files**:
  - `internal/eval_harness/langreg/aver.go` (new)
  - `internal/eval_harness/templates/agent_task_aver.txt` (new)
  - `prompts/aver.md` (new)
  - YAML updates
- **Acceptance**:
  - [ ] `ailang eval --lang aver --benchmark decision_block_capture` works
  - [ ] At least 3 contract benchmarks runnable
  - [ ] Pass rate captured
- **Risk**: high — same toolchain unknowns

### M8: Comparative chart + landing page polish + final QA (Day 5 PM, Sun — ~4h)

Render cross-language chart for self-audit page; foreground `std/ai` on landing; final QA before talk.

- **LOC**: ~100 chart script + ~50 landing updates
- **Files**:
  - Chart rendering (extend existing dashboard or new script)
  - `docs/docs/guides/three-camps-self-audit.md` updated with chart
  - `docs/docs/intro.md` (or landing) foregrounds `std/ai` + coordinator
- **Acceptance**:
  - [ ] Chart embedded in self-audit page (AILANG + N peers across the 14 gap benchmarks)
  - [ ] Landing page mentions `std/ai` and coordinator in headline area
  - [ ] `make verify-examples` passes
  - [ ] `cd docs && npm run build` succeeds
- **Risk**: low

---

## Day-by-Day Schedule

| Day | Date | Focus | Milestones | Hours |
|-----|------|-------|------------|-------|
| 1 | Wed 5/20 | Foundation | M1 (comparison page) + M2 start (4 of 7 gap benchmarks) | ~8h |
| 2 | Thu 5/21 | Benchmarks | M2 finish + M3 (remaining 10 gap benchmarks) | ~8h |
| 3 | Fri 5/22 | Self-audit + MoonBit | M4 (self-audit) + M5 start (MoonBit) | ~8h |
| 4 | Sat 5/23 | Peer langs | M5 finish + M6 (Vera) | ~8h |
| 5 | Sun 5/24 | Aver + polish | M7 (Aver, AM) + M8 (chart/polish, PM) | ~6h |
| — | Mon 5/25 | TALK | — | — |

Buffer: Saturday evening absorbs M5/M6 overrun.

---

## Risk-Driven Scope Reduction Rules

If any peer toolchain (M5/M6/M7) fails, apply in order:

1. **If MoonBit fails**: unexpected. Re-probe; fallback to Raskell (Haskell host = reliable). M5 hours redirect.
2. **If Vera fails**: drop M6 entirely; talk references Vera as "doc-only comparison." Pact would have been substitutable but is out of scope.
3. **If Aver fails**: drop M7; self-audit on `decision_block_capture` becomes "what would Aver test that AILANG doesn't" — still a talk insight.
4. **If MoonBit + Vera + Aver all fail**: Phase 3 collapses; M5–M7 hours redirect to deeper self-audit, more thoughtful comparison page, and a stronger gap-driven follow-up roadmap. The 14 gap benchmarks remain — the talk story is still strong on Phase 1 + Phase 2 alone.

---

## Success Metrics

**Must-have for talk (Mon 5/25)**:
- [ ] `docs/docs/guides/three-camps-comparison.md` published
- [ ] 14 gap benchmarks merged
- [ ] `docs/docs/guides/three-camps-self-audit.md` with AILANG pass/fail data
- [ ] At least MoonBit running on the comparison subset

**Should-have**:
- [ ] Vera + Aver running on contract tier + gap benchmarks
- [ ] Comparative chart in self-audit page
- [ ] Landing page foregrounds orchestration

**Phase 4 (post-talk)**:
- [ ] Audit memos with build/eval/shelve decisions
- [ ] Follow-up sprint plans for "build" decisions
- [ ] Pact + Zero deferred items revisited

---

## Handoff to sprint-executor

This plan is ready for execution. JSON progress file: `.ailang/state/sprints/sprint_M-THREE-CAMPS.json`.
