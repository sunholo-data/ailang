# D4: docparse Dockerfile — bump the ailang pin and add `--max-memory cgroup` to the serve-api CMD

- **Date**: 2026-09-23
- **Class**: feature
- **Recommend**: direct-fix
- **Searched**: `max-memory`, `docparse` (rg, design_docs/); `AILANG_VERSION` in `docker/`; `grep -rl docparse design_docs/planned/ailang-core-triage/`
- **Estimate**: 2 lines in `docparse/Dockerfile` (bump `ARG AILANG_VERSION` from `v0.37.2` to ≥ `v0.39.4`, add `--max-memory cgroup` to the serve-api CMD at the doc's cited `Dockerfile:127`)

The report names one file in the **docparse repo** (not this one — no `docparse/` checkout exists on
this machine; the message was filed to `pkg:sunholo/docparse`, an inbox with no serving agent) and
the two changes are exactly the downstream execution of already-approved rulings in
`design_docs/implemented/v1_0_0/m-v1-memory-footprint.md`: F11 lists `docparse/Dockerfile:127` as a
site where `--max-memory` is never set, and the D-D freeze checkbox (line 157: "the downstream
Dockerfile lines go out as part of the D1–D3 messages") approves exactly this opt-in flag. It is
therefore not a duplicate to drop — the doc *mandates* it and it is still undone. The precondition
is met: this repo's HEAD is v0.42.0 and the `cgroup` literal ships in source
(`internal/config/compiler.go`, `EnvMemLimit` row; `cmd/ailang/memory_limit.go`). One acceptable
way to do it, no semantics or public-surface change in AILANG itself, 2 lines / 1 file — direct-fix
by rows 3–7.

Routing notes for whoever executes:

1. The edit belongs in the **docparse repo**, which is not on this machine and whose inbox
   (`pkg:sunholo/docparse`) has no registered agent — dispatch is the operator's problem, not this
   triage's. Pin bump to ≥ v0.39.4; the shipped release is v0.42.0, so pin to that.
2. `--max-memory cgroup` is GC tuning, not a hard bound (doc's M4); the per-request MEM001 budget
   failure is a separate, unimplemented design (M-MEM-BUDGET-RUNTIME) — do not imply it in the
   Dockerfile commit message.
3. After re-pinning, `ailang doctor memory` prints what the container resolves — worth one
   smoke-run in the docparse deploy.
4. The xlsx `WORKBOOK_TOO_COMPLEX` re-measurement (250,000-cell ceiling) is gated on ailang-parse
   PRs #44/#45/#46 merging and re-pinning, per `xlsx_resource_tiers.md` "still open" §1 — separate
   ask, do not block this pin bump on it.
