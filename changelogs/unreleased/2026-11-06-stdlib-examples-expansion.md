### Added — 63 canonical stdlib and core-language examples

- `examples/runnable/stdlib/`: 49 stdlib examples and 14 core-language (`lang_*`) examples. Each
  is CI-verified by `make verify-examples` against pinned stdout and starts with a `-- Task:`
  header. Stdlib function coverage by examples rose from 400 to 469 of 496 exported functions.
  The set doubles as fine-tuning data (design doc m-eval-finetuning-data-pipeline, D-FT-5).

### Fixed — verify-examples never granted the Trace capability

- `scripts/verify_examples.go` now grants `Trace` to examples that `import std/trace`. Before
  this, any example using spans failed in CI with "effect 'Trace' requires capability".

### Fixed — fine-tune extractor rejected path-named modules

- `tools/finetune/extract_lang.py` called `ailang check` with absolute paths. That failed MOD010
  on every module named after its path (e.g. `examples/runnable/x`): only 25 of ~397 passing
  examples reached pilot-1's data. The extractor now checks paths relative to the git top and
  tries duplicate copies main-checkout first.
