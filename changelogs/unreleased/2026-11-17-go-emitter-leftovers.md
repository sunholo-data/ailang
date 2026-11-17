### Changed — Go emitter leftovers after #2389 (2026-11-17)

- **`ailang devtools-prompt`** serves the new devtools prompt v0.8.1 (and v0.8.1-compact). v0.8.0
  still taught `ailang compile --emit-go` and `DEBUG_CODEGEN=1`; v0.8.0 is frozen, so the new
  version leaves those lines out. v0.8.0-compact stays, because eval-suite's `full` condition pins it.
- **`ailang compile`**: a help flag anywhere in the arguments (`compile --emit-go --help`) now prints
  the removal message and exits 0, as `compile --help` did. Before, only a leading help flag did.
- **CLI reference**: `ailang compile` moves from "Compatibility spellings … keeps working" to a new
  "Removed commands" section, generated from the new `Command.Removed` flag.
- **Docs**: why-ailang drops its "Compiles to Go" row. effects.md and limitations.md no longer say
  Go code generation rejects terminal operations. VISION_BENCHMARKS uses `ailang check`. Example
  comments and design notes are updated: `runtime/` was deleted, not moved under `internal/`.
- **Dead code**: `lower.QualifyFuncRefs` and its helpers (its only caller was the `--emit-go-v2`
  path) and `mathx.Source` (an embed for the emitter to copy) are removed. Stale emitter comments
  are fixed.
- **Tests**: the programs from the removed Go emitter's codegen harness now run on the evaluator,
  `--bytecode` and `--strict-bytecode`, with pinned output (`TestEngineParityCorpus`,
  `tests/golden/codegen/engines/`): a 27-call std parity program, an ADT-alias program and ten
  known-failure programs. So do the audio behaviours that #2389's deleted test checked only through
  emitted Go. `nominal_same_name` is pinned as the known strict-VM BC001 (#2260).
- **Skills and design docs**: the skills no longer describe generated Go, and
  `sprint-executor/resources/codegen_patterns.md` is deleted. 19 planned design docs carry a
  one-line superseded note.
