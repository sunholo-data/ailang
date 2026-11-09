# Sprint Plan: M-QUORUM-REVIEWER-CAP-HIT-LOUD

**Status:** planned  
**Mission:** fleet iteration 62, planner role  
**Design:** [m-quorum-reviewer-cap-hit-loud.md](m-quorum-reviewer-cap-hit-loud.md) (approved for planning)  
**Milestone:** M1 / `M1_CAP_HIT_LOUD`  
**Duration:** one executor run, approximately 60–90 minutes  
**Estimate:** ~30 lines production Go, ~180 lines tests, ~8 lines changelog (~218 lines total)  
**Risk:** low; diagnostic text changes while absence classification and salvage remain intact.

## Goal and current state

Explain a failed reviewer response as truncation when the provider reports `length`, or when it omits the finish reason and its output count reaches the positive declared budget. Preserve valid votes at the cap, `AbsentReason == "invalid"`, salvaged reject vetoes, and ordinary parse errors for non-hits.

The complete approved design was read. At base commit `4aff39811d75da22b1bd218804e52c6a1e04976b`, `runReviewerWith` only rewrites errors for `FinishReason == "length"`; its diagnostic also incorrectly prescribes raising the budget. `run.go` is 193 lines, so no companion split is needed. Existing `stubCaller`, `cheapModel()`, parse fixtures, and salvage tests provide the required test infrastructure. Estimates are scoped to this mechanical change, rather than extrapolated from unrelated sprint velocity.

## Untouched-base verification — 2026-10-27

Every acceptance command below was run before writing either planning artifact or changing implementation, tests, or changelogs. Initial `git status --short` was empty. Temporary cloned Go caches and test temporary directories were kept inside this worktree using `GOCACHE`, `GOMODCACHE`, `GOTMPDIR`, and `TMPDIR`; `GOTELEMETRY=off`. The command bodies were unchanged. Temporary files are removed after the checks.

| Acceptance command | Base result | Evidence |
|---|---|---|
| `go test ./internal/mission/quorum/...` | PASS, exit 0 | `ok github.com/sunholo-data/ailang/internal/mission/quorum 9.368s` |
| `go vet ./internal/mission/quorum/...` | PASS, exit 0 | No output. |
| `gofmt -l internal/mission/quorum/` | PASS, exit 0 | Printed nothing, as required. |
| `make check-changelog` | PASS, exit 0 | Four existing fragments valid; root `CHANGELOG.md` is index-only and links `changelogs/v0.32-current.md`. This does not yet validate the planned new fragment. |
| `grep -c 'FinishReason' internal/ai/ollama/client.go` | Premise (g) confirmed; exit 1 expected | Printed `0`. For grep, zero matches produce exit 1; this is the required premise result, not a failing gate. |
| `grep -n 'Model:' internal/mission/quorum/call.go` | Premise (h), request half, confirmed; exit 0 | `44: Model: c.handler.Model(),` — request uses the provider API name. |
| `grep -n 'out := &ReviewerOutcome' internal/mission/quorum/run.go` | Premise (h), outcome half, confirmed; exit 0 | `112: out := &ReviewerOutcome{Model: modelID}` — outcome retains the models.yml entry ID. |

No base gate is red, and none of these results is **UNINFORMATIVE UNDER SANDBOX**. During execution, a failure caused by socket binds or writes outside the worktree must receive that label, with its error recorded, rather than a pass/fail verdict. Any actual base regression is a finding to report, not a reason to bypass a gate. Recheck and record premise (g)'s count even if upstream changes it: a nonzero count requires inspecting the provider path and documenting whether reported truncation now handles that seat.

## Registry reuse audit

`M1_CAP_HIT_LOUD`: **none**. No package-like AILANG capability is being introduced; this is a local Go diagnostic fix using existing reviewer and test infrastructure. Registry searches, package inspection, new dependencies, and `.ail` examples are not applicable. No AILANG syntax gate is needed because this milestone writes no `.ail` files.

## M1 — M1_CAP_HIT_LOUD: explain reviewer output-cap failures

Dependencies: the approved design and existing quorum test helpers; no other milestone or provider service is required. Implement only these three files:

| File | Planned change |
|---|---|
| `internal/mission/quorum/run.go` | Add `capHit` and `capLimitClause`; use them inside the existing parse-error branch; correct the diagnostic and its explanatory comment. |
| `internal/mission/quorum/cap_hit_test.go` | New tests for all nine design rows below, plus local `cappedModel()` derived from `cheapModel()` with `MaxOutputTokens = 65536`. |
| `changelogs/unreleased/2026-10-27-quorum-reviewer-cap-hit-loud.md` | New fragment beginning `### Fixed — Quorum reviewer output-cap diagnostics`; explain inferred versus provider-reported truncation and preserved valid votes/salvage. |

### Implementation contract

- Add `capHit(resp *ai.Response, budget int) (hit, inferred bool)` in `run.go`, importing the existing `internal/ai` response type. Return `(true, false)` for `FinishReason == "length"`. Otherwise set `hit := resp.FinishReason == "" && budget > 0 && resp.OutputTokens >= budget` and return `(hit, hit)`. Non-hits never report `inferred == true`.
- Call the helper only after `ParseReviewResult(raw)` fails, inside the existing `perr != nil` branch. Keep `recordInvalid(out, raw)` intact and preserve the original `perr.Error()` for non-hits. Do not discard a valid parsed verdict because its count reaches the cap.
- Add `capLimitClause(n, budget int, model string) string` in `run.go`. Use the consumed-budget clause only for `budget > 0 && n >= budget`; use the below-budget provider/context-limit clause for `budget > 0 && n < budget`; otherwise use the no-declared-budget provider/context-limit clause. Preserve the exact wording from Decision 3 of the design.
- Format cap-hit errors as `reviewer output truncated at N tokens (<how>) — <limit>. Parse error: …`. Reported hits say `finish_reason=length, reported by provider`; inferred hits say `finish_reason not reported; inferred from tokens_out N >= max_output_tokens M`. Pass `out.Model` to the limit helper so the quoted entry is `"oc-glm-5-3"`, never the response/request API name.
- Keep the reported-branch substrings `truncated` and `finish_reason=length`. Remove the old `raise it in models.yml` prescription and the comment claiming every truncation proves an understated registry budget. Keep the design's distinction between reported evidence and inference.

### Required test rows and mutations

Every row calls `runReviewerWith` through existing `stubCaller`, with `&ReviewerOutcome{Model: "oc-glm-5-3"}`. Use `cappedModel()` except where cap 0 explicitly requires `cheapModel()`. Use synthetic cheap pricing so the preflight cost check cannot prevent the stub from running. Provide a genuinely truncated reject fixture that the existing salvage logic recognizes, and a valid pass fixture with the required objection/catch fields.

| Row | Input: raw / finish reason / output tokens / cap | Required assertions | Mutation the test must kill |
|---|---|---|---|
| a | Empty string / empty reason / 65536 / 65536 | Absent; `AbsentReason == "invalid"`; `Salvaged == nil`; error contains `truncated`, `inferred`, `65536`, `"oc-glm-5-3"`, and `max_output_tokens`. | Delete the inferred disjunct, restoring today's reported-only behavior. |
| a2 | Empty string / empty reason / 65537 / 65536 | Error contains `truncated at 65537`. | Replace the inference comparison `>=` with `==`. |
| b | Cut-off reject / `length` / 65536 / 65536 | Error contains `finish_reason=length`, `reported by provider`, and `consumed the declared max_output_tokens (65536)`; reject is salvaged. | Delete the `length` disjunct; separately hardcode the signal label to `inferred`. The row must fail for each mutation. |
| b2 | Cut-off reject / `length` / 40000 / 65536 | Error contains `finish_reason=length`, `below the declared max_output_tokens (65536)`, and `provider-side or context-window limit`; error excludes `consumed`; reject is salvaged. | Make `capLimitClause` always return the consumed-budget text by dropping its `n >= budget` check. |
| b3 | Cut-off reject / `length` / 16384 / 0 (`cheapModel()`) | Error contains `declares no max_output_tokens`; excludes both `consumed` and `below the declared`. | Delete the `budget > 0` conjunct from `capLimitClause`'s consumed-budget condition, making `16384 >= 0` falsely select the consumed `(0)` clause. |
| c | `{"verdict":` / `stop` / 65536 / 65536 | Error equals the independently obtained `ParseReviewResult(raw)` error string; excludes `truncated`. | Delete the `FinishReason == ""` conjunct from inference, allowing a reported stop to become an inferred cap hit. |
| d | `{"verdict":` / empty reason / 65535 / 65536 | Error excludes `truncated`. | Delete the `OutputTokens >= budget` comparison. |
| f | `{"verdict":` / empty reason / 20 / 0 (`cheapModel()`) | Error excludes `truncated`. | Delete the inference predicate's `budget > 0` guard. |
| e | Valid pass JSON / empty reason / 65536 / 65536 | `Present == true`; `AbsentReason == ""`; `Err == ""`; non-nil `Result` with `Verdict == pass`. | Move the cap rejection ahead of `ParseReviewResult`, discarding a valid verdict at the cap. |

For each mutation, apply it independently in a disposable scratch source copy inside the executor worktree, run the matching row, and record the failing assertion. Row b requires two separate mutant runs. Restore the saved correct source between mutants; remove the scratch copy before final validation. Do not use git write operations for mutation cleanup. A test that still passes its specified mutant is incomplete.

### Execution sequence within M1

1. First 20 minutes: create `cap_hit_test.go` with all nine rows and confirm the newly covered inference/error-text cases fail against current behavior. Reuse existing fixtures/helpers without editing their files.
2. Next 20 minutes: implement both helpers and the parse-error integration in `run.go`; add the named changelog fragment with its `### Fixed — …` section.
3. Next 20–40 minutes: demonstrate every specified mutation fails its matching row, preserving the existing truncation and salvage tests unmodified. Remove scratch sources and format the touched Go files.
4. Final 10 minutes: run the acceptance commands below, record exit codes/output and premise counts/assignments, inspect the diff for the three-file scope, and update the single milestone's execution state with the evidence.

### Milestone acceptance criteria

- [ ] `capHit` and `capLimitClause` follow the approved predicates, label source, all limit clauses, and `out.Model` identity contract.
- [ ] All nine test rows exist in `cap_hit_test.go`, with the exact assertions and matching mutation failures above (including both row-b mutations).
- [ ] Existing `TestRunReviewerWith_TruncatedOutputFailsLoudly` and `TestSalvageProduction` pass unmodified; `recordInvalid` and the invalid reason remain unchanged.
- [ ] A valid pass at the cap remains present; an ordinary parse error remains byte-for-byte the parser error when the cap predicate does not match.
- [ ] The new named changelog fragment starts with a `### Fixed — …` section and explains the diagnostic change without prescribing budget changes.
- [ ] All four executable gates below pass, formatting prints nothing, and premise checks (g)/(h) have their actual results recorded and assessed.
- [ ] Only the three implementation files listed above change, apart from executor-owned updates to the planning artifacts; no policy, provider, model-registry, or seating edits occur.

## Acceptance commands

Run these exact command bodies from the repository root after implementation:

```bash
go test ./internal/mission/quorum/...
go vet ./internal/mission/quorum/...
gofmt -l internal/mission/quorum/
make check-changelog
grep -c 'FinishReason' internal/ai/ollama/client.go
grep -n 'Model:' internal/mission/quorum/call.go
grep -n 'out := &ReviewerOutcome' internal/mission/quorum/run.go
```

`gofmt -l internal/mission/quorum/` must print nothing. Premise (g) must print `0` on the traced base (grep exit 1 is expected for no matches); record any changed count and inspect the premise rather than changing provider code. Premise (h) must still show `Model: c.handler.Model()` on the request and `out := &ReviewerOutcome{Model: modelID}` on the outcome. The planned unit tests use stubs and do not claim to exercise a running ollama daemon.

## Scope boundaries, risks, and design assessment

No `internal/ai/**` edits, including no ollama `done_reason` mapping. No `models.yml` edits (especially `internal/modelreg/models.yml`) or output-budget reduction. No reasoning-budget restriction or `call.go` request changes. No seating change, re-seat suppression, `seating.go` edits, or mission gate policy edits. Those are separate routing/quota decisions reserved for Mark. No new absence reason or changes to artifact consumers. No new scripts, dependencies, live model calls, or `.ail` examples are needed.

The main risk is falsely describing a reported provider/context limit as consumed declared budget; rows b2/b3 and their negative substring assertions pin that distinction. Another risk is mistaking successful output at the ceiling for a failed vote; row e pins parse-first behavior. An absent finish reason supports inference rather than provider-confirmed truncation, and the diagnostic explicitly retains that distinction.

**Design disagreement:** none. The base checks support the design's traced premises. Planning follows the user's one-milestone `milestones` JSON shape; the generic skill's multi-milestone template and message-based handoff are not used. This stage produces only the requested plan and JSON, with no implementation, git writes, message commands, or executor launch.
