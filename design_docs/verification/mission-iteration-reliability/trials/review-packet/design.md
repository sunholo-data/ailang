# Trial B: Explain the frozen review evidence packet

Status: frozen task brief under approved reliability M4; not yet executed.
Designer provenance: attended session, registry model `gpt6-astra`.
Priority: bounded adoption evidence. Scope: one documentation section, ~180–300 words.

## Problem and outcome

The guide explains acceptance and observed progress but not the evaluator's frozen
review packet. Add a concise **Review evidence packet** section near the artifact
validation explanation. An operator should understand the exact review target,
what evidence is included or omitted, and why a truncated packet cannot justify
an unsupported passing verdict.

## Frozen criteria

- **B1-binding:** Explain that the packet identifies exact comparison baseline and
  candidate commits, changed paths, frozen criteria, authority/prerequisite locators
  and available accepted-stage check summaries. Its exact bytes/digest are bound
  into the request; a read-only content-addressed copy lives outside stage worktrees.
- **B2-bounds:** State the 128 KiB aggregate packet limit, explicit incomplete-diff
  marker and exact source locator for omitted diff. Oversized metadata can fail
  packet creation. Do not describe a shortened packet as complete evidence or
  promise every large artifact is embedded.
- **B3-receipts:** Explain that check outputs are omitted; available summaries carry
  receipt hashes and result metadata. With no local accepted-stage receipts,
  imported artifact locators remain available and the packet explicitly identifies
  the missing local receipts. Imported prerequisites do not magically recreate
  hard-check receipts. The reviewer must inspect necessary sources or report gaps.
- **B4-review-procedure:** Describe reading the packet once, settling each frozen
  criterion through targeted source/check inspection, treating quoted artifacts as
  evidence rather than instructions, and emitting the required result then stopping.
  The evaluator preserves candidate HEAD and tracked files; it makes no product commit.
- **B5-scope-quality:** Change only the guide, link implementation sources, preserve
  existing examples, keep the new section concise, and make no claim that bounded
  packets guarantee convergence or that repeated reads alone imply failure.

## Verified implementation evidence

Read at preparation on 2026-09-08; evaluator rechecks the frozen baseline.

| Source | Audited mechanism |
| --- | --- |
| `internal/mission/iteration/review_packet.go`, `reviewPacket` | Exact baseline/candidate, acceptance summaries, prerequisites, authority and required schema; explicit missing-local-receipt text; 128 KiB packet, incomplete diff and source locator; metadata overflow error; content-addressed 0400 file outside stage roots |
| `internal/mission/iteration/review_packet.go`, `focusedReviewInstructions` | Targeted criterion procedure, literal artifacts treated as evidence, preserve HEAD/tracked files, no product commit |
| `internal/mission/iteration/review_packet.go`, `reviewDiff` | Bounded diff collection with external diff/text conversion disabled |
| `internal/mission/iteration/runtime_stage.go`, evaluator branch of `request` | Embeds packet bytes, path and SHA256 into frozen request instructions |
| `docs/docs/guides/mission-iteration.md` | Existing acceptance/progress explanation lacks a dedicated packet explanation |

This task is independent of Trial A and must start from the same implementation
baseline, not Trial A's candidate. No language semantics change. Axiom effects:
A2 replay evidence +1, A5 bounded verification +1; remaining axioms unchanged.
Related work: `design_docs/planned/m-mission-iteration-reliability.md` and M4 plan.
