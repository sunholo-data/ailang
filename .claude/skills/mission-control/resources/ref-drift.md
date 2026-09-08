# Shared-clone ref drift

`origin/dev` is a moving observation, not an iteration-long constant. Every mission worktree on
this rig shares its clone's `.git` ref store, so a sibling mission or attended session can advance
`refs/remotes/origin/dev` by fetching. The current controller receives no notification and need not
have run `git fetch` itself. A later gate can therefore see a different SHA even though the earlier
read was correct when taken.

## The two measured instances

- V1 iteration 331 inherited a concurrent advance while the iteration was running. The new dev red
  surfaced only because it stranded Gate 4; without that side effect, the loop would have continued
  on a base it had not reviewed.
- V1 iteration 332 read one SHA at Gate 1, then created its worktree minutes later at a SHA four
  commits ahead. No V1 fetch occurred between those actions. A sibling fetch in the shared clone had
  moved the remote-tracking ref under the point-in-time reading.

This is silent because worktrees share remote-tracking refs but have separate working trees. The
current worktree's command history and `FETCH_HEAD` do not explain a sibling's update, and an
unrecorded earlier SHA leaves no prior against which the later value can be classified.

## Record and compare protocol

Use `tools/launchd/mission-base.sh`; do not quote a base with an ad hoc short SHA.

1. Gate 1 fetches, performs its existing sync checks, then runs `record gate1`. The helper reads the
   full SHA once, pairs it with its UTC read time, and appends `base-gate1` to the dedicated
   `$AILANG_STATE_DIR/mission-${MISSION_NAME}-base` file.
2. Immediately before a base-dependent action, run `snap`, extract the full SHA, and compare it with
   `last gate1`. `snap` is read-only and never fetches.
3. If the values disagree, run `drift gate1`. Re-read once, classify a persistent mismatch as
   `DRIFT`, and re-run the affected gate against the fresh SHA. A benign advance is not an operator
   error and does not itself require an abort.
4. Abort or park only when the movement invalidates the action's integrity: for example, when the
   worktree/provenance no longer identifies the commit whose output was reviewed. Missing Gate-1
   evidence is also an integrity failure (`drift` exit 2), not benign drift.
5. Carry the `<full-sha><TAB><iso>` pair as `base=$base` in worktree provenance and the iteration's
   Routing-evidence row. The worktree must be created from that full SHA, never by re-reading
   `origin/dev` in the `git worktree add` command.

The base record must never be written to `mission-${MISSION_NAME}-heartbeat`. The launchd driver
classifies a slot from the heartbeat's last label; a trailing `base-*` row would turn a normal
mid-gate reap into `CRASHED` and corrupt the diagnostic stamp count.

## Non-vacuity recipe

The CI arm in `tools/launchd/test_mission_base.sh` performs this with a scratch clone. To re-prove
the mechanism manually, use a scratch clone whose `HEAD` is commit B and whose
`refs/remotes/origin/dev` initially names commit A:

```bash
export AILANG_STATE_DIR="$(mktemp -d)"
export MISSION_NAME=test

bash tools/launchd/mission-base.sh record gate1
bash tools/launchd/mission-base.sh drift gate1        # control: exit 0, steady

# Simulate a sibling fetch by moving the exact shared ref read by snap; do not merely commit on HEAD.
git update-ref refs/remotes/origin/dev HEAD

bash tools/launchd/mission-base.sh drift gate1        # exit 1; prints old -> new DRIFT
```

Also prove both missing-record paths: with the base file absent, and with it present but containing
no `base-gate1` row, `drift gate1` must exit 2 and print `no base-gate1 record yet`. These controls
distinguish a live comparison from a vacuous grep or a false DRIFT with an empty old SHA.

All snippets are Bash 3.2 compatible: no associative arrays, `${value,,}`, or GNU `timeout`.
