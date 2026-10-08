# Sprint plan: M-MISSION-ROTATE-LOG-SAFE

**Design doc**: `design_docs/planned/m-mission-rotate-log-safe.md` (Revision 5, quorum-satisfied via the narrow-refinement carve-out; D-FLEET-9 = A RULED 2026-10-01)
**Sprint ID**: `M-MISSION-ROTATE-LOG-SAFE` · **Planned**: 2026-10-04 · **Baseline worktree HEAD**: `c2bf5bef020f576d1842bf8d75f9cc5006f07683` (clean; `std/VERSION` v0.52.1)
**Executor model**: prescriptive directive — exact files, edit shapes, and test layout below. No judgment-heavy steps. **No git operations: the controller commits.**

## Re-cut rationale

The mission brief's suggested M1 (CLI change **with** its Go tests) totals ~280 LOC (design Files table: mission_cmd.go ~70 + registry.go ~10–20 + new test ~190), over the 150-LOC hard cut. It is split into **M1 (implementation, ~110 LOC)** and **M2 (pinned synthetic tests, ~150 LOC)**. M1 is guarded by the existing hermetic loader tests plus grep/CLI-smoke assertions; M2 pins the new behavior with mutation-drill tests. M3 (driver) and M4 (changelog + done-gate) as suggested.

## Baseline at c2bf5bef0 (every acceptance command run against the pristine worktree)

| # | Command | Base output | Status |
|---|---|---|---|
| B1 | `go build ./cmd/ailang/` | rc 0 | GREEN at base |
| B2 | `go test ./cmd/ailang/ -run 'TestMissionRegistry\|TestRunMissionIteration' -count=1` | `ok github.com/sunholo-data/ailang/cmd/ailang 0.760s` | GREEN at base |
| B3 | `go test ./cmd/ailang/... -run 'Mission' -count=1` | `ok ... 12.640s` | GREEN at base |
| B4 | `go test ./cmd/ailang/ -run 'TestMissionRotateLog\|TestMissionNormalizeShared\|TestMissionRegistryRoot' -count=1` | `ok ... 0.666s [no tests to run]` | GREEN at base (vacuous; M2 makes it load-bearing) |
| B5 | `grep -c "repoRootFor" cmd/ailang/mission_cmd.go` | `4` (:357 call, :383 comment, :384 def, :429 call) | expected to go to 0 |
| B6 | `grep -rn "repoRootFor" cmd/ internal/ --include="*_test.go"` | rc 1 (zero hits) | GREEN at base; must stay zero |
| B7 | `grep -c -- "--status" cmd/ailang/mission_cmd.go` | `4` (:137 help prose, :316 case, :330 unknown-flag text, :336 usage) | expected to go to 1 |
| B8 | `grep -c "AILANG_MISSION_REGISTRY" tools/launchd/mission-control.sh` | `1` (binary branch :1976 only) | expected to go to ≥2 |
| B9 | `/bin/bash -n tools/launchd/mission-control.sh` | rc 0 | GREEN at base |
| B10 | `go build -o bin/ailang ./cmd/ailang && (cd / && "$OLDPWD/bin/ailang" mission rotate-log foo --status)` | `Error: failed to read mission registry missions: open missions: no such file or directory`, rc 1 | feature assertion — expected RED at base (registry-load error today; migration error after M1). CWD `/` has no `missions/` ancestor and the command performs no writes |
| B11 | `make test-launchd-drivers` | FAILS IN-SANDBOX: `mktemp: mkdtemp failed on /var/folders/... Operation not permitted` (suite-env.sh forces its own TMPDIR; sandbox denies) — **UNINFORMATIVE UNDER SANDBOX**, neither pass nor fail; controller re-runs out-of-sandbox | not a red-at-base defect |
| B12 | `ailang pkg search registry` / `ailang pkg search mission` | `Error: search failed: ... storage.googleapis.com/ailang-registry/index.json: Forbidden` (sandbox network) | registry search unavailable from this lane; in-repo reuse recorded instead |

No acceptance command below is red at base except the ones explicitly marked **feature assertion (RED→GREEN)**; their base outputs are recorded above (B4, B10) or are new-file existence checks.

## Registry reuse verdict

`ailang pkg search` is unreachable from this lane (B12, 403). **No package reuse — action `none`.** In-repo reuse: the existing `Mission.root` provenance (`internal/mission/registry.go:83–84`, set by `LoadFile` at :245–246; today read only by `Mission.DriverPath()` at render.go:179) via a new additive `Root()` accessor; `mission.RotateLog` / `mission.Normalize` unchanged; driver fixture follows the awk-extraction pattern of `tools/launchd/test_mission_stall.sh`/`test_controller_chain.sh` and the stub-PATH pattern of `test_mission_iteration.sh`.

---

## M1 — CLI implementation: registry-origin targets, strict `--stream`, ruled `--status` rejection, delete `repoRootFor` (~110 LOC)

**Dependencies**: none.

### Files and edit shapes

1. **`internal/mission/registry.go`** (~+8): additive read-only accessor immediately after the `Driver`/`Path`/`root` field block (fields end :85). Shape:
   ```go
   // Root is the repo root of the checkout that holds the registry this mission was
   // loaded from — the PARENT of the registry directory (LoadFile, :245–246), never the
   // registry directory itself. Empty only for a Mission constructed outside Load.
   func (m *Mission) Root() string { return m.root }
   ```
   No other change to this file; `LoadFile`/`Load` signatures and discovery semantics are untouched (design audit C11 pins this).

2. **`cmd/ailang/mission_cmd.go`** (~+95/−45):
   - **`loadMissionRegistry` (:156–187)**: error-path-only improvement (audit rows C2–C9). In the CWD/ancestor fallback branch, collect tried candidates and wrap the final failure: `return nil, fmt.Errorf("%w (tried: %s)", err, strings.Join(tried, ", "))`. The env-override branch (:157–171) stays byte-identical (pinned by `TestMissionRegistryExplicitInvalidDoesNotFallback`).
   - **`missionRotateLog` flag loop (:313–337)**: replace `case "--status": stream = "status"` (:316–317) with the ruled rejection, and add strict stream parsing. Exact shape:
     ```go
     case "--status":
         return fmt.Errorf("--status was retired (D-FLEET-9, ruled 2026-10-01): it rotated the STATUS archive and was never a report. Use: ailang mission rotate-log <name> --stream status")
     case "--stream":
         if i+1 >= len(args) {
             return fmt.Errorf("--stream needs a value (log|status)")
         }
         stream = args[i+1]
         if stream != "log" && stream != "status" {
             return fmt.Errorf("--stream %q unknown (want: log|status)", stream)
         }
         i++
     ```
     Keep `--keep` and the `default` arm; change the `default` unknown-flag text (:330) to `(want: --keep N, --stream log|status)` and the usage error (:336) to `usage: ailang mission rotate-log <name> [--keep N] [--stream log|status]`. The rejection is structurally BEFORE `loadMissionRegistry()` at :338 (flag loop already precedes the load — V2); do not reorder anything past it.
   - **`missionRotateLog` shared block (:354–358)**: replace the `repoRootFor` call and its silent Workdir rescue with registry-origin provenance that fails loudly on empty root (design decision "Canonical shared log root"; C6):
     ```go
     logDir := m.Workdir
     if m.Repo == sharedRepoSlug {
         root := m.Root()
         if root == "" {
             return fmt.Errorf("mission %q: shared-repo mission loaded without registry origin (Path %q) — refusing to guess a target root", m.Name, m.Path)
         }
         logDir = root
     }
     ```
     The `--stream status` archive remap (:362–366, keys off `stream == "status"`) and the `mission.RotateLog(logPath, keep)` call (:368) stay.
   - **`missionNormalize` (:425–443)**: delete the second CWD walk at :429–431 (`root, rerr := repoRootFor(...)` + `if rerr != nil { return rerr }`). Inside the mission loop replace `dir := root; if m.Repo != sharedRepoSlug { dir = m.Workdir }` with:
     ```go
     dir := m.Workdir
     if m.Repo == sharedRepoSlug {
         root := m.Root()
         if root == "" {
             return fmt.Errorf("mission %q: shared-repo mission loaded without registry origin (Path %q) — refusing to guess a target root", m.Name, m.Path)
         }
         dir = root
     }
     ```
     The loud no-registry failure is preserved: the :425 load error still fires before any target mapping (C7).
   - **DELETE `repoRootFor`** (:383–396, comment + func, V27: zero test references).
   - **Help text** (:134–137): `ailang mission rotate-log <name> [--keep N] [--stream log|status]` and replace the `--status rotates the STATUS-stamp archive instead` prose line (:137) with `--stream status rotates the STATUS-stamp archive instead`.

### Acceptance criteria (commands; baseline column above)

1. `go build ./cmd/ailang/` → rc 0. (B1: GREEN at base.)
2. `go test ./cmd/ailang/ -run 'TestMissionRegistry|TestRunMissionIteration' -count=1` → `ok` — C11 signature/precedence no-regression. (B2: `ok 0.760s` at base.)
3. `grep -c "repoRootFor" cmd/ailang/mission_cmd.go` → `0`. (B5: 4 at base.)
4. `grep -rn "repoRootFor" cmd/ internal/ --include="*_test.go"` → rc 1, zero hits (no test may revive the helper). (B6: rc 1 at base.)
5. `grep -c -- "--status" cmd/ailang/mission_cmd.go` → `1` (only the migration-rejection arm). (B7: 4 at base.)
6. `go build -o bin/ailang ./cmd/ailang && (cd / && "$OLDPWD/bin/ailang" mission rotate-log foo --status); echo rc=$?` → stderr contains `--stream status` and `D-FLEET-9`, does NOT contain `mission registry`, rc nonzero. **Feature assertion (RED→GREEN)** — B10 base: registry-load error. Delete `bin/ailang` after (controller commits; bin/ is ignored).

## M2 — Pinned synthetic Go tests: `cmd/ailang/mission_rotate_test.go` (new, ~150 LOC hard cap)

**Dependencies**: M1.

### Test file layout (package `main`; NO `t.Parallel` — `t.Chdir` is process-global)

Shared helpers (keep tight to stay ≤150 LOC):

- `writeSyntheticRepo(t *testing.T, name string, shared bool, entries int) (repoRoot, regDir string)` — `repoRoot = t.TempDir()`; `regDir = <repoRoot>/missions`; write `<regDir>/<name>.toml` with the TOML shape pinned to `cmd/ailang/mission_registry_env_test.go:14` (`name`, `repo` = `sunholo-data/ailang` if shared else `example/external`, absolute `workdir` = a separate `t.TempDir()`, `doc = "README.md"`, `[schedule]` `mode="interval"`, `interval_seconds=21600`, distinct `boot_offset`). Write `<repoRoot>/design_docs/<name>-mission-log.md` and `<workdir>/design_docs/<name>-mission-log.md` with the record-stream shape pinned to `internal/mission/rotate_test.go:13–19` (`# <Name> Mission Log\n\nPreamble that must survive rotation.\n` then `## N — 2026-09-DD — Did thing number N [TAGN]` headings). Write `<repoRoot>/design_docs/<name>-mission-status-archive.md` in the same pure-record shape (needed by test 5). Return the paths.
- `snapshot(t, paths...) map[string]string` / byte-compare helper for the byte-identical assertions.

Tests (each names the mutation it kills — mutation-drill discipline; the evaluator re-runs these):

1. `TestMissionRotateLogSharedRepoTargetsRegistryOrigin` — A := writeSyntheticRepo(shared, 10 entries); B := writeSyntheticRepo(same name, shared, 10 entries). `t.Setenv("AILANG_MISSION_REGISTRY", filepath.Join(A.regDir))` (absolute); `t.Chdir(B.repoRoot)`. Run `missionRotateLog([]string{name, "--keep", "5"})`. Assert: A's live log at the ABSOLUTE path `<A.repoRoot>/design_docs/<name>-mission-log.md` — the registry directory's PARENT (round-3 objection-2 pin: an implementation writing to `<regDir>/design_docs/...` one level low MUST fail this test) — shrank (no `## 1 —`, keeps `## 10 —`); A's archive/index exist; **B's live log byte-identical**; A's `workdir` clone log byte-identical. **Kills**: (a) reinstating the `repoRootFor` CWD walk (would rotate B), (b) silent Workdir rescue, (c) root = registry dir instead of its parent.
2. `TestMissionRotateLogExternalMissionUsesWorkdir` — external (non-`sharedRepoSlug`) mission; rotate-log writes only its synthetic workdir's design_docs; the registry repo's design_docs byte-identical. **Kills**: shared-origin mapping applied unconditionally.
3. `TestMissionRotateLogStatusRejectedBeforeRegistryLoad` — `t.Chdir(t.TempDir())` (no `missions/` ancestor), `t.Setenv("AILANG_MISSION_REGISTRY", "")`. `err := missionRotateLog([]string{"x", "--status"})`; assert err non-nil, `strings.Contains(err.Error(), "--stream status")`, and `!strings.Contains(err.Error(), "mission registry")`. **Kills**: rejection moved after `loadMissionRegistry()`; also kills silently mapping `--status` to the stream.
4. `TestMissionRotateLogStatusRejectedLeavesFilesByteIdentical` — synthetic repo + env override; snapshot live log, `-archive.md`, `-status-archive.md`, both indexes; run with `--status`; assert the migration error AND every snapshotted file byte-identical. **Kills**: rejection placed after any write (design: "Failures before mutation leave synthetic target files byte-identical").
5. `TestMissionRotateLogStreamStatusRotatesStatusArchive` — env override; run `--stream status --keep 5`; assert the status archive shrank, `<name>-mission-status-archive-old.md` and the status index were written (naming per `internal/mission/rotate.go:207–213`), and the default live log byte-identical. **Kills**: dropping the status-archive target in the flag rename.
6. `TestMissionRotateLogRejectsUnknownStream` — `--stream bogus` → error containing `log|status`; snapshot unchanged. **Kills**: unvalidated stream value.
7. `TestMissionNormalizeSharedRepoTargetsRegistryOrigin` — A, B as in test 1, but A's live log carries the wordy heading `## Iteration 3 — 2026-09-30 — Did the thing` (pinned rewritable by `wordyEntryRe`, `internal/mission/normalize.go:45–46`); B's log carries the SAME heading. Env override A; `t.Chdir(B.repoRoot)`; run `missionNormalize([]string{"--apply"})`. Assert: A's log now carries the canonical `## 3 — 2026-09-30 — Did the thing`; B's log byte-identical. **Kills**: normalize keeping its own `repoRootFor` walk at old :429.
8. `TestMissionNormalizeNoRegistryStillFailsLoudly` — `t.Chdir(t.TempDir())`, env empty; `missionNormalize(nil)` returns an error containing `mission registry`. **Kills**: any fallback that lets normalize proceed without a loaded registry (C7's preserved loud stop).
9. `TestMissionRegistryRootAccessor` — load a synthetic registry via `loadMissionRegistry()` with env override; assert `m.Root()` equals `filepath.Dir(<regDir>)` (the parent). **Kills**: accessor returning the registry dir itself (one-level-low).

### Acceptance criteria

1. `go test ./cmd/ailang/ -run 'TestMissionRotateLog|TestMissionNormalizeShared|TestMissionNormalizeNoRegistry|TestMissionRegistryRoot' -count=1 -v` → all 9 tests PASS, none vacuous. (B4: `ok ... [no tests to run]` at base — **feature assertion, RED→GREEN**; at base the selector matches nothing.)
2. `go test ./cmd/ailang/... -run 'Mission' -count=1` → `ok` (whole mission surface, no regressions). (B3: `ok 12.640s` at base.)
3. `wc -l cmd/ailang/mission_rotate_test.go` → ≤ 160 (cap + import/comment slack; split helpers into the same file, do not exceed).
4. Mutation drill (evaluator re-runs): revert each mutation named in tests 1–9 one at a time; the named test must go red. Executor records one drill row: with the M1 shared-block mutation `logDir = m.Workdir` (dropping the registry origin), tests 1 and 7 must FAIL.

## M3 — Driver: legacy-branch `AILANG_MISSION_REGISTRY` export + validation + bash 3.2 fixture (~75 LOC)

**Dependencies**: none (independent of M1/M2; order after M2 only for review convenience).

### Files and edit shapes

1. **`tools/launchd/mission-control.sh`** (~+12/−1). Add ONE function beside the other `_mc_*` helpers (before its first call site), bash 3.2-compatible, no local arrays:
   ```bash
   _mc_export_mission_registry() {
     if [ ! -d "$MC_DRIVER_ROOT/missions" ]; then
       log "AILANG_MISSION_REGISTRY root $MC_DRIVER_ROOT/missions is missing — refusing to spawn children without the pinned registry"
       exit 2
     fi
     export AILANG_MISSION_REGISTRY="$MC_DRIVER_ROOT/missions"
   }
   ```
   - Binary branch (:1976): replace the inline `export AILANG_MISSION_REGISTRY="$MC_DRIVER_ROOT/missions"` with `_mc_export_mission_registry` (same position, before `exec ailang mission iterate` at :1977 — behavior identical, V6).
   - Legacy branch: call `_mc_export_mission_registry` immediately beside the `MISSION_DRIVER_ROOT` export at :2167 (before the controller spawn at :2292/:2296; legacy-only region — the binary branch exec'd at :1977 — V20).
2. **`tools/launchd/test_mission_registry_env.sh`** (new, ~60 LOC), modelled on the awk-extraction pattern of `test_mission_stall.sh:25–38` and the TMP/trap discipline of `test_mission_iteration.sh:4–7`. Arms:
   - Extraction: `awk '/^_mc_export_mission_registry\(\) \{/,/^\}$/' "$DRIVER"` non-empty and containing `AILANG_MISSION_REGISTRY` (guard against vacuous extraction, per the stall-suite seam guard).
   - Arm 1 (export): with `MC_DRIVER_ROOT="$TMP/driver"` and `mkdir -p "$TMP/driver/missions"`, source the extract, stub `log(){ :; }`, call it, assert `$AILANG_MISSION_REGISTRY` = `$TMP/driver/missions`. **Kills**: export dropped or pointed at another root.
   - Arm 2 (missing root): without `missions/`, the call exits 2 in a subshell and the message names the missing path. **Kills**: validation removed (silent success).
   - Arm 3 (both branches wired): `grep -c "_mc_export_mission_registry" "$DRIVER"` = 3 (def + binary call + legacy call), and `awk '/AILANG_MISSION_WORK_ITEM/,/^fi$/' "$DRIVER" | grep -c "_mc_export_mission_registry"` = 1 (binary-branch call inside the guard, before exec). **Kills**: legacy-branch call deleted (the V20 gap regresses) or binary-branch call deleted.
   - Arm 4 (ordering): the legacy call's line number > the binary branch's closing `fi` line number, so the export cannot run before branch selection. **Kills**: export hoisted above the branch guard.
3. **`make/test.mk`** (+1): add `@$(LAUNCHD_SUITE) tools/launchd/test_mission_registry_env.sh` immediately after the `test_mission_iteration.sh` line in `test-launchd-drivers` (:89 today).

### Acceptance criteria

1. `/bin/bash -n tools/launchd/mission-control.sh` → rc 0. (B9: GREEN at base.)
2. `grep -c "AILANG_MISSION_REGISTRY" tools/launchd/mission-control.sh` → ≥ 2. (B8: 1 at base.)
3. `grep -c "_mc_export_mission_registry" tools/launchd/mission-control.sh` → `3`. (Base: 0 — new-symbol assertion.)
4. `/bin/bash tools/launchd/test_mission_registry_env.sh` → PASS all 4 arms. (**RED→GREEN**: file absent at base.)
5. `make test-launchd-drivers` → rc 0 with the new suite included. **UNINFORMATIVE UNDER SANDBOX** (B11: mktemp EPERM in-sandbox at base) — the controller re-runs `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` OUTSIDE the sandbox and records rc; in-sandbox results are labeled UNINFORMATIVE UNDER SANDBOX in the executor's report, neither pass nor fail.

## M4 — Changelog + mission-loop-change done-gate (~10 LOC)

**Dependencies**: M1, M2, M3.

### Files

1. **`changelogs/unreleased/2026-10-04-mission-rotate-log-safe.md`** (new, ~8 lines), fragment convention per `scripts/changelog_fold.sh:10` (V16): `### Changed — mission rotate-log resolves shared targets from the loaded registry, and `--status` is retired` — name the `--status` → `--stream status` migration (D-FLEET-9, immediate break, no deprecation period), the one-resolved-registry fix for rotate-log AND normalize (explicit override now wins over CWD for shared targets), and the legacy-controller `AILANG_MISSION_REGISTRY` export.

### Acceptance criteria (mission-loop-change done-gate)

1. `go test ./cmd/ailang/... -run 'Mission' -count=1` → `ok`. (B3: `ok 12.640s` at base.)
2. `grep -c "stream status" changelogs/unreleased/2026-10-04-mission-rotate-log-safe.md` → ≥ 1. (**RED→GREEN**: file absent at base.)
3. `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` → rc 0 — **controller-run outside the sandbox** (B11: UNINFORMATIVE UNDER SANDBOX in-lane).
4. Healthy dry-run on an IDLE SIBLING mission profile (the fleet cannot dry-run itself mid-iteration), pinned to the tested SHA via the worktree driver — controller-run, tokens only for probes:
   `AILANG_DRIVER_PINNED=worktree-test MISSION_WORKDIR=<sibling checkout> MISSION_PROFILE=<idle sibling, e.g. docs or world — NOT fleet> MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh 2>&1 | tail -3`
   → prints `DRY RUN ok` and `lanes=ok`; the driver banner names the tested SHA (worktree driver, not the pin).
5. Degraded dry-run (drought simulation), same profile: add `MISSION_DESIGNER_MODEL='claude:claude-drought-sim' MISSION_EVALUATOR_MODEL='claude-drought-sim'` → output matches `lanes=DEGRADED` and names each degraded handoff (mission-loop-change Gate 4).
6. Any test that binds sockets or writes outside the worktree (dry-runs, launchd suite, provider probes) is **controller-run out-of-sandbox**; results obtained inside the executor's sandbox are labeled **UNINFORMATIVE UNDER SANDBOX** in the executor's report, neither pass nor fail.

## Out of scope (do not touch)

- `internal/mission/rotate.go`, `internal/mission/normalize.go` — rotation/normalization semantics unchanged (V5).
- The shared loader's discovery order (env → CWD/ancestors → error) for every caller (design audit: only C6/C7 target mapping and C6's flag loop change).
- Gate-4 skill resources calling default `rotate-log --keep 20` — no behavioral edit needed (V7).
- No git operations; no rotation of real mission logs; synthetic fixtures only under `t.TempDir()`/`mktemp -d`.
