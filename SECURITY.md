# Security Policy

## Supported Versions

AILANG is pre-1.0 and evolves rapidly. Security fixes are applied to the latest
minor release on the [Releases page](https://github.com/sunholo-data/ailang/releases).
Earlier versions are not patched — upgrade to the latest release to receive fixes.

| Version | Supported |
|---------|-----------|
| Latest release (see [CHANGELOG.md](CHANGELOG.md)) | Yes |
| Older releases | No |
| `dev` branch | Best-effort; rolling development branch |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Use one of these private channels instead:

1. **GitHub Security Advisories (preferred)** —
   <https://github.com/sunholo-data/ailang/security/advisories/new>.
   This creates a private draft that only maintainers can see.

2. **Email** — `mark@aitanalabs.com` with subject `AILANG security:`
   followed by a short description.

### What to include

- A description of the issue and its impact
- Steps to reproduce (a minimal `.ail` program or command line is ideal)
- The AILANG version (`ailang --version`) and OS/architecture
- Any suggested fix or mitigation you've already considered

### Response SLA

This is a public commitment, not a target. If we can't meet it for a
given report, we'll say so explicitly in our reply.

- **Acknowledgement**: within 3 business days of receipt
- **Initial assessment** (confirmed / not a vulnerability / need more info): within 7 days
- **Fix timeline**: critical issues within 14 days; lower severity on the
  next minor release
- **Credit**: reporters are credited in the release notes unless they request
  otherwise

The 14-day fix SLA aligns with the [OpenSSF Best Practices](https://www.bestpractices.dev/)
`vulnerability_report_response` requirement.

## Scope

Vulnerabilities in scope include:

- The `ailang` CLI and compiler (`cmd/`, `internal/`)
- The standard library (`stdlib/`)
- The dashboard server and coordinator (`internal/server/`, `internal/coordinator/`)
- Build and release workflows under `.github/workflows/`
- Capability-system bypasses or effect-system soundness bugs

Out of scope:

- Issues in third-party dependencies — report those upstream. We track their
  advisories via Dependabot and OSV.
- Issues in generated AILANG code produced by external LLMs — those are a
  property of the model, not the language.
- Social-engineering or phishing against maintainers.

## Verifying Release Artifacts

Release archives are signed keyless with [Sigstore cosign](https://docs.sigstore.dev/cosign/) starting with v0.14.1. Each `.tar.gz` / `.zip` has a matching `<archive>.sig` (signature) and `<archive>.pem` (certificate) attached to the GitHub release. An aggregate `SHA256SUMS` file is also published.

The installer at `https://ailang.sunholo.com/install.sh` performs verification automatically: cosign first if available, SHA256 fallback otherwise. To verify manually:

```bash
# Download archive + signature + cert (replace arch/OS/version as needed)
REPO=sunholo-data/ailang
VERSION=v0.14.1
ARCHIVE=linux.x64.ailang.tar.gz
BASE="https://github.com/$REPO/releases/download/$VERSION"
curl -fsSLO "$BASE/$ARCHIVE"
curl -fsSLO "$BASE/$ARCHIVE.sig"
curl -fsSLO "$BASE/$ARCHIVE.pem"

cosign verify-blob \
  --signature "$ARCHIVE.sig" \
  --certificate "$ARCHIVE.pem" \
  --certificate-identity-regexp \
    "^https://github\.com/$REPO/\.github/workflows/release\.yml@refs/tags/v.+$" \
  --certificate-oidc-issuer \
    "https://token.actions.githubusercontent.com" \
  "$ARCHIVE"
```

A successful verification proves the archive was built and signed by a GitHub Actions run of [`.github/workflows/release.yml`](.github/workflows/release.yml) in this repository, against a `v*` tag. The signing certificate is logged to the public [Rekor transparency log](https://search.sigstore.dev/), so the provenance is independently auditable.

If cosign isn't available, the SHA256 fallback uses the per-archive `<archive>.sha256` files:

```bash
curl -fsSLO "$BASE/$ARCHIVE"
curl -fsSLO "$BASE/$ARCHIVE.sha256"
sha256sum -c "$ARCHIVE.sha256"   # or: shasum -a 256 -c on macOS
```

SHA256 confirms the archive wasn't corrupted in transit but does not by itself prove provenance — prefer cosign verification where possible.

### SLSA build provenance

Starting with v0.15.x, every release also carries [SLSA Build Level 3 provenance](https://slsa.dev/spec/v1.0/levels#build-l3) attached as `multiple.intoto.jsonl`. This is a separate claim from the cosign signature: cosign proves *this binary was signed by our workflow's OIDC identity*; SLSA provenance proves *which source commit, build steps, and inputs produced it*. The two compose.

Verify with [slsa-verifier](https://github.com/slsa-framework/slsa-verifier):

```bash
go install github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@latest

VERSION=v0.15.0
ARCHIVE=linux.x64.ailang.tar.gz
BASE="https://github.com/sunholo-data/ailang/releases/download/$VERSION"
curl -fsSLO "$BASE/$ARCHIVE"
curl -fsSLO "$BASE/multiple.intoto.jsonl"

slsa-verifier verify-artifact "$ARCHIVE" \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/sunholo-data/ailang \
  --source-tag "$VERSION"
```

A successful verification confirms the binary was produced by the
expected workflow run on the expected source tag. The provenance is
itself signed via Sigstore and logged to Rekor, same trust root as
the cosign signature above.

## Build Trust Boundary

CI workflows under [.github/workflows/](.github/workflows/) reference
two classes of GitHub Actions, with different pinning policies:

- **First-party (`actions/*`, `github/*`)** — pinned to major-version
  tags (e.g. `actions/checkout@v6`). The trust boundary is GitHub
  itself. SHA-pinning these would force constant Dependabot churn on
  identical-looking SHA bumps, which is itself an attack surface
  (reviewer rubber-stamping). Tags are the right granularity here.
- **Third-party (everything else)** — pinned to a 40-character SHA
  with a trailing version-tag comment for human readability:
  ```yaml
  uses: sigstore/cosign-installer@dc72c7d5c4d10cd6bcb8cf6e3fd625a9e5e537da  # v3.7.0
  ```
  Examples in this repo: `sigstore/cosign-installer`,
  `softprops/action-gh-release`, `SonarSource/sonarqube-scan-action`,
  `ossf/scorecard-action`, `docker/build-push-action`,
  `docker/setup-buildx-action`, `astral-sh/setup-uv`. A tag re-point
  on these actions actually changes the threat model (different
  organization holds the trust boundary), so SHA-pinning them is
  worth the maintenance cost.

Dependabot is configured to update both classes; for SHA-pinned
references it proposes new SHA + new comment together, preserving
the human-readable version tag.

## AI-Generated Code Disclosure

AILANG is developed autonomously by AI agents via its
[coordinator](https://ailang.sunholo.com/docs/guides/coordinator). Third-party
static-analysis badges on the README provide independent verification:

- **Reliability / Security / Maintainability** — [SonarCloud](https://sonarcloud.io/project/overview?id=sunholo-data_ailang)
- **Supply-chain** — [OpenSSF Scorecard](https://securityscorecards.dev/viewer/?uri=github.com/sunholo-data/ailang)
- **Go code health** — [Go Report Card](https://goreportcard.com/report/github.com/sunholo-data/ailang)

If you believe the AI-authored nature of the codebase has introduced a security
issue (e.g. a backdoor, unsafe default, credential leak), please report it via
the same channels above.
