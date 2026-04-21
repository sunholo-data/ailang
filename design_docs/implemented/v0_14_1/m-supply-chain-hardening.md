# M-SUPPLY-CHAIN-HARDENING — OpenSSF Scorecard response

**Status**: planned
**Target**: v0.13.1
**Related**: [M-SONAR-GATE-CLEANUP](m-sonar-gate-cleanup.md) (parallel quality-signal work)
**Baseline score** (2026-04-21): 4.6/10
**Target score**: ~7.5/10 (ceiling set by deliberate opt-outs below)

## Context

AILANG ships on the premise that its code is written autonomously by AI
agents via its own coordinator. That premise requires third-party
verification — "don't trust us, trust the static analysers." The repo
currently publishes three such signals: SonarCloud (correctness /
reliability / maintainability), Go Report Card (Go code health), and
OpenSSF Scorecard (supply-chain security).

Scorecard runs 18 checks against the repo and currently scores 4.6/10.
This document records which checks we're fixing, which we're
deliberately leaving at 0, and — importantly — why the "obvious"
fixes for some are security theatre and should be rejected.

The immediate work captured here has already shipped in commits
`77e98044` (SECURITY.md + workflow permissions) and `59586e02`
(CodeQL); what remains is the release-signing work and a written
decision record for everything else.

## Scorecard baseline — what fails and why it matters

| Check | Score | Decision | Next step |
|---|---|---|---|
| Dependency-Update-Tool | 10 | — | keep Dependabot |
| Dangerous-Workflow | 10 | — | — |
| Maintained | 10 | — | — |
| License | 10 | — | — |
| Packaging | 10 | — | — |
| Fuzzing | 10 | — | — |
| CI-Tests | 10 | — | — |
| Security-Policy | 0 → 10 | **fixed** | SECURITY.md shipped |
| Token-Permissions | 0 → ~8 | **fixed** | top-level `contents: read` on 5 workflows |
| Branch-Protection | 3 → 6 | **fixed** | no force-push / no delete / required status checks, *no review requirement* |
| SAST | 0 → 10 | **fixed** | CodeQL workflow added |
| Binary-Artifacts | 6 → 10 | **will fix** | move .wasm + sim_stub binaries out of repo, build in CI |
| Signed-Releases | 0 → 10 | **will fix** | cosign keyless + SHA256 + install-script verify — the real win |
| Pinned-Dependencies | 0 | **reject** | theatre — see below |
| Vulnerabilities | 0 | **triage, not chase** | pull osv-scanner output, fix what's real, suppress what's transitive-only |
| CII-Best-Practices | 0 | **reject** | pure form-filling |
| Code-Review | 0 | **reject on principle** | the whole project thesis conflicts with this check |
| Contributors | 6 | **accept** | grows organically with community |

## Decisions — what's real work vs what's theatre

### Real and worth doing

**Signed-Releases + install-script verification.**
This is the only Scorecard category where doing the full thing — including
end-user verification — delivers genuine attacker-surface reduction.
Today's installer (`docs/static/install.sh`) contains a checksum-check
stanza that *always silently skips* because no `.sha256` files are
attached to releases. That's worse than no check at all: it gives the
impression of verification while performing none. A `curl | bash`
install that doesn't verify is the classic supply-chain soft target.

Scope:
1. Release workflow generates `SHA256SUMS` and per-artifact `.sig` /
   `.pem` files via cosign keyless (workflow-OIDC identity, no key
   management).
2. Installer downloads the sigstore bundle, verifies the cert chain
   matches the expected `sunholo-data/ailang` workflow identity, then
   verifies the blob signature before extracting.
3. If cosign isn't on the user's PATH, fall back to SHA256 verification
   (still a real check now, not the silent-skip hack we have today).
4. Document the verification chain in SECURITY.md and the install docs.

The value isn't "get Scorecard to 10" — it's that someone piping
`install.sh` to bash gets a binary whose provenance can be traced to
a GitHub Actions run in *our* repo.

**CodeQL (already shipped in 59586e02).**
Sonar does pattern-level Go lint; CodeQL's `security-extended` pack
does inter-procedural taint tracking. For a language runtime that
executes user programs and exposes a capability/effect system, taint
flow from untrusted input → file open / exec / HTTP is the exact bug
class worth chasing. Different scanner, different bug class — not
redundant with Sonar.

**Token-Permissions (already shipped in 77e98044).**
Real. A compromised GitHub Actions run with the default (no explicit
permissions) can push to the repo, alter issues, create releases. With
top-level `contents: read` plus per-job overrides only where write is
actually needed, a hijacked run has a much smaller blast radius. This
one is not theatre — it's least-privilege applied to the automation
the AI coordinator already heavily relies on.

**Branch-Protection on dev (already shipped).**
Partly real. `allow_force_pushes: false` and `allow_deletions: false`
stop accidental history wipes — genuinely useful when the coordinator
is pushing autonomously. Required status checks on PRs catch red CI
before merge. What we *didn't* enable (`required_pull_request_reviews`)
is where theatre starts; see the Code-Review section below.

**Binary-Artifacts (will fix).**
`.wasm` files and the `sim_stub` demo binaries live in the source tree.
Scorecard treats checked-in executables as a supply-chain smell —
a hostile contributor (or a compromised coordinator run) could swap
the binary for a trojan and the PR diff would show "binary file
changed" with no reviewable content. Move them out: generate wasm in
CI on each deploy, attach sim_stub to releases.

### Reject as theatre

**Pinned-Dependencies.** The check wants every
`uses: actions/checkout@v6` pinned to a SHA
(`uses: actions/checkout@abc1234...`). The theory: prevent a future
tag re-point from hijacking the action. The reality:

- Dependabot is configured to bump action versions. Pinning by SHA
  means Dependabot rewrites the pins regularly.
- PRs that only change action SHAs train reviewers (human or AI) to
  rubber-stamp them — which *is* the attack vector. A malicious SHA
  looks identical to a legitimate one in a diff.
- The actions the project consumes are overwhelmingly
  `actions/*` and `github/*` — first-party GitHub-owned. The trust
  boundary is GitHub itself.

Pinning without a parallel discipline of manual SHA review is worse
than unpinned: it costs maintenance, it consumes attention, and the
attention it consumes is specifically attention on the attack surface
it's supposed to defend. Skip until/unless we're prepared to gate
Dependabot action-PRs on human review — which we're not.

**CII-Best-Practices badge.** Self-assessment form at
bestpractices.dev. Answering "yes we have a README" in 100 different
phrasings produces a badge that correlates with nothing. The useful
signal is already on the README (License, Reliability, Security,
Scorecard). Adding another badge of dubious information content is
dilutive. Skip.

**Code-Review = 0.** Scorecard wants two-human-approved merges for
every PR. AILANG's published thesis is AI-autonomous development via
the coordinator. Two incompatible resolutions:

1. Drop the autonomy claim and require human review. Kills the
   experiment that is the project.
2. Accept Code-Review = 0 and document *why*, so the Scorecard score
   reflects a known deliberate tradeoff rather than sloppiness.

We take option 2. The offsetting verification comes from **everywhere
else**: Sonar's reliability/security/maintainability ratings, CodeQL
SAST, Fuzzing (10/10), CI-Tests (10/10), the public benchmark
dashboard that runs 33 benchmarks across 8 frontier models on every
release, and — post this sprint — signed releases.

A single-digit Code-Review number next to a 10/10 everywhere-else
picture is the *honest* signal for a project that's chosen a different
path on the human-in-the-loop question. Hiding it behind
pseudo-compliance would be worse.

### Go Report Card `license` false negative

Cosmetic finding, upstream bug in goreportcard's license detector.
The repo has `LICENSE` (Apache 2.0) at the root; the detector misses
it anyway. Not worth a design decision — just tolerate the 94%/A+
instead of 99%/A+ until upstream fixes it.

## Implementation plan — Signed-Releases

Only the unshipped work is laid out below. Everything in the "already
shipped" column is cross-linked from the commit SHAs above.

### M1 — SHA256 sums on every release

**Goal**: make the installer's existing checksum check actually fire.

- Add a `compute-checksums` step to `release.yml` that runs after all
  build matrix jobs complete and before the release is published.
- Produce per-artifact `<name>.sha256` files and a combined
  `SHA256SUMS` file.
- Attach them to the release via the same `softprops/action-gh-release`
  step that handles the other artifacts.
- Update `install.sh` to *require* checksum verification (drop the
  silent-skip fallback) when the `.sha256` file is fetched
  successfully. If the fetch 404s on an older release, fall back to
  no-verify with a visible warning.

Acceptance:
- v0.13.1 release has `SHA256SUMS` + per-artifact `.sha256` files.
- Running the installer against v0.13.1 prints "Verifying checksum
  ... OK" and fails loudly if corrupted.

### M2 — Cosign keyless signing

**Goal**: cryptographic provenance tying the binary to a specific
GitHub Actions workflow run in `sunholo-data/ailang`.

- Add `sigstore/cosign-installer` to `release.yml`.
- For each published artifact, run
  `cosign sign-blob --yes --output-signature X.sig --output-certificate X.pem X`
  (keyless, uses the workflow's OIDC identity — no secrets to manage).
- Attach `.sig` and `.pem` files to the release.
- Declare the expected identity in SECURITY.md:
  `--certificate-identity-regexp "https://github.com/sunholo-data/ailang/.github/workflows/release.yml@refs/tags/v.*"`
  `--certificate-oidc-issuer "https://token.actions.githubusercontent.com"`.

Acceptance:
- v0.13.1 release has `.sig` + `.pem` for every binary artifact.
- Manual verification works:
  `cosign verify-blob --certificate X.pem --signature X.sig --certificate-identity-regexp '...' --certificate-oidc-issuer '...' X`
- OpenSSF Scorecard's Signed-Releases check reports 10/10 on the next
  weekly run.

### M3 — Installer cosign verification

**Goal**: end users running `curl | bash` get real verification, not
a silent skip.

- Detect `cosign` on PATH. If present, download `.sig` + `.pem`
  alongside the archive and `cosign verify-blob` before extracting.
- If cosign isn't installed, fall back to SHA256 (M1) with a clear
  `warn` line: "cosign not found — falling back to SHA256. Install
  cosign for provenance verification: https://..."
- If *both* the signature and the checksum fetch fail, abort with
  `err` — no silent accept.
- Add a `--no-verify` opt-out flag for local testing. Undocumented in
  README but handled in the script.

Acceptance:
- Fresh VM + cosign installed: install succeeds, prints
  "Signature verified: provenance GitHub Actions
  sunholo-data/ailang@refs/tags/v0.13.1".
- Fresh VM without cosign: install succeeds, prints SHA256 OK with
  a visible fall-back warning.
- Tampered artifact (`truncate -s 0`): install fails with a loud
  error, binary not installed.

### M4 — Binary-Artifacts cleanup

**Goal**: close Scorecard's Binary-Artifacts 6/10 finding + remove
the "binary file changed" blind spot in diffs.

Files currently in-tree:
- `docs/static/wasm/ailang.wasm`
- `web/ailang.wasm`
- `examples/sim_stub/sim_stub`
- `examples/sim_stub/sim_stub_release`

Plan:
- wasm: `docusaurus-deploy.yml` already runs `make build-wasm` then
  copies `bin/ailang.wasm` → `docs/static/wasm/ailang.wasm` at deploy
  time. No workflow change needed — the checked-in binaries are pure
  duplication. `web/ailang.wasm` has no referenced consumer; users are
  directed via `web/README.md` to either `make build-wasm` or fetch
  the `ailang-wasm.tar.gz` release bundle. Remove both; `*.wasm` is
  already globbed in `.gitignore` so they stay gone.
- sim_stub: `make test-sim-stub` invokes `make clean && make test`
  inside `examples/sim_stub/`, which regenerates `gen/` and rebuilds
  the binary. The `test-game-codegen.yml` workflow runs this on every
  push/PR — so the checked-in binary is never consumed by CI and the
  local `make test` path covers the developer use case. Remove
  `sim_stub` and `sim_stub_release`; add them to `.gitignore`.

Acceptance:
- `git ls-files | xargs file` reports zero ELF/wasm/Mach-O binaries.
- `make test-game-codegen` still passes locally without network.
- Scorecard Binary-Artifacts reports 10/10.

## Projected score after this sprint

| Check | Before | After |
|---|---|---|
| Security-Policy | 0 | 10 *(shipped)* |
| Token-Permissions | 0 | ~8 *(shipped)* |
| Branch-Protection | 3 | 6 *(shipped)* |
| SAST | 0 | 10 *(shipped)* |
| Signed-Releases | 0 | 10 *(this sprint)* |
| Binary-Artifacts | 6 | 10 *(this sprint)* |
| Pinned-Dependencies | 0 | 0 *(deliberate skip)* |
| CII-Best-Practices | 0 | 0 *(deliberate skip)* |
| Code-Review | 0 | 0 *(deliberate — documented)* |
| Vulnerabilities | 0 | TBD *(separate triage)* |

Baseline 4.6 → projected ~7.5 once M1–M4 land. Ceiling of ~8.5 given
the three deliberate-skip checks. Any push past that would require
either weakening the AI-autonomous stance (Code-Review) or performing
upkeep whose cost exceeds its security benefit (Pinned-Dependencies).

## Out of scope for this sprint

- Vulnerabilities (Scorecard reports 22 open). Needs a separate pass
  to pull osv-scanner output, classify direct vs transitive,
  and decide fix-or-suppress per finding.
- Weekly Scorecard cron is already wired (`scorecard.yml`); no change
  needed — it'll re-score on Monday 06:30 UTC and confirm the
  shipped deltas.

## References

- OpenSSF Scorecard docs: https://github.com/ossf/scorecard/blob/main/docs/checks.md
- Cosign keyless signing: https://docs.sigstore.dev/cosign/signing/overview/
- Current score: https://securityscorecards.dev/viewer/?uri=github.com/sunholo-data/ailang
- API: https://api.securityscorecards.dev/projects/github.com/sunholo-data/ailang
