# Fleet mission — iteration 34, 2026-10-13

- origin/dev `3a972e34a` (#1793). PR CI: 26 checks, 0 failures.
- **MEASURED** the Phase 3a skill-resolution spike (directive "after P1, before P2"). No harness code shipped. Judged **PASS 88** by sonnet (independent, Agent tool, own worktree), which also replicated the key claims with its own method.
- **claude** discovers project skills, but `~/.claude/skills` wins every same-name collision (pin, v1, `--add-dir`). A per-fire `CLAUDE_CONFIG_DIR` keeps auth (env token), and its `skills/` wins.
- **codex** reads `.agents/skills` + `$CODEX_HOME/skills` (CODEX_HOME wins), never `.claude/skills`. A per-fire CODEX_HOME needs a credential copy.
- **pi** has no catalog discovery. `--skill <abs>` loads a skill reachable only by `/skill:<name>`.
- **OPEN for Mark: D-FLEET-16**, the Phase 3b skill source. A (recommended): retire the user-level symlinks, with the pin's `.agents` mirror under a parity gate. B: per-fire config dirs. C: no change (default).
- Next: P2 #10 `skill:gate0-ledger-provenance-S`, then the P2 tier by `slots_lost`.
- The `blocking=all` ticket `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE.
- Evidence: `design_docs/fleet-mission-evidence/iteration34/` (raw streams kept on the rig at `~/.ailang/state/fleet-i34-spike-raw/`).
- Thread: #1768. OPEN human decisions: D-FLEET-16.
- Clauses: 1 unmeasured · 2 unmet · 3 met (52 open) · 4 prior evidence · 5 preserved.
