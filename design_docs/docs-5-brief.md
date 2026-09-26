# docs-5 sprint brief — examples hygiene (9 genuinely-failing runnable examples)

**Not a design doc.** Per `docs-mission.md`'s Guardrails ("most items here need no design doc at
all — prefer a Gate-2 reality-check straight into a sprint"), this is a routing declaration only —
it exists so `tools/launchd/derive-planner-lane.sh` can route the planner to the mission's cheap
lane instead of failing closed to opus for a missing `Planner-Lane` field. It carries no design
claims and needs no quorum.

**Planner-Lane**: codex-ok

## Task

Source: `docs/docs-sync-findings.md` DOCS-2-05 through DOCS-2-13 (9 findings, all clause 2). The
corrected relative-path sweep (docs-2, PR #955) found these 9 files genuinely fail
`.claude/skills/docs-sync/scripts/check_examples.sh`'s runnability check. **Do NOT touch
`check_examples.sh` itself** — that instrument's own bug is `docs-6`, a separate queued item.

**Group A — no `main` entrypoint (7 files).** Each is a small teaching snippet (block
expressions, nested `match`) exported as a named function but with nothing the checker recognizes
as an entrypoint:
- `examples/runnable/block_demo.ail` (exports `compute`, `singleLine`, `multiStep`)
- `examples/runnable/test_module_minimal.ail` (exports `hello`)
- `examples/runnable/simple_func_match.ail` (exports `test`, already `! {IO}`)
- `examples/runnable/match_arm_block.ail` (exports `test`, pure)
- `examples/runnable/match_in_block.ail` (exports `test`, pure)
- `examples/runnable/nested_match_minimal.ail` (exports `test`, pure)
- `examples/runnable/match_arm_block_io.ail` (exports `test`, already `! {IO}`)

Fix: add an `export func main() -> () ! {IO}` to each that calls the existing exported function(s)
and prints the result(s) with `println`/`show` (import `std/io (println)` if not already
imported), so the file becomes genuinely runnable AND still demonstrates the same language feature
the file was written to teach. Do not delete or rename the existing exported functions — `main`
should call them, not replace them. Keep the change minimal (this is examples hygiene, not a
rewrite): a few lines per file.

**Group B — needs a capability the generic checker never grants (2 files).** Both already have a
correct `main` and correct `! {IO, Env}` effect row; the checker's generic retry only ever adds
`IO`, never `Env`, so it fails them a different way (`effect 'Env' requires capability`) than
group A:
- `examples/runnable/batch_processing.ail` — already documents `--caps IO,Env` in its header
  comment.
- `examples/runnable/cli_args_demo.ail` — imports `std/env (getArgs)` and needs `Env`, but its
  header comment never states `--caps IO,Env` the way `batch_processing.ail` does (its trailing
  "Usage examples" comment omits the flag entirely).

Fix: add/normalize a header usage comment on `cli_args_demo.ail` matching `batch_processing.ail`'s
convention — a line of the form `-- ailang run --caps IO,Env examples/runnable/cli_args_demo.ail
<name> [<name> ...]` near the top, so both files document their capability requirement the same
way. No behavior change; this is documentation only, since fixing the checker itself is out of
scope (docs-6).

## Acceptance

For every one of the 9 files: `ailang check <file>` must pass (per this mission's Guardrail —
"Any AILANG syntax that lands ... must pass `ailang check` first"). For the 7 group-A files:
`ailang run <file>` (add `--caps IO` for the two files whose effect row includes `IO`) must exit 0
and print output. Do not modify any file outside `examples/runnable/`.

## Files

- `examples/runnable/block_demo.ail`
- `examples/runnable/test_module_minimal.ail`
- `examples/runnable/simple_func_match.ail`
- `examples/runnable/match_arm_block.ail`
- `examples/runnable/match_in_block.ail`
- `examples/runnable/nested_match_minimal.ail`
- `examples/runnable/match_arm_block_io.ail`
- `examples/runnable/batch_processing.ail`
- `examples/runnable/cli_args_demo.ail`
