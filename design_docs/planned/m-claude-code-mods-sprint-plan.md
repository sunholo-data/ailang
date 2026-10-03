# Sprint Plan: M-CLAUDE-CODE-MODS (Phases 0–2)

**Design doc**: [m-claude-code-mods.md](m-claude-code-mods.md)
**Sprint ID**: M-CLAUDE-CODE-MODS
**Duration**: ~1.5 sessions (one working day)
**Risk**: Medium — early-access external API; `claude plugin test` availability is server-gated (V11)
**Approval**: Mark, 2026-10-02 ("great please sprint plan and execute"); D1–D6 taken as the design doc's recommendations.

## Scope

In: Phase 0 (headless spike), Phase 1 (interactive), Phase 2 (local agent).
Out: Phase 3 (dev-agent A/B — needs a measurement window) and Phase 4 (telemetry
consolidation — needs a one-day parallel run). Both stay in the design doc, gated on M0.

## Findings from planning (change the design doc in M7)

1. **D3b shared cores cannot be imported by either channel.** `make pi-assets` copies
   `.pi/extensions/*.ts` flat into `cmd/ailang/pi_assets/` (Makefile:367), and a Claude
   mod is installed as its own plugin folder (no imports outside it). Resolution (D3 was
   agent-resolvable): each mod carries a verbatim copy of the pi file's **pure exported
   functions** in `hooks/core.ts`, headed with its source path, and
   `scripts/check_claude_mods_drift.sh` fails when a copied function body differs from
   the pi source. Same guarantee (one behaviour), no import graph.
2. **The model-visible channel exists.** A `tool.call` result's `context` field is "what
   the model reads after the tool's result and the user never sees" — check-on-edit and
   unowned-dirty use it; no rewriting of the tool's own output.
3. **D1 sync path**: ailang `make bootstrap-content` (make/build.mk:115) builds the
   release tarball that `ailang_bootstrap/.github/workflows/sync-ailang.yml` unpacks
   daily. Mods ride the same tarball → `plugins/<mod>/`. Takes effect at the next ailang
   release; until then PR #11 carries the lens.
4. `/ail-lens lens_demo.ail` (relative) resolved against the session cwd, not the file's
   location, and silently analysed nothing. M1 resolves relative paths against the
   session cwd explicitly and reports a missing file.
5. `fmt --write` on edit is already done by `format_ail.sh` in this repo; doing it again
   in a mod rewrites the file under Claude's feet. check-on-edit does **check only**; fmt
   is a `userConfig` option, default off.

## Registry reuse (step 0)

`ailang pkg search` for "claude code", "inbox messages", "git dirty", "prepush" → no
packages. The deliverables are TypeScript Claude Code plugins, not AILANG packages.

| Milestone | Package | Action | Reason |
|---|---|---|---|
| M0–M7 | — | none | No registry package covers Claude Code plugins; searches above returned nothing |

## Milestones

### ✅ M0 — Headless spike (closes V10) · ~40 LOC (scratch, not committed)
- Probe mod: `session.start` writes a marker via `$.fs`, `tool.call` appends the tool name, `$.process.run(['ailang','--version'])` result recorded.
- `claude -p --plugin-dir <probe> --max-turns 2 "run: echo hi"`; read the marker and stderr.
- **Acceptance**: V10 row updated Confirmed/Refuted with the transcript excerpt; if Refuted, Phases 3–4 marked dropped in the design doc.

### ✅ M1 — Lens in tree · ~90 LOC
- `tools/claude-mods/ailang-lens/` (from bootstrap PR #11 + mtime refresh), `tools/claude-mods/README.md` (layout, dev loop, how each mod ships).
- Relative `/ail-lens` paths resolved against the session cwd; missing file → `{ text: 'ail-lens: <path> not found' }`.
- **Acceptance**: validate + tsc clean; test for the missing-file reply.

### ✅ M2 — ailang-inbox (shipped as `ailang-inbox-band`, `/ail-inbox`; see V17) · ~250 LOC + tests
- `userConfig`: `inboxes` (default `user`), `pollSeconds` (default 60).
- `$.clock.every` poll of `ailang messages list --unread --json --inbox <each>` (timeout 10 s); AbovePrompt band with count + newest title (nothing when zero); `/inbox` pane with `Read` and `Ack` Buttons (`messages read`, `messages ack`); toast once per new id.
- Never touches model context (C3).
- **Acceptance**: tests — band text for 2 unread, empty band at 0, Ack runs `messages ack <id>` and removes the row, toast fires once per id.

### ✅ M3 — sprint-status · ~90 LOC
- Status line: the newest `.ailang/state/sprints/sprint_*.json` with an unfinished milestone → `sprint M-X 3/7 · M4_NAME`. Refresh on `session.start` and `turn.complete`.
- (Narrowed from "fleet-status": quota needs provider keys in the mod; deferred per the doc's Deferred Decisions.)
- **Acceptance**: test with a fixture sprint JSON.

### ✅ M4 — ailang-check-on-edit · ~150 LOC + tests
- After a successful Edit/Write of `.ail`: `ailang check --json --quiet` from the project root (V5); on failure append `ailang check: ✗ <line:col> <cause>` (max 5) to the result's `context`; on pass, nothing.
- `userConfig.formatOnEdit` (default false) runs `ailang fmt --write` first.
- **Acceptance**: tests — failing edit gets context lines, passing edit gets none, non-.ail untouched.

### ✅ M5 — unowned-dirty · ~110 LOC + tests
- `hooks/core.ts` = verbatim `parsePorcelain`, `unownedDirty`, `isSweepingGitOp` from `.pi/extensions/unowned-dirty.ts`.
- `$.state` set of files written via Edit/Write; on a sweeping Bash git op: `git status --porcelain` (10 s), warning as a toast **and** in the result `context` (headless value). Never blocks.
- **Acceptance**: tests — warning names unowned files, silent when all dirty files are own.

### ✅ M6 — prepush-gate · ~170 LOC + tests
- `hooks/core.ts` = verbatim `goRoots`, `makeTargetDefined`, `skipRequested`, `isPushCommand`.
- Before `git push` / `gh pr create|merge`: same chain as pi (gofmt on tracked roots, `make lint`, `make check-file-sizes` when defined); failure → `{ deny }` with the tail. Escape hatch `AILANG_SKIP_PREPUSH=1` read via `$.env` (launch environment, as pi).
- **Acceptance**: tests — unformatted file denies, no Go roots passes, undefined make target skipped, hatch skips.

### ✅ M7 — Distribution + doc · ~80 LOC
- `scripts/check_claude_mods_drift.sh` + `make claude-mods-check` (drift + `claude plugin validate` per mod when the CLI is present).
- `make bootstrap-content` copies `tools/claude-mods/*` into the tarball; bootstrap `sync-ailang.yml` copies them to `plugins/` and adds marketplace entries (bootstrap PR).
- Design doc: V10 result, D3 resolution, findings 4–5, Phase 3–4 status.
- **Acceptance**: drift script fails on a hand-edited core and passes clean; `make bootstrap-content` tarball lists the mods.

## Verification per milestone

Every mod: `claude plugin validate`, `tsc` against the engine types, `claude plugin test`
**when the rollout switch serves it on** (V11) — when it is off the run is recorded as
"not run: switch off", never as a pass. Live check in the hot-reload session for M1–M5.

## Totals

~980 LOC across 8 milestones; ~1.5 sessions at this session's observed pace (lens: ~260
LOC + tests in ~1 h).

## Outcome (2026-10-02)

All 8 milestones pass. `make claude-mods-check`: drift ok for both pi cores; every mod
validates and type-checks; 23 tests pass (lens 4, inbox-band 5, check-on-edit 4,
unowned-dirty 3, prepush-gate 6, sprint-status 1). Headless confirmed twice under
`claude -p` (probe mod; check-on-edit handing the model a real type error).

Deviations from plan: the inbox mod is `ailang-inbox-band` with `/ail-inbox` (name taken in
the marketplace); `fleet-status` narrowed to `sprint-status`; check-on-edit formats only on
opt-in. Found and not fixed here: pi `unowned-dirty` flags the session's own files (V18).
Still open: Phase 3 (dev-agent A/B, container check) and Phase 4 (telemetry consolidation).
