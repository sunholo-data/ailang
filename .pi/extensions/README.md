# Session Protocol Gate + Dev-Harness Extensions

Project-local pi extensions for the AILANG repo (M-DX-SESSION-GATE, M-DX-PI-HARNESS).
Ships via git — every machine inherits on pull. Tested against pi **0.84.3**.

| Extension | What it does |
|---|---|
| `session-protocol-gate.ts` | Blocks edit/write + fail-closes bash until the session protocol is acked (see below); **also appends `Co-Authored-By: pi (<provider>/<model>)` to allowed git commits** — the pi-analogue of Claude Code's convention, naming the model per session |
| `binary-freshness.ts` | `freshness_report` tool + `/fresh`: is the installed ailang binary fresh vs HEAD? (FRESH/STALE/DIRTY/UNKNOWN, fail-closed) |
| `sprint-steward.ts` | `/sprint-start <id>`, `/sprint-complete <id> <milestone>` — mechanical constrained-modification on sprint JSONs |
| `unowned-dirty.ts` | Warns (never blocks) when a git add/stash/checkout may sweep dirty files this session didn't write — authority is `git status --porcelain` itself |
| `builtin-sprint.ts` | `/builtin-finish`: golden refresh + **stdlib freeze** + verify + doctor + inventory count |
| `ailang-lsp-lite.ts` | `ailang_check(path)` → structured {code,message,file,line,col,hint}; `builtins_search({query,module})` → filtered real inventory |

All subprocesses run under the Subprocess Contract (per-command timeouts, structured
TIMEOUT failures, 64KB output caps, no silent retries).

## Session Protocol Gate

## What it does

While **armed**, a pi session in this repo cannot execute `edit`/`write`
(absolutely) or any `bash` command outside a read-only allowlist (fail-closed).
The gate disarms only after `session_protocol_ack` succeeds:

- **Interactive TUI**: the ack requires a real human `confirm` keypress.
- **Headless (`-p`/RPC)**: the ack verifies the protocol's observable steps in
  session history — a `read` of `CLAUDE.md` and a bash `ailang messages` call.
- Ack state persists across `/resume` (reconstructed from the session branch,
  the documented State Management pattern) and **fails closed**: ambiguous
  state re-arms the gate. Read-only tools are never blocked.

## How to disarm

Call the `session_protocol_ack` tool after doing the protocol steps. In TUI
mode the human confirms; in headless mode the session history must show the
steps. Read-only tools stay available, so the protocol can always be
completed.

## Escape hatches (Fail-open register)

`--no-extensions` / `-ne` disables all project extensions including this
gate — **forbidden in this repo's workflow** (register F1; the mechanical
gate is one layer of a two-layer defense whose other layer, AGENTS.md,
survives). Full register: design doc, F1–F8.

## Compatibility notes

- Tested against pi **0.84.3**. Platform claims are verified in the design
  doc's Verification Log (V1–V12); re-verify after `pi update`.
- Unit tests: `node --experimental-strip-types --test .pi/extensions/.session-protocol-gate.test.ts`
- Coordinator-spawned sessions: inheritance not yet verified (register F5).