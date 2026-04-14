# Sprint Plan: v0.11.3 Hotfix — Short-Circuit + parseFold Early Termination

**Sprint ID**: M-V0_11_3-HOTFIX
**Design Docs**:
- [m-eval-short-circuit-bool.md](m-eval-short-circuit-bool.md) — **P0 correctness**
- [m-parsefold-early-termination.md](m-parsefold-early-termination.md) — P1 perf
**Target Release**: v0.11.3
**Duration**: 1.5–2 days (3 milestones)
**Risk Level**: Low–Medium (P0 touches elaborator; P1 adds builtins only)
**Total LOC Estimate**: ~450 (160 short-circuit + 290 parseFold)

## Velocity Context

Recent sprints (last 14 days):
- M-PERF6B: design + implementation + benchmark, 1 session
- M-WASM-TRACE: trace streaming + OTEL span IDs, 1 session
- v0.11.1 / v0.11.2 releases: bug-fix cycles, <1 session each

Current velocity: **~150–200 LOC/hour for well-scoped, single-file changes**. ~100 LOC/hour when elaborator/type-system involved. Conservative estimate for this sprint: 6–8 productive hours.

## Scope Boundaries

**In scope:**
- Desugar `&&`/`||` → `If` at elaboration time
- Fail-fast guard in evaluator (DEBUG_STRICT=1)
- `FoldStep[a]` ADT + 4 new XML fold builtins
- Examples, tests, CHANGELOG, teaching-prompt updates
- v0.11.3 release

**Out of scope (deferred):**
- Linter warning for "dangerous RHS of &&" — separate future doc
- `parseFoldChildren` — m-perf6 Phase 4d
- Phase 4 closure/env optimizations — m-perf6 sprint
- Z3 string-theory encoding — future `m-smt-string-theory.md`

## Ship Strategy: Release P0 Independently if P1 Slips

**If M1 ships cleanly but M2 hits an unexpected blocker, ship v0.11.3 with M1 only** and defer the parseFold work to v0.11.4. The correctness bug is urgent; the perf win is not. Do NOT hold the hotfix release for the P1 work.

## Milestones

### M1 — Short-Circuit `&&` / `||` (P0 correctness) — ~4 hours, ~160 LOC

**Why first**: Correctness bug blocking idiomatic parser code. Must ship in v0.11.3 regardless.

**Tasks:**

1. **Pause point — confirm elaborator strategy** (15 min)
   - Read `internal/elaborate/expressions.go:71-72` and trace how `normalizeBinaryOp` handles `&&`/`||` today
   - Confirm there is no pre-existing `If`-lowering short-circuit path
   - **Pause for user confirmation before modifying the elaborator.**

2. **Desugar in `normalizeBinaryOp`** (~40 LOC, ~1.5 hours)
   - For `Op == "&&"`: emit `core.If(lhs, rhs, BoolLit(false))`
   - For `Op == "||"`: emit `core.If(lhs, BoolLit(true), rhs)`
   - Preserve source position + `Bool` type annotation
   - Ensure CoreTypeInfo is populated for the synthesized `If` (per type-system rule)

3. **Fail-fast guard in evaluator** (~10 LOC)
   - `internal/eval/eval_operations.go:405-418`: add `DEBUG_STRICT=1` panic on `&&`/`||` in `applyBinOp`
   - In non-strict mode, evaluate as today (fallback)

4. **Tests** (~80 LOC)
   - `internal/eval/short_circuit_test.go`:
     - `a && b` with effectful `b` — RHS must not execute when LHS is false
     - `a || b` dually
     - Nested chains: `a && b && c`, `a || b || c`, `a && (b || c)`
     - Trace recording: RHS absent when short-circuited
   - Elaborator unit test: assert `&&` → `If(_,_,False)` shape

5. **Examples** (~30 LOC)
   - `examples/short_circuit_and.ail` — guarded `charAt`
   - `examples/short_circuit_or.ail` — guarded `head`

6. **Acceptance check**
   - `make test` passes (including bytecode VM)
   - `make verify-examples` passes
   - `DEBUG_STRICT=1 make test` passes (no codepath reaches old `applyBinOp` branch)
   - Repro from msg ce6e078e:
     ```bash
     ailang run --entry main -e 'let s = "abc" in let i = 0 in (i > 0 && charAt(s, i-1) == "x")'
     # Expected: false (no panic)
     ```

**Checkpoint:** `.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M1_SHORT_CIRCUIT`

**Pause point before M2**: confirm M1 is green, CHANGELOG drafted, before touching builtins registry.

---

### M2 — `FoldStep` ADT + `parseFoldStep` / `*Limit` Builtins (P1 perf) — ~5 hours, ~290 LOC

**Why second**: Additive — cannot break existing code. Shippable in same release if time permits; defers cleanly if blocked.

**Tasks:**

1. **Pause point — confirm stdlib location** (15 min)
   - Decide: `FoldStep` in new `stdlib/std/iter.ail` vs appended to `stdlib/std/list.ail`
   - **Pause for user confirmation before creating a new stdlib module.**

2. **`FoldStep[a]` ADT** (~10 LOC)
   - `Continue(a) | Stop(a)` exported from chosen module
   - Add to `ailang docs` index if new module

3. **Builtin: `zipXmlScanFoldStep`** (~60 LOC)
   - `internal/builtins/xml.go` — adapt existing `zipXmlScanFold` loop
   - On handler return: `Continue(acc')` → update acc, continue; `Stop(acc')` → return Ok(acc')
   - Register in builtin table

4. **Builtin: `parseFoldStep`** (~40 LOC)
   - Pure-string variant, same pattern, same registry

5. **Convenience wrappers** (~30 LOC)
   - `zipXmlScanFoldLimit(zip, path, tag, maxN, init, f)`
   - `parseFoldLimit(xml, tag, maxN, init, f)`
   - Implemented as thin wrappers calling the `*Step` primitives

6. **Tests** (~100 LOC)
   - `internal/builtins/xml_foldstep_test.go`:
     - Early stop on first match returns Ok(stopped-value)
     - Late stop on Nth match — handler invoked exactly N times
     - `Continue`-only fold is semantically identical to existing `zipXmlScanFold`
     - `*Limit` wrapper processes exactly `maxN` elements

7. **Example + benchmark** (~50 LOC)
   - `examples/bounded_xml_fold.ail` — first-N rows + sentinel predicate
   - Optional: micro-benchmark demonstrating 10x improvement on synthetic 50K-element XML

8. **Acceptance check**
   - `make test`, `make verify-examples` pass
   - Bytecode VM handles new builtins (or they route through evaluator fallback cleanly)
   - `ailang docs std/iter` (or `std/list`) lists the new ADT + builtins

**Checkpoint:** `.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M2_PARSEFOLD_EARLY_TERM`

---

### M3 — Release v0.11.3 — ~1 hour

**Tasks:**

1. **CHANGELOG** — entries under v0.11.3:
   - **Bug Fixes (correctness)**: `&&`/`||` now short-circuit (fixes msg ce6e078e)
   - **Performance**: `FoldStep` + `parseFoldStep`/`*Limit` builtins (if M2 shipped)

2. **Teaching prompt / `ailang prompt`** — mention `&&`/`||` short-circuit semantics; document `FoldStep` pattern if M2 shipped

3. **ack agent messages**:
   ```bash
   ailang messages ack ce6e078e-fdd1-4f64-aff4-7c7a0041ba67  # short-circuit
   ailang messages ack 80f142f6-5cd8-48cc-8490-09accb28be80  # parseFold (only if M2 shipped)
   ```

4. **Move design docs to `design_docs/implemented/v0_11_3/`** via `move_to_implemented.sh`

5. **Release** via `release-manager` skill

**Checkpoint:** `.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M3_RELEASE`

---

## Day-by-Day Breakdown

### Day 1 (~4 hours)
- M1 tasks 1–5 (elaborator desugar + tests + examples)
- **End-of-day checkpoint**: M1 acceptance criteria all green
- Pause point: user confirms M1 before M2 begins

### Day 2 (~5 hours)
- Morning: M2 tasks 1–5 (ADT + 4 builtins)
- Afternoon: M2 tasks 6–8 (tests, example, benchmark)
- **End-of-day checkpoint**: M2 acceptance criteria all green

### Day 2 evening or Day 3 morning (~1 hour)
- M3 (CHANGELOG, ack messages, move docs, release)

**If slippage occurs:** ship v0.11.3 with M1 only at end of Day 1; defer M2 to v0.11.4.

---

## Risk Register

| Risk | Milestone | Impact | Mitigation |
|------|-----------|--------|-----------|
| Desugar happens before type check completes → wrong Core shape | M1 | Medium | Pause point before touching elaborator; read `normalizeBinaryOp` carefully; emit type annotation explicitly |
| CoreTypeInfo not populated for synthesized `If` node | M1 | Medium | Follow type-system rule: validate CoreTypeInfo after elaboration; use `traverse.Walk` where needed |
| Bytecode VM regression on new `If` shape | M1 | Low | `If` lowering is heavily exercised; add dedicated short-circuit test |
| SMT proofs change because of different Core shape | M1 | Low | SMT emits `(and a b)` from either shape; run `ailang verify` on existing contracts as regression |
| Go `xml.Decoder` can't cleanly abort on `Stop` | M2 | Low | Token loop already supports early exit; implementation is straightforward |
| Bytecode VM doesn't have new builtins wired | M2 | Low | New builtins route through standard builtin registry; bytecode VM uses the same table |
| v0.11.3 release blocked by unrelated CI flake | M3 | Low | release-manager skill handles retries; manual cherry-pick if needed |

---

## Success Metrics

- [ ] Reported bug from msg ce6e078e no longer panics
- [ ] All existing tests pass (`make test`)
- [ ] All existing examples verify (`make verify-examples`)
- [ ] `DEBUG_STRICT=1` confirms no codepath hits old `applyBinOp` `&&`/`||` branch
- [ ] 2 new short-circuit examples in `examples/`
- [ ] (if M2) 1 new bounded-fold example in `examples/`
- [ ] (if M2) parseFoldLimit benchmark shows measurable improvement
- [ ] CHANGELOG entry under v0.11.3 covers shipped milestones
- [ ] Design docs moved to `design_docs/implemented/v0_11_3/`
- [ ] v0.11.3 released, GitHub tag present, npm + install script updated

---

## Open Questions (resolve at pause points)

1. **M1 pause point**: After reading `internal/elaborate/expressions.go`, is there any reason the desugar shouldn't live in `normalizeBinaryOp`? (e.g., does the type checker need `BinaryOp` to see `&&` for class dispatch?)

2. **M2 pause point**: `FoldStep` in `std/iter` (new module) or `std/list` (existing)? New module is cleaner; existing module avoids cross-module import cost in hot paths.

3. **Benchmark infrastructure**: Is there an existing harness for "parseFold on 50K-element XML"? If not, micro-benchmark may slip to v0.11.4 — not blocking for release.

---

## Dependencies

- **None blocking** — both milestones are additive (M1) or new builtins (M2)
- **Cross-references**: after M1 ships, `design_docs/planned/v0_11_0/m-smt-cross-module-types.md` documentation-action should note that `&&`/`||` short-circuit aligns evaluator with SMT semantics

---

**Document created**: 2026-04-14
