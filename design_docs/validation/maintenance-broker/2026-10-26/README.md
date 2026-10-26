# Broker M1 — escaped-descendant stop gate

**Qualification: BLOCKED. Broker sprint incomplete (0/7 milestones passed).**
Live containment admission remains closed. No model/provider calls or campaign
were run. M2–M7 are unstarted.

The actual locked sandbox runtime admits a child that calls `setsid`, changes cwd
to `/`, and closes descriptors 0 through 255. That child survives the current
process-group kill followed by the real `proctree.ReapWorkspace` fallback.

| Control | Observed after group kill + real cwd reap |
|---|---|
| Detached child retains workspace cwd | Gone; actual reaper reports its PID |
| Escaped child, normal leader exit | Live `Ss`; matching start-time/command identity |
| Escaped child, group timeout | Live `Ss`; matching start-time/command identity |
| Escaped child, fixture owner killed | Live `Ss` even after an external group kill + cwd fallback |
| Unrelated bystander | Alive through observation, then joined by the test |
| Owned Python file work | Passed inside the actual sandbox |
| Owned AILANG check and run | Passed inside the actual sandbox |

Every fixture child is explicitly cleaned up after observation. The trusted
fixture PID is used only for emergency test cleanup, never as a proposed
production ownership mechanism. The fixture also self-expires after 30 seconds.

## Reproduce

Use the actual macOS host outside an enclosing tool sandbox, Node 26.12.1, Go,
Python 3.13.7 and the observed AILANG binary on PATH. The report records actual
versions/hashes; this is a macOS arm64/kernel 25.6.0 observation, not a portability
claim. No installed/global pi configuration is changed. The locked SDK package
is sandbox-runtime 0.0.71.

From the repository root:

```bash
npm ci --prefix tools/pi-extensions/sandbox --ignore-scripts --no-audit --no-fund
MAINTENANCE_FEASIBILITY_REPORT=/private/tmp/broker-blocker-new.json \
  node --test tools/pi-extensions/maintenance-broker-feasibility.test.mjs
```

The default test asserts the **known failure** and passes only if it reproduces
the expected controls. Its passing exit does not mean ownership is qualified.
Evidence output must be a new absolute path (create-only); do not overwrite a
previous report.

To require actual lifetime qualification instead:

```bash
MAINTENANCE_REQUIRE_LIFETIME=1 \
  MAINTENANCE_FEASIBILITY_REPORT=/private/tmp/broker-strict-new.json \
  node --test tools/pi-extensions/maintenance-broker-feasibility.test.mjs
```

The strict command exits **1**, with
`M1 STOP: detached cwd-changing child survives current host cleanup`.
Both modes write `qualified:false`, `backendAdmitted:false` and a named blocker.
An acquisition error is not a successful blocker reproduction.

## Banked artifacts

- `blocker-control.json` and `.log`: expected-blocker regression mode, exit 0,
  one test passed, zero skips. All three escaped children remain live before
  emergency fixture cleanup; the workspace-cwd control is reaped.
- `strict-lifetime.json` and `.log`: same source and controls, strict ownership
  assertion, exit 1. This is an expected failed qualification, not a green gate.
- `independent-review.json`: independent source/artifact review; no independent
  process rerun. The reviewer found the precise stopped-gate claim supported.
- `checks.txt`: focused Go/core tests, lint, source syntax and repository gates.
- `index.json`: artifact digests and qualification status.

Test log trailing whitespace is normalized for repository hygiene; substantive
output is preserved. The index hashes the committed forms.

Reports bind wrapper, canary, test, lock, thin reaper helper, production proctree
sources and compiler bytes. Each row includes the exact temporary policy and
its digest. Temporary source/policy paths are destroyed after the fixture joins;
their data and identities are retained in the reports.

The Go helper merely invokes the existing `ReapWorkspace`; the controller's
negative-PGID SIGKILL is the existing group-kill operation. In the owner-death
case, the *test* applies an external cleanup attempt after killing its fixture
owner. This does not implement or qualify a production parent-death supervisor.

## Scope and next decision

M1 was ordered to test lifetime feasibility first. The approved sprint's stop
rule says: “On any failed property, write `qualified:false` with the named blocker
and leave M2–M7 unstarted.” That rule was followed. The sibling socket/pi chain,
full source inventory and protected network/read canaries were not run here.

This result refutes sufficiency of the **current cleanup recipe** for the tested
escape. It does not prove macOS cannot support an owner or that the proposed
broker architecture is impossible. A new bounded design must choose and qualify
mechanical escape prevention or ownership that survives detachment, reparenting,
cwd changes and owner death before the remaining transport is built. A polling
PID/cwd scan or a candidate-reported PID list is not such a guarantee.

Do not extend the platform, weaken isolation, add paid canaries or mark the
original adapter sprint complete as a consequence of this report.
