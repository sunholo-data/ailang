### Changed — teaching prompt `v0.16.21` + `v0.16.21-compact` (active), semver compact resolution

- `ailang prompt` now serves `v0.16.21`. It fixes the six audited `v0.16.20` contradictions: library modules import `Option`/`Result` explicitly, no `@deprecated` stdlib spellings, braced `match` and no `= { a; b }`, canonical `\acc x. ...` lambdas for `foldl`, exported `std/xml` functions (not `_xml_*` builtins), canonical `ailang pkg add|install|lock|search|docs` and `sunholo/gcp_auth`. It also teaches `! {IO @limit=5}` budgets, sibling `import ./sibling (sym)`, multi-arg inline `tests [((a, b), expected)]`, the `xs |> filter(p, _)` pipe placeholder, `std/result.{mapResult,collect}` and `std/math.{minInt,maxInt,clampInt}`. The `std/tar` and `std/deflate` recipes now point at `ailang docs <module>`.
- New `v0.16.21-compact` (~3.4 KB) replaces the stale `v0.7.4-compact` as the `ailang prompt --compact` default. The frozen `v0.16.20.md` and `v0.7.4-compact.md` are untouched.
- `resolveCompactPromptVersion` now picks the highest compact version by parsed `(major, minor, patch)` instead of ASCII order, where `v0.7.4-compact` used to beat `v0.16.x-compact`.
- `docs/LIMITATIONS.md` and the canonical limitations page record the `v1.0.0-rc.1` boundaries (`#2404`, `#2410`, nominal record `Eq`, pipe `_`). `examples/runnable/v1_dx_showcase.ail` is new and `pipe_operator.ail` gains curried and `_` stages; both need the `M2`–`M4` changes to run.

### Fixed — CI skill-mirror drift and dead tracked scratch files

- `.agents/skills/{mission-loop-change,local-model-onboarding,local-ollama-eval}` re-synced from `.claude/skills` (`tools/launchd/sync-agents-skills.sh`), so `TestQuorumArtifactShellSuite` passes again.
- Removed the tracked scratch files `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak` and `examples/manifest.tmp`, and gitignored them (`#164`/`#165`). `.gitignore` re-admits `.ailang/state/evaluations/` and `.ailang/state/implementation/`.
