# M-QUORUM-ARTIFACT-DIR-STABLE: Durable, Repo-Scoped Quorum Evidence

**Status**: Planned — designer deliverable, not implementation approval
**Target**: Next patch after v0.54.9
**Priority**: P2 (correctness with an explicit-flag workaround)
**Estimated**: 1.5 days / 12 hours, including verification and buffer
**Dependencies**: Existing config Registry and policytool's pinned policy marker (#1552/#1554)
**Ticket**: `quorum:artifact-dir-cwd-relative` (V1, 2026-09-26); fleet iteration 38
**Created / updated**: 2026-10-14

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

- Every host-side mission writer and Gate 2 reader for the same repo resolves the
  same absolute directory across pins, sprint trees and separate clones.
- Preserve sandbox-relative default writes, explicit non-default flag paths,
  JSON schema, filename slugging and collision protection.
- Recover available legacy evidence additively; report missing sources honestly.
- No routing, lane order, reviewer selection, quota, budget or billing changes;
  no cross-machine replication, schema/freshness redesign, or automatic deletion.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Opt-in registered env; relative constant stays frozen | Preserves policytool's write gate | agent recommendation, normal design approval | design | high |
| One directory per repo, shared across its missions | Prevents duplicate review and cross-repo matches | agent recommendation | design | medium |
| Explicit additive import; no permanent read-both | Makes evidence survive source-tree removal | agent recommendation | design | medium |

**Design freeze:** policy-marked callers never use the env override for default writes; repo identity, not mission name or clone path, selects the host namespace. Implementation still requires the normal approval → planning → execution gates.

## Solution Design

### Recommendation and alternatives

Choose **(a)**: register `MISSION_QUORUM_ARTIFACT_DIR`, exported by the driver as an absolute host path; the quorum package resolves its default through this variable, and Gate 2 consumes the same variable with the same confinement exception.

Choose a **shared directory for all missions of one repository**, not per-mission: v1/docs/motoko/fleet review repo-scoped AILANG docs, including across separate clones. Namespace defaults by `MISSION_REPO` after profile sourcing:

| Repository | Host default |
|---|---|
| `sunholo-data/ailang` | `$HOME/.ailang/state/mission-quorum` |
| `sunholo-data/ailang-world` | `$HOME/.ailang/state/mission-quorum-world` |
| Other validated `owner/repo` | `$HOME/.ailang/state/mission-quorum-repos/<owner>/<repo>` |

The first two intentionally retain existing workaround directories. Validate two nonempty safe components; reject `.`/`..`, traversal and extra slashes. Require an absolute HOME for default construction. A deliberate nonempty absolute env override wins; operators must use the same override for all missions of that repo. An empty override selects the table. No directory is created merely to resolve or log it.

**(b) loses:** git-common-dir identifies one clone, not all clones of the same repo; bare repos need another convention. More seriously, unconditional resolution to the main checkout would move policytool's default write outside its checked sandbox. **(c) loses:** a reader-only change cannot recover writes into deleted trees and leaves future writers and readers disagreeing. Read-both postpones the loss until the next source deletion and makes historical lookup depend on clone layout.

### Resolution contract (writer and reader)

Keep `quorum.ArtifactDir` byte-for-byte unchanged. Add `ResolveArtifactDir(dir string) (string, error)` and call it before `MkdirAll`:

| Writer argument / process state | Effective directory |
|---|---|
| Nonempty argument unequal to `ArtifactDir` | Explicit argument, existing behavior |
| Empty or exact `ArtifactDir`, raw `AILANG_AGENT_POLICY` nonempty | `ArtifactDir`; ignore override even if invalid |
| Empty or exact `ArtifactDir`, no policy marker, override nonempty | Validated absolute override |
| Empty or exact `ArtifactDir`, no marker, override empty/unset | `ArtifactDir`, existing attended behavior |

Register `EnvMissionQuorumArtifactDir` in `internal/config/mission.go`, default `""`, AreaMission, with a getter using `get(...)`. The getter returns `""` whenever `get(EnvAgentPolicy) != ""` **without trimming that marker**; otherwise return the override verbatim. No raw `os.Getenv` outside config. Reject relative, literal-tilde, whitespace-only and NUL-containing overrides before creating files; preserve spaces inside valid absolute paths. Test NUL validation through a pure validator directly: environment values cannot contain NUL. Surface errors; never silently revert to local storage.

The CLI does not expose flag-presence information to the package. Consequently, explicit `--artifact-dir .ailang/state/mission-quorum` is also the default sentinel. To force a local path in an opted-in host process, use `--artifact-dir "$PWD/.ailang/state/mission-quorum"`. Distinguishing an explicitly supplied sentinel is an out-of-scope CLI follow-up, not an assumed capability.

Add a bash 3.2 library with pure namespace/effective-reader resolution and a lookup helper. Driver: source after all profile/secrets sourcing, explicitly check its return status (driver lacks `set -e`), export the result and log the path before any controller spawn. Existing Codex forwarding admits `MISSION_*`; stage inheritance also admits this prefix. Verify propagation without changing any lane recipe.

Gate 2 sources the library via `MISSION_DRIVER_ROOT` (attended AILANG fallback: current repo root), resolves the effective directory, then looks for the existing `<doc-id>-*.json` pattern with the directory quoted. Missing directory/zero matches means unreviewed; resolution/read-permission errors are errors, never empty evidence. Retain current slug/verdict rules. Name this directory for subsequent artifact reads and quorum invocations; leave the flag omitted so the package resolves the default. Include an executable shell block in Gate 2; the shell test extracts and runs that block, binding verification to the skill itself. Update `:215` as a historical local-directory observation, not a current lookup rule.

### Policytool safety argument — verified against code

`internal/policytool/cli_ops.go:60,72` assumes the sandbox-relative directory. `:317–322` checks write protection **before** launching; `:329–334` selects sandbox CWD. The `design_quorum` schema at `:108` does **not** admit `--artifact-dir`. `policytool.go:119` calls `childEnv`; `cli_ops.go:394–402` preserves other inherited variables but replaces every policy-marker entry with the host's policy path. `runAilang` at `:421–424` supplies this env/CWD to the unconfined child.

Thus merely exporting an override is **unsafe without the suppression above**. With suppression, every default call through that host still writes exactly the relative path the gate checked, even when a host override is inherited. The marker is a conservative suppression signal, not a new permission; no policy parsing or validity test may relax it. Existing arbitrary explicit library arguments retain their behavior; the mediated CLI cannot supply one. No policytool edit is required. `TestPolicyTool_QuorumArtifactDirInSync` alone only pins a constant: acceptance must also exercise the inherited-env child chain, so an unchanged constant cannot hide a changed effective write destination.

### Migration and rollout

Provide an **explicit, host-only, one-time additive import** tool. Require a repo identity and operator-selected source worktrees; verify their origin identity with the existing `pin-root.sh:97` normalizer. Default mode reports candidate JSON paths, `doc` values, missing sources and conflicts; an explicit apply mode copies to the resolved repo directory. Refuse import when the raw policy marker is nonempty. Source origin does not prove every historical artifact's attribution: the operator must attribute selected evidence before applying.

For AILANG, select the main checkout's 164 and fleet pin's 5 before any pin refresh; retain the 13 already in the destination. The observed combined 182 have distinct basenames and valid `doc`/`synthesis` keys. There are 33 non-`design_docs/` doc strings (21 absolute, 12 other relative); do not discard them merely because old trees vanished. Unattributed files are explicitly reported and left in place. World imports only World-attributed sources into its own directory; its 2 current files stay intact.

Refuse symlink artifact inputs/destination entries and symlink components beneath selected roots; reject source/destination aliasing. Copy JSON bytes unchanged, never rename their evidence timestamps or overwrite: identical existing bytes are a reported skip; differing same-name bytes are a named error, preserving both source and destination for resolution. Publish complete copies using same-directory temporary files plus exclusive link/create, not a check-then-overwriting `cp`. Reruns are idempotent. Keep originals; no automatic scan, import on fire, read-both, or cleanup. Deleted sources are reported `absent`, never reconstructed as a pass. Roll back by removing the driver export/reader opt-in; keep imported evidence. Activate the export and reader together only after the controller observes the updated binary and matching skills; a pinned source update alone does not prove binary reach. Until then retain the explicit host-side flag workaround. Live binary/World adoption remains **UNVERIFIED** until observed.

### Exact touch points and scope

All line references are at `b5ca4f6cf`, before edits.

| In-scope file / anchor | Planned change / approximate LOC |
|---|---|
| `internal/config/mission.go:13,42,65` | Constant, Registry row and policy-aware getter; +20 |
| `internal/mission/quorum/artifact.go:14,40` | Preserve constant; resolver and pre-write call; +35 |
| New `internal/mission/quorum/artifact_dir_test.go`, `artifact_policy_test.go` | Resolution and actual host/child boundary tests; ~150 |
| `tools/launchd/mission-control.sh:119,123` | Source pure helper, validate/export/log after sourcing; +10 |
| New `tools/launchd/lib/quorum-artifacts.sh` | Bash 3.2 namespace, reader and lookup helpers; ~80 |
| New `tools/launchd/migrate-quorum-artifacts.sh` | Explicit report/apply import; ~90 |
| New `tools/launchd/test_quorum_artifacts.sh`; new `internal/mission/dispatch/quorum_env_test.go` | Hermetic cross-CWD, namespace, import, documented reader and actual environment handoff arms; ~180 + 50 |
| `.claude/skills/mission-control/resources/gate-2-pick.md:139,140,199,215` | Effective path, quoting and historical clarification; ~15 |
| Same `.agents/skills/mission-control/resources/gate-2-pick.md` | Verbatim generated mirror |
| `make/test.mk:70` | Add hermetic shell test to existing launchd suite; +1 |
| `docs/docs/reference/env-vars.md` | Regenerate through `make docs-env` |

Config getter behavior tests belong in quorum tests: `internal/config/*_test.go` is outside the stated allowlist. Source/mirror changes stay within mission skills; use `tools/launchd/sync-agents-skills.sh`, the verified location, not the ticket's nonexistent `scripts/sync-agents-skills.sh`. Do not expand its six-skill scope.

## Milestones / Timeline

**M1 — Resolution and confinement (4h):** Registry/getter, writer resolver, unit and host/child tests; keep default constant/schema untouched. Pass AC1–4 and registry checks.

**M2 — Driver, reader, recovery (8h):** Pure shell helpers, driver export/logging, Gate 2/mirror, explicit importer, generated env docs, suite wiring and independent evaluation. Pass AC5–10. Neither milestone edits CLI or policytool files.

## Acceptance Criteria / Testing Strategy

New Go arms set HOME to `t.TempDir()`; all shell/integration/import arms use `HOME=$(mktemp -d)` and fixtures beneath it. Stub reviewers; spend no tokens. Run mutants only in disposable evaluator copies. Real home paths are observation-only.

| ID | Test arm / required outcome | Mutation killed |
|---|---|---|
| AC1 | Empty and sentinel arguments, env unset/empty: exact local default; constant remains pinned | Always-home default; changing constant |
| AC2 | Absolute env with spaces: real writes land there; distinct explicit absolute/relative arguments win | Ignore env; env overrides explicit flag; unquoted path |
| AC3 | Relative/tilde/whitespace env plus direct NUL-validator arm: error and zero artifact writes; unwritable destination errors | Silent local fallback; accept invalid env |
| AC4 | Actual policytool host with helper child calling real writer, inherited absolute/invalid override: allowed writes stay in sandbox; deny `.ailang/**` launches zero children; external fixture stays absent. Test a raw whitespace policy marker too | Remove/trim marker suppression; bypass deny check |
| AC5 | Fixture writer in tree A, delete A, reader in B and separate clone fixture finds exact artifact bytes in exported directory | Retain CWD reader/writer; read main clone only |
| AC6 | v1/docs/motoko/fleet map identically; World/other repo cannot match same-slug AILANG fixture; traversal/invalid HOME/relative override fail | Mission-name namespace; global cross-repo pool |
| AC7 | Execute the shell block extracted from the actual Gate 2 skill; absent is empty, unreadable/invalid config is error; missing env in mission fixture fails wiring check | Treat lookup errors as unreviewed; stale skill snippet |
| AC8 | Mock controller/stage recipes observe export through existing Codex `mc_codex_env_args` and dispatch; no model/quota probes; suite runs via `/bin/bash` | Unexported variable; strip `MISSION_*`; bash 4 construct |
| AC9 | Fixture import: report-only writes nothing; 164+5+13 distinct seeded JSONs → 182; rerun unchanged; absent source reported; conflict/symlink inputs cannot overwrite or escape selected fixtures; policy-marked import refuses | Remove host-only guard; destructive move; blind overwrite; non-idempotent import |
| AC10 | Focused Go + shell tests, registry checks, skill sync, boundaries and lint pass; env docs updated; real-directory manifest before/after unchanged or absent | Omitted Registry/suite/mirror; accidental real-HOME fixture |

Run `go test ./internal/mission/quorum ./internal/policytool`, focused config registry tests, the new shell test and `make test-launchd-drivers`; then `/bin/bash tools/launchd/sync-agents-skills.sh --check`, `make check-boundaries`, `make lint`. Full config tests also run on a runner permitting httptest listeners. An independent evaluator must remove suppression and revert the reader path and observe the corresponding arms fail; fixture-vs-fixture agreement alone is insufficient.

## Verification Log

Commands actually run, 2026-10-14; implementation arms above are **not yet run**.

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

Observation manifest SHA256s (full hashes, before = after; no real-path writes):

| Path under `~` | SHA256 |
|---|---|
| `.ailang-driver-pin/fleet/.ailang/state/mission-quorum` | `7b09a996ce8b90221bace1ab092aa23e6de20667a18f806669b03b0a8f57f787` |
| `dev/sunholo-data/ailang/.ailang/state/mission-quorum` | `079f080899fbfc58fb9a1731a80a8a1c3f2105290ac130960ee70f96e3d33d60` |
| `.ailang/state/mission-quorum` | `f910f1ef37d3a52c7d80d585a786c4b7304ea525af40be3ae377e78da624941f` |
| `.ailang/state/mission-quorum-world` | `7da76b3a307c6d221d9fec386599891da5c2932980f797beef9b9025f941dc04` |

## Axiom Compliance

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | +1 | Explicit stable directory, independent of CWD |
| A2 Replayability | +1 | Evidence survives ephemeral trees |
| A3 Effect Legibility | 0 | Existing artifact write remains explicit |
| A4 Explicit Authority | 0 | Policy-marked default remains confined |
| A5 Bounded Verification | +1 | Hermetic writer/reader and boundary arms |
| A6 Safe Concurrency | 0 | Preserve exclusive artifact creation; imports never overwrite |
| A7 Machines First | +1 | One exported machine-readable path contract |
| A8 Minimal Syntax | 0 | No language syntax |
| A9 Cost Visibility | 0 | Preserve review costs and billing policy |
| A10 Composability | +1 | All same-repo missions share evidence |
| A11 Structured Failure | +1 | Bad override/import conflicts surface errors |
| A12 System Boundary | 0 | Existing sandbox gate remains authoritative |

Net **+6**; hard checks A1/A3/A4/A7: no negative scores. No compiler conflict surface.

## Risks / Deferred Decisions

Legacy attribution and missing sources require explicit reporting, not fabricated evidence. Host storage is machine-local; disk loss remains possible. Env propagation can fail across harnesses: AC8 checks the actual handoff. Basename/freshness ambiguity already exists and is preserved. Implementer may choose helper/test names and import report formatting; namespace and confinement rules are frozen by this recommendation.

## Related Documents / Out-of-Scope Follow-ups

- [Driver pin rollout](m-driver-pin-rollout.md): explains ephemeral roots; its wider rollout is distinct.
- [Agentic quorum](../implemented/v0_30_0/m-mission-quorum-agentic-verify.md): review verification, unchanged here.
- [Fleet AB sprint](../implemented/v0_30_0/m-mission-fleet-ab-sprint-plan.md): original artifact evidence, not durable-path resolution.
- [Human-log path triage](ailang-core-triage/design-quorum-mission-log-path.md): optional markdown destination, separate problem.
- **Outside milestones:** `cmd/ailang/design_quorum.go:41,149` flag-presence handling/help and pre-review path validation; keep this CLI untouched.
- **Outside milestones:** `.claude/skills/design-doc-creator/SKILL.md:400` and `.agents/...:371` explain env opt-in; neither is a fleet-authorized skill edit.
- **Outside milestones:** any `internal/policytool/**` hardening/test expansion; current safety is supplied in the quorum package and tested from its scope.
- **External rollout follow-up:** observe deployed binaries/export reach, attribute/import real legacy files and verify World adoption separately. No real-HOME seeded test or cross-repo file edit is part of these milestones.
