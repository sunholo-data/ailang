# Session Protocol Gate + Dev-Harness Extensions

The eleven-extension AILANG pi suite (M-DX-SESSION-GATE, M-DX-PI-HARNESS,
M-DX-QUALITY-MONITOR, M-DX-MICRORAG-CONTEXT). Tested against pi **0.84.3**
(0.84.4 verified for quality-monitor and microrag-context).

## Distribution — who gets what, how

### Tier 0 — every repo on a machine: install from the ailang binary

Release binaries embed all eleven `.ts` extensions plus this README. Install the
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

**⚠ Headless sessions never see that prompt and are silently extension-less until
trust is saved** (measured 2026-08-31 on pi 0.84.4; pi ≤0.73 had no gate at all).
`-p` / `--mode json` / `--mode rpc` show no trust prompt — with no saved decision they
fall back to `defaultProjectTrust: "ask"`, which *ignores* project resources and reports
no error anywhere. The fix is one saved decision in `~/.pi/agent/trust.json` (flat map,
canonical path → `true`, parent directories inherit): trust the PARENT that contains all
checkouts and worktrees, e.g. on the rig `/Users/voightkampff/dev/sunholo-data` and
`/Users/voightkampff/.ailang-driver-pin`. Fresh mission worktrees are new paths every
fire, so per-project trust does not scale — parent trust is the mechanism. And when
verifying, require `tool_execution_start`/`end` events in the JSON stream: a substring
match on the tool name is satisfied by the model merely *echoing* it, and `pi list`
lists settings-installed packages only, not repo-auto extensions.

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
| `ail-fmt-autolint.ts` | After a successful write/edit of a `.ail` file, runs `ailang fmt --write` so saved AILANG is canonically formatted (motoko-measured fmt arm) |
| `quality-monitor.ts` | Bounded-excerpt rewrite of >16KB tool results (head+tail + narrowing directive); blocks the 3rd identical consecutive tool call with a directive; detects empty/zero-content turns and steers once (capped); opt-in thinking-budget fallback (`PI_QUALITY_THINKING_FALLBACK=1`). Kill switch `PI_QUALITY_MONITOR=0` (M-DX-QUALITY-MONITOR) |
| `microrag-context.ts` | μRAG retrieval frontend for pi: prompt-intent injection (`before_agent_start`, trailing message only — never a system-prompt edit), error-triggered injection from the last `ailang_check` failure (the lane no other frontend has), and the `microrag_search` tool. Engine untouched — rides `ailang micro-rag user-prompt`. REQUIRES explicit `AILANG_MICRORAG_ENABLED=1` (unset/0 = fully inert, eval-arm parity); knobs `PI_MICRORAG_INJECT=0` (no auto-injection), `PI_MICRORAG_TOOL=0` (no tool), kill switch `PI_MICRORAG_CONTEXT=0` (M-DX-MICRORAG-CONTEXT) |

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
- microrag-context unit tests: `node --experimental-strip-types --test .pi/extensions/.microrag-context.test.ts`
- Fleet inheritance is provided by the `Dockerfile.agent-pi` build-time install.
