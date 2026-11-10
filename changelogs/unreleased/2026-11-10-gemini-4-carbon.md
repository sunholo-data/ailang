### Added — Gemini 4 Carbon in the eval registry and `extended_suite`

`gemini-4-carbon` became callable on Vertex (global, `ailang-dev`) and the Developer API on
2026-11-10; on 2026-10-29 it was 404 on both. It is registered at the post-intro list price
$4/$20 per 1M (cache reads $0.20), the same as `gemini-4-argon`, whose $2/$10 intro rate was
never billed. Its thinking is adaptive, and `thinkingBudget` is rejected, so the row sets no reasoning knobs. The
output cap is 32768, the Vertex non-streaming ceiling (the API clamps larger values silently).

Placed alone at `--trials 1` across smoke, core, frontier, stretch and long-frontier: 81/81,
anchored ELO 2969 (a floor). It joins `extended_suite` next to Argon rather than replacing it.
On the 49 benchmarks both ran, Carbon uses 6.7x Argon's output+thinking tokens: $0.19 per run
uncached vs Argon's $0.135.
