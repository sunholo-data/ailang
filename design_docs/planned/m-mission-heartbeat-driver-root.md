# M-MISSION-HEARTBEAT-DRIVER-ROOT

**Status:** IMPLEMENTED (fleet iteration 16, M1 commit bb573f466; evaluator PASS 96/100; relocated to implemented/ at release). Planned mechanical correction; fleet iteration 16, 2026-10-03. **Revision 5.2** (R5: D-FLEET-10 = A absoluteness guard, round-5 Kimi residuals closed; R5.1: quorum round-6 premise rows; R5.2: round-7 reviewer fixes, controller-applied under the narrow-refinement carve-out). Every "measured" claim here maps to a Verification Log row run on 2026-10-03 (R5 rows @ c68ded4b2, R5.1 rows @ 21f045a38, R5.2 rows @ 5ad08e044; the commits differ only in this doc, see the coherence row); rulings and history are not re-measured.
**Priority:** P0 queue ticket `skill:heartbeat-relative-path-absent-in-world`.
**Target:** next fleet deployment. **Effort:** one short milestone.
**Planner-Lane:** codex-ok

**Revision history.** R1 (2026-09-29) fixed `.claude` only; rounds 1–2 blocked on scope.
**D-FLEET-8 (2026-10-01, Mark attended)** = YES: fix BOTH `.claude` and `.agents`
mission-control resources, rc-19 line in scope for both. R2–R3 added mirror, rc-19, guard arm;
R4 characterized the `:?` exit status by invocation shape.
**D-FLEET-10 (2026-10-02, Mark attended)** = A, verbatim: *"the measured two-shape fact (rc=127
for `bash -c` strings, rc=1 for script files, bash 3.2.57) is ruled correct; add the absoluteness
guard on `MISSION_DRIVER_ROOT` and satisfy Kimi's concrete residuals in Revision 5; fresh quorum
next fire. A reviewer that cannot run rig commands does not overrule a recorded rig measurement."*
R5 replaces the `:?` form with a `case` guard (which no longer depends on that two-shape fact at
all), and closes Kimi's six residuals (mapping at the end). R5.1 adds only the round-6 premise rows (Verification Log, rows tagged R5.1). R5.2 adds the round-7 rows (end-to-end propagation, invocation shape, commit coherence, `SKILL.md:338`), corrects two swapped conflict-table cells, widens Acceptance 1 to the whole skill trees and adds Acceptance 6a.

## Problem and scope

Eighteen heartbeat commands in fourteen gate resources — nine in seven `.claude` files, nine in
the byte-mirrored `.agents` copies — call `bash tools/launchd/mission-heartbeat.sh stamp …` by a
path relative to CWD. World (`ailang-world`) and Stapledon (`stapledons-godot`) are not the
ailang repo, so from their CWD the helper is absent and every gate stamp fails (`bash:
tools/launchd/mission-heartbeat.sh: No such file or directory`, rc=127, re-measured 2026-10-03).
`mission-control.sh:2167` exports `MISSION_DRIVER_ROOT` at top level (measured); propagation to driver-spawned mission shells is verified by the R5.2 end-to-end row; attended shells are covered by the setup sentence.

Two further gaps inside the D-FLEET-8 scope:

1. **rc-19 line.** `scripts/mission_pi_run.sh` emits `provider_quota` rc 19 (PR #1424; lines 71
   and 407). The gate-3-route.md rc comment (both copies) ends at `18=tool_hang` and never names
   19, so a route step reading it cannot classify a quota refusal.
2. **Scope-guard seam.** `tools/launchd/githooks/pre-push` `_scope_is_harness` allows
   `.claude/skills/mission-*|.claude/skills/sprint-*` but no `.agents` path, so the mirror half
   would be refused at push. Its comment says "Keep in step with design_docs/fleet-mission.md
   'Authority'", so the guard arm and the Authority allowlist change together.

## Decision

### Guarded command

Replace each `bash tools/launchd/mission-heartbeat.sh stamp <args>` with (one line, `<args>`
unchanged — `gate-N`, `abort <reason>`, `complete`):

```bash
case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp <args> ;; *) echo "MISSION_DRIVER_ROOT must be absolute (got: ${MISSION_DRIVER_ROOT:-})" >&2; false ;; esac
```

Why this form (all behaviour measured, Verification Log):

- **Absoluteness is enforced, not assumed.** Unset, empty and relative (`.`) roots all fall to
  the `*)` arm: message on stderr, rc=1, helper never invoked, no row written — including from a
  CWD that *does* contain `tools/launchd/mission-heartbeat.sh` (the silent-wrong-helper case).
- **One exit code on both shells and both shapes.** `false` yields rc=1 under `/bin/bash` 3.2.57
  and `/bin/zsh` 5.9, as a `-c` string and as a script file, with and without `-u`. The R4
  shape-dependence (D-FLEET-10) belonged to `:?` expansion failure; this form has no expansion
  failure, so it does not arise.
- **No `exit`.** Kimi's proposed `… || { echo …; exit 1; }` is rejected: `exit` would terminate a
  controller's persistent tool shell, and the `||` arm would also fire on a *heartbeat* failure
  and misreport it as "relative root". With `case`, a helper failure (e.g. an absolute root
  that does not exist → rc=127) surfaces with the helper's own message and rc.
- **No fallback.** Never CWD, never `AILANG_DRIVER_SRC` (the driver comment at
  `tools/launchd/mission-control.sh:2164-2166`, quoted in the Verification Log, names that clone as possibly behind what runs).

**Attended setup sentence.** Each of the fourteen files gets, beside its first stamp: `Attended setup: export MISSION_DRIVER_ROOT="$(cd /path/to/driver-checkout && pwd)"
before the first stamp; the stamp refuses an unset, empty or relative root.` (Key phrase for
acceptance: `Attended setup: export MISSION_DRIVER_ROOT`, absent from all resources today.) An
attended run with no export now fails loudly even from the driver tree — an intentional contract
change.

**Mirror.** Edit the seven `.claude` files, then run `tools/launchd/sync-agents-skills.sh` (rsyncs
the six mission-loop skills verbatim to `.agents/skills/`); never hand-edit `.agents`.
`tools/launchd/test_agents_skills_sync.sh` (make/test.mk:82) enforces equality.

**rc-19.** In `.claude/.../gate-3-route.md`, change the rc comment line whose current content is
`#    13=wall_timeout · 14=launch_failed · 18=tool_hang.  Anything non-zero except 18` so that
`18=tool_hang` becomes `18=tool_hang · 19=provider_quota` (rest of the line and the following
`#    is a LANE FAILURE, not a result: fall back and FLAG, never re-prompt in place.` line
unchanged), then re-sync. The unchanged catch-all sentence makes rc 19 a lane failure → fall back
along the role's declared chain per D-FLEET-2; no further edit.

**Guard seam.** Add `.agents/skills/mission-*|.agents/skills/sprint-*` arms to
`_scope_is_harness`, and `.agents/skills/mission-*/**`, `.agents/skills/sprint-*/**` to the
"May change" allowlist in `design_docs/fleet-mission.md` Authority. Mirror paths of the
already-allowed skills only; every other `.agents/**` path is still refused.

No new helper, export, state format, label or heartbeat semantics.

### Conflict surface — the other `MISSION_DRIVER_ROOT` uses (Kimi item 5)

`rg -n MISSION_DRIVER_ROOT .claude/skills .agents/skills | grep -v mission-heartbeat` (2026-10-03,
output in the Verification Log) finds exactly three non-heartbeat uses, identical in both copies:

| Site | Form | Unset/relative behaviour | This revision |
|---|---|---|---|
| gate-3-route.md:72 | `"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence: verdict path, rc, error>"` (bare) | unset → `/tools/launchd/mission-lane-dead.sh: No such file or directory`, rc=127 (Verification Log, 2026-10-03); relative → CWD-resolved | **not aligned** |
| role-spawn-routing.md:85 | `"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence>"` (bare) | same | **not aligned** |
| gate-3-route.md:618 | `MW="${MISSION_DRIVER_ROOT:-.}/tools/launchd/mission-worktree.sh"` | unset → **silent CWD fallback** to `./tools/launchd/mission-worktree.sh` (Verification Log, 2026-10-03) | **not aligned** |

Justification: D-FLEET-8/10 scope this fix to heartbeat sites plus rc-19; changing Gate-3 lane
death and worktree creation needs its own drills. The bare form already fails loudly when unset;
the `:-.` fallback is a genuine no-silent-fallback violation. Follow-up ticket
`skill:driver-root-guard-unify` applies this `case` guard to all three; the evaluator must
reject an implementation that touches them in this milestone.

## Files to modify

Heartbeat call sites (`rg -n 'mission-heartbeat\.sh' .claude/skills .agents/skills` at HEAD
c68ded4b2, 2026-10-03; `SKILL.md:338` is prose naming the helper, not an invocation; quoted in the R5.2 row, unchanged). Each path exists in
both `.claude/skills/mission-control/resources/` (edited) and
`.agents/skills/mission-control/resources/` (produced by sync):

| File | Lines | Commands |
|---|---|---|
| `gate-0-preflight.md` | 3 (`stamp gate-0`), 4 (`stamp abort <reason>`) | 2 |
| `gate-1-observe.md` | 3 (`stamp gate-1`) | 1 |
| `gate-2-pick.md` | 89 (`stamp gate-2`) | 1 |
| `gate-3-route.md` | 3 (`stamp gate-3`) + rc comment line (`… 18=tool_hang.  Anything non-zero except 18`) | 1 |
| `gate-3b-ci-green.md` | 3 (`stamp gate-3b`) | 1 |
| `gate-4-record.md` | 61 (`stamp gate-4`) | 1 |
| `gate-5-retro.md` | 3 (`stamp gate-5`), 5 (`stamp complete`) | 2 |

Totals: 9 commands × 2 copies = 18 sites in 14 files; acceptance is pinned to content, not lines.

Scope guard: `tools/launchd/githooks/pre-push` (`_scope_is_harness`) and
`design_docs/fleet-mission.md` (Authority "May change").

## Acceptance (content-pinned; one milestone)

1. **Inventory.** `rg -F 'bash tools/launchd/mission-heartbeat.sh' .claude/skills .agents/skills` (whole skill trees, not only the resource dirs) →
   0 matches (18 before, all in resource dirs). `rg -c -F 'case "${MISSION_DRIVER_ROOT:-}" in /*) bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh" stamp'`
   summed over both dirs → 18, distributed 2/1/1/1/1/1/2 per file per copy.
2. **Setup sentence.** `rg -l -F 'Attended setup: export MISSION_DRIVER_ROOT' .claude/skills/mission-control/resources .agents/skills/mission-control/resources | wc -l` → **14** (0 today), and exactly the fourteen files above.
3. **rc-19.** In both gate-3-route.md copies, `rg -c -F '14=launch_failed · 18=tool_hang · 19=provider_quota' <file>` → 1, and `rg -c -F 'Anything non-zero except 18' <file>` → 1 (catch-all intact). Positive control: `rg -n 'RC=19' scripts/mission_pi_run.sh` still matches.
4. **Mirror.** `tools/launchd/sync-agents-skills.sh --check` passes; `cmp` of each `.claude`/`.agents` pair is silent; launchd suite row `test_agents_skills_sync.sh` passes (incl. its drift negative control).
5. **Guard.** `_scope_is_harness .agents/skills/mission-control/resources/gate-0-preflight.md` → 0; `_scope_is_harness .agents/skills/model-manager/SKILL.md` → nonzero; Authority allowlist names both `.agents` arms.
6. **Drills** (temporary `AILANG_STATE_DIR`, `MISSION_NAME=<drill>`; never real state), extracting the command text from the edited gate-0 file, run under `/bin/bash` and `/bin/zsh`:
   - unset / empty / relative `.` (from the driver tree, which contains the helper) → **nonzero, 0 rows**, stderr names `MISSION_DRIVER_ROOT must be absolute`;
   - absolute nonexistent root → nonzero, 0 rows;
   - absolute correct root from World CWD, Stapledon CWD and the driver CWD → rc=0, exactly 1 row with the drilled label and attempt 1;
   - `stamp abort <reason>` → row carries the reason note; a root containing a space → rc=0, 1 row.
   Assert **nonzero** for failures, not a specific code.
6a. **Driver-spawned shell, no manual export.** Inside a controller tool shell spawned by a live pinned driver fire, with only `AILANG_STATE_DIR=<temp>` and `MISSION_NAME=<drill>` set by hand: `printenv MISSION_DRIVER_ROOT` → absolute and equal to the pin worktree; the guarded `stamp gate-0` extracted from the edited gate-0 file → rc=0, exactly 1 row, from the driver, World and Stapledon CWDs.
7. **Production root.** Extract `MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." … && pwd)` from `tools/launchd/mission-control.sh:47` and run it with the production argv (`$0` = the absolute pin path that `tools/launchd/lib/pin-root.sh:362` execs) from a World CWD → equal to the pin tree, not merely absolute; `:2167` exports it unconditionally (`MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT`).
8. **Suites + review.** `tools/launchd/test_mission_heartbeat.sh` passes. Evaluator inspects the diff (14 gate files, guard, fleet-mission.md) and rejects any other path, including the three conflict-surface sites.

## Verification log (2026-10-03, worktree fleet-i16 @ c68ded4b2; R5.1 rows @ 21f045a38)

Shells: `/bin/bash` → `GNU bash, version 3.2.57(1)-release`; `/bin/zsh` → `zsh 5.9`. Drill
harness: a script under `mktemp -d` setting per case `env [-u] MISSION_DRIVER_ROOT=…
AILANG_STATE_DIR=$T/state-<shell>-<case> MISSION_NAME=i16-drill <shell> -c "$CMD"` from the
listed CWD; rows = `wc -l` of `$S/mission-i16-drill-heartbeat` (0 if absent). `$CMD` = the
guarded command with `stamp gate-0`. `ls ~/.ailang/state | grep -c i16-drill` → 0 afterwards.

| Case (`-c` shape) | CWD | bash 3.2.57 | zsh 5.9 | stderr |
|---|---|---|---|---|
| unset | driver tree | rc=1, 0 rows | rc=1, 0 rows | `MISSION_DRIVER_ROOT must be absolute (got: )` |
| empty | driver tree | rc=1, 0 rows | rc=1, 0 rows | same |
| relative `.` | driver tree (helper present) | rc=1, 0 rows | rc=1, 0 rows | `… (got: .)` |
| absolute, correct | driver tree | rc=0, 1 row | rc=0, 1 row | — |
| absolute, nonexistent `/nonexistent/driver` | driver tree | rc=127, 0 rows | rc=127, 0 rows | `bash: /nonexistent/driver/tools/launchd/mission-heartbeat.sh: No such file or directory` |
| absolute, correct | `ailang-world` | rc=0, 1 row | rc=0, 1 row | — |
| absolute, correct | `stapledons-godot` | rc=0, 1 row | rc=0, 1 row | — |

Rows written were `<epoch>\t2026-10-03T12:04:…Z\tgate-0\t1\t` (label, attempt 1).

| Claim | Command → observed |
|---|---|
| Relative drill is meaningful (positive control) | From the driver tree, `test -f tools/launchd/mission-heartbeat.sh` → true; unguarded `env AILANG_STATE_DIR=$T/ctl MISSION_NAME=i16-drill /bin/bash ./tools/launchd/mission-heartbeat.sh stamp gate-0` → rc=0, 1 row. So the guard, not a missing file, blocks the relative case. |
| Ticket premise: helper absent in World/Stapledon | `test -f <repo>/README.md` → yes for both (positive control); `test -f <repo>/tools/launchd/mission-heartbeat.sh` → no for both; from each CWD, unguarded `bash tools/launchd/mission-heartbeat.sh stamp gate-0` → `No such file or directory`, rc=127. Fresh 2026-10-03, not re-listed. |
| Script-file shape, bash and zsh, `-u` | Guarded command (`stamp abort quota_dead`) saved as a file: `/bin/bash` relative → rc=1, 0 rows; `/bin/bash` unset → rc=1, 0 rows; `/bin/bash -u` unset → rc=1; `/bin/zsh -u` unset → rc=1; `/bin/zsh` absolute → rc=0, row `…\tabort\t1\tquota_dead`. |
| Space in root | `MISSION_DRIVER_ROOT="$T/sp ace"` (copy of the helper inside) → rc=0, row `…\tabort\t1\tquota_dead`. |
| Production root is absolute | `/bin/bash -c 'MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd); echo "root=$MC_DRIVER_ROOT"' tools/launchd/mission-control.sh` → `root=/Users/voightkampff/.ailang-driver-pin/fleet-i16`. `rg -n 'MISSION_DRIVER_ROOT\|MC_DRIVER_ROOT=' tools/launchd/mission-control.sh` → `47:MC_DRIVER_ROOT=$(cd …&& pwd)`, `2167:MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT` (top level, every mission). |
| Call-site inventory | `rg -n 'mission-heartbeat\.sh' .claude/skills .agents/skills` → 18 command sites at the lines in "Files to modify" + `SKILL.md:338` prose in each tree. |
| rc comment content, copies identical | `sed -n 389,391p` both copies → `# rc 0=ok · 10=empty_worktree · 11=reasoning_stall · 12=stream_dead` / `#    13=wall_timeout · 14=launch_failed · 18=tool_hang.  Anything non-zero except 18` / `#    is a LANE FAILURE, not a result: fall back and FLAG, never re-prompt in place.`; `cmp` silent. |
| rc 19 exists upstream | `rg -n provider_quota scripts/mission_pi_run.sh` → `71:#   19 provider_quota — …`, `407: … VERDICT_NAME="provider_quota"; RC=19`. |
| Setup sentence absent today | `rg -c 'Attended setup' <both resource dirs>` → no match, rc=1. |
| Guard has no `.agents` arm | `rg -n '\.agents' tools/launchd/githooks/pre-push` → rc=1; control: line 17 `.claude/skills/mission-*\|.claude/skills/sprint-*\|.pi/extensions/*) return 0 ;;`. `design_docs/fleet-mission.md:75-76` "May change" lists `.claude/skills/mission-*/**`, no `.agents`. |
| Tests exist and are wired | `ls tools/launchd/{sync-agents-skills,test_agents_skills_sync,test_mission_heartbeat}.sh` → present; `make/test.mk:82` runs `test_agents_skills_sync.sh`. |
| Helper state contract | `tools/launchd/mission-heartbeat.sh`: state dir `${AILANG_STATE_DIR:-$HOME/.ailang/state}`, file `mission-${MISSION_NAME}-heartbeat`, label whitelist `fired\|gate-0…gate-5\|gate-3b\|complete\|abort`, unset `MISSION_NAME` → no row, rc 0. Unchanged by this design. |
| Conflict-surface sites (R5.1) | `rg -n 'MISSION_DRIVER_ROOT' .claude/skills .agents/skills \| grep -v mission-heartbeat` → exactly six lines (three per copy), verbatim: ``` .agents/skills/mission-control/resources/role-spawn-routing.md:85:  `"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence>"`. The hook ``` · ``` .agents/skills/mission-control/resources/gate-3-route.md:72:`"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence: verdict path, rc, error>"`. ``` · ``` .agents/skills/mission-control/resources/gate-3-route.md:618:  MW="${MISSION_DRIVER_ROOT:-.}/tools/launchd/mission-worktree.sh"   # world/stapledon have no tools/ ``` · ``` .claude/skills/mission-control/resources/role-spawn-routing.md:85:  `"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence>"`. The hook ``` · ``` .claude/skills/mission-control/resources/gate-3-route.md:72:`"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" <role> <lane> "<evidence: verdict path, rc, error>"`. ``` · ``` .claude/skills/mission-control/resources/gate-3-route.md:618:  MW="${MISSION_DRIVER_ROOT:-.}/tools/launchd/mission-worktree.sh"   # world/stapledon have no tools/ ``` |
| Lane-dead bare form, unset (R5.1) | From the driver tree (which contains `tools/launchd/mission-lane-dead.sh`), `env -u MISSION_DRIVER_ROOT AILANG_STATE_DIR=$T/ld /bin/bash -c '"$MISSION_DRIVER_ROOT/tools/launchd/mission-lane-dead.sh" impl lane-x ev'` → rc=127, stderr `/bin/bash: /tools/launchd/mission-lane-dead.sh: No such file or directory`; under `/bin/zsh` → rc=127, `zsh:1: no such file or directory: /tools/launchd/mission-lane-dead.sh`. `$T/ld` never created (no write). |
| Worktree `:-.` form, unset (R5.1; resolution only, nothing executed) | `env -u MISSION_DRIVER_ROOT /bin/bash -c 'MW="${MISSION_DRIVER_ROOT:-.}/tools/launchd/mission-worktree.sh"; echo "$MW"'` → `./tools/launchd/mission-worktree.sh`, rc=0; `/bin/zsh` identical. That path exists from the driver tree → the silent CWD fallback is real. |
| Driver export is top level (R5.1) | `sed -n 2160,2170p tools/launchd/mission-control.sh` → `# so a new fire always starts with every declared lane presumed alive.` / `MISSION_FIRE_ID="${MISSION_NAME:-mission}-$(date +%s)-$$"; export MISSION_FIRE_ID` / `# Routing changes on this fire (controller switch, degraded lanes) for the digest's Routing line.` / `export MISSION_ROUTING_NOTE` / `# The pinned driver's own tree, for skill steps that call driver tools from a mission` / `# repo that has none (world, stapledon). NOT AILANG_DRIVER_SRC: that is the source clone,` / `# which can be far behind what actually runs.` / `MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT` / `# The over-ration buckets travel with the resolved plan, so resolve-role-spawn.sh can` / `# refuse a role recipe the driver itself would have refused (see that script's gate).` / `_mc_load_ration`. All column 0; the last block before it, `if [ -n "$_pin_age_degraded" ]` (2141), closes with `fi` at 2151; nothing opens in 2152–2167. |
| End-to-end propagation (R5.2) | In THIS fleet controller's Bash tool shell, spawned by live driver fire `fleet-1791028902-72480` (driver pid 72480 = `/bin/bash /Users/voightkampff/.ailang-driver-pin/fleet/tools/launchd/mission-control.sh`; the controller's parent is `claude -p Run one mission-control iteration…`), NO manual export: `printenv MISSION_DRIVER_ROOT` → `/Users/voightkampff/.ailang-driver-pin/fleet` (the pin). Guarded `stamp gate-0` with only `AILANG_STATE_DIR=$T MISSION_NAME=i16e2e-drill` set, `/bin/bash -c`, from the pin, `ailang-world` and `stapledons-godot` CWDs → rc=0 each, cumulative rows 1/2/3 (one per call). Controller switch / persistent shell: the Bash tool keeps no shell state between calls and each controller is a new `claude` process inheriting the driver env, so no stale-export path exists inside a fire; an attended persistent shell is the setup sentence's case. |
| Invocation shape (R5.2) | `PlistBuddy -c 'Print :ProgramArguments'` on `dev.ailang.mission-fleet.plist` and `dev.ailang.mission-world.plist` → `/bin/bash`, `/Users/voightkampff/dev/sunholo-data/ailang/tools/launchd/mission-control.sh` (absolute); World `WorkingDirectory` = `…/ailang-world`. `tools/launchd/lib/pin-root.sh:362`: `exec /bin/bash "$wt/tools/launchd/$script" "$@"` with absolute `$wt`. The `:47` derivation run from the World CWD with argv0 = the pin path → `/Users/voightkampff/.ailang-driver-pin/fleet` (equal to the pin); with the pre-exec argv0 → `…/dev/sunholo-data/ailang` (superseded by the re-exec before any gate runs); with a nonexistent absolute argv0 → empty, which the guard refuses. Every supported argv is absolute, so the `2>/dev/null` path yields empty, never absolute-but-wrong. |
| Commit coherence (R5.2) | `git diff --stat c68ded4b2 21f045a38 -- tools/launchd .claude/skills .agents/skills scripts/mission_pi_run.sh design_docs/fleet-mission.md` → empty; whole-commit stat → only this doc. R5 and R5.1 rows measured identical inputs. |
| SKILL.md:338 (R5.2) | `sed -n 338p` in both trees → "The per-gate `mission-heartbeat.sh` stamps above are the durable attribution contract for this" (prose, no invocation). `rg -n -F 'bash tools/launchd/mission-heartbeat.sh' .claude/skills .agents/skills \| grep -v /resources/` → no output. |
| Pre-push comment (R5.1) | `rg -n 'Keep in step' tools/launchd/githooks/pre-push` → `12:# Harness paths: the fleet mission's write scope. Keep in step with` (line 13: `# design_docs/fleet-mission.md "Authority".`). |

### Quorum verification log

- **Rounds 4/5 (gemini) exit-status objection** (first raised round 2) — "bash expansion errors exit 1, the measured
  127 is wrong": **adjudicated by D-FLEET-10 (2026-10-02)** — both shapes (rc=127 `-c`, rc=1
  script file, bash 3.2.57) ruled correct on the recorded rig measurement. Moot for R5 anyway:
  the `case` form has no expansion failure and yields rc=1 in every measured shape.
- **Round 5 (kimi) residuals** → (1) absoluteness: Guarded command + Drill 6; (2) fresh
  premise/per-CWD drills: Verification Log; (3) rc-19: Acceptance 3; (4) setup sentence:
  Acceptance 2; (5) conflict surface: table + ticket; (6) axioms scoped below. `|| exit 1`
  declined (see "Guarded command").
- **Round 6** (2026-10-03T12:07:53Z) — BLOCKED on premise verification only; direction not
  disputed. kimi + gemini rejected on unlogged premises: the "(measured)" conflict rows, the
  three-site `rg`, the `:2160-2170` export context, the pre-push "Keep in step" line and the
  Status re-run claim — closed by the five R5.1 Verification Log rows and the scoped Status.
  gpt6-1-sol absent (OpenAI API 429 "no credits remaining" — capacity, not a verdict);
  oc-glm-5-3 absent (invalid JSON; raw text began `"verdict": "pass"`).
- **Round 7** (2026-10-03T12:14:33Z) — BLOCKED, direction not disputed. gemini: two conflict-table
  cells swapped vs the log → corrected verbatim. kimi + glm: runtime propagation, invocation shape,
  commit coherence, `SKILL.md:338` → R5.2 rows + Acceptance 6a/7 + whole-tree Acceptance 1, applied
  by the controller under the narrow-refinement carve-out (reviewers' fixes, no designer run).
  gpt6-1-sol absent again (OpenAI API 429, no credits).

## Related documents

[`m-mission-slot-heartbeat`](v1_0_0/m-mission-slot-heartbeat.md) (heartbeat instrument),
[`m-mission-portability`](../implemented/v0_30_0/m-mission-portability.md) (cross-repo profiles),
[`m-spawn-pin-enforcement`](../implemented/v0_35_0/m-spawn-pin-enforcement.md) (resolved driver),
[`fleet-mission.md`](../fleet-mission.md) (Authority, D-FLEET-8). Duplicate gate (R1): 0.42/0.38, below threshold.

## Authority and axioms

Rulings: **D-FLEET-8** (scope), **D-FLEET-10** (guard), **D-FLEET-2** (rc-19 fallback chain).
Guard change stays in the fleet's write scope and adds mirror paths only.

Axiom scores, each scoped to what the command *enforces*: **A1 +1** — every stamp that runs uses
`$MISSION_DRIVER_ROOT/tools/launchd/mission-heartbeat.sh` from an absolute root, so the helper is
CWD-independent; a relative or missing root is refused, not resolved (drilled). It does not prove
the absolute root names the *intended* checkout — that is the driver's derivation (Acceptance 7)
or the attended operator's export. **A5 +1** — drills isolated in temp state. **A7 +1** —
refusal is a fixed stderr message + nonzero. **A11 +1** — unset/empty/relative roots fail before
the helper runs with no row and no fallback (drilled on both shells); the three non-heartbeat
sites keep their current behaviour (follow-up ticket). **A12 +1** — guard and Authority allowlist
change together. Others 0. Net +5; no hard violations.

Preserved: driver-tree missions. Repaired: World/Stapledon stamps, both mirrors. Intentional
incompatibility: attended stamps without an absolute export fail loudly. Rollout follows the
pinned-tree rules; saving a pin is not proof of fleet deployment.
