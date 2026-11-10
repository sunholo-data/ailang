### Changed — the registry validator deploys on final release tags only, behind the CI gate (2026-11-10)

`cloudbuild-registry-validator.yaml` gains two steps that must pass before the validator image is
pushed or deployed (m-v1-release-gates M3a):

- `release-guard` fails the build for any tag that is not exactly `vX.Y.Z`, so a release candidate
  or a manual rebuild never moves the one prod validator.
- `ci-gate` runs `tools/ci/require_checks.sh` — the same script and the same required-check list as
  the GitHub release gate — and fails closed on a red, missing or still-pending check after 90
  minutes. The annotated-tag `release-gate-override:` marker is honoured as it is there.

A validator upgrade changes what every `ailang publish` accepts, at once, and published packages
are never re-validated; until now any `v*` tag redeployed it with no gate. The build timeout is
raised to 6600 s to cover the gate's polling window. There is no ungated rebuild-from-a-commit path
any more: a validator fix ships as a new final tag.
