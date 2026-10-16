### Package tests validate all source modules

- `ailang test --package` now runs the full package check before tests, including non-exported modules, inter-function references, and strict fallbacks. Broken packages intentionally change from exit 0 to exit 1; no tests run on a source-check failure, even with `--allow-skips` or no test files.
- Add `--no-source-check` for callers that already checked the package. `pkg quality` uses it to avoid a duplicate pre-pass. File/directory mode and standalone package checking retain their behavior.
- Package JSON output adds an external `source_check` object (`passed`, `failed`, `errors`, `warnings`) and remains one document, including zero-test packages. Consumers rejecting unknown keys must accept this addition. The key is omitted when checking is disabled. Green zero-test packages retain exit 0 and the existing empty-suite `success:false` semantics.
- CI can use the package test command for checking and testing; compile-cache reuse and timing are measured in the sprint implementation report, without a single-compilation guarantee.
