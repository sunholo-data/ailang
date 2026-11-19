### Fixed — Python eval prompts showed models a literal `{{PYTHON_VERSION}}`

Both live Python prompt paths (the agent system prompt via `LoadSystemPromptForLanguage`
and standard mode via `BenchmarkSpec.PromptPartsForLanguage`) sent `prompts/python.md`
with its `{{PYTHON_VERSION}}` placeholder unsubstituted; banked transcripts show it
(e.g. 24 files under `eval_results/baselines/`). Only the agent task prompt and a dead
loader (`loadPythonPrompt`, now deleted) substituted it, and `TestPythonPromptEndToEnd`
guarded the dead loader. Substitution now lives in one helper,
`substituteRuntimePlaceholders`, called by all three live paths, and the test drives the
live system and standard-mode prompts.

### Changed — eval Python pinned to 3.14 (was 3.12)

`PinnedPythonVersion` is now 3.14, the newest minor that `uv` resolves to a stable build
on the rig and in CI. 3.15 is released upstream, but a `uv` that predates its final release
silently resolves `--python 3.15` to 3.15.0b1, so 3.15 waits on a `uv` upgrade. CI
installs 3.14. Together with the fix above this changes the Python eval prompt and runtime:
the next Python baseline is not comparable with earlier ones.
