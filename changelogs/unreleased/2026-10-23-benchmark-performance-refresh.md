### Changed — benchmark performance page uses versioned saved evidence

Replace the repeated radar/KPI/chart stack with an evidence table and overall
pass-rate/ELO trends. The page preserves single-language and undefeated models,
marks incomplete Core samples provisional, and separates measured versions from
publication snapshots. Model detail retains outcomes, repair, tokens, cost
categories and timing boundaries, with the full published snapshot downloadable.

The additive `benchmarkEvidence` export scopes metrics by model, harness, mode,
language and task tier. It reuses existing validated results and ELO fits;
legacy snapshots expose unavailable metrics explicitly. No new evaluation runs
or provider calls are required by a documentation deployment or language release.
