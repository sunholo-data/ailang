# M-SUPPLY-CHAIN-HARDENING-2 — Sprint Plan

**Design doc**: [m-supply-chain-hardening-2.md](m-supply-chain-hardening-2.md)
**Sprint ID**: `M-SUPPLY-CHAIN-HARDENING-2`
**Target version**: v0.15.x
**Estimated duration**: 1 day (4–6 hours focused work)
**Estimated LOC**: ~260 (mostly workflow YAML + a small filter script)
**Risk level**: low — all changes are mechanical version bumps or
additive workflow steps. The blast radius is contained to CI/release;
no compiler/runtime changes. Each milestone is independently
verifiable.

## Goal

Ship the M1–M5 work from the second supply-chain hardening design doc.
M6 (CII passing badge) already shipped via earlier commits.

Concrete outcomes:
1. Clear ~19 of 23 reachable govulncheck findings via toolchain + MCP
   SDK bumps.
2. Add SLSA build provenance to release artifacts.
3. Pin third-party GitHub Actions to SHA (first-party stays on tags).
4. Wire a `govulncheck` CI job with a documented suppression file
   for the four unfixed Ollama vulns.

Projected Scorecard score after this sprint: **6 → ~7.5**.

## Status going in

| Item | State |
|---|---|
| CII Best Practices passing badge | shipped (project 12676) |
| Token-Permissions top-level fix | committed on dev |
| Governance docs (CoC / Governance / Architecture) | shipped |
| Dependabot alert count | 18 (2 critical, 4 high, 10 moderate, 2 low) |
| `govulncheck ./...` reachable findings | 23 |

## Velocity check

Recent week shows multiple small CI/security sprints completing in
under a day each (M-AGENT-MCP-ONBOARDING M1+M2+M3 in ~1 day; v0.14.1
hardening sprint about 1 day for M1+M2+M3+M4). M-SUPPLY-CHAIN-HARDENING-2
is comparable in shape — small mechanical changes, no compiler logic —
so 1 day is realistic.

## Milestones

### M1 — Go toolchain bump (1.25.0 → 1.25.9)

**Effort**: 30 minutes
**LOC**: ~10 (go.mod + workflow `go-version` strings)
**Risk**: low — Go patch releases are ABI-compatible.

Tasks:
1. `go.mod` line 3: `go 1.25.0` → `go 1.25.9`.
2. Update `go-version: '1.25'` references in workflow files to
   `'1.25.9'` (visible in diff).
3. `go mod tidy`.
4. `make ci` locally.
5. `make verify-examples`.
6. `govulncheck ./...` to confirm vulns 1–5, 8–13, 15–24 cleared.

Acceptance:
- `govulncheck ./... 2>&1 | grep -c "^Vulnerability #"` ≤ 7.
- All workflows pass on the bumped toolchain (CI green).
- `bin/ailang` size delta < 5%.
- Dependabot stdlib alerts close.

Refs design doc: `### M1` block.

### M2 — MCP SDK bump (v1.3.1 → v1.4.1)

**Effort**: 30 minutes
**LOC**: ~5 (go.mod + go.sum)
**Risk**: low — patch-version bump on a focused dependency.

Tasks:
1. `go get github.com/modelcontextprotocol/go-sdk@v1.4.1`.
2. `go mod tidy`.
3. `make build`.
4. Smoke test: `bin/ailang-microrag-mcp --help`.
5. Run existing integration test in `internal/apiserver/`.
6. `govulncheck ./...` confirms `GO-2026-4773` and `GO-2026-4770` cleared.

Acceptance:
- `govulncheck` no longer reports vulns #6, #7.
- MCP integration test passes.
- Manual MCP tool registration smoke test succeeds.

### M3 — SLSA build provenance

**Effort**: 1.5 hours
**LOC**: ~40 (release.yml additions + SECURITY.md verification example)
**Risk**: low — additive; doesn't disturb cosign signing path.

Tasks:
1. Add SLSA generator job to `release.yml` referencing
   `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml`.
2. Wire it as a downstream of `build-release` using SHA256SUMS digest as input.
3. Attach `multiple.intoto.jsonl` to the GitHub release alongside the
   existing `.sig`/`.pem`/`.sha256` files.
4. Add a `slsa-verifier` example to SECURITY.md "Verifying Release Artifacts".

Acceptance:
- Next dry-run release tag produces `multiple.intoto.jsonl`.
- `slsa-verifier verify-artifact <archive> --provenance-path multiple.intoto.jsonl --source-uri github.com/sunholo-data/ailang --source-tag <tag>` succeeds.
- SECURITY.md documents the verification chain.

### M4 — Pin third-party Actions to SHA

**Effort**: 1 hour
**LOC**: ~25 across multiple workflow files + SECURITY.md paragraph
**Risk**: low — Dependabot already understands SHA-with-comment.

Tasks:
1. Identify all third-party Action references via:
   `grep -RE "uses: [^ ]+@" .github/workflows/ | grep -v "@[a-f0-9]\{40\}" | grep -vE "(actions|github)/"`.
2. For each, replace tag with the SHA from the release page, with a
   trailing comment for human readability:
   `uses: sigstore/cosign-installer@<sha>  # v3.7.0`.
3. Verify Dependabot config (`.github/dependabot.yml`) handles SHA pins.
4. Add a "Build trust boundary" paragraph to SECURITY.md: first-party
   GH actions stay on tags (Dependabot churn + GitHub trust boundary);
   third-party actions are pinned to SHA.

Targets (current third-party references):
- `sigstore/cosign-installer@v3.7.0` (release.yml)
- `softprops/action-gh-release@v3` (release.yml)
- `SonarSource/sonarqube-scan-action@*` (ci.yml)
- `github/codeql-action/*@*` (codeql.yml — Scorecard treats as third-party)

Acceptance:
- `grep -RE "uses: [^ ]+@" .github/workflows/ | grep -v "@[a-f0-9]\{40\}" | grep -vE "(actions|github)/"` returns empty.
- `make ci` passes (workflows still load).
- SECURITY.md documents the policy.

### M5 — govulncheck CI gate + suppression file

**Effort**: 2 hours
**LOC**: ~150 (workflow job + filter script + allowlist YAML)
**Risk**: medium — new CI gate could fail unexpectedly. Mitigated by
running locally first.

Tasks:
1. Add `.govulncheck-allow.yml` at repo root with the four Ollama vulns:
   ```yaml
   allow:
     - id: GO-2025-4251
       reason: upstream Ollama unpatched; tracked at <upstream-issue>
       expires: 2026-07-31
     - id: GO-2025-3824 (same shape)
     - id: GO-2025-3695 (same shape)
     - id: GO-2025-3689 (same shape)
   ```
2. Write `tools/govulncheck-filter/main.go` (~80 LOC): reads
   `govulncheck -format json` output, compares against allowlist,
   prints unallowlisted findings, exits non-zero on any unallowlisted
   finding or any allowlisted finding past `expires`.
3. Add a `govulncheck` job to `.github/workflows/ci.yml`:
   - Installs govulncheck.
   - Runs `govulncheck -format json ./... | tools/govulncheck-filter`.
   - Fails the build on any unallowlisted finding.
4. Document the allowlist review cadence in the post-release skill.

Acceptance:
- Local: `govulncheck -format json ./... | go run tools/govulncheck-filter` exits 0 on current state (only Ollama vulns, all allowlisted).
- CI: adding a fake allowlist entry with `expires: 2024-01-01` causes the build to fail with a clear message.
- CI: removing one Ollama vuln from the allowlist causes the build to fail.
- Post-release skill includes a checkbox for reviewing allowlist expiries.

## Day-by-day plan

This is a 1-day sprint. The milestones are independent and can run
sequentially without blocking each other.

| Hour | Milestone | Activity |
|---|---|---|
| 0:00 | M1 | Go toolchain bump + local CI |
| 0:30 | M2 | MCP SDK bump + smoke test |
| 1:00 | M4 | Pin third-party Actions (parallel-safe with M3) |
| 2:00 | M3 | SLSA provenance workflow integration |
| 3:30 | M5 | govulncheck filter + workflow + allowlist |
| 5:30 | finalize | full `make ci`, push to dev, monitor weekly Scorecard run on Monday |

## Dependencies

- M1 should land first; M5 depends on M1 having reduced the vuln list.
- M2, M3, M4 are independent of each other.
- M5 must come after M1+M2 so the only remaining findings are the
  four Ollama vulns we explicitly allowlist.

## Success metrics

- `govulncheck ./...` reports ≤ 5 findings (M1 + M2 outcomes).
- Dependabot critical/high count drops by ≥ 4.
- Next Scorecard cron (Monday) reports:
  - Vulnerabilities ≥ 7 (was 0)
  - Pinned-Dependencies > 0 (was 0)
  - Token-Permissions ≥ 8 (was 0; release.yml already fixed)
- Subsequent v0.15.x release artifact verifies via `slsa-verifier`.
- CI gate catches a planted fake CVE in the allowlist.

## Out of scope (rejected by design doc)

- First-party GH Actions pinning (theatre; Dependabot churn).
- Container image digest pinning (theatre; rebuilt per release).
- CII silver/gold (blocked by `bus_factor` and coverage).
- Required-review branch protection on dev (deliberate opt-out).

## References

- Design doc: [m-supply-chain-hardening-2.md](m-supply-chain-hardening-2.md)
- Predecessor: [m-supply-chain-hardening.md](../../implemented/v0_14_1/m-supply-chain-hardening.md)
- CII passing companion: [m-supply-chain-hardening-2-cii-answers.md](m-supply-chain-hardening-2-cii-answers.md)
- Silver triage: [m-supply-chain-hardening-2-cii-silver.md](m-supply-chain-hardening-2-cii-silver.md)
- Live Scorecard: https://securityscorecards.dev/viewer/?uri=github.com/sunholo-data/ailang
