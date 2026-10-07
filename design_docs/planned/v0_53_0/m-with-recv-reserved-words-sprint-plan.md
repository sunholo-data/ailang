# Sprint Plan: M-WITH-RECV-RESERVED-WORDS

**Design:** [Approved design](m-with-recv-reserved-words.md)
**Sprint ID:** M-WITH-RECV-RESERVED-WORDS
**Created:** 2026-10-07
**Status:** Planned; implementation follows coordinator plan approval.
**Target:** v0.53.0
**Duration:** 2 working days (12 hours planned work + 3 hours contingency).
**Risk:** Medium — parameter synchronization must preserve delimiter ownership.

## Goal and scope

Keep `with` and `recv` reserved. Make reserved-word parameter mistakes produce one correctly positioned `PAR_RESERVED_KEYWORD` rather than unrelated cascades; improve record-field messages and reservation-specific suggestions; restore the diagnostic's reference route. Preserve accepted syntax and PAR019. This is option (c) from the approved design, not an implementation of the parked contextual `case` proposal.

No general suppression policy, lexer change, keyword release, binding recovery redesign, import-alias change, runtime change, or prompt revision is included. Locals and function names retain their existing primary diagnostic and may retain their existing cascades; record-pattern bindings are outside R1–R4. Record fields retain `PAR_FIELD_NAME_EXPECTED`, with keyword-aware text and suggestions, as R2 specifies. The broad goal “every name position” is bounded by those concrete design sites.

## Current status and estimates

Planning base: `3053434e`, `coordinator/task-cef69c45`, std version v0.52.5. The approved artifact was absent locally and was recovered verbatim from `9018c13d` on `origin/coordinator/task-385c68ec`; it is included alongside this plan for executor continuity. No compiler changes have been made.

Source inspection confirms both silent name checks in `parseParams` (`internal/parser/parser_lambda.go:157`), keyword suggestions in `peekError`, and generic record-field errors in `parseRecordFieldDef`. `parseParams` serves functions and parenthesized lambdas; the backslash multi-argument lambda parser is separate and gets a valid-traffic control rather than a speculative recovery change. Existing reserved-word and recovery tests are available. The general cascade design still cites `parser_expr.go:750`; implementation must update its citation and document the local/general recovery boundary.

`analyze_velocity.sh 7` ran, but this checkout is shallow and exposes only the 2026-10-07 triage commit; its parent diff is unavailable. Archived LOC output is historical, not a recent velocity sample. The current changelog records v0.52.4/v0.52.5 deliveries on 2026-10-07 without implementation-hour metrics. No measured LOC/day rate or current coverage percentage can honestly be inferred. Use the design's 1.5-day lower estimate plus 25% time contingency, rounded to two working days. Target planning capacity is 200 authored LOC/day, an estimate, not observed throughput.

| Milestone | Production/tooling LOC | Test LOC | Docs/example LOC | Total | Work |
|---|---:|---:|---:|---:|---:|
| M1 diagnostics and local recovery | 90 | 180 | 10 | 280 | 8 hours |
| M2 reference route and integration | 15 | 25 | 80 | 120 | 4 hours |
| Total | 105 | 205 | 90 | 400 | 12 hours + 3 buffer |

Generated llms bundles are excluded from authored LOC estimates. Estimates are budgets, not exact patch-size requirements.

## Registry reuse audit

Both milestones: **none**; package is null. No package-like capability is introduced: M1 changes compiler-internal token diagnostics and parser cursor recovery; M2 repairs repository-local documentation and its existing generator. Registry search is not applicable under the skill's “package-like capability” condition; no registry searches or candidate inspections were performed. A runtime AILANG package cannot supply either compiler cursor recovery or Docusaurus navigation. Reuse existing `ParserError`, lexer `IsKeyword`, parser fixtures, and `tools/generate-llms-txt.sh` instead.

## Milestones

### M1: Reserved-word diagnostics and bounded parameter recovery (~280 LOC)

**Dependencies:** None. **Duration:** Day 1, 8 hours.

**Files:** `internal/parser/parser_lambda.go`, `internal/parser/parser_error.go`, `internal/parser/parser_type.go`; extend `internal/parser/reserved_keyword_test.go` and `internal/parser/error_recovery_test.go`, or add `internal/parser/reserved_keyword_recovery_test.go`. Update only the stale location/boundary prose in `design_docs/planned/m-parser-error-cascade-suppression.md`. Create a successful rename control at `examples/runnable/reserved_keyword_names.ail`; invalid snippets live in Go test fixtures, not runnable examples.

1. Capture HEAD parser/agent-format baselines with a worktree-built binary. PATH reports v0.52.5 commit 7200786 and is not the planning HEAD; do not treat it as current-source evidence. Fetch `ailang prompt` before authoring any `.ail` fixture/example.
2. Share reserved-word message/suggestion construction between peek-token and current-token callers without changing global error schema. Keep token positions and existing HelpURL; split MATCH and WITH suggestion handling. Add owner/reason and legal alternatives for WITH, SEND/RECV/TIMEOUT, CHANNEL/SPAWN/PARALLEL/SELECT, ASSERT; leave unrelated keyword suggestions intact.
3. Factor the duplicated parameter-name handling. Report a keyword at its actual current position, drop that invalid parameter, and resume at the next comma or the enclosing closing parenthesis; stop safely at EOF. Define cursor ownership explicitly so the caller observes the same closing-parenthesis state as successful parsing.
4. Guard nested type delimiters during recovery: a comma/parenthesis within a bad parameter's function/record type is not the outer separator. Use the smallest local depth-aware synchronization needed; no lexer lookahead/backtracking or declaration-boundary suppression. Test first, middle, final, repeated, untyped, and nested-type failures plus a valid next declaration.
5. Enrich only keyword-triggered record-field reports, retaining `PAR_FIELD_NAME_EXPECTED`; generic bad tokens keep existing text and recovery.
6. Run focused parser tests, then parser/lexer and CLI suites. Keep the suggestion-to-page contract test in M2 so M1 remains independently landable.

**Acceptance criteria:**

- [ ] Isolated `with`, `recv`, and `send` parameter fixtures fail with exactly one `PAR_RESERVED_KEYWORD`, at the offending token, and none of the parameter-corruption cascade codes. Bodies use surviving names or constants to avoid independent undefined-name defects.
- [ ] Table-driven keyword coverage proves the name-position diagnostic works for every entry in the existing 41-entry keyword map, without changing lexer files.
- [ ] Recovered AST contains exactly the surviving `a` and `b` parameters in order for a bad middle parameter; closing-parenthesis ownership, function body, and following declaration survive. Invalid source still fails checking. A separately corrected three-parameter control checks successfully; do not claim that dropping a parameter makes a three-argument call type-check.
- [ ] Nested type separators, repeated bad parameters, zero-argument controls, EOF, and parenthesized lambda cases terminate without lost valid parameters or fabricated empty names.
- [ ] A later independent malformed expression remains diagnosed alongside the keyword failure; global error suppression is absent.
- [ ] Local/function-name fixtures retain `PAR_RESERVED_KEYWORD` first, with counts no greater than their measured HEAD baselines and new actionable suggestions.
- [ ] Record-field keyword fixtures retain `PAR_FIELD_NAME_EXPECTED` and name the reservation; non-keyword malformed fields retain the generic message.
- [ ] PAR019 agent-format output is identical to the captured same-binary baseline; valid `case` identifier contexts, functions, records, and backslash lambdas remain valid.
- [ ] `go test ./internal/parser/... ./internal/lexer/... ./cmd/ailang/...` passes; renamed example checks successfully with the built binary.

**Risk:** Synchronization could eat a nested type or the next declaration. AST/cursor boundary assertions and two-defect fixtures must reject that mutation. If recovery requires general parser suppression or semantics changes, stop and return that scope question for design revision.

### M2: Reference route, discovery bundles, and integration checks (~120 LOC)

**Dependencies:** M1. **Duration:** Day 2, 4 hours + 3 hours contingency.

**Files:** Create `docs/docs/reference/reserved-keywords.md`; update `docs/sidebars.js`; remove `docs/reference/reserved-keywords.md`; narrowly update `tools/generate-llms-txt.sh` to include the canonical page once; regenerate `docs/llms.txt`, `docs/static/llms.txt`, and root `llms.txt` using that existing generator. Add the suggestion/page contract assertion to the focused parser test file. No new docs generator or registry dependency.

1. Build the reference table from the actual lexer map: 41 entries today, each listed once. Document current use versus planned reservation without inventing lane decisions. Record `with`'s effect-handler syntax-freeze trigger and PAR019 role; record `recv`/`send`/`timeout`'s CSP syntax-freeze trigger; cite the existing lane for each other unused reservation and its consume/release decision.
2. Add `reference/reserved-keywords` near `language-syntax` in the Reference sidebar. The sidebar header about `sync-registry.sh` applies to package data only; edit the static Reference group directly.
3. Fix the narrow discovery path in the existing generator: it currently enumerates `docs/reference/*.md`, so deleting the orphan alone would remove reserved-word documentation from the bundle. Include the new canonical page exactly once without migrating all reference pages in this sprint. Regenerate all three outputs the tool owns.
4. Assert suggestions and the reference table share the promised alternative names. Capture CLI diagnostics using the built binary; run the docs build and verify its generated reserved-keywords route. Review tracked side effects of `npm run build`'s sync-all step before including generated changes.

**Acceptance criteria:**

- [ ] Canonical page exists, sidebar resolves it, obsolete orphan is gone, and page entries match all 41 current keywords exactly once.
- [ ] Page and parser suggestions agree on the named alternatives and reservation reasons; owning-lane consume/release triggers are explicit, including the separate handler and CSP decisions.
- [ ] The existing llms generator includes the canonical reference once; all three generated bundles contain its corrected table, with no stale orphan section.
- [ ] `npm ci` followed by `npm run build` in `docs/` succeeds and produces the reserved-keywords route. Local served route returns HTTP 200. Production HTTP 200 is checked after the normal site deployment, not required before deploying this sprint and not claimed by planning.
- [ ] Focused suggestion/page assertions, parser/lexer/CLI suites, `make test`, and `make lint` pass; changed Go files are gofmt-clean. Run `make check-boundaries` only if implementation adds imports crossing package boundaries.
- [ ] Working rename example and existing `examples/docs/lambdas_full.ail` and `examples/docs/records_person.ail` check using the built binary. No lexer, typechecker, evaluator, or match production code changes appear in the final diff.

**Risk:** Generated bundle size and unrelated build synchronization could obscure the fix. Inspect generator output for duplicates and include only attributable source/generated changes. Doc snippet validation requires the current teaching prompt before editing AILANG.

## Execution schedule and finish gate

Day 1: baseline capture and shared diagnostics (2 hours), parameter recovery with boundary tests (4 hours), field messages and focused/regression checks (2 hours).

Day 2: canonical reference and generator repair (2 hours), contract/CLI/docs build and review (2 hours), recovery/docs contingency and full required checks (3 hours). If setup consumes the buffer, report remaining validation rather than claim completion.

Use branch-local binaries; no absolute coverage percentage is set without a baseline. Success means all new failure branches have discriminatory tests, valid-traffic controls pass, independent diagnostics remain, and the reference route is built. Optional paired eval is non-gating and excluded from the estimate. Optional quorum is a design-review activity, not an executable milestone.

The design is approved per handoff. This sprint is prepared for coordinator review; merging/approving the plan PR triggers sprint-executor under the configured coordinator workflow. JSON starts `not_started`, all milestone results null. Do not start implementation or send a duplicate executor task from this planning stage. PR #1620 is provenance, not an open implementation issue to auto-close; `github_issues` stays empty unless a real implementation issue is identified.
