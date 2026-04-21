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

### What to expect

- **Acknowledgement** within 3 business days
- **Initial assessment** (confirmed / not a vulnerability / need more info) within 7 days
- **Fix timeline**: critical issues within 14 days; lower severity on the
  next minor release
- **Credit**: reporters are credited in the release notes unless they request
  otherwise

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
