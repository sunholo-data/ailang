# Fleet iteration 110: independent idle-screen judge

- **Role:** evaluator, `MISSION-ROLE: evaluator`.
- **Spawn:** Agent tool, `model=sonnet`, foreground (`resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`). 91,306 tok, 15 tool uses, 68 s.
- **Controller:** `claude:claude-opus-5-5`, so the judge is a different model from the same vendor.
- **Brief:** the inputs and the fleet admissibility rules, without the controller's conclusion and without any earlier iteration's judge verdict. The judge was told to argue against idling.
- **Inputs:** this fire's `ailang mission ticket open --json` ([tickets.json](tickets.json)), `mission_decisions.sh --open` for the charter at `fleet/i109-record` ([open-decisions.tsv](open-decisions.tsv)), that charter, and the origin/dev delta since the last fleet landing (`47b123139..5ed4b3a0b`) with a diffstat over fleet-surface paths.

## Controller verification of the report

- **Draft PRs the judge cited but did not re-read:** the controller checked each one's state. #2065, #2067, #2082, #2108, #2110, #2141, #2143, #2150 and #2165 are all `OPEN`, draft. None was merged or closed.
- **Strongest objection (D-FLEET-43 A):** the controller read the row in the TSV. It says "Default if unanswered: B" and "built as a proposed diff only if you choo[se A]". The guardrail's idle-time build covers a row whose recommendation is code with no diff. Here, the row itself makes the build conditional on Mark's choice, so the objection does not survive. `tools/launchd/lib/idle-fingerprint.sh:23` still refuses `fleet|triage` (read at `5ed4b3a0b`). The judge had relied on the row for that line, and the row's premise still holds.
- **Counts:** the judge counted 27 tickets and 28 OPEN rows. Both match the controller's captures (`jq length` 27; `wc -l` 28; `--check` valid, 44 rows).

## Judge report (verbatim, as returned)

1. **Verdict: `NONE (idle)`, confidence 90%.** All 27 tickets are gated by an OPEN row or a scope boundary. Every OPEN row whose recommendation is code either has a draft diff already or has no in-scope first step I can build.

The row text, ticket evidence and PR searches show nothing buildable. `dev-delta.txt` shows no fleet-surface change other than the headless-runner skill deletion, which touches no ticket.

2. **Ticket to gate (27 tickets, 28 OPEN rows in the TSV, counted both with `grep -c .` and by listing IDs).**

| # | Ticket | Gate and gating phrase |
|---|---|---|
| 1 | reporting:daneel-reports-not-declared-triage | D-FLEET-22: "outside fleet scope. Who declares the inbox?", prod coordinator config |
| 2 | quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20: "Both fixes are outside fleet scope (`cmd/ailang/design_quorum.go`, billing)" |
| 3 | messaging:pubsub-notify-wrong-project | D-FLEET-19: "outside fleet scope", rig ADC quota project |
| 4 | ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18: "`tools/eval/`, which the scope guard refuses" |
| 5 | eval:motoko-connection-probe-orphaned | D-FLEET-18 (folded in at iteration 76) |
| 6 | policy:fleet-rulings-not-propagated-to-charters-and-skill | D-FLEET-25: "asks for a mechanism, so it needs your choice" |
| 7 | codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26: option A is already built as draft #2065 |
| 8 | quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32: ask 3 "is quota policy (HD-2a)" |
| 9 | pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33: "`.pi/extensions/**` change, so it parks per D-FLEET-3"; proposed diff exists in iteration66 evidence |
| 10 | skill:design-doc-creator-grep-portability | D-FLEET-36: "the skill is outside fleet scope. Widen... or fix it attended?"; guard arm is draft #2108 |
| 11 | skills:agents-copies-stale-non-mission | D-FLEET-38: "all outside fleet scope. Who syncs them?"; guard arm is draft #2108 |
| 12 | pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35: "Both files are outside fleet scope", `^402:` reclassification is policy |
| 13 | stall:watchdog:slow-on-hung-tool | D-FLEET-34: "Ask 2 is yours"; recommendation B waives it |
| 14 | crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23: "outside fleet scope. Who fixes it?" |
| 15 | sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37: "outside fleet scope. Who owns them?" |
| 16 | executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40: "all `internal/coordinator/**`, outside fleet scope" |
| 17 | executor:timeout-loses-work-and-may-never-start | D-FLEET-40 |
| 18 | coordinator:cloud-lane-capped-by-local-worktree-limit | D-FLEET-37 and D-FLEET-40 |
| 19 | executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39: "All fix paths... outside fleet scope" |
| 20 | coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 |
| 21 | executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 |
| 22 | ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 |
| 23 | quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31: recommendation A is "no further change"; B is seating policy (HD-2a); C is outside fleet scope |
| 24 | quorum:claude-p-ration-usage-429 | D-FLEET-28: "`design_quorum.go` is outside fleet scope"; changing `Blocked()` "is HD-2a" |
| 25 | ci:distinct-build-test-failures | D-FLEET-17: "Dev's REQUIRED checks... outside fleet scope... Who fixes it?" |
| 26 | tool-hang:triage-judge:home-metadata-search | D-FLEET-44: "later explicit design approval is still required before planning" |
| 27 | skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45: recommendation B is no change; "ticket stays open until you rule" |

OPEN rows with no ticket of their own are D-FLEET-16, -21, -27, -29, -41, -42 and -43. Together with the 21 rows above that gate tickets, that makes 28.

3. **OPEN rows whose recommendation is code, and whether an in-scope first step without a diff exists.**

| Row | Recommendation | Diff or reason it does not qualify |
|---|---|---|
| D-FLEET-16 | A: retire user-level skill symlinks | Attended step first; no fleet-built diff is possible |
| D-FLEET-17 | A: attended fix of check-git-exec and the docs MDX error | Outside fleet scope |
| D-FLEET-18 | A: scope-guard arm for two files | Diff exists, draft #2110 |
| D-FLEET-19 | A: attended `gcloud` command | Not repo code |
| D-FLEET-20 | A: route the OpenAI quorum seat through `codex exec` | Outside fleet scope; no in-scope first step |
| D-FLEET-21 | A: raise Gate 3b poll cap | Diff exists, draft #2067 |
| D-FLEET-22 | A: attended config change | Not buildable by the fleet |
| D-FLEET-23 | A: attended or V1 fix of Step 5b | Outside fleet scope |
| D-FLEET-25 | A: rulings ledger and doctor check, with the World fork-claim correction | Diffs exist, #2082 (fork claim) and #2165 (ledger and doctor) |
| D-FLEET-26 | A: add the capacity string to `TRANSIENT_SIG` | Diff exists, draft #2065 |
| D-FLEET-27 | A: keep luna allowed | Already shipped; default is A |
| D-FLEET-28 | A: attended or V1 change to the seat gate | Outside fleet scope, and HD-2a |
| D-FLEET-29 | A: attended `make quick-install` | Not a diff |
| D-FLEET-33 | A: land the `.pi/extensions/**` diff | Diff exists; parks per D-FLEET-3 |
| D-FLEET-36 and -38 | A: widen the scope guard | Diff exists, draft #2108 |
| D-FLEET-37, -39, -40 | A: coordinator fixes | Outside fleet scope |
| D-FLEET-41 | A: Dockerfile edit | Outside fleet scope (`docker/**`) |
| D-FLEET-41 | B: missing-toolchain rule in the sprint-evaluator skill | In scope with no diff, but it is a non-recommended alternative, not a step shared by all non-default options |
| D-FLEET-42 | A: per-lane secrets strip | Diffs exist, #2141, #2143 and #2150, covering M1 to M3 |
| D-FLEET-43 | A: lift the fleet idle-fingerprint exclusion | No PR found, but see section 4 |
| D-FLEET-44 | A: request a corrected design | Not code |

Recommendation is not code (default no change) for D-FLEET-31, -32, -34, -35, -45 (A or B is "no change" or "waive").

4. **Strongest objection to idling: D-FLEET-43 A.** It is the only recommended-code row that is in scope (`tools/launchd/**` plus Gate 5 text), has no diff, and is not policy class.

It does not survive the row text. The row says "built as a proposed diff only if you choose A", with default B. That makes the build itself conditional on Mark's choice, not just the merge. The iteration-94 judges reached the same conclusion, and `iteration93/pick-judge.md:78` notes that D-25 carries no such condition, unlike D-43. Building it would also pre-empt Mark's choice on a row he has been offered.

The runner-up is D-FLEET-25's shared World fork-claim correction, which both A and B begin with. It has no in-scope gap: draft #2082 already covers it, and #2165 covers the ledger and doctor check.

5. **Residual risks and what I could not verify.**
   - I did not open every PR body. The PR numbers come from `gh pr list` titles and the row text, not from reading each diff.
   - Whether `MISSION_NAME=fleet` is still refused at `idle-fingerprint.sh:23` was not re-read. I relied on the row.
   - The charter file in `/tmp` omits the log entries, so I read the iteration 92, 93 and 94 evidence in the repo for how D-FLEET-43 was treated before.
   - `gh pr list` showed the draft PRs as DRAFT but I did not check whether any has since been closed or merged. PRs #2065, #2067, #2082, #2108, #2110 and #2165 appeared as draft or merged in the listings above.
   - If Mark has answered any row since the TSV was captured, this verdict is stale. The brief says there are 0 new directives.
