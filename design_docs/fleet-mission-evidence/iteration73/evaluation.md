# Iteration 73 evaluation (independent, read-only, pin dd1832174)

Method: /usr/bin/grep under /bin/bash; every negative paired with a positive control.

## Claim verdicts
- C1 VERIFIED. All 8 sites present: .claude 128/137/154/163, .agents 125/134/153/162 (`grep -E "^\d+\." || echo ""`; control `grep -c grep` = 7 in .claude copy shows the instrument sees). Failure mode = empty result, not abort (the `|| echo ""` is on each line).
- C2 PARTLY (BSD half VERIFIED; GNU half UNMEASURABLE, documented claim sound).
  Measured: `grep -V` = "BSD grep, GNU compatible 2.6.0-FreeBSD"; `type grep` = /usr/bin/grep; input "1. design_docs/x.md (0.85)\nd. foo".
  `-E "^\d+\."` -> matches line 1 only, rc 0; `-E "^[0-9]+\."` -> same line; control `-E "^d+\."` -> "d. foo" (so BSD treats \d as a digit class, not literal d: the test discriminates). No ggrep/gnugrep/docker; `ls /opt/homebrew/bin | grep -i grep` shows only lz*/xz*/zstd*/msggrep/pcre2grep, no GNU grep. GNU ERE `\d` = literal d (3.8+ warns "stray \ before d") is correct per GNU docs; unmeasured here.
  Wording issues: "on the rig" generalises from "this Mac" (the same machine id was not proven to be the rig); "cloud executors ... return empty" is inference (Linux grep unmeasured); the ticket says it fails on "BSD/GNU" and the draft should say outright that the BSD half of the ticket is REFUTED on this machine.
- C3 VERIFIED. `git show b51353bcd` (2026-10-03T07:16:00Z): `-  { echo ...; } | grep -E "^[0-9]+\." | \ awk '{...}' | head -5` -> `+ ... | \ awk '/^[0-9]+\./ {...}' | head -5`. .agents lines 116 (grep|awk|head), 141-144 and 169-170 carry `|| true`. Nit: "line 141" is the comment start; the call is 144 (and PLANNED at 170). Also note #1542's subject is "project root from git"; the awk change is a side effect, not its stated purpose (the draft says "moved the filter" which is accurate, but "built" for ask (b) is incidental).
- C4 VERIFIED. pre-push `_scope_is_harness`: only `.claude/skills/mission-*|sprint-*`, `.agents/skills/mission-*|sprint-*`; `_scope_is_fleet_support` has no skill path (design_docs/*, changelogs/*, docs/*, ... only). `_scope_verdict`: fleet refuses anything not in harness|support. Queue row 25 residue line (fleet-mission.md:264) confirmed. Ask (c) is ailang_bootstrap: not in this repo.
- C5 VERIFIED. D-FLEET-5 (RULED, scripts/test_mission_* into harness) and D-FLEET-6 (RESOLVED, sandbox into support) are both Mark rulings. Mechanism claim correct: for non-fleet missions `_scope_is_harness` -> refuse (pre-push line ~60); support is fleet-only.
- C6 VERIFIED. #1688 face88a3f 2026-10-09T07:27:27Z created resources/unattended_runs.md (stat shows the file), line 6 "**Commit after each milestone** ... Uncommitted work is lost work"; #1905 3f9a0c092 (2026-10-20) adds "## Cloud tasks: push every milestone" (line 23) with cloud_branch_sync.sh. Both ancestors of HEAD. Ticket filed 06:17:36Z -> ~70 min. Caveat to state: this is a skill-doc rule; whether prod executors obey it is not measured (it is a ghost of the ask as written, not proof of behaviour).
- C7 PARTLY. Cap and flag verified (daemon.go:47,54; worktree.go:17,66,103-104; coordinator_lifecycle.go:43-49). MISSED, material: `daemon_tasks_exec_run.go:160-170` already handles `ErrWorktreeLimitReached` by resetting the task to PENDING for retry on the next poll ("will be retried when a worktree frees up"); `git log -S"resetting to pending for retry"` dates it to Jan 2026 (pre-dates the 10-08 incident). So "make evaluators queue" is not simply unbuilt: queue-on-limit exists and the ticket still saw `failed`. The draft's "making evaluators queue is a coordinator code change" is unverified and possibly wrong; the real question is why a reset-to-pending task surfaced as failed (handoff path? retry cap?). Also "raising the prod cap is a deploy-config change (multivac)" is unverified: there is no max_worktrees key in any repo yaml/tf (`grep -rni max_worktrees` over go/yaml/yml/tf finds only Go), and nothing here shows prod passes `--max-worktrees`. The cloud-lane ticket calls the cap "hard-coded".
- C8 VERIFIED (with wording fix). Cloud-lane ticket (first_seen 2026-10-12) cites daemon.go:47 and Mark's 10-12 quote and lists sprint-evaluator:worktree-cap-and-exit1 as Related; grep of fleet-mission*.md for both signatures: cloud-lane = 0 hits (control: agents-copies-stale-non-mission = 1 hit), worktree-cap = 0 hits. It also cites enforcement at daemon_tasks_exec_run.go:160, which the draft omits. Note the cloud-lane ticket's ask (a) (cloud lane must not take a local worktree slot) is a different fix from "queue", so "one fix for one cap" is a recommendation, not a fact.
- C9 VERIFIED as stated ("not measured"). Ask 2 is prod Cloud Run/coordinator investigation; no fleet-scope producer. The draft is honest that it was unmeasured.

## Parked wrongly?
- Record 1: no in-allowlist fix exists. I checked the hook: no design-doc-creator path in either predicate; design_docs/* support cannot host the script. Parking is right. Ghost-check: it is not a ghost for the `\d` ask on GNU; only the BSD claim in the ticket is refuted.
- Record 2: ask 3 ghost is right on content and dates (above). Asks 1-2 outside scope is right (internal/coordinator/** not in allowlist). But the draft should not imply ask 1 is a plain "unbuilt" feature (C7 gap).

## Recommendations
- D-FLEET-36 option A (add both design-doc-creator paths to `_scope_is_fleet_support`): sound and the mechanism consequence is accurate (support = fleet only; harness would refuse product loops). Omitted consequences: (i) it widens the fleet to the whole skill (SKILL.md, templates, move_to_implemented.sh), not just the 8 lines; say so; (ii) the pre-push hook edit is itself in tools/launchd/*, which IS fleet-writable, but the draft should say who makes the edit (the fleet, after Mark's ruling) and that Authority text in fleet-mission.md must be updated in step ("Keep in step"); (iii) ask (c) not fixed, so the daily bootstrap sync may revert on cloud copies; the "impact is Linux only" default statement is then not "nothing changes" for cloud users. Option B is accurate and arguably cheaper (8 lines, one attended commit); say that, as the fix is trivial relative to a scope ruling.
- D-FLEET-37 option A: ownership split is sound, but folding ask 1 into the cloud-lane ticket is premature until the reset-to-pending finding is explained: they are different fixes (cap exemption vs queueing vs visible failure). Reword to "route ask 1 to the owner of the cloud-lane ticket, noting that the daemon already resets limit-reached tasks to pending". Option B rejection reasoning is fine.

## Overstatement list
1. "does NOT reproduce on the rig" -> "does not reproduce on this Mac's BSD grep (hostname/rig identity not verified)".
2. "so on Linux ... the four searches return empty" -> mark as inference from GNU docs, unmeasured.
3. "making evaluators queue is a coordinator code change" -> unverified; code already requeues (see C7).
4. "raising the prod cap is a deploy-config change (multivac)" -> unverified; no multivac file inspected.
5. "Both are ancestors" fine; C6 "ghost" should add "behavioural effect on prod executors not measured".

## Scores
- Record 1 (D-FLEET-36): 82/100 PASS-WITH-EDITS. Strongest objection: the headline premise (BSD defect) is refuted here and the GNU half is unmeasured, so the record rests on documentation; and the "default: nothing changes" ignores cloud-synced copies.
- Record 2 (D-FLEET-37): 66/100 FAIL as drafted, passes after edits. Strongest objection: C7 misses that the coordinator already resets limit-reached tasks to pending, so the central characterisation of ask 1 ("evaluators fail instead of queue; queueing needs new code") and the recommendation to fold it into the cloud-lane ticket are unsupported.

## Required edits
1. Record 1 C2: replace first sentence with "Measured on this Mac only: BSD grep 2.6.0-FreeBSD treats `\d` as a digit class (control `^d+\.` matched only the literal-d line), so the ticket's 'BSD/GNU' claim is refuted for BSD. GNU grep was not available; the GNU behaviour (`\d` = literal d) is documented, unmeasured."
2. Record 1: in option A add "widens the fleet to the whole design-doc-creator skill; Authority text and pre-push comments updated together"; in Default add "bootstrap-synced cloud copies unaffected until ask (c)".
3. Record 2 C7: add "`daemon_tasks_exec_run.go:160-170` already resets ErrWorktreeLimitReached tasks to pending (present since Jan 2026); why the 10-08 evaluator handoffs surfaced as failed is the open question"; downgrade "queue is a coordinator code change" to "unverified"; downgrade the multivac claim to "assumed; not inspected".
4. Record 2 option A: reword fold as above (route, do not assert one fix).
5. C6: append "prod executor behaviour not measured".

## Round 2

Source: records-as-revised.md. Re-check: `git show -s debf5b66e` = 2026-01-01T22:07:27+01:00 "Fix coordinator workflow issues"; its diff adds `ErrWorktreeLimitReached` and the `errors.Is(wtErr, ErrWorktreeLimitReached)` retry branch (diff lines 9, 79, 455-467). Controller's date is correct and sharper than my "Jan 2026".

### Edit-by-edit
1. Record 1 C2 sentence: LANDED, wording matches (BSD-only measurement, control, GNU unmeasured, "refuted for BSD").
2. Record 1 option A / default: LANDED ("whole skill, not eight lines", "Authority section and pre-push comments change together", "bootstrap-synced cloud copies unaffected until ask (c)").
3. Record 2 C7: LANDED. Cites :163-170 and debf5b66e (verified), says why-failed is the open question, queueing code "unverified", prod `--max-worktrees` "assumed, not inspected". Accurate.
4. Record 2 option A reworded as "route ... start from why reset-to-pending did not hold": LANDED, and correctly notes the cloud-lane ticket asks for a different fix.
5. C6 caveat "Prod executor behaviour since then is not measured": LANDED.

### Remaining problems (new or residual wording)
- R1 title/headline still asserts "ask (a) is real on GNU grep only" as fact. The body (correctly) says GNU is unmeasured. A headline stronger than the body is an overstatement. Replace with: "ask (a) is real at HEAD (8 sites) and fails, per documented GNU semantics, only on GNU grep (unmeasured here)".
- R1 default clause "Linux duplicate searches stay silently empty": same issue. Replace with "Linux duplicate searches are expected (documented, unmeasured) to stay silently empty".
- R1 "an independent sonnet judge verified it (PASS-WITH-EDITS 82, edits applied)": the judge did not verify the GNU half and scored with required edits; say "an independent sonnet judge scored it PASS-WITH-EDITS 82 (GNU half unmeasurable); edits applied". Same for R2: "scored the draft FAIL 66 because it missed existing retry code; edits applied, re-judged in round 2".
- R2 Recommendation A ends "The fleet then resolves this ticket." Ask 2 (exit-1 root cause) is NOT covered by the cloud-lane ticket (its asks are (a) exempt cloud from local slot, (b) max_concurrent_tasks governs cloud, (c) what held task-06dc18ea/85d95401). Resolving here would drop ask 2. Required replacement: "The fleet resolves this ticket only once ask 2 has a home: either V1/attended confirms it is added to the routed ticket, or ask 2 is filed as its own ticket (signature of Mark's choosing)." Also say what "V1/attended owns asks 1 and 2" means operationally (who files).
- Minor: R2 title says asks 1 and 2 "are in the coordinator"; ask 2's cause is unmeasured, so it may not be. Use "appear to be in the coordinator (ask 2 unmeasured)".

### Scores
- D-FLEET-36: 88/100 PASS-WITH-EDITS (the three wording edits above; none changes the recommendation).
- D-FLEET-37: 84/100 PASS-WITH-EDITS (ask-2 orphaning is the one substantive edit). Strongest objection: closing the ticket after routing would silently drop the exit-1 root-cause ask, the only ask with no owner and no measurement.
Both rows may be committed once the listed replacements are applied; no re-judge needed for pure wording changes.
