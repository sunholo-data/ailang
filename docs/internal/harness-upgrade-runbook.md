# Harness upgrade runbook (rig)

The agent harness CLIs (pi, codex, opencode, claude, ollama) drift independently of this
repo. Drift is **surfaced weekly** — the `## Harness drift` section of
`tools/mission-weekly-report.py`, posted at every Monday bookkeeping-thread rotation —
and **applied only in attended sessions**, one tool at a time, per this runbook.
(Mark 2026-08-31: "we need to be careful on upgrades but need to keep on top of them.")

## Principles

- One tool per session; never mid-fire. Check `launchctl list | grep mission` and look
  for live agent processes of the tool being replaced before touching the binary.
- **Record the boundary.** Version-sensitive eval data must never pool across an
  upgrade (the three local-model baseline boundaries are the standing lesson). If the
  tool touches eval lanes, bank the date in the charter and memory the same session.
- Pin to a **verified** version, not blind-latest, when a test matrix exists (the pi
  extension suite names its tested version in `.pi/extensions/README.md`).
- Have the rollback command written down *before* upgrading.
- After upgrading, run the tool's post-checks below — a version number is not a verdict.

## pi (`@earendil-works/pi-coding-agent`, npm global)

The package **moved** from `@mariozechner/pi-coding-agent` (ends at 0.73.1) to
`@earendil-works/pi-coding-agent` (0.84.x+).

**The pin is `0.85.1`, in THREE places that a test holds equal** (M-PI-HARNESS-UPGRADE):
`internal/executor/pi` `ExpectedVersion` (HealthCheck refuses a mismatch — every eval and
mission dispatch runs it), and `ARG PI_VERSION` in `docker/Dockerfile.agent-pi`,
`Dockerfile.agent-eval`, `docker/resident/Dockerfile` (the build asserts `pi --version`);
`TestDockerfilesPinExpectedPiVersion` fails if they diverge. **An upgrade is therefore a
code change, not an `npm i`**: bump all four together, re-capture the differential
fixtures (`internal/executor/pi/testdata/v<version>/`, method: design doc M3.0 — the same
two directives on both versions, diff the event/field sets), and only then move the rig.

**Harness boundaries (eval rows must not pool across):**

| Plane | Boundary | From → to | Rows carry `executor_version`? |
|---|---|---|---|
| rig | 2026-08-31 | 0.73.1 → 0.84.4 | no (field landed 2026-09-16) |
| rig | ~2026-09-05 | 0.84.4 → 0.85.1 | no |
| cloud (`agent-pi`, `agent-eval`, `resident-pi`) | 2026-09-16 | 0.73.1 → 0.85.1 | yes, from the first build after `5ef7a2b24` |

The 0.73.1 → 0.84.4 rig boundary was sized after the fact from existing `os-rolling/v0.34.0`
rows (same AILANG version, same model `pi-qwen3-8-27b`, 77 vs 241 rows, 26 shared benchmarks):
**no benchmark got worse**; aggregate 91.9% → 99.1% but zero discordant pairs and the control
arm above the 90% headroom ceiling, so `eval-paired` reports it as unresolvable rather than as
a gain. The 0.84.4 → 0.85.1 step showed **no wire change** on any field the parser reads (V33)
and is not separately measurable on the rig (it coincides with AILANG releases).

- Upgrade: bump the four pins, re-capture fixtures, `go test ./internal/executor/pi/`, push (the
  dev build asserts the pin and runs `docker/test-agent-pi.sh`), then `npm i -g
  @earendil-works/pi-coding-agent@<version>` on the rig. Record the new boundary row above.
- Rollback: reverse the same four pins; `npm i -g @earendil-works/pi-coding-agent@0.85.1`.
  Rollback to the pre-move package is no longer supported — the parser now reads
  `usage.reasoning` and `rawStopReason` and refuses a `message_end` without usage.
- Post-checks (all four, re-measured 2026-09-16 on 0.85.1 — `quota_report` executed from a
  clean `ailang pi install` with no trust file, AND from the rig's checkout where the suite loads
  from the repo's own `.pi/extensions/` — the rig's global dir deliberately lacks the suite, see
  `pi_extension_collision.go`; `ailang pi status` reports that as `WORKSPACE`):
  1. Driver probe shape: `pi --mode json --no-session --no-tools --model
     ollama/glm-5.3-flash:cloud -p 'reply with exactly: ok'` → rc=0.
  2. Extension **execution** — NOT `pi list` (settings packages only) and NOT substring
     matching (the model echoing a tool name satisfies a grep): in a fresh worktree,
     `pi --mode json --no-session -p "call quota_report"` must produce
     `tool_execution_start`/`tool_execution_end` events naming `quota_report`.
  3. `make check-pi-wire-budget` → PASS (the min(declared, 32000) clamp).
  4. `message_end` still present in the event stream (`mission_pi_run.sh` banks it).
- **≥0.84 trust gate:** headless modes (`-p`, `--mode json`, `--mode rpc`) never prompt
  and, with no saved decision, **silently ignore** project-local `.pi/extensions` (the
  `defaultProjectTrust: "ask"` fallback). Saved decisions live in
  `~/.pi/agent/trust.json` — a flat map of canonical path → boolean, resolved by
  walking parent directories. The rig trusts the two mission roots
  `/Users/voightkampff/dev/sunholo-data` and `/Users/voightkampff/.ailang-driver-pin`,
  which cover every clone, sprint worktree and pin worktree. A NEW mission root needs
  its parent added there, or its fires run extension-less with no error anywhere.

## codex (`@openai/codex`, npm global)

- The 1-token probe **cannot see quota exhaustion** (rc=0 on a spent bucket; the real
  run dies mid-flight). Unchanged across upgrades so far — re-verify the driver's
  controller and role probes after upgrading.

## opencode (`opencode-ai`, npm global)

- The session DB is the tool-emission instrument; after upgrading confirm the schema
  the eval analysis reads still parses.

## claude

- Self-updating channel. The rig's headless billing rides keychain OAuth — after any
  update, re-verify no API key re-entered the environment (billing tripwire).

## ollama

- **Never casually.** Every ollama upgrade is a measured eval boundary (0.32.1→0.32.14
  was boundary #3). Requires: no rotation in flight, `launchd-hold` on GPU consumers,
  and a banked boundary date. Local-model eval data must not pool across it.

## ailang (PATH binary)

- Drifts **by design** on the shared rig; do not "keep it fresh" (installing mid-run
  disturbs concurrent agents). Scratch-build to a temp dir and prepend PATH for
  sessions that need HEAD behavior.
