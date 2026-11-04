# Iteration 83 — evaluations (sonnet, Agent tool, foreground)

## Gate-2 idle-verdict judge: REFUTED (narrow)

The controller claimed that nothing was buildable. Its reasoning: 25 open tickets, the newest first seen 10-28; every ticket maps to an OPEN row; no ruling arrived; and every in-scope code recommendation already has a diff (#2108, #2082, #2067, #2065, and the D-FLEET-33 row diff).

- **C1 confirmed.** 25 tickets. The newest `last_seen` is 11-03T21:28Z, the daneel-reports recurrence that iteration 82 already recorded.
- **C3 confirmed.** The only charter commit since `1b5cef0ae` is the i82 record merge. The four drafts have 0 comments and 0 reviews. #2039 has 7 comments and #1945 has 20, all from the bot, so the zero for allowlisted authors was measured rather than assumed.
- **C5 confirmed.** 18 check-runs at `aebc3dcbb`, none non-green.
- **C4 refuted.** D-FLEET-18 option A is the recommendation, and its first step is a `_scope_is_fleet_support` arm for the two `tools/eval` probe files. That is in-scope code (`tools/launchd/githooks/pre-push`, the scope suite and the charter's Authority line), it is not policy-class under HD-2a, and no diff for it exists. Iteration 82's pick screen had put row 18 in "outside fleet scope or no common step", but it is the same guard-arm-first shape as D-FLEET-36 A and D-FLEET-38 A, which got a draft.
- **D-FLEET-41 B** (skill text in the sprint-evaluator copies) is a weaker second candidate. It is not the recommendation, and iteration 76 moved that row's default from B to C on purpose, so it is not taken.
- **Strongest objection:** the screen asked "is the target path in scope today?" when the question should have been "does the recommended option *begin* with an in-scope guard arm?".

The controller verified this first-party. The pinned `pre-push:29-36` admits no `tools/eval/*` path. `git log origin/dev --grep D-FLEET-18` returns only record commits plus #1824. No open PR is for row 18.

## Diff judge: PASS 93 (`875581f46`), with no required edits

The drills ran in a `mktemp -d` scratch copy, with the hook found through the copy's own `$HERE`. Nothing was written to the worktree.

| Mutation | Suite rc | Failing arm |
|---|---|---|
| none (positive control) | 0 | — |
| drop path 1 | 1 | `fleet tools/eval/motoko_connection_probe.sh`: got refuse, want allow |
| drop path 2 | 1 | `fleet tools/eval/test_motoko_connection_probe.sh`: got refuse, want allow |
| widen to `tools/eval/*` | 1 | `fleet tools/eval/qwen38-motoko-rerun.sh`: got allow, want refuse |
| path 2 suffix glob | 1 | `fleet …test_motoko_connection_probe.sh-x`: got allow, want refuse |
| path 1 suffix glob | 1 | `fleet …motoko_connection_probe.sh.bak`: got allow, want refuse |
| path 1 basename-prefix glob | 1 | `fleet tools/eval/xmotoko_connection_probe.sh`: got allow, want refuse |
| delete the new arm | 1 | `fleet tools/eval/motoko_connection_probe.sh` |
| move both paths into `_scope_is_harness` | 1 | `motoko tools/eval/motoko_connection_probe.sh`: got refuse, want allow |

- **Premise.** At base, the guard refuses both paths for fleet and allows them for v1 and motoko. Both files exist. The diff touches exactly four files and nothing under `tools/eval/`.
- **Eval exposure.** No eval runtime path (`internal/eval*`, `cmd/ailang/eval*`, `make/eval.mk`, `Makefile`) references either probe. The probe runs `ailang eval-suite` (`motoko_connection_probe.sh:236`); nothing in the eval runtime runs the probe.
- **Changelog fragment and commit message.** Neither contains a closing keyword followed by `#N` or a personal email. `mission_decisions.sh --check` passes with 40 rows valid.
- **Minor findings, not required edits.** The two `v1` arms are redundant with the `motoko` arms, since motoko fails first under a move-to-harness mutation. No arm covers a `tools/eval/test_*` widening beyond the `-x` suffix trap.
- **Strongest objection.** The filenames are hard-coded, so a renamed or new probe file needs another guard change and another ruling. That is by design under option A, but nothing in the commit says so.
- **Unmeasured.** A merge dry-run against #2108, and a real fleet push of these two paths (the end-to-end arm pushes a `tools/launchd` path).
