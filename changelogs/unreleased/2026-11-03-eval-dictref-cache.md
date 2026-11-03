### Fixed — Evaluator rebuilt a type-class dictionary on every overloaded operator call (2026-11-03)

Every `+`, `==`, `<` the evaluator ran through a dictionary (`DictApp`) re-evaluated its
`DictRef`, which rebuilt the whole class dictionary each time: one `fmt.Sprintf` key and one
registry lookup per method (6–10 per class), a fresh map and fresh `BuiltinFunction` wrappers.
On a Haiku solution to `cp_60164` (hidden test 05) that was 47% of evaluator CPU, with
`MakeDictionaryKey` alone at 33%. The dictionary depends only on (class, type) and the registry,
so the evaluator now builds it once and reuses it (`internal/eval/eval_dict_cache.go`). The
registry gained a write counter (`DictionaryRegistry.Generation`); a `Register` after the
dictionary was built, or `SetDictionaryRegistry`, invalidates the cache, so derived instances
that land late are still seen.

Same test, same output, user CPU 58.1s → 21.3s (2.7×). The bytecode VM never had this cost.
`ailang run` (what the eval harness grades with) runs on the evaluator.
