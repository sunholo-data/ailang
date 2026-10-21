# M-GO-TOOLCHAIN-BUMP-1-26-10: Bounded Go toolchain patch bump 1.26.9 → 1.26.10 to clear GO-2026-6809

**Tracking**: ailang#1892 (triage iteration 11 evidence: [#issuecomment-6313645488](https://github.com/sunholo-data/ailang/issues/1892#issuecomment-6313645488))
**Status**: Planned — awaiting design approval. **Triage changed no pin, no allowlist entry, and no deployment while producing this document; nothing here is executed or sprint-approved yet.**
**Target**: v0.54.17
**Priority**: P0 (the `govulncheck (vuln gate)` CI job is red on `origin/dev`, so every Go-touching PR to dev fails a security gate)
**Estimated**: 0.5 day (≈2h pins + verification, ≈1h remote-CI wait, ≈1h docs/buffer)
**Dependencies**: None. Precedent: [m-supply-chain-hardening-2 (v0.15.0) M1](../implemented/v0_15_0/m-supply-chain-hardening-2.md), which performed the identical bump class 1.25.0 → 1.25.9 for 17 reachable stdlib vulns.

---

## Problem Statement

**Current State:**

- CI run [39582396242](https://github.com/sunholo-data/ailang/actions/runs/39582396242) on `origin/dev` `d75ef77f59cf4c3c3151fef4840335d269963b8d` has a failed `govulncheck` job (119163055738) reporting **GO-2026-6809**, **function-reaching**, **unallowlisted**, in the Go standard library's `crypto/x509` package (`Certificate.Verify` symbol).
- The advisory ([vuln.go.dev/ID/GO-2026-6809.json](https://vuln.go.dev/ID/GO-2026-6809.json), reviewed REVIEWED, alias CVE-2026-111943): *"Distrust compromised intermediate certificates in crypto/x509"* — a denylist was added so `Certificate.Verify` returns an error for known-compromised intermediate CA certificates. Affected ranges: introduced at `0`, **fixed in `1.26.10`**; introduced `1.27.0-0`, **fixed in `1.27.3`**.
- The repo's Go pin is **1.26.9** — one patch release below the fix.
- **Attribution is temporal, not a product regression.** The advisory was published `2026-10-21T00:27:29Z`. First red: run [39530700414](https://github.com/sunholo-data/ailang/actions/runs/39530700414) at `7ab609a6a18c1233cdfa8e287f5ffe458f106544`. Last green: run [39525459551](https://github.com/sunholo-data/ailang/actions/runs/39525459551) at `9aebccf9bf8737a4d92ecdfdc285a1b1ee387522`. The `govulncheck` job downloads a fresh vulnerability-database snapshot at run time, so the advisory's publication — not a code change between those SHAs — is what turned the gate red. This matches triage's independent finding in issue #1892.
- **Reach is real, not theoretical**: 176 non-test Go files import `net/http`; every outbound HTTPS request performs TLS chain verification, which calls `crypto/x509.Certificate.Verify` in the stdlib call graph. The Net effect enforces `https://` by default ([internal/effects/net.go:32](../../internal/effects/net.go)), and the Ollama/OpenRouter/GitHub/cloud clients all ride it. The only direct `crypto/x509` import ([internal/builtins/crypto_rsa.go](../../internal/builtins/crypto_rsa.go)) parses rather than verifies, so the function-reaching trace is the TLS-verification path — the same class m-supply-chain-hardening-2 already confirmed reachable for the crypto/x509 advisories of its era.
- **The allowlist does not cover it** (grep of `.govulncheck-allow.yml` for `GO-2026-6809`: 0 matches), and the gate's design (expiry-dated entries, stale-entry surfacing) means an entry would only delay, not resolve, the red.

**Impact:** A permanent red security gate on dev. Every PR with Go-code changes fails `govulncheck`; agents and humans must either suppress the advisory (explicitly rejected by triage's constraints and by this doc) or fix it. Shipped release binaries built with 1.26.9 also lack the compromised-intermediate denylist — a genuine trust-boundary weakness for every TLS connection an AILANG process makes.

---

## Goals

**Primary Goal:** Return the `govulncheck (vuln gate)` job to green on dev by rebuilding with a fixed toolchain — Go 1.26.10 — with zero suppression, verified across every actual build target (5 native release platforms, js/wasm, 6 GitHub workflows, 16 Cloud Build images) and remote CI.

**Success Metrics:**
- `govulncheck` job green on the bump commit: only the 9 allowlisted Ollama IDs remain reachable, all unexpired; zero new findings, zero stale entries.
- All Go pins agree: `go.mod`, 15 `go-version:` workflow lines, `deploy/claude-credit-gateway/Dockerfile` at 1.26.10; no orphan 1.26.9 string anywhere (checked by grep in acceptance).
- Full `make ci` (or equivalent test suite) green on 1.26.10; binary size delta < 5% (precedent's bar).
- No allowlist file changes in the diff.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Bump **within the 1.26 line** to 1.26.10, not to 1.27.3 | Patch releases carry only security/bug fixes (Go release policy); a minor jump widens the compat surface (new language/runtime behavior) for zero additional advisory coverage — 1.26.10 already contains the fix | human (this design) | design | low |
| **No allowlist entry** for GO-2026-6809 under any circumstance | The fix is a one-line pin; suppression would paper over a live trust-boundary vuln and the gate would force re-review at expiry anyway | human (this design; matches triage directive "never silently suppress") | design | low |
| Docker base tags `golang:1.26-bookworm` stay **floating at minor level** | Reaffirms m-supply-chain-hardening-2's explicit rejection of digest/patch pinning base images as theatre; they pick up 1.26.10 automatically at next build and regular rebuilds are the defense | human (reaffirmed precedent) | design | low |
| `docker/resident/m0-spike/Dockerfile` `GO_VERSION` 1.26.4 → 1.26.10 | Spike image drifts three patches below the repo pin; one-ARG change, keeps the resident fleet from running an unpatched toolchain | agent | implementation | low |
| `govulncheck` tool pin stays `@v1.3.0` | The v1.4.0 crash that motivated the pin is unrelated to this advisory; un-pinning while the gate is red confounds two variables | agent | implementation | low |
| Wasm `wasm_exec.js` support-file provenance (go1.22.0, [make/docs.mk:42](../../make/docs.mk)) | Pre-existing and unchanged by this bump; flagging only so reviewers know it was seen and deliberately left out | human at review | design | low |

### Design Freeze

- [x] Target version 1.26.10 (not 1.27.3) — advisory JSON fixes the 1.26 line at 1.26.10; verified against `vuln.go.dev`.
- [x] No allowlist/suppression route — hard constraint from triage directive and this design.
- [x] Docker minor-floating tags remain floating — reaffirmed precedent, not reopened.
- [x] Scope excludes any `internal/` code change — the vulnerability is in the stdlib, fixed only by rebuilding with a patched toolchain.

### Deferred Decisions

- Exact PR/commit message wording — agent may choose.
- Whether `go mod tidy` produces a diff (expected: no-op; the `go` directive bump does not change dependency hashes) — agent verifies, agent decides nothing further either way.
- Whether to also bump `.github/workflows/codeql.yml`/`scorecard.yml` if any hidden Go version assumption surfaces during execution — agent may adjust within the 1.26.10 target.

---

## Solution Design

### Overview

GO-2026-6809 lives in the Go **standard library**, not in a module dependency — so there is no `go get` to run. The only fix is rebuilding with a patched toolchain. The change is therefore a bounded, mechanical pin bump across the repo's toolchain surfaces, followed by verification that (a) the finding disappears, (b) every actual build target still compiles and passes its gates, and (c) remote CI (GitHub Actions) and image builds (Cloud Build via `docker/`) are green.

The security payload is real: every AILANG binary shipped from a 1.26.9 build lacks the compromised-intermediate-certificate denylist that `Certificate.Verify` gained in 1.26.10. The CI-green is the byproduct; the patched TLS trust boundary is the point.

### Files to Modify

All edits are one-line string changes `1.26.9` → `1.26.10` (or `1.26.4` → `1.26.10` for the spike), except where noted. Inventory verified by repo-wide grep (see Verification Log):

**Modified files:**
1. [go.mod](../../go.mod) line 3 — `go 1.26.9` → `go 1.26.10` (no `toolchain` line exists; none is added).
2. [.github/workflows/build.yml](../../.github/workflows/build.yml) line 51 — `go-version`.
3. [.github/workflows/ci.yml](../../.github/workflows/ci.yml) lines 110, 554, 665, 711, 800, 822, 852 — seven `go-version` pins, including the `govulncheck` job itself (line 822).
4. [.github/workflows/docusaurus-deploy.yml](../../.github/workflows/docusaurus-deploy.yml) line 204.
5. [.github/workflows/eval-weekly.yml](../../.github/workflows/eval-weekly.yml) line 33.
6. [.github/workflows/release.yml](../../.github/workflows/release.yml) lines 28, 116, 195, 250 — four pins, covering all five release-matrix platforms.
7. [.github/workflows/test-game-codegen.yml](../../.github/workflows/test-game-codegen.yml) line 39.
8. [deploy/claude-credit-gateway/Dockerfile](../../deploy/claude-credit-gateway/Dockerfile) line 4 — `FROM golang:1.26.9` → `golang:1.26.10` (the one Dockerfile that pins a patch version).
9. [docker/resident/m0-spike/Dockerfile](../../docker/resident/m0-spike/Dockerfile) line 9 — `ARG GO_VERSION=1.26.4` → `1.26.10`.

**Deliberately NOT modified:**
- `docker/Dockerfile.{agent-base,agent-codex-go,agent-eval-go,agent-go,agent-pi-go,dashboard,executor-mcp,mcp,registry-validator}` — all float on `golang:1.26-bookworm`; they inherit 1.26.10 on their next build (Cloud Build rebuilds per `cloudbuild-dev.yaml`/`cloudbuild-release.yaml` DAG of 16 images). Reaffirms the m-supply-chain-hardening-2 rejection of base-image pinning.
- `.govulncheck-allow.yml` — zero changes; the Ollama entries are unrelated and remain exactly as-is.
- `make/docs.mk:42` — fetches `wasm_exec.js` from `go1.22.0`; pre-existing, out of scope, flagged in High-Impact Decisions.

### Implementation Plan

**M0 — Pre-flight reproduction (~1h, on a machine with Go)**

- [ ] Install pinned gate tooling: `go install golang.org/x/vuln/cmd/govulncheck@v1.3.0` (unchanged pin).
- [ ] At current pin: `govulncheck -format json ./... | go build -o ./bin/govulncheck-filter ./tools/govulncheck-filter && ./bin/govulncheck-filter` — record the GO-2026-6809 trace frames (first-party evidence of the reach path, complementing CI job 119163055738's report).
- [ ] Build filter once; confirm the failure output names GO-2026-6809 as unallowlisted.

**M1 — Pin bump (~1h)**

- [ ] Apply the 9-file edit list above (14 line edits total).
- [ ] `go mod tidy` — expect no diff; if a diff appears, inspect and report, do not commit silently.
- [ ] `make build` — confirm binaries compile.
- [ ] Repo-wide orphans check: `grep -rn "1\.26\.9" go.mod .github/ docker/ deploy/` → must be empty.
- [ ] `grep -rn "1\.26\.4" docker/resident/` → must be empty.

**M2 — Reach/compatibility across actual build targets (~2h)**

- [ ] Full local suite at 1.26.10: `make test` (or `make ci`), `make fmt`, `make lint`, `make check-boundaries`.
- [ ] Release cross-compile matrix (mirrors `.github/workflows/release.yml`): build `./cmd/ailang` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 — all must compile.
- [ ] Wasm target: `GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm` (the `make/build.mk:113` target's build step).
- [ ] Binary size delta on linux/amd64 vs 1.26.9 build — acceptance bar < 5%.
- [ ] Gate re-run: `govulncheck -format json ./... | ./bin/govulncheck-filter` → exit 0, message form "N reachable finding(s), all allowlisted and unexpired", N = 9 (the Ollama set), zero stale entries.
- [ ] `make verify-examples` (or the current examples gate) — examples still run on the new binary.

**M3 — Remote CI verification (~1h, mostly waiting)**

- [ ] Open PR against dev containing only the pin-bump diff; the `changes` filter will see Go-relevant files (`go.mod`) and arm the `govulncheck` job.
- [ ] Confirm on the PR's run: `govulncheck (vuln gate)` green; no other job newly red.
- [ ] Docker note: Cloud Build images are rebuilt on deploy paths per their DAG; floating `golang:1.26-bookworm` picks up 1.26.10 automatically — no deploy action is part of this change, and this design **triggers no deployment**.
- [ ] Merge only on green, per the repo's normal flow; assert merge-commit CI green.

**Acceptance:**
- All M1–M3 checkboxes green.
- `govulncheck` reachable set is exactly the 9 allowlisted Ollama IDs; GO-2026-6809 absent.
- The PR diff touches no `internal/` file and no allowlist file.
- Changelog note added (e.g. under `changelogs/v0.32-current.md`) recording the bump, mirroring the v0.15.0 M1 changelog entry.

---

## Examples

**Before** (the red gate, current pins):

```
go.mod:3:            go 1.26.9
ci.yml:822:           go-version: '1.26.9'   # govulncheck job builds with the VULNERABLE stdlib
.govulncheck-allow.yml: no GO-2026-6809 entry (and none may be added)

→ govulncheck job: "unallowlisted finding: GO-2026-6809 (crypto/x509.Certificate.Verify) — function-reaching"
→ every Go-touching PR to dev red on a security gate
```

**After** (one-line-per-pin, no suppression):

```
go.mod:3:            go 1.26.10
all workflows:        go-version: '1.26.10'
deploy/Dockerfile:    FROM golang:1.26.10
m0-spike:             ARG GO_VERSION=1.26.10

→ release binaries carry the compromised-intermediate denylist in Certificate.Verify
→ gate: "9 reachable finding(s), all allowlisted and unexpired" → exit 0
```

---

## Success Criteria

- [ ] `govulncheck (vuln gate)` green on the bump commit on remote CI
- [ ] GO-2026-6809 not reported; allowlist byte-identical to pre-change
- [ ] No `1.26.9` / `1.26.4` orphans remain in build surfaces (grep-clean)
- [ ] All 5 native release targets + js/wasm compile; full test suite green on 1.26.10
- [ ] Binary size delta < 5%
- [ ] All tests passing; changelog note added; no documentation claims otherwise stale
- [ ] No deployment triggered by this change

---

## Conflict Surface

**Not applicable — no compiler surface is touched.** This change modifies no file under `internal/`, `cmd/`, or `std/`; AILANG language semantics, parser, typechecker, VM, and effects are byte-identical before and after. The only behavior deltas possible are Go-toolchain-level (build determinism of generated binaries, stdlib internals) — covered by M2's full-suite + cross-target verification. Goldens in `goldens/` are AILANG-language artifacts (e.g. `import_conflict.json`) and do not depend on stdlib patch internals; the full suite re-verifies them regardless.

---

## Testing Strategy

**Unit/integration:** unchanged codebase — the existing `make test` / `make ci` suites are the regression net; no new tests are warranted for a pin string change.

**Gate behavior:** the `govulncheck-filter` end-to-end run (M0 before, M2 after) is the security test: same pinned govulncheck v1.3.0, same filter, only the toolchain moves. Expectation is precise: 9 findings, all allowlisted, exit 0.

**Cross-target:** the 5-platform compile matrix + js/wasm build (M2) is the compatibility test — this is the "across actual build targets" requirement of the triage evidence, made concrete.

**Remote CI:** M3 proves it on the runners that matter (ubuntu, macos, windows) rather than only on one local platform.

---

## Non-Goals

- **Not bumping to Go 1.27.x / 1.27.3** — a minor-version jump is a separate, deliberate decision with its own compat review; the advisory does not require it.
- **No `govulncheck` tool version change** (stays `@v1.3.0`; the v1.4.0 crash pin is load-bearing and orthogonal).
- **No Docker base-image pinning change** — reaffirmed rejection from m-supply-chain-hardening-2; floating minor tags plus regular rebuilds remain the mechanism.
- **No allowlist edits of any kind.**
- **No `make/docs.mk` wasm_exec.js provenance fix** — pre-existing, flagged, out of scope.
- **No deployment, no Cloud Build trigger** — image rebuilds happen on their normal deploy paths.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| 1.26.10 toolchain breaks a build target (e.g. darwin/arm64, js/wasm) | Medium | M2 compiles all 5 native targets + wasm locally before any PR; patch releases are security/bugfix-only by Go policy, and the 1.25.0→1.25.9 precedent shipped across the same matrix without incident |
| Hidden non-`go-version` toolchain assumption (action internals, CodeQL, Scorecard) | Low | M3 watches every job on the PR, not just govulncheck; deferred-decision latitude covers in-scope adjustment |
| `go mod tidy` produces an unexpected go.sum diff | Low | Expected no-op (directive bump); if not, inspect and surface before committing — no silent diffs |
| Binary size regression > 5% | Low | Measured in M2 with the precedent's bar; a patch release exceeding it would be anomalous and would pause the PR for investigation |
| A second stdlib advisory lands mid-flight (DB snapshot moves) | Medium | M2/M3 re-run the gate at final HEAD; any NEW finding is reported, not suppressed — this doc's scope stays GO-2026-6809 only, and a new advisory gets its own triage round |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/compiler change; AILANG determinism guarantees untouched |
| A2: Replayability | 0 | No trace or execution-record format change |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No capability/ambient-authority change in AILANG itself |
| A5: Bounded Verification | +1 | Restores a machine-decidable security gate from a permanently-red state where every failure had to be human-triaged; the gate's verdict is again an automatic accept signal |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | 0 | No prompt/teaching-surface change |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No resource accounting change |
| A10: Composability | 0 | No module-system change |
| A11: Structured Failure | +1 | Fixes the root cause loudly (patched toolchain) instead of suppressing the failure signal; the allowlist's expiry mechanism is respected rather than abused |
| A12: System Boundary | +1 | Hardens the TLS trust boundary every AILANG outbound connection crosses — the compromised-intermediate denylist is exactly a system-boundary defense |

**Net Score: +3** ✅ Proceed

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): no human-convenience-over-machine-analysis tradeoff made

---

## Verification Log

Commands run first-party at `origin/dev` = `d75ef77f59cf4c3c3151fef4840335d269963b8d` (session worktree, coordinator task `task-18020ec1`). Negative results carry their command; where a known-positive control exists it is stated.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | Advisory content, affected symbol, fixed versions, publication time | `curl -s https://vuln.go.dev/ID/GO-2026-6809.json` | REVIEWED; CVE-2026-111943; `crypto/x509` `Certificate.Verify`; fixed `1.26.10` and `1.27.3`; published `2026-10-21T00:27:29Z` |
| V2 | The Go pin is 1.26.9 and there is NO `toolchain` line in go.mod | `sed -n 1,10p go.mod; grep -n toolchain go.mod` (control: line 3 shows `go 1.26.9`) | `go 1.26.9`; grep for `toolchain` empty (rc=1) |
| V3 | Complete 1.26.9 pin inventory | repo-wide grep (see Files to Modify; control: each listed file:line matched verbatim) | 15 `go-version: '1.26.9'` workflow lines across 6 files; `go.mod:3`; `deploy/claude-credit-gateway/Dockerfile:4` |
| V4 | Docker images float on minor tag | `grep -rn "golang:1.26" docker/` | 9 hits, all `golang:1.26-bookworm` (no patch pin); plus `m0-spike` `ARG GO_VERSION=1.26.4` |
| V5 | GO-2026-6809 is NOT allowlisted (negative-existence) | `grep -c "GO-2026-6809" .govulncheck-allow.yml` | `0` (rc=1); allowlist holds only the 9 Ollama IDs, expiry 2026-10-29 |
| V6 | The gate's mechanism (fail on unallowlisted/expired) and pinned govulncheck v1.3.0 | read `.github/workflows/ci.yml` govulncheck job (lines 807–837) + `tools/govulncheck-filter/main.go` header | job comment documents pin rationale; filter exits 1 on any unallowlisted finding |
| V7 | Function reach: TLS-verification path through stdlib | `grep -rln "net/http" --include="*.go" internal/ cmd/ serveapi/ executor_oauth/ mcp_tools/ \| grep -v _test \| wc -l` | 176 non-test files import net/http; `internal/effects/net.go:32` comment confirms `https://` enforced by default; only direct `crypto/x509` import is `internal/builtins/crypto_rsa.go` (control: the same grep at v0.15.0's doc found the same class reachable) |
| V8 | No TLS listener in the repo (so reach is client-side outbound) — negative-existence | `grep -rn "ListenAndServeTLS\|tls.Listen" internal/apiserver/ cmd/ serveapi/` | empty |
| V9 | Precedent exists for this exact change class | `grep -rn "1\.25\.9\|toolchain bumped" design_docs/` | `m-supply-chain-hardening-2.md` M1: 1.25.0 → 1.25.9, same file set, acceptance bar "< 5% binary size delta" adopted here |
| V10 | No duplicate/covering planned doc (dedup gate) | create-script SimHash+neural search ("26 10": none) + `grep -rln "GO-2026-6809\|6809\|x509" design_docs/planned/` | no planned doc covers a 1.26.10 bump; all `6809`/`x509` hits in design_docs are unrelated (line numbers, hashes, old-era x509 chain bugs in shipped v0.15.0 doc) |
| V11 | Build target inventory for the compat matrix | read `.github/workflows/release.yml` matrix (lines 152–210), `make/build.mk:113` | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 + `GOOS=js GOARCH=wasm ./cmd/wasm` |
| V12 | Cloud Build builds images, not Go directly (no hidden pin there) | `grep -n "golang\|go-version" cloudbuild-*.yaml` | no hits; cloudbuild-dev.yaml orchestrates the 16-image Docker DAG |
| V13 | m0-spike drifts below repo pin | `grep -n GO_VERSION docker/resident/m0-spike/Dockerfile` | `ARG GO_VERSION=1.26.4` |
| V14 | First-party govulncheck could NOT be run in this session | `which go govulncheck` | both absent (no Go toolchain in this sandbox) — reach (V7) rests on the CI job 119163055738 report, the advisory's symbol list, and the code inventory; **M0 makes the first-party trace a mandatory first implementation step, not an assumption** |
| V15 | Red attribution is advisory-publication timing, not product change | advisory `published: 2026-10-21T00:27:29Z` (V1) vs. first-red run 39530700414 at `7ab609a6`, last-green 39525459551 at `9aebccf9` | publication precedes the first red; job fetches a fresh vuln DB each run — consistent with triage's evidence in #1892 |

---

## References

- **Advisory**: [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809) · [vuln.go.dev JSON](https://vuln.go.dev/ID/GO-2026-6809.json) · CVE-2026-111943 · go.dev/issue/82345 · golang-announce post
- **Tracking issue / evidence**: [ailang#1892 comment 6313645488](https://github.com/sunholo-data/ailang/issues/1892#issuecomment-6313645488) (triage iteration 11)
- **CI evidence**: failed job 119163055738 in run [39582396242](https://github.com/sunholo-data/ailang/actions/runs/39582396242) on `d75ef77f`; first red run [39530700414](https://github.com/sunholo-data/ailang/actions/runs/39530700414) at `7ab609a6`; last green run [39525459551](https://github.com/sunholo-data/ailang/actions/runs/39525459551) at `9aebccf9`
- **Prior art**: [m-supply-chain-hardening-2 (v0.15.0)](../implemented/v0_15_0/m-supply-chain-hardening-2.md) — M1 toolchain bump 1.25.0→1.25.9; Docker base-image pinning rejection reaffirmed here
- **Governance**: this doc follows the triage boundary — triage owns red-dev reporting while V1 is paused and changes no pin/allowlist/deployment; the proposed bump executes only through the normal design-approval → sprint-planner → executor flow. Independent OpenAI verification and MiniMax concurrence (per the coordinator directive) support the route; final approval remains with the design review.

## Future Work

- Consider a quarterly "toolchain freshness" triage probe so the next stdlib advisory lands as a bounded checklist rather than a red-gate surprise (fits the friction→extension doctrine if it recurs).
- The deferred `wasm_exec.js` provenance question ([make/docs.mk:42](../../make/docs.mk)) deserves its own one-day doc if wasm targets ever drift against the compiled toolchain.
