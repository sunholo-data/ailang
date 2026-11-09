# Iteration 98 idle screen (independent evaluator)

VERDICT: NONE (idle). Confidence: moderate-high.

Scope: read-only. Inputs read first-party: `design_docs/fleet-mission.md` (Authority :73-91, Guardrails :93-127, Decisions :137+), `fleet-mission-evidence/iteration98/{tickets.json,open-decisions.tsv}`, log entries 92 and 94 (re-derived, not copied), `gh pr list`.

## 0. Premise and change versus iteration 97

| Check | Command | Result |
|---|---|---|
| Tickets unchanged | `cmp .../iteration97/tickets.json .../iteration98/tickets.json` | `tickets-identical`; 27 tickets, same signatures |
| Ledger | `scripts/mission_decisions.sh --check`; `--open` | valid, 44 rows; 28 OPEN = D-FLEET-16..23, 25..29, 31..45 (28 TSV lines, all `^D-FLEET`) |
| Ledger flips on origin/dev | `git diff HEAD...origin/dev -- design_docs/fleet-mission.md` | 8 changed lines, all `planned/`->`implemented/` link rewrites (#2186). No status flip; `tools/launchd` unchanged |
| Mark directives | `gh issue view 2039/2172 --json comments`, filtered to `MarkEdmondson1234` | 0 on #2039, 0 on #2172 (all comments authored by `sunholo-voight-kampff`) |
| Open fleet drafts | `gh pr list --author @me --state open`, `fleet/` heads | #2173 (i97 record), #2171, #2169, #2165, #2150, #2143, #2141. Also open drafts (by `gh pr view`): #2110, #2108, #2082, #2067, #2065 (all OPEN, draft, last updated 11-03..11-07) |

Nothing changed since iteration 97's end state: no new ticket, no flip, no directive.

## 1. Per-ticket gating (27)

Mapping method: signature substring search over `open-decisions.tsv` (python), plus charter queue lines for five tickets whose signature does not appear in a row: `fleet-mission.md:235` (item 13, D-FLEET-18), `:253` (19, D-FLEET-18 class), `:312` (44, D-FLEET-17 class), `:329` (58, D-FLEET-18), `:333` (61, D-FLEET-44).

| # | Ticket | Gating row | Buildable mechanical fix with no row? |
|---|---|---|---|
| 1 | reporting:daneel-reports-not-declared-triage | D-FLEET-22 | no (prod coordinator config) |
| 2 | quorum:gpt6-1-sol-openai-api-credits-exhausted | D-FLEET-20 | no (`cmd/ailang/design_quorum.go`) |
| 3 | messaging:pubsub-notify-wrong-project | D-FLEET-19 | no (rig gcloud / `cmd/ailang`) |
| 4 | ci:launchd-suite-flakes-openrouter-peer-and-gh-hang | D-FLEET-18 (charter :235) | no (`tools/eval/` refused) |
| 5 | eval:motoko-connection-probe-orphaned | D-FLEET-18 (charter :329) | no |
| 6 | policy:fleet-rulings-not-propagated-to-charters-and-skill | D-FLEET-25 | no (#2165, #2082 exist) |
| 7 | codex:server-capacity-error-crashes-finished-iteration | D-FLEET-26 | no (#2065 exists) |
| 8 | quota:admission-has-no-headroom-margin-and-flaps | D-FLEET-32 | no (recommended B waives) |
| 9 | pi:unowned-dirty-absolute-vs-relative-paths | D-FLEET-33 | no (`.pi/extensions/**` parks, D-FLEET-3) |
| 10 | skill:design-doc-creator-grep-portability | D-FLEET-36 | no (#2108) |
| 11 | skills:agents-copies-stale-non-mission | D-FLEET-38 | no (#2108) |
| 12 | pi:openrouter:uncapped-max-tokens-402 | D-FLEET-35 | no (`models.yml`, `tools/pi-extensions/`) |
| 13 | stall:watchdog:slow-on-hung-tool | D-FLEET-34 | no (recommended B waives) |
| 14 | crash:sprint-executor:sigbus-ram-tmp-full-make-test | D-FLEET-23 | no (terraform / coordinator) |
| 15 | sprint-evaluator:worktree-cap-and-exit1 | D-FLEET-37 | no (coordinator) |
| 16 | executor:3h-timeout-from-queue-and-unsliced-sprints | D-FLEET-40 | no |
| 17 | executor:timeout-loses-work-and-may-never-start | D-FLEET-40 | no |
| 18 | coordinator:cloud-lane-capped-by-local-worktree-limit | D-FLEET-37, D-FLEET-40 | no |
| 19 | executor:codex-credential-lease-collision-fails-concurrent-dispatch | D-FLEET-39 | no |
| 20 | coordinator:job-retry-success-dropped-after-failed-completion | D-FLEET-39 | no |
| 21 | executor:codex-credential-collision-drops-automatic-planner-handoffs | D-FLEET-39 | no |
| 22 | ci:motoko-probe-refusing-live-path-unexpected-success | D-FLEET-18 (charter :253) | no |
| 23 | quorum:oc-glm-5-3-output-cap-invalid | D-FLEET-31 | no (A = no change) |
| 24 | quorum:claude-p-ration-usage-429 | D-FLEET-28 | no (`design_quorum.go`) |
| 25 | ci:distinct-build-test-failures | D-FLEET-17 (charter :312) | no (`cmd/ailang`, `internal/*`) |
| 26 | tool-hang:triage-judge:home-metadata-search | D-FLEET-44 (charter :333) | no (parked; design BLOCKED) |
| 27 | skill:idle-record-barred-on-degraded-no-pick-fire | D-FLEET-45 | no (parked iteration 97) |

All 27 are ruling-gated; none is ungated. Rows that gate no ticket by signature: D-FLEET-16, 27, 29, 42, 43 (checked by the same substring search).

## 2. Per-row classification (28 OPEN)

Columns: (a) recommended / default; (b) first step of that option (or step shared by all non-default options) in fleet Authority scope?; (c) existing diff; (d) needs an attributable human choice?

| Row | (a) rec / default | (b) in-scope first step? | (c) diff | (d) human choice | Verdict |
|---|---|---|---|---|---|
| 16 | A (retire user-level skill symlinks, pin `--add-dir`, parity gate) / C | no: first step is attended symlink retirement; Phase 3b launch/routing is HD-2a | none | yes (ruling on A vs B vs C) | GATED |
| 17 | A (attended fixes `internal/gitexec` + `ai.go`) / C | no: `internal/executor/**`, `internal/config/ai.go` outside allowlist (`fleet-mission.md:76-85`) | none | yes | GATED |
| 18 | A (guard arm for two `tools/eval/` files) / C | yes (guard arm ships alone) | #2110 | yes (widens Authority) | GATED (diff exists) |
| 19 | A (rig `gcloud` quota project) / C | no: host ADC credential | none | yes (attended) | GATED |
| 20 | A (route OpenAI seat via codex, `design_quorum.go`) / C | no: `cmd/ailang`, billing | none | yes | GATED |
| 21 | A (Gate 3b cap 3600 s) / C | yes | #2067 (head `bdaad2825`) | yes (time-budget call) | GATED (diff exists) |
| 22 | A (prod coordinator `triage_only_inboxes`) / C | no: multivac cloud config | none | yes | GATED |
| 23 | A (attended/V1 fixes Step 5b `coordinator_cloud_task.go`) / C | no; B (fleet authority) is non-recommended and its guard arm is not a step shared with A | none | yes | GATED |
| 25 | A (rulings ledger design + doctor check; fix World fork claim) / C | yes: design doc, and the World-fork-claim correction is shared by A and B | #2165 (A), #2082 (shared sub-item) | yes (design approval) | GATED (diffs exist) |
| 26 | A (`Selected model is at capacity` into `TRANSIENT_SIG`) / C | yes | #2065 (head `edb099371`) | yes (HD-2a classification) | GATED (diff exists) |
| 27 | A (keep luna) / A as shipped | nothing to build | n/a | no | GATED (nothing to build) |
| 28 | A (`Blocked()` change in `design_quorum.go`) / C | no: `cmd/ailang` outside allowlist; also admission = HD-2a | none | yes | GATED |
| 29 | A (attended `make quick-install` on rig) / C | no: not a repo diff | none | yes | GATED |
| 31 | A (no further change) / A | nothing to build | n/a | no | GATED (nothing to build) |
| 32 | B (waive ask 3) / C | recommended B builds nothing; A (in-run cap) is non-recommended and quota policy HD-2a | none | yes | GATED |
| 33 | A (land `.pi/extensions/**` diff) / none stated | `.pi/extensions/**` parks for Mark (D-FLEET-3, `fleet-mission.md:88-90`); `pi_assets` sync out of scope | `iteration66/unowned-dirty-proposed.diff` | yes | GATED (diff exists) |
| 34 | B (waive ask 2) / thresholds unchanged | nothing to build | n/a | yes (waive) | GATED |
| 35 | A (V1 adds registry rows + `models.mission.json`) / open | no: `internal/modelreg`, `tools/pi-extensions/` beyond `sandbox/**` | none | yes | GATED |
| 36 | A (widen scope to design-doc-creator skill + fix 8 grep sites) / C-like | yes (guard arm alone first) | #2108 (commit for 36 A) | yes (Authority widening) | GATED (diff exists) |
| 37 | A (V1/attended owns coordinator asks) / open | no: `internal/coordinator/**` | none | yes | GATED |
| 38 | A (sync-script + `_scope_is_fleet_support` for 16 skills) / C | yes (guard arm alone first) | #2108 (commit for 38 A) | yes (Authority widening) | GATED (diff exists) |
| 39 | A (coordinator `credentialLaneHolder`) / C | no: `cmd/ailang/coordinator*`, `internal/credentiallease/**` | none | yes | GATED |
| 40 | A (coordinator shutdown re-queue) / C | no: `internal/coordinator/**` | none | yes | GATED |
| 41 | A (`Dockerfile.agent-go`) / C | no: `docker/**` outside scope. B (skill text, in scope) is non-recommended; A and B share no step, so the charter shortcut does not admit it | none | yes | GATED |
| 42 | A (per-lane `secrets.env` strip) / B | yes | #2141 (M1), #2143 (M2), #2150 (M3); remaining items are the ruling, `~/.zshenv` host line, key rotation, redaction | yes (HD-2a) | GATED (all buildable steps drafted) |
| 43 | A (lift fleet exclusion in `idle-fingerprint.sh`; Gate 5 text) / B | in scope (`tools/launchd/**`), no diff | none | yes by the row's own text: "built as a proposed diff only if you choose A"; default is B; gates zero tickets | GATED (see objection) |
| 44 | A (request corrected design evidence) / B immediately | a design request, not a build; row says "no planning/execution/merge" | design BLOCKED (`m-fleet-role-artifact-contract.md`) | yes | GATED |
| 45 | B (no change) / B | recommended B builds nothing; A is a re-revision of a twice-BLOCKED design | design BLOCKED twice | yes | GATED |

Source for every row: `fleet-mission-evidence/iteration98/open-decisions.tsv` (row text), PR states from `gh pr view <n> --json state,isDraft` (all OPEN drafts, none merged).

Tally: 28 rows. 9 have a recommended in-scope first step: 8 already have a diff (18, 21, 25, 26, 33, 36, 38, 42) and 43 has none (see the objection). The other 19 begin out of scope, are attended-only, or build nothing.

## 3. Verdict

NONE. The "all open tickets ruling-gated" fallback (`fleet-mission.md:96`) admits a row whose recommended option, or a shared non-default step, begins with an in-scope step that has no proposed diff yet. Re-deriving from the rows: every such first step (guard arms for D-FLEET-18/36/38, the D-FLEET-21 poll cap, the D-FLEET-26 transient signature, the D-FLEET-25 ledger and World-fork correction, the D-FLEET-42 scope library, wiring and dry-run) already has a draft PR, and rebuilding any of them would duplicate an unmerged proposal. Every remaining row either starts outside the Authority allowlist (coordinator, `cmd/ailang`, `internal/*`, `docker/**`, host config, `.pi/extensions/**`) or recommends no change. Nothing changed since iteration 97 (section 0), so iteration 92/94's NONE still holds, and iteration 97's park (D-FLEET-45) is itself a recommended "no change".

## 4. Strongest objection to my own verdict

D-FLEET-43 A is recommended, is a fleet-scope change (`tools/launchd/lib/idle-fingerprint.sh:23`, Gate 5 text), and has no diff. Read literally, the "step by row" rule would admit building it as a draft PR. I reject that for three reasons: the row itself says it is "built as a proposed diff only if you choose A" with default B; it gates no ticket, so building it is the self-sourced work the "Ticket-driven only" bullet forbids (`fleet-mission.md:95`); and iteration 94 already ruled this out. Confidence is not "high" because that rejection relies on the row's own wording and a prior iteration's ruling, not on a charter sentence that states it. A second, lesser risk: I did not read every PR diff, so "a diff exists" for D-FLEET-18/36/38 rests on PR titles and iteration 92's judge (`iteration92/idle-judge.md:15`), which I checked against the live PR list and states.

## 5. Next

A new ticket or an attributable Mark ruling (any flip of a D-FLEET row, or a `MarkEdmondson1234` comment on #2039/#2172). Otherwise the next fire reaches the same NONE.
