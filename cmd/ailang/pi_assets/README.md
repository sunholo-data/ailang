# Session Protocol Gate + Dev-Harness Extensions

The eight-extension AILANG pi suite (M-DX-SESSION-GATE, M-DX-PI-HARNESS).
Tested against pi **0.84.3**.

## Distribution — who gets what, how

### Tier 0 — every repo on a machine: install from the ailang binary

Release binaries embed all eight `.ts` extensions plus this README. Install the
managed global copy into `~/.pi/agent/extensions/` with:

```bash
ailang pi install
ailang pi status
```

Re-run `ailang pi install` after upgrading the binary. It is idempotent and
version-stamped. User-modified or independently installed files are never
overwritten; the new binary's copy is written under `.ailang-suggested/` for
review. `ailang pi uninstall` removes only unchanged files recorded in
`.ailang-managed.json`.

### Tier 1 — sessions inside the ailang repo: source tree

The extensions live in this repo under `.pi/extensions/`. On any machine:

```bash
git pull                    # in the ailang repo — that's the whole install
```

First session per machine prompts once to trust the project; after that every
session arms the gate and gets the tools (`ailang_check`, `builtins_search`,
`freshness_report`, `quota_report`). Verify with `pi -p --no-session "call quota_report"`.

### Tier 2 — cloud/fleet images: installed at build time

`docker/Dockerfile.agent-pi` runs `ailang pi install` as the `ailang` runtime
user after installing pi. Fleet sessions therefore inherit the global suite
without depending on a repository checkout. Image rebuild and publication remain
human-owned release operations.

| Extension | What it does |
|---|---|
| `session-protocol-gate.ts` | Blocks edit/write + fail-closes bash until the session protocol is acked (see below); **also appends `Co-Authored-By: pi (<provider>/<model>)` to allowed git commits** — the pi-analogue of Claude Code's convention, naming the model per session |
| `binary-freshness.ts` | `freshness_report` tool + `/fresh`: is the installed ailang binary fresh vs HEAD? (FRESH/STALE/DIRTY/UNKNOWN, fail-closed) |
| `sprint-steward.ts` | `/sprint-start <id>`, `/sprint-complete <id> <milestone>` — mechanical constrained-modification on sprint JSONs |
| `unowned-dirty.ts` | Warns (never blocks) when a git add/stash/checkout may sweep dirty files this session didn't write — authority is `git status --porcelain` itself |
| `builtin-sprint.ts` | `/builtin-finish`: golden refresh + **stdlib freeze** + verify + doctor + inventory count |
| `provider-quota.ts` | `quota_report` tool + `/quota`: OpenRouter budget (CRITICAL ≥95%, WARN ≥80%), ollama status, current session lane — key never exposed |
| `ailang-lsp-lite.ts` | `ailang_check(path)` → structured {code,message,file,line,col,hint}; `builtins_search({query,module})` → filtered real inventory |
| `prepush-gate.ts` | Blocks `git push` when gofmt, lint, or the repository file-size gate fails |

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
- Fleet inheritance is provided by the `Dockerfile.agent-pi` build-time install.
