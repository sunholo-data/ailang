Confined `pkg-docs` and `lock` previously fetched from the registry and wrote a persistent HOME cache. Registry reads now create no cache directories; network/cache write helpers refuse under `AILANG_AGENT_POLICY`. Hosts can provide an exclusive read-only `AILANG_PACKAGE_ROOT`, with no HOME or legacy lock-path fallback. Registry source hashes are verified before compilation, and mismatch or missing content fails `ailang check` immediately, applying Mark's D3/D5 rulings.

Validation: A1–A5 built-binary/local-server smoke scenarios and implementation regressions pass; lint, formatting, architecture boundaries, file sizes and child-root propagation pass. The requested focused selection and test-core were run: only three coordinator and eight brain-store SQLite tests fail because this executor has CGO disabled and no C toolchain. No compiler was installed. Full make test is reserved for CI due to RAM-backed temporary storage. See the sprint report for evidence and limitations.

Refs #1607 #1608
Closes #1607
Closes #1608
