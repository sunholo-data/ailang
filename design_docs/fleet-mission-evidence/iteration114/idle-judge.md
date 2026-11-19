# Fleet iteration 114 — independent idle-screen judge

- **Role**: `evaluator` (`MISSION-ROLE: evaluator`), Agent tool `model=sonnet` (resolver `agent-tool sonnet declared:alias-pin`), foreground, 74.9 s, 14 tool uses, 90,904 tok. Inputs only: [tickets.json](tickets.json), [open-decisions.tsv](open-decisions.tsv), the charter and live log at the `fleet/i113-record` tip, [issue-sweep.md](issue-sweep.md), read-only repo at `563c6c936`. Told to argue against idling; told to read log entries 101 and 113 for prior grounds but to engage each on its merits, not inherit it. No controller conclusion was supplied.
- **Controller re-checks of load-bearing claims (first-party)**: drafts #2065, #2067, #2082, #2108, #2110, #2141, #2143, #2150, #2165 all `OPEN draft=true`; `tools/launchd/lib/idle-fingerprint.sh:23` still refuses `fleet`; `coordinator:cloud-lane-capped-by-local-worktree-limit` sits at charter queue row 52 under D-FLEET-37 (the judge mapped it to "coordinator code, out of scope" — same gate either way).

---

VERDICT: NONE — confidence 75%

## Ticket map (27 tickets; all checked against /tmp/fleet_i114/open-decisions.tsv and the charter)
| Ticket signature | Gate |
|---|---|
| ci:distinct-build-test-failures | D-FLEET-17 (dev required-check reds, owner V1 disabled; CI workflows/non-mission code out of Authority) |
| ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18 (files in tools/eval/; guard arm is draft #2110, OPEN) |
| ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 (same; draft #2110) |
| eval:motoko-connection-probe-orphaned | D-FLEET-18 (retire-or-own, tools/eval/; draft #2110) |
| codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26 (diff already built: draft #2065) |
| coordinator:cloud-lane-capped-by-local-worktree-limit | out of scope (coordinator code); D-FLEET-37/39/40 family, no fleet path |
| coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 (coordinator, out of scope) |
| crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23 (terraform/coordinator, out of scope) |
| executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40 (coordinator) |
| executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 |
| executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39 |
| executor:timeout-loses-work-and-may-never-start | D-FLEET-40 |
| messaging:pubsub-notify-wrong-project | D-FLEET-19 (ADC quota project, host config) |
| pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35 (tools/pi-extensions models.json + modelreg, out of scope; 402 reclassification is HD-2a policy) |
| pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33 (.pi/extensions/** parks per D-FLEET-3; draft/diff noted in row) |
| policy:fleet-rulings-not-propagated-to-charters-and-skill | D-FLEET-25 (A = design doc; B world-fork fix is draft #2082) |
| quorum:claude-p-ration-usage-429 | D-FLEET-28 (quota/ration policy HD-2a, cause unmeasured) |
| quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20 (billing/credits; modelreg out of scope) |
| quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31 (recommendation A is "no change"; B is seating policy HD-2a; C is models.yml out of scope) |
| quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32 (quota policy HD-2a; recommended B = waive) |
| reporting:daneel-reports-not-declared-triage | D-FLEET-22 (coordinator config key, out of scope) |
| skill:design-doc-creator-grep-portability | D-FLEET-36 (skill outside allowlist; guard arm draft #2108) |
| skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45 (recommended B = no change; A is a design revision, not code; design twice quorum-blocked) |
| skills:agents-copies-stale-non-mission | D-FLEET-38 (guard arm draft #2108) |
| sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37 (coordinator, out of scope; ask 3 ghost of #1688) |
| stall:watchdog:slow-on-hung-tool | D-FLEET-34 (recommended B = waive; A changes ruled thresholds D-FLEET-4/-7) |
| tool-hang:triage-judge:home-metadata-search | D-FLEET-44 (design request; frozen design BLOCKED; draft #2171 parked for human review) |

Buildable with no OPEN row, no draft, in scope: none. Other in-scope OPEN rows that are not ticket-keyed already have drafts: D-FLEET-21 (#2067), 26 (#2065), 42 (#2141/#2143/#2150), 33 (diff noted). D-FLEET-16 (Phase 3b skill mechanism), 27, 29 (attended make quick-install / driver change that alters what every loop runs) and 17 are policy or host actions with no code-first in-scope step.

## Candidates considered

### 1. D-FLEET-43 A (lift the fleet exclusion at tools/launchd/lib/idle-fingerprint.sh:23 and the Gate 5 text) — the strongest candidate; REJECTED
The surface is verified: line 23 reads `''|fleet|triage) echo 'idle-fingerprint: MISSION_NAME must name a product mission (not fleet/triage)' ... exit 2`. It is in tools/launchd/** (Authority allowlist) and has no diff.
For admitting it (Guardrails): "When every open ticket is ruling-gated ..., take the in-scope OPEN row whose recommendation is code and which has no proposed diff yet, and build that diff as a draft PR without merging it. Screen by step, not by row ... a row qualifies when its recommended option ... *begins* with an in-scope step that has no diff."
Against (same bullet, and its opening): "Ticket-driven only. The queue is `ailang mission ticket open --json` plus Mark's directives. No self-sourced audits, no 'while I'm here'." and "It is not self-sourcing: the ticket is already in the queue."
My own check: none of the 27 signatures concerns the fleet's idle skip. I read the only near neighbour, skill:idle-record-barred-on-degraded-no-pick-fire (world, 11-09). It is about World's degraded-fire bar, and its ask is "World is not editing the shared skill or the driver"; the fleet exclusion is not in its text. D-FLEET-43 was filed by the controller at iteration 92 from the fleet's own idle-cost observation. A diff would therefore be sourced from the fleet's own row, not from a ticket or a Mark directive.
Where I disagree with prior grounds: I do NOT rely on the row's phrase "built as a proposed diff only if you choose A" (that was the older, circular ground; D-FLEET-21 and 26 have default C and were built anyway under the screen). I rely only on the no-originating-ticket ground: the screen's stated justification does not hold for this row, and the opening sentence of the bullet is the controlling prohibition. The screen's operative sentence is stated at row level and never says "with a ticket"; that wording gap is real and I note it as the strongest objection below.
Additional point: D-FLEET-45 shows Mark/owner has not ruled that an "idle record" mechanism is wanted for fleet; the row's default is B.

### 2. Guard arms #2108/#2110 and drafts for D-FLEET-21/26/42 — REJECTED
Charter: "build that diff as a draft PR ... which has no proposed diff yet". All named drafts are verified OPEN and draft via `gh pr view` (2108, 2110, 2065, 2067, 2141, 2143, 2150). Building more would duplicate them. "A fix that widens the scope guard ships ALONE, first": these are already the guard-alone PRs and await rulings.

### 3. D-FLEET-45 A (one more bounded design revision) / D-FLEET-44 A (request corrected design) — REJECTED
Neither recommended option is code (45 recommends B; 44's A is a design request whose frozen design was BLOCKED twice, with #2171 parked for human review). The screen requires "recommendation is code". D-FLEET-45's A is also a design revision with quorum, plus a retro-analysis, which would be a self-sourced audit.

### 4. D-FLEET-25 B / world.toml fork claim — REJECTED
Already drafted as #2082 (open). Row recommends A, a design doc.

## Strongest objection to my verdict
The screen's operative sentence is row-based ("take the in-scope OPEN row whose recommendation is code and which has no proposed diff yet") and says nothing outright about an originating ticket; the "not self-sourcing" sentence is a justification, not a condition. A purely structural reader (iteration 113's minimax judge) selects D-FLEET-43 A, and the pick costs only a draft PR with no merge. Idle screens since 108 have also spent real judge tokens (D-FLEET-43's own cost argument), and this loop has now idled for many fires. I still prefer NONE because the justification clause is the charter's own explanation of why drafting is not self-sourcing, and its premise fails here; but the text is ambiguous, so I give 75% and flag that a draft PR (not a merge) would be low-harm if the controller reads the text the other way. A ruling on D-FLEET-43 either way, or a charter wording fix by Mark, would end the repeat.

## Sweep (issue-sweep.md, 109 issues)
Fleet harness scope: NO. I filtered every zero-mention issue (all four fleet-record columns 0). They are language core / codegen / type checker / stdlib / docs-MCP / daneel / ailang-parse / eval or product items: #2337, 2334, 2333, 2332, 2329, 2326, 2325, 2324, 2323, 2322, 2318, 2317, 2315, 2314, 2312, 2311, 2310, 2308, 2307, 2303..2290, 2282, 2278, 2274, 2272, 2271, 2231, 2221, 2215, 2196, 2187, 2145, 1714, 1618, 1485, 1307, 1278, 1276, 1267, 1262, 1132, 959, 934, 905, 715, 662, 614, 612, 584, 578, 539, 534. Two near-misses checked and excluded: #1436 (docs mission bookkeeping, not the fleet) and #584 (repo-size boundary gates, not the loop harness). The one fleet-ish bookkeeping issue, #2172, has 0 allowlisted directives per the controller.
