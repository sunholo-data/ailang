### Fixed — single-file `ailang check` inside a package failed MOD010; large modules were never compile-cached (2026-10-06)

- `ailang check sim/x_test.ail`, run without `--package`, compared the declared module (`stapledons/sim/x_test`) with the file path relative to the cwd (`sim/x_test`), and failed MOD010.
  - This happened from the repo root and from inside the package dir alike, for a module that `check --package sim` accepts. The suggested fixes (`--relax-modules`, renaming the module) were both wrong inside a package.
  - Now the nearest `ailang.toml` above the file decides where the declared path lives. Only an exact match against the package layout passes, so a wrongly named module still fails.
  - Reported by stapledons-godot.
- The compile-cache size limits are raised from 16 MiB per blob and 32 MiB per module to 64 MiB and 128 MiB.
  - The old limits were sized from std modules. stapledons-godot's `protocol_test` produces 21.8 MiB of type information, and motoko's `session.ail` produces more than 16 MiB. Both printed `CACHE_WRITE_FAILED … ARTIFACT_TOO_LARGE` and recompiled on every run.
  - A warm `check` of `protocol_test` now takes 0.53 s, down from 7.3 s.
  - Named tests still recompile the test module once per test. That cost is tracked in #1328; see `design_docs/planned/ailang-core-triage/inline-test-per-case-recompile-1328.md`.
