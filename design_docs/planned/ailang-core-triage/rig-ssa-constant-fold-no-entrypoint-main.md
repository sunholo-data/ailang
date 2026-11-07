# Rig: ssa_constant_fold opencode rows fail with "exports no functions, no entrypoint 'main'" (2×)

- **Date**: 2026-11-05
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/planned/ailang-core-triage/rig-ssa-constant-fold-harness-split.md
- **Searched**: `1792922397`/`1793450798` (all 2,726 unread prod messages via `AILANG_MESSAGES_PROJECT=ailang-multivac` — zero hits); `exports no functions` (`internal/runner/entry_unexported.go`, `cmd/ailang/run_unexported_entry_test.go`); `export func main` in `ailang prompt` output (~20 occurrences); `os-rolling` across `eval_results/`, `tools/`, `internal/`; prod observatory spans for 2026-10-25 and 2026-10-31 run windows (empty — local-rig eval runs export to the rig's own observatory only); `ssa_constant_fold` in `design_docs/` → the sibling triage row.
- **Estimate**: n/a (duplicate)

## Why

These two rows (`v0.54.18/..._trial3_...`, `v0.55.0/..._trial2_...`) are a subset of the 8
opencode `ssa_constant_fold` failures the sibling triage (`rig-ssa-constant-fold-harness-split.md`,
2026-11-03) already ruled on: the deciding instruments are rig-resident (`eval_results/rotation/os-rolling/`
is gitignored and banked on the rig; the rig's opencode session DB and local chains DB are not
reachable from this workspace), so the verdict — harness lane vs AILANG gap vs benchmark-spec
defect — cannot be made here. Re-verified independently on 2026-11-05: no `os-rolling` tree on
this host, no rig eval chains in the prod observatory on either run window, and no row ID in any
unread prod message. The rig's friction digest (`inbox_1793866070736_480ce2e0`) repeats the same
1/8-opencode vs 7/7-pi split with no attachments.

**New code-level refinement for the eventual rig session** — this error kind *disfavors* two of
the four candidate lanes:

1. **Not the "forgot `export`" dialect trap.** `internal/runner/entry_unexported.go`
   (`noExportsEntryError`) has a dedicated, tested message for a declared-but-unexported
   `main` ("declared in module ... but not exported; add 'export'"). The reported variant —
   "exports no functions" — only fires when the loaded `benchmark/solution` module has **no
   `main` function at all** (empty/missing solution file, module with zero `func` items, or
   main simply never written). That is an agent-production failure, consistent with the
   opencode lane collapse precedent (#551/#554), not an AILANG error-message or
   teaching-prompt gap: `ailang prompt` teaches `export func main` ~20 times, and the message
   that did fire already names the exact fix.
2. **No missing-builtin or semantics component is plausible** — the error is entry resolution,
   which happens before any builtin/effect machinery is exercised.

So when the rig session reads the transcripts, the discriminating question is narrow: **what
did the solution file contain at run time?** (a) empty/absent → opencode harness lane; (b) a
module with top-level `let`s but no `func main` → a genuine dialect-trap candidate worth a
prompt-line fix in `ailang prompt`; (c) a full program with main under a different name →
same. The sibling triage's re-dispatch instructions (rig-attended session, or rig attaches the
rows + opencode session IDs to a prod message) stand unchanged.

## Related

- `design_docs/planned/ailang-core-triage/rig-ssa-constant-fold-harness-split.md` — the ruling row for these rows.
- `design_docs/v1-mission.md` iter-128/iter-191 (#551/#554) — opencode/ollama lane-collapse precedent.
- Coordinator note: this task arrived pre-deduplicated ("Task task-f8b3a191: deduplicated" in the ailang-core inbox), confirming overlap with the sibling row.
