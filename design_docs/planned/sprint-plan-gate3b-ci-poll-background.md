# Sprint Plan: Gate 3b CI poll background discipline

**Sprint:** M-FLEET-GATE3B-POLL-BACKGROUND  
**Status:** planned  
**Created:** 2026-10-16  
**Verified at:** c4fab92c29ab7e208dc96325a925f053cf0417a9  
**Ticket:** `skill:gate3b-ci-poll-exceeds-foreground-bash` (motoko, 2026-10-01)  
**Design doc:** none (`design_doc: null`). Fleet charter `design_docs/fleet-mission.md:69-70` designs only when warranted; this is a mechanical documentation repair.  
**Planner-Lane:** codex-ok  
**Duration:** 0.125 working day, approximately 60 minutes including buffer; one executor run.  
**Risk:** low; incorrect launch or artifact instructions could still kill or misattribute CI work.

## Verified premises and reuse

The worktree was clean before planning. HEAD equals the controller's supplied full SHA. No implementation was started; the unrelated active M-DOCS-MISSION-WORK-ITEM-GUIDE sprint stays untouched.

| Premise | Evidence at verified HEAD |
|---|---|
| The poll is SHA-pinned and checks base drift | `.claude/skills/mission-control/resources/gate-3b-ci-green.md:18-31`; same lines in `.agents/skills/mission-control/resources/gate-3b-ci-green.md` |
| Thirty-minute poll cannot fit a foreground Bash call | Both copies, lines 33-38: `+ 1800`, `sleep 30`; lines 16-40 have no background launch parameters. Whole-file search finds no `run_in_background`, 600 s foreground-cap explanation, or Bash tool `timeout` parameter. Line 11 mentions the unrelated GNU timeout utility. |
| Both copies must be edited together | `design_docs/fleet-mission.md:152` (D-FLEET-8), applied to this ticket by the controller. `stat -f '%i %N'` returned distinct inodes 184011609 and 184010883; `cmp` exited 0. |
| Existing timeout discipline covers this defect | `.claude/skills/mission-control/resources/gate-3-route.md:143-156`, codex recipe step 2: background call, explicit tool timeout plus margin, 2000000 ms example, 600 s foreground ceiling, and the failed 580 s / 30 s sleep example from fleet iterations 35/36. |
| Active waits and subject attribution already exist | `.claude/skills/mission-control/SKILL.md:322-350` (standing rule 7), `:434-443` (notification versus completion; no double background), and `:506-515` (unique artifact, subject SHA, kill superseded poller). |
| Unbounded waiting has a measured cost | Both Gate 3b copies, lines 6-11: iteration 13 wedged for 4 h before the driver watchdog reclaimed it. |
| Changelog convention | `changelogs/unreleased/README.md:3-11,18-22`: dated fragment, opening with `### `; use the required `### Fixed — ` heading. |

**Registry reuse (M1): none.** No package-like capability is added, so registry searches and package inspection are inapplicable. Reuse `gate-3-route.md` codex recipe step 2 and `SKILL.md` standing rule 7 by reference; introduce no package, helper script, copied rule block, or new process mechanism.

**Velocity basis:** `git show --stat c55ca4398` (2026-10-05, mirrored heartbeat documentation repair) changed 14 resource files with 34 insertions and 20 deletions. Recent edits to this Gate 3b file include `3ee5bb177` (+42 lines) and `0b7f3e3af` (+15/-1). These establish comparable change sizes, not measured engineering hours. Budget 25 added lines per copy plus a 5-line fragment, 55 LOC total and zero unit-test LOC; nominal capacity 440 LOC/day for 0.125 day is a planning estimate, not a claimed historical rate.

## Files to Modify

- `.claude/skills/mission-control/resources/gate-3b-ci-green.md`
- `.agents/skills/mission-control/resources/gate-3b-ci-green.md`
- `changelogs/unreleased/2026-10-16-gate3b-ci-poll-background.md`

This is the entire executor write scope. The two planning artifacts are planner outputs, not additional executor edit targets. Add about 15-30 lines per resource copy, with two existing verdict-line replacements and only the minimal snippet adjustments needed for its artifact. Add the single changelog fragment beginning `### Fixed — `. No routing, lane order, quota/ration thresholds, billing guards, CI workflow, SKILL.md, other resource, test, or example changes. No `.ail` work or teaching-prompt gate is needed.

## M1 — Make the existing poll runnable within the harness limits

**Estimated:** 55 documentation LOC; dependencies: none. Preserve the existing full-SHA target, mission-base recording/drift checks and their abort behavior, SHA-filtered run selection, 30-minute overall cap, 30-second poll cadence, and PARK on overall timeout. A completed failure remains a failure; completion must never be described as CI success by itself.

### Required edit contract

1. Insert a short launch note adjacent to the existing snippet explaining both independent limits: the Bash foreground ceiling is **600 s**, and the poll must retain its **1800 s** cap because iteration 13's unbounded wait wedged for **4 h**. Name and link `gate-3-route.md` (codex recipe step 2) and `SKILL.md` standing rule 7; reference their amendments rather than restating their history or reproducing whole rules.
2. Include this canonical operative parameter line for an unambiguous, checkable default:

   `Bash poll call: run_in_background: true, timeout: 2000000 (ms).`

   These are **Bash tool arguments**, not shell variables or a GNU timeout command. The tool timeout must be at least 2000000 ms, exceeding 1800 s plus margin. Run the poll loop in the **foreground inside that background tool call**, with **no extra shell `&`**; its completion notification and the work's completion are then one event. Keep the existing polling fence as the first bash fence so the evaluator can extract it precisely.
3. After pinning the target, allocate one unique terminal-verdict file per invocation whose path includes iteration and target SHA, for example via `mktemp "/tmp/ci_iter${iteration}_head${target:0:7}.XXXXXX"`. Record the resulting absolute path for subsequent tool calls. If setup and poll use separate Bash invocations, explicitly carry the pinned target, artifact path and deadline across that boundary; never re-resolve the target to a newer HEAD. No fixed shared filename. Only terminal results go in this file; progress and launch output must not make it look complete.
4. Change **both** existing echo verdicts to name the full target and write the verdict file. Preserve the surrounding loop's completion and timeout behavior; use these literal echo payloads/redirections so the mutation drills have concrete assertions:

   ```bash
   echo "CI for $target: $st" > "$verdict"
   echo "Gate 3b TIMEOUT for $target after 30m (status=$st) — PARK, do not hang" > "$verdict"
   ```

   Leave the echo commands in their existing case/deadline branches. A missing verdict after tool completion is missing evidence, never green. As standing rule 7 prescribes, kill a poller whose subject is superseded and prove it stopped before any reuse; a fresh poll gets a fresh path.
5. Show how the controller stays active: chained **bounded foreground reads** of that exact verdict file, each comfortably under 600 s. A compact read pattern is `read_until=$(( $(date +%s) + 45 )); while [ ! -s "$verdict" ] && [ "$(date +%s)" -lt "$read_until" ]; do sleep 5; done; cat "$verdict"`, called with foreground tool `timeout: 60000`. Explicitly pass the recorded verdict path into each call. Empty file means perform the next bounded read **in the same controller turn**, while also observing the background task's completion; never end the turn with work outstanding. Terminal content must name the expected SHA. On completion without a verdict or tool failure, report/park rather than spinning indefinitely. The poll's overall deadline bounds the chain; refer to rule 7 for its full lifecycle discipline.
6. Give the allowed foreground alternative in a few lines: foreground tool `timeout: 600000`, per-call `deadline = min(now + 540, overall_deadline)` seconds (thus **<= 540 s** from call start), retaining room for a 30-second sleep and margin. Establish `overall_deadline = first_start + 1800` **once** and explicitly carry that absolute deadline, target SHA, selected run and artifact path into every call. Slice expiry means re-poll immediately in-turn with those same values; it is not a terminal result. Never restart the cumulative 30-minute clock. At overall expiry write the SHA-bearing timeout verdict and PARK. If a tool interruption occurs, the carried absolute deadline still governs the next call. Do not use the known-bad 580 s slice.
7. Keep the addition proportionate: approximately 15-30 added lines per copy, including launch/artifact/active-read and foreground-alternative guidance. Mirror the finished resource byte-for-byte and write the one short changelog fragment linking this plan and describing the foreground ceiling repair.

### Day 1 execution order

- 0-10 minutes: recheck HEAD, clean scope and cited prerequisites; inspect the existing snippet and rule references.
- 10-30 minutes: make the compact documentation edit, update the two echo verdicts and artifact handling, mirror the other copy, add the fragment.
- 30-45 minutes: inspect rendered/text instructions, run the eight local gates and `cmp`, perform four temporary-text mutation drills.
- 45-60 minutes: buffer and controller review/real-CI verification. Network and controller-only checks are recorded separately; they cannot be satisfied by sandbox guesses. No unit test file is warranted for this prose-only change.

## Acceptance and validation

M1 is complete only when the following evidence is available:

1. Both resource copies are byte-identical; diff shows approximately 15-30 added lines each and no executor changes outside the three allowed paths. The changelog starts with `### Fixed — `.
2. Operative text satisfies every edit contract above: correct background tool arguments, no inner `&`, unique SHA-named terminal artifact, both SHA-bearing verdicts, bounded active foreground reads, and a <=540 s foreground alternative carrying the original cumulative cap. References point to existing discipline; target/drift/cap/PARK semantics remain intact.
3. All eight focused static gates below return rc 0 on the edited tree. A base-red gate is reported with evidence, not repaired or bypassed by this sprint. No full `make test` is required for this documentation-only scope.
4. All four mutation drills distinguish pristine text from the named mutant, with exact command/output/rc recorded by the evaluator. Review both branches of the snippet, not just presence of a prose keyword.
5. **CONTROLLER ONLY, outside the sandbox:** run `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR make test-launchd-drivers`; record actual rc/output. The executor does not run this gate in its restricted sandbox.
6. **Behavioural acceptance, controller outside the sandbox:** run the edited snippet for real against a real `dev` CI run, with `run_in_background: true` and `timeout: 2000000`, and no nested `&`. Preserve the SHA pin and base checks, keep the turn active through bounded foreground reads, and observe the background task finish. Bank target full SHA, run ID, exact launch parameters, unique verdict path, final contents naming that SHA, notification and read evidence, and resulting CI conclusion/PARK disposition. A fixture, grep, or already-written file is insufficient; a run longer than 600 s provides direct evidence of crossing the foreground ceiling. The controller must actually execute the edited body, not a parallel replacement recipe.
7. Sandboxed checks requiring network, sockets or `gh`, including any attempted real-CI probe, are **UNINFORMATIVE UNDER SANDBOX**, never pass or fail. Controller's prior host result is provenance only, not an executor success. Missing controller evidence remains pending for acceptance. Independent sprint-evaluator reviews scope, prose and mutation results; executor does not self-approve.

Run each static target separately and record its rc:

```bash
make check-skills
make check-context-docs
make check-referenced-paths
make check-changelog
make check-file-sizes
make check-doc-review
make check-no-personal-email
make check-tmpfile-hygiene
cmp .claude/skills/mission-control/resources/gate-3b-ci-green.md .agents/skills/mission-control/resources/gate-3b-ci-green.md
```

**Planning baseline:** controller reported all eight gates and the scrubbed host driver suite rc 0 at this exact HEAD. Planner independently ran all eight local gates, starting from the clean base: each returned rc 0. The doc-review gate warned about missing reviewBy on reserved-keywords.md and roadmap/index.md (0 malformed dates); these warnings are not red gates. `cmp` independently returned 0. Host driver suite and real network CI behaviour are not verified by the planner.

## Evaluator mutation drills

Work on disposable text copies outside the repository; never mutate the actual skill or write an extra tracked test file. Set `f` to the edited resource (then to each disposable mutant). Extract the **first** bash fence as `body` with this read-only command, saving the output only in a shell variable:

```bash
body=$(awk '/^```bash$/ {if (!seen++) {inside=1; next}} inside && /^```$/ {exit} inside {print}' "$f")
```

For each drill, first record the pristine control, change only the named text in a temporary copy, re-extract `body` when necessary, and record the changed result. Repeat the structural review on the mirror after `cmp`.

| Mutation | Distinguishing grep command | Pristine / mutant |
|---|---|---|
| Drop `run_in_background: true` from the operative Bash poll call line, leaving rule references intact | `grep -nE '^Bash poll call: run_in_background: true, timeout: 2000000 \(ms\)\.$' "$f"` | rc 0 / rc 1; an unrelated prose mention cannot satisfy the anchored line |
| Set operative timeout to 1700000 ms (below the 1800 s poll cap) | `grep -nE '^Bash poll call: run_in_background: true, timeout: 2000000 \(ms\)\.$' "$f"` | rc 0 / rc 1; inspect the operative tool argument, not a reference example |
| Append a shell `&` to the poll loop's `done` inside the background call | `printf '%s\n' "$body" \| grep -nE '(^\|[^&])&([[:space:]]\|$)'` | grep rc 1 / rc 0; expected absence in control, presence in mutant. In the executable command below the table, pipes are literal shell operators. |
| Remove `$target` from either terminal echo payload | `printf '%s\n' "$body" \| grep -Ec 'echo "(CI for \$target: \$st\|Gate 3b TIMEOUT for \$target after 30m .*)" > "\$verdict"'` | count 2 / count 1 (0 if removed from both); assert count exactly 2 |

Executable forms for the last two checks (without Markdown table pipe escaping):

```bash
printf '%s\n' "$body" | grep -nE '(^|[^&])&([[:space:]]|$)'
printf '%s\n' "$body" | grep -Ec 'echo "(CI for \$target: \$st|Gate 3b TIMEOUT for \$target after 30m .*)" > "\$verdict"'
```

These are text mutation drills, not runtime proofs. A careful reader must additionally verify that the matched parameter line governs the displayed snippet, the matched echoes are its actual terminal branches, and both write to the recorded per-invocation artifact.

## Handoff boundaries and unresolved premises

Only this plan and its sprint JSON are produced during planning. No skill edits, message dispatch, git add/commit/stash/checkout/branch, or executor launch. The controller owns commits, host checks, real-CI acceptance and the next stage. The single milestone is deliberate for this one mechanical ticket, overriding the general planner template's two-milestone minimum. No design, routing or policy decision remains open. Live CI availability and actual host-suite behaviour remain unverified here.
