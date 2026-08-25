# Claude Instructions for AILANG Development

## North Star — [design_docs/PROGRAM.md](design_docs/PROGRAM.md)

This work runs as one program: a **self-specializing harness**. The motoko core is **minimal and frozen**;
every improvement is routed to exactly one lane — an **AILANG fix**, a **motoko extension**, or (rarely) a
**core-floor fix** that re-freezes. We discover friction by running benchmarks through the data-led loop,
then route it. **Default bias: if it can be an extension, it is an extension — not a core change.** The
living roadmap (benchmark ladder · extension catalog · AILANG-fix backlog) lives in PROGRAM.md.

## Session start — messages live in TWO stores and the hook reads only one

The SessionStart hook injects unread **local** messages (SQLite, `~/.ailang/state/collaboration.db`).
That is one node's private inbox. **Everything written from outside this machine — public feedback,
package feedback, coordinator completions, other agents — lands in the canonical cloud store
(prod Firestore, project `ailang-multivac`), and the hook cannot see it.** Check both, every session:

```bash
ailang messages list --unread              # 1. local (what the hook already showed you)

# 2. canonical cloud store. Define this once per shell — do NOT export the vars (see below).
canon() { AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac ailang "$@"; }

canon messages list --inbox public-feedback --unread
canon messages list --limit 500 --json 2>/dev/null \
  | jq -r '.[]|select(.status!="read")|"\(if (.to_inbox//"")=="" then "(EMPTY)" else .to_inbox end)\t\(.id)\t\(.title)"'
```

JSON goes to **stdout**, the stale-binary and OTLP warnings go to stderr — so `2>/dev/null` is safe and
necessary. Then summarize to the user and ask what to do before acking. Ack local with
`ailang messages ack --all`; ack cloud with `canon messages ack <FULL-id>`. If a task fails,
`ailang messages unack MSG_ID`.

> Write the two vars on the command itself (as the function does). In **zsh** an unquoted `$VAR` is not
> word-split, so the tempting `env $CANON ailang ...` passes the whole string as ONE assignment and you get
> `unknown AILANG_STORAGE mode: "gcp AILANG_CLOUD_PROJECT=..."`. Verified 2026-08-25.

**Six traps, each measured 2026-08-25 — do not rediscover these:**

1. **Verify which project you are pointed at before believing any result.** `AILANG_CLOUD_PROJECT` is
   exported per-machine (`~/.zshenv` on the attended laptop pins it to `ailang-multivac-dev`), so a bare
   `AILANG_STORAGE=gcp` silently reads **dev** — a stale graveyard whose newest public feedback is weeks
   old. Run the control first: `echo "${AILANG_CLOUD_PROJECT:-(unset)}"`. Always pass the project explicitly.
2. **`GOOGLE_CLOUD_PROJECT` is ignored.** Only `AILANG_CLOUD_PROJECT` selects the project
   ([client.go:23](internal/storage/firestore/client.go#L23)). Setting the wrong one gives you a
   confident, wrong answer with no error — dev and prod queries return byte-identical output.
3. **`--unread` without `--inbox` fails on prod** — missing Firestore composite index, returns
   `FailedPrecondition`. The one query that answers "what needs attention" is the one that does not run.
   Filter by inbox, or pull `--json` and filter client-side.
4. **`--inbox public-feedback` is not the whole feedback channel.** Package feedback routes to
   `pkg:<vendor>/<name>`; coordinator completions carry an **empty** `to_inbox` and match no filter at all.
   Enumerate inboxes from `--json`, never assume.
5. **The list view truncates IDs to 8 chars**, so every cloud `inbox_<epoch>_<hash>` renders as
   `(inbox_17)`. You cannot ack what the list shows you — get full IDs from `--json`. (An ambiguous
   prefix errors loudly rather than acking the wrong message, so this costs time, not correctness.)
6. **`messages read <id>` marks it read as a side effect.** Triaging by reading silently drains the
   unread queue — the next session then sees an empty inbox and concludes nothing arrived. Read bodies
   out of `--json` when you want to inspect without acking.

**Never export `AILANG_STORAGE` globally.** It is a process-wide switch over *all three* backends —
coordinator, messaging, and observatory ([backend.go:83](internal/storage/backend.go#L83)) — so exporting
it also moves eval banking and coordinator task state to Firestore. Prefix it per-command, as above.
There is no messaging-only selector yet; `ailang chains` has the pattern to copy (`AILANG_CHAINS_READ`,
[chains_read_backend.go:44](cmd/ailang/chains_read_backend.go#L44)).

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
