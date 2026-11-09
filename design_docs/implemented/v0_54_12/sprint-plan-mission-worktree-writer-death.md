# M-FLEET-WORKTREE-WRITER-DEATH — terminal verdict after a writer dies

Ticket: `worktree:native-background-terminal-marker-missing`; mission world iteration 268,
filed 2026-10-15; fleet planning iteration 42.
Verified HEAD: `04496274e36d301b721886fb56ab1146ad4080e7` (2026-10-16). Initial worktree clean.
Scope: planning only. Controller owns approval, execution handoff, git writes and landing.
There is no separate design doc: the controller requested this small mechanical repair directly.

World evidence, verbatim:

> Native shell: authoritative mission-worktree.sh add sprint/world-iter268-row150-m2a <sha> sibling/.wt-world-iter268 returned rc0/path, then native exec task ended. Separate wait45 returned pending; path absent; branch exists; status file pending; no terminal helper marker observed. Process listing had wait helper, no matching creator. Fresh sibling path/branch via same helper add followed immediately by bounded wait in SAME active exec returned ready.

Controller reproduction supplied with the ticket (outside any sandbox, macOS `/bin/bash` 3.2,
same HEAD): scratch repo, post-checkout sleeps 6 seconds, scratch `MISSION_WORKTREE_STATE`.
Control `add` then `wait 20` reached ready/0. Treatment started add under
`perl -e 'setpgrp(0,0); exec @ARGV' /bin/bash -c '...add...'`, then sent TERM to the caller's
process group after one second. The nohup writer died, leaving a branch and pending status;
every subsequent `wait PATH 5` returned pending/75. These are controller-supplied observations,
not planner-run experiments. `nohup` ignores HUP, not process-group TERM.

## Premise checks at HEAD

Evidence is from numbered `git show HEAD:<path>` blobs. No process-killing reproduction or
socket-touching suite was run by the planner.

| Claim | file:line at HEAD | Verified yes/no |
|---|---|---|
| Helper is 65 lines and advertises ready/0, failed/1, pending/75 | `tools/launchd/mission-worktree.sh:1`, `:5`, `:6`, `:65` | Yes |
| State defaults to real HOME unless explicitly overridden; path refusal precedes checkout | `tools/launchd/mission-worktree.sh:20`, `:28` | Yes; every fixture must override state into scratch |
| add writes bare pending, then launches a nohup bash background job with no session/group separation | `tools/launchd/mission-worktree.sh:31`, `:33`, `:46` | Yes; no writer PID/birth record exists |
| Writer's cleanup removes worktree forcibly, removes remaining path, prunes, deletes branch | `tools/launchd/mission-worktree.sh:35`, `:36` | Yes; reuse exactly this sequence for writer-death recovery |
| Writer bounds checkout and checks cleanliness before writing terminal status | `tools/launchd/mission-worktree.sh:37`, `:39`, `:42`, `:43`, `:45` | Yes |
| wait consumes terminal files, otherwise only polls/time-bounds; no liveness check | `tools/launchd/mission-worktree.sh:52`, `:55`, `:57`, `:58`, `:60`, `:61` | Yes; dead writer is indistinguishable from progress forever |
| Existing suite has five scenario groups: happy, /tmp refusal, timeout, dirty tree, unknown path | `tools/launchd/test_mission_worktree.sh:17`, `:22`, `:26`, `:35`, `:43` | Yes; preserve all assertions, not just group headings |
| Suite uses scratch state and slow git shim/post-checkout hook; /tmp scratch is skipped | `tools/launchd/test_mission_worktree.sh:10`, `:11`, `:12`, `:28`, `:36` | Yes; a skipped Linux run is not coverage |
| Suite is already wired into test-launchd-drivers | `make/test.mk:68`, `:70`, `:89`; `tools/launchd/lib/suite-env.sh:12`, `:18` | Yes; wrapper passes selected host variables through env -i to /bin/bash |
| Claude Gate 3 loops pending indefinitely across bounded calls, then parks on failure | `.claude/skills/mission-control/resources/gate-3-route.md:624`, `:625`, `:626`, `:627`, `:629` | Yes; no call-site change is needed |
| Agents Gate 3 has the same authoritative helper/call sequence | `.agents/skills/mission-control/resources/gate-3-route.md:624`, `:625`, `:626`, `:627`, `:629` | Yes; both callers automatically consume failed:writer_died |
| CI selects the shell lane for tools/launchd changes | `.github/workflows/ci.yml:77`, `:82`, `:83` | Yes |
| CI job launchd-drivers runs the target on macOS, guards bash major 3 | `.github/workflows/ci.yml:747`, `:750`, `:756`, `:762`, `:765`, `:769` | Yes; intended rig shell is 3.2, guard checks major only |
| Linux jobs exist with bash 5 documented, but also run this shell target | `.github/workflows/ci.yml:97`, `:750`, `:769` | **No:** Ubuntu exists and bash 5 is documented; the sole target invocation is the macOS job. Require a separate Linux host run, not an invented matrix |
| CI explicitly provisions/verifies Perl POSIX::setsid or a setsid executable on both OSes | `.github/workflows/ci.yml:759`, `:762`, `:768` (job steps); workflow-wide search | **No:** neither is guaranteed by this config. Runtime capability detection and a bash fallback are required |
| Existing driver already uses ps lstart to guard against PID reuse | `tools/launchd/mission-control.sh:394`, `:402`, `:407`, `:415` | Yes; reuse the identity convention, without coupling this small helper to the driver |
| Charter requires actual-surface/reach, healthy/degraded dry-runs and suite green | `design_docs/fleet-mission.md:115`, `:122`, `:128`, `:132` | Yes; these are controller gates |

## M1 — detach the writer and terminalize abandoned pending state

One milestone, `M1`, no dependencies. Estimate **300 added LOC**, about **90 minutes plus
30 minutes buffer (0.25 day)** for implementation, focused tests and mutation checks; independent
evaluation and controller host/rollout gates are separate. Risk: medium, because process identity,
publication races and macOS/Linux process groups need real-host evidence.

| Executor file | Prescriptive change | Estimated added LOC |
|---|---|---:|
| `tools/launchd/mission-worktree.sh` | Shared cleanup, owner metadata/publication, liveness and terminal re-read, detached launch and fallback, usage header | 110 |
| `tools/launchd/test_mission_worktree.sh` | Named scratch-only lifecycle, process-group, race, identity and fallback arms; retain five controls | 180 |
| `changelogs/unreleased/2026-10-16-mission-worktree-writer-death.md` | Fragment opening `### Fixed — mission-worktree.sh reports a dead writer` | 10 |

The original helper/suite landed together in `9c9b7a655` (65 + 47 lines); there are no intervening
changes to those paths. This is a task-size estimate, not an asserted measured LOC/day history.
The JSON's 1,200 LOC/day is 300 / 0.25, a planning capacity equivalent only. Keep the helper small;
neither touched shell file approaches the 800-line limit, so no companion module is needed.

Registry reuse M1: **none**. No package-like capability is introduced; registry search is
inapplicable. Reuse this helper, its cleanup sequence, its existing shell fixture seams and the
repository's ps birth-time convention. No `.ail` code or language example is needed; named shell
fixtures are the executable examples. No routing, quota, threshold, billing, scheduling or caller
policy changes. Executor modifies only the three files above; this planner creates only plan + JSON.

Implementation order in the single session:

1. Add named regressions first, with bounded rendezvous files rather than sleeps as proof of
   readiness. Record the pre-fix reds for dead writer and caller-group teardown; retain the existing
   five green scenario groups. Build all repos, hooks, shims, state, logs and mutation copies inside
   each test's scratch tree.
2. Extract one cleanup function within the helper, callable by both writer and wait, preserving
   `git -C REPO worktree remove --force WT`, `rm -rf WT`, `git -C REPO worktree prune`,
   `git -C REPO branch -D BRANCH` in that order. Preserve best-effort cleanup semantics. Store the
   original repo and branch with ownership metadata so wait need not infer a branch or run in the
   repo that initiated add. Parse data as data, never source/eval the metadata.
3. Publish a sibling owner record alongside the existing status file: writer PID, normalized
   `LC_ALL=C ps -p PID -o lstart=` birth time, per-launch token, repo and branch. Keep public status
   words unchanged. Publish complete records by a temporary sibling file + rename. Include the
   unique launch token in writer argv and use full-width `ps -ww -o command=` when checking it;
   birth-time comparison prevents ordinary PID reuse, and the token covers reuse within lstart's
   one-second resolution. Never use kill -0 alone as proof that this is the original writer.
   Command/token is supplementary to PID + birth, not a replacement for birth-time comparison.
4. Close startup races: capture `$!` for the actual exec-replaced writer, publish its ownership
   before add prints PATH, and hold checkout behind a bounded registration rendezvous until the
   parent has published metadata. Metadata must identify the final bash writer across launcher
   exec (not an intermediate shell that exits). Wait must not see a new successful add with an
   unowned pending file. If the child dies during registration, retain enough ownership context
   for wait to terminalize it; do not create a second permanent pending case. All registration
   waits are short and bounded, never tied to checkout duration. Legacy bare pending without
   ownership is not permission to delete an unidentified tree; preserve pending conservatively.
5. In wait, handle ready/failed first exactly as now. On pending, compare the recorded identity
   with the current process. A matching, running writer is alive: perform **no cleanup**, then
   honor MAX and return pending/75 if the bound expires. Zombies count as dead even if kill -0
   succeeds. If process inspection is inconclusive while a PID still exists, return pending,
   never infer death from a transient ps failure. A missing PID or proven identity mismatch
   means the original writer is dead; do not signal an unrelated reused PID.
6. After detecting death, **re-read status once before cleanup**. If ready or failed appeared,
   consume that terminal result normally. Only unchanged pending for that owner permits cleanup
   and `failed:writer_died`, rc 1; consume status/owner files like existing terminal results. Recheck
   owner identity if the record changed. This prevents a normal completion between the initial
   read and liveness check from deleting a valid tree. Tests must force this interleaving.
   Ensure any checkout child left after writer-only death cannot recreate the tree after cleanup:
   record its identity before releasing the checkout, and quiesce only that proven owned child
   (and checkout hook descendants) before removal. Never kill the caller's group or any unrelated
   PID. Reuse the private detached group when safely identifiable; keep fallback signaling scoped
   to recorded identities. The dead-writer fixture must also verify no late recreation.
7. Prefer a runtime-probed Perl launcher, with checked setsid result and exec preserving PID:
   `perl -MPOSIX -e 'POSIX::setsid() >= 0 or die "setsid: $!"; exec @ARGV; die "exec: $!"' /bin/bash ...`.
   Keep nohup and redirected stdin/stdout/stderr, but launch this arm with monitor mode off so the
   child is not already a process-group leader. Probe the Perl POSIX module/call capability before
   selecting it; a command-v check alone is insufficient. Do not require an external setsid binary:
   the CI config proves no such dependency on macOS or Ubuntu.
8. If Perl/POSIX is unavailable, use bash's built-in `set -m` locally around the background launch
   to put the writer in a different process group, then restore monitor mode. Both CI shell families
   support bash job control; it needs no package installation. Preserve `$!` as the actual writer;
   confirm its PGID differs from the caller before returning PATH, using a bounded registration
   check. Avoid a background subshell whose PID differs from the recorded writer. If detachment
   cannot be established on an unusual host, emit a concise stderr diagnostic and retain the
   ordinary nohup launch with identity monitoring as the explicit safety net: future teardown
   becomes failed:writer_died rather than indefinite pending. A selected launcher failure must
   likewise leave a terminalizable owner record. No new public stdout or rc vocabulary.
9. Add failed:writer_died to the usage header (within the lines printed on usage error). Keep
   add's immediate PATH/0 return after registration; wait remains ready/0, failed:<why>/1,
   pending/75, with existing timeout, dirty-tree, path refusal and unknown-worktree behavior.
   Add the specified changelog fragment. Run focused validation and mutations on scratch copies;
   do not modify either Gate-3 resource or the CI workflow.

## Exact test arms to add

Use the following suite names. Capture stdout and rc separately; waitfor currently loses the
inner rc, so add a scratch-result helper for new arms rather than infer rc from text. Every loop
and harness wait must have a finite deadline and teardown must reap its own children.

All writes must be under one scratch tree, including `MISSION_WORKTREE_STATE`, HOME/config if
needed, hook/shim executables and Git metadata. Disable inherited Git hooks/config as needed in
scratch. Never touch real `~/.ailang/state`, a live worktree or a live process group. The `/tmp`
refusal control must remain a non-creating refusal. The current suite skips `/tmp` scratch; on
Linux, the controller must supply an approved non-/tmp TMPDIR and verify actual arms ran. A skip
is not a pass. Do not weaken production path refusal to make Linux tests convenient.

For real process-group arms, launch a harness in its own private group with POSIX::setpgid via
Perl on both macOS and Linux, or bash monitor mode when Perl is absent. Discover/record actual
PGID with ps, require it differ from the test runner's group, and have the harness remain alive
after add returns until an explicit release. Signal only this verified private group. Do not use
Linux-only `setsid`, `/proc`, GNU timeout or Bash 4 features. This harness is independent of the
production detachment choice; bound and reap it in the scratch fixture trap.

| Exact arm name | Fixture and assertion |
|---|---|
| `writer killed mid-checkout => failed:writer_died rc=1 and cleanup` | Slow post-checkout hook marks entered, then blocks on a scratch release file; require branch and partial tree exist and owner PID/birth match. Kill the recorded writer during this hold, safely stop/reap owned checkout processes, then wait. Require exact stdout failed:writer_died, rc 1, path absent, branch absent, registration pruned and state consumed. After releasing/reaping fixture children, require no path/branch recreation. Exercise the writer-only death path as well as cleanup after the group is gone. |
| `caller group TERM after add => ready rc=0` | Private caller group runs add and writes a scratch add-returned marker. Hook holds checkout until marker observed; immediately TERM only the caller group, then release hook from surviving test runner. Separate wait must reach ready/0 with clean tree and branch present. Assert writer PGID differed from killed caller PGID. No arbitrary one-second delay is the synchronization proof. |
| `live slow writer => pending rc=75 without cleanup` | Hook held, owner identity proven alive, wait MAX=1. Require exact pending/75, unchanged state and existing branch/path, matching writer still alive; then release hook and require ready/0. This kills cleanup-on-timeout or a false death report. |
| `completion between pending read and dead check => ready rc=0` | Scratch ps shim delegates normally except the selected wait-owner inspection: atomically publishes ready into fixture status and reports writer absent. Run the real wait path; require ready/0 and intact tree/branch. Add analogous failed:worktree_add_rc_1 terminal control, requiring preserved reason/1. This deterministically proves the terminal re-read, not timing luck. |
| `reused PID with different birth => writer_died without signaling replacement` | Fixture metadata names a live scratch sentinel PID with deliberately different saved birth time, a token matching that sentinel's actual argv, and fake-owned scratch branch/tree. Only birth differs, isolating the birth-check mutant. Require failed:writer_died/1 and cleanup, while sentinel survives. No actual PID-reuse race needed. |
| `same-second PID identity needs launch token` | Metadata matches sentinel PID/birth but has a different token from its full command. Require failed:writer_died/1, owned scratch cleanup, sentinel still alive. |
| `missing Perl => detached bash fallback reaches ready` | Scratch PATH shim makes Perl capability probe unavailable. Repeat caller-group TERM arm through bash job-control fallback; require ready/0 and different PGID. Record which launcher each detachment arm used. |
| `add publication => owner recorded before returned PATH` | Slow child; immediately at add return read complete metadata with PID/birth/token/repo/branch, confirm live identity and pending state, then release and require ready/0. A second fixture fails launcher startup and requires terminal failure rather than perpetual pending. |

Retain all existing assertions in the five controls: immediate add/clean ready, /tmp refusal,
timeout_3s plus path/branch removal, not_clean_after_add plus removal, unknown_worktree failure.
Also assert terminal consumption removes new owner records. No test may send external messages.

## Mutation table

Mutate scratch copies of the production helper, one change at a time; execute the same named
arms with MW pointed at that copy. Keep shims and mutations inside scratch and never mutate
the live helper or use git restoration. Record the pristine green control and each failing
arm/output/rc. A suite timeout or a skipped arm does not substitute for a discriminating assertion.

| Mutant | Arm that must go red |
|---|---|
| Drop pending writer-liveness check | `writer killed mid-checkout => failed:writer_died rc=1 and cleanup` |
| Drop terminal status re-read after detecting death | `completion between pending read and dead check => ready rc=0` |
| Drop detachment / restore ordinary nohup in both launch paths | `caller group TERM after add => ready rc=0` |
| Drop bash fallback detachment only | `missing Perl => detached bash fallback reaches ready` |
| Clean up on bounded pending even while identity is alive | `live slow writer => pending rc=75 without cleanup` |
| Use kill -0 alone / drop birth comparison | `reused PID with different birth => writer_died without signaling replacement` |
| Drop command token comparison | `same-second PID identity needs launch token` |
| Omit branch deletion or path removal on writer death | Dead-writer cleanup arm's separate branch/path assertions |
| Publish owner record after printing PATH | `add publication => owner recorded before returned PATH` (rendezvous forces the gap) |
| Leave checkout child running after writer-only death | Dead-writer cleanup arm's quiescence/no-recreation assertions |

## Controller validation and done-gate

Planner verification is limited to source premises, artifact consistency and JSON syntax. No
background process was started or awaited. **Socket-touching or process-killing checks inside
the sandbox are UNINFORMATIVE UNDER SANDBOX**, never pass/fail evidence. All real lifecycle,
detachment and mutation results below must be collected outside the sandbox by the executor/
controller. Do not claim Linux coverage from the macOS-only shell CI job.

Focused executor checks (no full `make test`): syntax-check both shell files using `/bin/bash -n`;
run the named suite outside the sandbox on macOS `/bin/bash` 3.2 and Linux bash 5 with non-/tmp
scratch; run shellcheck on the touched scripts if available, reporting inherited findings and
unavailability explicitly; run `make check-changelog` and `make check-file-sizes`. Independent
sprint-evaluator must review M1 against the named arms, identity/registration races, cleanup,
mutant failures and both launcher paths. Do not self-approve.

Controller only, after implementation and before landing/completion:

```bash
env -i HOME="$HOME" PATH="$PATH" TMPDIR="${TMPDIR:-/tmp}" make test-launchd-drivers
```

Use macOS `/bin/bash` 3.2 for that aggregate host gate, record rc 0 and actual suite counts;
ensure TMPDIR is non-/tmp for worktree arms. Also record Linux focused-suite rc/counts and bash
version. Require every new arm and all five old groups, with no process-check skip masquerading
as evidence. CI's existing macOS job is complementary evidence, not a substitute for missing
Linux or detachment experiments.

Apply `design_docs/fleet-mission.md:122` Guardrails: verify that the helper on the actual running
surface was edited, commit/push/reach of pinned origin/dev is confirmed, and nothing is reloaded
mid-iteration. Controller selects and records an **idle, armed sibling** after checking kill switch
and pidfile. Never dry-run fleet in its own active iteration. With the landed `worktree_fixed_sha`
and `worktree_idle_profile`, run these controller commands:

```bash
env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" \
  AILANG_DRIVER_PINNED="$worktree_fixed_sha" MISSION_PROFILE="$worktree_idle_profile" \
  MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh
env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" \
  AILANG_DRIVER_PINNED="$worktree_fixed_sha" MISSION_PROFILE="$worktree_idle_profile" \
  MISSION_DRY_RUN=1 MISSION_EXECUTOR_MODEL=codex:bogus MISSION_PROBE_TIMEOUT=10 \
  /bin/bash tools/launchd/mission-control.sh
```

The existing degradation seam is documented at `tools/launchd/mission-control.sh:2001`–`:2003`.
Require both rc 0, healthy lanes=ok and degraded lanes=DEGRADED(...), neither disabled/overlap
early yield nor pidfile write. If no sibling qualifies, use and record the synthetic-HOME fallback
at `design_docs/fleet-mission.md:132`–`:136` (symlink the listed config/credential surfaces, fresh
state, MISSION_PROFILE=fleet). These are controller work and were not attempted by the planner.
Keep the ticket open until independent evaluation, host gates, landing/reach and both dry-runs
are evidenced. No open design decision blocks this mechanical plan; runtime/host evidence is
the remaining dependency.
