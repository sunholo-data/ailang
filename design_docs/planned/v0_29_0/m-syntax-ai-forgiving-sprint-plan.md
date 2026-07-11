# Sprint Plan: M-SYNTAX-AI-FORGIVING — Forgiving statement-separator syntax

**Design doc**: [m-syntax-ai-forgiving.md](m-syntax-ai-forgiving.md)
**Planner model**: claude-opus-4-8
**Planned**: 2026-07-11 (mission iteration 9)
**Verification binary**: read-only against HEAD `v0.29.2` (`std/VERSION` = v0.29.2); parser sources read live, not rebuilt.
**Branch**: dev (clean; plan is doc + JSON only, no code touched)
**Risk level**: MEDIUM (R1 low, R2 medium — newline-significance is the class where a missed edge hides; M-TAINT precedent)
**Estimated duration**: 3 days (R1: 1d, R2: 1.5d, canonicalization-lite + doc/eval-handoff: 0.5d)

---

## Sprint Goal

Make the AILANG parser **accept the two statement-separator forms small models naturally
write**, eliminating the `PAR017`/`PAR020` parse-failure class (~32% of small-model failures,
directional) **without changing the model**:

- **R1** — accept `;`-separated statement sequences in `=` function bodies (`func f() = s1; s2; e`).
- **R2** — accept a newline as a soft statement separator inside `{ }` blocks.

Both are **additive and backward-compatible**: no currently-valid program changes meaning.
R1 ships first. R2 merges **only after** the corpus AST-diff fuzz pass is green.

---

## Scope Decision: the `ailang fmt` discrepancy (RESOLVED — option b)

The design doc's Phase 3 assumes `ailang fmt` exists and needs ~0.5d of "canonicalization."
**It does not exist** (verified: no `fmt` case in `cmd/ailang/main.go` dispatch, no formatter
package). A from-scratch AST-reprinting formatter is a multi-day item — it would blow the ≤3-day
budget and smuggle a formatter into a parser sprint.

**Chosen: option (b) — split `ailang fmt` into its own follow-up design doc; mitigate
canonical-form erosion in-sprint via (1) golden *parse* fixtures asserting all accepted forms
produce identical ASTs, and (2) prompt/doc guidance stating the canonical form.**

Justification:
- The v1.0 mission bar (bar v2 clause 3) names "**m-syntax-ai-forgiving landed**" = the parser
  accepts the forms. It does **not** require `ailang fmt`. Accepting the forms is the gate.
- Presentation determinism (Axiom A1 concern) is satisfied for the sprint by golden parse
  fixtures (same AST → same result) + a documented canonical form. Byte-level normalization is
  a *convenience*, deferrable without weakening any A1 guarantee (A1 is about execution/trace
  determinism, which is untouched).
- A real formatter deserves its own doc (idempotence, round-trip, comment preservation, CI gate)
  and is listed under the doc's own "Future Work."

**Deliverable of the split:** a stub follow-up doc `design_docs/planned/v0_29_0/m-ailang-fmt.md`
(≤1 page, deferred) capturing the deferred formatter + canonical-separator choice, so the
erosion risk is tracked, not dropped.

---

## Discrepancies Found (verified live against HEAD)

| # | Doc claim | Reality | Plan action |
|---|-----------|---------|-------------|
| **D1** | Phase 3: `ailang fmt` "0.5d canonicalization" | **No `fmt` subcommand, no formatter package** (`cmd/ailang/main.go` dispatch has no `fmt` case; no `internal/*fmt*` AST formatter). | Re-scoped to option (b): defer `ailang fmt` to `m-ailang-fmt.md`; in-sprint canonical form = golden parse fixtures + doc guidance. Removes the invalid `cmd/ailang/fmt*.go` file from "Files to Modify". |
| **D2** | Files to Modify: `changelogs/v0.10-current.md` | The current changelog is **`changelogs/v0.18-current.md`** (v0.10 changelog is the archived `v0.10-v0.17-bytecode-vm.md`). | CHANGELOG milestone targets `changelogs/v0.18-current.md`. |
| **D3** | Fuzz corpus = "`benchmarks/*.ail`" | **`benchmarks/*.ail` = 0 files** (top-level benchmarks are `.yml`). `.ail` sources are `benchmarks/**/*.ail` (31 recursive) + `examples/**/*.ail` (368; 159 in `examples/runnable/`). | Fuzz corpus glob = `benchmarks/**/*.ail` **and** `examples/**/*.ail`; assert identical ASTs only for files the **old** parser currently accepts. |
| **D4** | R1 "reuse `parseFunctionBody`'s `;` loop" (implied) | The `=` body branch (`parser_func.go:182–198`) calls `parseExpression(LOWEST)` **once** then wraps in `ast.Block`. It does **not** call `parseFunctionBody` (that path is the `{ }` branch only). R1 needs its **own** `; `-sequence loop in the `=` branch. | R1 adds a statement-sequence loop in the `=` branch; do NOT try to route the `=` body through `parseFunctionBody`. |
| **D5** | R2: "reuse `peekStartsBlockStatement`" + `peek.Line > cur.Line` | `peekStartsBlockStatement` (`parser_expr.go:476`) checks **only token type** (LET/LETREC/IF/MATCH/IDENT) — it has **no line awareness**. The line check must be added at the call sites as `p.peekToken.Line > p.curToken.Line`. Precedent: `parser_expr.go:53` already does exactly this comparison for the GAP2 LPAREN fix. | R2 combines `peekStartsBlockStatement()` (type) **AND** `p.peekToken.Line > p.curToken.Line` (line) at the two block loops. |
| **D6** | R2 touches "the block-statement loops" (singular framing) | There are **TWO independent block `;`-loops**: `parseFunctionBody` (`parser_func.go:343`, for `func … { }` bodies) **and** `parseBlockOrExpression` (`parser_expr.go:424`, for standalone `{ }` block expressions incl. if/else branch blocks). Both currently loop only `for p.peekTokenIs(SEMICOLON)`. | R2 must patch **both** loops identically. A single-site fix leaves one block form (e.g. `if … then { newline-block }`) still emitting PAR020. This is the systemic-fix requirement. |
| **D7** | PAR020 emitted only in one place | The `missingBlockSemicolonError()` PAR020 guard fires in **two** places: `parser_func.go:212` (after `parseFunctionBody`) and `parser_expr.go:443` (after `parseBlockOrExpression`). | R2's "keep PAR020 for same-line no-`;`" must preserve **both** guards; they only fire when the line-check is false (same line) after the newline rule is added. |
| **D8** | Dialect-traps: "remove trap #2" on adoption | Trap #2 = the `;`-in-`=`-body rule (R1 target). Trap #3 = match-arm commas (a Conflict-Surface item, unaffected). | If **both** R1+R2 land: rewrite trap #2 to note both forms now accepted (or delete if fully obsolete). **If only R1 lands this sprint** (R2 deferred behind the fuzz gate): do **not** delete — reword trap #2 to keep the newline-in-block guidance (still a live rule until R2). |
| **D9** | `examples/runnable/*.ail` is the example home | Confirmed: `make verify-examples` gates `examples/runnable/` (`make/examples.mk:10`). Feature-example precedent: `examples/if_then_else_blocks.ail`. | New feature example → `examples/runnable/syntax_ai_forgiving.ail` (so it is gated by CI). |

---

## Velocity Basis

Recent parser/diagnostic sprints (M-DIAG-FIXTURE-PROMOTION, M-AILANG-ERROR-QUALITY) landed
90–130 LOC/day with heavy test emphasis. This sprint is **~315 LOC total** (impl ~85, tests
~205 — Conflict-Surface fixtures + the reusable corpus AST-diff fuzz harness, docs ~25) across
3 days — deliberately test-heavy and conservative because the risk is *edge-case correctness*
(the R1→R2 fuzz gate), not typing volume. The fuzz pass and the R1→R2 gate are where the days go, not typing.

---

## Milestones

### M1 — R1: `;`-sequences in `=` function bodies (Day 1)

**Description.** In `parseFunctionDeclaration`'s `=` branch (`parser_func.go:182–198`), replace
the single `parseExpression(LOWEST)` with a statement-sequence parse: `expr (; expr)*`, stopping
at a **declaration boundary**. Wrap the collected exprs in the existing `ast.Block` (same node
`{ }` produces), so elaboration/typing are unchanged. Add a `peekIsDeclBoundary()` helper.

**Decl-boundary rule (verified safe in doc §Conflict-Surface):**
`export | type | import | EOF`, **or** `func` followed by an `IDENT` (a named declaration).
`func (` is an **anonymous-function expression** and stays *inside* the body.

**LOC:** impl ~40 (`parser_func.go`), tests ~55 (`syntax_ai_forgiving_test.go`).

**Test matrix (all must pass):**
| Fixture | Expect |
|---------|--------|
| `func g() -> int = let x = 5; x + 1` (near-miss) | parses; AST identical to braced `{ let x = 5; x + 1 }` |
| multi-stmt `= s1; s2; s3` | parses as 3-expr Block |
| single-expr `= x * 2` (**regression**) | unchanged (single expr, not wrapped-differently) |
| `= …` immediately followed by `func g() = 2` (back-to-back decls) | `f` body ends at `func g`; two decls |
| `= …` followed by `export func …` | body ends at `export` |
| `= …` followed by `type …` / `import …` / EOF | body ends at the keyword/EOF |
| **funclit-in-`=`-body**: `func f() -> int = g(func(x) -> int { x+1 }, xs)` | NOT cut at `func(`; parses whole call (M-TAINT guard) |
| `= map(\x. x*2, xs)` (backslash-lambda in body) | parses |
| operator line-continuation `= 1\n+ 2` | parses as one expr (operator not a stmt-start) |

**Acceptance criteria:**
- [ ] The `=`-body near-miss fixture and the `config_file_parser` `validateVersion` shape (`pure func … = let parts = split(…); length(parts) == 3`) compile with **no `PAR017`**.
- [ ] AST for `func g() = let x = 5; x+1` is **identical** to `func g() { let x = 5; x+1 }` (asserted in test).
- [ ] Every funclit/back-to-back/decl-boundary fixture above passes; `func (` is never treated as a boundary.
- [ ] `go test ./internal/parser/... -count=1` green; `make lint` clean on `internal/parser/`.
- [ ] Single-expr `=` bodies unchanged (no new Block wrapping semantics that break existing examples).

---

### M2 — R1 fuzz gate + example + R1 changelog (Day 1 end → Day 2 morning)

**Description.** Before R2, prove R1 changed **no** currently-valid program. Run the corpus
AST-diff fuzz: parse every `benchmarks/**/*.ail` + `examples/**/*.ail` under a pre-R1 parser
snapshot and the R1 parser; for every file the **old** parser accepted with zero errors, assert
the R1 AST is byte-identical (structural equality). Add the gated example + R1 CHANGELOG entry.

**LOC:** fuzz harness ~40 (`corpus_astdiff_test.go` — reusable for R2), example ~20, changelog ~10.

**Acceptance criteria:**
- [ ] `corpus_astdiff_test.go` parses all `benchmarks/**/*.ail` (31) + `examples/**/*.ail` (368); for every old-parser-valid file, R1 AST is identical → **zero re-parse diffs**.
- [ ] `examples/runnable/syntax_ai_forgiving.ail` created (demonstrates `=`-body `;`-sequence + is gated by `make verify-examples`); type-checks/runs.
- [ ] `make verify-examples` at baseline (no regression vs pre-sprint).
- [ ] R1 CHANGELOG entry added to `changelogs/v0.18-current.md` [Unreleased], referencing `PAR017`, `parser_func.go`, and this doc.
- [ ] **GATE:** M2 green is the precondition to start M3 (R2). If any re-parse diff appears, R1 is reworked before R2.

---

### M3 — R2: newline-as-soft-separator in blocks (Day 2 → Day 3 morning)

**Description.** In **both** block `;`-loops — `parseFunctionBody` (`parser_func.go:343`) and
`parseBlockOrExpression` (`parser_expr.go:424`) — continue the statement loop when `peek` is
`;` **OR** (`p.peekToken.Line > p.curToken.Line` **AND** `p.peekStartsBlockStatement()`).
Preserve the existing record-literal / record-update up-front detection in
`parseBlockOrExpression` (`:` / `|` checks at lines 407/414) — the newline rule must never reach
a record body. Preserve **both** PAR020 guards (`parser_func.go:212`, `parser_expr.go:443`):
they now fire only for the genuine same-line-no-`;` case (line-check false).

**LOC:** impl ~30 (split across both files), tests ~85.

**Test matrix (all must pass):**
| Fixture | Expect |
|---------|--------|
| newline-separated `func f() { let n = length(s)\n countOccurrences(s, ".") }` | parses; AST identical to `;`-separated |
| mixed `;` + newline in one block | parses |
| **record literal** `{ name: "a"\n age: 2 }` | parses as a **record**, NOT two statements |
| **record update** `{ base | f: v\n g: w }` | parses as record-update |
| single-expression block `{ x * 2 }` | unchanged (no false split) |
| multi-line `match x { A => 1, B => 2 }` inside a block | match arms unaffected (comma-separated; out of block loop) |
| operator line-continuation inside block `{ let a = 1\n a + 2 }` vs `{ 1\n+ 2 }` | `+ 2` continuation not split (operator not a stmt-start) |
| **same-line no-`;`**: `{ let x = 1 let y = 2 }` (negative) | still **PAR020** (line-check false) |
| newline block in `if … then { … } else { … }` branch (**parseBlockOrExpression path**) | parses (proves BOTH loops patched — D6) |

**Acceptance criteria:**
- [ ] Newline-separated block fixtures compile with **no `PAR020`**; AST identical to the `;`-form.
- [ ] Same-line-no-`;` still emits `PAR020` (both call sites) — the actionable error is preserved for genuine errors.
- [ ] Record literal / record-update fixtures parse as records, not statements (up-front detection intact).
- [ ] The `if … then { newline-block }` fixture proves **both** loops were patched (systemic fix, D6).
- [ ] Operator line-continuation is never split (matches doc's verified `= 1\n+ 2` / `1 +\n2`).
- [ ] `go test ./internal/parser/... -count=1` green; `make lint` clean.

---

### M4 — R2 fuzz gate, dialect card, `fmt` follow-up stub, R2 changelog + full suites (Day 3)

**Description.** Re-run the M2 corpus AST-diff fuzz with R2 active — **the R2 merge gate**.
Update the dialect-traps card per D8. Create the deferred `m-ailang-fmt.md` follow-up stub
(canonical-form choice + formatter deferred). Add R2 CHANGELOG entry. Run full Go suites +
`make verify-examples`. Hand the rig A/B to the mission controller as a **post-merge** step
(NOT an in-sprint gate).

**LOC:** dialect card ~10, fmt stub ~15, changelog ~10.

**Acceptance criteria:**
- [ ] `corpus_astdiff_test.go` re-run **with R2**: for every old-parser-valid `benchmarks/**/*.ail` + `examples/**/*.ail` file, AST is identical → **zero re-parse diffs**. (This is R2's merge gate, not the hand-picked fixtures.)
- [ ] `prompts/agent/dialect-traps.md` updated: if R1+R2 both landed → trap #2 reworded/removed (`;`-in-`=`-body + newline-in-block rules relaxed); **if only R1 landed** → trap #2 reworded to keep the newline-in-block guidance (do NOT delete). Trap #3 (match commas) untouched.
- [ ] `design_docs/planned/v0_29_0/m-ailang-fmt.md` stub created (deferred formatter + canonical-separator choice), so D1's erosion risk is tracked.
- [ ] R2 CHANGELOG entry in `changelogs/v0.18-current.md`.
- [ ] `docs/LIMITATIONS.md` updated (the `;`/newline separator limitation is relaxed).
- [ ] **Full Go suites green** (`make test`); `make verify-examples` at baseline; `make lint` clean; `make check-file-sizes` (parser files stay < 800 — verify `parser_expr.go`/`parser_func.go` don't cross).
- [ ] **Rig A/B is documented as a deferred post-merge measurement step** (mission controller runs it under `rig_lock_acquire wait`), NOT an in-sprint acceptance gate.

---

## In-Sprint Acceptance (the whole sprint is "done" when)

1. R1 + R2 parser changes landed (or R1-only, with R2 held behind the fuzz gate if it goes red — report which).
2. **Parser fixtures**: every Conflict-Surface fixture in M1 + M3 matrices passes.
3. **Fuzz pass**: `corpus_astdiff_test.go` shows **zero** re-parse diffs over `benchmarks/**/*.ail` + `examples/**/*.ail` for currently-valid files (run after R1 and again after R2).
4. **`make verify-examples`** at pre-sprint baseline; new gated example green.
5. **Full Go suites** (`make test`) green; `make lint` clean; `make check-file-sizes` clean.
6. Docs updated (CHANGELOG `v0.18-current.md`, `docs/LIMITATIONS.md`, dialect-traps card per D8); `m-ailang-fmt.md` follow-up stub created.

**NOT an in-sprint gate (deferred to mission controller):** the rig A/B on local models
(`compile_error` Δ). It is a GPU step; the controller runs/parks it under `rig_lock_acquire wait`
post-merge. The A/B is the *real success metric* but is measured after the parser lands.

---

## Phasing / Sequencing (mandatory)

```
M1 (R1 impl+tests) ──► M2 (R1 fuzz gate + example + changelog) ──[GATE: zero diffs]──►
M3 (R2 impl+tests) ──► M4 (R2 fuzz gate + docs + fmt-stub + full suites) ──[GATE: zero diffs]──► done
                                                                             │
                                                                             └─► (post-merge) rig A/B — mission controller, rig_lock_acquire wait
```

- **R1 ships independently.** If the R2 fuzz gate (M4) goes red, R1 still merges; R2 is
  reworked or deferred to a follow-up. Report the R1-only outcome explicitly (D8 wording rule).
- The fuzz pass — not the hand-picked fixtures — is the R2 merge gate (M-TAINT precedent: a
  2-token lookahead that passed hand-picked tests still mis-parsed ~14 real programs).

---

## Files to Modify / Create

**Modify:**
- `internal/parser/parser_func.go` — R1 `=`-body sequence loop + `peekIsDeclBoundary()` (~40); R2 `parseFunctionBody` loop (~15).
- `internal/parser/parser_expr.go` — R2 `parseBlockOrExpression` loop (line-aware continue) (~15).
- `prompts/agent/dialect-traps.md` — trap #2 reworded/removed per D8.
- `changelogs/v0.18-current.md` — R1 + R2 entries (**not** v0.10).
- `docs/LIMITATIONS.md` — relax the separator limitation.

**Create:**
- `internal/parser/syntax_ai_forgiving_test.go` — R1 + R2 + Conflict-Surface fixtures (~140).
- `internal/parser/corpus_astdiff_test.go` — reusable old-vs-new AST-diff fuzz harness (~40).
- `examples/runnable/syntax_ai_forgiving.ail` — gated feature example (~20).
- `design_docs/planned/v0_29_0/m-ailang-fmt.md` — deferred formatter follow-up stub (~15).

**Explicitly NOT created (D1):** `cmd/ailang/fmt*.go` — no formatter this sprint.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| R1 `func IDENT` vs `func (` lookahead insufficient (M-TAINT precedent) | High | funclit-in-`=`-body fixture (M1) + corpus fuzz (M2) before R1 merges. |
| R2 newline rule misfires on a multi-line expression | Med | Continue only when `peekStartsBlockStatement()` (type) AND `peek.Line > cur.Line`; operators excluded → line-split expressions never split. |
| R2 patched at only one of two block loops (D6) | Med | Systemic-fix requirement: the `if … then { newline-block }` fixture forces both `parseBlockOrExpression` and `parseFunctionBody` to be patched. |
| Record literal mis-read as two statements (D3-adjacent) | Med | Up-front record/record-update detection stays *ahead* of the loop; explicit record fixtures in M3. |
| Canonical-form erosion (two accepted forms, no `fmt`) | Med (accepted) | Golden parse fixtures (same AST) + documented canonical form + tracked `m-ailang-fmt.md` follow-up (D1 option b). |
| `parser_expr.go`/`parser_func.go` cross 800-line size gate | Low | `make check-file-sizes` in M4; changes are small (~30 LOC each). |

---

## Open Questions (non-blocking)

- Canonical-separator choice (`;`-on-line vs newline-per-statement) — deferred to `m-ailang-fmt.md`; does not block R1/R2 parsing (doc "Deferred Decisions").
- Tuple-type `(a; b) → (a, b)` slip — fold in only if trivial; else its own follow-up (doc "Deferred Decisions"). Default: **out of scope** to protect the ≤3-day budget.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_29_0/m-syntax-ai-forgiving-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_m_syntax_ai_forgiving.json`
