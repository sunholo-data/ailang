### Fixed — a shadow refused for want of its key grades as ungraded, not as a harness failure (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW. A plane without `OPENROUTER_API_KEY_SHADOW` refuses every shadow run before the
agent starts. The grader used to count that empty run as the shadow harness failing (`primary_better`), which
would skew the report if a shadow-enabled config reached a plane with no key. It now grades as
`ungraded(shadow_key_missing)`.
