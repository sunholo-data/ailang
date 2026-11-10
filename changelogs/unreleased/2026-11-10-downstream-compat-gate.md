### Added — `make downstream-check`: a release's blast radius, measured (2026-11-10)

`make downstream-check CANDIDATE=<binary> [BASELINE=latest|vX.Y.Z|<binary>]` runs `ailang check`
over every corpus in `tools/downstream/corpora.yaml` (in-repo examples, docparse, ailang-parse,
email-parse, daneel, the motoko fork, ailang-packages, world) with a baseline and a candidate binary
and reports only the difference: entries newly failing or newly passing, grouped by corpus and by a
normalised error code, in `report.md`, `report.json` and `pins.md`. A breaking PR acknowledges its
buckets in `tools/downstream/expected-breaks.yaml`; the gate exits 2 only on unacknowledged or
over-max buckets in blocking corpora, reports unused and expired entries, and exits 3 whenever it
could not measure (unreachable corpus, a binary that does not start, a failed built-in canary, or a
file written where nothing may be written) — never a silent skip.

Corpora are snapshots of each repo's remote ref fetched with the operator's `gh` (`REF=default`, the
release default, or `REF=pinned`); sibling checkouts are only read (`REF=worktree` copies one and
labels every row DIRTY). Each arm gets its own binary dir, HOME, compile cache, state and temp dirs,
a read-only `AILANG_PACKAGE_ROOT`, and no inherited `AILANG_*` variable. Milestones M1–M2 and the
acceptance half of M3 of `design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md`; the CI tier,
the derived release order, the registry and the eval corpus remain.
