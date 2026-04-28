# CII Silver tier — feasibility triage

Companion to [m-supply-chain-hardening-2-cii-answers.md](m-supply-chain-hardening-2-cii-answers.md).

Silver progress as of 2026-04-28: **4%** (all 48 fields unanswered).

**Key finding**: `two_person_review` is **gold-only**, not silver. The
prior design doc's hard-ceiling assumption was wrong — silver is
achievable without changing the AI-coordinator workflow. The actual
silver blockers are different (and smaller): one is a coverage gap,
one is an honest bus-factor reality.

## Triage

### Easy Met — answer-only, no new work (~33 fields)

These are already true; just need URL-fill.

| Criterion | Evidence |
|---|---|
| `achieve_passing` | Auto-Met once passing tier hits 100%. |
| `contribution_requirements` | CONTRIBUTING.md (already cited at passing). |
| `report_tracker` | Already Met at passing. |
| `vulnerability_report_credit` | SECURITY.md "reporters are credited in the release notes". |
| `vulnerability_response_process` | SECURITY.md Response SLA section. |
| `documentation_quick_start` | README "Quick Start" + plugin install instructions. |
| `documentation_current` | Active development; CHANGELOG.md updated per release. |
| `documentation_achievements` | CHANGELOG.md + Scorecard / Sonar / CodeQL badges. |
| `documentation_roadmap` | design_docs/planned/ + roadmap page on docs site. |
| `coding_standards` | .claude/rules/coding-standards.md + .golangci.yml. |
| `coding_standards_enforced` | CI runs `make lint`; failures block merge. |
| `build_standard_variables` | Go honors GOOS/GOARCH/CGO_ENABLED; Make honors PREFIX/DESTDIR. |
| `build_preserve_debug` | Release ldflags don't strip; debug binaries available via `go build`. |
| `build_non_recursive` | Go's compiler is non-recursive; Makefile has flat targets. |
| `build_repeatable` | go.sum + go.mod lock dependencies; cosign signs reproducible outputs. |
| `installation_common` | install.sh follows `curl \| bash` convention with verification. |
| `installation_standard_variables` | install.sh honors PREFIX, INSTALL_DIR. |
| `installation_development_quick` | README "Quick Start" + `make install`. |
| `external_dependencies` | go.mod is the manifest; `go mod graph` lists transitive. |
| `dependency_monitoring` | Dependabot enabled (Scorecard 10/10 on Dependency-Update-Tool). |
| `updateable_reused_components` | Go module proxy + go.mod versioning. |
| `interfaces_current` | API surface tracked via `cmd/ailang/help.go` (single source of truth). |
| `automated_integration_testing` | `make test` runs full integration suite in CI. |
| `regression_tests_added50` | Bug-fix PRs include regression tests (sprint-evaluator gates this). |
| `test_policy_mandated` | CONTRIBUTING.md + .claude/rules/coding-standards.md mandate tests. |
| `tests_documented_added` | Already Met at passing. |
| `warnings_strict` | Already Met at passing. |
| `implement_secure_design` | Capability-based effect system enforces least-authority. |
| `crypto_weaknesses` | Already Met at passing. |
| `crypto_used_network` | All HTTPS; effects/net.go uses Go stdlib TLS. |
| `crypto_tls12` | Go 1.25 supports TLS 1.2 + 1.3 (defaults to 1.3). |
| `crypto_certificate_verification` | Go default verifies cert chain; no custom InsecureSkipVerify. |
| `signed_releases` | Cosign keyless on every release since v0.14.1. |
| `version_tags_signed` | Tags are SSH-signed by sunholo-voight-kampff. |
| `input_validation` | Parser uses Hindley-Milner type checking + capability gating at boundaries. |
| `static_analysis_common_vulnerabilities` | Already Met at passing. |
| `dynamic_analysis_unsafe` | Already Met at passing. |

### Met after small new artifact (~5 fields)

Each needs a short doc (1–2 hours total).

| Criterion | What to add |
|---|---|
| `code_of_conduct` | CODE_OF_CONDUCT.md — Contributor Covenant 2.1 template (~80 lines, copy-paste). |
| `governance` | GOVERNANCE.md — describes maintainer model + AI coordinator decision-making. ~50 lines. |
| `roles_responsibilities` | Section in GOVERNANCE.md (same file). |
| `documentation_architecture` | ARCHITECTURE.md — high-level diagram + pointers to internal/ packages. ~60 lines. Or repurpose existing material from docs/docs/guides/. |
| `documentation_security` | Add a "Security model" section to docs/docs/guides/ — capability/effect system + threat model. May already exist; just need link. |

### N/A with justification (~3 fields)

| Criterion | Why N/A |
|---|---|
| `crypto_algorithm_agility` | AILANG ships no in-house crypto. Stdlib + cosign handle algorithm agility. |
| `crypto_credential_agility` | No managed secret store; cosign keyless uses ephemeral OIDC. |
| `crypto_verification_private` | No private cert verification (we publish public sigs only). |
| `sites_password_security` | No password auth on docs site or dashboard public surface (GitHub OAuth only). |
| `internationalization` | Not applicable — AILANG is a programming language whose syntax is English keywords. CLI/error messages are English-only by design. |
| `accessibility_best_practices` | Docs site uses Docusaurus defaults (WCAG-aligned). Project itself is a CLI/library, not a UI. |

### Honest Unmet — silver blockers (2 fields)

| Criterion | Reality | Path forward |
|---|---|---|
| `bus_factor` | One active human maintainer (mark@aitanalabs.com); one AI coordinator account (sunholo-voight-kampff). The "second human" doesn't exist. | Recruit a second co-maintainer with merge rights, OR mark Unmet with a written justification linking to the AI-autonomous thesis. |
| `test_statement_coverage80` | Current: **37.8%**. Silver requires ≥80%. | Coverage push as a separate sprint. ~6,000 lines of new tests across the lower-coverage packages. Worth doing for its own sake; not a quick win. |

### Unanswered higher-effort items (~5 fields)

| Criterion | Status | Notes |
|---|---|---|
| `dco` | Likely Unmet → mark **N/A justified**: AI-coordinator commits use `Co-Authored-By` rather than DCO sign-off. Document the trailer model. |
| `access_continuity` | Could be Met if there's a written succession plan. Otherwise needs a 1-paragraph addition to GOVERNANCE.md describing what happens if the maintainer is unreachable. |
| `maintenance_or_update` | Met (active development), just answer. |
| `hardening` | Partial — docs site has CSP via Docusaurus, but no formal hardening doc. Add a section to SECURITY.md or hardened_site? |
| `assurance_case` | Unmet without writing one. A short security-design assurance argument doc (~150 lines) is real work. Could be deferred or marked N/A justified. |

## Realistic silver outcome

**Achievable today (URL-fill only)**: ~33/48 = **70%**

**Achievable with 1-2 hours of doc writing** (CODE_OF_CONDUCT + GOVERNANCE + ARCHITECTURE + a few N/A justifications): ~38/48 = **80%**

**Hard blockers**: `bus_factor` (no second maintainer) and
`test_statement_coverage80` (37.8% vs 80%). CII rules require *all*
silver criteria to be Met or N/A — a single Unmet blocks the badge.

So the badge itself is **not achievable** without either:
1. Recruiting a second co-maintainer (changes the project model), or
2. A coverage push to ≥80% (real engineering work, ~weeks).

## Recommendation

Stop at **passing**. The remaining gap to silver isn't worth chasing:
- Coverage at 37.8% reflects that AILANG is in fast-iteration phase
  (v0.x). Forcing 80% coverage right now creates throwaway tests
  against unstable APIs. Revisit after v1.0 when the surface stabilises.
- Bus factor is the actual project shape — pretending otherwise on a
  badge form would be theatre. Honest signal: solo maintainer + AI
  coordinator + extensive third-party verification.

The narrow exception worth doing anyway: **add CODE_OF_CONDUCT.md and
GOVERNANCE.md**. They're cheap, useful for community legitimacy
independent of the badge, and would unblock 4–5 silver criteria
should circumstances change. Let me know if you want those drafted.

## If you do want to push silver anyway

The URL-fill batch for the ~33 easy-Met fields would look like the
passing-tier URL we generated, just with silver criterion keys. Send
me word and I'll produce it. The two blockers (`bus_factor`,
`test_statement_coverage80`) would still keep the badge at "in
progress" until they're resolved — silver requires zero Unmets.
