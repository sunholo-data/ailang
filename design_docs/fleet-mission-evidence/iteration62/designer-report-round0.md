I wrote the design doc at `design_docs/planned/m-quorum-reviewer-cap-hit-loud.md` (123 lines). That is the only file I wrote, and I ran no git write operations.

I re-checked your measurements against the banked artifacts and they hold:
- 23 artifacts seat `oc-glm-5-3`.
- Exactly two banked `tokens_out: 65536`, both on kerr-black-holes on 2026-10-21 (`04-39-59Z` and `04-45-36Z`).
- The later run of the same doc (`22-46-13Z`) was present at 56,487 tokens.
- The ollama `Generate` at `internal/ai/ollama/client.go:230` builds its response with no `FinishReason`, so the existing truncation branch never fires for ollama seats.

**Decisions 1–3:**
1. **Predicate:** `FinishReason=="length"`, OR (`FinishReason==""` AND `budget > 0` AND `OutputTokens >= budget`). It is only checked after `ParseReviewResult` has already failed, so a valid verdict at exactly the cap stays present.
2. **Providers that report a reason are believed.** A `"stop"` at the cap is not relabelled. `>=` covers backends that count a token or two past the limit. The `budget > 0` guard matters because test model configs carry a cap of 0; without it, every empty-reason parse failure would be relabelled.
3. **`AbsentReason` stays `"invalid"`.** The outcome still goes through `recordInvalid`, so a cut-off reject is still salvaged and still blocks. Only the `Err` text changes.
4. **Why not a new reason:** the mission-control skill's remedy for `invalid` with no salvaged verdict is already right for a cap-hit ("never raise the cap; at most one solo re-run at the same cap"). A new reason would fall through that table with no rule.
5. **Error text:** `reviewer output truncated at N tokens (finish_reason=length, reported by provider | finish_reason not reported; inferred from tokens_out N >= max_output_tokens M) — the response consumed the declared max_output_tokens (M) of models.yml entry "<id>". Parse error: …`. I dropped the old "raise it in models.yml" advice, because changing the budget parks for Mark.

**Consumers of `AbsentReason`/`ReasonInvalid` I checked:**
- `internal/mission/quorum/salvage.go:95` (`recordInvalid`)
- `internal/mission/quorum/quorum.go:196` (`foldAbsent`) and `quorum.go:51` (`absent_reviewers`)
- `internal/mission/quorum/tier2.go:83`
- `internal/mission/quorum/artifact.go:157` and `:172`
- `internal/mission/quorum/seating.go:143`, `escalate.go:83` and `:113` (these key on `Present` only)
- `cmd/ailang/design_review.go:132` (prints reason and `Err`)
- Places that set a reason, untouched by this fix: `cmd/ailang/design_quorum.go:195`, `claude_reviewer.go:110`, `agentic_caller.go:203`
- `.claude/skills/mission-control/resources/gate-2-pick.md:275-287`, and its mirror `.agents/skills/…/gate-2-pick.md:279-281`
- `tools/launchd/lib/quorum-artifacts.sh` and `test_quorum_artifacts.sh`: they only resolve artifact paths, and their `invalid` matches are namespace checks, not the reason field
- `scripts/`: no matches
- Tests that pin `ReasonInvalid` on a truncation: `reviewer_test.go:221`, `salvage_test.go:201` and `:229`

The doc lists seven tests in a new `internal/mission/quorum/cap_hit_test.go`: (a) to (f) from the brief plus an extra one at 65,537 tokens, each with the mutation it kills. It also covers the Files list, the three out-of-scope policy options, and the two verification commands.
