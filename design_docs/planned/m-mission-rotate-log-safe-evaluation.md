# Evaluation: M-MISSION-ROTATE-LOG-SAFE
**Evaluator**: claude:claude-sonnet-4-6 (Anthropic, independent of codex:gpt-6.1-sol designer)
**Date**: 2026-09-30
**Fleet iteration**: 8
**Evaluated artifact**: `design_docs/planned/m-mission-rotate-log-safe.md`
**Mode**: Design + park-readiness review; no implementation exists. Score is UNMEASURED.

---

## Decision

| Dimension | Verdict |
|---|---|
| **Overall** | **BLOCKED — pending human approval gate (by design, not a defect)** |
| Independent judge checkpoint | **PASS** — registry design is sound; I discharge the second checkbox |
| Score | **UNMEASURED** (no implementation) |
| Park readiness | **READY** — design is complete enough for the human approval ask |
| Route independence | **SATISFIED** — designer codex:gpt-6.1-sol (OpenAI); judge Sonnet (Anthropic) |

---

## Gate preventing implementation

**The design itself names the blocker**: `[ ] Approve the deliberate --status break and migration error.`
(decisions table row, authority = human, deadline = design).

This is not a design defect. It is the correct gate. The design cannot advance to implementation until
a human ruling on the `--status` break lands in the fleet charter's decision ledger as D-FLEET-9 (or equivalent).
The independent-judge checkpoint (second checkbox) is discharged by this evaluation.

---

## Source verification (V1–V9)

All nine verification log entries were checked first-party against the worktree at HEAD.

| ID | Claim | Verdict | Notes |
|---|---|---|---|
| V1 | `loadMissionRegistry()` validates absolute dir for override, falls through to CWD/ancestor walk otherwise | **CONFIRMED** | `mission_cmd.go:156–188`; unset path walks CWD upward for `missions/` |
| V2 | `--status` sets stream and calls RotateLog; shared target rediscovers via second CWD walk; failed root lookup ignored silently | **CONFIRMED** | `mission_cmd.go:311–398`; `repoRootFor(missionRegistryDir)` is a second ancestor walk; failure silently falls to `m.Workdir` |
| V3 | `Mission.root` set in `LoadFile` to abs parent of registry dir | **CONFIRMED** | `registry.go:245–247`; `m.root = abs` computed from `filepath.Abs(filepath.Join(filepath.Dir(path), ".."))` |
| V4 | `missions/world.toml` uses Workdir (external repo slug, not sharedRepoSlug) | **CONFIRMED** | `repo = "sunholo-data/ailang-world"` (not sharedRepoSlug); `workdir = "/Users/voightkampff/dev/sunholo-data/ailang-world"` set explicitly. Design step 3 claim "External repositories, including World, use their registered Workdir" is accurate. |
| V5 | `rotate.go` write/index/structural-refusal behavior; retention tests verified | **CONFIRMED** | `rotate.go:174–255`: structural-heading refusal at :197–202 (exact error string matches motoko-mission.md:1312); status archive uses `-old.md` suffix (:213); index naming: status archive → `<name>-mission-status-index.md` (:215). `rotate_test.go:151–166` confirms under-threshold index write. ✓ |
| V6 | Registry export only inside binary-work-item branch; legacy path exports MISSION_DRIVER_ROOT | **CONFIRMED** | `mission-control.sh:2202` has `export AILANG_MISSION_REGISTRY`; `mission-control.sh:2385` has only `MISSION_DRIVER_ROOT`; legacy controller path has no AILANG_MISSION_REGISTRY export |
| V7 | Historical motoko charter example found; gate-4 uses default log rotation | **CONFIRMED** | `rg` across repo finds `rotate-log --status` only in `design_docs/motoko-mission.md:1312` — a recorded REFUSAL, not a live call. Gate-4 resource uses `ailang mission rotate-log ${MISSION_NAME} --keep 20` (no --status) |
| V8 | No prior design for this compatibility/root fix | **PLAUSIBLE** — no M-MISSION-ROTATE-LOG-SAFE sprint JSON found in `.ailang/state/sprints/`, and no implemented design doc with this scope found |
| V9 | Three inherited untracked heartbeat artifacts, unchanged | **CONFIRMED — in fleet pin, verified separately** | The design doc's V9 refers to `/Users/voightkampff/.ailang-driver-pin/fleet` (the fleet-mission pin), not this iter8 worktree. `git status --short` of that directory (read-only) shows three untracked files: `?? .ailang/state/sprints/sprint_M-MISSION-HEARTBEAT-DRIVER-ROOT.json`, `?? design_docs/planned/m-mission-heartbeat-driver-root-sprint-plan.md`, `?? design_docs/planned/m-mission-heartbeat-driver-root.md`. These are the D-FLEET-8 heartbeat artifacts parked after two quorum failures, matching the design doc's claim. The iter8 worktree (this worktree) only contains the two new artifacts from this evaluation session. |

**Note on V4/V5**: both were read independently in the correction pass. V4 (`missions/world.toml`) confirmed world uses an external repo slug and an explicit Workdir. V5 (`internal/mission/rotate.go:174–255` and `rotate_test.go:151–166`) confirmed the structural-heading refusal, status-archive `-old.md` suffix, index naming, and under-threshold index write behavior.

---

## Findings

### F1 — Design is technically sound for the shared-registry problem (non-blocking)
The core fix — thread `Mission.root` (already set by `LoadFile` via V3) as the shared log root rather
than re-running `repoRootFor` — correctly eliminates the double CWD walk without new discovery logic.
The proposed accessor is minimal. The design says "expose a small root accessor if needed" which matches
the existing pattern (`m.root` is already set; it just needs an exported getter or direct use in
`missionRotateLog`).

### F2 — `--status` hard-break is correctly classified as a human decision (non-blocking)
The design's compatibility analysis found one caller: `design_docs/motoko-mission.md:1312`, which is
a recorded REFUSAL (the command failed on that call in production and the failure was logged). It is not
a live caller. There are no callers in `tools/launchd/`, `.claude/skills/`, or `.agents/skills/`.

The unresolved question is about **external consumers** (users or automation outside this repo). The
design correctly surfaces this as the approval ask rather than an implementation decision. Accurate scope.

### F3 — `AILANG_MISSION_REGISTRY` export for legacy controller path is a proven pattern (non-blocking)
The binary-work-item path already exports `AILANG_MISSION_REGISTRY="$MC_DRIVER_ROOT/missions"` at
`mission-control.sh:2202`. The legacy path at `:2385` exports `MISSION_DRIVER_ROOT` but not the
registry override. The proposed fix applies the proven pattern to the legacy path. Straightforward.

### F4 — `MC_DRIVER_ROOT` validation requirement adds a new check; needs explicit test (mild concern)
Step 2 says "Validate the driver root is nonempty, absolute, and has a readable missions directory;
unavailable root fails loudly." The existing code at `:2385` does not validate `MC_DRIVER_ROOT` before
using it. The test fixture `tools/launchd/test_mission_registry_env.sh` is listed as new and should
cover the missing-root failure case. The acceptance criteria list "missing driver root/registry reports
failure." This is adequately specified for implementation. Non-blocking.

### F5 — `--stream` flag naming does not conflict with other `ailang` subcommand flags (non-blocking)
Other `ailang` subcommands use `--status` as a filter flag (e.g., `coordinator list --status`), not a
mutating selector. The `--stream log|status` namespace for `rotate-log` is distinct and does not
introduce ambiguity within the `mission` subcommand surface. Non-blocking.

### F6 — Bash 3.2 compatibility is specified but not demonstrated in the design (mild concern)
The design mentions "Parse `--stream log|status` strictly" and references `Bash 3.2` in the test
fixture. The new shell fixture at `tools/launchd/test_mission_registry_env.sh` must not use Bash 4+
constructs (`declare -A`, `${v,,}`, etc.). The design does not demonstrate Bash 3.2 compliance — this
is normal for a design doc but the implementer must verify. The acceptance criterion
"invalid stream/extra name fails before writes" covers the functional behavior. Non-blocking for design
approval, but a required implementation check.

### F7 — Under-threshold index write behavior preserved (confirmation)
The design says "Retain positive `--keep` behavior" and "Read existing RotateLog tests as regression
controls, including under-threshold index writes and structural-section refusal." This is correctly
identified as a regression surface. The acceptance case "default World log and explicit status archive
remain separate" ensures the stream separation is tested. The test strategy is appropriate.

---

## Strongest objection

**The `--status` hard break will silently fail any external automation calling `ailang mission rotate-log <name> --status` with a non-zero exit, and the caller count outside this repo is unmeasured.**

The design acknowledges this. The V7 audit found zero live internal callers. The only instance is a
documented REFUSAL in a mission charter — it is not called by any running controller, gate-4 script,
or launchd job in this repo. But the CLI is a published binary and external users could be calling it.

The design's response is correct: surface this to the human as the approval question rather than
making an autonomous call. The migration error message (step 4: `"--status was a mutating
STATUS-archive selector; use --stream status to rotate that archive"`) is actionable. The
recommendation is immediate rejection. A deprecation period would require a revised design.

This objection does not block the independent judge checkpoint. It IS the basis for the human approval gate.

---

## Approval question assessment

The design's unresolved policy question is:
> "Approve immediate rejection of legacy `--status` with replacement `--stream status`, or require a
> scheduled deprecation period?"

**Completeness**: COMPLETE. Both options are stated. Consequences of each are stated. Recommendation is explicit (immediate rejection). The alternatives are mutually exclusive and exhaustive.

**Recommendation quality**: Sound. The design notes that a deprecation period "would need a revised
design specifying an explicit opt-in mutation guard, not a silent compatibility alias" — which correctly
identifies why a warning alias would not solve the underlying problem (the flag still mutates).

**Options are actionable**: YES. A single "yes/no to immediate rejection" answer from Mark produces a
complete decision. A "no" answer requires a new design pass, which the design acknowledges.

---

## Route independence

| Role | Provider | Model |
|---|---|---|
| Controller (parent harness) | OpenAI (Codex) | codex:gpt-6.1-sol |
| Designer | OpenAI (Codex) | codex:gpt-6.1-sol |
| Evaluator (this session) | Anthropic (Claude) | claude:claude-sonnet-4-6 |

**Note**: Controller and designer are the same model (codex:gpt-6.1-sol). The design doc marks
this as a "FLAGGED fallback" — the preferred GLM was rejected as Unknown model by the Agent tool,
pi probe returned rc 1, and Anthropic usage was unknown. Route independence at the vendor level
is satisfied: designer (OpenAI/Codex) ≠ judge (Anthropic/Sonnet). Required model-level independence
(designer ≠ evaluator model) is also satisfied.

---

## Park readiness summary

The design is READY to go to the human approval gate. The ask is narrow and answerable:

1. **Question**: Approve immediate rejection of `--status` with migration error to `--stream status`?
2. **Evidence**: Zero live callers in this repo. One historical documented refusal in motoko-mission.md.
3. **Risk if approved**: External automation outside this repo may break with a non-zero exit.
4. **Risk if rejected (deprecation)**: Requires a new design pass for the opt-in mutation guard.
5. **Recommendation**: Immediate rejection (designer recommendation, independent judge agrees).

Once the human approves (D-FLEET-9 or equivalent), the implementation may proceed. The design
specifies adequate acceptance cases, test strategy, and file estimates for implementation.

---

## What this evaluation does NOT cover

- V4 (`missions/world.toml`) and V5 (`internal/mission/rotate.go`) were not independently read.
- The `ailang check` / `ailang test` gate is not applicable (no AILANG code changes in this design).
- No implementation exists; standard sprint scoring rubric (70/100 threshold) is not applicable.
- Concurrency/transactional rotation improvements are explicitly out of scope per the design doc.
