### Added — verification reasons and mathematical-model evidence (2026-10-23)

Static and package verification reports now retain typed outcome reasons,
Real/Int numeric-model scope, bounded recursion depth and encoder rejection
details. Package-quality summaries distinguish module budget issues and property
test skips, and do not imply runtime contracts ran. Missing evidence from older
producers remains unavailable. Existing schemas, counters, statuses, exit codes,
publish gates and KPI definitions are unchanged; the parked skip-counter split
is not part of this change.
