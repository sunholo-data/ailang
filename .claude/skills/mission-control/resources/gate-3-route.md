## Gate 3 — ROUTE + EXECUTE (the inner loop, with the routing policy)

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-3`.

**Routing is ENFORCED per-role model pinning — NOT session-model inheritance.** Running every role
on the controller's single session model is the routing-never-enforced bug: with the driver on
Fable, 100% of every iteration billed Fable (fixed 2026-07-15, m-mission-agentic-provider-routing
M1 — memory `project-mission-routing-table-never-enforced`). **Invariant:** the controller session
(triage/pick/judge/retro) uses the driver-selected `$MODEL`; every HEAVY role — **including
design-doc-creator, which is the spawned ROTATION designer, never inline** (see the roles table below) —
is spawned as a **model-PINNED `Agent`/`Task`/provider sub-agent**, never inline. Read each role's
model from the driver-exported env (defaults track the charter table):

| Role | Model env | Default |
|---|---|---|
| Controller (this session: triage/pick/record/retro) | `$CONTROLLER_ID` (session) | Anthropic Opus/Fable preference order; `codex:gpt-5.6-sol` subscription fallback when all Anthropic probes are unavailable |
| Design-doc-creator | **ROTATION** (Mark 2026-07-17; `$MISSION_DESIGNER_MODEL` is the rotation SEED, not a fixed pin) | **ROTATION AMENDED 2026-09-05 (Mark, attended) — now `claude:claude-fable-5-1` → `codex:gpt-6-astra` → `pi:ollama/deepseek-v4-flash:0731-cloud` → repeat. Astra is ADDED as a THIRD entry; it does NOT take fable's slot and fable is NOT demoted to a fallback.** Mark's framing, and the thing to hold onto: *"we had only fable, and now we have another fable-class model as astra ... I would like to vary between them as we do for other models."* Astra is fable-CLASS — a high-thinking authoring lane — and BOTH bill a subscription (Anthropic for fable, the ChatGPT bucket for astra via `codex exec`, verified 2026-09-05 with OPENAI_API_KEY stripped: rc=0, prints `model: gpt-6-astra`, while a nonexistent model 400s). So the rotation now has two fable-class lanes to alternate between plus the flat-rate pi lane, which is what "vary" means here. Nothing is displaced and the driver's `MISSION_DESIGNER_MODEL` seed stays `claude:claude-fable-5-1`. **⚠ ASTRA IS ALSO A QUORUM REVIEWER as of the same day** (it straight-swapped `gpt5-6-sol` out of the design-quorum roster, `cmd/ailang/design_quorum.go`), so **rule (c) below now bites on astra's own turn**: when the rotation hands you astra, astra is a reviewer of the doc it just wrote — the 2026-08-26 self-marking defect exactly, not the weaker vendor-level version. Mark's stated principle is *"ideally no model provider marks its own work"*, so do NOT quietly accept the collision. Until the resolution below is ratified, on astra's turn either author with the NEXT rotation entry and say why in the evidence row, or run that doc's quorum with `--reviewers gpt5-6-sol,gemini-3-1-pro,oc-glm-5-2` (sol substituted back into the OpenAI seat for that doc only) and record the substitution. The clean fix, PROPOSED not ratified: make the OpenAI quorum seat *the OpenAI model that did not author this doc* — astra by default, sol when astra designed it — which keeps three independent vendors AND keeps every reviewer independent of the author. **ROTATION AMENDED 2026-08-28 (Mark, attended; V1 charter D-48) — was `claude:claude-fable-5` → `pi:ollama/deepseek-v4-flash:0731-cloud` → repeat; `pi:ollama/kimi-k3:cloud` REMOVED.** Two grounds, both Mark's: (1) kimi's one real designer run failed structurally — wall_timeout 1802s, 73 tool calls, **0 files written**, fell back to fable — and at ~30 min per attempt it is too slow to keep retrying; (2) quota resilience: the non-Fable lane must be NON-Anthropic, because when the Anthropic bucket dries out an all-Anthropic rotation has no working designer at all (this is also why sonnet was NOT added as a third entry). deepseek-v4-flash: flat-rate, vendor-independent of all three quorum reviewers (DeepSeek vs OpenAI/Google/Z-AI), the most-proven pi lane on the rig (fleet executor fallback; the zero-byte guard fix settled its transport), flash-class fast. The ≥3-evidence-rows rule in Gate 5 step 2 binds the LOOP changing policy on its own, not an attended human ruling — two flagged instances plus Mark's decision close it. Pointer migration: none needed — `mission-*-designer-rotation` files hold a last-used value (`claude:claude-fable-5` everywhere as of this edit), and deepseek is simply the new next entry; a last-used value no longer in the list (e.g. a mission seeded elsewhere) restarts at claude. The 2026-08-26 note below STANDS as history: its structural requirement — a second authoring lane independent of every quorum reviewer — is exactly what the replacement preserves. **ROTATION FIXED 2026-08-26 (Mark, attended) — was `claude:claude-fable-5` → `pi:ollama/kimi-k3:cloud` → repeat.** The old list (`fable-5` → `codex:gpt-5.6-sol` → gemini) had TWO structurally dead entries, documented at length below: gemini cannot author at all (server-side sandbox, edits never reach the worktree) and `codex:gpt-5.6-sol` IS quorum reviewer `gpt5-6-sol`, so a codex-authored doc was judged by its own author. That left ONE usable lane, which is why any doc needing a revision blew the Fable diet BY CONSTRUCTION. `pi:ollama/kimi-k3:cloud` fixes it on all three counts: it authors FILES locally (pi drives the local ollama daemon, so edits land in the worktree — unlike gemini); it is independent of ALL THREE quorum reviewers (`gpt5-6-sol` OpenAI, `gemini-3-1-pro` Google, `oc-glm-5-2` Ollama-Cloud/Z-AI — kimi-k3 is Moonshot); and it is the strongest open-weight model measured externally (88.3 Terminal-Bench 2.1, 81.2 FrontierSWE). Probed rc=0. The two-entry rotation is now genuinely two lanes, so the "fall to the NEXT in rotation" rule finally has somewhere to fall. State: **`~/.ailang/state/mission-${MISSION_NAME}-designer-rotation`** holds the LAST-USED value; pick the next list entry (missing file = start at claude), write back after the designer run. **NAMESPACE THAT PATH — the unnamespaced `mission-designer-rotation` this skill used to prescribe is ONE FILE SHARED BY EVERY MISSION ON THE RIG, so a sibling's designer run silently overwrites yours, and the loop cannot tell a clobbered pointer from its own** (fixed 2026-08-13 V1 iteration 188; two frictions, both first-party). The Repo Profile above says "namespaced state keys (M1) keep two missions on one rig from colliding" — true of the keys M1 actually covered, and **false of this one**, which is why nobody re-checked it. Measured: `~/.ailang/state/` holds `mission-world-designer-rotation` and `mission-motoko-designer-rotation` — namespaced files hand-created by careful sibling controllers — **that this skill's literal path never reads**, beside the unnamespaced file it does. Friction 1: iteration 187 recorded advancing V1's pointer `claude → codex`, and at pick time it read `claude:claude-fable-5` again, mtime **01:10**, while V1 was idle (23:42→01:12) and `mission-world` was mid-iteration with a fable designer — consistent with a sibling write, and certain either way that 187's recorded advance was lost. Friction 2: iteration 188 then had to adjudicate between the file and the log to choose a designer at all, which is a coin-flip no rule covers. Note the failure is SILENT and self-concealing: a clobbered pointer holds a *valid* rotation value, so the only tell is a disagreement between the file and the previous iteration's own record — and the natural reading ("trust the state file") is the wrong one. **Migration is one line and costs nothing**: on first read, if the namespaced file is absent but `~/.ailang/state/mission-designer-rotation` exists, seed from it, then write the namespaced path from then on and never write the unnamespaced one again. Generalises: **any `~/.ailang/state/` key this skill names as a literal is shared by all missions** — audit the whole path list before adding another, rather than one key at a time. Every design passes the quorum regardless of author — record `(designer, quorum outcome)` in the evidence row. A probe-failed designer falls to the NEXT in rotation (not to `$MODEL`), FLAGGED. **⚠ TWO OF THE THREE ROTATION ENTRIES CANNOT SERVE AS A DESIGNER FOR STRUCTURAL REASONS, NOT BECAUSE OF A PROBE — SO "FALL TO THE NEXT IN ROTATION" SILENTLY COLLAPSES ONTO FABLE, AND THEN COLLIDES WITH THE FABLE DIET ONE LINE BELOW** (added 2026-08-22 V1 iteration 251; instance 1 is iteration 228, instance 2 is this iteration, and both ended in the same FLAGGED overspend). The fallback rule immediately above is written for a *probe failure* — a lane that answered rc=1 and may answer rc=0 tomorrow. Neither of the two blockers below is that, which is why re-probing never clears them and why each iteration rediscovers them from scratch: **(a) gemini cannot author at all.** The managed_agents lane runs in a Google-hosted server-side sandbox (`CapRemoteSandbox`), so file edits never touch the local worktree and return only as TEXT — the roles-table lane note and the `PROVIDER=gemini` recipe both say so, but they say it about the EXECUTOR role, and nothing points it at the DESIGNER, whose deliverable is likewise a file. A probe of that lane returns rc=0 and tells you nothing. **(b) `codex:gpt-5.6-sol` is the same model as quorum reviewer `gpt5-6-sol`.** A codex-authored doc is then judged, at Gate 2, by its own author — and on a revision pass that is worse, because the objection being answered is that reviewer's own. Nothing in this file forbids it, because the quorum rule says only *"every design passes the quorum regardless of author"*, which is about coverage rather than independence. So the *usable* authoring rotation on this rig has ONE entry, and the Fable diet permits ONE bounded run per iteration — meaning any doc that blocks at quorum and needs a revision exceeds the diet **by construction**, not by carelessness. Both instances resolved it the same way and it is the right call: **re-quorum independence outranks the diet**, because Fable is a quota bucket (metered $0 either way) whereas a judge marking its own homework corrupts the gate itself. **Rules. (a)** When the rotation's next entry is structurally incapable, say WHICH incapacity in the evidence row (capability vs probe failure vs quota) — they have different resume conditions, and standing rule 8 only classifies the quota one. **(b)** Do not re-probe a capability limit; probing gemini for authoring is rule 3a's vacuous pass wearing a lane's clothes. **(c)** Never route a designer to a model that is also one of this doc's quorum reviewers; if that is the only lane left, the honest move is to say so and FLAG, not to quietly accept the collision. **(d)** This note deliberately does NOT change the rotation, because a routing-policy change needs **≥3 evidence rows** (Gate 5 step 2) and there are two — it records the measurement so the third is recognisable rather than rediscovered. When a third arrives, the fix is the ROTATION (widen it, or split "authoring lanes" from "review lanes"), not the iteration. Mission-independent: every mission on this rig reads the same rotation list and the same reviewer defaults. The tell: you are about to spend a second Fable run and the reason you cannot use the alternative has nothing to do with any probe you ran |
| Sprint-planner | `$MISSION_PLANNER_MODEL` | `codex:gpt-5.6-sol` configured default; effective lane = `derive-planner-lane.sh` output, used VERBATIM; Opus-required/fail-closed routes fall back to `$MISSION_PLANNER_ANTHROPIC_FALLBACK` (`codex:gpt-5.6-sol`) only when the driver proved Anthropic unavailable |
| Sprint-executor | `$MISSION_EXECUTOR_MODEL` | `codex:gpt-5.6-sol`; first fallback `pi:openrouter/deepseek/deepseek-v4-flash-0731:floor`; Opus last |
| Sprint-evaluator | `$MISSION_EVALUATOR_MODEL` | **Sonnet** (default changed fable→sonnet 2026-07-16 iter 38, Mark directive #399: "default … gemini (if able to git clone the codebase etc)? otherwise sonnet-5"; gemini-managed_agents VERIFIED not-viable-today — server-side sandbox sees no worktree + backend timed out; sonnet ≠ opus executor → generator≠judge, and it's Agent-tool-PINNABLE unlike fable) |

**Fable discipline (Mark 2026-07-16, amended iter 38):** Fable now bills at most **ONE** BOUNDED
sub-agent run per iteration — the **designer** (only when a new doc is actually needed). The
evaluator moved OFF Fable to **sonnet** (fable was Agent-tool-unpinnable → it silently re-routed to
sonnet every iteration anyway: iters 31/36; and it fires EVERY iteration, so it was the residual
Fable drain). Everything long-running or mechanical rides Opus. Do not "upgrade" a role to Fable ad
hoc; that is a routing-policy change requiring the charter's evidence rule. (Resolves the iter-36/37
inconsistency between this clause and the old "evaluator→sonnet unless ≥3 datapoints" rule.)

**⚠ THE DIET'S UNIT IS ONE BOUNDED *DOC*, NOT ONE BOUNDED *RUN* — BECAUSE THE QUORUM PROTOCOL
MANDATES A REVISION ON A BLOCK, SO A ONE-RUN CEILING IS UNSATISFIABLE BY CONSTRUCTION EXACTLY WHEN
THE GATE FIRES** (amended 2026-08-23 V1 iteration 255 at the ≥3-evidence bar iteration 251
pre-registered; instances are iterations 228, 229 and this one, each of which independently reached
the same resolution and each of which recorded it as a VIOLATION). The clause above says "at most
**ONE** BOUNDED sub-agent run per iteration — the **designer**". Gate 2 says a blocked doc gets a
designer revision and then **one** re-quorum. Those two rules are jointly unsatisfiable whenever the
usable authoring rotation has a single entry, which on this rig it does: `codex:gpt-5.6-sol` **is**
one of the two default quorum reviewers (measured — `internal/mission/quorum/call_test.go` resolves
`gpt5-6-sol` and `gemini-3-1-pro`), so routing it makes the doc's author its own judge, and
gemini/managed_agents is read-only under `CapRemoteSandbox` and cannot author a file at all. Neither
is a probe failure and neither clears by re-probing — do not spend a probe on a capability limit.
**So the controller's only compliant options were to violate the diet or to abandon a doc a reviewer
had just told it how to fix**, and three iterations have now picked the former and apologised for it.
An apology repeated three times is a rule that is wrong, not a controller that is careless.
**Rule.** The Fable budget is **one design DOC per iteration**: the initial authoring run plus **at
most one** protocol-mandated revision run. That is a ceiling, not an allowance — a second revision,
or a designer run for a second doc, is still an overspend and still FLAGGED. Say in the evidence row
whether the revision fired and why, so the *rate* of blocked-round-1 docs stays visible; if that rate
is high the problem is the designer directive, not the diet. **And do not read this as widening the
rotation** — it does not. The rotation still has one usable authoring lane, which is a separate,
now-3-instance defect whose fix is to split "authoring lanes" from "review lanes" (or widen the
list) rather than to keep paying for the collision one iteration at a time; that change needs a human,
because it is a routing-policy change on a shared file. Mission-independent: every mission on this rig
reads the same rotation list and the same reviewer defaults. The tell: you are about to abandon or
force-pass a doc whose reviewers gave you a concrete fix, and the only reason is a budget line.

**Spawn pattern (heavy roles) — run the resolver, follow its output VERBATIM.** For each role run
`tools/launchd/resolve-role-spawn.sh <role> [<design-doc>]`. Its output is exactly one line,
`<path> <value> <reason-token>`; do not second-guess it. `agent-tool <alias>` → spawn
`Agent(subagent_type="general-purpose", model="<alias>", …)`. `recipe <provider:model>` → the
Agent tool is NOT a valid path for that role; use the cross-provider recipe below.
`reroute <alias> generator-equals-judge` → spawn the named re-route target and say so in the
Gate-4 row. `refuse …` → that role is a routing **FAILURE**, not a FLAG: record the reason token
VERBATIM and continue the iteration without the role rather than spending un-budgeted opus.
**Every role prompt MUST begin with the line `MISSION-ROLE: <designer|planner|executor|evaluator>`**
— that token is the ONLY input the spawn-pin hook uses to map a spawn to a role, and while
`MISSION_CONTROL_ACTIVE=1` an unlabelled Agent/Task call is DENIED at the tool boundary. A
read-only reality-check needs no token: spawn it as `subagent_type: Explore`, the one
machine-readable exception.

**⚠ THE AGENT TOOL ACCEPTS A `fable` PIN — the pre-2026-08-20 rule saying otherwise was stale and
was silently costing every mission its rotation's Fable designer slot.** Spawn a Fable role with an
explicit `model="fable"` pin; session inheritance still works but is no longer the only route, so a
non-Fable controller must NOT re-route away from a rotation's Fable entry on pinnability grounds.
Two independent first-party readings established it: the Agent tool's `model`
enum in this build lists `sonnet`/`opus`/`haiku`/**`fable`**, and a role spawned with an explicit
`model="fable"` was ACCEPTED and ran to completion — no `InputValidationError`. What is established
is that the pin is **accepted** and the run **completes** — not that it is enforced end-to-end; do
not quote it for the stronger claim, and the Fable diet is unchanged. If a
pin is ever rejected again, re-probe with one bounded spawn and record the reading rather than
restoring the old rule from memory: **a capability claim about the harness is a measurement with a
date on it.** Evidence, scope and the two first-party readings:
[`resources/role-spawn-routing.md`](resources/role-spawn-routing.md) §1.
`provider:model` values (e.g. `codex:gpt-5.6-sol`) instead signal cross-provider
routing via `provider_executor` (fleet Phase C), not the Agent tool.

**Step 1b — derive the effective planner lane (MANDATORY; before ANY planner probe or spawn).**
Run `tools/launchd/derive-planner-lane.sh <the-picked-design-doc>` with the driver-exported
environment intact. Its output is exactly one line, `<lane> <reason-token>`; use that line
VERBATIM. If it begins `opus `, spawn the opus Agent path directly and do **not** perform a codex
probe or spawn for the planner role; copy the reason token VERBATIM into the Gate-4
routing-evidence row. If the driver exported `MISSION_ANTHROPIC_AVAILABLE=0`, an otherwise-Opus
result instead begins with `$MISSION_PLANNER_ANTHROPIC_FALLBACK` (default
`codex:gpt-5.6-sol`) and carries an `anthropic-fallback:*` reason; route it through the ordinary
cross-provider recipe and record the fallback explicitly. Any `codex:*` result enters the codex
planner recipe below. If the script is missing on disk, use the same conditional: Opus when
Anthropic is available, otherwise the configured Anthropic fallback, always **LOUDLY** and with a
missing-script reason in the evidence row. This rule is mission-independent and live wherever this
shared skill is resolved: the step-0 environment pin protects missions configured for opus, and
the missing-script rule protects missions whose checkout has no derivation script.
**⚠ AND WHEN THAT ANSWER IS `fail-closed:*` WHILE THE ROLE CARRIES A `provider:model` PIN, THE
SPAWN-PIN HOOK WILL *DENY* THE OPUS SPAWN THIS STEP JUST TOLD YOU TO MAKE** (three instances:
iterations 327, 328, 329, each burning a spawn on a guaranteed denial). Route straight to the pin
under its own lane recipe, record BOTH answers in the Gate-4 routing-evidence row, and treat a
denial as information rather than as a lane failure — the hook is authoritative because it is the
boundary the spawn crosses, and *"do not second-guess the resolver"* binds only where the hook has
no opinion. Why it fires on nearly every pick, and the durable fix (which is in the TOOL, not here):
[`resources/role-spawn-routing.md`](resources/role-spawn-routing.md) §2.

**Cross-provider spawn recipe (`provider:model`, M1b — currently `codex` only).** When a role's env
value matches `^([a-z_]+):(.+)$`, DO NOT use the Agent tool. Split it (`PROVIDER=${VAL%%:*}`,
`MODEL=${VAL#*:}`) and route:

- **`PROVIDER=codex`** (executor role — the landed M1b lane; codex CLI at `/opt/homebrew/bin/codex`,
  `OPENAI_API_KEY` set):
  1. **Pre-flight probe (token-cheap, ~1 reply-token, do this BEFORE the real directive):** run the
     probe with a bounded deadline (Standing rule 6 — never unbounded), and only proceed if it exits 0:
     ```bash
     deadline=$(( $(date +%s) + 120 ))
     out=$( codex exec --model "$MODEL" 'reply with exactly: ok' < /dev/null 2>&1 & pid=$!
            while kill -0 "$pid" 2>/dev/null; do
              [ "$(date +%s)" -ge "$deadline" ] && { kill "$pid" 2>/dev/null; break; }
              sleep 2; done
            wait "$pid" 2>/dev/null ); rc=$?
     [ "$rc" -eq 0 ] || { echo "codex probe failed — FALL BACK"; }   # → fallback rule below
     ```
     (Live-verified 2026-07-16 with `MODEL=gpt-5.6-sol`: exit 0, replied `ok`. Mirrors the driver's
     own Anthropic probe at `tools/launchd/mission-control.sh:102`.)
  2. **Real executor run** (recipe corrected 2026-07-16 iteration 32 after the FIRST real codex fire
     — the prior form had only ever been verified against the text probe and was underspecified on
     THREE points that all broke a real coding run: sandbox flags, build-cache writability, and the
     30-min cap vs the harness's 10-min foreground `Bash` limit). A real `codex exec` that edits
     files + runs `go build`/`go test` + git needs a WRITE sandbox that also reaches the Go caches
     (outside the worktree), and it CANNOT be run foreground (the wall-clock cap is 30 min but the
     `Bash` tool caps at 10 min). Write the directive to a file (avoid shell-escaping), then run the
     bounded wrapper via **`Bash` with `run_in_background: true`** — it stays bounded by the wrapper's
     own `date +%s` deadline (Standing rule 6) and notifies you on exit:
     ```bash
     # /tmp/codex_run.sh — launch with Bash run_in_background:true (30-min cap > the 10-min fg limit)
     WT=<sprint worktree path>; DIRECTIVE=/tmp/codex_directive.txt
     # ASSERT DELIVERY FIRST (false-green #2 below): an absent/empty directive makes the prompt
     # expand to "", codex asks "What would you like me to work on?" and exits rc=0 — success
     # reported for work never requested. Refuse to spawn instead.
     [ -f "$DIRECTIVE" ] || { echo "FATAL: $DIRECTIVE missing — refusing to spawn a no-op run" >&2; exit 64; }
     sz=$(wc -c < "$DIRECTIVE" | tr -d ' ')
     [ "$sz" -ge 200 ] || { echo "FATAL: $DIRECTIVE only ${sz}B — suspected truncation" >&2; exit 64; }
     PROMPT="$(cat "$DIRECTIVE")"; [ -n "$PROMPT" ] || { echo "FATAL: empty prompt" >&2; exit 64; }
     deadline=$(( $(date +%s) + 1800 ))   # 30-min hard cap
     GOCACHE=$(go env GOCACHE); GOMODCACHE=$(go env GOMODCACHE)
     ( exec codex exec --model "$MODEL" \
         --sandbox workspace-write \
         --add-dir "$GOCACHE" --add-dir "$GOMODCACHE" \
         -C "$WT" -o /tmp/codex_last.txt \
         "$PROMPT" < /dev/null ) > /tmp/codex_out.log 2>&1 &   # exec: the cap's kill reaches codex, not just the subshell
     pid=$!                                                     # < /dev/null: false-green #1 below
     while kill -0 "$pid" 2>/dev/null; do
       [ "$(date +%s)" -ge "$deadline" ] && { kill "$pid" 2>/dev/null; sleep 2; kill -9 "$pid" 2>/dev/null; echo "codex 30-min cap — FLAG"; break; }
       sleep 15; done; wait "$pid" 2>/dev/null; echo "codex rc=$?"
     ```
     **FIVE FALSE-GREENS this recipe used to carry, plus the gate list you write yourself — all
     moved to [`resources/codex-lane-false-greens.md`](resources/codex-lane-false-greens.md)
     2026-09-04, nothing reworded.** Read it IN FULL before writing a directive for any `codex:` or
     `pi:` role, and again before recording a Gate-4 verdict on that role's output. In one line
     each so you know what you are missing if you skip it: (1) stdin was never redirected, so a
     backgrounded run blocks until the cap looking like normal long work; (2) directive delivery was
     never asserted, so an absent file yields rc=0 for work never requested; (3) a gate verdict from
     inside the sandbox is not evidence — it invents failures AND hides real ones; (4) the gate list
     YOU write into the directive is an acceptance list too, and rule 3e(a) does not reach it;
     (5) a destructive WRITE outside the worktree is denied inside the sandbox, so the executor's
     green says nothing about it and the controller's mandatory out-of-sandbox re-run is where the
     damage lands — 92 lines of shared fleet evidence, unrecoverably, at iteration 327.

     **Hygiene, broadcast with it (not a recipe defect):** a shell "is this env var set?" probe
     written `${VAR:+YES}${VAR:-NO}` **prints the variable's value** — World leaked `OPENAI_API_KEY`
     into a transcript this way. Safe form: `[ -n "$VAR" ] && echo SET || echo UNSET`. No preflight
     check in this loop may use the `${VAR:-…}` form on a secret.
     `--sandbox workspace-write` confines codex to the worktree (blocks escape to the main checkout)
     while `--add-dir GOCACHE/GOMODCACHE` lets `go build`/`go test` write their caches; `-o` captures
     codex's final message. **The codex executor CANNOT commit to the worktree branch itself under
     this sandbox** (a linked worktree's `.git` is a file pointing under the main checkout's
     `.git/worktrees/…`, which `workspace-write` excludes — live-observed iter 32: codex finished
     green but its `git commit` was blocked). So: **read the UNCOMMITTED worktree diff** via
     `git -C "$WT" diff` / `git -C "$WT" status` (NOT `git log` — there's no commit yet), verify it,
     then the CONTROLLER finalizes the commit on the branch, crediting the codex executor in the
     message (`Co-Authored-By: codex <model>`). Everything else reuses the existing worktree-read.
     **MULTI-MILESTONE RUNS: the directive must SAY commits are the controller's job, and demand
     per-milestone SNAPSHOTS** (added 2026-08-03 iteration 135; two frictions in one iteration).
     The paragraph above is single-commit-shaped, and a directive written from the sprint plan's
     own language ("commit per milestone" — Standing rule 3) collides with the sandbox limit it
     documents: iter-135's run A hit exactly that, and codex — correctly, honestly — delivered M1
     then STOPPED rather than violate the commit ordering, burning a 30-min slot on one milestone.
     The run-B fix, now the prescription: (a) the directive states NO git write operations at all
     (add/commit/stash/checkout) and that the controller builds one commit per milestone; (b) after
     finishing EACH milestone the executor snapshots every file created-or-modified-so-far into
     `.snap/M<k>/` (cumulative, full post-milestone content — worktree-writable, so the sandbox
     allows it); (c) the controller reconstructs commits by copying snapshots over the tree in
     milestone order, running the relevant test package at EVERY boundary (bisectability), and
     (d) proves the reconstruction faithful by sha256-manifesting the executor's final tree BEFORE
     starting and `shasum -c` after the last commit — byte-identity or the reconstruction is wrong.
     Two milestones that touch the SAME file are exactly why snapshots beat file-lists here.
  2a. **Planner role — parameterize this executor recipe; do not fork it.** Apply every shared
     probe, bounded-background-run, directive-delivery, stdin, sandbox, output-capture, hygiene,
     timeout, and fallback guard above by reference (including the executor recipe's `exit 64`,
     `< /dev/null`, and `run_in_background` guards). There are exactly four planner deltas:
     - **Working directory:** first assert
       `git status --porcelain -- <design-doc>` is empty. From local `HEAD`, create an ephemeral
       detached worktree with `git worktree add --detach`, then pass its path with `-C`. The path
       MUST be a SIBLING OF THIS MISSION'S REPO — DERIVE it, never hardcode it
       (`"$(cd "$REPO/.." && pwd)/.planner-wt-iter<N>"`): this skill is shared by every mission on
       the rig, so an absolute path baked in for one of them is wrong for the others. Worktrees
       under `/tmp` are forbidden — CWD-relative path tests then fail for the LOCATION rather than
       the code, and CI never reproduces that red.
       Never use `-b` or base it on `origin/dev`: a committed-but-unpushed design doc must be
       visible to the planner.
     - **Directive:** use the per-iteration file
       `/tmp/codex_planner_directive_iter<N>.txt`, carrying the executor recipe's identical
       ≥200-byte delivery assertion and closed-stdin behavior on both probe and run by reference.
     - **Sandbox directories and evidence:** keep `--add-dir "$GOCACHE" --add-dir "$GOMODCACHE"`.
       **In-sandbox gate verdicts are NOT evidence**: socket-touching checks
       are `UNINFORMATIVE UNDER SANDBOX`, and the controller re-verifies load-bearing premises
       outside the sandbox before handing the plan to the executor.
     - **Post-run controller steps:** (1) assert both artifacts exist in the worktree and are
       well-formed (`jq -e . sprint_<id>.json`; plan non-empty and names the design doc); (2) reject
       placeholder vacuous-passes (`MILESTONE_ID` or `auto-parse failed`); (3) copy both artifacts
       to their main-checkout paths, refusing to overwrite unexpected existing files; (4) remove
       the planner worktree; (5) run
       `ailang messages import-github --labels bug,feature,ailang-message` outside the sandbox;
       (6) commit with `Co-Authored-By: codex <model>`.
  3. **generator≠judge guard (HARD, constraint #3):** before spawning the evaluator, assert the
     evaluator's PROVIDER ≠ the executor's PROVIDER. If the executor ran on codex, the evaluator MUST
     NOT be a codex `provider:model` — if `$MISSION_EVALUATOR_MODEL` collides, re-route the evaluator
     to a DISTINCT, PINNABLE Anthropic alias (`sonnet` — fable is unpinnable, gemini is not wired) and
     **FLAG** the collision in the Gate-5 report.
  4. **Fallback (never wedge the loop) — follow the RATIFIED CHAIN, never a straight drop to
     `$MODEL`** (ailang#611, Mark-ratified 2026-08-06; the DRIVER half landed `d14f106bb`, and this
     clause is the in-iteration half the issue explicitly required — a codex 1-token probe can
     return **rc=0 on a spent bucket**, so quota exhaustion is sometimes visible only HERE, on the
     real Gate-3 run, after every driver probe passed). If the pre-flight probe fails, or the real
     run errors / hits the cap: read `MISSION_<ROLE>_FALLBACK`, which since 2026-08-26 may be a
     **COMMA-SEPARATED CHAIN walked left to right**, with opus as the implicit tail — so the
     full ladder per role is `codex → ollama-cloud → openrouter-twin → opus`, i.e. flat-rate
     → metered → Anthropic. Take the FIRST entry; if that lane fails, advance to the next
     rather than dropping to opus. Driver defaults:
     executor `pi:ollama/deepseek-v4-flash:0731-cloud,pi:openrouter/deepseek/deepseek-v4-flash-0731`;
     planner `pi:ollama/kimi-k3:cloud,pi:openrouter/moonshotai/kimi-k3`;
     evaluator `pi:ollama/deepseek-v4-pro:0813-cloud,pi:openrouter/deepseek/deepseek-v4-pro-0813`.
     Each OpenRouter rung is the SAME WEIGHTS as the ollama rung before it, so exhausting the
     Ollama Cloud quota — whose denominator is unpublished, hence unpredictable — degrades the
     ROUTE, not the model. The executor's old OpenRouter `:floor` default is RETIRED and hand the
     role to that value under its OWN lane recipe, probe included — a `pi:*` value enters the pi
     recipe below, an alias enters the Agent tool. Only when the chain value is absent, already the
     failed lane, or itself fails does the role fall to `$MODEL` via the Agent tool. FLAG every
     link traversed in Gate-5 and record the ACTUAL final (role, model) in the routing-evidence row
     — same discipline as a quota-limited Anthropic pin below.
- **`PROVIDER=claude`** (added 2026-07-16, Mark — the true-Fable lane): the `claude` CLI takes FULL
  model IDs (`claude -p --model claude-fable-5`), unlike the Agent tool's sonnet|opus|haiku alias
  limit (F1). So a role value like `claude:claude-fable-5` routes around F1 to a REAL Fable run.
  **BILLING GUARD — MANDATORY at every nested `claude` call (added 2026-07-16 evening after a live
  incident):** `~/.zshenv` sources `secrets.env`, so EVERY tool shell re-exports
  `ANTHROPIC_API_KEY` — the driver's top-level strip does NOT survive into your Bash calls. A bare
  nested `claude -p` therefore bills the METERED API (real $), and when the key's monthly cap is
  hit it fails with an "until the 1st" quota error that MASQUERADES as OAuth-Fable exhaustion
  (the 2026-07-16 "Fable quota-exhausted until 2026-08-01" finding was exactly this — OAuth Fable
  was fine the whole time; OAuth buckets reset weekly Mon 07:00, so ANY until-the-1st reset date
  = you are on the API key). Invoke via the wrapper — NEVER bare `claude`:
  `claude-sub -p … --model claude-fable-5 …`
  (`~/.local/bin/claude-sub` = `exec env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN claude "$@"`
  — subscription-or-nothing by construction; guard the CALL-SITE, not just the helper. The ambient
  leak itself is also closed: `~/.zshenv` now unsets the Anthropic keys after sourcing secrets.env,
  so tool shells don't carry them — the wrapper is the belt on top.)
  Same discipline as codex: 1-token probe first (with the same `env -u` strip), run backgrounded
  from the role's working dir with a bounded ≤30-min `date +%s` deadline,
  `--permission-mode bypassPermissions`, fall back to `$MODEL` + FLAG on probe-fail/cap. Primary
  use: the DESIGNER role (deep spec synthesis on Fable — quota-bounded, fires only when a doc is
  created/revised). The evaluator MAY move here too (`claude:claude-fable-5` ≠ opus executor →
  generator≠judge holds) if the sonnet evaluator's verdicts look lenient — that switch needs the
  charter's ≥3-datapoint evidence rule, not vibes. Quota note: a probe-failed Fable (weekly bucket
  gone) falls back gracefully — never wedge on the scarce model.
- **`PROVIDER=gemini`** (added 2026-07-16 iteration 33, M1c — the managed_agents lane): reached via
  `ailang exec gemini "directive"`. **The agentic `gemini` provider routes to the `managed_agents`
  executor** (Vertex AI Managed Agents API via ADC) — the successor to the Gemini CLI retired in
  v0.22.0 (wired this iteration: `resolveAgenticExecutorName` in `cmd/ailang/exec.go`, PR from
  `sprint/m-gemini-exec-lane`; before it, `ailang exec gemini` failed `unknown executor: gemini` —
  the fleet directive's "wiring-only, no new plumbing" claim was REFUTED). Requires **ADC**
  (`gcloud auth application-default print-access-token` must succeed — probe it first; unset ADC →
  fall back to `$MODEL` + FLAG). `--model` selects the Vertex **agent** name (default
  `antigravity-preview-05-2026`), NOT a gemini-model string. Same probe/cap/fallback discipline as
  codex: ADC-gated 1-token probe (`ailang exec gemini "reply with exactly: ok"` under a bounded
  `date +%s` deadline; only proceed on rc=0), the real run backgrounded with a bounded ≤30-min cap,
  fall back to `$MODEL` + FLAG on probe-fail/cap.
  - **CRITICAL — CapRemoteSandbox (role-scope limit):** managed_agents runs the agent in a
    Google-hosted server-side sandbox, so **file edits do NOT touch the local worktree** — they
    return ONLY in the agent's TEXT output. This lane therefore fits **READ-ONLY roles**
    (evaluator / reviewer / quorum-verifier — the item-(c) agentic-verify lane) that read the repo
    and emit a verdict/text. It is **NOT usable for the file-editing EXECUTOR role** without a
    bridge (see the eval harness's `managed_agents_bridge.go`, which parses artifacts back out of
    the text response). Do NOT pin `MISSION_EXECUTOR_MODEL=gemini:…` expecting worktree edits — that
    is a follow-up (bridge work), not this lane. generator≠judge: gemini (Google) is a distinct
    provider from any Anthropic/OpenAI executor, so it is a valid independent evaluator/reviewer.
- **`PROVIDER=pi`** (executor role — added 2026-08-06, Mark: codex-quota offload after the codex
  bucket dried; gate trial = agent smoke 23/23 + the #590 mission replay, 12/13 held-out landed
  tests at $0.076/12.6 min — memory `project_pi_deepseek_flash_mission_trial`). pi CLI at
  `/opt/homebrew/bin/pi` (@mariozechner/pi-coding-agent); model form
  `pi:openrouter/deepseek/deepseek-v4-flash-0731`. The OpenRouter key rides the
  `~/.pi/agent/models.json` custom-provider block, NOT env — headless-safe. The models.yml twin
  is `pi-or-deepseek-v4-flash` (rate card + gate record live there).
  1. **Probe (bounded, ~1 reply-token):** `pi --mode json --no-session --no-tools --model "$MODEL"
     -p 'reply with exactly: ok'` under the same `date +%s` deadline shape as codex; rc=0 or fall
     back. The driver pre-probes `pi:*` pins the same way it probes codex, so an exported pi pin
     has already passed one probe this fire — re-probe anyway (the codex #486 lesson: the exported
     env and the effective lane must both be proven). Live-verified 2026-08-06.
  2. **Real executor run — reuse the codex recipe's guards BY REFERENCE** (per-iteration directive
     file + the `exit 64` delivery asserts, `< /dev/null` stdin, backgrounded bounded 30-min
     `date +%s` cap, output captured, NO git write operations in the directive, per-milestone
     `.snap/M<k>/` snapshots on multi-milestone runs). pi deltas, all four:
     - **Invocation (SANDBOXED since 2026-08-11 — do not drop the two `-e` flags):**
       ```bash
       mkdir -p /tmp/claude   # sandbox-runtime pins TMPDIR here; absent ⇒ `go build` dies
                              # with "creating work dir". Measured, not theoretical.
       cd "$WT" && PI_FENCE_ROOT="$WT" pi --mode json --no-session \
         -e "$REPO/tools/pi-extensions/sandbox/index.ts" \
         -e "$REPO/tools/pi-extensions/worktree-fence.ts" \
         --model "$MODEL" -p "$PROMPT" \
         > /tmp/pi_run_iter<N>.ndjson 2> /tmp/pi_run_iter<N>.stderr
       ```
       `--mode json` is MANDATORY: the NDJSON is both the transcript and the billing record;
       plain print mode loses both. `$REPO` is the mission's checkout — pass an ABSOLUTE path,
       since `-e` resolves relative to the process cwd, which is `$WT`, not the repo.
     - **SANDBOXED (2026-08-11).** pi has no `--sandbox` flag, but it is extensible, and
       containment now runs in TWO layers because neither covers the other:
       | tool | fenced by | mechanism |
       |---|---|---|
       | `bash` | `tools/pi-extensions/sandbox/` | `@anthropic-ai/sandbox-runtime` → Seatbelt |
       | `write`, `edit` | `tools/pi-extensions/worktree-fence.ts` | `tool_call` hook, path allow-list |
       The upstream sandbox extension fences ONLY bash (it replaces the bash tool); `write`/`edit`
       are Node `fs` calls inside the un-sandboxed pi process and bypass it entirely. Policy lives
       at `~/.pi/extensions/sandbox.json`, canonical copy in
       `tools/pi-extensions/sandbox/sandbox.mission.json`. Verified live: a bash write to `$HOME`
       and a read of `secrets.env` both return `Operation not permitted` (exit 1), while
       `go build` returns `BUILD_OK` — the Go caches are in `allowWrite` because a policy of just
       `['.','/tmp']` breaks every build.
       **This does NOT retire the post-hoc check.** Still run
       `git -C <main-checkout> status --short` before any Gate-4 verdict: the fence confines
       WRITES, not reads, and a defence you never verify is one you cannot claim. It also does
       not make executor-reported greens bankable — the controller re-runs the gates
       (generator≠judge). The codex sandbox false-green class (loopback-bind denials) still does
       not apply here: pi-run gate results fail or pass for real.
       **SM.B2a-class work (irreversible publish) stays off this lane** until the fence has run
       clean for N iterations — sandboxing writes is not the same as bounding blast radius on a
       publish, and the World charter's exception was written about the latter.
     - **METERED $ — ledger entry MANDATORY (the one structural difference from codex: OpenRouter
       bills real dollars, not a quota bucket).** After the run, extract spend from the NDJSON and
       post it to the Gate-3 metered ledger before the next metered call:
       `jq -rs 'map(select(.type=="turn_end")|.message.usage) | (map(.input)|add)*0.09 + (map(.output)|add)*0.18 + (map(.cacheRead)|add)*0.018 | ./1e6' /tmp/pi_run_iter<N>.ndjson`
       (deepseek-v4-flash-0731 per-1M rate card: $0.09 in / $0.18 out / $0.018 cache-read — keep
       in sync with models.yml). Reference: #590 replay = $0.076/sprint-execution; the 30-min cap
       bounds a runaway run to well under $0.50, so the $5 iteration ceiling is ~65 such runs deep.
     - **Credit:** the controller finalizes commits with
       `Co-Authored-By: DeepSeek V4 Flash 0731 (pi)`.
     - **`rc=0` FROM pi IS NOT A CLAIM THAT ANY WORK HAPPENED — AND NEITHER IS `stopReason`.
       RUN THE LANE THROUGH `scripts/mission_pi_run.sh` AND READ ITS TYPED VERDICT.**
       ```bash
       scripts/mission_pi_run.sh \
         --model "openrouter/deepseek/deepseek-v4-flash-0731" \
         --directive /tmp/pi_directive_iter<N>.txt \
         --workdir "$WT" \
         --out /tmp/pi_run_iter<N>.ndjson
       # rc 0=ok · 10=empty_worktree · 11=reasoning_stall · 12=stream_dead
       #    13=wall_timeout · 14=launch_failed.  Anything non-zero is a LANE FAILURE,
       #    not a result: fall back and FLAG, never re-prompt in place.
       ```
       **ROOT CAUSE, MEASURED 2026-08-26 FROM THE PROVIDER'S OWN SIDE OF THE WIRE.** Every
       silent pi failure on record has one shape: the model streams ONLY reasoning tokens and
       never emits content or a tool call. In the whole OpenRouter Broadcast corpus for
       08-18..08-22, **3 of 173** generations had no `finish_reason`; **all three** had
       `completion: ""` with `output_tokens == reasoning_tokens`, and the other 170 all carried
       content or `tool_calls`. It is **not deepseek-specific** — the same signature fired on
       `z-ai/glm-5.2` under OpenCode, on a different provider host.
       **The runs did not fail on their own — WE killed them**, and the ceiling that killed them
       was measuring the wrong thing. pi's `message_update` carries the WHOLE accumulated
       message, not a delta (verified in pi 0.73.1, `dist/core/agent-session.js:421-427`), so
       NDJSON bytes grow **quadratically** in emitted tokens: 7,130 reasoning tokens produced
       **330 MB**. Extrapolated to the declared 65,536-token budget that is **~28 GB** — so the
       old "poll the file size, kill at a few hundred MB" guard silently capped the lane at
       roughly **7,000 reasoning tokens**. No prompt change could ever have fixed that, which is
       why iterations 172 and 173 both failed after adding anti-runaway instructions.
       **What the runner does instead:** filters `message_update` out of the banked NDJSON (size
       becomes linear; `message_end` still carries the complete message, so nothing is lost),
       keeps the newest update in a bounded one-record snapshot for forensics, and uses the
       filtered file's mtime as a progress clock — which freezes *precisely* during a
       content-free reasoning turn. It separates `reasoning_stall` (model thinking, emitting
       nothing) from `stream_dead` (upstream host hung — measured live 2026-08-26: a bare-id
       deepseek call hung 90s at HTTP 200 with an empty body, while 14/14 immediate retries
       succeeded across 6 hosts, so `stream_dead` warrants ONE retry before falling back).
       **DO NOT re-add a `stopReason` assertion.** It is now known evadable in BOTH directions —
       `"length"` pre-2026-08-13 and a clean `"stop"` at 625 tokens post-fix — and it fired on
       **0 of 4** real failures. The load-bearing assertion is the worktree diff, which the
       runner makes for you (`worktree_changed_files` in the verdict JSON).
  3. **Fallback:** probe-fail / any non-zero verdict from `mission_pi_run.sh` → the next link in
     `MISSION_<ROLE>_FALLBACK` AFTER the pi entry if one exists, else `opus` via the Agent tool
     ("end of chain", mirroring the driver's pi loop) + FLAG — never re-prompt in place, and never
     loop back to a lane that already failed this iteration (ailang#611 chain rule; see the codex
     recipe's Fallback for the full semantics).
     **ONE exception, and only one:** verdict `stream_dead` (rc 12) is a transient upstream-host
     hang, not a lane failure — retry the run ONCE before falling back, and record both attempts.
     Measured 2026-08-26: one bare-id deepseek call hung 90s at HTTP 200 with an empty body while
     14/14 immediate retries succeeded across 6 different provider hosts. Every other verdict
     falls back on the first occurrence.
     Trial caveat stands (N=1): the replay's single miss was a discretionary refinement
     beyond the plan's letter — this lane wants PRESCRIPTIVE, sprint-plan-shaped directives;
     vague-plan or judgment-heavy work stays on opus until ≥3 datapoints say otherwise (the
     charter's evidence rule, same bar as every routing change).
  4. **PROMOTION RULE (Mark, attended 2026-08-26 — supersedes the `D-WORLD-20` suspension).**
     DeepSeek returns as the **fallback link**, not yet a rotation peer, because the five failures
     on record were all measured through instrumentation now known to be broken — they are not
     evidence about the model. Re-qualify it on runs, not on this fix:
     **after TWO consecutive real sprint executions returning verdict `ok` with a non-empty
     worktree diff, the lane is promoted into the executor rotation** alongside codex, and the
     controller records the promotion in the mission log's routing row. A single non-zero verdict
     between them resets the count to zero. Until promotion it is reached only when codex is dry.
- **Any other `PROVIDER`** (motoko/opencode): NOT wired (motoko needs the GPU `rig.lock`, out of
  scope). Treat as unavailable → fall back to `$MODEL` + FLAG.

If a pinned Anthropic planner model is quota-limited or unavailable/rejected, fall back to
`$MISSION_PLANNER_ANTHROPIC_FALLBACK` (default `codex:gpt-5.6-sol`), not the already-failed
`$MODEL`, and FLAG it. Other pinned roles fall back to their declared role chain and then `$MODEL`;
never silently inherit. If a pinned model is unavailable, always FLAG it in the Gate-5 report —
never wedge the loop on a role-model outage. **EXCEPTION — the
evaluator role never falls back to bare `$MODEL`** (alias-lane generator≠judge guard, added
iteration 31 after F1): before spawning the evaluator, compare its RESOLVED model (post-fallback)
against the model the executor ACTUALLY ran on. If they are equal — e.g. opus-first session, fable
evaluator pin rejected, `$MODEL`=opus == opus executor — re-route the evaluator to a distinct
pinnable alias (`sonnet`) and FLAG it. A degraded-but-independent judge beats a same-model judge.
**Gate 4 MUST
record the ACTUAL (role, model) used** in the routing-evidence row; a role that ran on the session
model instead of its pin is a regression to surface, not bury (observability is the enforcement
backstop until a Go orchestrator hard-pins it). Deterministic mechanical work (doc moves, regen) =
Sonnet, inline, is fine.

- No design doc yet → **design-doc-creator** on the ROTATION designer (see the roles table: next
  entry after `~/.ailang/state/mission-${MISSION_NAME}-designer-rotation` — NAMESPACED, see the roles table; claude via `claude-sub`, codex via the
  executor recipe carrying the design-doc-creator directive) — spawned pinned/bounded, never inline
  (its hard gates apply: live `ailang check` verification, Conflict Surface for
  parser/types/codegen). **But first
  `grep -ri "<item-id>" design_docs/` — a NEW-DOC queue tag is a claim, not a fact** (added
  2026-07-14 iteration 26; 2 of 2 recent NEW-DOC tags were wrong: m-lambda-open-record-pattern
  had a full doc at planned/v0_29_0 since May [iter 25], m-xmod-alias-poly likewise [iter 26] —
  both times the grep found it in seconds and saved a redundant design-doc-creator run).
  **THE DESIGNER DIRECTIVE MUST DEMAND A VERIFICATION ROW PER CODEBASE CLAIM — a cross-provider
  designer CANNOT READ THIS REPO'S SKILLS, so any gate you leave implicit does not exist for it**
  (added 2026-07-31 iteration 126; two instances in ONE doc, and they cost both quorum rounds).
  Gate 2's rules 3a/3b are written for the *controller's* instruments; nothing was aimed at the
  *designer*, and the designer is the role that writes the most load-bearing "the codebase
  currently does X" sentences. Iteration 126's doc was BLOCKED twice, and both blocks were the
  same defect wearing different clothes: R1 asserted "lowered Core metadata can enumerate every
  requires clause" and R2 asserted "the CLI defines `workspaceRoot` and passes it into test
  configuration". Neither carried a command. The controller measured both — the first came back
  **better** than assumed (repeated `requires` blocks are impossible by construction, which SHRANK
  scope) and the second came back **false** (zero matches, known-positive control firing, the
  field had to be designed in and the Conflict Surface widened). A quorum round costs real money
  and a designer re-spawn costs real wall-clock, so this is not a style note. Concretely:
  **(a)** the spawn directive states that every sentence claiming the codebase currently does X
  needs a Verification Log row with the command AND its observed output — and that an empty or
  negative result is a CLAIM, not a fact, so it needs a known-positive control in the same call
  (rule 3a, restated *to the designer* rather than assumed);
  **(b)** when a quorum objection is "unverified premise", the controller RUNS the check itself
  before routing the revision, and hands the designer the measurement rather than the objection —
  otherwise the designer re-asserts and you buy a third round;
  **(c)** if two rounds block on this same class, name the PATTERN in the revision directive, not
  just the two fixes. Iteration 126 did exactly that on round 3 and got 21 verification rows.
  This applies to EVERY provider lane, but it is load-bearing for `codex:`/`gemini:` designers,
  which never see `design-doc-creator/SKILL.md` at all — for them the directive IS the gate.
- Design doc but no plan → **sprint-planner** as a `$MISSION_PLANNER_MODEL`-pinned Agent sub-agent
  → sprint JSON + handoff.
- Plan exists → **sprint-executor** as a `$MISSION_EXECUTOR_MODEL`-pinned Agent sub-agent, in an
  isolated worktree (coordinator-managed or `git worktree add` — NEVER the shared main tree;
  concurrent agents stomp uncommitted work).
  Immediately before an `origin/dev`-based worktree is created, measure the shared ref again and
  use that full SHA as the worktree base. A sibling worktree's fetch can move this ref silently;
  [`resources/ref-drift.md`](ref-drift.md) records the two measured instances and the complete
  disagreement protocol.

  ```bash
  while :; do
    base=$(bash tools/launchd/mission-base.sh snap) || exit 2 # full SHA<TAB>read-time, no fetch
    newsha=${base%%$'\t'*}
    oldsha=$(bash tools/launchd/mission-base.sh last gate1) || {
      echo "no base recorded — abort, Gate 1 did not stamp" >&2
      exit 2
    }
    [ "$newsha" != "$oldsha" ] || break
    if bash tools/launchd/mission-base.sh drift gate1; then
      continue # ref changed during comparison; restart Gate 3 from a fresh paired read
    else
      drift_rc=$?
      [ "$drift_rc" -eq 1 ] || exit "$drift_rc"
      echo "DRIFT: shared-clone base moved after Gate 1; re-run Gate 3 against $newsha"
      break
    fi
  done
  echo "Worktree provenance: base=$base"
  git worktree add -b "$BRANCH" "$WT" "$newsha"   # submit in background as required below
  ```

  The comparison re-reads once through `drift gate1` and classifies disagreement as DRIFT, not an
  operator error. Re-run the affected Gate-3 checks against `$newsha`; a benign advance does not
  abort, but park when the move invalidates reviewed worktree/provenance integrity. Carry
  `base=$base` into the iteration's provenance and later Routing-evidence row. Never substitute
  `origin/dev` back into the `git worktree add` command after taking this reading.

  **NEVER PLACE A WORKTREE UNDER `/tmp` — the suite goes red for the LOCATION, not the code**
  (added 2026-08-03 iteration 133, executing the remedy iteration 127 pre-committed to on a second
  instance: *"If a second iteration hits it, the fix is to standardise the worktree location off
  `/tmp`."* It did, so here it is). A `/tmp`-rooted checkout fails tests that resolve paths against
  the CWD, because the CWD is itself temp-shaped — and the resulting red is one **CI will never
  reproduce**, so it reads as a regression the sprint caused. Two tests measured failing this way,
  iteration 133, on an otherwise-clean `origin/dev`: `TestIsTempPath` (`internal/loader`, 4
  subtests — `IsTempPath("./src/foo.ail")` returns **true** from `/tmp`) and
  `TestSolve_HardTimeout_FakeSolverIgnoringT` (`internal/smt`, a `TMPDIR` child-pid path); iter-127
  had only the first. Non-vacuous, and this is the control that matters: the identical two tests
  from a non-`/tmp` checkout are **rc=0**, from `/tmp` **rc=1** — the only variable is location.
  Use the established convention — a sibling of the repo, e.g.
  `/Users/…/dev/sunholo-data/.wt-iter<N>` (as `.wt-iter117`/`.wt-iter121`/`.wt-iter133` did) —
  and apply it to **throwaway probe worktrees too**, not just the sprint worktree: iteration 133
  put its sprint worktree in the right place and still bought the false red twice, from two
  `/tmp` scratch trees created to establish a baseline. The generalisable point is the one rule 3c
  already makes about services, aimed at the filesystem: **the location you run a check FROM is
  part of the instrument**, so a red that moves when you move the tree is a fact about the tree.
  **AND CREATING THE WORKTREE IS ITSELF A LONG OPERATION THAT THE TOOL LAYER WILL KILL — A
  HALF-BUILT WORKTREE THEN REPORTS ITS WHOLE TREE AS DELETED, WHICH READS AS CATASTROPHE RATHER THAN
  AS AN UNFINISHED INSTRUMENT** (added 2026-08-20 V1 iteration 234; two first-party frictions in one
  iteration). Every worktree rule above is about WHERE to put it. None is about the fact that making
  one is a full checkout — **23,796 files** in this repo, measured at ~2 minutes — which exceeds the
  foreground `Bash` limit, so the natural `git worktree add …` call is **killed mid-checkout**.
  Friction 1: the foreground call timed out having already created the **branch** but no directory,
  so the retry failed on an existing branch and needed `git worktree prune` + `git branch -D` first.
  Friction 2 is the dangerous one. An interrupted `worktree add` leaves an index that stages
  **every file as deleted** while the files sit on disk — `git status --porcelain | wc -l` returned
  **23,835** on a tree with nothing wrong but incompleteness — plus a live `index.lock`. That count
  is rule 3a's trap in its most alarming form: it is not a measurement of dirtiness, it is the
  instrument still being built, and it looks exactly like a tree someone has wrecked.
  **The recovery instinct is what actually breaks something.** `index.lock` invites you to declare
  the lock stale and remove it. Iteration 234 ran `pgrep`, which printed the owning
  `git worktree add` **alive** with its PID, wrote "owner process confirmed dead", and removed the
  lock anyway — killing a checkout that was 54% done and would have finished on its own. The control
  fired and was read backwards, which is worse than not running it: no repo damage here (the merged
  work and the main checkout both verified intact afterwards), but only because the casualty was a
  throwaway tree. Rules: **(a)** always create a worktree with `run_in_background: true`, and poll
  until the **process exits** — `pgrep -f <worktree-name>` — before reading, staging or committing
  anything in it; a directory that exists is not a checkout that finished. **(b)** Treat a giant
  all-deleted `git status` in a fresh worktree as *incomplete*, never as dirty, and never "fix" it
  with `git reset`/`checkout` while the creating process may still be running. **(c)** An
  `index.lock` is a claim that a process holds it — run `pgrep` and **believe the output**; if a PID
  comes back, the lock is not stale and the only correct action is to wait. **(d)** If a creation
  really was killed, clean up with `git worktree prune` and `git branch -D <branch>` and start
  again, rather than repairing the corpse. Mission-independent: the file count is per-repo, the
  trap is not. The tell: you are about to act on a `git status` from a worktree you created less
  than a couple of minutes ago, or you are about to delete a lock file you have not proven is
  unowned.
- Execution complete → **sprint-evaluator** as a `$MISSION_EVALUATOR_MODEL`-pinned Agent sub-agent
  (distinct from the executor model → generator≠judge). Max 3 rounds; on round-3 fail →
  `needs-human-review`, park, message controlplane.
  **GIVE THE JUDGE ITS OWN WORKTREE — A GOOD EVALUATOR MUTATES SOURCE, AND EVERY RULE IN THIS SKILL
  TELLS IT TO** (added 2026-08-14 V1 iteration 199; instance 1 was iteration 198, instance 2 is this
  iteration). The isolation rule above is written for the EXECUTOR and stops there, so the evaluator
  inherits whatever directory the controller names — normally the sprint worktree the controller is
  still verifying in. That was harmless while judges only read. It is not harmless now: rules 3h(c),
  3i and 3j all instruct the judge to re-run named mutations, so a *well-executed* evaluation
  necessarily edits files, rebuilds binaries and restores them — concurrently with the controller's
  own gate runs against the same tree. The two failure modes are opposite and both bad. **(a) The
  controller misreads the judge.** Iteration 198's evaluator mutated `chains_tree.go` mid-run and the
  controller's gate surfaced a transient FAIL it nearly attributed to the code — rule 3d exactly, with
  the co-occurrence supplied by a *teammate* rather than by chance. **(b) The judge destroys the
  work.** Sprint output is uncommitted by construction, so one `git checkout --` in the judge's
  restore step deletes the milestone; iteration 199's judge restored by `cp` and came to no harm,
  which is luck, not design. Note neither instance produced a wrong VERDICT, which is why this
  survived two iterations: the score was right both times and the tree was the casualty.
  **Rule.** Create a second worktree for the evaluator — same convention as the sprint one (a sibling
  of the repo, **never** `/tmp`) — from the sprint branch's commit, and name THAT path in its
  directive. Where a shared tree is genuinely unavoidable, say so in the directive, forbid mutation
  outright, and treat the evaluation as narrowed accordingly (rule 3b(ii)) — a judge that could not
  run the mutations has not verified them. And while any judge is running, never `git add -A`: stage
  named files, because the tree is not yours alone. Mission-independent; under `ailang-code` the same
  hazard is a judge re-running `ailang check` against a tree someone else is editing.

  **AND ON ROUND 2+ THE DIRECTIVE YOU HAND THE NEXT JUDGE IS ITSELF AN INSTRUMENT, DESCRIBING A TREE
  THAT NO LONGER EXISTS — EVERY MEASUREMENT AND EVERY HUNK LIST IN IT WAS TAKEN BEFORE YOU FIXED THE
  THING IT IS ABOUT** (added 2026-09-01 V1 iteration 316; two first-party frictions in one iteration,
  both caught by the judge rather than by the controller who wrote them). The three-round evaluator
  loop is well covered above: independent worktree, generator≠judge, max 3 rounds, park on a
  round-3 fail. All of it is about the JUDGE. Nothing is about the thing the controller writes
  *between* rounds — and that artifact is load-bearing in a way a first-round directive is not,
  because a fresh judge has no memory of round 1 and takes the summary as its starting map. Note the
  asymmetry that makes this durable: the round-1 directive is written *about* a tree you are looking
  at, while the round-N directive is written *about a tree you have just changed*, and the natural
  way to write it is to copy forward the measurements you took while diagnosing.
  Measured here, both by the round-N judge. **(a) An omitted hunk.** The round-2 summary listed five
  changes and left out a 3-line production hunk the controller had itself shipped in round 1; the
  judge found it only by re-walking the full `git diff` as instructed, and correctly filed the
  omission as a process gap even though the hunk was fine. **(b) A stale red set.** The round-3
  directive quoted a mutation's red set as two tests and "nothing else". The judge measured three.
  Re-measuring on the current tree showed the judge was right and the controller's number was
  **STALE, not wrong** — the set had grown because the round-2 fix ADDED a test after the reading was
  taken. That is rule 3b(v)(b) with the usual roles reversed: not a value transcribed from someone
  else's document, but the controller's own correct measurement, forwarded past its expiry.
  **Rules. (a)** Before re-spawning, RE-RUN every measurement the new directive quotes, on the tree
  the judge will actually see — mutation red sets, gate rc/`--- PASS:` counts, greps and their
  controls. A measurement's validity is bounded by the tree it was taken on, and between rounds the
  tree is exactly what changed. **(b)** Derive the "what changed" list from `git diff`, never from
  memory of what you edited; state it as an exhaustive list and invite the judge to name anything you
  missed — that framing is what converted (a) from a silent omission into a finding. **(c)** Where a
  measurement genuinely cannot be re-taken, date it and name the tree
  ("red set measured at `<sha>`, before the round-2 arm was added"), so a discrepancy reads as
  staleness rather than as an error to argue about. **(d)** A judge that contradicts a number in your
  directive is the loop WORKING — re-measure before replying, and record the correction in Ruled out
  (Gate 2 rule (d), aimed at the controller's own arithmetic). **(e)** Carry forward the round-N−1
  verdict and its findings by NAME, so the new judge can adjudicate each as closed or not rather than
  re-deriving them; a finding silently dropped between rounds looks identical to one that was fixed.
  Mission-independent — every mission on this rig runs the same multi-round evaluator — and the
  generalisation is this file's own recurring shape aimed at its own hands: **a remedy is an
  instrument too, so the document you write to fix a round is subject to every rule you apply to the
  round.** The tell: you are writing "what changed since round 1" and the numbers in it were produced
  by commands you ran before you made those changes. **And in a multi-milestone sprint, non-vacuity is measured against the MILESTONE's own production diff, never the sprint's — a named acceptance test whose mutation an EARLIER milestone already defends passes with the milestone under review entirely reverted, and every instrument in this protocol reports it green: rule 3o, added 2026-09-06 after V1's `m-compile-cache-unverified-artifacts` M4 headline test did exactly that. Put it in the evaluator directive; a judge handed the whole sprint's diff will measure against the whole sprint's diff.** **And in a multi-milestone sprint, non-vacuity is measured against the MILESTONE's own production diff, never the sprint's — a named acceptance test whose mutation an EARLIER milestone already defends passes with the milestone under review entirely reverted, and every instrument in this protocol reports it green: rule 3o, added 2026-09-06 after V1's `m-compile-cache-unverified-artifacts` M4 headline test did exactly that. Put it in the evaluator directive; a judge handed the whole sprint's diff will measure against the whole sprint's diff.**

**METERED-SPEND LEDGER (Mark 2026-07-18 — "make sure costs don't go crazy"):** keep a running
per-iteration tally of METERED dollars (every codex run's reported cost, every managed_agents
`CostUSD`, every quorum reviewer bill — subscription/quota-bucket spend does NOT count). BEFORE
each metered call: if `tally + estimated-cost > $MISSION_METERED_BUDGET_USD` (default $5), do NOT
make the call — fall back to a quota-bucket lane if the role allows, else park the step, FLAG the
ceiling hit in Gate 4/5. Existing per-call caps stay (quorum $0.10/reviewer; managed_agents
post-hoc budget flag; codex mid-stream CostBudget). Cost hygiene for managed_agents specifically
(live-measured 2026-07-18, `TestLiveEnvironmentReuseEconomics`): a TIGHT directive ("run exactly
these commands, do not explore") is worth ~12× vs exploratory ($0.07 vs $0.87); ENVIRONMENT REUSE
(persist `env_<id>`, never re-clone per round) saves a further ~42%. Both are MANDATORY for
gemini escalation runs. Record the final tally as a `metered=$X.XX` field in the evidence row.

**POST THE ITERATION CHAIN (M-MISSION-COST-CHAINS M2 — additive, fail-soft):** after writing the
evidence row (single source of truth → its projection), post ONE chain per iteration so the loop's
spend shows up in `ailang chains`. Build an `IterationPost` JSON from what actually ran and pipe it
to the bounded, LOUD, fail-soft Go subcommand — NEVER inline shell spooling:

```bash
# stages: metered lanes carry $ + model + tokens; quota lanes carry quota_bucket
# (fable|opus|sonnet) and ZERO tokens/cost (subscription burn is bucket-visible, not dollar-faked).
# EVERY stage carries "status" — its REAL outcome (completed|failed|running|awaiting_approval).
cat <<JSON | ailang chains post-iteration || true   # `|| true`: telemetry NEVER blocks the loop
{
  "source": "mission:${MISSION_NAME:-v1}/iter-${ITER}",
  "stages": [
    {"role":"executor","provider":"codex","model":"<model>","cost_usd":<metered $>,"tokens_in":<n>,"tokens_out":<n>,"status":"completed"},
    {"role":"controller","quota_bucket":"opus","status":"completed"},
    {"role":"evaluator","quota_bucket":"sonnet","status":"failed"}
  ]
}
JSON
```

**TOKENS AND STATUS ARE YOURS TO SUPPLY, and both were missing** (M-MISSION-LOOP-UNIFIED-TELEMETRY
M2, 2026-08-13). Measured on `mission:v1/iter-190`: four stages across three providers, all reading
`pending`, the two OpenRouter quorum stages recording **$0.0570/$0.0507 at ZERO tokens**, and a
chain total of **$0.0000** against $0.1077 actually spent. The writer forwards whatever it is
handed — those zeros came from this skill. So:

- **Tokens**: every metered stage posts `tokens_in`/`tokens_out` from the provider's own usage
  report (quorum reviewers included — a reviewer bill without tokens is what produced iter-190).
  Quota lanes still post zero, as they always have.
- **awaiting_approval also posts to the DECISION SPINE** (M-PIPELINE-RECONCILIATION M6, D4,
  ratified 2026-08-26). Any stage you record as `awaiting_approval` — a design frozen pending
  ratification, an executor result parked for a human — ALSO sends one message to the `approvals`
  inbox, which is the single "waiting on Mark" view (it reaches Discord as "🔔 Approval needed",
  and `ailang coordinator pending` unions it):

  ```bash
  ailang messages send approvals \
    "<one-paragraph: what is waiting, what deciding it unblocks, where the artifact lives>" \
    --title "Approval Required: <mission>/iter-<N> <stage role>: <short subject>" \
    --type approval_request --from "mission-${MISSION_NAME:-v1}" || true
  ```

  When the decision lands (directive, ratification, or explicit skip), **ack that message** —
  Gate 0's inbox triage treats a stale unread approval row as noise you created. Do NOT post
  quorum-internal pauses here; the spine is for decisions a HUMAN must make, not for the loop's
  own machinery.

- **Status**: post what actually happened. `ailang chains post-iteration` now prints a stderr
  notice naming how many stages posted no status and will read `pending` forever.
  **Do NOT blanket-post `completed`** — a stage that failed must be posted `failed`. Marking
  everything `completed` satisfies "nothing is pending" and hides exactly the failures this record
  exists to surface; the CLI has a regression test whose entire job is to block that shortcut.
  If a stage is genuinely mid-flight when you post, `running` is the honest answer.
- **Omitting `status` is still accepted** (an older payload must not break the loop) — it leaves
  that stage `pending`, i.e. the pre-v0.33.2 behaviour.

The subcommand: (a) flushes any previously-spooled iterations first; (b) writes the chain +
per-stage cost/tokens/model (metered) or quota bucket (encoded in `agent_id` as `<role>
(quota:<bucket>)` — NO schema change), applies each stage's status, and **rolls the stage
costs/tokens up into the chain total**; when every stage reports a terminal status the chain itself
closes as `completed`, or as `failed` if any stage failed; (c) if the observatory is unreachable,
buffers to a bounded JSONL spool (≤100 entries / 1 MiB, drop-oldest, stderr-LOUD) the next
iteration flushes. It exits 0 even on telemetry failure — a broken tracker must never wedge the
loop. Review the fleet's spend later with `ailang chains stats --by-mission` (M3).

**DUAL-WRITE to a remote observatory** (M-MISSION-LOOP-UNIFIED-TELEMETRY M3): a node that sets
`AILANG_CHAINS_CLOUD=gcp` (or passes `--cloud gcp`) writes the iteration to its local store AND to
the remote one, under the SAME chain and stage ids, so spans carrying those ids join either copy.
Nothing about it is specific to this machine — the node is a parameter, and with the variable unset
behaviour is exactly what it was. It does NOT touch `AILANG_STORAGE`, so this node's coordinator
and messaging stores stay where they are. Each target keeps its OWN bounded spool, so a cloud
outage cannot evict local posts, and an unreachable cloud is a stderr warning plus a buffered
retry — never a blocked iteration.

**GPU rule (two-tier)**: default iterations never touch `rig.lock` — it is a GPU mutex only.
If (and only if) a step drives ollama/local models: `source tools/launchd/rig-lock.sh &&
rig_lock_acquire wait` around THAT STEP, release immediately after. Ask explicitly at routing
time: "does this step touch the GPU?" — never let a test reach it by accident.

**Multi-week strategic items**: do not execute — the iteration's deliverable is DECOMPOSITION
into sprint-sized design docs (≤3–4 days each), queued individually.
