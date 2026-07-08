# M-VERA-BENCH-INTEGRATION Sprint Plan

**Design doc**: [m-vera-bench-integration.md](m-vera-bench-integration.md)
**Sprint ID**: M-VERA-BENCH-INTEGRATION
**Start**: 2026-05-21 (Thu)
**Target completion**: 2026-05-23 (Sat) — 2-3 days
**Total estimate**: ~14h across 5 milestones
**Risk level**: Medium

---

## Sprint Goal

Wire AILANG into [VeraBench](https://github.com/aallan/vera-bench) — an independent third-party benchmark suite designed by Vera's author with 50 problems across 5 difficulty tiers and published Vera/Python/TypeScript baselines. Adding AILANG as a target language produces the first apples-to-apples comparison of AILANG against multiple peer languages on a benchmark suite **AILANG didn't design**.

Phase 1 (this sprint): wire AILANG as a target. Phase 2 (`verify@1` parity) deferred to a follow-up sprint.

## Scope

**In scope:**
- Clone [sunholo-data/vera-bench](https://github.com/sunholo-data/vera-bench) fork
- Add AILANG to `vera_bench/baseline_runner.py` (wrapper builder + runner function)
- Write 50 AILANG reference solutions adapted from existing AILANG benchmarks/examples
- Run end-to-end validation on tier 1, then full 50
- Capture results comparing AILANG to published Vera/Python/TypeScript numbers
- Open PR upstream to [aallan/vera-bench](https://github.com/aallan/vera-bench)

**Out of scope** (Phase 2 / future):
- `verify@1` parity (translating VeraBench's `contracts.requires/ensures` to AILANG's syntax + running `ailang verify`)
- Multi-model evaluation matrix (this sprint uses 1 cheap model for validation; mainline eval later)
- Side-by-side blog post / paper-style writeup

---

## Working Directory

This sprint touches two repos:
1. **AILANG repo** (`/Users/mark/dev/sunholo/ailang`) — sprint state in `.ailang/state/sprints/`; reference solutions sourced from `benchmarks/` and `examples/runnable/`
2. **VeraBench fork** (cloned to `~/dev/sunholo/vera-bench` or similar) — all code changes happen here; PR opens from this fork

Sprint commits go in the **AILANG repo** only when adapting/refining AILANG-side material (e.g. a CHANGELOG entry). All VeraBench code changes commit in the fork.

---

## Milestones

### M1: Setup + problem-to-source mapping (Day 1 — ~2h)

Clone the fork. Walk all 50 problems. Map each to its closest existing AILANG source (benchmark or runnable example).

- **LOC**: ~150 of mapping documentation
- **Files**:
  - Clone `sunholo-data/vera-bench` to a sibling directory
  - `sunholo-data/vera-bench/AILANG_MAPPING.md` (new, for fork)
- **Acceptance**:
  - [ ] Fork cloned, working directory set up
  - [ ] All 50 problems read; each has either (a) a matched existing AILANG benchmark/example or (b) flagged "needs from-scratch authoring"
  - [ ] Mapping captured in a table: VeraBench ID → AILANG source path → adaptation notes
  - [ ] No-source candidates (~20 estimated) listed for M3/M4 fresh-write
- **Risk**: low

### M2: AILANG harness wiring in baseline_runner.py (Day 1 — ~3h)

Add AILANG as a target language to `vera_bench/baseline_runner.py`. Mirror the Python/TypeScript/Aver pattern.

- **LOC**: ~150 Python in baseline_runner.py
- **Files**:
  - `vera_bench/baseline_runner.py` — add `_EXT["ailang"] = ".ail"`, `_build_ailang_wrapper`, `run_ailang_baseline`
  - `vera_bench/cli.py` — add `"ailang"` to language choices
  - One test fixture: `solutions/ailang/VB_T1_001_absolute_value.ail` (placeholder for M3)
- **Acceptance**:
  - [ ] `_build_ailang_wrapper(problem, baseline_path)` generates valid AILANG that imports the baseline module, calls entry_point with each test case, and emits JSON results to stdout
  - [ ] `run_ailang_baseline(problem, solutions_dir, work_dir, timeout)` invokes `ailang run --entry main`, parses JSON output, returns `ProblemResult`
  - [ ] Wrapper handles test case args (Int, Float, String, Bool, lists) correctly via AILANG syntax
  - [ ] At least one end-to-end run on tier 1/001 placeholder succeeds (check@1 = true, run_correct@1 = true)
- **Risk**: medium — AILANG's module system + JSON encoding interplay needs care

### M3: Tier 1 + Tier 2 reference solutions (Day 1–2 — ~3h)

Write 20 AILANG reference solutions: tier 1 (10 pure arithmetic) + tier 2 (10 string/array). Adapt from existing AILANG material where possible.

- **LOC**: ~400 of AILANG across 20 files
- **Files**: `solutions/ailang/VB_T1_*.ail` + `solutions/ailang/VB_T2_*.ail` (20 files)
- **Acceptance**:
  - [ ] All 20 files written; each has the expected `entry_point` exported
  - [ ] Local verify: `ailang run` succeeds on each + produces correct output for at least one test case
  - [ ] Tier 1 end-to-end run via the harness: ≥9/10 problems pass `run_correct@1`
  - [ ] Tier 2 end-to-end run: ≥7/10 problems pass `run_correct@1`
- **Risk**: medium — string/array operations may surface AILANG stdlib gaps; expect 1-2 problems to need from-scratch implementation

### M4: Tier 3 + Tier 4 + Tier 5 reference solutions (Day 2 — ~4h)

Write 30 AILANG reference solutions: tier 3 (10 ADTs + match), tier 4 (10 recursion + termination), tier 5 (10 multi-function + effects).

- **LOC**: ~700 of AILANG across 30 files
- **Files**: `solutions/ailang/VB_T3_*.ail` + `solutions/ailang/VB_T4_*.ail` + `solutions/ailang/VB_T5_*.ail` (30 files)
- **Acceptance**:
  - [ ] All 30 files written; each has the expected `entry_point` exported
  - [ ] Tier 3 end-to-end run: ≥7/10 problems pass `run_correct@1`
  - [ ] Tier 4 end-to-end run: ≥6/10 problems pass (recursion + termination is hardest tier)
  - [ ] Tier 5 end-to-end run: ≥5/10 problems pass (effects are AILANG's strength but multi-function coordination may surface issues)
- **Risk**: medium-high — tier 4 problems often require `decreases` clauses Vera uses for termination; AILANG doesn't have direct equivalent but HM types should suffice for most cases

### M5: Full 50-problem run, results writeup, PR (Day 3 — ~2h)

Single full sweep with claude-haiku-4-5. Capture pass rates per tier. Write results section for VeraBench README. Open PR upstream.

- **LOC**: ~300 markdown (README update + AILANG_RESULTS.md)
- **Files**:
  - `AILANG_RESULTS.md` (new in fork) — per-tier + overall pass rates
  - Edit fork's `README.md` to add AILANG row in the headline results table
  - Git commit + push fork
  - Open PR to aallan/vera-bench
- **Acceptance**:
  - [ ] Full 50-problem run completes
  - [ ] AILANG pass rate captured per tier + overall
  - [ ] AILANG results vs published Vera/Python/TypeScript numbers documented
  - [ ] PR opened against aallan/vera-bench from sunholo-data/vera-bench
  - [ ] AILANG repo CHANGELOG entry added with PR link
- **Risk**: low — once M3/M4 land, this is mechanical

---

## Day-by-Day Schedule

| Day | Date | Focus | Milestones | Hours |
|-----|------|-------|------------|-------|
| 1 | Thu 5/21 (PM) | Foundation + harness | M1 + M2 | ~5h |
| 2 | Fri 5/22 | Bulk solution authoring | M3 + start M4 | ~5h |
| 3 | Sat 5/23 | Finish solutions + ship | M4 finish + M5 | ~4h |

Note: this is parallel to the talk-prep window (talk Mon 5/25). Sprint should NOT eat into talk-prep time — if M3/M4 slow down, ship Phase 1 with partial tier coverage and document remaining tiers as known-work.

---

## Risk-Driven Scope Reduction Rules

If we fall behind:

1. **If M3 is going slow (string/array surfacing stdlib gaps)**: Reduce M3 acceptance to ≥8/10 + ≥6/10. Move slow problems to "known-work" backlog.
2. **If M4 tier 4 (recursion + termination) hits language-design barriers**: Document specific problems where AILANG's lack of `decreases` clauses prevents direct translation. Ship Phase 1 with tier 4 partial.
3. **If we can't complete 50 by Sat**: Ship PR with tier 1-3 (30 problems) + clear "Tiers 4-5 in progress" note. Still a meaningful first contribution.
4. **If the harness has integration issues with AILANG's module system**: M2 may slip from 3h to 5-6h. Move M3/M4 work into Day 3. Cut tier 5 if needed.

**Talk-floor**: tier 1 + tier 2 results (20 problems × 4 languages) is enough to publish in the talk. Anything more is bonus.

---

## Success Metrics

**Must-have**:
- [ ] AILANG runs end-to-end through VeraBench's harness on at least tier 1
- [ ] At least 20 AILANG reference solutions in `solutions/ailang/`
- [ ] PR open to aallan/vera-bench (even with partial tier coverage)
- [ ] AILANG row in VeraBench's results table

**Should-have**:
- [ ] All 50 problems covered
- [ ] AILANG pass rate in published Vera/Python/TypeScript table
- [ ] CHANGELOG entry in AILANG repo

**Nice-to-have** (Phase 2 territory):
- [ ] `verify@1` parity (contracts translated)
- [ ] Side-by-side AILANG-Z3-vs-Vera-Z3 comparison data

---

## Open Questions

1. **PR strategy**: open against `aallan/vera-bench` `main` directly, or stage in `sunholo-data/vera-bench`? Default: stage in fork, PR upstream once results are clean.
2. **Model selection**: this sprint uses claude-haiku-4-5 for validation (consistent with M-THREE-CAMPS). Mainline eval matching VeraBench's published model set (Kimi K2.5 + GPT-4.1 + Opus 4 + Sonnet 4 + GPT-4o) deferred to follow-up.
3. **AILANG version pin**: use most recent stable (`v0.20.x` series). Note in PR + CHANGELOG.
4. **What if AILANG passes >90% of tier 1?** Strong result; bake into talk. What if <50%? Filed as "real AILANG gaps surfaced — sprint follow-ups for v0.23.x".

---

## Handoff to sprint-executor

Plan ready for execution. JSON progress file: `.ailang/state/sprints/sprint_M-VERA-BENCH-INTEGRATION.json`.
