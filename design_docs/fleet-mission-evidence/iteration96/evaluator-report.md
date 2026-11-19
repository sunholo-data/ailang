# Fleet iteration 96 — independent evaluator report (report-only evidence & authorization review)

## 1. Judge identity, model, independence

| Field | Value |
|---|---|
| Role | **evaluator** (independent judge), `MISSION-ROLE: evaluator` handshake completed inside this session |
| Model / lane | `pi:openrouter/minimax/minimax-m3` — this session's declared evaluator lane (`MISSION_EVALUATOR_MODEL` in the session environment); pi `0.85.1` |
| Lane probe | `design_docs/fleet-mission-evidence/iteration96/evaluator-probe-usage.json`: same lane, rc **0**, 1.76 s, $0.0007557 → the iteration-95 resume predicate ("re-probe the driver-selected declared independent evaluator lane next fire") is **satisfied** |
| Generator ≠ judge | ✔ cross-vendor: executor/planner default is `codex:gpt-6.1-sol` (OpenAI) per `design_docs/fleet-mission.md` § Routing policy; the judge is OpenRouter/MiniMax. The Anthropic lanes were unavailable at iteration 95 (sonnet and remaining opus probes rc1/HTTP429, weekly reset 2026-11-09 06:00 Europe/Copenhagen — iteration-95 STATUS stamp) |
| Worktree | `/Users/voightkampff/.ailang-worktrees/fleet-i96` |
| Reviewed HEAD | `2249a1923a02be2e4f3ce2f1dcb96239484a237b` (= draft PR #2169 head `fleet/i95-record`, unmerged — `current-external-capture.json`) |
| origin/dev at review | `252680e1beb4e6cf1709a010cc1c45838b91b9f8` — does **not** yet carry the iteration-95 STATUS stamp (its newest is ITERATION 94) |
| Supplied context | Controller triage only. **No controller candidate, conclusion or recommended answer was supplied.** No iteration-95/96 designer / planner / executor conclusion artifact was read before this assessment; findings below are re-derived from first-party artifacts |
| Mutations by me | exactly two new files: this report and `evaluator-read-audit.json`. No source, config, threshold, lane, decision-status or ticket-state change; no git write operation; no `ailang messages`; no `gh` mutation |

## 2. Task classification (explicit)

**Independent report-only mission evidence/authorization review.** This fire supplies **no implementation sprint, no approved design, no sprint plan and no production diff**. That is a fact of the fire, not a missing artifact on someone's part, and nothing here implies a sprint exists that should be executed.

Consequences, stated up front:

- **Implementation score: N/A.** The 100-point rubric in `.claude/skills/sprint-evaluator/resources/scoring_rubric.md` scores an implementation against an approved design + sprint plan (tests pass, lint, acceptance criteria, code quality, docs, design fidelity). With no implementation and no contract to score against, **no rubric score is awarded** — awarding a number would be the "arbitrary rubric score for no implementation" the task forbids.
- The sprint-evaluator's `EVALUATION_SCORE` / `EVALUATION_ROUND` markers are therefore **not emitted**; only the typed verdict marker at the end of this report.
- This report does **not** accept, re-accept or re-judge any previous draft record (#2065, #2067, #2082, #2108, #2110, #2141, #2143, #2150, #2165). Each is named only as *existing state* in the row screen. The iteration-95 record itself is a **draft awaiting review** (its own STATUS stamp: "no judge verdict … exists"); this report does not supply a retroactive verdict for iteration 95.

## 3. Protocol handshake — own evidence (done before any judge work or write)

1. **CLAUDE.md read in full** in the evaluator worktree (first action, before any other tool use).
2. **Classified** as an unattended mission stage → the inbox step was **skipped** per CLAUDE.md § "Session start (attended)" ("Unattended … there is no user: skip the inbox and do the task") and per the gate-3 `PI EVALUATOR SESSION HANDSHAKE` block ("The evaluator does NOT check the inbox: inbox triage is the controller's Gate 0"). `ailang messages` was never run.
3. **`session_protocol_ack` called with `{}`.** Tool result, verbatim:
   > `Session protocol acknowledged — mutating tools unlocked for this session. Work routing per AGENTS.md still applies.`

   **Honest note on the `acked=true` check:** the tool's returned text does **not** print a literal `acked=true` token. The observable success signal is (a) the acknowledgement sentence quoted above and (b) the mutating tools becoming available — the first write in this session succeeded and `git status --porcelain` shows only the controller's untracked evidence directory before my own two files were added. I record what the tool actually returned; I do **not** claim a literal `acked=true` string that was never printed.
4. Only after that success: the required full reads, the independent assessment, and these two writes.

**No prerequisite failed and no transport failure occurred**, so this attempt is a genuine judge report, not a fallback-route transport notice.

## 4. The ticket under review — what the artifacts actually show

Ticket: `tool-hang:triage-judge:home-metadata-search` (mission `triage`, triage iteration 33, filed 2026-11-07T12:34:46Z, `slots_lost=1`, `blocking=none`).

### 4.1 Facts (first-party, cited)

| Fact | Source |
|---|---|
| Queue now holds **26** open tickets; `new_signatures: []` vs iteration 95 (no newly filed ticket this fire); `worktree_head` = `2249a192…`; `origin_dev` = `252680e1…` | `iteration96/input-manifest.json` (ticket_count 26, prior_ticket_count 26) |
| The tool-hang ticket is the **newest** ticket in the queue by `first_seen` (2026-11-07 vs the next-newest 2026-10-28) | `iteration96/evaluator-input-tickets.json`, 26 objects, parsed and sorted |
| Attempt 1: `verdict: tool_hang`, **rc 18**, `fenced: true`, model `ollama/minimax-m3:cloud`, `elapsed_seconds: 1119`, `tool_executions: 148`, `provider_errors: 0`, `commits_since_start: 0`, `worktree_changed_since_start: false`, `worktree_changed_files: 1`, `predirty_files: 1` | `iteration95/new-ticket-first-attempt.json` |
| The hung tool was one bash command that printed `$HOME`, listed `$HOME/.pi`, and ran `find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin -120 -print 2>/dev/null \| head -100` | same file, `hung_tool` |
| The open tool was observed under the **existing** runner guard; no threshold was changed | `iteration95/new-ticket-open-tool.json`: `state: "open-tool-under-existing-runner-guard"`, `observed_no_progress_seconds: 152` |
| Attempt 2 (the remedy the skill already prescribes): `verdict: ok`, **rc 0**, 274 s, 75 tool executions, `provider_errors: 0`, `hung_tool: ""`, `agent_end_events: 1` | `iteration95/new-ticket-retry.json` |
| The runner classifies an **open tool at the stall boundary** as `tool_hang`, and maps it to rc 18 | `scripts/mission_pi_run.sh:418` (`OUTCOME="tool_hang"`), `:453` (`tool_hang) VERDICT_NAME="tool_hang"; RC=18`), `:486` (stderr says "the MODEL did not stall — pi was waiting on a tool call that never returned") |
| The shared recipe already says rc 18 is **not a lane failure** and already prescribes the remedy | `.claude/skills/mission-control/resources/gate-3-route.md:476-478` — "`tool_hang` (rc 18) is **not a lane failure at all** — the model was fine; a command it ran never returned (`hung_tool` in the verdict names it). Falling back just hangs the next lane on the same command. Retry ONCE on the same lane with the directive telling it not to run that command, and FLAG the command as a harness ticket." (identical clause in the `.agents` mirror — `grep -c` = 1 in each) |
| The runner's no-progress bound is **600 s** by default and is the bound that killed the run | `scripts/mission_pi_run.sh:89` (`STALL_SECONDS="${MISSION_PI_STALL_SECONDS:-600}"`), `:104` (documented default) |
| The shared **role** runner loads **no per-command cap**: its extension list is `${PI_EXT_ARGS[@]}` (the workdir's own `.pi/extensions/*.ts`) + the sandbox + the worktree fence | `scripts/mission_pi_run.sh:288` and `:302` |
| `_mc_pi_ext_args` emits only `--no-extensions` plus the workdir's `.pi/extensions/*.ts` — **no** bash-cap extension | `tools/launchd/lib/pi-ext-args.sh` (full read) |
| The two sibling lanes **do** carry a per-command cap | controller: `tools/launchd/mission-control.sh:2578` loads `-e "$MC_DRIVER_ROOT/tools/pi-extensions/controller-bash-cap.ts"` (wiring pinned by `tools/launchd/test_driver_notify.sh:1003-1005`); eval harness: `tools/pi-extensions/eval-shell-guard.ts` (`AILANG_EVAL_BASH_CAP_S`, default 120) |
| pi's bash tool has **no default timeout** | `tools/pi-extensions/controller-bash-cap.ts` header and `tools/pi-extensions/eval-shell-guard.ts` header ("pi's bash tool has NO default timeout and spawns every command detached") |
| The mission sandbox policy denies reads only for credential-shaped paths, **not** the home tree | `tools/pi-extensions/sandbox/sandbox.mission.json`: `denyRead: ["~/.ssh","~/.aws","~/.gnupg","~/.config/ailang/secrets.env","~/.pi/agent/models.json"]` |
| The eval harness's *refusal* rule would **not** have caught this command | `tools/pi-extensions/eval-shell-guard.ts`: `isUnboundedFind` returns `rooted && !/-maxdepth\s+\d/.test(cmd)` — the hung command **has** `-maxdepth 4`, so it is not refused |
| The same failure **class** has cost runs three times across three missions | `stall:watchdog:slow-on-hung-tool` ticket evidence: stapledon 2026-09-28 "an unbounded `find ~ -maxdepth 6` that never returned"; fleet 2026-10-04 "pi … made NO PROGRESS across 5 samples (600s)"; plus triage 2026-11-07 (this ticket). Recorded as D-FLEET-34 |
| The mission's own charter calls this class a defect that "comes back as a fix" | `design_docs/fleet-mission.md` § North star: "Every loop-harness defect they hit leaves them as a ticket and comes back as a fix." |
| The fleet may change the shared runner | `design_docs/fleet-mission.md` § Authority allowlist: `scripts/mission_*`, `scripts/test_mission_*` (D-FLEET-5), `.claude/skills/mission-*/**` + `.agents/skills/mission-*/**`, `tools/launchd/**`, and `tools/pi-extensions/sandbox/**` (D-FLEET-6) |

### 4.2 What the evidence **supports**

1. **The runner was correct and honest.** rc 18 `tool_hang` is the right verdict: zero provider errors, no commits, no report, no item actions from attempt 1. This was **not** a false green — the attempt produced no judge verdict and the ticket records that.
2. **The prescribed remedy worked**: one same-lane retry that forbids the named command returned rc 0 in 274 s.
3. **A specific, unaddressed gap exists, and it is not the model's behaviour alone.** The gap is *where the control lives*: the controller lane and the eval lane each have a per-command bash ceiling; the **shared role runner** (executor / planner / evaluator, used by every mission) has none, so a single non-returning command consumes the whole slot and the run dies with no verdict. In this instance that cost 1119 s and a complete judge report.
4. **A fix exists in the fleet's own allowlist.** The runner (`scripts/mission_pi_run.sh`) is explicitly fleet-owned (`scripts/mission_*`; D-FLEET-5 also gives the fleet `scripts/test_mission_*` for its test surface), and `tools/pi-extensions/controller-bash-cap.ts` is **not** a mirrored asset (it appears in neither `.pi/extensions/` nor `internal/executor/pi/profile_assets/`, and `make pi-assets` mirrors only `.pi/extensions/**` and `eval-shell-guard.ts`), so wiring it **by reference, unmodified** creates no `make verify-pi-assets` drift.
5. **The class is recurring** (three occurrences, three missions, two of them already filed as `stall:watchdog:slow-on-hung-tool` / D-FLEET-34), and AGENTS.md's "friction twice" doctrine is satisfied.

### 4.3 What the evidence **does NOT support** (explicit negatives, with positive controls)

- It does **not** support "the runner mis-classified the hang", "a false green occurred", or "the retry proves the judge reviewed every item substantively". The verdict JSON is a **transport** record, not a content review: `new-ticket-retry.json` shows `worktree_changed_files: 2`, `agent_end_events: 1` — a report was produced; its *substance* was not verified by these artifacts and is not verifiable from them.
- It does **not** support a **recurrence rate**: N = 1 for the role-runner gap. The two earlier instances were **controller** runs, a different lane with a different control; they evidence the *class*, not the role-runner gap's frequency.
- It does **not** support any change to thresholds, retries, lanes, quota/ration, or billing. Nothing measured suggests the 600 s stall bound, the 1800 s wall bound, the one-retry rule, or the lane chain is wrong. The rc 18 classification and the retry rule were exercised **exactly as written** and worked.
- It does **not** support reproducing the hang, and I did not attempt it. No broad `find` over `$HOME`, no session discovery, no `.pi` metadata sweep, no credential-location probing, no raw environment dump was performed at any point in this review (the only `$HOME` references in this report are quoted from the ticket's own sanitized artifact).
- It does **not** support the claim that "the fix is one line". See 6.3 for the band that changes and the design obligation.

## 5. Independent disposition — the ticket's valid next step

**Category (a): a real authorized code/design proposal.** Not (b) — a bare evidence disposition would leave a harness defect unfixed, against the charter's own north star ("every loop-harness defect they hit leaves them as a ticket and comes back as a fix"). Not (c) — no attended ruling is required, because nothing policy-class changes. Not (d) — a pick exists and it is the only admissible queue item (7.1).

**The next step, exactly:** route the ticket to the normal inner loop (designer → planner → executor → judge) and build a **draft PR, unmerged**, whose content is bounded as follows.

### 5.1 P1 — the directive clause (mandatory, minimal, in-scope)

`.claude/skills/mission-control/resources/gate-3-route.md` **and** its `.agents` mirror (both in the fleet allowlist):

- In the pi **role** recipe (the one that writes the executor/planner/evaluator directive), add a standing clause: *a role directive must forbid broad filesystem, `$HOME`, session-store and metadata-tree searches; the artifacts the role needs are named in the directive; if something else is needed, the role reports it instead of sweeping the disk.*
- In the **rc 18** clause, keep the existing "retry ONCE … and FLAG the command as a harness ticket" wording and make the retry directive's "do not run that command" requirement **explicit as a MUST**, with this ticket's command shape as the worked (sanitized) example.
- Mirror parity must hold (`scripts/test_agents_skills_sync.sh` / the mission skills sync check in `make test-launchd-drivers`).

*Why P1 is the direct countermeasure:* the measured cause was the judge choosing an unnecessary search. The retry proved the instruction channel works (rc 0 in 274 s). P1 makes the instruction **standing** (prevention) instead of **reactive** (recovery).

### 5.2 P2 — a per-command ceiling for role runs (in-scope wiring; design obligation attaches)

`scripts/mission_pi_run.sh` (in scope):

- Load the existing, **unmodified** `tools/pi-extensions/controller-bash-cap.ts` into role runs, mirroring the controller spawn at `tools/launchd/mission-control.sh:2578`, and export `MISSION_PI_BASH_CAP_S` (the extension's own env knob — so the extension file itself needs no edit, and no mirrored-asset drift is created).
- **Cap must be strictly below the run's own no-progress bound** (`--stall-seconds`, default 600). This is the load-bearing design constraint, and it is what makes P2 safe (5.3).
- **Known cosmetic limit, stated rather than hidden:** `controller-bash-cap.ts` renders its guidance as "## Command time limit (mission **controller**)", so a role agent would see controller wording. The file is outside the fleet's scope (only `tools/pi-extensions/sandbox/**` is inside it, D-FLEET-6), so it **cannot** be reworded by the fleet; only the cap value (`MISSION_PI_BASH_CAP_S`) and the loading can be set from the in-scope runner. The directive clause P1 supplies the missing "do not sweep the disk" instruction, because `controller-bash-cap.ts` bounds a command but does not refuse one — unlike `eval-shell-guard.ts`, whose refusal rule would not have matched this command anyway (it has `-maxdepth 4`).
- Wiring only. **No** change to `--stall-seconds`, `--max-seconds`, the retry rule, the lane chain, quota/ration or billing.

### 5.3 Why P2 does not truncate anything the current bounds already kill (the arithmetic the design must verify)

The runner's progress clock is the filtered NDJSON's mtime (`scripts/mission_pi_run.sh` § progress clock), and a tool-call event refreshes it. So a bash command that runs longer than the stall window already kills the run today: at 1119 s the attempt was killed with the tool still open → `tool_hang` (rc 18). Therefore:

- Any command longer than the stall window is **already fatal**; a cap below the window converts a fatal outcome into a recoverable one (the command returns `timeout:<s>`, the model continues, the run can still produce a verdict). This is precisely the property `tools/pi-extensions/controller-bash-cap.ts` documents: "With a cap below the 600 s window, every command returns (result or `timeout:<s>`) … A real wedge still dies; a patient poll no longer does."
- **The band that does change behaviour** is `(cap, stall]` — commands in it currently return their output and would instead return `timeout:<s>`. The design must keep that band as small as it can justify and must show it against **measured** command durations in this repo (e.g. `cmd/ailang`'s own `go test` package was measured at 395 s against a 416 s budget in D-FLEET-17's evidence — a real 395 s command). If the design cannot bound that band, P2 must not ship and P1 stands alone.

### 5.4 What is **not** authorized by this ticket or by this evidence

- No modification of `tools/pi-extensions/**` outside `sandbox/**` (out of the fleet's scope — D-FLEET-6: "the rest of `tools/pi-extensions/**` stays outside its scope because the eval harness shares it"), of `.pi/extensions/**` (parks for Mark — D-FLEET-3), of `internal/executor/**`, or of `cmd/ailang/**` (outside the allowlist).
- No change to the mission sandbox policy's `denyRead` set: `tools/pi-extensions/sandbox/**` is inside the allowlist, but denying home-tree reads changes what **every** role in **every** mission may read, which is an attended decision → it needs a decision row, and it is explicitly **out of scope for this ticket**.
- No threshold / retry / lane / quota / ration / billing change; no new decision row is required for P1+P2 (nothing policy-class moves).
- No ticket resolution yet: the ticket closes only with `ailang mission ticket resolve <signature> --resolution … --sha <commit on origin/dev>` after the fix lands and the done-gate passes. Iteration 95's own triage note is right — "A recovered command does not authorize a controller-only ticket resolution."
- No finding that the rc 18 verdict or the retry rule is wrong; both behaved as specified.

### 5.5 Done-gate and reach (when the proposal is built)

The charter's five-line mission-loop-change pre-flight: the surface that actually runs was edited (`scripts/mission_pi_run.sh` + both skill copies); reach confirmed (push — every mission runs the pinned `origin/dev`); dry-run healthy **and** degraded; `make test-launchd-drivers` green; nothing reloaded mid-iteration. The fleet cannot dry-run itself mid-iteration → dry-run under an idle, armed sibling profile, or `MISSION_PROFILE=fleet` under a synthetic HOME. The proposal ships **alone** (guard-arm-first discipline does not apply here — there is no scope widening — but a shared-harness change to every mission's role runs should not be stacked with anything else).

## 6. Strongest objection to this disposition

> **"This is one occurrence of an agent running a command it did not need. The harness already has a documented remedy, and it worked: the retry returned rc 0 in 274 s. The repo already contains the right control for the two lanes where this was measured (controller-bash-cap.ts, eval-shell-guard.ts) — and the mission's own current direction on this exact class is to WAIVE the analogous ask (D-FLEET-34 recommends B: 'waive ask 2 and resolve the ticket'). You are proposing to change the shared runner that every mission's executor, planner and evaluator uses, on N=1, while the loop has twice decided the class did not need more. If you are wrong about the band, the cost lands on a mission's executor milestone, not on this ticket. The honest disposition is (b): record the evidence, the correct rc 18 handling, and the exact limits — and do not churn a shared harness for a self-inflicted command."**

**My answer, and why it does not move me:**

1. The objection's premise ("the harness already has a documented remedy") is true of the **recovery** path and false of the **prevention** path, and it is the prevention path that cost the slot: attempt 1 produced **no judge verdict at all** (0 item actions, no report). The remedy was exercised *after* the loss.
2. The objection's D-FLEET-34 precedent is weaker than it looks: that row's ask 2 is about **shortening the no-progress threshold** (which would re-open ruled D-FLEET-4/-7), and its ask 1 is a ghost of #1391 (the **controller** cap). The mission has never ruled on a role-run command bound — the gap is un-addressed, not waived.
3. The "one occurrence" count is 1 for the role-runner gap but 3 for the **class** across three missions (stapledon 2026-09-28, fleet 2026-10-04, triage 2026-11-07). AGENTS.md's "friction twice → fast-track the fix instead of re-solving it manually each session" applies.
4. The blast radius is bounded by arithmetic, not by hope: anything the current 600 s bound already kills is not truncated by a cap below it (§5.3); only the narrow `(cap, stall]` band changes, and the design must measure and justify it. If it cannot, P2 is dropped and P1 stands alone — the proposal is explicitly split so that the cheap, zero-risk half is not held hostage by the risky half.
5. **Where the objection lands and is adopted:** the "self-inflicted command" framing is fair about *this* incident and I do not claim the ticket proves a systemic defect. That is exactly why the recommendation is a **bounded, split, draft proposal with a stated band** rather than a sweeping harness change, and why the ticket is **not** closed by this report.

## 7. Authorization screen of the queue and of all 26 OPEN decision rows

### 7.1 Queue screen (26 open tickets) — why the tool-hang ticket is the pick

All 26 open tickets were read in full and screened for admissibility (in-scope **and** not policy-class). 25 are ruling-gated; exactly one is admissible:

| Rank (slots_lost) | Ticket | Admissible? | Why |
|---|---|---|---|
| 20 | `reporting:daneel-reports-not-declared-triage` | no | fix is a prod coordinator-config entry in the multivac repo (D-FLEET-22) — outside scope |
| 3 | `quorum:gpt6-1-sol-openai-api-credits-exhausted` | no | `cmd/ailang/design_quorum.go` (V1 lane) / billing (D-FLEET-20) |
| 2 | `messaging:pubsub-notify-wrong-project` | no | attended `gcloud` quota-project change or `internal/messaging` (D-FLEET-19) |
| 1 ×22 | `ci:launchd-suite-flakes…`, `eval:motoko-connection-probe-orphaned`, `policy:fleet-rulings…`, `codex:server-capacity…`, `quota:admission…`, `pi:unowned-dirty…`, `skill:design-doc-creator-grep-portability`, `skills:agents-copies-stale…`, `pi:openrouter:uncapped-max-tokens-402`, `stall:watchdog:slow-on-hung-tool`, `crash:sprint-executor:sigbus…`, `sprint-evaluator:worktree-cap-and-exit1`, `executor:3h-timeout…`, `executor:timeout-loses-work…`, `coordinator:cloud-lane-capped…`, `executor:codex-credential-lease-collision…`, `coordinator:job-retry-success-dropped…`, `executor:codex-credential-collision-drops…`, `ci:motoko-probe-refusing-live-path…`, `quorum:oc-glm-5-3-output-cap-invalid`, `quorum:claude-p-ration-usage-429`, `ci:distinct-build-test-failures` | no | each is behind an OPEN row whose recommended option is attended / out-of-scope / policy-class; three of them (`ci:launchd-suite…`, `ci:motoko-probe…`, `eval:motoko-connection-probe-orphaned`) already have the D-FLEET-18 guard arm drafted (#2110); `coordinator:cloud-lane-capped…` has **no** row, and gate-2-pick's "no capability without its authority row" forbids widening scope on a ticket that has no recommendation |
| 1 | **`tool-hang:triage-judge:home-metadata-search`** | **YES** | in-scope (shared runner `scripts/mission_pi_run.sh` + shared skill copies), not policy-class (no routing / lane order / quota / ration / billing change), harness-class per gate-2-pick ("a probe timeout" is listed as mechanical) — and the charter's north star says a harness defect "comes back as a fix" |

So the queue is **not** wholly ruling-gated, and the screen-by-step fallback into the decision ledger is **not** the primary path. The row screen below was still performed in full (the task requires it) and is reported for completeness.

### 7.2 Row screen — all 26 OPEN rows, screened by STEP

Test applied (charter): a row qualifies when **its recommended option**, or **a step shared by all its non-default options**, *begins* with an in-scope step that has **no diff yet**; **a completed draft does not qualify**.

| Row | Recommended option's first step | Draft exists? | Conclusion |
|---|---|---|---|
| D-FLEET-16 | A's first step is retiring user-level mission-skill symlinks — an **attended rig-home** change, not an in-scope repo step; A and B share nothing; default C | no | **NO PICK** — policy/mechanism row, first step not in scope |
| D-FLEET-17 | A = attended fix of `internal/executor/managed_agents` + `internal/config/ai.go`; B = a standing-authority grant (a ruling), not shared with A | no | **NO PICK** — ruling-gated |
| D-FLEET-18 | A's first step = the scope-guard arm — **built**: `875581f46` on draft #2110 (iteration 83) | **yes** | **NO PICK** — completed draft; dependent work waits on the ruling and the arm landing |
| D-FLEET-19 | A = attended `gcloud` quota-project set; B = `internal/messaging` (V1) | no | **NO PICK** — no in-scope step |
| D-FLEET-20 | A = `cmd/ailang/design_quorum.go` (out of scope, billing); B = attended credit top-up | no | **NO PICK** — ruling-gated |
| D-FLEET-21 | A = raise the Gate 3b poll cap in both skill copies (in scope) | **yes** — #2067 (`bdaad2825`, judge PASS 90, iteration 80) | **NO PICK** — completed draft; merges on the ruling |
| D-FLEET-22 | A = prod coordinator config in the multivac repo; B = another repo | no | **NO PICK** — no in-scope step |
| D-FLEET-23 | A (recommended) = attended/V1 coordinator fix; B's guard arm is B-only and B is itself a grant | no | **NO PICK** — ruling-gated |
| D-FLEET-25 | A = design → quorum → plan → judge for the rulings ledger; shared sub-item built as #2082 (`399cf799b`, PASS 94); the mechanism is drafted as #2165 (`985aa0161`) and was **BLOCKED twice at quorum**, final independent judge NEEDS-HUMAN-REVIEW | **yes** | **NO PICK** — draft exists and is blocked; needs an attended design correction, not another build |
| D-FLEET-26 | A = anchored `Selected model is at capacity` in `TRANSIENT_SIG` + regression arms (in scope) | **yes** — #2065 (`edb099371`, PASS 94, iteration 79) | **NO PICK** — completed draft; merges on the ruling |
| D-FLEET-27 | A = "keep luna allowed, **as shipped**" — the allowlist already landed at iteration 53; nothing to build; B is non-recommended | n/a | **NO PICK** — no in-scope step remains |
| D-FLEET-28 | A = `cmd/ailang/design_quorum.go` (out of scope; HD-2a admission change); B = another mission's one-off env | no | **NO PICK** — ruling-gated |
| D-FLEET-29 | A = attended `make quick-install`; B = driver change altering what every loop executes (routing/policy → needs a ruling); not shared with A | no | **NO PICK** — ruling-gated |
| D-FLEET-31 | A = **no change** (recommended); B = seating policy in `internal/mission/quorum/seating.go` (in scope but HD-2a and non-recommended); C = `internal/modelreg` (out of scope) | no | **NO PICK** — no in-scope step under the recommendation |
| D-FLEET-32 | A = an in-run spend cap (quota-policy design, HD-2a); B (recommended) = waive ask 3 + resolve — an attended waiver, not an in-scope code/design step; default C | no | **NO PICK** — ruling-gated |
| D-FLEET-33 | A = approve + land the `.pi/extensions/**` diff → **parks for Mark** (D-FLEET-3); the diff already exists (`iteration66/unowned-dirty-proposed.diff`); B's scope widening is B-only | **yes** | **NO PICK** — human park (Mark) |
| D-FLEET-34 | B (recommended) = waive ask 2 + resolve citing `7eaf9bece`; A = pi-threshold design that would re-open ruled D-FLEET-4/-7 | no | **NO PICK** — ruling-gated. *Note: this row's evidence is the same failure class as the new ticket — it is the corroboration for recurrence, not a pick* |
| D-FLEET-35 | A = registry rows in `internal/modelreg/models.yml` + `tools/pi-extensions/models.mission.json` (both out of scope); B = 402 reclassification (D-FLEET-11/13 policy) | no | **NO PICK** — no in-scope step |
| D-FLEET-36 | A's first step = the scope-guard arm — **built**: `bcf7f97ac` on draft #2108 (iteration 82) | **yes** | **NO PICK** — completed draft |
| D-FLEET-37 | A = V1/attended owns asks 1–2 (coordinator code); B = widen to `internal/coordinator/**` (non-recommended, a grant) | no | **NO PICK** — ruling-gated |
| D-FLEET-38 | A's first step = the scope-guard arm — **built**: `261fb3b64` on draft #2108 (iteration 82) | **yes** | **NO PICK** — completed draft |
| D-FLEET-39 | A/A2/B = `internal/coordinator/**` + `cmd/ailang` (out of scope) | no | **NO PICK** — ruling-gated |
| D-FLEET-40 | A = attended/V1 coordinator session; B = keep both open for one coordinator design | no | **NO PICK** — ruling-gated |
| D-FLEET-41 | A (recommended) = attended `Dockerfile.agent-go` change (out of scope). **B = add the executor's missing-toolchain rule to both sprint-evaluator skill copies — IN SCOPE (`.claude/skills/sprint-evaluator/**`, `.agents/skills/sprint-evaluator/**`) and not policy-class.** "A and B are compatible" — but *compatible ≠ shared*, and no step is shared by A and B | no | **NO PICK — closest rival.** B is a genuinely authorized in-scope step the loop has not taken; it does not qualify under the charter's test because the row's **recommended** option is A and the default-if-unanswered is C. Named here so it is not lost; if Mark rules for B, or if the row's default is clarified toward the in-scope option, it becomes a clean pick |
| D-FLEET-42 | A = per-lane secrets scope; **every buildable step is drafted**: M1 #2141 (`4ea46f05`), M2 #2143 (`7f1e13d5`), M3 #2150 (`26b71388`), judged PASS 92 / 90 / 93 | **yes** | **NO PICK** — all buildable steps drafted; what remains is Mark's ruling + the host line + key rotation + redaction (human) |
| D-FLEET-43 | A (recommended) = lift the fleet's idle-fingerprint exclusion: `tools/launchd/lib/idle-fingerprint.sh` + driver pre-check + Gate 5 text (both skill copies) — **all in scope, no diff yet** | no | **NO PICK — second closest rival, and the one I would take next.** It fails the charter's test only because the row's own text conditions the diff on Mark choosing A ("built as a proposed diff only if you choose A") and its default-if-unanswered is B (no change). It also alters *when fires run* — loop behaviour, adjacent to policy — so I do not treat it as an autonomous build |

**Row-screen conclusion: no OPEN row qualifies for a new draft under the charter's step test.** Two rows (D-FLEET-41 option B, D-FLEET-43 option A) carry authorized, in-scope, undrafted steps and are named above rather than hidden; neither is the pick this fire, because the queue is not wholly ruling-gated (the tool-hang ticket is admissible) and because both are non-recommended options or are explicitly conditioned on an attended choice.

### 7.3 Ledger consistency measured on the reviewed HEAD

`design_docs/fleet-mission.md` at HEAD `2249a192…`: **42** decision rows, **26 OPEN**, **16 RESOLVED** (counted by parsing the `<!-- decision-ledger:start/end -->` block). Open IDs: 16, 17, 18, 19, 20, 21, 22, 23, 25, 26, 27, 28, 29, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43 — i.e. **D-FLEET-30 does not exist** (numbering gap). Matches the charter's "42 rows / 26 OPEN". No status token was changed by this review.

**Observation (non-load-bearing):** the manifest's `ledger_sha256` = `f2d75d09…` matches **neither** the worktree copy of `design_docs/fleet-mission.md` (`1873ffab…`) **nor** `origin/dev`'s copy (`e90ae1fc…`), nor any iteration-96 evidence file I hashed. The ledger I screened is the one at the reviewed HEAD, which is what the task specified. Two further ledger-text inconsistencies are noted for the record, not acted on: D-FLEET-13/14/15 carry Status `RESOLVED` while their Evidence cell still ends "OPEN (filed …)", and D-FLEET-21's row text still describes draft #2067 as pending. None of these changes any row's status.

## 8. Implementation score

**N/A — no implementation to score.** No production diff was supplied for review, no approved design or sprint plan exists for this fire, and the 100-point rubric's categories (tests pass, lint, acceptance criteria, code quality, documentation, design fidelity) have no referent. No score is awarded, and none is inferred from the drafts of other iterations.

## 9. Typed verdict

Marker meanings used here (stated so it cannot be misread): **EVIDENCE-DISPOSITION** = the verdict is a *disposition of evidence* — there is no implementation to score, no failure occurred, and no attended decision is required. It is **not** NONE (a pick exists: the tool-hang ticket, with a bounded authorized proposal), **not** NEEDS-HUMAN-REVIEW (nothing here needs Mark; the two closest row candidates are named for the controller to weigh, not escalated), and **not** FAIL (no step failed).

EVALUATION_VERDICT: EVIDENCE-DISPOSITION reasons=report-only independent review; rc18 classification and one-retry remedy verified correct against scripts/mission_pi_run.sh and gate-3 rc18 clause; ticket tool-hang:triage-judge:home-metadata-search is the only admissible queue item and its valid next step is a bounded in-scope draft proposal (directive clause + role-run per-command ceiling, no threshold/retry/lane/policy change); row screen of all 26 OPEN rows yields no new pick (closest: D-FLEET-41 B, D-FLEET-43 A, both named); implementation score N/A
