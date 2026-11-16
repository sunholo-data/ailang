### Changed — docs: secrets returned across user modules are trace-redacted (2026-11-16)

`std/secret` and the debugging guide said a labelled value returned by a function in another
user module still reached execution traces. That stopped being true when #2185
(M-IFC-CROSS-MODULE M3/M4) routed imported IFC summaries through the same walk that stamps
trace-redaction sites (`types.CheckModuleIFCWithImportsAndTraceSites`). The caveat is removed,
and a two-module canary pins it: `internal/runner/trace_label_redaction_xmod_test.go`
(0 copies at `standard` and `deep`; 1 and 5 with the import wiring removed).

### Changed — triage mission: Ollama controller rung (2026-11-16)

The triage controller fallback chain gains `pi:ollama/glm-5.3:cloud` after `codex:gpt-6-luna`
(Mark, attended). With Sonnet and both codex buckets over ration it had no rung left and
refused every fire from 11-13 to 11-16, while fleet/world/stapledon kept running on that rung.
