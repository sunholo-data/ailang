# Claude Code mods for AILANG

Claude Code **mods** are plugins whose hooks run inside the Claude Code process
(`hooks/hooks.json` → `{ "modules": ["./register.tsx"] }`). They can watch and
rewrite tool calls, add text the model reads after a tool result, register slash
commands, and draw UI: a pane, a band above the prompt, the status line, toasts.
Design: [`design_docs/planned/m-claude-code-mods.md`](../../design_docs/planned/m-claude-code-mods.md).

This folder is the **source of truth** (D1). The release's `bootstrap-content`
tarball carries every mod here, and `ailang_bootstrap`'s daily sync copies them
to `plugins/<mod>/` with a marketplace entry, so users install them with
`/plugin install <mod>@ailang-marketplace`.

| Mod | Who it serves | What it does |
|-----|---------------|--------------|
| `ailang-lens` | the person | Pane of each edited module's functions, types, effect rows and errors; status line summary; `/ail-lens [file]` |
| `ailang-inbox-band` | the person | Band above the prompt with unread `ailang messages`; `/ail-inbox` pane with Read / Ack; toast on arrival. Never puts message text in model context |
| `sprint-status` | the person | Status line: current sprint and its next milestone |
| `ailang-check-on-edit` | the model | After a `.ail` Edit/Write, `ailang check` errors are added to what the model reads after the tool result |
| `unowned-dirty` | both | Warns (toast + model context) when a git sweep would take files this session did not write. Never blocks |
| `prepush-gate` | both | Runs the repo's CI gates before `git push`; denies the push when they fail |
| `ailang-brand` | the person | `λ AILANG <version>` beside the prompt hint, compiler-flavoured spinner words (Unifying…, Inferred effects for 3s), `/ailang` pane with the logo (a picture in kitty/Ghostty, `λ AILANG` text elsewhere) |

## Rules every mod follows

- **Not a security boundary.** A mod runs inside the agent's own process. Gates
  here are convenience; the boundary is per-lane IAM and `ailang run --policy`.
- **Never load-bearing.** Mods availability is switched per process by Claude
  Code. A session without the mod must still be correct.
- **Bounded.** Every `$.process.run` sets `timeoutMs`.
- **Project root.** `ailang` runs from the nearest `ailang.toml` / `.git` with a
  root-relative path, or module names fail MOD010.
- **Messages are data.** No mod puts message payloads into model context.

## Ports from pi

`unowned-dirty` and `prepush-gate` carry a verbatim copy of their pi
extension's pure functions in `hooks/core.ts`. Neither channel can import a
shared file (pi extensions are copied flat into `cmd/ailang/pi_assets/`; a
plugin cannot import outside its folder), so `scripts/check_claude_mods_drift.sh`
fails when a copied function body differs from `.pi/extensions/<name>.ts`.

## Developing

```bash
make claude-mods-check                      # validate + tsc + tests + drift, every mod
scripts/check_claude_mods.sh ailang-inbox-band   # one mod
claude --plugin-dir tools/claude-mods/ailang-lens   # try it in a session
```

For hot reload while editing, load the `plugin-authoring` skill in a Claude Code
session and work on a copy in the folder it names; each saved edit reloads at
the end of the turn. `claude plugin test` reports "rollout switch served off" on
some processes; the check script then reports the tests as NOT RUN.
