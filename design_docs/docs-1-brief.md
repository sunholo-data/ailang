# docs-1 sprint brief — build the inbox-routing trigger

**Not a design doc.** Per `docs-mission.md`'s Guardrails ("most items here need no design doc at
all — prefer a Gate-2 reality-check straight into a sprint"), this is a routing declaration only —
it exists so `tools/launchd/derive-planner-lane.sh` can route the planner. It carries no design
claims and needs no quorum.

**Planner-Lane**: codex-ok

(`tools/messaging/docs_inbox_router.sh` is a new file under `tools/*`, which is inside this
mission's `MISSION_PLANNER_ALLOWLIST` — widened 2026-08-28, attended, commit `29a467cac` —
so this plans on the cheap `codex:gpt-5.6-luna` lane.)

## Task

Build the inbox-routing **trigger** that queue item `docs-1` (clause 7) calls for. `send` and
`forward` are already verified working primitives — this item adds no `internal/`/`cmd/` change.
What's missing is a script that periodically polls for doc-related traffic and calls
`ailang messages forward --to docs-mission` (or the equivalent working forward invocation —
confirm the exact flag shape against clause 7's verification log in `design_docs/docs-mission.md`
before use, since a wrong flag name must fail loudly, not silently no-op).

### Scope

1. Add `tools/messaging/docs_inbox_router.sh`: a bounded, idempotent poller that:
   - Checks the canonical (cloud) message store for new items in the `public-feedback` inbox and
     any `pkg:<vendor>/<name>` inbox, plus GitHub issues opened on `sunholo-data/ailang` since a
     persisted watermark, for content that plausibly concerns documentation (docs site, examples,
     guides, published pages — use a defensible keyword/label heuristic, and say in the script
     header what it does and does not catch; do not overclaim precision).
   - For each match, calls the forward primitive to route it into the docs-mission inbox, and
     records what it forwarded (so a re-run does not re-forward the same item — an idempotency
     key, not just a time watermark, since messages can arrive out of order).
   - Runs standalone (invocable by a launchd job or by hand) and prints a one-line summary per run
     (`checked=N forwarded=M`).
   - Every external call (`ailang messages list`, `ailang messages forward`, `gh issue list`) is
     wrapped with the bounded-wait / non-vacuous-check discipline this mission's shared skill
     requires — no unbounded waits, no silent-empty-means-clean reads (pair any zero-result read
     with a known-positive control in the script's own self-test, see below).

2. Add a **self-test mode** (`docs_inbox_router.sh --selftest`) that exercises the forward path
   end-to-end against a synthetic/test message (or the real primitives in dry-run form) without
   requiring a live external message to exist, so CI or a human can verify the mechanism without
   waiting for real traffic.

### Acceptance behavior

- A message sent from outside docs-mission's own inbox (e.g. via `ailang messages send
  public-feedback "..."` or an equivalent test-visible channel) is observed, by the script,
  forwarded into the docs-mission inbox, and is readable there via the verified read command from
  clause 7's own verification log (`ailang messages list --inbox docs-mission` or equivalent —
  confirm the actual inbox name/command against that log rather than assuming it).
- Re-running the script against the same already-forwarded message does not forward it twice
  (idempotency check, demonstrated with two consecutive runs).
- The script exits non-zero and prints a clear error if it cannot reach the message store at all
  (never a silent `checked=0 forwarded=0` that looks identical to "nothing new").

### Non-goals

- No `internal/`/`cmd/` changes — this is `tools/*` only.
- No change to the dispatch path referenced in ailang#900 (36/36 failures) — this item **polls**,
  it does not fix or depend on push dispatch.
- Do not wire this into a real launchd job as part of this sprint; a script that runs correctly by
  hand or under a test harness satisfies the acceptance criteria. Wiring a recurring job is
  separate operational work, not part of this queue item's scope.

## Files

- `tools/messaging/docs_inbox_router.sh`
