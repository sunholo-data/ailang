# M-QUORUM-ARTIFACT-DIR-STABLE: Durable, Repo-Scoped Quorum Evidence

**Status**: Planned — designer deliverable, not implementation approval **Target**: Next patch after v0.54.9 **Priority**: P2 (correctness with an explicit-flag workaround) **Estimated**: 1.5 days / 12 hours, including lane verification and buffer **Dependencies**: Existing config Registry and policytool's pinned policy marker (#1552/#1554) **Ticket**: `quorum:artifact-dir-cwd-relative` (V1, 2026-09-26); fleet iteration 38 **Created / updated**: 2026-10-14

## Problem Statement

`internal/mission/quorum/artifact.go:14` defines the relative constant `.ailang/state/mission-quorum`. `cmd/ailang/design_quorum.go:41` uses that constant as the flag default and `:97` passes it to `WriteJSONArtifact`; `artifact.go:40–44` creates the directory relative to the process CWD. `git check-ignore -v` confirms `.gitignore:105` excludes these artifacts from commits.

The driver pins its execution root to a committed worktree; sprint/record trees are removed after merge. Evidence left there is absent from subsequent readers, even when the reviewed document survives. This is a writer/reader agreement defect across worktrees and separate clones, not a missing quorum review.

First-party measurements on 2026-10-14 reproduce the controller's numbers:

| Location | JSON files | Lines | Consequence |
|---|---:|---:|---|
| This designer worktree's relative directory | absent | absent | Gate 2 sees no local evidence |
| `~/.ailang-driver-pin/fleet/.ailang/state/mission-quorum` | 5 | 377 | Stranded in a pin |
| `~/dev/sunholo-data/ailang/.ailang/state/mission-quorum` | 164 | 8190 | Invisible from other trees |
| `~/.ailang/state/mission-quorum` | 13 | 981 | Existing explicit-flag workaround |
| `~/.ailang/state/mission-quorum-world` | 2 | 154 | Existing World workaround |

Both Gate 2 copies contain the relative lookup at `resources/gate-2-pick.md:139` and a historical relative-path measurement at `:215` (ticket's prose location has shifted from `:208`). They are byte-identical. A reviewed document can therefore read as UNREVIEWED, re-spending quorum or misrouting. V1 iteration 261's four reviews over two rounds with zero reachable artifacts is ticket evidence, **UNVERIFIED** historically in this session. Main/pin removal frequency is also **UNVERIFIED** live; the inspected pin helper confirms the committed-worktree mechanism.

## Goals / Non-Goals

- For driver-managed host missions with an exported, validated absolute override and **no raw policy marker**, writers and Gate 2 agree across worktrees/clones. Marked mission controllers fail closed; this guarantee does not cover sandboxed writers.
- Attended, no-export writers invoked at the current repo root and Gate 2 in that same state agree on `<repo-root>/.ailang/state/mission-quorum`, without `MISSION_REPO`.
- Preserve sandbox-relative default writes, explicit non-default flag paths, JSON schema, filename slugging and collision protection.
- Legacy import is an **out-of-scope follow-up**, not an implementation deliverable. No routing, reviewer, quota, billing, replication, schema/freshness or deletion changes.

## High-Impact Decisions

| Decision | Why high impact | Chosen by / deadline | Change cost |
|---|---|---|---|
| Opt-in Registry env; relative constant frozen | Preserves policytool's write gate | recommendation / design approval | high |
| Repo-scoped host directory; marked missions error | Prevents cross-repo matches and silent path divergence | recommendation / design approval | medium |
| Defer additive import; no permanent read-both | Keeps a P2 durability fix bounded | recommendation / design approval | low |

**Design freeze:** raw nonempty policy markers always confine default writes; never trim or validate the marker to relax suppression. Namespace by repo identity, not mission or clone path. Implementation still requires approval → planning → execution gates.

## Solution Design

### Recommendation and alternatives

Choose **(a)**: register `MISSION_QUORUM_ARTIFACT_DIR`, exported by the driver as an absolute host path; the quorum package resolves its default through this variable, and Gate 2 consumes the same variable with the explicit decisions below; marked active missions error.

Choose a **shared directory for all missions of one repository**, not per-mission: v1/docs/motoko/fleet review repo-scoped AILANG docs, including across separate clones. Namespace defaults by `MISSION_REPO` after profile sourcing:

| Repository | Host default |
|---|---|
| `sunholo-data/ailang` | `$HOME/.ailang/state/mission-quorum` |
| `sunholo-data/ailang-world` | `$HOME/.ailang/state/mission-quorum-world` |
| Other validated `owner/repo` | `$HOME/.ailang/state/mission-quorum-repos/<owner>/<repo>` |

The first two intentionally retain existing workaround directories. Validate two nonempty safe components; reject `.`/`..`, traversal and extra slashes. Require an absolute HOME for default construction. A deliberate nonempty absolute env override wins; operators must use the same override for all missions of that repo. An empty override selects the table. No directory is created merely to resolve or log it.

**(b) loses:** git-common-dir identifies one clone, not all clones of the same repo; bare repos need another convention. More seriously, unconditional resolution to the main checkout would move policytool's default write outside its checked sandbox. **(c) loses:** a reader-only change cannot recover writes into deleted trees and leaves future writers and readers disagreeing. Read-both postpones the loss until the next source deletion and makes historical lookup depend on clone layout.

### Resolution contract and observable suppression

Keep `quorum.ArtifactDir` byte-for-byte unchanged. Add `ResolveArtifactDir(dir string) (string, error)` in `artifact.go`, called before `MkdirAll`:

| Writer argument / process state | Effective directory |
|---|---|
| Nonempty argument unequal to `ArtifactDir` | Explicit argument, existing behavior |
| Empty/exact sentinel, raw policy marker nonempty | `ArtifactDir` relative to writer CWD; suppress override even if invalid |
| Empty/exact sentinel, marker empty, override nonempty | Validated absolute override |
| Empty/exact sentinel, marker empty, override empty/unset | `ArtifactDir` relative to writer CWD |

Register `EnvMissionQuorumArtifactDir` in `internal/config/mission.go`, default `""`, AreaMission. Provide a raw override getter using `get(...)` and a raw-policy-presence accessor using `get(EnvAgentPolicy) != ""`; do **not** hide the override in a getter, which would lose the information needed to report suppression. No raw `os.Getenv` outside config. Reject relative, literal-tilde, whitespace-only and NUL overrides; preserve spaces in absolute paths. Test NUL with a pure validator (env cannot hold NUL). Invalid unsuppressed overrides error, with zero writes and no local fallback.

For every default resolution with a nonempty override AND marker, emit to stderr: `quorum artifact directory: <absolute CWD/ArtifactDir>; MISSION_QUORUM_ARTIFACT_DIR suppressed because AILANG_AGENT_POLICY is nonempty`. Emit before any write, including for an invalid override; report effective directory and reason, never policy contents. If determining absolute CWD fails, error. Gate 2's standalone policy reader emits the equivalent diagnostic. The existing CLI already prints `quorum artifact: <path>` after successful writing (`design_quorum.go:102`); keep it. Suppression is explicit confinement, not an unnoticed fallback.

The CLI does not expose flag presence: explicit `--artifact-dir .ailang/state/mission-quorum` is also the sentinel. To force local host storage, supply the absolute local path. Flag-presence handling remains an out-of-scope CLI follow-up.

### Driver and Gate 2 reader decisions

Add a bash 3.2 library for namespace resolution and effective-reader lookup. Driver sources it after all profiles/secrets, checks return status explicitly (no `set -e`), validates `MISSION_REPO` and HOME for default construction, exports/logs the absolute result before controller spawn. **Raw nonempty `AILANG_AGENT_POLICY` aborts host mission startup**, naming confinement and effective local directory; do not unset it to escape. Gate 2 repeats this check when `MISSION_CONTROL_ACTIVE=1`, catching downstream changes. Missing mission export is an error; readers never synthesize a host default themselves.

Gate 2 determines the current repo root with `git rev-parse --show-toplevel`; in attended mode it enters that root before both lookup and any subsequent quorum invocation. A standalone policy reader uses sandbox CWD instead (never ascend to a git root).

| Reader state (default quorum invocation) | Directory / result |
|---|---|
| Active mission, marker nonempty or override empty/unset | Wiring/confinement error, not UNREVIEWED |
| Marker empty, override nonempty (mission or attended) | Validate absolute override; use it; invalid is error |
| Attended, marker empty, override empty/unset | `<current-repo-root>/ArtifactDir`; same as writer run there |
| Standalone policy reader, marker nonempty | `<sandbox-CWD>/ArtifactDir`; suppress nonempty override loudly |
| `MISSION_REPO` unset, attended | No namespace construction; local or explicit override row still works |
| `MISSION_REPO` unset, active reader with valid export | Export suffices for lookup; driver initialization still requires valid repo identity |
| `MISSION_DRIVER_ROOT` unset, attended | Source `<current-repo-root>/tools/launchd/lib/quorum-artifacts.sh` |
| `MISSION_DRIVER_ROOT` unset, active mission | Wiring error; no fallback to another skill/helper tree |
| Driver root supplied | Require absolute root and readable helper; invalid/missing helper is error |
| No repo root when attended fallback needs it | Configuration error, not UNREVIEWED |
| Resolved directory absent, or no matching JSON | UNREVIEWED |
| Permission/read/parse error, or path exists but is not a directory | Error, not UNREVIEWED |

Source failures stop the gate. With directory quoted, retain `<doc-id>-*.json`, slug, freshness/verdict rules. Print `quorum artifact directory: <absolute path>` before lookup; use that directory for all subsequent reads. Leave writer flag omitted and keep the same cwd/env for quorum invocation. Include an executable block in Gate 2 and test the block extracted from the actual skill. Clarify `:215` as a historical local-path measurement.

### Policy-marker audit by lane (revision measurement)

The six requested trees contain **85 references in 36 files**, including tests/docs; there are **two production marker setters**, not a proof that every host lane is empty.

| Lane | Marker set? / evidence |
|---|---|
| Claude controller | No driver setter; `mission-control.sh:2383` directly inherits parent env. Local profiles/plists have no reference; inherited contamination is possible. |
| Codex controller | No setter; `:2337` calls `mc_codex_env_args`. Helper forwards `MISSION_*`, excludes this marker; Codex process still inherits parent env, shells depend on its policy. |
| Codex designer/planner/executor recipe | `dispatch/run.go:taskFor` sets no `PolicyPath`. `BuildEnvironment` admits inherited `AILANG_*`; nonempty parent marker survives. Codex daemon explicitly sets filtered env; exec shells also depend on local shell policy. |
| Pi recipe (and pi controller) | Recipe has no `PolicyPath`; executor inherits marker via envpolicy. Controller `:2381` inherits parent directly. Neither guarantees empty without the startup check. |
| Agent-tool roles | Driver exports `agent-tool` role path at `:2258`, sets no marker. Roles run inside controller harness; no independent empty-env guarantee. Observe presence in role shells before activation. |
| `ailang_only` / policytool children | Executor injects `Task.PolicyPath` at `environment.go:163`; policytool replaces/pins marker in `cli_ops.go:394–402`. Profile has no Bash/native file tools; quorum only through allowed `AilangCLI`, whose schema forbids `--artifact-dir`. |
| Other configured coordinator agents | `provider_executor.go:154–165` forwards any configured policy path; `CheckLanePolicy` explicitly permits full profiles. Marker injection is **not exclusive to ailang_only**. |

Current designer process: marker empty/unset. Live presence in other running roles is **UNVERIFIED**; source/config audits are not remote process measurements. Activation requires presence-only observations at controller and each used role shell (no secrets or model probes), plus AC8's actual helper/environment tests. If marked, stop and report; do not count sandbox evidence as durable host evidence.

### Pre-review validation and cost limit

Inspected call order: CLI selects seats at `:84`, calls `RunSeatedQuorum` at `:89`, then `WriteJSONArtifact(*artifactDir, ...)` at `:97`. `RunSeatedQuorum` is an in-package pre-review entry point, but receives **no artifact directory** and returns only `*QuorumResult`, not an IO error. `RunQuorum` has the same limitation. Adding an ambient check there cannot honor explicit non-default paths and would conflate IO failure with review synthesis. There is no sound full writer pre-flight in scope without changing the forbidden CLI handoff; do not redesign quorum result semantics for this P2.

Driver/helper validation rejects malformed mission overrides before controller/reviewer launch. Standalone CLI validation remains at write time. Out-of-scope follow-up: CLI calls resolver after parsing, before `:89`, exits on failure, then passes resolved path to writer; optional writability probing still cannot prevent later filesystem races. At defaults, three primary seats have declared caps **$0.30 × 3 = $0.90** before a late write error. This is not a strict whole-run billing bound: reserve/recalled seats may also run (`seating.go:112–128`), and per-seat budget enforcement is unchanged.

### Policytool safety argument — verified against code

`internal/policytool/cli_ops.go:60,72` assumes the sandbox-relative directory. `:317–322` checks write protection **before** launching; `:329–334` selects sandbox CWD. The `design_quorum` schema at `:108` does **not** admit `--artifact-dir`. `policytool.go:119` calls `childEnv`; `cli_ops.go:394–402` preserves other inherited variables but replaces every policy-marker entry with the host's policy path. `runAilang` at `:421–424` supplies this env/CWD to the unconfined child.

Thus merely exporting an override is **unsafe without the suppression above**. With suppression, every default call through that host still writes exactly the relative path the gate checked, even when a host override is inherited. The marker is a conservative suppression signal, not a new permission; no policy parsing or validity test may relax it. Existing arbitrary explicit library arguments retain their behavior; the mediated CLI cannot supply one. No policytool edit is required. `TestPolicyTool_QuorumArtifactDirInSync` alone only pins a constant: acceptance must also exercise the inherited-env child chain, so an unchanged constant cannot hide a changed effective write destination.

### Rollout and deferred recovery

The legacy importer is **out of scope**, including a report script: attribution, symlink handling and exclusive-copy recovery are a separate operational task, heavier than this P2 fix. Future import must be explicit, host-only, additive, byte-preserving and never overwrite or delete evidence. Do not automatically import or read both trees. Prior measurements found 164 main + 5 pin + 13 host = 182 distinct valid artifacts, including 33 noncanonical doc strings requiring attribution; World has 2 separate files. Sources already deleted stay absent. Report stranded evidence; recovery is not promised.

Activate export/reader together after observing updated binary, synchronized skills, matching effective paths and empty markers in used lanes. Retain the explicit host flag workaround until then. Rollback removes opt-in/export; retain all evidence. Deployed binary reach and World adoption remain **UNVERIFIED**. Real home paths are observation-only.

### Exact touch points and scope

Original anchors were at `b5ca4f6cf`; revision source audit is at `d94759501`.

| In-scope file / anchor | Planned change / approximate LOC |
|---|---|
| `internal/config/mission.go:13,42,65` | Constant, Registry row, raw override/marker accessors; +20 |
| `internal/mission/quorum/artifact.go:14,40` | Preserve constant; resolver, diagnostic and pre-write call; +45 |
| New `internal/mission/quorum/artifact_dir_test.go`, `artifact_policy_test.go` | Resolution and actual host/child boundary tests; ~150 |
| `tools/launchd/mission-control.sh:119,123` | Source pure helper, validate/export/log after sourcing; +10 |
| New `tools/launchd/lib/quorum-artifacts.sh` | Bash 3.2 namespace, reader and lookup helpers; ~80 |
| New `tools/launchd/test_quorum_artifacts.sh`; new `internal/mission/dispatch/quorum_env_test.go` | Hermetic cross-CWD, namespace, attended documented reader and actual environment handoff arms; ~180 + 50 |
| `.claude/skills/mission-control/resources/gate-2-pick.md:139,140,199,215` | Effective path, quoting and historical clarification; ~15 |
| Same `.agents/skills/mission-control/resources/gate-2-pick.md` | Verbatim generated mirror |
| `make/test.mk:70` | Add hermetic shell test to existing launchd suite; +1 |
| `docs/docs/reference/env-vars.md` | Regenerate through `make docs-env` |

Config getter behavior tests belong in quorum tests: `internal/config/*_test.go` is outside the stated allowlist. Source/mirror changes stay within mission skills; use `tools/launchd/sync-agents-skills.sh`, the verified location, not the ticket's nonexistent `scripts/sync-agents-skills.sh`. Do not expand its six-skill scope.

## Milestones / Timeline

**M1 — Resolution and confinement (4h):** Registry/getter, writer resolver, unit and host/child tests; keep default constant/schema untouched. Pass AC1–4 and registry checks.

**M2 — Driver and reader (8h):** Pure shell helpers, marker checks, driver export/logging, Gate 2/mirror, generated env docs, lane verification, suite wiring and independent evaluation. Pass AC5–10. Neither milestone edits CLI or policytool files.

## Acceptance Criteria / Testing Strategy

New Go arms set HOME to `t.TempDir()`; all shell/integration arms use `HOME=$(mktemp -d)` and fixtures beneath it. Stub reviewers; spend no tokens. Run mutants only in disposable evaluator copies. Real home paths are observation-only.

| ID | Test arm / required outcome | Mutation killed |
|---|---|---|
| AC1 | Empty and sentinel arguments, env unset/empty: exact local default; constant remains pinned | Always-home default; changing constant |
| AC2 | Absolute env with spaces: real writes land there; distinct explicit absolute/relative arguments win | Ignore env; env overrides explicit flag; unquoted path |
| AC3 | Relative/tilde/whitespace env plus direct NUL-validator arm: error and zero artifact writes; unwritable destination errors | Silent local fallback; accept invalid env |
| AC4 | Actual policytool host with helper child calling real writer, inherited absolute/invalid override: allowed writes stay in sandbox; deny `.ailang/**` launches zero children; external fixture stays absent. Capture stderr naming suppression/effective path for absolute and invalid overrides; include raw whitespace marker | Remove/trim marker suppression; silence diagnostic; bypass deny check |
| AC5 | Fixture writer in tree A, delete A, reader in B and separate clone fixture finds exact artifact bytes in exported directory | Retain CWD reader/writer; read main clone only |
| AC6 | v1/docs/motoko/fleet map identically; World/other repo cannot match same-slug AILANG fixture; traversal/invalid HOME/relative override fail | Mission-name namespace; global cross-repo pool |
| AC7 | Execute the shell block extracted from the actual Gate 2 skill; absent is empty, unreadable/invalid config is error; missing env/driver root or marked active mission fails wiring check; attended unset driver root uses repo helper | Treat lookup errors as unreviewed; stale skill snippet |
| AC8 | Actual Codex forwarding + dispatch environment builders with fake CLIs capture override AND marker presence; clean host lanes agree; contaminated startup/reader fails; policy children stay confined; `/bin/bash` 3.2, no model/quota probes | Unexported variable; strip `MISSION_*`; remove marked-mission guard; bash 4 construct |
| AC9 | Extract Gate 2 shell block; attended process with override, marker, `MISSION_REPO`, `MISSION_DRIVER_ROOT`, active flag all unset; real writer at same fixture repo root; reader finds exact bytes/path. Repeat absent directory, zero matches and unreadable fixtures | Require mission identity/export for attended use; generate home default; reader/writer CWD mismatch |
| AC10 | Focused Go + shell tests, registry checks, skill sync, boundaries and lint pass; env docs updated; real-directory manifest before/after unchanged or absent | Omitted Registry/suite/mirror; accidental real-HOME fixture |

Run `go test ./internal/mission/quorum ./internal/policytool`, focused config registry tests, the new shell test and `make test-launchd-drivers`; then `/bin/bash tools/launchd/sync-agents-skills.sh --check`, `make check-boundaries`, `make lint`. Full config tests also run on a runner permitting httptest listeners. An independent evaluator must remove suppression and revert the reader path and observe the corresponding arms fail; fixture-vs-fixture agreement alone is insufficient.

## Verification Log

Original-pass commands below were recorded at `b5ca4f6cf`; revision audit commands at `d94759501` are appended. Implementation arms above are **not yet run**.

| Command / inspection | Observed output/count |
|---|---|
| `cat CLAUDE.md`; skill and `resources/design_doc_structure.md` reads; `git status --short`; `git rev-parse --short HEAD` | Required instructions read; initial status 0 entries; `b5ca4f6cf` |
| `nl -ba` writer/CLI, policytool CLI/host/deny/env tests, config mission/Registry | Verified 1 writer default, 1 CLI handoff, 1 relative gate, pinned marker and excluded artifact flag |
| `rg -n 'mission-quorum'` Gate 2 and design-doc skill copies; `cmp` Gate 2 copies | 2 path mentions per Gate 2 copy; 1 path mention per design-doc skill; cmp rc0 |
| `git check-ignore -v .ailang/state/mission-quorum/probe.json` | 1 match, `.gitignore:105`; no probe file created |
| Python read-only `glob('*.json')`, byte line count and SHA256 of sorted `basename + NUL + bytes`, twice | Counts/lines in Problem table; four manifest hashes below identical before/after verification; current-tree dir absent |
| Python `json.loads`, key checks and basename comparison across AILANG main/pin/home | 182/182 valid doc+synthesis; 182 unique names; 0 duplicate/conflicting names; 149 canonical-relative / 21 absolute / 12 other doc strings |
| `ailang docs search --no-github --limit 5 --json 'quorum artifact directory stable'`; second query `'mission driver pin worktree'` (limit 3) | Each corpus 1987 docs, 200 candidates, 0 embeddings; unrelated rank matches read, not neural duplicate scores |
| `rg -l -i 'quorum.*artifact.*(cwd\|relative\|stable\|throwaway)\|artifact-dir-cwd-relative' design_docs/planned design_docs/implemented` | 1 related triage doc (optional human-log destination), distinct from machine artifact storage; neural duplicate coverage **UNVERIFIED**, cache-mutating search omitted |
| `go test ./internal/mission/quorum ./internal/policytool ./internal/config` | Quorum PASS 0.495s; policytool PASS 0.824s; config FAIL: httptest IPv6 bind denied by sandbox, not a verified code regression |
| `go test ./internal/config -run 'TestEveryGetenvNameIsRegistered\|TestNoLiteralGetenvInConfig\|TestGetOrUsesRegistryDefault' -count=1` | PASS 0.217s; 3 selected registry checks |
| `/bin/bash tools/launchd/sync-agents-skills.sh --check`; `/bin/bash --version`; `/bin/bash -n` driver + Codex env helper; `make check-boundaries` | 6 skills synced; bash 3.2.57; syntax rc0; 0 boundary violations |
| Source reads: pin-root identity helper, driver sourcing/export, Codex forwarding, executor envpolicy and dispatch | Repo/driver separation verified; `MISSION_*` admitted in both forwarding surfaces; live propagation **UNVERIFIED** |
| Revision: `rg --hidden --no-ignore -n 'AILANG_AGENT_POLICY|EnvAgentPolicy' tools/launchd scripts internal cmd .pi/extensions tools/pi-extensions` (Python counted output by tree) | References/files: launchd 0/0; scripts 0/0; internal 51/22; cmd 21/10; pi 13/4; pi-tools 0/0; total 85/36. Zero-match rg rc1, no read errors. |
| Revision: inspected all matching production code; `environment.go:160–164`, `cli_ops.go:394–402`, `dispatch/run.go:taskFor`, `envpolicy.go`, `CheckLanePolicy`, controller/helper/Codex spawn code | 2 actual production marker setters: executor `env.set`, policytool append/pin. TOML env is a different variable; TS constants/reads are not setters. Inherited `AILANG_*` survives envpolicy; full-profile policies allowed. |
| Revision: Python presence-only scan of `~/.config/ailang/mission-*.env`, `secrets.env`, `~/Library/LaunchAgents/dev.ailang.mission-*.plist`, `~/.codex/config.toml`; `os.environ.get('AILANG_AGENT_POLICY','')` | 6 profiles + 1 secrets file + 8 plists + 1 Codex config: 0 exact-marker references; designer process empty/unset. No secret values printed; no inference about other live role envs. |
| Revision: `nl -ba cmd/ailang/design_quorum.go`; reads of `seating.go`, `quorum.go`, `run.go`, policytool schema/profile | Seat selection :84 → reviewers :89 → writer :97 → stderr path :102. Entry point has no dir/error channel; default seats 3, cap $0.30; reserve/recalled seats can add calls. Profile excludes Bash; schema excludes artifact flag. |
| Revision: Python read-only sorted basename + NUL + bytes SHA256 snapshots twice | Pin/main/World counts and hashes unchanged from original pass; host now 14 files, hash `bb97999d417e4144872ef589eab7da75a3f99a5a625b3e5d0524d1e4da7aef75`; both revision snapshots equal. One additional host artifact since original pass; no real-path writes. |


Original-pass observation manifest SHA256s (before = after within that pass; revision host change logged above):

| Path under `~` | SHA256 |
|---|---|
| `.ailang-driver-pin/fleet/.ailang/state/mission-quorum` | `7b09a996ce8b90221bace1ab092aa23e6de20667a18f806669b03b0a8f57f787` |
| `dev/sunholo-data/ailang/.ailang/state/mission-quorum` | `079f080899fbfc58fb9a1731a80a8a1c3f2105290ac130960ee70f96e3d33d60` |
| `.ailang/state/mission-quorum` | `f910f1ef37d3a52c7d80d585a786c4b7304ea525af40be3ae377e78da624941f` |
| `.ailang/state/mission-quorum-world` | `7da76b3a307c6d221d9fec386599891da5c2932980f797beef9b9025f941dc04` |

## Axiom Compliance

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | +1 | Exported host-mission directory independent of CWD; attended/policy paths explicitly scoped |
| A2 Replayability | +1 | Future host-mission evidence survives ephemeral trees; no legacy recovery claim |
| A3 Effect Legibility | 0 | Existing artifact write remains explicit |
| A4 Explicit Authority | 0 | Policy-marked default remains confined |
| A5 Bounded Verification | +1 | Hermetic writer/reader and boundary arms |
| A6 Safe Concurrency | 0 | Preserve exclusive artifact creation; import deferred |
| A7 Machines First | +1 | One exported machine-readable path contract |
| A8 Minimal Syntax | 0 | No language syntax |
| A9 Cost Visibility | 0 | Preserve review costs and billing policy |
| A10 Composability | +1 | Unmarked, opted-in same-repo host missions share evidence |
| A11 Structured Failure | 0 | Invalid overrides error; suppression warns on stderr but has no structured status, and standalone IO validation is late |
| A12 System Boundary | 0 | Existing sandbox gate remains authoritative |

Net **+5**; hard checks A1/A3/A4/A7: no negative scores. No compiler conflict surface.

## Risks / Deferred Decisions

Legacy attribution and missing sources require explicit reporting, not fabricated evidence. Host storage is machine-local; disk loss remains possible. Env propagation can fail across harnesses: AC8 checks the actual handoff. Basename/freshness ambiguity already exists and is preserved. Implementer may choose helper/test names; namespace, attended resolution and confinement rules are frozen.

## Quorum round 1 response

- **oc-glm-5-3 / objection 1:** Added six-tree counts and per-lane setter/inheritance audit; rejected the ailang_only-only premise; narrowed Goal 1, added marked-mission failure, loud suppression/effective-path diagnostics and activation observations; A11 is now 0.
- **oc-kimi-k3 / objection 2:** Added reader decision table including unset repo/driver root, errors versus UNREVIEWED, repo-root cwd convention and extracted-block AC9 against an attended writer with all mission exports unset.
- **gemini-4-argon / objection 3:** Verified pre-review entry points lack directory/error handoff; driver pre-flight is in scope, standalone CLI pre-flight is deferred; stated $0.90 primary-seat cap exposure and reserve-seat caveat.
- **Controller scope note:** Deferred the importer entirely; core milestones only fix future durability/agreement, with legacy recovery explicitly separate.

## Related Documents / Out-of-Scope Follow-ups

- [Driver pin rollout](m-driver-pin-rollout.md): ephemeral roots and separate deployment.
- [Agentic quorum](../implemented/v0_30_0/m-mission-quorum-agentic-verify.md): review verification unchanged.
- [Fleet AB sprint](../implemented/v0_30_0/m-mission-fleet-ab-sprint-plan.md): original evidence.
- [Human-log path triage](ailang-core-triage/design-quorum-mission-log-path.md): markdown destination, separate problem.
- CLI flag-presence/help and pre-review path validation (`cmd/ailang/design_quorum.go`); design-doc-creator skill changes; any `internal/policytool/**` changes: outside allowlist.
- Legacy import and deployed binary/World rollout: separate follow-ups, real-path observation only.
