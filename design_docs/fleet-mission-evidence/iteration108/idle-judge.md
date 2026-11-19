# Fleet iteration 108: independent idle-screen judge

- **Role:** evaluator, `MISSION-ROLE: evaluator`.
- **Spawn:** Agent tool, `model=sonnet`, foreground. 58,954 tok, 6 tool uses, 37 s.
- **Controller:** `claude:claude-opus-5-5`, so the judge is a different model from the same vendor.
- **Brief:** the inputs and the fleet admissibility rules, without the controller's conclusion.
- **Inputs:** this fire's `ailang mission ticket open --json`, iteration 107's and iteration 105's captures (control), the charter at `fleet/i107-record`, `mission_decisions.sh --open` for this fire and iteration 107, the fleet admissibility section of `gate-2-pick.md`, and the recent dev Build-and-Release results.

## Controller corrections to the report

- The judge wrote "26 OPEN rows, D-FLEET-16 to 45 (no 24 or 30)". That count is wrong: 16 to 45 is 30 IDs, and removing 24 and 30 leaves 28. `open-decisions.tsv` has **28** lines, matching `--check`'s 44 rows with 28 OPEN. The ticket-to-row mapping does not depend on the count.
- Residual risk: the judge said the failing-test names came from the brief. The controller read them from the job log for Build-and-Release run 43146305342 (head `ad67f9d82`, job `Build macos-latest`). The log was 5.0 MB, with 23,412 `--- PASS:` lines as the control. The failure was `--- FAIL: TestWorkerCLIProcessExitLifecycle/async/strict/sigint`.

## Judge report (verbatim)

**Verdict: NONE (idle).** Confidence: high, about 90%.

I checked the inputs myself.
- `tickets.json` is byte-identical to `tickets_iter107.json`, so the list did not change.
- `open.tsv` is identical to `open_iter107.tsv`: 26 OPEN rows, D-FLEET-16 to 45 (no 24 or 30).
- `git diff --stat 6d28e1ad5 5343b5430` over the harness paths and `design_docs/fleet-mission.md` is empty, so no harness path and not the charter changed.
- Each of the 27 tickets maps to an OPEN row or a scope boundary. None is buildable.

| # | Ticket signature | Gate | Quoted phrase or reason |
|---|---|---|---|
| 1 | reporting:daneel-reports-not-declared-triage | D-FLEET-22 | "the coordinator config key `triage_only_inboxes`… lives in the multivac repo's cloud config, which no fleet path reaches"; "Default if unanswered: C" |
| 2 | quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20 | Quota/billing lane policy; "Default if unanswered: C, and the ticket stays open" |
| 3 | messaging:pubsub-notify-wrong-project | D-FLEET-19 | "Default if unanswered: C, and the ticket stays open" |
| 4 | ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18 | The flakes sit in `tools/eval/`, which the scope guard refuses. "Widen the authority?" "Default if unanswered: C" |
| 5 | eval:motoko-connection-probe-orphaned | D-FLEET-18 class (item 58) | Asks to retire or own the same two `tools/eval` files, so it is blocked by the same scope ruling. |
| 6 | policy:fleet-rulings-not-propagated-to-charters-and-skill | D-FLEET-25 | Policy-class propagation. "Default if unanswered: C, and the ticket stays open" |
| 7 | codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26 | Option A is already built as draft PR #2065 and is awaiting the ruling. "Default if unanswered: C" |
| 8 | quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32 | Quota threshold policy (HD-2a). "Default if unanswered: C" |
| 9 | pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33 | "parks per D-FLEET-3. Approve the diff?" The diff is proposed. Option A needs Mark's approval, and the `pi_assets` copy is outside scope. |
| 10 | skill:design-doc-creator-grep-portability | D-FLEET-36 | "Widen the fleet's scope to the design-doc-creator skill, or fix it attended?" The skill is outside fleet scope. |
| 11 | skills:agents-copies-stale-non-mission | D-FLEET-38 | "Default if unanswered: C; the ticket stays open." |
| 12 | pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35 | The files are outside fleet scope. Reclassifying `^402:` is D-FLEET-11/13 policy. "Default until answered: the ticket stays open" |
| 13 | stall:watchdog:slow-on-hung-tool | D-FLEET-34 | "Ask 2 is yours." Option B is recommended, which waives it. The sample counts and budget are already ruled in D-FLEET-4/-7. |
| 14 | crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23 | "The rest… is outside fleet scope." The remaining pieces are terraform, coordinator, and `coordinator_cloud_task.go` Step 5b. "Default… C" |
| 15 | sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37 | "Who owns them?" The coordinator is outside scope. "the ticket stays open" |
| 16 | executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40 | PARKED. "Default if unanswered: C; both tickets stay open." |
| 17 | executor:timeout-loses-work-and-may-never-start | D-FLEET-40 | Same row. |
| 18 | coordinator:cloud-lane-capped-by-local-worktree-limit | D-FLEET-37/40 | Coordinator code is outside scope. Option B (widen to `internal/coordinator/**`) is "Not recommended". |
| 19 | executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39 | "Default if unanswered: C; both tickets stay open." |
| 20 | coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 | Same row; the coordinator is outside scope. |
| 21 | executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 | Same row. |
| 22 | ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 class (item 19) | Non-required job, `tools/eval` files, same widening question. |
| 23 | quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31 | Quorum model-cap policy. "Default if unanswered: A. The ticket stays open until this is answered." |
| 24 | quorum:claude-p-ration-usage-429 | D-FLEET-28 | Ration policy. "Default if unanswered: C." |
| 25 | ci:distinct-build-test-failures | D-FLEET-17 class (item 44) | Required-check reds outside fleet scope. "Who fixes it?" "Default… C" |
| 26 | tool-hang:triage-judge:home-metadata-search | D-FLEET-44 | PARKED. "Default if unanswered: B immediately; no planning/execution/merge." |
| 27 | skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45 | "The design was blocked twice by quorum." The recommendation is B, no change. "The ticket stays open until you rule." |

Rows with no ticket attached, which I did not count as buildable:
- D-FLEET-16: skill-source mechanism for Phase 3b.
- D-FLEET-21: Gate 3b poll cap. Option A is already built as draft PR #2067, awaiting the ruling.
- D-FLEET-27: two rulings disagree on `gpt-6-luna`.
- D-FLEET-29: nothing rebuilds the binary the loops run.
- D-FLEET-41: the evaluator image has no make or C compiler.
- D-FLEET-42: loop environment inspection prints secrets.
- D-FLEET-43: whether an all-gated fleet fire should skip itself.

D-FLEET-41's option B (add a missing-toolchain rule to both copies of the sprint-evaluator skill) is in scope and mechanical. It is still an option the row offers Mark, with "Default if unanswered: C", and it is not a ticket-sourced fix, so building it would pre-empt the row. D-FLEET-43's option A is an explicit fleet-scope change that is "built as a proposed diff only if you choose A".

**CI question.** It changes nothing for the fleet. Both latest reds are timing tests on docs-only heads: `TestWorkerCLIProcessExitLifecycle/async/strict/sigint` in `cmd/ailang`, and `TestRuntimeNestedStopUsesRemainingDeadline` in `internal/effects`. Both are language-core or runtime code owned by the V1 mission, whose kill switch is set, so they are outside fleet scope. They fall in the class the charter already records as "V1-owned… recorded not picked" under D-FLEET-17 and item 44. D-FLEET-17's option B would give the fleet standing authority to fix a required-check red. That option is OPEN with default C, so building a fix would pre-empt it. The latest dev head `5343b5430` is a docs-only record, so there is no new signal.

**Strongest objection.** Option B of D-FLEET-41 (the evaluator skill rule), or D-FLEET-36 or D-FLEET-38 (skill-copy fixes), could be argued as a mechanical in-scope skill edit. My answer is that each row explicitly asks Mark to choose, and building one would take an option the row offers him. D-FLEET-36 is explicitly outside scope, and D-FLEET-38 has a default of C with the ticket staying open. The skill-copy tickets are the closest to buildable.

**Residual risk.**
- I read the full text of D-FLEET-17, 18, 29, 33 to 37, 41, 43, 44, 45, 22 and 23. For the other rows I read only the signature mentions and the "Default if unanswered" clauses.
- The tail of D-FLEET-36 was truncated in the TSV itself.
- I did not read the charter's numbered evidence items beyond the grep hits.
- I did not open `build-release-recent.txt` beyond the date/SHA/status lines, nor `job_130944816037.log`. The failing-test names come from your brief.
- I did not check whether any unlisted draft PR changes the picture.
