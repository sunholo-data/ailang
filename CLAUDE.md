# Claude Instructions for AILANG Development

## North Star — [design_docs/PROGRAM.md](design_docs/PROGRAM.md)

This work runs as one program: a **self-specializing harness**. The motoko core is **minimal and frozen**;
every improvement is routed to exactly one lane — an **AILANG fix**, a **motoko extension**, or (rarely) a
**core-floor fix** that re-freezes. We discover friction by running benchmarks through the data-led loop,
then route it. **Default bias: if it can be an extension, it is an extension — not a core change.** The
living roadmap (benchmark ladder · extension catalog · AILANG-fix backlog) lives in PROGRAM.md.

## Session start — every machine reads the SAME message store

**The canonical inbox is prod Firestore, project `ailang-multivac`.** A bare
`ailang messages list` reads only *this* machine's private SQLite — real user feedback once sat
unread for weeks that way. Every machine doing AILANG work exports:

```bash
export AILANG_MESSAGES_STORE=gcp
export AILANG_MESSAGES_PROJECT=ailang-multivac
```

These are scoped to messaging only (`ailang storage status` must still say `Mode: local`).
**Never export `AILANG_STORAGE` for this** — it moves coordinator and observatory to Firestore
too ([backend.go:83](internal/storage/backend.go#L83)).

Then `ailang messages list --unread` spans all inboxes, and a non-local listing names its store
in the header — **no `store: gcp (...)` header means you are reading local**, usually a stale
binary. Summarize to the user and ask **before** acking. Triage via `list --unread --json`
(full IDs + bodies; the list view truncates IDs, and `messages read` marks read as a side
effect). Ack per message id — `ack --all` also sweeps outbound cross-mission inboxes;
`unack <id>` if a task fails.

The measured traps (stale-binary control, project pinning, `pkg:*` inboxes) and the full
topology: [docs/internal/message-plane-topology.md](docs/internal/message-plane-topology.md).

---

## Critical Principles

### 0. NEVER DESTROY LOCAL WORK WITH GIT OPERATIONS

**NEVER run these commands — they destroy uncommitted work:**
- `git checkout <branch>` when there are uncommitted changes
- `git pull` on a branch with local commits (use `git status` first!)
- `git reset --hard`, `git clean -fd`
- `git stash` followed by branch switching that causes fast-forward

**CORRECT approach:**
1. ALWAYS `git status` first
2. If uncommitted changes exist, **ASK THE USER** how to handle them
3. Work on CURRENT branch — don't force a branch switch
4. NEVER assume it's safe to discard or stash work

### 0.1. VERIFY GITHUB ACCOUNT BEFORE RELEASES/TAGS

The developer uses multiple GitHub accounts. Before ANY release or git push:

```bash
gh auth status                              # Check active account
# This repo needs: sunholo-voight-kampff (Claude Code agent account)
# MarkEdmondson1234 = human developer account
# rw-markedmondson = WRONG (Rockwool project)
gh auth switch --user sunholo-voight-kampff  # Switch if needed
```

### 1. Look the question up, not the tool

Before writing a new script, check `make help`, `tools/`, and the `ailang` CLI (`chains`,
`messages`, `eval-*`, `dashboard`) — raw SQLite queries and ad-hoc scripts rebuild worse
versions of what exists. Knowing a tool exists is not the same as knowing which question it
answers; that gap is where we rebuild things.

| Question | Instrument |
|----------|-----------|
| **What was the agent actually told?** | `ailang chains chat <chain-id> --stage N`. The banked `agent_transcript` holds tool **calls only** — no results. The tool RESULTS live in motoko's session JSONL (`<motoko-repo>/.motoko/logfile/session_*.jsonl`) and in the observatory via `ailang chains import-motoko <file>`. |
| How hard is a benchmark, really? | `ailang eval-elo <dir> --json` — fits difficulty separately from model strength, so it survives baseline shifts that raw pass rates do not. |
| Did an A/B actually measure anything? | `ailang eval-paired <on> <off>` — reports discordant pairs and a headroom warning, not just aggregate rates. |
| **Will a design doc survive independent review?** | `ailang design-quorum <doc.md>` — N off-Anthropic reviewers, reject-by-default (each states a `strongest_objection`), plus your in-session controller verdict (`--controller-verdict`/`--controller-note`); exit 0 = proceed, 3 = blocked; artifacts in `.ailang/state/mission-quorum/`. The optional pre-sprint step documented in the design-doc-creator skill. |
| Why did a run fail? | `error_category` on the banked row FIRST. `api_error` is the catch-all meaning "cause unknown", not "model failed". |
| **What did the PROVIDER actually see and return?** | OpenRouter **Broadcast** traces, in the prod observatory — `curl "https://dashboard.ailang.sunholo.com/api/observatory/spans?limit=1000&start_after=<RFC3339>&start_before=<RFC3339>"`. Each `LLM Generation` span carries ~92 attributes: the **full `rawRequest`** (so you can read the budget and reasoning config that actually went on the wire, not the one we declared), the completion text, the token split incl. `output_tokens.reasoning`, `finish_reason`, the provider host, and a `cancelled` flag. This is a SECOND, independent instrument on every OpenRouter call — use it before theorising from our own logs. |

The provider-side trace settles arguments our own logs cannot (2026-08-26: it proved five
"model returned zero bytes" failures were our own size ceiling, after two prompt fixes had been
aimed at a model problem that did not exist). The OpenRouter account key is inference-only —
`/api/v1/activity` 403s — so go via the observatory, which is indexed by time anyway.

Asking "what did the agent DO?" and inferring the rest is how `ailang fmt` spent two weeks telling
every eval model its correct code was non-canonical — the message was in the session JSONL the
whole time.

### 2. No silent fallbacks — fail loudly

If a fallback value would affect data integrity, business logic, or user decisions (pricing,
model configs, required env vars, validation), return zero/null/error instead. Fallbacks are
fine for UI defaults, optional features, and caching.

### 3. Systemic fixes — audit before patching

Before fixing a bug, ask whether it's part of a larger pattern; search for similar code paths
and design one unified fix instead of patching case-by-case.

### 4. Read the control before refusing on its behalf

Before declining a requested action to protect a safeguard, **check what the safeguard actually
enforces**. If the artifact is identical whether you or the user performs the step, refusing buys
no integrity — it only moves the work back to the user and costs the session.

Measured 2026-09-02: asked to record an attended ruling with `mission_answer.sh`, the agent
refused on the grounds that a loop-authored ruling would be indistinguishable from a human one.
But that script overrides the git identity for *every* caller (`git -c user.name=… -c user.email=…`),
and this machine's git identity is the fleet bot for every session including Mark's — so the two
commits are byte-identical. The guarantee being defended did not exist. Cost: about an hour, and
the decision still had to be made afterwards. Related trap: a rule addressed to *the unattended
loop* (mission-control's SKILL.md) is not automatically a rule about an attended session that
happens to contain an agent.

If the control is genuinely weak, say so and offer to fix it — do not simulate it by refusing.

---

## Project Overview

**AILANG is a deterministic language designed for autonomous AI code synthesis and reasoning.**
File extension: `.ail`. Priorities: machine decidability, semantic transparency, compositional determinism.

Design principles: **explicit effects** (all side effects declared in signatures) · **everything is an
expression** (no statements) · **type safety** (Hindley-Milner + row polymorphism) · **deterministic**
(all non-determinism explicit) · **AI-friendly** (structured execution traces).

For syntax, run `ailang prompt` — never write AILANG from memory, and verify any syntax claim with
`ailang check` before putting it in a prompt or doc.

---

## Reference Documentation

Skills and path-scoped rules (`.claude/rules/`) load themselves when relevant — you don't need to
look them up. These are the docs you would NOT think to look for:

- **[MOTOKO.md](MOTOKO.md)** — which motoko_agent checkout do evals actually use? Read BEFORE
  touching any `~/dev/mk-*` directory or the `motoko` shim.
- **[design_docs/PROGRAM.md](design_docs/PROGRAM.md)** — the living roadmap and routing lanes.
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — layer map and enforced import directions.
- **[docs/LIMITATIONS.md](docs/LIMITATIONS.md)** — what AILANG genuinely cannot do yet.
- **[docs/docs/guides/](docs/docs/guides/)** — development-workflow, coordinator, agent-messaging,
  evaluation, telemetry, debugging, collaboration-hub, database-architecture, lsp.
- **[design_docs/](design_docs/)** · **[examples/](examples/)**
