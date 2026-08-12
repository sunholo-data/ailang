# M-DRIVER-PIN-ROLLOUT: Roll the committed-ref driver pin out to the remaining launchd entry points

**Tracking**: ailang#558 ("launchd drivers execute from the stale main checkout — #556's qwen3.5 retirement never reached the rig")
**Status**: PARKED `needs-human-review` at the quorum gate (iteration 188) — 2 rounds, both BLOCKED, both reviewers present in both. Round 2's objections were measured first-party rather than forwarded and BOTH are confirmed and LARGER than filed (V23, V24 below). The remaining resolution requires a design choice the controller may not invent (Standing rule 2); see **Quorum round 2 — PARKED** at the end of this document. The milestones and acceptance criteria below are NOT approved to route.
**Target**: v0.34.0
**Priority**: P1 (the P0 instance — the V1 mission driver, the lane #556 actually ran through — shipped in PR #666; this is the systemic completion for the other launchd entry points)
**Estimated**: 3.5 days across five independently landable milestones
**Dependencies**: PR #666 (merged 2026-08-12: `tools/launchd/lib/pin-root.sh` + mission-control wiring), #667/#674 (onboarding-gate fixes). All merged; the unblocking gate on live evidence is MET (V1 below).

---

## Problem statement

`pin-root.sh` closes the #558 class — a launchd driver reads its own script, config and data
from a shared mutable clone that nothing keeps current — by re-execing the driver out of a git
worktree pinned to committed `origin/dev`. PR #666 wired it into `mission-control.sh` **only**.
Four other launchd entry points still run whatever the source clone's working tree holds:
`nightly-eval.sh` (the driver the #556 defect actually ran through — its `MODEL=` line kept
qwen3.5 for 2 days after retirement), `os-rotation-filler.sh`, `mission-recovery.sh`, and
`rig-watchdog.sh`.

This doc designs that rollout. It is **not** a re-design of the helper: the helper's contract
(source-never-execute, loud-non-fatal STALE, caller-owned emission — `pin-root.sh:20-26`) is
frozen except for four small opt-in extensions (M0) that the new call sites need and
mission-control does not use.

**The pickability gate is met** (charter: ">=3 consecutive V1 fires logging the pin note with a
normal iteration completing"): 5 pinned fires, 0 failures, with known-positive controls — see V1.

---

## Verification Log

All repo commands run first-party at `origin/dev` = `996bcccd7` (source clone
`~/dev/sunholo-data/ailang`; this session runs from the pin worktree
`~/.ailang-driver-pin/v1` at the same SHA). Rig state measured 2026-08-13. Every negative/empty
result carries a known-positive control **in the same command against the same scope**.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | Unblocking gate met: ≥3 pinned V1 fires, 0 pin failures | `grep -c "driver pin: running committed origin/dev @" /tmp/ailang-mission-control.log` ; `grep -c "DRIVER PIN FAILED" /tmp/ailang-mission-control.log` ; controls: `grep -c "DRIVER PIN FAILED" tools/launchd/mission-control.sh` ; `grep -c "mission iteration starting" /tmp/ailang-mission-control.log` | `5` pinned; `0` FAILED; control `1` (string exists in source) and `126` (log readable). Fires 185/186/187 pinned and completed normally. |
| V2 | `nightly-lang-eval.sh` is NOT a launchd entry point: script exists but nothing schedules it | `ls ~/Library/LaunchAgents/ \| grep -i -E "ailang\|ollama"` ; `ls tools/launchd/*.plist` ; `crontab -l` ; `ls /tmp/ailang-nightly-lang-eval.log` | 13 installed plists, **none** for lang-eval, while `dev.ailang.nightly-eval.plist` IS present in both locations (known-positive, same listing). crontab has one unrelated aitana entry. Its log file `/tmp/ailang-nightly-lang-eval.log` does not exist (control in same block: `/tmp/ailang-mission-control.log` exists with 126 iteration lines) → not run recently either. The script itself exists at `tools/launchd/nightly-lang-eval.sh`; header says "schedule WEEKLY" + "dashboard wiring is a manual step". **Disposition: manual-only orphan — scoped OUT, see Scope.** |
| V3 | The four real entry points split into two classes | `for f in …; do grep -c '^REPO=' tools/launchd/$f.sh; done` | repo-rooted: `nightly-eval`=1 (line 28), `os-rotation-filler`=1 (line 12); script-only: `rig-watchdog`=0, `mission-recovery`=0 (known-positive in same loop: the two repo-rooted hits). Recovery/watchdog read only `$HOME/.ailang/state`, `/tmp` logs, `launchctl`, `pgrep`, health endpoints (read in full). |
| V4 | None of the four spawns `claude` | `for f in …; do grep -c 'claude' tools/launchd/$f.sh; done` ; control: same grep on `mission-control.sh` | nightly-eval `0`, nightly-lang-eval `0`, os-rotation-filler `0`, rig-watchdog `0`, mission-recovery `1` — and that hit is the comment at `mission-recovery.sh:50` ("it guards the claude run"), not an invocation. Control: `mission-control.sh` = `24`. **The charter's "one human onboarding action per pinned entry point that runs claude" therefore applies to ZERO of them** — the charter asked for this to be measured rather than assumed, and this is the measurement. |
| V5 | Onboarding gate mechanics + escape hatch | Read `pin-root.sh:192-201` | Gate accepts if `$wt` OR `$src` has `hasCompletedProjectOnboarding`; `AILANG_DRIVER_SKIP_ONBOARD_CHECK` bypasses. Source clone is onboarded (V1's five pinned fires prove `claude -p` runs from a worktree of it), so the gate would pass today. Decision D4 makes the bypass explicit anyway. |
| V6 | Pin-dir collision hazard is CONCRETE for mission-recovery | `plutil -extract EnvironmentVariables json -o - ~/Library/LaunchAgents/dev.ailang.mission-recovery.plist` ; `ls ~/.ailang-driver-pin/` ; read `pin-root.sh:130` | Recovery's plist injects `MISSION_NAME=v1`. Default pin dir is `$HOME/.ailang-driver-pin/${MISSION_NAME:-$(basename $script .sh)}` → recovery would pin to `~/.ailang-driver-pin/v1` — the SAME dir mission-control's multi-hour iterations run from (`ls` shows exactly `motoko` and `v1` exist). A recovery fire every 240s would `checkout --force` under a live mission session. **D3 eliminates this by construction.** The other three plists set no `MISSION_NAME`; no launchd domain global (V10); no shell-profile export (V11). |
| V7 | `~/.ailang-nightly/worktree` is owned by nightly-eval.sh itself — no collision | `git worktree list` (from source clone) ; read `nightly-eval.sh:63` | Listed as a linked worktree of the source clone, detached at `eabab0611` (= origin/dev at the 2026-08-12 03:00 fire). It is the M-EVAL-NIGHTLY-REPRO **build/data** worktree (`WT="$HOME/.ailang-nightly/worktree"`), refreshed to `origin/dev` by the driver each fire. Different path family from `~/.ailang-driver-pin/*`; the rollout neither collides with nor reuses it (D2 explains why they stay separate). |
| V8 | Installed plists are regular files pointing into the source clone; repo↔installed divergence is LIVE | `ls -l ~/Library/LaunchAgents/dev.ailang.*.plist` ; `plutil -extract ProgramArguments …` ; `diff tools/launchd/dev.ailang.nightly-eval.plist ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist` | All four are `-rw-r--r--` regular files (not symlinks), `ProgramArguments` absolute into `~/dev/sunholo-data/ailang/tools/launchd/`. The diff is NON-EMPTY: the repo copy carries #618's `AILANG_OLLAMA_V1_STREAM=1`, the installed copy still the old `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800` stopgap. **Consequence (V-H confirmed): repo plist edits are inert until a human re-installs — so this rollout makes ZERO plist changes (D7). Script edits, by contrast, reach the rig at the next fire** because launchd invokes the script by path into the clone's working tree, which updates when commits land on local `dev`. |
| V9 | Fire cadences | `plutil -extract StartInterval raw …` per plist | nightly-eval: `StartCalendarInterval` 03:00 daily; os-rotation-filler: 2700s (45 min); mission-recovery: 240s; rig-watchdog: 60s. These cadences drive D5 (fetch budget) and D6 (report throttling). |
| V10 | No `MISSION_NAME` launchd domain global; instrument proven on a positive | `launchctl getenv MISSION_NAME` ; control `launchctl getenv AILANG_NOT_A_REAL_VAR` ; positive `launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC` | Both first two print empty (rc=0 either way — rc carries no signal); the positive prints `1800`, proving the instrument reads set globals. Empty is therefore a measurement. |
| V11 | No shell-profile `MISSION_NAME` export (filler + watchdog run via `bash -lc`) | `grep -l "MISSION_NAME" ~/.bash_profile ~/.bashrc ~/.profile ~/.zshenv ~/.zshrc` ; control `grep -l "PATH" ~/.zshrc` | Empty (rc=2, some files absent); control hits `~/.zshrc`. D3 removes any dependence on this staying true. |
| V12 | rig-lock is an mkdir mutex with an EXIT trap + 6h stale-steal | Read `rig-lock.sh:16-41` | `mkdir` atomic lock at `~/.ailang/state/rig.lock.d`, holder file with pid, `trap "rm -rf …" EXIT`, steal after `RIG_LOCK_STALE_MIN=360`. Load-bearing for D1: bash does NOT run EXIT traps on `exec`, and `exec` preserves the PID. |
| V13 | Test lane exists and is CI-wired; it syntax-checks ALL drivers | `sed -n '38,44p' make/test.mk` ; `grep -n "test-launchd-drivers" .github/workflows/ci.yml` | `test-launchd-drivers` runs `test_pin_root.sh` (11 sections, 1–9) + `test_driver_notify.sh` + `bash -n` over `tools/launchd/*.sh tools/launchd/lib/*.sh`, explicitly under `/bin/bash` (3.2.57 — `bash --version` confirms `3.2.57(1)-release`). CI: `ci.yml:472`. |
| V14 | **Baseline of the gate — and a real bug found while baselining**: the suite is GREEN in a clean env but RED (9 passed / 26 failed) when run from inside a pinned session | `make test-launchd-drivers` (this session) vs `env -u AILANG_DRIVER_PINNED -u AILANG_DRIVER_SRC -u AILANG_DRIVER_DRIFT -u AILANG_DRIVER_REF -u MISSION_WORKDIR -u MISSION_NAME make test-launchd-drivers` | Pinned-env run: `==== 9 passed, 26 failed ====`, every failure showing the fixture "pinned" to THIS session's inherited `AILANG_DRIVER_PINNED=996bcccd7`. Clean-env run: all green, `launchd drivers: tests + bash 3.2 syntax OK`. The fixtures don't sanitize `AILANG_DRIVER_*`/`MISSION_WORKDIR`, so the helper's already-pinned fast path short-circuits every scenario. CI is green only because CI's env is clean. **Fixed in M0; is also AC1's red→green.** |
| V15 | nightly-eval durable-write sites that would be DESTROYED if left `$REPO`-relative after pinning | Read `nightly-eval.sh:315,363-376,504,526-529` ; `git check-ignore -v eval_results/rotation/os-rolling` | `HIST`/`FMT_HIST` append to `docs/static/benchmarks/*.jsonl` (tracked files) and `git -C "$REPO" add/commit/push`. In a pin worktree those commits land on a detached HEAD and the tracked-file appends are wiped by the next `checkout --force` — silent data loss. (`eval_results/` is gitignored — `.gitignore:86` — so the filler's accumulator has the sibling problem: a fresh pin worktree starts EMPTY.) Drives D2. |
| V16 | filler's release-pickup trigger can NEVER fire from a pin worktree | Read `os-rotation-filler.sh:192-211` | Trigger is `std/VERSION` (checkout) ≠ `origin/dev:std/VERSION`. A pin worktree IS `origin/dev`, so the two are always equal → `make quick-install` + `os-release-snapshot --reset` + per-release accumulator reset would silently stop happening. Drives the M2 rework (state-file trigger). |
| V17 | rig-watchdog's wedge-killer still matches a pinned filler parent | Read `rig-watchdog.sh:87-90` | Match is `case "$pcmd" in *os-rotation-filler*)` on the parent command line. The pinned parent is `/bin/bash $HOME/.ailang-driver-pin/os-rotation-filler/tools/launchd/os-rotation-filler.sh` — substring present twice. No breakage. |
| V18 | Full installed-launchd inventory audited — nothing else is in the #558 class | `plutil -extract ProgramArguments …` over the remaining 6 plists | `com.sunholo.ailang.daemon` + `dev.ailang.coordinator`: run the **installed Go binary** (`~/go/bin/ailang daemon/coordinator`) — binary-install staleness is a different class (deliberate versioned installs), out of scope. `dev.ailang.mission-resume`: self-deleting inline one-shot (quota-pause lift), no repo code. `dev.ailang.mission-motoko` / `mission-world`: `mission-control.sh` in their OWN checkouts — already covered by #666 (motoko clone syncs from the same origin) or explicitly not ours (world fork — Non-goals). `dev.ollama.serve`: vendor binary. **The four in this doc are the complete remaining #558 surface.** |
| V19 | Duplicate gate: no existing doc covers this | `ls design_docs/planned/ \| grep -ci "pin\|driver\|launchd"` ; `grep -rl "pin-root\|driver-pin" design_docs/` | `0` in planned/ (control: the same listing prints 21 real entries, scope non-empty and readable). No implemented doc claims the rollout. |
| V20 | mission-control's emit pattern + the caller-emits contract this rollout must honor | Read `mission-control.sh:301-336`, `pin-root.sh:20-26` | Pin placed before probes/pidfile, after state block; STALE → `log "DRIVER PIN FAILED …"` + degradation block on both human channels via `_mc_notify`. Helper contract: "emitting is the caller's job, because only the caller knows its own early-exit points — a fire that never runs must never post." |
| V21 | rig-watchdog's `exit 0` is the script's TERMINAL statement — and its ONLY exit | `wc -l tools/launchd/rig-watchdog.sh` ; `sed -n '110,114p' tools/launchd/rig-watchdog.sh` ; `grep -n 'exit' tools/launchd/rig-watchdog.sh` | `114` lines total; lines 112–114 are the script's final statement (`# Exit 0 always — the next tick will re-check…` + `exit 0`); the grep finds NO other exit statement — its only hits are the line-112 comment and line 114 itself (the grep's own known-positive). So "continue and `exit 0`" in round 1's D6 cell described the terminal exit, but sat in the STALE-behavior column where it read as "exit immediately on pin failure" — the fixed cell (D6) cannot be read that way. |
| V22 | The pin fetch is ALREADY bounded in the helper — objection 2's premise is refuted; what's wrong for these cadences is the DEFAULT | `grep -n '_pin_bounded\|FETCH_TIMEOUT' tools/launchd/lib/pin-root.sh` ; `sed -n '/^_pin_bounded()/,/^}/p' tools/launchd/lib/pin-root.sh` | `pin-root.sh:103` `fetch_s="${AILANG_DRIVER_FETCH_TIMEOUT:-120}"`; `:114` `_pin_bounded "$fetch_s" git -C "$src" fetch --quiet origin`; `:116` timeout ⇒ `_pin_stale "git fetch origin exceeded ${fetch_s}s"`. `_pin_bounded` (lines 59–77) is a real bounded wait, not a comment: backgrounds the command, computes a `date +%s` deadline, polls at 1s, then `kill` + `sleep 1` + `kill -9`, returns 124. BUT the 120s default is **2× rig-watchdog's 60s interval** and half of mission-recovery's 240s (V9) — a hung fetch could occupy the watchdog across one-to-two respawn ticks. Drives D5's per-driver explicit bounds + AC10. |

| V23 | **D2's data-continuity mechanism cannot work as designed** — the two durable-write helpers resolve their root from `$0`, NOT from the caller's cwd, so invoking them with `cwd=$REPO_DURABLE` does nothing | `ls -la tools/os-release-snapshot.sh tools/publish-unified-dashboard.sh` (both present — scope asserted before reading it, rule 3a(i-d)) ; `grep -nE 'REPO\|\$HOME\|~/\|^[A-Za-z_]+=/\|"/Users\|dirname "\$0"\|cd ' tools/os-release-snapshot.sh tools/publish-unified-dashboard.sh` ; control `grep -cE 'REPO' tools/launchd/nightly-eval.sh` | `publish-unified-dashboard.sh:26-27` = `REPO="$(cd "$(dirname "$0")/.." && pwd)"` then `cd "$REPO" \|\| exit 1`, and `:30` `AILANG="$REPO/bin/ailang"`; `os-release-snapshot.sh:24` = `cd "$(dirname "$0")/.."`. Control fires (**19**). So a pinned filler invoking `$WT/tools/publish-unified-dashboard.sh` has that script immediately `cd` to `$WT` regardless of the caller's cwd, and its writes land in the throwaway pin worktree — **exactly the silent data loss the objection predicted, by a mechanism (`$0`) the objection did not name**. `gemini-3-1-pro`'s `proposed_fix` presumes the audit comes out clean ("confirming they strictly use relative paths and respect the caller's cwd"); it does not, so applying that fix verbatim would write a FALSE row into this document. |
| V24 | **An automatic source-clone updater DOES exist — and it is the very driver this rollout would pin, so M2 destroys the delivery path for every other milestone** | `grep -rnE "merge --ff-only\|git pull\|checkout -B dev\|checkout dev" tools/ scripts/` ; `git -C <source clone> rev-list --count HEAD..origin/dev` ; read `pin-root.sh:110-130` | `os-rotation-filler.sh` runs `git pull --rebase --autostash origin dev` at lines **197, 398, 426, 458** — it fires every 45 min (V9), which is why the source clone measures **0 behind** right now. (`nightly-lang-eval.sh:106` pulls too but is the orphan of V2, so it never runs.) `pin-root.sh` only ever **fetches** the source clone (`:114`); it never advances its working tree. So `gpt5-6-sol`'s literal premise — "V8 verifies … not any mechanism that advances the rig working tree" — is **REFUTED**: the mechanism exists and is named here. Its CONCLUSION nevertheless stands, and for a worse reason than it gave: **pinning `os-rotation-filler.sh` (M2) re-execs it out of the pin worktree, so its `git pull` would target the throwaway checkout instead of the source clone — removing the only automatic updater, after which no later milestone's script edits could ever reach the rig.** D7's "ships entirely through git / no human rig action" claim is therefore self-falsifying from M2 onward. |

**INHERITED, not re-verified here** (implementer re-checks before relying): launchd skips a
`StartInterval` fire while the same label is still running (platform-documented behavior; the
design does not depend on it — D8's same-SHA fast path makes overlapping same-driver checkouts a
no-op in the common case, and manual concurrent invocation remains possible regardless).

---

## Scope

**IN**: `nightly-eval.sh`, `os-rotation-filler.sh` (repo-rooted class); `mission-recovery.sh`,
`rig-watchdog.sh` (script-only class). Plus the M0 helper extensions and the V14 test-env fix.

**OUT — `nightly-lang-eval.sh`, with the measurement (V2)**: it has no plist (installed or in
repo), no crontab entry, and no log file on this rig — it is a manual-only orphan, not a launchd
entry point; the charter's list of five is wrong by one. It is NOT pinned by this rollout:
sourcing pin-root into a script a human runs by hand would re-exec their possibly-deliberately-
edited copy into committed code — surprising exactly the person testing a change (the
`AILANG_DRIVER_PIN=0` escape exists, but a default that surprises the only invoker buys nothing;
there is no unattended fire to protect). Separately, its 4-language sweep is now substantially
covered by the filler's cross-language pass (`os-rotation-filler.sh:323-373` names the same
rule) — its deletion/reschedule is flagged to Mark as a one-line follow-up issue, not smuggled
into this rollout.

### The two classes change what pinning buys (V3)

| Class | Drivers | What goes stale today | What the pin moves |
|---|---|---|---|
| repo-rooted | nightly-eval, os-rotation-filler | driver script AND benchmark specs, helper tools, plist-adjacent config read via `$REPO`/cwd | script + specs + tools together, at one named commit |
| script-only | mission-recovery, rig-watchdog | ONLY the script text (zero repo artifact reads — V3) | just the script — but that is precisely the #556 shape: both are reliability backstops whose fixes (wedge-killer, probe-cooldown) must actually reach the rig |

The classes get different acceptance criteria and different failure-path budgets (below); they
are not one list of four.

---

## Design decisions

**D1 — Pin BEFORE the rig lock, always; kill-switch guards may precede it.**
Bash does not run EXIT traps on `exec`, and `exec` preserves the PID (V12). If a driver pinned
*after* `rig_lock_acquire`, the lock's cleanup trap would be lost across the re-exec while the
holder file names a still-live PID; the re-exec'd copy then re-runs `rig_lock_acquire` against
its own lock — `nowait` mode (filler) yields forever ("rig busy" from its own ghost, until the
6h stale-steal), `wait` mode (nightly) spins 30s-sleeping against itself. So: in nightly-eval
the pin goes after `log()` is defined (line 35) and before `rig_lock_acquire` (line 43); in the
filler, before the `rig_lock_acquire nowait` at line 176 (the cheap blackout/ollama guards at
lines 166-173 may stay ahead of it — they are early-exits that make "a fire that never runs
never posts" true). In mission-recovery, the two kill-switch checks (lines 60-64) stay ahead of
the pin: a deliberately disabled loop must not fetch or post. **nightly-eval runs
`set -euo pipefail` (line 26): the call must be wrapped (`if pin_root_to_committed_ref "$@";
then :; fi`-style) so the STALE return of 1 cannot kill the fire** — the same wrapped form is
used in all four for uniformity.

**D2 — Code from the pin; durable data in the source clone; `$HOME`//tmp` state untouched.**
`pin-root.sh` already exports `AILANG_DRIVER_SRC` across the exec. Each repo-rooted driver adds
one line after the pin: `REPO_DURABLE="${AILANG_DRIVER_SRC:-$REPO}"`, and every path that must
SURVIVE fires moves to it. Rationale (V15): a pin worktree is a throwaway detached checkout
refreshed with `checkout --force` — a tracked-file append or a commit made there is destroyed on
the next fire; a gitignored accumulator there starts empty, forking weeks of banked history.
Concretely:
- nightly-eval: `HIST`/`FMT_HIST` and their `git -C` add/commit/push blocks → `$REPO_DURABLE`
  (behavior identical to today: those commits land on the source clone's `dev`, as they do now).
  `HISTORY` is already `$HOME`; `RESULTS_DIR` is `/tmp`; the fetch/rev-parse/worktree-add block
  (lines 74-88) stays `$REPO` — a linked worktree shares the object store, so those are
  equivalent, and `$WT` (V7) keeps doing exactly what it does today. Yes, that means the script
  pin and the build worktree coexist and the fire fetches twice; the second fetch is ~1s and
  buying defense-in-depth (a STALE script pin still yields a pinned *build*).
- os-rotation-filler: `ROLL` → `$REPO_DURABLE/eval_results/rotation/os-rolling` (**data
  continuity by construction — the accumulator never moves, no migration step**); `FULL_CURSOR`/
  lap markers live under `ROLL` and follow it; `CURSOR` is already `$HOME`; the three dashboard
  JSONs and the bucket-sync sources → `$REPO_DURABLE/docs/static/benchmarks/…` (if they were
  regenerated in the pin worktree, the source clone's copies would go stale and the post-release
  W4 snapshot would sweep STALE fallbacks into the release); the legacy `BENCH_GIT_COMMIT=1`
  blocks get `git -C "$REPO_DURABLE"` (default-off, but a detached-HEAD `git pull --rebase`
  would hard-fail there); `tools/os-release-snapshot.sh` and `tools/publish-unified-dashboard.sh`
  are invoked as pinned code by absolute path with cwd `$REPO_DURABLE` — M2 carries an explicit
  path audit of both before the switch (their internal cwd-relative reads/writes are data-plane).
  Benchmark enumeration (`benchmarks/*.yml`, lines 234/330) deliberately stays cwd = pin: specs
  are code.

**D3 — Every new call site sets `AILANG_DRIVER_PIN_DIR` explicitly; the helper's default is not
touched.** V6 shows the default (`${MISSION_NAME:-basename}`) would collide mission-recovery
into the live `v1` mission worktree. Changing the helper's default would move mission-control's
existing pin dir (frozen surface, three live missions). Instead each driver sets
`AILANG_DRIVER_PIN_DIR="$HOME/.ailang-driver-pin/<script-basename>"` immediately before sourcing
— collision impossible regardless of what launchd, profiles, or future plists export, and the
four dirs (`nightly-eval`, `os-rotation-filler`, `mission-recovery`, `rig-watchdog`) sit beside
the existing `v1`/`motoko` with self-describing names.

**D4 — Onboarding gate: bypass via the EXISTING flag, in the driver, next to the measurement.**
Options considered: (a) leave the gate active — it would pass today (V5) but couples four
drivers that never run Claude Code (V4) to `~/.claude.json` state; if that file were ever
reset/moved, all four would go STALE for a reason that CANNOT apply to them — a category-error
failure mode. (b) teach the helper a "caller declares it spawns claude" mode — new helper
surface, new tests, to express what an existing flag already expresses. (c) **chosen**: each of
the four sets `AILANG_DRIVER_SKIP_ONBOARD_CHECK=1` with a one-line comment citing V4's
measurement ("this driver spawns no `claude` — grep count 0; re-gate if that changes"). The
declaration sits in the driver, adjacent to any future `claude` call a reviewer would add, and
the helper stays frozen. mission-control (24 hits) keeps its gate.

**D5 — Fetch budget for high-cadence drivers: frequency (`AILANG_DRIVER_FETCH_MAX_AGE`, M0,
opt-in) AND duration (`AILANG_DRIVER_FETCH_TIMEOUT`, set explicitly per driver).**

*Frequency.* A watchdog firing every 60s (V9) must not `git fetch` 1,440×/day — that adds a
network dependency and GitHub load to a job whose purpose is LOCAL reliability. New helper env:
when set to N seconds and the source clone's `.git/FETCH_HEAD` mtime is younger than N
(`stat -f %m`, bash-3.2-portable), skip the fetch and resolve the ref as-is. Recovery and
watchdog set 3600; worst-case staleness bound = 1h + whatever the last sibling fetch left, vs
today's UNBOUNDED — and in practice mission-control (2h cadence) and the nightly keep
`FETCH_HEAD` fresh for them. The repo-rooted drivers don't set it (nightly fires 1×/day; filler
32×/day is acceptable fetch load and its release-pickup already fetched anyway). Unset ⇒
behavior byte-identical to today — mission-control unaffected.

*Duration.* The fetch that DOES happen is **already bounded by the helper today** — this doc
must not imply otherwise: `_pin_bounded` runs it under `AILANG_DRIVER_FETCH_TIMEOUT` (default
120s) with kill → kill -9 escalation and a typed STALE on expiry (`pin-root.sh:103,114,116`;
implementation verified at V22). Nothing in this rollout introduces an unbounded wait. What IS
wrong is the **default value at these cadences**: 120s is 2× rig-watchdog's fire interval and
half of mission-recovery's, so a single hung fetch could occupy the watchdog for one-to-two
respawn ticks — and `AILANG_DRIVER_FETCH_MAX_AGE` only reduces how OFTEN a fetch happens; it
does nothing to bound the one that does. So every call site sets the timeout explicitly, at the
same place D3 sets `AILANG_DRIVER_PIN_DIR`, strictly below its own cadence with headroom for
the driver's real work:

| Driver | Cadence (V9) | `AILANG_DRIVER_FETCH_TIMEOUT` | Why this number |
|---|---|---|---|
| rig-watchdog | 60s | **20** | worst-case pin delay ≤20s leaves ≥2/3 of the tick for the respawn checks; a hung fetch can no longer swallow even one full tick |
| mission-recovery | 240s | **60** | 1/4 of the interval; the recovery probes keep ≥180s |
| os-rotation-filler | 2700s | **300** | ~1/9 of the interval; generous for a cold fetch, negligible beside the multi-minute eval cycle behind the lock |
| nightly-eval | daily 03:00 | **600** | one fire/day can afford a patient fetch; still ~1/144 of the cadence |

The 120s default is relied on NOWHERE in this rollout. The invariant is *bound strictly less
than the driver's own fire interval* — a per-driver value exceeding its interval is the defect,
and AC10 asserts the comparison numerically (a presence-grep would pass at any value).

**D6 — Failure reporting is per-driver, throttled where cadence demands (the failure path IS
the design).** The helper's contract makes emission the caller's job (V20). Per driver:

| Driver | Cadence | STALE behavior | Log line | Human channel |
|---|---|---|---|---|
| nightly-eval | 1/day | continue on working tree (never abort — data provenance is STILL pinned via `$WT`, V7) | every fire | `ailang messages send controlplane` every fire (≤1/day, no throttle needed) |
| os-rotation-filler | 45 min | continue unpinned | every fire | controlplane, throttled ≥6h |
| mission-recovery | 240s | **continue** — refusing to kickstart a wedged mission because `git fetch` failed would couple loop recovery to network health, the exact inversion of its purpose | every fire (its own `/tmp` log) | controlplane, throttled ≥6h |
| rig-watchdog | 60s | **continue INTO the ollama/server/wedge checks** — on STALE the pin block falls through and every downstream check still runs in the same fire; the script's ONLY exit statement is its pre-existing TERMINAL `exit 0` at `rig-watchdog.sh:114` (V21), which this rollout neither moves nor duplicates, and no new exit path is added anywhere before it | throttled ≥1h (else 1,440 lines/day in `/tmp/ailang-rig-watchdog.log`, V9/V18) | controlplane, throttled ≥12h |

Throttling ships as `tools/launchd/lib/pin-report.sh` (M0): `pin_report_stale <name>
<log-throttle-s> <msg-throttle-s>` — logs/posts only when the corresponding
`~/.ailang/state/driver-pin-stale-<name>-{log,msg}` stamp is older than the throttle; always
safe under `set -u`, never returns non-zero. Kill-switch early exits precede the pin (D1), so a
disabled job never posts — preserving the "a fire that never runs must never post" contract.

**The honest residual (replaces round 1's absolute "never blocks" claim):** the pin adds no
UNBOUNDED wait anywhere — the only network step is the `_pin_bounded` fetch (V22) — but it DOES
add a bounded delay before the reliability checks on the fires that actually fetch: worst case
≈ the driver's explicit `AILANG_DRIVER_FETCH_TIMEOUT` (D5) plus ~1–2s of local git work. For
rig-watchdog that is ≤~22s of a 60s tick, and on the happy path at most once per
`AILANG_DRIVER_FETCH_MAX_AGE` window (3600s); for mission-recovery, ≤~62s of a 240s tick. The
respawn backstop can be *delayed within a tick*, never *disabled* — that quantified statement
is the only form in which "pin failure doesn't block the checks" is claimed.

**D7 — Zero plist changes, deliberately.** V8 shows plist edits are inert until a human
re-installs, AND that an unrelated plist rollout (#618's stream flag-on) is currently pending
with its own hard ordering constraint (install flag-on plists BEFORE `launchctl unsetenv`, per
`.claude/rules/dev-workflow.md`). Everything in this rollout — pin dirs, fetch ages, skip flags
— is set inside the scripts, which launchd reads from the clone's working tree at each fire. So
the rollout ships entirely through git, requires **no human rig action** (V4 killed the
onboarding action; D7 kills the install action), and cannot entangle with or reorder the #618
plist procedure.

**D8 — Two hardening extensions in the helper (M0, both unconditional but behavior-preserving).**
(1) *Same-SHA fast path*: if the pin worktree's HEAD already equals the target, skip the
`checkout --force` — at 60s cadence this makes the steady-state pin a rev-parse + exec, and it
eliminates tracked-file churn under any concurrently running sibling instance except in the
moment a new commit actually lands. (2) *`bash -n` pre-exec guard*: refuse (STALE) to exec into
a target driver that fails a syntax check — closes the "pin faithfully delivers a broken script
to a 60s-cadence job" hole for the one failure class detectable in 10ms; semantic breakage
remains covered by CI on `origin/dev` (V13), which is strictly stronger review than today's
unreviewed working tree.

**D9 — Fix the V14 test-env leak.** `test_pin_root.sh` gets an env-sanitization preamble
(`unset AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF
MISSION_WORKDIR MISSION_NAME AILANG_DRIVER_PIN_DIR AILANG_DRIVER_FETCH_MAX_AGE
AILANG_DRIVER_SKIP_ONBOARD_CHECK`) so the suite measures the fixtures, not the invoking
session. Found because this doc's baseline run was executed from inside a pinned mission
iteration — exactly where future mission executors will run it.

---

## Conflict surface

**Shared helper, three live missions.** `pin-root.sh` is sourced by V1, motoko and world
mission drivers every ~2h (V18). M0's extensions are opt-in env (D5) or fast-path/guard changes
that preserve outcomes (D8); tests 1–9 must pass unchanged, and M0 lands ALONE and observes one
clean V1 fire (`driver pin: running committed` in the V1 log) before M1 proceeds.

**Shared GPU mutex.** nightly-eval (wait) and os-rotation-filler (nowait) contend on
`rig.lock.d` (V12). D1's ordering means no pin ever executes with the lock held; the
lock-acquisition semantics after the re-exec are identical to today because the pinned copy
re-runs the same acquisition code. The self-deadlock scenario (pin after lock) is enumerated in
D1 precisely so no future edit "simplifies" the ordering.

**Blast radius when a pin fails at 03:00 with no human present** (STALE = fetch/checkout/guard
failure; BROKEN = pin succeeds but the committed driver itself is defective):

| Driver | STALE at fire time | BROKEN target (worst case) |
|---|---|---|
| nightly-eval | Fire proceeds on the working tree — byte-for-byte today's behavior, plus a loud notice. Eval data provenance is UNAFFECTED (own `$WT` pin, V7). | One night's regression-guard data lost; `fail()` alerts controlplane; human sees it in the morning. No banked-data corruption (refuse-to-bank guards are in the driver). |
| os-rotation-filler | Cycle proceeds unpinned; ≤4 controlplane notices/day. | Rotation pauses (retries every 45 min); dashboards stop refreshing; zero data loss (accumulator + cursors are durable-side, D2). |
| mission-recovery | Recovery proceeds unpinned — a stale recoverer beats no recoverer. | Blocked-loop recovery stops → mission stall window regresses to the pre-recovery ≤90 min. Degraded, not dead. |
| rig-watchdog | Checks still run in the SAME fire, after a bounded ≤~22s worst-case pin delay (D5/D6); the terminal `exit 0` (V21) is unchanged. | **The rig's respawn backstop is gone: a SIGKILL'd ollama stays down and every eval job skips until a human intervenes.** This is why the watchdog lands LAST, with the most conservative settings, and why D8's `bash -n` guard exists. Net risk still DECREASES vs today: today the watchdog runs unreviewed working-tree edits; pinned, it runs only CI-green committed code. |

**Cross-driver worktree interference**: none by construction — four distinct pin dirs (D3),
same-SHA fast path (D8), and launchd's single-instance-per-label behavior (inherited, above).

---

## Non-goals

- **`ailang-world`**: separate repo with a hand-synced FORK of the driver — explicitly not ours
  to change from here (its pin arrived with its fork of #666's mission-control wiring; V18).
- **`mission-motoko`**: same driver file in a sibling clone of THIS repo — covered by #666
  already; nothing to roll out.
- **Plist edits / the #618 flag-on install** (D7): not touched, not resequenced.
- **`nightly-lang-eval.sh`** scheduling, pinning, or deletion (Scope; follow-up issue to Mark).
- **Binary-install staleness** (`~/go/bin/ailang` used by daemon/coordinator/filler): a
  different class (V18); the filler's quick-install rework (M2) keeps today's behavior current
  but does not redesign binary distribution.
- **Changing mission-control's pin dir naming** (`v1` stays `v1`).

---

## Milestones

Each is independently landable and independently revertible (a revert restores the exact
current text of one file set; no milestone depends on a later one; M1, M2, M3, M4 each touch
exactly one driver + tests).

**M0 — helper extensions + test hygiene (0.5d).** `pin-root.sh`: `AILANG_DRIVER_FETCH_MAX_AGE`
(D5), same-SHA fast path + `bash -n` guard (D8). New `lib/pin-report.sh` (D6).
`test_pin_root.sh`: env sanitization (D9) + new sections 10–13 (max-age skips fetch; max-age=0/
unset still fetches; same-SHA skips checkout; syntax-broken target ⇒ STALE). `test_driver_notify.sh`
or a new `test_pin_report.sh` covers throttle stamps. **Gate before M1**: one live V1 fire
logging `driver pin: running committed` (the frozen-surface regression check).

**M1 — nightly-eval.sh (1d, FIRST because it is the driver #556 ran through).** Pin block per
D1 (wrapped for `set -e`), D3 pin dir, D4 skip flag, D5 fetch timeout 600s, D6 reporting;
`REPO_DURABLE` redirect of the four V15 site groups. `$WT` logic untouched.

**M2 — os-rotation-filler.sh (1d, the heaviest).** Pin block (after blackout/ollama guards,
before the lock — D1); D5 fetch timeout 300s; `REPO_DURABLE` for `ROLL`/JSONs/legacy-git sites (D2); release-pickup
rework (V16): trigger becomes "pin's `std/VERSION` ≠ `~/.ailang/state/os-filler-last-version`",
actions become `make -C <pin> quick-install` (committed-code build — better provenance than
today's working-tree build) + `os-release-snapshot.sh <old> --reset` + state-file update + end
cycle; **first-run seeding**: absent state file ⇒ write current version, NO reset (a spurious
reset would wipe the active accumulator — this is an explicit test). Path audit of
`os-release-snapshot.sh` + `publish-unified-dashboard.sh` (D2) recorded in the sprint notes.

**M3 — mission-recovery.sh (0.5d).** Kill-switch guards stay first; pin with D3 dir (the V6
collision fix is THE point of this milestone), D5 max-age 3600 + fetch timeout 60s, D6
throttled reporting.

**M4 — rig-watchdog.sh (0.5d, LAST, most conservative).** Pin wrapped so no failure path can
reach past it to affect the checks or the exit code — on STALE execution falls through to the
ollama/server/wedge checks, and the script's only exit remains the terminal `exit 0` at line
114 (V21); D5 max-age 3600 + fetch timeout 20s; D6 heavy throttling.

---

## Acceptance criteria

Baseline on the pristine tree at `996bcccd7`, recorded in V14: `make test-launchd-drivers` is
**GREEN under a clean env** and **RED (9 passed / 26 failed) under a pinned session env**. All
fixture-based ACs below extend suites that `make test-launchd-drivers` executes (V13 — the gate
demonstrably looks at the changed files, and its `bash -n` loop covers every edited driver), so
"make test-launchd-drivers passes" is non-vacuous for every milestone.

- **AC1 (M0/D9)**: `make test-launchd-drivers` passes when invoked WITH
  `AILANG_DRIVER_PINNED`/`MISSION_WORKDIR` exported (simulating a mission-executor session).
  *Baseline: RED (V14: 9/26). Mutation that re-reddens: delete the sanitization preamble.*
- **AC2 (M0/D5)**: new test — pin once against a live fixture origin, then make the origin
  unreachable and pin again with `AILANG_DRIVER_FETCH_MAX_AGE=3600`: second pin SUCCEEDS
  (fetch skipped). *Baseline: RED (helper always fetches ⇒ STALE). Mutation: set max-age
  handling to ignore the env ⇒ red.*
- **AC3 (M0/D8)**: new test — commit a syntax-broken driver to the fixture ref: pin reports
  STALE, does NOT exec. *Baseline: RED (current helper execs into it and the fixture records
  the broken script running/failing). Mutation: drop the `bash -n` guard ⇒ red.*
- **AC4 (M1)**: new rollout test — a stale fixture clone of `nightly-eval.sh` run with
  `AILANG_NIGHTLY_EVAL_DRY_RUN=1`: log contains `driver pin: running committed` and the DRY-RUN
  line executes from the pin worktree path. *Baseline: RED (no pin block ⇒ grep finds
  nothing; the grep's known-positive control is the same assertion against the
  mission-control fixture).* Each driver's dry/guarded fixture gets the analogous assertion in
  M2–M4.
- **AC5 (M1/D2)**: the dry-run path dump (a `AILANG_NIGHTLY_EVAL_DRY_RUN=1` extension printing
  `REPO`, `REPO_DURABLE`, `HIST`) shows `HIST` under the SOURCE clone while `REPO` is the pin
  worktree. *Baseline: RED (`REPO_DURABLE` doesn't exist). Mutation: point `HIST` back at
  `$REPO` ⇒ red.*
- **AC6 (M2/V16)**: fixture with a stubbed `os-release-snapshot.sh` recording its argv: (a)
  first pinned cycle with no state file seeds the file and the stub records NO `--reset`;
  (b) a subsequent cycle after the fixture ref bumps `std/VERSION` records exactly one
  `--reset` for the OLD version and updates the state file. *Baseline: RED (no state-file
  logic exists). Mutation: drop the first-run seed guard ⇒ (a) red.*
- **AC7 (M3/V6)**: fixture run of `mission-recovery.sh` with `MISSION_NAME=v1` exported: the
  pin note names `…/.ailang-driver-pin/mission-recovery`, NOT `…/v1`. *Baseline: RED (no pin
  note at all; control: mission-control fixture's note names its own dir). Mutation: remove
  the `AILANG_DRIVER_PIN_DIR` line ⇒ red.*
- **AC8 (M4/D6)**: fixture run of `rig-watchdog.sh` with an unreachable origin and no prior
  worktree (forced STALE) and stubbed `curl`/`launchctl`: exit code is 0 AND the stub records
  that the ollama check ran. *Baseline: RED (no pin ⇒ the STALE-specific log assertion fails).
  Mutation: let the pin block `return`/`exit` on STALE ⇒ red on both halves.*
- **AC9 (live, per milestone — mirrors the V1 gate)**: ≥3 consecutive live fires logging
  `driver pin: running committed origin/dev @` in the driver's own log
  (`/tmp/ailang-nightly-eval.log`, `/tmp/ailang-os-filler.log`,
  `/tmp/ailang-mission-recovery.log`, `/tmp/ailang-rig-watchdog.log` — V18) with zero
  STALE/`DRIVER PIN FAILED` lines, with the same two controls as V1 (pattern exists in the
  driver source; log readable via a known-present line). Wall-clock to the gate: watchdog
  3 min, recovery 12 min, filler ~2.5h, nightly 3 days — the nightly's gate is therefore the
  last thing that closes #558, and closing the issue waits for it.
- **AC10 (M1–M4/D5)**: new test — for each of the four drivers, extract the numeric value
  assigned to `AILANG_DRIVER_FETCH_TIMEOUT` at the pin call site and assert it is a positive
  integer **strictly less** than that driver's fire cadence (60 / 240 / 2700 / 86400 s — the
  V9 measurements, recorded as constants in the test beside a pointer to V9). The assertion is
  the numeric comparison, NOT variable presence — a presence-grep would pass at any value and
  is exactly the vacuous form this AC exists to forbid. A missing or non-numeric value fails
  the test outright (which doubles as the extractor's known-positive control: once a milestone
  lands, its driver must yield a parseable value). *Baseline: RED for all four (no driver sets
  the variable today). Mutation that re-reddens: raise rig-watchdog's value to 120 — above its
  60s interval — and the comparison fails.* Per milestone, only that milestone's driver is
  required green; all four are required green at M4.

---

## Quorum revision log

**Round 1 (iteration 188): BLOCKED — both reviewers present, two objections, both resolved in
this revision.** Nothing outside D5/D6 (and the Verification Log rows, milestone lines, and
AC10 those two decisions drive) was changed; the 20-row Verification Log, scope resolution,
D1–D4, D7–D9 and the milestone structure stand as reviewed.

1. **`gemini-3-1-pro` — valid as stated; fixed.** Round 1's D6 watchdog cell put "continue and
   `exit 0`" in the STALE-behavior column, where it reads as "exit immediately on pin failure"
   — that reading would skip every downstream respawn check and disable the backstop, which is
   the opposite of the intent. Measurement (V21): the `exit 0` is the script's pre-existing
   TERMINAL statement at `rig-watchdog.sh:114` (114 lines total; no other exit statement
   exists). The cell now says "continue INTO the ollama/server/wedge checks" and pins the
   terminal exit as unmoved and unduplicated; the blast-radius row and M4 were reworded to
   match so no copy of the ambiguous phrasing survives.

2. **`gpt5-6-sol` — premise REFUTED by measurement; substance ADOPTED.** The objection claimed
   D5's fetch "specifies no timeout" and can hang "indefinitely". Both halves of that premise
   are false: the helper already bounds the fetch via `_pin_bounded` under
   `AILANG_DRIVER_FETCH_TIMEOUT` (default 120s) with kill → kill -9 escalation and a typed
   STALE on expiry (V22). But the conclusion survives by a different mechanism: the 120s
   DEFAULT is 2× rig-watchdog's 60s interval and half of mission-recovery's 240s (V9), so one
   hung fetch could still cost the watchdog one-to-two respawn ticks — and
   `AILANG_DRIVER_FETCH_MAX_AGE` bounds fetch *frequency*, not fetch *duration*. Adopted in
   full: (a) D5 now records the existing bound so the doc cannot imply the helper is unbounded;
   (b) every call site sets `AILANG_DRIVER_FETCH_TIMEOUT` explicitly (20/60/300/600s), each
   justified as strictly below its own cadence, with the 120s default relied on nowhere;
   (c) AC10 asserts bound < cadence as a numeric comparison whose stated mutation (raise one
   driver's value above its interval) goes red; (d) D6's absolute "never blocks" claim is
   replaced by the quantified residual — a bounded ≤~22s worst-case delay within a watchdog
   tick, never a disable.

---

## Quorum round 2 — BLOCKED, and PARKED `needs-human-review` (iteration 188)

Artifact `.ailang/state/mission-quorum/m-driver-pin-rollout-2026-08-12T23-45-23Z.json`.
`absent_reviewers: []` — both `gpt5-6-sol` and `gemini-3-1-pro` present, so this is a full-strength
block, not an N−1 degrade. Metered `$0.0984` (round 1: `$0.0794`).

Both objections were **measured first-party by the controller rather than forwarded** (rule 3f),
and **both are confirmed and larger than filed** — see V23 and V24. That measurement is also what
disqualifies the narrow-refinement carve-out: neither reviewer's `proposed_fix` is correct given
what the commands returned, so applying either verbatim would write a false statement into this
document, and the correct resolution would have to be **invented by the controller**. Standing
rule 2 binds; the document parks rather than being force-passed or narrowed until it passes.

The two findings share one root, which is why they cannot be fixed independently: **inside a
pinned context `$REPO`/`$0` resolve to the throwaway pin worktree, and both durable-data writing
and source-clone updating currently depend on them resolving to the source clone.**

**The decision, framed for one word.** Three options, all consistent with the measurements:

- **(A) Exclude `os-rotation-filler.sh` from the rollout.** Pin `nightly-eval.sh`,
  `mission-recovery.sh`, `rig-watchdog.sh` only; the filler stays unpinned and keeps its
  `git pull` (V24), preserving the delivery path for everything else. Cheapest and safest;
  costs the filler's own staleness, which is the `#556` lane.
- **(B) Pin all four and move the source-clone advance into the helper.** `pin-root.sh` gains an
  explicit fast-forward of the source clone when it is clean and 0 ahead (the same precondition
  as Mark's standing fast-forward authorisation), so the delivery path stops depending on any one
  driver. Widest blast radius — it makes every mission's driver write to its clone.
- **(C) Keep the scope and re-root the durable paths explicitly.** Milestones invoke
  `$AILANG_DRIVER_SRC/tools/...` for durable writes instead of relying on cwd (V23), and the
  filler keeps an explicit source-clone `git pull` step executed against `$AILANG_DRIVER_SRC`
  (V24). Preserves the design's intent; adds a second place where `$AILANG_DRIVER_SRC` being
  wrong causes silent data loss.

Until one is chosen, D2, D7, M2 and AC-set are all **unapproved**. Everything else in this
document — the scope resolution (V2), the two-class split (V3), the zero-`claude` measurement
(V4), the `MISSION_NAME=v1` pin-dir collision (V5/D3), the `make test-launchd-drivers` env-leak
bug (V14) and the round-1 fixes (V21/V22) — is banked and survives whichever option is taken.

---

## Related documents

- `tools/launchd/lib/pin-root.sh` (PR #666) — the helper this rolls out; its header is the
  canonical statement of the failure class and the loud-STALE contract.
- [m-ollama-v1-streaming-idle-timeout](m-ollama-v1-streaming-idle-timeout.md) — owner of the
  pending PLIST rollout this design deliberately does not touch (D7/V8).
- `design_docs/v1-mission.md` — charter item + unblocking gate this doc discharges (V1).
