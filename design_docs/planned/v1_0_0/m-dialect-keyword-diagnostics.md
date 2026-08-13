# M-DIALECT-KEYWORD-DIAGNOSTICS: Diagnose the Observed `case` Trap

**Status**: Planned
**Target**: v1.0.0
**Priority**: P2 — real DX defect, one measured occurrence; not a v1.0 release bar
**Estimated**: 1–1.5 days
**Dependencies**: None
**Issue**: #539

## Decision

Implement one evidence-backed AILANG parser diagnostic for the observed foreign shape
`case e { ... }`. At expression start, when `case` is followed by an expression and a
match-shaped body, emit one `PAR_DIALECT_KEYWORD` diagnostic saying to replace `case` with
`match`, then reuse normal match parsing. Do not accept `case` as an alias and do not reserve it
as a keyword; valid identifier uses such as `case(c)` remain valid.

Do not add spelling-specific behavior for `switch`, `return`, `data`, `enum`, `struct`, `class`,
`elif`, or bare `fn` in this iteration. All eight have zero observed occurrences in the durable
mission record (V1). Their explicit, cheap, uniform fallback is today's normal structured
parser/type-checker failure path. The implementation leaves a small data-driven diagnostic
constructor/mapping seam so another spelling can be added without a new error schema, but the
mapping contains only `case`; it performs no edit-distance matching and no generic foreign-syntax
recovery.

This is intentionally much smaller than the first draft's 2.5–3.5-day nine-spelling plan. That is
the evidence-consistent outcome required by the Minimal Frozen Core and data-before-conclusions
rules, not reduced completeness. In particular, this design does not consume `enum`, `struct`, or
`class` declarations, so it cannot create the contradictory state in which a declaration is
discarded and a later reference produces a secondary `undefined variable` error.

**Trigger to expand scope:** add one of the eight deferred spellings only after either (a) two
independent recent agent failures contain that exact spelling and fail because of it, or (b) a
controlled A/B shows the generic fallback causes a repeated wrong-fix loop. Before expansion, run
the same identifier-collision, live-syntax, recovery-boundary, and base-failing acceptance gates
used here. Consider aliases only after at least five independent failures show the same
unambiguous intended AST and formatter round-trip tests prove canonical output.

## Demand Evidence

The durable mission record contains one relevant occurrence: `case` in the 2026-07-30
`config_file_parser` trial, after 12 turns and 4,867 output tokens. The same record says the agent
used `match` correctly 3–7 times in 07-26 through 07-29 trials that failed for other reasons (V1).
Thus the evidence establishes one actionable diagnostic defect, not a recurring nine-keyword
language gap and not a benchmark-lift claim.

With `ailang check --relax-modules`, the foreign `case` fixture fails first with the uncoded text
`expected ; or }, got =>` and then four `PAR_NO_PREFIX_PARSE` errors; it never names `match` (V2).
The `--relax-modules` narrowing is part of every language probe so MOD010 path validation does not
mask parser behavior. The canonical replacement passes live checking (V3):

```ailang
module probe/valid_match
export func classify(c: int) -> string = match c { 0 => "zero", _ => "other" }
```

## Goals and Non-Goals

Goals:

- Make the one observed foreign `case` shape fail with one coded, actionable parser diagnostic.
- Name `match` and provide the copyable replacement `case` → `match`.
- Reparse the remainder through normal match parsing so the recovered AST is downstream evidence
  that the mechanism ran and the four current cascade errors disappear.
- Preserve valid bindings, parameters, bare identifiers, field access, and calls named `case`.

Non-goals:

- Special handling for the eight deferred spellings.
- Accepting aliases, silently rewriting source, or adding formatter recovery.
- A general typo engine, edit-distance matcher, or declaration synchronization framework.
- Suppressing unrelated errors later in the file; the exact-one-error contract applies to a
  fixture whose only defect is the recognized `case` spelling.
- Claiming this fixes `config_file_parser` or improves benchmark pass rate.

## High-Impact Decisions

| Decision | Rationale | Owner | Change cost |
|---|---|---|---|
| Diagnose only `case` | One occurrence supports one mapping; eight zero-demand mappings violate the scope guard | design | low |
| Diagnostic plus canonical reparse, not alias | Keeps accepted syntax frozen while removing the measured cascade | compiler | low |
| Match an invalid shape, not the bare word | Preserves legal identifier traffic through Pratt parsing | compiler | medium |
| Existing failures are the uniform fallback | Avoids speculative recovery and AST fabrication | design | low |
| Expansion requires a named demand trigger | Prevents the one-entry seam becoming an unmeasured keyword list | design | low |

### Design Freeze

- [x] Implementation mapping contains exactly `case`.
- [x] Diagnostic code is `PAR_DIALECT_KEYWORD`; it is unallocated at base (V5).
- [x] `case` remains an IDENT and is detected contextually (V6).
- [x] Recovery must produce the same match AST as the canonical control, plus one error.
- [x] The eight zero-occurrence spellings receive no production-code change.

## Solution Design

### Recognition and recovery

At the existing Pratt expression entry, recognize `case` only when all of these are true:

1. the current token is IDENT with literal `case` at expression start;
2. it is not immediately in identifier-call or field-access form; and
3. lookahead/reparse reaches the same subject-plus-`{` shape required by `match`.

On recognition, report `PAR_DIALECT_KEYWORD` with the message `` `case` is not AILANG pattern
matching; replace `case` with `match` ``, retain the original near-token, and invoke the canonical
match parser through a narrow adapter that treats the already-consumed leading IDENT as MATCH.
The parser remains in an error state, so checking exits nonzero, while the returned node is a real
`ast.Match` produced by the canonical parser. This mechanism genuinely prevents the measured
arrow/delimiter cascade for an otherwise-valid fixture; it does not promise to hide independent
later errors.

Keep the spelling-to-replacement data as a one-entry local mapping or helper near expression
dispatch. The implementer may choose the helper name and whether the adapter receives a synthetic
token type or an `alreadyConsumed` flag. It must not loop over unrecognized identifiers, reserve
names in the lexer, or synchronize arbitrary declarations.

### Files

- `internal/parser/parser_expr.go:10-61,206-296` — contextual expression dispatch, canonical match
  parsing, and the existing recover-and-report precedent.
- `internal/parser/parser_error.go:10-89` — instantiate the existing structured error; no schema
  change is planned (V9).
- New `internal/parser/dialect_keyword_test.go` — focused recovery, AST-equivalence, negative
  controls, and mutation-killing tests.
- `internal/lexer/`, declaration parsers, formatter, and CLI production code are untouched.

## Conflict Surface

Only expression dispatch and canonical match parsing change.

| Site | Other traffic through the site | Regression risk | Required guard |
|---|---|---|---|
| `parser_expr.go:10-61` | every Pratt prefix expression, including arbitrary IDENT variables, calls, and field access | globally reserving `case` or stealing legal calls | exact contextual predicate; compile controls for a parameter, binding, bare value, field, and call named `case` |
| `parser_expr.go:206-296` | canonical match subjects, arms, guards, commas, nested matches, and `PAR_MATCH_ARROW` recovery | corrupting delimiter state or changing valid `match` behavior | AST equality with canonical `match`; nested match and following-statement fixtures; existing arrow tests unchanged |
| `parser_error.go:10-89` | rendering of every structured parser error | changing global error formatting | instantiate existing `ParserError`; no constructor/schema modification |

Programs/shapes that must still work:

- the live-checked canonical `match` module in V3;
- parameters, local bindings, calls, and field access named `case` (V7);
- nested canonical matches and matches followed by another block expression;
- the existing bad-arrow fixture still reports `PAR_MATCH_ARROW`, and its correct control parses
  (V8).

Intentional incompatibility: none for valid AILANG. The invalid match-shaped `case` form changes
from an uncoded error plus four cascade errors to one coded diagnostic and a recovered AST.

There is no lexer, declaration-parser, type-checker, formatter, or CLI production conflict surface
in this revision because the design no longer modifies those sites.

## Milestones

### M1 — Contextual `case` diagnostic and canonical recovery (0.5–1 day)

- Add the one-entry mapping/helper and `PAR_DIALECT_KEYWORD` report.
- Reuse canonical match parsing and prove recovered AST equality.
- Add negative identifier controls and recovery-boundary tests.
- Independently landable: focused parser tests demonstrate the feature without CLI changes.

### M2 — End-to-end regression gate (0.5 day)

- Add the exact `ailang check --relax-modules --format agent` fixture assertion.
- Run focused parser tests and the parser/CLI blast-radius suites.
- Independently landable after M1: test-only integration coverage; no new production surface.

Total: 2 milestones, 1–1.5 days.

## Acceptance Criteria

Each feature AC states its observable on unmodified HEAD and names the mutation it kills.

1. `ailang check --relax-modules --format agent case_kw.ail` exits nonzero, contains exactly one
   `PAR_DIALECT_KEYWORD`, contains both `case` and `match`, and contains none of the uncoded
   `expected ; or }, got =>` text or `PAR_NO_PREFIX_PARSE`. **Base:** rc=1, zero
   `PAR_DIALECT_KEYWORD`, the uncoded text appears once, and `PAR_NO_PREFIX_PARSE` appears four
   times (V2). **Mutation killed:** suggestion-only reporting that does not enter canonical match
   recovery, or omission/wrong text in the one-entry mapping.
2. The parser result for the foreign fixture contains an `ast.Match` structurally equal, ignoring
   positions, to the V3 canonical fixture, while the parser error list contains exactly the one
   dialect error. **Base:** the foreign fixture has no recovered canonical match AST and has five
   errors (V2). **Mutation killed:** reporting then returning `nil`, or consuming delimiters without
   invoking the match parser.
3. A foreign fixture with a nested canonical match in one arm and a following block expression
   returns both recovered nodes and exactly one dialect error. **Base:** the first foreign arm
   loses delimiter state and emits the V2 cascade. **Mutation killed:** partial recovery that works
   only for flat arms or over-consumes the rest of the block.
4. Separate fixtures for a parameter, local binding, bare value, field, and call named `case`
   contain zero `PAR_DIALECT_KEYWORD`; the well-typed fixtures check successfully. **Base:** the
   parameter/binding/call control is rc=0 (V7), while the new combined mechanism test is absent.
   **Mutation killed:** matching the literal without checking the match-shaped context.
5. A fixture containing recognized `case` followed by an independent malformed expression reports
   one dialect error plus the independent error. **Base:** `case` itself produces five errors before
   the independent defect (V2). **Mutation killed:** clearing the global error list or synchronizing
   past unrelated syntax.
6. `go test ./internal/parser/... ./cmd/ailang/...` exits 0 after implementation. **Base:** exits 0
   but does not satisfy AC1–AC5, which all discriminate the feature. **Mutation killed:** regressions
   in valid match parsing or CLI diagnostic rendering outside the focused fixture.

The eight deferred spellings have no success AC because they have no implementation in this
iteration. A scope-lock test may assert the local mapping has one entry; its mutation is adding an
unmeasured spelling without revising the evidence and this design.

## Test Plan and Mutation Matrix

| Test | Downstream observable | Mutation killed |
|---|---|---|
| `case` CLI fixture | rendered coded diagnostic, exact count, cascade absence | omit mapping; wrong replacement; report without recovery |
| canonical-vs-foreign parser pair | equal recovered `ast.Match` | return `nil`; fabricate a noncanonical node |
| nested match + next expression | both nodes survive | delimiter corruption; over-consume block |
| five identifier contexts | successful parse/type-check and zero dialect code | global reservation; literal-only trigger |
| independent second defect | both independent diagnostics remain | global suppression; recovery to EOF |
| existing match-arrow tests | `PAR_MATCH_ARROW` behavior unchanged | bypass canonical arrow recovery |

## Risks

| Risk | Mitigation |
|---|---|
| Context matcher steals a valid identifier | five positive identifier contexts; no lexer keyword addition |
| Adapter diverges from canonical match parsing | compare recovered AST to a canonical fixture and reuse the same parser |
| Recovery suppresses an unrelated error | two-defect fixture requires both diagnostics |
| One-entry seam invites speculative growth | scope-lock test plus quantified expansion trigger |
| Benefit remains too small for even 1.5 days | stop after M1 if contextual recovery cannot reuse the canonical parser cleanly; retain the generic fallback |

## Axiom Compliance

| Axiom | Score | Rationale |
|---|---:|---|
| A5 Bounded Verification | +1 | One contextual shape has exact AST and error-count observables |
| A7 Machines First | +1 | The machine-readable first diagnostic names the copyable correction |
| A8 Minimal Syntax / Frozen Core | +1 | No accepted syntax and no speculative mappings; eight spellings are deferred |
| A11 Structured Failure | +1 | One measured cascade becomes one coded parser failure |
| All others | 0 | No runtime, effect, authority, concurrency, or boundary change |

**Net: +4.** No hard A1/A3/A4/A7 violation. The re-scope is what makes the A8 score credible.

## Verification Log

All live language probes used `--relax-modules`. PATH `ailang` and the worktree-built binary both
reported `AILANG v0.33.0-184-gfd01a37c1`; worktree HEAD was `fd01a37c1` and was clean apart from
this document. V10 records why the PATH binary's stale warning is an mtime artifact of a fresh
worktree and does **not** indicate a content mismatch. This warning will recur for designers and
sprint executors created in fresh worktrees unless the mtime heuristic changes.

| ID | Exact command | Observed output |
|---|---|---|
| V1 | `rg -n "m-dialect-keyword-diagnostics\|case hypothesis\|case appears on" design_docs/v1-mission.md design_docs/v1-mission-log.md` | Mission queue line 1364 records one 2026-07-30 `case` occurrence and says the 07-26..29 scan refuted sustained causation; log lines 6393–6409 independently record the same result. No occurrence for the other eight spellings is established by this scoped durable record. |
| V2 | `for f in /private/tmp/ailang-dialect.OUzl2c/{valid_match,case_kw}.ail; do ailang check --relax-modules --format agent "$f"; printf 'rc=%s\n' "$?"; done` | Known-positive `valid_match`: `ok`, rc=0. `case_kw`: rc=1, raw `expected ; or }, got =>` once and `PAR_NO_PREFIX_PARSE` four times; zero `PAR_DIALECT_KEYWORD`. Same call, same fixture directory, so the negative diagnostic-code observation has a firing control. |
| V3 | `ailang check --relax-modules --format agent /private/tmp/ailang-dialect.OUzl2c/valid_match.ail; printf 'rc=%s\n' "$?"` | `ok: /private/tmp/ailang-dialect.OUzl2c/valid_match.ail`, rc=0. This is the exact AILANG snippet printed in Demand Evidence. |
| V4 | `nl -ba internal/parser/parser_expr.go \| sed -n '10,61p;206,296p'` | `parseExpression` dispatches by token type; match parsing owns the subject, braces, arm loop, delimiters, and arrow recovery in the cited ranges. This is the mechanism site for contextual dispatch and canonical reparse. |
| V5 | `rg -n 'PAR_DIALECT_KEYWORD' internal/parser cmd/ailang || true; rg -n 'PAR_MATCH_ARROW' internal/parser cmd/ailang` | Candidate has zero hits; same-path known-positive finds `PAR_MATCH_ARROW` at `parser_expr.go:282` and `match_arrow_test.go:43`. `PAR_DIALECT_KEYWORD` is unallocated at base. |
| V6 | `rg -n '"(case|match)"' internal/lexer/*.go` | `match` has keyword/token-map hits; `case` has zero hits in the same scoped call. Thus `case` currently lexes through the identifier path rather than as a reserved keyword. |
| V7 | `ailang check --relax-modules --format agent /private/tmp/ailang-dialect.OUzl2c/valid_identifiers.ail; printf 'rc=%s\n' "$?"` | Known-positive module with a `case` parameter/binding/call prints `ok`, rc=0. The implementation suite splits contexts so a stolen spelling is localized. |
| V8 | `nl -ba internal/parser/parser_expr.go \| sed -n '276,296p'; nl -ba internal/parser/match_arrow_test.go \| sed -n '10,46p'` | `PAR_MATCH_ARROW` names `=>`/`->`, provides a replacement, consumes the wrong arrow as recovery, and tests both bad and correct forms. This is the local report-and-recover precedent. |
| V9 | `nl -ba internal/parser/parser_error.go \| sed -n '10,89p'` | `ParserError` already carries code, message, near token, expected tokens, suggestions, URL, and confidence; constructors/reporting append structured errors. No schema addition is required. |
| V10 | `nl -ba cmd/ailang/help.go \| sed -n '37,53p'; stat -f '%Sm %N' -t '%H:%M:%S' /Users/voightkampff/go/bin/ailang internal/parser/parser_expr.go; cd /Users/voightkampff/dev/sunholo-data/ailang && /Users/voightkampff/go/bin/ailang check --relax-modules /private/tmp/ailang-dialect.OUzl2c/valid_match.ail; cd /Users/voightkampff/dev/sunholo-data/.wt-iter190 && make build; shasum -a 256 /Users/voightkampff/go/bin/ailang bin/ailang /Users/voightkampff/dev/sunholo-data/.wt-old190/bin/ailang; for f in /private/tmp/ailang-dialect.OUzl2c/{valid_match,case_kw,switch_kw,return_kw,data_kw,enum_kw,elif_kw,struct_kw,class_kw,bare_fn}.ail; do diff -u <(/Users/voightkampff/go/bin/ailang check --relax-modules --format agent "$f" 2>&1 \| sed '/Binary may be stale/,+1d') <(bin/ailang check --relax-modules --format agent "$f" 2>&1); done; /Users/voightkampff/dev/sunholo-data/.wt-old190/bin/ailang --version; /Users/voightkampff/go/bin/ailang --version; for f in /private/tmp/ailang-dialect.OUzl2c/{valid_match,case_kw,switch_kw,return_kw,data_kw,enum_kw,elif_kw,struct_kw,class_kw,bare_fn}.ail; do diff -u <(/Users/voightkampff/go/bin/ailang check --relax-modules --format agent "$f" 2>&1 \| sed '/Binary may be stale/,+1d') <(/Users/voightkampff/dev/sunholo-data/.wt-old190/bin/ailang check --relax-modules --format agent "$f" 2>&1 \| sed '/Binary may be stale/,+1d'); done` | `help.go:37-53` walks four CWD-relative source directories and warns when any `.go` ModTime exceeds the binary's; it is an mtime heuristic, not a content check. Measured binary mtime `07:02:33`, fresh-worktree `internal/parser/parser_expr.go` mtime `07:04:25`; the same PATH binary emitted no warning from the non-worktree checkout whose source mtime was older at measurement time. Worktree-local `make build` returned rc=0. Artifact SHA-256 values: PATH `ce9ddd4c4514acf1fe6ef215d153acd29a2542510b5050488984b4e577782662`; worktree build `c2311213a1c6bbb4c9fbd31aea8230902f0de006bff455fb3bbe8d46fd59af35`; old control `3fae4b4652a2a32ef8546cae8fafe7eda5a6fb6cbf2a1e37e04b343513d523cc`. After removing only the two warning lines, PATH-vs-worktree output was byte-identical on 10/10 fixtures and both versions were `v0.33.0-184-gfd01a37c1`. Negative control against binary `0a84f5377` (`v0.33.0-141`) detected 1/10 differing fixtures (`ctl_match`, HEAD adds relaxed `MOD010`) and the version difference. The differential is therefore sensitive, and the fresh-worktree warning does not indicate a content mismatch. |
| V11 | `sed -n '1,180p' design_docs/implemented/v0_30_0/m-syntax-ai-forgiving.md; rg -n "M-SYNTAX-AI-FORGIVING R1\|M-SYNTAX-AI-FORGIVING R2" internal/parser` | The implemented design accepted separator variants only after measured frequency and an AST-diff gate; parser comments locate both mechanisms. It did not add keyword aliases. This supports evidence-gated expansion, not broader present scope. |

## Related Documents

- [M-SYNTAX-AI-FORGIVING](../../implemented/v0_30_0/m-syntax-ai-forgiving.md) — acceptance-lane
  precedent distinguished by stronger frequency evidence and an AST-diff gate.
- `internal/parser/parser_expr.go:276-296` and
  `internal/parser/match_arrow_test.go:10-46` — local diagnostic/recovery precedent.
- [V1 mission queue](../../v1-mission.md) and [mission log](../../v1-mission-log.md) — demand and
  refutation provenance.

---

**Document created**: 2026-08-13
**Revised after quorum**: 2026-08-13

---

## Quorum verification log (controller, iteration 190, 2026-08-13)

**Status: PARKED `needs-human-review`.** Two rounds, BLOCKED both times, `absent_reviewers: []`
in both (no N−1 degrade). Total metered $0.1077. The narrow-refinement carve-out does **not**
apply: neither round-2 objection carries a reviewer-authored `proposed_fix` (both `<none>` in the
artifact), and objection R2-B requires choosing a detection mechanism the reviewers did not
specify — a controller-invented resolution, which the carve-out forbids. Standing rule 2 binds.

### Round 1 — BLOCKED ($0.0570), artifact `m-dialect-keyword-diagnostics-2026-08-13T05-12-43Z.json`

- **`gpt5-6-sol` — PREMISE, REFUTED BY CONTROLLER MEASUREMENT.** Claimed the probes were taken
  with a binary that "explicitly warned it was stale". Measured: the warning is an **mtime
  heuristic** at `cmd/ailang/help.go:37-53`, walking four **CWD-relative** directories and warning
  if any `.go` mtime post-dates the binary. A fresh `git worktree` rewrites all 23,610 files, so it
  trips on a **content-identical** binary; the same binary run from the non-worktree checkout emits
  nothing. Content identity proven by differential rather than assertion: a binary built **from the
  worktree** and the PATH binary produce **byte-identical output on 10/10 fixtures** (only the two
  warning lines differed). Negative control — the same differential **is sensitive**: against a
  `HEAD~40` build (`0a84f5377`, `v0.33.0-141`) it detects a difference (1/10 fixtures, plus
  `--version`). The reviewer was right that the doc's original justification was an appeal to
  authority; it was wrong that the probes were untrustworthy. Applied in revision.
- **`gemini-3-1-pro` — DESIGN, ADJUDICATED CORRECT.** 2.5–3.5 days of parser surface for nine
  spellings, eight with zero recorded occurrences, against the Minimal Frozen Core axiom. Aligned
  with `design_docs/PROGRAM.md` and the mission's demand-evidence guardrail. Directed a hard
  re-scope rather than a defence → **`case` only, 1–1.5 days, 2 milestones**; the other eight
  deferred behind a named evidence trigger. Its second half (consuming a foreign declaration
  without an AST node still yields a downstream `undefined variable`) was also correct and was
  resolved explicitly in the revision.

### Round 2 — BLOCKED ($0.0507), artifact `m-dialect-keyword-diagnostics-2026-08-13T05-22-27Z.json`

- **`gemini-3-1-pro` — CONFIRMED FIRST-PARTY, AND BIGGER THAN FILED. This is the blocker.** It
  asked whether the parser can peek past an arbitrary subject expression. It cannot, and the
  revised mechanism ("lookahead/reparse to detect the `{`") is **infeasible as written**. Measured:
  the parser is a streaming Pratt parser with **fixed 4-token lookahead** (`curToken` +
  `peekToken`..`peek4Token`, primed by five reads at `parser.go:134-139`); `nextToken()`
  (`parser.go:214-220`) is a pure forward shift that overwrites the consumed token with **no saved
  position**; and there are **ZERO** `save`/`restore`/`rewind`/`mark`/`backtrack`/`snapshot`
  methods (control: **125** `*Parser` method definitions, so the zero is a measurement). A subject
  expression is unbounded in tokens, so no fixed lookahead can reach its `{`.
- **`gpt5-6-sol` — CONFIRMED as a genuine verification gap.** The "zero observed occurrences for
  the eight deferred spellings" premise is not established. Measured: the charter+log contain
  backticked hits for seven of the eight (`switch` 2, `return` 4, `enum` 1, `struct` 2, `class` 3,
  `elif` 1, `data` 0; control `` `match` `` 8) — but those are this queue row's own prose, and the
  mission record stores **conclusions, not raw transcripts**, so it cannot establish absence in
  either direction. The instrument that could is a scan of banked eval transcripts
  (`ailang chains chat`, motoko session JSONL) across 468 recorded nights — an investigation, not
  a doc edit.

### Bounded options for the human decision (controller-enumerated FROM the measurements above; not a chosen resolution)

The design direction (`case`-only dedicated diagnostic) survived both rounds. Only the **detection
mechanism** is blocked, and the 4-token/no-rewind constraint bounds it to:

- **(A) Recovery-site detection** — no rewind needed. Detect at the existing failure point
  (`parser_literals.go:562`, where `expected ; or }, got =>` is raised, i.e. *after* the subject
  is consumed), carrying a breadcrumb of the statement's first identifier. Cheapest; fits the
  current parser exactly.
- **(B) Statement-initial soft keyword** — recognise `case`/`switch` only in statement-initial
  position. Fits inside the 4-token window for simple subjects; an arbitrary subject still escapes.
- **(C) Add parser backtracking** — a `save`/`restore` mechanism. Touches the assumptions of all
  125 parser methods and is a core change, against the north star's default-to-extension bias.
- **(D) Drop the item** — demand is one observed occurrence, and the standing counter-argument
  (auto-fix is the lever, not sharper prose) is unrefuted. `ailang fmt` cannot serve as that
  auto-fix today: it parses before formatting and leaves parse-invalid input byte-identical (V5).

A secondary, independent question: is the transcript scan that would settle `gpt5-6-sol`'s
occurrence premise worth the rig time, or should the doc simply narrow its claim to "no occurrence
is *recorded*" and stop asserting absence?
