# Reliability implementation verification

Isolated branch: `sprint/mission-iteration`. M1–M3 committed at `48e72ef7e`; M4 live adoption incomplete after evaluator noncompletion. See [live result](live-trial.md).
No provider calls are included in the hermetic checks below.

| Check | Result | Retained log |
| --- | --- | --- |
| Full `make test`, including Pi extension suite | PASS | `/private/tmp/reliability-full-tests-final.log` |
| Internal mission/coordinator tests | PASS | `/private/tmp/reliability-internal-tests.log` |
| Dispatch/iteration/activation/coordinator race tests | PASS | `/private/tmp/reliability-race-tests.log` |
| Final CLI lifecycle/recovery race tests | PASS | `/private/tmp/reliability-cli-race-final.log` |
| New multi-role retry approval tests/race | PASS | Worker independently ran full iteration suite and new race cases |
| Retained binding/progress race boundaries | PASS | `/private/tmp/reliability-retained-race-final.log` |
| Full `make lint` | PASS, zero issues | `/private/tmp/reliability-lint-final.log` |
| `make build` | PASS | `/private/tmp/reliability-build.log` |
| Architecture boundaries | PASS | `/private/tmp/reliability-boundaries.log` |
| File sizes | PASS, all within 800 lines | `/private/tmp/reliability-file-sizes.log` |
| Binary iteration driver fixture | PASS | `/private/tmp/reliability-driver-tests.log` |
| Real failed-canary retry preparation and dry-run | PASS, no dispatch | `/private/tmp/ailang-reliability-trials/` |

The first internal suite could not bind a loopback HTTP test server under the sandbox;
it passed with local server permission. The first full suite observed an in-progress
multi-role test fixture that tried to accept an unapproved designer output. The fixture
was corrected without weakening the runtime gate; the final full run above passed.

Coverage includes real subprocess death at 13 installation mutation checkpoints,
concurrent host activation, exact prior binding restoration, foreign marker/edits,
frozen child release, cancelled and ambiguous outcomes, typed no-dispatch failure,
read-only status after restoration, partial receipt progress, immutable request replay,
provenance/authority tampering, all missing prerequisite approvals and evaluator-only dispatch.

The current implementation defaults preserve old request/report digests when optional
new fields are absent. Imported prerequisite manifests explicitly identify unavailable
original acceptance digests; they do not invent a complete receipt chain.

Live trial source, metered/list-price costs, fresh/cache tokens, interventions and cleanup
will be recorded separately. A passing code review is not a successful live adoption claim.
