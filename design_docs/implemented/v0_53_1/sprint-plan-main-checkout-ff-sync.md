# Sprint plan: M-MAIN-CHECKOUT-FF-SYNC

Date: 2026-10-08. Ticket: `skill-surface:main-checkout-not-synced-to-dev`.
Authority: D-FLEET-15 = A, Mark, attended 2026-10-08. This is a planning artifact,
not implementation or execution approval. The ticket/ruling is the design input.

## Goal and scope

Make merged skill changes reach the shared skill checkout on an eligible real fire,
using only `git -C <checkout> merge --ff-only origin/dev`. Require branch `dev`,
zero ahead commits, no operation in progress, and zero dirty paths intersecting
the incoming range. Otherwise report why and continue the fire. Never reset,
stash, checkout, rebase, retry a held lock, or delete another process's lock.

Two milestones, approximately two engineering days including mutation review:
M1 450 LOC (150 helper + 300 tests), M2 100 LOC (small driver seam + test additions,
make wiring and fragment), total 550. These are task estimates, not measured
LOC/day velocity; recent launchd history shows recurring routing/driver changes,
but supplies no defensible duration denominator. Safety review dominates typing.
Day 1: M1 fixtures, helper, negative controls. Day 2: M2 integration and done gates.
Risk: medium, because the target is an attended dirty checkout.

Only planned implementation files:

- `tools/launchd/lib/skill-sync.sh` (new).
- `tools/launchd/test_skill_sync.sh` (new, extended for M2 integration).
- `tools/launchd/mission-control.sh` (few-line call site).
- `make/test.mk` (one suite invocation, help wording if needed).
- `changelogs/unreleased/2026-10-08-main-checkout-ff-sync.md` (new `### Added` section).

Planning/state artifacts are this file and
`.ailang/state/sprints/sprint_M-MAIN-CHECKOUT-FF-SYNC.json`. No Go change, env-file
deployment, plist edit, launchctl reload, skill rewrite, or world-fork port is planned.

## Premises verified

Quotes below refer to the base worktree read for this plan; recheck locations at execution.

| Premise | File:line read and quoted evidence |
|---|---|
| Ruling is recorded | `design_docs/fleet-mission.md:148`: "ANSWERED — A (Mark Edmondson, attended 2026-10-08, recorded directly in this ledger)"; the row specifies `dev`, zero ahead, no operation, no intersecting dirty paths. Historical OPEN text in its last column is stale; the attended answer governs. |
| Main skill surface | `design_docs/fleet-mission.md:148`: "Loops read skills through `~/.claude/skills/mission-control` → `~/dev/sunholo-data/ailang`". The supplied current 0-ahead/3-behind/dirty measurement is operator evidence, not remeasured by this planner. |
| Pin source refresh | `tools/launchd/lib/pin-root.sh:187`: `_pin_bounded "$fetch_s" git -C "$src" fetch --quiet origin`; `:260`: `wt="${AILANG_DRIVER_PIN_DIR:-$HOME/.ailang-driver-pin/${MISSION_NAME:-$(basename "$script" .sh)}}"`; `:283`: `git -C "$src" worktree add --quiet --detach --force "$wt" "$target"`. Git worktrees share refs through their common git dir. The actual shared path given in the ticket was not probed here. |
| Driver helper root | `tools/launchd/mission-control.sh:1166-1168`: `if [ -f "$MC_DRIVER_ROOT/tools/launchd/lib/pin-root.sh" ]; then`, source that path, `pin_root_to_committed_ref "$@"`. `:1159-1164` says world is de-forked and helper code ships beside the shared driver, independently of `$REPO`. |
| Kill switch | `tools/launchd/mission-control.sh:1625-1627`: `if [ -f "$KILL_SWITCH" ]; then`, `log "kill switch present ($KILL_SWITCH) — skip"; exit 0`. |
| Overlap yield | `tools/launchd/mission-control.sh:1908-1914`: pidfile check; `:1911`: `log "previous iteration still running (pid $oldpid) — yield (next interval retries)"; exit 0`. |
| Dry-run seam | `tools/launchd/mission-control.sh:1924`: `if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then`; `:1930`: `log "DRY RUN ok: mission=$MISSION_NAME ...` ending `| pin=$PIN_STATUS($PIN_DRIFT behind)"; exit 0`. Append `| skill-sync=$SKILL_SYNC_STATUS` here. |
| Boot stagger | `tools/launchd/mission-control.sh:1933`: `# 3b. BOOT STAGGER`; `:1940`: `BOOT_WINDOW="${MISSION_BOOT_WINDOW:-900}"`; `:1947`: `sleep "$_off"`. Real apply belongs between dry-run exit and this block. |
| Driver size | `wc -l tools/launchd/mission-control.sh`: 2545. Keep logic out of this file. |
| Bounded idiom | `tools/launchd/lib/pin-root.sh:63`: "hard wall-clock cap; rc = CMD's rc, or 124 on expiry"; `:68-85`: `_pin_bounded` uses mktemp, background exec, date deadline, TERM/KILL, wait and cleanup. `:54`: "Portable to macOS bash 3.2.57". |
| Go Registry boundary | `internal/config/registry.go:67-74`: "Registry is every environment variable this package reads" and concat includes `missionVars`. `internal/config/registry_test.go:14-20` parses "this package's non-test sources"; `:104-107` applies the getenv-name gate "inside this package". `internal/config/mission.go:6-20,34-48` defines Go constants and missionVars. |
| Shell env convention | `tools/launchd/lib/pin-root.sh:35-37` documents `AILANG_DRIVER_PIN=0`, timeout and pin-dir in its contract; `tools/launchd/mission-control.sh:1198` reads `${AILANG_DRIVER_DRIFT_WARN:-25}`, with validation/logging at `:1202-1205`. `rg -n 'AILANG_DRIVER_PIN|AILANG_DRIVER_DRIFT_WARN' internal/config docs/docs/reference/env-vars.md` has no matches. These shell-only knobs are not Go Registry entries. Document new knobs in the helper's contract and fragment; no invented Go getters or generated docs changes. |
| Suite wiring | `make/test.mk:68`: `LAUNCHD_SUITE := /bin/bash tools/launchd/lib/suite-env.sh`; `:70`: `test-launchd-drivers:`; `:71-72` invokes suite-env and pin-root tests with `@$(LAUNCHD_SUITE)`. `:64-65` explicitly requires bash 3.2. |
| Suite isolation | `tools/launchd/lib/suite-env.sh:12` allows HOME/PATH/TMPDIR and login/locale values; `:18`: `exec env -i "${clean_env[@]}" /bin/bash "$@"`. The new suite must replace inherited HOME itself. |
| Fragment format | `changelogs/unreleased/README.md:3`: "Write new changelog entries here"; the following example specifies date-slug filename and `###` headings, checked by `make check-changelog`. |
| JSON precedent | Read recent-by-mtime `sprint_m_syntax_ai_forgiving.json` and `sprint_m_serveapi_ws_bridge.json`: sprint_id, created, design_doc, sprint_plan, velocity, features with id/description/estimated_loc/dependencies/acceptance_criteria/passes/started/completed/notes. Use pending values, not inherited completion data. |

Base read-only syntax checks: `/bin/bash -n tools/launchd/mission-control.sh` and
`/bin/bash -n tools/launchd/lib/pin-root.sh` returned 0. No driver, live inbox,
network, launchctl, pin test or full suite was run. This is an unattended mission
planner; ambient approvals and the unrelated active sprint are not this task.

## M1 — helper and synthetic safety tests

Files: new `tools/launchd/lib/skill-sync.sh`, new `tools/launchd/test_skill_sync.sh`.
Dependency: none. Estimate: 450 LOC, one day.

Export one function `mc_skill_sync <apply|report>`, sourceable under bash 3.2 and
`set -u`/`set -e`; initialize outputs on every call. Always return 0, never exit
the caller. Status vocabulary: `synced:<n>`, `current`, `skip:<reason-token>`,
`error:<token>`. Note is a one-line human explanation with no file contents.
In report mode an eligible range returns `synced:N` with note `would-sync N
commits (report mode; checkout unchanged)`: the status is the computed verdict,
not a claim that a merge ran. Log notes on real fires and assert the would-sync
note in tests. This avoids adding an unauthorized status family.

Document `AILANG_SKILL_SYNC=0` (default enabled) and
`AILANG_SKILL_SYNC_CHECKOUT` (unset resolves the skill symlink). Use a fixed
wall-clock cap per git call rather than adding another knob. Reuse the sourced
`_pin_bounded` primitive; missing primitive is `error:bounded-unavailable`,
not an unbounded fallback. Tests source pin-root before skill-sync; sourcing it
defines helpers but does not invoke its pinning function. Preserve its output
state around use so pin diagnostics do not change. Use bounded subprocesses
with output redirected to scratch files for NUL data: command substitution
cannot preserve NULs. Scratch cleanup never touches repository locks.

Algorithm, all git reads and the merge bounded:

1. Validate mode, honor opt-out first (`skip:disabled`), resolve override or
   require a symlink at `$HOME/.claude/skills/mission-control`, resolve with
   `readlink -f`, ask git for its worktree toplevel. Use canonical directory
   identity, not git-common-dir equality, to reject the current pinned driver
   worktree (`MC_DRIVER_ROOT`, with current worktree resolution for standalone
   tests). Skip absent/broken link, non-worktree or self target distinctly.
2. Set `GIT_OPTIONAL_LOCKS=0` for inspection, particularly porcelain status,
   so report mode does not refresh the target index. No fetch: pin-root already
   refreshes the shared refs. Missing/unreadable `origin/dev` is an explicit
   error. Resolve HEAD and origin/dev commits, require symbolic branch `dev`,
   count ahead using `origin/dev..HEAD`, behind using `HEAD..origin/dev`.
3. Resolve per-worktree operation paths using bounded `git rev-parse --git-path`
   (normalize relative paths against target). Detect rebase-merge, rebase-apply
   (including am), MERGE_HEAD, CHERRY_PICK_HEAD, REVERT_HEAD, sequencer, BISECT_LOG
   or BISECT_START. Check index.lock at the target's git-path, not at assumed
   `.git/index.lock`. Held lock means `skip:index-locked`, no retries/deletion.
   Perform these checks even for an already-current checkout; an eligible
   zero-behind range then returns `current`.
4. Get incoming paths with NUL-delimited name-status diff, explicitly including
   both rename sides (and copy sides where emitted), across captured HEAD..tip.
   Get `status --porcelain=v1 -z --untracked-files=all`, union both sides of
   staged/unstaged renames with modified/staged/deleted/untracked paths. Parse
   NUL records with bash 3.2 `read -r -d ''` and indexed arrays, never whitespace
   splitting or grep over quoted pathnames. Compare exact paths, deduplicate
   collisions, and include directory/file prefix obstruction cases. A dirty
   directory blocking an incoming file also needs a safe skip. Nonzero count
   means `skip:dirty-range`, note `colliding paths=N` without names or contents.
5. Report exits with the computed verdict and no target writes. Apply rechecks
   branch/HEAD/tip and safety state immediately before its sole bounded
   `git -C "$checkout" merge --ff-only origin/dev`. Changed snapshot means
   `skip:state-changed`, no retry. Preserve the literal authorized ref/command.
   Concurrent lock/ref/Git worktree refusal maps to a skip, distinguishing
   index.lock where possible; unexpected command failure or timeout is error.
   Do not infer success from exit 0 alone: verify HEAD and resulting count.
   A timeout may have advanced HEAD before expiry: report that uncertainty
   truthfully and never roll back. All paths return 0.

Git's internal locking/refusal remains the last defence against state changes
after the final preflight; there is no atomic read-check-merge transaction in
this scope. Tests include a late lock and moving-ref refusal. Do not claim the
preflight eliminates every possible concurrent attended edit.

Every arm creates its OWN mktemp lab: bare origin, clone on dev, second clone
that pushes new commits, local fetch into the target. Set HOME to a temp home
before source/calls; explicit local git identity, hooks disabled in fixtures,
isolate inherited git config/environment. Use only local remotes. All writable
paths, including TMPDIR, state and fake skills, are under the lab. Never inspect
or mutate the real main checkout, real HOME skills or real mission state.
Tests restore mutations only in copied helper/driver files under the lab.
Use before/after `cmp` for bytes, HEAD/index/refs snapshots and lock existence.
For report mode compare target worktree and git-dir inventories/bytes, not only
HEAD. Scratch outside the target is permitted; “writes nothing” means no target
repository/worktree/state mutation. No test is allowed to rely solely on Git
refusing to merge: assert the exact preflight status and a merge-call spy count.

M1 acceptance commands (to run during execution, not by this planner):

```bash
/bin/bash -n tools/launchd/lib/skill-sync.sh
/bin/bash -n tools/launchd/test_skill_sync.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_skill_sync.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_skill_sync.sh --mutations
```

Normal suite and mutation harness return 0; mutation harness requires each
isolated mutant to fail its named arm, then pristine control to pass. Harness
must reject unknown selectors, zero substitutions and skipped fixture setup.

## Mutation table

Each row names one single mutation and its discriminating arm. Mutants run only
the listed selected arm; shared code mutations can naturally affect other arms
in a full suite. “Exactly that arm red” is verified with isolated selection,
not claimed as global independence of shared helper code. Every arm asserts
status, caller survival and merge attempt count as well as its content witness.

| Arm | Expected witness | Single mutation that must turn this selected arm red |
|---|---|---|
| ff happy path (N=2) | synced:2; HEAD==origin/dev; one merge | Bypass the eligible apply merge. |
| current | current; no merge | Remove zero-behind current return. |
| not dev | skip:not-dev; HEAD unchanged; no merge | Remove branch guard. |
| ahead >0 | skip:ahead; no merge | Remove ahead-count guard. |
| rebase active | skip:operation-in-progress; marker preserved | Remove rebase-merge check. |
| merge active | same, MERGE_HEAD preserved | Remove MERGE_HEAD check. |
| am/rebase-apply | same, rebase-apply preserved | Remove rebase-apply check. |
| cherry-pick | same, CHERRY_PICK_HEAD preserved | Remove CHERRY_PICK_HEAD check. |
| revert | same, REVERT_HEAD preserved | Remove REVERT_HEAD check. |
| sequencer | same, sequencer preserved | Remove sequencer check. |
| bisect-log / bisect-start (separate arms) | same, respective marker preserved | Remove the respective bisect marker check. |
| dirty modified in range | skip:dirty-range, count=1, cmp unchanged, no merge | Exclude unstaged modified paths. |
| dirty staged in range | same, index bytes preserved | Exclude staged paths. |
| dirty outside range | synced:N, dirty bytes cmp identical | Replace intersection predicate with any-dirty rejection. |
| untracked incoming addition | skip:dirty-range, count=1, cmp identical | Exclude untracked paths. |
| dirty rename source / destination (separate arms) | collision on respective side only | Drop the respective porcelain rename side from path union. |
| incoming rename source / destination (separate arms) | collision on respective side only | Drop the respective diff rename side. |
| whitespace/newline filenames | exact collision count, bytes preserved | Replace NUL parsing with line/word splitting. |
| duplicate collision records | unique count=1 | Remove deduplication. |
| directory obstruction | skip:dirty-range; unchanged bytes | Remove path-prefix obstruction comparison. |
| index.lock held | skip:index-locked; lock still present; no merge | Remove lock preflight guard. |
| late index.lock | skip:index-locked; lock still present; merge refuses once | Map Git lock refusal to error instead of skip. |
| report eligible | synced:N + would-sync note; repository snapshot identical; no merge | Remove report-mode early return before merge. |
| opt out | skip:disabled; no git/merge calls | Remove opt-out guard. |
| symlink absent | skip:symlink-absent; no merge; override unset | Remove symlink-presence guard. |
| symlink resolves valid lab | synced:N on resolved lab; override unset | Replace resolved path with fixed HOME checkout guess. |
| non-git target | skip:not-worktree; no merge | Remove worktree validation. |
| self target | skip:self-target; no merge | Remove canonical self-target guard. |
| missing origin/dev | error:ref-unavailable; unchanged | Replace failed ref lookup with current/zero fallback. |
| bounded git timeout | error:git-timeout; finite runtime; caller alive | Replace that intercepted bounded call with direct git. |
| missing bounded primitive | error:bounded-unavailable; no git call | Fall through to unbounded execution. |
| ref/HEAD changes after preflight | skip:state-changed; no merge | Remove final snapshot comparison. |
| invalid mode | error:invalid-mode; no mutation | Remove mode validation. |
| caller survival on refused sync | rc=0, sentinel after call observed | Return a nonzero rc on that refusal. |
| M2 disabled driver | no sync invocation | Move source/call above kill-switch guard. |
| M2 overlap driver | no sync invocation | Move source/call above pidfile yield. |
| M2 dry-run seam | one report call, no apply, log has skill-sync | Change report argument to apply. |
| M2 dry-run field | exact appended skill-sync verdict | Remove appended field. |
| M2 real-fire seam | one apply before boot stagger; note logged | Move apply below boot stagger. |

For guard-removal mutants Git may independently refuse the unsafe operation;
the arm still turns red on missing explicit skip/no-merge evidence. Timeout
mutants run under an independent lab watchdog, so mutation testing cannot hang.
Use per-command shims targeting the relevant git invocation, not a blanket fake
git that erases the real synthetic repository behavior.

## M2 — driver integration, suite wiring and release note

Files: `tools/launchd/mission-control.sh`, `tools/launchd/test_skill_sync.sh`,
`make/test.mk`, `changelogs/unreleased/2026-10-08-main-checkout-ff-sync.md`.
Dependency: M1. Estimate: 100 LOC, one day including done gates.

Source the helper from `$MC_DRIVER_ROOT`, after pidfile yield, immediately before
the dry-run branch. In dry-run call report once and append the status to the
existing DRY RUN log. After the dry-run branch, call apply once and log status
plus note, then enter the existing stagger. A missing helper produces an
explicit error status/note and continues. No sync invocation on disabled or
yielding missions. This guarantees the new sync writes nothing in those paths;
it does not assert that earlier existing driver pin/state code writes nothing.
Keep the source/mode/call/log seam to roughly 8 added executable lines; put no
git/path/guard logic in the driver. Document modes and env knobs in helper and
fragment; follow the observed shell convention without touching internal/config.

Add `@$(LAUNCHD_SUITE) tools/launchd/test_skill_sync.sh` to
`test-launchd-drivers`; verify the suite appears in `make -n`. Extend the new
test with non-vacuous extraction of the production guard/call/dry-run/stagger
seam, using lab log/pidfile/kill-switch paths and spies for the boot/iteration
continuation. No controller/provider/socket probes. Tests assert production
ordering, exact call count/mode, missing-helper diagnostics and full dry-run
field; any extraction matching no block is a failure. These synthetic checks
exercise the changed code without running the full live driver in the sandbox.

M2 acceptance commands:

```bash
/bin/bash -n tools/launchd/mission-control.sh
/bin/bash -n tools/launchd/lib/skill-sync.sh
/bin/bash -n tools/launchd/test_skill_sync.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_skill_sync.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_skill_sync.sh --mutations
make -n test-launchd-drivers
make check-changelog
make test-launchd-drivers
git diff --check
jq -e . .ailang/state/sprints/sprint_M-MAIN-CHECKOUT-FF-SYNC.json
```

Inspect make dry output for the exact new wrapped invocation. Review changed
paths against the allowlist, confirm only planned files changed and no forbidden
main-checkout operation exists. On a trusted unsandboxed mission controller,
the done gate ALSO runs the real dry-run once with the working driver selected:

```bash
AILANG_DRIVER_PINNED=worktree-test MISSION_WORKDIR="$PWD" \
  MISSION_PROFILE=fleet MISSION_DRY_RUN=1 \
  /bin/bash tools/launchd/mission-control.sh
```

Capture the unpiped exit code and log, require DRY RUN ok with `skill-sync=...`,
and verify the target HEAD is unchanged by report. This live gate reads external
config/state and may probe providers; do not run it in this planner. Any verdict
depending on sockets or external paths under sandbox is **UNINFORMATIVE UNDER
SANDBOX**, never a pass/fail. Full-suite inherited base reds are findings, not
authorization to expand scope or silently omit the suite. No reload is required
for a code-only pinned-driver change; reach is committed origin/dev when landed
by the authorized controller, not a planner git operation.

## Registry reuse and findings

M1: `none` for AILANG package dependency; this is host bash/Git worktree safety,
not an AILANG-importable capability. Reuse the existing `_pin_bounded` instead
of a new timeout implementation. M2: `none`; existing driver and make harness
are the integration surface. Package search is not applicable to either
milestone; no package-like functionality is planned.

1. The shell env convention needs no out-of-scope Registry/generated reference
   file. If execution discovers a broader shell census gate requiring a file
   outside the fleet scope, report that as a finding instead of editing it.
2. The mission-loop-change skill contains historical world-fork and sourcing
   assumptions contradicted by the current driver at :1159-1164. Follow the
   current shared-driver root; no port or deployment files in this sprint.
3. The main checkout measurement and live symlink/common-dir paths are supplied
   operator premises. All implementation acceptance arms use synthetic repos;
   no planner claim of live measurement is made.
4. Report mode uses the required status vocabulary and an explicit would-sync
   note; target index-refresh writes must be disabled during inspection.
5. No already-red base runtime gate was measured. Only read-only syntax gates
   were run and passed; network/external-path gates remain unrun here.

Execution/evaluation handoff is left to the fleet controller. No messages,
git writes, implementation, live approval decisions or unrelated sprint changes
are authorized by this planner assignment.
