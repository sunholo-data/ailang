# Claude Instructions for AILANG Development

## North Star — [design_docs/PROGRAM.md](design_docs/PROGRAM.md)

This work runs as one program: a **self-specializing harness**. The motoko core is **minimal and frozen**;
every improvement is routed to exactly one lane — an **AILANG fix**, a **motoko extension**, or (rarely) a
**core-floor fix** that re-freezes. We discover friction by running benchmarks through the data-led loop,
then route it. **Default bias: if it can be an extension, it is an extension — not a core change.** The
living roadmap (benchmark ladder · extension catalog · AILANG-fix backlog) lives in PROGRAM.md.

## Session start

The SessionStart hook injects unread agent messages automatically. If there are any: summarize to the
user, ask what to do, then `ailang messages ack --all`. If a task fails, `ailang messages unack MSG_ID`.
Full command reference: `ailang messages --help`.

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

### 1. ALWAYS USE EXISTING TOOLS FIRST

Before writing ANY new script or code: check `make help`, check `tools/`, then
`grep -r "function_name" internal/`.

**The `ailang` CLI exists to make YOUR life easier.** Use `ailang chains`, `ailang messages`,
`ailang eval-*`, and `ailang dashboard` instead of raw SQLite queries or ad-hoc scripts.

**Look the question up, not the tool.** Knowing a tool exists is not the same as knowing
which question it answers — the gap between those two is where we rebuild worse versions of
what we already have.

| Question | Instrument |
|----------|-----------|
| **What was the agent actually told?** | `ailang chains chat <chain-id> --stage N`. The banked `agent_transcript` holds tool **calls only** — no results. The tool RESULTS live in motoko's session JSONL (`<motoko-repo>/.motoko/logfile/session_*.jsonl`) and in the observatory via `ailang chains import-motoko <file>`. |
| How hard is a benchmark, really? | `ailang eval-elo <dir> --json` — fits difficulty separately from model strength, so it survives baseline shifts that raw pass rates do not. |
| Did an A/B actually measure anything? | `ailang eval-paired <on> <off>` — reports discordant pairs and a headroom warning, not just aggregate rates. |
| Why did a run fail? | `error_category` on the banked row FIRST. `api_error` is the catch-all meaning "cause unknown", not "model failed". |

Asking "what did the agent DO?" and inferring the rest is how `ailang fmt` spent two weeks telling every
eval model its correct code was non-canonical — the message was in the session JSONL the whole time.

### 2. NO SILENT FALLBACKS - FAIL LOUDLY

If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return zero, null, or error instead.

**Apply to:** Pricing/costs, model configs, required env vars, data validation.
**Fallbacks OK for:** UI defaults, optional features, caching.

### 3. SYSTEMIC FIXES - AUDIT BEFORE PATCHING

Before fixing a bug, ALWAYS ask: "Is this part of a larger pattern?" Search for similar code paths. Design ONE unified fix instead of patching case-by-case.

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
