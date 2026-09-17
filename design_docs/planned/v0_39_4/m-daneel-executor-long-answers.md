# M-DANEEL-EXECUTOR-LONG-ANSWERS — when an executor answer is long, publish it as a Google Doc and mail the link

**Status**: Planned
**Target**: v0.39.4 (one small wrapper change in this repo; the rest is deployment config + the Daneel repo)
**Priority**: P1 — the first live `daneel+search@` ask already produced a mutilated answer; every future multi-source answer is affected until this lands
**Estimated**: ~2 days (M1 wrapper ~0.5d, M2 template ~0.5d, M3 Daneel finishAnswer ~0.5d, M4 probe ~0.5d)
**Dependencies**: [M-DANEEL-AILANG-EXECUTOR](../../implemented/v0_39_3/m-daneel-ailang-executor.md) (landed v0.39.3 — the lane, the template contract, the summary-as-answer D1); this doc is that doc's named Deferred Decision ("Answer size beyond the Summary bound… the escalation is a wrapper-committed answer artifact under a bounded path… that decision waits for the measurement") — the measurement has now arrived (V6)

**Created**: 2026-09-17 · **Author**: attended session with Mark (goal, premises, proposed shape, decisions D1–D3)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The wrapper's copy is a pure function of the workspace state: `.ailang-scratch/answer.md` exists → it is copied byte-for-byte to the artifact dir next to transcript.txt, by the same `os.WriteFile` path. No ordering, retry or state dependence. |
| A2: Replayability | +1 | The answer becomes a durable, addressed artifact (`{artifact_gcs_path}/answer.md`) instead of existing only as bytes inside a transcript; the mail names the Drive URL, which is itself replayable evidence. |
| A3: Effect Legibility | +1 | The Drive write happens inside the host-lent `publish : (string, string) -> string ! {FS, Process, Clock}` — the executor's own effect row is unchanged, and the one new effectful step on the Daneel side runs under an already-declared row. |
| A4: Explicit Authority | +1 | The lending boundary stays exactly where it was: the executor still holds no Drive credential, no mail credential, no publish call — it writes a file inside its sandbox. Publishing and mailing remain host-lent functions called by Daneel's host. |
| A5: Bounded Verification | 0 | No verification surface changes; the contract lines are template convention, checked by the Daneel-side parser and the probe, as in the parent doc. |
| A6: Safe Concurrency | 0 | One task, one answer file, one mail. No concurrency change. |
| A7: Machines First | +1 | The contract stays machine-parseable and gets MORE structured: `ANSWER_DOC` names a fixed relative path, `ANSWER_SOURCES` is already a fixed line, and the failure paths are fixed strings a host can match, not prose. |
| A8: Minimal Syntax | +1 | No language change. One bounded file-copy in the wrapper, one template-convention edit, one branch in `finishAnswer`. |
| A9: Cost Visibility | 0 | Per-question cost attribution (completion `cost_usd`, pinned model) is unchanged; the Drive publish is inside the host's existing cost envelope. |
| A10: Composability | +1 | Composes existing pieces only: the scratch-dir discipline, the GCS artifact mount, the completion payload's `artifact_gcs_path`, the writer route's artifact fetch, the host-lent `publish`, and `finishAnswer`'s existing mail path. |
| A11: Structured Failure | +1 | This is half the design: `publish` returning empty and the artifact fetch failing are BOTH mailed as explicit "the full answer could not be published" text — never a silent drop, never a summary that pretends to be the answer. |
| A12: System Boundary | +1 | The answer crosses boundaries at named, typed points only: workspace → GCS via the existing artifact mount, GCS → Daneel via the writer route's existing fetch, Daneel → Drive via the lent `publish`, Daneel → requester via the lent mail. The executor calls nothing new. |

**Net Score: +9** → **Decision: ✅ Proceed to implementation** (after D1–D3 are ruled by Mark)

### Hard Violation Check

- [x] A1 (Determinism): the copy is state-free; no implicit nondeterminism introduced
- [x] A3 (Effects): the one new effectful step (Drive publish) runs under an already-declared host effect row; nothing hidden
- [x] A4 (Authority): the executor gains zero authority; the wrapper copy moves bytes the agent already wrote, into a bucket the wrapper already writes
- [x] A7 (Machines First): fixed line formats and fixed failure strings; no human-relay prose added

---

## Problem Statement

**Current state (the parent doc's D1 chose the completion `summary` as the answer channel, knowing it was bounded; the bound has now bitten):**

1. **The first live `daneel+search@` ask arrived mutilated.** Task `task-e0f06d06` (prod, 2026-09-17) produced a good multi-source answer comparing Daneel's security model to other AI email assistants — LONGER than the template's 1000-char final-message contract. The completion summary is the **1200-byte TAIL** of the transcript (`completionSummaryMax = 1200`, prefixed `…(transcript truncated; full text at the artifact path)` — V1), so the requester received the **second half** of the answer plus the `ANSWER_SOURCES` line, and **lost the first half** (V6, read first-hand from the completion message). The full text sits in GCS at the completion payload's `artifact_gcs_path` (`tasks/task-e0f06d06`) inside `transcript.txt` — where no requester will read it.

2. **The tail is doing the right job for the wrong payload.** The 1200-byte cap exists to keep a burst of completions from slowing the message plane (V1's own rationale) — it was never a statement about how long an ANSWER may be. Using it as the answer channel means every genuinely useful multi-source research answer (the lane's whole purpose) is structurally guillotined at the point of delivery.

3. **The escape hatches that ALMOST work, don't:**
   - `.ailang-scratch/` is excluded from wrapper commits (`stageForCommit`, V3) — good for probes, fatal for answers: **it is ALSO not uploaded as an artifact.** `writeTaskArtifacts` writes exactly three files — `transcript.txt`, `metrics.json`, `session.jsonl` — and nothing from the workspace checkout (V2). A file written to `.ailang-scratch/answer.md` today **vanishes with the container**. (The measured completion even lists `.ailang-scratch/search1.ail`/`search2.ail` in `changed_files` — the evidence collector SEES scratch files, which makes the illusion of reachability worse, not better.)
   - A committed `answers/` directory would put per-task content into the Daneel repo, require Daneel's host to git-fetch a branch (a path with a live measured pushed-ref bug, parent doc V12), and make a read-only answer service depend on the commit/push machinery the parent doc's D1 already rejected for v1.
   - Raising `completionSummaryMax` taxes every completion on the plane to fix one payload type, and any fixed byte cap just moves the guillotine.

**Impact:** every requester who asks Daneel a question whose answer is worth having (multi-source, comparative, structured) gets half of it. The lane's measured first real use is its own advertisement for the fix.

## Goals

**Primary Goal:** a requester who asks a long question receives the FULL answer — published as a Google Doc via the host-lent `publish` — mailed as a short summary plus a link, with the sources line intact, and with any publish/fetch failure stated in the mail rather than swallowed.

**Success Metrics:**

1. **Round trip (M4 probe):** a dispatched question whose true answer exceeds the template bound produces (a) `.ailang-scratch/answer.md` in the workspace, (b) `answer.md` at `{artifact_gcs_path}` (asserted first-party by the probe reading the artifact path), (c) a completion whose `summary` ends with `ANSWER_DOC:` and `ANSWER_SOURCES:` lines and fits the tail whole, and (d) a mailed answer containing the short summary, a working Drive URL whose document content equals `answer.md`, and the sources.
2. **No silent drop:** a probe where `publish` returns empty (and one where the artifact file is missing) mails the short summary plus explicit could-not-publish text — both paths asserted, not assumed.
3. **Boundary unchanged:** the executor's admission policy, effect row and tool policy are untouched (same `policy_digest` class as before); the probe's NDJSON shows zero calls to Drive/mail from the executor.
4. **Short answers unaffected:** an answer that fits the bound produces NO `ANSWER_DOC` line and is mailed inline exactly as today (regression probe).
5. **No plane change:** `completionSummaryMax` stays 1200; no new message-plane field; the only AILANG-core diff is the artifact copy.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1 — where the full answer lives and how it reaches Daneel**: `.ailang-scratch/answer.md` + a wrapper copy into the artifact dir (`{artifact_gcs_path}/answer.md`), fetched by Daneel the way the writer route fetches artifacts today. Alternatives rejected as primary: committed `answers/` dir (git-fetch dependency + live pushed-ref bug + repo pollution), raising `completionSummaryMax` (plane tax; moves the guillotine), transcript.txt parsing (zero-core-change but brittle: convention-only boundaries inside tool chatter) | This is the load-bearing mechanism: everything else (template, finishAnswer, publish) hangs off where the bytes are. It is also the ONLY decision that touches this repo | **human (freeze)** | design | high |
| **D2 — the short-summary bound**: proposed ≤600 chars of summary prose, with the whole final message (summary + `ANSWER_DOC` + `ANSWER_SOURCES` lines) ≤ ~1100 bytes so the 1200-byte tail keeps every marker line whole (arithmetic in Solution Design) | Too small → the mail's summary is useless; too big → the tail capture itself cuts the `ANSWER_DOC` line and the whole mechanism silently reverts to today's behaviour | **human (freeze)** | design | med |
| **D3 — Drive destination**: same Drive folder as writer documents (reuses the writer route's convention) vs an "Answers" subfolder (separates machine-published Q&A from human-curated documents) | Affects Daneel's Drive layout and the writer route's folder assumption if shared code is reused | **human (freeze)** | design | low |
| **D4 — the wrapper copy is generic**: any agent's `.ailang-scratch/answer.md` is copied to its task's artifact dir — the marker (`ANSWER_DOC`) stays a daneel-executor template convention, the copy is repo-agnostic data plumbing like `ScratchDir` itself | Deciding generic-vs-gated now avoids per-agent special cases in wrapper code that runs in every repo | **agent** (follows the wrapper's existing repo-agnostic doctrine, `ScratchDir` precedent) | design | low |

### Design Freeze

Before implementation begins, these must be ruled by Mark:

- [ ] **D1** — the answer file location and upload path (recommendation: `.ailang-scratch/answer.md` + wrapper artifact copy — the verification log shows the scratch dir is NOT otherwise reachable, V2)
- [ ] **D2** — the short-summary length bound (recommendation: ≤600 chars prose / ≤1100 bytes whole final message)
- [ ] **D3** — same Drive folder as writer documents, or an "Answers" subfolder (recommendation: "Answers" subfolder, so machine-published answers never interleave with curated documents — but the writer-route reuse argues for same-folder; Mark rules)

## Solution Design

### Overview

Three small changes, one per side of the boundary, and none of them moves authority:

1. **The executor's template** gains a convention: when the full answer does not fit the short bound, write it as markdown to `.ailang-scratch/answer.md` and end the final message with a short summary plus `ANSWER_DOC: answer.md` and `ANSWER_SOURCES: …` lines. When it fits, no `ANSWER_DOC` line — today's inline behaviour.
2. **The execute-job wrapper** (this repo, the only core change) copies `.ailang-scratch/answer.md` from the workspace into the task's artifact directory, where it lands in GCS beside `transcript.txt` and is addressed by the completion payload's existing `artifact_gcs_path`.
3. **Daneel's `finishAnswer`** learns the `ANSWER_DOC` branch: fetch the file from the artifact path (reusing how the writer route reads artifacts today), call the host-lent `publish(title, markdown)` to get a Drive URL, and mail the short summary + `Full answer: <url>` + Sources. If `publish` returns empty, or the fetch fails, mail the short summary and say the full answer could not be published — never silently drop it.

The 1200-byte summary cap is untouched: it exists for the message plane, not for answers, and the short summary is now small by design rather than truncated by accident.

### Architecture

```
daneel-executor (Cloud Run Job, workspace = daneel clone)
  final answer too long for the final-message bound
  → writes .ailang-scratch/answer.md        (markdown, the FULL answer)
  → final message (≤ ~1100 bytes, ends):
        <short summary, ≤600 chars — D2>
        ANSWER_DOC: answer.md
        ANSWER_SOURCES: <url>, <url>, …

execute-job wrapper (this repo, M1)
  after the run: writeTaskArtifacts
  → transcript.txt, metrics.json, session.jsonl   (unchanged)
  → answer.md copied from {workDir}/.ailang-scratch/answer.md, if present  (NEW, step 4)
  → completion payload: summary = transcript tail (unchanged 1200-byte cap),
    artifact_gcs_path = tasks/{taskID}   (unchanged)

Daneel host, finishAnswer (daneel repo, M3)
  parses the summary
  ├─ no ANSWER_DOC line → mail the answer inline (today's behaviour, unchanged)
  └─ ANSWER_DOC line →
        fetch {artifact_gcs_path}/answer.md          (writer-route artifact fetch, reused)
        url = host.publish(title, markdown)           (lent: (string,string)->string ! {FS,Process,Clock})
        ├─ url ≠ "" → mail: short summary + "Full answer: <url>" + Sources
        └─ url = "" OR fetch failed →
              mail: short summary + "the full answer could not be published" + Sources
              (never silently drop; never mail the summary AS IF it were the answer)
```

**Components:**

1. **The template contract v2** (deployment config; where the file lives is verified at M2 start from the registry's `invoke.template_file` — the brief names `ailang-multivac config/templates/daneel-executor-task.md`, the parent doc placed it in the daneel repo's `templates/`; the registry entry is the source of truth). It states: the final message must END with the marker lines; the whole final message is ≤ ~1100 bytes (D2); `ANSWER_DOC: answer.md` appears ONLY when the full answer was written to the file; the file is markdown whose body IS the full answer (no preamble). Because the lane's model route discards the system role (parent doc V13), the contract lines must be in the conversation message, not only the system prompt.

2. **The wrapper copy** (`cmd/ailang/coordinator_cloud_github.go`, M1): inside `writeTaskArtifacts`, after the existing three writes, copy `filepath.Join(workDir, ScratchDir, "answer.md")` to `filepath.Join(artifactDir, "answer.md")` when it exists. `workDir` is the deterministic `/workspace/{taskID}` already used by `executeCloudTask` (V7) — it must be passed to `writeTaskArtifacts` (a one-parameter signature change; the call site is one line, V2/V7). Failure to copy is logged to stderr and non-fatal, exactly like the existing artifact writes. Written via `os.WriteFile` like `transcript.txt`, so the gcsfuse legacy-staged-write flush concern (V2's doc comment) does not apply — the agent never appends into `/artifacts`.

3. **`finishAnswer`'s ANSWER_DOC branch** (daneel repo, M3): parse the summary's trailing lines; on `ANSWER_DOC`, fetch the file from the task's artifact path **the same way the writer route reads artifacts today** (daneel-repo machinery, re-verified at M3 start — this workspace does not hold that code, V8), then call the lent `publish`. The doc title is derived from the request question (truncated to a sane subject length) — no new contract line needed; exact truncation is a deferred decision. Both failure paths produce explicit mail text.

### The tail-capture arithmetic (why D2's bound is what it is)

The summary is the LAST 1200 bytes of the transcript, prefixed by a ~52-byte truncation marker when cut (V1). The final message is the last thing in the transcript, so the tail keeps its END. For every marker line to survive, the whole final message must fit under the cap with headroom:

| Piece | Worst case |
|---|---|
| truncation prefix (when the answer is long, the cut always fires) | ~52 B |
| short summary prose (D2) | ≤ 600 B |
| `ANSWER_DOC: answer.md` line | ~24 B |
| `ANSWER_SOURCES:` + URLs (measured: 6 URLs ≈ 300 B, V6) | ~400 B |
| **Total** | **≤ ~1076 B < 1200** |

A 600-char prose bound leaves ~120 bytes of slack; if Mark prefers more prose, the slack shrinks first and the `ANSWER_DOC` line is what the tail would eventually cut — which silently reverts the whole mechanism. That asymmetry is why D2 is a freeze item and not an agent detail.

### Implementation Plan

**M1: the wrapper copy** (~4 hours, this repo)
- [ ] Pass `workDir` into `writeTaskArtifacts`; add step 4: copy `.ailang-scratch/answer.md` → `answer.md` in the artifact dir when present; stderr-log failures, non-fatal (match the existing pattern)
- [ ] Unit test: workspace with the file → copied byte-for-byte and the return prefix unchanged; without the file → no `answer.md`, no error; unreadable file → warning, non-fatal
- [ ] Grep guard in the test comment: `writeTaskArtifacts` has exactly one caller (`coordinator_cloud.go:237`, V2) so the signature change breaks nothing else — assert by grep, conflict-surface habit
- [ ] Changelog entry (Unreleased, v0.39.x)

**M2: the template contract v2** (~4 hours, deployment config / daneel repo)
- [ ] Verify where the live template file is (registry `invoke.template_file`); record the path in the implementation report — the brief and the parent doc name different checkouts
- [ ] Edit the final-message contract: the short-summary shape, the two marker lines, the ≤~1100-byte whole-message bound (D2 as ruled), the ONLY-when-it-doesn't-fit rule, and the file-is-pure-markdown rule
- [ ] Re-state the load-bearing lines in the conversation message, not only the system prompt (V13, parent doc)

**M3: Daneel `finishAnswer`** (~4 hours, daneel repo)
- [ ] Re-verify at start: `publish`'s exact signature and effect row in `ext/abi/types.ail` at HEAD (V8: brief says `(string, string) -> string ! {FS, Process, Clock}`, parent doc V14 confirmed `publish` is host-lent but not the row); and how the writer route (`finishDocument`) fetches artifacts and mails Drive links — reuse both
- [ ] Parse `ANSWER_DOC` / `ANSWER_SOURCES` from the summary tail; on `ANSWER_DOC`: fetch, `publish`, mail (summary + `Full answer: <url>` + Sources)
- [ ] Failure paths: `publish` → empty string, OR fetch error → mail summary + explicit could-not-publish text + Sources. Never a silent drop; never the summary alone pretending to be the answer
- [ ] Title derivation from the request question (deferred detail); no-answer-fits path untouched (regression)

**M4: the end-to-end probe** (~4 hours)
- [ ] Long-answer probe (Success Metrics 1): assert `answer.md` at the artifact path first-party, marker lines whole in `summary`, mail carries a URL whose fetched content equals the file, sources intact
- [ ] Failure probes (Metric 2): force `publish` → empty (stub/ revoke) and a missing artifact file; both mails carry explicit could-not-publish text
- [ ] Short-answer regression probe (Metric 4): no `ANSWER_DOC` line, inline mail, byte-identical to today's shape
- [ ] Boundary assertion (Metric 3): executor admission JSON/policy digest unchanged; no Drive/mail calls from the executor side

### Files to Modify/Create

**Modified files (this repo):**
- `cmd/ailang/coordinator_cloud_github.go` — `writeTaskArtifacts` gains the `workDir` param and the copy step (~+30 LOC)
- `cmd/ailang/coordinator_cloud_github.go` or a sibling `_test.go` — the three unit cases (~+50 LOC)
- `cmd/ailang/coordinator_cloud.go` — the one call site passes `workDir` (~+2 LOC)
- `changelogs/v0.32-current.md` — Unreleased entry (~+8 LOC)

**Modified files (not in this repo):**
- the executor task template (deployment config or daneel repo — verified at M2 start, ~+20 LOC)
- `tools/daneel_poll.ail` (daneel repo) — `finishAnswer` ANSWER_DOC branch + failure paths (~+60 LOC AILANG)

## Examples

### Example 1: the mutilated answer, before and after

**Before (measured, task-e0f06d06, V6)** — the requester's mail is the transcript tail:
```
…(transcript truncated; full text at the artifact path)
Daneel differs structurally: no capability without an authority row (limits,
refusal, rung, what leaves the machine); grants are narrowest-scope with human
consent and closed scope choices proven by Z3 contracts; ...

ANSWER_SOURCES: https://www.shortwave.com/docs/guides/security/, https://…
```
The comparison's setup — who "others" are, what the first half concluded — is gone.

**After** — the executor wrote the full comparison to `.ailang-scratch/answer.md`; the wrapper copied it to `tasks/task-…/answer.md`; the requester's mail:
```
Answer [DNL-…] — How does Daneel's security model compare to other AI
email assistants (Shortwave, Superhuman, Handler, Outloop)?

Short answer: Daneel enforces least privilege with compiled, verified,
measured guarantees — capability rows, Z3-proven scope closure, tiered
rungs — where the others rely on policy documents and audit trails.
Full answer: https://docs.google.com/document/d/…
Sources: https://www.shortwave.com/docs/guides/security/, https://…
```

### Example 2: the failure IS the design working

The Drive publish hits a transient error and `publish` returns `""`. The mail says: short summary, `the full answer could not be published`, Sources. The requester knows there WAS a full answer, knows it did not arrive, and still has the summary and sources — three facts a silently truncated tail conflates into one.

## Success Criteria

- [ ] Metrics 1–5 (Goals), each with its measurement named (probe reads, mail assertions, admission JSON)
- [ ] M1's unit cases green; `writeTaskArtifacts` return value (the GCS prefix) unchanged for every existing caller
- [ ] `completionSummaryMax` untouched (asserted by the diff itself); no new message-plane field (grep: `topics.go` unchanged)
- [ ] Executor policy/template-of-admission untouched — `policy_digest` class unchanged; probe NDJSON shows no Drive/mail calls from the executor
- [ ] Short-answer path regression probe passes — no behavioural drift when the answer fits
- [ ] Docs updated: changelog entry (this repo); template contract recorded in the daneel repo's own docs as the parent doc did
- [ ] This doc moved to implemented with the template-path and publish-signature verification records from M2/M3

## Testing Strategy

**Unit tests (this repo, M1):**
- Copy happens iff the file exists; byte-for-byte; failure is a warning, not an error; prefix unchanged
- The one-caller grep guard documented in the test

**Integration (M4, live):**
- The long-answer round trip asserted at all four points (workspace file, artifact file, summary markers, mail content)
- Both failure probes (publish-empty, fetch-miss) — the explicit-mail assertions are the point, not an afterthought

**Manual / live:**
- The Drive document's rendered content vs the markdown source (publish's own contract, spot-checked)
- A `daneel+search@` question asked for real, read as the requester reads it

## Deferred Decisions

- **Title derivation** (request question → doc title): truncation length and wording — agent may choose; only the input (the question) and the type (string) are pinned
- **Whether the could-not-publish mail includes the artifact path** for a human follow-up — agent may choose; it must not include a link the requester cannot open
- **A retry on transient publish failure** — deferred until measured; v1 mails the failure once, honestly
- **Publishing formats other than markdown** (the executor writes markdown by template rule; if a future task type wants HTML or a sheet, that is a new contract line, not an extension of `ANSWER_DOC`)
- **Genericising the marker** (any agent adopting `ANSWER_DOC` with its own consumer) — the wrapper copy is generic already (D4); the message convention spreads only when a second consumer exists

## Non-Goals

- **Changing `completionSummaryMax` (the 1200-byte cap).** It exists for the message plane (a burst of completions must not slow it, V1) — not for answers, and this design stops using it as the answer channel instead of resizing it.
- **The executor calling Drive itself.** It holds no such capability and gains none; publishing stays a host-lent function (parent doc's lending boundary, unchanged).
- **A committed `answers/` directory in the daneel repo.** Rejected as the primary path (D1): per-task content in a shared repo, a host-side git-fetch dependency on a path with a live pushed-ref bug (parent doc V12), and a read-only answer service coupled to commit machinery.
- **Parsing the full answer out of `transcript.txt`.** The zero-core-change fallback (named in D1, available if Mark rules against the wrapper copy), but convention-only boundaries inside tool chatter is exactly the brittleness this doc exists to remove.
- **Streaming/progressive publishing, doc updates after mail.** One task, one answer, one publish.
- **Fixing the V12 pushed-ref bug.** Filed separately (parent doc); this design avoids the path it affects rather than fixing it.

## Timeline

| Day | Work |
|---|---|
| 1 | D1–D3 ruled by Mark · M1 (wrapper copy + tests + changelog) |
| 2 | M2 (template path verified, contract v2) · M3 (finishAnswer branch, re-verified publish/writer-route premises) |
| 3 | M4 (long-answer probe, both failure probes, short-answer regression, boundary assertion) · docs |

**Total: ~2 days.** The estimate assumes the daneel-repo premises (V8) verify as inherited; if `publish`'s signature has moved past the brief's statement, M3 grows by the amount of the re-learn, and the implementation report records the delta.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The model writes `ANSWER_DOC` but no file (or the file with no line): contract halves drift apart | High — the mechanism's two halves must agree | The template states file and line in one breath; the probe asserts BOTH; the fetch-miss failure path makes the mismatch a loud mail, not a silent one |
| The short summary + sources exceed the tail bound and the tail cuts `ANSWER_DOC` | High — silently reverts to today's behaviour | D2's arithmetic (≤~1100 B whole message) is stated with the measurement; the probe asserts both marker lines appear in `summary`; sources ride the mail from the parsed line, so an over-long sources line cuts PROSE first, markers last |
| `publish` returns empty on transient Drive errors | Med | v1 mails the explicit failure (A11); a retry is a named deferred decision, not an unmeasured mechanism |
| The wrapper copy leaks scratch content for agents that did not mean to publish it | Med | Only a file named exactly `answer.md` at the scratch root is copied (D4); the copy is opt-in by writing that path, and the scratch dir is already the agent's private space — nothing lands in a git tree |
| The gcsfuse staged-write flush issue loses `answer.md` | Low | The copy is a single `os.WriteFile` by the wrapper (same guarantee as `transcript.txt`, V2); the agent never appends into `/artifacts` |
| The template lives in a different checkout than the brief names | Low | M2 task 1 verifies the registry's `invoke.template_file` before editing; the implementation report records the resolved path |
| Daneel's abi moves and `publish`'s row changes | Low | M3 re-verifies `ext/abi/types.ail` at HEAD first (V8); the parent doc's template-rides-the-checkout pattern keeps the drift visible in one diff |

## Related Documents

**Implemented (may inform design):**
- [m-daneel-ailang-executor.md](../../implemented/v0_39_3/m-daneel-ailang-executor.md) — the parent: the lane, the policy, the template contract, D1 (summary as answer channel), and the Deferred Decision this doc is ("Answer size beyond the Summary bound… waits for the measurement" — the measurement arrived 2026-09-17)
- [m-agent-ailang-only-execution.md](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) — the lane and gate this executor runs on; `.ailang-scratch`'s original purpose
- [m-completion-path-parity.md](../m-completion-path-parity.md) — the completion path whose `summary`/`artifact_gcs_path` fields this design rides unchanged

**Planned (check for overlap):**
- (none found — doc searches over planned/ for daneel/answer/publish topics returned no semantic matches; the related-doc search script's no-results path is broken, see session note)

**External:**
- sunholo-data/daneel — `ext/abi/types.ail` (Host record, `publish`), `tools/daneel_poll.ail` (`finishAnswer`, `finishDocument`), the writer route's artifact fetch (V8)
- deployment config (ailang-multivac) — the executor task template (`config/templates/daneel-executor-task.md`, per the brief; verified at M2 start)

## References

- `cmd/ailang/coordinator_cloud_summary.go` — `completionSummaryMax = 1200`, the tail capture and its plane rationale (V1)
- `cmd/ailang/coordinator_cloud_github.go` — `writeTaskArtifacts`: the three files it writes, the `/artifacts` volume mount, the flush comment (V2)
- `cmd/ailang/coordinator_cloud_scratch.go` — `ScratchDir`, `stageForCommit`'s exclusion (V3)
- `internal/pubsub/topics.go` — `TaskCompletion.Summary` and `ArtifactGCSPath` (V5)
- `internal/executor/pi/pi.go` — `Transcript: output`, the full conversation log that becomes transcript.txt (V4)
- `cmd/ailang/coordinator_cloud_github.go` `branchIsAutoMergeable` — the markdown floor and declared-pattern scope, why a committed answers/ path would add merge-surface concerns
- Parent doc's Verification Log V12/V13/V14 — the pushed-ref bug, the no-system-role route, the host-lent function list

## Future Work

- A publish-retry with backoff on the Daneel side, if the empty-return rate is measured to matter
- A second `ANSWER_DOC` consumer (any agent whose consumer wants its long output as a Doc) — the wrapper half already generalises (D4)
- Moving the could-not-publish text from fixed strings to a structured failure field in the mail, if a host needs to branch on it programmatically
- The transcript-parse fallback (D1 alternative) as a degraded mode if the wrapper copy is ever refused — kept named so it is designed, not improvised

---

## Verification Log

**VERIFIED HERE** = grep/read against this checkout, 2026-09-17. **INHERITED** = measured premises from the brief/attended session whose artifacts live in the Daneel checkout, the deployment config, or the prod store — the sprint re-verifies each at the named milestone and records the result in the implementation report.

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The summary is the 1200-byte TAIL of the transcript, cut on a line boundary, prefixed `…(transcript truncated; full text at the artifact path)` when cut; the cap's rationale is the message plane, not payload semantics | read `cmd/ailang/coordinator_cloud_summary.go` in full — `completionSummaryMax = 1200`, the tail slice, the line-boundary forward, the prefix constant | **VERIFIED HERE** |
| V2 | `.ailang-scratch/` is NOT uploaded as an artifact: `writeTaskArtifacts` writes exactly `transcript.txt`, `metrics.json`, `session.jsonl` (plus the session-JSONL re-write) and nothing from the workspace checkout; the artifact bucket is the `/artifacts` volume mount; `os.WriteFile` is the flush guarantee | read `cmd/ailang/coordinator_cloud_github.go:479-557` in full | **VERIFIED HERE** — this refutes the reachability of a bare `.ailang-scratch/answer.md` and is D1's reason for the wrapper copy |
| V3 | `.ailang-scratch/` is excluded from wrapper commits at any depth (`stageForCommit` pathspec), and the exclusion is a `ScratchDir` const shared by the design's copy step | read `cmd/ailang/coordinator_cloud_scratch.go` in full | **VERIFIED HERE** |
| V4 | The full answer text IS in `transcript.txt` today (the pi executor's `Transcript` is the whole conversation log) — so the fallback of parsing the transcript is real, only brittle | read `internal/executor/pi/pi.go` — `Transcript: output // the completion's summary is the transcript tail (D1)`; `transcriptBuf` accumulates the full stream | **VERIFIED HERE** |
| V5 | The completion payload carries `artifact_gcs_path` = `tasks/{taskID}`, relative to the per-environment artifact bucket | read `internal/pubsub/topics.go:58-112` (`TaskCompletion`, `ArtifactGCSPath`) | **VERIFIED HERE** |
| V6 | The first live ask was mutilated as described: task-e0f06d06's completion `summary` is the truncated-prefix + the answer's second half + `ANSWER_SOURCES` (6 URLs); `changed_files` lists `.ailang-scratch/search1.ail`, `search2.ail` (scratch is visible to evidence but committed and uploaded nowhere); `artifact_gcs_path: tasks/task-e0f06d06` | read the completion message first-hand from the prod message store (`ailang messages list --unread --json`, inbox `daneel-executor`, correlation `…e0f06d06`, 2026-09-17T05:09:33Z) | **VERIFIED HERE** (first-hand, prod) |
| V7 | `workDir` is deterministic (`/workspace/{taskID}`) and available at the `writeTaskArtifacts` call site, so the copy step needs only a one-parameter signature change with exactly one caller | read `cmd/ailang/coordinator_cloud.go:234-237` (the call) and `executeCloudTask`'s `workDir := fmt.Sprintf("/workspace/%s", taskID)` | **VERIFIED HERE** — one caller confirmed by grep within the file set |
| V8 | Daneel premises: `publish : (string, string) -> string ! {FS, Process, Clock}` in `ext/abi/types.ail` v0.4, "puts markdown on Daneel's Drive and answers the URL or empty"; `finishAnswer` (tools/daneel_poll.ail, v0.2.4/#59) mails the summary as `Answer [DNL-…]`; `finishDocument` (daneel-writer) already mails Drive links and reads artifacts — all daneel-repo code | brief + parent doc V14 (host-lent function list, attended 2026-09-16); **the Daneel checkout is not present on this authoring workspace** | **INHERITED — sprint re-verifies at M3 start** against `ext/abi/types.ail` and `tools/daneel_poll.ail` at HEAD; if `publish`'s row or return contract has moved, the failure paths are what must be re-derived |
| V9 | The executor template pins the final-message contract (<1000 chars, `ANSWER_SOURCES` line) at `ailang-multivac config/templates/daneel-executor-task.md` | brief; parent doc placed the template in the daneel repo (`templates/daneel-executor-task.md`, workspace-resolved `invoke.template_file`) | **INHERITED — sprint verifies the live path from the registry entry at M2 start**; the discrepancy between the two names is recorded, not assumed away |
| V10 | The committed-artifact path has a live measured failure mode (pushed-ref mismatch), and auto-merge is markdown-floor-gated | parent doc V12 (backlog 2026-09-15, INHERITED there); read `branchIsAutoMergeable` here (markdown floor, declared patterns) | **Code VERIFIED HERE (merge floor); bug INHERITED** — feeds D1's rejection of the committed path |
| V11 | The lane's model route discards the system role, so the template's contract lines must ride the conversation message | parent doc V13 (verified there against the agent-tool-policy guide) | **INHERITED (verified in the parent doc)** — re-stated because it governs M2's edit |

**Session note (process friction, instance 1):** `create_planned_doc.sh` dies before creating the file when the related-doc search returns no matches (`set -euo pipefail` + `grep`'s exit 1 in `merge_results`); the doc was created from the script's own template with the search run manually. If this bites a second time, it is extension-fix-sized per the friction→extension doctrine.

## Quorum

**Triggers fired: 2 of 4** — (2) **overrides shared machinery**: the host-lent `publish` contract gains a second consumer (the writer route's) whose failure semantics — empty string means "could not publish", and that emptiness must be MAILED, not treated as a link-less mail — is a reading of the contract the writer route may not share; and `writeTaskArtifacts`' artifact set (the meaning of `artifact_gcs_path`) gains a fourth file every consumer of that path sees. (4) **external systems**: Google Drive (the publish), the GCS artifact bucket, the Daneel repo, and the deployment config template. Triggers 1 (no core freeze items beyond D1–D3, which are ruled by Mark regardless) and 3 (no banked-row schema change — `TaskCompletion` is untouched) do not fire.

**Run 2026-09-17, degraded to ZERO reviewers — a recorded WAIT, not a pass.** All three reviewer lanes were absent from the authoring machine, each recorded by name: `gpt6-astra` (auth: no `OPENAI_API_KEY`), `gemini-3-1-pro` (unreachable: Vertex ADC 403 on `aiplatform.endpoints.predict`), `oc-glm-5-2` (unreachable: no Ollama device key / local daemon refused). The tool's synthesis line mechanically reads PROCEED (no present reviewer rejected), but D-56's ruling is explicit: *unavailable independent capacity is a recorded wait, never an absent-judge pass* — so this doc is treated as **quorum-pending**: the quorum must be re-run from a machine with reviewer capacity before sprint planning, exactly as the parent doc allowed ("the quorum can precede, accompany or follow the freeze"). Artifacts: `.ailang/state/mission-quorum/m-daneel-executor-long-answers-2026-09-17T05-32-56Z.json` (+ two single-reviewer attempts). The controller's in-session verdict (pass, premises V1–V7 first-party) is recorded in the artifact but is not an independent judge.

**The freeze items D1–D3 must be ruled by Mark before the sprint starts**, per the parent doc's discipline — and now the quorum re-run joins them as a pre-sprint gate.

---

**Document created**: 2026-09-17
**Last updated**: 2026-09-17
