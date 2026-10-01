# M-MISSION-ROTATE-LOG-SAFE: Resolve the Registry Once and Name Archive Mutation Explicitly

**Status**: Planned — independent review and approval pending
**Created**: 2026-09-30
**Target**: Next harness release (current std/VERSION v0.49.0)
**Priority**: P0, human-ranked fleet queue #2
**Estimated**: 1 day including synthetic integration tests and review
**Dependencies**: Independent judge verdict; compatibility decision approved
**Planner-Lane**: codex:gpt-6.1-sol
**Scope**: Feature/semantics; design only. No implementation authorized by this artifact.

## Routing evidence

Fleet iteration 8 designer rotation wrapped to Codex: **FLAGGED fallback**. Requested GLM was rejected by the Agent tool as Unknown model; pi probe returned rc 1 with extension conflict; Ollama and OpenRouter were over ration; Anthropic usage was unknown. These are controller-supplied route facts, not independently measured here. Generator and independent judge must differ; a controller-only verdict does not discharge review. D-FLEET-8 heartbeat work remains parked and untouched.

## Problem and goals

The paired tickets `mission:rotate-log-registry-cwd` and `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive` concern one command. Fleet's dated human queue reports 3 + 1 lost slots. The existing CLI can load a registry from an explicit absolute directory but rediscover the shared-repository log root from CWD, silently using Workdir if discovery fails. From a product checkout with no registry, legacy controller calls cannot locate the shared registry without explicit configuration. `--status` selects a STATUS archive and invokes the write path; it is not a report. Even an under-threshold rotation rewrites the index.

Success means one resolved registry determines shared-repo targets, legacy controller children receive the pinned registry, and archive mutation has an explicit name. Failures before mutation leave synthetic target files byte-identical.

## High-impact decisions and design freeze

| Decision | Recommendation | Authority | Deadline | Cost |
|---|---|---|---|---|
| Ambiguous `--status` compatibility | Reject with migration error; use `--stream status` for the old mutation | human | design | medium |
| Registry discovery | Explicit env override, then existing CWD/ancestor discovery; driver supplies pinned root explicitly | agent within approved design | implementation | low |
| Canonical shared log root | Parent of the successfully resolved registry, never a second CWD search or Workdir rescue | human review | design | medium |

- [ ] Approve the deliberate `--status` break and migration error.
- [x] Independent Anthropic Sonnet judge confirms registry authority and target mapping; this does not discharge the separate BLOCKED quorum or human approval gate.

A rename is recommended over adding a read-only report: it resolves the misleading flag without introducing a second rotation planner. The existing command verb already declares mutation. Rejected `--status` must error before loading or writing anything, even alongside `--stream status`.

## Solution design

1. Resolve registry paths once in `loadMissionRegistry`. Preserve explicit `AILANG_MISSION_REGISTRY` precedence and its absolute existing-directory validation. Preserve attended CWD/ancestor discovery when no override exists. On failure, name the attempted locations and explain the absolute override; do not guess `$HOME`, binary installation roots, `AILANG_DRIVER_SRC`, or another checkout. Preserve the loaded registry's absolute origin using its existing Mission.Path/root provenance rather than duplicating discovery; expose a small root accessor if needed.
2. Export `AILANG_MISSION_REGISTRY="$MC_DRIVER_ROOT/missions"` for the legacy controller path before launching it, as already done for the binary iterate branch. Validate the driver root is nonempty, absolute, and has a readable missions directory; unavailable root fails loudly. Do not use the lagging source clone. This export selects the driver registry for its child process tree; no daemon reload or installed config edit is required.
3. In rotate-log, sharedRepoSlug missions use the root of the loaded registry. External repositories, including World, use their registered Workdir. Keep the two target filenames explicit: `design_docs/<name>-mission-log.md` (default) and `design_docs/<name>-mission-status-archive.md` (`--stream status`). Synthetic evidence confirms default World already selects its log correctly; preserve that behavior rather than claiming a default-stream bug. Missing selected input fails; do not switch streams or repositories. Preserve existing archive/index naming, structural-heading refusal, and RotateLog mechanics.
4. Parse `--stream log|status` strictly, reject invalid or repeated stream values and multiple positional names before mutation, retain positive `--keep` behavior, and reject `--status` with: `--status was a mutating STATUS-archive selector; use --stream status to rotate that archive`. Update CLI help to declare writes and this migration. No legacy alias with a warning: that would preserve the misleading mutating spelling.

## Compatibility and conflict surface

Default `rotate-log <name> --keep N` remains mutating log rotation. `--stream status` retains the old archive-rotation behavior, including `-old.md` and status index output. Old `--status` commands deliberately stop with a nonzero exit and migration guidance. Existing callers found in repository search: a historical motoko charter example; both gate-4 resources call default rotation, so they need no behavioral edit. External consumers may need migration; this is the unresolved approval choice.

The shared registry loader is used by other mission subcommands: preserve its return signature where practical and existing precedence, validation, and errors; do not change their target behavior. This design touches CLI/harness code, not compiler or language semantics. Concurrency/transactional rotation improvements are outside this paired ticket.

## Files

| File | Change estimate | Responsibility |
|---|---:|---|
| cmd/ailang/mission_cmd.go | ~60 changed lines | Single registry authority, rotate target resolution, strict stream parsing and help |
| internal/mission/registry.go | ~10–20 lines if needed | Read-only accessor for existing absolute origin |
| tools/launchd/mission-control.sh | ~10–20 lines | Legacy controller export and root validation |
| cmd/ailang/mission_rotate_test.go (new) | ~180 lines | Synthetic end-to-end path and flag regression cases |
| tools/launchd/test_mission_registry_env.sh (new, if separate fixture useful) | ~60 lines | Legacy and binary child export, missing-root failure |
| make/test.mk | ~1 line if new shell suite | Wire driver fixture into existing test target |
| changelogs/unreleased/2026-09-30-mission-rotate-log-safe.md (new) | ~5 lines | Explicit flag migration and CWD fix |

## Implementation plan and validation

First add CLI synthetic fixtures and root-export coverage, then make registry/target and flag changes, then update help/migration notes. Use t.TempDir and temporary synthetic TOML registries/workdirs only. No command may rotate real mission logs. Refactor a small testable command helper if needed instead of mutating global live process configuration in parallel tests.

Acceptance cases: explicit registry A wins over CWD registry B; shared mission writes only A's docs; external World writes only its synthetic Workdir; legacy child sees the pinned registry when CWD has none; invalid explicit directory does not fall back to B; missing driver root/registry reports failure; default World log and explicit status archive remain separate; rejected `--status` changes neither input, archive, nor index; invalid stream/extra name fails before writes; status stream still retains all bodies/index naming. Read existing RotateLog tests as regression controls, including under-threshold index writes and structural-section refusal.

Run focused Go tests for CLI/mission, the new Bash 3.2 fixture, and make test-launchd-drivers. All relevant tests pass; help and release documentation updated. Apply the mission-loop-change done-gate during execution, with healthy/degraded dry runs pinned to the tested SHA and no mid-iteration reload. Implementation and landing remain gated on independent judge and approval.

## Verification log

| ID | Evidence read/run | Finding |
|---|---|---|
| V1 | cmd/ailang/mission_cmd.go:155–186 | Override validates absolute existing directory; unset path searches CWD/ancestors |
| V2 | same file:311–400 | `--status` sets stream and calls RotateLog; shared target rediscovers literal missions and ignores failed root lookup |
| V3 | internal/mission/registry.go:83–85,233–282 | Mission retains Path and absolute root from the loaded TOML; Registry holds Missions |
| V4 | missions/world.toml | External repo slug and registered Workdir; unlike sharedRepoSlug, uses Workdir |
| V5 | internal/mission/rotate.go:174–255; rotate_test.go:151,295–345 | Writes/index behavior, structural refusal, status naming and retention tests verified by body reading |
| V6 | tools/launchd/mission-control.sh:2194–2204,2375–2390 | Registry export currently only inside binary-work-item branch; legacy path exports MISSION_DRIVER_ROOT |
| V7 | repository rg for rotate-log + --status; both gate-4 resources | Historical motoko example found; gate-4 uses default log rotation |
| V8 | rg across planned/implemented + ailang docs search (SimHash) | Existing workbench defines registry and harness doc defines fleet ownership; runtime contract delegates rotation. No located design defines this compatibility/root fix. SimHash top results are unrelated; neural search not run against over-ration Ollama lane |
| V9 | git status --short | Three inherited untracked heartbeat artifacts parked; unchanged |
| V10 | tools/launchd/mission-control.sh:40–49,2202,2385 | MC_DRIVER_ROOT is assigned from the invoked driver's location before changing CWD; binary branch already exports exactly "$MC_DRIVER_ROOT/missions"; MISSION_DRIVER_ROOT is exported from MC_DRIVER_ROOT. Reviewer premise that MC_DRIVER_ROOT is undefined/necessarily empty is refuted; exceptional resolution failure still requires validation |
| V11 | Controller synthetic artifact /tmp/fleet-iter8-repro.json, override_A_from_B + controls | rc 0 with override registry A and CWD B mutated B's v1 log (3→1); A remained 3 entries. Reproduces provenance bug |
| V12 | Same artifact, foreign_without_registry | Foreign CWD without override returned rc 1: failed to read mission registry missions; no guessed source path |
| V13 | Same artifact, default_world_log | rc 0 targeted W/design_docs/world-mission-log.md, retaining 1 of 3; default World status-selection allegation not reproduced |
| V14 | Same artifact, legacy_status_mutates + controls | rc 0 targeted W/design_docs/world-mission-status-archive.md; 3→1 entries with old/archive and status-index writes |
| V15 | Controller fixture correction report | Initial synthetic registry collided boot offsets and validation rejected it; corrected offsets 1 versus 0 yielded V11–V14. Initial rejection is not evidence for target behavior |
| V16 | scripts/changelog_fold.sh:10,57–67; changelogs/unreleased/README.md | Per-change dated fragment is the supported release-note surface; shared active [Unreleased] must stay empty between releases |

No AILANG language claims or new diagnostic codes are introduced. Source read only; unsafe live rotation was not used as verification.

## Related documents and duplicate gate

M-MISSION-LOOP-WORKBENCH (`design_docs/planned/v0_36_0/m-mission-loop-workbench.md`) supplies the existing registry model; M-HARNESS-MISSION-LOOP (`design_docs/planned/m-harness-mission-loop.md`) supplies scope/routing. M-MISSION-RUNTIME-CONTRACT already delegates rotation to this command but does not specify this root/flag repair. The design-quorum mission-log-path triage concerns optional reviewer logs in another repo, distinct from rotating mission records. This document changes neither registry ownership nor role routing; no duplicate implementation is proposed.

## Axiom compliance

The twelve language axioms below are required by `.agents/skills/design-doc-creator/resources/design_doc_structure.md:573–615` (Axiom Compliance template), not invented for this design. The separate mission-principle assessment follows the actual reviewer prompt in `internal/mission/quorum/reviewer.go:106`.

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | +1 | One resolved registry controls shared target |
| A2 Replayability | 0 | Record format unchanged |
| A3 Effect Legibility | +1 | Mutating archive selector explicitly named |
| A4 Explicit Authority | +1 | Configured registry retained; missing authority errors |
| A5 Bounded Verification | +1 | Synthetic fixtures |
| A6 Safe Concurrency | 0 | Rotation concurrency unchanged |
| A7 Machines First | +1 | Strict flags and actionable errors |
| A8 Minimal Syntax | 0 | No language syntax |
| A9 Cost Visibility | 0 | No billing changes |
| A10 Composability | +1 | Reuses registry provenance |
| A11 Structured Failure | +1 | Unavailable root and ambiguous selector fail loudly |
| A12 System Boundary | +1 | Driver origin separated from product Workdir |

Net +8; A1/A3/A4/A7 hard violations absent. Axiom score is an author claim, not independent approval.

### Five mission principles measured by quorum

| Principle | Assessment |
|---|---|
| Minimal frozen core | Respected: changes stay in mission CLI/registry and launchd harness; language and motoko cores remain untouched |
| Route-to-extension bias | Respected: this is a fleet-owned mission harness defect, repaired at its existing CLI/driver seam; it does not add a core-floor capability or extend the shared pi evaluation harness |
| No silent fallbacks | Respected: retain explicit registry provenance; invalid override, unavailable driver root, missing selected file and ambiguous old flag fail visibly; never rescue with guessed paths or Workdir for shared missions |
| Bounded waits | Respected: adds local resolution/validation only, no provider call, poll loop, or unbounded wait; synthetic verification remains finite and existing bounded done-gate applies |
| Deterministic behavior | Improved: one resolved registry fixes the shared target regardless of conflicting CWD; external Workdir and explicit stream govern their own targets |

### Review history and disposition

First quorum returned rc 3: Gemini rejected on driver-root evidence and mission-principle coverage; Sonnet was absent due to quota. This protocol revision supplies V10 and the five-principle table. The MC_DRIVER_ROOT premise is refuted by source; the missing assessment was a documentation gap and is corrected. Neither correction converts the failed quorum into approval. Human compatibility approval and a fresh independent disposition remain required.

## Risks and deferred decisions

The intentional old-flag break may affect external automation: require human approval and exact migration guidance. Wrong registry provenance could rotate another checkout: synthetic conflicting-root cases must fail the old implementation. Global test env/CWD races: keep CLI fixtures serial or dependency-inject context. Implementer may choose helper names and fixture layout. No model routing, ration thresholds, heartbeat mirrors, registry schema redesign, or rotation transaction rewrite is included.

## Unresolved policy question

Approve immediate rejection of legacy `--status` with replacement `--stream status`, or require a scheduled deprecation period? Recommendation: immediate rejection prevents the misleading mutation from continuing. A deprecation period would need a revised design specifying an explicit opt-in mutation guard, not a silent compatibility alias. Controller reports a successful Sonnet subscription provider probe (rc 0), but the subsequent quorum reviewer was absent due to quota. Agent-tool Sonnet is unsupported. Judge availability must be checked for the actual review; a successful probe does not override an absent verdict. Execution must not proceed on controller-only review.

Artifact handling: controller safely moved the worktree to `/Users/voightkampff/.ailang-driver-pin/fleet-iter8-rotate-log`; this revision changes only this design file. Scaffold script was not run because it would create an additional repository artifact outside this single-file ownership; the documented structure and related-document gate were applied directly.

## Controller disposition — iteration 8 (2026-09-30)

Independent `claude:claude-sonnet-4-6` review is banked in [evaluation](m-mission-rotate-log-safe-evaluation.md): technical checkpoint PASS, overall BLOCKED on human approval, implementation score UNMEASURED. After the permitted designer revision, quorum round 2 remains BLOCKED: Gemini requires a complete audit of shared `loadMissionRegistry` callers or isolation of stricter resolution to rotate-log; Sonnet quorum seat was absent due to quota. The independent evaluation does not overturn quorum. No execution or formal sprint plan is authorized. D-FLEET-9 asks for the compatibility choice and the next design direction; default is park this paired fix. Full measured reproduction and both quorum outcomes: [evidence](m-mission-rotate-log-safe-evidence.json).
