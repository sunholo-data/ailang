# M-WORKER-PTY-PIDFILE-ACK: complete-PID startup acknowledgement for the worker PTY lifecycle test

**Status**: Planned
**Target**: v0.54.17
**Priority**: P1 (Medium-High — CI gate flake, no product impact)
**Estimated**: 1 day (fix 2h, focused race validation 2h, full matrix + gates 4h; 2x rule applied)
**Dependencies**: None (completes the startup-ack contract that `m-background-worker-lifecycle` already states for its other fixtures)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Test-fixture-only change: no language surface, effect row, capability, or runtime path is
touched. Scores reflect the fixture's role as a verification instrument.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a scheduler-dependent TOCTOU from the fixture; complete-PID visibility becomes an atomic-rename invariant instead of an open/write interleaving |
| A2: Replayability | 0 | No trace/replay surface touched |
| A3: Effect Legibility | 0 | No effect annotations or capability signatures change; `--process-allowlist` route exercised exactly as before |
| A4: Explicit Authority | 0 | No capability changes; the worker remains an explicitly allowedlisted `sys.executable` |
| A5: Bounded Verification | +1 | Startup ack stays inside the existing 5s bounded deadline and now fails with a complete, parseable diagnostic instead of a bare `ValueError` traceback |
| A6: Safe Concurrency | 0 | No production concurrency change |
| A7: Machines First | +1 | Ack failure yields a structured machine-readable message (`'worker never acknowledged startup'` with the deadline and directory), not a Python stack artifact |
| A8: Minimal Syntax | 0 | No language syntax change |
| A9: Cost Visibility | 0 | No resource accounting change; `@limit=3` budget path asserted as before |
| A10: Composability | +1 | Adopts the same atomic-publish idiom already used by `internal/effects/worker_ai_lifecycle_test.go`, unifying the fixture ack protocol family |
| A11: Structured Failure | 0 | Failure modes already structured; preserved verbatim |
| A12: System Boundary | 0 | No boundary crossings added |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the fix *removes* one
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`TestWorkerTerminalPTYLifecycle/eval/sigint` failed in CI run
[39544923639](https://github.com/sunholo-data/ailang/actions/runs/39544923639) (job
119040812981, step "Check coverage gate (M-P2)") at commit `d3d389efeba794013e72f861ad3a5317a2880dab`,
reported by Go source line `internal/runner/worker_terminal_pty_test.go:77` — the
`t.Fatalf("worker PTY: %v\n%s")` that surfaces the Python harness exit. The harness died with
`ValueError: invalid literal for int() with base 10: ''` at the injected startup-ack line
`worker_pid = int(open(pidfile).read())`.

**Current State (mechanism, reproduced against current source `d75ef77f59cf4c3c3151fef4840335d269963b8d`):**

- The fixture worker (`worker_terminal_pty_test.go:41`) publishes its PID with
  `open("worker.pid","w").write(str(os.getpid()))` — the CPython evaluation order is:
  create/truncate (0-byte file exists) → `str(os.getpid())` → buffered write → close/flush →
  content lands. The file is *observable and empty* for the whole open→write window.
- The harness wait (`worker_terminal_pty_test.go:47-49`) polls **existence only**
  (`while not os.path.exists(pidfile) ... time.sleep(.01)`), then does a single unguarded
  `int(open(pidfile).read())`. Any stat landing inside the worker's open→write window reads `''`.
- Reproduction on this machine (exact current-source protocol, no product code): **1 race in 200
  sequential pairs** and **3 races in 200 eight-way-concurrent pairs** — 4/400 total
  `ValueError('')` observations. The window is scheduler-load dependent, which matches a
  once-in-many-runs CI failure on shared runners.
- The failed run skipped 18 downstream steps (16 gates, 2 cleanup); **total coverage for that
  run is UNMEASURED** — this is a fixture readiness race, *not* evidence of a coverage-threshold
  miss and *not* evidence of a product lifecycle regression.

**Scope of the flake:** the test's own origin contract
(`m-background-worker-lifecycle.md`, "Validation and Acceptance Criteria": *"Test helpers must
emit a startup acknowledgement and expose their exact PID/group identity before cancellation;
avoid startup races and broad process-name killing."*) is satisfied by the other fixtures but
violated by this one: the same race class was already fixed for the blocked-AI helper
(`worker_ai_lifecycle_test.go:26-35` writes a `.pending` file and `os.Rename`s it atomically —
"Publish only a complete PID: another process may observe file creation before WriteFile has
written the contents, especially on Linux CI") and recorded in
`m-background-worker-lifecycle-validation.md` ("Release check corrections"). The PTY worker
variant was left with the existence-only wait.

**Impact:** every PTY matrix subtest (11 cases: eval/strict × normal/exit/error/sigint/sigterm/budget,
minus strict/budget) can fail stochastically, and a red step-53 blocks all 16 downstream gates.
CI signal for *actual* terminal-restoration or worker-reaping regressions is diluted.

**Routing note (per task framing):** the target `internal/runner/worker_terminal_pty_test.go` is
outside the fleet mission write allowlist (`tools/launchd/githooks/pre-push`). This is a
product-test route: a normal PR touching only that test file, human-approved, with the full
downstream CI gates. Do **not** send this as a fleet repair. The unrelated govulncheck
`GO-2026-6809` finding (first red run 39530700414, commit `is7ab609a`) is a separate issue and is
explicitly out of scope.

## Goals

**Primary Goal:** Make the PTY worker fixture's startup acknowledgement observable only as a
complete PID — atomically published and content-verified — while preserving every existing
assertion byte-for-byte (exact-PID disappearance via `ps`, exit status 130/143/7/1/0, termios
restoration `before == after` with the PENDIN mask, DECRC/DECUSR escape sequences, budget
`OVER_BUDGET` absence, SIGKILL cleanup fallback).

**Success Metrics:**
- Startup-ack race rate: 4/400 observed (current) → **0** over ≥400 reproduction pairs including
  8-way-concurrent scheduler pressure
- All 11 `TestWorkerTerminalPTYLifecycle` subtests green; `TestTerminalCLIPTYLifecycle` and
  `TestWorkerCLIProcessExitLifecycle` untouched and green
- Diff confined to `internal/runner/worker_terminal_pty_test.go` (Python fixture strings inside
  the Go test); zero product/runtime LOC changed
- All existing deadlines preserved (5s pidfile ack, 12s `until(b'READY')`, 30s per-subtest)
- CI PTY matrix run green at the fix commit with the 18 downstream steps no longer skipped

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| Ack protocol: atomic same-dir tmp+`os.replace` publish (producer) **plus** content-poll reader (defense in depth) vs. two-file causal marker protocol (pidfile then `ready`, reader gates on `ready`) | Defines the fixture's readiness contract; wrong choice reintroduces the flake or adds a second ordering obligation | human (this doc) | design | med |
| Test-only scope: no product/runtime/CI-workflow changes | Keeps the flake fix from sprawling into runtime "improvements" the failure does not justify | human (this doc) | design | high |
| Preserve all existing assertions verbatim (ps absence probe, exit codes, termios equality, escape sequences, cleanup fallback) | These are the regression the test exists to catch; any weakening converts a flake fix into coverage loss | human (this doc) | design | high |
| All deadlines unchanged (5s ack / 12s READY / 30s subtest) | Retuning waits while fixing readiness would mask new latencies | agent | design | low |
| Route as product-test PR (file outside fleet allowlist) | Wrong route sends a test fix through the fleet repair path and mislabels provenance | human | design | med |
| Producer tmp-file mechanism (mkstemp vs. fixed `+ ".pending"` name) | Implementation detail; both satisfy the atomic-rename invariant | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Ack protocol = atomic tmp+rename publish **and** content-poll reader (both sides)
- [x] Scope = `internal/runner/worker_terminal_pty_test.go` only; product code untouched
- [x] All existing assertions and deadlines preserved verbatim
- [x] Routing = product-test PR, not a fleet repair

## Solution Design

### Overview

The worker's PID becomes observable only when it is complete: the worker writes
`str(os.getpid())` to a same-directory temporary file and atomically `os.replace`s it onto
`worker.pid` — POSIX `rename(2)` makes the destination visible as either absent or fully
written, never empty-partial. The harness reader stops polling for *existence* and instead
polls (same 10ms cadence, same 5s bounded deadline) until it can *parse a complete PID*,
retrying through `FileNotFoundError` and `ValueError`; on success it liveness-checks the exact
PID with `os.kill(worker_pid, 0)`, exactly as today. This is the identical idiom already
shipped in `worker_ai_lifecycle_test.go:26-35` (producer) and
`process_managed_lifecycle_test.go:76-88` `managedAwaitFile` (reader), applied to the one
fixture the v0.54.4 sprint left on the racy protocol.

Both sides are required, and each is independently sufficient in practice: the atomic publish
makes `os.path.exists(pidfile) ⇒ complete PID` true (so the *current* reader would already be
safe), and the content-poll reader tolerates *any* producer protocol within the deadline (so
the *current* producer would already be safe). Keeping both means a future edit to either side
cannot silently reintroduce the CI flake.

### Architecture

**Components (all inside the Python harness strings of the one Go test file):**

1. **Producer — worker.py** (`worker_terminal_pty_test.go:41` replacement string): PID bytes to
   `tempfile.mkstemp(dir=".", …)` fd → `os.close` → `os.replace(tmp, "worker.pid")` → `time.sleep(60)`.
   Same-directory mkstemp guarantees the rename crosses no filesystem (no `EXDEV` fallback path).
   No stdout/stderr output is added, so the PTY byte assertions are unaffected.
2. **Reader — wait block** (`worker_terminal_pty_test.go:44-49` replacement string): bounded
   poll that treats empty/unparseable/absent as "not yet acknowledged" and retries at the same
   10ms cadence, 5s deadline; asserts a parsed PID `> 1`; then `os.kill(worker_pid, 0)` exactly
   as before. Placement after `until(b'READY')` is unchanged.
3. **Untouched:** the `ps -o stat=` absence probe (lines 52-56 replacement), the returncode
   table, `termios.tcgetattr` before/after with the PENDIN mask, escape-sequence assertions,
   the `worker_absent` bookkeeping, and the `finally` SIGKILL cleanup (lines 58-67 replacement,
   which already tolerates `ValueError`).

### Implementation Plan

**Phase 1: Fixture fix** (~2 hours)
- [ ] Rewrite the worker.py injection string at line 41 to the atomic-publish form
- [ ] Rewrite the wait-block injection at lines 44-49 to the content-poll form
- [ ] Confirm no other replacement string in the file changes semantics (diff review)

**Phase 2: Reproduction harness validation** (~2 hours)
- [ ] Re-run the local protocol reproduction: current-source protocol ≥1 race/200 (recorded),
      fixed protocol 0 races/≥200 sequential and ≥200 concurrent pairs (extend `/tmp/ptyrepro`
      pair scripts to the exact new strings)
- [ ] Confirm `os.kill(pid, 0)` liveness succeeds on every ack (no dead-PID parse)

**Phase 3: Matrix and gates** (~4 hours)
- [ ] `go test ./internal/runner -run 'TestWorkerTerminalPTYLifecycle' -count=1` (all 11 subtests)
- [ ] `go test ./internal/runner -run 'TestWorkerTerminalPTYLifecycle/eval/sigint' -count=20`
      (the failing subtest, repeated)
- [ ] Full `make test`, `make lint`, `make fmt-check`; `make check-boundaries` not required
      (no import changes) but cheap
- [ ] Fresh CI run green; confirm step-53 coverage gate computes (18 downstream steps unskipped)

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `internal/runner/worker_terminal_pty_test.go` (+~15/−~6 LOC) — the worker.py producer string
  (line 41) and the reader wait-block replacement string (lines 44-49). No Go logic, no
  product package, no build tags change.

## Examples

### Example 1: Producer (worker.py injection, test line 41)

**Before (current source, racy):**
```python
import os,time
open("worker.pid","w").write(str(os.getpid()))   # file exists EMPTY between open() and write()
time.sleep(60)
```

**After:**
```python
import os,time,tempfile
fd,tmp = tempfile.mkstemp(dir=".",prefix=".worker.pid.",suffix=".tmp")   # same dir ⇒ same fs
os.write(fd,str(os.getpid()).encode()); os.close(fd)
os.replace(tmp,"worker.pid")   # atomic: worker.pid is absent or COMPLETE, never empty
time.sleep(60)
```

### Example 2: Reader (wait-block injection, test lines 44-49)

**Before (current source, racy):**
```python
deadline = time.monotonic() + 5
pidfile = os.path.join(work, 'worker.pid')
while not os.path.exists(pidfile) and time.monotonic() < deadline: time.sleep(.01)
assert os.path.exists(pidfile), 'worker never acknowledged startup'
worker_pid = int(open(pidfile).read())          # <-- CI-failing line: int('')
os.kill(worker_pid, 0)
```

**After:**
```python
deadline = time.monotonic() + 5
pidfile = os.path.join(work, 'worker.pid')
worker_pid = None
while worker_pid is None and time.monotonic() < deadline:
 try:
  with open(pidfile) as f: text = f.read().strip()
  if text: worker_pid = int(text)
 except (FileNotFoundError, ValueError): pass    # absent/empty/partial ⇒ not yet acknowledged
 if worker_pid is None: time.sleep(.01)          # same 10ms cadence, same 5s deadline
assert worker_pid and worker_pid > 1, 'worker never acknowledged startup'
os.kill(worker_pid, 0)                           # preserved: liveness of the EXACT PID
```

### Example 3: What must NOT change (regression fixtures the test exists for)

The post-run probe stays byte-identical — complete PID makes it *more* reliable, not different:
```python
probe = subprocess.run(['ps','-o','stat=','-p',str(worker_pid)],capture_output=True,text=True)
assert probe.returncode in (0, 1), ('PID probe failed', probe.returncode, probe.stderr)
assert not probe.stdout.strip(), ('owned worker remains alive or unreaped', worker_pid, probe.stdout)
after = termios.tcgetattr(slave)   # restoration equality, PENDIN-masked, unchanged
```

## Success Criteria

- [ ] Reproduction recorded (current protocol: 4 races/400 pairs observed locally) and fix
      validated: 0 races over ≥400 pairs under sequential **and** 8-way-concurrent load
- [ ] All 11 PTY matrix subtests pass; the formerly failing `eval/sigint` passes `-count=20`
- [ ] `TestTerminalCLIPTYLifecycle`, `TestWorkerCLIProcessExitLifecycle`, and all sibling
      worker fixtures unchanged and passing
- [ ] Existing assertions byte-identical: ps absence, exit status 130/143/7/1/0, termios
      `before == after` (PENDIN masked), `\x1b[?25h` / `\x1b[?1049l`, `OVER_BUDGET` absence,
      finally-SIGKILL fallback
- [ ] All tests passing (`make test`) and `make lint` clean; diff confined to
      `internal/runner/worker_terminal_pty_test.go`
- [ ] Documentation: this doc's Verification Log updated with the fix-commit evidence
- [ ] CI run at fix commit green through step 53 (coverage gate computes; 18 downstream steps
      no longer skipped)

## Testing Strategy

**Unit tests:** none added — the change *is* the test fixture. The fixture's own failure
surface is the regression instrument.

**Integration tests:** the full existing matrix (2 backends × 6 scenarios, minus strict/budget
= 11 subtests), plus the base `TestTerminalCLIPTYLifecycle` and `TestWorkerCLIProcessExitLifecycle`
to prove no cross-fixture drift.

**Regression-surface / race repetitions:**
- Standalone pair-reproduction (Phase 2) proving 0/≥400 on the new protocol under load, mirroring
  the 4/400 measured on the current protocol
- `-count=20` on `eval/sigint` (the observed failing cell)

**Manual testing:** fresh PR CI run (GitHub-hosted runner, where the original flake showed);
confirm the coverage gate step itself completes so coverage is MEASURED next time.

**Manual verification of preserved assertions:** read the final diff against the pre-change
replacement strings and check each assertion listed in Success Criteria is byte-identical.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Producer tmp-file mechanism — `tempfile.mkstemp(dir=".")` vs. fixed `".worker.pid.pending"`
  name (either satisfies the same-dir atomic-replace invariant) — agent may choose
- Exact reader exception breadth — `except (FileNotFoundError, ValueError)` vs. also
  `PermissionError`/`OSError` (both bounded; keep the tuple tight) — agent may choose
- Whether the harness prints a diagnostic line on ack timeout beyond the existing assert tuple —
  agent may choose
- Whether to add an explanatory comment line in the fixture about the atomic-publish invariant —
  agent may choose (the `worker_ai_lifecycle_test.go:26-27` comment is the model)
- Whether the fix PR also bumps `changelogs/v0.32-current.md` or ships under the next patch
  changelog entry — human at review

## Non-Goals

**Not attempted in this feature:**
- Any product/runtime change to `spawnProcess`, worker reaping, terminal restoration, or
  capability gating — the failure is a fixture readiness race, not a product lifecycle
  regression (do not infer one)
- Any coverage-threshold change — total coverage for the failed run is UNMEASURED; no
  coverage miss is evidenced
- Fixing the govulncheck `GO-2026-6809` finding (first red run 39530700414 at `is7ab609a`) —
  separate issue, separate route
- Touching `internal/runner/terminal_pty_test.go` (base harness) or the sibling fixtures
  (`worker_process_exit_test.go`, `worker_ai_lifecycle_test.go`,
  `process_managed_lifecycle_test.go`) — audited race-free or already fixed (see
  Verification Log)
- A shared Go/Python "fixture ack" helper package — one straggler fixture does not justify a
  new abstraction; the idiom is already consistent with `worker_ai_lifecycle_test.go`
- Routing this through the fleet repair path — the target file is outside the fleet mission
  write allowlist (`tools/launchd/githooks/pre-push`)
- Any deadline retuning (5s ack / 12s READY / 30s subtest / 120s build)

## Timeline

**Day 1** (8 hours):
- Phase 1: fixture fix (2h)
- Phase 2: reproduction harness validation (2h)
- Phase 3: matrix, `-count=20`, `make test`/`make lint` (3h)
- Documentation: Verification Log, changelog line, CI observation (1h)

**Total: ~8 hours across 1 day** (initial estimate 3h, 2x rule applied)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Race is load-specific and CI-only; local zero-failure does not prove CI determinism | Med | The fix is an invariant (atomic rename ⇒ existence implies complete), not a tuned timing; reader retry adds a second independent bound. Both are deterministic protocol properties, not statistical ones |
| `os.replace` semantics differ across CI filesystems (overlayfs, tmpfs) | Low | Same-directory mkstemp ⇒ same filesystem ⇒ no EXDEV path; `rename(2)` atomicity for same-dir renames is POSIX-mandated on ext4/APFS/overlayfs/tmpfs |
| PID reuse between ack and the post-run `ps` absence probe mislabels a survivor | Low | Pre-existing exposure, unchanged by this fix (bounded by the 12s `process.wait` window); noted, not escalated |
| Stray `.worker.pid.*.tmp` if the worker dies between mkstemp and replace | Low | Files live in the per-subtest `TemporaryDirectory`, removed with it; no assertion reads them |
| Test matrix runtime growth | Low | Same waits, same cadence; reader polls at identical 10ms/5s budget |

## Related Documents

<!-- Auto-populated search results were keyword-collision only (no topic match: neural
     embeddings unavailable, "fallback-simhash"); dedup was done by reading the top hits. -->

**Implemented (may inform design):**
- [m-background-worker-lifecycle](../../implemented/v0_54_4/m-background-worker-lifecycle.md) —
  origin of this test file; its "Validation and Acceptance Criteria" states the startup-ack
  contract ("avoid startup races") this fixture violates
- [m-background-worker-lifecycle-validation](../../implemented/v0_54_4/m-background-worker-lifecycle-validation.md) —
  records the same race class fixed for the blocked-AI helper ("file existence could be observed
  before the PID contents were written … writes a same-directory temporary file and renames it
  atomically") and the macOS EPERM teardown lineage

**Planned (check for overlap):**
- None — top SimHash hits (`m-package-test-discovery` 1.00, `run-policy-worker-cache-dir` 0.95)
  are keyword collisions on "test/worker"; read and confirmed unrelated to PTY fixture readiness

**Dedup statement:** this doc covers only the gap the v0.54.4 origin sprint left open — the PTY
worker variant's existence-only reader and non-atomic producer — and adopts that sprint's own
established fix idiom rather than proposing a new one.

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| 1 | Failing Go line 77 is the `t.Fatalf` surfacing the harness exit | `sed -n '70,85p' internal/runner/worker_terminal_pty_test.go` | Confirmed: line 77 = `t.Fatalf("worker PTY: %v\n%s", err, out)` |
| 2 | Current-source producer is open-empty→write (race window exists) | Read `worker_terminal_pty_test.go:41` worker.py string; CPython evaluation order `open()` → `str(os.getpid())` → buffered write → close/flush | Confirmed: file observable empty between open() and write() |
| 3 | Current-source reader polls existence, then single unguarded `int()` | Read `worker_terminal_pty_test.go:44-49` | Confirmed |
| 4 | The race reproduces against current source `d75ef77` | Standalone pair reproduction using the exact strings (no product code): 200 sequential pairs; 200 pairs at 8-way concurrency | **4/400 races** — identical failure mode `ValueError: invalid literal for int() with base 10: ''` (1 sequential + 3 concurrent) |
| 5 | Proposed protocol closes the race | Same pair reproduction with atomic-publish producer + content-poll reader, 200 pairs at 8-way concurrency | **0/200 races**; `os.kill(pid, 0)` liveness verified on all 200 acks |
| 6 | Same race class already fixed in the AI-helper fixture (precedent) | Read `internal/effects/worker_ai_lifecycle_test.go:26-35` | Confirmed: `.pending` + `os.Rename`, comment "Publish only a complete PID: another process may observe file creation … especially on Linux CI" |
| 7 | Reader-side content-poll precedent exists | Read `internal/effects/process_managed_lifecycle_test.go:76-88` (`managedAwaitFile`) | Confirmed: `os.ReadFile` poll with 5ms cadence, 3s deadline |
| 8 | Sibling fixtures are NOT racy (no systemic pattern; single straggler) | Audit: `worker_process_exit_test.go:67-97` (shell `echo $$ > pidfile` completes *before* `exec`; reader gated on separate `ready` marker ⇒ causal order); `process_timeout_group_test.go:24,41` (pidfile read only after the script's `Call` returned) | Confirmed: only the PTY worker fixture races |
| 9 | Cleanup path already tolerates partial pidfile reads | Read `worker_terminal_pty_test.go:58-67` (`try: int(open(pidfile).read()) except ValueError: pass`) | Confirmed; fix strengthens it (complete PID guaranteed) |
| 10 | No shared fixture-ack helper exists (inline protocols) | `grep -rn "pidfile\|\.pid\b" internal/ cmd/ --include="*_test.go"` | Confirmed: three inline protocol families, no shared helper |
| 11 | Working tree clean; HEAD = `d75ef77f59cf4c3c3151fef4840335d269963b8d`; failing commit `d3d389ef` not in this clone | `git status --short` (empty); `git merge-base --is-ancestor d3d389ef… HEAD` → false | Confirmed: reproduction necessarily uses current source (recorded honestly) |
| 12 | Current CI run for `d75ef77` (39582396242) still in progress; no local Go toolchain to rerun the full matrix | Task framing + `find / -name go` (no toolchain); `ailang version` = v0.54.16 built from `4cc0f6a3`, not this tree | Recorded: full-matrix rerun is a Phase 3 gate at fix time |
| 13 | Origin design doc states the "avoid startup races" ack contract | `grep -n "acknowledg\|startup\|race" design_docs/implemented/v0_54_4/m-background-worker-lifecycle.md` | Confirmed: line 222 ("Test helpers must emit a startup acknowledgement … avoid startup races …") |
| 14 | No language-capability claims made (no `ailang check` gate applies) | This doc asserts only fixture/test behavior, verified by file reads above | Confirmed: zero "AILANG does/doesn't support X" statements |

## References

- **Motivation / origin:** [m-background-worker-lifecycle](../../implemented/v0_54_4/m-background-worker-lifecycle.md)
  and its [validation record](../../implemented/v0_54_4/m-background-worker-lifecycle-validation.md)
- **Observed failure:** CI run
  [39544923639](https://github.com/sunholo-data/ailang/actions/runs/39544923639) at
  `d3d389efeba794013e72f861ad3a5317a2880dab`, job 119040812981, step 53 "Check coverage gate
  (M-P2)" — evidence:
  [issues/1892#issuecomment-6313645488](https://github.com/sunholo-data/ailang/issues/1892#issuecomment-6313645488)
- **Current-dev CI:** run 39582396242 at `d75ef77` (in progress at design time) — its
  govulncheck failure is a separate `GO-2026-6809` finding (first red run 39530700414,
  `is7ab609a`), out of scope
- **Route:** product-test PR (independent OpenAI verification and MiniMax concurrence support
  this route); the target file is outside the fleet mission write allowlist
  (`tools/launchd/githooks/pre-push`) — not a fleet repair
- **Precedent idiom:** `internal/effects/worker_ai_lifecycle_test.go:26-35` (producer),
  `internal/effects/process_managed_lifecycle_test.go:76-88` (reader)
- **Axiom reference:** [Design Axioms](/docs/references/axioms)

## Future Work

- If a second fixture ever adopts the same ack protocol, extract the pair scripts into a
  shared testdata helper (deferred until the second consumer exists — see Non-Goals)
- Post-fix, re-observe the coverage gate (M-P2) on a green run to confirm total coverage is
  MEASURED and the 18 downstream steps run
- A fixture-level stress knob (repeat count driven by an env var) for the readiness protocol,
  so CI-only load races can be provoked deliberately — deferred, not needed for this fix

---

**Document created**: 2026-10-21
**Last updated**: 2026-10-21