# Sprint plan: codex capacity stop is transient (D-FLEET-26 A, draft #2065)

**Ticket** `codex:server-capacity-error-crashes-finished-iteration` · **Ruling** D-FLEET-26 = A
(attended 2026-11-16, `a82f0180a` on origin/dev) · **Branch** `fleet/i79-dfleet26-capacity-transient`
rebased onto `6998e97e0`, head `d84004daa` (3 commits, +64/-1, 3 files) · **Planner** fleet, 2026-11-16.

> Hygiene rule for every file this sprint touches, this one included: no line may START with the
> codex stop text (in either rendering below). Codex echoes command output into `$LOG` at column 0,
> so a column-0 copy in a file the controller reads would fake a capacity stop. Quote it mid-line or
> indented, and build test fixtures with `$'\n'`, as the existing arm does.

## 1. Premise re-check at HEAD `d84004daa`

**Verdict: the draft is correct for raw `codex exec`, but mission controllers have not run raw
`codex exec` since 2026-10-09.** As written, it never fires on the live controller path. One fix-up
is required (§4, F1).

| # | Draft claim | Command run (worktree root, `/bin/bash`, `/usr/bin/grep` unless noted) | Observed | Status |
|---|---|---|---|---|
| P1 | `TRANSIENT_SIG` lives in `mission-control.sh`; the draft appends the anchored alternative | `grep -n 'TRANSIENT_SIG' tools/launchd/mission-control.sh` | defined at `:1160` (alternative is the last one), used only at `:2687`. `RUNTIME_QUOTA_SIG` is at `:1152` (the ledger's `:1142`/`:1150` are stale) | TRUE |
| P2 | The retry path greps only this attempt's `$LOG` output | `sed -n 2655,2705p tools/launchd/mission-control.sh` | `logpos=$(wc -l < "$LOG")` before `_mc_run_once`. Then `tail -n +$((logpos+1)) "$LOG" \| grep -qiE "$TRANSIENT_SIG"`, gated by `attempt < TRANSIENT_RETRIES`, rc not 143/137, and `RUNTIME_QUOTA_SIG` checked first. **Note `-i`:** the match is case-insensitive | TRUE |
| P3 | Codex output reaches `$LOG` via `>>"$LOG" 2>&1` | `grep -n '>>"$LOG" 2>&1' tools/launchd/mission-control.sh` | `:2536`: `mc_codex_exec … "$PROMPT" >>"$LOG" 2>&1 &` (pi `:2584`, claude `:2590`) | TRUE |
| P4 | That process is `codex exec`, which prints server errors as `ERROR: ` at column 0 | `sed -n 21,28p tools/launchd/lib/codex-auth-profile.sh`; `grep -n mc_select_codex_profile tools/launchd/mission-control.sh` | `:944` calls `mc_select_codex_profile`, which **exports `AILANG_CODEX_RUNTIME=daemon` unconditionally**. `mc_codex_command` then maps `daemon` to **`ailang mission codex-exec`**, not `codex exec`. `git log -S` dates this to `9d9190082`, 2026-10-09: before the draft (11-03) and after World iter 219 (10-02) | **FALSE for the live path** |
| P5 | (derived) What the live path actually prints on a failed turn | `internal/executor/codex/daemon.go` turn/completed handling; `cmd/ailang/commands_platform.go:189`. Real capture: `grep -o 'Error: Codex daemon turn failed.*' /tmp/ailang-mission-triage.log` (line 37559, 2026-11-06 probe on the same `MC_CODEX_COMMAND`) | Go returns `Codex daemon turn failed: <turn.error.message>`, and `ailang` prints `Error: %v` on stderr (no ANSI when not a tty: `od -c` shows plain `Error:`) with rc 1. Captured byte-exact: `Error: Codex daemon turn failed: {"type":"error","status":400,…"model_not_found"}}`. Its payload is the same JSON the draft's exec-form 400 fixture carries after `ERROR: `, so the two renderings share one message | TRUE (measured for 400; capacity text by analogy, see P6) |
| P6 | The capacity message text is `Selected model is at capacity. Please try a different model.` | `strings <codex-darwin-arm64 binary> \| grep -o '.{200}Selected model is at capacity.{200}'` (codex-cli 0.167.0) | Found once, among the CodexErr display strings (`Quota exceeded…`, `Flex capacity unavailable.`, `Policy-restricted capacity is unavailable…`). Control in the same binary: `model_not_found` ×14 | TRUE (string exists); daemon payload = PLAUSIBLE |
| P7 | Draft `TRANSIENT_SIG` vs both renderings | Three one-line fixtures through `grep -ciE "$TRANSIENT_SIG"` (bash, `/usr/bin/grep`) | exec form → **1**, daemon form (`Error: Codex daemon turn failed: Selected…`) → **0**, control `API Error: Overloaded` → **1**. `RUNTIME_QUOTA_SIG` → 0 for all three | **draft misses the live path** |
| P8 | Anchoring rationale: ledger and ticket quote the string mid-line; codex echoes what it reads into `$LOG` | `grep -c 'Selected model is at capacity'` vs `grep -c '^ERROR: Selected model is at capacity'` over `/tmp/ailang-mission-*.log` | world 2/0, triage 2/0, stapledon 1/0, fleet 16/0: every hit is a mid-line quote (ticket JSON `"evidence":`, ledger rows). Daemon writer `codexTextWriter` prints `command_execution` output verbatim, so column-0 lines do survive into `$LOG` | TRUE |
| P9 | A broad `^ERROR:` would be wrong | same loop: `grep -c '^ERROR: '` and `grep -c '^Error: '` | stapledon has 9 column-0 `ERROR:` lines from **Godot** (RID leaks, `Failed to open … log`). There are also 3/10/4/14 column-0 `Error:` lines (fleet/world/stapledon/triage), and `-i` folds those in too | TRUE, and stronger than the draft says |
| P10 | One occurrence in total (10-02) | P8 counts | 0 column-0 hits in any live log, in either form | TRUE (logs rotate; absence is bounded by retention) |
| P11 | "The World fork needs a port" (ledger row) | `grep -o '<string>[^<]*mission-control[^<]*</string>' ~/Library/LaunchAgents/dev.ailang.mission-world.plist`; `ls ~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh` | World's plist runs **this repo's** `tools/launchd/mission-control.sh` with `MISSION_PROFILE=world`. No driver copy exists in ailang-world. Control: the docs and motoko plists DO point at their own repos' drivers | **stale**: no port needed |

**The 5 commits on `mission-control.sh` since 11-03** (`git log --since=2026-11-03`, hunks listed with
`git show <c> -- tools/launchd/mission-control.sh | grep '^@@'`): `74daa5741` (11-04, post-fire transcript
scan), `96882f837` (11-05, scan byte window), `635ea94da` and `33190af48` (11-06, idle fingerprint,
pre-controller), `41c628b94` (11-16, quorum seat, `:1362`). None touch `:1157–1171` or the retry loop
(`:2655–2705`): their hunks fall before the controller starts or after the slot verdict. The seam did not
move. The draft was already wrong at 11-03; the cause is the 10-09 daemon switch, not these commits.

## 2. Test plan

Suite: `tools/launchd/test_controller_capacity.sh` (fake `pi`, zero inference, ~20 s). It awks the
real blocks out of `MC_CAPACITY_DRIVER`, which is how the mutation drill runs. At HEAD: **19/19 PASS**.

**Draft arms (keep):**

| Arm | Kills |
|---|---|
| `codex-capacity-is-transient` (exec form, stderr, rc 1 → retry → COMPLETED, no demotion or re-walk) | revert, or a move into `RUNTIME_QUOTA_SIG` |
| `prose-codex-capacity-not-transient` (backtick/ledger/grep-hit/indented quotes; non-vacuous check) | de-anchor, unanchored literal |
| `codex-other-error-not-transient` (exec-form 400 `ERROR: {…}`) | broadening to `^ERROR:` |

**New arms (F1):**

| Arm | Fixture | Kills |
|---|---|---|
| `codex-daemon-capacity-is-transient` | clone of the exec arm with the line `Error: Codex daemon turn failed: $_cap`, assembled with `$'\n'` | the draft as-is (exec branch only) |
| `codex-daemon-other-error-not-transient` | the byte-exact captured P5 400 line (model name genericised) | broadening the daemon prefix to `^Error: Codex daemon turn failed:` |
| extend `prose-codex-capacity-not-transient` | add a mid-line and an indented daemon-form quote | de-anchoring the daemon branch |

**Mutation drill.** Rows marked *measured* were run by the planner against scratch copies
(`/tmp/capmut-drill/*.sh`, real worktree untouched). The executor must re-run all of them on the final head.

| Mutant (edit to the new `TRANSIENT_SIG` alternative) | Expected killer | Planner result |
|---|---|---|
| M1 drop `^` | prose arm | measured: killed (draft and candidate fix) |
| M2 drop the alternative (revert) | `codex-capacity-is-transient` (+ daemon arm) | measured: killed |
| M3 broaden to `^ERROR:` | `codex-other-error-not-transient` | measured: killed |
| M4 unanchored literal only | prose arm | measured: killed |
| M5 `^ERROR: .*capacity` | **none** | measured: **SURVIVES**. Accept and record: it widens only to codex's sibling capacity strings (`Flex capacity unavailable.`, `Policy-restricted capacity…`), which are a ruling-scope question, not a misclassification of a non-capacity crash. File it as a follow-up note, not an arm that asserts those are crashes |
| M6 literal in `RUNTIME_QUOTA_SIG` instead (option B) | `codex-capacity-is-transient` ("no demotion", "same rung") | measured: killed |
| M7 drop the daemon branch (= draft) | `codex-daemon-capacity-is-transient` | measured with a scratch copy of that arm: draft **fails** it, candidate fix 20/20 |
| M8 broaden both prefixes `^(ERROR\|Error: Codex daemon turn failed):` | `codex-other-error-not-transient` | measured: killed |
| M9 broaden daemon prefix only | `codex-daemon-other-error-not-transient` (its only killer) | predicted; must be run |

## 3. Done-gate (charter: mission-loop-change five-line pre-flight)

Run everything from `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter79-dfleet26`, outside the
sandbox, under `/bin/bash`, unpiped (a piped `make` reports the pipe's rc).

1. **Suites:** `make test-launchd-drivers`, green at base `6998e97e0` and at head. Known load-sensitive
   inherited failures, with banked evidence: `test_cloud_branch_sync.sh` A20 deadline (iter 91),
   `tools/eval/test_motoko_connection_probe.sh` "host too slow" (iter 69, 94), `test_skill_sync.sh`
   git timeout and once `test_controller_capacity.sh` itself under load (iter 66). On a red, re-run
   that suite alone; it must then pass. Do not run gates concurrently with the executor.
2. **Checks:** `make check-changelog check-tmpfile-hygiene check-home-isolation`, `git diff --check origin/dev...HEAD`, and `bash -n tools/launchd/mission-control.sh`.
   Shellcheck driver finding count at base equals head (30 at iter 91).
3. **Column-0 hygiene with control:** `git diff origin/dev...HEAD | /usr/bin/grep -cE '^\+(ERROR|Error: Codex daemon turn failed): Selected model'` → want 0. In the same block, run the pattern on a `printf`-built positive line → want 1.
4. **Dry-runs, healthy AND degraded** (iterations 86–91 recipe). They prove only that the patched driver
   boots and routes; a dry-run exits before the controller, so it never reaches the retry seam. The suite
   and drill cover that. First pick an idle, armed sibling from `triage`, `world`, `stapledon`: no
   `~/.ailang/state/mission-<p>.disabled`, and no live pid in `~/.ailang/state/mission-<p>.pid` (`kill -0`).
   If none qualifies, use the charter's synthetic-HOME `MISSION_PROFILE=fleet` variant.

        W=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter79-dfleet26; SHA=$(git -C "$W" rev-parse --short HEAD)
        (cd "$W" && env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR AILANG_WORKTREE_SWEEP=0 \
          AILANG_DRIVER_PINNED=$SHA MISSION_PROFILE=<p> MISSION_DRY_RUN=1 \
          /bin/bash tools/launchd/mission-control.sh < /dev/null); echo "healthy rc=$?"
        (cd "$W" && env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR AILANG_WORKTREE_SWEEP=0 \
          AILANG_DRIVER_PINNED=$SHA MISSION_PROFILE=<p> MISSION_DRY_RUN=1 \
          MISSION_EXECUTOR_MODEL=codex:bogus-model-i<N> \
          /bin/bash tools/launchd/mission-control.sh < /dev/null); echo "degraded rc=$?"

   Pass: both rc 0 with `DRY RUN ok`; degraded shows the codex probe fail (rc 1, `model_not_found`) and a
   named fallback (`claude:claude-haiku-5-5` historically). `AILANG_DRIVER_PINNED` stops the pin from
   re-execing into origin/dev (`lib/pin-root.sh:153`). `< /dev/null` is load-bearing (iter 90, bash 3.2
   stdin). Before and after, record the profile, the pidfile, and the `~/.ailang/state` entry count.
   Bank everything in `design_docs/fleet-mission-evidence/iteration<N>/gates.md`.
5. Nothing reloaded mid-iteration. Reach comes when the branch is merged to origin/dev, because every
   mission, World included (P11), runs the pinned origin/dev.

## 4. Fix-ups that block landing

- **F1 (blocking, code).** In `tools/launchd/mission-control.sh:1160`, replace the alternative with the
  two-rendering anchored literal: `|^(ERROR|Error: Codex daemon turn failed): Selected model is at capacity`.
  ERE, grep `-i` unchanged. The candidate passed 20/20 in the planner's scratch run. Rewrite the
  `:1166–1171` comment: the controller runs `ailang mission codex-exec` (daemon, since 10-09), which
  renders the stop as `Error: Codex daemon turn failed: …`; raw `codex exec` (0.164.0) uses `ERROR: …`.
  Keep both branches because `mc_codex_command` still has the legacy `codex exec` arm. Add the
  §2 arms. Mark's ruling text, "the anchored literal `Selected model is at capacity`", covers both
  prefixes; nothing widens beyond the literal.
- **F2 (changelog).** Rename the fragment to `changelogs/unreleased/2026-11-16-codex-capacity-transient-retry.md`
  and date the heading 2026-11-16. Say it was ruled A on 2026-11-16, name both renderings and the new
  arms, and drop "(proposed)" wording. Keep every quote of the stop text mid-line.
- **F3 (record, not code).** The ledger row's "World fork needs a port" is stale (P11). Note this in the
  iteration record. The seam's Go side (`internal/executor/codex/daemon.go`, the
  `Codex daemon turn failed: %s` format) is **outside fleet write scope**. If someone rewords it, the
  daemon branch silently goes dead, and only the shell arm's fixture would still pass. Record this as
  a follow-up ticket for a Go-side pin test in `daemon_test.go`, owned outside the fleet.
- **F4 (known residual, out of scope by ruling).** A codex retry starts a NEW daemon thread with the full
  `$PROMPT` (no session resume, unlike pi's `MC_PI_SESSION_ID`). So a finished iteration that hits the
  stop is re-run from Gate 0 rather than resumed. Pushing the finished record branch before exit
  is the separate design the ruling split off.
- **Landing:** force-push the rebased branch to #2065, mark it ready, wait for CI (poll `mergeable`
  too), and merge. Then run `ailang mission ticket resolve codex:server-capacity-error-crashes-finished-iteration --resolution "D-FLEET-26 A landed" --sha <merge sha on origin/dev>`
  (rev-parse it; never hand-expand).

## 5. Milestone and estimate

**M1, the only milestone (~1.5 h executor + gates):**
- F1 code and arms (20 min).
- Suite plus a 9-mutant drill on head (15 min).
- F2 (10 min).
- Done-gate items 1–4 (40 min; `make test-launchd-drivers` alone takes about 5–8 min per run, twice).

Acceptance criteria:
- Suite 21+/21+ PASS.
- M1–M4 and M6–M9 killed, M5 recorded as an accepted survivor.
- Hygiene count 0 with its control at 1.
- Both dry-runs rc 0.
- Gates banked.
- Independent judge before merge.
