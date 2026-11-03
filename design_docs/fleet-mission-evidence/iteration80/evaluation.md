# Iteration 80 evaluation: D-FLEET-21 option A (commit bdaad2825)
Verdict: PASS, 90/100 (no edits required; one optional nit).

Scores: spec completeness 28/30, correctness/verification 27/30, scope discipline 18/20, docs/changelog/gates 17/20.

## Findings
1. Option A complete. `git diff 29ddaec78 bdaad2825`: cap 3600, timeout 3800000 (2 mentions), comment corrected, TIMEOUT text 60m, chain bound 60-minute, background-only sentence 3600 s, SKILL.md rule 6 exception, both mirrors `cmp` identical.
   Residual grep over `.claude/skills/mission-control`: no remaining 1800/2000000/30-min wording referring to the CI poll in gate-3b, gate-5, ci-health, verification-protocol, ref-drift.
   gate-3-route.md:160 `2000000 ms for 1800 s` is CONFIRMED the codex executor wrapper's cap (lines 150-173, `/tmp/codex_run.sh`), not the CI poll. Correctly untouched.
   SKILL.md:497 "a 30-minute CI poller" is an illustrative aside in rule-7 prose; now slightly stale (poller is 60 min). Cosmetic nit, not blocking.
   design_docs/fleet-mission.md row 170 still says 1800 s, which is correct while OPEN.
2. Ceiling: the row itself states the tool allows 7200000 ms; 3800000 is below it. I did not independently test the tool limit. Margin: 200 s over 3600 s covers the 30 s sleep plus a `gh` call, so it is adequate. The loop checks the deadline after each ~30 s sleep, so worst case is ~3630 s plus API latency, well inside 3800 s.
3. SKILL.md is 628 lines (`wc -l`). `make check-context-docs` passes. `make check-changelog` passes (2 fragments valid). The fragment is correctly named, opens with `###`, and has no `#`/`##` headings.
4. CI honesty. I re-measured 200 completed CI runs (updatedAt-createdAt): median 15.2 min, p90 31.1, max 99.4, 23 over 30 min, 7 over 60. Last 40: p50 15.0, p90 18.5, max 30.2. "Median ~15 min, tail past 30 min" is honest. Note 7/200 runs exceed even 60 min, so 3600 s still parks about 3.5% of runs (mostly the 10-29 incident).
5. Mutation drills (worktree files restored; `git status --porcelain` empty; sync --check back to rc=0):
   - M1 revert only the .agents gate-3b copy to base: `sync-agents-skills.sh --check` rc=1 STALE; `test_agents_skills_sync.sh` rc=1 (4 passed, 1 failed). Killed.
   - M2 3600 -> 1800 in the .claude copy only: `--check` rc=1 and test rc=1 (both). Killed.
   - M3 remove the rule-6 exception from the .claude SKILL.md only: `--check` rc=1. Killed.
   Limitation: all three are killed only by the mirror-parity check. If both mirrors were reverted together, nothing mechanical kills it; only the residual grep would show it. No test pins the 3600 value itself. Acceptable for a prose skill.
6. `make test-launchd-drivers` at bdaad2825: `/tmp/fleet_i80_tld_bdaad28.rc` = `rc=0 for bdaad28`. MEASURED pass.
7. Behaviour beyond the time budget: none. Diff touches only numbers, timeout text, the comment, and the rule-6 parenthetical. Poll structure, SHA pinning, 30 s sleep, and park semantics are unchanged (the TIMEOUT message still says PARK).

## Strongest objection
The change only moves the cliff. 7 of 200 runs exceed 60 min, so a poll can still park a CI run that would later go green. It also lets a stuck headless iteration wait twice as long (up to 1 h of a 6 h slot). The comment "tail past 30 min" understates a tail reaching 99 min. These are time-budget tradeoffs for Mark's ruling, not defects in the diff. Option B (one auto re-poll) would handle the long tail better.

## Required edits
None. Optional: reword SKILL.md:497 "30-minute CI poller" to "a CI poller".
