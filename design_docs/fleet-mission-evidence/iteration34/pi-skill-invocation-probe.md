# Iteration 34: pi `--skill` invocation probe (controller, outside any sandbox)

**Why this exists.** The spike script's pi arms (`runs/1791874311-fd860f830d667bc5d024a1cc`) all use
the discovery prompt D ("use the available skill NAME"). Every pi discovery subject came back `ABSENT`,
including `I+`, which passes `--skill <abs>/SKILL.md`. Those runs show that pi does not list the skill
in a catalog the model can see. They cannot tell "the skill was not loaded" apart from "it was loaded
but is reachable only by explicit invocation". These probes separate the two.

**Conditions.** pi 0.85.1, `--print --mode json --no-session --approve`, model `ollama/kimi-k3:cloud`.
Each call is bounded at 150 s with stdin `< /dev/null`. The marker skill sits in a scratch directory
inside the iteration worktree, made with `mkdir` and removed with `rm -rf` afterwards (`ls` confirmed
it was gone). The `SKILL.md` frontmatter has `name: zz-pi-probe-i34`. The body returns a probe token
that was rotated between rounds. The synthetic agent dir `PS` holds only symlinks to the real
`~/.pi/agent/models.json` and `auth.json`.

| # | Agent dir | `--skill` argument | Other flags | Prompt | Final answer (verbatim, truncated) |
|---|---|---|---|---|---|
| 1 | synthetic PS | `<dir>/SKILL.md` (file) | `--tools read --no-extensions --no-context-files` | "List, verbatim, the names of every skill in your available skills list… If none, say NONE." | `NONE` |
| 2 | synthetic PS | `<dir>` (directory) | same as 1 | same as 1 | `NONE` |
| 3 | synthetic PS | `<dir>` | `--tools read --no-extensions` (context files ON) | same as 1 | `NONE` |
| 4 | synthetic PS | `<dir>` | defaults (all tools, extensions) | same as 1 | `NONE` |
| 5 | real (`~/.pi/agent`) | `<dir>` | defaults | same as 1 | `NONE. … no skills list was provided to me in this session.` |
| 6 | real | `<dir>` | defaults | `/skill:zz-pi-probe-i34` | `PROBE-OK-7f3a` (the round's token) |
| 7 | synthetic PS | `<dir>` | `--no-extensions --no-context-files` | `/skill:zz-pi-probe-i34` | `PROBE-OK-91c2` (the round's token) |
| 8 | real | `<dir>/SKILL.md` (file) | defaults | `/skill:zz-pi-probe-i34` | `PROBE-OK-91c2` |
| 9 (negative) | real | **none** | defaults | `/skill:zz-pi-probe-i34` | `…skill not found…` |

**Reading.**
- In `--print` mode, pi **loads** a skill passed by `--skill` with an absolute file or directory path,
  and the `/skill:<name>` invocation reaches it (rows 6–8). Without `--skill` the same invocation fails
  (row 9, negative control).
- The skill is **not presented as a discoverable catalog entry** (rows 1–5). A pi controller told in
  prose to "use the skill X" will not find it. The prompt must invoke it as `/skill:<name>` (or name the
  file path to read).
- Only one model (kimi-k3) was measured. Whether pi shows the catalog to the model is decided by the
  harness, not the model: rows 1–5 vary only harness flags, and the model reports that no skills list
  was provided.
