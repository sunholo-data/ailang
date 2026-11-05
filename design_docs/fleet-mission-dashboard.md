# Fleet mission — iteration 86, 2026-11-05

- Latest release: v0.57.1. Fleet work landed on dev since then: #2122 (`26b35b081`, transcript secret detector), #2127 (`9ea8adcf4`, its byte-window fix).
- LANDED this fire: the post-fire transcript secret scan now matches only bytes written during the fire's window. Before the controller spawns, a bounded metadata-only snapshot records paths and byte sizes; the scan then reads only what was appended. A long-lived attended session no longer re-reports a 17-day-old key on every World fire. A missing baseline falls back to a whole-file scan, visibly. Judged PASS 92, 0 blocking. Ticket `transcript-scan:mtime-window-reattributes-old-hits-to-fire` resolved ~3 h after filing.
- WATCH: the baseline lookup is O(candidates × entries) in bash. Measured: 15 s for 60 fresh files, 66 s for 300; the cap is 120 s, and a timeout logs INSTRUMENT FAILURE (never clean). A real 6 h fire touches ~48 files.
- PENDING ASK: **D-FLEET-42** (unchanged). Per-lane env strip at spawn (A, recommended), detection only (B, default), or strip all and re-add per recipe (C). Also: rotate the four exposed keys, and permission to redact.
- PROPOSED, NOT MERGED (5 drafts, each waiting on your ruling):
  - **D-FLEET-18 A guard step**, draft #2110 (`875581f46`). Judge PASS 93.
  - **D-FLEET-36 A + D-FLEET-38 A guard step**, draft #2108. Judge PASS 92.
  - **D-FLEET-25 shared sub-item**, draft #2082 (`399cf799b`). Judge PASS 94.
  - **D-FLEET-21 A**, draft #2067: Gate 3b CI poll cap 1800 s → 3600 s. Judge PASS 90.
  - **D-FLEET-26 A**, draft #2065: codex capacity error → transient. Judge PASS 94.
- Dev: `9ea8adcf4`. CI and Build and Release green, 18 check-runs, none red.
- Roles this fire: planner and executor `codex:gpt-6.1-sol`, evaluator sonnet (Agent tool, cross-vendor). Designer not spawned.
- NEXT: a ruling on D-FLEET-42; any new ticket; otherwise a ruling on any draft.
- Decisions: 41 rows, 25 OPEN (D-FLEET-16 to -23, -25 to -29, -31 to -42). None new.
- Cadence: 6h while tickets are open; 25 open. Bookkeeping thread #2039.
- Quota: codex served this fire; metered $0.
- Bar: clause 1 UNMEASURED (goal unmoved); clause 2 UNMET (one in-window resolution this fire, ~3 h); clause 3 MET; clause 4 prior evidence; clause 5 preserved.
- Evidence: `fleet-mission-evidence/iteration86/`.
