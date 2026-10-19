### Added — executor shadowing, M6: comparison with the merged fix, and the reviewer (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M6. A graded comparison now waits for the primary's PR to settle
(`internal/coordinator/shadow_merged.go`, a GitHub read throttled to every 10 minutes, giving up after 14 days)
and records how it ended: `merged`, `closed` (the primary's work was rejected) or `unsettled`. For a merged PR it
records the head and merge commits, the changed paths, each candidate's file overlap with them (Jaccard), and
whether the merge is circular (GitHub merged exactly the primary's head, so its tests are the primary's own). When
the merged PR added or changed `*_test.ail` files, a `merged_tests` grade phase lays those test files (never
modules) over each candidate and runs them; unless circular, that layer outranks the mechanical gates. When the
layers tie or disagree and the agent sets `shadow.reviewer_model`, a `review` phase asks that off-Anthropic
reviewer (resolved as design-quorum resolves one; Anthropic models are refused at config load) to pick A or B,
in an order fixed per comparison so position bias cannot pass for a harness preference. It must name its
strongest objection and a confidence. Cost is capped before the call (`reviewer_max_cost_usd`, default $0.25)
and computed from token counts afterwards. A reviewer that is needed but unreachable, over budget or
unparseable leaves the comparison `ungraded`, never silently decided. Each comparison carries one stage field;
every phase is claimed by compare-and-set, dispatched as a mode of the existing executor job
(`AILANG_SHADOW_PHASE`, new registry var), settled as ungraded if it never reports, and never overwrites a
result that landed first. The reviewer was verified live against `gemini-4-argon` ($0.002).
