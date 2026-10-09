# Per-Agent Auto-Merge for Code PRs (check-gated, non-author approver)

Refs #1599 (sunholo-data/ailang) · upstream use case: sunholo-data/daneel#335 (M-DANEEL-SITE-PUBLISH, merged 2026-10-06) · maintainer ruling 2026-10-08 (issue #1599 comment): code PRs may auto-merge per agent, provided required checks pass and a non-author identity approves.

**Status**: Planned
**Target**: v0.52.6
**Priority**: P2 (Medium) — matches issue labels (`priority:P2`, `area:messaging`)
**Estimated**: ~4 days (plumbing 1d, wrapper 1.5d, tests+docs+runbook 1.5d)
**Dependencies**: None in this repo. Deployment precondition: the target repo (first user: the Daneel site repo) carries the M-DANEEL-SITE-PUBLISH S0 ruleset — PR required, 1 approving review, `require_last_push_approval`, no force-push/deletion, bypass = OrganizationAdmin only — applied by Mark.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The merge decision stays a pure function of declared config + diff + GitHub's own state; no polling loop, no timing dependence of ours |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No new effect kinds; the wrapper's GitHub side effects grow by one REST call (approval) whose failure is loud |
| A4: Explicit Authority | +1 | Merge-on-green becomes an explicit per-agent capability with a NAMED non-author approver, instead of being unavailable or globally granted |
| A5: Bounded Verification | +1 | The required-check names are an explicit, machine-checkable bound; a missing name refuses rather than degrading |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Every gate (patterns, check names, approver identity) is machine-decidable from registry config + GitHub API; audit is enumerable by label |
| A8: Minimal Syntax | 0 | YAML registry config only; no language surface |
| A9: Cost Visibility | 0 | Two extra API calls per code PR |
| A10: Composability | +1 | Composes with (never bypasses) repo rulesets and branch protection: GitHub native auto-merge is the only merge actor |
| A11: Structured Failure | +1 | Every refusal carries a machine-printable reason to stderr, exactly like the current guard's reasons |
| A12: System Boundary | +1 | The PR-review boundary crossing is explicit identity: approver ≠ author is enforced by the wrapper, not implied |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects (approval failure is printed, never swallowed)
- [x] A4 (Authority): no ambient access granted — the .md floor stays the default for every agent that does not opt in
- [x] A7 (Machines First): gates and audit are machine-readable

## Problem Statement

**Current State (read on this branch, see Verification Log):**

- `branchIsAutoMergeable` (`cmd/ailang/coordinator_cloud_github.go:161-181`) enables GitHub **native** auto-merge on an agent's PR only when TWO conditions both hold: every changed file matches one of the agent's DECLARED artifact patterns, AND every changed file ends in `.md` (`strings.HasSuffix(f, ".md")`, L173). The markdown floor is hard-coded; there is no per-agent override.
- The floor is correct as a default and is NOT redundant: patterns declare SCOPE, not safety (`pkg-sunholo-ailang-parse` legitimately declares `**/*`). The floor means auto-merge can only ever land documents, so widening a pattern cannot quietly widen what merges unreviewed (see the guard's own comment, L156-160).
- The consequence for non-document agents: an agent whose declared work is HTML/images — e.g. the Daneel **site agent** that edits a static GitHub Pages site — can never have its PR auto-merge, no matter how it is configured.
- The only more-autonomous route that exists is `skip_approval: true` (`internal/coordinator/daemon_tasks_exec.go:226-229`): a direct push to `main` with no PR at all. On the Daneel site repo that is impossible anyway — the repo's `main` carries the M-DANEEL-SITE-PUBLISH ruleset (applied 2026-10-06): PR required, 1 approving review, `require_last_push_approval`, no force-push/deletion, bypass = OrganizationAdmin only, and the fleet identity (`sunholo-voight-kampff`) was lowered to `write`. Measured as the fleet identity: push to `main` → GH013 refused.
- So the Daneel "rung 3" (a mailed poster becomes a merged, published PR with no human merge step) is structurally unreachable: GitHub native auto-merge on that repo would wait forever for an approval the fleet identity cannot give its own PR (GitHub does not count the PR author's approval toward required reviews), and the direct-push lane is closed by design.

**Impact:**

- First user: Daneel site publishing (M-DANEEL-SITE-PUBLISH rung 3; site repo `sunholo-data/rdasouthwestgroup`, a charity site). Today every poster PR waits for Mark to merge it — the exact "PR left for a human is a delay" cost the docs-only auto-merge path was built to remove for markdown agents.
- Every agent whose declared artifacts are not markdown has the same gap; the site agent is the first, not the only, case.
- Who is affected: the coordinator's autonomy ladder (skip_approval → auto_merge docs-only → auto_merge code) is missing its middle rung for code PRs, leaving direct push (no PR, no checks) as the only non-human option — the *less* safe one.

**Maintainer ruling (2026-10-08, issue #1599 comment):** YES — code PRs may auto-merge per agent, not only .md-only PRs, provided the required checks pass and a non-author identity approves. This doc designs: the per-agent opt-in, the required-check set, the non-author approver identity, how `branchIsAutoMergeable` changes, and the audit trail.

## Goals

**Primary Goal:** A per-agent opt-in lets the wrapper enable GitHub-native auto-merge on a CODE pull request — gated by the agent's declared path scope, a non-empty required-check set verified to exist on the base branch, and an approving review from a non-author identity — with every autonomous-merge-enabled PR enumerable afterwards.

**Success Metrics:**

1. A site agent PR (HTML + images, inside its declared patterns) with `auto_merge_code: true` merges itself once the named check passes and the approver identity approves — no human merge.
2. Zero behavior change for every other agent: the `.md` floor remains the default; the existing auto-merge tests pass unmodified in semantics.
3. A misspelled, renamed, or removed check name refuses loudly at enable time instead of enabling auto-merge blind.
4. A same-identity approver (approver login == PR author login) refuses loudly.
5. Every PR on which code auto-merge was enabled carries a label and a body note making it enumerable (`gh pr list --label …`) without touching any new store.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: The `.md` floor is lifted per agent (`auto_merge_code`), never globally and never by path-type sniffing | Keeps the default safe for every other agent; a global lift would silently widen what merges unreviewed fleet-wide | human (ruling 2026-10-08) | design | high |
| D2: The approval is given by the job wrapper, using a second-identity token whose Secret Manager NAME is in the registry (SSHKeySecret pattern), not by the coordinator daemon and not by a GitHub App in v1 | Puts the approval at the same place/time as the PR (satisfies `require_last_push_approval`), reuses a proven secret-plumbing pattern, keeps the coordinator out of the agent's repo auth | agent | design | med |
| D3: Required-check verification = the named checks appear as check-runs on the base branch HEAD | Works with the token the wrapper already holds; catches renamed/moved checks loudly. Existence ≠ required-by-ruleset — that stronger guarantee is the deployment precondition (ruleset, applied by Mark) | agent | design | med |
| D4: One unified scope guard — `branchIsAutoMergeable` gains a mode (docs/code), not a parallel function | Prevents the two guards drifting apart; the anti-pattern the skill warns about (growing special cases) | agent | design | low |
| D5: Audit = PR label + PR body note + job log line; no new store | The PR itself is the durable public record; the label makes the set enumerable with existing tooling | agent | design | low |
| D6: Approve AFTER enabling auto-merge, in the same wrapper run, after the final push | `require_last_push_approval` needs the approval to postdate the last push; a re-executed task re-runs the wrapper and re-approves | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — per-agent opt-in only; `.md` floor stays the default for every agent without the opt-in (maintainer ruling, 2026-10-08)
- [x] Non-author approver is MANDATORY for code auto-merge — no configuration may enable code auto-merge without an approver secret (maintainer ruling)
- [x] Required-check set must be non-empty for code auto-merge — an empty set would mean "merge the moment the approver approves", unguarded by any check (maintainer ruling: "the required-check set")
- [ ] Deployment precondition confirmed by Mark: the target repo's ruleset is applied and `require_last_push_approval` is on (S0 of M-DANEEL-SITE-PUBLISH) before the site agent's `auto_merge_code` flips to true in the registry

## Solution Design

### Overview

GitHub native auto-merge stays the ONLY merge actor (unchanged). What changes is when the wrapper is allowed to turn it on: a second, per-agent mode in which the markdown floor is replaced by FOUR other gates — declared-path scope, a required-check set verified against the base branch, an approving review from a non-author identity, and an audit trail. The dispatcher forwards registry config; the wrapper (which already opens the PR) enforces everything, as it is the security boundary closest to the diff.

**All six conditions must hold for a code PR to auto-merge:**

1. **Registry opt-in** (trusted metadata, never message content): `auto_merge: true` AND `auto_merge_code: true` on the agent's registry entry.
2. **Path bound**: every changed file matches one of the agent's DECLARED `artifact_patterns`; empty declaration → refuse (existing semantics, unchanged — `GetEffectiveArtifactPatterns`'s `**/*` default is discovery-only and is deliberately NOT read here).
3. **Required-check set**: `auto_merge_required_checks` is non-empty, and every named check EXISTS on the base branch (check-runs on the base HEAD). A missing name → refuse.
4. **Non-author approver**: the wrapper fetches the approver token from the named Secret Manager secret, verifies its login differs from the PR author's, and posts an approving review.
5. **Repo-side ruleset** (deployment precondition, human-applied): PR + 1 approving review + `require_last_push_approval` + no force-push/deletion — the durable guarantee that nothing reaches `main` without a PR and a review. GitHub enforces this; the wrapper cannot and must not work around it.
6. **Audit**: a label and a body note on the PR, a job log line naming approver, checks, and scope.

If any of 1-4 fails, auto-merge is NOT enabled and the reason is printed — the PR stays open for a human, exactly like today. If the check never passes, GitHub simply never merges (native behavior; there is no polling loop of ours to get wrong).

### Architecture

**Registry (trusted metadata):** four new optional fields on `AgentConfig` (`internal/coordinator/agent_registry.go`), all default-zero so absence means "docs-only or nothing", the loud direction:

```yaml
- id: site-agent
  # ... existing fields ...
  auto_merge: true                      # existing: GitHub may merge this agent's PR on green
  auto_merge_code: true                 # NEW: lift the .md floor for THIS agent only
  auto_merge_required_checks:           # NEW: non-empty required for code mode
    - site-wellformed                   #   names of the checks the site repo itself runs
  auto_merge_approver_secret: site-approver-token   # NEW: Secret Manager NAME, never the token
  auto_merge_approver_identity: sunholo-voight-approver  # NEW: expected approver login (sanity + audit)
```

- `AutoMergeCode bool` — the per-agent opt-in (D1).
- `AutoMergeRequiredChecks []string` — the required-check set. Non-empty REQUIRED for code mode; a code auto-merge with no required check merges the moment the approver approves, regardless of what the change did.
- `AutoMergeApproverSecret string` — Secret Manager secret NAME holding the second identity's token. Only the name lives in the registry and in the job env; the job's own service account fetches the material (the `SSHKeySecret` pattern: nothing secret ever enters the Cloud Run execution spec, which project viewers can read).
- `AutoMergeApproverIdentity string` — the expected approver login. The wrapper verifies the token's actual login matches (a rotated/overwritten secret holding a DIFFERENT identity must be loud, not silently trusted) and that it differs from the PR author.

**Dispatcher plumbing (trusted metadata, never message content):** `internal/coordinator/daemon_tasks_exec.go` copies the four fields into dispatch params (`cloud_dispatcher.go` gains the param fields with the same authority-boundary comments as `AutoMerge`); `internal/dispatch/cloudrun/dispatcher.go` forwards them as job env vars, following the existing discipline:

- `AILANG_AUTO_MERGE_CODE` — only set when true (absent and "0" mean the same thing).
- `AILANG_AUTO_MERGE_REQUIRED_CHECKS` — newline-separated; a separator that can appear in the data is how a scope guard silently widens (same rationale as patterns).
- `AILANG_APPROVER_SECRET`, `AILANG_APPROVER_IDENTITY` — only set when non-empty.

`internal/config/job.go` gains the env names and getters, mirroring `EnvAutoMerge`/`EnvArtifactPatterns`.

**Wrapper (the security boundary), all in `cmd/ailang/coordinator_cloud_github.go`:**

1. **Unified scope guard (D4).** `branchIsAutoMergeable(ctx, workDir, baseBranch string, patterns []string, mode AutoMergeMode) (bool, string, error)` — mode `MergeModeDocs` (default) keeps the `.md` suffix floor verbatim; mode `MergeModeCode` skips the suffix check and keeps the pattern loop: declared patterns non-empty, every changed file matches. No parallel function; the two modes share the loop so they cannot drift.
2. **Required-check verification (D3).** New REST helper `requiredChecksPresentOnBase(ctx, token, owner, repo, baseBranch string, checks []string) (missing []string, err error)`: `GET /repos/{owner}/{repo}/commits/{base}/check-runs`, collect check names reported on the base HEAD, and return the declared names absent from that set. Any missing name → refuse with the names printed. This catches config drift (renamed workflow, moved check) loudly instead of a PR that auto-merges with no check ever gating it.
3. **Auto-merge enable (unchanged).** `enableGitHubAutoMerge` (GraphQL `enablePullRequestAutoMerge`, SQUASH) is reused as-is; it is the only API that sets native auto-merge.
4. **Non-author approval (D2, D6).** New REST helper `approvePRAsNonAuthor(ctx, approverToken, owner, repo string, prNum int, note string) error`: `GET /user` on the approver token (must match `auto_merge_approver_identity`, else loud refusal), `GET /repos/{o}/{r}/pulls/{n}` for the PR author login (same login → loud refusal — belt under GitHub's own author-approval exclusion), then `POST /repos/{owner}/{repo}/pulls/{n}/reviews` with `{"event":"APPROVE","body":<audit note>}`. Called AFTER the enable, in the same wrapper run, after the final push.
5. **Audit (D5).** On successful enable: `addGitHubLabels(ctx, token, owner, repo, prNum, []string{"auto-merge-code"})` (best-effort, same call shape as the existing label step) and the PR body gains a note (see below). Stdout line: `execute-job: auto-merge enabled on #%d (code; approver=%s; required checks: %s) — GitHub will merge when the checks pass`.

**PR body note (audit, written at PR creation for code-mode agents):** `agentPRBody` gains a section when the agent is code-auto-merge-configured:

```
**Auto-merge (code)**: enabled per-agent (`auto_merge_code`); scope: <patterns>;
required checks: <checks>; approver: <identity> (non-author). Refs #1599.
```

This makes each PR its own durable public record of WHO approved it and WHAT gated it, readable without access to the registry.

**Why native auto-merge + second identity, and not our own merge call:** unchanged from the existing design — GitHub does the waiting, honours branch protection, and never merges something protection would refuse. The problem on a ruleset-protected repo was never the merge mechanics; it was that the required approval could never appear, because the author cannot self-approve. The second identity is the missing reviewer, not a merge authority of its own: it can only approve, and only the ruleset decides when that is enough.

### Coordination Conflict Surface

Not a parser/typechecker change, but the wrapper runs in EVERY agent's repo, so the coordination surface must be enumerated:

1. **Positions extended:** the auto-merge decision path (`maybeEnableAutoMerge` → `branchIsAutoMergeable`) and the agent registry YAML schema.
2. **Other valid constructs already living there:** every agent with `auto_merge: true` today (docs-only mode) flows through the same two functions; `skip_approval` agents (direct push, `PushBranch = MergeBranch`) never reach the PR path at all; SSH-key agents (`ssh_key_secret`, paired with `push_branch`) cannot open PRs and are unaffected; the coordinator-plane approval workflow (`approval` config, message-plane approvals/handoffs) is a different plane from GitHub PR reviews and does not interact.
3. **Disambiguation:** mode is resolved from the agent's registry entry only (`AILANG_AUTO_MERGE_CODE` env, set by the dispatcher) — a message/directive can never select it. Absent variable = docs mode, the current behavior.
4. **Existing programs that MUST still work (regression fixtures):**
   - `daneel-writer` in `sunholo-data/daneel-memory` (`documents/**/*.md`) — docs mode, unchanged.
   - `design-doc-creator` in this repo (`design_docs/**/*.md`) — docs mode, unchanged.
   - An agent with `auto_merge: true` and NO declared patterns — still refused ("nothing bounds what it may merge"), unchanged.
   - `pkg-sunholo-ailang-parse`-style `**/*` declarations in DOCS mode — still floored by `.md`, unchanged.
   - The `matchAll`-based table in `cmd/ailang/coordinator_cloud_automerge_test.go` — extended with code-mode rows, existing rows' semantics untouched.
5. **Deliberate changes:** none for any agent without `auto_merge_code: true`. The `.md` floor becomes conditional per agent rather than universal — that is the whole feature, and it is the ruling.

### Implementation Plan

**Phase 1: Config + plumbing (~1 day)**
- [ ] `AgentConfig`: four fields + comments (`internal/coordinator/agent_registry.go`)
- [ ] Dispatch params + docs (`internal/coordinator/cloud_dispatcher.go`), copy from registry (`internal/coordinator/daemon_tasks_exec.go`)
- [ ] Env names + getters (`internal/config/job.go`), forward (`internal/dispatch/cloudrun/dispatcher.go`)
- [ ] Guide update: config table + YAML examples (`docs/docs/guides/coordinator.md`)

**Phase 2: Wrapper (~1.5 days)**
- [ ] Mode parameter on `branchIsAutoMergeable` + refactor (D4), code path in `maybeEnableAutoMerge`
- [ ] `requiredChecksPresentOnBase` (check-runs on base HEAD)
- [ ] `approvePRAsNonAuthor` (GET /user, GET PR author, POST review)
- [ ] Audit: label, PR body note, log line

**Phase 3: Tests + runbook (~1.5 days)**
- [ ] Guard table extension (code mode), required-check parsing, identity-comparison unit tests
- [ ] API-level tests for the three new helpers (httptest, following the existing ad-hoc `http.NewRequestWithContext` layer; introduce a small injectable transport if the existing pattern cannot be reused)
- [ ] Daneel S0 runbook section: ruleset verification command, ten-clean-merges gate

### Files to Modify/Create

**Modified files:**
- `internal/coordinator/agent_registry.go` — 4 struct fields + comments (~25 LOC)
- `internal/coordinator/cloud_dispatcher.go` — 4 param fields + authority comments (~35 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — copy fields into params (~12 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — env forwarding (~30 LOC)
- `internal/config/job.go` — 4 env names + getters (~20 LOC)
- `cmd/ailang/coordinator_cloud_github.go` — mode refactor + 3 helpers + audit (~190 LOC)
- `cmd/ailang/coordinator_cloud_automerge_test.go` — code-mode table rows + parsing tests (~90 LOC)
- `cmd/ailang/coordinator_cloud_github_test.go` — API tests for new helpers (~120 LOC)
- `docs/docs/guides/coordinator.md` — config table + example (~30 LOC)

**New files:** none.

## Examples

### Example 1: The Daneel site agent (rung 3)

**Before (today):** site agent pushes `site/posters/2026-10/poster.html` + `.jpg`, opens a PR. `maybeEnableAutoMerge` → `branchIsAutoMergeable` → `poster.html is not a document; auto-merge only ever lands markdown` → PR waits for Mark. Every poster, forever.

**After:**

```yaml
# registry (trusted metadata)
- id: site-agent
  auto_merge: true
  auto_merge_code: true
  auto_merge_required_checks: ["site-wellformed"]
  auto_merge_approver_secret: site-approver-token
  auto_merge_approver_identity: sunholo-voight-approver
  artifact_patterns: ["site/**"]        # the path bound that replaces the .md floor
```

```
execute-job: opened PR #42: https://github.com/sunholo-data/rdasouthwestgroup/pull/42
execute-job: required checks on main: site-wellformed ✓ (present)
execute-job: auto-merge enabled on #42 (code; approver=sunholo-voight-approver; required checks: site-wellformed)
execute-job: approved #42 as sunholo-voight-approver (non-author)
→ GitHub merges #42 when site-wellformed passes; Daneel mails the merged PR.
```

Failure modes, each loud: `site-wellformed` renamed → `auto-merge NOT enabled: required check(s) missing on main: site-wellformed`; approver secret rotated to a different identity than declared → loud refusal; approver token == PR author → `auto-merge NOT enabled: the approver must not be the PR author`; agent strays outside `site/**` → existing refusal, unchanged wording.

### Example 2: Every other agent

No registry entry sets `auto_merge_code` → `AILANG_AUTO_MERGE_CODE` unset → `MergeModeDocs` → byte-identical behavior to today, including the refusal reasons and the existing tests.

## Success Criteria

- [ ] Code-mode agent PR (HTML+jpg inside declared patterns, check present, approver non-author) enables native auto-merge and receives the approving review — acceptance: unit + API test, plus one staging-repo end-to-end merge
- [ ] Docs-mode behavior byte-identical for all agents without the opt-in — acceptance: existing `coordinator_cloud_automerge_test.go` rows pass unchanged
- [ ] Missing check name refuses with the name printed — acceptance: API test
- [ ] Same-identity approver refuses loudly — acceptance: API test
- [ ] Empty `auto_merge_required_checks` with `auto_merge_code: true` refuses — acceptance: unit test
- [ ] PR carries the `auto-merge-code` label and the audit body note — acceptance: API test
- [ ] All tests passing (`make test`), documentation updated (`docs/docs/guides/coordinator.md`)
- [ ] Daneel S0 runbook + ten-clean-merges gate recorded (deployment side)

## Testing Strategy

**Unit tests:**
- Guard table: code-mode rows (in-scope HTML merges scope-wise; out-of-scope file refused; empty patterns refused; docs rows unchanged)
- `auto_merge_required_checks` env parsing (newline separator, trim, empty → refuse)
- Identity comparison (approver == author, mismatch vs declared identity)

**Integration/API tests (httptest):**
- `requiredChecksPresentOnBase`: present / missing / API error (error → refuse, never degrade)
- `approvePRAsNonAuthor`: login mismatch, author-collision, approve success, approve API failure (loud, PR stays open)
- Label + body-note emission

**Manual testing (Daneel S0):**
- `gh api repos/sunholo-data/rdasouthwestgroup/rules/branches/main` — ruleset present, `require_last_push_approval` on
- One poster PR through the full flow on the live repo; then the ten-clean-merges gate (M-DANEEL-SITE-PUBLISH decision 5) before the registry entry is considered production

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact label name (`auto-merge-code` suggested) — agent may choose, keep it greppable and stable
- Whether the approver secret may later hold a GitHub App installation token (token-type detection) — agent may choose at implementation; v1 assumes a PAT for a second identity
- Whether `requiredChecksPresentOnBase` should ALSO read repo rulesets (`GET /repos/{o}/{r}/rules/branches/{base}`) when the token permits — agent may choose; v1 existence check is sufficient with the S0 precondition
- Test transport injection style for the GitHub API helpers — agent may choose

## Non-Goals

**Not attempted in this feature:**
- Merging PRs ourselves (any direct merge API call) — GitHub native auto-merge only, unchanged
- Removing or weakening the `.md` floor for any agent that does not opt in — the floor IS the default
- Auto-merge on repos without the wrapper's PR path (SSH-key/push-branch agents) — they cannot open PRs at all
- Re-approving after a post-approval push by a third party (no polling loops; a re-executed task re-runs the wrapper and re-approves)
- The site repo's own check workflow (HTML well-formedness, image size cap) — Daneel-side, out of this repo's scope; AILANG only consumes the check NAME
- Changing `skip_approval`, the autonomy router, or the message-plane approval workflow

## Timeline

**Week 1** (~4 days):
- Phase 1: config + plumbing + guide (1d)
- Phase 2: wrapper guard, checks, approval, audit (1.5d)
- Phase 3: tests + Daneel S0 runbook (1.5d)

**Total: ~4 days across 1 week** (2x the naive estimate, per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Approver token IS merge authority if the target repo's ruleset is missing (no PR/approval requirement) | High | Deployment precondition (S0, human-applied, in Design Freeze); non-empty required checks; existence check; label audit makes every autonomous merge enumerable for review |
| Approver token is broader than one repo (a fleet-wide PAT) | High | The secret is per-deployment (`site-approver-token`), documented as one-identity/one-repo; the wrapper verifies the login against the DECLARED identity, so a token swapped for a broader one is loud |
| Check renamed or moved after S0 | Med | `requiredChecksPresentOnBase` refuses loudly at enable time with the missing name |
| Agent pushes again after the approval (stale review dismissal) | Med | Wrapper approves after the final push of each run; a re-executed task re-runs the wrapper (re-push + re-approve) — the PR otherwise waits for a human, which is the safe direction |
| Approver secret expired/rotated | Low | Approval fails loudly (stderr); PR stays open; no silent degradation |
| Native auto-merge flag disabled at repo/org level | Low | `enableGitHubAutoMerge` reports GraphQL errors in-band (already handled); loud, PR stays open |

## Verification Log

Every load-bearing claim above was verified against the code or the issue on 2026-10-08 (this branch, which is `origin/dev` `658ff76a3` as triaged):

| Claim | Evidence |
|---|---|
| `branchIsAutoMergeable` requires patterns match AND `.md` suffix, hard-coded, L161-181 | Read `cmd/ailang/coordinator_cloud_github.go:161-181` (`strings.HasSuffix(f, ".md")` at L173) |
| `maybeEnableAutoMerge` gated on `AILANG_AUTO_MERGE` (registry → dispatcher, never message content) | Read `cmd/ailang/coordinator_cloud_github.go:114-137`; `internal/config/job.go:182-183`; `internal/dispatch/cloudrun/dispatcher.go:296-310`; comment at `cloud_dispatcher.go` AutoMerge field |
| Patterns are newline-separated in env because commas can appear in patterns | Read `internal/dispatch/cloudrun/dispatcher.go:303-310` and `cmd/ailang/coordinator_cloud_github.go:183-198` |
| `params.AutoMerge = agent.AutoMerge`; DECLARED patterns (not `GetEffectiveArtifactPatterns`, whose default is `**/*` discovery-only) are passed | Read `internal/coordinator/daemon_tasks_exec.go` (auto-merge block, ~L275) and `internal/coordinator/agent_registry_effective.go:82-93` |
| `skip_approval` = direct push to `MergeBranch`, no PR | Read `internal/coordinator/daemon_tasks_exec.go:226-229` (`if agent.SkipApproval && agent.MergeBranch != "" { params.PushBranch = agent.MergeBranch }`) |
| Auto-merge enable is GraphQL-only, SQUASH, and errors in-band | Read `cmd/ailang/coordinator_cloud_github.go:611-643` |
| Secret-name-only plumbing precedent (`SSHKeySecret`) exists; material fetched by the job via Secret Manager (`fetchSecret`) | Read `internal/coordinator/agent_registry.go` SSHKeySecret comment; `cmd/ailang/coordinator_cloud_sshkey.go:53, 83-86, 134-147` |
| No existing code queries GitHub check-runs or rulesets (the verification helper is NEW) | `grep -rn "rules/branches\|rulesets\|check-runs" cmd/ internal/ --include="*.go"` → no non-test hits |
| No existing code posts PR reviews (the approval helper is NEW) | `grep -rn "pulls/.*reviews\|/reviews" cmd/ internal/ --include="*.go"` → no non-test hits |
| No GitHub-App installation-token machinery exists (v1 PAT secret is not a duplicate of something) | `grep -rn "github app\|GitHub App\|installation_id\|appId" cmd/ internal/coordinator/*.go` → only codex/gemini provider auth, unrelated |
| The guard's own rationale for the `.md` floor (patterns declare scope, not safety) | Read the comment block `cmd/ailang/coordinator_cloud_github.go:140-160` |
| Daneel use case, ruleset facts (PR + 1 approval + `require_last_push_approval`, bypass = OrganizationAdmin, fleet identity `write`, push to main refused), ten-clean-merges gate, "first user" | Issue #1599 body + maintainer comment (GitHub API, 2026-10-08); sunholo-data/daneel#335 (merged 2026-10-06) |
| Docs-only auto-merge was itself a deliberate, previously-triaged extension (not an accident to "fix") | Read `design_docs/planned/ailang-core-triage/coordinator-design-pr-auto-merge.md` (2026-09-15 triage) |
| `docs/docs/guides/coordinator.md` documents `auto_merge` in the per-agent config table (guide update is real work) | Read `docs/docs/guides/coordinator.md` (~L110-125, ~L485-505) |
| Existing auto-merge tests live in `cmd/ailang/coordinator_cloud_automerge_test.go` with a table to extend, not replace | Read the file (`TestAutoMergeScope_PerAgentPatterns`, `TestArtifactPatternsFromEnv`) |

Platform claims that remain unverifiable from this repo (GitHub does not count the PR author's approval toward required reviews; `enablePullRequestAutoMerge` semantics) are cited from the issue itself, which measured them live on the site repo on 2026-10-06, and from the existing wrapper comments that encode the same behavior. Whether a GitHub App's approving review satisfies a ruleset's "1 approving review" is deliberately NOT asserted — v1 uses a second user/bot identity PAT (see Deferred Decisions).

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_10_0/m-pkg-autonomous-updates.md` — the always-PR v1 design this extends; "no autonomous merge" was v1, the `.md`-floor was its loosening, this is the per-agent code rung
- `design_docs/planned/m-coordinator-execution-trust.md` — trusted-registry authority boundary (WorkTier/AcknowledgeOnly): the new fields follow the same "registry, never message content" rule

**Planned (check for overlap):**
- `design_docs/planned/ailang-core-triage/coordinator-design-pr-auto-merge.md` — the 2026-09-15 triage of the docs-ONLY auto-merge; distinct by design (it pinned the docs-only scope; this doc is the maintainer-approved widening to code, per-agent)
- `design_docs/planned/HANDOVER-message-plane-approvals-2026-09-23.md` — message-plane approval authority; a different plane from GitHub PR reviews, no overlap

## References

- Issue #1599 — this feature's tracker (reported by daneel; ruling comment 2026-10-08)
- sunholo-data/daneel#335 — M-DANEEL-SITE-PUBLISH (rung 3 is this feature's first user)
- `design_docs/planned/ailang-core-triage/coordinator-design-pr-auto-merge.md` — docs-only auto-merge triage
- `cmd/ailang/coordinator_cloud_github.go` — `maybeEnableAutoMerge`, `branchIsAutoMergeable`, `enableGitHubAutoMerge`
- `cmd/ailang/coordinator_cloud_sshkey.go` — the secret-name-only plumbing pattern this reuses
- `docs/docs/guides/coordinator.md` — per-agent config reference (to be updated)
- GitHub docs: [auto-merge](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-in-a-pull-request/automatically-merging-a-pull-request), [required approving reviews](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches#require-approval-from-specific-reviewers), [check-runs API](https://docs.github.com/en/rest/checks/runs)

## Future Work

- **GitHub App approver:** mint per-repo installation tokens instead of a PAT secret (removes long-lived credentials); requires resolving the open question of whether an App review satisfies the ruleset's approving-review count
- **Ruleset verification in the wrapper:** read `GET /repos/{o}/{r}/rules/branches/{base}` when the token permits, upgrading existence → required-by-ruleset (v1 relies on the S0 precondition)
- **Merge-completion audit span:** an observatory record when the PR actually merges (today the merge is GitHub's act; our record ends at "enabled + approved")
- **Per-agent merge cooldown / rate limit** if autonomous code merges become routine beyond the site agent

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08
