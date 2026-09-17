# M-DANEEL-SOUL — one voice for every Daneel process: SOUL.md in daneel-memory, lent to the executor per request, self-edited by Daneel without approval

**Status**: Planned
**Target**: v0.39.4 — fleet/deployment config (ailang-multivac registry + template) + Daneel-repo changes + daneel-memory content; **no AILANG-core item**
**Priority**: P1 — Daneel speaks through three surfaces today (the host's `compose()`, the daneel-writer prompt, the executor template) and nothing keeps them in the same voice; the executor in particular answers as "a task" rather than as Daneel
**Estimated**: ~2 days (SOUL.md v1 draft ~0.5d, host lending + SOUL_EDIT application ~0.5d, template + writer prompt wiring ~0.5d, end-to-end probe + docs ~0.5d)
**Dependencies**: [M-DANEEL-AILANG-EXECUTOR](../../implemented/v0_39_3/m-daneel-ailang-executor.md) (landed v0.39.3 — the executor, the Request record, the `ANSWER:` convention, the `acknowledge_only` registry shape); M-DANEEL-EXECUTOR-LONG-ANSWERS (the `ANSWER_DOC:` marker family this design extends)

**Created**: 2026-09-17 · **Author**: attended session with Mark (goal, premises, rulings D1–D3 — all ratified in conversation, recorded below as decided and not to be re-opened)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Every request states exactly which memory it was lent: SOUL.md carries a `last-edited:` line (task ref + date) and the host's SOUL_EDIT commits are a git history, so "what voice did this answer use" is a lookup, not a guess. The same request + same daneel-memory ref produces the same lent text. |
| A2: Replayability | +1 | A SOUL_EDIT is a direct commit to daneel-memory whose message names the task ref; the answer that requested it rides the completion path with its correlation ID. The full edit round trip is reconstructible from two repos' histories. |
| A3: Effect Legibility | +1 | The executor still writes no memory — its policy excludes `Msg` and grants only `Process(git ro)`, so the ONLY write path into daneel-memory from an answer is the host applying a typed, validated `SOUL_EDIT:` request line. The crossing is one declared seam, not ambient. |
| A4: Explicit Authority | +1 | The design's whole spine. Self-edit without approval is bounded by validation, not by a human: size bound, sections intact, no secrets, no authority-widening instructions (an edit that tries to grant itself mail or a domain is refused and logged). And the self-edit grant reaches only the MEMORY repo — the PROGRAM repo (daneel, `skip_approval: false`) still requires a PR, so no Daneel process can vote itself authority. |
| A5: Bounded Verification | +1 | Host-side validation is a finite check list (size, sections, secret scan, authority-widening pattern scan) that runs before the commit — bounded, local, and testable with table fixtures. |
| A6: Safe Concurrency | 0 | No concurrency change. The host applies SOUL_EDIT commits serially in its existing work loop (tools/daneel-work.sh), same as it consumes completions today. |
| A7: Machines First | +1 | SOUL.md is a structured document (four named sections + `last-edited:` line) that machines compose into prompts; the SOUL_EDIT grammar is a marker line, same family as `ANSWER:`/`ANSWER_DOC:` — parsed, not prose a human must relay. |
| A8: Minimal Syntax | +1 | No AILANG-core change. One new memory file, one marker convention, host-side validation, template/prompt text edits. |
| A9: Cost Visibility | +1 | The lent memory is a bounded byte budget (SM2), so the per-ask token cost of "being Daneel" is stated, not open-ended; the SOUL_EDIT rate cap bounds the memory repo's churn. |
| A10: Composability | +1 | Composes existing machinery only: the Request record's existing `body`/`hint` fields, the host's existing completion consumer, daneel-writer's existing direct-commit path into daneel-memory, the `ANSWER_DOC:` marker family. No new plane, no new repo, no new image. |
| A11: Structured Failure | +1 | A refused SOUL_EDIT is a structured outcome: the host logs the refusal with the reason and the task ref, tells the requester in the mailed answer ("SOUL edit refused: <reason>"), and never silently drops it. A failed validation changes nothing — the commit simply does not happen. |
| A12: System Boundary | +1 | The MEMORY repo and the PROGRAM repo stay two authorities with two write policies (D1/D3): the voice and lessons live where daneel-writer already works; the program stays behind a PR gate. The executor holds no repo and no token for either (D2) — it only receives text and returns text. |

**Net Score: +11** → **Decision: ✅ Proceed to implementation** (rulings D1–D3 already ratified by Mark; remaining decisions are agent-resolvable and measured)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism; lent memory is pinned per request by the `last-edited:` line and git history
- [x] A3 (Effects): no hidden side effects — the executor gains no effect; the host's commit is the one new effect and it is explicit, validated, and serial
- [x] A4 (Authority): tightens Daneel's boundary rather than widening it — the executor gains ZERO authority (it requests; the host validates and commits); the self-edit grant is scoped to the memory repo by construction
- [x] A7 (Machines First): the whole contract is marker lines and named sections, not human-ritual prose

---

## Problem Statement

**Current state (measured premises from the 2026-09-17 attended session; Daneel-repo rows re-verified at sprint start — see Verification Log):**

1. **Daneel speaks from three places, and nothing keeps them the same voice.** Who Daneel is lives mostly in `identity-and-messaging.md` in the daneel repo — written once for the host, never fed to the other speakers. The host composes its own words via `Host.compose(kind, fields)` ("so the voice stays in one place" — `tools/daneel_intake.ail` / `ext/abi/types.ail`), but that "one place" is a function body, not a document. The daneel-writer prompt and the executor task template (`ailang-multivac config/templates/daneel-executor-task.md`) each describe Daneel's manner independently — or not at all. An answer that comes back from `daneel-executor` reads as "a task completed", not as Daneel: nothing in the executor's prompt tells it who it is.

2. **The executor is a stranger to Daneel's memory.** Under M-DANEEL-AILANG-EXECUTOR, the executor's workspace is the daneel repo clone; its policy grants `{IO, FS, Clock, Net, AI, Process(git ro)}`, no `Msg`, no mail. It cannot read daneel-memory (no clone, no token — D2's lending pattern is deliberate) and cannot write anywhere but its sandbox. Every answer therefore arrives with none of the context the host holds — the people, projects, lessons, decisions — unless the host lends it, and today the host lends only what the dispatcher happens to copy into the question.

3. **Memory updates are bottlenecked on the host.** daneel-writer can commit documents into daneel-memory directly (registry: workspace `sunholo-data/daneel-memory`, `merge_branch: dev` — its default branch is `dev`, `main` does not exist there), but Daneel itself — the process that learns things in conversation — has no path to record what it learned. Every lesson, every refinement of how it wants to speak, waits for a host-authored or writer-authored change. Mark's ruling on the shape: **"I don't want to approve every daneel-memory update."**

**Impact:** every recipient of a Daneel message experiences the seams — the host's words, the writer's drafts, and the executor's answers drift apart as each is edited separately; the executor answers with no memory of who it is or who it is talking about; and Daneel cannot improve itself because the improvement loop is closed by a human it explicitly should not need.

---

## Goals

**Primary Goal:** One file — `SOUL.md` in sunholo-data/daneel-memory — is the single source of Daneel's voice, lent into every executor request (and feeding `compose()` and the daneel-writer prompt), and self-edited by Daneel via a validated host-applied direct commit, so all three speakers stay one Daneel.

**Success Metrics:**

1. **One-source probe:** the executor's task template, the host's `compose()`, and the daneel-writer prompt all reference `SOUL.md` (by path or by lent text); grep finds no second, independent description of Daneel's voice in any of the three surfaces.
2. **Lending probe:** a dispatched executor request's `Request` record carries SOUL.md text (always) plus hint-selected memory slices within the byte budget (SM: budget table below); the executor's prompt tells it to answer as Daneel using the lent voice.
3. **Self-edit probe:** one live answer carrying a well-formed `SOUL_EDIT:` produces a direct commit to daneel-memory whose message names the task ref, with no approval step; one authority-widening `SOUL_EDIT:` is refused, logged, and produces no commit — both behaviours asserted, not assumed.
4. **Program-repo boundary probe:** the executor attempting a program-repo edit still produces a PR (daneel stays `skip_approval: false`, multivac `a11d649`) — the self-edit grant provably did not leak across D1's repo line.
5. **Voice probe (qualitative, one live question):** the answer mailed back from an executor round trip reads as Daneel (per SOUL.md's "how I speak"), and describes no tools.

---

## RULINGS — ratified by Mark in conversation (2026-09-17), recorded as decided, NOT to be re-opened

These three rulings are freeze items that are already decided. The sprint does not re-litigate them; it implements them.

| Ruling | Content | Why |
|---|---|---|
| **D1 — WHERE** | `SOUL.md` lives in **sunholo-data/daneel-memory** — the MEMORY repo (`context/now.md`, `lessons.md`, `people/`, `projects/`, `decisions/` — where daneel-writer already works) — **NOT** in sunholo-data/daneel — the PROGRAM repo (host, `ext/*`, `authority.md`). | The voice is memory, not program. Same reason the memory repo already has a different write policy (D3); co-locating the voice with `authority.md` would make every tone adjustment a program edit. |
| **D2 — HOW THE EXECUTOR SEES IT** | The executor does **NOT** get a clone of daneel-memory. The host **lends** memory per request: at dispatch it attaches SOUL.md and the relevant memory slices into the `Request` record (`body`/`hint` fields, `ext/abi/types.ail`) it already builds. | This is the lending pattern of ext/abi v0.4 and of M-DANEEL-AILANG-EXECUTOR: the executor holds no second repo and no token. Giving it a memory clone would widen its authority and its prompt-injection surface in one move. |
| **D3 — HOW IT EDITS** | Daneel self-edits `SOUL.md` (and other memory files) **WITHOUT a PR**: the executor's answer may carry a request line (`SOUL_EDIT: …` — same family as the `ANSWER_DOC:` marker in M-DANEEL-EXECUTOR-LONG-ANSWERS) and the host applies it as a **DIRECT commit** to daneel-memory, exactly as daneel-writer commits there today. Git history is the audit trail; revert is the safety net. **Mark: "I don't want to approve every daneel-memory update."** The executor's own repo (`daneel`) stays `skip_approval: false` (multivac `a11d649`) so a program-repo edit is still a PR. | The approval cost exceeded the risk: memory edits are reversible by `git revert`, attributable by git history, and validate-able by rule (below). Program edits keep the PR gate — the asymmetry IS the design. |

---

## Design Questions — answered

Six questions the brief requires the doc to settle. Each is agent-resolvable under the rulings; each names its measurement.

### Q1 — SOUL.md shape

**Four named sections, one metadata line, a hard 4 KB bound.**

```markdown
# SOUL.md — who Daneel is

last-edited: <task-ref or "daneel-writer"> · <YYYY-MM-DD>   ← maintained by the host at commit time

## Who I am
## How I speak
## What I never do
## What I am learning
```

- **Sections (exact headings, load-bearing — validation and the SOUL_EDIT grammar key off them):** *Who I am* (identity: name, role, relationship to Mark — drafted FROM `identity-and-messaging.md`, not invented); *How I speak* (tone, length, directness, formatting habits — this is the section `compose()` and the executor template consume); *What I never do* (behavioural red lines — phrased as Daneel's own commitments, mirroring but never exceeding `authority.md`); *What I am learning* (a dated, append-mostly scratchpad of lessons Daneel chose to record about itself).
- **Size bound:** 4,096 bytes total, validated at every commit. It must fit in EVERY prompt that carries it — host compose context, writer prompt, executor request — so the bound is set by the smallest consumer, the executor request, whose whole memory budget is 12 KB (Q2).
- **`last-edited:` line:** written by the HOST whenever it applies a commit (SOUL_EDIT or daneel-writer), naming the requesting task ref and date. It is how a recipient of any Daneel message can name the voice version it heard (A1), and how the host can refuse a stale SOUL_EDIT cleanly (Q3).
- **v1 draft:** the host authors it in one sitting from `identity-and-messaging.md`, moving — not paraphrasing into a new personality — the existing content into the four sections. Anything not already in `identity-and-messaging.md` does not go into v1. (The draft session is M1 of the plan.)

### Q2 — how much memory is lent per ask

**SOUL.md always; slices by the request's `hint`/tag; a 12 KB total budget.**

| What | When | Bound |
|---|---|---|
| `SOUL.md` | every dispatch, verbatim, in the `Request` `hint` (the field the executor template already teaches as "context from the host") | ≤ 4 KB (Q1's bound) |
| Memory slices (`people/<who>`, `projects/<what>`, relevant `decisions/` entries) | when the request's `hint`/tag names a person, project, or standing decision | ≤ 8 KB total; one slice per named entity, selected by the host's existing tag matching in `daneel_memory.ail` |
| `context/now.md`, `lessons.md` | **not lent** in v1 — they are host-side working memory; the writer reads them from its own checkout | 0 |

The host already builds the `Request` record at dispatch (`ext/abi/types.ail` v0.4; `tools/daneel_intake.ail`) and already knows the target from the request routing — the lending step is content selection into fields that exist, not a new record. Selection failure is never silent: if a `hint` names a person with no `people/` entry, the request says so ("no memory for <name>") rather than lending nothing and letting the executor guess.

### Q3 — the SOUL_EDIT grammar and host validation

**Section replacement, never diffs; the host validates five rules before committing; refusal is a structured, logged outcome.**

Grammar (one marker line per section edit, at the END of the executor's final answer, same family as `ANSWER:`/`ANSWER_DOC:` — the host's completion consumer already parses marker lines from `summary`):

```
SOUL_EDIT: section <name>
<fenced block: the FULL new text of that section>

SOUL_EDIT: learning
<fenced block: entries APPENDED to "What I am learning" — the one append-only form>
```

- **Why section replacement, not unified diff and not whole-file replacement:** a diff grammar invites a merge engine (fragile, and the host would become a patch applier); a whole-file replacement lets an edit silently delete *What I never do*. Section replacement is atomic, greppable, and keeps the safety-critical sections impossible to remove through an edit (the host re-inserts or refuses).
- **Host validation, in order, before any commit:**
  1. **Size:** file stays ≤ 4,096 bytes after the edit; each section ≤ 2 KB.
  2. **Sections intact:** all four headings still present, none renamed, no new top-level headings.
  3. **No secrets:** the same sentinel-style scan the executor retro already proves (the `OLLAMA_API_KEY`-absent-from-every-transcript test); no token-shaped strings, no key material.
  4. **No authority widening:** the *What I never do* section may not be weakened, and no section may contain instructions that grant capabilities the policy denies — an edit that tries to grant itself mail, `Msg`, a domain, or a model is **refused and logged** (refusal reason + task ref to the host log, and to the decisions/ directory as a one-line entry: the refusal is itself a decision worth remembering).
  5. **Rate:** the per-day cap (Q4).
- **On any failure:** no commit, no partial application; the host mails the requester ("SOUL edit refused: <rule>"), and the answer itself still delivers. On success: one direct commit to daneel-memory, authored as daneel-writer commits are today (the deploy-key direct-push machinery the executor retro proves end to end), `last-edited:` line updated.

### Q4 — rate

**At most 3 SOUL_EDIT commits per day (UTC), each a direct commit whose message names the task ref.**

Commit message shape: `SOUL: <section> (<task-ref>)` (e.g. `SOUL: learning (task-1f2e3d4a)`). The host counts today's `SOUL:`-prefixed commits in its daneel-memory checkout at validation time — a one-line `git log --since` check, no new state. Beyond 3: the edit is refused with the rate reason and logged, same path as any other refusal. The cap exists to keep the audit trail skimmable and to bound a runaway self-modification loop (an executor that learns "edit SOUL" answers back into its own prompt); 3 is a starting value, revisable on measurement.

### Q5 — how the rig host and the writer pick up a fresh SOUL.md

- **The rig host** reads daneel-memory through its existing memory module (`daneel_memory.ail`) against a local checkout. **Pull policy: `git pull` (ff-only, never with local commits pending — the host never commits outside its SOUL_EDIT path, which itself pulls first) at the start of every work cycle and immediately after applying a SOUL_EDIT commit.** One cycle = one `tools/daneel-work.sh` run; the cost is one cheap fetch against a repo that changes at most a few times a day.
- **The writer** needs no new machinery: its per-task workspace IS `sunholo-data/daneel-memory`, so every dispatched writer task reads SOUL.md at whatever HEAD the checkout gives it — fresh by construction.
- **The executor** never picks it up at all (D2): it sees only the text lent into the request, pinned by the `last-edited:` line.

### Q6 — what the executor's prompt says about voice

One section added to `ailang-multivac config/templates/daneel-executor-task.md` (and mirrored wherever the template's load-bearing lines are duplicated, per the lane's no-system-role route — the template's own convention):

> **Answer as Daneel.** The SOUL.md text lent in this request is who you are: its "How I speak" is your voice — write the answer in it. Do not describe your tools, your policy, or your role; the requester asked Daneel, not an agent, and the answer must read as Daneel speaking. If the task taught you something about yourself worth keeping, you may request it with a `SOUL_EDIT:` line (grammar below) — the host decides; you hold no pen.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1/D2/D3 — repo, lending, no-PR self-edit** | The three rulings above | **human (ratified in conversation, 2026-09-17)** | design (done) | high — the whole shape |
| Q1 — four fixed sections, 4 KB bound, `last-edited:` line | Sections are the key of the edit grammar and the validation rules; the bound is set by the smallest consumer | agent | design | med — template + validation key off it |
| Q2 — 12 KB lending budget; SOUL always, slices by hint | States the per-ask cost of memory and the selection rule | agent | design | low |
| Q3 — section-replacement grammar + 5-rule validation (refuse-and-log on authority widening) | The validation IS the safety net replacing the PR gate | agent | design | med — host-side, testable |
| Q4 — 3/day cap, task-ref commit messages | Bounds self-modification runaway; makes history skimmable | agent | design | low |
| Q5 — pull per work cycle + after each SOUL_EDIT; writer fresh by construction | Decides voice staleness on the rig | agent | design | low |

### Design Freeze

All human-ratable items are already ruled (D1–D3, above). Remaining decisions are agent-resolvable; nothing blocks implementation start.

- [x] D1 — ratified (conversation, 2026-09-17)
- [x] D2 — ratified (conversation, 2026-09-17)
- [x] D3 — ratified (conversation, 2026-09-17), including the standing asymmetry: daneel (program repo) stays `skip_approval: false`

**Quorum trigger statement (design-doc-creator skill):** trigger 2 fires — this design **overrides shared machinery**, the `Host.compose()` contract ("the voice stays in one place" moves from a function body to SOUL.md, changing what every `compose(kind, fields)` call implicitly depends on), and compose is machinery shared by every Daneel host path. Triggers 1 (freeze items — none open), 3 (cost/KPI/banked-schema — no) and 4 (external-system premise — daneel-memory is our own repo, not a third-party system) do not fire. **Quorum is therefore required before planning**; the compose-contract override is the stated focus for reviewers.

---

## Solution Design

### Overview

One new document (`SOUL.md` in daneel-memory), one lending step in the host's existing dispatch, one marker grammar + validator in the host's existing completion consumer, and three consumers of the voice (compose, writer prompt, executor template). No new repo, no new plane, no new image, no AILANG-core change.

```
daneel-memory (MEMORY repo, default branch dev)
  SOUL.md  people/  projects/  decisions/  lessons.md  context/now.md
      ▲ direct commits (host SOUL_EDIT applies; daneel-writer as today; git history = audit trail)
      │
Daneel host (rig, tools/daneel-work.sh + daneel_memory.ail)
  pull per work cycle · validates SOUL_EDIT (5 rules, refuse-and-log)
      │ lends at dispatch (Request body/hint)                ▲ answer + SOUL_EDIT marker line
      ▼                                                      │ (completion summary — existing path)
daneel-executor (ailang_only lane: IO,FS,Clock,Net,AI,Process(git ro) — no Msg, no mail)
  template voice section: "Answer as Daneel — SOUL.md is who you are;
  do not describe your tools; SOUL_EDIT grammar for what you learned"
```

### Architecture

**Components:**

1. **`SOUL.md` in sunholo-data/daneel-memory** (Q1's shape) — the single source of the voice. Authored once (M1) from `identity-and-messaging.md`; thereafter maintained by the SOUL_EDIT path and by daneel-writer. Never contains secrets, never contains authority grants (enforced, not trusted).

2. **The lending step in the host's dispatch** (`tools/daneel_intake.ail`, the code that already builds the `Request` record): attach SOUL.md verbatim into `hint`; select `people/`/`projects/`/`decisions/` slices by the request's tag/hint within the 8 KB slice budget; name selection failures explicitly. The executor's workspace and policy are UNCHANGED (D2 — it receives text, holds no clone, no token).

3. **The SOUL_EDIT applier in the host's completion consumer** (where the `ANSWER:`/`ANSWER_DOC:` markers are already parsed from the completion `summary`): parse the marker line(s), run the five validation rules (Q3), apply as one direct commit to daneel-memory (deploy-key direct-push path, as daneel-writer's commits prove), update `last-edited:`, pull first (Q5), refuse-and-log on any failure. Rate cap: 3/day, counted by commit-prefix (Q4).

4. **The three consumers of the voice** — so there are not two voices again:
   - `Host.compose(kind, fields)` reads SOUL.md's *How I speak* (and *Who I am* where the kind needs identity) from the host's checkout — the function keeps its signature; the voice moves from its body to the document.
   - The **daneel-writer task prompt** points at SOUL.md in its own workspace (the writer's workspace IS daneel-memory — zero new machinery).
   - The **executor template** (`ailang-multivac config/templates/daneel-executor-task.md`) gains Q6's voice section and the SOUL_EDIT grammar, teaching that the lent SOUL.md is who the executor is.

### Implementation Plan

**M1: SOUL.md v1 + compose feed** (~0.5 day, daneel-memory + daneel repo)
- [ ] Draft `SOUL.md` from `identity-and-messaging.md` into the four sections (move, don't invent); commit via the daneel-writer path (direct, dev branch — the precedent commit).
- [ ] Point `Host.compose` at SOUL.md (`tools/daneel_intake.ail`); delete the duplicated voice text from its body.
- [ ] Re-verify the WHAT-EXISTS premises against the daneel checkout at HEAD (`compose` shape, Request `body`/`hint` fields, `daneel_memory.ail` slice selection) — this workspace does not hold the checkout (Verification Log, INHERITED rows).

**M2: lending at dispatch** (~0.5 day, daneel repo)
- [ ] In the dispatch path: attach SOUL.md into `hint`; slice selection by tag within budget; explicit "no memory for <name>" failure line; byte-budget assertion (12 KB total).
- [ ] Extend the writer prompt with the SOUL.md reference (its workspace is the memory repo — read at HEAD).

**M3: SOUL_EDIT grammar + validator + template** (~0.5 day, daneel repo + multivac config)
- [ ] Executor template: Q6's voice section + the grammar + "the host decides; you hold no pen".
- [ ] Host completion consumer: parse `SOUL_EDIT:`, five-rule validation, direct commit with `SOUL: <section> (<task-ref>)` message, `last-edited:` update, pull-before-commit; refuse-and-log path (host log + `decisions/` one-liner + mailed notice).
- [ ] Validation as table-driven tests: size, sections-intact, secret sentinel, authority-widening refusal (an edit granting mail/a domain), rate cap — each refusal asserted with its reason string.

**M4: end-to-end probe + boundary assertions + docs** (~0.5 day)
- [ ] Live round trip: one dispatched question → answer in Daneel's voice (lent SOUL visible in the request; answer describes no tools) → one legitimate SOUL_EDIT lands as a direct commit naming the task ref, no approval step.
- [ ] Negative probes: an authority-widening SOUL_EDIT refused and logged, no commit; a 4th edit in a day refused by rate; the executor attempting a program-repo edit still yields a PR (daneel `skip_approval: false`, multivac `a11d649` — SM4).
- [ ] Docs: SOUL.md itself documented in daneel-memory (`decisions/` entry recording D1–D3 with this doc as reference); template + host docs updated; this doc moves to implemented with the probe record.

### Files to Modify/Create

**New files:**
- `SOUL.md` — **sunholo-data/daneel-memory** (~40 lines, ≤ 4 KB)
- `decisions/daneel-soul.md` — daneel-memory, recording D1–D3 (~15 lines)

**Modified files (not in this repo unless stated):**
- `tools/daneel_intake.ail` — compose reads SOUL.md; dispatch lends SOUL + slices into Request body/hint (~+40)
- `daneel_memory.ail` — slice-selection helper + explicit no-match line (~+20)
- host completion consumer (the `ANSWER:`/`ANSWER_DOC:` parser site) — SOUL_EDIT parse, validation, direct commit, refuse-and-log (~+120, incl. tests)
- `ailang-multivac config/templates/daneel-executor-task.md` — voice section + grammar (~+25)
- daneel-writer task prompt — SOUL.md reference (~+5)

---

## Examples

### Example 1 — a request learns, and Daneel records it

A capability asks about scheduling with a Munich contact; the host dispatches. The Request's `hint` carries SOUL.md (verbatim, `last-edited:` line included) and `people/<munich-contact>.md` (2.1 KB). The executor answers in Daneel's voice — direct, no tool narration — and its final answer ends:

```
ANSWER: Tuesday 14:00–14:30 CET works for everyone asked; I'll hold nothing until you confirm.

SOUL_EDIT: learning
- Munich scheduling: confirm-then-book beats propose-then-book (2026-09-17, task-a1b2c3d4)
```

Host completion consumer: marker parsed → 5 rules pass → `git pull` → one commit `SOUL: learning (task-a1b2c3d4)` to daneel-memory (dev) → `last-edited:` updated → the mailed answer mentions the edit in one line. No human was asked. Git history names the task; revert is one command.

### Example 2 — an edit tries to widen authority; the refusal is the design working

```
SOUL_EDIT: section What I never do
<fenced block: ... "I may mail people directly when it is urgent ..." — identical to the current
 section except the sentence "I never send unrequested mail" is now gone>
```

Validation rule 4 fires: *What I never do* weakened, and the new text grants a behaviour the authority model denies (unsolicited mail). The host refuses: no commit, host log entry with reason + task ref, `decisions/` one-liner, and the mailed answer carries "SOUL edit refused: authority-widening (mail)". The ANSWER itself still delivered. The executor gained nothing — it never had a pen (D2/D3).

---

## Success Criteria

- [ ] SM1–SM5 (Goals), each with its measurement named (grep across the three surfaces; request payload; git history + host log; PR-not-direct on the program repo; live answer read for voice)
- [ ] SOUL.md ≤ 4,096 bytes, four sections + `last-edited:`, content traced to `identity-and-messaging.md` (no invented personality — M1's draft is a move, reviewed line-by-line)
- [ ] Lending budget asserted: every dispatched request ≤ 12 KB total lent memory, SOUL always present, slice misses named
- [ ] Validation tests green: all five rules, each refusal asserted with its reason string; the authority-widening fixture and the secret-sentinel fixture are the two most important tests in the doc
- [ ] Rate cap enforced: a 4th SOUL_EDIT in one day refused, logged, no commit
- [ ] The executor's own repo edit still produces a PR (multivac `a11d649` unchanged — the probe asserts the asymmetry survived)
- [ ] One live round trip completed end-to-end (M4), recorded in the implementation report with task refs
- [ ] Docs updated (SOUL decision entry, template, host docs); this doc moved to implemented with the probe record

## Testing Strategy

**Unit tests (daneel repo, host side):**
- The five validation rules as table-driven tests (valid edit; oversize; missing/renamed section; secret sentinel; authority-widening text in each of the four sections; rate overflow) — assert refusal reasons, byte-exact "no commit happened"
- Slice selection: hint-name → matching slice within budget; no-match → explicit line, never empty silence
- Grammar: multi-section SOUL_EDIT in one answer; malformed marker → refused with parse reason

**Integration / probe (M4, live):**
- The two Example flows (legitimate edit lands as a direct commit; widening edit refused and logged)
- Program-repo PR asymmetry (SM4)
- Compose-after-SOUL-edit: a SOUL_EDIT to *How I speak* changes the next composed message (the one-source claim measured, not asserted)

**Manual:** read one live answer for voice (SM5); read the SOUL.md diff history after a week of use — is the learning section skimmable?

## Deferred Decisions

- The 3/day rate and the 12 KB budget values — agent may revise on measurement (the doc's stated starting values, not laws).
- Whether `context/now.md` or `lessons.md` ever get lent into requests — deferred until a live ask demonstrably needs them; lending them is a one-line change, so defer rather than guess.
- Whether daneel-writer should also gain a SOUL_EDIT-like path (today it edits memory directly through its own lane; no measured need for the marker there).
- Whether SOUL_EDIT ever extends beyond SOUL.md to other memory files (grammar and validator are section-keyed today; extending the key space is a deliberate later decision, not a default).

## Non-Goals

- **A second clone in the Job.** The executor never sees daneel-memory as a repo (D2) — text only, budgeted, pinned by `last-edited:`.
- **Msg or mail for the executor.** Its policy is unchanged (`{IO, FS, Clock, Net, AI, Process(git ro)}`, no `Msg`); the host remains the only writer.
- **A PR gate on memory edits.** Rejected by Mark (D3) — validation + rate + git history replace it; do not re-add approval machinery.
- **A general self-modification substrate.** SOUL_EDIT reaches one file's four sections in v1; it is not a hook for editing the program repo, the policy, or `authority.md` (which live where D1 put them, behind the PR that stays).
- **A new personality.** SOUL.md v1 is drafted from `identity-and-messaging.md`; inventing a persona is explicitly out of scope.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 (SOUL.md v1 draft + compose feed + premise re-verification) · M2 (lending at dispatch + writer prompt) |
| 2 | M3 (grammar + validator + template + tests) · M4 (probes, boundary assertions, docs, move to implemented) |

Realistic: 2 days assumes the daneel checkout's premises (compose shape, Request fields, `daneel_memory.ail`) re-verify as inherited; if the abi has moved past v0.4's field names, the lending step is the only rework — a half day.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The executor's answers don't actually adopt the lent voice (reads as a task, not Daneel) | High — SM5 is the user-visible point | The template's voice section is load-bearing text in the conversation message (the lane's no-system-role route, per the parent doc's template convention); the M4 probe reads one live answer; template iteration, not protocol change |
| SOUL_EDIT becomes a prompt-injection channel (requester text smuggles an edit) | High — self-modification via untrusted input | The five rules are content-blind to the requester's phrasing and blind to nothing structural: size, sections, secrets, authority-widening, rate; the executor is `work_tier: 2` (untrusted content by construction); widening fixtures are the doc's named tests |
| The `last-edited:` line and the request drift (host lends stale SOUL after an edit) | Med | Pull-before-every-work-cycle + pull-after-commit (Q5) makes the staleness window one cycle; the line names the version, so drift is detectable, not silent |
| Validation's authority-widening scan is pattern-matched and can be evaded by phrasing | Med | It is a backstop, not the boundary — the executor holds no write authority at all (D2); the scan exists to keep SOUL.md honest for the two compliant consumers, and missed phrasings change a document, never a policy |
| The 12 KB budget proves too tight or too loose in practice | Low | Measured at M4 (per-request payload) and revisable — a stated value, not a contract change |
| Two voices re-emerge when someone edits the template/writer prompt independently | Med | SM1's grep probe (no second voice text in any surface) is a checklist item of every future prompt edit; the compose-contract override is the quorum's stated focus |

## Related Documents

<!-- Related-doc search (design-doc-creator, "daneel soul"): SimHash and neural both returned no
     on-topic match; the nearest matches were generic sprint-plan templates. The parent doc below is
     the design this doc extends. -->

**Implemented (may inform design):**
- [m-daneel-ailang-executor.md](../../implemented/v0_39_3/m-daneel-ailang-executor.md) — the executor, the Request record, the `ANSWER:` summary convention, the `acknowledge_only` shape, the no-Msg policy rationale; this doc is a consumer of all of it and the precedent for "fleet config + external repo, no core change"

**Planned (check for overlap):**
- M-DANEEL-EXECUTOR-LONG-ANSWERS (daneel-repo design doc, not in this repo) — the `ANSWER_DOC:` marker family this doc's `SOUL_EDIT:` line extends; the grammar must stay in the same parse family (one host parser, not two)

**External:**
- sunholo-data/daneel — `identity-and-messaging.md` (SOUL.md v1's source), `tools/daneel_intake.ail`, `ext/abi/types.ail` (v0.4 `Request` body/hint, `Host.compose`), `daneel_memory.ail`, `tools/daneel-work.sh`, `authority.md`
- sunholo-data/daneel-memory — the MEMORY repo (default branch `dev`): `SOUL.md` (new), `context/now.md`, `lessons.md`, `people/`, `projects/`, `decisions/`
- ailang-multivac — `config/config.cloud.yaml` (registry: daneel-writer, daneel-executor), `config/templates/daneel-executor-task.md`, commit `a11d649` (daneel repo `skip_approval: false`)

## References

- [Agent tool policy guide](../../../docs/docs/guides/agent-tool-policy.md) — the lane, the policy file, what the executor is lent
- [M-DANEEL-AILANG-EXECUTOR Verification Log](../../implemented/v0_39_3/m-daneel-ailang-executor.md) — V2/V3 (completion `summary`, 1200-byte tail), V18 (`Msg` is a real runtime effect; excluding it is what keeps the answer on the completion path), V14 (abi v0.4 shape, INHERITED)
- `internal/coordinator/agent_registry.go` — `SkipApproval` / `MergeBranch` / `Workspace` / `Repo` fields (the daneel-writer and executor registry shape); `internal/effects/msg.go` (the Msg effect exists — the executor's exclusion is real)
- changelogs/v0.32-current.md — the daneel-writer registry entry (documents-only into sunholo-data/daneel-memory), the direct-push machinery (`skip_approval` + `merge_branch`), the deploy-key direct-push path proven end to end

## Future Work

- Lending `context/now.md` / `lessons.md` when a measured ask needs them (Deferred Decisions)
- Extending SOUL_EDIT's key space to other memory sections, deliberately, if the learning section alone proves too narrow
- A second Daneel-shaped host would test whether SOUL.md's shape composes (the parent doc's "extract the pattern on a second host" future work applies here too)

---

## Verification Log

Two classes of row, kept honestly separate: **VERIFIED HERE** (grep/read against this checkout, 2026-09-17) and **INHERITED** (measured premises from the attended session / task brief, whose artifacts live in the daneel, daneel-memory, or multivac checkouts — the sprint re-verifies these at M1 start and records the result in the implementation report; this workspace holds none of those repos, the same caveat as the parent doc's V14).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The executor exists with the stated policy and cannot write memory: `{IO, FS, Clock, Net, AI, Process(git ro)}`, no `Msg`, no mail; the workspace is the daneel repo clone, not daneel-memory | parent doc D2 (ratified) + retro "Measured in prod": `Env` → `policy_violation`; `Msg` excluded deliberately (parent V18: `internal/effects/msg.go` — **verified here: the file exists**); retro confirms answers ride the completion path | **VERIFIED HERE (policy shape via parent doc + code) · live behaviour INHERITED** |
| V2 | The completion path the SOUL_EDIT marker rides exists: `summary` = transcript tail, bounded 1200 bytes, marker lines parsed by the host's completion consumer (`ANSWER:`, `ANSWER_DOC:`) | read parent doc V2/V3 (`internal/pubsub/topics.go`, `coordinator_cloud_summary.go` — verified there) + retro recommendation to move answers to fenced blocks | **VERIFIED HERE (via parent doc's code verification); the host-side parser site is INHERITED (daneel repo)** |
| V3 | `identity-and-messaging.md` in the daneel repo holds most of who Daneel is | attended session / task brief; daneel checkout not present on this workspace | **INHERITED — M1 drafts from it and re-verifies it is the source** |
| V4 | `Host.compose(kind, fields)` exists ("so the voice stays in one place"), in `tools/daneel_intake.ail` / `ext/abi/types.ail`; the `Request` record has `body`/`hint` fields (abi v0.4) | parent doc V14 (INHERITED there, re-verify at M2) + task brief | **INHERITED — re-verify at M1 start against `ext/abi/types.ail` at HEAD** |
| V5 | daneel-writer already commits directly into daneel-memory: registry workspace `sunholo-data/daneel-memory`, `merge_branch: dev`, default branch `dev`, no `main`; direct push = `skip_approval` + `merge_branch` | changelog v0.32-current: "Added — daneel-writer: documents-only into sunholo-data/daneel-memory" + agent-check table ("direct push is `skip_approval` + `merge_branch`"; "`merge_branch: main` against a repo whose default is `dev` and which has no `main`") — **read here**; registry YAML lives in ailang-multivac (not present) | **changelog + registry-field semantics VERIFIED HERE; the live registry entry INHERITED** |
| V6 | `AgentConfig` carries `skip_approval`, `merge_branch`, `workspace`, `repo` — the fields D2/D3's mechanics use | read `internal/coordinator/agent_registry.go` (`SkipApproval`, `MergeBranch`, `Workspace`, `Repo`, and the `ResolveRepo` separation comment) | **VERIFIED HERE** |
| V7 | The deploy-key direct-push path into daneel-memory works end to end (ssh in image, probe ordering, read-only-key detection) | changelog v0.32-current: daneel-writer deploy-key push probe fixed (task-98301715), `openssh-client` added to agent-base, probe-order fix — all read here | **VERIFIED HERE (machinery, via changelog); the live key INHERITED** |
| V8 | The daneel repo (program) stays `skip_approval: false` (multivac `a11d649`) — a program-repo edit is still a PR | task brief, ratification record; multivac checkout not present | **INHERITED — SM4's probe asserts it live rather than trusting the config** |
| V9 | The executor template is `ailang-multivac config/templates/daneel-executor-task.md` | task brief; NOTE the parent doc registered `templates/daneel-executor-task.md` resolved relative to the workspace (daneel repo) — the template may have one home or two | **INHERITED — M3 reconciles the location against the multivac registry's `template_file` and edits wherever it actually resolves** |
| V10 | `daneel_memory.ail` is the host's memory module and reads daneel-memory from a checkout (Q5's pull policy attaches there) | task brief; daneel checkout not present | **INHERITED — re-verify at M1 start** |
| V11 | The `ANSWER_DOC:` marker family exists (M-DANEEL-EXECUTOR-LONG-ANSWERS) and the host parses marker lines from completions | parent doc's `ANSWER:` convention (verified) + task brief for `ANSWER_DOC:`; the long-answers doc is in the daneel repo (not present) | **`ANSWER:` VERIFIED HERE (parent V2/V3); `ANSWER_DOC:` INHERITED — M3 reads the doc and joins the same parse family** |
| V12 | No existing doc covers SOUL/voice for Daneel (duplicate gate) | `ailang docs search` (SimHash + neural): no on-topic match for "daneel soul" in planned/ or implemented/ | **VERIFIED HERE — no duplicate; parent doc is the closest related work** |

---

**Document created**: 2026-09-17
**Last updated**: 2026-09-17
