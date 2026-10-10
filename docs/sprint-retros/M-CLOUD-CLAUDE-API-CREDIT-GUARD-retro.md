# Sprint retrospective: M-CLOUD-CLAUDE-API-CREDIT-GUARD

The approved sprint added a request-reserving shared Anthropic credit authority,
a dedicated gateway, isolated cloud jobs and explicit operator controls. The
repaired release is v0.54.1. Dev/prod canaries share one grant and reconciled
13,025 microUSD across four complete provider receipts. The first regular lane
is the existing read-only Haiku sprint evaluator.

## Scope and validation

Seven milestones cover inventory, transactional accounting, exact pricing,
gateway validation, executor routing, operator commands and production rollout.
Independent workers owned accounting, operator commands and pricing; an independent
evaluator reviewed their integration and rollout readiness. Full tests/lint/CI,
two-client Firestore races, installed cloud CLI completion/tool loops and all12
mocked Terraform plans passed. Production activation uses Terraform CI only.

The original estimate was2,610 lines across9 days. Measured source additions in
commits d1f2fc4c7,81f9a3772,be15f1042 total10,118, with131 deletions, including
tests and documentation. This measure excludes unrelated merges/release bumps
and infrastructure changes. The extra scope was strict installed-CLI compatibility,
immutable recovery operators, retained provider receipts and broad failure/identity
regressions; the original estimate understated that work.

## Friction and decisions

- The installed CLI sent beta features outside the supported cost contract.
  Reviewed client proxy compatibility suppressed them; gateway rejection remained.
- The first live stream exposed cumulative usage and nullable detail contracts.
  The failed reservation remained held; repaired fixtures and billing-only receipts
  preceded new canaries. Unknown exposure was booked in full without a refund.
- Provider Console rounding/lag could not settle original requests. User-attested
  balance consumption was separately counted conservatively, allowing overlap to
  reduce capacity rather than inventing charge attribution.
- Raw agents-only CI uploads could erase Terraform-rendered credit routing.
  Presence of credit variables now delegates synchronously to Terraform CI.
- Simultaneous registry/config builds hit a Terraform state lock. Retry after the
  owning build completes; never force-unlock or disable state locking. A queued
  activation build was cancelled and rerun after its preceding metadata build.
- Cloud Run execution image IDs resolve the OCI index's platform child digest;
  verifying its descriptor avoids confusing it with a different release.

## Follow-ups

Evaluate federated Anthropic credentials before revisiting deployment administrator
IAM through Terraform. No folder IAM changes belong in this rollout. Add
automated50/75/90% spend notifications and extend explicit confirmation/accounting
to other provider credit pools in separate approved sprints. Keep expiry/manual
fresh-credit confirmation and the shared authority under federation.
