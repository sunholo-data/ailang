# `run --policy`: the restricted worker drops `AILANG_CACHE_DIR` — the compile cache lands inside `fs_sandbox`

Refs #1547

- **Date**: 2026-10-08
- **Status**: Implemented 2026-10-09
- **Target**: v0.52.6
- **Priority**: P1
- **Estimated**: 2 days
- **Class**: bug (security, low; P1 by issue label)
- **Recommend**: design-doc
- **Shape**: allowlist + supervised default + tests. That is over the direct-fix limit (2 lines, 1 file), so the rubric calls it design-doc. The decisions section below rules on the one real decision (the default when the var is unset), so a sprint plan can be written straight from this row; no quorum needed.
- **Dependencies**: none. Neighbour of, and deliberately out of scope of, [run-policy-result-line-forgeable.md](run-policy-result-line-forgeable.md) (#1548) — that doc's "Neighbour, not part of this doc" paragraph is this doc.
- **Searched**: `AILANG_CACHE_DIR`, `workerEnvAllow`, `MkdirTemp`, `cache` across `cmd/ailang/`, `internal/pipeline/`, `internal/policytool/`, `design_docs/`, `docs/`; `git log --grep 1547` (shallow checkout: no fix commit); the design-doc-creator related-doc search (SimHash + neural: no matches). Nearest coverage: `implemented/v0_41_0/m-executor-policy-hardening.md` (built this supervisor and *mandates* "Resolve caches outside agent write authority or verify entries against the captured graph/toolchain" — its own run lane violates that line), `internal/policytool/cli_ops.go` (`ed1df2d3f`, #1554: the same hole already fixed for policy-tool's CLI children), and [ailang-core-backlog.md](../ailang-core-backlog.md) row 2026-10-03 (names #1547 as #1548's neighbour).

## Problem Statement

**Verified live.** Triage 2026-10-08 reproduced the defect on origin/dev `658ff76a3`; re-verified by code read at `1dfd5615` (origin/dev, 2026-10-08). The issue's repro: `AILANG_CACHE_DIR=$PWD/../cache ailang run --policy p.toml readit.ail` creates `.ailang/` **inside the sandbox** and never touches `../cache`, while the control (`ailang policy-tool` with the same env) honours the var.

**Mechanism** (all three links verified, see Verification Log):

1. The restricted worker's whole environment is `workerEnvAllow` (`cmd/ailang/run_policy_supervise.go:44-48`), which does not list `AILANG_CACHE_DIR`. `workerEnv` (`:228`) drops everything unlisted for restricted policies. `trusted_host` already passes the parent's full environment (`os.Environ`), so the defect is restricted-mode only.
2. The compile cache resolves in `internal/pipeline/cache_store.go:65-76`: `AILANG_CACHE_DIR` override → `<dir>/compile`; unset → `<projectDir>/.ailang/cache/compile`.
3. `projectDir` is `filepath.Dir(src.Filename)` (`internal/pipeline/pipeline_module_phases.go:302-306`) — the entry file's directory, which under a policy must be inside `fs_sandbox` (`cmd/ailang/run_policy.go:131-142` refuses entries outside it). So whenever the operator does not set `AILANG_CACHE_DIR`, the cache lands inside the agent-writable sandbox.

**Why it matters.** The compile cache is *read* by every later `check`/`run`/`test` of the project but is *written-adjacent* to the confined program: with an FS cap the program can write `.ailang/cache` unless the host also denies it. Entries are keyed by source+dependency digests, but the cached interface and artifacts are trusted once the key matches — an agent with write access to the cache controls what a later compile believes a module's interface is. That violates the supervisor's own design freeze ("Resolve caches outside agent write authority **or** verify entries against the captured graph/toolchain" — `design_docs/implemented/v0_41_0/m-executor-policy-hardening.md`, Solution Design §1). The trust hole is not hypothetical: AILANG World row 134 (R-SE-4, V57) works around it today with `fs_deny_write += ".ailang/**"` — a mitigation every host must remember to add. The same class of hole was already fixed for policy-tool's CLI children (`ed1df2d3f`, #1554: a private temp `AILANG_CACHE_DIR` outside the sandbox, removed when the child exits); the `run --policy` lane never got that fix.

**Impact:** every host using `run --policy` in restricted mode without setting `AILANG_CACHE_DIR` (the default) leaves a trusted input inside the program's write authority. Severity low as a compromise (the poison target is the same project's later compiles), P1 as a live footgun.

## Decisions to rule on

1. **Honour `AILANG_CACHE_DIR`.** Add it to `workerEnvAllow`. This is parity with `policy-tool`, `check`, and the motoko adapter (per-task cache dirs), and it is what the issue asks for. No downside: the operator's chosen path was denied only by omission.
2. **Default when the var is unset — this is the real decision.** Today's default is in-sandbox. Options:
   - **(a) Per-run private temp dir outside the sandbox** (recommended). The supervisor creates `os.MkdirTemp("", "ailang-policy-cache-*")` before spawning, injects `AILANG_CACHE_DIR=<dir>` into the worker env, and removes the dir when the worker exits — exactly `ed1df2d3f`'s pattern, hoisted into the supervisor. `MkdirTemp("")` resolves under the supervisor's `TMPDIR`, which is itself in `workerEnvAllow` and operator-controlled, so the injected dir gets the same inside-root check as decision 3 (via `entryInsideSandbox`, which resolves symlinks) — see Review notes 2026-10-08. The invariant "the compile cache is never inside the sandbox in restricted mode" holds with **no host action**. Cost: every restricted run cold-compiles (no cross-run reuse).
   - **(b) Persistent dir outside the sandbox**, e.g. `~/.cache/ailang/policy-compile/<hash-of-sandbox-root>`. Warm compiles, but two concurrent runs against the same sandbox race on `manifest.json` — a whole-file rewrite via `os.WriteFile` (`cache_store.go:82`) with no lock — which is precisely the race the `AILANG_CACHE_DIR` override exists to let orchestrators avoid (`cache_store.go:56-60`; the motoko adapter sets per-task dirs for that reason, `internal/executor/motoko/motoko.go:461`). It also adds a new un-GC'd growth location.
   - **(c) Keep the in-sandbox default** and rely on hosts setting the var (now honoured). Leaves the trust hole for every host that has not read this issue.
   **Recommend (a).** A security default must hold the invariant by itself; cold-compile cost is bounded (the supervisor's deadline already covers a cold run) and measurable in the sprint; an operator who wants reuse sets `AILANG_CACHE_DIR` knowingly (decision 3 governs where they may point it). If cold-compile latency proves material, (b) is the follow-up with per-sandbox keying — not in this fix.
3. **Operator points `AILANG_CACHE_DIR` inside `fs_sandbox`.** Refuse, or warn? A host that deliberately keeps the cache in-sandbox **and** denies writes to it via `fs_deny_write` (the documented World workaround; the knob is real, `internal/policy/policy.go:68-74`) meets the freeze's first arm by other means — the cache is outside the program's *effective* write authority — so a hard refusal would break that legitimate setup. **Recommend: warn once** on the supervisor's stderr when a non-empty operator value resolves inside `res.Root`, naming the poison risk and the `fs_deny_write` mitigation; the run proceeds. Escalate to refusal only if a host asks.
4. **Who may write the cache.** The *worker process*, during compilation (pre-admission, exactly as policy-tool's children run today); the *confined program* may not — under (a) the cache is outside `fs_sandbox` and FS writes are root-confined. `trusted_host` is unchanged (full env, operator's machine, operator's cache).

## Fix sketch

- `cmd/ailang/run_policy_supervise.go` (~20 LOC):
  - add `AILANG_CACHE_DIR` to `workerEnvAllow` (the list already carries a cache-path var, `GOCACHE`, so the precedent is in place);
  - in `supervisePolicyRun`: for a restricted `res` with an empty `config.CacheDir()`, `MkdirTemp` **immediately before** `cmd.Start` (after the pipe setup, whose failures call `refusePolicy` → `os.Exit` at `run_policy_supervise.go:84,88` and would skip a deferred `RemoveAll`), and remove it explicitly if `cmd.Start` fails (`:92-93` also refuses via `os.Exit`); `defer os.RemoveAll` covers the normal path after `cmd.Wait` (the supervisor owns the worker's lifecycle, so the dir cannot outlive the run);
  - apply decision 3's inside-root check to the injected temp dir as well (`entryInsideSandbox(res.Root, dir)`, `cmd/ailang/run_policy.go:149`, resolves symlinks): a `TMPDIR` inside `fs_sandbox` would otherwise land the "outside" cache in-sandbox;
  - thread the injected dir into `workerEnv` (signature change) so a non-empty operator value always wins over the temp default (`config.RawSet` is true even for an empty value — `internal/config/registry.go:153-155` — so the precedence check must be on the value, not on presence);
  - the decision-3 warning (one stderr line) when the operator value resolves inside `res.Root`. The line must not begin with `policy` (sibling #1548 / PR #1679 treats supervisor lines with that prefix as the authoritative result channel).
- Tests (`cmd/ailang/`):
  - unit, following the `TestWorkerEnv_*` pattern (`run_policy_hardening_test.go:454`): a restricted `workerEnv` carries the injected `AILANG_CACHE_DIR`; a set operator value wins over the temp default; `trusted_host` is untouched;
  - end-to-end, following `TestRunPolicy_SupervisorKeepsAllProgramOutput` (`run_policy_supervise_test.go:58`, `runAilangBin`/`buildAilang`/`writePolicy`/`writeAil`): after a restricted run, no `.ailang/` exists in the sandbox; with `AILANG_CACHE_DIR` set outside the sandbox, `<dir>/compile` is written there and nothing in-sandbox; the in-sandbox-value warning appears on supervisor stderr and the run still succeeds.
- Docs: `docs/docs/reference/env-vars.md:75` — record the restricted-run semantics ("the supervised restricted worker honours `AILANG_CACHE_DIR`; when unset it gets a per-run temp dir outside `fs_sandbox`"); `docs/docs/guides/agent-tool-policy.md:131` already documents the policy-tool analogue — add the run lane beside it.

## Goals / Success Criteria

**Primary Goal:** in restricted mode, the compile cache is never inside the sandbox and never program-writable, with `AILANG_CACHE_DIR` honoured as the operator's opt-in.

- [x] Restricted run with `AILANG_CACHE_DIR` unset leaves no `.ailang/` in the sandbox; cache lands in a supervisor-created temp dir removed at exit.
- [x] Restricted run with `AILANG_CACHE_DIR` set writes `<dir>/compile` and nothing in-sandbox.
- [x] An operator value resolving inside `fs_sandbox` produces the one-line supervisor warning (not prefixed `policy`).
- [x] A `TMPDIR` inside `fs_sandbox` does not put the default temp cache in-sandbox (inside-root check on the injected dir).
- [x] No temp cache dir is left behind when the supervisor refuses before or at `cmd.Start`.
- [x] `trusted_host` behaviour unchanged (full env passthrough).
- [x] Existing policy tests pass (`go test ./cmd/ailang/` policy suites); docs rows updated.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A restricted run compiles from the captured module graph, not from cache state a prior run's program could have written |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No signature/effect changes |
| A4: Explicit Authority | +1 | Removes ambient program write authority over a trusted compile input |
| A5: Bounded Verification | 0 | Unchanged |
| A6: Safe Concurrency | 0 | Per-run temp dir cannot race by construction |
| A7: Machines First | 0 | Machine-readable warning only |
| A8: Minimal Syntax | +1 | No language surface at all |
| A9: Cost Visibility | 0 | Cold-compile cost noted; operator can opt into reuse |
| A10: Composability | 0 | Same env contract as policy-tool/motoko lanes |
| A11: Structured Failure | 0 | MkdirTemp failure is a named refusal, not a silent fallback |
| A12: System Boundary | +1 | The cache moves explicitly outside the sandbox boundary |

**Net Score: +5** → **Decision: proceed** (no hard violations; no new syntax, no new CLI surface).

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| 1 | `workerEnvAllow` omits `AILANG_CACHE_DIR` | Read `cmd/ailang/run_policy_supervise.go:44-48`; the list is quoted in Mechanism §1 |
| 2 | Restricted mode drops unlisted vars; `trusted_host` passes `os.Environ()` | Read `run_policy_supervise.go:228-240` (`workerEnv`) |
| 3 | Cache override/default resolution | Read `internal/pipeline/cache_store.go:65-76` (`NewCacheStore`) |
| 4 | `projectDir` is the entry file's directory | Read `internal/pipeline/pipeline_module_phases.go:302-306` |
| 5 | Under a policy the entry file must be inside `fs_sandbox` | Read `cmd/ailang/run_policy.go:131-142` (refusal via `entryInsideSandbox`) |
| 6 | The supervisor never sets `cmd.Dir` (worker inherits the parent cwd) | `grep -n "cmd.Dir\|\.Dir =" cmd/ailang/run_policy_supervise.go` → no hits |
| 7 | The compile cache is the run lane's only default write rooted in the sandbox | `grep os.MkdirAll\|os.WriteFile internal/pipeline/*.go` (production): `cache_store.go:73`, `cache_artifacts.go:89-90` only; the run worker's prompt cache resolves under `$XDG_CACHE_HOME`/`~/.cache` (`paths.go:29`, `XDG_CACHE_HOME` is not allowlisted so the worker falls back to the home path) and the state dir is untouched by `run` |
| 8 | The policy-tool precedent (private temp cache per child, removed at exit) exists | Read `internal/policytool/cli_ops.go:391-405` (`runAilang`, commit `ed1df2d3f`, #1554) |
| 9 | `fs_deny_write` is a real policy knob | Read `internal/policy/policy.go:68-74`, resolved in `internal/policy/resolve.go:258-264` |
| 10 | Concurrent cache sharing races on the manifest (whole-file rewrite, no lock) | Read `cache_store.go:82` (`writeManifest: os.WriteFile`) and `Save()` (marshals the whole manifest); the override's documented purpose is per-process isolation (`cache_store.go:56-60`), which the motoko adapter uses (`internal/executor/motoko/motoko.go:461`) |
| 11 | `RawSet` is true even for an empty value | Read `internal/config/registry.go:153-155` |
| 12 | The supervisor's design freeze already mandates caches outside agent write authority | Read `design_docs/implemented/v0_41_0/m-executor-policy-hardening.md`, Solution Design §1 (the "Resolve caches outside agent write authority or verify entries…" line) |
| 13 | `AILANG_CACHE_DIR` is a documented env var | `internal/config/paths.go:6,21`, `docs/docs/reference/env-vars.md:75` |
| 14 | No planned doc already covers #1547 | design-doc-creator related-doc search (SimHash + neural over planned/ and implemented/): no matches; `ailang-core-backlog.md` row 2026-10-03 names #1547 only as #1548's neighbour; `git log --grep 1547` on this checkout: no fix commit |
| 15 | The issue's live repro (`.ailang/` created in-sandbox, `AILANG_CACHE_DIR` ignored by `run --policy`, honoured by `policy-tool`) | Triage 2026-10-08 on origin/dev `658ff76a3`; mechanism chain (rows 1-5) re-read at `1dfd5615` |

Scope note: this change touches the run supervisor and its focused cache helper (plus tests/docs and environment-description metadata) — not `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects` or `cmd/ailang/exec.go`, so the Conflict Surface section is not triggered. The systemic audit (skill rule: "is this part of a larger pattern?") found the sibling instance of the same hole — policy-tool's CLI children — already fixed in `ed1df2d3f`; this fix adopts that pattern rather than inventing a second mechanism. `AILANG_STATE_DIR` is also absent from `workerEnvAllow`, but the `run` worker never touches the state dir (row 7), so it needs no change.

## Review notes 2026-10-08

Folded from review; the first two are amended above, recorded here for traceability.

1. **`TMPDIR` can put the "outside" cache inside the sandbox.** `os.MkdirTemp("", …)` uses the supervisor's `TMPDIR`, and `TMPDIR` is in `workerEnvAllow` (`cmd/ailang/run_policy_supervise.go:45`). If an operator's `TMPDIR` resolves inside `fs_sandbox`, decision 2(a)'s invariant breaks silently. Apply decision 3's inside-root check to the injected dir too, via `entryInsideSandbox` (`cmd/ailang/run_policy.go:149`, resolves symlinks on both sides). Sprint to choose the response for the injected dir (fall back to a non-`TMPDIR` location or refuse); a warn-and-proceed is not enough here because no operator chose that path.
2. **Temp dir leak on refusal.** `refusePolicy` exits the process (`run_policy_supervise.go:84,88,93` — the stdout/stderr pipe and `cmd.Start` failures), so a `defer os.RemoveAll` registered earlier never runs. Create the dir immediately before `cmd.Start` and remove it explicitly on the `cmd.Start` refusal path.
3. **Residual, out of scope:** a pre-existing (possibly poisoned) `.ailang/cache` already inside the sandbox is still read by `trusted_host` runs and by unsupervised `run`/`check`; this fix stops new restricted runs from writing or reading there, nothing more.
4. **Sibling #1548 (PR #1679):** any new supervisor stderr line added here (the decision-3 warning, a temp-dir failure) must not start with `policy`, so it cannot be confused with the authoritative `policy-result:` / `policy:` channel.

## Cross-links

- Neighbour: [run-policy-result-line-forgeable.md](run-policy-result-line-forgeable.md) (#1548) — the forgeable `policy-result:` line; same supervisor, same release train (2026-10-03 policy fixes closed #1551-#1554/#1558/#1559 but neither #1547 nor #1548).
- Implemented supervisor design: [../implemented/v0_41_0/m-executor-policy-hardening.md](../implemented/v0_41_0/m-executor-policy-hardening.md) (the freeze line this fix enforces).
- Backlog: [../ailang-core-backlog.md](../ailang-core-backlog.md) (2026-10-03 row, which names this doc's issue as the neighbour).

Issue: https://github.com/sunholo-data/ailang/issues/1547

## Implementation record (2026-10-09)

Completed via M-RUN-POLICY-WORKER-CACHE. Cache selection/start cleanup and
symlink-aware operator classification live in `cmd/ailang/run_policy_cache.go`;
the supervisor allocates through that helper only after all pipes are ready and
retains cleanup until worker termination. Generated inside-root placement refuses
and removes explicitly; operator inside-root placement warns through the existing
fresh-line `supervisorLine`. Empty roots are not classified as cwd.

The generated env reference is sourced from `internal/config/paths.go`. Both
milestones and all requested focused/core/lint/size/home-isolation checks passed.
No full `make test`, push or merge was performed. See the
[implementation report](../v0_52_6/run-policy-worker-cache-dir-implementation.md)
for test environment details, latency measurements, artifact lists and PR body.
