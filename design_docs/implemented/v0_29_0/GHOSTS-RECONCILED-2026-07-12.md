# Ghost reconciliation — 2026-07-12 (v1.0 backlog triage)

These design docs were found marked `Status: Planned` in `planned/` but their features had
**already shipped** — confirmed by commit/code evidence during the v1-mission full backlog triage
(5 parallel reality-check agents; see `design_docs/v1-mission-log.md` entry 10). Moved here to
`implemented/v0_29_0/` to make the open-backlog count honest. Their in-file `Status:` headers may
still read "Planned" — the shipping commit below is the authoritative record.

| doc | shipped (commit) | what shipped |
|---|---|---|
| m-cloud-observatory | 9ff2fb531 | Firestore-backed exec/span-hierarchy endpoints (interface, not SQLite-asserted) |
| m-dashboard-pubsub-events | deba5f333 | dashboard pulls `ailang-events` Pub/Sub → WebSocket live progress |
| m-diagnostic-coverage | ff58a3259 / e14b0fe4a | CI footgun table + fix-carrying diagnostics (M1–M3 + fixture promotion) |
| m-effect-row-poly-params | 011dd4904 | nested function-type effect rows in module interfaces (M-IFACE-NESTED-EFFECTS) — core fix confirmed; std/stream signature rollout unverified |
| m-eval-bounded-pipeline | d41e43894 | fused bounded combinators (takeMap/takeFlatMap) + memory ceiling |
| m-file-handling-improvements | 8697c9d01 | Gemini fileData/fileUri support + serve-api POST param fix |
| m-module-error-messages | f5919ed1e | actionable MOD010 (flag + alt fixes, no nested-error noise) |
| m-record-update-local-resolution | 01fb8676a | local funcs in record-update fields resolve (#327) |
| m-wasm-typecheck-limits | 7658fd365 | WASM type-checker budget guard (clear error, no silent freeze) |
| m-eval-local-ollama | e66c0f3aa | eval-suite practical on the local Ollama rig (was loose in planned/) |

**Still OPEN pending repro** (two agents disagreed ghost-vs-open — kept in the queue, NOT moved):
`m-dx-record-cons-pattern`, `m-dx-tapp-trecord-unification`.
