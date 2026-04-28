# Governance

AILANG is an experimental programming language whose compiler, stdlib,
and tooling are developed primarily by AI agents driven by an in-repo
[coordinator](https://ailang.sunholo.com/docs/guides/coordinator). This
document describes who makes decisions, how, and what happens if the
project changes hands.

## Project model

AILANG operates a **single-maintainer + AI-coordinator** model. This
is not the most common open-source governance shape; the document
calls it out plainly so contributors understand what they're signing
up for.

| Role | Identity | Responsibilities |
|---|---|---|
| Lead maintainer | Mark Edmondson, mark@aitanalabs.com (GitHub: MarkEdmondson1234) | Final decision-maker on language design, release timing, security disclosures, and merging to `main`. Owns the bestpractices.dev / Scorecard / Sonar accounts. |
| AI coordinator agent | sunholo-voight-kampff (GitHub bot account) | Implements approved sprint plans via the coordinator. Cannot merge to `main` without lead-maintainer approval. Pushes directly to `dev` for direct-push velocity within an approved sprint. |
| Contributors | Anyone | Open issues, propose design docs, submit PRs to `dev`. See [CONTRIBUTING.md](CONTRIBUTING.md). |

## Decision-making

Design changes flow through written design docs in `design_docs/`.
The lead maintainer approves a doc before any sprint is planned;
sprint plans (in `design_docs/planned/<version>/m-*-sprint-plan.md`)
require the same approval. Implementation runs autonomously by the
AI coordinator under that approved plan; the maintainer reviews
sprint output before promoting `dev` → `main` and cutting a release.

Disagreements are resolved by the lead maintainer. There is currently
no voting body, technical steering committee, or vote-of-no-confidence
mechanism. If/when contributors emerge with sustained merge rights,
this section will be updated to describe the resulting structure.

## Bus factor and access continuity

The project's bus factor is **1**. The lead maintainer holds:

- The `mark@aitanalabs.com` mailbox and the GitHub Security
  Advisories notification routing.
- Push and admin rights on `github.com/sunholo-data/ailang`.
- The cosign keyless signing identity (workflow OIDC; recoverable
  from the workflow definition itself, not a held key).
- The `ailang.sunholo.com` DNS and the install-script hosting.

If the maintainer becomes unreachable for >90 days, the project's
recovery plan is:

1. Sunholo Ltd (the legal entity owning the GitHub organization,
   registered in the United Kingdom) holds organization-admin rights
   independently of the lead maintainer's personal account and can
   appoint a successor.
2. The cosign signing identity is workflow-bound, not key-bound — a
   successor with merge rights to `release.yml` automatically
   inherits the ability to sign new releases. Old releases remain
   verifiable via the public Rekor transparency log.
3. The `ailang.sunholo.com` domain and registry are owned by Sunholo
   Ltd; transfer follows standard registrar processes.

This is not a substitute for a real second maintainer. It is the
honest current state.

## Adding maintainers

A second maintainer would be added when:

- A contributor demonstrates sustained design judgment (typically
  4+ approved design docs landed) and review competence.
- The lead maintainer and the candidate explicitly agree to the
  addition.
- The change is announced in CHANGELOG.md and reflected in this
  document.

There is no minimum bar for "contributor in good standing." The
project welcomes single-PR contributions; the maintainer bar is
distinct and higher.

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Enforcement decisions
are made by the lead maintainer.

## Security disclosures

See [SECURITY.md](SECURITY.md). Vulnerability reports are handled by
the lead maintainer with the SLA documented there.

## License changes

The project is Apache-2.0. License changes require explicit notice
in CHANGELOG.md and a major-version bump. The lead maintainer cannot
unilaterally change the license to a more restrictive one without
contributor consent — Apache-2.0 grants are irrevocable for past
contributions.

## This document

Updates to GOVERNANCE.md are made by the lead maintainer. Material
changes (e.g. moving to a multi-maintainer model, changing the
decision-making process) are announced in CHANGELOG.md.
