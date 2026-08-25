# Mission Dashboard — V1

> 30-second control context. Snapshot, not a record — history lives in the STATUS stamp + log.

**Last iteration**: 278 · 2026-08-25 · controller `opus` · metered **$0.00** of $5

## In flight
- **PR [#886](https://github.com/sunholo-data/ailang/pull/886)** — the guide said the cross-inbox
  unread index was handled; it was not. Docs-only. MERGEABLE; CI polling at Gate 3b.

## Headline finding (iteration 278)
`ailang messages list --unread` — the command the protocol leads with, and the only one spanning
inboxes nobody thought to name — was failing `FailedPrecondition` against prod, **rc=1 deterministic
2/2**, hiding **59** unread (16 substantive feedback items, 10 filed that morning). Both
`inbox_messages` indexes lead with `to_inbox`, so both served only the per-inbox *fallback*. The
guide already described this **in the past tense**, citing a declaration written **five minutes
after** the note — and declaring is not applying: no apply ran, prod had no index 1h19m later.
Created → **rc=0, 2/2**. Second defect, same channel: a binary predating `6759ea4fa` **accepts
`AILANG_MESSAGES_STORE`, reads local SQLite and exits 0**, so the protocol is vacuous and the
session concludes the inbox is quiet.

## Next picks
1. `m-fmt-cognition-roundtrip-soundness` — shipped formatter soundness defect: `std/cognition.ail`
   is valid input whose formatted output fails to re-parse. Fails closed, but real.
2. `m-firestore-index-provenance` — **new**: prod carries 10 composite indexes added reactively
   after each `FailedPrecondition` (this is instance 3); nothing in-repo records what the code needs.
3. `m-fmt-attach-boundary-class` (38 files fail comment attachment) · 4. `m-ai-modes-regression-window`.

## Parked on Mark — 6 OPEN decisions (none resolved this iteration)
`D-38` canonical-form ruling · `D-37` `routeable`→`fixed` AI edge (sole cause of RED `make ci`) ·
`D-36` 3-round evaluator budget · `D-31` designer rotation authoring-vs-review · `D-30`
harness↔`ai-check` coupling · `D-32` `inconclusive` exemption.

## Loop health
- Thread **#852**; rotation not owed. **0** allowlisted directives since the watermark.
- ⚠ **Terraform import owed** — prod index created out of band, so upstream `messages_status_only`
  is not in state; next apply hits `ALREADY_EXISTS`. Command in the record. Flagged, not buried.
- ⚠ Driver still exports `MISSION_GH_ISSUE=745` (V1's `-prev`, **closed**); namespaced = 852.
- ⚠ Running skill still drifts by `065a4f16c`; read before proceeding. Main checkout 3 ahead / 17
  behind with a concurrent agent's unique commits → reconcile obligation 1 fails, none attempted.
