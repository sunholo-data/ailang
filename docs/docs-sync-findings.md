# Docs-sync findings register

Internal tracking page for sprint `docs-2` (2026-08-28). This page is intentionally not
published or added to the Docusaurus sidebar. Each finding is sized as a possible future queue
item and has one primary mission clause.

## Evidence summary

| Instrument | Result |
|---|---|
| `audit_design_docs.sh` | `rc=0`; five architecture pages referenced planned material; all 159/1030 design-doc counts were reported as audit scope |
| `check_versions.sh` | `rc=0`; git and stable release `v0.34.0`, active/latest prompt `v0.16.6`; `intro.mdx` contains stale `v0.16.0` |
| `check_examples.sh` | `rc=0`; raw output 12 passed / 29 failed / 176 skipped is unreliable because it passes absolute paths |
| `derive_roadmap_versions.sh --full --check` | `rc=0`; 126 planned docs, 682 implemented docs in this instrument's scope; roadmap consistency clean |
| `generate_report.sh` | `rc=0`; report generated; repeats the version status, five future-page hits, and embedded-snippet warning |

The prescribed fresh binary was built at `bin/ailang` with version and commit ldflags. The corrected
sweep ran from the repository root and passed relative paths to that binary. It executed all 217
`examples/runnable/*.ail` files: **166 pass, 9 genuine failures, and 42 no-module/non-running**.
Observatory memory warnings were not counted as failures. The 9 failures are listed separately
below. The two design-doc population totals above are instrument-scope differences, not silently
merged into one number; the roadmap checker is the source for the overdue aggregate.

## Scored findings

| ID | Description | Clause | Severity | Evidence / reproduction | Disposition |
|---|---|---:|---|---|---|
| DOCS-2-01 | `check_examples.sh` over-reports module-path failures when given absolute paths. | 1 | HIGH | `bash .claude/skills/docs-sync/scripts/check_examples.sh` produced 12/29; relative-path execution produced 166/9/42. Absolute paths conflict with declarations such as `module examples/runnable/X`, causing false `MOD010` results. | Standalone tooling queue item. Do not edit the script in this sprint; fixing it requires widening the allowlist to `.claude/skills/docs-sync/scripts/*` or routing to V1. |
| DOCS-2-02 | Intro page names IFC Labels as `v0.16.0` while the latest prompt is `v0.16.6`. | 1 | MEDIUM | `check_versions.sh` reports `[STALE] intro.mdx references v0.16.0, latest is v0.16.6`. | Defer to a future item; `docs/docs/intro.mdx` is nested and outside this sprint's single-level `docs/*` allowlist. |
| DOCS-2-03 | 126 planned design documents are overdue relative to `v0.34.0`. | 1 | HIGH | `bash .claude/skills/docs-sync/scripts/derive_roadmap_versions.sh --full --check` reports overdue material from `v0.29.0` through `v0.34.0`, with summary `Planned design docs: 126`. | Aggregate follow-up queue item. Triage and move/relabel incrementally; do not rewrite all 126 here. |
| DOCS-2-04 | The design-doc audit and roadmap report expose different population totals and should state their scopes. | 1 | MEDIUM | `audit_design_docs.sh` reports Planned 159 / Implemented 1030, while `derive_roadmap_versions.sh --full --check` reports Planned 126 / Implemented 682. Both returned `rc=0`. | Route to a bounded diagnostic reconciliation item; retain both raw results until scope rules are documented. |
| DOCS-2-05 | `block_demo.ail` is runnable by module path but has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; exports are `singleLine`, `compute`, `multiStep`. | Classify explicitly as helper/library or add a documented entrypoint in a future examples cleanup; not a compiler regression. |
| DOCS-2-06 | `test_module_minimal.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `hello`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-07 | `simple_func_match.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `test`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-08 | `match_arm_block.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `test`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-09 | `match_in_block.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `test`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-10 | `nested_match_minimal.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `test`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-11 | `match_arm_block_io.ail` has no `main` entrypoint. | 2 | LOW | Relative sweep: `entrypoint 'main' not found`; export is `test`. | Future examples-harness classification cleanup; invoke an explicit export or mark non-runnable. |
| DOCS-2-12 | `batch_processing.ail` requires `Env`, but the generic checker retries only with `IO` and no caps. | 2 | MEDIUM | Relative sweep reports `effect 'Env' requires capability`; the file's own usage comment specifies `--caps IO,Env`. | Fix the verifier invocation or classify this example as capability-specific; no source edit made. |
| DOCS-2-13 | `cli_args_demo.ail` requires `Env`, but the generic checker retries only with `IO` and no caps. | 2 | MEDIUM | Relative sweep reports `effect 'Env' requires capability`; the source imports `std/env (getArgs)`. | Fix the verifier invocation or classify this example as capability-specific; no source edit made. |

## Clean controls and non-findings

- The five `audit_design_docs.sh` hits were inspected: `debug-tools.mdx`, `adding-operators.md`,
  `anf.md`, and `architecture/index.md` describe implemented/current architecture; `types.md`
  clearly labels structural reflection for user-defined type classes as planned. No false
  planned/current claim was scored. These nested pages remain unedited.
- Version constants are clean: stable release matches git tag and active prompt matches the latest
  prompt file. Only the nested intro reference is stale.
- Roadmap-page versus design-doc-folder consistency is clean (`derive_roadmap_versions.sh`), and
  the four reference pages checked for implemented-design-doc links are clean.
- The example sweep's 166 passes include successful files without a module declaration. The 42
  bucket contains files that could not be run under the default `main`/capability invocation and
  are not asserted to be all defects without a bounded classification pass.
- No clause-3 site-build, clause-4 missing-page, clause-5 taxonomy, clause-6 benchmark, or
  clause-7 request-handling defect was established by these diagnostics. Those remain separate
  mission work rather than unsupported findings here.

## Routing notes

The highest-priority work is the diagnostic instrument defect (`DOCS-2-01`), the overdue-doc
aggregate (`DOCS-2-03`), and the stale nested intro reference (`DOCS-2-02`). The nested-file and
skill-script restrictions are deliberate sprint boundaries; none of those fixes landed here.
