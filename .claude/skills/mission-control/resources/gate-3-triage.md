## Gate 3 — TRIAGE (`MISSION_NAME=triage` only; replaces gate-3-route.md for this mission)

The triage mission ([charter](../../../../design_docs/triage-mission.md), design
[M-TRIAGE-MISSION](../../../../design_docs/planned/m-triage-mission.md)) runs **no inner loop**.
It has no designer, planner or code executor. Each iteration verifies the item list Gate 2
built (cards ↔ PRs ↔ issues), acts within the **current level**, and writes the shadow ledger.
Gate 3b does not apply: this gate lands nothing on `dev`.

**CURRENT LEVEL: 1 (Tidy, with attestations).** Read the charter's `CURRENT LEVEL` line first.
If it says anything other than 1, follow the charter and record the difference. The level changes
only by a recorded ruling, never by the loop.

### Never, at any level this file covers

- **Never approve, reject or merge.** No `ailang coordinator approve|reject`, no `gh pr merge`,
  no `gh pr close` on a coordinator PR. Approving dispatches the next chain stage and then merges.
  Hand-merging makes the landed-card sweep approve with dispatch, and remote reject is final
  (design doc "What approving and merging actually do", H1–H7). Approval is the coordinator's job
  (Part A, `design_docs/planned/m-coordinator-auto-approval-policy.md`). This loop attests.
- **Labels: only `priority:*`, `area:*`, `needs-decision`.** Never `bug`, `feature` or `from:*`.
  They are the coordinator's `watch_labels`, and adding one re-imports an old issue as a new task
  (charter clause 4).
- **Never act on** Dependabot majors or red Dependabot PRs, other missions' PRs (`fleet/*`, …),
  or prod. Report them in the digest.
- **Never fix anything.** Defects go to `design-doc-creator`, harness defects to
  `ailang mission ticket file`, language gaps to the V1 backlog. Red `dev` follows the same rule,
  below.

### 1. Verify (read-only verifiers, executor role)

Resolve the lane with `tools/launchd/resolve-role-spawn.sh executor` and follow its output
verbatim (gate-3-route.md "Spawn pattern"). The default chain is `codex:gpt-6-luna` →
`pi:ollama/glm-5.3-flash:cloud` → `pi:openrouter/z-ai/glm-5.3-flash`, so the cross-provider recipe
applies, not the Agent tool. Give each verifier **about 20 items**, and run batches in parallel
where the lane allows.

Each verifier gets [`github-issue-triage/resources/verify_agent_prompt.md`](../../github-issue-triage/resources/verify_agent_prompt.md),
filled in with `<DEV_SHA>` = `git rev-parse origin/dev` after a fetch. Append these instructions for
PRs and cards:

- For a PR: read `gh pr view N --json headRefOid,isDraft,mergeable,mergeStateStatus,statusCheckRollup,files,body,closingIssuesReferences`
  and the diff. Ask whether it does what its issue or card asked, whether its checks are green on
  the head, and whether `dev` already contains or supersedes it.
- A **draft** PR (`isDraft: true`) is never `VERIFIED`: its author has not declared it ready, and
  `gh pr merge` refuses it. Report it as `NEEDS-DECISION` ("ready to leave draft?"). Measured
  2026-10-16: draft PR #1754 was attested ready and the merge failed.
- For a card: read the joined PR. A card with `diff_available=false`, or with no PR at all, is
  never `VERIFIED` (H4). Report it as `NEEDS-DECISION` or as an orphan.
- Return the PR's `headRefOid` for every PR verdict. An attestation is bound to that SHA.

Verifiers are **read-only**: no comments, labels, closes or pushes. Every write below is the
controller's, after the judge.

### 2. Verdicts

Exactly one per item, with evidence (SHAs, paths, check names), read against `origin/dev`:

| Verdict | Means | Level 1 action (after the judge concurs, where marked ⚖) |
|---|---|---|
| `VERIFIED` | The PR does what its issue/card asked, checks green on the head, mergeable, not superseded | ⚖ Post an **attestation** (§4). List it in the digest as "ready for Mark to approve". Never approve it. |
| `NEEDS-REBASE` | Correct, but BEHIND or conflicting with `dev` | Evidence comment on the PR. Digest. (No rebase: this loop pushes nothing to agent branches.) |
| `NEEDS-FIX` | Wrong, incomplete or failing checks | Evidence comment on the PR naming what fails. Digest. |
| `SUPERSEDED` | `dev` already contains it or a newer PR replaces it | Evidence comment citing the superseding commit/PR. Digest as a decision for Mark: closing a coordinator PR **rejects** its card, finally. |
| `CLOSE-FIXED` | The issue is fixed on `origin/dev` | ⚖ Close under T1–T4 with the fix SHA |
| `CLOSE-OBSOLETE` | The premise is moot, or the code or feature is gone | ⚖ Close under T1–T4 with the evidence |
| `CLOSE-DUPLICATE` | Same defect as #X | ⚖ Close in favour of the sharper, current one, under T1–T4 |
| `KEEP` | Real and open | Label `priority:*` / `area:*` if missing. Note a planned doc if one covers it. |
| `NEEDS-DECISION` | Needs Mark's ruling | Label `needs-decision`. One line stating the decision, in the digest's decisions section. |

**T1–T4** (charter, verbatim in force):

- **T1** The judge (a different model from the verifier; same vendor allowed, D-TRIAGE-10) concurs, and the close comment cites the
  fix commit or reason and says "verified against origin/dev <sha>". The same applies to every
  attestation.
- **T2** Never close an issue that still has a pending card or an open `coordinator/task-*` PR. The
  coordinator imports only OPEN issues (`internal/messaging/github.go:397`), and closing one cancels
  nothing, so its chain would keep running against a closed issue. Those go in the digest instead.
- **T3** Never close the current bookkeeping thread; at most 10 closes per iteration.
- **T4** `prs --landed --apply` ends a landed card's chain WITHOUT starting the next stage. Every card
  resolved that way is listed in the digest with its handoff targets, so a chain worth continuing
  is visible to Mark (who can `--fire-handoffs` attended).

Landed cards: `ailang coordinator prs --landed --remote gcp` (dry run) first; `--apply` only for
pre-cutoff landed cards; **never `--fire-handoffs`**. Under T4, every card resolved this way goes
in the digest with its `handoff_targets`.

To close, post the evidence first with `gh issue comment N --body-file <file>`, then run
`gh issue close N --reason completed|"not planned"`. `gh issue close` has no file form, and a
backtick body passed inline under zsh loses its evidence silently (gate-5-retro 4.2(c)).

### 3. Judge (different model, evaluator role) — REQUIRED

Resolve with `tools/launchd/resolve-role-spawn.sh evaluator`. The default chain is `sonnet` →
`pi:ollama/minimax-m3:cloud` → `pi:openrouter/minimax/minimax-m3`. **The judge must be a different
model from the verifier that produced the verdict; sharing a vendor is allowed (D-TRIAGE-10, Mark
2026-10-15), e.g. a haiku verifier judged by sonnet.** The resolver already reroutes a same-model
judge. If no distinct judge is reachable, take no ⚖ action this iteration and put the items in the digest. A verdict
without a judge is a report, never an action.

The judge reviews **every verdict that triggers an action** (all ⚖ rows) and **every `VERIFIED`**.
It gets the item, the verifier's verdict and evidence, and `<DEV_SHA>`. It returns
`CONCUR | DISSENT: <one line>` per item. A dissent downgrades the item to `NEEDS-DECISION` with both
opinions in the digest. The controller never breaks a tie itself.

### 4. Attestations — the contract Part A parses

**The format is defined in `design_docs/planned/m-coordinator-auto-approval-policy.md` §5
("The attestation contract").** That section, and the `docs/internal/triage-attestation-v1.md`
it names, are authoritative. This file restates only the writer's side; it does not duplicate
the parser rules. If they disagree, §5 wins and this file is stale.

One PR comment per attested PR (`gh pr comment N --body-file <file>`), containing exactly one
block, with the fields **in this order, one per line, and no extra keys**:

```
<!-- ailang-triage-attestation v1
verdict: VERIFIED
repo: sunholo-data/ailang
pr: <number>
head_sha: <40 lowercase hex: the PR head the verifier read>
dev_sha: <40 lowercase hex: the origin/dev the verifier read>
verifier: <provider:model that produced the verdict>
judge: <provider:model that concurred>
evidence: <URL of this iteration's evidence comment on the bookkeeping issue>
issued_at: <RFC3339 UTC, e.g. 2026-10-15T12:00:00Z>
-->
Triage: VERIFIED at <short head> by <verifier>, judged by <judge>. [evidence](<url>)
```

- **`sig` (`sig: hmac-sha256:<64 hex>`, after `issued_at`) is written ONLY when a signing key is
  configured** by the mechanism §5 names (Part A milestone M-signing; decision D3, HMAC key in
  Secret Manager or a separate GitHub identity, is unresolved). Until then, **omit the `sig`
  line entirely. Never invent a key, secret or placeholder signature.**
- **Unsigned attestations are informational.** Part A's parser rejects them (rule 2), so they
  feed the digest's "ready for Mark to approve" list and the shadow ledger. They are not an
  approval signal.
- `head_sha` is the head the verifier read. Re-read `headRefOid` immediately before posting. If
  it has moved, do not post; the item is re-verified next iteration (a new push invalidates the
  attestation, D-TRIAGE-9).
- Post the **evidence comment first** (the iteration's verdict table on the bookkeeping issue),
  then use its URL as `evidence`.
- Never edit an attestation comment. Part A treats an edited one as invalid (rule 4). Post a new one.
- Only `VERIFIED` is attested. Only `verifier != judge` attestations are posted.

### 5. Shadow ledger (clause 3)

Every verdict, actioned or not, is appended to
**`~/.ailang/state/mission-triage-shadow-ledger.jsonl`**, one JSON object per line, append-only.
The same rows go in the bookkeeping comment as a table. This is the evidence Part A's policy is
scored against for at least two weeks before it auto-approves anything.

```json
{"ts":"2026-10-15T12:00:00Z","iteration":1,"kind":"pr|issue|card","number":1747,"card":"apr-a4f0a81c",
 "head_sha":"<40hex or null>","dev_sha":"<40hex>","verdict":"VERIFIED","verifier":"codex:gpt-6-luna",
 "judge":"claude:claude-sonnet-5-5","judge_result":"CONCUR|DISSENT|none","action":"attested|closed|labelled|commented|none",
 "evidence":"<url>","outcome":null}
```

- `outcome` is filled in **by a later iteration** as a NEW row with the same `kind`/`number`,
  `"verdict":"OUTCOME"`, and `outcome` set to one of `merged`, `closed-unmerged`, `card-approved`,
  `card-rejected`, `reopened` or `still-open`. Never rewrite a past line.
- **Each digest scores the ledger**: of the items verdicted in the last 14 days that now have an
  outcome, how many agreed (`VERIFIED` → merged or approved; `CLOSE-*` → still closed, not
  reopened; `NEEDS-FIX` → not merged unchanged)? Report `agree/total` and list the disagreements.

### 6. Red `dev` (D-TRIAGE-2) — lead the digest, route the fix, never fix

Gate 1 reads the latest `CI` runs on `dev`. If `dev` is red while V1 is paused
(`~/.ailang/state/mission-control.disabled` present), this loop owns it:

1. The digest **leads** with it: failing run URL, first failing job, and the first red commit.
2. Route the fix. A product defect goes to `design-doc-creator` (a design-doc request naming the
   failing job and commit). A harness defect (`tools/launchd/`, CI plumbing) goes to
   `ailang mission ticket file`. Link whichever you filed.
3. **Never fix it in this loop**, and never push. Ownership returns to V1 when V1 is re-armed.
4. It triggers one Discord post and one `[urgent]` Daneel line (batched, §7).

Level 1 actions merge nothing, so a red `dev` does not block them.

### 7. Communication (D-TRIAGE-3) — exact commands

Every send uses the prod store prefix. Never export it globally:

```bash
MSG="env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac"
```

- **Bookkeeping issue (every iteration).** The full digest and the ledger table, via Gate 5's
  report step (`--body-file`). The digest leads with red `dev` if any, then **Decisions for Mark**,
  then **Ready for Mark to approve** (unsigned attestations, each with
  `ailang coordinator approve <task> --remote gcp`), **Orphans**, **Resolved without handoff**
  (T4), counts, and the ledger score.
- **Discord: ONLY when there is a decision for Mark, an orphan, a red `dev`, or a level change.**
  Never a routine post. One short message (Discord truncates at 2000 chars): counts, the top 3,
  and the link.
  ```bash
  $MSG ailang messages send user "<counts · top 3 · link>" --title "Triage: <headline>" --from mission-triage
  ```
- **Daneel: the digest as `[normal]` every iteration.** Add **at most ONE `[urgent]` per
  iteration**, batching every `NEEDS-DECISION`, orphan and red-`dev` item into it (Daneel caps
  urgent mail at 5/day). It is a pointer, not a payload.
  ```bash
  $MSG ailang messages send daneel-reports "<counts · link to the digest comment>" --from mission-triage --title "[normal] Triage digest: <headline>"
  $MSG ailang messages send daneel-reports "<items · link>" --from mission-triage --title "[urgent] Triage: <n> decisions / orphans / red dev"
  ```
  The body is **positional**. `messages send` has no `--body-file` flag and would send the
  literal string (gate-5-retro step 3).

### 8. Recording the queue fingerprint (the LAST step of the iteration)

The driver's TRIAGE IDLE PRE-CHECK skips a fire whose queue fingerprint equals the recorded one,
so the file must be written **only after this iteration's digest is posted** (the end of Gate 5).
A fire that dies before its digest leaves the old fingerprint, and the next fire re-runs.

```bash
/bin/bash "$MISSION_DRIVER_ROOT/tools/launchd/lib/triage-fingerprint.sh" --record
```

Run it before Gate 5's final `stamp complete`.

Run it **after** every label, comment and close, because those move issue `updatedAt`. The
bookkeeping thread is excluded, so the digest comment itself does not count. It is the same helper
the driver runs, so the reader and the writer cannot disagree. A non-zero exit means the
fingerprint was not recorded. Say so in the Gate-5 report: the next fire will run again, which
costs tokens but loses nothing.
