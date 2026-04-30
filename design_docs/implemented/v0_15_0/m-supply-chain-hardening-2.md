# M-SUPPLY-CHAIN-HARDENING-2 — second pass on Scorecard findings

**Status**: Implemented (v0.15.0)
**Target**: v0.15.x
**Predecessor**: [M-SUPPLY-CHAIN-HARDENING](../../implemented/v0_14_1/m-supply-chain-hardening.md) (v0.14.1)
**Companion docs**:
- [m-supply-chain-hardening-2-cii-answers.md](m-supply-chain-hardening-2-cii-answers.md) — CII passing-tier walkthrough
- [m-supply-chain-hardening-2-cii-silver.md](m-supply-chain-hardening-2-cii-silver.md) — CII silver-tier feasibility triage

**Live score** (2026-04-28, commit `374af8503`): **6/10**
**Target score**: ~7.5–8.0/10 (ceiling set by Code-Review opt-out)

## Status as of 2026-04-28

- **CII Best Practices**: passing badge **earned** (project ID 12676, https://www.bestpractices.dev/projects/12676). Silver tier prepped in `.bestpractices.json` but blocked by two honest Unmets (`bus_factor`, `test_statement_coverage80`); see silver companion doc.
- **Governance artifacts shipped** in commit `fa64b79e`: CODE_OF_CONDUCT.md, GOVERNANCE.md, ARCHITECTURE.md.
- **Token-Permissions** (top-level `release.yml`): fixed in this branch.
- **Dependabot** is currently flagging **18 vulnerabilities** (2 critical, 4 high, 10 moderate, 2 low) on the dev branch. Substantially overlapping with the 23 govulncheck findings below; M1+M2 below should clear most of them.
- **M1–M5 still pending**: Go toolchain bump, MCP SDK bump, SLSA provenance, third-party Action pinning, govulncheck CI gate.

## Context

The v0.14.1 sprint shipped SECURITY.md, top-level workflow permissions,
CodeQL, branch protection on `dev`, cosign keyless signing for releases,
and removed checked-in binaries. Scorecard rose from 4.6 → 6.

A weekly cron has continued running and the report
(commit `374af8503`, scanned 2026-04-28) flags four categories where
incremental work since v0.14.1 either drifted or was deferred:

1. **Token-Permissions** — one residual `contents: write` at top level
   in `release.yml`. Already fixed in this branch (top-level →
   `contents: read`, `create-release` job retains its scoped write).
   Listed here only for changelog completeness.
2. **Vulnerabilities (0/10, 23 reachable)** — `govulncheck ./...`
   shows 17 of 23 are stdlib bugs fixed in Go 1.25.x patch releases;
   2 are an MCP SDK bug fixed in v1.4.1; 4 are unfixed Ollama issues.
   The previous doc deferred this with "needs a separate pass to pull
   osv-scanner output." This is that pass.
3. **Signed-Releases (3/10)** — cosign signing landed for v0.14.1 and
   v0.14.2. Scorecard now also wants **build provenance** (SLSA
   attestation) on the artifacts, separate from the signature.
4. **Pinned-Dependencies (0/10)** — the v0.14.1 doc rejected this as
   theatre. **This doc partially flips that decision**: third-party
   GitHub Actions (10 of the 62 unpinned references) sit outside the
   GitHub trust boundary and *are* worth pinning. First-party
   `actions/*` and `github/*` stay unpinned for the reasons in the
   prior doc.
5. **CII Best Practices badge (0/10)** — the v0.14.1 doc rejected
   this as a "100-question form" with diluted signal. **This doc
   reverses that.** Reading the actual questionnaire, ~60% of the
   criteria are already substantively met by existing infrastructure
   (LICENSE, SECURITY.md, CodeQL, Fuzzing, Signed-Releases, public CI
   results); the remaining items are mostly URL-fills pointing at
   SECURITY.md anchors. The badge form does not require new work — it
   requires *attesting* to work that already shipped. The prior
   rejection conflated "100 questions" with "100 hours of work";
   they're not the same. The disclosure-track criteria flagged in the
   user's review (`vulnerability_report_process`,
   `vulnerability_report_private`, `vulnerability_report_response`)
   are all already met by [SECURITY.md](../../../SECURITY.md) — the
   form just needs the public URL.

The two `Warn` job-level `contents: write` flags
([ci.yml:322](../../../.github/workflows/ci.yml#L322) and
[release.yml:236](../../../.github/workflows/release.yml#L236)) are
both legitimate scoped escalations (dependency-graph submission, and
release publishing + cosign OIDC). They will continue to surface as
warnings; the right answer is documentation, not removal.

## Live scan summary

| Check | Score | Action |
|---|---|---|
| Maintained | 10 | — |
| License | 10 | — |
| Dependency-Update-Tool | 10 | — |
| Dangerous-Workflow | 10 | — |
| Security-Policy | 10 | — |
| SAST (CodeQL) | 10 | — |
| Binary-Artifacts | 10 | — |
| Fuzzing | 10 | — |
| Packaging | 10 | — |
| Token-Permissions | 0 → ~8 | **fixed in this branch** — top-level read on `release.yml` |
| Vulnerabilities | 0 → ~7 | **M1+M2** — Go toolchain bump, MCP SDK bump |
| Signed-Releases | 3 → 6 | **M3** — add SLSA provenance attestation |
| Pinned-Dependencies | 0 → ~3 | **M4** — pin third-party actions only; reject first-party + Docker pinning |
| Branch-Protection | 3 | accept (deliberate, see prior doc) |
| Code-Review | 0 | accept (deliberate, see prior doc) |
| CII-Best-Practices | 0 → 5 | **shipped** — passing badge earned at https://www.bestpractices.dev/projects/12676. Silver out of scope. |
| CI-Tests | -1 | structural — no PRs to score |
| Contributors | 6 | accept |

## Decisions

### Real and worth doing

**Go stdlib + toolchain bump (1.25.0 → 1.25.9).**
17 of the 23 reachable vulnerabilities are Go stdlib bugs fixed in
patch releases. The traces include real attack surface for AILANG:

- TLS 1.3 KeyUpdate DoS (`GO-2026-4870`) — reachable via
  [internal/effects/net.go:545](../../../internal/effects/net.go#L545)
  and the apiserver listener. The coordinator and MCP server both
  expose long-lived TLS connections.
- `archive/tar` unbounded allocation (`GO-2026-4869`) — reachable from
  [internal/pkg/tarball.go](../../../internal/pkg/tarball.go); package
  install reads tarballs from the registry.
- `html/template` XSS / context bugs (`GO-2026-4865`, `GO-2026-4603`)
  — reachable from [internal/eval_analysis/export.go](../../../internal/eval_analysis/export.go)
  (HTML eval reports) and the apiserver.
- `crypto/x509` chain-building issues — reachable from any outbound
  HTTPS via the Net effect.

A toolchain bump is a one-line change with broad coverage. This is the
single highest-ROI item in the doc.

**MCP SDK bump (`github.com/modelcontextprotocol/go-sdk` v1.3.1 → v1.4.1).**
Closes `GO-2026-4773` (cross-site tool execution on the HTTP server
without authorization) and `GO-2026-4770` (null-Unicode JSON parsing
bug). The vulnerable code path is the AILANG MCP server itself
([cmd/ailang-microrag-mcp/main.go](../../../cmd/ailang-microrag-mcp/main.go),
[internal/apiserver/mcp.go](../../../internal/apiserver/mcp.go)). Real
attack surface for users running the MCP locally — unauthenticated
LAN access could trigger tools.

**SLSA build provenance.**
Cosign signing (shipped v0.14.1) proves *the binary was signed by our
workflow's OIDC identity*. SLSA provenance proves *what source commit,
build steps, and inputs produced it*. Different claim, both verifiable.
The standard generator is
[`slsa-framework/slsa-github-generator`](https://github.com/slsa-framework/slsa-github-generator);
adding it to `release.yml` is ~15 lines and the resulting
`*.intoto.jsonl` files are auto-consumed by Scorecard. The two claims
compose: a verifier checks signature → binds to workflow identity →
SLSA attestation describes which commit/branch ran inside that
identity.

**Third-party GitHub Actions pinning (narrow scope).**
The prior doc's rejection of action pinning was correct *for
GitHub-owned actions* (`actions/*`, `github/*`) — Dependabot churn
plus reviewer rubber-stamping makes pinning a net loss when the trust
boundary is already GitHub. That argument does **not** extend to
third-party actions. The current report flags 10 third-party
references (`sigstore/cosign-installer`, `softprops/action-gh-release`,
`SonarSource/sonarqube-scan-action`, `github/codeql-action`, etc.)
where the trust boundary is *not* GitHub. A tag re-point on those
actually changes the threat model.

Plan: pin only third-party actions to SHA, leave first-party on tags,
and document the policy in [SECURITY.md](../../../SECURITY.md). This
gets Pinned-Dependencies from 0 to ~3 (10/72 dependencies pinned)
without the maintenance burden the prior doc warned about.

### Reject — reaffirm prior decisions

**First-party GitHub Actions pinning.** Argument from the prior doc
still holds: Dependabot rewrites SHAs as fast as we pin them, and
trained-rubber-stamp PR reviews on action SHA bumps *are* the attack
vector the check claims to defend against. Trust boundary is GitHub
itself — pinning doesn't move it.

**Container image pinning** at the digest level (separately discussed
below). Same theatre argument as first-party actions.

**Container image pinning at the digest level** (`docker/Dockerfile.*`,
19 unpinned base images). Same theatre argument as first-party actions:
`debian:bookworm-slim`
and `gcr.io/distroless/static-debian12` are pulled fresh at build time;
pinning to a digest means manual digest churn whenever the upstream
publishes security patches. The right defense here is rebuilding
regularly (which we already do on every release) plus the Dependabot
docker ecosystem (already enabled), not freezing on a digest that goes
stale.

**Code-Review (0/10) and Branch-Protection (3/10).** Both are
penalties for the deliberate "no required PR review on dev" stance.
Restated: the project's thesis is autonomous AI development; flipping
to required human review changes what AILANG is rather than improving
its security posture. Documented in the prior doc; nothing has
changed.

**Ollama vulns** (`GO-2025-4251`, `GO-2025-3824`, `GO-2025-3695`,
`GO-2025-3689` — all "Fixed in: N/A"). Upstream has not released a
fix. We track via Dependabot; the alternative is dropping the Ollama
integration, which we're not doing. M5 below proposes a `govulncheck`
CI job that documents the suppression with a comment per finding,
rather than letting them silently float in the report.

## Implementation plan

### M1 — Go toolchain bump (1.25.0 → 1.25.9)

**Goal**: clear 17 of 23 reachable stdlib vulnerabilities.

- Edit [go.mod](../../../go.mod) line 3: `go 1.25.0` → `go 1.25.9`.
- Update `go-version: '1.25'` references in workflow files. Most use
  the major-only spec which auto-pulls `1.25.x`, but pin to `1.25.9`
  explicitly during the bump to make the change visible in the diff.
- Run `go mod tidy`, `make ci`, `make verify-examples`.
- Re-run `govulncheck ./...` and confirm vulns 1, 2, 3, 4, 5, 8, 9,
  10, 11, 12, 13, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24 are clear.

Acceptance:
- `govulncheck ./... 2>&1 | grep "^Vulnerability" | wc -l` ≤ 6.
- All workflows pass on the bumped toolchain.
- Binary size delta < 5%.

### M2 — MCP SDK bump (v1.3.1 → v1.4.1)

**Goal**: clear `GO-2026-4773` and `GO-2026-4770`.

- `go get github.com/modelcontextprotocol/go-sdk@v1.4.1`.
- `go mod tidy`.
- Run the MCP server smoke test:
  `make build && bin/ailang-microrag-mcp --help` and the existing
  end-to-end test in [internal/apiserver/](../../../internal/apiserver/).

Acceptance:
- `govulncheck ./...` no longer reports vulns #6, #7.
- MCP integration test (registers and calls a tool over stdio) passes.

### M3 — SLSA build provenance

**Goal**: lift Signed-Releases past 3/10 by adding the provenance
claim that complements existing cosign signatures.

- Add the SLSA generator job to
  [release.yml](../../../.github/workflows/release.yml) as a
  downstream of `build-release` (mirror pattern of `create-release`).
- Use `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml`
  with the SHA256SUMS digest as input. Outputs a single
  `multiple.intoto.jsonl` for the whole release.
- Attach `multiple.intoto.jsonl` to the GitHub release alongside the
  existing `.sig`/`.pem`/`.sha256` files.
- Add verification example to
  [SECURITY.md](../../../SECURITY.md) using `slsa-verifier`.

Acceptance:
- v0.15.x release has `multiple.intoto.jsonl`.
- `slsa-verifier verify-artifact <archive> --provenance-path multiple.intoto.jsonl --source-uri github.com/sunholo-data/ailang --source-tag v0.15.x` succeeds.
- Next Scorecard run reports Signed-Releases ≥ 6.

### M4 — Pin third-party Actions only

**Goal**: move third-party action references to SHA pins. Document
the policy so future PRs default to it.

Third-party actions in current use:
- `sigstore/cosign-installer@v3.7.0` (release.yml)
- `softprops/action-gh-release@v3` (release.yml)
- `SonarSource/sonarqube-scan-action@*` (ci.yml)
- `github/codeql-action/*@*` (codeql.yml — note: GitHub-owned but in `github/codeql-action`, treat as third-party for Scorecard purposes since the report classifies it that way)
- any other non-`actions/*` references at scan time.

For each: pin to the SHA listed on the action's release page, with
a trailing comment containing the version tag for human readability:
`uses: sigstore/cosign-installer@<sha>  # v3.7.0`.

Configure Dependabot to update *only* these third-party pins (it
already does — confirm `dependabot.yml` pulls latest SHA, not just
tag, when using SHA pinning).

Add a paragraph to [SECURITY.md](../../../SECURITY.md) under "Build
trust boundary" explaining the first-party-vs-third-party policy.

Acceptance:
- `grep -RE "uses: [^a]" .github/workflows/ | grep -v "@[a-f0-9]\{40\}"` returns only `actions/*` and `github/*` references.
- Next Scorecard run reports Pinned-Dependencies > 0.
- Dependabot proposes a SHA bump (not a tag bump) on at least one
  third-party action within two weeks.

### M5 — `govulncheck` CI gate + suppression file

**Goal**: catch stdlib regressions early and keep deferred findings
visible in code rather than buried in the weekly Scorecard run.

- Add a `govulncheck` job to [ci.yml](../../../.github/workflows/ci.yml).
- Job runs `govulncheck -format json ./...` and pipes through a
  filter script that compares against
  [.govulncheck-allow.yml](../../../.govulncheck-allow.yml) (new
  file) — a list of `{id, reason, expires}` entries.
- Initial allowlist: the four Ollama vulns, each with a one-line
  reason ("upstream unpatched, tracked at <issue-url>") and a 90-day
  expiry. Expired entries fail the build, forcing re-review.
- Job fails the build on any unallowlisted finding. New stdlib bugs
  in a Go patch release will surface within a day rather than waiting
  for the Monday Scorecard run.

Acceptance:
- Adding a fake CVE to the allowlist with `expires: 2024-01-01` fails CI.
- Removing one Ollama vuln from the allowlist fails CI with a
  governance reminder.
- Allowlist entries are reviewed in the post-release skill checklist.

### M6 — CII Best Practices badge (passing tier) — SHIPPED

**Status**: complete on 2026-04-28. Project ID **12676** at
<https://www.bestpractices.dev/projects/12676>. Badge in [README.md:19](../../../README.md#L19).

**What landed** (across commits `64e8f238` and `fa64b79e`):
- [.bestpractices.json](../../../.bestpractices.json) covering all
  passing-tier criteria + silver-tier prep.
- [CODE_OF_CONDUCT.md](../../../CODE_OF_CONDUCT.md) (Contributor
  Covenant 2.1 by reference).
- [GOVERNANCE.md](../../../GOVERNANCE.md) (single-maintainer + AI
  coordinator model + bus-factor + access continuity).
- [ARCHITECTURE.md](../../../ARCHITECTURE.md) (repo map + pipeline
  diagram + capability-effect overview).
- [SECURITY.md](../../../SECURITY.md) "Response SLA" subsection
  added for stable deep-link.
- README badge added.

**Why silver was deferred**: silver requires zero Unmets, and
`bus_factor` (single maintainer at v0.x by design) plus
`test_statement_coverage80` (38% vs 80% target) are real blockers,
not bypassable via theatre. See
[m-supply-chain-hardening-2-cii-silver.md](m-supply-chain-hardening-2-cii-silver.md).

The original passing-tier plan is preserved below for reference; no
further work is required for this milestone.

#### Original passing-tier plan (reference only)

**Goal**: earn the [OpenSSF Best Practices passing badge](https://www.bestpractices.dev/)
by attesting to work that already shipped. Adds a fourth trust signal
to the README alongside Sonar / Scorecard / Go Report Card.

The questionnaire has ~70 criteria across `BASIC` / `CHANGE_CONTROL` /
`REPORTING` / `QUALITY` / `SECURITY` / `ANALYSIS` sections. Most
project-side work is already done; this milestone is form-filling
plus three small SECURITY.md additions to make URLs unambiguous.

**Pre-met criteria** (no new work, just URL-fill):
- `floss_license`, `license_location` → [LICENSE](../../../LICENSE)
- `documentation_basics`, `documentation_interface` → website + README
- `discussion`, `english` → GitHub Issues + Discussions
- `repo_public`, `repo_track`, `repo_distributed` → GitHub
- `version_unique`, `version_semver` → CHANGELOG.md
- `release_notes` → CHANGELOG.md + GitHub releases
- `report_process`, `report_tracker` → [GitHub Issues](https://github.com/sunholo-data/ailang/issues)
- `vulnerability_report_process` → [SECURITY.md#reporting-a-vulnerability](../../../SECURITY.md)
  (this is the user's flagged "Met")
- `vulnerability_report_private` → same anchor; GitHub Security
  Advisories + email both documented
- `vulnerability_report_response` → SLA stated in SECURITY.md
  ("Acknowledgement within 3 business days, fix critical within 14
  days"); meets the ≤14-day requirement
- `build`, `build_common_tools`, `build_floss_tools` → Makefile + Go
- `automated_test`, `automated_test_policy`, `automated_test_added` → `make test`, CI
- `static_analysis`, `static_analysis_common_vulnerabilities`,
  `static_analysis_fixed`, `static_analysis_often` → CodeQL + Sonar
  weekly
- `dynamic_analysis_unsafe` → not applicable (no `unsafe` use in
  hot paths; verifiable via `grep -rn "unsafe\\." --include='*.go'`)
- `crypto_published`, `crypto_call`, `crypto_floss`, `crypto_keylength`,
  `crypto_working`, `crypto_pfs`, `crypto_password_storage` → cosign
  + TLS via Go stdlib; no custom crypto in AILANG
- `delivery_mitm`, `delivery_unsigned` → cosign keyless + SHA256 on
  releases (this is the M-SUPPLY-CHAIN-HARDENING v0.14.1 win)
- `vulnerabilities_fixed_60_days`, `vulnerabilities_critical_fixed` →
  M1+M2 above ship the fixes within this sprint window

**Criteria needing small SECURITY.md additions**:
- `release_notes_vulns` — release notes must call out CVEs/security
  fixes when present. Add a "Security" section template to the
  release-manager skill so the next release that includes M1+M2
  documents the cleared CVE IDs explicitly.
- `vulnerability_report_credit` — already in SECURITY.md ("reporters
  are credited in the release notes"). Confirmed.
- `crypto_used_network` — confirm in SECURITY.md that all network
  effects use TLS (Go stdlib, no plaintext fallbacks). One-line
  addition.
- `installation_common`, `installation_compromised` — pointer to
  cosign-verified install.sh.

**Criteria flagged as "Unmet" or "N/A — justify"**:
- `contribution`, `contribution_requirements` → already met by
  [CONTRIBUTING.md](../../../CONTRIBUTING.md) (194 lines, AI-pipeline
  workflow + human-PR path).
- `coding_standards` → reference `.claude/rules/coding-standards.md`
  + golangci-lint config.
- `warnings`, `warnings_fixed`, `warnings_strict` → Sonar +
  golangci-lint already enforce; provide URL.
- `dco`, `signed_off_by` → N/A justification (AI-coordinator commits
  use a Co-Authored-By trailer; document the signing model).

**Criteria likely to fail honestly and need explicit "Unmet"**:
- `two_person_review` — same trade-off as Scorecard's Code-Review.
  Mark "Unmet" with a written justification linking to
  [m-supply-chain-hardening.md](../../implemented/v0_14_1/m-supply-chain-hardening.md).
  CII allows justified non-met items at the *passing* tier.

**Implementation steps**:
1. Add anchor IDs / improved section structure to SECURITY.md so
   each form question has a stable deep-link. (Partially done in
   this branch: "Response SLA" subsection added.)
2. Add `Security` template section to the release-manager skill so
   future releases auto-fill cleared-CVE notes.
3. Register at https://www.bestpractices.dev/ and submit answers,
   linking to SECURITY.md / CONTRIBUTING.md / LICENSE / CHANGELOG.
4. Add the resulting badge to README alongside the existing trust
   badges. Pursue *passing* tier only — silver/gold require
   `two_person_review` which we've deliberately opted out of.

Acceptance:
- Badge URL displayed on bestpractices.dev as "passing".
- README gains the CII passing badge.
- `vulnerability_report_*` items in the form show "Met" with the
  SECURITY.md anchor URL filled in.
- Next Scorecard run reports CII-Best-Practices ≥ 5 (Scorecard checks
  the badge via the projects API; passing tier counts).

## Projected score

| Check | Live (2026-04-28) | After M6 (now) | After M1–M5 |
|---|---|---|---|
| Token-Permissions | 0 | 0 → ~8 (after next Scorecard run) | ~8 |
| Vulnerabilities | 0 | 0 | ~7 *(M1+M2; ceiling set by Ollama)* |
| Signed-Releases | 3 | 3 | 7 *(M3; full 10 needs ≥3 releases with provenance)* |
| Pinned-Dependencies | 0 | 0 | ~3 *(M4; ceiling set by first-party-pinning rejection)* |
| CII-Best-Practices | 0 | **5** *(passing earned)* | 5 |

Overall **6.0 → ~8.0** once M1–M5 land. M6 already moved the needle:
once the next Scorecard cron runs, CII-Best-Practices flips from 0 to
5 (passing tier counts as 5 in Scorecard's rubric). Token-Permissions
also flips when the next scan picks up the `release.yml` top-level fix.
M1–M5 are the remaining sprint work.

## Out of scope

- Pinning first-party GH actions or container images (rejected above).
- CII silver/gold tier (requires `two_person_review` — same opt-out
  as Scorecard's Code-Review).
- Required-review branch protection on `dev` (rejected — prior doc).
- Migrating off Ollama or vendoring its security-fixed fork. Tracked
  separately if upstream silence persists past Q3 2026.

## References

- Live Scorecard: https://securityscorecards.dev/viewer/?uri=github.com/sunholo-data/ailang
- API: https://api.scorecard.dev/projects/github.com/sunholo-data/ailang
- govulncheck: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
- SLSA generator: https://github.com/slsa-framework/slsa-github-generator
- Predecessor: [M-SUPPLY-CHAIN-HARDENING](../../implemented/v0_14_1/m-supply-chain-hardening.md)
