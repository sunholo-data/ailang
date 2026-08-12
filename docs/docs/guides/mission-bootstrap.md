---
title: Bootstrapping a Mission Loop
description: Point AILANG's autonomous mission loop at another repo — driver, skill, charter, and a token-free dry-run acceptance
---

# Bootstrapping an autonomous mission loop

AILANG ships with a **mission loop**: a scheduled outer loop that picks the top item off a written
backlog, routes it through a design → plan → execute → evaluate inner loop of specialised agents,
records what happened, and reports to a human — then does it again, unattended, on a cadence. The
loop that builds AILANG itself (the "V1 mission") runs this way. This guide shows how to point a
*second* loop at a *different* repository, so one machine can advance several missions concurrently
without them colliding.

> **Who this is for.** You have the `ailang` binary installed, a GitHub repo you want worked
> autonomously, and a machine that stays on. You do **not** need to have read the AILANG source. The
> mission loop is a general harness; the language it happens to build is incidental to *this* recipe.

## How the loop is put together

Three layers, each replaceable per mission without touching the others:

| Layer | What it is | Per-mission? |
|---|---|---|
| **Driver** | `tools/launchd/mission-control.sh` — the shell script launchd fires. Handles the billing guard, model selection, overlap/pidfile guard, stall watchdog, and invokes one agent iteration. | Shared script, parameterised by env |
| **Skill** | `.claude/skills/mission-control/SKILL.md` — the agent's gate-by-gate instructions for one iteration (observe → pick → route → record → retro). | **One skill, never forked** |
| **Charter** | `design_docs/<name>-mission.md` — the mission's bar, queue, guardrails, and **Repo Profile**. The loop's memory + backlog. | One per mission |

The single skill is the point: every improvement the loop discovers about *how to run a mission*
(a sharper gate, a new guardrail) is edited into the one skill and benefits **all** missions. Fork
the skill per mission and you lose that. What varies per mission is data — the charter and a small
env profile — not logic.

## Prerequisites (hard preconditions)

Do these once on the machine that will host the loop:

1. **The target repo has CI workflows.** Gate 3b ("an item is not landed until remote CI passes")
   is meaningless without them. Note the exact workflow *names* — you will list them in the charter's
   Repo Profile.
2. **`gh` authenticated as the automation account** with push rights to the repo:
   `gh auth status` should show your bot account (for AILANG's own missions that is
   `sunholo-voight-kampff`). The loop commits and comments as this identity.
3. **`ailang` installed** and on `PATH` — `ailang install` from a release, or `make install` from
   source. The fleet substrate the loop uses (design-quorum, exec lanes, agent messaging,
   coordinator) ships *inside* the binary, so it ports with the install; there is no per-repo
   plumbing to stand up.
4. **The billing guard is in place** — API keys stripped from the loop's environment so it runs on a
   subscription, never a metered key. (For AILANG this is a `~/.zshenv` unset plus a `claude-sub`
   wrapper; adapt to your shell. The driver probes auth cheaply and refuses loudly with zero spend if
   the subscription is not reachable.)
5. **Claude Code has been run interactively in the mission's checkout ONCE** — added 2026-08-12 after
   it cost the motoko mission its entire first fire. A checkout Claude Code has never seen has no
   `hasCompletedProjectOnboarding` entry in `~/.claude.json`, so a headless `claude -p` blocks on the
   trust dialog it cannot display. **Every** model in `MISSION_MODEL_PREFS` then burns its full probe
   timeout twice (2 × 120s × 3 models ≈ 12 minutes) and the driver correctly refuses with
   *"NO usable model in prefs"* — a message that reads exactly like a quota outage or an auth
   failure, which is the trap. The captured probe output names the real cause, so **read it** rather
   than the summary line.

   ```bash
   cd <mission-checkout> && claude     # accept the trust dialog once, then quit
   ```

   Measured across three checkouts, including the counterfactual: `ailang` (trust=T, onboarded=T)
   works; `ailang-world` (trust=**F**, onboarded=T) works; `ailang-motoko` failed with **neither**,
   then worked with **trust=T and onboarding still absent**. So **either flag suffices** — the error
   message's own advice is correct, and running `claude` once in the checkout sets what is needed.

   > An earlier draft of this section claimed onboarding was *the* discriminator and the trust flag
   > was not. That was inferred from three config snapshots by picking the variable that correlated,
   > without testing the counterfactual — `ailang-world` licensed *"trust=false can work"*, never
   > *"trust=true alone cannot"*. Kept as a caution: with N=3 configs and no counterfactual, a
   > correlation is not a mechanism.

   Separately, `hasCompletedProjectOnboarding` **is** load-bearing for the #558 launchd-driver pin —
   see PR #667, which gates the pin root on it after finding the pin would otherwise walk every fire
   into this same probe-hang.

   **This bites only missions whose `MISSION_WORKDIR` is a NEW checkout.** A mission anchored on an
   already-used working tree (V1) never sees it, which is why four prior mission bootstraps did not
   surface it.

## Step 1 — Write the charter

Copy the template (it lives in the AILANG checkout, at `design_docs/mission-charter-TEMPLATE.md`)
into the target repo's `design_docs/` and fill it in:

```bash
cp <ailang-checkout>/design_docs/mission-charter-TEMPLATE.md design_docs/<name>-mission.md
touch design_docs/<name>-mission-log.md   # the append-only log the loop writes each iteration
```

Fill every `<PLACEHOLDER>` — and update the template's `PROGRAM.md` / `v1-mission.md` header links to
point at your own repo's equivalents (or drop them). The two load-bearing sections:

- **Repo Profile** — the single source of truth the skill reads: repo slug, mission name (this
  becomes the state namespace), bookkeeping-issue number, the CI workflow names Gate 3b polls, and
  the **verify profile**. Pick the verify profile that matches the repo:
  - `go-compiler` if the repo compiles a Go toolchain (build both binaries, `make test`).
  - `ailang-code` if the repo is AILANG source — the shipped binary is the whole gate:
    `ailang check` / `ailang test` / `ailang ai-check` (the unified check+verify; do not
    reinvent a split gate).
- **The bar** — the concrete, checkable clauses that define "done" for the mission. Number them;
  queue items clause-tag against them.

## Step 2 — Create the bookkeeping issue and seed state

The loop reports every iteration to a GitHub issue (a human reads it by email) and rolls it weekly.
Create it and record its number:

```bash
gh issue create --repo <owner>/<repo> \
  --title "<Name> mission bookkeeping — week of <this Monday>" \
  --body "Bookkeeping thread for the <name> mission loop. Comments from the directive account steer the loop."
# then seed the state file with the issue number it returns:
echo <NNN> > ~/.ailang/state/mission-<name>-gh-issue
```

Because the mission name is anything other than `v1`, the driver automatically namespaces **all**
its state under `~/.ailang/state/mission-<name>-*` (pidfile, kill switch, model override, gh-issue,
watermark). It cannot collide with the V1 loop's state — that isolation is guaranteed by the M1
driver parameterisation, not by convention.

## Step 3 — Write the env profile

One file per mission, sourced by the driver when launchd sets `MISSION_PROFILE=<name>`:

```bash
# ~/.config/ailang/mission-<name>.env
MISSION_NAME=<name>
MISSION_REPO=<owner>/<repo>
MISSION_DOC=design_docs/<name>-mission.md
MISSION_WORKDIR=/absolute/path/to/the/repo/checkout
# Optional per-mission routing overrides (spread quota across providers):
# MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol   # a non-Anthropic executor from day one
```

New missions default their executor to a **non-Anthropic lane** where possible, so a second
concurrent loop does not double the Anthropic burn. The evaluator must differ in provider from the
executor (generator ≠ judge) — the skill enforces this.

## Step 3.5 — Make the shared skills visible to the new checkout

The mission skills live in the AILANG checkout (`.claude/skills/…`) — one copy, never forked.
A session working in *your* repo finds them via **user-level symlinks** (edits made through them
land in the single shared file, so every mission inherits every retro improvement):

```bash
mkdir -p ~/.claude/skills
for s in mission-control design-doc-creator sprint-planner sprint-executor sprint-evaluator agent-inbox use-ailang; do
  ln -s <ailang-checkout>/.claude/skills/$s ~/.claude/skills/$s
done
```

Do NOT copy the skill files into your repo — a copy is a fork, and forks stop learning.

## Step 3.6 — Put the driver in the target repo

The plist you are about to install points at `__WORKDIR__/tools/launchd/mission-control.sh` —
**inside your repo's checkout**, so the driver must exist (and be committed) there:

```bash
mkdir -p tools/launchd
cp <ailang-checkout>/tools/launchd/mission-control.sh tools/launchd/
cp <ailang-checkout>/tools/launchd/mission-template.plist tools/launchd/
git add tools/ && git commit -m "infra: mission-loop driver (from sunholo-data/ailang)" && git push
```

The driver is fully parameterized — committing a copy here is *deployment*, not a fork: mission
behavior comes from the shared skill and your env profile, and driver improvements are synced by
re-copying on upgrade.

## Step 4 — Install the launchd job

**Set the kill switch FIRST.** The template ships `RunAtLoad=true` (so reboots can never
silently kill the cadence) — which means `launchctl bootstrap` **fires an iteration immediately**.
Without the switch, that first fire runs unattended against a not-yet-ratified charter, spending
real tokens. The proven launch sequence (this is how Ailang World launched on 2026-07-23):

```bash
touch ~/.ailang/state/mission-<name>.disabled       # armed-but-silent: every fire exits at Gate 0
sed 's/__NAME__/<name>/g; s#__WORKDIR__#/absolute/path/to/checkout#g' \
  tools/launchd/mission-template.plist > ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist
launchctl bootstrap gui/$UID ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist
```

The switch comes off in Step 6, deliberately — never as a side effect.

**Stagger the `StartInterval` offset** against any other live mission so two loops never fire on top
of each other (they share the rig's quota and would contend for the model). The template sets a
90-minute interval and `RunAtLoad=true` so a reboot can never silently kill the cadence — the kill
switch (`~/.ailang/state/mission-<name>.disabled`) is how you turn it off deliberately.

## Step 5 — Dry-run acceptance (no tokens spent)

Before the first real fire, prove the wiring in isolation. `MISSION_DRY_RUN=1` runs the driver's
setup — profile sourcing, path namespacing, model resolution — and exits **before** spending a
single token:

```bash
MISSION_PROFILE=<name> MISSION_DRY_RUN=1 tools/launchd/mission-control.sh
```

It logs one line: `DRY RUN ok: mission=<name> repo-slug=<owner>/<repo> doc=… pidfile=… roles: …`.
Confirm the mission name, repo slug, doc path, and — critically — that the **pidfile path contains
your mission name** (`mission-<name>.pid`, not `mission-control.pid`). That distinct pidfile is the
proof the new loop cannot disturb the V1 loop. Run the V1 dry-run alongside it and check the two
pidfiles differ.

## Step 6 — Iteration 0: ratify the charter

Lift the kill switch and fire the first iteration while the human is present:

```bash
rm ~/.ailang/state/mission-<name>.disabled
launchctl kickstart gui/$UID/dev.ailang.mission-<name>
```

The first *real* iteration is not a sprint — it ratifies the charter itself. Run one iteration
attended, with the human, and put the bar, queue, and guardrails through the design quorum
(`ailang design-quorum`). Only once the bar is agreed does the loop start picking backlog items.
From then on it is autonomous: every subsequent fire is one gate-walk of the inner loop, reported to
your bookkeeping issue.

## Step 7 — Wire the cross-mission channel (and test it)

Missions on the same machine share one message bus (`ailang messages` — a rig-level store, not
per-repo), which gives every loop an **upstream lane** to its siblings: defect reports, language-gap
requests, corroborations. Conventions, learned from the first live use (Ailang World → AILANG v1,
2026-07-23, received→verified→fixed→acknowledged in under an hour):

- **Send to `controlplane`** (the inbox every loop's Gate-0 triages) and identify as your mission:
  ```bash
  ailang messages send controlplane "<summary + repro + issue link>" \
    --title "<what and where>" --from "mission-<name>"
  ```
- **Attach a repro.** The receiving loop applies its ghost discipline (live-repro your claim at its
  HEAD) before anything enters its queue — a claim without a repro is a rumor.
- **What your message can and cannot do** (the cross-mission contract, enforced by the shared
  skill): it **never sets the sibling's priorities** — only their human and genuine regressions
  outrank their queue. A verified language-gap request enters their queue as a normal item tagged
  `[<your-mission>-DEMAND]` — and note that a real downstream consumer is the *strongest* demand
  evidence there is. Soundness bugs and crashes you actually hit triage like regressions on their
  side — those can outrank.
- **Expect an acknowledgment** on YOUR bookkeeping issue with the triage verdict (queued / fixed /
  refuted-with-evidence), so your loop can plan around it.
- **Respect ownership**: report defects in *shared* machinery (the skill, the driver, the manual)
  upstream — never edit another mission's files or the shared skill from your loop.

**Smoke-test the channel as part of bootstrap**: send one test message from the new mission and
confirm the sibling's next triage sees it. Delivery is unconditional (same store), so this is
really testing your `--from` naming and their triage classification.

## What ports for free, and what does not

**Ports unchanged** (already repo-agnostic): the directive-author allowlist, quorum-at-pick, the
billing tripwire, the pidfile/overlap guard, the designer-model rotation, weekly issue rotation, and
the entire fleet of exec lanes — they all live in the shared skill and the `ailang` binary.

**Does not port automatically**: the machine-level setup (Steps in Prerequisites) is per-machine, not
per-mission — do it once. And the charter's *content* (bar, queue, guardrails) is yours to write;
the template gives you the format, iteration 0 gives you the agreement.
