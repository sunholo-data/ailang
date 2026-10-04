# Independent evaluation — M-MISSION-ROTATE-LOG-SAFE (fleet iteration 18)

**Evaluator**: independent judge (NOT GLM-5.3 designer, NOT Kimi-K3 planner, NOT DeepSeek executor)
**Date**: 2026-10-04
**Worktree HEAD**: `60f2b7716` (clean; branch `mission/fleet-iter18-evaluator`)
**Scope**: design contract compliance + acceptance-criteria re-run + mutation-matrix re-run + report-honesty audit
**Inputs reviewed**: design Rev 5 (D-FLEET-9 = A, narrow-refinement carve-out), sprint plan M1–M4, diff `3391da544..60f2b7716`

## Verdict

**PASS — score 97/100 (70 to pass).**

| Component | Score |
|---|---|
| Correctness vs design | 30 / 30 |
| Test quality + mutation coverage | 29 / 30 |
| Acceptance-criteria satisfaction | 19 / 20 |
| Risk of regression | 10 / 10 |
| Report honesty | 9 / 10 |

**No blocking findings.** One minor adjudication (AC5 count) and one robustness note, both non-blocking.

## Re-run gate table (in-sandbox, evaluator worktree)

| # | Command | Plan expectation | Result | Status |
|---|---|---|---|---|
| 1 | `go test ./cmd/ailang/ -run 'TestMissionRotateLog\|TestMissionNormalizeShared\|TestMissionNormalizeNoRegistry\|TestMissionRegistryRoot' -count=1 -v` | 9/9 PASS | 9/9 PASS, no skips, no vacuous runs (each `-v` shows real rotation/normalize/registry assertions) | **PASS** |
| 2 | `go test ./cmd/ailang/... -run 'Mission' -count=1` | ok | `ok github.com/sunholo-data/ailang/cmd/ailang 11.801s` (and 0.874s in the focused 9-test re-run) | **PASS** |
| 3 | `grep -c "repoRootFor" cmd/ailang/mission_cmd.go` | 0 | 0 | **PASS** |
| 3b | `grep -rn "repoRootFor" cmd/ internal/ --include="*_test.go"` | rc 1, zero hits | rc 1, zero hits | **PASS** |
| 3c | `grep -rn "repoRootFor" cmd/ internal/ tools/ scripts/` (broader sweep) | zero hits | rc 1, zero hits | **PASS** |
| 4 | `grep -c "AILANG_MISSION_REGISTRY" tools/launchd/mission-control.sh` | ≥ 2 | 2 | **PASS** |
| 4b | `grep -c "_mc_export_mission_registry" tools/launchd/mission-control.sh` | 3 | 3 (def at :1968, binary call at :1986, legacy call at :2178) | **PASS** |
| 4c | `grep -c -- "--status" cmd/ailang/mission_cmd.go` (AC5) | plan: 1; executor: 2 | **2** (case label :325 + error message :326 — see adjudication below) | **PASS** (adjudicated) |
| 5 | `go build ./cmd/ailang/` | rc 0 | rc 0 | **PASS** |
| 6 | `/bin/bash -n tools/launchd/mission-control.sh` | rc 0 | rc 0 | **PASS** |
| 7 | `wc -l cmd/ailang/mission_rotate_test.go` | ≤ 160 | 159 | **PASS** |
| 7b | `grep -c "t.Parallel" cmd/ailang/mission_rotate_test.go` | 0 | 0 | **PASS** |
| 7c | `grep -nE '"/Users/[^"]*ailang[^"]*"' cmd/ailang/mission_rotate_test.go` (real-path writes) | rc 1, zero | rc 1, zero | **PASS** |
| 8 | `grep -c "stream status" changelogs/unreleased/2026-10-04-mission-rotate-log-safe.md` | ≥ 1 | 1 | **PASS** |
| 9 | `go test ./cmd/ailang/ -run 'TestMissionRegistry\|TestRunMissionIteration' -count=1` (C11 hermetic regression control) | ok | `ok ... 0.731s` | **PASS** |
| 10 | `/bin/bash tools/launchd/test_mission_registry_env.sh` (4-arm driver fixture) | PASS | `PASS mission registry env: export arm, missing-root refusal, both branches wired, ordering` (rc 0) | **PASS** |
| 11 | M1 AC#6 ordering proof: `go build -o .test_ailang ./cmd/ailang && (cd <no-missions dir> && AILANG_MISSION_REGISTRY= ./.test_ailang mission rotate-log foo --status)` | stderr contains `--stream status` AND `D-FLEET-9`, does NOT contain `mission registry`, rc nonzero | stderr = `Error: --status was retired (D-FLEET-9, ruled 2026-10-01): it rotated the STATUS archive and was never a report. Use: ailang mission rotate-log <name> --stream status`, rc=1, no `mission registry` text | **PASS** (observable ordering proof: rejection fires BEFORE any `loadMissionRegistry()` call) |
| 12 | `make test-launchd-drivers` | rc 0, new suite included | **UNINFORMATIVE UNDER SANDBOX** in evaluator session (mktemp/sandbox binding hazards). Plan flagged this; the 4-arm fixture (gate 10) is the in-sandbox equivalent and passes. Controller-run out-of-sandbox: CONFIRMED green by the controller (per input). | **LABEL**: UNINFORMATIVE UNDER SANDBOX in-lane; controller-CONFIRMED out-of-sandbox |
| 13 | Mission-loop-change healthy/degraded dry-runs (M4 ACs 4–5) | rc 0, `DRY RUN ok`, `lanes=ok` / `lanes=DEGRADED` | **UNINFORMATIVE UNDER SANDBOX in-lane** (plan flagged). Controller-run out-of-sandbox, tokens only for probes. | **LABEL**: UNINFORMATIVE UNDER SANDBOX in-lane; controller-run required |

Worktree state after all evaluator runs: `git status` reports clean; mutations were applied one-at-a-time and restored from a temporary `git diff`-equivalent (cp to a same-dir hidden file → edit → test → restore → md5-verify); the committed bytes at `60f2b7716` are unchanged.

## Mutation matrix (re-run, one at a time, restored after each)

All three mutations: **RED on the named test(s), GREEN on restore**. The same pattern of edit+test+restore was used; the working tree was `git status --short`-clean after each restoration (md5 of `cmd/ailang/mission_cmd.go` returned to `279f69a1fd3cf19bc75f111bc9deaa1c`).

### Mutation 1 — rotate-log shared block: drop registry origin, keep `m.Workdir`

**Edit** (in `cmd/ailang/mission_cmd.go`, the `logDir = root` line of the shared block at :379):

```diff
-		logDir = root
+		logDir = m.Workdir // MUTATION: drop registry origin, keep Workdir
```

**Run**: `go test ./cmd/ailang/ -run 'TestMissionRotateLogSharedRepoTargetsRegistryOrigin' -count=1 -v`

**Result**: **FAIL** — `mission_rotate_test.go:74: missing artifact rotcanary-mission-log-archive.md`, `mission_rotate_test.go:74: missing artifact rotcanary-mission-log-index.md`, `mission_rotate_test.go:60: /tmp/.../rotcanary-mission-log.md changed (expected byte-identical)`. Verbatim failure line: `--- FAIL: TestMissionRotateLogSharedRepoTargetsRegistryOrigin (0.04s)`. This is exactly the regression the design predicts: the silent Workdir rescue rotates the WRONG repo (B's `design_docs/...` instead of A's) and leaves A untouched.

**Restore**: file md5 returned to `279f69a1fd3cf19bc75f111bc9deaa1c`. Confirmed clean.

### Mutation 2 — normalize shared block: drop registry origin, keep `m.Workdir`

**Edit** (in `cmd/ailang/mission_cmd.go`, the `dir = root` line of the normalize shared block at :443):

```diff
-			dir = root
+			dir = m.Workdir // MUTATION: drop registry origin, keep Workdir
```

**Run**: `go test ./cmd/ailang/ -run 'TestMissionNormalizeSharedRepoTargetsRegistryOrigin' -count=1 -v`

**Result**: **FAIL** — `0 heading(s) rewritten, 0 unconvertible`, then verbatim: `mission_rotate_test.go:141: shared normalize did not rewrite at registry origin: # normcanary Mission Log\n\nPreamble that must survive rotation.\n\n## Iteration 3 — 2026-09-30 — Did the thing\n\nbody`. The full line: `--- FAIL: TestMissionNormalizeSharedRepoTargetsRegistryOrigin (0.00s)`. Confirms normalize silently runs against B's repo when CWD is B and the registry override is A.

**Restore**: file md5 returned to `279f69a1fd3cf19bc75f111bc9deaa1c`. Confirmed clean.

### Mutation 3 — `--status` rejection: restore pre-D-FLEET-9 behavior

**Edit** (in `cmd/ailang/mission_cmd.go`, the rejection case at :325–326):

```diff
 	case "--status":
-		return fmt.Errorf("--status was retired (D-FLEET-9, ruled 2026-10-01): it rotated the STATUS archive and was never a report. Use: ailang mission rotate-log <name> --stream status")
+		stream = "status" // MUTATION: pre-D-FLEET-9 behavior, --status sets stream
```

**Run**: `go test ./cmd/ailang/ -run 'TestMissionRotateLogStatusRejected' -count=1 -v`

**Result for `TestMissionRotateLogStatusRejectedBeforeRegistryLoad`**: **FAIL** — verbatim: `mission_rotate_test.go:97: bad --status rejection: failed to read mission registry missions: open missions: no such file or directory (tried: /tmp/.../missions, ...)`. The test expected an error containing `--stream status` and NOT containing `mission registry`; the mutation makes the rejection disappear entirely, so the registry-load error is what surfaces — proving the rejection is structurally inside the flag loop and that nothing between the loop and the load intercepts `--status`.

**Result for `TestMissionRotateLogStatusRejectedLeavesFilesByteIdentical`**: **FAIL** — `mission_rotate_test.go:110: bad migration rejection: <nil>` AND `mission_rotate_test.go:60: /tmp/.../rejcanary-mission-status-archive.md changed (expected byte-identical)` AND `mission_rotate_test.go:60: /tmp/.../rejcanary-mission-status-index.md changed (expected byte-identical)`. The status archive and its index were actually written, and `err` came back nil.

**Restore**: file md5 returned to `279f69a1fd3cf19bc75f111bc9deaa1c`. Confirmed clean.

### Mutation matrix summary

| Mutation | Test that must go red | Result |
|---|---|---|
| 1 (rotate-log shared block → `logDir = m.Workdir`) | `TestMissionRotateLogSharedRepoTargetsRegistryOrigin` | **RED** — wrong repo rotated, A's artifacts missing |
| 2 (normalize shared block → `dir = m.Workdir`) | `TestMissionNormalizeSharedRepoTargetsRegistryOrigin` | **RED** — 0 headings rewritten, A's wordy heading preserved |
| 3 (`--status` → `stream = "status"`) | `TestMissionRotateLogStatusRejectedBeforeRegistryLoad` AND `TestMissionRotateLogStatusRejectedLeavesFilesByteIdentical` | **BOTH RED** — registry-load error surfaces, status archive mutated |

All three mutations are killed by their named tests, confirming the test suite actually pins the design's regressions.

## AC5 = 2 adjudication

The plan predicts `grep -c -- "--status" cmd/ailang/mission_cmd.go` → `1` (B7 at base: 4). The executor reported 2. The evaluator's re-run returns **2**, both inside the same case arm:

```
325:		case "--status":
326:			return fmt.Errorf("--status was retired (D-FLEET-9, ruled 2026-10-01): it rotated the STATUS archive and was never a report. Use: ailang mission rotate-log <name> --stream status")
```

- Line 325 is the `case` label (the rejection's switch arm).
- Line 326 is the error message text, which must name the rejected flag (`--status`) for the user AND name the replacement (`--stream status`) per the design's ruled migration text.

These are NOT two separate arms; they are the same arm. The plan's "1" was a quick text-search count that assumed the case label and the error message were merged onto one line. The plan's *intent* — B7 base 4 → 1 reference that does something with `--status` (vs the original 4 where the flag actually mutated stream) — is satisfied: the entire `--status` footprint in the file is now one rejection case, and the count=2 is solely because the case label and the error message text both happen to contain the literal `--status` string. The error message naming the rejected flag is a design requirement, not a defect.

**Adjudication: ACCEPTABLE — the discrepancy is a counting-method artifact, not a missing or extra arm. Score 0 deductions.**

## Design-contract compliance

| # | Requirement | Status | Evidence |
|---|---|---|---|
| (a) | `--status` rejected BEFORE `loadMissionRegistry` and any write; ruled migration text | **PASS** | rejection in flag loop at :325–326; `loadMissionRegistry()` first reached at :356; M1 AC#6 ordering proof (in-sandbox `cd <no-missions>; ailang mission rotate-log foo --status` returns the D-FLEET-9 message, not the registry error). Mutation 3 (revert rejection) makes `TestMissionRotateLogStatusRejectedBeforeRegistryLoad` see the registry error instead. |
| (b) | shared-repo targets = `m.Root()` (parent of registry dir) in BOTH rotate-log and normalize, loud empty-root failure | **PASS** | `logDir = root` at :379, `dir = root` at :443; identical empty-root `return fmt.Errorf(... "refusing to guess a target root" ...)` at :377 and :441. `m.Root()` accessor at `internal/mission/registry.go` returns `m.root`, populated by `LoadFile` to `filepath.Dir(regDir)` (the parent). `TestMissionRegistryRootAccessor` enforces `m.Root() == filepath.Dir(reg) AND m.Root() != reg` — the round-3 objection-2 pin. |
| (c) | `repoRootFor` deleted, zero references anywhere incl. tests | **PASS** | `grep -rn "repoRootFor" cmd/ internal/ tools/ scripts/` → rc 1, zero hits. Function and its callers removed in M1. |
| (d) | loader discovery semantics unchanged for every other caller (audit C1–C11) | **PASS** | `loadMissionRegistry()` signature unchanged (`func() (*mission.Registry, error)`). The env-override branch (`if dir := config.MissionRegistry(); dir != ""`) is byte-identical (pinned by `TestMissionRegistryExplicitInvalidDoesNotFallback`). The CWD/ancestor walk is preserved; only the error-text improvement was added (the `tried []string` wrap). `mission.Load(` only appears inside the loader at :168 and :189. All 11 caller rows (C1–C11) still call the same loader; hermetic regression suite (TestMissionRegistry*, TestRunMissionIteration) passes in 0.731s. |
| (e) | driver: both branches export `AILANG_MISSION_REGISTRY` with validation, 4-arm fixture | **PASS** | `_mc_export_mission_registry()` defined at :1968–1974 with `[ ! -d "$MC_DRIVER_ROOT/missions" ]` validation and `exit 2` on missing; binary-branch call at :1986 (just before `exec ailang mission iterate` at :1987); legacy-branch call at :2178 (right after `MISSION_DRIVER_ROOT` export). 4-arm fixture `tools/launchd/test_mission_registry_env.sh` runs and PASSes all 4 arms: export, missing-root refusal (exit 2 + path-named), both-branches-wired (count = 3; binary-branch call inside `AILANG_MISSION_WORK_ITEM` guard = 1), ordering (legacy line > binary fi). |
| (f) | M2 test file ≤160 lines, no `t.Parallel`, fixtures synthetic (`t.TempDir`), no writes to real mission logs | **PASS** | 159 lines; 0 `t.Parallel`; every path under `t.TempDir()` (lines 13, 94, 145); no `/Users/.../ailang...` absolute paths anywhere; `writeSyntheticRepo` helper writes TOML + log + status-archive under the tempdir. |

## Per-milestone assessment

| Milestone | One-line assessment |
|---|---|
| **M1** (CLI implementation) | **PASS** — registry-root provenance, strict `--stream`, ruled `--status` rejection, and `repoRootFor` deletion all match the plan's edit shapes verbatim; observable ordering proof (AC#6) is repeatable in-sandbox. |
| **M2** (pinned synthetic tests) | **PASS** — 9 tests, all real (none vacuous — every `-v` run shows substantive assertions or rotation/normalize output), each names the mutation it kills; 159 lines; mutation matrix confirms all 3 named mutations go red. |
| **M3** (driver export + 4-arm fixture) | **PASS** — function definition, both branch call sites, validation, and exit code all match plan; 4-arm fixture passes; wired into `make/test.mk` line 91. |
| **M4** (changelog) | **PASS** — `changelogs/unreleased/2026-10-04-mission-rotate-log-safe.md` present, contains `stream status` AND `D-FLEET-9` AND names both code paths (rotate-log AND normalize) and the legacy controller branch export. Mission-loop-change done-gate items 1–3 verifiable in-sandbox; 4–5 are UNINFORMATIVE UNDER SANDBOX in-lane and require the controller's out-of-sandbox dry-run per the plan. |

## Findings

### Blocking

**None.**

### Non-blocking

1. **AC5 = 2 (vs plan's predicted 1)** — both `--status` matches are the same case arm (line 325 case label + line 326 error message). The error message *must* name the rejected flag, and the plan's intent (one rejection arm, no working `--status` behavior) is satisfied. No code change recommended; the plan's "1" was a counting-method prediction, not a contract.

2. **M3 fixture fragility** — arms 3 and 4 use `awk` pattern-matching against the driver's source layout (`/AILANG_MISSION_WORK_ITEM/,/^fi$/`, the `if [ -n "${AILANG_MISSION_WORK_ITEM:-}" ]; then` line, the `^fi$` close). If a future driver edit moves the binary branch's guard, these assertions need to be updated. This is the same fragility `test_mission_stall.sh:25–38` (cited by the plan) accepts; not a current defect. Worth a one-line comment in the fixture explaining the seam guard (the `awk` extraction at the top of the file is the seam guard for arm 1; the structure-matching arms have no equivalent).

3. **Test 9 is the round-3 objection-2 pin but is implicit, not drilled** — `TestMissionRegistryRootAccessor` checks `m.Root() != reg` (so a `Root() = m.Path` accessor would fail), but the mutation matrix did not include a fourth drill that mutates the accessor itself. The plan did not ask for it; the test is correct; this is a coverage note, not a finding.

4. **In-sandbox vs out-of-sandbox gate labeling** — the evaluator followed the plan's labeling discipline: items 12 and 13 are `UNINFORMATIVE UNDER SANDBOX in-lane` and rely on the controller's out-of-sandbox confirmation (provided in the input). The 4-arm fixture (gate 10) is the in-sandbox equivalent for the driver contract and PASSes.

## Report-honesty audit

| Executor claim | Verifier finding |
|---|---|
| 9/9 new tests pass | **VERIFIED** in-sandbox (re-run, 0 skips, every `-v` body shows real assertions) |
| whole Mission surface ok | **VERIFIED** (`go test ./cmd/ailang/... -run 'Mission' -count=1` → `ok ... 11.801s`) |
| `make test-launchd-drivers` 59 arms + registry-env suite PASS | **LABELED** UNINFORMATIVE UNDER SANDBOX in-lane (sandbox mktemp hazard); **CONTROLLER-CONFIRMED** out-of-sandbox per input |
| Mutation 1 (rotate shared block) reddens `TestMissionRotateLogSharedRepoTargetsRegistryOrigin` | **RE-RUN, VERIFIED RED** (verbatim in matrix above) |
| Mutation 2 (normalize shared block) reddens `TestMissionNormalizeSharedRepoTargetsRegistryOrigin` | **RE-RUN, VERIFIED RED** (verbatim in matrix above) |
| Mutation 3 (remove `--status` rejection) reddens `TestMissionRotateLogStatusRejected*` | **RE-RUN, VERIFIED RED on BOTH tests** (verbatim in matrix above) |
| `--status` count in mission_cmd.go = 2 | **VERIFIED = 2**; adjudicated as acceptable (same arm, two lines) |
| `repoRootFor` deleted everywhere | **VERIFIED** (broader sweep `cmd/ internal/ tools/ scripts/` → zero hits, rc 1) |
| changelog file present with `stream status` and D-FLEET-9 | **VERIFIED** |

The executor report's claims all hold up under independent re-run. The only unverifiable claim (`make test-launchd-drivers` end-to-end) is correctly labeled by the plan as out-of-sandbox for in-lane work and was controller-run.

## Final verdict

**PASS — 97/100.**

The implementation is correct against the design Rev 5, the test suite genuinely pins the regressions (mutation matrix confirmed), the 4-arm driver fixture is sharp, the M3 driver edits match the plan, and the M4 changelog covers both code paths and the legacy controller export. The one count discrepancy (AC5 = 2 vs plan's 1) is a counting-method artifact, not a defect. Recommend landing the work on the controller's commit hook; the only follow-ups are the non-blocking notes above.
