### Added — Gemini 4 Neon in the eval registry (opt-in)

`gemini-4-neon` is callable on Vertex (global, `ailang-dev`) and the Developer API since
2026-11-19. It is registered at $3.00/$15.00 per 1M with cached input at $0.15 and no
introductory schedule: twice Hydrogen, 0.75x Argon/Carbon's post-2026-12-01 rate. Thinking is on
by default and heavy; `thinkingBudget` and `MINIMAL` are rejected, so the row sets no reasoning
knobs. `max_output_tokens` is 65536, the Developer API's declared limit.

Placed alone across 81 benchmarks (smoke through long-frontier) at `--trials 1`: **80/81,
anchored ELO 2735**, banked. Smoke and core were clean, and every run finished `stop`. The one
miss, `contract_sorted_merge`, printed a set as `{1, 2, 3}` where the spec wants `[1,2,3]`. The
run cost $17.53 at list ($0.216/run, no cache hits recorded).

### Changed — Gemini 4 Neon joins `extended_suite`; managed-agents row added

`gemini-4-neon` is now in `extended_suite` next to Argon and Carbon (2026-11-23). A new opt-in row
`gemini-agents-neon` pins the Developer API managed agent (`antigravity-preview-09-2026`) to Neon.
The pin was accepted and echoed, and a made-up model name was rejected with a 404. Billing has not
yet confirmed that Neon served it, and the agent smoke tier has not been run.
