# Fixture (n) — a Files bullet carrying NO backticked token at all

Covers the `__UNPARSABLE_PATH_ENTRY__` sentinel arm of `derive-planner-lane.sh`, which fixture
(j) does NOT reach: (j)'s first backticked token exists but is not path-shaped, so (j) is caught
one arm later by the path-shape check. Without this fixture, neutering the sentinel is a
mutation that survives the whole matrix.

**Planner-Lane**: codex-ok

### Files to Modify/Create

- tools/launchd/derive-planner-lane.sh — path written as bare prose, no backticks
