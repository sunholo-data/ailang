# Sprint Plan: M-MISSION-HEARTBEAT-DRIVER-ROOT (Rev 5.2)

**Design doc:** [`m-mission-heartbeat-driver-root.md`](m-mission-heartbeat-driver-root.md), Revision 5.2 (commit `11a517b33`)
**Rulings:** D-FLEET-8 = YES (fix both `.claude` and `.agents`, rc-19 in scope) · D-FLEET-10 = A (absoluteness guard) · D-FLEET-2 (rc-19 falls back along the declared chain)
**Ticket:** P0 `skill:heartbeat-relative-path-absent-in-world` · fleet iteration 16 · planned 2026-10-03
**Planner-Lane:** codex-ok (the doc's own tag; the change is mechanical text plus one shell-guard arm)
**Supersedes:** the untracked 2026-09-29 Revision-1 plan in the pin checkout. Do not reuse it.

## Summary

| | |
|---|---|
| Goal | Every mission-control heartbeat stamp resolves its helper through an absolute `$MISSION_DRIVER_ROOT`, so World and Stapledon fires stop failing every gate stamp with rc=127. Both skill trees get the fix, gate-3 names rc 19, and the push guard admits the `.agents` mirror. |
| Milestones | **1** (M1). The doc says "one short milestone" and the work cannot be split usefully. |
| Duration | 0.5 day. About 1h of edits, about 1h of drills, plus a 6.5 min suite run (baseline measured: 382 s). |
| LOC | About 45 changed lines: 14 command lines in 7 hand-edited files (`.agents` copies are generated), 7 setup sentences, 1 rc line, 1 guard arm, 2 Authority lines, about 12 changelog lines. No new code or tests. |
| Risk | Low. The text is content-pinned, every runtime behaviour was drilled in the Verification Log, and the suite baseline is green. |

## Premise check at HEAD (planner, 2026-10-03, worktree `fleet-i16` @ `11a517b33`)

| Doc claim | Command → observed | Holds? |
|---|---|---|
| 18 heartbeat command sites, 9 per tree, at the listed lines; `SKILL.md:338` is prose | `rg -n 'mission-heartbeat\.sh' .claude/skills .agents/skills` → gate-0 :3 and :4, gate-1 :3, gate-2 :89, gate-3 :3, gate-3b :3, gate-4 :61, gate-5 :3 and :5 in each tree, plus `SKILL.md:338` prose in each | yes |
| Sync script exists; `--check` mode | `tools/launchd/sync-agents-skills.sh` takes no flag (copy, `rsync -a --delete` over 6 skills: mission-control, mission-brief, mission-loop-change, sprint-planner, sprint-executor, sprint-evaluator) or `--check` (`diff -rq`, exit 1 on drift). No other flags. `--check` today → `agents skills in sync (...)`, rc=0 | yes |
| Both test scripts exist and are wired | `test_agents_skills_sync.sh` (make/test.mk:82), `test_mission_heartbeat.sh` (make/test.mk:86), both in `test-launchd-drivers` | yes |
| `_scope_is_harness` has no `.agents` arm; "Keep in step" comment | `tools/launchd/githooks/pre-push:14-22`: line 17 `.claude/skills/mission-*\|.claude/skills/sprint-*\|.pi/extensions/*) return 0 ;;`; lines 12–13 "Keep in step with design_docs/fleet-mission.md "Authority"". Functions can be sourced with `MISSION_SCOPE_LIB=1` (last line) | yes |
| Authority "May change" lists `.claude/skills/mission-*/**` and no `.agents` | `design_docs/fleet-mission.md:75-81` | yes |
| rc comment content (gate-3-route.md:390, both copies) | `#    13=wall_timeout · 14=launch_failed · 18=tool_hang.  Anything non-zero except 18` | yes |
| rc 19 upstream | `scripts/mission_pi_run.sh:71` `#   19 provider_quota`, `:407` `VERDICT_NAME="provider_quota"; RC=19` | yes |
| Three conflict-surface sites | `rg -n MISSION_DRIVER_ROOT … \| grep -v mission-heartbeat` → gate-3-route.md:72, :618, role-spawn-routing.md:85 in each tree | yes |
| `mission-control.sh:47` and `:2167` | `:47` `MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd)`; `:2167` `MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT` | yes |
| `pin-root.sh:362` execs the absolute pin path | **There is no `tools/launchd/pin-root.sh`.** The file is `tools/launchd/lib/pin-root.sh`. Its line 362 is `exec /bin/bash "$wt/tools/launchd/$script" "$@"` as the doc quotes it | **path wrong, content right (D1)** |
| Baseline suite | `make test-launchd-drivers` (unpiped, output to a file) → rc=0, 382 s | yes |

### Doc-vs-HEAD discrepancies

- **D1. Path.** Acceptance 7 and the Verification Log cite `pin-root.sh:362`. The real path is
  `tools/launchd/lib/pin-root.sh:362`, and the line content matches. This plan uses the real path.
  It is a citation error with no behavioural effect.
- **D2. Evaluator path list.** Acceptance 8 says the evaluator "rejects any other path" beyond
  "14 gate files, guard, fleet-mission.md". The mission brief requires a changelog fragment
  (`changelogs/unreleased/2026-10-03-heartbeat-driver-root.md`), and the coding standards require
  one for every change. Executor bookkeeping also updates the sprint JSON. Acceptance 8's literal
  list would reject both. **This plan widens the allowed diff to those two paths and nothing else**
  (see "Allowed diff"). The evaluator should apply that list, not the doc's literal three-item list.
- **D3. Side effect the doc does not state (an observation, not an error).** `_scope_verdict` refuses
  any `_scope_is_harness` path to *product* missions (v1, docs, motoko). Today a product loop can
  push `.agents/skills/mission-*` and `.agents/skills/sprint-*`. After the new arm it cannot. This
  follows from the intent ("mirror paths of the already-allowed skills") and matches how `.claude`
  is treated, so the plan accepts it. The changelog should name it.
- **D4. No regression row (an observation).** `tools/launchd/test_mission_scope_guard.sh` (in the
  suite at make/test.mk:94) has no `.agents` case. The doc keeps Acceptance 5 as an ad-hoc check
  and does not list that test file in its diff, so this plan does **not** edit it. Follow-up: add
  `check fleet .agents/skills/mission-control/… allow`, `check fleet .agents/skills/model-manager/SKILL.md refuse` and
  `check v1 .agents/skills/sprint-planner/SKILL.md refuse` rows under `skill:driver-root-guard-unify` or its own ticket.

## Constraints (binding on the executor and the evaluator)

1. **Hand-edit only the seven `.claude/skills/mission-control/resources/gate-*.md` files**
   (gate-0-preflight, gate-1-observe, gate-2-pick, gate-3-route, gate-3b-ci-green, gate-4-record,
   gate-5-retro). Produce the seven `.agents/skills/mission-control/resources/` copies **only** by
   running `tools/launchd/sync-agents-skills.sh`. Never hand-edit `.agents`.
2. **Also change** `tools/launchd/githooks/pre-push` (add the `.agents/skills/mission-*` and
   `.agents/skills/sprint-*` arms to `_scope_is_harness`) and the Authority "May change" allowlist in
   `design_docs/fleet-mission.md` (add `.agents/skills/mission-*/**` and `.agents/skills/sprint-*/**`).
3. **Add** the changelog fragment `changelogs/unreleased/2026-10-03-heartbeat-driver-root.md`, opening
   with a `### Fixed — …` heading.
4. **Do NOT touch the three conflict-surface sites:** `gate-3-route.md:72` (bare lane-dead),
   `gate-3-route.md:618` (`${MISSION_DRIVER_ROOT:-.}` worktree), and `role-spawn-routing.md:85`
   (bare lane-dead), in either tree. Those belong to follow-up ticket `skill:driver-root-guard-unify`.
5. **Done-gate:** `make test-launchd-drivers` must be green. Run it **unpiped**
   (`make test-launchd-drivers > "$T/suite.log" 2>&1; echo rc=$?`), never `| tail`, so the exit code
   is make's own. It takes about 6.5 min, so bound any background poll with a `date +%s` deadline of 15 min or more.
6. **Drills use a temporary `AILANG_STATE_DIR`** (`T=$(mktemp -d)`), never the real `~/.ailang/state`.
   Afterwards, `ls ~/.ailang/state | grep -c i16-drill` → `0`.
7. No new helper, env var, state format, label or heartbeat semantics. `mission-heartbeat.sh` and
   `mission-control.sh` are not edited.

### Allowed diff (evaluator rejects anything else)

```
.claude/skills/mission-control/resources/gate-{0-preflight,1-observe,2-pick,3-route,3b-ci-green,4-record,5-retro}.md   (7, hand-edited)
.agents/skills/mission-control/resources/gate-{…same 7…}.md                                                            (7, sync output)
tools/launchd/githooks/pre-push
design_docs/fleet-mission.md
changelogs/unreleased/2026-10-03-heartbeat-driver-root.md
.ailang/state/sprints/sprint_M-MISSION-HEARTBEAT-DRIVER-ROOT.json   (bookkeeping only)
```

Check with `git diff --name-only <base>..HEAD`. It should list 17 paths, or 18 with the JSON.
Also check `git diff <base>..HEAD -U0 -- '*gate-3-route.md' '*role-spawn-routing.md' | grep -E '^[-+].*(mission-lane-dead|mission-worktree)'` → no output.

## Milestone M1: guarded driver-root heartbeat stamps, rc-19 line, `.agents` guard seam

**Estimated LOC:** about 45 changed lines (no new code or tests) · **Dependencies:** none · **Registry reuse:** none (shell and markdown harness text, no package-like capability)

### Tasks (in order)

1. **Edit the 7 `.claude` gate files.** Replace each `bash tools/launchd/mission-heartbeat.sh stamp <args>` (9 sites) with the doc's guarded one-liner, keeping `<args>` exactly as they are (`gate-0`, `abort <reason>`, `gate-1`, `gate-2`, `gate-3`, `gate-3b`, `gate-4`, `gate-5`, `complete`):
   ```bash
   case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp <args> ;; *) echo "MISSION_DRIVER_ROOT must be absolute (got: ${MISSION_DRIVER_ROOT:-})" >&2; false ;; esac
   ```
   Keep the command on one line inside its existing inline-code span. `test_mission_heartbeat.sh` greps `stamp <label>` and `stamp abort <reason>`, and both substrings survive.
2. **Setup sentence:** add one to each of the 7 files, beside its first stamp:
   `Attended setup: export MISSION_DRIVER_ROOT="$(cd /path/to/driver-checkout && pwd)" before the first stamp; the stamp refuses an unset, empty or relative root.`
3. **rc-19:** in `.claude/…/gate-3-route.md`, change `18=tool_hang.  Anything non-zero except 18` to `18=tool_hang · 19=provider_quota.  Anything non-zero except 18`. Leave the rest of the line and the following `is a LANE FAILURE…` line as they are.
4. **Sync:** `tools/launchd/sync-agents-skills.sh`, then `tools/launchd/sync-agents-skills.sh --check` → rc=0. Afterwards `git status --short .agents` should list exactly the 7 gate files. If anything else appears, stop: that means `.claude` drift in another of the six skills, which this sprint must not carry.
5. **Guard:** in `tools/launchd/githooks/pre-push` `_scope_is_harness`, add `.agents/skills/mission-*|.agents/skills/sprint-*` (either extend line 17 or add a sibling arm). Then add `.agents/skills/mission-*/**` and `.agents/skills/sprint-*/**` to `design_docs/fleet-mission.md` Authority "May change", next to the `.claude` entries on lines 76–77.
6. **Changelog fragment** `changelogs/unreleased/2026-10-03-heartbeat-driver-root.md`: a `### Fixed — mission heartbeat stamps resolve the helper via an absolute MISSION_DRIVER_ROOT (2026-10-03)` section. It should cover the World/Stapledon rc=127, both trees, rc 19, the guard arm, the D3 product-loop refusal, and the intentional contract change that attended stamps without an export now fail loudly. Then run `make check-changelog` → rc=0.
7. **Acceptance 1–8 and 6a** (below), then the unpiped done-gate.

### Acceptance (doc items 1–8 and 6a, as commands with expected output)

Run from the worktree root. `R1=.claude/skills/mission-control/resources`, `R2=.agents/skills/mission-control/resources`, `G="gate-0-preflight gate-1-observe gate-2-pick gate-3-route gate-3b-ci-green gate-4-record gate-5-retro"`.

**1. Inventory**
```bash
rg -F 'bash tools/launchd/mission-heartbeat.sh' .claude/skills .agents/skills; echo rc=$?
#   → no output, rc=1   (18 matches before)
P='case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp'
for d in $R1 $R2; do for g in $G; do printf '%s %s\n' "$(rg -c -F "$P" $d/$g.md)" "$d/$g.md"; done; done
#   → per copy: 2 gate-0, 1 gate-1, 1 gate-2, 1 gate-3, 1 gate-3b, 1 gate-4, 2 gate-5; sum = 18
```

**2. Setup sentence**
```bash
rg -l -F 'Attended setup: export MISSION_DRIVER_ROOT' $R1 $R2 | sort
#   → exactly the 14 gate files (7 per tree), `| wc -l` → 14   (0 before)
```

**3. rc-19**
```bash
for f in $R1/gate-3-route.md $R2/gate-3-route.md; do
  rg -c -F '14=launch_failed · 18=tool_hang · 19=provider_quota' $f   # → 1
  rg -c -F 'Anything non-zero except 18' $f                           # → 1
done
rg -n 'RC=19' scripts/mission_pi_run.sh    # positive control → 407: … RC=19
```

**4. Mirror**
```bash
tools/launchd/sync-agents-skills.sh --check; echo rc=$?     # → "agents skills in sync (…)", rc=0
for g in $G; do cmp $R1/$g.md $R2/$g.md; done               # → silent
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_agents_skills_sync.sh
#   → 3 passed, 0 failed (includes "negative control: a drifted copy is detected")
```

**5. Guard**
```bash
( MISSION_SCOPE_LIB=1; . tools/launchd/githooks/pre-push
  _scope_is_harness .agents/skills/mission-control/resources/gate-0-preflight.md; echo "mc=$?"     # → mc=0
  _scope_is_harness .agents/skills/sprint-executor/SKILL.md; echo "sx=$?"                          # → sx=0
  _scope_is_harness .agents/skills/model-manager/SKILL.md; echo "mm=$?"                            # → mm=1 (nonzero)
  _scope_verdict fleet .agents/skills/mission-control/resources/gate-0-preflight.md                # → no output (allowed)
  _scope_verdict fleet .agents/skills/model-manager/SKILL.md )                                     # → refuse path outside the fleet allowlist …
rg -n -F '.agents/skills/mission-*/**' design_docs/fleet-mission.md     # → 1 match in Authority
rg -n -F '.agents/skills/sprint-*/**'  design_docs/fleet-mission.md     # → 1 match in Authority
```

**6. Drills.** Use temp state only, extract the command from the edited `.claude` gate-0 file, and run under both `/bin/bash` and `/bin/zsh`. Assert **nonzero**, not a specific code, for every failure case.
```bash
T=$(mktemp -d); DRV=$PWD; WORLD=$HOME/dev/sunholo-data/ailang-world; STAP=$HOME/dev/sunholo-data/stapledons-godot
CMD=$(grep -oE 'case "\$\{MISSION_DRIVER_ROOT:-\}" in /\*\) bash [^`]* esac' $R1/gate-0-preflight.md | grep 'stamp gate-0 ;;')
ABORT=$(grep -oE 'case "\$\{MISSION_DRIVER_ROOT:-\}" in /\*\) bash [^`]* esac' $R1/gate-0-preflight.md | grep 'stamp abort' | sed 's/<reason>/quota_dead/')
rows() { [ -f "$1/mission-i16-drill-heartbeat" ] && wc -l < "$1/mission-i16-drill-heartbeat" | tr -d ' ' || echo 0; }
# per case: S=$T/state-<shell>-<case>; (cd <cwd> && env [-u MISSION_DRIVER_ROOT | MISSION_DRIVER_ROOT=<v>] AILANG_STATE_DIR=$S MISSION_NAME=i16-drill <shell> -c "$CMD"); rc=$?; rows $S
```
| Case | CWD | Expected (both shells) |
|---|---|---|
| unset (`env -u`) | `$DRV` (helper present) | rc ≠ 0, rows 0, stderr contains `MISSION_DRIVER_ROOT must be absolute` |
| empty `''` | `$DRV` | rc ≠ 0, rows 0, same stderr |
| relative `.` | `$DRV` | rc ≠ 0, rows 0, stderr `… (got: .)` |
| absolute nonexistent `/nonexistent/driver` | `$DRV` | rc ≠ 0, rows 0 |
| absolute `$DRV` | `$WORLD` | rc=0, rows 1, row field 3 `gate-0`, field 4 `1` |
| absolute `$DRV` | `$STAP` | rc=0, rows 1, `gate-0`, attempt `1` |
| absolute `$DRV` | `$DRV` | rc=0, rows 1, `gate-0`, attempt `1` |
| `$ABORT` with absolute `$DRV` | `$WORLD` | rc=0, rows 1, `…\tabort\t1\tquota_dead` |
| root with a space: `mkdir -p "$T/sp ace/tools/launchd"; cp tools/launchd/mission-heartbeat.sh "$T/sp ace/tools/launchd/"`, root `"$T/sp ace"` | `$WORLD` | rc=0, rows 1 |

Preflight positive control: `test -f $WORLD/README.md && test -f $STAP/README.md` → true, and `test -f $WORLD/tools/launchd/mission-heartbeat.sh` → false.
Cleanup check: `ls ~/.ailang/state | grep -c i16-drill` → `0`.

**6a. Driver-spawned shell, no manual export.** This only works when the executor or evaluator is itself a controller inside a live pinned driver fire, as in the Verification Log row from fire `fleet-1791028902-72480`.
```bash
printenv MISSION_DRIVER_ROOT          # → absolute, equal to the pin worktree (e.g. /Users/voightkampff/.ailang-driver-pin/fleet)
printenv MISSION_FIRE_ID              # → non-empty (proves this shell is driver-spawned)
S=$T/e2e; for c in "$MISSION_DRIVER_ROOT" "$WORLD" "$STAP"; do
  (cd "$c" && env AILANG_STATE_DIR=$S MISSION_NAME=i16-drill /bin/bash -c "$CMD"); echo "rc=$? rows=$(rows $S)"
done                                  # → rc=0 each, cumulative rows 1, 2, 3
```
If no live fire is available (`MISSION_FIRE_ID` empty), record 6a as **NOT RUN, with the reason**. Do not substitute a manual export, because that would be drill 6 again. The evaluator scores an unrun 6a as an open item, not a pass.

**7. Production root**
```bash
(cd $WORLD && /bin/bash -c 'MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd); echo "root=$MC_DRIVER_ROOT"' "$DRV/tools/launchd/mission-control.sh")
#   → root=$DRV  (equal to the tree, not merely absolute; argv0 = absolute pin path as tools/launchd/lib/pin-root.sh:362 execs it)
sed -n 47p tools/launchd/mission-control.sh     # → MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd)
sed -n 2167p tools/launchd/mission-control.sh   # → MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT
sed -n 362p tools/launchd/lib/pin-root.sh       # → exec /bin/bash "$wt/tools/launchd/$script" "$@"
```

**8. Suites and review**
```bash
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_heartbeat.sh; echo rc=$?   # → 0 failed, rc=0
make test-launchd-drivers > "$T/suite.log" 2>&1; echo rc=$?                                     # DONE-GATE → rc=0 (unpiped)
git diff --name-only <base>..HEAD                                                               # → exactly the "Allowed diff" list
```
The evaluator reads the diff and rejects any path outside "Allowed diff". That includes the three conflict-surface sites, `SKILL.md`, `mission-heartbeat.sh`, `mission-control.sh` and `test_mission_scope_guard.sh`.

### Risks

| Risk | Mitigation |
|---|---|
| The sync rsyncs six skills with `--delete`, so other `.claude` drift could ride along | Today `--check` is clean. Task 4 inspects `git status --short .agents` and stops on any unexpected path. |
| Markdown inline code mangles the one-liner (backticks, `|`) | The command contains neither a backtick nor `|`. Drill 6 extracts the text *from the edited file*, so a mangled form fails the drill. |
| Attended sessions without the export now fail loudly | This contract change is intentional (doc §Attended setup). The setup sentence states it and the changelog names it. |
| Product loops lose push rights on the `.agents` mission and sprint mirrors (D3) | Consistent with `.claude`. Named in the changelog. |
| 6a needs a live fire | Run it inside the fleet fire that executes this sprint. Otherwise mark it NOT RUN and do not fake it. |

## Success metrics

- Acceptance 1–8 and 6a are met, with each command's output banked in the sprint JSON `notes`.
- `make test-launchd-drivers` rc=0, unpiped (baseline 382 s, rc=0 at `11a517b33`).
- After deployment, World and Stapledon heartbeat files gain `gate-N` rows on their next fires. This is observed post-merge, not gated in the sprint.

## Follow-ups (not this sprint)

- `skill:driver-root-guard-unify`: apply the same `case` guard to gate-3-route.md:72, :618 and role-spawn-routing.md:85. The `:-.` fallback at :618 is a real no-silent-fallback violation.
- Add `.agents` rows to `test_mission_scope_guard.sh` (D4).
- Correct the doc's `pin-root.sh` citation to `tools/launchd/lib/pin-root.sh` when the doc moves to `implemented/` (D1).
