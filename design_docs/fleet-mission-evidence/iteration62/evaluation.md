# Iteration 62 — independent evaluation (verbatim report)

Judge: Agent tool `sonnet` (resolver `agent-tool sonnet declared:alias-pin`). Generator: codex `gpt-6.1-sol` (planner + executor), designer `claude-opus-5-5`. Judge worktree `.wt-fleet-iter62-judge`, detached at `56e1a2055`; clean at the end. Subagent tokens 56,815; 5 tool uses; 107 s.

Because the return was fast, the controller checked it against disk before accepting it. `/tmp/run.go.orig` exists and is byte-identical to the judge tree's `run.go`. The judge tree is clean. Mutant M1 (`>=` to `>`), re-applied by the controller, reddens `TestRunReviewerWith_CapHit/a`.

---

VERDICT: PASS 93/100

**Rubric (100 points)**
- Design premise and root-cause fidelity: 20/20
- Correctness of `capHit` and `capLimitClause`: 24/25
- Tests and mutation kill rate: 24/25
- Scope and authority: 10/10
- Changelog and docs accuracy: 8/10
- Plan and design coherence: 7/10

**Gates** (outside any sandbox, unpiped): `go test -count=1 ./internal/mission/quorum/...` exited 0 (`ok … 5.719s`). `go vet` exited 0. `gofmt -l` exited 0 with no files. `make check-changelog` exited 0 ("5 changelog fragment(s) valid").

**1. Premise trace: confirmed.** `handlerCaller.CallJSON` calls `Provider().Generate`. `ollama.Client.Generate` (client.go:120–238) never sets `FinishReason`. `OutputTokens` comes from `tokens.go` `EvalCount`. The only `FinishReason` writers in the ollama package are on the Step paths (`step.go`, and `streamstep.go:295`), so the old `length` branch could not fire for oc-glm-5-3.

**2. Edge cases.**
- A reported "stop" at the cap is not relabelled (row c).
- A reported "length" is always a hit and never marked inferred.
- A reported "length" below the budget gets the provider/context-limit text (b2).
- No declared budget gets the "declares no max_output_tokens" text (b3).
- OutputTokens above the budget is a hit (a2).
- Budget 0 is never inferred (f).
- One token below the budget is not inferred (d).
- A valid pass at the cap stays present (e).
- AbsentReason stays `invalid` throughout.

**3. Mutations.** All 9 were killed, and 7 of them were the judge's own (not on the plan's list).

| Mutation | Killer |
|---|---|
| M1 `>=` to `>` | CapHit/a |
| M2 drop `FinishReason==""` | CapHit/c |
| M3 drop `budget>0` | CapHit/f |
| M4 `inferred` always false | CapHit/a |
| M5 reported length marked inferred | CapHit/b, b2, b3 |
| M6 clause `n>=budget` to `n>budget` | CapHit/a, b |
| M7 drop `recordInvalid` | CapHit/a, a2, b, b2, b3 |
| M8 infer on any non-length reason | CapHit/c |
| M9 model name blanked | CapHit/a, b, b2, b3 |

**4. Existing tests.** `TestRunReviewerWith_TruncatedOutputFailsLoudly` (reviewer_test.go:208) and `TestSalvageProduction` (salvage_test.go:206) are unmodified and pass. Salvage of a cut-off reject still works (b, b2, b3).

**5. Scope.** 6 files, all inside the fleet allowlist. Nothing changed under `internal/ai/**`, `internal/modelreg/**`, seating or skills.

**6. Changelog.** The fragment is accurate. Removing "raise it in models.yml" was correct: the budget is already the model's full declared strength, and changing it is a routing/quota decision. The old advice was also wrong for a reported "length" below the budget.

**Findings (all NON-BLOCKING)**
- `run.go:187-193`: `capHit` and `capLimitClause` have no doc comments, unlike the neighbouring functions. The inline comment at :167-169 covers the intent.
- `cap_hit_test.go`: the exact-string wording assertions are deliberate but brittle.
- Inference cannot tell a reasoning-exhausted cap from a model that happens to emit exactly the budget. The `inferred` label in the message says so.
- No test drives the real ollama `Generate` path. The premise is verified only by reading the code, which the design records as premise check (g).
