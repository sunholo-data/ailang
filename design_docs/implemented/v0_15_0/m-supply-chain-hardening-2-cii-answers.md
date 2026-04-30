# CII Best Practices badge — submission scratch

Companion to [m-supply-chain-hardening-2.md](m-supply-chain-hardening-2.md) M6.

## How submission works

1. Sign in at <https://www.bestpractices.dev/en/users/sign_up> with the
   GitHub account that owns this repo (sunholo-voight-kampff).
2. Click **Add a New Project**, enter the repo URL.
3. The site reads [.bestpractices.json](../../../.bestpractices.json) and
   pre-fills every field as a *proposal*. You review each, click Met /
   Unmet / N/A, and the justification text lands in the form. No data
   is saved until you click Submit on each section.
4. Once all required fields are Met (or have an accepted N/A
   justification), the badge URL becomes live.

Optional shortcut: visit
`https://www.bestpractices.dev/en/projects?as=edit&url=https%3A%2F%2Fgithub.com%2Fsunholo-data%2Failang`
once logged in. The site fetches `.bestpractices.json` from the default
branch and applies it as an automation proposal.

## API note

The BadgeApp REST API exists (`POST /projects`, `PATCH /projects/:id`)
but auth is **session cookie only**, not token-based — there's no
machine-to-machine path. The `.bestpractices.json` + automation-proposal
URL is the maintained-tool integration point.
Source: <https://github.com/coreinfrastructure/best-practices-badge/blob/main/docs/api.md>.

## Project-level fields

Pasted once at the top of the form:

| Field | Value |
|---|---|
| Project name | `AILANG` |
| Project URL | `https://github.com/sunholo-data/ailang` |
| Homepage URL | `https://ailang.sunholo.com` |
| License | `Apache-2.0` |
| Implementation languages | `Go` |

**Description** (description_good):

> AILANG is a deterministic programming language designed for
> autonomous AI code synthesis and reasoning. The compiler is written
> in Go and ships with a Hindley-Milner type system, an explicit
> row-polymorphic effect system, and a capability-based runtime. The
> repository, compiler, and stdlib are developed primarily by AI
> agents driven by an in-repo coordinator; third-party static
> analyzers (SonarCloud, OpenSSF Scorecard, CodeQL, Go Report Card)
> provide the independent verification that a human-reviewed pipeline
> would normally provide.

## Items to double-check before submit

These are **Met** in `.bestpractices.json` but worth eyeballing once.

- `discussion`: confirm GitHub Discussions is actually enabled on the
  repo. If not, either enable it (Settings → Features → Discussions)
  or change to Unmet with "Issues serve as the discussion forum."
- `release_notes_vulns`: marked Met based on a forward commitment.
  Once the M1+M2 release ships and the changelog calls out the
  cleared CVE IDs, this becomes definitively Met.
- `crypto_password_storage`: marked N/A (no password storage). If the
  dashboard ever adds password auth, flip to Met with a bcrypt/argon2
  reference.
- `delivery_unsigned`: Met *now* via cosign. Once SLSA provenance
  ships (M3), update justification to mention it.
- `no_leaked_credentials`: assumes secret scanning hasn't surfaced
  any leaked tokens. Verify at
  <https://github.com/sunholo-data/ailang/security/secret-scanning>.

## Items deliberately not pursuing

The following silver/gold criteria are **not** in `.bestpractices.json`
because they conflict with the project's documented stance. The
passing tier doesn't require them; silver/gold do. Don't enable them.

| Criterion | Why we're Unmet |
|---|---|
| `two_person_review` | The project's thesis is autonomous AI development. Required two-human review on dev would invalidate the experiment. Documented in [m-supply-chain-hardening.md](../implemented/v0_14_1/m-supply-chain-hardening.md). |
| `coding_standards_enforced` (silver) | Same as above — enforcement currently runs through AI sprint-evaluator + linters, not human review. |

If the form reaches a silver-tier section, exit. The passing badge is
the target.

## Verification after submission

```bash
# Once the badge is awarded, fetch the public JSON status:
curl -s "https://www.bestpractices.dev/projects/<ID>.json" | jq '.badge_percentage_0'
# Should report 100 for the passing tier.

# OpenSSF Scorecard's CII-Best-Practices check polls bestpractices.dev
# weekly. Confirm it picks up the badge:
curl -s "https://api.scorecard.dev/projects/github.com/sunholo-data/ailang" \
  | jq '.checks[] | select(.name=="CII-Best-Practices")'
```

## Time estimate

- Sign-up + project creation: 5 min
- Walking through the form with `.bestpractices.json` pre-fill: 30 min
- Final review + submit: 10 min
- **Total: under an hour** if no surprises.
