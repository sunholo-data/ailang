### Added — Claude Code mods for AILANG (`tools/claude-mods/`, M-CLAUDE-CODE-MODS) (2026-10-02)

Six Claude Code mods (plugins whose hooks run inside Claude Code), shipped through the
`ailang_bootstrap` marketplace by the release's `bootstrap-content` bundle:

- `ailang-lens`: a pane showing each edited module's functions, types, effect rows and errors, plus a status line. `/ail-lens [file]`.
- `ailang-inbox-band`: unread `ailang messages` in a band above the prompt, an `/ail-inbox` pane with Ack, and a toast when one arrives. For the person only; message text never enters model context.
- `sprint-status`: the in-progress sprint and its next milestone on the status line.
- `ailang-check-on-edit`: after a `.ail` Edit/Write, `ailang check` errors are added to what the model reads after the tool result. Verified headless: under `claude -p` the model saw `3:8 cannot unify type constructors: int vs string` without running a command.
- `unowned-dirty` and `prepush-gate`: ports of the pi extensions. Their pure functions are generated from `.pi/extensions/` and guarded by `scripts/check_claude_mods_drift.sh`.

`make claude-mods-check` validates, type-checks, tests and drift-checks them. Mods are advisory and
never load-bearing: Claude Code switches them on or off per process. Design:
`design_docs/planned/m-claude-code-mods.md`.
