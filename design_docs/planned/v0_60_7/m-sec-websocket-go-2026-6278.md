# Security: bump gorilla/websocket past GO-2026-6278 (weak mask-key PRNG)

**Status**: Planned
**Target**: v0.60.7
**Priority**: P0 (High) — red REQUIRED CI gate on `dev` blocks all merges; security exposure on a production network path
**Estimated**: 0.5 day (bump + local verify + PR/CI watch)
**Dependencies**: None

> **Scope isolation (per triage directive):** this doc covers ONLY the
> govulncheck GO-2026-6278 red at `dev` `d3c2c785` (CI run 45289159090). It does
> not absorb or depend on the Windows compile request
> `inbox_1795447998263_b0372c15` or the pipeline package timeout — those are
> separate triage threads and must stay in separate PRs.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a dependency version bump with zero AILANG-language-surface changes,
so most axioms are untouched (scored 0, not "don't care" — each 0 below is a
verified no-impact statement, not an omission).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | WS masking is transport-layer entropy (random per frame either way); AILANG's execution traces and semantic determinism are untouched. |
| A2: Replayability | 0 | No trace, effect-log, or replay-surface change. |
| A3: Effect Legibility | 0 | No signature or effect-row changes; the fix is internal to the dependency. |
| A4: Explicit Authority | 0 | No capability or authority-surface change. |
| A5: Bounded Verification | 0 | Verification cost unchanged; the vuln gate keeps running on every code PR. |
| A6: Safe Concurrency | 0 | No concurrency-surface change in AILANG code. |
| A7: Machines First | +1 | Fixes the finding for real instead of narrating it away in an allowlist: the security posture stays machine-decidable (govulncheck green, `allow: []`), and a green REQUIRED gate unblocks the autonomous merge loop. |
| A8: Minimal Syntax | 0 | No syntax involved. |
| A9: Cost Visibility | 0 | No resource-cost change. |
| A10: Composability | 0 | Module swap is drop-in (API identical). |
| A11: Structured Failure | 0 | No error-path changes. |
| A12: System Boundary | +1 | Hardens a real boundary crossing: outbound WS frames on the `streamws` dial path get cryptographically strong mask keys, restoring the RFC 6455 protection against on-path intermediaries. |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (mask-key entropy is transport noise, deliberately out of the determinism surface)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): machine-verifiable fix, not a human narrative

## Problem Statement

The `govulncheck (vuln gate)` CI job went red on `dev` at `d3c2c785`
(CI run 45289159090, job 138093356254), reporting **GO-2026-6278**
(alias GHSA-w67g-5rqw-f597): *"Gorilla WebSocket Uses Cryptographically Weak
PRNG for WebSocket Mask Key"*.

**Current State:**

- `go.mod:22` pins `github.com/gorilla/websocket v1.5.3` as a direct
  dependency. The advisory's affected ranges are
  `[0, 1.5.1)` and `[1.5.3, 1.5.4-0.20240701034025-d67f41855da4)` —
  **v1.5.3 sits in the second range**: the v1.5.3 release itself reintroduced
  a weak PRNG for mask-key generation, and upstream reverted to `crypto/rand`
  in commit `d67f41855da4` ("Use crypto/rand for mask key", 2024-07-01).
  (Consequence: v1.5.1 and v1.5.2 are *not* affected — the defect is a v1.5.3
  regression, not a long-standing one.)
- The finding is **reaching**, not module-only: AILANG calls affected symbols
  in non-test production code — `websocket.Dialer` at
  `internal/platform/streamws/streamws.go:39` and
  `internal/executor/codex/daemon_transport.go:29`, `Upgrader`/
  `Conn.WriteMessage` across `internal/websocket` and `internal/apiserver`
  (328 call sites repo-wide). With `.govulncheck-allow.yml` at `allow: []`,
  the filter exits 1.
- The gate is a **REQUIRED dev-gate context** (per
  [m-v1-release-gates.md](../v1_0_0/m-v1-release-gates.md)), so a red here
  blocks every merge to `dev`.

**Impact:**

- Real security exposure where AILANG acts as a WS **client**: RFC 6455
  masking exists so intermediaries (proxies, middleboxes) cannot read or
  tamper with frames in flight. Predictable mask keys (weak PRNG) defeat that
  on the `streamws` path, which dials **remote** endpoints through
  caller-supplied dialers. The `codex` daemon transport dials a localhost
  unix socket only — negligible there, but the same code path.
- Every PR to `dev` is blocked by the red required gate until this is fixed
  or allowlisted.

## Goals

**Primary Goal:** Turn `govulncheck (vuln gate)` green on `dev` by moving
`github.com/gorilla/websocket` to the upstream fixed version — with the
smallest possible diff (go.mod + go.sum + changelog, zero `.go` changes) and
zero allowlist suppression.

**Success Metrics:**
- Affected-version pin: `v1.5.3` → `v1.5.4-0.20240701034025-d67f41855da4`
  (1 line in `go.mod`, 2 lines in `go.sum`)
- `govulncheck ./...` reports zero findings for GO-2026-6278 (was: reaching)
- Filter exit code: 1 → 0 with `.govulncheck-allow.yml` still `allow: []`
- All REQUIRED CI contexts green on the fix PR

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Bump **up** to the upstream fix pseudo-version `v1.5.4-0.20240701034025-d67f41855da4`, not **down** to unaffected v1.5.1/v1.5.2 | A downgrade would also clear the advisory (both are outside the affected ranges) but reverts upstream's own head and re-opens everything fixed between v1.5.2 and v1.5.3; the pseudo-version is the advisory's canonical `fixed` event | human | design | low |
| D2: **No allowlist entry** for GO-2026-6278 | The allowlist precedent (nine Ollama entries, since removed) was reserved for advisories with **no upstream fix**; here a fix exists and a production network path is exposed — suppression would trade a one-line diff for 90 days of real exposure | human | design | low |
| D3: Keep the pseudo-version until upstream tags a release; swap pseudo-version → tag when one exists | `@latest` still resolves to affected v1.5.3 (no tagged release contains the fix), so a tag cannot be pinned today; a tagged pin is preferable long-term | agent | runtime (whenever a tag lands) | low |
| D4: Fix rides to `dev` as its own PR, isolated from the Windows compile request and the pipeline package timeout | Mixed PRs couple a P0 security gate-red to unrelated red threads and make bisection/evaluation ambiguous | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — bump to `v1.5.4-0.20240701034025-d67f41855da4` (decided in this doc)
- [x] D2 — no allowlist suppression (decided in this doc)
- [x] D4 — isolated PR (decided in this doc)
- [ ] D3 stays open by design (runtime deadline, agent-resolved; tracked as a follow-up note in the changelog entry and by the post-release skill's allowlist-expiry check pattern)

## Solution Design

### Overview

A single dependency-version move. `go get` the fixed pseudo-version, tidy,
verify locally, PR. No AILANG source changes: the upstream fix commit touches
only `conn.go` (+10/−3) and `prepared_test.go` (+7/−2) *inside* the module —
the exported API surface AILANG compiles against is unchanged.

### Architecture

**The defect and the fix, precisely:**

- gorilla/websocket v1.5.3 generates the per-frame 4-byte mask key with a
  cryptographically weak PRNG. Mask keys must be unpredictable to on-path
  observers; predictable keys let an intermediary unmask (read) and forge
  client→server frames.
- Upstream fix commit `d67f41855da42d7bccd9ef050c49f7e54e783b95` ("Use
  crypto/rand for mask key") restores strong entropy. It is published on the
  Go module proxy as pseudo-version `v1.5.4-0.20240701034025-d67f41855da4`
  and is the advisory's exact `fixed` event. **No tagged release contains
  the fix** — `@latest` still resolves to affected v1.5.3 — so the
  pseudo-version is the only pinnable fixed version today.
- AILANG's exposure is client-side (clients mask; servers don't):
  `internal/platform/streamws` dials remote endpoints (real exposure),
  `internal/executor/codex/daemon_transport.go` dials localhost unix (not
  exploitable, same code path).

**Components (of the change):**
1. **go.mod / go.sum**: the version move itself.
2. **Verification harness**: existing CI (`govulncheck (vuln gate)` +
   `tools/govulncheck-filter` + `.govulncheck-allow.yml`) — unchanged; it is
   both the thing that caught this and the thing that proves the fix.
3. **Regression suite**: the existing websocket tests across
   `internal/websocket`, `internal/platform/streamws`, `internal/apiserver`,
   `internal/effects`, `internal/executor/codex`, `internal/server` — they
   compile and run against the new module version, which is the entire
   behavioral risk surface of a drop-in swap.

### Implementation Plan

**Phase 1: bump + local verification** (~1.5 hours)
- [ ] `go get github.com/gorilla/websocket@v1.5.4-0.20240701034025-d67f41855da4 && go mod tidy`
- [ ] Confirm `go list -m github.com/gorilla/websocket` reports the pseudo-version; confirm the diff is exactly `go.mod` + `go.sum`
- [ ] `go build ./...` (compile proof of API compatibility)
- [ ] Reproduce the gate locally, before-and-after style:
      `go install golang.org/x/vuln/cmd/govulncheck@v1.3.0` (the CI pin),
      `go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter`,
      `govulncheck -format json ./... | ./bin/govulncheck-filter` — expect
      exit 0 and no GO-2026-6278 in findings
- [ ] `make test` (full suite; the websocket-consuming packages enumerated
      above are the regression surface)
- [ ] `make fmt` / `make lint` for gate parity

**Phase 2: PR + CI** (~0.5 day, mostly waiting)
- [ ] PR to `dev` containing only `go.mod`, `go.sum`, and the changelog entry
- [ ] The `changes` classify job puts `go.mod` on the code lane → full
      matrix, including `govulncheck (vuln gate)` (V12)
- [ ] Watch the PR's required contexts go green: `test`, `checks`, `lint`,
      `build`, `test-windows`, `govulncheck (vuln gate)`, `float determinism
      (arm64)`; confirm the vuln gate itself is green from a real
      `pull_request` event
- [ ] Merge; confirm the next `dev` push shows the gate green

**Phase 3: documentation + follow-up** (~15 min)
- [ ] Changelog `Security` entry in `changelogs/v0.32-current.md`, following
      the v0.53.1 `x/net` precedent (version range, advisory IDs, what the
      fixed version picks up, why the gate was red)
- [ ] Note in the entry: the pin is a pseudo-version; swap to the first
      tagged release containing the fix (≥ next tag after v1.5.3) when one
      exists — picked up by the next dependency sweep

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `go.mod` — 1 line: `github.com/gorilla/websocket v1.5.3` → `v1.5.4-0.20240701034025-d67f41855da4`
- `go.sum` — 2 lines: the module's zip + go.mod hashes move to the new version
- `changelogs/v0.32-current.md` — Security entry (~10 lines)

## Examples

### Example 1: the entire code change

**Before** (`go.mod:22`):
```
github.com/gorilla/websocket v1.5.3
```

**After:**
```
github.com/gorilla/websocket v1.5.4-0.20240701034025-d67f41855da4
```

**After** (local gate, executor transcript):
```
$ go list -m github.com/gorilla/websocket
github.com/gorilla/websocket v1.5.4-0.20240701034025-d67f41855da4
$ govulncheck -format json ./... | ./bin/govulncheck-filter
(exit 0, GO-2026-6278 absent)
```

## Success Criteria

- [ ] `go list -m github.com/gorilla/websocket` = `v1.5.4-0.20240701034025-d67f41855da4`
- [ ] Local `govulncheck -format json ./... | ./bin/govulncheck-filter` exits 0; no GO-2026-6278 finding (acceptance: the same command at the pre-bump commit exits 1 with a reaching GO-2026-6278 finding)
- [ ] `.govulncheck-allow.yml` still `allow: []` — the gate is green because the vuln is gone, not suppressed
- [ ] `make test` green; `make fmt`/`make lint` green
- [ ] PR diff is exactly 3 files: `go.mod`, `go.sum`, changelog — zero `.go` changes
- [ ] All REQUIRED contexts green on the PR from real `pull_request` events, including `govulncheck (vuln gate)`
- [ ] Documentation updated: changelog Security entry (v0.53.1 x/net format)
- [ ] PR is isolated from the Windows compile request and the pipeline package timeout (no unrelated hunks)

## Testing Strategy

**Unit tests:**
- None new. Mask-key entropy is upstream's own tested behavior (the fix
  commit extends `prepared_test.go`); AILANG must not re-test its dependency's
  internals.

**Integration tests:**
- The full existing suite (`make test`) recompiled against the new module
  version is the integration proof — especially `internal/websocket`,
  `internal/platform/streamws` (dial/accept round-trips),
  `internal/apiserver` (`routes_ws*`), `internal/effects`
  (`stream_ws`/`stream_bridge`), `internal/executor/codex`
  (`daemon_transport`).

**Regression-surface tests:**
- Conflict Surface not required (no parser/typechecker/effect-row surface
  touched); the regression surface *is* the suite above, already run by
  `make test` and CI.

**Manual/gate testing:**
- The before/after govulncheck transcript in Phase 1 is the acceptance
  evidence; CI's vuln gate re-runs the identical command on the PR.

## Verification Log

| # | Claim | Evidence | Status |
|---|---|---|---|
| V1 | `go.mod` pins `github.com/gorilla/websocket v1.5.3` as a direct dependency; `go.sum` carries exactly one version (2 lines, zip + go.mod hashes) | `grep -n gorilla/websocket go.mod` → line 22, main require block; `go.sum:135-136` | Confirmed |
| V2 | GO-2026-6278 (= GHSA-w67g-5rqw-f597) affected ranges are `[0,1.5.1)` ∪ `[1.5.3, 1.5.4-0.20240701034025-d67f41855da4)`; v1.5.3 is affected; v1.5.1/v1.5.2 are not | live OSV API fetch 2026-11-24 (`api.osv.dev/v1/vulns/GO-2026-6278`), SEMVER events; fixed commit `d67f41855da4`, advisory published 2026-08-25, REVIEWED | Confirmed |
| V3 | The fixed version is published on the module proxy as pseudo-version `v1.5.4-0.20240701034025-d67f41855da4`; **no tagged release contains the fix** (`@latest` = affected v1.5.3, tag 2024-06-14; version list ends at v1.5.3) | `proxy.golang.org/.../@v/v1.5.4-0.20240701034025-d67f41855da4.info` (Origin hash `d67f41855da4`, 2024-07-01); `@v/list` + `@latest`, fetched 2026-11-24 | Confirmed |
| V4 | The upstream fix touches only `conn.go` (+10/−3) and `prepared_test.go` (+7/−2) — no exported API change; the bump is drop-in | GitHub API commit `d67f41855da4` files list, message "Use crypto/rand for mask key", fetched 2026-11-24 | Confirmed |
| V5 | AILANG reaches affected symbols in non-test code: `websocket.Dialer` at `internal/platform/streamws/streamws.go:39` and `internal/executor/codex/daemon_transport.go:29-32`; `Upgrader`/`Conn.*` across `internal/websocket`, `internal/apiserver`; 328 call-sites repo-wide | repo grep (`Dialer`/`DialContext`/`Upgrade`/`WriteMessage`), 2026-11-24; both dial sites read | Confirmed |
| V6 | The CI gate: `govulncheck (vuln gate)` runs govulncheck v1.3.0 (pinned) over `./...` piped through `tools/govulncheck-filter`, compared against `.govulncheck-allow.yml`, currently `allow: []` | `.github/workflows/ci.yml:971-1001`; `.govulncheck-allow.yml` read | Confirmed |
| V7 | Filter exit contract: 0 = clean/allowlisted, 1 = unallowlisted or expired finding, 2 = filter failure; reaching and module-only findings are partitioned (module-only never gates, per #703/#717) | `tools/govulncheck-filter/main.go` header + `decide()` read | Confirmed |
| V8 | `govulncheck (vuln gate)` is a REQUIRED dev-gate context — a red blocks merges | [m-v1-release-gates.md](../v1_0_0/m-v1-release-gates.md) required-contexts list | Confirmed |
| V9 | No other module requires gorilla/websocket at a higher version (the bump cannot be blocked by MVS) | Negative-existence: `go.mod` requires v1.5.3 and that is the *selected* version, and `go.sum` shows exactly one version — had any module in the graph required a higher one, the selection would already exceed v1.5.3 | Confirmed |
| V10 | No existing design doc covers GO-2026-6278 or a gorilla/websocket bump (duplicate gate) | `grep -rln 'gorilla\|GO-2026' design_docs/` → only mission logs/changelogs mentions; create-script SimHash+neural search top matches are name-derived noise (pattern guards, poly ADT inference — unrelated) | Confirmed |
| V11 | Precedent: advisory bumps land as small Security changelog entries with version ranges and gate rationale (x/net 0.59.0→0.60.0 for GO-2026-6603..6617, v0.53.1) | `changelogs/v0.32-current.md` §[v0.53.1] Security | Confirmed |
| V12 | A `go.mod`-only PR classifies as `code=true` → full matrix including the vuln gate | `ci.yml` `changes` job classify script: non-`design_docs|changelogs .md` paths → code lane; default is code | Confirmed |
| V13 | The triage session had no local Go toolchain, so all local commands are executor-phase, not author-phase | `go`/`govulncheck` not on PATH in the triage container | Noted (design-only, as directed) |

## Deferred Decisions

- **D3 — when to swap the pseudo-version for a tag:** agent-resolved,
  whenever gorilla/websocket publishes a tagged release containing commit
  `d67f41855da4` (any tag > v1.5.3). The changelog note makes it findable;
  dependency sweeps and the post-release skill's expiry check are the
  natural pickup points.
- **Exact PR mechanics** (squash message wording, whether the changelog entry
  lands in the same commit) — sprint-executor latitude.

## Non-Goals

**Not attempted in this feature:**
- The Windows compile request `inbox_1795447998263_b0372c15` — separate triage thread; explicitly out of scope per the directive
- The pipeline package timeout — separate triage thread; explicitly out of scope per the directive
- Allowlisting GO-2026-6278 — rejected (D2): a real fix exists
- Replacing gorilla/websocket with another WS library — large diff, no security need beyond the bump
- Any AILANG language/compiler/runtime change — none needed; no Conflict Surface section required (no parser/typechecker/effect-row surface touched)
- Auditing *other* dependencies for advisories — the vuln gate does that continuously; not this doc's job

## Timeline

**Day 1** (~2 hours active + CI wait):
- Phase 1: bump + local verify (~1.5 h)
- Phase 2: PR + required-context watch
- Phase 3: changelog + follow-up note (~15 min)

**Total: ~0.5 day** (dominated by CI turnaround, not work)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Upstream pseudo-version is a moving-target style pin (no tag) | Low | It is an immutable proxy-published version of a specific commit; swapped for a tag when one exists (D3). Same pattern the Go ecosystem uses routinely for untagged fixes. |
| Pseudo-version churn / Dependabot noise | Low | Dependabot reads `go.mod` directly and will propose the tag bump when it exists — which is the desired end state, not noise. |
| Hidden behavior change in the 2-commit delta (v1.5.3 → fix commit) | Low | The fix commit is the *only* commit between the v1.5.3 tag and the pseudo-version tip touching anything (verified via the commit itself: conn.go + a test); the delta is exactly "mask key uses crypto/rand". |
| `crypto/rand` per-frame overhead regression (the original motivation for v1.5.3's weak PRNG) | Low | Applies to AILANG only as a WS client on streaming paths — masking at most a few frames per message; the security property is not negotiable and upstream accepted the cost. Measure only if a WS benchmark regresses. |
| Weak-PRNG finding resurfaces via a *different* module embedding gorilla/websocket | Low | go.sum shows a single version in the graph (V9); the vuln gate would catch any new embedding immediately. |

## Related Documents

- [m-v1-release-gates.md](../v1_0_0/m-v1-release-gates.md) — REQUIRED dev-gate contexts incl. `govulncheck (vuln gate)`; "new advisory, red until allowlisted" triage class
- `design_docs/v1-mission.md` entries for `#703`/`#717` — govulncheck-filter reach/module-only partitioning and expiry semantics (why a *reaching* finding with `allow: []` must fail the gate)
- `changelogs/v0.32-current.md` §v0.53.1 — the x/net 0.59.0→0.60.0 advisory-bump precedent (entry format for this fix)
- `.govulncheck-allow.yml` — currently `allow: []`; stays that way (D2)
- `tools/govulncheck-filter/main.go` — gate exit-code contract used by the verification plan

## References

- [GO-2026-6278](https://pkg.go.dev/vuln/GO-2026-6278) / GHSA-w67g-5rqw-f597 — advisory (OSV record fetched live 2026-11-24)
- Upstream fix commit: `gorilla/websocket@d67f41855da4` — "Use crypto/rand for mask key"
- CI run 45289159090 / job 138093356254 — the red gate at dev `d3c2c785`
- [Design Axioms](/docs/references/axioms)

## Future Work

- Swap the pseudo-version pin for the first tagged release containing the fix (D3)
- If gorilla/websocket stays untagged for long, consider tracking upstream
  release cadence in the post-release skill's allowlist/expiry review so a
  stale pseudo-version gets re-reviewed rather than floating forever

---

**Document created**: 2026-11-24
**Last updated**: 2026-11-24
