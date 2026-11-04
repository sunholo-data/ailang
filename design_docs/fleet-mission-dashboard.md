# Fleet mission — iteration 85, 2026-11-04

- Latest release: v0.57.1. Fleet work landed on dev since then: #2122 (`26b35b081`, transcript secret detector).
- LANDED this fire: each legacy mission-controller fire now scans the session files it wrote (codex, codex-missions, pi, claude) for `secrets.env` values, bounded at 120 s and non-fatal. Key names and paths go to the private controlplane inbox; the public issue gets a count only. Judged PASS, 0 blocking. Ticket `security:loop-env-inspection-prints-secrets-into-transcripts` resolved ~11 h after filing.
- FOUND: besides the rotated setup-token (0 files now), `OPENROUTER_API_KEY`, `GOOGLE_API_KEY` and `OLLAMA_API_KEY` sit verbatim in 4 local transcripts each and `ANTHROPIC_API_KEY` in 1. The driver strips only the Anthropic and OpenAI keys, so the other three reach every loop.
- NEW ASK: **D-FLEET-42**. Per-lane env strip at spawn (A, recommended), detection only (B, default), or strip all and re-add per recipe (C). Also: rotate the four keys, and permission to redact.
- PROPOSED, NOT MERGED (5 drafts, each waiting on your ruling):
  - **D-FLEET-18 A guard step**, draft #2110 (`875581f46`). Judge PASS 93.
  - **D-FLEET-36 A + D-FLEET-38 A guard step**, draft #2108. Judge PASS 92.
  - **D-FLEET-25 shared sub-item**, draft #2082 (`399cf799b`). Judge PASS 94.
  - **D-FLEET-21 A**, draft #2067: Gate 3b CI poll cap 1800 s → 3600 s. Judge PASS 90.
  - **D-FLEET-26 A**, draft #2065: codex capacity error → transient. Judge PASS 94.
- Dev: `26b35b081`. CI and Build and Release green, 18 check-runs, none red.
- Roles this fire: planner and executor `codex:gpt-6.1-sol`, evaluator sonnet (Agent tool, cross-vendor). Designer not spawned.
- NEXT: a ruling on D-FLEET-42; then the pre-push mirror guard carried from iteration 84; otherwise a ruling on any draft.
- Decisions: 41 rows, 25 OPEN (D-FLEET-16 to -23, -25 to -29, -31 to -42). New: D-FLEET-42.
- Cadence: 6h while tickets are open; 25 open. Bookkeeping thread #2039.
- Quota: codex served this fire; metered $0.
- Bar: clause 1 UNMEASURED (goal unmoved); clause 2 UNMET (one in-window resolution this fire); clause 3 MET; clause 4 prior evidence; clause 5 preserved.
- Evidence: `fleet-mission-evidence/iteration85/`.
