# M-PKG-QUALITY-LADDER: Package Quality Gates, an Intent × Measured-Class Approval Ladder, and Staged Effect Tightening

**Status**: Planned
**Target**: v0.40.0 (Sprint 1), v0.40.x (Sprint 2), v0.41.0 (Sprint 3)
**Priority**: P1 — High. The registry's two strongest quality signals (Z3 contracts, interface identity) are currently measuring nothing, and the autonomous cascade has been dormant for four months. Nothing is *blocked*, which is exactly why it has gone unnoticed.
**Estimated**: 3 sprints, ~9 days total (Sprint 1: 3 days · Sprint 2: 3 days · Sprint 3: 3 days). Each sprint is independently shippable.
**Dependencies**:
- [m-registry-interface-hash-blind-to-signatures](../v0_35_0/m-registry-interface-hash-blind-to-signatures.md) — Sprint 1 (M1–M5) **shipped dark** 2026-09-01; this doc *is* its deferred Sprint 2 (M6–M8), re-planned against the live-registry measurement in §Verification Log.
- [m-package-test-discovery](../m-package-test-discovery.md) (planned, v0.38.x) — `pkg quality` reports the union of `*_test.ail` + inline blocks; lands first or is folded into Sprint 1 M3.
- Package messaging graph (M-PKG-MSG, shipped v0.9.9), autonomy router (M-PKG-CASCADE-DETERMINISTIC-FIRST, v0.16.0), `ApprovalCheckpoint` (v0.6.4) — reused, not replaced.
**Source**: Attended research session 2026-09-17 (Mark): "improve the ailang package pipeline — more checks for higher-quality packages, a required change description per version, auto-approve security patches vs human review for signature/effect changes, tighter effects (Net → one domain), more Z3, pure-core/effectful-shell style, hook approvals into `ailang messages`, and revisit semver vs hashes."
**Lane** (PROGRAM.md): AILANG fix / tooling — registry validator, `cmd/ailang` package commands, messaging, coordinator routing. **No core language change**; the motoko core is untouched.
**Quorum**: triggers **1** (design-freeze items D1–D5 need a human ruling) and **3** (adds fields to the banked `metadata.json` / `index.json` schema) fired. **Two rounds run 2026-09-17** (`gpt6-astra`, `gemini-3-1-pro`, `oc-glm-5-2`; $0.20 + $0.23; artifacts in `.ailang/state/mission-quorum/m-pkg-quality-ladder-2026-09-17T*.json`), both **blocked**, every objection accepted and applied: round 1 — `ack` is not approval (V28), no server-side execution of untrusted tests/smoke (server/attested split, V29), blast radius unmeasured for 34/53 (re-measured, V8 b/c); round 2 — v1 routing must not persist during shadow (D6 rewritten: routing is v2/`U` from M2, only the refusal gate shadows; M9 ordered after M6/M7), three unlogged premises (V30–V32). The re-quorum-once guardrail is exhausted; the doc went to Mark with these applied rather than a third round, and **Mark ratified D1–D7 in the same attended session (2026-09-17)**.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Replaces "the author picked a version and an AI decided if it was safe" with a deterministic function of (signature diff, effect delta, contract delta, declared kind). Same inputs → same tier, replayable from the banked envelope. |
| A2: Replayability | +1 | Every publish banks the inputs the ladder used (`interface_hash_v2`, `interface_signatures`, `effects_measured`, `release.kind`, contract counts), so a routing decision can be recomputed months later. |
| A3: Effect Legibility | +2 | `effects_measured` (what the bodies actually require) is banked next to `effects_max` (what the author claimed); over-declaration becomes visible per package and per function. Scopes (`Net = {domains}`) make *where* authority goes legible, not just *which* effect. |
| A4: Explicit Authority | +2 | Effect widening, scope widening and privilege-rank escalation are the only path to Tier 3 and can never auto-apply; a `kind = "security"` claim cannot smuggle a wider ceiling because the validator refuses intent/measurement inconsistency. |
| A5: Bounded Verification | +1 | Contract verification finally runs per package (today 0/373 versions verified anything); the classifier is a local diff of two signature sets, no whole-graph reasoning. |
| A6: Safe Concurrency | 0 | No concurrency model change. |
| A7: Machines First | +2 | `pkg quality --json` is one structured document consumed identically by `publish --dry-run`, the validator, the explorer and agents; the phantom `pkg quality` command agents were already told to run becomes real. |
| A8: Minimal Syntax | 0 | No language syntax. Two manifest sections (`[release]`, `[effects.scopes]`) — TOML the manifest already uses; unknown sections are tolerated by every shipped binary (V13). |
| A9: Cost Visibility | +1 | Tier 0/1 stay on the $0 deterministic path; only Tier 2/3 spend model or human time, and the tier is banked so the cost of the ladder is auditable. |
| A10: Composability | +1 | Composes with the existing message kinds (adds one producer for `contract-regression`, which has a schema and a router arm but no emitter today), the cascade envelope, and `ailang run --policy`. |
| A11: Structured Failure | +1 | Every refusal gets a `PUBnnn` code with the measurement that produced it (which signature changed, which effect widened, which section is missing) instead of a free-text 400. |
| A12: System Boundary | +1 | The registry boundary now carries the authority delta (effects, scopes, rank) explicitly on every crossing. |

**Net Score: +14** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): tier = pure function of banked inputs; the AI path is only reached from Tier 1/2 and never decides the tier.
- [x] A3 (Effects): nothing hidden — the doc's whole point is to bank the measured footprint.
- [x] A4 (Authority): no ambient grant; widening is the strictest tier and cannot auto-apply; nil ceiling (today "unlimited") becomes a refusal at `stable`.
- [x] A7 (Machines First): all outputs are JSON/TOML; the changelog section is the one prose artifact and it is *required to exist*, not parsed for meaning.

## Problem Statement

The publish pipeline has the right shape — client checks → validator → hashes → package messages → cascade → deterministic bump or AI repair → autonomy-routed PR — but four of its quality signals were found, by measuring the live registry rather than reading the design tree, to be inert. All numbers are from 2026-09-17 against `storage.googleapis.com/ailang-registry` (53 packages, 373 versions); commands are in the Verification Log.

**Current State:**

1. **The Z3 gate has never verified a contract.** `contracts_total = 0` on all 53 latest `metadata.json` (V1), yet 10+ package files carry `requires`/`ensures` (V4). Cause: `runAilangVerify` runs `ailang verify --json <absolute file>` per file, which fails MOD010 on every package module because tarballs are flat (`settle.ail` declares `module sunholo/deontic/settle`) (V2, V22); and even when verify succeeds its output is an object with a `results` array, which the validator tries to decode as a bare array, so nothing is ever counted (V3). `sunholo/deontic/settle` alone verifies **7/7** contracts under `--relax-modules` (V4). The registry says 0.
2. **Interface identity is blind to signatures.** `pkg.InterfaceHash` hashes name, edition, ailang constraint, exported *module paths* and `effects.max` (V5). Retyping or removing an exported function is class **A** → `SkipApproval + AutoMerge` in the autonomy router (V18). The signature-sensitive `InterfaceHashV2` + `U` class shipped 2026-09-01 with **zero non-test callers** (V6). Its wiring (M6–M8) was deferred pending a blast-radius count nobody in the loop could run. Measured here (V8): the path-mapping in `BuildCanonicalJSON` cannot find *any* flat-layout module (V7) — the same layout defect as (1) — so "blast radius" today is 53/53 for a reason that is a bug, not a policy.
3. **The cascade is dormant.** 28 `[cascade]` PRs exist in ailang-packages, all dated 2026-05-17 (V9). `sunholo/daneel_ext_abi 0.4.0` (2026-09-15) had six registry dependents and fired nothing (V10). Dependents now live in other repos and the topic publisher is only constructed when a cloud project resolves on the publishing machine.
4. **Version arithmetic and change class are disconnected.** The author picks any version; nothing checks that a class-C change carries a major bump. A version was burned this morning (`agui 0.2.1`, "Ship corrected registry metadata") because metadata cannot be corrected without a release.
5. **No change description exists anywhere.** 0 `CHANGELOG*` files across 45 package directories (V23); `metadata.json` has no per-version notes; `ai_summary` describes the package, not the release. Consumers, cascades and humans cannot tell a security fix from a refactor.
6. **`ailang pkg quality --strict` is a phantom.** ailang-packages' AGENTS.md, SKILL and package READMEs have told agents to run it since 2026-09-08 (V11); it was never designed or shipped. Agents were instructed to "skip it rather than report it as blocked" — a gate that exists only as a sentence.
7. **Effect ceilings are coarse and unmeasured.** A missing `[effects]` section is *unlimited* (`sunholo/logging`, V25). All `motoko_ext_*` declare nine effects because the ABI hook record types force the full row (V20) while, e.g., `motoko_ext_fmt`'s own functions require `{FS, Process}`. Nothing warns when a declared effect is never required by a body (V14). Budgets (`FS @limit=5`) exist in signatures but not in ceilings; `Net`/`Process` scoping exists only at run time (`--net-allow-domains`, `--process-allowlist`, `agent-policy.toml`) with no way for a package to *declare* what it needs.
8. **Provenance and stability are empty.** `published_by` is empty on 52/53 (superuser key, V21); `stability` unset on 18/53, `stable` on 1/53; `publish --dry-run` runs no check, test or verify (the packages-repo AGENTS.md says so).

**Impact:**
- Every "auto-merged" cascade bump to date was auto-merged on a classifier that cannot see signatures. The autonomous path is unsafe in exactly the case it was built to protect.
- Publishers get no useful feedback before upload: `--dry-run` cannot fail on anything the validator will fail on.
- The two things Mark asked for — auto-approve security patches, human review for signature/effect changes — are unimplementable until (1) and (2) are fixed, because "security patch" and "signature change" are both unmeasured.

**Systemic audit (why one doc, not three bug fixes):** (1) and (2) share one root — module path ↔ flat file mapping is done by `resolveModuleToFile` in `internal/pkg/discover.go` for `check --package`, and *re-derived wrongly* by `verify` (per-file) and `BuildCanonicalJSON` (canonical path). One package-aware resolution used by all three closes both. The ladder (Sprint 2) is meaningless without Sprint 1's measurements; Sprint 3's effect tightening is a third consumer of the same banked-metadata schema. Splitting them would produce three metadata migrations.

## Goals

**Primary Goal:** Every publish is routed by a deterministic ladder — **author intent × measured change class** — where the measurements (signatures, contracts, effects) actually run, and publishers see the same verdict in `--dry-run` that the validator will give.

**Success Metrics:**
1. `contracts_total > 0` on the next publish of every package that carries contracts (≥10 today); `sunholo/deontic` banks `7/7`.
2. `interface_hash_v2` + `interface_signatures` banked on 100% of publishes after Sprint 1 M2; a retyped exported function on a fixture package classifies **C**, not A (seam test, mutation-checked).
3. `ailang publish --dry-run` and the validator disagree on **zero** gates for the 31 packages that build clean in V8(b) (`server` fields of `pkg quality --json` byte-identical on both sides).
4. A `kind = "security"` release with an interface change or effect widening is refused with `PUB003`; a class-A `kind = "fix"` release from a fixture reaches auto-merge with `turns=0, cost=$0` (the deterministic cascade path already measured at 7.5 s in v0.16.0).
5. Effect widening or privilege-rank escalation produces exactly one `effect-widening-warning` in the operator inbox and **never** auto-applies (mutation arm: delete the router case, watch the test fail).
6. `effects_measured ⊆ effects_max` banked for every publish; over-declaration surfaces in `pkg info` and the explorer as a badge, and as `PUB010` at `stability = stable|frozen`.
7. Friction bound: the median honest publish adds **≤ 2 authoring actions** (a changelog section, a `kind`) and no new blocking gate at `experimental`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1 — Semver stays as the label; the **measured class is the source of truth** and the version bump must be ≥ the class's required bump (A→patch, B→minor, C→major; under `0.x`, C→minor is accepted as "major" per the 0.x convention). Hash-only identity (Unison-style) rejected. | Determines whether agents and humans can still say "the one after 0.4.1"; refusing under-bumps is the first behaviour change publishers will hit. | human | design | high |
| D2 — Release description surface: `CHANGELOG.md` in the tarball with a required non-empty `## <version>` section **plus** `[release] kind = security\|fix\|feature\|breaking` in `ailang.toml`. Alternative rejected: notes only in the manifest (immutable per release, no accumulated history). | Becomes a required authoring action on every publish; the `kind` is the intent half of the ladder. | human | design | med |
| D3 — Gate vs badge split by stability: at `experimental` only compile, tool-name, changelog-section, kind-consistency and bump-consistency **block**; contracts, tests, measured-effect tightness and pure-export ratio are **badges**; at `stable`/`frozen` the badges become gates (`PUB010`–`PUB012`). | This is the friction dial. Too tight = nobody publishes (the explicit non-goal). | human | design | med |
| D4 — Effect privilege rank for Tier-3 routing: `Process > Net > FS > Env > AI > SharedMem > Stream > IO > Clock > Rand`, with `Declassify` always Tier 3 (it is a taint sink, not a capability). Widening *to* a higher-ranked effect than any currently in the ceiling is "escalation"; widening within rank is still Tier 3 but labelled "widening". | Encodes a security opinion that will route human attention; wrong order = wrong alerts. | human | design | low |
| D5 — Scope declaration surface: `[effects.scopes]` in the manifest (`Net = { domains = [...] }`, `Process = { allow = [...] }`, `FS = { sandbox = "workdir" }`), **declared not proved**, validated best-effort against string literals, consumed by `ailang policy derive`. Versus waiting for type-level `Net[scope=…]` (m-effect-scope-params, v1.1.0). | Introduces manifest fields that a later type-level design must honour or migrate. | human | Sprint 3 | med |
| D6 — V2 transition: **routing is v2 from the day M2 lands; only the *refusal gate* (PUB005) shadows.** The validator computes and banks `interface_hash_v2` + signatures on every publish. Classification for routing uses `classifyChange` on signature sets immediately: both sides have signatures → A/B/C; only the new side has them (every package's first post-M2 publish) → `U` → Tier 2, human — the shipped M5 rule. **No publish after M2 is ever routed by the signature-blind v1 hash.** The one-time cost is explicit: up to 53 first-publishes land in Tier 2 instead of auto-merging. PUB005 (refuse when V2 cannot build an exported module) stays a badge until N=20 consecutive publishes bank v2 cleanly, then becomes a gate; if N=20 is not reached in 30 days the shadow log goes to Mark as a decision — with routing already safe, the stall costs nothing but the badge. Pre-measured bound: ≤ 4/53 refusals post-M1 (V8). | Closes the authority gap instead of scheduling it: v1 auto-merge of a retyped export is impossible from M2 onward. | agent | Sprint 1 | low |
| D7 — Error-code namespace `PUB001`–`PUB0nn` for validator/publish refusals (verified unallocated, V12). | Shared namespace; a collision would mislabel an existing diagnostic. | agent | Sprint 1 | low |

### Design Freeze

Before sprint-executor starts (Sprint 1 needs only D6/D7; D1–D3 before Sprint 2; D4–D5 before Sprint 3):

- [x] D1 ratified (bump ≥ class; 0.x rule) — **Mark, attended 2026-09-17**
- [x] D2 ratified (CHANGELOG section + `[release] kind`) — Mark, 2026-09-17
- [x] D3 ratified (gate/badge table below) — Mark, 2026-09-17
- [x] D4 ratified (privilege rank order) — Mark, 2026-09-17
- [x] D5 ratified (`[effects.scopes]` declared-not-proved) — Mark, 2026-09-17
- [x] D6 agent-resolved: shadow mode, N=20
- [x] D7 agent-resolved: `PUBnnn`

## Solution Design

### Overview

Three sprints, one banked schema:

1. **Sprint 1 — make the gates measure something.** One package-aware module resolution for `verify`, `internal-dump-iface` and `check`; the validator runs package-level verify and V2 hashing (shadow); `ailang pkg quality [--json]` becomes real and is the single implementation behind `publish --dry-run` and the validator's Steps 6–8; `[release] kind` + `CHANGELOG.md` section are required and banked.
2. **Sprint 2 — the ladder.** Tier = f(kind, class, effect delta, contract delta). Validator refuses intent/measurement inconsistency and under-bumps. Cascade envelope carries tier; router maps tier → autonomy; Tier 3 goes through `ApprovalCheckpoint` to the operator inbox and is acked with `ailang messages ack`. Cascade re-armed for out-of-repo dependents. `contract-regression` gets a producer.
3. **Sprint 3 — effect tightening, staged.** `effects_measured` banked from `collectRequiredEffects`; over-declaration badge/gate; `[effects.scopes]` declared and consumed by `ailang policy derive`; nil ceiling refused at `stable`.

### Architecture

```
publisher                                  registry-validator                         coordinator
─────────                                  ──────────────────                         ───────────
ailang publish [--dry-run]                 POST /publish
  └─ pkg.QualityReport(dir)  ──same code──►  pkg.QualityReport(tempDir)
       ├─ check --package                        ├─ (auth, immutability, tool names: unchanged)
       ├─ tests (files ∪ inline)                 ├─ QualityReport ─► gates by stability (D3)
       ├─ verify --package  (Sprint 1 M1)        ├─ InterfaceHashV2 + signatures (shadow, D6)
       ├─ InterfaceHashV2 + signatures           ├─ ladder: kind × class × Δeffects × Δcontracts
       ├─ effects_measured  (Sprint 3)           │      → tier ∈ {0,1,2,3}, PUB0xx on inconsistency
       ├─ changelog section + [release].kind     ├─ metadata.json v2 (+ index.json fields)
       └─ verdict: same PUB0xx the validator      └─ cascade envelope (+tier, +kind, +rank delta)
          would return                                            │
                                                                  ▼
                                                  AdjustAutonomyForTier(tier)
                                                    T0 → deterministic bump, auto-merge
                                                    T1 → deterministic bump, auto-merge iff consumer green, else AI repair PR
                                                    T2 → PR, coordinator approval (human)
                                                    T3 → ApprovalCheckpoint (blocks) → inbox notification → `coordinator approve|reject` only
```

**The ladder (Sprint 2):**

| Tier | Condition (all must hold) | Autonomy |
|---|---|---|
| **0 auto** | class **A** (same signature set) · no effect/scope widening · contracts verified ≥ previous · smoke passed · `kind ∈ {security, fix}` | `SkipApproval=true, AutoMerge=true` — the existing v0.16.0 deterministic path, $0 |
| **1 agent** | class **B** (strict superset of signatures) · no widening · `kind ∈ {fix, feature}` | deterministic bump; auto-merge iff the consumer's `check --package` + tests pass in the cascade job; otherwise AI repair PR with `AutoApproveHandoffs=true`, merge needs approval |
| **2 human** | class **C** or **U**, or `kind = breaking`, or contract count regressed | PR; `coordinator approve` required |
| **3 authority** | any effect added to `effects_max`, any scope widened, or privilege rank escalated (D4), or `Declassify` added | never auto; `ApprovalCheckpoint.RequestApproval` blocks the consumer bump; the `effect-widening-warning` in the operator inbox is a **notification only**; release is exclusively `ailang coordinator approve <task>` (→ `ProcessApprovalRequest` → `ApprovalCheckpoint.Approve`). `ailang messages ack` is `MarkInboxMessageRead` and has no effect on the checkpoint (V28). |

**Consistency rules the validator enforces (Sprint 2):**
- `kind = security | fix` ⇒ class A and no widening, else `PUB003 intent/measurement mismatch` naming the signature or effect that contradicts the claim.
- `kind = feature` ⇒ class ≤ B, else `PUB003`.
- Version bump ≥ required bump for the class (D1), else `PUB004 under-bump` with the required version printed.
- `kind = breaking` never *lowers* a tier; it only documents.

A `security` claim is therefore a *stronger* assertion than `fix`, not a fast lane: it is the one label the ladder checks hardest, so it can be trusted enough to auto-merge.

**Two provenances, one report.** The validator must never execute untrusted package code: tests and `_smoke.ail` run **only on the publisher's machine** (as `_smoke.ail` already does today — the validator has never run it). Every field in the report therefore carries a provenance:

- `server` — static and bounded: compile (`check --package`), Z3 verify (per-function `--timeout 5s` today, plus a new per-package wall cap `PublishLimits.VerifyWall = 120s`), V2 interface build (bounded today by `PublishLimits{Overall, PerModule, MaxExportedModules}` with `PerModule` enforced via `context.WithTimeout` + `proctree.Kill`, V32), `effects_measured` (a type-checker pass), changelog/kind/bump checks, hashes. **Only `server` fields can be gates at the validator.**
- `attested` — executed publisher-side and reported in the upload (tests, smoke): banked with `attested_by` = the API-key owner (empty for the superuser key, which is V21's provenance gap made visible), shown as badges, **never a validator gate at any stability**. `publish` itself refuses locally on a failed attested check (as it does for smoke today), and `--strict` promotes them locally; a hostile publisher can lie about its own tests, which is why they gate nothing server-side.

`pkg.QualityReport` is still one function; it takes a `Mode{Server|Publisher}` and in `Server` mode the `attested` block is whatever the upload carried, never recomputed.

**`ailang pkg quality` report (Sprint 1, the one implementation):**

```json
{
  "schema": "ailang.package-quality/v1",
  "package": "sunholo/deontic", "version": "0.3.1",
  "compile":   {"source": "server", "ok": true, "files": 9},
  "contracts": {"source": "server", "verified": 7, "total": 7, "counterexample": 0, "skipped": 0, "uncontracted_exports": 14, "wall_seconds": 4.2},
  "tests":     {"source": "attested", "attested_by": "daneel", "files": 1, "inline": 4, "passed": 12, "failed": 0},
  "smoke":     {"source": "attested", "attested_by": "daneel", "present": true, "passed": true, "seconds": 1.9},
  "interface": {"hash_v1": "sha256:…", "hash_v2": "sha256:ifacev2:…", "signatures": 21,
                "class_vs_previous": "A", "previous": "0.3.0"},
  "effects":   {"max": ["Env"], "measured": ["Env"], "over_declared": [], "ceiling_declared": true,
                "scopes": {}, "rank_max": "Env"},
  "release":   {"kind": "fix", "changelog_section": true, "required_bump": "patch", "bump_ok": true},
  "docs":      {"agent_md": true, "ai_summary": true},
  "style":     {"exported_funcs": 21, "pure_exports": 17, "pure_ratio": 0.81},
  "tier":      0,
  "gates":     [], "badges": [{"code": "PUB011", "level": "info", "msg": "14 exported functions carry no contract"}]
}
```

`gates` is the list of `PUB0xx` that block at this package's stability; `badges` are the same codes at a lower level. `--strict` promotes badges to gates locally (what the phantom flag promised). Server-side, only `source: server` fields can appear in `gates`; PUB012 (tests) is therefore a publisher-local gate at `stable|frozen` and a badge in the registry.

**Error codes (D7):**

| Code | Blocks at | Meaning |
|---|---|---|
| PUB001 | all | `CHANGELOG.md` missing or has no non-empty `## <version>` section |
| PUB002 | all | `[release] kind` missing or not in the enum |
| PUB003 | all | intent/measurement mismatch (kind vs class/widening) |
| PUB004 | all | version bump below the class's required bump |
| PUB005 | all | exported module fails V2 interface build (post-shadow; during shadow: badge) |
| PUB010 | stable, frozen | effect over-declaration (`effects_max ⊄ effects_measured ∪ ABI-forced`) or nil ceiling |
| PUB011 | stable, frozen | exported function without a contract (frozen only) / contracts regressed |
| PUB012 | stable, frozen (publisher-local only; registry badge) | no tests discovered / attested tests failed |
| PUB013 | frozen | class ≠ A (frozen packages may only take internal changes) |

### Implementation Plan

**Sprint 1 — measure (3 days, v0.40.0)**

- **M1 — One module resolution for verify + iface (0.5 d).** Extract `resolveModuleToFile` (`internal/pkg/discover.go:128`) into a package-aware resolver and use it from `pipeline.BuildCanonicalJSON` and a new `ailang verify --package <dir>`; verify walks exported modules, emits one object `{results:[…]}` per module. V8(b) shows the two resolvers are contradictory today (self-package loader: flat; V2 builder: canonical), so this is a prerequisite for *any* V2 measurement, not a cleanup. Regression fixtures: a flat tarball layout with intra-package `pkg/<self>/…` imports (`sunholo/deontic` shape) *and* a `module_prefix` package (`sunholo/ailang_parse` shape); both must produce identical signature sets whether built from the flat tarball or a canonical checkout. Seam test: `runAilangVerify` on the deontic fixture returns `7/7/0`, and the JSON decode path is mutation-checked (change `results` to an array → test fails).
- **M2 — Validator wiring, shadow (0.5 d).** `handlePublish` Steps 6–8 call `pkg.QualityReport(Mode: Server)` — compile, verify (per-package wall cap), V2 build, static checks; **no test or smoke execution server-side**, the upload's attested block is banked as received; bank `interface_hash_v2`, `interface_signatures`, contract counts, `quality` sub-document in `metadata.json` (`ailang.package-metadata/v2`; v1 readers ignore new fields). Router still reads v1 (D6). Version-skew guard from the parent doc's M7: publisher-computed v2 ≠ validator-computed v2 → `400 PUB005` with both values.
- **M3 — `ailang pkg quality [--json] [--strict] <dir>` (1 d).** Same function; test discovery = files ∪ inline blocks (or depend on m-package-test-discovery); `publish --dry-run` prints the report and exits non-zero on any gate. Replace the phantom text in ailang-packages AGENTS.md/SKILL with the real flag set (separate PR in that repo).
- **M4 — `[release]` + `CHANGELOG.md` (0.5 d).** `ReleaseConfig{Kind, Notes}` on the manifest; validator gates PUB001/PUB002; bank `release.kind` and the section text (first 2 KB) in metadata + index; `pkg info` shows it; explorer reads it. `ailang init package` scaffolds both.
- **M5 — Blast-radius measurement in the validator (0.5 d).** Shadow-mode log line per publish `v2=ok|fail reason=…`; after N=20 clean publishes (D6) a one-line config flip makes PUB005 a gate. Routing does not wait for this: from M2 the envelope carries signature sets and the coordinator's existing `U → never auto` rule applies (V18), so the flip only changes what the *validator refuses*, never what the *router auto-merges*.

**Sprint 1 → 2 ordering invariant:** M9 (cascade re-arm) **depends on M2 + M6/M7** and may not land before them. Re-arming a cascade that is still routed by v1 would auto-merge exactly the retype/removal case this doc exists to stop; the sprint plan must encode the dependency, and the M9 seam test asserts that a retyped-export fixture routed through the re-armed cascade lands in Tier 2, not auto-merge.

**Sprint 2 — route (3 days, v0.40.x)**

- **M6 — Ladder + consistency rules (1 d).** `pkg.ComputeTier(report, previous)`; PUB003/PUB004; tier, kind, rank delta on the cascade envelope (`pubsub.CascadeEnvelopeFields`) and message payloads (`PackageRef`).
- **M7 — Router by tier (0.5 d).** `AdjustAutonomyForTier` replaces the class switch; Tier 3 → `ApprovalCheckpoint.RequestApproval` with the widening payload, which *blocks* the consumer bump; the `effect-widening-warning` is delivered to the operator inbox (prod Firestore via `AILANG_STORAGE_MESSAGING=gcp`) as a notification carrying the task id. The only release path is `ailang coordinator approve <task>` / `reject`; the inbox message's read state is irrelevant to the checkpoint. Mutation arms: delete the Tier-3 case → test fails; mark the inbox message read → bump still blocked. **Known weakness, flagged not fixed:** `coordinatorApprove` stamps `ApprovedBy: "cli-user"` (`cmd/ailang/coordinator_actions.go:40`) — approval is a store transition, not an authenticated identity. Strengthening it (signed approvals / key-scoped approvers) is Future Work; this doc does not widen what that rail already protects.
- **M8 — `contract-regression` producer (0.5 d).** Emitted when `verified` drops or a `counterexample` appears vs the previous version; routes Tier 2 (already in the router).
- **M9 — Cascade re-arm (1 d; after M2, M6, M7).** Publisher emits to the cascade topic whenever a project resolves *or* `AILANG_REGISTRY_API_KEY` is set (the validator can also emit server-side — decide in planning, the validator already knows the dependents via the index); dependents outside ailang-packages resolve their repo from `metadata.repository` (present on 35/53, V30) else land in a `pkg:<name>` inbox for a human. Seam test with a two-package fixture across two temp repos.

**Sprint 3 — tighten (3 days, v0.41.0)**

- **M10 — `effects_measured` (1 d).** Aggregate `collectRequiredEffects` (`internal/pipeline/validate_effects.go:305`) over every function in exported modules (body-required, not annotation); bank; `over_declared = max − (measured ∪ effects required by exported function *types*)` so ABI-forced hook rows are not misreported. Badge everywhere; PUB010 at `stable|frozen`; nil ceiling → PUB010 at `stable|frozen`, badge below.
- **M11 — `[effects.scopes]` (1 d).** Manifest + metadata + index; best-effort static check (Net string literals whose host is not in `domains` → badge, never a gate — it is declared authority); scope *narrowing* is class-neutral, scope *widening* is Tier 3.
- **M12 — `ailang policy derive <dir>` (1 d).** Emits an `agent-policy.toml` candidate = union of the package's and its locked dependencies' scopes and ceilings, for `ailang run --policy`. The operator edits and commits it; nothing runs on it automatically.

### Files to Modify/Create

- `internal/pkg/quality.go` — NEW (~350 LOC): `QualityReport`, `ComputeTier`, gate/badge tables, JSON schema.
- `internal/pkg/quality_test.go` — NEW (~400 LOC): fixtures for flat / `module_prefix` / contract-bearing / widening packages; mutation arms for the decode path, the Tier-3 case, the consistency rules.
- `internal/pkg/discover.go` — export the resolver (~20 LOC); `internal/pipeline/canonical_json.go` — use it (~30 LOC).
- `cmd/ailang/verify.go` — `--package` mode, per-module object output (~120 LOC).
- `cmd/ailang/pkg_quality.go` — NEW (~150 LOC): the subcommand; `cmd/ailang/pkg_commands.go` — dispatch (+5).
- `cmd/ailang/pkg_publish.go` — call `QualityReport` in dry-run and pre-upload; send v2 hash + kind headers (~80 LOC net).
- `cmd/registry-validator/validate.go` — replace `runAilangVerify`/`runAilangCheck` bodies with `QualityReport` (−120/+60 LOC); `cmd/registry-validator/main.go` — gates, PUB codes, schema v2, shadow log (~150 LOC).
- `internal/pkg/manifest.go` — `ReleaseConfig`, `EffectScopes` (~60 LOC); `internal/pkg/registry_types.go` — metadata v2 + index fields (~50 LOC).
- `internal/messaging/pkg_schema.go`, `pkg_events.go` — tier/kind/rank on `PackageRef`; `EmitContractRegression` producer (~120 LOC).
- `internal/pubsub/publisher.go` — envelope fields (~20 LOC); `internal/coordinator/autonomy_router.go` — `AdjustAutonomyForTier` (~60 LOC net); `internal/coordinator/store*.go`, `internal/storage/firestore/coordinator_convert.go` — three new columns (~60 LOC).
- `internal/coordinator/approval_checkpoint.go` — Tier-3 request shape (~40 LOC).
- `cmd/ailang/policy_derive.go` — NEW (~180 LOC, Sprint 3).
- `docs/docs/guides/packages.md`, `ailang docs package-authoring`, ailang-packages `AGENTS.md`/SKILL — replace the phantom command; changelog + kind authoring section.

## Conflict Surface

Not a parser/typechecker change, but the design *reuses or overrides* five shared mechanisms; each row is a reuse decision, not an override, except where marked.

| Position | What lives there today | Decision |
|---|---|---|
| Module path → file resolution | `resolveModuleToFile` (check --package), per-file `verify` (fails MOD010), `BuildCanonicalJSON` (canonical path only) | **Reuse** the first from the other two. `ailang verify <file>` single-file behaviour unchanged. |
| Change classification | `classifyChange` (messaging, A/B/C/U from signatures with hash fallback), `mapChangeClassToSchema` (publish, module-list heuristic), coordinator `ClassifyChange` (envelope → A/B/C) | **Reuse** `classifyChange`; retire `mapChangeClassToSchema`'s module-count heuristic once v2 is on (D6); coordinator `ClassifyChange` becomes tier-based (**override**, with the M5 `U → never auto` rule preserved and tested). |
| Autonomy fields | `SkipApproval / AutoMerge / AutoApproveHandoffs` on `AgentConfig` | **Reuse**; Tier 0/1/2 map onto the existing A/B/C settings byte-for-byte; only Tier 3 adds the checkpoint. |
| Human approval rail | `ApprovalCheckpoint` (coordinator) + `coordinator approve|reject` (`ProcessApprovalRequest`); notify daemon; the inbox (`ailang messages`) is notification only — `ack` = `MarkInboxMessageRead` | **Reuse**; no new approval CLI; the inbox never becomes an approval surface. |
| Metadata schema | `ailang.package-metadata/v1` read by `pkg info`, explorer, `FetchMetadata` in publish | **Extend** to v2 additively; v1 readers keep working (fields are `omitempty`); index gains `release_kind`, `contracts_total`, `effects_measured`, `tier`. |
| Tool-name gate (`tool_names.go` Pattern 2) | Over-matches any `name:` literal (core backlog 2026-09-15) | Out of scope but **must not be made stricter** by this doc; PUB codes are additive to it. |

**Programs/flows that MUST still work:** `ailang publish` of a package with no `[release]` section during Sprint 1 M4's grace window (badge, then gate one minor later — the same pattern as `--allow-dotted-tool-names`); `ailang verify <single file>`; `ailang check --package .` on `sunholo/ailang_parse` (module_prefix); every cascade fixture in `cmd/ailang/pkg_publish_test.go`; `ailang pkg info` against a v1 `metadata.json`.

## Examples

### Example 1: Security fix, Tier 0, $0

```toml
# ailang.toml
[package]
version = "0.8.2"        # was 0.8.1
[release]
kind = "security"
```
```markdown
# CHANGELOG.md
## 0.8.2
- token refresh no longer logs the refresh token on failure (Env, FS only; no interface change)
```
```
$ ailang publish --dry-run
  quality: compile ✓  tests 4/4 ✓  contracts 2/2 ✓  smoke ✓ (0.8s)
  interface: class A vs 0.8.1 (21 signatures unchanged)
  effects: max [FS, Net, Env]  measured [FS, Net, Env]  over-declared none
  release: kind=security  required bump=patch  0.8.1→0.8.2 ✓
  tier: 0 (auto)   gates: none
```
Validator banks the same report; cascade fires class A / tier 0 for `firestore`, `gemini_files`, `gcs_storage`, `ailang_parse`; each bump is the v0.16.0 deterministic path (`turns=0, cost=$0`), auto-merged.

### Example 2: A "fix" that changed a signature — refused

```
$ ailang publish --dry-run
  interface: class C vs 0.8.1 — changed: sunholo/gcp_auth/token.getToken
      was: (string) -> Result[string, string] ! {Net, Env}
      now: (string, int) -> Result[string, string] ! {Net, Env}
  release: kind=fix
✗ PUB003 intent/measurement mismatch: kind=fix requires class A; measured C (1 signature changed)
  Fix: set kind = "breaking" and bump to 0.9.0 (required bump for class C under 0.x: minor), or restore the signature.
```

### Example 3: Effect widening — Tier 3, never auto

`sunholo/email 0.3.4` adds `Process` to run a local MIME helper. The report shows `effects: max [Net, FS, Env, IO, Process] (+Process, rank escalation IO→Process)`. Publish succeeds (widening is legal), tier 3. Consumers get **no** bump: the cascade task parks in `ApprovalCheckpoint`. The operator inbox receives one `effect-widening-warning` carrying the before/after ceiling, the rank delta and the task id — reading or acking it changes nothing. `ailang coordinator approve <task>` releases the Tier-1 bump for consumers; `reject` closes it. (`ailang messages ack` only marks the notification read — V28.)

### Example 4: Scopes (Sprint 3)

```toml
[effects.scopes]
Net = { domains = ["api.stripe.com"] }
FS  = { sandbox = "workdir" }
```
```
$ ailang policy derive .
# agent-policy.toml candidate (edit, then commit)
caps = ["Net", "FS", "Env"]
net_domains = ["api.stripe.com", "oauth2.googleapis.com"]   # union: this package + sunholo/gcp_auth@0.8.2
fs_sandbox = "workdir"
```

## Success Criteria

- [ ] `runAilangVerify` on the deontic fixture banks `7/7/0`; decode-path mutation test fails when the shape is wrong.
- [ ] `interface_hash_v2` + signatures banked on every publish (shadow); retype fixture classifies C.
- [ ] `ailang pkg quality --json` `server` fields identical from the publisher and the validator on the 31 clean packages (V8b).
- [ ] PUB001–PUB005 refusals carry the measurement; grace window for PUB001/002 documented and time-boxed.
- [ ] Tier 0 fixture publish → auto-merged consumer PR with `cost=$0`; Tier 3 fixture → checkpoint + inbox message, no consumer PR.
- [ ] `contract-regression` emitted on a fixture whose contract count drops.
- [ ] Cascade fires for a dependent in a second repository (fixture) and for the daneel_ext_* dependents on the next `daneel_ext_abi` publish.
- [ ] `effects_measured` banked; `motoko_ext_fmt`'s report shows `over_declared` empty *because* the ABI-forced rows are attributed to function types, with `measured = [FS, Process]`.
- [ ] `stable` package with nil ceiling → PUB010; `experimental` → badge only.
- [ ] Phantom `pkg quality` text replaced in ailang-packages; `ailang init package` scaffolds `CHANGELOG.md` + `[release]`.
- [ ] All tests passing, `make simplicity-audit` no regression (one new command, two new manifest sections, zero new env vars — use `internal/config` Registry if any appears).
- [ ] Documentation updated (guides, `ailang docs package-authoring`, CHANGELOG).

## Testing Strategy

- **Seam tests over artifact tests** (the recorded lesson): every gate is tested where two implementations meet — publisher vs validator report equality; verify's JSON vs the validator's decoder; `classifyChange` vs the router's tier.
- **Mutation arms** per gate: revert the fix (per-file verify, array decode, Tier-3 case, PUB003 rule) and assert the test goes red before it goes green.
- **Fixtures**: `internal/pkg/testdata/quality/{flat,prefixed,contracts,widening,retype,regress}` — small packages, each with a `_smoke.ail`.
- **Live measurement gates** (not unit tests): D6's N=20 shadow count; the first real Tier-0 cascade after M9 recorded in the mission log with its cost.
- **No language claims to verify** — the doc adds no syntax; the two manifest sections are shown tolerated by shipped binaries (V13).

## Deferred Decisions

Agent latitude during execution:
- Whether the validator or the publisher emits to the cascade topic in M9 (server-side is simpler and already has the index; publisher-side keeps laptop publishes symmetric).
- Exact `CHANGELOG.md` heading grammar accepted (`## 0.8.2`, `## [0.8.2]`, `## v0.8.2 — date`): accept all three, match on the version token.
- Whether `pure_ratio` counts functions with only ghost effects as pure (recommend yes; `Debug` is erased at the type level already).
- Size cap on banked changelog text (recommend 2 KB, first section only).
- Whether PUB005 during shadow is a badge or silent log (recommend badge — visible, non-blocking).

## Non-Goals

- **No hash-only versioning.** Versions stay human-readable; hashes stay identity (D1).
- **No type-level scope parameters** (`Net[scope=…]`) — that is m-effect-scope-params / m-effect-clock-net-fs-modes; Sprint 3 declares scopes so that work has something to prove later.
- **No fix to the motoko ABI's nine-effect hook rows** — issue #800 / m-ext-registry-effect-row-ruling / m-effect-row-var-unification; this doc only stops *misreporting* it.
- **No CI-publish workflow for ailang-packages** (m-pkg-ci-publish is still open) — separate, though `pkg quality --json` is what such a workflow would run.
- **No tool-name gate changes** (core backlog row 2026-09-15).
- **No registry backfill (parent doc M9)** — historical versions keep v1 identity; the `U` class handles the first post-migration publish honestly.
- **No package signing / trust policy** (deferred/m-pkg-trusted-autonomous-evolution) — the ladder is a hook point for it, not a replacement.

## Timeline

| Sprint | Days | Milestones | Ships as |
|---|---|---|---|
| 1 — measure | 3 | M1–M5 | v0.40.0; validator deploy is release-gated |
| 2 — route | 3 | M6–M9 | v0.40.x; needs D1–D3 ratified |
| 3 — tighten | 3 | M10–M12 | v0.41.0; needs D4–D5 ratified |

Parent-doc honesty check: its Sprint 2 was estimated at 5 days including a 2.5-day backfill; this doc drops the backfill (non-goal) and adds the ladder, so Sprint 1+2 here ≈ 6 days against its 5.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Publishers hit PUB001/PUB002 on day one and stop publishing | One-minor grace window as badges (same pattern as the tool-name migration); `ailang init package` and the packages-repo skill scaffold both artifacts; agents already believed a quality gate existed. |
| V2 refuses packages that build today (parent doc's V-log #37) | Measured, not assumed (V8): after M1 the refusal set is bounded at 4/53, all of which already fail the compile gate on the current binary; 31/53 build clean with real dependency resolution and 18 more pass `check --package` in their shipped layout. Shadow mode (D6) confirms the 49 rather than certifies them. **Fallback if N=20 is not reached within 30 days of M2:** the router stays on v1, PUB005 stays a badge, and the shadow log's refusal reasons go to Mark as a decision — no silent flip and no silent stall. |
| `effects_measured` misreports ABI-forced rows as over-declaration | Attribute effects required by exported function *types* (record fields with effect rows) as forced; badge only below `stable`. |
| Tier 3 floods the operator inbox | One message per publish, superseded by the next (`Store.SupersedeOlderMessages(pkgName, newVersion)`, `internal/messaging/pkg_status.go:75`, V31); rank escalation is a label, not a second message. |
| First post-M2 publish of every package lands in Tier 2 (`U`) — a one-time wave of up to 53 human decisions | Deliberate (D6): it is the explicit, visible cost of never routing on the blind hash. Batched in `coordinator pending`; each decision also seeds the package's signature baseline so the wave does not recur. Rejected alternative: keep v1 auto-merge during shadow (round-2 quorum objection — it left the authority gap open). |
| Cascade re-arm reopens the 2026-09-05 backstop dispatch loop | Envelope idempotency key = `root@version → dependent`; the cascade scheduler's cost cap (`[cascade] max_cost_usd`) is unchanged. |
| Validator and publisher binaries skew | Version-skew guard (M2) returns both hashes; the validator's `/version` already exposes its build. |
| Server-side execution of untrusted package code (RCE / unbounded waits) | Structural: the validator runs only static, bounded steps (`check`, Z3 with per-function + per-package caps, iface build with `PerModule` cap, effect pass); tests and smoke are `attested`, executed publisher-side, badges only. The validator already never runs `_smoke.ail`; this doc keeps that line and makes it explicit in the schema. |
| A publisher lies in the attested block | Attested fields never gate at the validator; `attested_by` is banked and visible; a false attestation is a provenance problem for the trust-policy work (Future Work), not a registry-integrity problem. |

## Related Documents

- [m-registry-interface-hash-blind-to-signatures](../v0_35_0/m-registry-interface-hash-blind-to-signatures.md) + [sprint plan](../v0_35_0/m-registry-interface-hash-blind-to-signatures-sprint-plan.md) — Sprint 1 shipped dark; this doc is its Sprint 2 minus backfill.
- [m-pkg-package-system](../../implemented/v0_9_5/m-pkg-package-system.md) (v0.9.5) — exact versions + content/interface hash lock; the dual-hash model this doc completes.
- [m-pkg-cascade-deterministic-first](../../implemented/v0_16_0/m-pkg-cascade-deterministic-first.md) (v0.16.0) — the $0 deterministic bump path Tier 0/1 reuse; its "Known Gap 5: function-level interface diff" is closed here.
- [m-pkg-msg-package-messaging-graph](../../implemented/v0_9_9/m-pkg-msg-package-messaging-graph.md) (v0.9.9) — message kinds; `contract-regression` gains its producer.
- [m-pkg-install-latest](../../implemented/v0_10_0/m-pkg-install-latest.md) (v0.10.0) — "AILANG can do better than semver"; D1 is the ruling on that.
- [m-pkg-trusted-autonomous-evolution](../../deferred/m-pkg-trusted-autonomous-evolution.md) (deferred) — SAFE/ADDITIVE/BREAKING/AUTHORITY/UNKNOWN table; the ladder is its runnable subset.
- [m-secret-effect-remote-approval](../../implemented/v0_26_0/m-secret-effect-remote-approval.md) — the `ApprovalCheckpoint` + notify-daemon rail Tier 3 reuses.
- [m-package-test-discovery](../m-package-test-discovery.md) — test discovery union `pkg quality` reports.
- [m-ext-registry-effect-row-ruling](../m-ext-registry-effect-row-ruling.md), [m-effect-row-var-unification](../v1_0_0/m-effect-row-var-unification.md) — the ABI over-declaration root cause (non-goal here).
- [m-effect-clock-net-fs-modes](../v1_0_0/m-effect-clock-net-fs-modes.md), [m-effect-scope-params](../v1_0_0/m-effect-scope-params.md), [m-process-modes](../v1_1_0/m-process-modes.md) — type-level successors to `[effects.scopes]`.
- [m-ext-portability-gate](../../implemented/v0_18_11/m-ext-portability-gate.md) — the `_smoke.ail` gate `pkg quality` folds in.
- [m-pkg-ci-publish](../../implemented/v0_10_0/m-pkg-ci-publish.md) — still "Planned" in body; not addressed here.
- Distinct from the neural-search neighbours: m-dx-package-check / m-dx-package-test (v0.10.0) are the `check --package` / `test --package` commands this doc *calls*; m-dx-quality-monitor (v0.35.0) is eval-quality, not package quality.

## Verification Log

All run 2026-09-17 on `AILANG v0.39.4-10-g6f3a96683` against the live registry; the live validator reported `v0.39.4-11-g1f78cd2bc` (`curl https://registry.ailang.sunholo.com/version`).

| # | Claim | Command / evidence | Result |
|---|---|---|---|
| V1 | No published version has verified contracts | fetched `packages/<n>/<latest>/metadata.json` for all 53 packages; tabulated `validation.contracts_*` | `contracts_total = 0`, `contracts_verified = 0`, `contracts_skipped = 0` on 53/53 |
| V2 | Validator's per-file verify fails MOD010 on flat tarballs | extracted `sunholo/deontic/0.3.0/package.tar.gz`; `ailang verify --json --timeout 2s <abs>/settle.ail` | `exit=1`, `Error MOD010: module 'sunholo/deontic/settle' doesn't match file path …` |
| V3 | Validator decodes the wrong JSON shape | `cmd/registry-validator/validate.go:108-142` decodes `[]struct{Status}`; actual output is `{"file":…,"verified":7,…,"results":[…]}` | shape mismatch → `json.Unmarshal` error ignored → nothing counted |
| V4 | Contracts exist and verify under a package-aware run | `grep -rlE '(requires|ensures)' ailang-packages/packages` → 10 files (deontic, auth, billing-stripe, billing-service-api); `ailang verify --json --relax-modules --timeout 2s settle.ail` | `"verified": 7, "counterexample": 0, "total_exported": 7` |
| V5 | v1 interface hash is manifest-only | `internal/pkg/hasher.go:73-98` | folds name, edition, ailang, sorted `Exports.Modules`, sorted `Effects.Max`; no signatures |
| V6 | V2 has no production caller | `git grep -n "InterfaceHashV2" -- '*.go' \| grep -v _test \| grep -v hasher_v2.go` | only a comment in `internal/messaging/pkg_events.go:25` |
| V7 | `BuildCanonicalJSON` cannot find flat-layout modules | `internal/pipeline/canonical_json.go:18-25` joins `packageDir/<modulePath>.ail`; `ailang internal-dump-iface . sunholo/deontic/settle` in the flat dir vs a restaged `sunholo/deontic/settle.ail` layout | flat: `cannot read file "sunholo/deontic/settle.ail"`; restaged: full interface JSON (`capAt`, `daysLateAt`, … `pure: true`) |
| V8 | Blast radius of a V2 gate — **three runs** | (a) real flat tarballs: `internal-dump-iface` per exported module; (b) canonical restage + `ailang lock` (real registry dependency resolution) + dump per module; (c) for every (b) failure, `ailang lock && ailang check --package .` in the real flat layout | **(a) 0/53** — every flat package fails on V7's path mapping. **(b) 31/53 all modules OK** (incl. `ailang_parse`, whose tarball is already canonical, 56 modules); **3/53** genuine type errors (`http_helpers 0.1.4`, `registry_validator 0.1.1`, `testing_utils 0.1.1` — March-2026 `++`-on-string dialect); **19/53** fail only on intra-package `pkg/<self>/…` imports, because the self-package loader resolves *flat* (`tried …/src/types.ail, …/types.ail`) while the V2 builder needs *canonical* — no single layout satisfies both today (the M1 seam). **(c) 18/19** of those pass `check --package` in their real flat layout; the 19th, `motoko_ext_a2a 0.2.2`, fails current effect checking (May-2026 drift). **Bound:** post-M1 the refusal population is **≤ 4/53** (3 type + 1 effect), all published Mar–May 2026 and all already refused by the existing compile gate on republish; 49/53 are expected clean, which is what shadow mode (D6) confirms rather than assumes. Raw outputs: session scratchpad `blast.out`, `blast2.out`. |
| V9 | Cascade dormant | `gh pr list --state all --label cascade` in ailang-packages | 28 PRs, every `createdAt` = 2026-05-17 |
| V10 | A recent publish with dependents fired no cascade | index: `sunholo/daneel_ext_abi` (0.4.0, 2026-09-15) has 6 dependents (`daneel_ext_{help,activity,writer,design,calendar,search}`) | no cascade PR, no bump |
| V11 | `pkg quality` does not exist | `ailang pkg quality .` → `Error: unknown pkg command 'quality'`; `ailang pkg` lists 8 subcommands, none `quality`; `git grep -il "pkg quality"` in ailang → 0 design docs; in ailang-packages → AGENTS.md, SKILL.md, README.md, 2 package READMEs (commit `ea6b0b3`, 2026-09-08) | phantom confirmed |
| V12 | `PUB0xx` unallocated | `git grep -n '"PUB0' -- internal cmd` | empty |
| V13 | Unknown `[release]` section tolerated by shipped binaries | appended `[release] kind="fix" notes="test"` to a deontic copy; `ailang check --package .` | `✓ 9 files checked, all passed!` (BurntSushi `toml.Unmarshal`, non-strict, `manifest.go:226`) |
| V14 | No unused-declared-effect warning exists | `git grep -n -i "declared but not used\|unused effect\|EFF_UNUSED\|over-declared" -- internal/types internal/pipeline` | empty |
| V15 | Body-required effects are already computed | `internal/pipeline/validate_effects.go:305 collectRequiredEffects`, `:103 ValidateEffects` | exists; used for the "uses effects not declared" error |
| V16 | Human approval rail exists | `internal/coordinator/approval_checkpoint.go:71 type ApprovalCheckpoint`, `cmd/ailang/coordinator.go:55 case "approve"` | exists |
| V17 | `contract-regression` has no producer | `git grep -n "EmitContractRegression\|PkgMsgContractRegression" -- '*.go' \| grep -v _test` | schema + router arm + status only; no emitter |
| V18 | Router treats class A as auto-merge | `internal/coordinator/autonomy_router.go:33-46`; `U → ChangeClassC` at `:66` | confirmed |
| V19 | Index updater fields | `cmd/registry-validator/main.go:383+ tryUpdateIndex` updates `ContractsVerified, LastUpdated, AISummary, Tags, Effects, Stability` | confirmed (no release/measured fields yet) |
| V20 | ABI forces the nine-effect row | `ailang-packages/packages/motoko-ext-abi/*.ail:134-141` hook fields typed `! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream}`; `motoko-ext-fmt` functions: `! {Process}`, `! {FS}`, `! {FS, Process}` | confirmed |
| V21 | Provenance empty | `published_by` in 53 latest metadata.json | empty on 52/53 (`daneel` on `daneel_ext_abi`) |
| V22 | Published tarballs are flat | `tar xz` listing of deontic 0.3.0 | `_smoke.ail AGENT.md ailang.toml api.ail engine_test.ail engine.ail … settle.ail types.ail` at root |
| V23 | No changelogs | `find ailang-packages/packages -maxdepth 2 -iname 'CHANGELOG*' \| wc -l` | 0 (45 package dirs) |
| V24 | `ailang messages` has ack/send/reply, no approve; approval lives in `coordinator approve` | `ailang messages --help`; `ailang coordinator --help` | confirmed |
| V25 | Nil ceiling = unlimited | `internal/pkg/loader.go:282-285` (`maxEffects == nil → return nil`); index: `sunholo/logging` `effects: null`, its `ailang.toml` has no `[effects]` | confirmed; 11 packages declare `max = []` explicitly |
| V26 | CI publish never landed | `design_docs/implemented/v0_10_0/m-pkg-ci-publish.md` Status: Planned; `ls ailang-packages/.github` → no such directory | confirmed |
| V28 | `ailang messages ack` is not an approval | `cmd/ailang/messages_crud.go:143-228 runMessagesAck` → `store.MarkInboxMessageRead(msgID)` / `MarkAllInboxMessagesRead`; `cmd/ailang/coordinator_actions.go:17 coordinatorApprove` → `coordinator.ProcessApprovalRequest(ctx, &ApprovalParams{… ApprovedBy: "cli-user"})`; `internal/coordinator/approval_checkpoint.go:138 RequestApproval`, `:195 Approve(requestID, resolvedBy)` | ack only flips read state; the checkpoint transition exists only behind `coordinator approve`; approver identity is the literal `"cli-user"` |
| V29 | The validator never executes `_smoke.ail` today | `cmd/registry-validator/main.go:125-341 handlePublish` calls `runAilangCheck`, `runAilangVerify`, hashes, upload — no `RunSmokeInTempDir`; smoke runs in `cmd/ailang/pkg_publish.go:118 runPrePublishSmoke` (publisher) | confirmed; the server/attested split preserves this |
| V30 | `metadata.repository` coverage for M9's out-of-repo routing | `manifest.repository` non-empty across the 53 latest `metadata.json`; same field in `index.json` | **35/53** (not 39 as first drafted); the other 18 fall to the `pkg:<name>` inbox path |
| V31 | Superseding exists | `internal/messaging/pkg_status.go:75 func (s *Store) SupersedeOlderMessages(pkgName, newVersion string) (int, error)`; called from `cmd/ailang/pkg_publish.go` after emit | confirmed |
| V32 | V2 interface build is bounded | `internal/pkg/iface_subprocess.go:20 type PublishLimits {Overall, PerModule time.Duration; MaxExportedModules int}`; `DefaultPublishLimits()` at `:27` = 60s overall / 10s per module / 64 modules (`ailang_parse` exports 56 — raise or make server-configurable in M2); `BuildModuleIface` wraps the subprocess in `context.WithTimeout(ctx, lim.PerModule)` and `proctree.Kill`s on deadline (`:100-127`) | confirmed; Z3 verify has only the per-function `--timeout` today — the per-package wall cap is new (M2) |
| V27 | Registry composition | `index.json`: 53 packages, 373 versions; `stability`: experimental 34, unset 18, stable 1; `has_agent_doc` 29; `_smoke.ail` 11; packages with `*_test.ail` 8 | measured |

## References

- Parent doc's D7a refusal gate and V-log #37 (unmeasured until V8 here).
- ailang-packages `AGENTS.md` (2026-09-08): "`ailang publish --dry-run` checks packaging; it does not establish that tests passed or contracts were proved."
- Core backlog rows 2026-09-15: tool-name gate false positive; lock absolute paths; `[replace]` override — adjacent friction, out of scope.

## Future Work

- Registry backfill of v2 identity for historical versions (parent doc M9) once shadow data shows the refusal rate.
- Type-level scopes (`Net[scope=…]`) proving what `[effects.scopes]` declares.
- Signed publishing + publisher trust policy on top of the tier (deferred/m-pkg-trusted-autonomous-evolution) — including an authenticated approver identity for `coordinator approve` (today `"cli-user"`, V28) and verification of `attested` blocks.
- `ailang upgrade --safe|--compatible|--patch` selecting by banked tier/class rather than version arithmetic (m-pkg-install-latest's sketch).
- CI publish for ailang-packages running `pkg quality --json --strict` as the PR gate.
