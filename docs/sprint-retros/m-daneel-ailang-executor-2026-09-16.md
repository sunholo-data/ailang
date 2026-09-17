# Sprint Retrospective: M-DANEEL-AILANG-EXECUTOR

## Summary
- **Sprint**: M-DANEEL-AILANG-EXECUTOR (design `design_docs/planned/v0_39_1/m-daneel-ailang-executor.md`)
- **Duration**: one attended session, 2026-09-16 ~17:30–20:20 UTC (estimated: 4 days)
- **Execution mode**: sequential, attended; M5 as one worktree sub-agent (Daneel repo)
- **Milestones**: 6/6 passed. Three repos: ailang (M1–M3), ailang-multivac (M4), daneel (M5), plane (M6)

## Milestone timing

| Milestone | Est. LOC | Actual | Status |
|---|---|---|---|
| M1 `ai_provider` binding | 300 | ~120 impl + 3 binary tests | ✅ `56f1d1c33` |
| M2 `std/web` | 720 | ~430 impl/tests + example, goldens | ✅ `6377683dc` |
| M3 release | 40 | v0.39.2 built, **not promoted**; v0.39.3 promoted | ✅ `7494064c1`, `5560feca2` |
| M4 multivac | 320 | secret + binding + agent/policy/template (4 commits) | ✅ `6a54b96` |
| M5 daneel | 480 | daneel#59 (14 files) | ✅ `b05a063` |
| M6 e2e | 110 | 1 dev + 3 prod tasks | ✅ |

## What the sprint found that the design did not
1. **pi never set `Result.Transcript`** → every completion `summary` on the plane was `""`. D1 (the
   return path) would have answered nothing. Caught by reading the consumer (`completionSummary`)
   before building the template; fixed in one field, fixture-asserted, released as v0.39.3.
2. **The wrapper clones `merge_branch`**, and daneel has no `dev` — first dev run failed at clone.
3. **Narration in the final message** rides into the summary tail; template now says answer-first.
4. **Cloud Run refuses a Job whose secret ref has no version** — secret resource and binding had to
   land as two pushes with the value populated between them.

## Friction
- Two CI-only gates tripped that `make test`/`lint` do not run: `verify-pi-assets` (embedded
  extension copy) and `fmt-check`; plus `check-git-exec` (bare git calls must use `internal/gitexec`).
- A tag briefly pointed at a commit not on `dev` (two doc PRs auto-merged between CI check and
  push) — deleted, pipelines cancelled, re-tagged; now push `dev` first, tag second, always.
- The second-backend design doc landed twice (one per message); deduped.

## Measured in prod (M6)
- Question: correct, two-source answer, 11 turns, $0.0125, 0 bash, 2/2 programs admitted.
- `Env` control: `policy_violation`, `missing_from_policy: ["Env"]`, no workaround.
- Host control: `DisallowedHost(example.org)`; only `ollama.com` is lent.
- `OLLAMA_API_KEY` absent from every summary and transcript (sentinel test + prod inspection).

## Recommendations
- Next: M-STD-WEB-SECOND-BACKEND (`design_docs/planned/v0_39_2/`) so search does not depend on Ollama.
- The `report_friction` pi tool (so executors can message the plane themselves) is still the
  cheapest lane improvement outstanding.
- Watch the answer-first convention on the next few prod asks; if narration keeps leaking, move the
  answer into a fenced `ANSWER:` block the host extracts rather than relying on the tail.
