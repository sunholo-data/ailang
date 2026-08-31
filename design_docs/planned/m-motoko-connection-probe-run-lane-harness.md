# M-MOTOKO-CONNECTION-PROBE-RUN-LANE-HARNESS: behavioral pin for production lane process-group kill

**Status**: Planned
**Target**: v0.34.1
**Priority**: P0 (High — loop-health guardrail; prevents false conclusions about local-model behavior)
**Estimated**: 1 iteration (~4-6 hours implementation + mutation validation)
**Dependencies**: Row 6g process-group kill is already landed; this doc adds the missing behavioral harness for the production half only.

## Problem Statement

Row 6g fixed two timeout wrappers in `tools/eval/motoko_connection_probe.sh` and
`tools/eval/test_motoko_connection_probe.sh`: on timeout, they now attempt to kill the
child process group instead of only the wrapper PID. The self-test wrapper (`run_bounded`)
is pinned by a fixture-cwd + `lsof -c sleep` survivor check, but the production lane runner
(`run_lane`) is not.

The mission row records the load-bearing failure mode: reverting the production
`run_lane` hunk back to single-PID kill left `/bin/bash tools/eval/test_motoko_connection_probe.sh`
green at 40/40. The only current gate that sees the production script is syntax (`bash -n`),
which cannot distinguish a process-group kill from a direct PID kill.

**Current state:**

- `run_lane` owns the production live probe path that runs on the GPU rig.
- A surviving driver descendant on the rig can masquerade as "the local model declined to act",
  which is exactly the conclusion the connection probe exists to prevent.
- The existing self-test already has the right observable for wrapper-grandchild leaks:
  a unique fixture cwd, enumerated with `lsof -a -c sleep -d cwd`.
- That observable is not attached to the production `run_lane` timeout path.

**Impact:**

- Mission/eval operators can receive a green test suite for an instrument whose most important
  timeout behavior has regressed.
- A connection-probe failure can become a model-behavior story rather than an instrument-health
  story.
- The drift is easy to reintroduce because the shell syntax remains valid.

## Goals

**Primary Goal:** Add a hermetic behavioral arm that exercises the production `run_lane` timeout path and fails if the lane cap kills only the wrapper PID or if job-control setup is neutered.

**Success Metrics:**

- A normal probe timeout with a wrapper-grandchild fixture leaves zero `sleep` descendants in the unique fixture cwd.
- The arm proves readiness before timeout: the stub writes a ready file containing wrapper/child PIDs only after it has spawned the child and verified the fixture cwd.
- The arm observes the ready child live before the production timeout fires by reading the ready file and checking the child PID; `fixture_sleep_pids` is reserved for the post-timeout cleanup assertion.
- A group→single-PID-kill mutant in `tools/eval/motoko_connection_probe.sh` makes the self-test fail.
- A `set -m` neutering mutant in the production `run_lane` path makes the self-test fail.
- `make test-launchd-drivers` continues to run the connection-probe self-test and bash syntax checks.
- The new arm remains bounded and sandbox-friendly on its supported platform: no socket bind, no GPU, no Ollama, no real network dependency.
- Unsupported non-Darwin platforms report a structured skip for this arm rather than a false pass; Darwin reports a named hard failure if real `lsof` is missing.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Test `run_lane` through the full production script, not by copying its timeout logic into another helper | A copied helper can pass while production regresses; the bug is explicitly a production-hunk coverage gap | agent | design | med |
| Refactor the existing fixture-cwd + `lsof -c sleep` observable into a parameterized helper | The reviewer correctly rejected adding a second bespoke PID helper; one helper must serve both `run_bounded` and `run_lane` survivor checks | agent | design | low |
| Add a bounded ready-file handshake before waiting for timeout | Without readiness, a "zero survivor" result can mean the child never spawned or never reached the fixture cwd | agent | design | med |
| Resolve real `lsof` outside the stubbed PATH for fixture-cwd checks | The hermetic live path stubs `lsof`; the survivor query must observe the host process table, not the fake TCP fixture | agent | design | med |
| Treat the probe's timeout exit as expected, then assert cleanup behavior out-of-band | The production path correctly exits non-zero on timeout; the acceptance condition is descendant cleanup, not rc=0 | agent | design | low |
| Keep the arm hermetic: stub `ailang`, `dig`, platform, and lane peers; do not bind sockets | The launchd-driver suite must be satisfiable in constrained CI/sandbox environments | agent | design | med |
| Mutation acceptance is mandatory for the two known drift shapes | The row exists because a green suite had zero killers for a reverted production hunk | human/agent | design | med |

### Design Freeze

Before implementation begins, these decisions are fixed:

- [x] The new behavioral arm must exercise `tools/eval/motoko_connection_probe.sh`'s production `run_lane` path.
- [x] The survivor observable is unique fixture cwd + `lsof -a -c sleep -d cwd`, exposed through one parameterized helper reused by the existing orphan arm and the new `run_lane` arm.
- [x] The run-lane arm must use a bounded ready-file handshake before interpreting timeout or survivor results.
- [x] Fixture-cwd survivor queries must bypass the PATH `lsof` stub via an explicitly resolved real `lsof`.
- [x] The arm must be hermetic and must not require the GPU rig, Ollama, loopback socket binds, or real OpenRouter access.
- [x] The acceptance drill includes both production mutants: group→single-PID kill and neutered `set -m`.
- [x] The CI-facing support boundary is macOS bash 3.2; unsupported non-Darwin platforms must structured-skip the run-lane fixture arm, while Darwin/no-real-`lsof` is a named hard failure.

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The production timeout path is in `run_lane` and currently uses job control plus negative-PID TERM/KILL | Read `tools/eval/motoko_connection_probe.sh`; `rg -n "run_lane\\(\\)|set -m|kill -TERM|kill -9" tools/eval/motoko_connection_probe.sh` | Confirmed at `run_lane` lines 227-263; `set -m` at line 232, negative-PID TERM at 249, negative-PID KILL at 258 |
| V2 | The self-test already has the fixture-cwd survivor pattern, but it is attached to `run_bounded`, not the production `run_lane` timeout | Read `tools/eval/test_motoko_connection_probe.sh`; `rg -n "run_bounded\\(\\)|lsof -a -c sleep -d cwd" tools/eval/test_motoko_connection_probe.sh` | Confirmed: `run_bounded` begins at line 28; fixture-cwd `lsof -a -c sleep -d cwd` helper is at line 396 |
| V3 | The connection-probe self-test is part of `make test-launchd-drivers`, along with bash syntax checks for both probe files | Read `make/test.mk` lines 39-50 | Confirmed: target invokes `/bin/bash tools/eval/test_motoko_connection_probe.sh`, then `bash -n` for both probe scripts |
| V4 | The current probe has 24 refusal branches covered by the self-test's drift counter | `grep -c 'instrument_failure "' tools/eval/motoko_connection_probe.sh` → 19; `grep -cE '\\|\\| usage$' tools/eval/motoko_connection_probe.sh` → 5 | Confirmed: 19 + 5 = 24, matching `expected_refusal_branches=24` in the test |
| V5 | The live syntax baseline is green before this design | `/bin/bash -n tools/eval/motoko_connection_probe.sh && /bin/bash -n tools/eval/test_motoko_connection_probe.sh` | Confirmed: rc=0 |
| V6 | The row-6i premise is a recorded mission finding, not inferred from this design | Read `design_docs/motoko-mission.md` row 6i and `design_docs/motoko-mission-log.md` iteration 24 | Confirmed: mission records production hunk revert leaving the suite green 40/40 and explicitly files row 6i |
| V7 | The live-form contract and prerequisite gates are exact and can be hermetically satisfied with stubs | Read `tools/eval/motoko_connection_probe.sh` lines 119-134, 173-176, 227-278, and 281-310 | Confirmed: CLI is `TREATMENT_LANE CONTROL_LANE ARTIFACT [BENCHMARK]`; env knobs are `AILANG_BIN`, `PROBE_TIMEOUT_SECS`, `PROBE_MAX_TREE_NODES`, and test-only `PROBE_TEST_PID_SCOPE`/`PROBE_TEST_DESCENDANT_FAILURE`; prerequisite commands are `uname`, `dig`, `lsof`, `pgrep`, `jq`, and `$AILANG_BIN`; run order is treatment first (line 284), then control (line 288), then JSON write/assertions (lines 295-310) |
| V8 | A traced hermetic temp copy enters the production `run_lane` timeout path without network/socket/GPU/Ollama | Instrumented a temp copy at `/var/folders/kr/h0wr2sj94vd6ljtmsxv8jkt00000gn/T/run-lane-trace.aj_tn3b9/probe.sh`; ran with PATH containing stubs for `uname`, `dig`, `pgrep`, `lsof`, and `ailang-stub`, `AILANG_BIN=ailang-stub`, `PROBE_TIMEOUT_SECS=2`, `PROBE_TEST_DRIVER_SLEEP=6`, benchmark `numeric_modulo` | Confirmed: rc=1; stderr markers: `TRACE run_lane_entry lane=treatment`, three `TRACE run_lane_before_sample`/`after_sample` pairs, `TRACE run_lane_timeout lane=treatment group_safe=1 pid=17879`, `TRACE run_lane_after_wait lane=treatment`, `INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline`; marker file: `uname -sm`, two `dig ... openrouter.ai`, `ailang lane=treatment args=--models treatment --benchmarks numeric_modulo --trials 1 --dry-run=false`, three `pgrep -P 17879`, three `lsof -nP -iTCP -sTCP:ESTABLISHED -a -p 17879`; stdout empty; final artifact absent because timeout aborts before JSON write; retained diagnostics: `artifact.json.treatment.driver.log` (27 bytes) and `artifact.json.treatment.lsof` (396 bytes, 6 lines). No minimal production testability hook is required; temp-copy trace instrumentation is enough. |
| V9 | The CI target for this shell harness is macOS bash 3.2, not a generic Ubuntu shell | Read `.github/workflows/ci.yml` lines 535-555 and mission evidence in `design_docs/motoko-mission.md` lines 966-988 plus `design_docs/motoko-mission-log.md` lines 2662-2676 | Confirmed: CI job `launchd drivers (bash 3.2)` runs on `macos-latest`, asserts `/bin/bash` major version 3, has `timeout-minutes: 15`, and runs `make test-launchd-drivers`. Existing CI evidence is platform-labeled: row 6j enumerated 58 settled `dev` runs of that macOS job (55 success / 2 failure / 1 cancelled), and the mission log states this is the CI leg for launchd-driver bash 3.2 behavior. |
| V10 | A real `lsof` is available outside the hermetic PATH on this authoring platform | Ran `command -v lsof`, `command -p -v lsof`, and `/usr/sbin/lsof -v` | Confirmed on this machine: both `command -v` and `command -p -v` resolve `/usr/sbin/lsof`; `/usr/sbin/lsof -v` reports lsof 4.91. The sprint must resolve this before replacing PATH with the stub directory, store it as an absolute `REAL_LSOF`, hard-fail on Darwin if no executable real `lsof` is available, and structured-skip only on unsupported non-Darwin platforms. |

This design makes no AILANG language-surface claims, so no `ailang check` claim fixture is required.

## Quorum Verification Log

| Reviewer fix | Applied where | Status |
|--------------|---------------|--------|
| Bounded readiness before timeout/survivor assertions | Goals, Design Freeze, Bounded Readiness Protocol, Success Criteria, Testing Strategy | Applied: ready file contains wrapper/child PIDs, is atomically written only after cwd verification and child spawn, and is observed before timeout |
| Fixture survivor helper must not call the PATH `lsof` stub | High-Impact Decisions, V10, Parameterized Real-`lsof` Helper, Conflict Surface | Applied: helper resolves absolute `REAL_LSOF` before PATH is replaced and calls `"$REAL_LSOF"` for cwd checks |
| Platform-labeled CI evidence or explicit unsupported-non-Darwin skip | V9, External dependency/stub contract, Success Criteria, Testing Strategy | Applied: CI target is macOS `launchd drivers (bash 3.2)`; unsupported non-Darwin platforms structured-skip only this arm; Darwin/no-real-`lsof` hard-fails |
| Verification rows for baseline and both actual mutants with readiness/timeout/survivors/suite rc | Required Implementation Verification Rows | Applied as a mandatory sprint completion log; these rows cannot be filled pre-implementation because the ready-file harness does not yet exist in the repo |
| Reuse existing/shared process-group bounded runner for the emergency outer cap | Overview, Bounded Readiness Protocol, Conflict Surface, Success Criteria, Testing Strategy, Required Implementation Verification Rows | Applied: the full probe runs inside `run_bounded` or one shared parameterized equivalent with cap `PROBE_TIMEOUT_SECS + 10`; outer-cap firing is a distinct failure marker and must not fire for baseline or expected mutants |
| Darwin `REAL_LSOF` absence must hard-fail, not skip | Bounded Readiness Protocol, Success Criteria, Testing Strategy, Required Implementation Verification Rows | Applied: missing/non-executable real `lsof` is `not ok` on Darwin; only non-Darwin unsupported platforms may emit structured `UNINFORMATIVE` skip |

## Solution Design

### Overview

Extend `tools/eval/test_motoko_connection_probe.sh` with a new arm that drives the full
`tools/eval/motoko_connection_probe.sh` live form through its existing hermetic PATH/toolchain.
The traced baseline in V8 proves this route reaches production `run_lane`, its sampler, and
its timeout branch before any real network/socket/GPU/Ollama dependency is touched. The
`AILANG_BIN` stub will be taught a test-only mode that creates a wrapper-grandchild shape:
it changes into a unique fixture directory, starts a long-lived `sleep` child, then waits.

The full probe runs with a tiny `PROBE_TIMEOUT_SECS` so `run_lane` reaches its timeout branch.
The probe is executed inside the existing `run_bounded` process-group runner, or a single shared
parameterized equivalent if the sprint refactors that helper, with an emergency cap longer than the
production deadline (`PROBE_TIMEOUT_SECS + 10`). That outer runner is only a harness leak/hang guard:
it sends no signal before the production timeout window has already expired, and the arm must record
a distinct outer-cap marker/result and require that it did not fire. Inside that bounded wrapper, a
small harness script launches the production probe asynchronously so the test can wait for the ready
file before accepting any timeout result.

The probe is expected to exit non-zero with the normal production sampling-deadline failure. After
`run_lane`'s own timeout/cleanup has completed, the test enumerates real `sleep` processes whose cwd
equals the fixture directory through a single parameterized helper reused by the existing orphan arm
and the new `run_lane` arm:

```bash
fixture_sleep_pids "$fixture_dir"
```

The assertion is simple: the timeout may fail the probe, but it must not leave a sleep process
behind in that fixture cwd. A single-PID kill leaves exactly the survivor this check can see.
The normal process-group kill leaves none.

### Bounded Readiness Protocol

The new arm must make "the child existed in the fixture cwd before timeout" an observed fact:

1. Resolve and validate the real `lsof` path before constructing the hermetic `PATH`:

   ```bash
   REAL_LSOF=$(command -p -v lsof 2>/dev/null || true)
   if [[ -z "$REAL_LSOF" || ! -x "$REAL_LSOF" ]]; then
     if [[ "$(uname -s)" == "Darwin" ]]; then
       echo "not ok - run_lane fixture arm requires real lsof on Darwin CI target"
       exit 1
     else
       echo "UNINFORMATIVE: run_lane fixture arm requires real lsof for cwd survivor checks"
       skip_run_lane_fixture=1
     fi
   fi
   ```

   On macOS, V10 confirms this resolves `/usr/sbin/lsof`. If no real executable is available on
   Darwin, the suite hard-fails with the named `not ok` above. Only unsupported non-Darwin platforms
   may structured-skip this arm. In no case may the helper fall back to the PATH `lsof` stub.

2. Create a unique `run_lane_fixture_dir`, a `ready_file`, and a same-directory `ready_tmp`.
3. Extend `ailang-stub` so that when `PROBE_TEST_RUN_LANE_GRANDCHILD_CWD` is set it:
   - verifies `cd "$PROBE_TEST_RUN_LANE_GRANDCHILD_CWD"` succeeded;
   - verifies `pwd -P` equals the expected fixture cwd;
   - spawns a long `sleep "$PROBE_TEST_RUN_LANE_GRANDCHILD_SECS"` child;
   - writes `wrapper_pid=$$`, `child_pid=$child_pid`, and `cwd=$(pwd -P)` to `"$ready_tmp"`;
   - atomically publishes readiness with `mv "$ready_tmp" "$ready_file"`;
   - waits for the child.
4. Write a small run-lane fixture harness that launches the full production probe asynchronously:

   ```bash
   env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=2 \
     PROBE_TEST_RUN_LANE_GRANDCHILD_CWD="$run_lane_fixture_dir" \
     PROBE_TEST_RUN_LANE_GRANDCHILD_READY="$ready_file" \
     PROBE_TEST_RUN_LANE_GRANDCHILD_READY_TMP="$ready_tmp" \
     /bin/bash "$probe" treatment control "$tmp_dir/run-lane.json" numeric_modulo \
       >"$tmp_dir/run-lane.stdout" 2>"$tmp_dir/run-lane.stderr" &
   probe_pid=$!
   ```

5. Wait for readiness with a separate bound (`RUN_LANE_READY_CAP_SECS`, e.g. 5s). On expiry:
   write a named readiness-failure marker and exit non-zero rather than continuing to a vacuous
   survivor count. The surrounding process-group runner owns emergency termination if the harness
   itself does not return.
6. After reading the ready file, assert all readiness fields:
   - wrapper PID and child PID are non-empty integers;
   - cwd equals the fixture cwd;
   - `kill -0 "$child_pid"` succeeds before the probe timeout fires. Do not use `fixture_sleep_pids`
     for this pre-timeout readiness assertion; it is the post-production-cleanup survivor oracle.
7. Run that harness through `run_bounded` (or one shared parameterized process-group runner reused by
   `run_bounded`) with `RUN_LANE_OUTER_CAP_SECS=$(( PROBE_TIMEOUT_SECS + 10 ))`. The runner must TERM,
   then KILL, the harness-owned process group and perform bounded reap if the emergency cap fires.
   Record the outer result separately (for example `outer_cap_rc=199` or `outer_cap_fired=1`).
8. Require the outer cap did not fire for baseline, single-PID-kill mutant, or `set -m` mutant. If it
   fires, fail by name as a harness leak/hang, not as a successful production timeout assertion.
9. Assert the expected production timeout shape:
   - probe rc is non-zero;
   - stderr contains `INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline`;
   - stderr or marker file proves production `run_lane` was entered.
10. Assert post-production-timeout survivors with `fixture_sleep_pids`:
   - baseline: `fixture_sleep_pids "$run_lane_fixture_dir"` returns zero;
   - single-PID or `set -m` mutants: survivor count is non-zero and the suite rc is non-zero.
11. Cleanup any surviving fixture sleeps after recording the assertion result. This cleanup is
    separate from the production cleanup being tested and must not run before the survivor count is
    recorded.

### Parameterized Real-`lsof` Helper

Refactor the existing single-fixture helper into one helper that always calls the resolved real
`lsof`, never the PATH stub:

```bash
fixture_sleep_pids() {
  local fixture_dir=$1
  "$REAL_LSOF" -a -c sleep -d cwd 2>/dev/null |
    awk -v fixture_dir="$fixture_dir" 'NR > 1 && $NF == fixture_dir { print $2 }'
}
```

The existing orphan arm uses `fixture_sleep_pids "$orphan_fixture_dir"`. The new `run_lane` arm
uses `fixture_sleep_pids "$run_lane_fixture_dir"`. This preserves one observable and removes the
reviewed duplication risk.

### Architecture

**Components:**

1. **Stub-driver grandchild mode**: A new test-only branch in the existing `ailang-stub` fixture.
   When `PROBE_TEST_RUN_LANE_GRANDCHILD_CWD` is set, it enters that directory, spawns a long
   `sleep`, writes the ready file with wrapper/child PIDs after cwd verification, and waits.
2. **Parameterized survivor helper**: Replace `orphan_fixture_pids` with
   `fixture_sleep_pids <fixture_dir>`, backed by absolute `REAL_LSOF`, and update the existing orphan
   arm to call it with `"$orphan_fixture_dir"`. The new `run_lane` arm calls the same helper with its own fixture cwd.
3. **Expected-timeout arm**: A self-test arm that invokes the full probe through `run_bounded` (or a
   single shared parameterized process-group runner) as an emergency cap, while the bounded harness
   launches the probe asynchronously to wait for readiness. It records a distinct outer-cap result,
   requires the outer cap did not fire, requires the production sampling-deadline marker, checks
   post-production-timeout survivors, cleans up any surviving fixture sleeps after recording the
   assertion, and emits one `ok`.
4. **External-dependency stub contract**: Keep all dependencies that production live form invokes
   inside the hermetic PATH and prove them with marker files.
5. **Mutation drill notes**: Documented commands or comments beside the arm naming the two required
   mutants and the expected red condition.

### External dependency/stub contract

The live form must not reach the machine's real network, GPU, Ollama, or socket bind surface during
this self-test. Every external command on the production route is either already stubbed or must be
stubbed by this sprint:

| Production call site | Contract | Existing/proposed stub |
|----------------------|----------|------------------------|
| `uname -sm` at `tools/eval/motoko_connection_probe.sh:129` | Satisfies the Darwin/arm64 live-form gate without depending on host OS | Existing `uname` stub in `test_motoko_connection_probe.sh:184-187` |
| `command -v dig/lsof/pgrep/jq/$AILANG_BIN` at lines 130-134 | Verifies dependencies are visible on the hermetic PATH | Existing live-bin setup at test lines 175-230; keep `jq` symlink and stubs executable |
| `dig ... A/AAAA openrouter.ai` at lines 173-176 | Provides deterministic OR_IPS without DNS/network | Existing `dig` stub at test lines 188-191; add marker output for the new arm if needed |
| `$ailang_bin eval-suite --agent --models "$lane" --benchmarks "$benchmark" --trials 1 --dry-run=false` at lines 232-234 | Drives the production CLI shape while avoiding GPU/Ollama/model calls | Existing `ailang-stub` at test lines 217-229; extend it with grandchild fixture mode and marker output |
| `pgrep -P "$current"` inside `descendant_pids` at line 200 | Keeps process-tree discovery deterministic | Existing `pgrep` stub at test lines 192-199; marker-file output may be added because stderr is redirected in production |
| `lsof -nP -iTCP -sTCP:ESTABLISHED -a -p "$pids"` at line 221 | Provides deterministic lane peers without real sockets | Existing `lsof` stub at test lines 200-216; marker-file output may be added because stderr is redirected in production |
| `cp`/`rm` in cleanup at lines 148-170 | Retains diagnostics and removes only the probe temp dir | Existing symlinks in live-bin setup include `cp` and `rm`; no custom stub required |
| `jq` at lines 277-278 and 295-307 | Formats peers/artifact if the timeout path does not abort first | Existing symlink to real `jq`; safe because it processes local fixture files only |
| `fixture_sleep_pids <dir>` in the test harness | Observes real fixture-cwd sleeps after production timeout/cleanup | Must bypass `PATH` and call absolute `REAL_LSOF` resolved before the live-bin stub directory is activated |

### Conflict Surface

This design does not touch parser/typechecker/codegen surfaces, but it does touch a shell-harness
surface with real conflict risk: the existing probe self-test, its hermetic PATH, and its process
cleanup fixtures.

### Shell positions touched

- `tools/eval/test_motoko_connection_probe.sh` live-bin fixture setup around lines 175-230.
- The existing wrapper-grandchild arm around lines 388-423.
- The production live-form invocation shape currently expressed by `run_live` at lines 232-235.
- The refusal/drift counter at lines 473-492 if a new refusal branch is added.

### What else lives here

| Position | Existing valid form | Shape |
|----------|---------------------|-------|
| Live-bin setup | Stub external commands | `uname`, `dig`, `pgrep`, `lsof`, `ailang-stub`, plus symlinks for shell/coreutils and `jq` |
| Wrapper-grandchild check | Existing `run_bounded` orphan arm | `orphan_fixture_dir` → spawn nested `sleep` → enumerate cwd → cleanup |
| Live probe success/refusal arms | Existing `run_live` calls | Hermetic PATH + `AILANG_BIN=ailang-stub` + treatment/control lanes + artifact path |
| Refusal-branch drift gate | Counts production refusal branches | `grep -c 'instrument_failure "'` + `grep -cE '\|\| usage$'` must equal `expected_refusal_branches` |

### Disambiguation strategy

- The existing orphan arm and the new `run_lane` arm must differ only by fixture cwd and launcher;
  they share `fixture_sleep_pids <dir>` so future edits cannot update one survivor query and forget the other.
- `fixture_sleep_pids` must call absolute `REAL_LSOF`; the production live-path TCP sampler must keep using
  the PATH `lsof` stub. This deliberately separates "fake TCP peers" from "real cwd survivor process table."
- Reusing `run_bounded` or a shared parameterized process-group runner for the outer cap does not mask
  production `run_lane` behavior: the outer cap is set to `PROBE_TIMEOUT_SECS + 10`, performs no TERM/KILL
  before that point, and acceptance requires the earlier production sampling-deadline marker. If the outer
  cap fires, the arm reports a named harness leak/hang failure rather than passing.
- The new arm's label and assertions must name production `run_lane`, not the generic arm cap, so a failure
  routes to the correct hunk.
- External command markers should be written to a marker file, not stderr, for `pgrep`/`lsof`, because the
  production script redirects their stderr to `/dev/null`.
- The arm accepts a non-zero probe rc only when stderr contains the sampling-deadline failure for the target
  lane. Any other non-zero rc remains a test failure.

### Behaviors that MUST still work

- Classification fixtures and IPv6 normalization at test lines 127-151.
- Treatment/control assertion arms at test lines 153-173.
- Existing hermetic live success/refusal arms and diagnostic retention at test lines 232-333.
- Existing `run_bounded` arm-cap and orphan-grandchild arms at test lines 366-423, after refactoring to the
  parameterized survivor helper.
- Refusal-branch drift gate at test lines 473-492.

### What deliberately changes

- `orphan_fixture_pids` stops being a single-fixture helper and becomes a parameterized helper. The observable
  (`lsof -a -c sleep -d cwd`) is unchanged.
- The helper now calls an absolute real `lsof`, so the live-bin PATH `lsof` stub is not used for cwd survivor checks.
- The `run_lane` fixture arm uses the existing/shared process-group runner only as an emergency outer cap;
  it does not reinterpret an outer-cap timeout as a production timeout.
- The self-test suite gains one additional `ok` arm and may gain marker-file plumbing in existing stubs.
- Unsupported non-Darwin environments may structured-skip this arm; Darwin environments hard-fail if real
  `lsof` is missing or non-executable. No skip reports a green behavioral verdict for `run_lane`.
- No production connection-routing behavior deliberately changes.

### Implementation Plan

**Phase 1: Add the production-path fixture arm** (~2 hours)

- [ ] Extend the existing `ailang-stub` fixture with a `PROBE_TEST_RUN_LANE_GRANDCHILD_CWD` mode.
- [ ] Resolve `REAL_LSOF` before constructing/activating `live_bin`; if unavailable, hard-fail on Darwin and structured-skip only on unsupported non-Darwin platforms.
- [ ] Refactor `orphan_fixture_pids` into `fixture_sleep_pids <fixture_dir>` and update the existing orphan arm to use it.
- [ ] Add cleanup helpers that call the parameterized fixture helper for both the old orphan fixture and the new `run_lane` fixture.
- [ ] Add a bounded self-test arm that runs a run-lane fixture harness through `run_bounded` (or a shared parameterized equivalent) with `PROBE_TIMEOUT_SECS=2` and outer cap `PROBE_TIMEOUT_SECS + 10`.
- [ ] Inside the harness, launch the full probe asynchronously, wait for the ready file under its own cap, assert wrapper/child PID fields and child liveness before timeout, then wait for the production probe result.
- [ ] Record the outer-cap result separately and require it did not fire.
- [ ] Assert the probe reached the production sampling-deadline path and that post-production-timeout fixture survivor count is zero.

**Phase 2: Mutation acceptance** (~1-2 hours)

- [ ] Mutate only the production probe copy/path under test from `kill -TERM "-$pid"` to `kill "$pid"` and from `kill -9 "-$pid"` to `kill -9 "$pid"`; verify the new arm fails with a survivor count.
- [ ] Mutate only the production `run_lane` job-control setup by neutering `set -m`; verify the new arm fails via degraded/survivor behavior.
- [ ] Restore the unmutated tree without destructive git operations and re-run the baseline suite.
- [ ] Fill the Required Implementation Verification Rows below with actual readiness/timeout/survivor/suite-rc evidence.

**Phase 3: Wire and polish** (~1-2 hours)

- [ ] Keep `expected_refusal_branches` accurate if the implementation adds or removes refusal branches.
- [ ] Run `/bin/bash tools/eval/test_motoko_connection_probe.sh`.
- [ ] Run `make test-launchd-drivers` when the environment supports the launchd-driver suite; otherwise run the probe self-test plus `bash -n` and state the launchd limitation.
- [ ] Update comments so the arm names the production `run_lane` hunk it pins.

### Files to Modify/Create

**New files:**

- None.

**Modified files:**

- `tools/eval/test_motoko_connection_probe.sh` (+60-100 LOC) — add the `run_lane` wrapper-grandchild fixture arm and mutation instructions.
- `tools/eval/motoko_connection_probe.sh` (expected +0 LOC) — no production behavior change is intended. V8 proves a temp-copy instrumentation path can validate `run_lane` entry/timeout without a production testability hook; any production change must be separately justified in the sprint.

## Examples

### Example 1: existing self-test gap

**Before:**

```text
Mutant: revert production run_lane group-kill hunk
Command: /bin/bash tools/eval/test_motoko_connection_probe.sh
Observed by mission row 6i: rc=0, 40 ok, 0 not ok
```

**After:**

```text
Mutant: same production run_lane single-PID kill
Command: /bin/bash tools/eval/test_motoko_connection_probe.sh
Expected: non-zero; new arm reports a run_lane fixture survivor
```

### Example 2: normal timeout semantics

The probe is still allowed to fail the lane on timeout:

```text
INSTRUMENT FAILURE: lane treatment exceeded 1s sampling deadline
```

The new behavioral assertion is not that the timeout succeeds. It is that, after this failure,
the process tree owned by the timed-out lane is gone:

```text
run_lane_fixture_pids | wc -l
0
```

## Success Criteria

- [ ] The new self-test arm exercises the full `motoko_connection_probe.sh treatment control artifact` form, not a copied helper.
- [ ] The arm is bounded at three levels: ready-file wait, `run_bounded`/shared-runner emergency outer cap, and final survivor cleanup.
- [ ] The emergency outer cap is longer than the production deadline (`PROBE_TIMEOUT_SECS + 10`), records a distinct marker/result, and is required not to fire for baseline or expected mutants.
- [ ] The ready file is atomically published and contains wrapper PID, child PID, and verified cwd.
- [ ] Before timeout, the child PID from the ready file is observed live by PID check; `fixture_sleep_pids` is used only after production timeout/cleanup to assert survivors.
- [ ] `fixture_sleep_pids` calls absolute `REAL_LSOF`, not the PATH `lsof` stub.
- [ ] On Darwin, missing or non-executable real `lsof` hard-fails with `not ok - run_lane fixture arm requires real lsof on Darwin CI target`.
- [ ] On unsupported non-Darwin platforms, only the run-lane fixture arm may emit a structured `UNINFORMATIVE` skip; it must not print `ok` for this behavioral verdict.
- [ ] Baseline `/bin/bash tools/eval/test_motoko_connection_probe.sh` passes.
- [ ] Baseline `/bin/bash -n tools/eval/motoko_connection_probe.sh` and `/bin/bash -n tools/eval/test_motoko_connection_probe.sh` pass.
- [ ] Mutation A, production group→single-PID kill, fails the suite.
- [ ] Mutation B, production `set -m` neutered, fails the suite.
- [ ] The new arm does not require socket binding, Ollama, GPU access, or real OpenRouter access.
- [ ] `make test-launchd-drivers` remains the CI-facing target that runs this coverage.

## Required Implementation Verification Rows

The sprint is not complete until these rows are filled with actual run evidence from the final
implementation. "Actual mutant" here means the production probe under test was mutated in a temp
copy or through `PROBE_UNDER_TEST`, the mutant was asserted present, `bash -n` stayed green, and the
suite outcome below came from running the self-test against that mutant. Do not treat expected values
as evidence.

| Row | Probe under test | Ready observed | Timeout observed | Pre-timeout child evidence | Outer cap | Post-timeout survivors | Expected suite rc | Actual evidence required |
|-----|------------------|----------------|------------------|----------------------------|-----------|------------------------|-------------------|--------------------------|
| IV1 baseline | unmutated `tools/eval/motoko_connection_probe.sh` | yes: ready file with wrapper PID, child PID, verified cwd | yes: `INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline` in arm stderr | yes: ready child PID passes `kill -0` before timeout | did not fire (`outer_cap_rc != 199` / marker absent) | 0 | 0 | Record ready fields, timeout line, outer-cap result, survivor count, and `/bin/bash tools/eval/test_motoko_connection_probe.sh` rc |
| IV2 mutant A | production `run_lane` group kill changed to single-PID kill (`kill -TERM "$pid"` / `kill -9 "$pid"`) | yes | yes | yes: ready child PID passes `kill -0` before timeout | did not fire (`outer_cap_rc != 199` / marker absent) | >0 before test cleanup | non-zero | Record mutation assertion, ready fields, timeout line, outer-cap result, survivor count, failing arm text, and suite rc |
| IV3 mutant B | production `run_lane` `set -m` neutered so the child is not a distinct job-group leader | yes | yes, or named degraded process-group refusal before timeout | yes: ready child PID passes `kill -0` before timeout | did not fire (`outer_cap_rc != 199` / marker absent) | >0 before test cleanup or named degraded branch plus non-zero suite rc | non-zero | Record mutation assertion, ready fields, timeout/degraded line, outer-cap result, survivor count if applicable, failing arm text, and suite rc |

If an environment cannot run IV1 because it is not the macOS bash 3.2 CI target, the implementation
report must record the structured non-Darwin skip and then provide IV1-IV3 from the macOS CI leg or
an equivalent macOS runner before the design is moved to implemented. A Darwin environment that lacks
a real executable `lsof` is a named hard failure, not a structured skip.

## Testing Strategy

**Unit/self-test level:**

- Add one TAP-style arm in `tools/eval/test_motoko_connection_probe.sh`.
- Verify expected timeout stderr so the arm cannot pass without reaching the production timeout path.
- Verify zero fixture-cwd survivors through the shared `fixture_sleep_pids <dir>` helper.
- Verify bounded readiness before timeout: ready file exists within the ready cap, contains valid wrapper/child PIDs and cwd, and the child PID is live by `kill -0`.
- Verify the run-lane harness is executed under `run_bounded` or a shared parameterized process-group runner with outer cap `PROBE_TIMEOUT_SECS + 10`; record the outer-cap marker/result and fail if it fires.
- Verify `fixture_sleep_pids` uses absolute `REAL_LSOF`; force a hostile PATH `lsof` stub and confirm cwd survivor checks still observe the real child.
- Verify Darwin missing/non-executable `REAL_LSOF` is a named `not ok`, while unsupported non-Darwin reports `UNINFORMATIVE` without printing `ok` for the behavioral verdict.
- Cross-check marker-file output for `uname`, `dig`, `ailang-stub`, `pgrep`, and `lsof` when debugging the arm; `pgrep`/`lsof` cannot rely on stderr markers because production redirects stderr.

**Mutation tests:**

- Production group→single-PID kill: the new arm must fail with survivor count > 0.
- Production `set -m` neuter: the new arm must fail because the guard degrades away from process-group ownership and leaves the observable survivor or emits the named degraded branch.
- Both mutation tests must fill IV2/IV3 with actual readiness, timeout/degraded, survivor, and suite-rc evidence.

**Integration tests:**

- `/bin/bash tools/eval/test_motoko_connection_probe.sh`
- `/bin/bash -n tools/eval/motoko_connection_probe.sh`
- `/bin/bash -n tools/eval/test_motoko_connection_probe.sh`
- `make test-launchd-drivers` on the macOS bash 3.2 CI leg. On other platforms, the run-lane fixture arm must structured-skip rather than claim coverage.

**Manual review:**

- Confirm the new arm's label names `run_lane`, not generic "arm cap", so future failures point to the production hunk.
- Confirm the fixture cleanup leaves zero `sleep` processes in the fixture cwd after both success and failure paths.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact helper/function names in the shell test — agent may choose.
- Whether to create a copied mutant file under a temp directory or use `PROBE_UNDER_TEST` against a generated temp copy during mutation validation — agent may choose.
- Whether the neutered-`set -m` acceptance asserts survivor count, degraded stderr, or both — agent may choose, but at least one observable must kill the mutant.
- Whether `make test-launchd-drivers` is run locally or reported as environment-limited after the probe self-test and syntax gates pass — agent may choose based on the active environment.
- Exact marker-file schema for dependency tracing — agent may choose, as long as the output names each external dependency invoked by the new arm.

## Non-Goals

**Not attempted in this feature:**

- Re-diagnosing V38's rig-only mechanism — this row is a coverage pin for the production kill hunk, not a claim about why the earlier live probe slowed to ~8 minutes.
- Changing connection routing classification, OpenRouter IP resolution, or treatment/control assertions.
- Changing `run_lane` production behavior when the current behavior passes the new harness.
- Adding Go-side process-group helpers; this script is Bash and already has the guarded job-control mechanism from row 6g.
- Adding socket-based live sampling coverage; the row explicitly prefers fixture-cwd + `lsof -c sleep` because it does not need loopback binds.

## Timeline

**Iteration work** (~4-6 hours):

- Phase 1: Add fixture mode and self-test arm (~2h)
- Phase 2: Run and record two mutation checks (~1-2h)
- Phase 3: Baseline validation, cleanup, and comments (~1-2h)

**Total: ~1 mission iteration**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `lsof -c sleep -d cwd` behaves differently across macOS/Linux runners | Medium | The existing self-test already uses this observable; keep the new arm in the same bash self-test context and cleanly report environment limitations |
| The arm passes without reaching `run_lane`'s timeout branch | High | Require the expected sampling-deadline stderr before checking survivor count |
| Survivor cleanup fails and contaminates later arms | High | Register a cleanup helper before invoking the fixture and run it on both pass and fail paths |
| Mutations are applied to the wrong call site | Medium | Mutation instructions must name production `tools/eval/motoko_connection_probe.sh:run_lane`, not `run_bounded` |
| Arm runtime becomes flaky under CI contention | Medium | Use the existing bounded-arm pattern; choose distinctive long sleep duration but tiny probe timeout, and avoid per-second socket polling |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Makes timeout cleanup behavior mechanically reproducible with a hermetic fixture instead of inferred from live rig outcomes |
| A2: Replayability | +1 | Mutation acceptance creates replayable evidence for the two known drift shapes |
| A3: Effect Legibility | 0 | No AILANG effect-system change; shell process effects remain explicit in the test |
| A4: Explicit Authority | +1 | Verifies that process authority is scoped to the child job group before negative-PID kill is trusted |
| A5: Bounded Verification | +1 | Adds a bounded local check for a production timeout branch |
| A6: Safe Concurrency | +1 | Reduces orphaned-process risk on shared CI/rig machines |
| A7: Machines First | +1 | Turns an invisible instrument-health premise into a machine-checkable gate |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | 0 | No metered-model or billing change |
| A10: Composability | +1 | Reuses existing test harness stubs and `make test-launchd-drivers` wiring |
| A11: Structured Failure | +1 | The arm names the failed invariant and reports survivor counts instead of a vague timeout |
| A12: System Boundary | +1 | Keeps live external systems stubbed; the test boundary is explicit |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Related Documents

**Direct mission context:**

- [design_docs/motoko-mission.md](../motoko-mission.md) — row 6i defines the queue item and scope.
- [design_docs/motoko-mission-log.md](../motoko-mission-log.md) — iteration 24 records the production-hunk zero-killer finding and the fixture-cwd lesson.

**Related design docs:**

- [design_docs/planned/m-motoko-fmt-remeasurement-instrument.md](m-motoko-fmt-remeasurement-instrument.md) — broader connection-probe origin; this row is only the harness pin for the production process-group kill.
- [design_docs/implemented/v0_35_0/m-dx-pi-harness.md](../implemented/v0_35_0/m-dx-pi-harness.md) — harness self-improvement doctrine and wrap-don't-reimplement pattern.
- [design_docs/implemented/v0_24_0/m-eval-network-mock-fixture.md](../implemented/v0_24_0/m-eval-network-mock-fixture.md) — precedent for replacing live network uncertainty with deterministic local fixtures.

**Auto-search results reviewed:**

- [design_docs/implemented/v0_18_4/m-motoko-executor-adapter.md](../implemented/v0_18_4/m-motoko-executor-adapter.md) (0.37)
- [design_docs/implemented/v0_26_0/m-motoko-agent-system-prompt.md](../implemented/v0_26_0/m-motoko-agent-system-prompt.md) (0.35)
- [design_docs/implemented/v0_16_3/m-motoko-inline-test-harness.md](../implemented/v0_16_3/m-motoko-inline-test-harness.md) (0.33)
- [design_docs/planned/m-motoko-fmt-remeasurement-instrument-sprint-plan.md](m-motoko-fmt-remeasurement-instrument-sprint-plan.md) (0.38)
- [design_docs/planned/v0_29_0/m-motoko-ext-per-task-sprint-plan.md](v0_29_0/m-motoko-ext-per-task-sprint-plan.md) (0.36)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles.
- [tools/eval/motoko_connection_probe.sh](../../tools/eval/motoko_connection_probe.sh) - Production connection probe.
- [tools/eval/test_motoko_connection_probe.sh](../../tools/eval/test_motoko_connection_probe.sh) - Existing self-test harness to extend.
- [make/test.mk](../../make/test.mk) - `test-launchd-drivers` wiring.

## Future Work

- If the mutation harness finds another unpinned production branch, file a follow-up row rather
  than expanding this sprint.
- If fixture-cwd survivor enumeration flakes on a specific CI platform, split that platform's
  behavior into a named row with measured evidence.

---

**Document created**: 2026-08-30
**Last updated**: 2026-08-30
