# M-PROCESS-MODES: Replay-Contract Modes for the Process Effect

**Status**: Planned
**Target**: v1.1.0
**Priority**: P2 — Low (extends an already-shipped effect; unblocks deterministic-test workflows)
**Estimated**: ~36 hours (~1–2 sprints)
**Dependencies**:
  - [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) — must land first; this doc plugs Process into that framework's `[mode=...]` parameter and replay-contract taxonomy
  - [M-PROCESS-EXEC](../../implemented/v0_8_1/m-process-exec.md) (✅ implemented v0.8.1) — the existing Process effect; this milestone parameterises it, doesn't replace it

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | `Process[mode=mocked]` makes subprocess output deterministic on replay — the strongest guarantee available for an effect that crosses to the host OS |
| A2: Replayability | +2 | **Primary goal.** Closes the "Full replay requires --replay-trace mode" follow-up flagged in M-PROCESS-EXEC's own A2 score |
| A3: Effect Legibility | +1 | Mode is visible in the effect row — readers see whether a function depends on live host state |
| A4: Explicit Authority | +1 | M-ENTROPY envelopes can pin modules to `mocked` only; capability discipline can deny `live` |
| A5: Bounded Verification | 0 | No change to type-level verification surface |
| A6: Safe Concurrency | 0 | No concurrency changes; reuses existing `spawnProcess` infra |
| A7: Machines First | +1 | Test harnesses can read a function's effect row and decide whether to record/replay/run live without inspecting the body |
| A8: Minimal Syntax | 0 | Reuses M-EFFECT-REFINEMENT's `[mode=...]` parameter syntax — no new grammar |
| A9: Cost Visibility | +1 | `live` mode = real subprocess cost; `mocked` mode = trace replay (cheap). Cost differential is visible in the type. |
| A10: Composability | +1 | Composes with M-ENTROPY envelopes, existing trace machinery, and other `[mode=...]`-bearing effects |
| A11: Structured Failure | +1 | Mode mismatches (e.g. `mocked` with no recorded transcript) become structured errors with the same `ProcessError` ADT |
| A12: System Boundary | +1 | The mode parameter makes the boundary crossing into host OS *typed* — `mocked` declares "no host crossing", `live` declares "real boundary" |

**Net Score: +9** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): `mocked` strengthens determinism, `live` is unchanged from today's behaviour — no implicit nondeterminism added
- [x] A3 (Effects): Mode is part of the effect row, not ambient
- [x] A4 (Authority): M-ENTROPY can deny modes per-module; no ambient elevation
- [x] A7 (Machines First): Mode reading requires no body inspection

## Problem Statement

The Process effect ships in v0.8.1 with full security plumbing (allowlist, timeout, output cap, structured `ProcessError`) — but it has **no replay contract**. Every `! {Process}` call talks to the live host today, no matter what the trace harness asks for. The implemented design itself flagged this gap explicitly:

> "Full replay requires `--replay-trace` mode that returns recorded outputs without spawning processes (not in v1, but structure supports it)."
> — [M-PROCESS-EXEC §Axiom Scoring (A2)](../../implemented/v0_8_1/m-process-exec.md)

[M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) introduces the framework that closes this gap: parameterised effects (`!{E[mode=...]}`) with a three-element replay-contract taxonomy `{deterministic, re-sampleable, opaque}`. M-EFFECT-REFINEMENT's worked examples target Rand, Clock, Net, and FS. **This doc adds Process as the fifth axis under that framework.**

A note on adjacent surfaces:

- **MCP tool calls and HTTP RPC** route through the **Net effect** today and inherit `Net[mode=recorded|live]` for free once M-EFFECT-REFINEMENT lands. They are *not* in scope for this doc.
- **True subprocesses** (`os.exec`, shell-out) are what this doc is about.

**Current State:**
- `Process` is a single effect token with no parameters.
- Subprocess invocations are always live; trace replay cannot return recorded `ProcessOutput`.
- Deterministic tests that depend on subprocess output (e.g. `git rev-parse HEAD`, `uname -s`, locale-dependent tools) are not currently expressible in AILANG without external sandboxing.
- M-ENTROPY envelopes have nothing to constrain on the Process axis.

**Impact:**
- Test reproducibility: any AILANG program calling `exec` is non-replayable, breaking the broader replay story.
- Cross-platform brittleness: tests on different hosts produce different outputs, with no language-level mechanism to pin them.
- Framework asymmetry: M-EFFECT-REFINEMENT delivers replay contracts for Rand/Clock/Net/FS, but Process — arguably the most non-deterministic of all effects — is left out.

## Goals

**Primary Goal:** Add `[mode=mocked|live|recorded]` parameters to the Process effect so it integrates with M-EFFECT-REFINEMENT's replay-contract taxonomy, with `mocked` blocking host execution at runtime, `recorded` capturing transcripts on first run, and bare `! {Process}` continuing to work as `! {Process[mode=live]}` for backwards compatibility.

**Success Metrics:**
- `Process[mode=mocked]` causes any `exec`/`execText`/`spawnProcess` call to return from the trace, never spawning a subprocess. Missing transcript = structured error.
- `Process[mode=recorded]` writes subprocess transcripts (cmd, args, stdout, stderr, exit, duration) into the trace under the same key the replay harness reads.
- `Process[mode=live]` is observationally identical to today's `! {Process}` — no behavioural changes for existing code.
- M-ENTROPY envelopes can constrain mode selection per-module (e.g. "test modules may only use `mocked`").
- At least one example (`process_modes_test.ail`) demonstrates a deterministic test using `mocked` against transcripts captured by `recorded`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Transcript identity key (cmd+args+cwd+env vs cmd+args only) | Determines whether two runs in different cwd or with different env hit the same recorded transcript | human | design | high |
| Behaviour on transcript miss in `mocked` mode (error vs fall-through to live vs return synthetic empty) | Wrong choice silently lets non-deterministic data leak past `mocked` | human | design | high |
| Trace storage format for transcripts (extend existing trace schema vs separate sidecar file) | Affects the replay harness, dashboard, and any external consumers | human | design | med |
| Modes for `spawnProcess` / interactive stdin (mocked replay of streams is hard) | `spawnProcess + writeProcessStdin` is interactive; deterministic replay of a streaming session is non-trivial | human | design | med |
| Default for bare `! {Process}` — desugars to `live` (back-compat) vs `recorded` (safer) | Affects every existing program that uses Process; back-compat picks `live` | agent | compile | low |
| Whether `recorded` mode requires explicit capability beyond the existing Process capability | Recording can leak host state to traces; may need a separate `Process[mode=recorded]` capability | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Transcript identity key
- [ ] `mocked` transcript-miss behaviour
- [ ] Trace storage format
- [ ] `spawnProcess` mode handling (in-scope vs deferred)

## Solution Design

### Overview

Three parameter values on the existing Process effect token:

- `Process[mode=live]` — re-sampleable. Spawns a real subprocess, records to trace if recording is on. **Today's behaviour.** Bare `! {Process}` desugars here.
- `Process[mode=recorded]` — re-sampleable on first run, deterministic on replay. Always spawns; always writes the transcript to the trace under a stable key.
- `Process[mode=mocked]` — deterministic. Never spawns. Reads transcript from trace; structured error if missing.

The runtime dispatches on the mode parameter at the effect-handler boundary in `internal/effects/process.go`. The replay harness reads `Process[mode=mocked]` and serves transcripts; `live` is dispatched to the existing exec path; `recorded` runs through both (live exec + write to trace).

### Architecture

**Components:**

1. **Mode parser/AST** — provided by M-EFFECT-REFINEMENT; this milestone just declares the three Process modes.
2. **Replay-contract registration** — in `internal/replay/contracts.go` (created by M-EFFECT-REFINEMENT), register `(Process, live) -> re-sampleable`, `(Process, recorded) -> re-sampleable+capture`, `(Process, mocked) -> deterministic`.
3. **Mode-aware effect handler** — `internal/effects/process.go` dispatches on the mode parameter pulled from the effect row context. `mocked` reads from trace; `recorded` runs + writes; `live` is the existing path.
4. **Trace schema extension** — add `process_transcripts` keyed by `<identity-key, sequence>` to the trace event format. Identity key resolution is a Design Freeze item.
5. **`ProcessError` extension** — add `TranscriptMissing` variant for `mocked` misses.
6. **M-ENTROPY hook** — Process appears in entropy envelopes alongside Rand/Clock/Net/FS, with the same `modes:` declaration shape.

### Mode dispatch (sketch)

```
exec(cmd, args) at effect row containing Process[mode=M]:

  match M {
    live     => spawn(cmd, args); return ProcessOutput
    recorded => out = spawn(cmd, args); trace.write(transcript_key, out); return out
    mocked   => match trace.read(transcript_key) {
                 Some(t) => return ProcessOutput.from(t)
                 None    => return Err(TranscriptMissing { cmd, args, key })
               }
  }
```

### Implementation Plan

**Phase 1: Mode plumbing on synchronous exec** (~16 hours)
- [ ] Register `(Process, live|recorded|mocked)` in the M-EFFECT-REFINEMENT contract registry
- [ ] Resolve mode from effect-row context inside the exec handler
- [ ] Implement transcript write in `recorded` mode
- [ ] Implement transcript read in `mocked` mode + `TranscriptMissing` error
- [ ] Back-compat shim: bare `! {Process}` desugars to `! {Process[mode=live]}`
- [ ] Tests: round-trip recorded → mocked produces byte-identical output
- [ ] Example: `examples/runnable/process_modes_test.ail` demonstrates a deterministic test

**Phase 2: M-ENTROPY integration + spawnProcess decision** (~12 hours)
- [ ] Add Process to entropy envelope schema; envelope-level mode validation at module load
- [ ] Decide and document `spawnProcess` mode handling (defer to v1.2 or implement minimal)
- [ ] If implementing: streaming-transcript schema for spawn + stdin

**Phase 3: Docs + prompt updates** (~8 hours)
- [ ] Update `ailang prompt` to teach Process modes
- [ ] Update existing process_demo.ail to optionally use `recorded` mode
- [ ] CHANGELOG entry referencing M-EFFECT-REFINEMENT and M-PROCESS-EXEC

### Files to Modify/Create

**New files:**
- `internal/effects/process_modes.go` — Mode dispatch + transcript read/write (~180 LOC)
- `internal/replay/process_contract.go` — Register Process contracts with the M-EFFECT-REFINEMENT registry (~60 LOC)
- `examples/runnable/process_modes_test.ail` — Deterministic test using mocked mode (~80 LOC)

**Modified files:**
- `internal/effects/process.go` — Mode-aware handler dispatch (~80 LOC delta)
- `internal/effects/process_spawn.go` — Mode handling for spawn (or deferred — see Phase 2) (~40–120 LOC delta)
- `internal/types/effects.go` — Process is a known mode-parameterised effect (~20 LOC)
- `cmd/ailang/prompts/v1.1.0.md` — Teach Process modes (~40 LOC)
- `CHANGELOG.md` — Entry under v1.1.0

## Examples

### Example 1: Deterministic test using `mocked`

```ailang
module myapp/tests/git_branch_test

import std/process (exec)

-- Test under Process[mode=mocked]: replays a recorded transcript instead of
-- shelling out to a real `git` binary. Result is byte-identical across hosts.
export func test_currentBranch() -> bool ! {Process[mode=mocked]} {
  let r = exec("git", ["rev-parse", "--abbrev-ref", "HEAD"]);
  match r {
    Ok(out) => out.stdout == "main\n",
    Err(_)  => false
  }
}
```

### Example 2: Recording fixtures

```bash
# First run: capture transcripts
ailang run --caps Process --process-mode recorded tests/git_branch_test.ail

# Subsequent runs: replay deterministically, no host crossing
ailang run --caps Process --process-mode mocked tests/git_branch_test.ail
```

### Example 3: M-ENTROPY envelope pinning a test module

```yaml
entropy:
  behavioral:
    effects:
      Process:
        modes: [mocked]    # this module may NOT call live subprocesses
```
A module under this envelope that contains `! {Process[mode=live]}` fails to compile with `EntropyViolation: Process[mode=live] not in envelope`.

## Success Criteria

- [ ] `Process[mode=mocked]` blocks live execution at runtime — verified by a unit test that asserts no `exec.Cmd` is constructed
- [ ] `Process[mode=recorded]` writes transcripts to the trace under a stable identity key — verified by reading back the trace
- [ ] Round-trip: a transcript captured under `recorded` is byte-identical to the output served under `mocked`
- [ ] M-ENTROPY envelope can constrain Process modes per-module — at least one negative test (`live` denied) and one positive (`mocked` allowed)
- [ ] Bare `! {Process}` continues to behave exactly as today — no regression in `process_demo.ail` or any v0.8.1 test
- [ ] At least one runnable example (`process_modes_test.ail`) demonstrates the deterministic-test workflow
- [ ] All existing Process tests passing
- [ ] `ailang prompt` teaches the three modes
- [ ] CHANGELOG entry references both M-EFFECT-REFINEMENT and M-PROCESS-EXEC

## Testing Strategy

**Unit tests:**
- Mode resolution from effect-row context
- Transcript write under `recorded`: identity key derivation, schema correctness
- Transcript read under `mocked`: hit, miss (TranscriptMissing error), corrupted data
- Back-compat shim: bare `Process` desugars to `Process[mode=live]`

**Integration tests:**
- Recorded → mocked round-trip on `git rev-parse`, `uname -s`, `echo` fixtures
- M-ENTROPY envelope rejection of disallowed modes
- ProcessError surfaces correctly in all three modes

**Cross-platform:**
- Recorded transcripts captured on macOS replay correctly on Linux (proves `mocked` doesn't depend on host)

## Deferred Decisions

- **`spawnProcess` interactive replay strategy** — agent may choose between (a) deferring spawn-mode entirely to v1.2, (b) recording stream events as ordered transcripts. Phase 2 records the choice.
- **Identity-key normalisation** — agent may choose lowercase/canonicalise paths or keep verbatim, but must document the choice.
- **Whether `mocked` errors include the resolved transcript key in the user-visible error** — agent's call; structured error payload always carries it.

## Non-Goals

**Not in scope (separate Process effect extensions):**
- **Streaming stdout/stderr callbacks** — orthogonal to mode; future Process API extension.
- **Per-call env vars** — orthogonal; future Process API extension.
- **Per-call cwd** — orthogonal; future Process API extension.
- **MCP tool calls and HTTP RPC** — these go through the Net effect; M-EFFECT-REFINEMENT covers them via `Net[mode=recorded|live]`.
- **Generalising mode discovery** — modes are declared by the contract registry, not user-extensible.

## Timeline

**Sprint 1** (~16 hours):
- Phase 1 — synchronous exec mode plumbing, transcript read/write, round-trip test

**Sprint 2** (~12 hours):
- Phase 2 — M-ENTROPY hook, spawnProcess decision, additional examples

**Sprint 3 (partial)** (~8 hours):
- Phase 3 — docs, prompt update, CHANGELOG

**Total: ~36 hours across 1–2 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Identity-key choice too coarse → unrelated commands collide; too fine → trivial cwd/env changes break replay | High | Design Freeze must rule. Default proposal: `(cmd, args, cwd-relative-to-module, sorted-env)`. Provide a `--process-key-debug` flag that prints the key for any exec call. |
| `mocked`-on-miss falls through to `live` and silently leaks host state | High | The chosen behaviour MUST be `error`. Make falling-through impossible — enforce in the handler with a single unconditional code path. |
| Streaming `spawnProcess` mocked replay is fundamentally awkward | Med | Defer spawn modes to v1.2 if Phase 2 stretches. Phase 1 + sync exec already delivers ~80% of the value. |
| Transcript bloat in trace files for any test that runs many subprocesses | Med | Add a `--process-transcript-cap` flag (default 1MB total) and a structured warning when exceeded. |
| Cross-platform transcript portability (line endings, locale-dependent output) | Med | Document explicitly: transcripts are platform-pinned by default; portability is the user's responsibility. Don't try to normalise. |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_8_1/m-process-exec.md](../../implemented/v0_8_1/m-process-exec.md) — the existing Process effect; this doc extends, doesn't replace
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — capability discipline that `recorded` mode may extend

**Planned (load-bearing):**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — parent framework; **must land before this milestone**
- [design_docs/planned/v0_13_0/m-cryptorand.md](../v0_13_0/m-cryptorand.md) — pilot of mode-aware effects (Rand)
- [design_docs/planned/v0_16_0/m-taint-types.md](../v0_16_0/m-taint-types.md) — adjacent (effects vs labels orthogonality applies; Process modes don't introduce labels)

**External:**
- None directly. Erik Meijer's *Guardians of the Agents* paper sets up the broader replay/verification context but doesn't address subprocess replay specifically.

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [M-PROCESS-EXEC §Axiom Scoring (A2)](../../implemented/v0_8_1/m-process-exec.md) — the original A2 = +0.5 score that this milestone closes to +1
- [M-EFFECT-REFINEMENT §Replay contract taxonomy](../v1_0_0/m-effect-refinement.md) — the framework table this milestone extends with a Process row

## Future Work

- **Process streaming transcripts** for `spawnProcess` — separate milestone; needs a streaming trace schema.
- **Process per-call env / cwd** — orthogonal Process API extensions.
- **Cross-mode harness diff** — a CI tool that runs a test under `live` and `mocked` and reports drift in the transcripts.
- **MCP tool-call replay parity** — verify `Net[mode=recorded]` covers MCP traffic with the same fidelity as Process recording covers subprocesses.

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30
