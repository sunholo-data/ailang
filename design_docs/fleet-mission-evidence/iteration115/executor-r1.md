# Executor report, fleet iteration 115: codex capacity stop is transient (D-FLEET-26 A, M1)

Worktree: `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter79-dfleet26`, branch `fleet/i79-dfleet26-capacity-transient`. Not pushed. No PR edits.

## Commits (on top of base d84004daa)

| SHA | Subject |
|---|---|
| 18b0192d3 | fix(launchd): codex daemon-form capacity stop is transient (D-FLEET-26 A, F1) |
| 688a3701a | docs(changelog): codex capacity retry fragment, both renderings (D-FLEET-26 A, F2) |
| e25a265ae | docs(plan): sprint plan for the codex capacity stop (D-FLEET-26 A) |

## Suite: `/bin/bash tools/launchd/test_controller_capacity.sh`

- Baseline at d84004daa, unmodified: 19/19 PASS, rc 0.
- After adding the tests, against the draft driver (red): 20 pass, 1 fail, rc 1. The failing arm is `codex-daemon-capacity-is-transient` with `process rc 0 (got 1)`, `same rung retried`, `retry log line`, `verdict COMPLETED`, `no FAILED/PAUSED sends`. The other new arms pass on the draft, as expected: `codex-daemon-other-error-not-transient` and the extended prose arm, because the draft never matches daemon form at all.
- After F1 (green): 21/21 PASS, rc 0. A final run on the committed head also gave 21/21, rc 0.

## Daemon fixture bytes

The `_d400` fixture was checked against the captured payload from `/tmp/ailang-mission-triage.log` line 37559. `cmp` reports byte-identical after genericising the model name (`secrets-scope-nonexistent-model` becomes `some-nonexistent-model`).

## Mutation drill (scratch copies in `/tmp/fleet_i115_mut/`, `MC_CAPACITY_DRIVER` pointed at each; real driver untouched)

Each mutant passed `bash -n` before its suite run. Edits are to the F1 alternative `|^(ERROR|Error: Codex daemon turn failed): Selected model is at capacity` in `TRANSIENT_SIG`.

| Mutant | Edit | rc | Failing arms |
|---|---|---|---|
| M1 drop `^` | `\|(ERROR\|Error: Codex daemon turn failed): Selected model is at capacity` | 1 | prose-codex-capacity-not-transient |
| M2 drop alternative (revert) | alternative removed | 1 | codex-capacity-is-transient, codex-daemon-capacity-is-transient |
| M3 broaden to `^ERROR:` | `\|^ERROR:\|^Error: Codex daemon turn failed: Selected model is at capacity` | 1 | codex-other-error-not-transient, codex-daemon-other-error-not-transient |
| M4 unanchored literal | `\|Selected model is at capacity` | 1 | prose-codex-capacity-not-transient |
| M5 `^ERROR: .*capacity` | `\|^ERROR: .*capacity\|^Error: Codex daemon turn failed: Selected model is at capacity` | **0 (21/21, SURVIVES)** | none |
| M6 literal in RUNTIME_QUOTA_SIG (option B) | alternative removed from TRANSIENT_SIG, `\|Selected model is at capacity` appended to RUNTIME_QUOTA_SIG default | 1 | codex-capacity-is-transient, codex-daemon-capacity-is-transient, prose-codex-capacity-not-transient |
| M7 drop daemon branch (= draft) | `\|^ERROR: Selected model is at capacity` | 1 | codex-daemon-capacity-is-transient |
| M8 broaden both prefixes | `\|^(ERROR\|Error: Codex daemon turn failed):` | 1 | codex-other-error-not-transient, codex-daemon-other-error-not-transient |
| M9 broaden daemon prefix only | `\|^ERROR: Selected model is at capacity\|^Error: Codex daemon turn failed:` | 1 | codex-daemon-other-error-not-transient (only killer) |

M5 is the accepted survivor. It widens only to codex's sibling capacity strings (`Flex capacity unavailable.`, `Policy-restricted capacity…`), which is a ruling-scope question. It should be filed as a follow-up note, not an arm.

Raw outputs: `/tmp/fleet_i115/mutation_drill.txt`, and per-mutant suite output in `/tmp/fleet_i115_mut/*.out`.

## Checks (each unpiped, rc shown)

| Check | rc | Result |
|---|---|---|
| `bash -n tools/launchd/mission-control.sh` | 0 | ok |
| `git diff --check origin/dev...HEAD` | 0 | ok |
| `make check-changelog` | 0 | 13 fragments valid |
| Column-0 hygiene on the diff: `grep -cE '^\+(ERROR\|Error: Codex daemon turn failed): Selected model'` | count 0 (grep rc 1 = no match) | 0 |
| Same pattern on a `printf` positive line (control) | count 1 (rc 0) | control holds |
| Column-0 hygiene per touched file (driver, suite, changelog, plan) | n/a | 0 in each |
| `make check-tmpfile-hygiene` | 0 | ok |
| `make check-home-isolation` | 0 | ok |
| `tools/launchd/test_controller_capacity.sh` (final) | 0 | 21/21 |
| shellcheck `-S warning` on driver, base vs head | n/a | 14 vs 14 (no new finding) |

## Deviations from the plan

1. **First mutation drill discarded.** My mutator used `${s/"$OLD"/"$new"}`. On macOS `/bin/bash` 3.2, quotes in the replacement are kept literally, which broke M1's `TRANSIENT_SIG` line (syntax error). The suite then "killed" M1 spuriously. I stopped that run and discarded its output. I switched to prefix/suffix splicing and added a `bash -n` gate per mutant. Only the second, clean run is reported above.
2. **M3 is also killed by the daemon other-error arm.** The grep runs with `-i`, so `^ERROR:` also matches the daemon-form `Error:` 400 line. The plan predicted only the exec arm.
3. **M6 is killed by three arms,** not one. The mid-line quotes match via `RUNTIME_QUOTA_SIG`, so the prose arm fails too.
4. **M5, M8, M9 interpretation.** The plan gives M5, M8 and M9 as fragments. I applied each to the F1 alternative (which keeps the other branch intact) and recorded the exact replacement above. The plan does not fix the exact text, so this is my reading.
5. **Not run, by brief:** `make test-launchd-drivers` and the healthy/degraded driver dry-runs. Those are left for the controller, per the brief. The plan's dry-run and gate evidence (`design_docs/fleet-mission-evidence/iteration<N>/gates.md`) is not banked by me.
6. **Shellcheck method.** Plan §3 cites "30 at iter 91" from a method I can't reproduce here. I measured base and head with the same `-S warning` command, which gives 14 and 14. Only the equality is claimed.
7. **Changelog wording.** I kept the full proposed-to-ruled language out and quoted both renderings mid-line. The F4 residual (a codex retry starts a fresh daemon thread) is named in the fragment as a known limit.

## Not done (out of scope)

- F3, the ledger record and Go-side pin test follow-up: outside fleet write scope; the iteration record should note it.
- Push, PR #2065 update, and the landing sequence: not in the brief.
