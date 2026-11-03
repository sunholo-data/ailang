# Iteration 80 — CI duration re-measure for D-FLEET-21 (2026-11-03, controller)

Durations are `updatedAt − createdAt` of completed `CI` runs (success or failure), so they include queue wait.

- Last 40 completed runs (`gh run list --workflow CI --status completed --limit 40`): n=40, min 11, p50 14, p90 18, max 30 min; 1 at ≥30 min. Raw rows: `ci-last40.tsv`.
- Full PR check-set span (first `started_at` to last `completed_at` over all check-runs on the PR head): #2066 15 min, #2065 15 min, #2063 14 min, #2053 15 min (slowest legs test-windows / Build macos-latest).
- Last 200 completed runs, by day: 10-28 n=40 max 30 (1 ≥30); 10-29 n=90 max 99 (18 ≥30); 10-30 n=12 max 37 (1); 10-31 n=6 max 30 (1); 11-02 n=47 max 37 (2); 11-03 n=5 max 15 (0). 23 of 200 ≥30 min.
- Judge's independent re-measure over the same 200: median 15.2, p90 31.1, max 99.4; 23 over 30 min, 7 over 60 min.

Reading: the row's 10-16 evidence (31–55 min) no longer describes the median, but the 30-min cap still misses about 1 run in 9 over this week, and a 60-min cap would still miss about 1 in 30.
