# Phase 3b mechanism choice: FOR MARK (filed as charter decision D-FLEET-16)

Phase 3a runs `1791874306-…` (claude), `1791874447-…` (codex) and `1791874311-…` (pi), plus the pi
hand probe, measured the following ([findings](findings.md)):

- **claude**: project skills are discovered from cwd. A user-level same-name copy wins over every
  project copy and over `--add-dir`. A fresh `CLAUDE_CONFIG_DIR` keeps subscription auth under the
  loops' env token, and its `skills/` wins.
- **codex**: discovers project `.agents/skills` and `$CODEX_HOME/skills` natively, never
  `.claude/skills`, and `$CODEX_HOME` wins a collision. A fresh `CODEX_HOME` cannot authenticate
  without `auth.json`.
- **pi**: no catalog discovery in `--print` mode. `--skill <abs path>` loads the skill, and it is
  reached by `/skill:<name>`.
- **World**: gets user-level skills only. `--add-dir <pin>` adds the pin's copy but loses a collision.
- **v1**: user level beats both the v1 and the pin project copies. After user level is retired, the
  cwd's project copy is what loads.

**Auth/isolation limits.** Keychain-only claude auth was not measured. Codex isolation needs a
credential copy, which the spike forbade. Only one model per CLI was measured; resolution is
harness-side.

**Mark must choose** the per-provider skill source before release-channel implementation. **No
mechanism has been selected or activated by this spike.**

| Option | claude | codex | pi | Cost / risk |
|---|---|---|---|---|
| **A (recommended)** retire the user-level mission-skill symlinks and resolve from the pin | cwd = pin's project `.claude/skills` (measured C-project); World gets `--add-dir <pin>` | pin `.agents/skills` (measured K-project), kept in parity with `.claude` by a check (the #544 divergence becomes a gate); `$CODEX_HOME/skills` must stay free of mission-skill names | `--skill <abs pin path>` per role skill plus a `/skill:` or by-path prompt | Attended sessions **outside** the repo lose `mission-control`/`sprint-*` from `~/.claude/skills`; sessions inside a checkout still get the project copy. Smallest change; no credential handling |
| **B** per-fire config dirs | `CLAUDE_CONFIG_DIR=<per-fire dir>` whose `skills/` links to the pin (measured S/S, auth held) | needs a per-fire `CODEX_HOME` holding a copy of `auth.json` (a credential copy, refused by this spike); otherwise as A | as A | Each fire must replicate settings, hooks, plugins and user memory into the config dir, or lose them. Codex isolation needs a credential-handling ruling. Touches the billing path |
| **C** no change for skills | loops keep reading the main checkout via the symlinks (now fast-forwarded per fire by D-FLEET-15's `skill-sync.sh`) | as today | as today | Phase 3b promotes driver + binary only, and goal 4 is reported per seam ("skills: unpinned, synced") |

Default if unanswered: **C**. Phase 3b proceeds for driver and binary only, and skills stay on the
D-FLEET-15 fast-forward.
