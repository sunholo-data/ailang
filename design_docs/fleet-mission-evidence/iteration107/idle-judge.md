# Iteration 107 — independent idle-screen judge (verbatim)

Evaluator: `sonnet`, Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only. It got the inputs and the fleet admissibility rules, not the controller's conclusion. It was told to re-derive the ticket-to-row mapping first-party, read each mapped row in full, and say whether the new dev CI reading changed anything for `ci:distinct-build-test-failures`. Cost: 74,218 tokens, 8 tool uses, 45 s.

Inputs staged for the judge:
- this fire's `ailang mission ticket open --json` capture, and iteration 106's;
- the charter at the `fleet/i106-record` tip;
- the ledger block on that tip and on origin/dev;
- the OPEN-row TSVs from this fire and from iteration 106;
- iteration 106's log entry;
- the origin/dev delta since `b01b303e4`, with a harness-path diff;
- this fire's dev CI reading;
- the fleet admissibility rules from `gate-2-pick.md`.

Controller notes (not part of the judge's text):
- `open-decisions.tsv` in this directory came from this fire's `mission_decisions.sh --open` run against the `fleet/i106-record` tip. It is byte-identical to iteration 106's file (`cmp` rc 0). `tickets.json` is also byte-identical to iteration 106's (`cmp` rc 0). Iteration 105's capture (28 tickets) is the positive control: the diff can see a change.
- Controller check of the judge's strongest objection (D-FLEET-25 B): the row offers it to Mark as option **B**, recommends **A**, and says "Default if unanswered: C, and the ticket stays open." Building B would pre-empt the ask, so it parks and the verdict stands.
- Residual risk the judge named: it read the tails of D-FLEET-26, 36, 39 and 40 only through a `cut` window. Those four rows are byte-identical to iteration 106's ledger, and iteration 106's judge read them in full and found nothing buildable. That re-use rests on the byte-identity, not on a fresh full read.
- Directive count: the judge took 0 from the brief. The controller measured it with `scripts/mission_directives.sh` on #2172 (0 of 12 comments) and on #2039 (0 of 29).

---

**Verdict: NONE (idle)**

All 27 open tickets are gated by an OPEN ledger row or by a scope boundary. Nothing is both in scope and mechanical without a ruling from Mark.

## 1. Deltas, measured first-party

- **Tickets:** `cmp tickets_i107.json tickets_i106.json` returns rc 0, so they are byte-identical. Both hold 27 signatures, which matches i106's log.
- **OPEN decisions:** `diff open_decisions_i107.tsv open_decisions_i106.tsv` is empty. 28 OPEN rows: D-FLEET-16..23, 25..29, 31..45.
- **Ledger blocks:** the record tip and origin/dev differ in three ways.
  - D-FLEET-4 and D-FLEET-6 differ only in `planned/` → `implemented/…` path moves, with no status change.
  - D-FLEET-7 differs on the same line pattern.
  - D-FLEET-44 and D-FLEET-45 exist only on the record branch (not yet merged to dev), both OPEN.
  - No row flipped to RESOLVED.
- **dev delta:** two docs-only `finetune` commits (`fa616fe68`, `6d28e1ad5`). The harness-path diff is empty.
- **Directives:** 0 allowlisted, as stated in the brief. I did not re-measure this.

## 2. Per-ticket table

The "gating phrase" is quoted from the row text I read in `open_decisions_i107.tsv`. Every row below is OPEN.

| # | Ticket | Gating row | Gating phrase | Buildable |
|---|---|---|---|---|
| 1 | reporting:daneel-reports-not-declared-triage | D-FLEET-22 | "outside fleet scope. Who declares the inbox?"; the fix lives in the multivac cloud config and needs a coordinator revision roll | no |
| 2 | quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20 | "Both fixes are outside fleet scope (`cmd/ailang`, billing)"; A is a routing/billing change; default C | no |
| 3 | messaging:pubsub-notify-wrong-project | D-FLEET-19 | A is an ADC quota-project change on the rig; B is "V1's lane, outside the fleet allowlist" | no |
| 4 | ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18 | "sit in `tools/eval/`, which the scope guard refuses … Widen the authority?"; default C | no |
| 5 | eval:motoko-connection-probe-orphaned | D-FLEET-18 | Same `tools/eval/motoko_connection_probe.sh` allowlist ask. I mapped this one by the file it names, not by a literal signature hit. | no |
| 6 | policy:fleet-rulings-not-propagated… | D-FLEET-25 | "asks for a mechanism, so it needs your choice"; the mechanical sub-part is the world.toml fork claim, which is option B | no (see section 4) |
| 7 | codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26 | A is the `TRANSIENT_SIG` literal, offered to Mark, default C; ties to the D-FLEET-11/13 retry rulings | no |
| 8 | quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32 | "Changing it is quota policy (HD-2a)"; A (design) vs B (waive), default C | no |
| 9 | pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33 | "parks per D-FLEET-3. Approve the diff?" | no |
| 10 | skill:design-doc-creator-grep-portability | D-FLEET-36 | "the skill is outside fleet scope. Widen the fleet's scope …, or fix it attended?" | no |
| 11 | skills:agents-copies-stale-non-mission | D-FLEET-38 | "all outside fleet scope. Who syncs them?"; A widens `_scope_is_fleet_support`, default C | no |
| 12 | pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35 | "Both files are outside fleet scope", and reclassifying `^402:` is "D-FLEET-11/13 policy" | no |
| 13 | stall:watchdog:slow-on-hung-tool | D-FLEET-34 | "Ask 2 is yours"; it would re-open the D-FLEET-4/-7 thresholds; A vs B | no |
| 14 | crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23 | "outside fleet scope. Who fixes it?"; Step 5b is in `cmd/ailang/coordinator_cloud_task.go`; B would grant authority | no |
| 15 | sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37 | "in the coordinator … outside fleet scope. Who owns them?" | no |
| 16 | executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40 (and D-FLEET-23) | "Unbuilt, all `internal/coordinator/**`, outside fleet scope" | no |
| 17 | executor:timeout-loses-work-and-may-never-start | D-FLEET-40 | Same row, ask (e) | no |
| 18 | coordinator:cloud-lane-capped-by-local-worktree-limit | D-FLEET-37 / D-FLEET-40 | The coordinator is "the shared task plane for every agent, not the mission loop harness" | no |
| 19 | executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39 | "All fix paths … are outside fleet scope" | no |
| 20 | coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 | Same row, the "adjacent" ticket | no |
| 21 | executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 | Same row | no |
| 22 | ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 class | Charter backlog item 19: "Outside fleet scope until D-FLEET-18 is ruled" | no |
| 23 | quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31 | "The waste itself is policy"; B is seating policy (HD-2a); C is outside scope; default A (no change) | no |
| 24 | quorum:claude-p-ration-usage-429 | D-FLEET-28 | "`design_quorum.go` is outside fleet scope. Changing `Blocked()` … is HD-2a" | no |
| 25 | ci:distinct-build-test-failures | D-FLEET-17 class | Charter backlog item 44: reds "all in … packages … outside fleet authority"; D-FLEET-17 asks "Who fixes it?" | no (see section 3) |
| 26 | tool-hang:triage-judge:home-metadata-search | D-FLEET-44 | "ticket stays open … Default if unanswered: B immediately; no planning/execution/merge" | no |
| 27 | skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45 | "This asks because the design failed its gate"; default B (no change) | no |

Mapping differences from my first-party read:
- **Tickets 4 and 5:** I mapped them to D-FLEET-18 by file path. I also read D-FLEET-18 in full.
- **Tickets 22 and 26:** these have no literal ledger-row signature hit. They are gated by charter backlog items 19 and 61, plus D-FLEET-18 and D-FLEET-44 respectively.
- **Ticket 8:** a RESOLVED row (D-FLEET-12) also names it, but D-FLEET-32 is the gating row.

## 3. The CI observation

It does not change ticket 25 or its row. Three checks:

1. **Scope.** The new failing test, `TestRuntimeNestedStopUsesRemainingDeadline`, is in `internal/effects/process_runtime_lifetime_test.go`. The scope guard's `_scope_is_core` lists `internal/effects/*` as language-core. The other failing tests in the last six runs are in `cmd/ailang` (worker CLI lifecycle), `internal/effects` and `internal/runner`. None is a harness path. Charter item 44 already says the same of earlier reds.
2. **Which job.** The failing leg is "Build macos-latest", one of two macOS legs. The CI note does not say whether it is a required check. D-FLEET-17 is about required-check reds, which are `test` and `docs-gate`, so this flake is adjacent to its row rather than inside it.
3. **Authority.** D-FLEET-17 is OPEN and asks "Who fixes it?". Option B, a standing authority for the fleet to fix a REQUIRED-check red on dev when the owner's kill switch is set, is exactly what Mark must grant. The V1 kill switch being present matches the row's premise. The note adds one more instance of that class and gives no new authority.

Four reds in the last six runs make the flake rate worth Mark's attention. But the ticket is unchanged (`slots_lost` is still 1) and the CI note is observation, not a ticket, so building a fix would be self-sourcing.

## 4. Strongest objection to my own verdict

The closest candidate is **D-FLEET-25 option B: correct the World "fork" claim in `missions/world.toml:3` and `internal/mission/registry.go:70`.**
- The row calls it "mechanical" and "one factual sub-item still wrong". Both paths are harness paths in `_scope_is_harness`.
- Against it: B is one of three options offered to Mark, with A recommended and a default of C. Building B pre-empts the ask.
- A would also correct the same claim ("The fleet also corrects the World fork claim … and the doctor note"), so the sub-fix is not separable from the option set.
- The brief says "a mechanical sub-part the row itself does not ask Mark about" is buildable. This one sits inside what the row asks Mark to choose.
- I did not verify the plist-versus-registry claim myself. If the row's factual claim is wrong, nothing is buildable on that front either.

Second-closest: D-FLEET-26 A, the `TRANSIENT_SIG` literal (draft #2065). It is the option offered to Mark with default C, so the same reasoning applies.

## 5. Confidence and residual risk

**Confidence: 90%** in NONE.

Residual risk in my method:
- **Row text.** I read D-FLEET-23, 33, 34, 35, 37, 38, 44, 45, 17, 18, 19, 20, 22, 25, 28, 31 and 32 to the end of the row. I read the tails of 26 (past about 2500 characters), 36, 39 and 40 only through the `cut` window. Their leads and ask structure were clear and their defaults are all C or "stays open". A buildable sub-part in a truncated tail is possible but unlikely.
- **Charter items.** I took the gating of tickets 22 and 26 from charter backlog items 19 and 61 via `grep`, not a full read.
- **Directives.** The count of 0 comes from the brief, not from my own measurement.
- **Self-sourcing.** My answer assumes the rule that building a ledger option without a ticket or ruling counts as self-sourcing.
