# Stapledon's Voyage v0.54.6 Bump: Workaround Register & Retirement Plan

**Status**: Implemented upstream; downstream retirement pending
**Target**: v0.54.7
**Priority**: P1
**Estimated**: ~1 week
**Dependencies**: None for Phases 1–3. Full closure of workaround 4 depends on the B4
unsafe-replay policy round ([m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md),
parked pending A2) — that dependency is explicitly out of scope here.

## Problem Statement

sunholo-data/stapledons-godot (Stapledon's Voyage) bumped to AILANG v0.54.6 and re-checked
every workaround the game carries. All four were reported "still needed". This doc is the
single register so they are not lost, with each claim re-verified against the actual v0.54.6
toolchain (binary at commit `71f4cf6`) and routed to its remedy.

**Current State — the four workarounds, verified:**

| # | Workaround in the game | Verified status on v0.54.6 | Remedy |
|---|---|---|---|
| W1 | `sim/` builds lookups with std/array or lists because `std/map` is evaluator-only under `--strict-bytecode` | **Reproduced live** (below). Worse than reported: the *whole* `std/map` builtin family is unported, not just `fromList` | **Fix here** (Phase 1): VM Map value + native port |
| W2 | `sim/ship.ail` treats `readLine() == ""` as end of session, so the NDJSON protocol must never send a blank line | **Already fixed upstream.** `std/io.readLineOpt()` shipped in v0.54.2 — `Some("")` for a blank line, `Some(text)` for a final partial line, sticky `None` at EOF. Verified live. `readLine` retains legacy string semantics by design | **Retire** (Phase 4): game migrates to `readLineOpt`; no language change |
| W3 | deps target runs `git diff -I '"generated_at"'` then restores the file, because `ailang lock` restamps `generated_at` every run | **Confirmed at code level.** `NewLockFile` always stamps `time.Now().UTC()`. However `ailang lock --check` already exists (exit 1 on drift, ignores generation metadata, writes nothing) — the CI git-diff dance is replaceable today | **Fix here** (Phase 2): byte-identical rewrite on unchanged resolution |
| W4 | `ai/voice.ail` builds stub tones by doubling, citing #1576 (VM silently drops a service request whose handler builds a ~1100+ element list by non-tail recursion) | **Issue remains open, but the reported repro shape was fixed in v0.53.1** (frame limit 1,000 → 10,000, matching both backends; changelog: "deep service handlers within that ceiling now finish"). What is still open is the B4 unsafe-replay policy on non-strict fallback | **Cross-track** (Phase 4): re-test on a genuine v0.54.6 binary; keep the workaround until #1576 closes; no new VM work in this doc |

**Impact:** the game's code is shaped around engine gaps (array-based lookup tables, a
blank-line-hostile protocol, git-diff laundering in CI, doubling-built tone lists). Each
workaround is downstream friction that re-surfaces at every version bump, and W1 forces
`--strict-bytecode` users off std/map entirely — the module the stdlib itself recommends for
bulk-built lookup tables (`std/map.ail`: "USAGE PATTERN: Build once with fromList, then many
lookups").

**Stale-binary observation (recorded for the bump process):** the reporter's auto-attached
binary metadata reads `ailang version: v0.54.5, commit 296231d, md5 c08c18cf…` while the re-check
claims "v0.54.6, commit 71f4cf6". Those cannot both be the binary that ran. W1/W2 were
re-verified here on the real v0.54.6 binary; W4 must be re-tested on a confirmed-fresh binary
before concluding the in-ceiling fix did not cover the game's handler.

## Execution and downstream bump checklist (2026-10-15)

This sprint implements the upstream remedies. The original verification log below records
v0.54.6 findings; it is historical evidence, not the status of the new implementation.
No changes to sunholo-data/stapledons-godot are included in this repository sprint.

| Workaround | Upstream status | Downstream retirement action |
|---|---|---|
| W1: array/list lookup tables | All ten map builtins now native; reporter and full-family fixture pass interpreter/strict VM engine tests | After upgrading to a release containing this sprint, re-run both engine commands and replace lookup builders where appropriate |
| W2: blank line ends session | Existing `readLineOpt` API; game migration **pending** | Migrate `sim/ship.ail`; test `Some("")` for blank input, a final partial line, then sticky `None`; remove protocol's blank-line prohibition only after those tests pass |
| W3: timestamp-ignore CI filter | Metadata preserved for unchanged resolutions; targeted CLI tests pass | After upgrading, replace the ignore/restore dance with `ailang lock --check` or `ailang lock` plus `git diff --exit-code -- ailang.lock` |
| W4: doubling-built stub tones | Keep workaround; #1576 remains cross-tracked under [B4/A2](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) | Re-test the actual service handler on a confirmed source/version/commit binary; retain doubling in `ai/voice.ail` until #1576 closes; attach any still-failing handler evidence to that issue |

Local source-built evidence: source stdlib version **v0.54.9**, code commit
`0145da1aee83e10bc668820da0cce6f8d855aa90`, built 2026-10-15 09:16:22 UTC with
`make build VERSION=v0.54.9` (local build label, not a release). Teaching prompt refreshed:
**v0.16.7**. The exact reporter module prints `2` on both engines; the all-ten-operation
fixture prints `true` under strict bytecode and its inline test passes. Consecutive CLI
locks plus `--check` leave the generated file byte-identical; a malformed lock errors and
is not overwritten. Unit regressions additionally cover path dependencies, changed
resolutions, prior generator/version metadata and unreadable files. Native map tests run
20 times against evaluator behavior, including duplicates and negative/integer key order.

At each bump, record the actual `ailang version` and commit used by the test commands,
then update all four cited game call sites. The v0.54.5 auto-attached metadata does not
establish a v0.54.6 reproduction. `ailang doc` continues to suggest `docs` without adding
an alias. Swapped map lookup arguments now name `lookup` in the application diagnostic.

## Verification Log

Every language/tooling claim above carries live or code-level evidence, run against the v0.54.6
binary at commit `71f4cf6` (2026-10-12):

| ID | Claim | Evidence | Verdict |
|----|---|---|---|
| V1 | `std/map.fromList` is evaluator-only under `--strict-bytecode` | `ailang run --quiet --entry main fl.ail` → `2`; `ailang run --quiet --bytecode --strict-bytecode --entry main fl.ail` → `vm: CALL: std/map.fromList is evaluator-only (compiler: pure builtin "__map_from_list" has no VM implementation (polymorphic: …) but no interop bridge is wired` (exit 1) | Confirmed |
| V2 | The whole `__map_*` family is unported, and the VM has no Map value | `internal/vm/builtin_coverage_test.go` `unportedPure` lists `__map_empty/insert/lookup/member/remove/size/keys/values/from_list/to_list` (ratchet for #1447). A lookup-only strict-bytecode program fails earlier: `std/map.empty is evaluator-only (… (map-value: the VM has no Map value))` | Confirmed |
| V3 | `readLine` returns `""` for both blank line and EOF | `printf '\n' \| ailang run … rl.ail` → `[][]` (live) | Confirmed (legacy semantics retained by design) |
| V4 | The requested Option-variant API already exists | `std/io.ail:20-22`: `readLineOpt() -> Option[string]`, "Preserve blank lines and distinguish EOF (None)"; shipped **v0.54.2** (changelogs/v0.32-current.md). Live: blank line → `Some()`, `x\n` → `Some(x) None`, `x\ny\n` → `Some(x) Some(y)`, `x` (no newline) → `Some(x) None` | Confirmed — W2 retireable |
| V5 | `ailang lock` rewrites `generated_at` on every run; no preservation path exists | `internal/pkg/lockfile.go:65` `GeneratedAt: time.Now().UTC()` in `NewLockFile`, called unconditionally by `ResolveLock` (lockcheck.go); `pkgLockCommand` flag set contains only `--help`/`--check` — no `--preserve*` flag (grep of cmd/ailang/pkg_commands.go) | Confirmed |
| V6 | A resolution-only CI check already exists | `internal/pkg/lockcheck.go`: `CheckLock` "ignoring the generation metadata (timestamp, generator, ailang_version)"; `ailang lock --check` help: "exit 1 on drift", writes nothing | Confirmed — W3's git-diff workaround replaceable today |
| V7 | On a clean machine `ailang lock`, not `install`, fills the cache for a committed manifest | `cmd/ailang/pkg_install.go` extracts to the registry cache only for the *named* package spec and appends it to ailang.toml; `pkgLockCommand` resolves *all* deps in the existing manifest (git deps via `pkg.NewGitCache`) | Confirmed at code level — record in the dependency-workflow docs |
| V8 | #1576 still open; the silent-drop shape was in-ceiling | v0.53.1 changelog: frame limit 1,000 → 10,000, "Deep service handlers within that ceiling now finish … Refs #1576". `docs/docs/reference/limitations.md`: "it does not make replay safe (Refs #1576)". [m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) addendum 2026-10-08 documents the exact reported shape (rc 0, stderr silent, `--verbose` reveals `bytecode path unavailable (vm: stack overflow); falling back to evaluator`) and files the residual as **B4 unsafe effect replay, P0, parked pending A2** | Confirmed — issue open, but the ~1100-frame repro is within the new ceiling; re-test before trusting the workaround is still load-bearing |
| V9 | `ailang doc` is gone | Live: `Error: unknown command 'doc'` + `Did you mean 'docs'?` (suggestCommand already handles it, cmd/ailang/commands.go:286) | Confirmed — suggestion works as designed |
| V10 | Unification errors for argument-order mistakes do not name the function | Live: `lookup("b", m)` → `type unification failed at [function application at lo.ail:5:9]: failed to unify parameter 0: cannot unify Map[α7, α8] with string` — no mention of `lookup` or std/map. Source: `internal/types/unification_types.go:53` wraps the parameter error without callee context | Confirmed |
| V11 | `std/map.lookup` argument order is map-first | `std/map.ail:42`: `lookup[k, v](m: Map[k, v], key: k) -> Option[v]` | Confirmed |

## Goals

**Primary Goal:** Retire every workaround Stapledon's Voyage carries on v0.54.6 by fixing what
is genuinely broken, migrating what is already fixed, and cross-tracking what lives in
another doc — so the v0.55 bump has an empty (or #1576-only) re-check list.

**Success Metrics:**
1. The reporter's exact `fl.ail` repro prints `2` under
   `ailang run --quiet --bytecode --strict-bytecode --entry main fl.ail` (W1).
2. An unchanged `ailang lock` run leaves `ailang.lock` byte-identical
   (`git diff --exit-code` clean) (W3).
3. `ailang lock --check` documented as the CI drift gate; the game's `-I generated_at`
   laundering deleted (W3).
4. `sim/ship.ail` migrates to `readLineOpt` and the NDJSON "never send a blank line"
   constraint is dropped (W2 — downstream change, verified by the game's own tests).
5. All four workaround call sites in sunholo-data/stapledons-godot cite this doc and
   state their retirement status.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Byte-identical lock rewrites; engine parity removes engine-conditional program shapes |
| A2: Replayability | 0 | No trace/replay changes; the replay-policy half of #1576 stays in its own doc |
| A3: Effect Legibility | 0 | No effect changes (`readLineOpt` already exists under `IO`) |
| A4: Explicit Authority | 0 | No new ambient access |
| A5: Bounded Verification | +1 | The unported-builtin ratchet (coverage test) enforces the family port mechanically |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Diagnostics name the callee (machine-actionable); generated code stops sniffing the engine to pick a data structure |
| A8: Minimal Syntax | +1 | No new syntax, no new language surface |
| A9: Cost Visibility | 0 | Map cost model unchanged from the evaluator |
| A10: Composability | 0 | std/map API unchanged |
| A11: Structured Failure | +1 | Unification error gains the missing function context — a typed-diagnostic improvement |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: lock preservation *removes* a nondeterminism (wall-clock timestamp in a committed artifact)
- [x] A3: no hidden side effects
- [x] A4: no ambient access
- [x] A7: no human-convenience surface added (the `doc` alias is *rejected* on this axiom — see DX1)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: VM Map value representation reuses the evaluator's canonical string-key encoding (`eval.MapValue{Entries: map[string]*MapEntry}` with sorted-key iteration), not a new typed map | Must preserve exact evaluator semantics (keys limited to int/string/bool per `internal/builtins/map.go`; deterministic iteration via `sortedCanonicalKeys`; copy-on-write insert) or the two engines diverge silently | agent | design | high |
| D2: On unchanged resolution, `ailang lock` preserves the *entire* generation metadata block (`generated_at`, `generator`, `ailang_version`) from the committed lock; a changed resolution stamps fresh | Defines the CI contract `git diff --exit-code` relies on; preserving only `generated_at` would still churn if `ailang_version` bumps | human | design | med |
| D3: Application unification errors carry the callee name when the head is a named reference; raw `Unify` callers keep neutral wording | Error-text change ripples into goldens and teaching prompts; the M-ARITY-STYLE diagnostic directly above this site must not regress | agent | design | med |
| D4: No `doc` command alias | Adds a command route (simplicity-audit gate counts routes) for zero machine value; `suggestCommand` already answers the typo | human | design | low |
| D5: #1576 gets no new work here | The residual scope is B4 unsafe replay (P0, parked pending A2) — duplicating it here would fork the tracking | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D2 (lock metadata preservation semantics) — approved in the user handoff; preserve all generation metadata on unchanged resolution
- [x] D4 (`doc` alias rejection) — approved in the user handoff; no alias added

## Conflict Surface Analysis

This design touches `internal/vm/`, `internal/bytecode/`, `internal/types/`, and
`cmd/ailang/pkg_commands.go`, so the surface is enumerated:

1. **Positions extended:** (a) VM CALL dispatch table for `__map_*` builtin names
   (currently routed to the "evaluator-only" stub branch in
   `internal/bytecode/compiler/builtins.go:134`); (b) the parameter-unification error path
   in `internal/types/unification_types.go` used by *every* function application;
   (c) the `ailang lock` write path used by every lock invocation.

2. **Other constructs living in those positions:**
   - (a) The still-unported builtins (`__html_*`, `__xml_*`, `__get_header`,
     `__has_header`, `__list_takeMap`, `__list_takeFlatMap`) must **remain** evaluator-only
     with their exact current error shape; the ratchet only shrinks the `__map_*` entries.
   - (b) The arity-mismatch path directly above (M-ARITY-STYLE coded diagnostic,
     `arityMismatchMsg`) must keep its wording and MOD codes; raw unit-test callers of
     `Unify` pass functions with arbitrary orientation, so the callee context must be
     added by the *application* call site, not inside `Unify`.
   - (c) `ailang lock --check` must never write; the empty-dependencies early path still
     creates a lock; path deps stay manifest-relative (`ailang#1498`,
     `TestLockFile_PathDepsAreRelativeAndPortable`).

3. **Disambiguation:** builtins dispatch on exact string names in the registration tables
   (`bytecode.BuiltinNames`); no parsing or syntax changes anywhere.

4. **Programs that MUST still work (fixtures):**
   - `std/map` usage on the evaluator — byte-identical results (run `fl.ail` both engines,
     diff stdout).
   - The strict-bytecode *error* for a still-unported builtin, e.g. a `std/xml` program,
     still names the builtin and reason.
   - An arity-mismatch program still gets the M-ARITY-STYLE diagnostic (regression: the
     nearest existing test is `internal/types` arity style tests — implementer locates by
     `arityMismatchMsg` grep).
   - `ailang lock --check` on a current lock exits 0 and writes nothing
     (`internal/pkg/lockcheck.go` behavior + `pkgLockCheck` path).
   - `TestPureBuiltinCoverage` fails if a ported builtin is left on the ratchet (already
     enforced by the test itself).

5. **Deliberate changes (intentional incompatibilities):**
   - Strict-bytecode `std/map` programs go from a hard error to correct results.
   - `ailang lock` on an unchanged resolution no longer restamps `generated_at`.
   - Application unification errors include the callee name — golden files and any
     prompt text embedding the old message shape change.

## Solution Design

### Overview

Four workarounds, three remedies: **fix** (VM Map port; lock determinism), **migrate**
(readLineOpt), **cross-track** (#1576 stays with the B4 doc until the replay policy lands).
Plus two recorded DX notes: one rejected (doc alias), one small fix (unification errors name
the callee).

### W1 — VM Map value + native std/map family port (Phase 1)

The VM has no Map value (`map-value: the VM has no Map value`). Port the whole family, not
just `fromList` — the ratchet lists all ten `__map_*` builtins as unported, so a fromList-only
port would leave `lookup` broken one call later (V2 evidence).

**Components:**
1. **VM Map value**: mirror `eval.MapValue` — `Entries map[string]*MapEntry` with the same
   canonical string key encoding (keys are int/string/bool only, per `internal/builtins/map.go`),
   sorted-key canonical iteration (`sortedCanonicalKeys`), copy-on-write `insert`. Reusing the
   encoding is what guarantees engine parity; a structurally different representation would
   need its own equivalence proof (D1).
2. **Native builtin implementations**: register the ten `__map_*` names in the VM's native
   builtin table (the `bytecode.BuiltinNames` path), implementing each against the VM value.
   The polymorphism that blocks the auto-adapt converter (`ReasonPolymorphic` for
   `from_list`/`to_list`) is not a problem for a hand port: values are opaque to the
   implementation, exactly as in the evaluator.
3. **Ratchet shrink**: delete the ten `__map_*` rows from `unportedPure` in
   `internal/vm/builtin_coverage_test.go`; the test's own both-directions failure mode
   (fails when a listed entry became native) enforces this.

### W3 — Lock determinism (Phase 2)

`ResolveLock` always builds a fresh `LockFile` with `GeneratedAt: time.Now().UTC()` (V5).
Change `pkgLockCommand`: after resolving, if a committed lock exists and
`LockDrift(current, expected)` is empty, write `expected` with the committed lock's
generation metadata block (`generated_at`, `generator`, `ailang_version`) instead of fresh
values (D2). Any drift → fresh stamp, as today. `Save` already writes deterministic JSON
(sorted packages, sorted effects/exports), so with metadata preserved the output is
byte-identical.

Document `ailang lock --check` as the CI drift gate (it already ignores generation metadata,
V6), and record the clean-machine note (V7): `ailang install` fills the registry cache only
for the package it installs; a fresh checkout with a committed manifest gets its
dependencies resolved and cached by `ailang lock`.

### DX2 — Unification errors name the callee (Phase 3)

`internal/types/unification_types.go:53` wraps parameter mismatches without any callee
context (V10), which is why a swapped `lookup(key, m)` reads as a bare
`cannot unify Map[α7, α8] with string`. Thread the applied function's name (when the
application head is a named reference — the elaborate layer already has the span) into the
application-site error, e.g.:

```
type unification failed at [function application at lo.ail:5:9]: failed to unify parameter 0 of `lookup` (Map[k,v], key: k): cannot unify Map[α7, α8] with string
```

Constraint from the Conflict Surface: the context is added at the application call site, not
inside `Unify` (raw callers pass arbitrary orientation and existing tests assert on the
current wording); the adjacent M-ARITY-STYLE diagnostic must not change.

### W2 / W4 / DX1 — Record and retire (Phase 4, docs + downstream only)

- **W2**: the game migrates `sim/ship.ail` to `readLineOpt` and drops the
  "never send a blank line" protocol constraint. No AILANG change; `docs/docs/reference/limitations.md`
  and `effects.md` already document `readLineOpt` correctly. Downstream cites this doc.
- **W4**: keep the `ai/voice.ail` doubling workaround, citing #1576, until the issue closes.
  Re-test the game's service handler on a confirmed-fresh v0.54.6 binary first (V8: the
  in-ceiling shape was fixed in v0.53.1; the residual open scope is B4 replay policy).
  Any new still-failing evidence goes on #1576, not into a new doc.
- **DX1**: `ailang doc` → `Did you mean 'docs'?` already works (V9). **Rejected: no alias**
  (D4) — it would add a command route against the simplicity gates for zero machine value.
- Add a "downstream bump re-check" note listing W1–W4 with their retirement status so the
  v0.55 bump re-check starts from this register.

### Implementation Plan

**Phase 1: VM Map value + std/map port** (~3 days)
- [x] VM Map value type mirroring `eval.MapValue` (encoding, COW insert, sorted iteration)
- [x] Native implementations for the ten `__map_*` builtins in the VM
- [x] Shrink the `unportedPure` ratchet; add strict-bytecode parity tests (build/lookup/remove/iterate)
- [x] Acceptance: the reporter's `fl.ail` prints `2` under `--strict-bytecode`

**Phase 2: Lock determinism** (~0.5 day)
- [x] Preserve generation metadata on drift-free re-lock in `pkgLockCommand`
- [x] Tests: unchanged re-lock is byte-identical; drifted re-lock restamps; `--check` still never writes
- [x] Docs: `lock --check` as the CI gate; clean-machine cache behavior (V7)

**Phase 3: Callee-named unification errors** (~1 day)
- [x] Thread the callee name into the application unification error at the application site
- [x] Update affected goldens; add a regression test using a swapped-args `std/map.lookup`
- [x] Verify the M-ARITY-STYLE diagnostic is unchanged

**Phase 4: Register, migrate, cross-track** (~0.5 day)
- [x] Doc note: bump re-check register with retirement statuses (W1 fixed, W2 retireable, W3 fixed, W4 pending #1576/B4)
- [ ] Downstream (stapledons-godot, their repo): migrate `readLineOpt`; re-test #1576 handler; delete the git-diff laundering once Phase 2 ships

### Files to Modify/Create

**New files:**
- `internal/vm/values_map.go` — VM Map value type (~150 LOC)
- `internal/vm/builtins_map.go` — native `__map_*` implementations (~300 LOC)

**Modified files:**
- `internal/vm/builtin_coverage_test.go` — shrink the ratchet by ten entries (~10 LOC)
- `internal/bytecode/compiler/builtins.go` / bytecode registration — map builtins become native (~20 LOC)
- `cmd/ailang/pkg_commands.go` + `internal/pkg/lockfile.go` — metadata-preserving re-lock (~40 LOC)
- `internal/types/unification_types.go` + its application-site caller — callee-named errors (~30 LOC)
- `docs/docs/reference/limitations.md`, dependency-workflow guide — W2/W3 documentation (~20 LOC)
- Golden files touched by the error-text change

## Examples

### Example 1: W1 — std/map under strict bytecode

**Before (v0.54.6, live):**
```
$ ailang run --quiet --bytecode --strict-bytecode --entry main fl.ail
Error: bytecode execution failed: vm: CALL: std/map.fromList is evaluator-only
(compiler: pure builtin "__map_from_list" has no VM implementation …) but no
interop bridge is wired
```

**After:**
```
$ ailang run --quiet --bytecode --strict-bytecode --entry main fl.ail
2
```
The game's `sim/` lookup-table builders switch back to `std/map` (`fromList` + `lookup`).

### Example 2: W3 — lock in CI

**Before:** `git diff -I '"generated_at"' ailang.lock && git checkout ailang.lock`
**After:** `ailang lock && git diff --exit-code ailang.lock` (byte-identical on unchanged
resolution); drift detection in CI may use `ailang lock --check` (exit 1 on drift) instead of
any git diff.

### Example 3: DX2 — swapped map arguments

**Before (live):**
```
Error: type error in lo (decl 0): type unification failed at [function application at
lo.ail:5:9]: failed to unify parameter 0: cannot unify Map[α7, α8] with string
```
**After:**
```
Error: type error in lo (decl 0): type unification failed at [function application at
lo.ail:5:9]: failed to unify parameter 0 of `lookup`: cannot unify Map[α7, α8] with string
```

## Success Criteria

- [x] `fl.ail` (reporter's exact repro) prints `2` on both engines, including `--strict-bytecode` (acceptance: run both, diff)
- [x] A strict-bytecode `std/xml` or `takeMap` program still fails with the evaluator-only error naming the builtin (ratchet parity)
- [x] Two consecutive `ailang lock` runs with no manifest change produce byte-identical files (`git diff --exit-code` clean)
- [x] Drifted resolution still restamps `generated_at`; `ailang lock --check` behavior unchanged (exit codes 0/1, never writes)
- [x] Swapped-args `lookup` error names `lookup`; arity-mismatch diagnostic unchanged (golden diff review)
- [x] Focused tests passing (`make test-changed`; full suite remains CI); docs updated; bump re-check register written
- [ ] Downstream: stapledons-godot W2 migration merged; W4 re-test result recorded on #1576

## Testing Strategy

**Unit tests:**
- VM Map value: encoding round-trip per key kind (int/string/bool), COW insert isolation,
  sorted iteration order — mirror the evaluator's map builtin tests
- Lock: byte-identity on unchanged resolution, restamp on drift, `--check` write-free
- Unification: callee-name inclusion; neutral wording for raw `Unify` callers

**Integration tests:**
- Strict-bytecode parity: run std/map programs on both engines and diff stdout
- The unported-builtin ratchet (`TestPureBuiltinCoverage`) shrinks by exactly ten and
  still fails for the remaining unported families

**Manual testing:**
- The reporter's two exact repro commands (fl.ail, rl.ail) plus `printf 'x\ny\n'` variants
- A clean-machine `ailang lock` on a committed manifest (cache-fill note, V7)

## Deferred Decisions

- VM Map value GC/footprint policy (the evaluator had an M-V1-MEMORY-FOOTPRINT F17 note on
  `MapValue.Insert`) — agent may resolve, following the evaluator's existing approach
- Exact wording of the callee-named error (subject to golden review) — agent may resolve
- Whether `ailang lock --check` gains a `--json` output for CI reporting — agent may
  resolve if cheap, otherwise drop (not part of the ask)

## Non-Goals

- **The B4 unsafe-replay policy** — the residual open half of #1576. Tracked in
  [m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) (P0, parked
  pending A2). This doc does not touch `internal/runner/entrypoint.go` fallback behavior.
- **Porting the other unported builtin families** (`__html_*`, `__xml_*`, headers, closure
  HOFs) — same mechanism, separate ratchet shrink, not what the bump reported.
- **A `doc` command alias** — rejected (D4); `suggestCommand` already answers it.
- **Changing `readLine` semantics** — legacy string contract is intentional; the Option
  variant is the supported EOF-distinguishing API.
- **Migrating the game's code** — downstream repo's work; this doc records what to migrate to.

## Timeline

**Week 1** (~5 days):
- Days 1–3: Phase 1 (VM Map port + ratchet)
- Day 4 (half): Phase 2 (lock determinism) + Phase 3 start
- Day 5: Phase 3 finish, goldens, Phase 4 register, docs, full test run

**Total: ~5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| VM Map port diverges from evaluator semantics (iteration order, COW, key encoding) | High — silent engine divergence, the exact class this program exists to prevent | Reuse the evaluator's encoding verbatim (D1); both-engine stdout-diff parity tests in the sprint |
| Error-text change breaks downstream prompt/golden consumers | Med | Bounded to the application unification path; golden sweep in the sprint; old shape never carried the callee name so consumers cannot depend on it |
| Lock metadata preservation masks a genuine resolution change | Med | `LockDrift` is package-level comparison already used by `--check`; preservation applies only when drift is empty; drift tests cover restamping |
| #1576 re-test shows a *second* in-ceiling VM defect | Med | Evidence goes on #1576 / the parity-bugs doc; this doc's W4 row updates to cite it — no scope change here |

## Related Documents

- [m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) — B4 unsafe replay
  (the open half of #1576) and the 2026-10-08 #1576 addendum
- [m-vm-stack-limit-parity.md](../../implemented/v1_0_0/m-vm-stack-limit-parity.md) — the
  v0.53.1 frame-limit alignment that fixed the in-ceiling #1576 shape
- [ailang-core-backlog.md](../../planned/ailang-core-backlog.md) — lock portability history
  (path-dep relativity, `ailang#1498`)
- [#1447 ratchet] `internal/vm/builtin_coverage_test.go` — unported-pure-builtin policy
- [docs/docs/reference/limitations.md](../../../docs/docs/reference/limitations.md) —
  recursion ceilings, `readLineOpt` reference

## References

- Issue #1576 — bytecode VM drops deep non-tail service handlers (stapledons_godot)
- Issue #1447 — VM/evaluator builtin parity ratchet
- changelogs/v0.32-current.md — v0.53.1 (frame limit), v0.54.2 (`readLineOpt`) entries
- Reporter's bump report: sunholo-data/stapledons-godot, v0.54.6 re-check (2026-10-12)

## Future Work

- Port the remaining unported builtin families off the ratchet (same mechanism as Phase 1)
- B4 unsafe-replay policy (A2 round) closing #1576 fully — then the ai/voice.ail doubling
  workaround retires

---

**Document created**: 2026-10-12
**Last updated**: 2026-10-15
