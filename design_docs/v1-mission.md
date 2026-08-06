# V1 Mission — work the backlog to a v1.0.0 release

**Type**: Long-running mission (peer of [motoko-mission.md](motoko-mission.md)); advanced by a
scheduled nightly outer loop on the always-on rig, coordinated by Fable.
**North star**: Ship AILANG v1.0.0 — a release whose bar is *written down, met, and verified*,
with the backlog worked through the honed inner loop (design-doc → sprint-plan → execute → evaluate)
rather than ad-hoc sessions.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration. **Scheduling: launchd `dev.ailang.mission-control`** (CONTINUOUS since
2026-07-10 per Mark: StartInterval 2h + overlap guard = back-to-back iterations, ≤2h idle; was
22:00-nightly for the first supervised runs) behind the billing guard — API keys are stripped from the environment
(subscription-or-nothing by construction) and a cheap auth probe runs first: keychain OAuth
suffices while the rig is logged in (verified 2026-07-10); `CLAUDE_CODE_OAUTH_TOKEN` in
secrets.env is an optional belt-and-braces for post-reboot login screens. Probe failure refuses
loudly with zero spend. The Claude Code
scheduled-tasks path was TESTED AND RULED OUT for this job (2026-07-10 canary): that system is
desktop-side — tasks landed on /Users/mark (Mark's machine, not the rig) and a probe task never
dispatched even there (a June one-time task was also found a month overdue). Wrong machine +
unreliable dispatch → launchd is primary, not fallback.
**Log**: [v1-mission-log.md](v1-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue
[#329](https://github.com/sunholo-data/ailang/issues/329) — every iteration posts its morning
report there as a comment (Mark follows by email via issue subscription, no Claude login
needed); driver crashes post there too.

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

The single source of truth for the values that differ per mission. The **one** `mission-control`
skill reads this block (and the driver env it exports) instead of hardcoding — so the same skill,
unforked, runs any mission. V1's values are the skill's built-in defaults, so nothing here changes
current behavior.

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/v1-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `v1` (driver: `MISSION_NAME`; v1 keeps the legacy un-namespaced
  `~/.ailang/state/mission-*` paths bit-for-bit — see M1)
- **Bookkeeping issue**: origin `#329`, rotates weekly; live number in
  `~/.ailang/state/mission-gh-issue` (this week `#422`), watermark in `~/.ailang/state/mission-329-last-seen`
- **CI workflows Gate 3b / Gate 1 poll**: `CI`, `Build and Release`, `Deploy Documentation to GitHub Pages`
- **Verify profile**: `go-compiler` — this repo compiles the AILANG toolchain, so gates rebuild
  BOTH binaries (`make quick-install && make build`) and run `make test`; `~/go/bin/ailang` (PATH)
  and `bin/ailang` go stale independently (confirm `--version` == `git describe`). (The alternative
  profile `ailang-code`, for an AILANG-code repo like Ailang World, uses the shipped binary's own
  gates — `ailang check` / `ailang test` / `ailang ai-check` — with no compile step.)

---

## STATUS (rotation rule — added 2026-07-14, Fable-quota diet)

Newest **3** STATUS stamps live here; older ones move to
[v1-mission-status-archive.md](v1-mission-status-archive.md). **Loop: at Gate 4, after
adding your stamp, move the now-4th stamp to the TOP of the archive file.** **SELF-HEAL
(added 2026-07-22 iter-83, 2nd instance of drift after iters 77+82 hand-corrected an
already-drifted N>4): if MORE than 3 stamps remain after adding yours, move ALL but the
newest 3 to the archive top, newest-first — do not assume exactly one over-count.**
Rationale: every iteration re-reads this charter — 30+ stamps were ~500 lines of history
tax per read, on the scarcest model budget. The append-only history lives in the log + archive.

## STATUS 2026-08-06 — ITERATION 155: **THE OUTAGE STILL BLOCKS LANDING, SO THE PICK WAS THE UNPARKED `m-net-effect-proxy-boundary` — AND ITS SPRINT PLAN REFUTED A CLAIM THIS CHARTER HAS CARRIED FOR THREE ITERATIONS: THE `AC10(d)` TRIPWIRE THAT IS SUPPOSED TO TELL THIS ITEM IT SUCCEEDED **CANNOT TRIP**.** Gate 0/1: kill switch armed, billing **CLEAN**, gh account correct, **ZERO** open `[nightly-eval]` alarms, inbox **empty**, **no new Mark comment** on `#559` (watermark `2026-08-04T12:16:12Z`; control: **2** Mark comments exist, both ≤ watermark, so the empty result is the filter working). No rotation due (`#559` created `05:08Z` = **07:08 CEST**, after the Monday-07:00 LOCAL boundary; 51 < 80). `dev == origin/dev == 291142de3` at pick and the running `SKILL.md` byte-identical to origin (`cmp` silent) — **the 08-03 reconcile has HELD for a FIFTEENTH consecutive iteration**, so Gate 4 wrote IN PLACE (charter/log re-confirmed byte-identical to origin immediately before writing). Cheap tell case-correct with control (ITERATION 154 = 1, 153 = 2, rotation invariant 3). Died-mid-flight check on its **sixth** use found `#606`, `#608` and `.wt-iter154` — all iteration 154's, all known. **THE RED dev IS STILL THE OUTAGE, AND THE CORRECTED QUESTION ANSWERS IT IN ONE READ**: CI's `test` and `lint` are `cancelled` at **steps=0**, `build` skipped, and **no step failed anywhere** — so nothing in the diff did. githubstatus: Actions **major_outage**, incident `15:22Z` still `investigating`, and GitHub's own `20:34Z` update says webhooks are throttled to **~15%**, so *pushes and PRs are not creating workflow runs at all*. That is the mechanism behind iteration 154's `#608` mystery. **FIND 1 — I ALMOST BANKED A ZERO FROM AN INSTRUMENT I HAD MIS-INVOKED, AND THE CONTROL IS WHAT CAUGHT IT.** `gh api actions/runs?head_sha=<sha>` returned `total=0` for `#606` — a PR that `gh run list` simultaneously showed **5** runs for. The cause is that I passed a **9-character truncated SHA**; the endpoint requires the full 40 and returns an empty set, not an error, for anything shorter. Measured both ways on the SAME commit: full SHA → `total=5`, truncated → `total=0`. Iteration 154's "`#608` has ZERO workflow runs created" is now **CONFIRMED** — but it was only ever supported once the instrument had been validated on a known-positive. Rule 3a striking in the exact place Gate 3b sends you for "maximum certainty". **FIND 2 — `workflow_dispatch` IS WEBHOOK-INDEPENDENT, AND IT WORKS WHEN `push`/`pull_request` DO NOT.** Since the outage's mechanism is dropped webhook *events*, a POST to the dispatch API is a different path entirely: it creates the run directly. Fired against `#608`'s branch and measured the transition **`total=0` → `total=1`** (`CI: queued`, `event=workflow_dispatch`) — the first run that PR has ever had. A reusable outage lever the playbook did not have; its limit is that `docs-gate` still needs a real `pull_request` context (its detector diffs against the PR base), so dispatch reaches 3 of the 4 required contexts. **PICK = `m-net-effect-proxy-boundary`**, unparked by Mark's `D-6 = (A)`, and chosen because a design/planning pick needs no landing gate — exactly what the outage rule prescribes. Blocker re-verified **by purpose, not status** (iter-145's rule): CI-flake `M1`–`M5` all landed (`c440a1628`…`c9e1a4f98`) and the `AC10(d)` tripwire exists. **BOTH BLOCKED QUORUM OBJECTIONS RESOLVED WITHOUT A THIRD ROUND.** `gemini-3-1-pro`'s unverified-premise objection was **satisfied by measurement rather than argument** (rule 3f): row **V20** re-derives `E_NET_IP_BLOCKED` (`net_security.go:27,34,46,51,56`), `E_NET_DNS_FAILED` (`:90,94`) and their surfacing through `makeResultErr("Transport", …)` (`net.go:551,556,567,605,631,639`), naming the exact two sites the change must update — `:567` (the preflight being moved) and `:631` (where a proxy failure arrives as `*url.Error`). Control: 12 `makeResultErr(` occurrences = **11 call sites + 1 definition** at `:773`, and six non-`Transport` constructors prove the matcher discriminates. `gpt5-6-sol`'s objection was resolved by the human ruling; **`#612` filed** as the `go/packages` AST-analyzer follow-up, at resume time rather than during M4 so it cannot be lost if the sprint slips — Option A is "cheap gate now **and** the durable gate filed". **V21 re-derived all five shapes a textual gate cannot see: still zero — WITH TWO INSTRUMENT CAVEATS RECORDED RATHER THAN SMOOTHED OVER.** (a) The naive alias matcher returns a **false positive**, matching the `import` keyword itself in single-line `import "net/http"` — **iteration 150 hit this identical defect and I reproduced it before correcting**, so it is a property of the pattern, not of one author. (b) **The `new(` control does NOT fire in production scope at all**: repo-wide there are exactly **4** `new(` hits and all four are in a *test* file, so iteration 150's banked "control: 4" was measured in a wider scope than the claim it supported (rule 3b(i)). Shape 2's zero is the weakest of the five — which argues **for** `#612`, not against it. Freshness swept from the doc's own base `263df3df8` (rule vi-b): **none** of the files its rows cite has changed (control: `internal/apiserver/mcp.go` correctly detected as changed); `V2` re-derived first-party — still **7** literals across **4** files at the same sites. **FIND 3, THE HEADLINE — AND IT REFUTES MY OWN BRIEFING AND THIS CHARTER.** I handed the planner, as established fact, that this item's landing "must flip the `AC10(d)` tripwire RED"; the queue row has said so since iteration 149. **It is false.** `testEffectsProxyResidual` (`internal/testutil/egress_posture_test.go:66-85`) constructs its **OWN** `&http.Transport{}` at `:74` and exercises that, while its comment states it "intentionally trips red when Option B adds `ProxyFromEnvironment` to `internal/effects`". Confirmed with the tool that cannot miss: `go list -f '{{join .TestImports}}' ./internal/testutil` = **12 imports, ALL stdlib, ZERO ailang packages** (control: `./internal/effects` shows **6** ailang imports, so the instrument sees them when they exist). **A tripwire watching a local replica of the thing it claims to watch** — this mission's "a guard whose removal reds nothing" discipline in a new shape: a guard never CONNECTED to its subject. It shipped through a full sprint, a quorum and an evaluator PASS. The design doc was right all along ("**helper-only** residual logic" → M4 is a deliberate DELETION); the charter framing was wrong, and the queue row is corrected below. **FIND 4 (planner, rule 3e): all three `M1`/`M2` named-test gates return `rc=0` with **0** `=== RUN` lines and `[no tests to run]` at base** — written as "rc=0" they would pass after a revert that DELETES the tests; every named-test AC now asserts a `=== RUN` count. `go build ./...` **confirmed rc=1 at base** (`cmd/wasm`, `gen/main` have no native `main`) and excluded, replaced by a scoped build baselined `rc=0`. Plan: **4 milestones, 12 ACs each owned exactly once**, doc's 3-day shape kept; the planner also stripped an auto-linked `Fixes #612` that would have wrongly auto-closed the out-of-scope follow-up. **Routing evidence**: model=**claude-opus-5** · task-class=**plan** · planner=**opus** (`opus fail-closed:env-pin`, derived and used VERBATIM) · designer **NOT fired** (doc existed — rotation unchanged, next `claude:claude-fable-5`) · executor/evaluator **NOT fired** (no code written this iteration). `metered=$0.00` of the $5 ceiling. **GATE 3b: NOT GREEN — NOTHING LANDED, AND THE TWO COMMITS (`945f36727`, `7c7e5e58a`) ARE DOC-ONLY AND UNPUSHED PENDING THE OUTAGE.** `#606`: Build-and-Release and Docs **success**, CI + CodeQL **queued**; `#608`: CI **queued** via my dispatch. **Next**: merge `#606` + `#608` the moment CI drains, push these two commits, then route `m-net-effect-proxy-boundary` M1 to an executor. **`D-1` (the proxied-route SSRF pinning trade) is still owed a human ratification** and is the only ask on this item.

## STATUS 2026-08-06 — ITERATION 154: **THE GITHUB ACTIONS OUTAGE IS STILL LIVE, SO NOTHING COULD LAND — AND THE PICK WAS CHOSEN FOR THAT: `#602`, A FLAKE WHOSE INVESTIGATION NEEDS NO LANDING GATE. IT DID NOT REPRODUCE IN TWO FULL-SUITE RUNS; IT WAS CONFIRMED INSTEAD BY MEASURING THE RACE DIRECTLY AT 1 SPAWN IN 300 — AND THAT MEASUREMENT REFUTED BOTH THE ISSUE'S STATED CAUSE AND ITS SUGGESTED FIX.** Gate 0/1: kill switch armed, billing **CLEAN**, gh account correct, **ZERO** open `[nightly-eval]` alarms (control: **52** in `--state all`, so the zero is a measurement), inbox 4 messages acked BEFORE sending anything (the `ack --all` outbound-sweep hazard), **no new Mark comment** on `#559` (watermark `2026-08-04T12:16:12Z`; control: **2** Mark comments exist, both ≤ watermark, so the empty result is the filter working). No rotation due (`#559` created `05:08Z` = **07:08 CEST**, after the Monday-07:00 LOCAL boundary; 50 < 80). `dev == origin/dev == e70fb7484` and the running `SKILL.md` byte-identical to origin (`cmp` silent) — **the 08-03 reconcile has HELD for a FOURTEENTH consecutive iteration**, so Gate 4 wrote IN PLACE. Cheap tell case-correct with control (ITERATION 153 = 1, 152 = 2, rotation invariant 3). **Died-mid-flight check on its FIFTH use found real work**: iter-153's `#606` open and MERGEABLE, plus the queued `#545`. **THE OUTAGE IS NOT ONE-SHAPED, AND ITER-153'S OWN INVARIANT WOULD HAVE MISREAD IT.** That rule says the discriminating signature is that **no repo command ever ran** (`steps=0` / dead in `Set up job`). Eleven of twelve failing jobs on `#606` matched exactly that. **The twelfth did not**: `Build macos-latest` ran **17 steps, every one success or skipped — including `Run tests`, `Build binary`, `Upload artifact` and `Complete job` — yet the JOB's conclusion is `failure`.** A job whose every step passed is not a code failure; it is the platform failing the job outside step execution. Read strictly, the invariant would have classified that one job as a genuine regression and pointed at a revert. **The generalisation: an outage signature is a FAMILY, so the question is not 'did steps run' but 'is the failure attributable to any step' — and when no step failed, nothing in the diff did.** **I ALSO SHIPPED A VACUOUS PASS INTO MY OWN GATE-3b POLL, AND CAUGHT IT ONLY BY READING WHAT IT COUNTED.** My first poll reported `pending=0 failures=0 SETTLED` — over a check-runs list of **ONE** (`automerge/skipped`) where **18** had existed minutes earlier. Re-running a workflow **EMPTIES the `check-runs` collection**, and every aggregate over an empty set is vacuously green. This is not my invention: **Gate 3b explicitly recommends `check-runs` as the instrument that 'cannot drift'** — true against SHA drift, and precisely wrong during a re-run, which is the one situation Gate 3b's own re-run advice puts you in. Fixed by asserting COMPLETENESS before any verdict counts (all 4 expected workflows present, else 'INSTRUMENT INCOMPLETE — no verdict'); rule 3a's known-positive control, applied to a poll rather than a search. **PICK = `#602`**, the FINDINGS-BATCH row's remaining half — chosen precisely because its work needs no landing gate, which is what the outage rule prescribes. **THE REPRO FAILED, AND THAT IS THE FINDING'S STRENGTH.** Arm C `go test ./...` rc=0 — reported WITH its limitation attached: **86 of 108 packages were cached, only 22 genuinely ran**, so it was a far lighter load than the issue's conditions. Arm D `go test -count=1 ./...` was the honest arm (**108 ran, 0 cached**) and was ALSO rc=0. Two clean runs do not refute a flake, so I measured the variable instead. **THE ISSUE'S STATED CAUSE IS OFF BY 3×**: the fake solver is not killed by 'the 1s solver timeout' — `solver.go:158` gives it `max(Timeout, effectiveTimeout) + solverKillGrace` = **3s** (corroborated, not just read: the isolated arm takes **3.274s**). **THE MEASUREMENT, 500 trials with a probe replicating the spawn exactly**: idle **200/200**, mean 209ms, max 232ms, **0** over budget; under a full `-count=1` suite, **1 trial in 300 exceeded 3s, at 3.435s** (margin **−435ms**). So **~0.3% per spawn, ~0.7% per suite run** — which is exactly why CI stays green, why it survived the CI-flake sprint, and why iteration 151 saw it once. Note the over-budget trial landed at load **13.6**, NOT at the higher **24.4**: a tail effect (first exec of a freshly written script), not a smooth function of load — a fix tuned to 'more load' would have missed it. **THE ISSUE'S SUGGESTED FIX IS REFUTED BY THE SAME MEASUREMENT**: polling for the pidfile cannot work, because a lost race means the shell was killed BEFORE `echo $! > file` ever ran, so the file never appears at all. The trial simply proves nothing about process-group kill — so the fix is to treat it as **inconclusive and retry**, bounded at 3 with a loud `Fatalf`. **EXECUTOR = the pi lane** (`pi:openrouter/deepseek/deepseek-v4-flash-0731`) — **datapoint 3 of ≥3** for Mark's new routing, and it followed a prescriptive directive exactly: probe rc=0, run rc=0, directive 7,858 B delivery-asserted, zero git writes. **pi has NO sandbox, so I re-checked the main checkout for out-of-worktree writes** — byte-identical to preflight. **CONTROLLER RE-RAN EVERY GATE and ran TWO mutations the executor did not**, both sha256-proven to LAND and reverted byte-identical: **A** — `syscall.Kill(pid)` instead of `-pid`, breaking process-group kill — reds on **every** attempt (`child process 49684 still exists after 2s`), which is the one that matters: **the retry cannot mask a real regression**; **B** — force the race lost every time — reds with three inconclusive logs and the loud exhaustion message, so the exhaustion path is reachable rather than a silent pass. Gate list **DERIVED from `ci.yml`**, not recalled (iter-152's rule 3g): `check-changelog`/`check-file-sizes`/`check-boundaries`/`fmt-check`/`vet` all rc=0. **EVALUATOR sonnet PASS 88/100 r1, zero blocking** (generator≠judge holds: DeepSeek/OpenRouter executor ≠ Anthropic judge), and it devised its OWN third mutation rather than replaying mine. **ITS NON-BLOCKING FINDING WAS ACTED ON, NOT FILED — and it was a genuine scope error of MINE** (rule 3b(b)): I quoted the ~3s budget as if it covered BOTH tests. It does not — `Z3Version` passes `versionProbeTimeout` (**5s**) directly as its context deadline, since `solverKillGrace` is only its `WaitDelay`. So the two tests do not share a budget, my 0.3% figure is the SMALLER budget's rate, and **0 of 500 trials exceeded 5s**. Corrected in the constant's doc comment (`5b06718b6`) where the next reader will ask why 3 attempts suffices. **Routing evidence**: model=**claude-opus-5** · task-class=**execute** · executor=**pi `deepseek-v4-flash-0731`** · evaluator=**sonnet** · designer/planner **not fired** (bug-fix lane; rotation unchanged, next `claude:claude-fable-5`). `metered=$0.0063` of the $5 ceiling (pi/OpenRouter — 27.9k in / 4.2k out / 165.6k cache-read). **GATE 3b: NOT GREEN, SO `#602` IS NOT LANDED — PR #608 is a RESUME POINT.** `#608` has **ZERO** workflow runs created at all — GitHub never created them for that push, so there was nothing to re-run and no instrument to poll. `#606`'s four re-runs were accepted (rc=0, unlike iter-153's refusal) but 0 of 4 had started after 28 minutes. **AMENDED at 21:52 rather than left stale — iter-153's own lesson that a count over an in-flight run expires as it is written, applied to my own row.** By then the re-runs had partly drained and the picture is BETTER evidence, not worse: **`Deploy Documentation` went `failure` → `success` on a byte-identical tree**, which is the strongest control there is (only the environment can be the variable); `Build and Release` sat `queued`; and `CI` + `CodeQL` came back `failure` whose every job is **`cancelled` at `steps=0` with ZERO failed steps** — i.e. the corrected question ("is the failure attributable to any STEP?") answers no, so it is the outage and not `#603`'s code. CI/CodeQL re-fired (rc=0) to drain as capacity returns. githubstatus still reads Actions **major_outage / investigating** (incident `15:22Z`, last update `19:43Z`). **Next**: confirm `#606` and `#608` green and merge both once the incident closes, then `#545` or Lane B1. **`D-6` is answered and `m-net-effect-proxy-boundary` is UNPARKED; the RELEASE ask is settled as WAIT.**

## STATUS 2026-08-06 — ITERATION 153: **dev CI WAS RED AND IT WAS NOT OUR CODE — A DECLARED GITHUB ACTIONS MAJOR OUTAGE. THE PICK WAS `#603`, WHERE THE CodeQL TAINT TRACE IS *CORRECT* AND THE FINDING IS *NOT EXPLOITABLE*, AND THE REASON TO FIX IT ANYWAY IS THAT ALL THREE THINGS MAKING IT SAFE ARE INHERITED FROM A THIRD-PARTY DEPENDENCY AND ASSERTED NOWHERE IN THIS REPO.** Gate 0/1: kill switch armed, billing **CLEAN**, gh account correct, **ZERO** open `[nightly-eval]` alarms (control: **30** in `--state all`, so the zero is a measurement), inbox 3 messages all acked, **no new Mark comment** on `#559` (watermark `2026-08-04T12:16:12Z`; control: **2** Mark comments exist, both ≤ watermark, so the empty result is the filter working). No rotation due (`#559` created `05:08Z` = **07:08 CEST**, after the Monday-07:00 LOCAL boundary; 48 < 80). `dev == origin/dev == a4e918e1b` and the running `SKILL.md` byte-identical to origin (`cmp` silent) — **the 08-03 reconcile has HELD for a THIRTEENTH consecutive iteration**, so Gate 4 wrote IN PLACE. Cheap tell case-correct with control (ITERATION 152 = 1, 151 = 2, rotation invariant 3). Died-mid-flight check clean on its **fourth** use: one open PR by this loop (`#545`, already queued), no `.wt-iter152-*` worktree. **RED dev OUTRANKED THE QUEUE and was the first deliverable — the diagnosis is the deliverable, not a fix, and rule 3d is why.** Four CI jobs went red; the seductive read was 'the pi-lane commit broke dev'. Measured instead: all four had **`steps=0`** — they never acquired a runner — and on the re-run `test` and `dependency-submission` died in **`Set up job`**, before checkout, with **zero repo commands executed**. Negative control: the SAME four jobs were **green on the parent commit `65f287107`** 76 minutes earlier, pre-incident. Corroboration: githubstatus reports **Actions major_outage**, incident opened `15:22Z`, our run created `16:04Z` — inside the window; `mission-world` iteration 58 independently hit the identical `Set up job` signature in a different repo. And the cleanest control of all: across re-runs the `docs` job went **cancelled → success on a byte-identical tree**, which is only possible if the variable is the environment. Re-runs fired; **Build and Release recovered to green with no code change**, which is itself the proof. No revert warranted. **PICK = the queue head (FINDINGS-BATCH iter-151), `#603` half.** `#602` deliberately NOT taken — it needs `go test ./...` under load, and the CI signal was unavailable all iteration. **THE TRIAGE, MEASURED NOT ARGUED**: 12 hostile shapes driven through the real handler. **The taint is REAL — 2 of 12 reflect a literal `<script>alert(1)</script>`** (malformed payload; unknown JSON-RPC method), so the battery is not vacuous and CodeQL is not hallucinating. **But it is NOT exploitable**: Content-Type present on **all 12** so Go never sniffs; both reflecting paths come from the SDK's `http.Error`, which sets `text/plain` **and** `nosniff`; and `encoding/json` escapes `<`,`>`,`&` by default — which is why the *tool-argument* case does not reflect at all. `DetectContentType` returned `text/plain` on every body, never `text/html`. **THE POINT: all three guards are inherited from `go-sdk`'s internals.** This wrapper copies headers wholesale and asserts nothing; a `go.mod` bump removes any of them silently. So the disposition is FIX, not dismiss — dismissing would have banked an inherited accident as a security control. **EXECUTOR = the NEW pi lane** (`pi:openrouter/deepseek/deepseek-v4-flash-0731`, Mark's directive delivered by message this fire and already live in env at HEAD): probe rc=0, run rc=0, directive 7,462 B delivery-asserted, zero git writes. **pi has NO sandbox, so I checked the main checkout for out-of-worktree writes** — byte-identical to preflight. Datapoint **2 of ≥3** for the lane; it followed a prescriptive directive exactly. **CONTROLLER RE-RAN EVERY GATE and ran TWO mutations the executor did not.** Mutation A (drop `nosniff`) reds **6 of 12** — the other 6 already inherit nosniff from `http.Error`, which is precisely the split the new line covers. **Mutation B is the find: deleting the Content-Type default reddened NO test in the entire package.** That branch is unreachable through the public HTTP path because the SDK always labels, so the guard I had just added was, by the mission's own standard, not a guard. Fixed by injecting a transport that writes an unlabelled HTML-shaped body, with a control asserting it **would** sniff to `text/html`; re-running mutation B now reds (rc 0→1). Every mutation proven to LAND by sha256 and reverted byte-identical. **Gate list DERIVED from `.github/workflows/ci.yml`, not recalled** — iteration 152's rule 3g on its first use, and `make check-changelog` was in the sweep this time. **EVALUATOR sonnet PASS 91/100 r1, zero blocking** (generator≠judge holds: DeepSeek/OpenRouter executor ≠ Anthropic judge — cross-provider independence RESTORED after iter-152 lost it to the codex outage). It reproduced both mutations and matched my counts exactly. **BOTH its non-blocking findings were ACTED ON, not filed** (rule 3b(b) — a NON-BLOCKING label is the judge's opinion of severity, not a measurement): (1) `writeMCPEnvelope`, the file's OTHER response path, echoes a request-controlled `id` and set Content-Type but **not** nosniff, so after my first commit the two paths in one file disagreed — verified first-party that the id IS escaped (`\u003cscript`, raw `<script>` absent), so not exploitable, but 'the encoder escapes by default' is exactly the inherited guarantee this sprint exists to stop trusting; now labelled and mutation-proven. (2) The battery case named 'reflected html in tool args' does **not** reflect the literal payload — my own earlier probe had shown the same and I had not noticed the name overstated it; renamed to say what it demonstrates. **MY OWN CONTROL FAILED ON FIRST RUN AND THAT IS WHY IT IS WORTH ANYTHING**: the new envelope test asserted a literal `<script>` in the body, which correct escaping guarantees is ABSENT — the test only became meaningful once the control asserted the *escaped* form. **Also my own error, recorded rather than buried**: I `Write`-ed the dashboard to `sunholo-data/design_docs/` instead of `sunholo-data/ailang/design_docs/`, creating a stray file outside the repo; removed immediately, repo working tree verified unchanged. **Routing evidence**: model=**claude-opus-5** · task-class=**execute** · executor=**pi `deepseek-v4-flash-0731`** · evaluator=**sonnet** · designer/planner **not fired** (bug-fix lane; rotation unchanged, next `claude:claude-fable-5`). `metered=$0.0057` of the $5 ceiling (pi, OpenRouter — real dollars, 21.8k in / 3.9k out / 169k cache-read). **Gate 3b: PR #606 polled bounded; a full green was NOT obtainable within the cap, so the item is NOT marked LANDED.** **EVERY failure observed on #606 carried the outage signature** — `govulncheck (vuln gate)` and `test-windows` both at `steps=1`, dead in `Set up job` before checkout, both logging `Failed to resolve action download info. Error: Service Unavailable` verbatim. Stated as a CLASS, not a count, because I corrected this row once from '0 failures' to '1' and it was 2 within minutes: **a failure count over an in-flight run is stale the moment it is written**, and the invariant (no repo command ever ran) is the thing that is actually true. Now proven by GitHub's own words rather than by inference: `steps=1`, died in `Set up job` before checkout, log reads `Failed to resolve action download info. Error: Service Unavailable` after two retries. Re-run was **REFUSED** (`This workflow is already running`) — owed once the run settles; and my own `rc=0` on that command was a **lie**, because `$?` read the `head` in the pipe rather than `gh` (step 3's exit-codes-through-pipes trap, tripped by me in the same iteration I wrote a rule about not banking claims). That message is the sharpest form of this iteration's new Gate-1 rule and was added to the record rather than left as a stale '0 failures'. **Next**: confirm #606 green and merge, then `#602`. **The RELEASE ask and `D-6` remain unanswered; `D-7` is NEW** — CodeQL analyses `dev` only weekly, so a Tuesday defect is invisible for ~6 days and first surfaces as noise on an unrelated contributor's PR.

## CURRENT GOAL

1. **Iteration 0 (definition)**: write the v1.0 bar (see "The v1.0 bar" below — draft to be
   ratified by Mark), re-score all 93+ planned docs against it into: `required-for-v1` /
   `nice-for-v1` / `post-v1`. Output: updated folder assignments + ordered queue in this doc.
2. **Then**: work the queue P0-first through the inner loop, one sprint-sized item per iteration,
   recording routing evidence every time.

## The v1.0 bar — v2, PRODUCT-SHAPED (RATIFIED 2026-07-11, Mark; supersedes the 2026-07-10 hygiene bar)

**The 1.0 claim**: ***the verified AI-orchestration language*** — an AI author gets a
verified-correct program at the lowest cost, and AI orchestration is type-checked. Derived from
[m-fable-strategy-review](planned/m-fable-strategy-review.md) (Design Freeze items 1+2 ratified
by Mark 2026-07-11: cost-per-success is the headline KPI; orchestration is the vertical. Item 3,
trace publication, stays deferred — post-v1).

**The cutoff rule**: a design doc gates v1.0 **only if it serves an open clause below.**
Everything else ships on the normal v0.2x road or is post-v1 — regardless of folder history.
The v1 hygiene bar (2026-07-10) is absorbed: its clauses are 1–2 below, both essentially done.

1. **STABLE** ✅ — the 1.x surface promise (docs/docs/reference/stability.md, iteration 5;
   tier assignments RATIFIED by Mark 2026-08-04, attended — clause 1 fully CLOSED; was: ratification parked for Mark).
2. **SOUND** — zero P0s ✅ (all four closed, iterations 1–4); residue: ~~m-check-strict-fallbacks~~
   **[LANDED iter 101, PR #479]**, m-bytecode-vm-parity-bugs (≤2d, queued).
3. **ACCESSIBLE TO THE FLEET TIER** (strategy R1+R4): the finite, documented mid-tier footgun
   list burned down — the 3 parser/type inconsistencies fixed (match-in-HOF-lambda parse,
   polymorphic-arithmetic panic, arity call-style diagnostic), m-syntax-ai-forgiving landed
   (kills the ~32% small-model failure class), and the teaching prompt ≤1,500 lines with a
   rig-A/B showing no pass-rate loss (R3.1 measures the curve first; the deletion pass stays
   gated on replacement diagnostics landing, per m-diagnostic-coverage's deferred section).
   **Gate = this finite work.** The sonnet-class ≥ −5pts outcome is measured and published at
   release, NOT blocking (per Mark: partially vendor-dependent).
4. **ORCHESTRATION FLAGSHIP** (R6 + R7 + effect refinement): the four effect sprints (public
   docs promise); a **verified multi-step AI pipeline** as the flagship example (typed LLM calls
   + budgets + secret-flow + replay) with orchestration benchmarks promoted into the default
   rotation and README/site positioning led by it; **linear-time regex + URL-parse builtins**
   (both verified absent — an orchestration 1.0 without them is a credibility hole).
5. **COST CREDIBILITY** (R3): the dashboard headline KPI flips to **cost-per-verified-success
   vs Python, per tier**, and v1.0 ships with the measured baseline + trajectory. The ≤3×
   zero-shot / ≤1.5× agent targets are the tracked post-1.0 trajectory, NOT release gates.

## How the mission runs (each iteration — codified in the mission-control skill)

1. **OBSERVE** — read this doc's backlog + last log entry + agent inbox + eval health. Deterministic, cheap.
2. **PICK** — top open queue item per the ordering policy. **Verify against repo reality first**
   (git log + code + tests), never trust a status header — stale-status docs are how we shipped
   M-EVAL-BENCH-UI twice (2026-07-10 lesson: doc said Planned, all 4 milestones were long done).
3. **ROUTE + EXECUTE** — through the honed inner loop with the model routing policy below:
   design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator. Sprint work runs in
   an isolated git worktree (concurrent-agent safety). Max 3 evaluator rounds, then park as
   `needs-human-review` and move on.
4. **RECORD** — append a log entry (fixed template in v1-mission-log.md): what shipped, evaluator
   score, routing evidence row, ruled-out ledger additions, next.
5. **RETRO** — route observed friction into exactly one lane: **skill fix** (edit the offending
   SKILL.md — max ONE skill edit per iteration, each traced to ≥2 recorded frictions), **process
   fix** (edit this doc), or **backlog item** (new/re-prioritized design doc). Then send the
   morning report to controlplane.

## Model routing policy (evidence-updated, not vibes)

| Role | Model | Why / evidence |
|---|---|---|
| Mission controller (this loop: triage, pick, judge, retro) | **Opus** — opus-first PREFS since 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"; the long orchestration session is mechanical and was the residual Fable drain even after M1a). Fable = emergency fallback only | The 07-14 Fable revert burned the weekly bucket at 2h cadence; orchestration doesn't need the top tier |
| Design docs (create/review) | **ROTATION across top-of-line models (Mark 2026-07-17)**: `claude:claude-fable-5` (via `claude-sub`) ⇄ `codex:gpt-5.6-sol`. **gemini caveat (iter 53):** G4 clone-over-egress LIVE-LANDED, so gemini is fleet-ready — but as an in-sandbox **evaluator**, not a designer: `CapRemoteSandbox` means it cannot edit a worktree, so a designer spawn can't write the doc without the text-bridge (unwired). gemini's designer-rotation entry is therefore PARKED for Mark's fleet-role ratification (evaluator recommended). Each new-doc iteration takes the next designer in rotation; record `(designer, quorum outcome)` in the evidence row | Every design passes the QUORUM regardless of author — the quorum is the quality gate, so authorship diversity is free comparative signal on which frontier model designs best for AILANG. Fires only when a doc is created/revised |
| Sprint planning | **Opus** (claude-opus-4-8) | Plan quality determined execution success historically |
| Sprint execution | **Opus** — the default, per Mark 2026-07-10 | Sonnet execution was a false economy (needed corrections); also `dev-cycle.md` had silently pinned sonnet |
| Sprint evaluation | **Sonnet** — `$MISSION_EVALUATOR_MODEL`-PINNED sub-agent (default changed fable→sonnet 2026-07-16, Mark directive #399; see below). generator≠judge holds STRUCTURALLY (sonnet ≠ the opus executor pin) AND is now ENFORCEABLE (sonnet is an Agent-tool alias; fable was not — F1 — so the fable default re-routed to sonnet every iteration anyway: 31, 36) | Behavioral independence (fresh sub-agent, re-runs tests, adversarial probes) retained on top |

> **✅ Evaluation-independence (RESOLVED 2026-07-16, iteration 38):** the evaluator default is now
> **Sonnet** — a distinct model from the Opus executor, so generator≠judge model-diversity is
> restored (the 2026-07-11 "Opus-evaluates-Opus rubber-stamp risk" is gone) AND it is *enforceable*
> (sonnet is an Agent-tool pin; the old `fable` default was not — F1 — so it silently re-routed to
> sonnet every iteration anyway; this makes that the standing state, not a per-iteration patch).
> Behavioral value (independent test re-runs, cross-history non-vacuity, distinct-sample recounts)
> is unchanged. Fable is retired from the every-iteration evaluator slot to protect the weekly quota
> (it fires every iteration, unlike the designer which fires only on new docs).
>
> **⚠ CORRECTION (2026-07-16 evening, Mark + interactive session): "Fable quota-exhausted until
> 2026-08-01" was a MISDIAGNOSIS — OAuth Fable was available the whole time.** The tell: OAuth
> buckets reset **weekly Monday 07:00**; an "until the 1st" date is the **API key's monthly
> cycle**. Root cause: `~/.zshenv` sources `secrets.env`, so every tool shell re-exports
> `ANTHROPIC_API_KEY`; nested `claude -p` calls (the `claude:` CLI lane) therefore billed the
> METERED API — iteration 37's fable designer+evaluator runs were API-billed $, and the key's cap
> then produced the fake "Fable exhausted" error. Fixed in the skill: every nested `claude` call
> now strips the keys at the call-site (`env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN`).
> **The `claude:claude-fable-5` designer lane is AVAILABLE again — do not treat Fable as gone
> until 08-01.** Any future "quota" error naming a reset date that is not a Monday = you are on
> the API key; fix the leak, don't fall back.
| Mechanical tasks (doc moves, regen, banking) | Sonnet allowed | Only with deterministic verification; promotion beyond this requires evidence |

**DEMAND-EVIDENCE GATE (Mark 2026-07-23 — ?-op, block-let-separator, and |> ALL failed it in one week):** ERGONOMICS-SUGAR items (new operators, syntax conveniences) require DEMONSTRATED DEMAND BEFORE PICK — a 60-second corpus grep (parse-error/usage counts in eval banks) is mandatory Gate-2 evidence; zero-demand items go to EVIDENCE-GATED ICEBOX without spending a quorum round. Technical soundness is not the bar; observed need is.

**FORUM RULE (Mark 2026-07-21 — "it takes too long to do quick tasks like ab evals… reserve
this for features"):** the full iteration pipeline (quorum → plan → execute → evaluate → CI) is
for FEATURE-shaped work: code that ships, with quality risk worth the ceremony. Quick
EXPERIMENT/EVAL-class tasks (A/Bs, probes, demos, measurements) do NOT ride full iterations —
they run DIRECT: an interactive session, or a single bounded controller-lane step banking results
+ one evidence row. The fmt A/B's 3-day design→park→greenlight→integrity→execute arc vs the
58-cent 20-minute interactive demo is the canonical example. In-flight experiments finish as
planned; new ones default to the direct lane.

**Evidence rule**: every sprint's log entry records `(model, task class, evaluator round-1 score,
rounds-to-pass, corrections)`. A routing change (either direction) requires ≥3 data points and is
made in RETRO, recorded here with a dated stamp.

> **ENFORCED 2026-07-15 (m-mission-agentic-provider-routing M1):** this table is no longer prose.
> The driver exports `$MISSION_PLANNER_MODEL` / `$MISSION_EXECUTOR_MODEL` / `$MISSION_EVALUATOR_MODEL`
> and mission-control Gate 3 spawns each heavy role as a model-PINNED sub-agent (the controller
> session runs `$MODEL` only). **Before M1, every role inherited the single session model → 100%
> Fable burn** (the driver had been Fable-first since 07-14). Execution now bills the executor pin
> (Opus), not the controller (Fable); generator≠judge is restored (Fable evaluator ≠ Opus executor).
> M2 extends the evidence rows with `(provider, agent, $/quota)`; M3 A/Bs the **sprint-planner
> down-tier** (kept at Opus until ≥3 datapoints — do NOT lower it on this hypothesis alone).
> Cross-provider AGENT executors (codex/motoko/managed_agents) ride the same env once fleet Phase C
> resolves a value like `codex:gpt-5.6` in the spawn.
>
> **AMENDED 2026-07-16 (Mark: "Fable for real high cognition stuff not execution"):** the
> controller session itself was the residual Fable drain after M1a (a ≤6h mostly-mechanical
> orchestration session on the scarcest model). Driver PREFS are now **opus-first**
> (`claude-opus-4-8,claude-fable-5`; Fable = emergency fallback only) and design-doc-creator moved
> from inline to a **`$MISSION_DESIGNER_MODEL`-pinned sub-agent (fable)**. Net Fable spend per
> iteration = ~~two bounded sub-agents: designer (only when a new doc is needed) + evaluator~~
> ONE bounded sub-agent: the **designer** only (fires only when a new doc is needed). The evaluator
> moved OFF Fable to **sonnet** in iteration 38 (below) — this also RESOLVES the iteration-36/37
> inconsistency between this clause and the "evaluator→sonnet unless ≥3 datapoints" rule (Mark's
> #399 directive settles it: not fable).
>
> **AMENDED 2026-07-16 iteration 38 (Mark directive #399: "once we have gemini via managed agents
> and openai we can use one of those instead for evaluator? so default can be gemini (if able to git
> clone the codebase etc)? otherwise sonnet-5"):** evaluator default moved **fable → sonnet**.
> gemini (managed_agents) — Mark's *preferred* default — is NOT viable as the evaluator today, on
> two independent counts VERIFIED this iteration: **(1) architectural (code-proven)** — the
> managed_agents request body carries only `Directive`+`SystemPrompt` over a server-side
> `CapRemoteSandbox` (`internal/executor/managed_agents/managed_agents.go:164`); there is no repo
> upload, so the agent cannot see the sprint's UNCOMMITTED worktree changes nor re-run local tests
> (at most it could `git clone` the *public* origin/dev, which lacks the changes) — exactly the
> "if able to git clone the codebase" gap Mark flagged; **(2) operational (live-observed)** — a
> bounded `ailang exec gemini` probe timed out (`http2 timeout awaiting response headers`), same
> class as iterations 36-37. Per Mark's own ladder this resolves to **sonnet-5**. gemini-as-evaluator
> is a queued follow-up (**m-gemini-evaluator-diff-bridge**): needs a bridge that ships the sprint
> diff + changed files into the directive text AND the Vertex backend returning reliably. NOTE:
> **codex (openai)** is a viable local distinct-provider evaluator alternative (it runs a sandboxed
> local CLI → CAN read the worktree + re-run tests; openai≠anthropic satisfies generator≠judge) —
> but Mark's stated default ladder is gemini→sonnet-5, so sonnet is the default; codex-as-evaluator
> requires the executor NOT be codex (generator≠judge) and stays opt-in.

### Right-sizing table — the (provider, agent, tier) hypothesis (M2)

Landed 2026-07-16 (m-mission-agentic-provider-routing M2). This is the *hypothesis* that the routing
evidence rows below test — updated by the ≥3-datapoint evidence rule, never by vibes. Canonical source:
[design_docs/planned/v0_30_0/m-mission-agentic-provider-routing.md](planned/v0_30_0/m-mission-agentic-provider-routing.md).

| Role | Agentic? | Needs | Tier hypothesis | Agent candidates |
|---|---|---|---|---|
| Controller (pick/judge/retro) | agent (claude-code) | orchestration judgment | **mid** | claude-code (home harness) |
| Design-doc-creator | agent (`check` in loop) | deep spec reasoning (highest leverage) | **strong** | strong claude/codex + live quorum |
| **Sprint-planner** | agent-capable | decompose a quorum-reviewed doc | **MID (down-tier)** — kept at Opus until M3's ≥3-datapoint A/B | mid codex/gemini/motoko |
| Sprint-executor | AGENT (heavy) | tool-using coding | **strong AGENT** (not just a model) | **codex / motoko / claude**; motoko may over-perform on AILANG (M1b wired codex) |
| Sprint-evaluator | AGENT (re-runs tests) | behavioral verification | **mid**, distinct provider from executor | gemini/codex ≠ executor |
| Mechanical (moves/regen) | no | deterministic | **low / local** | local-GPU (Phase D) |

> The model-routing table above (Opus-first) is the CURRENT enforced assignment; this right-sizing
> table is the tier *hypothesis* those assignments are converging toward as evidence accrues. Where
> the two differ (e.g. controller runs Opus today but the hypothesis is mid-tier), the gap is a
> deliberate, evidence-gated decision — a routing change requires the ≥3-datapoint rule.

## Rig integration — the two-tier rule

`rig.lock` (`~/.ailang/state/rig.lock.d`) is a **GPU mutex, nothing more** (Mark, 2026-07-10).

1. **Default iteration (cloud models: Fable/Opus coding, `make test`, git): NEVER touches
   rig.lock.** CPU/disk co-tenancy with the eval rotation is fine; the loop runs 24/7 without
   starving the rotation and vice versa.
2. **GPU-touching steps only** (a sprint whose acceptance includes local-model validation, wire
   diagnostics, anything driving ollama): `rig_lock_acquire wait` for **that step only** — never
   held across a whole sprint. Same discipline as `os-rotation-filler.sh`.

Hygiene: a sprint must not *accidentally* reach the GPU (the port-8080-zombie class). "Does this
step touch the GPU?" is an explicit routing question in the skill, not an accident of what a test
invokes.

## Guardrails (the loop may not…)

- **No releases** by the loop — but a rolling release cadence (Mark, 2026-07-12): the loop lands
  to `dev` continuously and never cuts a release; **Mark snapshots interim releases (v0.30.x,
  v0.31.x…) as needed**, each carrying whatever's accumulated. **v1.0.0 is a MILESTONE declared
  when all five bar clauses are satisfied — not a single big-bang release.** Implications: (1) dev
  must stay release-ready at EVERY commit — the "Dev stays GREEN" guardrail already enforces this
  and it is now load-bearing (any commit may become a release point); (2) each iteration's #329
  report should note when it CLOSES a bar clause (e.g. "clause 3 footgun burn-down: N of M
  landed"), so Mark can watch the bar fill and time the v1.0 call; (3) a version bump mid-mission
  is expected, not a stop signal — the loop already handled v0.29.0/.1/.2 landing between iterations.
- **No pushes without account check** (`gh auth status` → `sunholo-voight-kampff`).
- **No work on a dirty main worktree** — sprints run in coordinator-managed worktrees; the
  controller session itself is read-mostly + doc edits.
- **Budgeted**: hard wall-clock kill in the driver (default 6h); one backlog item per iteration.
- **Kill switch**: `touch ~/.ailang/state/mission-control.disabled` (checked in preflight) or
  `launchctl unload ~/Library/LaunchAgents/dev.ailang.mission-control.plist`.
- **Subscription billing only** (2026-07-10): the nightly bills the Claude subscription via
  `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, stored in secrets.env) — the driver strips
  `ANTHROPIC_API_KEY` and refuses to start without the token. The first kickstarted run billed
  ~13 min of API credits before this was caught; never again.
- **Escalation**: evaluator `needs-human-review`, merge conflicts, or any guardrail trip →
  `ailang messages send controlplane`, park the item, pick the next; never force through.
- **Skill edits**: max one per iteration, ≥2 recorded frictions each, called out in the morning
  report (git history is the rollback).
- **Dev stays GREEN** (2026-07-10, Mark): an item is not [LANDED] until remote CI passes on its
  merge commit (Gate 3b — local gates miss fmt-check/govulncheck/file-sizes/docs build), and a
  red dev CI outranks the queue at OBSERVE, including time-based reds from newly published vuln
  advisories.
- **BENCHMARK CURATION CYCLES RUN THROUGH THE LOOP, NOT AS ATTENDED SIDE-SESSIONS** (RATIFIED
  2026-08-04, Mark: *"Route curation through mission loop"* — his one-line answer to iteration 140's
  DECISIONS ask). A curation cycle (tier promotion/demotion, retirement, rotation — the operations
  `benchmarks/CURATION.md` governs) is a **queued mission item** from now on, picked and routed like
  any other, never applied by a concurrent attended session. **The evidence is iteration 140's whole
  iteration**: `f574c4b58` (the v0.32.0 curation cycle, run from a non-mission session) moved 12
  benchmarks between tiers and updated **neither** of the two gates that pin the tier distribution
  (`TestAllBenchmarksHaveTierAndTags`, `TestFilterBenchmarksByTier`) — dev CI was red on every commit
  for ~2h, iteration 139 had already misfiled that red as a known runner flake, and v0.33.0 came
  within minutes of shipping on a red dev. The defect was 0.5s-reproducible and the tests document
  their own remedy; nothing about it was hard *except that nobody whose job it was ever saw it*.
  Two consequences: (1) the curator inherits the loop's gates — Gate 2's reality-check, Gate 3b's
  SHA-addressed CI green, and the Gate-4 record — so a tier move cannot land without the
  distribution gates being re-centered in the same commit; (2) `benchmarks/CURATION.md` §5
  *"Applying tier moves — REQUIRED follow-through"* (added iteration 140) stays the operative
  checklist, and this guardrail is what guarantees somebody actually reads it. Attended curation is
  still fine as *authoring* — writing or scoring benchmarks — but the **tier-move commit** goes
  through the loop.
- **A POSITIVE result from ONE confirming instance is not a general claim** (process fix, iteration
  122; the mirror-image of the skill's rule 3a, which covers *empty/negative* readings). Rule 3a
  made this loop good at distrusting silence — an empty `grep`, a failed handshake, a vacuous pass.
  Iteration 122 produced **three** misses of the opposite shape in one run, all from *positive*
  evidence generalised past what it supported: (1) I reproduced a `case`-vs-`match` diagnostic
  cascade first-party with a clean positive control, then inferred it was the *sustained* cause of a
  7-night benchmark failure — refuted by my own per-night scan, which found `case` on exactly one
  night; (2) I wrote `#538`'s headline ("a strictly worse benchmark gets the quieter label") from the
  0/10-vs-1/10 boundary alone, without reading the design rationale that made the asymmetry
  deliberate; (3) I proposed replacement text for the very message whose falseness I had filed the
  bug about, and **my replacement was also false** — and then the correction I posted was wrong on
  mechanism, which only the evaluator's reachability check exposed. The tell is identical each time:
  *one measurement that is true, restated as a claim about all cases.* Before a positive finding
  becomes a general claim — especially one handed to a sub-agent or written into an issue — ask
  **"how many instances is this true of, and did I count them?"** and prefer the cheap census (all N
  nights, all call sites, all reachable shapes) over the single confirming probe. Corollary earned
  the same iteration: **a reachability question is a census question** — "can this path fire?" is
  answered by enumerating the state space (19,607 streams → exactly one escalation pair), never by
  finding one case where it does. Watch-item, not yet a skill edit: needs one more independent
  instance before it earns the one-edit-per-iteration slot.

## Backlog ordering policy

1. Open **P0s** first (list above), oldest-known-risk first.
2. **Unblockers** — items other queued items depend on (e.g. m-effect-row-poly-params blocks
   sunholo/demos).
3. **P1 by impact-per-day** (the census has estimates; prefer ≤3-day items to keep iterations
   sprint-sized).
4. **Strategic multi-week items enter only after decomposition** into sprint-sized design docs
   (a decomposition is itself a valid iteration deliverable).
5. Anything re-scored `post-v1` in iteration 0 leaves the queue.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**RATIFIED (attended, 2026-08-03 evening) — Mark's two one-word rulings on the iter-135 digest asks:**
1. **Standing fast-forward authorization: YES** — when local dev is **0 commits ahead** AND the
   working tree is clean apart from the known rig-synced dirty files, the controller MAY
   `git merge --ff-only origin/dev` without a per-instance ask (safe by construction: nothing
   local exists to lose). ANY other state still triggers Critical Principle 0. Stop re-asking.
2. **recorded-stream S2 does NOT jump the queue: NO** — order stands `#498 Lane B` → S2
   (S1 already covers the author's primary need; #498 remains World's sole clause-6 blocker
   per the 2026-08-03 directive).


**Required-for-v1 (the bar's critical path):**

1. [LANDED 2026-07-10] m-named-test-blocks closeout (iteration 1a; deontic criterion deferred,
   package absent locally)
2. [LANDED 2026-07-10] m-typeenv-sub-fix (iteration 1b: RESOLVED — pre-closed by adjacent
   M-TYPE-LIST-SOUND work, regression-guarded, eval 92/100, merge f59421ac8)
3. [LANDED 2026-07-10] m-feedback-triage-gate (iteration 2: full loop headless, eval 93/100
   PASS round 1, merge 40f1cdc3f, remote CI green on dev post-merge; gate logic complete +
   merged, off by default — production activation gated on the next item)
4. [LANDED 2026-07-10] m-feedback-gate-cloud-adapter (iteration 3: full loop headless, round-1
   eval FAIL → round-2 PASS 97/100, merge 842d7d501, dev CI fully green on 4c22032de; gate
   code complete incl. cloud adapters, OFF by default — production enablement is a HUMAN ops
   task: sibling-repo terraform TTL + ANTHROPIC_API_KEY secret, then DRY_RUN=1 week 1)
5. [LANDED 2026-07-10] m-diagnostic-coverage (iteration 4: M1–M3 found pre-shipped 2026-07-09
   under a stale "Planned" status; remainder sprint M-DIAG-FIXTURE-PROMOTION promoted 4 rows
   to covered — 7 CI fixtures across 6 footgun rows — eval PASS 96/100 round 1, PR #336 →
   fe807aac8, dev CI green per-workflow. DEFERRED, rationale in doc: deletion pass + rig A/B
   until deletable surface ≥ 100 lines; PARKED for human: haiku causal re-run, API-billed)
6. [LANDED 2026-07-10] m-v1-stability-promise (iteration 5: FULL loop headless round-1 clean —
   Fable design doc → Opus plan (caught 2 premise errors: 42 modules not 39; LIMITATIONS
   double-maintained + diverged, both copies fixed) → Opus execute in worktree → Fable eval
   PASS 96/100 round 1. Stability page docs/docs/reference/stability.md (3 tiers, full stdlib +
   CLI tables), both LIMITATIONS files live-verified at HEAD, 4 website vX-promises retracted,
   PR #337 → fcccd7208, dev CI green per-workflow. PARKED for human at RELEASE gate: tier-
   assignment ratification — ⚠ proposed: std/net, crypto, jwt, xml, zip, process, CLI
   watch/serve-api)
7. [LANDED 2026-07-11] m-effect-refinement **decomposition** (iteration 6: repo-verified phase
   census — P1/P2 + AI port shipped v0.15.0 under the parent's stale "Planned"; P7 CryptoRand
   never existed (m-cryptorand.md swept to implemented/ in error — header corrected); P6 routed
   OUT to M-ENTROPY. Remaining ~64h split into 4 sprint docs (below, items 9/12/13/14) with live-
   verified premises; parent doc now the umbrella. BONUS finding: the public guide's "typechecker
   rejects unknown values" is FALSE (`Rand[mode=banana]` passes check) — interim accuracy note
   shipped, enforcement is sprint 1)
8. [LANDED 2026-07-11] m-eval-frontier-tier (iteration 7: full loop headless, round-1 clean —
   Opus plan (9 discrepancies) → Opus execute (frontier tier + 8 re-tiered + prefix_line
   structural grader + 7 core→stretch demotions via 4-dim rule from banked data) → Fable eval
   PASS 96/100 round 1 w/ independent distinct-sample recount. PR #339 → 0515578ae, dev CI
   green per-workflow. PARKED for human: frontier-failure validation of the 8 (API-billed —
   each must fail ≥1 frontier model or demote back per CURATION.md §5) + 4 remaining sketches)
*(Queue re-derived 2026-07-11 from bar v2 — clause tag on every open item. NEW-DOC items start
with design-doc-creator; existing-doc items start at reality-check.)*

9. [LANDED 2026-07-11] m-effect-mode-validation (iteration 8: full loop headless, round-1 clean —
   Opus plan (2 discrepancies: bridge carries no params, scope-reduced; EFF_* codes frozen) →
   Opus execute (effectSchema + validateEffectParams at elaboration, 3 fix-carrying diagnostics
   CI-fixtured, guide truth-up: the public closed-set claim is now TRUE, prompt names the codes)
   → Fable eval PASS 96/100 round 1 w/ independent transcript re-production. PR #340 → 8faa49de9,
   dev CI green per-workflow. Unlocks effect sprints 2-4. BONUS: dev-health issue #341 filed
   (5 pre-existing example type-check failures; verify-examples not a CI gate))
10. [LANDED 2026-07-11] m-syntax-ai-forgiving (iteration 9 — the first iteration SPLIT ACROSS
    TWO scheduled runs: run A did reality-check 192a79149 + Opus plan a7bd8257c + Opus execute
    (worktree, M1–M4 64ddd6021) then died pre-evaluation; run B resumed at sprint-evaluator.
    Fable eval PASS 96/100 round 1 (FIFTH consecutive) w/ independent fuzz-gate re-run (zero
    AST diffs over 389 currently-valid corpus files), rebuilt-binary transcripts, non-vacuity
    vs v0.29.2 (PAR017/PAR020 fire on exactly the now-accepted fixtures). R1+R2 BOTH landed —
    R2 systemically patched FOUR block loops (plan's D6 knew two; if/then + \-lambda route via
    parseRecordLiteral). PR #342 → merge, dev CI green per-workflow. DEFERRED: ailang fmt →
    m-ailang-fmt.md stub (D1). PARKED for controller/human: the rig A/B compile_error Δ on
    ;-family benchmarks — the REAL success metric, GPU step, rotation held the rig)
11. [LANDED 2026-07-11] m-stdlib-regex (iteration 11: full loop headless, round-1 clean — Opus
    plan (3 de-risking findings: F1 `_str_slice`/`_str_len` are RUNE-indexed but Go `regexp`
    returns BYTE offsets → span conversion is load-bearing; F2 embed is a glob; F4 changelog
    path) → Opus execute (worktree: 6 `_regex_*` builtins in the MODERN `internal/builtins/`
    RegisterEffectBuiltin system — NOT the doc's outdated `internal/eval` path, **D-ARCH**;
    memoized RE2 cache; `std/regex.ail` + 3 examples incl. the log-orchestration clause-4 use
    case) → Opus eval PASS 97/100 round 1 w/ INDEPENDENT reproduction (backref reject, CJK
    `日本語 world` rune span [4,9) not byte [10,15), findAll). PR #343 → squash-merge 0b0ed7ea0,
    all required checks green. `std/regex` = linear-time (RE2): compile/isMatch/findFirst/findAll/
    replaceAll/split; RE2 subset (no backref/lookaround) → `compile` Err, never panics. Docs:
    LIMITATIONS + stability (Experimental) + CHANGELOG. Design → implemented/v0_30_0)
12. [LANDED 2026-07-12] m-stdlib-url-parse (iteration 13: full build loop headless — Opus executor
    (worktree: `_net_url_parse` + `_net_url_parse_query` pure builtins in the modern
    `internal/builtins/net.go`, wrapping Go `net/url`; `Url` record + wrappers in `std/net.ail`;
    26 non-vacuous tests incl. IPv6 `[::1]:80`, error-never-panics, order+dupe preservation,
    round-trip; 2 examples; docs) → independent Opus evaluator round-1 FAIL 80/100 (single BLOCKER:
    stale `builtin_types.golden` not regenerated → repo-wide `make test` red) → round-2 golden
    regen → PASS 100/100. Design → implemented/v0_30_0. PR #347 → squash-merge `a8628a40c`,
    auto-merge on green required checks. `std/net` now parses URLs: `parseUrl(s) -> Result[Url,string]`
    (Err on malformed, never panics/fallbacks — CP2; `port:string` ""=absent) + order-preserving
    percent-decoded `parseQuery` (inverse of `urlEncodeForm`). Pure `! {}`, no Net cap. Closes v1.0
    bar clause 4's URL-parse half (regex half = #11). BONUS finding: `cmd/ailang`
    `TestRunCommand_PipedStdoutFlushesPerLine` is a pre-existing flaky under parallel `make test`
    load — passes 3/3 in isolation, unrelated to this sprint; flagged for dev-health, not a gate)
13. [LANDED 2026-07-12] m-module-less-run-fail-loud (iteration 14: full build loop headless, round-1
    clean — reality-check caught the doc's **MOD011 collision** (already the module-path-collision
    code) → reassigned **MOD014**; Opus plan → Opus execute (worktree: `validateModulePath` early-
    accept replaced with a loud MOD014 error gated on `len(Funcs) > 0`, fires for both `run` AND
    `check`; the doc's 3-way `Funcs||Statements||Decls` guard was code-refuted mid-sprint — a bare-
    expr FILE does reach `validateModulePath`, so the OR would break `ailang run 1+1`; block_demo
    remediated; footgun fixture 17→18) → independent Opus evaluator PASS 100/100 round 1 w/ a
    base-origin/dev binary proving test non-vacuity + pre-existing-failure claims. PR #349 →
    merge `c2ffd1b5c`, post-merge dev CI green per-workflow. Design → implemented/v0_30_0. Module-less
    files now FAIL LOUDLY (CP2). Skill-fix: design-doc-creator error-code + mechanism verification gates)
14. [LANDED 2026-07-12] m-match-xcheck-error-quality (iteration 15: full build loop headless, round-1
    clean — Gate-1 origin-sync caught local dev 4 commits behind origin/dev (iter 14 landed via #350),
    read state from origin; reproduced the empty `Option's constructors are: ` line live at HEAD →
    **Option A** (design doc's own recommendation): a diagnostic-only `Constructor→ADT` registry
    (`moduleImports.AllCtorTypes`) built from ALL transitively-loaded ifaces via
    `modLinker.GetLoadedModules()`, plumbed via new `SetDiagnosticConstructorTypes`, consulted by
    `lookupADTConstructors` ONLY when the primary direct/local scan is empty — never enters scope
    (`types` can't import `link` → passed as a plain `map[string]string`). Opus plan → Opus execute
    (worktree, commits `3ded459cc`/`f5498ca0e`/`ecca08b3b`) → independent Opus evaluator **PASS 96/100
    round 1** w/ base-binary non-vacuity proof + scope-non-leak + format-unchanged checks; 2
    non-blocking deductions folded into the hardening commit
    (`TestSchemeImport_DiagnosticRegistryDoesNotLeakIntoScope` + collision note). PR #352 →
    squash-merge `5aaaff2ed`, required checks green (auto-merge). Design → implemented/v0_30_0.
    Foreign-ctor errors now enumerate transitively-known constructors (`None, Some` + did-you-mean).
    SonarCloud PR gate red = advisory/non-required (merge succeeded) — flagged for sonarcloud-triage)
15. [LANDED 2026-07-13] m-module-let-func-resolution (iteration 23: full build loop headless, round-1
    clean — first CI-red fix-forward (gofmt miss from `366c5bbb2` broke dev fmt-check 2 runs →
    `39171a4f9`, observed green); Opus plan (caught the design doc's WRONG test path: the #327
    40-cell matrix is `internal/pipeline/record_update_positions_test.go`, NOT `internal/types/`;
    proposed MOD007 from the reserved block) → Opus execute (worktree: M0 spike **GO** — evaluator
    binds any `core.Let`, `CheckCoreProgram` threads forward env → unified SCC over lets+funcs,
    `wrapInLets` + BOTH re-elaboration loops DELETED; module `letrec` SUPPORTED via `core.LetRec`;
    dup module-scope name → **MOD007** hard error, zero corpus collisions; hint truth pass — 0
    `known bug #327` hits, residual hint cites #366 + real workaround "declare it as a func") →
    independent **Fable** evaluator (model diversity restored — controller reverted from Opus)
    **PASS 98/100 round 1** w/ own worktrees+binaries, base-binary non-vacuity (v3/v7/v8 fail at
    `116ebcb49` → run 16/0/4 post-fix; v10 silent shadow → MOD007), adversarial probes (func→let→func
    topo chain, let↔func cycle → LetRec no crash, effectful module let rejected identically).
    PR #368 → squash-merge `fd38ec14e`, post-merge dev CI green per-workflow. Design →
    implemented/v0_30_0. Module lets now resolve module funcs uniformly (4th family member CLOSED).
    ⚠ PICK-ORDER MISS recorded: Mark's [NEXT-FIRST] below (added 13:04, pre-session) should have
    outranked this pick; Gate-2 read the queue head + prior log's Next but not the fresh directive.
    Sprint was already through eval when caught → landed; iteration 24 is HARD-PINNED to it)
**[LANDED 2026-07-13, iteration 24 — was Mark's NEXT-FIRST, ⚠ missed by iteration 23, taken
first by iteration 24 as pinned]** m-public-feedback-delivery-audit
([implemented/v0_30_0](implemented/v0_30_0/m-public-feedback-delivery-audit.md), P1): full inner
loop headless, round-1 clean — Opus plan (killed 2 feared ops steps: prod sub exists, ADC owner
on both projects; corrected "structural, not novel" → real multi-project fan-in) → Opus execute
in worktree (Defect A: `isExternalFeedbackInbox` tags `pkg:*` as `public-feedback`, allow-list
untouched; Defect B: `Daemon` N-message-sources refactor + `firestore.NewClientForProject` +
opt-in `extra_message_envs`/`--also-subscribe`, default OFF byte-identical) → Fable evaluator
**PASS 97/100 round 1** (base-binary non-vacuity both defects; 0 test deletions; conflict surface
intact). PR #378 → `4fee247a8`, post-merge dev CI green per-workflow (observed). ⚠ PARKED for
Mark: daemon reload + 2 live prod test-sends (checklist: sprint plan §Parked-for-human +
docs/docs/guides/notify-daemon.md); until reloaded, prod feedback still doesn't ping — the CODE
is landed, the OPS switch is human.

16. [LANDED 2026-07-13] iteration 25 — **R4a+R4b GHOST-CLOSE + m-lambda-open-record-pattern
    EXECUTED**: Gate-2 reality check live-probed the queue's R4 rows (the sourcing strategy
    review admitted they were never individually re-verified) → R4a `m-dx-match-hof` GHOST
    (retired `match … with` syntax was the culprit; design doc archived Not-Applicable
    2026-05-09; `\x ->` already has a teaching diagnostic) + R4b `m-poly-arith-lambda` GHOST
    (fixed v0.7.0) — guards `examples/match_hof_lambda.ail` + `poly_arith_lambda.ail`, PR #379
    → `ea8116f83`, CI green observed. Then the full inner loop on m-lambda-open-record-pattern
    (REAL at HEAD; mislabeled NEW-DOC — full design doc existed at planned/v0_29_0): Opus plan
    (refuted the doc's H3-primary via an IIFE probe) → Opus execute (found the TRUE primary
    site absent from doc+plan: `unifyRecord` rejected on field-count BEFORE consulting row
    variables; `core.RecordPattern.Rest` + `unifyOpenRecords` row-polymorphic subsumption;
    closed-pattern strictness preserved) → independent Fable evaluator **PASS 92/100 round 1**
    (own base+sprint worktrees/binaries, non-vacuity both directions, 8 adversarial probes, 0
    test deletions; found an arm-order-dependent acceptance) → hardening commit `89b75bd3f`
    (order-independence fix proven load-bearing, dead-code removal, cacheKeyVersion v2 for the
    gob-struct change). PR #380 → `47576e25d`, dev CI green per-workflow observed. Design +
    sprint plan → implemented/v0_30_0.

**[LANDED 2026-07-16 (M1a+M1b+M2) / M3 PARKED-protocol]** **m-mission-agentic-provider-routing**
([planned/v0_30_0](planned/v0_30_0/m-mission-agentic-provider-routing.md)) — mission-infra P0.
Fixed the routing-never-enforced bug (memory `project-mission-routing-table-never-enforced`).
**M1a LANDED 2026-07-15** (interactive, 8ee07ef23 + amended d545d4a9e): per-role env pins, opus-first
controller, fable designer/evaluator by inheritance. **M1b+M2 LANDED 2026-07-16 iteration 31**
(direct-on-dev main checkout, zero Go — the planner found registry/DryRun/codex executor all
pre-exist since v0.22.0): Gate-3 `provider:model`→bounded `codex exec` recipe (probe live-verified:
gpt-5.6-sol exit 0; default-env fire = no-op, codex strictly opt-in) `956fda55c` + charter
right-sizing table & provider/agent/cost evidence-row schema `8d12e8e9c`; eval PASS 87/100 round 1;
hardening `1c964aae2` — **F1: the Agent tool pins only sonnet|opus|haiku, `fable` is REJECTED**
(fable roles run by session inheritance only; alias-lane generator≠judge guard added: evaluator
never falls back to bare $MODEL, re-routes to sonnet + FLAG) + F2 `exec` orphan-kill fix.
**Open by design**: first REAL cross-provider fire (opt-in `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol`,
= the doc's M1b acceptance) · **M3** (planner down-tier A/B) PARKED with a concrete protocol in the
sprint plan until 3 quorum-reviewed docs accrue. Doc stays in planned/ until those close.

**[NEXT-FIRST, Mark 2026-07-16 — FLEET ROLLOUT ("should be awesome")]** The ratified starting
fleet is **claude (Anthropic) + codex gpt-5.6-sol (OpenAI) + managed_agents gemini (Google) +
motoko/qwen3-6 (local GPU)**. Sequenced, one per iteration:
- **(a) ~~Iteration 32 — codex LIVE-FIRE~~ DONE 2026-07-16**: FIRST real cross-provider fire landed.
  `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol` (one-shot override consumed) executed `20251013_auto_caps`
  M1 (`--caps auto`) end-to-end: Opus planner → **codex/gpt-5.6-sol executor** (OpenAI, ~4.5-min run,
  metered) → **Sonnet evaluator** (generator≠judge: openai≠anthropic; fable pin unenforceable →
  re-routed to sonnet + FLAGGED per the F1 guard) PASS 98/100 r1. PR #397 → `e542065c0`. **Recipe
  frictions found & fixed (Gate-5 skill edit): the codex real-run recipe had only ever been verified
  against the text probe** — a real coding run needs `--sandbox workspace-write` + `--add-dir` for
  GOCACHE/GOMODCACHE, cannot self-commit (worktree `.git` lives under the main checkout →
  controller finalizes the commit from the uncommitted worktree diff), and must run backgrounded
  (the 30-min cap exceeds the harness's 10-min foreground bash limit).
- **(b) ~~M1c — gemini managed_agents recipe branch~~ DONE 2026-07-16 (iteration 33, PR #398 →
  `bd89418a6`)**: the "no new plumbing" claim was REFUTED — `ailang exec gemini` (agentic) was
  unreachable (`unknown executor: gemini`; managed_agents registers under its own name, no gemini
  alias). Landed a real ~30-LOC `exec.go` fix (`resolveAgenticExecutorName`: gemini→managed_agents,
  `--api-only` untouched) + test + the Gate-3 `PROVIDER=gemini` recipe branch. **CapRemoteSandbox
  scoping**: the lane serves READ-ONLY roles (evaluator/reviewer/quorum-verifier) only — the
  server-side sandbox never writes the local worktree, so the file-editing executor role needs a
  bridge (follow-up). Sonnet eval PASS 96/100 r1. First LIVE gemini fire deferred to (c).
- **(c) [CORE LANDED 2026-07-16 iter 36 (M1-M3) — PR #400 → `0e83a1b12`; M0/M4/M5 now UNBLOCKED — (c0) plumbing landed iter 37, this is the `[← NEXT fleet step]`]** m-mission-quorum-agentic-verify+HONE
  — **M1-M3 shipped**: `agenticCaller` behind the `JSONCaller` seam (frozen verdict JSON via the coordinator
  executor layer), `ShouldEscalate` two-tier trigger + additive-optional `proposed_fix` (option (a), contract
  frozen), Tier-2 codex+claude read-only verify. 43 tests pass, verdict contract independently verified
  unchanged, evaluator PASS 91/100 r1. **M0 (gemini network probe) BLOCKED**: `ailang exec gemini` fails
  `GCP project not set` — `cmd/ailang/exec.go` never plumbs `Task.GCPProject` outside the eval harness
  (fix = item (c0)). Once (c0) lands: M0 (live gemini probe) → M4 (conditional on M0 result) → M5 (live-fire
  + doc → implemented/). Watch items carried: `agentic_caller.go:85` ctx.Background→caller-ctx before a live
  Tier-2 fire; `premiseSignals` breadth; M4 fallback must carry an explicit `VerificationDegraded` marker.
  — iteration 34's Gate-2 quorum-at-pick park is RESOLVED: Mark chose **(a) `proposed_fix` optional,
  not validated, contract frozen** (doc's HONE section stamped; the code-cited Verification-Log rows
  for the refuted sol objection added — provider_executor.go exposes ctx-cancel/Timeout/CostUSD/
  read-only-AllowedTools/WorkingDir, reuse premise HOLDS). Doc is quorum-cleared for routing: **start
  at sprint-planner** (both quorum rounds + revisions already done; do NOT re-quorum — the two rounds
  + resolved authorial decision ARE the quorum outcome). M0 = the managed-sandbox network probe (doc
  §Agentic reviewer backend). Meta-finding stands (text quorum blocked premises TRUE-in-code — the
  motivating case for this doc). Meta-finding (Gate-5): the TEXT quorum-at-pick blocked a doc whose premises are TRUE-in-code
  precisely because text reviewers can't read code — the motivating case for this very doc. Original ask:
  reviewers become tool-using agents (codex/managed_agents/
  claude-CLI, read-only worktrees) that VERIFY premises against the repo AND attach a concrete
  `proposed_fix` per objection; the AUTHOR (designer role, now true-Fable via the Gate-3
  `claude:claude-fable-5` CLI lane — driver default updated) accepts/rejects each by name.
  Single-author + adversarial-proposers, NOT co-authoring. Preconditions all satisfied (doc
  updated). Two-tier stays: text quorum always, agentic escalation when contested/high-stakes.
- **(c0) [LANDED 2026-07-16 iter 37 → implemented/v0_30_0; PR #401 → `60351087b`, eval PASS 96/100 r1]**
  m-gemini-exec-project-plumbing — `resolveGCPProjectEnv()` (`AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT`,
  coordinator precedence) + `GCPProject`/`GCPLocation` now set on the shared `executor.Task` in
  `cmd/ailang/exec.go:executeCLI`; empty location defers to executor `defaultLocation="global"`, unset
  project keeps the loud error (no silent default). **Live-verified**: env-unset → loud error preserved;
  `AILANG_CLOUD_PROJECT` set → error moved past "GCP project not set" to Vertex `HTTP 400: Resource setup
  has just started` (project REACHES the backend — the "resource setup" state is fleet (c) M0/M4 territory).
  Non-vacuous `t.Setenv` regression test. **Fleet (c)'s M0/M4 gemini reviewer lane is now UNBLOCKED** —
  next fleet step is (c) M0 (live gemini network probe) → M4 (conditional on M0) → M5 (bounded live-fire +
  doc → implemented/).
- **(c1) [LANDED 2026-07-17 iter 39 → implemented/v0_30_0; PR #405 → `ae5f0a00f`, eval PASS 96/100 r1]** m-gemini-evaluator-diff-bridge — Mark's #399
  directive ("default evaluator = gemini if able to git clone the codebase, otherwise sonnet-5")
  forced fleet (c)'s M0 live probe early. **Two findings**: (1) **M0 live gemini probe TIMED OUT**
  (`ailang exec gemini` → `http2: timeout awaiting response headers` on the Vertex
  `interactions` POST — the request reaches the backend but no response returns; same class as the
  iter-37 "Resource setup has just started"). Backend reliability is still unproven — M4/M5 stay
  blocked on it. (2) **The evaluator role needs a diff-bridge, not just the executor** (extends
  fleet (b)'s note): the managed_agents request body carries only `Directive`+`SystemPrompt`
  (`managed_agents.go:164`), so even a READ-ONLY evaluator sees NO local repo — it cannot inspect
  the sprint's uncommitted worktree changes nor re-run tests. To make gemini a real evaluator: ship
  the `git diff` + changed files INTO the directive text (mirror `managed_agents_bridge.go`), accept
  it's reasoning-only (no local test re-runs), AND land backend reliability. **BOTH DONE (iter 39)**:
  backend reliability confirmed (4/4 bounded probes SUCCESS); the diff-bridge capability shipped
  (`internal/eval_harness/gemini_evaluator_bridge.go` — `BuildDiffBundle` untracked-inclusive +
  reasoning-only directive + `GeminiVerdict` + `RunGeminiEvaluator` injectable caller seam +
  caller-enforced `VerificationDegraded`; PASS 96/100 r1). Default evaluator STAYS **sonnet** —
  capability only; a gemini-default flip needs a live diff-bridge fire + the ≥3-datapoint evidence rule.
**[GAP CLOSURE PRIORITY — Mark 2026-07-17: "I want the gaps here worked on as priority"]**
Work these BEFORE returning to the clause queue; one per iteration, cheapest-confirmation-first:
- ~~**(G1) gemini FIRST LIVE ROLE FIRE**~~ **CONFIRMED iter 43** — live `ailang design-quorum` →
  `gemini-3-1-pro` **present, verdict=reject, $0.023** (its first clean live reviewer verdict). The
  evaluator arm (`RunGeminiEvaluator`, PR #405) has no CLI seam yet, so the **quorum-reviewer seat**
  (G1's explicit OR) carried it. Reliability blocker found+fixed same iteration: gemini's THINKING
  tokens overran the `reviewMaxTokens=4096` cap → intermittent silent-truncation N-1 quorum (PR #408
  → `885725f06`: cap→16384, fail-loud on `finish_reason=length`, wired gemini `finishReason`). Log 48.
- ~~**(G2) 3-provider quorum CONFIRMATION round**~~ **CONFIRMED iter 43** — same live quorum:
  `gpt5-6-sol` (OpenAI, restored post-#407) + `gemini-3-1-pro` (Google) BOTH present + claude
  controller = 3 providers, both `reject`. The solo-gemini-veto era is over. Log 48.
- ~~**(G3) DESIGNER ROTATION live test**~~ **CONFIRMED iter 44** — `codex:gpt-5.6-sol` (rotation next
  after `claude:claude-fable-5`) authored the G4 design doc via the cross-provider `workspace-write`
  worktree recipe carrying the design-doc-creator directive (**first codex-designer fire**), then ran a
  competent objection-addressing revision. The rotation MECHANISM works end-to-end (design → quorum-gate →
  revise). Evidence row: `(designer=codex:gpt-5.6-sol, quorum=reject→revise→reject over 2 rounds × 3
  providers)` — the content reject is the quorum enforcing data-before-conclusions (unverified external
  contract), NOT a designer failure. Rotation state advanced to `codex:gpt-5.6-sol`; next new-doc iteration
  returns to `claude:claude-fable-5` (gemini joins after G4). Log 49.
- **(G4) gemini REPO-MOUNT upgrade** — **[✅ FULLY LANDED + LIVE-VERIFIED iter 53 — Mark "vertex git clone
  test granted" (#399 2026-07-18T11:59:47Z). The last INCORPORATED premise (provider `git fetch --depth 1
  <sha>` support) is now VERIFIED-LIVE: `TestLiveCloneOverEgressE2E` pinned a real non-HEAD SHA `80cbd9612…`
  through the production `Executor.Execute` path → fetch-by-SHA → exact-SHA echo → `CLONE_OK` → PASS (113.6s,
  $0.865, 527k in/8.2k out). Doc + sprint-plan MOVED to `implemented/v0_30_0/`. **Fleet role (reported to Mark,
  ratification parked):** gemini/managed_agents is now a proven in-sandbox EVALUATOR/reviewer (clone→`ailang
  check`→verdict; Google provider = valid generator≠judge) — but `CapRemoteSandbox` means it canNOT edit a
  worktree, so "gemini joins the DESIGNER rotation" needs the text-bridge and is NOT auto-wired; recommend
  gemini enter as evaluator. **→ RATIFIED by Mark 2026-07-18 (interactive, after cost review):
  gemini is ADMITTED to the fleet as the ESCALATION-TIER in-sandbox evaluator/reviewer — NOT
  every-iteration (sonnet stays the default evaluator) — with THREE mandatory cost guards:
  (1) ENVIRONMENT REUSE ("environment reuse for sure"): clone once per review target, persist the
  `env_<id>`, reuse across rounds — never re-clone per round; (2) tight directives (targeted
  `ailang check`/grep, no repo wandering); (3) two-tier discipline (text quorum first; in-sandbox
  only when a premise is contested/high-stakes). Cost basis VERIFIED against official docs
  2026-07-18: NO managed-agents premium — standard Gemini token rates only ($1.50/M in, $9/M out
  incl. thought tokens at output rate; our client math reconciles $0.865 = 0.79 in + 0.07 out);
  sandbox compute is FREE during preview. ⚠ WATCH ITEM: at GA, Google adds environment-compute
  charges — re-benchmark the escalation-tier economics when GA pricing lands. Designer rotation
  UNCHANGED (claude⇄codex) pending the text-bridge. Next iteration wires the evaluator seat +
  env-reuse.** Prior: LANDED (code) iter 52; both approved fixes (typed `RequiresEgress`/
  `CapNetworkEgress` gate + `ValidateTaskCapabilities`; bounded-execution) + iter-52 shallow-fetch-by-SHA fix;
  opus executor, sonnet evaluator 91/100. Log 57–58.]** iter-45 refuted the `repository`/`inline`
  mount model (only `gcs`+`skill_registry`; egress OFF by default; egress param "undiscovered"). iter-46
  (Mark #399 → philschmid.de/managed-agents-gh) **found the egress param and superseded the mount model**:
  it is a structured list `environment.network.allowlist:[{domain,transform}]` (not iter-45's scalar
  guesses). Re-probing OUR Vertex endpoint (probes O–R, same ADC harness): `network.allowlist:[{domain:"*"}]`
  is **accepted and provisions an egress-enabled sandbox** (Vertex allows wildcard `*` only today;
  per-domain + header-`transform` = "not supported now"). Probe **R**: an egress-only env (NO data source)
  **cloned the public ailang repo end-to-end** (`git clone` OK, `rev-parse HEAD`=`806b3b4a4`=current dev,
  file listing + `go.mod` returned). **⇒ new dominant option (d) CLONE-OVER-EGRESS:** give the executor an
  egress env + have the agent `git clone` the public repo at a SHA itself, then `ailang check`/review
  in-sandbox — no encoder/GCS/inline/mount. Small; directly delivers #399's "gemini can git clone the
  codebase" for the reviewer role. **Recommendation: (d)** (fallbacks: (a) GCS for *private* code, (b)
  shelve, (c) skill_registry). **DECIDED by Mark 2026-07-27 (attended interactive session, quota-relief
  directive): GREENLIGHT Phase-2 clone-over-egress.** Scope: wire gemini managed-agents into the
  DESIGNER rotation via option (d) (egress env + in-sandbox `git clone` of the public repo at a SHA —
  design docs need only committed HEAD); follow-on under the same greenlight = evaluator-REVIEW lane
  (executor pushes the sprint branch pre-merge → gemini clones the branch, CI stays the test oracle —
  the iter-38 uncommitted-worktree objection does not apply to a pushed branch). Same
  `MISSION_METERED_BUDGET_USD` ceiling; public trace comment on #399. [Historical ask was:
  greenlight the Phase-2 clone-over-egress decomposition, or shelve.] Reproducible probe: `internal/executor/managed_agents/managed_agents_live_test.go`
  (`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`, CI-inert, probes O–R). Doc RESHAPED with full VERIFIED-LIVE
  contract. Log 51. **Note:** the blog is the Gemini *Developer* API surface (`ai.google.dev`, API-key) —
  a different contract from the Vertex executor; our-project Developer-API confirm is parked (the available
  `GOOGLE_API_KEY` is invalid even for generateContent).
- ~~(G5)~~ **REMOVED from the gap path (Mark 2026-07-17): the qwen3-6 lane is a NICE-TO-HAVE,
  not a gap.** See (d) below — sequenced only after the cloud fleet is fully proven (G1–G4
  done), and NOT at gap priority: after G4, the loop returns to the clause queue; (d) is picked
  on normal cheapest-impact ordering.

- **(d) Phase D — motoko + qwen3-6 local-GPU lane** (fleet doc Phase D, ~2–3d) — **NICE-TO-HAVE,
  post-cloud (Mark 2026-07-17)**: the standing role of this lane is the **local assignee for
  slow-but-free task classes** — long-running, low-urgency work with deterministic verification
  (bulk regens, wide test sweeps, corpus churn) where wall-clock doesn't matter and $0/token does.
  It is NOT a peer of the cloud lanes for interactive-cadence roles. HARD constraints unchanged:
  `rig.lock` two-tier discipline (GPU mutex per-step, never iteration-wide), the port-8080 zombie
  hazard (memory: a hung motoko holding 8080 breaks all later runs), and the same evaluator gate
  as cloud work — no quality discount for free tokens.

**[LANDED 2026-07-23 iter-92 — was HARD PIN, Mark 2026-07-23: PORTABILITY M2+M3]** Mark asked "do we have the ability to work on multiple missions yet?" — answer was NO because M2 (skill repo/verify profiles) + M3 (public bootstrap guide) had been skipped for 8 iterations. **Both landed iter-92** (headless-greenlit, no quorum, Mark-ratified split): M2 = `## Repo Profile` block in SKILL.md + mission-doc charter header (two verify profiles `go-compiler`/`ailang-code`; 3 `--repo` args parameterized); M3 = public `docs/docs/guides/mission-bootstrap.md` + `design_docs/mission-charter-TEMPLATE.md`; dry-run isolation acceptance PROVEN (`mission-worldtest.pid` distinct, v1 untouched); evaluator sonnet PASS 83/100. **Ailang World launch is now UNBLOCKED** (its iteration 0 = charter ratification, attended w/ Mark). The triage sweep below is COMPLETE — its NEXT-FIRST is spent:

**[completed 2026-07-23 iters 85–87: FULL BACKLOG RE-TRIAGE]**
("lets do another review of the docs in planned and see which we can put into the cycle.") The
third triage pass, against July reality: fleet live, fmt shipped, v0.30.0 released, arch-boundaries
landed, quorum-at-pick in force. Sweep **ALL planned/ folders** (~114 docs: root 14 · v0_29_0 38 ·
v0_30_0 19 · v0_31_0 3 · v1_0_0 5 · v1_1_0 30 · docparse-billing 5). Rules:
- **Sequencing**: run AFTER the currently-authorized work (raw-handler M1 · reasoning-effort final
  round · fmt polish pair · strict-fallbacks) — those are decided; triage before picking anything
  NEW beyond them. May span 2–3 iterations (folder-group per iteration; oldest first: root +
  v0_29_0, then v0_30/31/v1_0_0, then v1_1_0 + docparse).
- **Per doc**: reality-check the status claim FIRST (the ghost discipline — cheap live probes for
  bug-claims, `git log --grep` for landed-claims; iteration-0/25/48 precedent: statuses LIE), then
  tag exactly one: **[GATING clause-N]** (serves an open bar clause → queue placement) ·
  **[CYCLE]** (non-gating but net-valuable now → normal v0.3x road, loop may pick when gating
  queue is blocked) · **[POST-V1]** · **[GHOST/SUPERSEDED → close with a CI-enforced guard where
  the claim was a bug]** · **[FOLD-INTO <doc>]**.
- **Controller-lane** (read+verify, no generation — iterations 45/48 pattern; no quorum during
  triage, quorum-at-pick fires when a doc is actually PICKED). Deliverable: triage table on the
  bookkeeping issue + charter queue rewrite + archive moves for ghosts/superseded. Docs promoted
  to [CYCLE] get an explicit one-line WHY (what changed since they were shelved).
- **(B) GITHUB-ISSUE TRIAGE (Mark 2026-07-20: "for v1.0.0 we should also triage all the other
  github issues — see if they have design docs, are stale or defunct superseded etc or need a new
  doc")** — the ~19 open non-thread issues (span Dec 2025–Jul 2026: CI flakies #351/#338,
  test-runner litter #328, runtime/test paths #324/#326, effect-row bug #386, the May
  motoko_explore trio #231/#226/#225, CLI asks #223/#157/#155, Z3 #215, docparse #224/#153/#143,
  Sonar #104, stapledons #18, nightly-watch #384). Use the **github-issue-triage skill**. Per
  issue: reality-check at HEAD (cheap live repro where the claim is a bug — issue bodies age like
  doc statuses), then exactly one: **FIXED → close citing the commit** · **STALE/SUPERSEDED →
  close with evidence** · **COVERED-BY an existing/queued doc → link both ways + tag** ·
  **NEEDS-NEW-DOC → note for the designer rotation (doc authored on pick, quorum applies)** ·
  **GENUINE v1-GATING → clause-tag + queue placement** · **POST-V1 → label + say so on the
  issue**. External-author issues ALWAYS get a reply (public repo — same courtesy as #417).
  Runs alongside/after the doc sweep; same 2–3-iteration budget, same evidence discipline.

  **RE-TRIAGE BATCH 1 — root + v0_29_0 (iteration 85, 2026-07-23; 52 docs, controller-lane, 4
  read-only sonnet workers @ $0).** Full evidence table on #422. Outcomes:
  - **LANDED → swept to implemented/** (7): `m-eval-elo-priority-rotation` (`b3de7e70f` #423 →
    v0_30_0) · `m-eval-local-cloud-unify` (`c533bb51c` → v0_30_0) · `m-eval-regression-detector-
    contract` (all 3 clauses shipped `9a1c43f34` → v0_29_0) · sprint-plan stubs whose design docs
    already landed: `m-eval-bounded-pipeline-sprint-plan` (`d41e43894` → v0_29_0) ·
    `m-file-handling-improvements-sprint-plan` (`8697c9d01` → v0_29_0) · `m-pattern-and-invocation-
    repair-sprint-plan` (v0_22_0 designs → v0_22_0) · `m-ailang-fmt.md` planned STUB deleted
    (superseded by `implemented/v0_30_0/m-ailang-fmt.md` + phases).
  - **GHOST/SUPERSEDED → archived** (`design_docs/archive/2026-07/`, 4): `M-TOOLING-DETERMINISTIC`
    (Mark self-closed `3df673994`) · `m-motoko-editdecl-astedit` (A/B NEUTRAL `978bd371a` — not a
    pass-rate lever) · `motoko-agent-v0.15.0-migration` + `motoko-integration-sequence` (both track
    the arniwesth fork migration; rig moved to internal `mk-ast` → headline outcome unreachable).
    None were bug-claims → archive-with-stamp, no CI guard needed.
  - **GATING-candidate → queued (clause 3/4; VERIFY-FIRST at pick — survey-sourced, inherit ghost
    debt)** (5): **[GATING clause-3, PARKED iter-93 needs-human-review]** `m-pure-prng` (pure cap-free
    `std/prng`; removes the widen-`--caps`-for-reproducible-randomness footgun; small, stdlib-only —
    **CORE proven bit-exact SplitMix64 at HEAD**, revised+re-quorumed; sole block = `split` scope,
    Path X defer / Path Y 2-word `Gen` — Mark decides, then unpark) · **[LANDED iter-98 → implemented/v0_29_0]**
    `m-budget-scoping-bug` (effect `@limit`/`@min` per-function vs cumulative-across-chain — bug-claim;
    live-repro REAL, quorum-ratified hierarchical per-invocation budget frames. Mark 2026-07-24
    "apply and route" ratified the QUORUM narrow-refinement carve-out's FIRST USE; controller applied
    both reviewer-verbatim fixes; planner opus → executor opus (worktree; `internal/effects/budget_frame.go`,
    defer-guarded frame stack, bubbling charge, +2 latent bugs fixed) → evaluator sonnet PASS 87/100 r1;
    PR #474 squash `f1bf7b77c`, required checks green. Follow-up D1 [non-blocking]: budget error omits
    the frame's function-name — see the doc's Future-work). **[GATING clause-4]** `m-agent-step-cancellation` (`std/ai.step` graceful
    abort/SIGINT — no impl) · `m-serve-api-live-tool-registry` (MCP live re-registration of new
    `.ail` tool files mid-session — watcher exists, ~50 LOC missing) · `m-contracts-as-code-vertical`
    (deontic engine landed as `sunholo/deontic` 0.1.0; the four-moat orchestration docs flagship +
    `examples/contracts/` remain — clause-4 showcase).
  - **CYCLE** (net-valuable now, non-gating; picked when the gating queue is blocked): `m-eval-data-
    hosting-decouple` (W6 prod-promote) · `m-eval-os-version-trend-redesign` · `m-eval-reasoning-
    model-fairness` (D4 re-run) · `m-eval-validity-discipline` (W4 remainder) · `m-motoko-compaction-
    quality` · `m-motoko-self-improvement-loop` · `m-ui-vite8-migration` · `m-ailang-semantic-context`
    (R2/R3/R6) · `m-concurrency-leverage` · `m-coordinator-inbox-wildcards` · `m-dashboard-
    simplification` · `m-dx27-docs-search-github-fallback` · `m-eval-slim-prompt-self-discovery` ·
    `m-eval-stream-health-retry` · `m-motoko-ext-per-task`(+sprint-plan) · `m-ollama-v1-streaming-
    idle-timeout` · `m-stdlib-html-streaming` · `m-verify-stdlib-stale-path` (1h dead-gate fix).
  - **POST-V1**: `m-ailang-native-harness` · `m-dynamic-data-runtime-plane` · `m-anthropic-sandbox` ·
    `m-apple-container-local-eval-sandbox` · `m-cascade-observability` · `m-coord-thinking-levels` ·
    `m-eval-openrouter-baseline-rotation` · `m-eval-results-folder-structure` · `m-fable-strategy-
    review` (strategy index, kept as reference).
  - **UNSURE → left in planned, need a cheap follow-up probe before tag**: `m-bytecode-vm-parity-bugs`
    (run parity verify) · `m-contracts-as-code-sprint-plan` (docs-flagship portion open) · `m-dx-agent-
    eval-gaps` (gaps 2–4 status) · `m-eval-rig-reliability`(+`m-rig-reliability-sprint-plan`) (2 P1
    open: docx recording, A/B contamination) · `m-pkg-feedback-loop`(+sprint-plan) (M3/M4 status).
  - `20251013_auto_caps` intentionally kept in planned (M1 `e542065c0` landed; 3 follow-ups open).
  - **Next batch (iteration 86): v0_30_0 + v0_31_0 + v1_0_0**, then v1_1_0 + docparse.

  **RE-TRIAGE BATCH 2 — v0_30_0 + v0_31_0 + v1_0_0 (iteration 86, 2026-07-23; 30 docs,
  controller-lane, 4 read-only sonnet workers @ $0).** Full evidence table on #422. Outcomes:
  - **LANDED → swept to implemented/v0_30_0/** (11): `m-mission-agentic-provider-routing`(+sprint-
    plan) (M1a `8ee07ef23` · M1b `956fda55c` · M2 `8d12e8e9c`; M3 = parked-with-protocol, the
    documented outcome) · `m-mission-fleet-ab-sprint-plan` (A `3bee6b6df` + B PR #383 `1186a48e6`;
    parent `m-mission-adaptive-multiprovider-routing` STAYS planned — C/D/E opt-in open) ·
    `m-mission-quorum-agentic-verify` (M1-M3 PR #400 `0e83a1b12`; header was stale-PARKED) ·
    `m-ailang-fmt-phase2-sprint-plan` (`3815ba617` PR #414) · `m-fmt-properties-printer-roundtrip-
    sprint-plan` (`942931816` PR #424) · `m-smt-callee-sort-gate`(+sprint-plan) (`94e2a5d27` +
    `efd251f16`) · `m-std-yaml`(+sprint-plan) (`62d681a8e`). Plus `m-ailang-fmt-inline-interior-
    sprint-plan` planned-copy DELETED (canonical already in implemented/ from PR #434 `3c1cec57d`).
  - **GHOST/SUPERSEDED**: none this batch — every doc traced to real commits or a live queue item.
  - **GATING-candidate → queued (VERIFY-FIRST at pick — bug-claims must live-repro before routing)**
    (6): **[EVIDENCE-PARKED iter 91 2026-07-23]** `m-parser-block-let-separator` (bug REAL at HEAD
    but evidence gate MEASURED NEGLIGIBLE — 0 decisive occurrences in 27,359 eval files; every
    attributable case is cascade-noise in already-broken output → stays parked, do NOT route a core
    parser change; re-open only on a decisive rotation case) · `m-diag-
    primitive-field-suggestions` (primitive-field "no methods" hint — severed Part C of the landed
    footguns-to-diagnostics doc; P3/extension-lane, frozen-core + ADT-name premise still to resolve).
    ~~**[GATING clause-4]** `m-check-strict-fallbacks`~~ **[LANDED iter 101 → implemented/v1_0_0;
    `STRICT_FALLBACK_001` Core-level post-name-resolution pass, GlobalRef-keyed known-empty-builder
    registry + ANF Let-env resolver, dual channel (dev WARNING / `check --package` HARD ERROR),
    `@allow_empty_ok` opt-out; evaluator sonnet PASS 88/100 r1; PR #479 `1978ab44b`. D1 (match-arm
    scoping for bare `Ok("")`) → Future Work]** · `m-parmap-effectful` (in-AILANG fan-out for the
    orchestration flagship; M0 `EffContext.Clone()` fork-safety `22e4c11b7` is a HARD prerequisite —
    shallow copy panics under concurrency) · `m-effect-replay-contracts` (effect sprint 2/4) ·
    `m-effect-clock-net-fs-modes` (effect sprint 3/4). `m-effect-scope-params` (sprint 4/4) is a
    release-gate RE-SCORE candidate (Mark may push to v1.1); `m-effect-refinement` is the decomposed
    UMBRELLA — stays planned tracking 3 open children (sprints 2/3/4), sweeps only when all ship.
  - **CYCLE** (net-valuable now, non-gating): `m-mission-adaptive-multiprovider-routing` (phases
    C/D/E opt-in loop infra) · `m-mission-portability` (✅ COMPLETE iter-92 — M1 `825e77c64`, M2+M3
    landed; Ailang World UNBLOCKED) · `m-mission-cost-chains` (clause-5 cost-per-verified-success substrate) · `m-ai-
    structured-step`(+sprint-plan) (composable structured output → vision+JSON grading) · `m-
    comments-for-ai-authors` (M1 = $0 prompt-manager lane, Mark-ratified) · `m-eval-kimi-k3-agentic`
    (standard done; agentic entries gated on `m-eval-reasoning-model-fairness` P1) · `m-managed-
    agents-model-eval` (Gemini Developer API pivot design-frozen; blocked only on an AI Studio key)
    · `m-mem-budget-runtime` (P1 host-safety `MEM001` runtime cap — motivated by the 2026-07-20
    kernel panic; design complete, no impl).
  - **POST-V1**: `m-arch-boundaries-eval-exclusion-tighten` (evidence-gated, trigger unmet — no
    second dashboard `internal/eval` import at HEAD).
  - **PARKED (leave as-is)**: `m-decision-entropy-monitor` (needs-human-review since iter 84,
    quorum-blocked ×2). ⚠ INTEGRITY NOTE: the MAIN checkout has UNCOMMITTED in-progress edits to
    this doc (V11–V13 producer-side evidence rows answering the quorum objections) + 3 unpushed
    local commits (`ff089b7eb`/`5753897e1`/`faeb16d13`, `m-managed-agents-model-eval` doc) — local
    `dev` has DIVERGED from origin/dev. NOT touched this iteration (Critical Principle 0); flagged
    for human sync.
  - **Next batch (iteration 87): v1_1_0 (30) + docparse-billing (5)** — closes the full sweep. **DONE below.**

  **RE-TRIAGE BATCH 3 — v1_1_0 + docparse-billing (iteration 87, 2026-07-23; 35 docs,
  controller-lane, 4 read-only sonnet workers @ $0).** Full evidence table on #422. **This CLOSES
  the full planned/ sweep** (batches 1+2+3 = root + v0_29_0 + v0_30_0 + v0_31_0 + v1_0_0 + v1_1_0 +
  docparse-billing). Outcomes:
  - **LANDED → swept to implemented/** (4 docs / 2 features): `m-type-v2-migration`(+sprint-plan) →
    **v0_10_0** (`b29c391ff` delete legacy TFunc + `a314e7fca` open-effect-row fix; verified `TFunc`
    fully gone from `internal/types/` — comment refs only) · `m-executor-variants`(+sprint-plan) →
    **v0_15_0** (`721550fdb` M1 wiring + `c07bf73c1` codex/opencode images + `af36a00a1` gemini/eval
    images; `ExecutorVariant` live in `internal/coordinator/`; the FULL design promoted OVER the
    rough-draft stub that already lived at v0_15_0 and cross-linked back to planned).
  - **GHOST/SUPERSEDED**: none this batch — every doc traced to real commits or clearly-open work.
  - **GATING-candidate → queued (VERIFY-FIRST at pick — survey/bug-claims must live-repro before
    routing)** (11):
    - **[GATING clause-3]** accessibility / syntax-ergonomics (5): `m-error-propagation` (`?`
      operator — **LIVE-CONFIRMED bug at HEAD**: `PAR_NO_PREFIX_PARSE: unexpected token: ?`;
      LIMITATIONS lists "not yet implemented"; pure desugar ~2d — cheapest high-confidence pick) ·
      `m-pipe-operator` (`|>` — no `PIPE_GREATER` token / `PipeApp` node; clean design, zero deps,
      ~6–8h) · `m-dx-package-dogfooding` [**LANDED 2026-07-23 iter 90** — jint `ceecdd0f1` (PR #467);
      Issues 1/2 prior `7d1e4b82a`/`99f76ec7a`; doc → `implemented/v1_1_0/`] · `m-call-sugar-optional`
      (`f()` optional call sugar — still parse-errors at HEAD; ~1–2d) · `m-forall-properties-direct-
      core-eval` (`properties [forall(...)]` → "empty program" via the broken source-synthesis path
      `internal/testing/runner.go`; sibling `ensures`/`requires` fixed `3ebf60b1b`; doc self-labels
      low-priority "zero users"; ~3–4h).
    - **[GATING clause-4]** agent-orchestration surface (6): `m-effect-handlers` (Koka-style algebraic
      effect handlers — enables deterministic AI-mock handlers from `.ail`; Phase 1 ~38h, cross-cuts
      parser/types/elaborate/eval; LARGE, high-value) · `m-ai-effect-modes-followups` (replay/byok/
      reroute runtime — TODO at `internal/ai/routing.go:157`; items 1+2+4 close the A1/A2 replay-
      determinism story ~3–4d; VERIFY-FIRST item-1 vs already-shipped M-AI-TOOL-LOOP) · `m-agent-
      safe-runner` (safe runner — **M1 policy spike LANDED `4effc002d`** [`internal/policy/`]; M2–M5
      unstarted ~5–7d; one transitive-import-closure design-freeze item) · `m-agent-loop-architecture`
      (runTools hook-extension ADTs — design decision A/B/C unresolved, needs arni input; ~5d;
      VERIFY-FIRST the design freeze first) · `m-process-modes` (`Process[mode=mocked]` runtime replay
      — parser `[mode=...]` landed but the runtime is absent; ~36h; blocked on M-EFFECT-REFINEMENT
      full landing) · `m-agent-orchestration` (`std/agent` effect — LARGE ~2–3wk, the big open
      orchestration surface; DECOMPOSE before executing).
  - **CYCLE** (net-valuable loop/rig/DX infra, non-gating) (9): `m-oracle-adequacy` (eval evidence-
    bundles) · `m-trace-feedback` (`ailang trace diagnose`) · `m-entropy-budgets` (design-doc
    completeness infra) · `m-d4-design-doc-driven-development` (`ailang verify --spec` compliance;
    budget substrate partial) · `m-pkg-inflight` (cloud/Firestore package events) · `m-zero-language-
    learnings` (Phase-1 `check --json` landed; Ph1.5–3 rig/DX meta) · `m-eval-finetuning-data-pipeline`
    (rig fine-tuning loop) · `m-eval-trust-signals` (eval credibility / HumanEval port) · `dx-
    improvements-from-billing-packages` (partial-landed external DX friction log — FIXED items code-
    verified, open child items live in their own `m-dx-package-check`/`-test` docs).
  - **POST-V1** (real but out of the v1 LANGUAGE scope) (11): `m-csp-session-types` (6–8wk session
    types) · `m-eu-compliance-effects` (author-downgraded, domain lib) · `m-game-engine-effects` (v1.1
    domain lib) · `m-perf4-bytecode-interpreter` (perf stretch, doc says v2.0) · `m-quasi-typed-
    quasiquotes` (parse-only lexer/AST, no runtime; 3–4wk) · `m-reflect-structural-reflection` (class/
    instance parser TODO, 2wk) · `global-collaboration-hub` (cloud infra, Mark-downgraded non-gating)
    · the **docparse-billing/ cluster** = `m-billing-docparse-billing-agent-payment` +
    `responsibility-docparse` + `responsibility-multivac` + `responsibility-packages` (external
    DocParse-billing SaaS spanning `docparse`/`ailang-multivac`/`ailang-packages` repos — NOT AILANG
    language work).
  - **Tally**: 4 LANDED / 0 GHOST / 11 GATING-candidate / 9 CYCLE / 11 POST-V1 = 35. Zero GATING-
    candidate touches the frozen core beyond parser/desugar surface; the clause-4 orchestration items
    are the heavy ones (effect-handlers, std/agent, safe-runner) and need decomposition before route.
  - **Sweep-complete → NEXT for iteration 88**: the GATING backlog is now fully surfaced across all
    three batches. Cheapest-first pickable order (once live-verified at pick): ~~`m-error-propagation`
    (clause-3, live bug, ~2d)~~ **[PARKED needs-human-review iter 88 — `?`-op REAL at HEAD but the
    design quorum-blocked ×2 (Rev-0 fatal local-desugar flaw fixed → Rev-1 re-quorum surfaced deeper
    open questions: compiler↔stdlib lang-items linkage for user-space `Ok`/`Err` ConstructorIDs +
    unverified `core.go:309`/`normalizeToAtomic` lowering premises). Rev-1 design + open questions
    preserved in `planned/v1_1_0/m-error-propagation.md`; unpark needs an arni/human architecture
    decision]** → ~~`m-pipe-operator` (clause-3, ~6–8h)~~ **[PARKED needs-human-review iter 89 — `|>`
    REAL at HEAD (`42 |> show` → `PAR016`) but Rev-1 re-quorum came back 1 pass (gemini) / 1 reject
    (gpt5-6-sol). Rev-1 fixed both Rev-0 objections (LHS-first `Let` desugar for eval-order; 14-row
    Conflict Surface + frozen-core/AILANG-fix-lane justification). The sole remaining objection
    (non-callable/arity>1 RHS "may fail only at runtime") was **controller-VERIFIED REFUTED** — both
    `x |> 42` and arity>1 RHS are caught at type-CHECK time (`No instance for Num[int -> int]`;
    `TC_ARITY_001`). Parked per the one-re-quorum gate, NOT force-passed. **Unpark is LOW-RISK: route
    straight to sprint-planner, no design change needed.** Rev-1 + refutation in
    `planned/v1_1_0/m-pipe-operator.md`; PR #466]** → **iter 90 LANDED `m-dx-package-dogfooding`
    jint micro (PR #467 `ceecdd0f1`; doc closed → implemented/v1_1_0). iter 91 EVIDENCE-PARKED
    `m-parser-block-let-separator` (measured negligible, 0/27,359 decisive). iter 92 LANDED
    `m-mission-portability` M2+M3 (loop now portable). iter 93 `m-pure-prng`: quorum caught real
    soundness defects → 1 designer-revision fixed the CORE (proven bit-exact SplitMix64) → re-quorum
    still blocks on `split` → PARKED needs-human-review (Path X defer / Path Y 2-word `Gen`). NEXT for
    iter 94:** `m-budget-scoping-bug` (GATING clause-3 bug-claim, live-repro first) OR unpark
    `m-pure-prng` on Mark's Path pick. clause-4 effect chain (replay→clock/net/fs→scope)
    stays sequential; the big orchestration items (`std/agent`, effect-handlers) need decomposition
    iterations first.

**[NEXT]** clause-3 accessibility cluster (the bulk of v1.0). Loop ordering within a group:
P0/unblockers first, then cheapest impact-per-day. The DOC-READY/small diagnostics AND the
VERIFY-then-route backlog are now EXHAUSTED (module-less/xcheck/json-bool/split-arg landed iters
14–17; both VERIFY-then-route items closed as ghosts iter 18). **Iteration 25 (2026-07-13)
Gate-2 reality check found the strategy review's R4 rows were never individually re-verified:
R4a `m-dx-match-hof` and R4b `m-poly-arith-lambda` are BOTH GHOSTS** (R4a: original failure used
the retired `match … with` syntax, brace-form works in every probed position, design doc was
already archived Not-Applicable 2026-05-09; R4b: fixed v0.7.0, verified incl. one let-bound
lambda at BOTH int and float) — guard examples `match_hof_lambda.ail` + `poly_arith_lambda.ail`,
PR #379 → `ea8116f83`, dev CI green observed. Same iteration EXECUTED
**m-lambda-open-record-pattern** (REAL at HEAD — doc existed at planned/v0_29_0, so NOT NEW-DOC;
see queue item 16). The parser/type footgun row is now FULLY BURNED DOWN (m-xmod-alias-poly
landed iter 26). **Iteration 27 (2026-07-14) opened the Prelude/discovery group:
m-prelude-option-result LANDED (PASS 98/100 round 1, mission high; PR #382 → `d26215341`) +
m-prompt-option-none-idiom closed SUPERSEDED by it. **Iteration 29 (2026-07-14) EXECUTED
m-dx-examples-coverage → LANDED** (doc re-scoped through the FIRST LIVE 5-round quorum, PR #392
→ `3d451947c`, eval round-2 PASS after a one-line Windows fix `881711325`; 5 red examples
quarantined under #386, verify-examples now a REAL CI gate, docs --examples un-inert —
doc → implemented/v0_30_0). **Iteration 30 (2026-07-14) EXECUTED the last clause-3 starter
m-dx-ai-discovery → LANDED** (RESUMED from a died-mid-execution prior run [transient Anthropic
rc=1 at 16:05, pre-dating the 17:16 driver-retry fix]; doc re-scoped + quorumed by that run at
`39d671a52`, executor completed M1/M3/M4/M5, PR #393 → squash `c07c36b25`, eval round-1 PASS
93/100 + hardening `ea6069815` [arrays→array alias] + Windows guard fix `0ad27444c`. Interleave:
dev went RED mid-iteration from sibling M-STD-YAML/M-SMT merges — fixed forward direct-to-dev
`9a314772d` [yaml builtin golden + Z3-gating verify e2e] + `4caddfd23` [>800-line split of
verify.go/codegen.go]; `ailang docs --all-functions [filter]`, unknown-module did-you-mean +
module list, `ailang docs prelude`, V16 effect-row fix — doc → implemented/v0_30_0).**
**Iteration 22 (2026-07-13) front-ran R4a with a regression-derived NEW-DOC pick** (nightly
`higher_order_functions` triage → real decl-class resolver gap #366); **iteration 23 EXECUTED it
→ LANDED** (PR #368 → `fd38ec14e`, eval PASS 98/100 round 1 — queue item 15). Full inner-loop
sprints, NOT bookkeeping.
*(m-match-xcheck-error-quality LANDED iter 15; m-dx-json-bool-coercion in-repo half LANDED iter 16
[`std/json.asBoolLoose`; Phase-1 firestore fix PARKED out-of-repo]; m-dx-split-argument-warning LANDED
iter 17; m-dx-record-cons-pattern + m-dx-tapp-trecord-unification GHOSTS/verified-closed iter 18;
m-arity-style-diagnostic (R4c) LANDED iter 21 [TC_ARITY_001, PR #363 → `5b54509d1`] —
all → implemented/v0_30_0.)*

*(SCOPE EXPANDED 2026-07-12, Mark — full-v1.0 triage of the 69 non-gating docs. The clause-3
accessibility cluster, BOTH DX tooling investments, and the FULL clause-4 orchestration surface
are all IN. v1.0 = the complete "verified AI-orchestration language, accessible to mid-tier
models" — ~33 open items, ~40–55 sprint-days. Rig/cloud/motoko/post-v1 infra stays OUT. Full
triage evidence = log entry 10.)*

### Clause 3 — fleet-tier accessibility (the footgun burn-down; the thesis's core deficit)
- **Parser/type footgun fixes** (NEW-DOC, Conflict Surface mandatory): ~~m-module-let-func-resolution~~
  **[LANDED iter 23 → implemented/v0_30_0; unified SCC over lets+funcs, wrapInLets deleted, module
  letrec via core.LetRec, MOD007 dup-name, truthful hint; PR #368 → `fd38ec14e`, eval PASS 98/100
  round 1]** ·
  ~~m-dx-match-hof (R4a)~~ **[GHOST iter 25 — retired `match … with` syntax was the real culprit,
  brace-form match works in every probed position (block-body/direct/mid-block/nested-HOF/curried
  foldl); `\x ->` wrong-arrow already has a teaching diagnostic; guard
  `examples/match_hof_lambda.ail`, PR #379 → `ea8116f83`]** ·
  ~~m-poly-arith-lambda (R4b)~~ **[GHOST iter 25 — fixed v0.7.0 (m-poly-arithmetic-fix); verified
  incl. let-bound lambda at BOTH int and float; guard `examples/poly_arith_lambda.ail`, PR #379 →
  `ea8116f83`]** · ~~m-arity-style-diagnostic (R4c, 1–2d)~~ **[LANDED iter 21 →
  implemented/v0_30_0; `TC_ARITY_001` coded/directional/style-aware arity diagnostic at
  `unification_types.go`, 5 golden/regression tests, eval PASS 97/100 round 1, PR #363 →
  `5b54509d1`]** · ~~m-lambda-open-record-pattern (1d)~~ **[LANDED iter 25 → implemented/v0_30_0;
  `{name, ...}` in lambda params now infers OPEN `{name: τ | r}`; PRIMARY root cause was
  `unifyRecord`'s pre-row field-count rejection (deeper than the doc's hypotheses) + Rest erased
  at AST→Core; closed-pattern strictness preserved + arm-order-independence hardened + cacheKey
  v2; eval PASS 92/100 round 1, PR #380 → `47576e25d`, dev CI green per-workflow observed]** ·
  ~~m-xmod-alias-poly (1–2d, VERIFY-FIRST)~~ **[LANDED 2026-07-14, iter 26 →
  implemented/v0_30_0; VERIFY-FIRST probe confirmed REAL at HEAD (NOT a ghost — but the NEW-DOC
  tag was wrong, full doc existed at planned/v0_29_0); parameterized aliases now instantiate
  (`Box[int]` → `{items: [int]}`, single- + cross-module) via `expandAlias` `*TApp` branch keyed
  strictly on alias-env membership (ADTs stay nominal, proven); `TC_ALIAS_ARITY_001`; cacheKey
  v3; eval PASS 93/100 round 1 (first zero-correction pass); PR #381 → `fd1b11a47`, dev CI green
  per-workflow observed]** · **m-parser-block-let-separator** (PARKED, evidence-gated, split out
  of m-dx-expected-fail-fixes iter 40 → planned/v0_30_0): a simple-RHS `let x = e` tolerates
  eliding the statement separator before a trailing expr, but a block-RHS `let x = match{...}`
  does not — a minor parser ASI inconsistency. NOT auto-fixed (default-bias-not-core); route only
  with a measured eval failure-rate + Conflict Surface.
- **VERIFY-then-route** (ran the doc repro FIRST — both were ghosts): ~~m-dx-record-cons-pattern~~
  **[LANDED/GHOST iter 18 → implemented/v0_30_0; `{…} :: rest` type-checks; guard
  `TestListConsPatternWithRecord` + `examples/record_cons_pattern.ail`, PR #358 → `adde9e9d0`]** ·
  ~~m-dx-tapp-trecord-unification~~ **[LANDED/GHOST iter 18 → implemented/v0_30_0; `[[TableCell]]`
  extraction type-checks; guard `examples/record_list_extraction.ail`, PR #358 → `adde9e9d0`]**
- **Diagnostics** (DOC-READY / small): ~~m-module-less-run-fail-loud (MOD014)~~ **[LANDED iter 14 →
  implemented/v0_30_0]** · ~~m-match-xcheck-error-quality~~ **[LANDED iter 15 → implemented/v0_30_0]** ·
  ~~m-dx-json-bool-coercion~~ **[in-repo half LANDED iter 16 → implemented/v0_30_0 (`std/json.asBoolLoose`);
  Phase-1 firestore-package fix PARKED out-of-repo in `ailang-packages`]** ·
  ~~m-dx-split-argument-warning (1d)~~ **[LANDED iter 17 → implemented/v0_30_0; compile-time
  non-blocking reversed-`split` warning, extensible `swapTraps` table, PR #356 → `8339b6421`]**
- **Prelude / discovery**: ~~m-prelude-option-result (Some/None no-import, 1.5d)~~ **[LANDED
  2026-07-14, iter 27 → implemented/v0_30_0; Gate-2 probe confirmed REAL at HEAD (`undefined
  variable: Some`/`Err` without import); planner CORRECTED the doc's mechanism (the proposed
  `InjectPreludeValues` never existed — real fix = implicit lowest-precedence std/option +
  std/result imports at ONE loader call-site consumed by both compile and runtime, entry-modules
  only); explicit imports + local types shadow cleanly, library modules unchanged, no
  cacheKeyVersion bump (verified); 15 new tests, 0 deletions; eval PASS 98/100 round 1 (mission
  high; 20 adversarial probes incl. entry-only through real multi-module runs + PR-#381 alias-env
  non-interaction); PR #382 → `d26215341`, dev CI green per-workflow observed]** ·
  ~~m-dx-ai-discovery (2d)~~ **[LANDED iter 30 → implemented/v0_30_0; re-scoped (one-shot
  discovery: docs --all-functions, unknown-module recovery, docs prelude, V16 fix); PR #393 →
  `c07c36b25`, eval PASS 93/100 round 1]** · ~~m-dx-examples-coverage (1d)~~ **[LANDED iter 29 →
  implemented/v0_30_0; first live 5-round quorum subject; PR #392 → `3d451947c`; 5 red examples
  quarantined under #386; verify-examples now a real gate + validate_manifest --ci wired;
  docs --examples un-inert via manifest `modules` field]** ·
  ~~20251013_auto_caps (infer caps, 2d)~~ **[M1 LANDED iter 32 (kept in planned/v0_29_0 — 1 of 4
  phases): `ailang run --caps auto` infers the entrypoint's effect row + grants exactly those
  (planner refuted the doc's ~200-LOC new-package mechanism → 74-line reuse of the existing
  `iface`/`TFunc2`/`EffectRow` path); FIRST cross-provider codex live-fire (executor = OpenAI
  gpt-5.6-sol, evaluator = Sonnet PASS 98/100 r1), PR #397 → `e542065c0`, all required checks green
  observed. Deferred: `--auto-caps` flag, `AILANG_AUTO_CAPS` env, always-on preflight+exit-2,
  bench-harness integ, cap manifest]** · ~~m-dx-expected-fail-fixes (1–2d)~~ **[GHOST-CLOSED
  2026-07-17 iter 40 → implemented/v0_30_0; Gate-2 live-repro CONFIRMED largely-ghost — 0 of 4
  "bugs" needed a language fix. Bug4 effect_budgets: `@limit` enforcement WORKS at runtime
  ("budget exhausted: semantic limit=3"); the doc's repro put `--caps` AFTER the filename where
  it's ignored (flag must precede the file). Bugs1/2 (arrow-lambda, multi-`requires`) + the 2
  match_foreign files: good teaching diagnostics / intended type-rejections, not bugs. Bug3
  serve_api_webhook: non-canonical example (omitted `;`/`in` after a block-RHS `let`, deprecated
  string `++`). CLOSED with regression guards: the 3 parser-bug examples fixed to canonical syntax
  + promoted to `examples/runnable/` (now CI-gated), effect_budgets README corrected, manifest
  de-drifted (2 mispathed contracts entries repaired). Executor Opus / evaluator Sonnet PASS
  92/100 r1 (generator≠judge). Split-out: the block-RHS-`let` separator ASI inconsistency →
  new backlog `m-parser-block-let-separator` (evidence-gated, default-bias-not-core). PR #406]**
- **Prompt teaching** (batchable, ~0.5d each): ~~m-prompt-option-none-idiom~~ **[SUPERSEDED
  2026-07-14 by m-prelude-option-result's structural fix (its own doc named this band-aid as
  superseded-on-ship); prompt v0.16.2 already teaches the prelude availability; doc → archive/
  with library-module caveat noted]** · ~~m-prompt-single-file-module · m-prompt-split-list-operations ·
  m-prompt-log-file-analyzer-string-ops~~ **[CONSOLIDATED iter 47 into `m-prompt-footguns-to-diagnostics`,
  RATIFIED by Mark 2026-07-18, LANDED iter 54 (2026-07-18) → implemented/v0_30_0.** Part A (PRIMARY):
  wired dormant `MOD002` + new `PAR_MODULE_PLACEMENT` at `parseTopLevelDecl` (mirrors `reportMisplacedImport`)
  + gemini's error-recovery state-isolation fix (two late modules → `PAR_MODULE_PLACEMENT`×1 + `MOD002`×1
  genuine-dup, never a FALSE MOD002) — the ~10% multi-module footgun's opaque `PAR_NO_PREFIX_PARSE` cascade
  is now a coded teaching diagnostic. Part B: split-list-operations GHOST-closed with CI-gated
  `examples/runnable/split_map_join.ail`. Part C (primitive-field, ~2%) SEVERED → extension backlog stub
  `m-diag-primitive-field-suggestions.md`. Planner opus / executor opus / evaluator **sonnet** (generator≠judge)
  **PASS 91/100 round 1**; 3 superseded prompt docs → archive/. No re-quorum (Parts A+B unanimous both rounds).
  Log entry 59.]**
- **DX tooling** (Mark: both in → resolved 2026-07-18: M-TOOLING-DETERMINISTIC **CLOSED-SUPERSEDED**
  by Mark; fmt is the DX item): **m-ailang-fmt [LANDED 2026-07-19 (iter 56)]** — `ailang fmt [--write]
  [--check]` canonical formatter shipped, doc → [implemented/v0_30_0/m-ailang-fmt.md](implemented/v0_30_0/m-ailang-fmt.md).
  New `internal/format` package (exhaustive precedence-aware AST→source printer, no `String()` fallback),
  `cmd/ailang` fmt subcommand (stdout/`--write`/`--check`, atomic same-dir-temp+`os.Rename`, exit 0/1/2),
  opt-in lossless lexer comment scan; `internal/ast/print.go` untouched; newline-per-statement braced
  canonical form, Phase-1 fail-CLOSED on comments (exit 2, byte-identical). Author codex-rotation doc
  (quorum-complete, no re-quorum per Mark ratify). Planner opus / executor opus / evaluator **sonnet**
  (generator≠judge) **PASS 87/100 round 1**. **Controller independent verification caught + fixed a real
  defect the corpus test missed** — the explicitly-pure empty effect row `! {}` was dropped (round-trip
  failed on the doc's own V2–V5 idiom; `ast.FormatEffects` collapses nil vs non-nil-empty; no comment-free
  example uses `! {}`) → `formatEffectRow` helper at all 3 sites + regression fixtures (`0b983a8f8`); 2
  evaluator lint nits cleaned (`305a37dd6`). `metered=$0.00`. Log entry 61. ·
  **m-ailang-fmt-phase2 [LANDED 2026-07-19 (iter 63) — PR #414 squash `3815ba617`; evaluator (sonnet) PASS 78/100 r1; doc → `implemented/v0_30_0/m-ailang-fmt-phase2.md`]**
  — Executor (opus, isolated worktree) shipped M0–M3 (6 commits `83f7ebf23`→`b29e871c4` + lint fix `fe236572c`).
  **Corpus gate V22**: 386 parse-valid → 327 formatted, 0 comment-loss, 0 Phase-2 round-trip regressions, interpolation-refusal 0/386, idempotence 299/299.
  **Calibrated fail-closed boundaries** (both never-lossy, in Future Work): (1) 15.28% (59/386) inline-interior refusal
  (`let … in` chains the parser collapses → no stable idempotent boundary → exit 2, byte-identical) → **follow-up sub-sprint queued below**;
  (2) 28 pre-existing Phase-1 `properties[...]` printer round-trip bugs surfaced (verified not caused; fail comment-free r/t on dev too) → **separate item queued below**.
  Controller fixed 3 sprint-introduced lint issues + moved the doc. Was: DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; UNPARKED by Mark option (b) `c624b456d` (iter 62); planned iter 62.
  — Phase-2 lossless comment preservation, the UNBLOCK for fmt on the 94.7% commented corpus. Doc authored iter-59,
  Rev-3 iter-60 (`design_docs/planned/v0_30_0/m-ailang-fmt-phase2.md`, `d1ed2fe57`); token-anchored envelope
  (AST spans proven unusable at design time). Rev-3 FIXED the 2 R2 defects (V21: **386/393 parse-valid** via
  `ailang check`; hard-left-wall widening clause), but the re-quorum surfaced **2 NEW architecture-level objections**:
  (a) gpt5-6-sol → attacher-totality inventory unproven (no code-audit of all printer child-list boundaries —
  params/type-args/ctor-args/record-fields/annotations); (b) gemini → interpolation clamping structurally fatal
  (collapses inner-AST boundaries; would silently delete comments in `${…}`). **→ RESOLVED by Mark 2026-07-19
  (interactive, option (b) + recommendations): UNPARKED, [NEXT] route to sprint-planner, do NOT re-quorum.**
  (1) M0 of the sprint = the PRINTER CHILD-LIST CODE AUDIT — proven inventory folded into the design before
  attachment code; (2) interpolation = FAIL-CLOSED CARVE-OUT (preflight refuses files with comments inside
  `${…}` holes — silent deletion structurally impossible; full interpolation-aware attachment deferred,
  evidence-gated on measured refusal rate, expected ≈0). Doc Status stamped. ·
  **m-ailang-fmt-adoption [LANDED 2026-07-20 (iter 65) — PR #415 squash `b787bb98f`; executor (opus, worktree) M1–M3, evaluator (sonnet, generator≠judge) PASS 89/100 r1; teaching prompt v0.16.3 (append-only) + `formatter.md` Adoption section + `make fmt-check-ail` (renamed off the Go gofmt gate; `make ci` byte-identical) + opt-in `format_ail.sh` hook w/ Mark-approved SIGTERM→grace→SIGKILL escalation; doc → `implemented/v0_30_0/m-ailang-fmt-adoption.md`. Controller disproved a false "docs build failure" (skipped CI-only sync-registry.sh gen step). Was: IN-SPRINT plan iter 64; UNBLOCKED iter 63; DOC CREATED + QUORUM-BLOCKED ×3 → PARKED; Rev-3 iter 60]**
  — discoverability + opt-in hooks. **NOTE (iter 63): with phase2's 15.28% inline-interior fail-closed refusal, a per-turn
  auto-`fmt --write` hook still no-ops on ~15% of real commented files — adoption scope should account for this (teach
  fmt in the prompt + `--check`/`--write` hooks are now viable for the ~85% lossless majority; the ~15% refuse cleanly, exit 2).**
  Rev-3 iter-60 (`…/m-ailang-fmt-adoption.md`, `d1ed2fe57`). Rev-3 FIXED the jq
  defect (`command -v jq` probe + dropped first-jq `2>/dev/null`); re-quorum accepted it but both reviewers reject
  the timeout fix (SIGTERM-then-unbounded-`wait` wedges on a signal-ignoring proc) — **1 trivial SIGKILL-escalation
  from clean**, but hard-gated behind phase2. **→ Mark 2026-07-19: SIGKILL-escalation correction APPROVED as
  written in the doc; no re-quorum; still rides behind phase2** (which is now unparked). Original scope retained below.
  — Mark #399: "Is ailang fmt discoverable by agents via prompt… run every turn after .ail writes by Motoko or a
  hook in other harnesses?" iter-58 findings: (1) `ailang fmt` is **NOT** in `ailang prompt` (embedded v0.16.2
  teaches `check`/`run`/`test`, not `fmt`) → agents don't know it exists. (2) A per-turn auto-`fmt --write` hook
  (Claude Code PostToolUse on `*.ail`, Motoko per-edit) is **near-useless pre-Phase-2** — it would exit-2/no-op on
  87.5% of real files. Scope once Phase-2 lands: teaching-prompt line + CLI discoverability + opt-in harness hooks
  (`--check` in CI, `--write` post-write). Deliberately NOT teaching fmt in the prompt yet — teaching a tool that
  refuses 87.5% of commented files would frustrate agents (no-premature-adoption). ·
  **m-ailang-fmt-inline-interior** (**[LANDED 2026-07-21 (iter 70) — PR #434 squash `3c1cec57d`; UNPARKED by
  Mark #422 "Continue the AILANG fmt sprint" (bc61ea8ce: quorum objections DATA-REFUTED, proceed on the data, no
  re-quorum); planner (opus) → executor (opus, worktree) M0–M3 → evaluator (sonnet, generator≠judge) PASS 91/100 r1.
  Printer-local conditional multi-line let-chain emitter (option (a)); `internal/parser`+`internal/ast` READ-ONLY.
  **M0 surface-AST gate** (`TestInlineInterior_LetChainSurfaceShape`) proved all 28 targets chain via nested
  `*ast.Let.Body` (0 `Block.Exprs`) — the iter-67 R2 quorum objection data-refuted by a real TEST, not an assertion.
  `comment-unattached` refusals **59→32** (15.28%→8.27%, 27-file/45.76% reduction); `let-chain-interior` sub-class
  **fully eliminated (==0)**; residual 32 = deferred non-let/inline-tests/no-enclosing-list classes (records.ail's
  footer comment keeps it fail-closed → 27 achievable not 28). Idempotent, never-lossy (marker-fail=0,
  PHASE2-rt-regression=0). doc → `implemented/v0_30_0/`. **Orphaned-crashed-run recovery**: iter-70's inner loop
  ran + finalized + opened PR #434, then an 18h reboot outage killed it pre-Gate-3b/4; THIS run (same iter-70)
  resumed at Gate-3b — diagnosed the docs-`build` red as a **stale-base mermaid npm ERESOLVE** (branch cut before
  `08da65dc4`/#435 re-pinned `@mermaid-js/layout-elk` to `^0.1.9`), `gh pr update-branch` merged dev in → CI green
  → auto-merge squashed `3c1cec57d`. `metered=$0.00`]**) →
  [implemented/v0_30_0/m-ailang-fmt-inline-interior.md](implemented/v0_30_0/m-ailang-fmt-inline-interior.md).
  Log entries 72 (design/park) + 75 (land). ·
  **m-fmt-properties-printer-roundtrip** (**[LANDED 2026-07-20 (iter 69) — PR #424 squash `942931816`;
  UNPARKED by Mark #422 "Continue the AILANG fmt sprint and go to sprint planner"; planner (opus) →
  executor (opus, worktree) M1–M2 → evaluator (sonnet, generator≠judge) PASS 98/100 r1; contract-clause
  printer round-trip fix + silent-contract-deletion data-loss fix (`parser_func.go` `=`→append) + 2
  adjacent Phase-1 printer bugs the full-corpus sweep exposed (precedence-driven `;`-separation;
  `@verify(depth:)` key re-synthesis) → `preexisting-Phase1-rt-bug` gate 28→0, hardened; 30 contract
  examples reformatted; doc → `implemented/v0_30_0/`. Was: PARKED needs-human-review iter 66 (quorum R2
  data-refuted). `metered=$0.00`]** →
  [implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md](implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md),
  see its ⛔ Quorum Record. **Controller repo-wide re-check DATA-REFUTES the residual objection**: the
  only `ast.FuncDecl.Properties` consumers repo-wide are exactly the V17 sites (`internal/elaborate`
  + `internal/testing`); the `cmd/ailang/test.go` `.Properties` hits are a distinct `[]PropertyResult`
  results field, not the AST field; no accessor/interface/visitor indirection. Scope-corrected doc:
  the real defect is `requires`/`ensures` contract clauses (NOT `properties[...]` blocks) failing the
  Phase-1 printer round-trip (exit 2) on 30 corpus files, PLUS a latent silent-contract-deletion
  data-loss bug (`parser_func.go:169` `=`→append). **Human fork on #399:** (1) authorize routing to
  sprint-planner [RECOMMENDED — sole objection data-refuted, not fmt-phase2's deepening gaps],
  (2) authorize one more bounded round to fold the repo-wide audit into the Verification Log,
  (3) keep parked. ~1d impl, LOW risk/conflict, metered $0.1347 quorum (iter-66). Log entry 71. ·
  ~~M-TOOLING-DETERMINISTIC (normalize/suggest-imports/apply, 3–4d)~~ **[REALITY-CHECKED iter 48
  (2026-07-18) → PREMISE SUPERSEDED; scope-close PARKED for Mark.** The CLI trio doesn't exist, but
  its premise (single-shot fragment + LLM repair) is obsolete — `prompts/repair_prompts/` deleted,
  eval flow is agentic w/ per-edit `ailang check` feedback — and its core capability already ships as
  `normalizeProgram` (`internal/eval_harness/normalize.go`, deterministic module-wrap + std/io inject).
  Per-goal: G1 normalize=SHIPPED; G2 suggest-imports=PARTIAL/ABSORBED (std/io only; general
  symbol→import now met by implicit prelude + agentic feedback + `ailang docs`); G3 apply=obsolete
  (agents edit files directly). Regression guard `TestNormalizeProgram_MToolingMotivatingFragment`
  landed. Doc header → REALITY-CHECKED w/ per-goal table. **Mark scoped DX tooling "both in" → not
  ruled out unilaterally; awaiting his SUPERSEDED-close vs. much-smaller "expose `ailang normalize`"
  decision on #399.** Recommend prefer m-ailang-fmt for any DX budget. Log entry 53.]**
- **Prompt-diet** (GATED — unblocks once the diagnostics above land + the curve authorizes):
  m-eval-slim-prompt-self-discovery (R3.1 pass-rate-per-token curve, 2d) → prompt-deletion pass R1.2

### Clause 4 — orchestration flagship (Mark: full surface in)
- **Effect sprints** (decomposed): **m-effect-replay-contracts (2/4, 3d) [LANDED PARTIAL iter-99 →
  implemented/v0_30_0 — Mark option-(b) unparked it; controller fold → planner opus → executor opus/worktree
  → evaluator sonnet PASS 86/100 r1. Registry + os/seeded/crypto dispatch machinery + trace + golden
  gate all GREEN. BUT seeded/crypto not `.ail`-reachable → new gating item below]** ·
  **m-effect-replay-subsumption [M0+M1 LANDED iter-108 `b8249ef35`; M2 LANDED iter-110 `63b0ba3dd` — validate-path relaxation, evaluator sonnet PASS 88/100 r1 zero blocking, asymmetry proven by a two-binary matrix (`blocker`/`c2`/`c6` flip 1→0 while `c3`/`c4`/`c7`/`c8` stay rejected); **M3 LANDED iter-111 `12e5df162` → SPRINT COMPLETE**, doc moved to `implemented/v0_30_0/`. End-to-end acceptance: `main` held os-only per F3 (`verify_examples.sh` hardcodes `--entry main`), seeded/crypto behind separate entrypoints, and BOTH test controls **mutation-proven** — a constant `seeded_roll` breaks the different-seed assertion (line 67), flipping the `Rand/os` taxonomy entry to `Opaque` breaks the os-trace assertion (line 77); both reverted SHA-256-identical. Evaluator sonnet **PASS 84/100 r1, zero blocking**; its NB-1 (the guide taught `deterministic_roll`, a name absent from the very example file the guide tells you to run) and NB-2 (os-mode never exercised at runtime) were both reproduced first-party and FIXED before landing. `ai_modes.ail` red confirmed pre-existing against a pristine-base capture taken before handoff, and deliberately NOT greened with an AI edge (Q1)]**
  → [implemented/v0_30_0/m-effect-replay-subsumption.md](implemented/v0_30_0/m-effect-replay-subsumption.md),
  doc `011da3c81`, plan [m-effect-replay-subsumption-sprint-plan.md](planned/v1_0_0/m-effect-replay-subsumption-sprint-plan.md)
  `23a7a8210`; P0, ~2.5–3d, **4** milestones (the planner added M0 — `effects.go` is 720/800 against a
  hard `check-file-sizes` gate). **M0+M1 landed iter-108**: planner opus → executor `codex:gpt-5.6-sol`
  → evaluator sonnet **PASS 81/100 r1, zero blocking**. M1 is deliberately a **STRICTNESS increase, not
  the relaxation** — declarations now carry full elaborated `*types.Row` through validation instead of
  `ast.EffectNames` name-only erasure, so `c3`/`c7`/`c8` flip 0→1 (controller-verified BEFORE/AFTER with
  two binaries on its own fixtures); `blocker`/`c2` stay 1, which is M2's job. **The real M1 deliverable
  turned out to be F4**, which the quorum logged non-blocking and the planner promoted to hard:
  `UnionEffectRows` prefers its LEFT param map on conflict, so once M1 stopped erasing modes, a body
  calling both a `seeded` and an `os` helper would collapse to one mode — and if `os` survived, an `os`
  declaration would wrongly ACCEPT the `seeded` callee, reopening the exact hole M1 closes, in a build
  where every fixture is green and the suite passes. Closed with a conflict-preserving union;
  **mutation-tested non-vacuous** (restoring the left-preferring bug compiles and turns all three F4
  tests RED). **M2 carries two evaluator findings**: a conflict rejection exits 1 but does not NAME the
  offending mode (it falls into "no specific missing effects identified" — the doc's defect (2) in a new
  costume), and the conflict set can render as `! {Rand[mode=os|seeded]}` in a Suggested-fix line, which
  is not valid AILANG. Both are M2's structured-diagnostics deliverable. **DECIDED by Mark 2026-07-27 (attended, `4d32c71bb`):
  YES** — an explicitly declared mode SUBSUMES the bare/os effect requirement; implement as the narrow
  relaxation on the `SubsumeEffectRows` validate path ONLY (function-value mode distinctness unchanged).
  Unparks effect sprints 2–4. (Was: NEW GATING clause-4, PARKED needs-human-review iter-99.)
  **SHARED gate for the whole effect-mode-dispatch line** — the parent doc's `Clock[mode=pinned] = now()`
  examples have the identical structure, so sprints 3/4 (clock/net/fs) hit it too.
  ⚠ **CORRECTION (this row's own prior text was WRONG, refuted by iter-107's live repro at a CLEAN build
  of `4d32c71bb`)**: the claim *"`SubsumeEffectRows` treats effect modes as INVARIANT"* — inherited from
  the parent doc — is **FALSE** as observed on the `.ail` path. Enforcement is **one-directional**: an
  os caller silently ACCEPTS a `seeded` or `crypto` callee (exit 0), while the reverse rejects. Cause is
  structural, not a rule: `validate_effects.go:109-114` threads `declaredEffects map[string][]string`
  built via `ast.EffectNames` (labels only), so `stringSliceToEffectRow` **cannot carry mode params** —
  the required row is **mode-blind** for every declared-function call. Two further defects found: the
  rejection prints **`Missing effects:` with an EMPTY payload** (label-set difference; a mode mismatch
  has identical labels), and naively relaxing the blocker without closing the mode-blindness would make
  seeded/crypto declarations **vacuous**. The doc scopes full-row propagation in and flips the
  wrongly-accepted direction to rejected — measured in-repo breakage: **ZERO** `.ail` files declare a
  non-default mode outside comments. **Quorum**: designer `codex:gpt-5.6-sol`; reviewers `gpt5-6-sol` +
  `gemini-3-1-pro` + controller opus; 3 rounds — R1 narrowed the rule from "any effect with a registered
  default" to explicit **schema-registered edges, Rand-only**; R2's attribution objection was answered by
  rebuilding clean at the exact SHA and re-running all 9 fixtures (**identical**), then the
  narrow-refinement carve-out. **NEXT**: route to sprint-planner. ·
  m-effect-clock-net-fs-modes
  (3/4, 3d — BLOCKED behind sprint-2 AND the subsumption decision) · m-effect-scope-params (4/4, 2.5d — release-gate re-score candidate)
- **Flagship + surface**: m-v1-orchestration-flagship (verified AI-pipeline example + orchestration
  benchmarks into rotation + README/site lead, 2–3d; m-contracts-as-code-vertical folds in as the
  worked example) · m-serve-api-live-tool-registry (hot MCP tool registry, 3–4d) ·
  **m-serveapi-raw-handler-mcp** (**[LANDED 2026-07-22 (iter 78) → implemented/v0_30_0; M1 `@nomcp`
  shipped, M2 DROPPED → doc COMPLETE. Planner opus → executor opus (`2d6596292`) → evaluator sonnet
  (generator≠judge) PASS 96/100 r1, no defects; PR #452 squash `ee04f13d0`, dev CI green per-workflow
  (19 checks). Closes the live docparse `getKeyUsage`/`requestHistory` MCP capability leak with a
  one-line annotation; diff confined to parser allowlist + `internal/apiserver/` — NO eval-core change.
  Was: DECIDED by Mark 2026-07-20 → ROUTABLE; M2 fake-envelope DROPPED, no re-quorum. Historical park:
  PARKED iter 57 — QUORUM-AT-PICK BLOCKED ×2]** →
  [implemented/v0_30_0/m-serveapi-raw-handler-mcp.md](implemented/v0_30_0/m-serveapi-raw-handler-mcp.md), see its ⛔ Quorum Reblock section;
  M1 `@nomcp` MCP-exclusion annotation — keep a `@route` on HTTP but off the `--mcp-http` tool surface
  (`@noexpose` can't: it also kills HTTP + is overridden by `@route`) — closes the live docparse
  `getKeyUsage`/`requestHistory` MCP leak; **M1 is CLEAN + unobjected in both rounds → independently shippable**.
  M2 `@raw`-over-MCP twice-rejected: R1 default-on = authority-widening + silent header-fabrication; R2 the
  `@mcp`-opt-in + typed-sentinel fix itself violates the frozen core (`headers`/`query` are `Json` → a non-`Json`
  sentinel type-panics at binding; a `Json` sentinel needs core `std/json` changes). **Human fork on #399:**
  (1) split+ship M1 now [RECOMMENDED], (2) pick an M2 arch — valid-`Json` provenance marker
  `{"_transport":"MCP_UNAVAILABLE"}` + require `req.method=="MCP"` branch, OR drop the fake-envelope entirely,
  (3) keep parked. Unblocks docparse quota-hardening item 5. Log entry 62. ~0.5d for M1 alone) ·
  m-agent-step-cancellation (1.5d) ·
  **m-ai-reasoning-effort** (**[LANDED 2026-07-22 (iteration 83) → implemented/v0_31_0; full inner
  loop headless — planner opus (7-milestone plan, caught OpenRouter-already-has-Effort premise drift
  → M5 replace-not-extend) → executor opus (M0–M6, 7 commits `08e2aa935`→`3e784748c`) → evaluator
  sonnet (generator≠judge) PASS 96/100 round 1, ZERO defects. Typed `ai.Request.ReasoningEffort` +
  ONE shared fail-loud resolver across all 12 Generate/Step/streaming constructors of 4 clients;
  5 typed sentinels reuse `AIError`/`CodeSchemaValidation` (via `Unwrap()`); capability table ships
  EMPTY → unknown model + explicit control = typed reject (no silent fallback; OpenAI `"off"` never
  → `"minimal"`); Anthropic hook precedes `MaxTokens=4096` default; Gemini/Anthropic `B=0` exemption;
  OpenRouter Effort-wins branch REPLACED. AC #14 byte-identical goldens per provider; 16/17 ACs
  network-free (4 NEEDS-LIVE-SMOKE cap-entries + AC17 notify = parked M7 metered follow-up).
  Minimal-Frozen-Core (`internal/ai/**` only). PR #460. Mark fork (a) authorized; M0 code-audit
  executed gpt5-6-sol's inventory ask; no re-quorum. `metered=$0.00`.]**
  **[was: Mark 2026-07-22 fork (a) → ROUTE TO SPRINT-PLANNER: both authorized
  fixes GREEN (gemini PASS); Sol's machinery-inventory ask folds into the planner's mandatory M0
  code-audit. Doc front-matter stamped. Historical: PARKED — Mark's authorized bounded round EXECUTED
  iter 81 (2026-07-22): BOTH named R2 objections RESOLVED (reasoning_max_tokens 4th resolver input;
  Gemini B=0 MaxTokens exemption); re-quorum gemini→PASS but gpt5-6-sol→REJECT on a NEW out-of-scope
  "inventory existing AI-package machinery" objection → bounded round consumed → RE-PARKED. Rev-2 doc
  merged PR #457 → `893873c81` (docs-only, CI green). Human fork: (a) route to sprint-planner folding
  gpt5-6-sol's inventory ask into planner M0 code-audit [RECOMMENDED]; (b) one more bounded revision;
  (c) keep parked. Designer=claude-fable-5 ($0), re-quorum metered=$0.062. Log entry 86.]**
  **[was: AUTHORIZED by Mark 2026-07-20 → ROUTABLE: ONE more bounded
  revision+re-quorum round, scoped to the 2 named R2 objections only — doc front-matter stamped.
  Historical park:]** **[was: PARKED iter 61 — QUORUM-AT-PICK: R1 BLOCKED
  (no-silent-fallback + missing MaxTokens conflict surface) → codex-designer Rev-1 resolved both
  (fail-loud contract w/ 5 typed errors + capability gating + full Conflict Surface) → R2
  re-quorum BLOCKED on 2 NEW *narrower converging* fixes]** →
  [planned/v0_29_0/m-ai-reasoning-effort.md](planned/v0_29_0/m-ai-reasoning-effort.md), see its
  ⛔ Quorum Record. R2 objections: (1) resolver omits OpenRouter's `reasoning_max_tokens` 4th input;
  (2) Gemini rule over-reaches forcing `MaxTokens` for `B=0` "off" (breaks docparse consumer).
  Both small/concrete — NOT fmt-phase2's deepening gaps. **Human fork on #399:** (1) authorize
  ONE more bounded round [RECOMMENDED — close to green], (2) amend scope (drop `reasoning_max_tokens`
  from the typed resolver), (3) keep parked. ~14h impl (doc est), metered $0.23 (iter 61). Log entry 66.
  **REALITY-CHECK (iter 66, log entry 71): STILL PARKED — the feature did NOT land out-of-loop.** The
  iter-65 "Next" flagged commit `5afa9a1e1` ("feat(eval): reasoning_effort knob") as a possible
  out-of-loop landing. REFUTED: that commit is an EVAL-HARNESS-only OpenRouter `reasoning.effort` knob
  (`models.yml` + `internal/eval_harness/` + `openrouter/chat.go`), NOT this doc's typed
  `ai.Request.ReasoningEffort` field. Verified absent on origin/dev: `git show origin/dev:internal/ai/provider.go`
  has no `ReasoningEffort`, and the 5 sentinel errors (`ErrUnsupportedReasoningEffort`, …) do not exist.
  The v0.31.0 cross-provider feature is unbuilt; the R2 fork above still awaits Mark.)

### Clause 5 — cost credibility
- **[LANDED 2026-07-27 — M1–M3; M4 parked-for-Mark]** m-cost-per-success-kpi (dashboard KPI flip to cost-per-verified-success + v1.0 measured baseline) — **DESIGN DOC iter-103** ([planned/v1_0_0/m-cost-per-success-kpi.md](planned/v1_0_0/m-cost-per-success-kpi.md), quorum-cleared via carve-out). **Iter-104: M1–M3 shipped** via full sprint loop (planner/executor opus, evaluator sonnet **PASS 86/100 r1**, dev CI green @ `d869ec12d`): M1 `9bdc9319c` observatory strict `cost_per_verified_success` rollup (single `isVerifiedSuccess` predicate reusing `ClassifyStageCost`/`TotalKnownCost()`; `verify_*` propagated into both `EvalAssessment` constructors; cohort filter) · M2 `2a2a40f31` CLI `--cost-per-verified-success --baseline --strict` + HTTP + additive `latest.json headlineKpis` (one struct, field-for-field) · M3 `2d76b2cc3` headline `ValueDashboard` card + Fallback/stale badge + available/zero-denom/incomplete/absent states. Doc stays in `planned/` until M4b lands. **Iter-105: Mark RATIFIED all three M4 inputs** (#484 comment `2026-07-27T07:53:53Z` — verified-success definition "Yep ok"; cohort "assume current cohort but this may have light changes depending on release date"; headline placement "Fine") → M4 unparked and **split M4a/M4b**. **M4a LANDED** (planner opus → executor opus [codex probe FAILED, fallback FLAGGED] → evaluator sonnet **PASS 96/100 r1, zero blocking**): M4a-0 `37c070dd9` file-size extraction · M4a-1 `612cb78af` `--baseline <id>` cohort-freeze flag + BF-2 SQL-`LIKE` `_`-wildcard escape via ONE validator shared write+read · M4a-2 `fa4c1d095` data-driven `cohort_manifest.json` + `cohort_hash` (models resolve from `models.yml`, zero model names in Go → re-freezable per Mark's caveat) · M4a-3 `522ad61f1` **closes BF-1** · M4a-4 `6b252b9b1` docs/changelog/doc-split. **BF-1 was the iteration's real find**: agent-mode contract verification was NEVER wired on the live path — the only agent `RunAICheck` sat in `RunAgentBenchmark`, a function whose sole repo reference is a comment saying it must not be used, while `RunAgentBenchmarkWithExecutor` had zero Verify. So `verify_verified` was always 0 → `isVerifiedSuccess` always false → **the M1–M3 headline KPI structurally could never produce a number**; a cohort run before this fix would have burned metered dollars banking a guaranteed zero denominator. **M4b DECIDED by Mark 2026-07-27 (attended; was PARKED-for-Mark)**: (i) metered spend APPROVED,
  cap **$20 total** for the cohort run (raise `MISSION_METERED_BUDGET_USD` to 20 for that single
  iteration if needed) — BUT the cohort run **WAITS for the Anthropic key-quota reset (~2026-08-01)**
  per Mark's same-day call ("we can wait a few days for an actual eval run") — do NOT fire it
  before; (ii) ACCEPTED — publish the explicitly-labelled *list-price-equivalent* KPI with a
  distinct provenance status, per the iter-105 recommendation. Historical ask: (i) approval to spend metered dollars on the real cohort run (OpenRouter lanes are the real-dollar exposure; `$5` iteration ceiling); (ii) **cost-provenance decision** — `ClassifyStageCost` rule 1 treats any `cost>0` as authoritative "reported", but the *subscription* claude CLI reports non-zero `total_cost_usd` while nothing is billed (live-probed with both Anthropic keys stripped: 10 in/46 out → `$0.0108355`), so a claude+OpenRouter cohort blends list-price-equivalent and truly-metered dollars under one label, against the doc's "attributable **metered** dollars" goal.

### Clause 2 — soundness (near-done; no new holes found in triage)
- **[LANDED 2026-07-28 — iteration 116] m-z3-hard-timeout** (`#510`; PR **#514** → squash `9253ec8a8`, dev CI green on all 14 checks incl. `test-windows`; plan → [implemented/v0_31_0/m-z3-hard-timeout-sprint-plan.md](implemented/v0_31_0/m-z3-hard-timeout-sprint-plan.md); evaluator **sonnet PASS 90/100 r1, zero blocking**) — **Mark's option (B) pick**, the precondition for `m-z3-adt-record-sort`. Both Z3 `exec.Command` sites in `internal/smt` are now bounded: `Solve` (was `solver.go:147-148`) at `max(config.Timeout, effective -T: secs) + 2s`, and — **systemic twin the issue never named** — `Z3Version()` (was `solver.go:271`), which is NOT cold since `cmd/ailang/verify_print.go:23` calls it on every human-mode `verify` header. `grep "exec.Command(" internal/smt/` now returns **zero** non-Context sites. Process-group kill (`Setpgid`, SIGKILL to `-pid`, ESRCH tolerated) + `cmd.WaitDelay`; the deadline is classified **BEFORE** output parsing so a truncated prefix from a killed process can never be read as a verification result. Caller-visible shape preserved (`StatusUnknown` + `"solver timeout"`), so `verify.go:427`/`ai_check.go:370` see nothing new. **Non-vacuity proved by the controller in THREE directions**, each restored byte-identical (sha256 `b9e65c78…`): pre-fix `Solve` → FAIL 30.36s · **the NAIVE fix** (deadline kept, group-kill removed) → passes timing AND status, fails only on `child process 88072 still exists after 2s` · pre-fix `Z3Version` → FAIL 30.35s. The middle mutation is the valuable one — it proves the orphan assertion catches the half of the bug a plain `CommandContext` would miss. Tests use a fake solver ignoring `-T:`, need **no real z3**, run in CI, **zero `t.Skip`**. Closes a Standing-rule-6 violation on the verification surface for every `ai-check` caller incl. sibling Ailang World. **Not fixed, filed as `#513`**: per-call bounds do not bound a whole run of N functions.
- **[LANDED 2026-07-29 — iteration 117; PR #516 → squash `5998f4039`, dev CI green; docs → [implemented/v0_31_0/m-z3-adt-record-sort.md](implemented/v0_31_0/m-z3-adt-record-sort.md) + its sprint plan; evaluator sonnet r1 **FAIL 59/100** → r2 **PASS 83/100**] [world-DEMAND] m-z3-adt-record-sort** — **the defect was TWO layers**, not the one the doc localised: the encoder's Step-0 alias fixpoint dropped the record declaration silently (`if !progress { break }`) AND `filterADTTypesForFunction` never walked record-alias fields, so the ADT's variant list never reached `EncodeFunction` — an encoder-only fix could not have worked. Both drivers now share one `filterSMTInputsForFunction` (`ai-check` had NO demand filter, so the KPI was **deflated**, not inflated). `validateDeclarations` now covers `declare-const`, typed `define-const`, recursive `(Seq X)` and plural groups **atomically** — closing a pre-existing hole where `HasPrefix(decl, "(declare-datatype ")` never matched `(declare-datatypes (`, so every mutually-recursive group bypassed the guard. `ai-check` exits 1 on `verify.errors > 0` (**breaking** for out-of-repo shell callers; JSON still emitted first). Declaration order is now deterministic (**A1**) — measured 40 runs → 3 orders before, 40/40 identical after. Corpus verified **76→81**, skipped **10→7** (a coverage fix, NOT a model improvement — flagged in CHANGELOG). ⚠ **Follow-up owed**: AC1.3 verify/ai-check parity table, AC3.3 JSON-before-exit subprocess guard, AC2.4/2.5 named mutations. Was: (DOC WRITTEN `planned/v0_31_0/m-z3-adt-record-sort.md`, designer `codex:gpt-5.6-sol`, quorum BLOCKED ×2; **park is ONE decision for Mark** — options (A)/(B)/(C) in the doc's "Quorum round 2 — PARKED" section; recommended **(B)** sprint `#510` first, then this routes to sprint-planner unchanged. Controller re-reproduced the bug at `f495885b1` and **REFUTED the issue's stated root cause** — the record's `declare-datatype` is ABSENT, not mis-ordered, and a direct ADT param verifies, so the ADT machinery already works. Round-2 objections split: gemini's CLOSED by controller measurement, gpt5's CONFIRMED → filed **`#510`** (unbounded Z3 exec, `solver.go:147-148`). Original sizing: P1 for clause-4/5 credibility; two lanes, ~0.5d + ~2–3d) — **`ailang#477`, filed by the Ailang World mission 2026-07-24, live-REPRODUCED at HEAD by iter-106** on a freshly-built `v0.30.0-197-g22c1eecd5` (ghost discipline applied — not a ghost). A `requires`/`ensures` contract on a function whose parameter is a **record transitively containing a user sum type** cannot be verified: Z3 gets `Invalid constant declaration: unknown sort '<Record>'` because the encoder declares the record's sort without declaring a sort/datatype for the contained ADT — **and `ai-check` exits 0** with `verify.errors: 1`. Sibling's bisection (confirmed): scalars / `list[string]` / nested user *records* encode fine; **any** sum-type field breaks the enclosing record, even single-constructor, even unreferenced by the predicate body. **Lane A — exit-code honesty** (small): `ai-check` must exit non-zero when `verify.errors > 0`; audit every caller FIRST, since gates that pass today only because of the bug will start failing (that is the point, but it should be a decision, not a surprise). **Lane B — the encoder** (real work): declare Z3 datatypes for user ADTs reachable from a contract's parameter types. **Controller-verified impact bound (do not overstate it)**: this does NOT corrupt the v1.0 headline KPI — `VerifyOk` is derived from the JSON block, not the exit code (`internal/eval_harness/verify.go:141`), and `isVerifiedSuccess` independently requires `verify_errors == 0` (`internal/observatory/cost_per_verified_success.go:94`), so an encoder error correctly EXCLUDES a run rather than counting it as proved. The real exposure is (1) any gate consuming `ai-check` by **exit code** — the normal shell/CI idiom, and a NO-SILENT-FALLBACKS violation on the verification surface — and (2) contract coverage silently bounded by an undeclared type-shape restriction. Repro fixture: the sibling's `adtsort.ail` (in #477). Triage verdict posted to ailang#477 and to the World bookkeeping thread `sunholo-data/ailang-world#9`. **Demand evidence is satisfied by construction** — a real downstream consumer is blocked (World's `w-m1-ailang-hardening` drops from 7 to 4 provable predicates).
- **[LANDED-LANE-A 2026-07-29 (iter-120) — PR #529 → squash `aa02f0d9f`, dev CI GREEN (15 checks, 0 failures, SHA-addressed); 5 commits incl. a Windows `.exe` fix Gate 3b caught; evaluator sonnet PASS 89/100 r1, zero blocking. Lane A shipped `serve-api --no-feedback-tool` (DEFAULT UNCHANGED — the live public Cloud Run service runs `--mcp-http --routes-only` and depends on the built-in surviving; tightening `--routes-only` would open a version-skew window that silently kills the public feedback channel) + the `--caps` discovery-vs-execution docs. **BONUS, and the more serious defect: `#528`** — A2A `tasks/send` never checked `isExposed`, so `@noexpose`/`--routes-only` hid a function from the agent card while leaving it CALLABLE; HTTP/MCP/OpenAPI and A2A's own card all gated, dispatch was the only hole, violating `.claude/rules/api-server.md`'s documented single-filtering-point invariant. Planner marked it cuttable; controller confirmed it first-party and upgraded it to REQUIRED (Principle 3). **Two premises REFUTED**: `--a2a` discovery is NOT affected (`buildAgentCard` never touches the MCP registry — `#498`'s title over-reaches and the controller's restatement inherited it; a milestone 'fixing' it would have been a no-op shipping green), and the `std/io` leak did not reproduce at HEAD. Five mutations all controller-run red, reverted byte-identical. **LANE B STILL OPEN — see the [NEXT] row below**] [world-DEMAND] m-mcp-exact-tool-surface Lane A** (was: NEW-DOC needed; P2 — **not a v1.0 bar item**, filed in this section beside its sibling World filing for discoverability, NOT because it is a soundness row; interim lane ~0.5d, broad lane ~2–3d) — **`ailang#498`, filed by the Ailang World mission 2026-07-28, live-REPRODUCED at HEAD by iter-110** on a freshly-built `v0.30.0-215-g7c0568797` (ghost discipline applied — not a ghost). `ailang serve-api --mcp/--mcp-http/--a2a` cannot project an **exact caller-supplied tool surface**: the built-in `submit_feedback` tool is advertised under EVERY flag combination — measured `unfiltered [addOne, submit_feedback]` · `--routes-only [submit_feedback]` · `--caps '' [addOne, submit_feedback]` · `--caps IO [addOne, submit_feedback]` (the last row is the controller's addition and corroborates that **`--caps` gates effect execution, not discovery**). **The sibling labelled its cause a HYPOTHESIS; the controller VERIFIED it** — `internal/apiserver/mcp.go:43` calls `registerFeedbackTool()` unconditionally inside `NewMCPServer`, *after* the `registerTools()` surface that `--routes-only` filters; the function takes no predicate, has no other caller, and **no env or flag off-switch exists anywhere in the repo** (`AILANG_FEEDBACK_GATE_*` is the coordinator's triage gate — `internal/feedbackgate/decide.go:14` explicitly says do not entangle the two). **Controller-verified impact bound (do NOT overstate it — this is the P0/P2 hinge)**: the *discovery* defect is unconditional and real (every connected model is told a public-feedback egress tool exists), but *egress itself* is gated on `AILANG_STORAGE=gcp` **and** `AILANG_CLOUD_PROJECT` (`internal/feedback/publisher.go:123-129`) — without both, the first call returns a structured error envelope and opens no client. So a default local server **advertises an egress tool it cannot perform**: a false capability claim to every connected model, plus a live path for environments that happen to carry those two vars — NOT a default-config exfiltration. **Lane A — interim** (small, unblocks the consumer soonest): a flag/option that suppresses `submit_feedback`, plus documenting the `--caps` discovery-vs-execution split. **Lane B — the real ask** (design-doc-sized, quorum first): export the existing serving machinery behind a narrow callback-driven Go API — caller-owned mux, principal/session resolved BEFORE discovery *and* invocation, caller-supplied exact descriptors, MCP tools and A2A skills generated from that one set, nothing built-in unless the caller supplies it. **Demand evidence satisfied by construction** — World's `w-mcp-projection` is recorded BLOCKED on it. Verdict posted to ailang#498 and to `ailang-world#9`, with the explicit note that **no date was promised**, so World keeps its item BLOCKED rather than waiting on us. Informational, independently confirmed: **`#145` is genuinely fixed** (`--routes-only` did suppress the 8 embedded `std/io` exports in every run).
- **[DOC LANDED + QUORUM-CLEARED 2026-08-04 (iter-137) — PR #582 → squash `2629ad8fa`, dev CI GREEN
  SHA-addressed (20 checks, 0 non-success) + per-workflow confirm. Doc:
  [implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md](implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md),
  785 lines, 28 verification rows, 10-row Conflict Surface. Designer `codex:gpt-5.6-sol`; reviewers
  `gpt5-6-sol` + `gemini-3-1-pro` BOTH PRESENT both rounds (no N−1 degrade); R1 BLOCKED ×2 → revision;
  R2 BLOCKED ×2 → **narrow-refinement carve-out** (both objections carried concrete reviewer-authored
  `proposed_fix`, neither disputed the design DIRECTION) → reviewers' VERBATIM fixes applied and
  recorded in the doc's Quorum Verification Log. Metered $0.1910. **The premise that shrank the work**:
  the MCP Go SDK already hands us `getServer func(*http.Request) *mcp.Server` and `mcp.go:303` calls it
  while DISCARDING the request — so #498 reqs 2–3 are a wiring/authority problem, not a transport one.
  **Two reviewer claims MEASURED, not forwarded**: (1) "per-request SDK servers break SSE" REFUTED on
  this path (Stateless mode answers GET/DELETE 405 `Allow: POST`, so no stream exists to correlate) —
  but its real adjacent landmine is closed: `Stateless:true` frozen, stateful/resumable MCP an explicit
  non-goal, GET⇒405 asserted; (2) "A2A timeout mapping unverified" — process point correct, corruption
  impossible: `a2a.go:304` already writes HTTP 200 + JSON-RPC (V27), `-32603` new to the file with a
  known-positive control (V28). R2 also caught that a deadline bounds the WAIT not the RESOURCE →
  `MaxConcurrentCallbacks` with the token held until the goroutine EXITS, plus the honest statement
  that in-process Go callbacks cannot be forcibly terminated.
  **[PLAN-READY 2026-08-04 (iter-138) — commit `6e82d2a1b`: plan
  [implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b-sprint-plan.md](implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b-sprint-plan.md)
  (631 lines) + `sprint_M-MCP-EXACT-TOOL-SURFACE-LANE-B.json` (`jq -e` rc=0, repo validator PASSED,
  zero placeholders with a known-positive control). Planner **opus**, fired via the Agent tool
  because `derive-planner-lane.sh` returned `opus declared:opus-required` — so NO codex probe or
  spawn happened for the planner role, the first end-to-end exercise of the iter-136 flip's opus arm.
  **Estimate revised 17h → 25h** (test LOC ≈ 70% of impl LOC; calibrated against Lane A's ~490 LOC
  of real work), no AC cut. **The planner refuted FIVE doc premises, all reproduced first-party by
  the controller → rows V29–V33 + an AC-corrections section:** two of M3's acceptance criteria
  **could not fail** (`make check-file-sizes` enumerates `find internal cmd` only, so it is blind to
  the `serveapi/` package this sprint creates; `make check-boundaries` iterates three fixed package
  sets containing neither `apiserver` nor `serveapi`) — and the doc's own row **V18 had already
  measured the second one**, cleared by two reviewers across two quorum rounds, with the AC citing
  that gate left standing. Both replaced with checks that can fail. **New requirement from V31:**
  `mcp.Server.AddTool` PANICS on a missing/non-object input schema (SDK v1.7.0 `server.go:282,294`)
  and the design calls it PER REQUEST from CALLER-supplied descriptors, so nil-schema rejection must
  precede any `AddTool` and the adapter must recover into the frozen `-32603` envelope. V32: `@nomcp`
  is already a second MCP-only filter, so it must NOT be folded into the shared gate. V33
  (favourable): `internal/apiserver` binds zero sockets, so most of the sprint is authoritative
  inside the codex sandbox. Milestones reordered — three of M1's six ACs are unsatisfiable at the M1
  boundary, so the callback runner becomes protocol-neutral and the `isExposed` generalization moves
  M1 → M3 as the sharpest bisect boundary. Systemic gate-scope defect filed as **`#584`**.
  **NEXT: EXECUTE** — executor `codex:gpt-5.6-sol` with the no-git-writes + cumulative `.snap/M<k>/`
  snapshot protocol, worktree pinned to `.wt-iter139-mcp-lane-b` (sibling of the repo, never
  `/tmp`), evaluator sonnet.
  **[M1 LANDED 2026-08-04 (iter-139) — PR #585 → squash `f5ebcc0b5`, Gate 3b GREEN SHA-addressed
  (20 check-runs, 0 non-success) + per-workflow confirm; evaluator sonnet PASS 94/100 r1, zero
  blocking. Shipped the public `serveapi` package (139 LOC, stdlib types only), `callerSurface`
  deep-copy/validate/sort gateway, and the bounded callback runner whose capacity token is held
  until the callback goroutine EXITS — 691 insertions, 6 files, no existing call site touched and
  no wire behaviour changed (`MCPHandler`/`A2AHandler` are declared for the compile fixture and
  return `NotFoundHandler` until M2/M3). **Controller mutation testing found TWO VACUOUS
  ASSERTIONS the executor shipped**, both fixed and both now mutation-killed: (1) the
  "next callback never entered" clause was guarded on `if err == nil`, provably false there, so
  the counter assertion was a tautology (proven — an instrumented marker never printed while the
  test ran); (2) the deep-copy proof compared two `All()` results, which under a shallow clone
  both alias caller storage and change together — **the shallow-clone mutation SURVIVED, refuting
  the sprint plan's own "would still pass if the claim were false? No" answer for AC3**. 4
  mutations, each proven applied by `cmp` before its result was read, all reverted byte-identical.
  **M2 PREREQUISITE, evaluator NB-1, controller-reproduced**: a FIFTH `AddTool` panic case —
  `validateParamHeaderAnnotations` (`mcp/server.go:312-313`, invalid/duplicated/non-primitive
  `x-mcp-header`) — is NOT covered by M1 validation, so **M2 AC5 is not sound as written**;
  recorded in the plan (`63e051de6`) demanding BOTH loud validation AND a `recover()` backstop.
  `go test ./...` rc=1 was NOT this sprint: `TestNetHttpPost` on a live httpbin.org 504,
  reproduced at the untouched base, tracked as `#561`, CI-safe (proven both ways). Latent fixture
  traps filed as **`#586`**. **M2 + M3 REMAIN** — M2 next (~8.5h, architecture pinned in plan
  §0.6), then M3 (~10h).]**
  **[M2 LANDED 2026-08-05 (iter-144) — PR #592 → squash `6166adab8`, Gate 3b GREEN SHA-addressed
  (19 check-runs, 19 completed, 4/4 required contexts success, 0 non-success) + per-workflow
  confirm; evaluator sonnet PASS 94/100 r1, zero blocking (2 own mutations, both red,
  byte-identical reverts). Shipped the request-scoped MCP adapter (`embedded_mcp.go`, 232 LOC +
  414 test) per the plan's pinned §0.6 architecture: exact per-session surfaces, no ambient
  `submit_feedback`, frozen `-32603` timeout/capacity/cancel envelopes, 401/403 typed
  authorization, `Stateless: true`, and BOTH §0.5 halves — loud gateway rejection of the
  `x-mcp-header` annotation panic class AND a `recover()` backstop. **Controller mutation testing
  found the shipped AC5 test NON-DISCRIMINATING** (third sprint running for this class): the
  gateway-validation-removed mutant survived because the backstop also writes an error envelope;
  fixed by asserting the gateway's specific rejection messages, mutant now red. Evaluator NB-1
  (overload envelope HTTP-200 assertion) reproduced + applied (`811ac16b4`); NB-4
  (context.Canceled envelope vs the doc's "no completed wire response" prose) deferred to M3.
  **[M3 LANDED 2026-08-06 (iter-151) — PR #601 → squash `b8c038647`. Gate 3b GREEN SHA-addressed
  on the PR head: all FOUR required contexts (`test`/`lint`/`build`/`docs-gate`) `success`,
  20 check-runs completed, 14 success / 5 skipped. ⚠ ONE non-required failure, NOT buried:
  **CodeQL** `go/reflected-xss` high at `embedded_mcp.go:119` — established by measurement as a
  SCAN-CADENCE artifact describing **M2's** already-landed code, not this diff (file byte-identical
  between dev and PR head; CodeQL analyses dev only WEEKLY and last ran `2026-08-04T09:04`, ~22h
  BEFORE M2 landed `2026-08-05T07:33`, so that file had never been analysed on dev). Filed as
  **`#603`** with the true-positive question left open. The 2nd alert (`#129`) is pre-existing on
  dev since 2026-04-21 — my first read said "0 on dev" from a `per_page=50` page against ≥100
  alerts, corrected by paginating. Executor codex `gpt-5.6-sol`, evaluator **sonnet PASS 81/100 r1,
  zero blocking**. **LANE B IS COMPLETE.** Shipped `embedded_a2a.go` (162 LOC — cards + `tasks/send`
  projected from the SAME `AuthorizedSurface` as MCP, dispatch via `Lookup` so an unauthorized send
  is `-32602` and never reaches the invoker), live `A2AHandler()`, `Mount` onto a caller-owned mux,
  and `loadedExportMember` as the single protocol-neutral MEMBERSHIP gateway behind all 6 production
  `isExposed` sites with `@nomcp` kept as an MCP-only PROJECTION applied after membership (the §0.7
  hazard; controller-mutated, caught by THREE tests incl. the pre-existing
  `TestNoMCP_StillServedOverHTTPAndOpenAPI`). **THE FIND — the 4th consecutive non-discriminating
  test in this sprint, and the first where the REVIEWER'S PROPOSED FIX was also wrong**: the
  evaluator caught `TestMountRecorderRoutesAndMCPStripPrefix` passing with `StripPrefix` REMOVED and
  proposed a sub-path POST; measuring instead of applying it showed the MCP SDK's
  `StreamableHTTPHandler` **never dispatches on path** (`URL.Path` = 0 across `go-sdk@v1.7.0/mcp`
  non-test, `.URL` = 12 control), so the wrapper is behaviourally INERT and the AC was vacuous BY
  CONSTRUCTION — no assertion can distinguish it. Fixed by removing the false claim (test renamed,
  comment records the measurement and that the wrapper is deliberately uncovered) and replacing it
  with a `/mcp/` SUBTREE assertion that IS discriminating. Executor also REFUTED the planned
  transitive-import boundary check as impossible (`serveapi` transitively reaches 13 compiler
  packages; the correct gate is DIRECT-import, 0 with control 1). Doc + plan →
  `implemented/v1_0_0/`. **→ RELEASE ASK now owed to Mark** (World consumes pinned releases only).]**
  Was: M3 REMAINS — the FINAL Lane B milestone (~10h: A2A projection, Mount, exposure
  generalization, docs, CHANGELOG, gates).]**
  Was: NEXT — route to sprint-planner; the QUEUE HEAD after m-planner-codex-lane LANDED at iter-136.
  Was NEXT #3, REORDERED by Mark 2026-08-03 afternoon when the PLAN-READY m-planner-codex-lane
  execution jumped ahead (one ~8h mechanical sprint; every later iteration burns less opus).
  Original directive: PRIORITIZED for Ailang World — was pick #2 after m-recorded-stream-api] [world-DEMAND]
  m-mcp-exact-tool-surface LANE B** (doc DONE — was: NEW-DOC needed, quorum required; P2→**P1 by directive** —
  still not a v1.0 bar item, but it is World's SOLE clause-6 external blocker (their
  w-mcp-projection is BLOCKED on it, recorded in their charter) and Lane A alone does not
  give them the caller-supplied per-session surface. RELEASE NOTE for the controller: World
  consumes upstream via PINNED RELEASES only — when Lane B lands, surface a release ask to
  Mark in the report's DECISIONS row (releases are Mark's sole decision); a tag would also
  carry the already-landed #510/#477/#498-Lane-A fixes World is waiting on. (Original row:  ~2–3d) — the REAL ask in `ailang#498`, untouched by iteration 120's Lane A. Export the existing serving machinery behind a narrow callback-driven Go API: caller-owned mux, principal/session resolved BEFORE discovery *and* invocation, caller-supplied exact descriptors, MCP tools and A2A skills generated from that one set, nothing built-in unless the caller supplies it. **Demand evidence satisfied by construction** — World's `w-mcp-projection` remains BLOCKED on this (Lane A only unblocks it if a suppression flag suffices; the sibling was told explicitly that no date is promised). Lane A's landing means the interim workaround exists, so this is no longer urgent — but it is still the thing `#498` actually asked for.
- **[LANDED-LANE-A 2026-07-30 (iter-121) — PR #536 → squash `a81d66983`, dev CI green on the PR (15 pass, 0 failures); 5 commits. Doc RESUMED from a prior iteration-121 attempt that died 14:14 the same day leaving a 453-line doc on an UNMERGED branch (invisible to a `design_docs/` grep and to Gate 2's origin/merged-PR checks — see the log's process-fix note). Quorum: designer `claude:claude-fable-5`; R1 BLOCKED (both objections Lane-B2-only) → B2 DEFERRED on `gpt5-6-sol`'s own proposed option, blocked on a deterministic evaluator fuel budget; R2 BLOCKED **N−1 DEGRADED** (⚠ gpt5-6-sol absent) with a genuine new B1 catch from `gemini-3-1-pro` (a `()` generator would have hit the new loud-error default arm) → narrow-refinement carve-out, fix applied verbatim + acceptance row B-4a. Planner opus **refuted 5 controller premises**, incl. my inference that skip sites 331/454 were a live bug (they are unreachable dead code — I verified the refutation) and my 17-file blast-radius claim (really 4 silent-green — **my own later baseline confirmed the planner against me**). Executor codex `gpt-5.6-sol` (first run correctly REFUSED to start: the state validator rejects `estimated_loc == 0`, and M5 is docs-only). Evaluator sonnet **PASS 88/100 r1, zero blocking**; both NBs re-verified first-party and NB-2 was **bigger than filed**. Five mutations controller-run RED, reverted sha256-identical, incl. one the evaluator ran that I had not. Shipped: reachable `TypeApp{"list"}` arm (the `ListType` arm dead since `b9ab84e6f`) · total fail-closed skip taxonomy over all SIX `StatusSkip` sites · `Success()` requires `VacuousSkips == 0` with `--allow-skips` the single opt-out · `--format json` emits only JSON to stdout. Strictness is CLASS-SCOPED — `out_of_contract` stays forgiven, `cross_module_functions_lib.ail` is the discriminator and stays rc 0. ⚠ `runner.go` now **790/800** lines, so Lane B1 must route additions elsewhere. **THE BIGGER FIND IS `#535`**: property generation is wall-clock seeded, so the same file on the same binary gives rc=1, rc=1, rc=0 — pre-existing, violates Principle 4, no `--seed` flag exists, and it retroactively explains why the executor's and my corpus numbers differed (neither was wrong). **LANE B1 STILL OPEN — see the [NEXT] row below**] [world-DEMAND] m-property-generator-coverage Lane A** (was: NEW-DOC needed, quorum required; P2 — **not a v1.0 bar item**, filed in this section beside its sibling World filings for discoverability, NOT because it is a soundness row; Lane A ~0.5d, Lane B ~2–3d) — **`ailang#517`, filed by the Ailang World mission 2026-07-29, live-REPRODUCED at HEAD by iter-118** on a binary built from `5998f4039` (ghost discipline applied — not a ghost). Contract-derived property tests **run zero cases and report `skip`** for any parameter the generator table does not cover, while the suite still reports **`success: true` and exits 0**. This is the mission's own **vacuous-pass class**, third instance in this repo after the silent-`z3` skip and CI `t.Skip` — a check reporting success for work it never performed. **The controller's repro is WIDER than the filing on two counts, both measured, and each changes the fix**: (1) it is **not "ADTs and records"** — `createGeneratorForType` (`internal/testing/runner.go:630`) has exactly two arms, `*ast.SimpleType` in {`int`,`float`,`bool`,`string`} and `*ast.ListType`, so **tuples, ADT-free plain records, AND `list[T]` all skip** (measured in one file: `c: Color` · `r: { a: int, b: string }` · `t: (int, int)` · `xs: list[int]`, all `tests_run=0`); (2) **the list arm is DEAD CODE** — `DX-17 Phase 2` (`b9ab84e6f`, "Normalize [T] syntax to TypeApp at parse time") made the parser emit `*ast.TypeApp` for **both** `[int]` and `list[int]`, and this consumer was never updated, so `NewListGenerator` is unreachable from any real program; `&ast.ListType{}` is now constructed **only in test files** (`internal/types/cycles_test.go`, `internal/gen/golang/adt_test.go`), which is exactly what kept it looking alive. That is the repo's recurring **guard-the-call-site-not-the-helper** shape. **Controller-verified impact bound (do NOT overstate it)**: the guard is **half-present, not absent** — `SuiteResult.Success()` (`internal/testing/result.go:97`) is `ran > 0 && FailedTests == 0`, and an `AllSkipped()` sentinel plus `--allow-skips` already exist with the comment "an all-skipped suite is NOT success". Measured: **1 pass + 1 skip → `success:true`, exit 0**; **all-skipped → `success:false`, exit 1**. So the silent shape requires **≥1 passing test alongside** — which is every real module, and means **a minimal one-function repro exits 1 and will read as already-fixed**. Any regression test MUST use the mixed shape or it passes for the wrong reason. Human mode is softer than it looks too: it prints `⊘` and a `1 skipped` line but headlines **`✓ All tests passed!`** and drops the reason entirely — `no generator for parameter …` lives **only** in JSON `properties[].error`, and `properties[]` is a **separate array from `tests[]`** (which is why the sibling's `len(tests[])` count check was blind). **Lane A — make it loud** (small, lands first): surface partial skips at the exit-code/`success` level (or at minimum on stderr), and fix the dead `TypeApp` list arm — that one is a straight bug fix, not a design question. **Lane B — derive generators structurally** (the doc; product-of-fields for records, sum-over-constructors for ADTs, with recursion depth/size bounds, shrinking, and a user-supplied-generator escape hatch for what cannot be derived — the sibling's ask (3), folded in here because Lane B needs it). **Demand evidence satisfied by construction** — World's CI gate printed `✓ all 14 required named tests pass` while five properties over its core world types ran zero cases. Verdict posted to `ailang#517` and to `ailang-world#9`, with the explicit note that **no date was promised** and the interim assertion that actually holds (`skipped_tests == 0`), so World keeps its local gate fix rather than waiting on us.
- **[M1 LANDED 2026-08-05 (iter-143) — PR #591 → squash `c440a1628`, Gate 3b GREEN SHA-addressed (19 check-runs, 19 completed, all 4 required contexts `test`/`lint`/`build`/`docs-gate` `success`); 2 commits; evaluator sonnet **PASS 92/100 r1, zero blocking**, and the evaluator independently re-verified three mutations, the inert-gate removal and the skip-honesty question rather than taking the controller's word. Shipped `internal/testutil` **357 LOC**: a THREE-state `LiveNetworkStatus()` predicate (skip/fatal/run) deliberately extracted — you cannot assert "`t.Skip` was NOT called" from inside a test that would be skipped, so testing the wrapper is vacuous *by construction*, and testing the predicate makes both directions real · `RequiresLiveNetwork` fail-loud on the poisoned-live-lane mis-combination, which **never unsets** (V29: Go caches proxy env process-wide, so a runtime unset silently does nothing) · `HangGuard`/`HangGuardContext` · `RunBounded`. Package stays stdlib-only (`go list -deps | grep sunholo` = **1**; control: `internal/effects` = 11). All **6** assertions mutation-proven by the controller OUTSIDE the sandbox, negative control green first, each mutation proven landed by byte-diff, both sources reverted sha256-identical. **THE FIND: the executor shipped an inert `testing.Short()` gate INTO THE PACKAGE BUILT TO REPLACE THEM** — `-short` is passed nowhere in this repo, so it is defect class **C4**, and M3's gatelint **R1 (zero exceptions)** would have red-lighted this very sprint two milestones later; measured **7→8** with the doc's own **V2** command, removed, back to **7**. **CI caught a second defect unreproducible locally**: Windows env vars are case-INSENSITIVE, so `http_proxy`/`HTTP_PROXY` are ONE variable and the predicate correctly reports the upper-cased name — a case-sensitive assertion can never match there (production code was right; fixed `f45e8bab1`, still non-vacuous). ⚠ **M5 FOLLOW-UP, measured not guessed**: SonarCloud `new_coverage 67.9% < 80%` (**non-required**; control shows `success` on `97a4ac9d3`, so it is genuinely new with M1). The real gap is `String()` at **0.0%**; `RequiresLiveNetwork` also reads 0.0% but that is a **re-exec artifact** — it executes in the subprocess child, invisible to the parent coverage profile, and the evaluator confirmed that path IS exercised. Sonar therefore **UNDERSTATES** real coverage: do NOT "fix" it by weakening the subprocess pattern. **`D5` IS DECIDED (iter-145, Option A) AND M2/M3/M4 ARE UNBLOCKED — see the sequencing note below**] m-ci-flake-systemic-fix — **M1 LANDED** (iter-143 `c440a1628`) · **M2 LANDED** (iter-145 `368f940cf`, PR #593) · **M3 LANDED** (iter-146 `13c570063`, PR #597 — gatelint R1/R2/R3 + AC8/AC10(a-d), 941 files scanned / 0 violations; allowlist seed RE-MEASURED post-M2 because the plan's predated it: R1 **empty**, R2 **one** entry reusing the reason M2 authored in-file, R3 **five**; `net_test.go`/`gitcache_test.go` are GATED and deliberately NOT allowlisted) → **M4 LANDED** (iter-147 `4b47f8b0a`, PR #599 — poison wired across all **6** legs + AC9 gatelint registration, closing **AC9/AC11/AC12**; evaluator sonnet PASS 88/100 r1 zero blocking; Gate 3b 20 checks / 0 non-success incl. all 4 build legs. `#569` was re-verified BY PURPOSE and merged first as `bc30912ea` to clear the collision. **The first CI run went RED on M4's own AC9 step**: `go mod download all` touches the tracked `go.sum` and the staleness detector compares binary mtime against newest Go source, so prefetching AFTER `Build binaries` made every binary read STALE and silently skipped 3 binary-gated tests — fixed by moving the prefetch BEFORE the build in both legs, as build.yml already did. **`test-windows` green closes the PowerShell guard, which was unverifiable locally.** C2 watch-item re-measured and WEAKENED: the 30s budget wraps only `probeServeAPIMCPTools` (subtests 0.75s/0.03s), so the margin is ~40× not 2.9×) → **M5 LANDED** (iter-148 authored / iter-149 merged: `c9e1a4f98`, PR #600 — changelog, the `Deterministic Test Boundaries` guide section, and the doc↔plan reconciliation; evaluator sonnet PASS 88/100 r1 zero blocking, full AC1–AC12 sweep re-run out-of-sandbox, all pass) — **⚠ THE SPRINT IS COMPLETE; DOC + PLAN MOVED TO `design_docs/implemented/v0_33_1/`** (plans travel with their doc). Closes `#583`/`#494`/`#509`/`#587`/`#561`, each verified in code before closing rather than on the doc's `Closes:` claim. **M5 was written by an UNRECORDED iteration 148** that opened PR #600, went green, then died before Gate 3b — leaving zero charter/log/STATUS trace (`grep -c 'ITERATION 148'` = **0** in both files) and an **OPEN** PR that Gate 2's *merged*-PR search structurally cannot see; the worktree `.wt-iter148-ci-flake` was the corroborating trace. See ITERATION 149's stamp** (`#583`, `#494`, `#509`, `#587`, `#561`; **P0** — flakes red-light `dev`, and a red `dev` outranks this queue every time it occurs. Doc `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` LANDED `dec17dab1`, quorum-cleared over 3 rounds; **sprint plan LANDED `7cb798d98`**: 5 milestones / 1265 LOC / **26h ≈ 4.5d**, revised up from the doc's 3–4d. No new doc and no re-quorum — the iter-142 corrections are *measurements*, not design changes) — **⚠ THIS ROW WAS MISSING UNTIL ITER-142.** Iteration 141 landed the design doc under Mark's directive ("Yes sprint a CI flake fix") and recorded it **only in its STATUS stamp**; a grep of the queue for the item returned **zero** rows. Filed retroactively, and the generalisable point is worth more than the fix: *a doc that lands without a queue row is invisible to every later Gate-2 pick* — only iter-141's own hand-written "Next" line kept it alive. **SEQUENCING IS NOT FREE CHOICE**: (1) **M1** (`internal/testutil` gate + bounded-subprocess helpers, 370 LOC / 5h) is the **only unblocked milestone** — new package, nothing imports it, **zero blast radius**; take it first. (2) ~~**M2 and M4 are BLOCKED on `D5`**~~ → **`D5` DECIDED 2026-08-05 (iter-145) = OPTION A, by Mark** (verbatim: *"D5 - option A and then queue the B afterwards. I'm cool with 2."*). AC3 was **vacuous** because the poisoned-proxy boundary does not cover AILANG's own `Net` effect (6 hand-built `http.Transport{}` in `internal/effects`, none setting `Proxy`; `ProxyFromEnvironment` in **0** first-party files — **V33**). **Applied:** AC3 → **AC3′(a/b/c)** narrowed to `./internal/pkg/`, the `internal/effects` egress tests move behind `RequiresLiveNetwork`, and the residual is asserted *as open* by the new **AC10(d)** — which carries a `Proxy: http.ProxyFromEnvironment` arm as its known-positive control and will go RED if Option B ever lands, making it the tripwire that retires itself. **Option (B) is queued separately** (see the Option-B row) — it must not ride in on a sprint scoped and reviewed as test-only. (3) **M3 depends on M2** (`TestGateLint_Repo` scans the real tree), so M2 goes first; **M5** is docs-only. (4) **M4 is the ONLY CI-touching commit** — it wires the poison across **6** legs (**V34**: `build.yml`'s matrix has 4 jobs, not 3 — `macos-latest` appears twice, amd64 + arm64 — plus ci.yml `test` and `test-windows`); watch the first `dev` run after it lands, with `git revert --no-edit <M4-sha>` staged. ⚠ **COLLISION — DECISION TAKEN iter-145, plan §6.1 option (b) not the recommended (a):** PR **`#532`** rewrites `buildAilang` in `cmd/ailang/main_test.go` and the body under the `testing.Short()` gate in `serve_api_mcp_surface_test.go`, and touches `ci.yml` — **exactly M2's surface**. **M2 goes first; #532 rebases onto it after.** Measured basis: #532 is authored by `sunholo-voight-kampff` (*this loop's own PR* — no external author to coordinate with), and it has been `CONFLICTING`/`DIRTY` against `dev` since **2026-07-29**, untouched for 7 days — so **M2 does not make it any more conflicted than it already is**; the resolve cost is pre-existing debt, not one M2 creates. Re-application is symmetric whichever side goes second, so ordering was chosen on unblocking value. A comment recording this is posted on #532 so its rebase re-applies `HangGuardContext` to the new `sync.Once` body. `#569` (dependabot actions bump) touches `ci.yml` + `build.yml` = M4's surface, re-check before M4. ✅ **RESOLVED iter-145:** the iter-141 narrow-refinement carve-out (R3 quorum fix applied with no re-quorum on that fix) is **RATIFIED by Mark** — *"ACCEPTED as-is. No re-quorum needed"* — **veto window closed**.
- **[NEXT — SPRINT-PLANNED, READY FOR M1 2026-08-06 (iter-155): UNPARKED by Mark's `D-6 = (A)`; BOTH quorum objections resolved with NO third round; plan + sprint JSON landed (`m-net-effect-proxy-boundary-sprint-plan.md`, 636 lines, 4 milestones / 12 ACs each owned exactly once, 3-day shape kept; `sprint_M-NET-EFFECT-PROXY-BOUNDARY.json`). Commits `945f36727` (doc revision: rows **V20**/**V21**, D-6 recorded) + `7c7e5e58a` (plan) — DOC-ONLY and **UNPUSHED pending the Actions outage**. `gemini-3-1-pro`'s unverified-premise objection was satisfied **by measurement, not a revision round** (row V20 names the two call sites the error-mapping must update: `net.go:567` preflight, `:631` post-`client.Do`); `gpt5-6-sol`'s AST-analyzer objection was resolved by the human ruling, with the analyzer **FILED AS `#612`** at resume time — Option A is "cheap gate now AND the durable gate filed", so that filing is part of its definition of done. Planner opus, lane `opus fail-closed:env-pin` used verbatim. Planner's baseline sweep (rule 3e) found `go build ./...` **already rc=1 on unmodified dev** (excluded, scoped build substituted) and all three M1/M2 named-test gates **rc=0 with 0 `=== RUN` lines at base** — every named-test AC now asserts a `=== RUN` count, not just an exit code. ⚠ **`D-1` STILL OWED A HUMAN RATIFICATION** (this design knowingly trades target-IP SSRF pinning on PROXIED routes; preserved on direct/`NO_PROXY`, and the doc never claims equivalence). Was: PARKED needs-human-review 2026-08-06 (iter-150) — DESIGN DOC LANDED (`design_docs/planned/v0_33_1/m-net-effect-proxy-boundary.md`, 662 lines incl. its quorum log), QUORUM BLOCKED ×2, ONE DECISION OWED. Designer codex `gpt-5.6-sol` (rotation advanced). R1 blocked on a REAL defect: `gemini-3-1-pro` caught target-IP resolution specified in TWO places (preflight `resolveAndValidateIP` AND the new RoundTripper) = a TOCTOU DNS-rebinding race, plus a broken-proxied-request risk on hosts without external DNS; fixed by making the direct RoundTripper the sole resolve-validate-dial site and skipping local target DNS entirely on proxy routes (V17/V18/V19 added, 520→592 lines). R2's `gemini-3-1-pro` objection is CARVE-OUT-ELIGIBLE and its answer is ALREADY MEASURED, so the resume is cheap: `E_NET_IP_BLOCKED` at `net_security.go:27,34,46,51,56`, `E_NET_DNS_FAILED` at `:90,94`, surfaced via `makeResultErr("Transport", …)` at `net.go:551,556,567,605,631,639` (control 11 sites; `:567` is the preflight path being moved, `:631` the post-`client.Do` path where a `url.Error` arrives). **R2's `gpt5-6-sol` objection is what parks it**: replace V2/V17 AND the M4 completeness gate with a checked-in `go/packages` AST/type analyzer plus positive fixtures, because textual matching cannot see aliased imports, `new(http.Transport)`, post-construction `Client.Transport =`, transport-returning factories, or custom `RoundTripper`s. That materially expands scope and needs a judgment call → Standing rule 2: park, do not force through. **The decision is cheap to answer because I tested the reviewer's own hypothesis: ALL FIVE shapes are ZERO at HEAD**, each with a firing control (aliased imports 0 / control 1505; `new(http.Transport)` 0 / control 4; `.Transport =` 0 / control 8; transport factories 0 / control 2; `RoundTrip(` 0). The seven-site claim is therefore empirically COMPLETE today; the live argument is only about the gate's DURABILITY against future escapes. ⚠ Two of my five controls failed on first run (the alias matcher matched the `import` keyword itself; the `new(` control returned 0, which rule 3a makes uninformative) and were re-run before any number was used. **D-6 ANSWERED 2026-08-06 (Mark, attended): (A)** — grep gate now, sprint stays 3d; the AST analyzer is to be FILED AS A SEPARATE FOLLOW-UP row when this sprint resumes (that filing is part of option A's definition of done). Option (B) declined. Row is UNPARKED — resume via the pre-measured R2 carve-out above. Non-blocking but worth ratifying alongside: **D-1 knowingly trades target-IP SSRF pinning on PROXIED requests** (preserved on direct/`NO_PROXY`; the doc is explicit and never claims equivalence). Was:] **[NEXT] [NEW-DOC] m-net-effect-proxy-boundary — `D5` OPTION B, queued by Mark's directive 2026-08-05** (*"queue the B afterwards"*; **P2**, sequenced AFTER the CI-flake sprint completes — it is the better end state, but it is a **production runtime change** and must not ride in on a sprint scoped, reviewed and quorum-cleared as test-only). **The work:** set `Proxy: http.ProxyFromEnvironment` on the proxy-ignoring hand-built `http.Transport{}` literals, bringing AILANG's own `Net` effect inside the poisoned-proxy egress boundary. ⚠ **SCOPE CORRECTED 2026-08-06 (iter-149) — it is 7 literals across 4 files, NOT the 6-across-3 this row used to claim.** Re-measured first-party with controls (the matcher sees **11** including tests; `ProxyFromEnvironment` appears only in M3's deliberate control arm, so V33's production-zero still holds): `internal/effects/net.go:96,212,587` · `internal/effects/stream_ndjson.go:80` · `internal/effects/stream_sse.go:70,329` (= 6 across 3 files, the old count) **PLUS `internal/executor/managed_agents/client.go:141`**, which sets only timeouts and so bypasses the poison identically — but lives OUTSIDE `internal/effects`. Surfaced by iteration 148's M5 sweep and re-derived rather than inherited. **The design pass must decide whether the `managed_agents` client is in scope at all** — it is not part of the `Net` effect, so "bring `Net` inside the boundary" and "close the first-party residual" are two different jobs, and AC10(d) asserts the *mechanism*, which is file-independent. **Why it needs its own design pass + quorum, not a queue row's worth of thought:** a proxy resolves the hostname *itself*, which is exactly what `net.go`'s **pinned-IP SSRF guard** exists to prevent — so this plausibly breaks a security control, and the interaction is the whole design question. It is also a behaviour change for every AILANG program that uses `Net`, in a repo whose CI-flake doc explicitly disclaims runtime changes. **⚠ CORRECTED 2026-08-06 (iter-155) — THE "ACCEPTANCE SIGNAL ALREADY BUILT AND WAITING" WAS FALSE, AND THIS ROW ASSERTED IT FOR THREE ITERATIONS.** The row used to read: the CI-flake sprint's **AC10(d)** "measures this residual as OPEN and will go **RED** when Option B lands — so the tripwire that tells this item it succeeded already exists". **It cannot go red.** `testEffectsProxyResidual` (`internal/testutil/egress_posture_test.go:66-85`) builds its **OWN** `&http.Transport{}` at `:74` and exercises that, while its comment claims it trips when `internal/effects` gains `ProxyFromEnvironment`. Measured with the tool that cannot miss: `go list -f '{{join .TestImports}}' ./internal/testutil` = **12 imports, ALL stdlib, ZERO ailang packages** (control: `./internal/effects` = **6** ailang imports, so the instrument sees them when they exist). No production change can alter its outcome — a tripwire watching a local replica of its own subject, shipped through a full sprint, a quorum and an evaluator PASS. Surfaced by the iter-155 sprint-planner refuting the controller's own briefing, then confirmed first-party. **The design doc was right all along** ("**helper-only** residual logic"), so **M4 is a deliberate DELETION, not an observed red**, and no AC depends on watching that tripwire flip; retiring it + the matching Non-Goals text stays part of this item's definition of done. The old sequencing constraint ("do not start before M2–M5 land") is **satisfied** — M1–M5 all landed (`c440a1628`…`c9e1a4f98`) — and was in any case predicated on the same false coupling. Sibling defect, same root cause: `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix-sprint-plan.md:460` describes an AC10(d) falsification drill that is not executable.
- **[CLOSED 2026-08-06 (iter-152)] [SWEEP-BATCH iter-145] Three zero-mention open issues found by the weekly external-issue sweep** — **ROW RESOLVED, all three dispositioned.** `#588` closed at iter-150. **`#590` FIXED** — PR #605 → squash `34951811d`, Gate 3b GREEN SHA-addressed (20 checks, 0 pending, all four REQUIRED contexts `test`/`lint`/`build`/`docs-gate` **success**; the single non-success is SonarCloud, **non-required**: 78.7% new-code coverage vs 80% and 8.0% duplication vs 3%, the same class as iter-126, analysed revision == PR head so it is a genuine reading and not the stale-Sonar trap — the duplication is the 7 near-identical `.ail` fixtures and is worth revisiting, not hiding). Reproduced 100% at HEAD before routing; **the find is wider than the filing** — `assert` is parseable in exactly ONE construct (`registerPrefix(lexer.ASSERT)` = **0**, control **26**) and that construct was 100% broken, so the keyword had **no working call site anywhere in the language**. Lowered in the **fold**, not the printer, because the printer cannot see sequencing (a per-node lowering would make `{ assert false; assert true }` PASS). Evaluator sonnet **PASS 88/100 r1, zero blocking**, 7 mutations killed + an 8th controller mutation (5 tests red, reverted byte-identical). **`#589` NOT REPRODUCED, deliberately left OPEN** — 0/10 on the closest available shape with a firing control, but this repo's widest multi-inline-test file has **1** import against the report's 15-module closure, so the negative is equally consistent with "fixed" and "the instrument cannot express the trigger". That is an unmet repro burden, not a refutation; verdict posted naming what would move it (the reporter's module, or a synthetic wide-closure module run in a loop). Was:] (Gate 0.5; measured 2026-08-05 against 45 open issues, control `#498` = 8 charter mentions so the instrument sees positives). None outranks the queue — a sweep never does by itself. (1) ~~**`#588`**~~ **DONE — closed 2026-08-05 citing the M2 squash; iter-150 confirmed the close had already happened (my close attempt was a no-op and posted no comment).** ⚠ Two corrections to how this row described it, both measured first-party rather than inherited: M2 gated the live-network **subtest** (`internal/effects/net_test.go:364`), **not** the whole `TestNetHttpPost` function — whose remaining subtests are pure `E_NET_TYPE_ERROR` type-checking and are correctly left ungated (control: the same gate appears at `net_test.go:441` and `internal/pkg/gitcache_test.go:51`, so the ungated outer function is a measurement). Practical effect for the issue is unchanged. (2) **`#589`** *[motoko_agent] `ailang test`: cluster harness nondeterministically fails a passing test (6/10) with 'record has no field' naming a field outside the call's dependency closure* — a genuine language/runtime bug from a real downstream consumer, and **nondeterministic**, which makes it a soundness smell rather than a cosmetic one; needs ghost-discipline live-repro at HEAD before it earns a design doc. (3) **`#590`** *[motoko_agent] Named test blocks with `assert` always fail: `EvaluateNamedTestBodyExprs` round-trips `AssertStmt` through the general parser, which doesn't accept it* — reads as a small, well-localised parser/eval defect with the mechanism already named by the reporter; likely a same-day AILANG-fix lane item, **not** a sprint. ⚠ Both `#589`/`#590` are `from:motoko_agent` reports, so the standing rule applies: **verify against HEAD before filing** — motoko_agent reports pin a stale `ailang_version` and have twice been already-fixed or superseded.
- **[NEXT — HALF DONE] [FINDINGS-BATCH iter-151] Two CI/security findings measured during M3 verification, batched into one row.** ⚠ **`#603` FIXED at iteration 153 (PR #606, 2 commits; evaluator sonnet PASS 91/100 r1, zero blocking) — NOT yet marked LANDED, because a declared GitHub Actions major outage meant Gate 3b saw 0 failures but never a full green, and a timed-out wait is not green. Resume = confirm #606 green and merge.** The triage verdict: CodeQL's taint trace is CORRECT (2 of 12 hostile shapes reflect a literal `<script>`) but NOT exploitable (Content-Type present 12/12, reflecting paths carry `nosniff`, json escapes) — fixed rather than dismissed because all three guards were inherited from `go-sdk` and asserted nowhere locally. Two mutations proved the fix; a THIRD showed the Content-Type default was unreachable through the public path and therefore not a guard at all until an injected-transport test reached it. `writeMCPEnvelope` was labelled to match. ⚠ **`#602` FIXED at iteration 154 (PR #608, 2 commits; evaluator sonnet PASS 88/100 r1, zero blocking) — NOT marked LANDED: the SAME outage was still live, and `#608` had ZERO workflow runs created at all, so Gate 3b never even had an instrument. Resume = confirm #606 AND #608 green and merge both.** It was picked precisely because its investigation needs no landing gate. **It did NOT reproduce** — two full-suite arms rc=0, the first reported WITH its limitation (86 of 108 packages cached, only 22 ran), the second fully cold (108 ran, 0 cached). It was confirmed instead by measuring the race directly, and that measurement refuted BOTH halves of the issue: the stated cause is off by 3× (the budget is `max(Timeout,effective)+solverKillGrace` = **3s**, not 1s — corroborated by the isolated arm's 3.274s), and the suggested fix (poll for the pidfile) **cannot work**, because a lost race means the shell was killed before `echo $! > file` ever ran so the file never appears. 500-trial probe: idle 200/200 mean 209ms/max 232ms, 0 over; under a full `-count=1` suite, **1 trial in 300 exceeded 3s at 3.435s** = ~0.3%/spawn, ~0.7%/suite run — which is exactly why CI stays green and why iter-151 saw it once. The over-budget trial landed at load **13.6**, not the higher **24.4**: a tail effect, not a smooth function of load. Fix = treat a trial that never recorded a child PID as **inconclusive** and retry, bounded at 3 with a loud `Fatalf`. Two controller mutations, sha256-proven and reverted byte-identical: killing `pid` instead of `-pid` still reds on **every** attempt (so the retry cannot mask a real regression), and forcing the race permanently lost reds with the exhaustion message rather than passing silently. The evaluator's NON-BLOCKING finding was **acted on and was a genuine scope error of mine**: `Z3Version` uses `versionProbeTimeout` (**5s**) directly, so the two tests do NOT share a budget and 0 of 500 trials exceeded 5s. The weekly-CodeQL-cadence question was `D-7`, ANSWERED 2026-08-06 (Mark, attended): KEEP WEEKLY — no cadence change. The RELEASE ask (iter-151) was answered the same session: WAIT, World does not need a pinned release yet. (neither outranks the queue — a finding never does by itself; both were established with negative controls, not assumed). **(1) `#602`** — `internal/smt TestSolve_HardTimeout_FakeSolverIgnoringT` reds `go test ./...` on **clean dev**: a load-sensitive 3s deadline for a fake solver's child to write its pidfile, which parallel load can beat. Controlled both ways (full suite → rc=1 on unmodified dev with no concurrent load; isolation → rc=0 in both trees), so it is pre-existing and in the TEST, not the product. **A survivor of the CI-flake sprint that just closed**, green in CI only because the runners' load profile stays on the winning side — i.e. a latent CI flake, and the natural home for the fix is `internal/testutil`'s bounded helpers from `#591`. ~2-3h. **(2) `#603`** — CodeQL `go/reflected-xss` **high** at `embedded_mcp.go:119`, which is **M2's** code and byte-identical between dev and the PR head. It surfaced on PR #601 only because CodeQL analyses `dev` **weekly** and last ran ~22h BEFORE M2 landed, so that file had never been scanned. Two decisions owed: is it a true positive (the replayed `Content-Type` comes from the SDK, which would make HTML interpretation unlikely — but the headers are copied wholesale from a buffer, so confirm rather than assume), and should CodeQL run on push to `dev` so a finding is attributed to the change that caused it instead of appearing as noise on an unrelated PR. ~2-4h incl. the cadence change. **Method note worth keeping**: my first read said "0 reflected-xss on dev" from a `per_page=50` page against ≥100 open alerts — rule 3b(v)(a) truncation, caught by paginating. An enumeration you truncated is not an enumeration.
- **[NEXT] [iter-152] `#604` — named test blocks check only the LAST expression; earlier failing checks are discarded** (P2, ~0.5-1d; **not** a v1.0 bar item, but it is the **vacuous-pass class** this mission has now closed four times elsewhere — the silent `z3` skip, CI `t.Skip`, `#517`, `#524`). Found while planning the `#590` fix and **reproduced first-party with a discriminating control before filing**: `{ add_one(1) == 99; add_one(1) == 2 }` reports **`All tests passed!` rc=0**, while the reverse order correctly fails — so it is position, not luck. `FoldBodyExprs` binds every non-final expression to a dead `_seq` and `EvaluateNamedTestBodyExprs` returns only the final value, which `runner.go:156-158` documents as intentional. Defensible for an effectful sequence; wrong for a **pure** test body, where a discarded `bool` is either a swallowed check or dead code. ⚠ **Deliberately scoped OUT of the `#590` sprint** and verified unchanged by it — fixing it alters the type obligation on currently-passing tests, so it must not ride in on a bug-fix-lane commit. The `#590` lowering already short-circuits correctly *within* assert-bearing bodies, so this row covers only the non-assert path. Design question to settle first: require every non-final expression to be `()`-typed (making a discarded `bool` a type error), or conjoin all `bool`-valued expressions so every check counts.
- **[NEXT] [ORPHAN-PR iter-150] `#545` — "fix(eval): agent cost was the wrong model's price, and the budget was the wrong unit"** (P2; rebase-and-revalidate, ~0.5–1d; **not** a v1.0 bar item). Found by iteration 149's new died-mid-flight check on its **first independent use** — the check looks for OPEN PRs authored by this loop, and this is one nobody ever picked up. Opened **2026-07-31**, last touched the same day, **CONFLICTING**, **125 commits behind** dev, with **zero** mentions in the charter (control `#532` = 2) and **zero** in the mission log (control `#544` = 4) — invisible to every existing surface, because the weekly sweep covers *issues* and the already-landed check covers *merged* PRs. **Its purpose is NOT superseded — measured at HEAD, not assumed** (the standing rule treats OPEN + long-untouched as evidence *toward* superseded, so this had to be checked): `ResolveCostModel` **0** hits, `CostProvenance`/`cost_provenance` **0**, `internal/eval_harness/cost_tally.go` and `internal/executor/codex/cost.go` **absent**, with `internal/executor/cost.go` present as the control proving the absences are measurements. So both defects are still live: agent cost banked from the executor's hardcoded table rather than per-model rates (the PR documents banked `$0.34259` vs a budget that saw `$0.26980` — two different price tables, so the kill threshold and the banked number disagreed), and `cost_usd` summing subscription list-price-equivalents together with genuinely metered spend under one label. That second one is **KPI provenance**, which this mission's own cost reporting depends on. ⚠ 47 files across `eval_harness`, `executor/*`, `observatory`, `storage/firestore` — a surface that moved a lot in those 125 commits (v0.32.0 confidence-gating, v0.33.0 recorded-stream), so this is a rebase-and-revalidate job, **not** a merge. Triage comment posted on the PR. Decide on pick-up: rebase it, or close it and re-cut the fix from the two defects it documents.
- **[NEXT] [world-DEMAND] m-property-generator-coverage LANE B1** (design doc EXISTS + quorum-cleared for B1 — no new doc, no re-quorum needed; P2 — not a v1.0 bar item; ~1.5–2d) — structural generator derivation: product-of-fields for records, sum-over-constructors for ADTs, tuples, with recursion depth/size bounds and shrinking. This is the REAL fix for the shapes Lane A can only make LOUD: after Lane A, 6 in-repo contract files still carry vacuous skips (records, ADTs, `string<email>`, `list[Tree]`), and World's five `w-mcp-projection` properties over its core types still run zero cases. **Two hard constraints from iter-121, both measured**: (1) `runner.go` is at **790/800** lines, so B1's additions MUST land in a new file in the same package — the CI gate has 10 lines of headroom (evaluator NB-3); (2) the `valueToLiteral` arm list MUST add `UnitValue → ast.Literal{Kind: ast.UnitLit}` BEFORE the default arm becomes a loud error, else the `()` generator fails the harness (quorum R2, `gemini-3-1-pro`, now recorded in the doc with acceptance row B-4a). **B2 (user-supplied `gen<TypeName>` escape hatch) stays DEFERRED** — blocked on a deterministic evaluator fuel/step budget, per `gpt5-6-sol`'s own proposed option; do not fold it back in without that budget. Expect Lane B1 to surface GENUINE contract violations as previously-vacuous properties start running (Lane A already exposed two in `list_verify.ail`) — budget triage time rather than treating them as regressions. ⚠ **Sequencing: `#535` (wall-clock seeding) should land FIRST or alongside**, because B1 multiplies the number of properties actually executing and every one of them inherits a non-reproducible verdict. ⚠ **UPDATE 2026-07-31 (iter-126): B1's `runner.go` 790/800 constraint is RELIEVED — it is now 670**, because M1 below moved `runEnsuresProperty` into a new `internal/testing/contract_domain.go`. B1 additions still belong in a new file, but the gate is no longer 10 lines from the cap.
- **[LANDED-M1 2026-07-31 (iter-126) — PR #549 → squash `a9e26ffd6` (commits `940d1108e` + `3ebd4d19a`); doc `01c36db8d`, plan `7df443e25`. **Gate 3b GREEN SHA-addressed**: all FOUR required checks pass (`test`/`lint`/`build`/`docs-gate`), 13 success + 5 skipped/N-A, 0 required failures. ⚠ Non-required **SonarCloud RED**, deliberately not hidden: 77.9% coverage on new code (gate ≥80%) and 4.6% duplication (gate ≤3%). Required contexts on dev are exactly `["test","lint","build","docs-gate"]` (verified via branch protection), so UNSTABLE ≠ BLOCKED — but the duplication is the price of splitting the ensures path out of `runner.go` and should be revisited in M2. Designer `codex:gpt-5.6-sol` (rotation advanced from claude), planner opus, executor `codex:gpt-5.6-sol`, evaluator sonnet **PASS 95/100 r1, zero blocking** (8 independent mutations, 7 killed). M2/M3 PARKED to the 2026-08-03 re-arm. ⚠ **UPDATE 2026-08-01 (iter-127): M2 IS NO LONGER BLOCKED.** The AC9 fixture parse failure that §5.3 required be resolved before M2 starts was a symptom of `#548` (declaration-aware strip, landed `f64659b12`), NOT of the contract form. AC9's ORIGINAL module-less fixture now passes unmodified — paired control: pre-fix rc=1 `PAR_NO_PREFIX_PARSE ... ensures`, post-fix rc=0 — so **keep AC9 as written**; no substitute fixture and no restatement over a module-bearing file are needed. The plan's §0.3/§5.3 and risk-row B3 are updated in place, and `internal/testing/testdata/strip/moduleless_contract.ail` is the committed regression guard.]** m-property-test-trust (`#535` + **`#547`**, doc → [planned/v0_31_0/m-property-seed-determinism.md](planned/v0_31_0/m-property-seed-determinism.md) + its `-sprint-plan.md`; P0 prerequisite for LANE B1 above) — picked because `#546` is PARKED on Mark's a/b/c call and both quota offloads are date-gated to 08-03, leaving LANE B1 as the live queue head; its own row names `#535` as the thing that must land first. **The pick was `#535`; the FIND was `#547`.** Reality-checking `#535` (reproduced 5/5, exit codes 1,0,0,1,1 on unchanged input) surfaced a larger, previously unknown defect: `runEnsuresProperty` evaluated a function's postcondition **without ever evaluating its `requires` precondition**, so it reported `ensures violated` for inputs the contract EXCLUDES — a **vacuous FAILURE**, mirror of the vacuous-pass class `#517` Lane A closed. Decisive argument = the **asymmetry**: `runRequiresProperty` meets the identical condition and reports `skip`, its own comment saying such inputs "aren't a function bug". Minimal repro 6/6: `requires { x > 100 }`, body `x > 100`, every reported counterexample ≤ 100. This **corrected the read of `list_recursive_verify.ail`** — its ~50% failure is a FALSE POSITIVE, not a genuine violation (the designer reached that independently before being told). **M1 shipped the discard filter** (100 accepted / 1000 attempts; cap exhaustion → `skip:out_of_contract` + `unverified:`, never pass never fail); `#535` stays OPEN by design as M2. **Quorum BLOCKED ×2, direction never contested**, resolved under the narrow-refinement carve-out with both reviewers' verbatim fixes; **both rounds blocked on the SAME root cause — a codebase premise asserted rather than measured** (R1 metadata enumeration; R2 `TestConfig.WorkspaceRoot`, which the controller measured as NOT EXISTING). The controller's own check of R1 came back **better than the objection assumed and SHRANK scope**: repeated `requires` blocks are impossible by construction (`PAR_DUPLICATE_REQUIRES`). **Planner refuted 3 doc premises**, incl. that M1's ACs were written against a `--seed 42` flag M1 never adds — **none of them could have run**. **Two further defects filed**: `#547` and `#548` (named test + contract in one file breaks the named-test path via orphaned `requires`/`ensures` in the generated temp module — 0 in-repo instances, a latent user trap). ⚠ **Behaviour change**: `list_recursive_verify.ail` goes flaky `1,0,0,1,1` → stable `0 pass / 0 fail / 6 skip`; `extractBounded` is now honestly *unverified* rather than luckily passing. **No CI impact — `make verify-examples` runs `ailang run`, never `ailang test`** (controller-verified, and it is why the seed pin is low-risk). **A surviving mutation is recorded, not buried**: relaxing the guard to accept 99 as a pass SURVIVED; the test claiming to pin that boundary was renamed `TestEnsuresNinetyNineAcceptedIsNotAPass` → `TestEnsuresSparseDomainIsSkipNotPass`, because acceptance at `x > 900` is ~5% so `TestsRun` never reaches 99 — pinning it genuinely requires `#535`/M2. The evaluator's NON-BLOCKING F1 was **acted on anyway** (a severity label is an opinion, not a measurement): the negative control only rejected negatives and `0`, so `x=50` would have passed an assertion whose own message claimed to prove `x > 100`; strengthened, and **proven to catch** the evaluator's mutation 8 (`x=-612`) which it previously missed.
- **[LANDED 2026-08-01 (iter-127) — PR #550 → squash `f64659b12`. **Gate 3b GREEN SHA-addressed**: 20 checks, 0 pending, **0 failures**; all four REQUIRED contexts (`test`/`lint`/`build`/`docs-gate`) success and SonarCloud green (unlike iter-126's red). Planner opus, executor `codex:gpt-5.6-sol`, evaluator sonnet **PASS 81/100 r1, zero blocking**; designer NOT fired (bug-fix lane, no new doc, no quorum — the `#524` precedent), so the rotation did NOT advance. `metered=$0.00`.]** m-strip-decl-aware (`#548`, CLOSED; plan → [planned/v0_31_0/m-strip-contract-awareness-sprint-plan.md](planned/v0_31_0/m-strip-contract-awareness-sprint-plan.md); P1 — unblocks the seed sprint's M2 above) — filed by iter-126, never routed. `stripNonPureFunctions` deleted exactly ONE LINE of a declaration that may span many, and treated any function not written with the `pure` keyword as effectful, so the temp modules `ailang test` generates were corrupted and the error was reported against a `_namedtest_body_*.ail` the user never wrote. **The controller's first diagnosis was too narrow and the opus planner REFUTED it** (re-verified first-party): the defect is **not contract-specific** — a plain multi-line function with no contract at all corrupts identically (`unexpected token: }`), and `@verify` annotations sit ABOVE `Span.Start` so even a span-only strip misses them. It also refuted `#548`'s own known-positive control as passing for the wrong reason (its test body calls nothing). **The obvious fix was rejected on MEASURED blast radius**: deriving purity from the effect annotation would flip a flag that `internal/format/decl.go:57` uses to emit source, making **`ailang fmt` insert a `pure` keyword the author never typed** — so inference stayed LOCAL to `internal/testing`. Shipped one unified fix across all THREE call sites (Principle 3) in a new `internal/testing/source_strip.go`; `executor.go` 739 → **654**. **A surviving mutation is recorded, not buried**: the evaluator found the `endLine < startLine` disjunct untested; the controller reproduced the survival, made the fallback test a table over both disjuncts, and **proved** the new subtest kills it. All three non-blocking findings acted on. Zero seed work — `#535` stays OPEN.]

- **[LANDED 2026-08-01 (iter-128) — FILED AND FIXED IN ONE ITERATION. PR #552 → squash `9c2081b05`; Gate 3b GREEN SHA-addressed (20 checks, 0 pending, 0 failures, all four REQUIRED contexts + SonarCloud). Planner opus, executor codex `gpt-5.6-sol`, evaluator sonnet **PASS 91/100 r1, zero blocking** with **6 independent mutations all killed** and the threshold margins re-derived first-party from both the live `/tmp` dirs and the new in-repo fixtures (identical → fixtures faithful). Not a queue pick: a Gate-0.4 measurement-validity regression, which outranks the queue.]** m-nightly-unmeasured-category-gate (`#551`; no new doc, no quorum — a bug fix inside `tools/nightly_classify.py`, the `#524`/`#548` lane) — the run-validity gate `#524` built counted only `INFRA_CATEGORIES = {api_error, timeout, executor_error}`, so the 2026-08-01 nightly, in which **12/12 benchmarks failed with `non_agentic`**, scored `tainted = 0/12` and passed as **VALID**: it entered the trend and emitted 11 verdicts (5 SUSPECTED-FLAKE), which is verbatim the second-order harm `#524` exists to prevent. Measured: `non_agentic` was **0 across all 336 rows on the eight prior nights**, then 12/12; pass rate 56/84 → 1/24; every trial `duration_ms=0` with `executor "opencode" produced non-agentic result: 1 turns, 0 tool calls` — the tool-delivery branch `error_categorizer.go:5-9` names itself, i.e. infra, not capability. **Containment ran BEFORE the sprint** (08-01 marked invalid via the tool's own `--mark-invalid`, 348 lines before and after). **Three controller premises REFUTED by the opus planner and re-verified first-party**, one of which was already public in `#551` and required a correction comment: a category-agnostic concentration gate at 0.30 is unsafe (`thrash_aborted` = **13/42 = 0.31 on 07-28 AND 07-31, both good nights**); an all-trials `duration_ms==0` rule does not fire on its own incident (**22/24**, and zero-duration is 17–21% of a *healthy* night); and the `validity_backstop_test.go` objection was mis-aimed (it governs Go's per-row backstop). **Shipped**: `INFRA_CATEGORIES` untouched for per-benchmark taint + a new `RUN_UNMEASURED_CATEGORIES` consulted **only** by `run_validity()` — widening `INFRA_CATEGORIES` instead would have permanently silenced every INDIVIDUAL non-agentic benchmark, since that set has two callers. Threshold **unchanged at 0.30**. Suite 74 → **88**, CI floor 70 → **84**. ⚠ **Time-critical side effect, done first**: the `/tmp` corpora are being reaped (07-24…07-27 trial data already gone), so the five surviving nights are now frozen as in-repo fixtures. ⚠ **STILL OPEN, PARKED FOR HUMAN**: the opencode/**qwen3.5** lane itself is broken and will keep producing unmeasurable nights — `opencode-qwen3-6` and `pi-qwen3-6` passed 4/4 on the same rig the same night, so the fault looks lane-specific. The gate now marks those nights INVALID instead of trusting them, but it cannot make the lane produce data.
- **[LANDED 2026-07-29 (iter-119) — PR #526; M1–M4 + an evaluator BLOCK-1 fix, three commits. Executor codex `gpt-5.6-sol` (two bounded runs), planner opus, evaluator sonnet PASS 86/100 r1. Decisive evidence is a replay against the REAL `/tmp/nightly_eval_20260729_rag_on/agent`: old code emits 7 filings incl. 4 `REGRESSION` = exactly `#520`–`#523`, new code emits one `INVALID` line and zero verdicts. Suite 40→60; CI anti-skip floor 20→55 (it had drifted to guard a 40-test suite at half strength). **#524 rejected in part**: the pass-rate-deviation disjunct was dropped — it cannot distinguish "we failed to measure" from "the subject genuinely broke", so it would silence the detector on a real 40/42-benchmark regression. **Deployment step owed**: `--mark-invalid 2026-07-29` on live history (with a `cp` backup) before the 2026-07-30 05:00 nightly]** m-nightly-run-validity-gate (`#524`; was ~0.5d, actually 9h, no new doc needed — this is a bug fix inside `tools/nightly_classify.py`, sized like `m-nightly-flake-guard` itself) — **found by iter-118 while triaging the 2026-07-29 nightly, which filed FOUR regression issues (`#520`–`#523`) that were all noise**, during a total serving failure of `opencode-qwen3-5-35b-a3b-mxfp8`. Measured: **42 of 42 benchmarks hit `api_error`** (baseline 1–2/night across the prior five), and suite passes fell `52/54/61/65/54` → **14/84**. The detector filed anyway. **Two distinct defects.** (1) **The infra filter is defeated by MIXED trials** — `nightly_classify.py:98-102` excludes a benchmark only when `set(cats) - INFRA_CATEGORIES` is *empty*, and all four filed rows had one trial dying `api_error` while the other survived to a `compile_error`/`thrash_aborted`, so a single non-infra category passes the filter. With `trials: 2` a *partial* outage produces mixed pairs routinely, so the filter guards the case that never happens (clean total outage) and not the one that does. (2) **There is no run-level validity gate at all** — nothing notices 100% infra-tainted benchmarks plus a ~75% pass-total collapse. **Second-order harm, and the reason this is [NEXT] rather than P3**: the invalid run is now IN the history file, so it becomes the baseline the next nights are compared against — tomorrow's genuine results will read as a dramatic *improvement*, and `m-nightly-flake-guard`'s trailing-window solidity check is computed over polluted data. Fix shape: run-level gate FIRST (infra-fraction threshold or trailing-window deviation → mark `INVALID`, file nothing, notify once, and keep it out of the trend or flag it excluded), plus treat a benchmark as infra-tainted if **any** trial hit an infra category rather than all (a benchmark whose only clean signal is one trial has `n=1`, which the variance guard already calls insufficient). This is the same measurement-validity contract as **M-EVAL-MEASUREMENT-CONTRACT** (`970d90e29`, "invalid rows never enter a trend"), which the nightly detector simply never consults. **Ruled out at triage**: fallout from `5998f4039` making `ai-check` exit non-zero on `verify.errors > 0` — it landed hours before this nightly and `contract_roman_numeral` made it the obvious suspect, but an exit-code change cannot produce `api_error` on all 42 benchmarks, including ones with no contracts at all.
- **[LANDED 2026-07-30 (iter-122) — PR #542 → squash `df9466c0d`, Gate 3b GREEN SHA-addressed on the PR (20/20 checks: 14 success, 6 skipped/N-A, **0 failures**); 4 commits. `#538` auto-closed via `Fixes` in the PR body. Evaluator sonnet **PASS 90/100 r1, zero blocking**, and it ran three mutations the controller had not — all RED. Shipped: escalation emits a distinct `SUSTAINED-FAILURE` terminal label (still `--type bug`, honest title/body, **no Discord ping** — the ping means "something broke tonight", and a sustained failure is the absence of a change) · `PAGING_CLASSES = {"regression","sustained-failure"}` with `already_regressed`→`already_paged`, the **legacy** string retained because live 07-29/07-30 rows carry it and dropping it re-pages tomorrow · shell `SUSTAINED` extractor + filing block, with the now-dead `if [[ "$ESCALATED" != "-" ]]` branch removed · emitter↔router vocabulary-lockstep guard · CI anti-skip floor **55→67** (suite 60→71). **The GAP exemption was deliberately PRESERVED and is now mutation-locked** — `#538`'s own headline ("a strictly worse benchmark gets the quieter label") was REFUTED by the planner: the ladder is ordered by *evidence of achievability*, and flake-guard D4 guarantees "no benchmark that has ever passed goes unpaged past its 3rd consecutive failing night". M1/M2/M3 are ONE commit because the routing block **silently drops unrecognised labels**, so the classifier change alone would make chronic failures *invisible* — splitting is not bisect-safe. **Nine mutations total, all RED, all reverted byte-identical.** Replay moves non-vacuously (`guarded_regressions 2→1`, `sustained=1`; bug-reaching total unchanged at 2 — a mislabel suppressed, not a signal). Live-stream check: 07-30 → `SUSTAINED-FAILURE`, 07-31/08-01 suppressed, i.e. pages ONCE not nightly; an `INVALID` night still files nothing, so `#524` is not bypassed. **BONUS reachability find, larger than the evaluator filed it**: D4's "label-agnostic across SUSPECTED-FLAKE and INSUFFICIENT-HISTORY" wording — added in the flake-guard's **quorum round 1** to close "unbounded low-history chains" — guards a state that **CANNOT OCCUR** (`consecutive>=3` already implies `nights>=3`, `trials>=6`); brute-forced 19,607 streams, the only escalation pair that exists is `(SUSTAINED-FAILURE, SUSPECTED-FLAKE)`. A quorum objection satisfied by a fix for a phantom — same shape as iter-121's dead `ast.ListType` arm and iter-105's unreachable `RunAgentBenchmark`. Condition kept (correct if thresholds are retuned), docs corrected]** m-nightly-sustained-failure-label (`#538`; ~0.9d, no new doc needed — a bug fix in `tools/nightly_classify.py` + `tools/launchd/nightly-eval.sh`, same shape as `m-nightly-run-validity-gate`) — **found by iter-122's Gate-0 triage of `#537`**, which the nightly filed as "Nightly regression: `config_file_parser`" and which **refuted under measurement**: 1 pass in 14 trials over 7 nights, never solid, with a *different cause every night* (`TC_ARITY_001` ×2, an `Option[α50]`/`int` unification, a `logic_error`, `undefined variable: null`, a `case`-vs-`match` parse cascade, `thrash_aborted` on the trial-2s). Measured boundary: **0/10 never-green → `GAP` (exempt), 1/10 → `REGRESSION`**, positive control 10/10→break still pages. The consumer already computed the distinction and threw it away, and `:674` banked `label.lower()` into the sole trend input. Sibling filings: `#540` (the honest capability-gap record, so closing `#537` didn't discard the real signal), `#539`, `#541`.
- **[LANDED 2026-07-30 (iter-123) — PR #543 → squash `46d508e7b`, Gate 3b GREEN SHA-addressed (20/20 completed, 0 failures: 14 success + 6 N/A-skipped); 2 commits. Executor codex `gpt-5.6-sol`, evaluator sonnet **PASS 81/100 r1, zero blocking**. Suite 71 → 74, CI anti-vacuity floor 67 → 70. Four mutations controller-run RED **outside the codex sandbox**, every restore sha256-identical, and the evaluator independently reproduced all four plus every claimed gate. **The executor REFUTED the controller's fix directive**: `\\n` → `\n` alone is insufficient at the suspected-flake site, because command substitution strips trailing newlines and the old literal `\n` was what separated the last row from `Model:` — the message template needed a real newline, pinned by MUT-4 (revert ONLY the template newline, keep the awk fix → still red). **Principle-3 census closed**: the three `wc -l` sites (`RCOUNT`/`SCOUNT`/`GCOUNT`) carry the same phantom-row shape but are all guarded, and the `$CLASSIFIED` extractors are filter-based, so `INSUFFICIENT_BODY` was the SINGLE unguarded instance — the fix is complete, not a one-off patch. **Gate 3b green doubles as the cross-awk portability proof** (the new tests assert REAL newlines and `test` passes on ubuntu's awk, not just the rig's BSD awk). Evaluator NOB-1 **ACTED ON** (changelog — measured 2 of 2 for this file family, so skipping would have been inconsistent with the immediately preceding commit) and NOB-5 **REFUTED by measurement** (with BOTH D2 mutations applied, that test's scenario contains no literal `\n` at all → an unreachable guard, declined deliberately). Known residual, out of scope: with no insufficient rows the summary ends with one trailing blank line]** m-nightly-summary-text-fix (`#541`; no doc needed, ~3 lines / <0.5h; P3 but trivial and it is noise a human reads every morning) — found by iter-122's planner (F-4) and **confirmed first-party against a live 06:47 inbox message**, not just read from source. TWO defects in `tools/launchd/nightly-eval.sh`: (1) `:578` `INSUFFICIENT_BODY` is unguarded, unlike its three siblings which are all wrapped in `if [[ -n "$X" ]]` — `echo ""` still emits one line, so awk prints `insufficient history:  ( over  nights, failing /3 toward escalation)` in **EVERY** nightly summary; (2) same line and `:548` use `\\n` inside a **single-quoted** awk program, so awk's `printf` emits a **literal `\n`** instead of a newline (visible mid-message as `…toward escalation)\nModel: …`). Deliberately kept out of `#542` so a behavioural fix was not diluted with cosmetics — the same call `#524` made for its own F-4.
- **[NEXT] m-dialect-keyword-diagnostics** (`#539`; **NEW-DOC needed + quorum**, parser-touching so a Conflict Surface is required; P2 — not a v1.0 bar item; ~1–2d) — found by iter-122 while triaging `#537`. A wrong-*dialect* keyword produces a cascade whose first and most-read entry prescribes **a fix that cannot work**. Measured first-party with a one-token positive control (`match c {…}` → rc=0, `✓ No errors found!`; `case c {…}` → rc=1 with **8** errors): the top diagnostic is `PAR020 missing ';' between block statements … Add a ';' after the previous statement`, and **nothing in the 8 errors names `match`** — `case` is not a keyword, so it parses as a variable reference. Full measured table: `case`/`switch`/`return` → misleading `PAR020`; `data C = A | B` → `PAR015 bare assignment … Use: let C = … in`, i.e. a **wrong** fix (the answer is `type`); `enum` → `expected }, got ,`; `elif` → never names `else if`; bare `fn` (no `export`) → useless, because the good `PAR_EXPORT_REQUIRES_FUNC` message is **`export`-gated**; `struct`/`class` → **degrade to a *type* error** (`undefined variable: struct`), so the agent is not even told the problem is syntactic. **The repo already ships the right shape for this class** — `PAR_MATCH_ARROW` ("match arms use '=>' … not '->'" + verbatim fix) and `PAR_EXPORT_REQUIRES_FUNC` — it just does not cover the rest; follow that template and suppress the downstream cascade. ⚠ **Demand evidence is deliberately WEAK and must not be oversold**: the `case` instance appeared in the 2026-07-30 nightly (`config_file_parser` trial 1: 12 turns, 4867 output tokens, ending in this cascade), but the hypothesis that it drives that benchmark's sustained failure was **REFUTED by per-night scan** — `case` appears on that one night only; 07-26/27/28/29 used `match` correctly 3–7× and failed for unrelated reasons. So this is a real DX defect with ONE observed occurrence. Note the standing counter-argument from `project_docx_stuck_dialect_confusion`: sharper error text may not be the lever, auto-fix might be — the design doc should decide between a dedicated diagnostic and accepting `case` as a `match` alias (`m-syntax-ai-forgiving` established that lane).
- **issue #386 — effect-row inference regression** **[LANDED 2026-07-22 (iteration 82) →
  implemented/v0_31_0; full loop headless — planner opus → executor opus (M1 `6c7a92570` / M2+M3
  `b690a33e0` / M4 `b85860382` / round-2 `456d05afd`) → evaluator sonnet (generator≠judge) PASS
  95/100 after ONE round-2 fix. Application-local equality solver realizing Mark's ratified
  replace-not-delete as NON-deletion (keep constraints + local closed-row arg substitution =
  strictly stronger, guarded by `TestReplaceNotDelete_LetBoundaryPropagation`); `row_unification.go`
  UNCHANGED, no `EffectJoin`; row-var generalization (`RowVars: []string{}` → full free-row-var
  collection, `std/list.ail` untouched); + 2 in-scope secondary fixes (`ValidateEffects` walker
  skipped let/letrec bodies = the `println(show(x))` hole; `std/stream.ail`+`std/ai/streaming.ail`
  handlers row-polymorphic). Non-vacuity independently re-verified (controls still reject undeclared
  IO/FS/Env). All 4 quarantined #386 examples un-quarantined + `working`; `make verify-examples`
  green. Round-1 FAILED 61/100 (stale binary hid 3 stream examples still failing → verify-examples
  red), fixed round 2. PR #459. Known in-scope gap: explicit `! {}` inline-lambda reject is
  elaboration-erased (parser change, out of scope) — proxy fixture uses a non-empty wrong
  annotation. `mcp_tools.ail` stays quarantined (separate `Option[string]` bug)]** — DESIGN DOC created via codex-rotation
  designer: `design_docs/planned/v0_31_0/m-effect-row-show-interp.md` (PR #456). Controller
  live-verified + SHARPENED the root cause: NOT show-specific — two interacting mechanisms
  (`combineEffects` tail-drop → nested pure call erases IO [`println(show(x))` accepted as pure];
  `RowVars: []string{}` never generalized → repeated combinator uses collide). Proven: pure
  `mapE(\x. x*2)` then effectful `foldlE` ALSO fails. Quorum (gemini-3-1-pro, generator≠judge)
  REJECTED ×2: R1 EffectJoin/UnifyRows gap RESOLVED in revision; **R2 OPEN (the human decision)** —
  how the application-local solver drains/preserves solved constraints without breaking
  let-boundary global propagation (gemini: replace, don't delete → flattened-substitution). Parked
  per Gate-2 bounded-quorum rule (1 revision + 1 re-quorum). **UNPARK:** Mark ratifies the
  constraint-preservation mechanism → route straight to sprint-planner, no re-quorum. Est ~1.5–2d.
- **m-check-strict-fallbacks** **[LANDED iter 101 2026-07-24 → implemented/v1_0_0; PR #479 `1978ab44b`,
  evaluator sonnet PASS 88/100 r1. The historical decision record follows:]** **[DECIDED by Mark 2026-07-18 — option 2: post-name-resolution
  pass + curated known-empty-builder registry (catches `Ok(jo([]))`), warning-in-dev / hard-error at
  `check --package`; doc Status stamped UNPARKED → route to sprint-planner, no re-quorum. The historical
  park record follows:]** — iter-42 re-attempt (both iter-41 blockers cleared: Fable designer back; #407 quorum
  fix). Resolved the iter-41 "OPEN decision" to option (a) (syntactic surface-AST pass, hooks
  live-verified) + grounded Pattern C in the language-enforced uppercase-constructor rule — BUT a
  clean re-quorum (on a REBUILT binary; the #407 fix was NOT in the stale installed binary, so
  gpt5-6-sol had been silently unreachable) **BLOCKED** on a goal-contradicting objection:
  **the purely-syntactic pass cannot catch its own motivating incident** `None => Ok(jo([]))` —
  `jo` is a LOWERCASE function call, which the doc (and Pattern C) never flags; catching it needs
  resolved callee identity (name resolution), refuting option (a). Human decision = the architecture
  fork: (1) run after name resolution, (2) narrow the goal (literal-empties only), or (3) curated
  known-empty-builder list; + resolve the warning-vs-error/exit-1 `--package` channel. Doc has the
  full REBLOCK write-up. See log entry 47.
- **m-bytecode-vm-parity-bugs** — **[iter-114: DOC RE-SCOPED A+B then PARKED `needs-human-review`
  on the A2 classification question; ONE decision needed from Mark, options A/B/**C recommended**,
  in the doc's header box]**. Not "output divergences" — **three soundness bugs**: **#505** the VM
  ignores fixed-length **list-pattern length** (`[x]` matches any non-empty list → `recursion_quicksort`
  silently returns `[3]`; general at n=1,2,3; no error, no fallback), the `arith-on-Closure` dispatch
  family (`array_basic`, `array_grid`, `module_let_helpers`), and **#506** the VM→eval fallback
  **restarts the program after committed effects** (`tar_gzip_reader` prints its header twice;
  duplicates a `println` here, would duplicate an FS write / HTTP POST). Fresh iter-114 data at
  `33be8f5a7`: **149 / 2 / 7 / 16** — and the **MATCH headline is inflated by 6 fake rows**: the
  harness passes `--quiet` (`verify_bytecode_parity.go:235`), which suppresses the fallback warning
  (`run_helpers.go:375`), so the evaluator re-runs and matches itself while the VM never ran the
  program (6/6 controller-verified). Quorum BLOCKED twice; both round-0 objections were real and
  reproduced first-party before adoption. **Recommended unblock (C): split — sprint `#505` (B1+B2)
  now** (root cause settled, minimal repro, acceptance test is a pattern table independent of the
  harness) and send the A1/A2 harness lane for a 2nd design round on semantic effect extraction.
  *Superseded below: [RE-SCOPED + Lane-B PARKED
  for Mark, iter-102 Gate-2 fresh data at `64f1e2924`]* — live `verify_bytecode_parity.go` showed
  **150 MATCH / 2 NON_DET / 6 DIVERGE / 16 EVAL_SKIP** (86.2%). Fresh per-file categorization (doc's
  Reality-check-refresh box is authoritative): old bug #3 `string_parsing` now MATCHes; the character
  changed. **Lane A (loop-fixable, ~1d, NO bytecode internals)**: eval `builtinShow` tuple gap
  (`pattern_sugar` — eval wrong, VM correct) + parity-harness honesty (exclude timing `xml_walk_perf`
  + Net `claude_haiku_call`; count clean eval-fallback `tar_gzip_reader` as VM_BRIDGE not DIVERGE).
  **Lane B (DECIDED by Mark 2026-07-27 (attended): GO A+B — full scope INCLUDING the VM codegen
  soundness fixes; was PARKED-for-Mark (h))**: `recursion_quicksort` (VM **silently returns `[3]`** — soundness)
  + `array_basic` (`GET_TAG on Closure` dispatch) are genuine bytecode-VM codegen bugs in the
  "Go/bytecode compilation story" Mark hand-holds; a silent wrong result exceeds the *"3 small output
  divergences"* delegation → Mark scope call before routing Lane B. On his decision: route Lane A (and
  Lane B if in-scope) to the ROTATION designer (next = `codex:gpt-5.6-sol`) → quorum → planner.

**HUMAN-LED lanes (Mark, 2026-07-14 — the loop keeps HANDS OFF unless a sub-item is
explicitly delegated in this queue):**
- **The coordination dashboard** (`ui/` Collaboration Hub + internal/server): six build passes
  since v0.4.4, feature-complete but architecturally unfinished (simplification 1-of-6 PRs done;
  EvolutionTree 2,061 lines), unmaintained since Feb. Day-to-day coordination-watching has moved
  to `ailang chains` CLI + issue #329. Mark hand-holds; in/out decided at the release gate.
- **The Go/bytecode compilation story** (`internal/gen/*`, `internal/vm`, `internal/bytecode`):
  strategy of record = evaluator-first + Statement IR + bytecode (Tier B perf path, ~95% parity);
  Go source emission DEMOTED to diagnostic projection; emit-go-v2 PAUSED (415 symbols short, open
  design-committee question). Mark hand-holds; posture ratified at the release gate.
  (Exception already delegated: m-bytecode-vm-parity-bugs — 3 small output divergences — stays
  in the clause-2 queue.)

**v1.0 RELEASE-GATE AUDIT (one human session, Mark + controller, when the gating queue is
empty — the bundled in/out + ratification calls):**
1. Stability tier assignments (parked since iteration 5: std/net, crypto, jwt, xml, zip,
   process, CLI watch/serve-api).
2. Dashboard: in/out of 1.0 (evidence: dashboard-lineage review 2026-07-14; keeping it OUT costs
   nothing user-facing — chains CLI covers the live path; IN = commit 4–7d to finish the
   abandoned simplification).
3. Compilation posture: ratify evaluator-first for 1.0 (`--bytecode` labeled experimental).
   **PRE-DECIDED (Mark 2026-07-14): emit-go-v2 FROZEN** (contracts projection stays live) —
   formal ratification here; VERIFY the contracts codegen caveat (gen/golang/contracts is live
   via --verify-contracts — if 1.0 materials mention contract compilation, that ships).
4. Boundary split: **PRE-DECIDED (Mark 2026-07-14): m-arch-boundaries ADOPTED** — Phases 1–3
   pre-1.0 (queued, loop-executable), Phase 4 physical restructure AT this boundary (schedule it
   here), separate repos rejected. Audit confirms Phase 1–3 landed + green, then greenlights
   Phase 4 as the first v1.1 act.
5. Effect-scope-params re-score (standing flag from iteration 6).

**The v1.1 arc (spine, Mark 2026-07-14):** *"the bytecode VM grows up, proven by a game"* —
m-arch-boundaries Phase 4 (physical core/apps/tools split) → the game engine as typed effects
(`m-game-engine-effects`, [planned/v1_1_0](planned/v1_1_0/m-game-engine-effects.md)): Stapledon's
Voyage revived on `!{Render, Input, Clock}` host effects, evaluator-first, with the game's
frame-budget as the VM's standing flagship KPI. Go source codegen stays demoted (emit-go-v2
frozen; contracts projection live).

**Mission-infrastructure backlog** (improves HOW the loop runs; not a v1.0 gate):
- **[NEW 2026-07-31, iter-124 retro] m-vuln-allowlist-expiry-warning** (P3, ~2–3h, no design doc
  needed — a flag on `tools/govulncheck-filter`): the allowlist gate fails **ON** the expiry date
  with zero advance warning, so all 8 Ollama entries — which shared a single `expires: 2026-07-31`
  — fired together at midnight and took dev CI red before any human or loop was looking. Iter-124
  re-armed them (`73f4e38bf`) after verifying upstream is still unpatched, but the *mechanism* is
  the finding: an expiry that only ever announces itself by breaking `dev` converts routine
  hygiene into an outage. Fix shape: warn (non-fatal) when any entry is within N days of expiring
  so it surfaces in a normal green run, and have the nightly or `post-release` path report
  upcoming expiries; optionally stagger the dates so a whole cohort cannot fire at once. Note the
  file header already claims "Reviewed: post-release skill prompts a check of expiries" — that
  prompt only fires around releases, which is not often enough to catch a dated fuse. **Demand
  evidence: 1 real outage (this one)** — deliberately P3, and it should NOT be oversold beyond it.
- **[S1 LANDED 2026-08-03 (iter-135) — PR #577 → squash `ab209fcbf`, dev CI GREEN on the merge (19 checks, 0 failures, SHA-addressed); evaluator sonnet 92/100 r1. Patch adopted VERBATIM in a credited commit (`fd838911f`); shared core `ai_stream_core.go` `{record, failLoud}`; fail-loud latch (public prefix `unencodable stream chunk`) + bounded INERT drain (256 chunks / 1 MiB, NO panic/sentinel); 14-row matrix + budget-contract pinning after a controller mutation probe caught the budget test self-referential. Follow-up `#578` filed; `IncompleteStream` question PENDING with @arniwesth on #546. **S2 REMAINS (~1.5–2d)**: example file, both false open-row repairs + CI guard, prompt/μRAG/website, exhaustiveness guard — plan already cut; queues by normal ordering. Was:] [IN-SPRINT 2026-08-03 (iter-133) — PARK CLEARED; DOC REVISED + SPRINT PLANNED, PR #562.**
  Doc → [planned/v0_32_0/m-recorded-stream-api.md](planned/v0_32_0/m-recorded-stream-api.md)
  (moved from `v0_31_0/` — **v0.31.0 SHIPPED 2026-07-29**, so the target and every
  `Since: "v0.31.0"` string were stale; now **v0.32.0**), plan →
  [planned/v0_32_0/m-recorded-stream-api-sprint-plan.md](planned/v0_32_0/m-recorded-stream-api-sprint-plan.md),
  state `sprint_M-RECORDED-STREAM-API-S1.json` (validator **rc=0** after the controller fixed
  `S1_M1`'s `estimated_loc == 0`, which the validator rejects as an unfilled placeholder — see
  `#563`). Designer `claude:claude-fable-5` (rotation advanced), planner **opus**; executor and
  evaluator **NOT fired — S1 execution is the NEXT iteration's Gate 3**, a deliberate deferral
  because the doc's estimate grew to 5.5–6.0 d and a 4–5 day sprint must not be rushed into an
  iteration tail (iter-125 precedent). **Cut into TWO sprints**: S1 = 3.75 d / 5 milestones (file
  split for headroom → verbatim credited patch adoption → shared core → fail-loud + inert bounded
  drain → test matrix → contract text); S2 ≈ 1.5–2.0 d (example, both false "open row" repairs +
  CI guard, prompt/μRAG/website, exhaustiveness guard). **THE FIND: the designer's own
  sentinel-panic abort is UNSOUND ON WASM** — `cmd/wasm/effects.go` hands `onChunk` to JS as a
  `js.FuncOf` wrapper and awaits a promise, and Go's own `syscall/js` doc plus that file's comment
  agree that such callbacks run on a NEW GOROUTINE, so a `recover` scoped to the `StepWithStream`
  call cannot catch the sentinel and an unrecovered panic there is fatal to the module; the file
  is `//go:build js && wasm`, so the proposed containment test would have passed green while the
  WASM path crashed. Controller refuted it, planner confirmed it against the Go source and ruled
  **drain mode ships, the sentinel does not** — every element of Mark's option (c) survives
  (locality, no interface change, 256-chunk/1-MiB budgets, `drain_exhausted` trace, preserved
  typed `Internal`); only a post-failure WALL-CLOCK bound is lost, on a path unreachable for any
  `StreamChunk` constructible today. **That loss is the ONE open decision for Mark.** Verified:
  the reference patch applies to current dev **rc=0** (all 5 touched files byte-unchanged since
  the v0.31.0 tag; controls `--reverse` rc=1 and re-apply rc=1), and the four offered tests pass
  against dev (4/4; package-wide minus two live-network tests = **483 PASS / 0 FAIL**). Was:]
  **[PARKED needs-human-review 2026-07-31 (iter-124) — DESIGN DOC LANDED, quorum BLOCKED ×2, ONE
  SCOPE CALL OWED BY MARK.** Doc was `planned/v0_31_0/m-recorded-stream-api.md`,
  commit `d85934df4`. **Ghost discipline: NOT a ghost — the claim is REAL and understated.**
  Verified first-party at HEAD `130ad1da2` on a freshly rebuilt binary, positive control beside the
  negative probe: an `{IO}` rendering callback checks (rc=0), an `{FS}` recording callback FAILS
  rc=1 with `incompatible closed rows: r1 has extra labels [IO], r2 has extra labels [FS]`;
  `StepResult` carries no chunks; `std/io` has no file write. So live-streaming and chunk-recording
  ARE mutually exclusive. **ADR-009 line 134 independently reproduces the same result against
  v0.30.0** — two parties, same finding. **BONUS DEFECT, folded in as a milestone**: the repo
  TEACHES the opposite in two live places — `std/ai.ail:324` ("the callback's effect row is open")
  sits directly above the closed-row declaration contradicting it, and
  `examples/runnable/ai_streaming.ail:40-42` promises websocket/TUI/metrics side-channels that
  cannot type-check; adopting the recorded sibling does NOT widen the row, so both stay false
  unless fixed. Designer `codex:gpt-5.6-sol` (rotation advanced). Verdict **ADOPT with
  productionization**, routing judged **core, not extension**. **Quorum R1 BLOCKED** (gemini: an
  UNVERIFIED premise — fair, resolved by the controller running the 4 offered tests outside the
  sandbox, all PASS; gpt5-6-sol: silently skipping unencodable chunks contradicts "lossless" and
  Critical Principle 2 — accepted, designer chose FAIL-LOUD, est. 3–4d → 4–5d). **R2 BLOCKED**
  (gemini: my own `-run` isolation was too narrow — resolved,
  `go test ./internal/effects -skip TestNetHTTPRequestBytes_RoundTripSHA` rc=0 with **658 PASS**,
  the patch breaks nothing; **gpt5-6-sol: the fail-loud drain is UNBOUNDED** — and its fix's own
  conditional FIRES, because `internal/effects/ai.go:87` takes **no `context.Context`** and has
  **7 implementers across 6 files including `cmd/wasm`**, so closing it destroys the "purely
  additive" property the ADOPT verdict rests on). **Deliberately NOT force-passed and NOT taken
  under the narrow-refinement carve-out** — that carve-out covers only objections leaving the
  design DIRECTION intact, and this one changes scope; Standing rule 2 → park. **DECIDED by Mark 2026-08-03 (attended): OPTION (c) — bound the drain LOCALLY inside the
  recorded op, no interface change.** Endorsed independently by the AUTHOR (@arniwesth,
  #546 comment 2026-08-01): `StreamChunk` is a SEALED interface (unexported marker,
  exactly 3 implementers, all handled by `encodeStreamChunk`; the one variable-forwarding
  call site is nil-guarded) → the fail-loud drain trigger is UNREACHABLE at current code,
  so (c) is proportionate and (b) is over-engineering — the working iteration inherits
  that evidence from the issue comment, verify the sealed-interface claim first-party
  before relying on it. → ROUTABLE. [Was: **THE ASK (a/b/c in
  the doc header): (a) land now with a documented unbounded-drain caveat, (b) take the `AIHandler`
  cancellation change as a blocking dependency, or (c) bound the drain locally inside the recorded
  op with no interface change — controller's read is (c), avoid (a).**] Quorum metered $0.1086 of
  the $5 ceiling. Was:] **[NEXT-ON-RESUME #1, Mark directive 2026-07-31 (attended): MOTOKO DEMAND — pick FIRST at the
  2026-08-03 re-arm, ahead of the offloads] m-recorded-stream-api** (`ailang#546`, filed by
  arniwesth 2026-07-31 — the STRONGEST demand class: a real external consumer with a WORKING
  IMPLEMENTATION OFFERED): `std/ai.stepWithStream`'s contract (unit-returning `{IO}`-closed
  callback; `Result[StepResult, AIError]` carries no chunks) makes live-streaming and
  chunk-recording mutually exclusive — motoko's deterministic-replay testing needs BOTH at once.
  Arni ships the reference implementation as a PR on his fork —
  **https://github.com/arniwesth/ailang/pull/2** (branch `spike/motoko-009-prototype-v031`) —
  plus a patch verified `git apply --check`-clean against v0.31.0. REQUIRED design context
  (Arni, Discord 2026-07-31): the two motoko DST ADRs this must serve —
  Project 009 Deterministic Test-World Architecture:
  https://github.com/arniwesth/motoko_agent/blob/arniwesth/mot-44-motoko_dst_execution_primer/.agent/projects/009_motoko_dst_execution/ADR-001-deterministic-test-world-architecture.md
  and Project 007 DST definition/taxonomy:
  https://github.com/arniwesth/motoko_agent/blob/arniwesth/mot-44-motoko_dst_execution_primer/.agent/projects/007_dst_consolidation/ADR-001-motoko-dst-definition-and-taxonomy.md
  — the designer/quorum read BOTH before judging the patch. The iteration should
  (1) ghost-discipline the repro at HEAD, (2) evaluate ADOPTING the offered implementation
  (review-the-patch lane — do not reinvent; quorum reviews the DESIGN it embodies, incl. the
  core-vs-extension routing call on a std/ai surface change), (3) credit authorship in the
  commit. Ack posted on #546. Note: the five upstream motoko PRs (#73/#76/#96/#97/#98) are all
  green as of 2026-07-31 (the #98 AILANG_REF guard fixed attended, `324d86d`) and await ARNI's
  review — not our work.
- **[LANDED 2026-08-04 (iter-136) — PR #580, six commits; ALL FOUR MILESTONES + THE FLIP.
  `MISSION_PLANNER_MODEL` now defaults to `codex:gpt-5.6-sol`, so opus is controller-only
  (Mark quota-offload #1). Shipped: `tools/launchd/derive-planner-lane.sh` (Bash 3.2, pure text,
  fail-closed) + 12 fixtures; the driver's role-generic D3 probe loop + the `mission-<name>.env`
  rollback source that D6 already claimed existed; the skill's MANDATORY Gate-3 step 1b; and the
  line-286 flip, gated on a REAL codex planner rehearsal the repo's own validator passed rc=0.
  Doc + plan → `implemented/v1_0_0/`. **Controller mutation probe found a SURVIVOR** — neutering
  the `__UNPARSABLE_PATH_ENTRY__` sentinel changed nothing, because fixture (j) is caught one arm
  earlier by the path-shape check, so a whole arm had zero coverage while the matrix read green;
  fixture (n) closes it. **The plan's own AC9 is VACUOUS** (it pins `opus`, which pre-flip is also
  the built-in default, so it passes whether or not the file is sourced) — only a sentinel makes
  it a measurement. **`#486` non-regression re-proven**: the pinned probe fails with a 400 naming
  the model while the model-LESS control returns rc=0 `ok`. **Engagement rate, not oversold**: 41
  planned docs carry a Files heading, 40 fail closed, exactly ONE declares `**Planner-Lane**`
  today — the lane engages on newly authored infra docs only. **World driver deliberately NOT
  synced** (the plan's recommendation assumed World was kill-switched; it is live) — World fails
  closed to opus via two independent guards; cross-mission message sent; the sync is a PENDING OPS
  CALL for Mark. ⚠ **Evaluator NOT fired — a sonnet judge is owed and is the first task of
  iter-137.** Was:] [(1) DOC + SPRINT PLAN LANDED 2026-07-31 (iter-125);
  EXECUTION was parked to the 2026-08-03 re-arm — PLAN-READY, NO human decision owed.** Doc →
  [implemented/v1_0_0/m-planner-codex-lane.md](implemented/v1_0_0/m-planner-codex-lane.md), commit
  `e980c72d5` (rev 3, 685 lines); sprint plan `b43af2a3e` — 4 milestones / ~307 LOC / ~8h,
  `validate_sprint_json.sh` **rc=0** (the sprint JSON is gitignored local rig state: it survives
  on this machine but is NOT in the repo). Designer `claude:claude-fable-5` (rotation advanced),
  planner **opus**; executor/evaluator **NOT fired** — deliberate, see below.
  **THE FIND: quorum R1 caught a `declare -A` in the proposed driver loop that would have WEDGED
  THE LOOP on the next launchd fire, before any role spawned.** Controller-confirmed first-party:
  the driver's `#!/usr/bin/env bash` resolves to `/bin/bash` **3.2.57**, and **no 4.x bash exists
  on this rig at all** (`/opt/homebrew/bin/bash` and `/usr/local/bin/bash` both absent — which is
  precisely why the launchd `PATH` listing those dirs looked safe). Live repro emits both
  `declare: -A: invalid option` and bash-3.2 arithmetic-evaluating the model name
  (`gpt-5.6-sol: syntax error: invalid arithmetic operator (error token is ".6-sol")`); the
  driver's own shebang form prints `ASSOC UNSUPPORTED`. Second `declare -A` defect here after
  iter-107. The designer then found a **second** Bash-4.0-ism neither reviewer caught
  (`${role,,}` → `bad substitution`), also reproduced first-party. **Quorum BLOCKED ×2**, design
  DIRECTION never contested in either round: R1 (gemini: bash 3.2; gpt5-6-sol: D2 enforced by
  controller judgement rather than deterministically) → revision; R2 (gpt5-6-sol: the classifier
  was **denylist-** not allowlist-based, so unlisted/future language paths silently received the
  same-model planner+executor pairing D2 claims to prohibit; gemini: `derive-planner-lane.sh`
  ignored `$MISSION_PLANNER_MODEL`, **breaking the doc's own D6 rollback**) → **narrow-refinement
  carve-out**, both reviewers' verbatim `proposed_fix` applied, no controller-invented resolution
  substituted. **Then the opus planner refuted FIVE premises**, the two decisive ones
  controller-verified: **R1 — D6's "rollback is one env var" is a NO-OP FOR V1, and BOTH quorum
  rounds accepted it** (`~/.config/ailang/mission-v1.env` does not exist — control:
  `mission-world.env` does — and the driver sources the profile file only when `$MISSION_PROFILE`
  is set, while the V1 plist sets only `PATH`/`HOME`; it works for World, the mission the doc
  called insulated). **R4 — "World blast radius = zero until synced" is FALSE**: `ailang-world`
  has **no repo-local `.claude/skills/` directory at all**, so it loads the GLOBAL skill copy D4
  mandates editing, reaching World at its next fire. R4 refutes a sentence in the controller's
  own commit message and is a **rule-3b SCOPE error by the controller** — the *driver* really is
  byte-identical (verified), but that fact was cited for a broader *blast-radius* claim it does
  not support. R2 — AC1/AC2/AC7 unrunnable as written (the overlap guard at line 334 yields
  BEFORE the dry-run exit at 351, so the ACs reproduce the very vacuous-pass class D5 was written
  to kill). R3 — AC4 can never pass (the main checkout is chronically rig-dirty). R5 — the
  controller's Files-to-Modify ruling is necessary but not sufficient (fixture prose backticks
  `internal/parser/...`, so a literal "every declared path" reader still fail-closes the doc).
  **WHY EXECUTION IS PARKED — a scheduling call, not a blocker**: `tools/launchd/` has **ZERO CI
  coverage** (no workflow references it, no shellcheck/`bash -n` gate, no test file —
  control-verified, since the same grep DID find `tools/` in two workflows), so a green Gate 3b
  says nothing about the driver and the ACs are the only real gate; the point of no return is
  **M2, not M4** (it edits the file launchd fires every 5400 s, and its revert is a *code* revert
  — an env var cannot un-break a driver that fails to parse). Landing unattended on a Friday
  before a quiet weekend buys no capacity, because the offload's whole purpose is stretching the
  week Monday-to-Monday — which is the date Mark set. Also surfaced: only **1 of 102** planned
  docs carries a `Planner-Lane` field and 8 heading spellings exist, so the lane engages only on
  newly authored infra docs (stated plainly, not sold as immediate); the header template lives in
  `design-doc-creator/resources/design_doc_structure.md`, not its `SKILL.md`; and a **third**
  git-tracked skill copy (`.agents/`, 44067 B vs 70544 B) is already drifted (iter-123's `#544`).
  Quorum metered **$0.189** of the `$5` ceiling. Was:] **[NEXT-ON-RESUME, Mark directive 2026-07-30 (attended): QUOTA OFFLOADS — pick these FIRST
  when the loops re-arm 2026-08-03 07:00, before M4b and effect sprints 3/4]** ⚠ **GATE EXPIRED
  2026-08-03 07:00 and item (1) was DEFERRED ONE ITERATION by iter-131, on evidence, not
  preference**: `m-planner-codex-lane`'s HIGH-risk milestone edits `tools/launchd/mission-control.sh`,
  and `#558` measured that launchd executes that file from the **stale main checkout** — so the
  sprint would land green, pass CI, and change nothing about which model the planner actually runs
  on, while the report claimed the capacity was gained. It stays the **next pick** and needs no
  re-greenlight; route it after the checkout reconcile, or land it with an explicit inert-until-
  reconcile note. — two sibling
  recipe/lane items (mission infra, not language changes; neither gates v1.0 but both protect
  its timeline — capacity multipliers land before capacity consumers):
  **(1) m-planner-codex-lane** — route `MISSION_PLANNER_MODEL` through the hardened
  `PROVIDER=codex` spawn recipe (M1b machinery; stdin `< /dev/null` + directive-delivery
  assertion per codex-spawn-recipe-false-greens): planner moves to the ChatGPT-subscription
  bucket, opus remains controller-only. **(2) m-evaluator-gemini-review-lane** — the #399
  Phase-2 follow-on Mark greenlit 2026-07-27: evaluator-as-reviewer over a PRE-MERGE PUSHED
  sprint branch via managed-agents clone-over-egress (CI stays the test oracle;
  generator≠judge preserved — gemini judges codex/opus executors; sonnet remains the fallback
  evaluator when the gemini lane is unavailable). Rationale: the opus bucket dried Thursday
  this week at ~55% duty cycle; these two move the remaining Anthropic-heavy sub-agent roles
  off-bucket so the week stretches toward Monday-to-Monday.
- **[NEW 2026-08-03, Mark directive (attended): TRIAGE BATCH — queue BELOW the reordered top
  picks (#546 → planner-codex → #498 Lane B → evaluator-gemini), never above them]
  m-github-issue-triage-batch** — 12 open issues have ZERO charter mentions (measured 2026-08-03
  ~16:30). Highest-signal: the FIVE `[motoko_agent]` integration defects filed 2026-08-03
  (**#572** step/stepWithStream result omits the required `images` field; **#573** effect checker
  not transitive through function-valued record fields — potential soundness, triage FIRST;
  **#574** `iface` pure-vs-effects contradiction on 12 std/ai exports; **#575** `iface` MOD010 on
  package files with both suggested fixes failing; **#576** unreachable-match-arm warning gap) +
  **#495** (contract/test trio — THIRD surfacing, repeatedly slipping triage) + **#493** (driver
  launchd PATH omits /opt/homebrew/bin — FACT-CHECK against the live codex lane before dismissing
  as stale) + #534/#533/#509/#494/#476. Treatment: triage-lite per issue (ghost-discipline the
  repro, verdict comment on the issue, queue-or-close), NOT a mega-sprint; anything genuine
  soundness (#573 candidate) may then outrank normally per the standing regression rule.
- **[NEW 2026-07-29, Mark directive (attended)] m-outage-triage-lane** (NEW-DOC needed; P3
  resilience — design-only until a second outage recurs; does NOT gate v1.0): when ALL Anthropic
  controller probes fail with a SERVER-error signature (500 Internal / 529 Overloaded — distinct
  from `QUOTA_SIG`, which must keep its existing quota semantics), the driver falls through to a
  bounded `codex exec` OUTAGE-TRIAGE iteration instead of aborting: a distilled Gate-0/1 recipe
  (inbox triage, nightly-regression ghost-probe, bookkeeping, park-everything-else) — NO sprints,
  NO skill edits, NO quorum (planner/evaluator/controller-verification stay Anthropic-only).
  Motivation: 2026-07-29 evening Anthropic 500/529 outage — the 21:22 iteration aborted after 3
  bounded retries (rc=1, correct behavior) and the 13:52 sprint iteration had already stretched
  into the 6h hard timeout under the same degradation; the loop was pause-only because the
  controller is structurally Claude Code (per-model failover exists, per-provider does not).
  Demand evidence: first outage in ~3 weeks of continuous operation → LOW priority by design;
  the doc should reuse the hardened `PROVIDER=codex` spawn recipe (stdin `< /dev/null`,
  directive-delivery assertion) from codex-spawn-recipe-false-greens below.
- **[LANDED 2026-07-28] [world-DEMAND] codex-spawn-recipe-false-greens** (iter-112 Gate 5 — the one skill edit; BOTH fixes applied to `mission-control/SKILL.md`'s `PROVIDER=codex` recipe: `< /dev/null` on both the probe and the real run, plus a directive-delivery assertion (exists · ≥200 bytes · non-empty loaded prompt → `exit 64`). Iter-112 also **used** the fixed recipe first-party for its own codex executor run, and the secret-safe `[ -n "$VAR" ]` env-probe form is documented alongside. Original row: **NEW iter-111**; SKILL FIX, ~10 min, no design doc — apply at the NEXT iteration's Gate 5, since iter-111 had already spent its one skill edit) — the shared `PROVIDER=codex` spawn recipe in `mission-control/SKILL.md` has **two false-greens**, proposed by `world-coordinator` (mission-world iter-26, which cannot edit the shared skill itself) and **corroborated first-party by iter-111 rather than taken on trust**. **Defect 1 — stdin is never redirected**: `codex exec` reads stdin IN ADDITION to the positional prompt, so under a backgrounded launch with an open (never-EOF) stdin it prints `Reading additional input from stdin...` and blocks until the 30-min cap — a hang that *looks* like normal long work. World observed a 39-byte log and zero diff after 6 minutes. **Iter-111 corroboration: that exact line appears once in this iteration's own `codex_out.log`** — the read happens here too; the run only survived because stdin happened to EOF under the harness's background launch. Fix: append `< /dev/null` to the `codex exec` invocation. **Defect 2 — no assert that the directive was delivered**: with the directive file absent, `"$(cat …)"` expands to empty, codex replies "What would you like me to work on?" and **exits rc=0**, and the wrapper reports success for a run that did nothing. **Iter-111 corroboration: a genuine near-miss — this iteration's FIRST `Write` of `/tmp/codex_directive.txt` FAILED** (pre-existing file, "has not been read yet"); had the runner been launched before that was noticed, the spawn would have been a silent rc=0 no-op. Fix: assert the file exists and is non-trivially sized AND the loaded prompt variable is non-empty, aborting loudly otherwise. This is the same vacuous-pass class the mission has closed twice elsewhere (silent z3 skip, silent `t.Skip`): **an exit code reporting success for work never requested**. Meets the ≥2-frictions bar on its own (2 distinct defects, same gap, one iteration) and is independently corroborated here. **Hygiene note, broadcast with it (not a skill defect)**: a shell "is this env var set?" probe written `${VAR:+YES}${VAR:-NO}` **prints the variable's value** — World leaked `OPENAI_API_KEY` into a transcript this way. Safe form: `[ -n "$VAR" ] && echo SET || echo UNSET`. Any preflight env check in this loop must avoid the `${VAR:-…}` form for secrets.
- **[LANDED 2026-07-28] m-docs-gate-not-required** (iter-112, FULL loop in one iteration — planner **opus** → executor **`codex:gpt-5.6-sol`** → evaluator **sonnet PASS 88/100 r1, zero blocking**; PR **#501** → `a3e781b26`, PR **#502** → `cdac5bf04`; sprint plan → `implemented/v0_30_0/`. **`docs-gate` is now a REQUIRED check on `dev`** — `["test","lint","build","docs-gate"]`, applied via `PATCH .../required_status_checks` with the whole protection object diffed byte-wise outside that key. Option (a) as filed: `build`→`docs-build` (collision gone, measured exactly ONE `build` check-run), `on.pull_request.paths` dropped, `docs-changes` detector (`git diff` vs merge base, path list in `.github/docs-build-paths.txt`, no new third-party action), always-reporting `docs-gate` with a **mandatory catch-all arm that exits 1**. **Both branches OBSERVED green before the flip**: #501 build branch (`docs_changed=true`, build success) and #502 skip branch (one `design_docs/` file, `docs_changed=false`, `docs-build` **skipped**, gate success in **38 s**) — then #502 merged under live protection as the canary. Concurrency also rescoped per-ref (the shared `pages` group had already cancelled 4 real PR runs in 60 s; `deploy` keeps the singleton and was proven still green on the dev push). **Refuted premise worth keeping**: a job skipped by an `if:` reports **Success**, not Pending — so requiring `docs-build` directly would have made "not checked" indistinguishable from "passed". Systemic twin → **[#503](https://github.com/sunholo-data/ailang/issues/503)** (`ui-build`, same not-required half, live `/ui` npm Dependabot; carries the deferred least-privilege `permissions` tightening). #497 CLOSED with the verdict + rollback command. Original row: **NEW iter-109**, filed as **[#497](https://github.com/sunholo-data/ailang/issues/497)**; P2, ~0.5d; no design doc needed — the issue enumerates the options) — the docs build **never actually gates Dependabot PRs**, so a docs-breaking bump auto-merges and takes dev red. Two independent causes, both first-party verified this iteration: (1) `dev` branch protection requires only `["test","lint","build"]`, and `dependabot-automerge.yml` runs `gh pr merge --auto`, which merges the moment those go green — on #488 the docs run was **cancelled mid-flight by the merge** (`Deploy Documentation to GitHub Pages | pull_request | completed/cancelled` @ `23c428378`, PR merged `13:00:32Z`, run started `12:45`); (2) `docusaurus-deploy.yml:58` names its job **`build`**, colliding with `ci.yml:315`'s `build` — required contexts match by NAME, so ci.yml's Go build satisfies the requirement while the identically-named docs job is silently irrelevant, i.e. **the gate appears wired in the checks list without gating anything**. `ci.yml:337`'s `docs` job is NOT a docusaurus build (prompt-sync only, and `if: push && ref==dev` so it is skipped on every PR). ⚠ **The naive fix is WRONG and must not be applied**: `docusaurus-deploy.yml` is path-filtered, so making it a required check would leave it permanently pending — and thus blocking — on every non-docs PR. Options in #497: **(a)** rename `build`→`docs-build` (worth doing regardless — the collision is a latent trap) + an always-reporting wrapper job that internally skips when no docs paths changed, which CAN safely be required; **(b)** exclude the `/docs` npm ecosystem from auto-merge (cheapest stopgap, costs the batching auto-merge exists for); **(c)** have auto-merge wait explicitly on the docs run. Blast radius already demonstrated twice: #488 took dev red ~7h, and #490's react/recharts bumps merged 60s later with a docs build that **never once ran green** (verified fine only because iter-109 built it locally). Mitigation shipped in `4b757f63d`; the gate itself is untouched.
- **[LANDED 2026-07-28 — iteration 113]** m-nightly-flake-guard (PR #504, squash `038d9322d`, dev CI green; evaluator sonnet **PASS 87/100 r1, zero blocking**; doc + sprint plan → `implemented/v1_0_0/`). Shipped: classifier extracted to `tools/nightly_classify.py` (behavior-preserving, byte-identical over all five surviving nights) · history off `/tmp` onto a JSONL with atomic write + ownership-checked lock (PID + random token, `os.kill(pid,0)` liveness gating any steal) · trailing-window solidity rule (W=5, MIN 2 nights/4 trials) with `suspected-flake`/`insufficient-history` labels where only `REGRESSION` reaches `--type bug` · replay over the real corpus: **`filed=5 → guarded=2, suppressed=4`**, non-vacuous because **#483 `higher_order_functions` (4/4 solid) still pages the SAME night** and `csv_to_json_converter` newly escalates on 07-27. Controller mutation-proved the guard in **three** directions (suppress-everything → 3 red incl. the same-night test; suppress-nothing → 6 red; liveness-blind steal → 2 red incl. `test_Lock_stale_but_alive_holder_is_not_stolen_from`), each reverted byte-identical (`sha256 a53f1e3a…`). CI wiring is real, not vacuous: **`make ci` is run by NO workflow**, so the tests attach directly to `ci.yml`'s test job + a step failing under 20 `--- PASS:` lines — and the job log proves it RAN (40 PASS in Actions). Deviation: `consec >= K AND not already-regressed` replaces the doc's literal `consec == K`, which loses the escalation forever on a missed night. ⚠ **Landing action**: run `--bootstrap` once before the next 05:00 nightly, else night 1 is loudly DEGRADED with zero issues filed. Was: (**DOC LANDED + quorum-cleared iter-106** → [planned/v1_0_0/m-nightly-flake-guard.md](planned/v1_0_0/m-nightly-flake-guard.md), commit `6ad39b863`; P2, ~1.3d, 4 milestones) — the nightly eval regression detector has **no variance/flakiness guard**: it compares this run against the single previous run (N=2 trials each) and files a GitHub issue on any pass→fail flip, so a benchmark that is merely *bimodal on a local model* generates recurring false alarms. Evidence: `json_parse` on `opencode-qwen3-5-35b-a3b-mxfp8` produced **four** nightly issues (#286, #292, #480, #485), all closed as noise; banked history 1/2 → 2/2 → **0/2** → 2/2 → **0/2** (07-23…27, rag_on), i.e. the 07-25 and 07-27 alarms are the SAME flip and 07-25 self-recovered with zero action. Each false alarm costs a Gate-0 triage slot and, left open, reads to external viewers as an unresolved regression (#417). **Design decides**: history off `/tmp` (the amnesia's cause — only 6 nights survive) onto a classifier-owned JSONL with an explicit idempotency/atomicity contract + ownership-checked lock; trailing-window solidity (W=5, MIN 2 nights/4 trials) with a label-agnostic K=3 escalation backstop, replayed on the real history to show #480/#485 suppressed while the solid→broken control still pages SAME night; new `suspected-flake` / `insufficient-history` labels where only `REGRESSION` reaches `--type bug` (which is what creates the issue); explicit `--bootstrap` so an absent history file is ALWAYS the loud DEGRADED state. **Quorum**: designer `claude:claude-fable-5`; reviewers `gpt5-6-sol` + `gemini-3-1-pro` + controller opus; R1 blocked → designer revision; R2 blocked on new narrower objections → **narrow-refinement carve-out** (both reviewers' `proposed_fix` applied verbatim). ⚠ **CORRECTION (V8, this row's own prior text was WRONG)**: the earlier bullet *"fold in: compare like-for-like CONDITIONS — #485 compared against yesterday's `_rag_on` while today produced both"* is **false and is dropped** — both sides of the comparison already use the rag_on arm (`nightly-eval.sh:233` and the `*_rag_on/agent` glob at `:245`). Do not design a fix for it. **NEXT**: route to sprint-planner.
- **m-mission-adaptive-multiprovider-routing** ([planned/v0_30_0](planned/v0_30_0/m-mission-adaptive-multiprovider-routing.md); EXPANDED 2026-07-14 per Mark — quota now the binding constraint) — the heterogeneous model FLEET. **[Phases A+B LANDED 2026-07-14, iteration 28]**: Phase A (quota-aware multi-candidate probing in the driver) landed `3bee6b6df` direct-to-dev by the interactive session + verified/hardened by the sprint; Phase B (design-doc QUORUM: gpt-5.6-sol + gemini-3-1-pro-via-Vertex-ADC + Claude controller in-session, reject-by-default, N−1 named-absence degrade, budget-capped) landed PR #383 → `1186a48e6`, eval PASS 94/100 round 1 — `ailang design-review`/`design-quorum` live, artifacts under `.ailang/state/mission-quorum/`. REMAINING (opt-in as evidence accrues): Phase C cross-provider executors (re-scoped ~1d, audit binding); Phase D local-GPU lane (~2–3d); Phase E full (provider, model)×task-class assignment (~3–4d). Quorum-on-sprint-plans deferred (hook scoped to design docs). Requested + prioritized by Mark.
- **m-arch-boundaries Phases 1–3** **[LANDED 2026-07-20 (iter 68) — PR #420 squash `ee97fada6`; evaluator (sonnet, generator≠judge vs opus executor) PASS 88/100 r1; doc → [implemented/v0_30_0/m-arch-boundaries.md](implemented/v0_30_0/m-arch-boundaries.md)]** — `scripts/check_boundaries.sh` self-testing import-boundary CI gate (Rule 1: no core→dashboard; Rule 2: no dashboard→compiler-surface except via `internal/embed`; MODULE-vs-`go.mod` drift guard; `eval` excluded from Rule 2 for the sanctioned `eval.Value` bridge type, documented) + `make check-boundaries` + CI step + `ARCHITECTURE.md`/`CLAUDE.md` boundary docs + `.github/CODEOWNERS`. **No physical restructure** — Phase 4 (`git mv` core/apps split) reserved for the v1.0→v1.1 boundary; dual-release-tracks out of scope. Planner (opus) corrected 5 stale doc premises; executor (opus) caught 2 real defects (wrong module anchor → false-pass; `server→eval` bridge import); `metered=$0.00`. **Follow-on queued: m-arch-boundaries-eval-exclusion-tighten** (evidence-gated — tighten the `eval` exclusion package→file level; only 1 file uses it today).
- **m-mission-quorum-agentic-verify** ([planned/v0_30_0](planned/v0_30_0/m-mission-quorum-agentic-verify.md), 2026-07-14; P1) — the shipped text quorum REASONS but cannot VERIFY (no repo access); this makes reviewers tool-using agents (codex/managed_agents, read-only worktree) that actually run `ailang check`/grep to confirm premises, two-tier (cheap text first → agentic escalation only when a premise is contested). Reuses the quorum contract + executor registry. Sequenced after fleet Phase C. Precondition: confirm Tier-1 has fired LIVE (no artifacts found yet). Requested by Mark.

- **m-mission-portability** — **✅ COMPLETE (M1 attended 2026-07-21, M2+M3 landed iter-92 2026-07-23).
  M1: driver parameterized (MISSION_PROFILE/NAME/REPO/DOC, v1-legacy-exact vs namespaced state,
  template plist). M2+M3: `## Repo Profile` in SKILL.md + charter header (verify profiles
  go-compiler/ailang-code), public bootstrap guide + charter template, dry-run isolation proven,
  evaluator sonnet PASS 83/100. Doc → implemented/v0_30_0/. Ailang World launch UNBLOCKED:**
  ([planned/v0_30_0](planned/v0_30_0/m-mission-portability.md), 2026-07-18;
  **P1 mission-infra — GATES THE AILANG WORLD MISSION LAUNCH**, Mark: "design doc this up and plan
  it in") — extract the loop into a portable template: M1 driver parameterization + per-mission
  state namespace (`MISSION_NAME/REPO/DOC` profile env; backward-compatible defaults — this
  mission's behavior unchanged), M2 skill repo/verify profiles (go-compiler vs ailang-code —
  World verifies via `ailang check/test/ai-check`, which the binary ships), M3 bootstrap kit +
  charter template + scratch-repo dry-run (no state collision with the live loop). ~1–1.5d, zero
  language surface. **Pick order: after the greenlit clause-3 trio (fmt → footguns → strict-
  fallbacks) — OR earlier if the clause queue blocks on anything.** ONE skill parameterized, never
  forked (Gate-5 retro fixes must keep benefiting all missions). Expect quorum-at-pick (doc
  authored interactively, no creation-time quorum).
- **m-eval-reasoning-model-fairness** ([planned/](planned/m-eval-reasoning-model-fairness.md);
  authored 2026-07-11, **QUEUED by Mark 2026-07-19, P1**: "why does GLM 5.2 perform worse than
  5.1? We think it may be our eval harness's fault — thinking tokens/limits with OpenRouter") —
  the doc already carries the evidence: GLM-5.2 40/56 vs 5.1 48/56 with negative token counts,
  empty `code` fields despite compile_ok, and NO reasoning request/budget (MaxTokens bounds total
  output → thinking crowds out the answer). Iteration 43 proved the same mechanism live in our
  quorum (PR #408) — apply the same remedies: reasoning-aware budgets, fail-loud on
  `finish_reason=length`, per-turn finish_reason capture, then RE-RUN the GLM pair to split
  harness-artifact from genuine regression. ~1–2d, metered-cheap (OpenRouter), no GPU. Expect
  quorum-at-pick. Eval-infra (non-gating for v1.0) but Mark-prioritized — pick after the
  greenlit clause-3 trio unless the queue blocks. **RE-RUN VERIFICATION MODELS: the GLM 5.1/5.2
  pair + Kimi K3 (top OpenRouter model, 97/109 standard — also reasoning-class).**
- **m-comments-for-ai-authors** ([planned/v0_31_0](planned/v0_31_0/m-comments-for-ai-authors.md),
  **direction RATIFIED by Mark 2026-07-20**: prompt style guidance + first-class `---`
  doc-comments + contracts/tests-as-self-documentation "as much as is reasonable" + the eval) —
  M1 prompt comment-style section (≤15 net lines, prompt-manager lane, ~0.5d) · M2 the
  comment-variant A/B (V-strip / V-keep / V-migrate on MODIFICATION tasks, haiku, N-run
  aggregates; registered hypotheses; SHARES m-eval-fmt-weakmodel-ab's variant machinery — build
  once, run both) · M3 first-class `---` doc-comments as AST nodes (v0.31; dissolves fmt
  attachment at the root for the doc position; sequence AFTER the fmt polish pair) · M4
  contracts-as-docs exemplars (rolling). First measured comment semantics for AI authors.
- **m-eval-fmt-weakmodel-ab [M1+M2a LANDED iter-72 (PR #438 squash `260faa42a`); M2b LANDED iter-74
  (60 runs banked, 30/arm × 5-trials × 6 frozen benchmarks; cloud-haiku via `claude` CLI on
  SUBSCRIPTION, NO rig.lock; treatment delivery PROVEN 29/30 ON runs fired the hook vs ~8% baseline;
  arm gating clean OFF=0 fmt events); M3_ANALYSIS LANDED iter-76 (PR #450; opus executor + sonnet
  evaluator PASS 87/100; **VERDICT = NULL published**: delta ON−OFF +0.033, Newcombe95 [−0.083,+0.167]
  includes 0 & < +0.10 → not H1-supported/not harm; treatment 32/32=100% exit-0 → true null NOT void;
  green-stability NOT-COMPUTABLE from banked data, reported honestly; helper `tools/analyze_fmt_ab.py`);
  **M4 VERDICT LANDED iter-77 (PR TBD): final verdict = NEUTRAL / true NULL, treatment proven delivered
  — hook stays opt-in, NO adoption-policy change; doc set moved to implemented/v0_31_0/ → SPRINT COMPLETE 5/5]** →
  [implemented/v0_31_0](implemented/v0_31_0/m-eval-fmt-weakmodel-ab.md)** — Mark #422 "Green light weakmodel
  ab" (2026-07-21) UNPARKED it → planner (opus) → executor (opus, worktree) → evaluator (sonnet,
  80/100 PASS r1) + round-2 fix. **LANDED: M1 preregistration** (`-prereg.md`: 6 `.ail`-editing
  benchmarks frozen — fizzbuzz/gcd_lcm/adt_option/higher_order_functions/json_parse/cli_args — N=5/arm,
  Wilson CIs, refutation threshold +0.10 AND CI-excludes-0) + **M2a fmt-hook ON/OFF toggle**
  (`-fmt-hook on|off` CLI flag, default off; `FmtHookMode` on the `microrag_mode.go` precedent; ON
  emits workspace `.claude/settings.json` registering `format_ail.sh` PostToolUse Edit|Write; active
  path `agent_runner_multi.go`; fail-closed hook-reality capture banked as `fmt_hook_events`).
  **M2b DONE (iter-74)**: haiku ON vs OFF ran at N=5 on the 6 frozen benchmarks, banked to
  `eval_results/fmt_ab_haiku_M2b/{on,off}/` (60 run files). Cloud-haiku is an API/subscription model,
  NOT GPU → NO rig.lock (`-no-rig-lock`; the sprint-plan's rig.lock text was superseded by mission
  commit `69501e6dd`). Config-diff clean (both arms prompt_version v0.16.3, model, seed 42, parallel
  4, trials 5 — only `fmt_hook_state` differs). `TODO(M2b)` RESOLVED: the file-sink capture works —
  29/30 ON runs banked a `formatted` fmt event (vs ~8% baseline); OFF banked 0 (arm gating).
  **NEXT = M3 analysis + M4 verdict** (no-GPU, on the banked data): rigorous Wilson-CI deltas,
  convergence, per-turn fmt exit-code coverage; the headline (OFF 29/30, ON 30/30, +1-run driven by
  cli_args only) trends NEUTRAL/NULL-at-haiku-ceiling with treatment delivery proven. SonarCloud
  new-coverage 39.3% (non-required) is expected for the integration wiring. (Mark 2026-07-20
  — "fmt should be a real help for weaker
  models creating AILANG… can we do a test with a weak model to see if its making a difference?"
  + his #422 directive "test it's used by small model such as haiku"): A/B agent-mode evals,
  ONE weak model (haiku first; optionally a local small model as replication), fmt PostToolUse
  hook ON vs OFF, same benchmarks/N-runs. Metrics: pass rate + compile-stuck/green-stability
  convergence (the noisy-agentic-metrics rule: N-run aggregates, never single runs) + per-turn
  fmt exit codes (was fmt actually invoked/useful). Hypothesis: canonical formatting reduces
  weak-model syntax drift. Depends: fmt+adoption (LANDED); sequence AFTER the fmt polish pair
  below lands (test the finished tool, not the interim). ~0.5d + eval time, subscription/cheap.
- **m-eval-kimi-k3-agentic** ([planned/v0_31_0](planned/v0_31_0/m-eval-kimi-k3-agentic.md),
  Mark 2026-07-19: "Kimi K3 did very well — look into using it within the suite via OpenRouter
  and Pi or motoko harness") — K3 = **97/109 (89%) standard, the strongest OpenRouter model on
  the v0.30.0 board** (beats GLM-5.2 88, K2.7-code 88, GLM-5.1 85). Onboard it AGENTICALLY:
  `motoko-or-kimi-k3` + `pi-or-kimi-k3` roster entries (K2.6 precedent, mechanical), smoke→core
  tiered runs, 4-way comparison (vs its own standard score, vs K2.6, vs GLM-5.x, motoko-vs-pi
  harness effect), routing-evidence rows; if it clears the sweet-spot bar → PROPOSE for the
  fleet's Phase-E table (admission stays a routing-policy decision). ~0.5–1d, metered-cheap,
  no GPU. **HARD-SEQUENCED AFTER m-eval-reasoning-model-fairness** — K3 is always-reasoning;
  measuring it agentically on the pre-fix harness = the broken ruler. Expect quorum-at-pick.
- **m-mission-loop-heartbeat [NEW, 2026-07-21 — born from the 18h reboot outage]**: a tiny
  SECOND launchd agent (independent of the loop it watches) that every ~2h checks: newest driver-log
  line older than ~4h AND no kill switch AND no live pidfile → send a controlplane alert + ⚠ comment
  on the bookkeeping issue + `launchctl kickstart` the mission job (recovery, not just alarm). The
  2026-07-20 reboot silenced the loop for 18h and only a human ping caught it — the loop needs a
  pulse that does not share its failure domain. ~0.5d; pairs with RunAtLoad=true (b5b9899a0: repair)
  as detect+repair. Also: the driver should DELETE a stale pidfile whose boot-time predates uptime
  (reboot invalidates PIDs — a reused PID would false-yield every fire; cleared by hand this time).
- **m-mission-cost-chains** ([planned/v0_30_0](planned/v0_30_0/m-mission-cost-chains.md), 2026-07-18;
  **P1½ — the clause-5 KPI's data substrate**, Mark: "keep an eye on these budgets… that should
  all appear in ailang chains CLI") — **[LANDED iter-100 (2026-07-24) — Mark scoped-inference
  decision `4e1348adb` folded → FULL sprint loop, evaluator sonnet PASS 92/100 r1, PR #478 → squash
  `08f9204d0`, all required checks green; doc → implemented/v0_30_0]** (parked-history below). Mark chose SCOPED read-side
  inference (estimate only token-bearing/no-cost/no-quota-bucket/non-zero-rate; quota lanes
  `$0`-by-design; NO schema migration), overriding the reviewers' persistence-required conclusion.
  Controller folded it into the M1 body (no re-quorum, apply-verbatim precedent iters 98/99); planner
  (opus) proved the M2 quota-marking is structurally sound with no migration + found M1 is smaller
  than framed (rate registry already wired into `observatory/pricing.go`); executor (opus, worktree)
  shipped M1 classifier + M2 `chains post-iteration`/bounded-loud spool + M3 `--by-mission`; the
  round-2 soundness hole is closed (quota gate fires before model recovery + a `Validate()` guard).
  **Gate-2 reality-check (iter-97, `v0.30.0-147`):** the doc's Defect-A headline (**cost=$0.0000
  everywhere**) is STALE — recent eval chains attribute cost ($9.59/48h; at-source fix `43333e7a8`
  2026-07-19). Residual M1 (rollup rate-fallback), M2 (mission-ingest — 0 mission chains/14d), M3
  (`--by-mission` absent) all confirmed REAL → valid pick, not a ghost; baseline re-pinned in the
  doc. **QUORUM-AT-PICK:** r1 BLOCKED → 1 fable designer revision (cost provenance + registry
  verification + bounded/loud spool) → re-quorum r2 STILL BLOCKED on a **convergent soundness**
  objection: M1's CLI-side `cost==0 → estimated` inference would corrupt legitimately-free/quota $0
  stages (incl. **M2's own quota-lane `cost_usd=0`**) because the `float64` schema can't distinguish
  absent-cost from reported-$0. Converged fix (both reviewers): persist provenance (nullable
  `Cost *float64` OR `cost_status` field + `quota_bucket`) as a schema migration. Genuine
  schema-design gap → parked. **Human fork (in the doc's Quorum record):** (1) pick M1 provenance
  persistence (pointer vs field); (2) authorize `quota_bucket` on `ChainStage`. `metered=$0.0705`.
  M2/M3 direction unobjected. Original scope: M1 rollup fallback · M2 Gate-4 posts one chain per
  iteration (`mission:<name>/iter-N` — portability-ready for World; bounded+loud spool) · M3
  `chains stats --by-mission`. ~1.5–2d + the migration. **Sequence BEFORE m-cost-per-success-kpi.**
- **m-public-feedback-delivery-audit** ([planned/v0_30_0](planned/v0_30_0/m-public-feedback-delivery-audit.md), 2026-07-12; **P1**) — external user feedback (Kevin's) silently lost: ROOT-CAUSED: dev/prod env split (Mark). Public MCP writes feedback to PROD (`ailang-multivac`) — Kevin's June-30 messages are there, triaged; the rig daemon subscribes to DEV only, so external feedback never pings Discord. Fix = daemon dual-subscribes dev+prod; plus the latent pkg:*-inbox Discord-filter bug. The human-input channel that feeds the data-led loop — prioritize. Requested by Mark.

- **m-mem-budget-runtime** ([planned/v0_31_0](planned/v0_31_0/m-mem-budget-runtime.md), 2026-07-21;
  **P1 — host-safety, DOC-READY**, Mark: "make a design doc for this to insert into our mission
  loop sequence") — the 2026-07-20 rig kernel panic (watchdogd starved under swap-thrash; Jetsam:
  3 model-generated Python procs at ~80-120GB, ailang at 7.7GB) proved generated code WILL
  occasionally be a memory bomb. AILANG's protection today is incidental (no while/mutation +
  interpreter speed) — this makes it guaranteed: `--max-mem`/`AILANG_MAX_MEM` → Go soft limit +
  memguard monitor + cooperative unwind → typed `MEM001` (verified unallocated) instead of host
  death; harness banks it as a distinct error category (model signal, not rig outage). Extension
  lane, zero syntax change, `Mem`-as-effect explicitly rejected (A3/A8). Complements (does not
  replace) the harness-side RSS watchdog task covering the Python/JS/Go lanes. Verification Log
  complete incl. negative-existence rows; Design Freeze needs quorum ratify of the two frozen
  decisions (runtime-control-not-effect; default-off CLI / explicit-on harness). ~2-3d. Phase 2
  (deterministic logical meter, replayable exhaustion) split to a future `m-mem-meter-logical`.

- **m-decision-entropy-monitor** ([planned/v0_31_0](planned/v0_31_0/m-decision-entropy-monitor.md),
  2026-07-22; **[PARKED needs-human-review] — quorum-blocked ×2 (iter 84 2026-07-22).** Rev-1
  flaw (ifaceSeverity can't produce JSON on the broken post-edit file it targets) was fixed by the
  Rev-2 designer pass, but re-quorum surfaced DEEPER blocks needing Mark's design call: (1) A1/A2
  non-determinism — the grader runs the LIVE binary under a wall-clock timeout on reconstructed
  single files with no banked compiler-identity/workspace closure; (2) A5 — `ailang iface` needs
  the workspace to resolve imports, so single-file `/tmp` extraction fails; (3) Conflict-Surface
  omission + overlap with `analyze_stuck.py`/`analyze_run_steps.py`. **Human fork (in the doc's
  ⛔ Quorum Record):** bank iface-JSON + compiler-identity at COLLECTION time (fixes 1+2) vs
  hermetic replay; OR ship M2 `iface --diff`-only (unblocked, independently useful) + defer the
  `D`-grade iface feature. Was: **P2 eval/mission-infra, DOC-READY**, Mark: "detect when big decisions (that have
  large entropy consequences) are made during AILANG code generation — a way to grade when we need
  to closely examine the decisions") — grade every agent step with a decision-weight `D` from
  signals already banked (per-edit `typecheck` green→red, move class WriteFile/bash-write vs
  EditFile/EditDecl, per-path churn) plus the AILANG-native consequence measure: **interface
  delta** — `ailang iface` already emits normalized signatures+effect rows; the diff over them is
  the ONE unbuilt piece (V1–V10 verification log in doc, incl. the negative-existence greps).
  Grounded in green-stability: decision class predicts convergence, so grade the fork-step, not
  just the post-mortem. M1 offline validation on the labeled docx spiral/converger corpus
  (prereg + honest-null, fmt-weakmodel-ab template) → M2 `ailang iface --diff` severity
  none/additive/breaking (additive, independently useful — an agent can check its own blast
  radius pre-commit) → M3 `decision_profile` on RunMetrics + observatory top-`D` view,
  **evidence-gated on M1's report + human review**. Extension lane, zero language surface, no
  motoko-fork changes (pure session-JSONL consumer). Future work under its own evidence bar:
  best-of-N branching / reasoning-effort escalation AT high-`D` steps (consumer:
  m-ai-reasoning-effort). ~2.5–3d, no GPU. Expect quorum-at-pick.

**Not gating** (the ~30 non-gating docs (eval-infra rig/harness, cloud-infra, motoko-fork, post-v1)): ship on the normal v0.2x road or post-v1 per the
clause rule. `planned/v1_0_0/` now contains ONLY gating docs (17 non-gating docs re-bucketed to
v1_1_0 on 2026-07-11); v0_29_0 docs that appear above gate v1 via the queue, not the folder.

**Post-v1**: everything in `planned/v1_1_0/`.

## Ruled out / resolved

- **Sonnet as default executor** — ruled out 2026-07-10 (Mark: corrections needed; false economy).
  Re-entry only via the evidence rule.
- **Scheduling via cron / scheduled-tasks MCP** — ruled out; this rig's substrate is launchd
  (nightly-eval + os-rotation-filler precedents), and the coordinator has no internal timer.

## Done / superseded

*(nothing yet — mission initialized 2026-07-10)*
