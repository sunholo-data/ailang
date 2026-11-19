# M-IDLE-RECORD-DEGRADED-BAR — the idle record should bar a bad verdict, not a fallback lane

**Status**: Planned (fleet iteration 97, designer role, unattended); revised after quorum round 1
**Ticket**: `skill:idle-record-barred-on-degraded-no-pick-fire` (filed by world iteration 370, 2026-11-09T08:55Z, slots_lost=1)
**Lane**: motoko/harness extension (skill text + one test arm). No core change and no driver change.
**Related**: `driver:idle-fire-writes-full-block-report` (#2160 → `d08031597`, fleet iteration 91), `33190af48`

## Problem

The IDLE RECORD rule in `.claude/skills/mission-control/resources/gate-5-retro.md` (lines 11-31)
refuses to record whenever "the fire degraded (a failed probe, a controller or role fallback, a gate
you could not run)". On a no-pick fire, the lane resolution does not affect any input to the record:

- `--record` only promotes the snapshot the driver took **before** the controller started. It never
  recomputes anything, so which controller ran cannot change the fingerprint (V6, V8).
- The fingerprint hashes head, ledger statuses, directives, the unread inbox and red checks. No lane,
  probe or controller identity is an input (V7).
- A NONE pick spawns no designer, planner or executor, so their handoffs never execute (V10).

So the bar does not protect the fingerprint. What it does is stop the mechanism ever engaging. Every
live World fire since its queue became fully human-gated has fallen back somewhere, so none has
recorded. As a result, each no-pick fire runs a full 1-2.5 h block-record fire (V11) instead of
exiting at the driver's pre-check. No promoted record file exists for any mission (V5).

The bar has one legitimate concern, and it is not lane identity. A **weaker fallback controller
could call NONE carelessly**, or skip a read that the verdict depends on. Both are failures of the
verdict's inputs and its classification, and the bar should be written in those terms.

## Evidence

- No `mission-*-idle-fingerprint` record exists. Only two `.pending` snapshots exist, for world and
  stapledon (V5).
- World log lines 1613 and 1656 (iterations 369 and 370) say "NOT recorded … the idle-record rule
  bars recording a degraded fire" (V10).
- World's COMPLETED fires from 2026-11-07 to 2026-11-09 each took 3605-8931 s (V11).
- The bar was added in `33190af48` as one bullet ("never on a degraded fire") with no stated
  rationale, and no ledger row rules on it (V1, V2, V14). It was a conservative controller choice,
  not an attended ruling.

## Decision

**D1 — what still bars the record, and what no longer does (Q1).**

These still bar the record:
- (a) A gate that did not run, or did not run to completion, for lack of capacity or because a
  command failed. A gate that the no-pick path skips by design does not count.
- (b) A failed or unverified read of any input to the verdict: directives (Gate 0), the ledger
  (`mission_decisions.sh --check`), the mission inbox, CI on the tip, or the queue.
- (c) A fallback that left the NONE classification itself unperformed.
- (d) Any item that is PARKED-ON-LANE or otherwise waiting on a clock (standing rule 8), worded as
  today (V20).
- (e) Anything picked, built or landed, as today.
- (f) A gate table that fails (D2).

These no longer bar it:
- A controller fallback rung that ran every gate.
- Role-lane handoffs or failed probes for roles that this fire never spawned.

**D2 — the gate table, required on EVERY record, whether or not the fire degraded (Q2).** A
candidate qualifies only when its ONLY remaining unblock event is a fingerprint input. That means
one of two things: the cited row is resolved (a ledger status flip, hashed from
`origin/<branch>:$MISSION_DOC`, V9), or an allowlisted directive answers it (directives are hashed,
V7). Citing an OPEN row is not enough on its own. If any other condition must ALSO become true, the
record is barred even though a row is cited. Such conditions include an attended step that is not
recorded as that row's resolution, an external predicate, a lane, a clock, or a row that exists
only on an unmerged branch. Any of these can be satisfied while the hash stays the same, and the
next fire would then skip for up to 24 h. That is the delay `33190af48` called a defect. A
`behind:<row>` entry qualifies only if the row it points to qualifies.

The table has one row per candidate that Gate 2 screened, with three columns: the candidate, the
cited ids, and **"unblocked by the row flip alone: yes/no"**. Any "no" bars the record. The shell
block checks only that the cited rows are OPEN in the hashed copy. **The "flip alone" column is a
controller judgement that the shell cannot check.** It is made visible in the report so that a record
judge (where the mission runs one) can challenge it. A uniform rule is simpler than a separate rule
for degraded fires, and it tightens nothing that has ever been used, since nothing has ever recorded
(V5).

**D3 — skill text only, plus a wording guard (Q3).** `idle-fingerprint.sh` and the driver are
unchanged. A code-enforced `--record --gates` was considered and rejected. The snapshot keeps only
the hash and counts, not row ids, so enforcing it in code would mean changing the snapshot format,
and it still would not check that the candidate list is complete. The guard is a test arm (see the
Test plan). **It checks wording, not controller behaviour.** The behavioural evidence is the first
recorded fingerprint and a following `verdict=IDLE` row (see Risks).

**D4 — not HD-2a policy class (Q4).** HD-2a covers routing, lane order, quota or ration thresholds,
and billing guards (V13). This change alters none of them. The driver's lane walk, probes, rations
and spend guards are untouched. Fewer full fires will run, and so less will be spent, but that is a
consequence of a bookkeeping record, not a change to a threshold or a guard. No guard is loosened.
The text being narrowed was a fleet controller's own choice in iteration 91, with no D-row
behind it (V14), and fleet authority covers `.claude/skills/mission-*` and `tools/launchd/**`
(V13).

### Replacement text

Replace gate-5-retro.md lines 13-17, from "For missions other than fleet/triage" through "a gate you
could not run).", with:

````
For missions other than fleet/triage, record ONLY when ALL of these hold:
- The pick was NONE and nothing was picked, built or landed, and NO item is PARKED-ON-LANE or
  otherwise waiting on a clock (standing rule 8).
- Every gate the no-pick path runs ran to completion, and no read behind the verdict failed
  (directives, `mission_decisions.sh --check`, mission inbox, CI on the tip, queue).
- The gate table passes. It has one row per candidate Gate 2 screened, listing the candidate,
  the cited ledger ids (or `behind:<row>`, which qualifies only if that row does), and
  "unblocked by the row flip alone: yes/no". A cited row must be OPEN in the charter on
  `origin/<branch>` (the copy the fingerprint hashes), and its resolution, by status flip or
  allowlisted directive, must be the candidate's ONLY remaining unblock event. If an attended
  step not recorded as that row's ruling, an external predicate, a lane, a clock or a
  branch-only row must ALSO change, write "no". Any "no" bars the record.
A controller fallback rung that ran every gate does NOT bar the record, and neither do
handoffs or failed probes for roles this fire never spawned: the record is the driver's
pre-controller snapshot, and no lane is one of its inputs. Run this under /bin/bash with your
cited ids, and paste the table and the output into the report. It checks that the rows are
OPEN, not the "flip alone" column, which is your judgement:

```bash
t=$(mktemp) && ref=$(git -C "$MISSION_WORKDIR" symbolic-ref --quiet --short refs/remotes/origin/HEAD); rc=$?
case "$rc:$ref" in 0:origin/?*) br=${ref#origin/} ;; 1:) git -C "$MISSION_WORKDIR" show-ref --verify --quiet refs/remotes/origin/HEAD; [ $? -eq 1 ] && br=dev ;; esac
[ -n "${br:-}" ] && git -C "$MISSION_WORKDIR" show "origin/$br:$MISSION_DOC" >"$t" || { echo "BRANCH/CHARTER READ FAILED: do not record"; br=; }
open=$([ -n "$br" ] && bash "$MISSION_DRIVER_ROOT/scripts/mission_decisions.sh" --open --file "$t" | cut -f1)
echo "branch: ${br:-NONE} open rows: $(printf '%s\n' "$open" | grep -c .)"
for id in D-X-1 D-X-2; do printf '%s\n' "$open" | grep -qx "$id" && echo "OK $id" || echo "MISS $id"; done; rm -f "$t"
```

Any of these means no record: a READ FAILED line, `branch: NONE`, zero open rows, any MISS, or any "no".
````

The branch resolution mirrors `idle-fingerprint.sh` lines 61-68: when `origin/HEAD` is absent
(rc 1, no ref), use `dev`; any other failure means do not record. The remaining lines, from "After
every report has been sent" to the Ticket line, stay as they are.

## Files

- `.claude/skills/mission-control/resources/gate-5-retro.md`: replace the IDLE RECORD paragraph as
  above.
- `.agents/skills/mission-control/resources/gate-5-retro.md`: generated mirror. Regenerate it with
  `tools/launchd/sync-agents-skills.sh` and never hand-edit it. It must stay byte-equal, which
  `tools/launchd/test_agents_skills_sync.sh` enforces (V15).
- `tools/launchd/test_mission_idle_fingerprint.sh`: one new arm and five new mutants.
- `changelogs/unreleased/2026-11-09-idle-record-degraded-bar.md`: a `### Changed` fragment.

## Test plan

| # | Asserts | Kills mutant |
|---|---|---|
| T1 | A new arm in `test_mission_idle_fingerprint.sh` (file overridable as `IDLE_TEST_RETRO`, default the `.claude` gate-5-retro.md). It extracts `### IDLE RECORD` up to the line `1. Scan`, and the extract must contain `mission_decisions.sh" --open --file` | `retro-no-ledger-proof`: delete that line |
| T2 | The same extract must NOT contain `a controller or role fallback` | `retro-old-bar`: sed the old sentence back in |
| T3 | The extract contains `ran to completion` and `no read behind the verdict failed` | `retro-drop-read-bar`: delete the second bullet |
| T3b | The extract contains `unblocked by the row flip alone` and `ONLY remaining unblock event` | `retro-drop-flip-alone`: delete the gate-table bullet's flip-alone sentences |
| T3c | The extract contains `symbolic-ref --quiet --short refs/remotes/origin/HEAD` and does NOT contain `origin/dev:$MISSION_DOC` (no hardcoded branch) | `retro-hardcode-dev`: sed the resolution back to `origin/dev:` |
| T4 | The extract contains `idle-fingerprint.sh" --record` exactly once (V16 shows the quote form) | no dedicated mutant (structural: catches a duplicated or dropped record line) |
| T5 | `--mutations` grows from 12 to 17 mutants, and all are killed. The harness copies the retro file as it copies DRIVER and HELPER (V17) | each of the five above |
| T6 | `tools/launchd/test_agents_skills_sync.sh` passes, so the mirror is byte-equal | an unsynced mirror |
| T7 | `tools/launchd/test_mission_heartbeat.sh` still finds `stamp complete` in gate-5 | an edit that drops the stamp line |
| T8 | `make test-launchd-drivers` passes, and it already runs this test file (V18) | any of the above in CI |

Today the test file contains no reference to gate-5-retro at all (V16), so T1-T5 are new coverage.
**Honest limit:** T1-T5 prove the words are present. They cannot prove that a controller follows
them, and nothing can check the "flip alone" column except a reader of the report.

## Risks

- **A wrongly written record** costs at most 24 h of skipped fires. Any change to head, ledger,
  directives, the inbox or red checks wakes the next fire, and a record older than 24 h forces a
  full fire (driver pre-check, V8). Mitigations: D2 makes the OPEN-row check mechanical; the
  "flip alone" judgement is visible in the report; and missions that run a record judge (World)
  have it reviewed independently.
- **Inputs the fingerprint does not watch**: lane recovery, clocks, branch-only ledger edits, and
  attended or external conditions that sit beside a cited row. D1(d) and D2's "flip alone" column
  bar every one of these. The remaining gap is a wrong "yes" in that column, which is the
  controller's judgement and is visible in the report.
- **The fix may not engage for World immediately.** In World's iteration-370 screen, rows 150 and
  151 wait on an attended step and an external predicate, with no ledger row. The record chain
  waits on D-WORLD-113, which exists only on the record branch: origin/dev has 4 OPEN rows and the
  branch has 5 (V12). Under D2 that fire would still not record. That is the correct outcome:
  World's fires with gates like these keep running in full unless the attended step itself is
  recorded as the ruling on a ledger row on the dev side. That is World's lane, and the ticket
  resolution should say so.
- **Observation criterion** (behaviour, not wording): within the first 10 world or stapledon fires
  after landing, either a `~/.ailang/state/mission-<name>-idle-fingerprint` file appears and a later
  `verdict=IDLE … elapsed_s=0` slot row follows, or each report shows which D2 or D1 bar fired.

## Out of scope

- D-FLEET-43 (whether the fleet itself should skip an idle fire).
- Any change to `idle-fingerprint.sh` inputs or the 24 h expiry.
- World's own queue and ledger hygiene.
- A weekly or other scheduled duty can be delayed by a record that lasts up to 24 h. That property
  already exists in the mechanism; this change neither introduces nor fixes it.

## Verification Log

All commands were run 2026-11-09 in the worktree at `b5b5d149c` unless a different path is given.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | The degraded bar has one origin | `git log --oneline -S 'fire degraded' -- .claude/skills/mission-control/resources/gate-5-retro.md` | `33190af48 fix(launchd): idle record = the snapshot taken BEFORE the controller starts` (only line) |
| V2 | No rationale beyond one bullet | `git show 33190af48` | message bullet: "gate-5-retro.md: record before `stamp complete`; never on a degraded fire; a charter edit that flips no ledger row is not an input." Diff adds the "fire degraded (…)" sentence; no other reasoning |
| V3 | The bar lives at line 17 in both copies | `grep -n 'fire degraded' .claude/…/gate-5-retro.md .agents/…/gate-5-retro.md`; control `grep -c 'IDLE RECORD' .claude/…/gate-5-retro.md` | both `:17:fire degraded (a failed probe, a controller or role fallback, a gate you could not run).`; control `1` |
| V4 | No other skill or driver text carries the bar | `grep -rn 'fire degraded\|degraded fire' .claude/skills/mission-control/ tools/launchd/*.sh \| grep -v gate-5-retro` | one hit only, `mission-control.sh:2264` (an unrelated digest comment), which also serves as the positive control |
| V5 | No record has ever been promoted | `ls ~/.ailang/state/ \| grep -E 'idle-fingerprint\|slot-verdicts'` | `mission-stapledon-idle-fingerprint.pending`, `mission-world-idle-fingerprint.pending` and 7 `*-slot-verdicts.log` files (the control); no bare `-idle-fingerprint` file |
| V6 | `--record` promotes and never recomputes | `cat -n tools/launchd/lib/idle-fingerprint.sh` lines 37-52 | reads `$pend_file`, checks `fire=$fire`, `mv` to `$fp_file`, `exit 0` before any read |
| V7 | No lane is a hash input | `grep -n '^hash=' tools/launchd/lib/idle-fingerprint.sh` | `150:hash=$(LC_ALL=C sort "$tmp/head" "$tmp/ledger" "$tmp/directives" "$tmp/inbox" "$tmp/red" …` |
| V8 | The snapshot is taken before the controller; the pre-check has 24 h expiry | `grep -n 'IDLE' tools/launchd/mission-control.sh` then `sed -n 1735,1790p` and `2355,2378p` | PRE-CHECK 1735-1787 (`_mi_age -gt 86400` → full fire); SNAPSHOT 2357-2377, "before the controller starts" |
| V9 | The ledger is read from origin/<branch> | `cat -n …/idle-fingerprint.sh` line 80 | `git -C "$work" show "origin/$branch:$doc"` |
| V10 | World did not record because of the bar; no roles were spawned | `git -C ~/dev/sunholo-data/ailang-world grep -n 'Idle fingerprint: NOT recorded' origin/mission/world-iter370-record -- design_docs` | lines 1613 and 1656 as quoted above; 1656's entry says "Designer, planner and executor were not spawned, because there was no pick" |
| V11 | Cost of a full fire | `grep -E '2026-11-0[789]' ~/.ailang/state/mission-world-slot-verdicts.log` | 8 COMPLETED rows, elapsed_s 8931, 6080, 5035, 6050, 4739, 4769, 4104, 3605; plus PAUSED rows of 137-271 s |
| V12 | World OPEN rows: dev vs branch | `mission_decisions.sh --open --file` on `origin/dev:` and `origin/mission/world-iter370-record:design_docs/world-mission.md` | dev `D-WORLD-105 110 111 112`; branch adds `D-WORLD-113` |
| V13 | HD-2a list and fleet write scope | `sed -n 77,140p design_docs/fleet-mission.md` | "routing, lane order, quota or ration thresholds, or a billing guard"; may change `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, `tools/launchd/**` |
| V14 | No ruling on the idle record or the degraded bar | `mission_decisions.sh --all --file design_docs/fleet-mission.md \| grep -iE 'idle'` (and `degrad`); control `… \| wc -l` | idle: D-FLEET-7 (CPU threshold), D-42, D-43 (fleet self-skip); degrad: D-31, D-42; none about this bar; control 42 rows |
| V15 | The mirror is byte-equal today | `cmp .claude/…/gate-5-retro.md .agents/…/gate-5-retro.md && echo MIRROR-EQUAL; sync-agents-skills.sh --check` | `MIRROR-EQUAL`, rc=0 |
| V16 | The idle test does not reference gate-5 yet | `grep -n 'gate-5\|degraded\|retro' tools/launchd/test_mission_idle_fingerprint.sh`; control `grep -rn 'gate-5-retro' tools/launchd/*.sh` | first: no output; control: `test_agents_skills_sync.sh:42,58`, `test_mission_heartbeat.sh:218,227`. `grep -c 'idle-fingerprint.sh --record'` → 0, because the text has a quote: `idle-fingerprint.sh" --record` |
| V17 | The mutation harness has 12 mutants over DRIVER and HELPER copies | `sed -n 1,30p tools/launchd/test_mission_idle_fingerprint.sh` | `for mutant in excludes wrong-record … snapshot-keeps-stale` (12); `cp "$DRIVER" …; cp "$HELPER" …` |
| V18 | The test file runs in the launchd target | `grep -n test_mission_idle_fingerprint make/test.mk` | `120:	tools/launchd/test_mission_idle_fingerprint.sh \` |
| V19 | The revised gate-table block works, with a positive control, a negative control and a failure control | the replacement block saved as `/tmp/i97-block.sh` (ids taken from `$IDS`), run with `/bin/bash` against `MISSION_WORKDIR=~/dev/sunholo-data/ailang-world` and `MISSION_DOC=design_docs/world-mission.md`, ids `D-WORLD-111 D-WORLD-112 D-WORLD-113`; then again with `MISSION_WORKDIR=/nonexistent` | world: `branch: dev open rows: 4`, `OK D-WORLD-111`, `OK D-WORLD-112`, `MISS D-WORLD-113`. Bad workdir: `BRANCH/CHARTER READ FAILED: do not record`, `branch: NONE open rows: 0`, `MISS D-WORLD-111`. World's `symbolic-ref` printed `origin/dev`, rc 0, so the rc-1 fallback to `dev` was not exercised |
| V20 | Standing rule 8's clock wording; no scheduled-duty rule exists | `grep -n -i 'clock' .claude/skills/mission-control/SKILL.md`; `grep -rn -i 'scheduled duty' .claude/skills/mission-control/`; control `grep -c 'PARKED-ON-LANE' .claude/skills/mission-control/SKILL.md` | `567:   park unblocks on a **clock**, needs no ask, and must NOT appear there at all.`; scheduled duty: no output (rc 1); control `2` |

**Not verified:** whether World's rows 150 and 151 could be mapped to existing OPEN rows (the
attended "A1" step was not traced). Stapledon's recent fires were not read. No test was run, because
this is a design only.
