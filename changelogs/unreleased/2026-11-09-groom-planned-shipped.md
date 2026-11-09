### Changed — design docs: 165 shipped docs moved from planned/ to implemented/ (2026-11-09)

Design docs whose work has already shipped now live under `design_docs/implemented/<version>/`
instead of `design_docs/planned/`. The moves come from a read-only classification of every
planned/ doc against origin/dev; only the high-confidence SHIPPED rows were moved. `git mv`
semantics keep each file's history. The documents' own Status lines and prose are unchanged.

Moved per target version: v0_29_0 (2), v0_32_0 (1), v0_33_1 (1), v0_33_2 (5), v0_34_0 (2),
v0_35_0 (25), v0_35_1 (1), v0_35_2 (12), v0_35_3 (8), v0_36_0 (1), v0_38_7 (1), v0_40_0 (1),
v0_45_0 (4), v0_47_2 (5), v0_48_0 (5), v0_49_0 (1), v0_50_0 (3), v0_51_0 (1), v0_52_0 (3),
v0_52_1 (2), v0_52_2 (6), v0_53_0 (4), v0_53_1 (16), v0_54_0 (2), v0_54_6 (4), v0_54_7 (2),
v0_54_9 (1), v0_54_10 (2), v0_54_12 (6), v0_54_13 (1), v0_54_14 (4), v0_54_15 (2), v0_54_16 (1),
v0_54_17 (6), v0_54_18 (2), v0_54_19 (3), v0_55_0 (8), v0_56_0 (8), v0_57_0 (2), v0_57_1 (1).

Deleted 3 stale planned/ duplicates. In each case the implemented/ copy is the later revision:
`planned/m-background-worker-lifecycle.md` (kept in `implemented/v0_54_4/`),
`planned/v0_52_6/m-sprint-loc-null-sentinel-sprint-plan.md` (kept in `implemented/v0_52_6/`) and
`planned/v0_53_0/m-inline-test-expected-values-sprint-plan.md` (kept in `implemented/v0_53_0/`).

References to the old paths now point at the new ones. This covers `.github/workflows/ci.yml`, skill
resources, Go and TypeScript comments, benchmark headers, docs pages, sprint JSON and design docs.
Relative links inside the moved docs were re-based. The generated design-docs and roadmap pages
were regenerated. Evidence captures, append-only log prose and released changelogs keep the old
paths, because they record where each file was at the time.
