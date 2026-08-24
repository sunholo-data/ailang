# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` and `v1-mission-log.md`.*

**Iteration 266 · 2026-08-24 · dev green · latest release v0.33.1**

## In flight
- **`#764` M1 LANDED** — squash `d54672b85` (PR #848, non-closing `refs #764`; #764 stays OPEN).
  Extracts the stdlib-only `serveapi/protocol` package behind a transitional shim.
- Independent `sonnet` evaluator (distinct provider from the codex executor — generator≠judge):
  **PASS 95/100, zero blocking.** All 8 M1 gates green first-party; stdlib-only closure holds
  (one non-stdlib line, module root `github.com/sunholo-data/ailang`); 16/16 shims marked.
- The **four-iteration evaluator lane park (iter 262–265) is cleared.** Root cause: those runs had
  the controller fall back to codex because Anthropic was down, so `sonnet` + `fable` (both
  Anthropic) were both unavailable. This iteration's controller is `opus` — Anthropic up.

## Resume predicate / next
- **M2** — rewire `serveapi`, unexport moved machinery, delete the `// TRANSITIONAL` shim, evict
  the machinery tests — is the unblocked queue head. Route next iteration (executor→evaluator).
- Then M3 (refusal gate), M4, reply on `#764`, then surface the pre-authorized v0.34.0 ask.

## Parked on Mark (3 open decisions + 1 routing-policy signal)
- `D-30`: harness↔`ai-check` version coupling (PATH vs `os.Executable()`).
- `D-31`: split/widen designer authoring lanes.
- `D-32`: effective-KPI treatment of `inconclusive` verification.
- **Routing signal (rule 8(e)):** the evaluator lane has no non-Anthropic, non-codex,
  worktree-capable option, so an Anthropic outage wedges ALL evaluation. Fix = configure a GCP
  project for the read-only gemini/managed-agents lane, or repair the sonnet subscription probe.

## Loop health
- Routing: controller `opus` · evaluator `sonnet` (PASS) · executor n/a (M1 by codex at iter 262).
- Cost: metered **$0.00** (all quota buckets). CI: 16 checks green on the merged head.
