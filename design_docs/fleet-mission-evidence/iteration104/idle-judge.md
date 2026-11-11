# Iteration 104 — independent idle-screen judge (verbatim)

Evaluator: `sonnet`, Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only. It was given the inputs and the fleet admissibility rules, not the controller's conclusion. It was told explicitly to re-derive the ticket-to-row mapping first-party rather than carry iteration 103's forward (iteration 103's errata). 64,823 tokens, 8 tool uses, 42 s.

Controller notes, not part of the judge's text:
- The judge did not re-run #2172; the controller's `mission_directives.sh` read found 0 allowlisted of 9 comments there.
- The judge says it read the leading and key sections of each row, not every row's full recommendation and default text. It names that as its own residual risk. This fire's coverage is deeper than iteration 103's 3 tool uses, but it is still not a full read of all 28 rows.

---

1. **Input deltas**
- `cmp tickets.json prev_tickets.json` returned rc=0 and `cmp open-decisions.tsv prev_open.tsv` returned rc=0, so both captures are byte-identical to iteration 103's.
- #2039 has 29 comments, all by `sunholo-voight-kampff`. There are 0 from MarkEdmondson1234, so no allowlisted directive. The controller's #2172 result (0 of 9) was not re-run.
- Dev moved by one commit, `de918c252` "docs(triage): record iteration 36 (#2199)". It touches only `design_docs/triage-mission-log.md`. That is not in `design_docs/fleet-*` and touches none of the listed fleet surfaces.

2. **Per-ticket table** (mapping re-derived from the row text in `open-decisions.tsv` and `charter.md`)

| Signature | Gating row or reason | Evidence | Buildable now? |
|---|---|---|---|
| reporting:daneel-reports-not-declared-triage | D-FLEET-22 | "outside fleet scope. Who declares the inbox?" (key lives in multivac repo) | no |
| quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20 | "OpenAI API key, which has no credits… Which fix?" | no |
| messaging:pubsub-notify-wrong-project | D-FLEET-19 | "is outside fleet scope. Who fixes it, and how?" (rig ADC quota project) | no |
| ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18 | "sit in `tools/eval/`, which the scope guard refuses" | no |
| eval:motoko-connection-probe-orphaned | D-FLEET-18 (charter item 58) | "Retire-or-own is D-FLEET-18's question; folded there" | no |
| ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 (charter item 19) | "Outside fleet scope until D-FLEET-18 is ruled" | no |
| policy:fleet-rulings-not-propagated-to-charters-and-skill | D-FLEET-25 | "asks for a mechanism, so it needs your choice" | no |
| codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26 | "classify codex's `Selected model is at capacity` as transient?" | no |
| quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32 | "ghost for 3 of its 4 asks. Should ask 3… still be built?" | no |
| pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33 | "a `.pi/extensions/**` change, so it parks per D-FLEET-3" | no |
| skill:design-doc-creator-grep-portability | D-FLEET-36 | "the skill is outside fleet scope. Widen… or fix it attended?" | no |
| skills:agents-copies-stale-non-mission | D-FLEET-38 | "all outside fleet scope. Who syncs them?" | no |
| pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35 | "Who adds them to the repo, and should a 402… read as config?" | no |
| stall:watchdog:slow-on-hung-tool | D-FLEET-34 | "ask 1 is a ghost… Should ask 2… still be built?" | no |
| crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23 | "rest of … is outside fleet scope. Who fixes it?" (ask 1 landed #1905) | no |
| sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37 | "asks 1 and 2 … in the coordinator… outside fleet scope" | no |
| coordinator:cloud-lane-capped-by-local-worktree-limit | D-FLEET-37 recommendation A (rows 37 and 40 name it) | "Route ask 1 together with `coordinator:cloud-lane-capped…`"; code in `internal/coordinator` | no |
| executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40 | "what is left is coordinator code. Where does each remaining ask go?" | no |
| executor:timeout-loses-work-and-may-never-start | D-FLEET-40 | same row (named as the addendum) | no |
| executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39 | "Tickets: … lease-collision-fails-concurrent-dispatch… Who fixes it?" | no |
| executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 | same row | no |
| coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 | "plus the adjacent `coordinator:job-retry-success-dropped…`" | no |
| quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31 | "diagnosis half is fixed. Should the quorum also stop spending…" | no |
| quorum:claude-p-ration-usage-429 | D-FLEET-28 | "Anthropic seat is refused whenever usage cannot be read… Which fix?" | no |
| ci:distinct-build-test-failures | D-FLEET-17 class (charter item 44) | "[MEASURED · iteration 68 · D-FLEET-17 class]" | no |
| tool-hang:triage-judge:home-metadata-search | D-FLEET-44 | "Request a corrected policy-scope/conflict-surface design…?" | no |
| skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45 | "narrow Gate 5's idle-record 'degraded fire' bar? Blocked twice" | no |

- Quotes are paraphrased or lightly trimmed to the 15-word limit.
- Rows with no ticket of their own are D-FLEET-16, 21, 27, 29, 41, 42 and 43.
- Two rows are not gates but still read as policy asks. D-FLEET-21 is the CI-poll cap, "a time-budget call, not a mechanical fix". D-FLEET-43 is the idle-skip question.
- Several defaults are recorded. D-FLEET-17 defaults to "C: no change; the fleet keeps building and parks every landing". D-FLEET-41 also defaults to C. D-FLEET-42 defaults to B.
- I did not independently confirm the rest of the 28 per-row defaults.

3. **Dev red assessment**
- Not fleet-scope. The failing test `TestWorkerCLIProcessExitLifecycle` lives in `internal/runner/worker_process_exit_test.go`, product code owned by V1.
- It is flaky. In the saved log the `async/eval/sigint` case failed on a python assertion, `AssertionError: (0, 130, …)`, meaning the process exited 0 where 130 was expected. The `managed/*/sigint` cases and `async/*/sigterm` passed in the same run.
- The same commit passed `CI`. The `Build and Release` history alternates success and failure (2f3ebb81f, 0df45ce29 passed; cfaf5f836, de918c252 failed), which matches intermittent behaviour.
- It is the same class as `ci:distinct-build-test-failures`. Charter item 44 lists three different Build and Release reds in `cmd/ailang`, `internal/daemon` and `internal/runner`, including `TestWorkerTerminalPTYLifecycle`, and ties them to D-FLEET-17. This is a fourth distinct test of that class. The ticket remains gated on D-FLEET-17, and the failure neither unblocks it nor makes it buildable.

4. **Verdict**
- **NONE (idle).** All 27 tickets map to an OPEN row or to a coordinator, multivac or `tools/eval` scope boundary, and none is mechanical and in scope. There is no allowlisted directive.
- Nothing is new since iteration 103: the captures are identical and the only dev change is a triage log.
- Confidence: 93%.

5. **Strongest objection**
- Several rows have default options that a controller could read as the standing state, such as D-FLEET-17 C, and some asks look small. Examples are the pi caps in D-FLEET-35 and the `.agents` sync in D-FLEET-38. A controller could call one of them mechanical.
- I did not read the full recommendation and default text of every row, only the leading and key sections. Each gated item either needs a scope the guard refuses (`tools/eval`, `.pi/extensions`, skills outside the mission set, `internal/coordinator`) or is a policy ask parked under HD-2a. I could have under-read one that carries a harness-only mechanical sub-ask.

6. **Tool-use count:** 7 calls, with several run in parallel in the same turn.
