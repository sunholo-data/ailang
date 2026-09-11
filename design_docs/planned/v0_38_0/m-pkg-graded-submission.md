# M-PKG-GRADED-SUBMISSION: Description Freshness Gates + Graded Package Review Lanes

**Status**: Planned
**Target**: v0.38.0
**Priority**: P1 (Medium-High — registry trust surface; nothing is blocked, but every published package is affected once shipped)
**Estimated**: 1 week (~5 days)
**Dependencies**: None (builds on shipped M-PKG-AUTONOMOUS-UPDATES change classes and the registry validator)
**Milestone ID**: M-PKG-GRADED-SUBMISSION
**Created**: 2026-09-11
**Source**: Mark's request — "when we publish packages the description must be updated to include any changes… differing grades of package submission — security patches go through more easily than if function signatures change, and package effect updates need a human review… enforcing best practices… minimising effects now we have more fine-grained controls, encourage AILANG features that make it unique."

---

## Premise Verification Log

Every load-bearing claim below was verified against the code on 2026-09-11 (repo @ v0.37.2). Reject-by-default reviewers: each row names the command and the evidence.

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| V1 | `description` in `ailang.toml` is parsed but **never consumed anywhere** — not in `metadata.json`, not in `index.json`, not by the validator | `grep -rn "Description" internal/ cmd/ --include="*.go"` → only `internal/pkg/manifest.go:135` (`PackageInfo.Description`) and unrelated structs (`internal/manifest/manifest.go`, builtins specs). `MetadataManifest` (`internal/pkg/registry_types.go:73-86`) and `IndexEntry` (`cmd/registry-validator/main.go:347-420`) carry no description field | **Confirmed** — description is a write-only field today |
| V2 | Registry validator gates today are exactly: API-key auth, manifest parse, vendor/name shape, exact-version immutability, provider-safe tool names, `ailang check --package .` compile (incl. effect ceiling), best-effort contract verification | Read `cmd/registry-validator/main.go:99-305` (`handlePublish` steps 0-11) and `cmd/registry-validator/validate.go:20-143` (`runAilangCheck`, `runAilangVerify`) | **Confirmed** — no metadata-hygiene, ordering, or review-lane gates exist |
| V3 | No **monotonic version** check on publish — only exact-version conflict (HTTP 409) | `handlePublish` step 4 (`main.go:167-175`) checks only "version already published". `lookupLatestVersion` (`validate.go:191`) is called only from `rewritePathDepsToRegistry` (`validate.go:159`), never for ordering. `semverTuple.gte` (`internal/pkg/version_compat.go:52`) is used only for `ailang` toolchain constraints | **Confirmed** |
| V4 | An **A/B/C change-class taxonomy already exists**: A = content-only (interface hash unchanged), B = additive (new module exports), C = breaking/effect-widening; plus `ProvenanceInfo.AutoApproved` ("true for class A") | `mapChangeClassToSchema` (`cmd/ailang/pkg_publish.go:532-555`, comment block 517-531); `ProvenanceInfo.ChangeClass`/`AutoApproved` (`internal/pkg/registry_types.go:97-101`); schema comment `internal/pubsub/publisher.go:72` | **Confirmed** — but it is computed **client-side, post-publish, for cascade messaging only**. The validator itself does not classify submissions |
| V5 | **Effect widening is already detectable**: `effectsWidened(old, new)` compares ceiling lists and forces class C | `cmd/ailang/pkg_publish.go:558-573` (mirrors unexported `messaging.effectsWidened`) | **Confirmed** |
| V6 | The validator **can** read the previous version's metadata server-side: it holds a GCS bucket handle and the metadata path scheme `packages/<vendor>/<name>/<version>/metadata.json` | `handlePublish` step 4 reads that exact path for the immutability check (`main.go:167`); client-side analogue `FetchMetadata` used at `cmd/ailang/pkg_publish.go:447` | **Confirmed** — the description-diff and effect-diff gates can run server-side without new infrastructure |
| V7 | The validator is **synchronous accept/reject**; no pending-review state exists | Read `handlePublish` end-to-end: every outcome is an immediate HTTP response (200/400/403/409/500) | **Confirmed** — graded human review requires a new pending state (this doc's main architectural decision) |
| V8 | Fine-grained/parameterised effects exist: `!{Rand[mode=seeded]}` etc. since v0.15.0 (M-EFFECT-REFINEMENT Phase 1), with bare forms desugaring to default modes | `docs/docs/guides/parameterised-effects.md`; parent doc `design_docs/planned/v1_0_0/m-effect-refinement.md` | **Confirmed** — gives the effect-minimality lint something concrete to encourage |
| V9 | A single-use HMAC approval-token mechanism exists and is reusable for publish approval | `internal/approvaltoken/token.go` (mint/verify, nonce single-use, TTL) | **Confirmed** |
| V10 | `PKG_*` structured code namespace is nearly empty — only `PKG_MANIFEST` is allocated | `grep -rn '"PKG_[A-Z_]*"' internal/ cmd/ --include="*.go"` → single hit `cmd/ailang/check_package.go:59` | **Confirmed** — proposed codes (`PKG_DESCRIPTION_STALE` etc.) are unallocated |
| V11 | `HasAgentDoc` is already detected and recorded (AGENT.md uploaded as first-class artifact) but is not gated | `handlePublish` steps 10-11 (`main.go:220, 255-269`) | **Confirmed** — promoting it to a score/badge is additive |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Grades are a pure function of (old metadata, new tarball): same inputs → same lane, same verdict. No wall-clock or ambient input in classification |
| A2: Replayability | +1 | Every verdict is recorded in `metadata.json` provenance (grade, evidence, approver), so any publish decision can be audited/replayed |
| A3: Effect Legibility | +2 | Core of the request: effect-ceiling widening forces human review; effect-minimality lint compares declared `[effects].max` against inferred export effects |
| A4: Explicit Authority | +1 | Human review is bound to single-use HMAC tokens (`internal/approvaltoken`); no ambient "admin" path |
| A5: Bounded Verification | +1 | All new gates are bounded, local validator checks (string diff, semver compare, set inclusion) — no open-ended AI judgement in the hot path |
| A6: Safe Concurrency | 0 | Index updates already use generation-retry (`updateIndex`); pending-state writes follow the same pattern |
| A7: Machines First | +1 | Structured `PKG_*` rejection codes + machine-readable grade in every response; agents can self-correct without parsing prose |
| A8: Minimal Syntax | 0 | No language syntax change. One optional manifest field (`[package] security_contact`) and one CLI flag |
| A9: Cost Visibility | +1 | Review cost becomes explicit per submission (lane A ≈ free, lane C = human time); publishers can see and reduce their grade |
| A10: Composability | +1 | Reuses the shipped A/B/C taxonomy, `approvaltoken`, messaging inboxes, and the existing validator pipeline rather than inventing parallel machinery |
| A11: Structured Failure | +1 | Rejections carry code + reason + remediation, not bare strings |
| A12: System Boundary | +1 | The pending-review boundary (untrusted submission → trusted registry) becomes an explicit, queryable state instead of an implicit accept |

**Net Score: +11** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — classification is a pure function of registry state + tarball
- [x] A3 (Effects): No hidden side effects — pending writes and notifications are logged; the gate makes effect changes *more* visible, never less
- [x] A4 (Authority): No ambient access granted — approval requires a scoped single-use token; validator gains no new ambient authority
- [x] A7 (Machines First): Not optimizing for human convenience — the fast lane exists for *machine-verifiable* (interface-unchanged) changes, and all verdicts are machine-readable

---

## Problem Statement

The AILANG registry is the trust root for the package ecosystem, but its submission gate checks only that code **compiles** — not that the submission is **honest, described, or proportionally reviewed**.

**Current State (all verified — see Premise Verification Log):**

1. **`description` is a write-only field.** It is parsed from `ailang.toml` (V1) and then dropped: never stored in `metadata.json`, never indexed, never displayed, never compared. A package can publish ten versions with a stale or empty description and nothing notices. (The v0.10.0 stale-metadata incident — `sunholo/logging` published with a new effect set while the index showed the old one — was the same class of bug; M-PKG-CI-PUBLISH fixed the index-update half, but there is still no freshness *gate*.)
2. **Every publish takes the same lane.** A one-line security patch with an unchanged interface goes through exactly the same (non-)scrutiny as a release that removes exported functions and widens the effect ceiling from `{FS}` to `{FS, Net, Process}`. The A/B/C change-class machinery already exists (V4) but runs *after* publish, for dependent-notification routing — it gates nothing.
3. **No review-lane proportionality.** The validator is synchronous accept/reject (V7). There is no pending state, so "this needs a human" can only mean "reject", which pushes breaking-change authors to split/sneak changes or just not document them.
4. **Metadata hygiene is unenforced.** No monotonic-version check (V3), no license requirement, no `ai_summary` requirement, no description-presence check.
5. **No incentive gradient toward AILANG's differentiators.** Contracts verified, effect minimality, parameterised effects (`Rand[mode=seeded]`, V8), pure-function ratio — all detectable at validation time, all currently invisible in the index. Publishers get no signal for doing the things that make AILANG packages uniquely trustworthy.

**Impact:**
- **Consumers** (humans and agents choosing dependencies) cannot trust registry descriptions; they must diff tarballs by hand.
- **The cascade system** (M-PKG-AUTONOMOUS-UPDATES) auto-bumps dependents on class A; if a breaking change is mis-laned, the cascade propagates breakage autonomously.
- **The ecosystem narrative** — "AILANG packages are effect-audited and contract-verified" — is currently aspirational rather than enforced.

## Goals

**Primary Goal:** Make every registry submission pass through a deterministic, proportionate review lane — where the description must reflect the change, effect-ceiling updates require human review, and machine-verifiable (interface-unchanged) patches publish fast.

**Success Metrics:**
- 100% of published versions (post-launch) carry a description that differs from the previous version's, or an explicit recorded attestation of no user-visible change
- 100% of effect-ceiling widenings carry a human-approval provenance record
- 0 class-C (breaking) publishes without review; class-A median publish latency unchanged (< ~30s validator time)
- Registry index exposes `description`, `change_class`, and a best-practice score for every package version
- Rejections are actionable: every `PKG_*` rejection names the failing check and the fix

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Grade taxonomy reuses shipped A/B/C classes (A=content-only, B=additive, C=breaking/effect-widening) | A parallel taxonomy would fork the cascade's semantics; reusing keeps one source of truth | human | design | high |
| Lane C requires **pre-approval** (publish → pending → human approve → finalize), not post-hoc flagging | Post-hoc flagging lets breaking code live in the registry before review; pre-approval needs a new pending state | human | design | high |
| Description freshness is a **hard gate** (reject identical description on any new version) with an explicit `--description-unchanged` attestation escape hatch, recorded in provenance | Hard gate = ecosystem-wide behavior change; escape hatch prevents gaming-by-noise while keeping attestations auditable | human | design | med |
| Effect-minimality and best-practice checks are **advisory scores** (badges in index), not rejections, in v1 | Blocking on lint-style findings would strand legitimate packages; scores create the incentive gradient without the cliff | human | design | med |
| Approval authority reuses `internal/approvaltoken` (single-use HMAC) delivered via the existing coordinator approval queue | One authority mechanism, already audited for the notification approve/deny flow | human | design | med |
| Grade is computed **server-side** in the validator (from old metadata vs new tarball), not trusted from the client | The publisher is the untrusted party; client-declared grades are self-attestation | compiler (forced by V6/V7 architecture) | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Lane taxonomy = A/B/C reuse (per table above) — human sign-off
- [ ] Pending-state storage: GCS `pending/<vendor>/<name>/<version>/` prefix + `pending.json` index vs. Firestore collection — human sign-off (GCS keeps registry self-contained; Firestore is queryable)
- [ ] Description-gate strictness for **first publishes** (no previous version to diff): require non-empty description ≥ 40 chars? — human sign-off
- [ ] Who holds lane-C approval authority per namespace (registry admins only, or namespace owners too) — human sign-off
- [ ] `--description-unchanged` attestation wording recorded in provenance — human sign-off

## Solution Design

### Overview

Extend the registry validator from a synchronous compile-checker into a **grading pipeline**:

```
tarball → [existing gates: manifest, immutability, tool-names, compile, contracts]
        → NEW: metadata hygiene gates (description present/fresh, license, monotonic version)
        → NEW: server-side change classification (A/B/C from old metadata vs new)
        → NEW: lane routing
              A (content-only, incl. security patches)     → publish immediately, provenance.AutoApproved=true
              B (additive interface)                        → publish immediately + steward notification (async review, revert-by-unpublish)
              C (breaking OR effect widening)               → pending state; finalize only with single-use approval token
        → NEW: best-practice scoring (advisory) written into metadata.json + index.json
```

The publisher experience stays one command (`ailang publish`); grades appear in the CLI output and every rejection carries a `PKG_*` code with a remediation line.

### Architecture

**Components:**

1. **Metadata plumbing** (`internal/pkg/registry_types.go`, `cmd/registry-validator/main.go`)
   Add `Description string \`json:"description"\`` to `MetadataManifest` and to `IndexEntry`; populate in `handlePublish` step 10 and `tryUpdateIndex`. This is the prerequisite for every other gate — the description must be *stored* before it can be *diffed*. (V1/V6)

2. **Metadata hygiene gates** (`cmd/registry-validator/validate.go`, new `validate_metadata.go`)
   - `PKG_DESCRIPTION_MISSING` — first publish with empty/short description → 400
   - `PKG_DESCRIPTION_STALE` — subsequent publish whose description is byte-identical to the previous version's stored description → 400, unless the request carries the `X-Description-Unchanged: <reason>` attestation (emitted by `ailang publish --description-unchanged "..."`), which is recorded in `ProvenanceInfo`
   - `PKG_LICENSE_MISSING` — no `[package] license` → 400 (warning-only during a one-cycle grace, mirroring the `--allow-dotted-tool-names` precedent at `pkg_publish.go:27`)
   - `PKG_VERSION_REGRESSION` — new version ≤ latest published version (semver compare via `semverTuple`, V3) → 400
   - `AILANG_VERSION_MISSING` stays a client-side warning (current behavior, `pkg_publish.go:57-64`) — promoted to validator warning in the response body

3. **Server-side classification** (`cmd/registry-validator/classify.go`, new)
   Port the *semantics* of `mapChangeClassToSchema` (`pkg_publish.go:532`) server-side, but compute from authoritative inputs: previous `metadata.json` (fetched via bucket handle, V6) vs. the submitted manifest/interface hash. Effects widened (`effectsWidened`, V5) ⇒ **forced C regardless of interface hash** — this is the "package effect updates need a human review" requirement. Classification result stored in `ProvenanceInfo.ChangeClass` (field already exists, V4).

4. **Lane routing + pending state** (`cmd/registry-validator/review.go`, new)
   - **Lane A**: publish as today; `ProvenanceInfo.AutoApproved=true`.
   - **Lane B**: publish; emit a `review-notice` message to the package inbox + registry-stewards inbox (existing messaging store, cf. `emitPublishMessages`). Revert path = existing `ailang unpublish`.
   - **Lane C**: upload tarball + `pending.json` under `pending/<vendor>/<name>/<version>/`; emit an approval request to the coordinator approval queue; respond **HTTP 202** with `{ "status": "pending_review", "code": "PKG_PENDING_REVIEW", "grade": "C", "reasons": [...] }`. A new validator endpoint `POST /review/finalize` (guarded by `approvaltoken.Verify`, V9) moves the artifacts to the live path and writes `ProvenanceInfo.ApprovedBy/ApprovedAt`. Expiry: pending entries auto-expire after 14 days (delete + notify).
   - Client: `ailang publish` prints the 202 body verbatim ("submitted for human review; you will be notified via the package inbox"). No polling logic client-side in v1.

5. **Best-practice scoring — advisory** (`cmd/registry-validator/score.go`, new)
   A deterministic 0–100 score + per-signal breakdown, stored in metadata/index and surfaced on the package explorer website (M-PKG-EXPLORER-WEBSITE, shipped). Signals (each verified detectable at validation time):
   - Contracts: `contracts_verified / contracts_total` already computed (step 8, V11) — weight 25
   - **Effect minimality**: union of effect rows inferred on exported functions during `ailang check --package` vs. declared `[effects].max`; penalize each declared-but-unused ceiling entry; bonus for parameterised narrowing (e.g. `Rand[mode=seeded]` over bare `Rand`, V8) — weight 25
   - Description quality: present, ≥ 40 chars, changed-this-version — weight 15
   - `ai_summary` + tags present — weight 10
   - `AGENT.md` present (already detected, V11) — weight 10
   - `stability` level declared — weight 5
   - Purity ratio (share of exported functions with empty effect row) — weight 10
   No rejection is ever based on score in v1.

### Implementation Plan

**Phase 1: Plumbing + hygiene gates** (~1.5 days)
- [ ] Add `Description` to `MetadataManifest` + `IndexEntry`; populate in publish + index update (both new-entry and existing-entry paths in `tryUpdateIndex`)
- [ ] Backfill note: existing index entries get `description: ""` until next publish (documented, not silently fabricated)
- [ ] Implement `validate_metadata.go`: description-present, license-present (grace-warning), monotonic-version
- [ ] Client: surface new `PKG_*` rejection bodies verbatim (already the behavior for 400s, `pkg_publish.go:294-295`)
- [ ] Unit tests per gate; validator integration test with fixture packages

**Phase 2: Description freshness + classification** (~1.5 days)
- [ ] Previous-metadata fetch helper (bucket read of `packages/.../<prev>/metadata.json`; prev = highest version < new)
- [ ] `PKG_DESCRIPTION_STALE` gate + `X-Description-Unchanged` attestation → `ProvenanceInfo`
- [ ] Client flag `ailang publish --description-unchanged "<reason>"` (requires non-empty reason)
- [ ] Server-side `classify.go` (port A/B/C semantics; forced-C on effect widening)
- [ ] Tests: identical-description rejection; attestation path; each class transition (A/B/C, widening-forced-C)

**Phase 3: Lane routing + human review** (~1.5 days)
- [ ] Pending-state write/read (`pending/` prefix), 202 response shape, `PKG_PENDING_REVIEW`
- [ ] `POST /review/finalize` + approvaltoken verification + provenance write
- [ ] Approval-queue message emission; expiry sweeper (validator startup ticker)
- [ ] `ailang publish` CLI: render 202 pending response
- [ ] E2E test: C-submission → pending → approve → finalized + indexed; reject → deleted

**Phase 4: Scoring + surfacing** (~0.5 day)
- [ ] `score.go` with the 7 signals; store in `metadata.json` + `index.json`
- [ ] Expose via registry API (`handlers_api.go`) for the website
- [ ] Docs: `docs/docs/guides/package-publishing.md` update (lanes, gates, score)

### Files to Modify/Create

**New files:**
- `cmd/registry-validator/validate_metadata.go` (~200 LOC) — hygiene gates
- `cmd/registry-validator/classify.go` (~150 LOC) — server-side A/B/C classification
- `cmd/registry-validator/review.go` (~250 LOC) — pending state, finalize endpoint, expiry
- `cmd/registry-validator/score.go` (~200 LOC) — best-practice scoring
- `cmd/registry-validator/classify_test.go`, `review_test.go`, `validate_metadata_test.go`, `score_test.go` (~600 LOC total)

**Modified files:**
- `internal/pkg/registry_types.go` (+15 LOC) — `Description` on `MetadataManifest`/`IndexEntry`, `Score` on metadata, pending provenance fields
- `cmd/registry-validator/main.go` (+80 LOC) — wire gates into `handlePublish` between steps 5.5 and 6 (hygiene) and between 8 and 9 (classification/routing); populate description in `tryUpdateIndex`
- `cmd/registry-validator/handlers_api.go` (+30 LOC) — expose score/description in API responses
- `cmd/ailang/pkg_publish.go` (+40 LOC) — `--description-unchanged` flag → header; render 202 pending body
- `internal/pkg/version_compat.go` (+10 LOC) — exported `CompareSemver` helper (currently only unexported `gte`)
- `docs/docs/guides/package-publishing.md` (+60 LOC) — lanes, gates, attestation, score

## Examples

### Example 1: Security patch (fast lane)

`sunholo/auth` 1.2.3 → 1.2.4: one-line fix, interface hash unchanged, effects unchanged, description updated.

**Before:** publishes silently; description not stored; dependents notified with no class context.
**After:**
```
$ ailang publish
Publishing sunholo/auth@1.2.4...
✓ Grade A (content-only — interface unchanged) → fast lane
✓ Description freshness: updated ("...fixes token replay in refresh flow")
✓ Published. Dependents notified (class A, auto-bump eligible).
```

### Example 2: Effect widening (human review lane)

`sunholo/logging` 0.3.0 → 0.4.0: adds `Net` to `[effects].max`.

**Before:** publishes if it compiles; dependents' cascades treat it as class C after the fact.
**After:**
```
$ ailang publish
Publishing sunholo/logging@0.4.0...
⚠ Grade C (effect ceiling widened: +Net) → human review required
→ Submitted for review (PKG_PENDING_REVIEW). Reviewers notified via approval queue.
  Track: ailang messages list --inbox pkg:sunholo/logging
```
On approval, the validator finalizes and the publisher receives an inbox message; on rejection the pending entry is deleted with the reviewer's reason.

### Example 3: Stale description (hard gate)

```
$ ailang publish
Publishing sunholo/json@2.1.0...
✗ 400 validation failed:
  PKG_DESCRIPTION_STALE: description is identical to version 2.0.0.
  Update ailang.toml [package] description to reflect this release's changes,
  or attest no user-visible change:
    ailang publish --description-unchanged "internal refactor only"
```

### Example 4: Best-practice score (advisory)

```
✓ Published sunholo/auth@1.2.4 (grade A)
  Registry score: 82/100
    contracts 25/25 · effects 17/25 (declared ceiling {IO,FS,Net}; exports infer {IO,FS})
    description 15/15 · ai_summary 10/10 · agent_doc 0/10 · stability 5/5 · purity 10/10
  Tip: 2 unused effect ceiling entries — narrowing [effects].max raises score and
       reduces consumer-side capability prompts.
```

## Success Criteria

- [ ] `description` stored in `metadata.json` and `index.json` for every new publish (acceptance: publish fixture, `curl` both artifacts)
- [ ] Identical-description republish rejected with `PKG_DESCRIPTION_STALE`; attestation path succeeds and records the reason in provenance (integration test)
- [ ] Effect-ceiling widening forces grade C and lands in pending state; finalize requires a valid single-use approval token; forged/reused/expired tokens rejected (integration test reusing `approvaltoken` test vectors)
- [ ] Grade A publishes with no added latency (>95th percentile validator time within +10% of baseline)
- [ ] Monotonic version enforced: publishing 1.0.0 when 1.2.0 exists → `PKG_VERSION_REGRESSION`
- [ ] Score present in registry API responses; no publish is rejected on score (assertion test)
- [ ] All existing validator tests pass unchanged (backward compatibility: pre-existing packages without stored description get one grace publish with a warning, then the gate applies)
- [ ] All tests passing (`make test`)
- [ ] Documentation updated (`docs/docs/guides/package-publishing.md`)

## Testing Strategy

**Unit tests:**
- Each hygiene gate (table-driven: present/missing/stale/attested description; license; version ordering incl. pre-release-free semver)
- `classify.go`: all A/B/C transitions + forced-C-on-widening matrix (old/new effect sets, export sets, interface hashes)
- `score.go`: golden-score fixtures for each signal

**Integration tests:**
- Full `handlePublish` against in-process bucket fake: lanes A/B/C end-to-end
- Pending lifecycle: submit → list pending → approve → finalized + indexed; submit → reject → deleted; expiry sweeper
- Backward compat: package with no prior stored description → one grace warning publish

**Manual testing:**
- Real publish of a test package to the staging registry bucket through all three lanes
- `ailang publish --dry-run` unchanged behavior verified

## Deferred Decisions

- Pending-state storage medium (GCS prefix vs. Firestore) — **human at design freeze** (listed above); implementer builds behind a `PendingStore` interface either way
- Exact score weights per signal — **agent may choose** within the documented structure; weights live in one const block for easy tuning
- Grace-period length for `PKG_LICENSE_MISSING` (one minor version vs. two) — **human at review**
- Whether lane-B steward notification also opens a GitHub issue on the package repo — **agent may choose** (message inbox is the baseline)
- Pending-entry retention UI on the dashboard — **human at review** (out of scope for v1; API suffices)

## Non-Goals

- **Function-level interface diffing** (distinguishing "added function" from "removed function" inside an unchanged module list). The existing taxonomy deliberately treats this conservatively as C (`pkg_publish.go:549-554`); refining it is a separate design building on interface-file diffing.
- **Namespace ownership / publisher ACLs** — still "accept all publishers" (validator step 5). Grading assumes the current trust model; per-namespace ownership is M-PACKAGE-PROTOCOL-MANIFESTS territory.
- **AI-judged review** (LLM reading the diff to auto-approve lane C). Deliberately excluded: lane C means *human*. An AI-assist summarizer for the reviewer is future work.
- **Retroactive re-grading** of already-published versions.
- **Blocking on best-practice score.** Advisory only in v1.

## Timeline

**Week 1** (~5 days):
- Phase 1: plumbing + hygiene gates (1.5d)
- Phase 2: description freshness + classification (1.5d)
- Phase 3: lanes + human review (1.5d)
- Phase 4: scoring + docs (0.5d)

**Total: ~5 days, single milestone, one release.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hard description gate annoys publishers into noise diffs ("updated description.") | Med | Attestation escape hatch is *easier* than gaming, and attestations are recorded in provenance — auditable, shaming-by-transparency; score rewards real descriptions |
| Pending state adds validator complexity/failure modes (orphaned pendings) | Med | 14-day expiry sweeper; pending artifacts isolated under `pending/` prefix; finalize is idempotent |
| Lane-C latency stalls urgent breaking fixes (e.g. yanked-function security issue) | Med | Approval tokens can be minted by any registry admin from the approval queue in minutes; `ailang unpublish` remains the emergency brake and is *not* gated |
| Classifier misgrades (e.g. interface hash churn from comment-only changes) | Low | Interface hash is already the cascade's source of truth (V4); if it changes the change is, by definition, interface-visible. Misgrade direction is conservative (toward review), never toward fast-lane |
| Backward compatibility: old clients hitting new gates | Low | All new rejections are 400/202 with `PKG_*` codes; old `ailang publish` prints server bodies verbatim today (`pkg_publish.go:294-295`), so even old binaries fail *legibly* |

## Related Documents

**Implemented (may inform design):**
- [M-PKG-AUTONOMOUS-UPDATES](../../implemented/v0_10_0/m-pkg-autonomous-updates.md) — source of the A/B/C change-class taxonomy, cascade envelopes, `ProvenanceInfo` fields this design reuses
- [M-PKG-CI-PUBLISH](../../implemented/v0_10_0/m-pkg-ci-publish.md) — registry validator index-update fixes; the stale-description incident this design closes the gate on
- [M-PKG-ECOSYSTEM-STATUS](../../implemented/v0_10_0/m-pkg-ecosystem-status.md) — post-publication audit; baseline "what's stable" list
- [M-DX-PKG-CHECK](../../implemented/v0_10_0/m-dx-package-check.md) — package-level type checking the validator's compile gate runs

**Planned (check for overlap):**
- [M-PACKAGE-PROTOCOL-MANIFESTS](../m-package-protocol-manifests.md) — repo-declared agent protocols and the "governed thing controls the manifest" authority problem. **Distinct**: that doc governs *what a package repo may declare to agents*; this one governs *what the registry requires at submission*. Shared boundary: namespace authority (Design Freeze item 5) must stay consistent with its ACL conclusions.
- [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) — parameterised effects roadmap; the effect-minimality score rewards adopting its narrowing modes but does not depend on unshipped phases.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- `cmd/registry-validator/main.go` — current publish pipeline (steps 0–11)
- `cmd/ailang/pkg_publish.go` — client publish flow, `mapChangeClassToSchema`, `effectsWidened`
- `internal/pkg/registry_types.go` — `PackageMetadata`, `ProvenanceInfo`, `IndexEntry`
- `internal/approvaltoken/` — single-use HMAC approval tokens
- [package-publishing guide](/docs/guides/package-publishing) — publisher-facing docs to update
- Prior art: crates.io (description/license required at publish), PyPI (metadata 2.x validation), Go modules (monotonic pseudo-versions, immutable versions), npm (2FA-graded publish lanes)

## Future Work

- Function-level interface diffs → split lane C into C-additive / C-breaking with different review depth
- AI-generated review briefs for lane-C approvers (diff summary, contract deltas) — assist, never decide
- Score-gated registry features (e.g. "featured" requires score ≥ 70) once the score has calibrated for a release or two
- Signed attestations (sigstore-style) replacing HMAC tokens for cross-org registries
- Description freshness extended to `ai_summary` (currently ungated)

---

**Document created**: 2026-09-11
**Last updated**: 2026-09-11
