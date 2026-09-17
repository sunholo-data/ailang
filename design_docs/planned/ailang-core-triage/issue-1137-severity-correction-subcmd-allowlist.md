# Issue #1137 severity correction — process-allowlist downgrade to improvement

- **Date**: 2026-09-15
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/implemented/v0_38_0/m-process-subcmd-allowlist.md
- **Searched**: `process-allowlist` across `design_docs/` and `docs/`; `1137` in `design_docs/`; `allowlist` in `docs/docs/guides/cli.md`
- **Estimate**: n/a (no code change recommended)

The correction's two factual premises are stale against HEAD. (1) The design doc is NOT
"Status: Planned" in `design_docs/IMPLEMENTED` — `design_docs/implemented/v0_38_0/m-process-subcmd-allowlist.md`
carries **Status: ✅ Implemented (v0.38.0, 2026-09-11)** and even names Daneel/#1137 as the
consumer on record, with the argument-free wrapper recorded as the interim workaround.
(2) The documentation points are largely fixed: `docs/docs/reference/effects.md:328-329`
documents both the per-binary allowlist and the `cmd:sub[:sub…]` subcommand narrowing
(positionally matched, fail-closed, empty-segment-is-startup-error), and the flag also
appears in `docs/docs/guides/streaming.md` and `docs/docs/guides/cli.md`. Notably the
reference already states as *intended* the very behaviour the report asks someone to
confirm: "Allowlisting a shell *script* does not grant its interpreter."

The genuinely actionable residue — downgrading #1137 from blocker to improvement, and
the maintenance-burden-vs-capability framing — is exactly what the implemented subcommand
allowlist delivers, so there is nothing new to design or fix. No code change recommended;
the GitHub issue should simply be relabelled/closed against M-PROCESS-SUBCMD.
