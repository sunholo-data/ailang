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
`@earendil-works/pi-coding-agent` (0.84.x+). Upgraded 0.73.1 → 0.84.4 on 2026-08-31.

- Upgrade: `npm i -g @earendil-works/pi-coding-agent@<version>`
- Rollback to pre-move: `npm rm -g @earendil-works/pi-coding-agent && npm i -g @mariozechner/pi-coding-agent@0.73.1`
- Post-checks (all four, measured 2026-08-31):
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
