# Iteration 145 — independent judge (sonnet, Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only)

Subject: branch `fleet/i145-idle-fp-nightly-rig` ([#2481](https://github.com/sunholo-data/ailang/pull/2481)) on origin/dev `9b35c44d0`.
The brief named the ticket text, the sources and six questions; it did not carry the controller's verdict.

## Round 1 — `d1c8d568b` (helper + test + skill phrase + changelog)

**VERDICT: PASS 82/100.** Blocking: none. 64,766 tok, 10 tool uses, 380 s.

Non-blocking:
1. The `startswith("The nightly ")` filter is a convention: a loop comment that itself begins "The nightly " would wake the next fire. Real loop comments on #392 start "Triage…", "Mission iteration…", "Fixed by…", "Iteration 95…"; none match today.
2. An open `nightly-rig` issue with no matching report makes every read die; the driver then logs "cannot read idle fingerprint … proceeding with a full fire" (`mission-control.sh:1790`). Fails safe, costs spend; it is the intended format-drift alarm.
3. The summary line has no `nightly=N` counter (cosmetic).
4. Race between the issue's `.comments` count and the comments fetch dies with "incomplete or changed comment list"; correct fail-loud.

Mutation drills (scratch copies under `$TMPDIR` via `IDLE_TEST_HELPER`):

| Mutant | Result |
|---|---|
| widen (`"The nightly "` → `"The "`) | SURVIVED |
| nopr (drop `pull_request` filter) | SURVIVED |
| nocount (drop count check) | KILLED — `FAIL failed read accepted rc=0` |
| nodrift (format-drift error → `empty`) | KILLED — `FAIL failed read accepted rc=0` |
| narrow (→ `"The nightly make test"`) | SURVIVED — real #392 also carries `make golden` reports |
| nouniq (drop duplicate-id check) | SURVIVED |

Answers: (1) fixes the ticket against the producer's real format; (2) triage comments do not move the hash, repos with no open issue hash byte-identically, closing restores the legacy hash; (3) every failed/partial read dies, the `while read` runs in the main shell, `gh api` without `--input` does not consume the loop's stdin, Bash 3.2-safe; (4) the fixed `nightly-rig` label convention is acceptable because the producer hard-codes it; (6) all paths in fleet Authority, the two `gate-5-retro.md` copies `cmp` 0.

Strongest objection: the test would not catch a changed prefix, a dropped PR filter or a dropped duplicate-id check.

## Round 2 — `ccee1caca` (test-only hardening by codex, helper unchanged)

**VERDICT: PASS 88/100.** Blocking: none. 78,553 tok, 9 tool uses, 204 s (resumed by `SendMessage`, same agent).

| Mutant | Result |
|---|---|
| widen | KILLED — `FAIL nightly triage comment moved hash` |
| nopr | KILLED — `FAIL nightly pull request triggered a comments read` |
| nocount | KILLED — `FAIL failed read accepted rc=0` |
| nodrift | KILLED — `FAIL failed read accepted rc=0` |
| narrow | KILLED — `FAIL nightly golden failure comment did not move hash` |
| nouniq | KILLED — `FAIL nightly duplicate comment id was accepted` |
| noopened (new: `$opened or length>0` → `length>0`) | KILLED — `FAIL new nightly issue read: … comments invalid` |
| nostate (new: `state=open` → `state=all`) | KILLED — unexpected gh api call |
| nowhole (new: `whole` checks → `true`) | SURVIVED |
| idonly (new: hash row drops the comment id) | SURVIVED |

Non-blocking:
1. `nowhole`: nothing tests a non-integer or negative `.number`/`.comments` (defence in depth).
2. `idonly`: no test pins that comment ids, not just their count, are hashed; a deleted-and-replaced report in one window would be missed by a simplified row. The helper keeps the ids, so no bug today.
3. `nightly_with_pr=$(hash); rc=$?` captures `cut`'s status, so its `rc -eq 0` assertion is vacuous; the following hash-equality assertion still fails on an empty hash.
4. `BAD_READ_FAIL` is a one-call prefix; other `bad` calls keep their message.

Strongest objection: no test pins id-hashing over counting (`idonly`), and the PR arm's `rc` assertion is vacuous.
