# Sprint Plan: docs-2 — docs-sync diagnostic sweep and drift register

## Overview

- Routing brief: `design_docs/docs-2-brief.md` (routing declaration; no design doc required).
- Mission: documentation website upkeep, first real docs-sync sweep.
- Size: S, approximately one focused day.
- Primary output: [`docs/docs-sync-findings.md`](docs/docs-sync-findings.md), an internal page with scored, clause-tagged queue rows.
- No commits. The controller will review and finalize the diff.

The executor must use the prescribed fresh binary and relative-path example methodology. The existing `check_examples.sh` absolute-path failure is a verified tooling defect and must be recorded, not fixed here.

## Day plan

### M1 — Reproduce and establish the evidence ledger

Run all five diagnostics and record return codes and material output:

```text
bash .claude/skills/docs-sync/scripts/audit_design_docs.sh
bash .claude/skills/docs-sync/scripts/check_versions.sh
bash .claude/skills/docs-sync/scripts/check_examples.sh
bash .claude/skills/docs-sync/scripts/derive_roadmap_versions.sh --full --check
bash .claude/skills/docs-sync/scripts/generate_report.sh
```

Build the prescribed scratch `bin/ailang`, then execute the runnable examples from the repository root using relative paths. Partition results into passing, genuine failures, helper/library examples without `main`, no-module cases, and environmental/runtime warnings. The controller baseline to carry forward is 166 pass / 9 genuine fail / 42 no-module or non-running among 217 runnable files; the script’s 12/29 raw verdict count is unreliable because of its absolute-path bug.

Confirm `git check-ignore -v docs/docs-sync-findings.md` is empty before banking the deliverable.

### M2 — Classify and score

Produce future-queue-sized rows. Each row must contain:

- stable finding ID;
- one-line concrete description;
- primary clause tag (`1`–`7`);
- rough severity (`HIGH`, `MEDIUM`, `LOW`, or `INFO`);
- evidence and reproduction command/output;
- disposition (in-scope fix, follow-up queue item, or control/non-finding).

Required established rows:

1. HIGH, clause-1-adjacent tooling: `check_examples.sh` over-reports module-path failures when passed absolute paths. State the exact allowlist decision needed to fix it (`.claude/skills/docs-sync/scripts/*` widening or V1 routing); do not edit the script.
2. MEDIUM, clause 1: `docs/docs/intro.mdx` references prompt v0.16.0 while the latest is v0.16.6. The fix is nested/out of scope here.
3. Aggregate clause 1: 126 planned design docs are overdue relative to shipped versions. Do not perform a 126-document rewrite; list it as a separate follow-up queue item and mention only trivially verified mislabels if found.
4. Clause 2 example rows: record the nine corrected genuine failures individually or in small actionable groups, classify `block_demo.ail` as a possible helper/library false positive, and keep the 42 no-module/non-running cases lower-priority/unclassified unless a bounded check proves more.

Inspect the five audit hits (`debug-tools.mdx`, `adding-operators.md`, `anf.md`, `architecture/index.md`, and `types.md`) before scoring. A page that accurately describes a still-future feature is a control; a shipped feature described as future is a real clause-1 row. These nested pages may be reported but must not be edited under this sprint’s single-level `docs/*` allowlist.

Record the clean roadmap consistency check as a control, not a finding. Do not invent clause-3/4/5/6/7 defects without bounded evidence; clause-5 and clause-6 work belongs to docs-4/docs-3.

### M3 — Write and gate the register

Write the top-level internal page with methodology, summary counts, scored findings, clean controls, and routing notes. Apply only a truly trivial in-scope fix if one is found; do not touch nested docs source/constants, mission controller files, or docs-sync tooling.

## Acceptance and verification

- `test -s docs/docs-sync-findings.md` passes.
- `git check-ignore -v docs/docs-sync-findings.md` produces no output.
- All five diagnostics were attempted and their rc values are recorded.
- The page is not added to the published sidebar.
- Any changed `.ail` file passes `make verify-examples`; any syntax-bearing docs change is checked with `ailang check`.
- No evals, benchmark banking, rig lock, or git commit.
- Final diff contains no forbidden-path changes.

## Risks and routing

The main risk is confusing diagnostic defects with product defects. The absolute-path `MOD010` issue is already verified as an instrument bug; the executor should preserve that fact and prominently route its repair decision to Mark/controller. The overdue-doc aggregate is intentionally a queue seed, not a mass audit. The nested `intro.mdx` and architecture-page fixes require a future item with a widened/appropriate allowlist.
