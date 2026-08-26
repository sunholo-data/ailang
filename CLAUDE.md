# Claude Instructions for AILANG Development

## North Star — [design_docs/PROGRAM.md](design_docs/PROGRAM.md)

This work runs as one program: a **self-specializing harness**. The motoko core is **minimal and frozen**;
every improvement is routed to exactly one lane — an **AILANG fix**, a **motoko extension**, or (rarely) a
**core-floor fix** that re-freezes. We discover friction by running benchmarks through the data-led loop,
then route it. **Default bias: if it can be an extension, it is an extension — not a core change.** The
living roadmap (benchmark ladder · extension catalog · AILANG-fix backlog) lives in PROGRAM.md.

## Session start — every machine reads the SAME message store

**The canonical inbox is the shared cloud store: prod Firestore, project `ailang-multivac`.**
Public feedback, package feedback, coordinator completions, and every other machine's agent
traffic land there. A bare `ailang messages list` reads only *this* machine's private SQLite,
so it shows none of that — which is why real user feedback sat unread for weeks.

Every machine that does AILANG work (the voightkampff Studio, cloud sessions, Claude Code
managed runs, this laptop) should have these exported in its shell profile:

```bash
export AILANG_MESSAGES_STORE=gcp
export AILANG_MESSAGES_PROJECT=ailang-multivac
```

**These are safe to export** — unlike `AILANG_STORAGE`, they are scoped to messaging and leave
coordinator and observatory (eval banking, `ailang chains`) on local storage. Confirm with
`ailang storage status`, which must still say `Mode: local`.

Then the ordinary commands just work, and non-local listings name the store in their header:

```bash
ailang messages list --unread                    # canonical inbox, all inboxes
ailang messages list --inbox public-feedback --unread
```

Summarize to the user and ask what to do **before** acking; then `ailang messages ack <id>`
(or `ack --all`). If a task fails, `ailang messages unack MSG_ID`.

To check this machine's private local inbox, override for the one command:

```bash
AILANG_MESSAGES_STORE=local ailang messages list --unread
```

**Traps, each measured 2026-08-25 — do not rediscover these:**

1. **Name the project explicitly; never rely on `AILANG_CLOUD_PROJECT`.** It is pinned per-machine
   (`~/.zshenv` on the attended laptop points it at `ailang-multivac-dev`, a stale graveyard). That is
   exactly why `AILANG_MESSAGES_PROJECT` exists and why it wins. Control: `ailang messages list` prints
   `store: gcp (Firestore, project ...)` in the header whenever the store is not local — read it.
2. **`GOOGLE_CLOUD_PROJECT` is ignored.** Only `AILANG_CLOUD_PROJECT` / `AILANG_MESSAGES_PROJECT` select
   the project ([client.go:23](internal/storage/firestore/client.go#L23)). Setting the wrong one gives a
   confident, wrong answer with no error — dev and prod queries returned byte-identical output.
3. **`--inbox public-feedback` is not the whole feedback channel.** Package feedback routes to
   `pkg:<vendor>/<name>`. Enumerate inboxes rather than assuming; `--unread` with no `--inbox` spans
   them all.
4. **The list view truncates IDs to 8 chars**, so every cloud `inbox_<epoch>_<hash>` renders as
   `(inbox_17)`. You cannot ack what the list shows you — get full IDs from `--json`. (An ambiguous
   prefix errors loudly rather than acking the wrong message, so this costs time, not correctness.)
5. **`messages read <id>` marks it read as a side effect.** Triaging by reading silently drains the
   unread queue — the next session then sees an empty inbox and concludes nothing arrived. Read bodies
   out of `--json` when you want to inspect without acking.

6. **A binary predating `6759ea4fa` IGNORES `AILANG_MESSAGES_STORE` silently** — it reads local
   SQLite and exits 0, so the whole protocol above is vacuous and the session concludes the inbox
   is quiet. `~/go/bin/ailang` drifts by design (installing mid-run would disturb concurrent
   agents), so assume it is stale. One-command control — an INVALID value must be refused:
   `AILANG_MESSAGES_STORE=not-a-real-store ailang messages list --unread` must error
   `unknown message store mode`; a normal listing means you are on local. A non-local listing also
   names its store in the header — no `store: gcp (...)` header, no cloud. Build a fresh binary to
   a scratch dir and prepend it to PATH rather than `make quick-install`.
   (Confirmed on the Studio 2026-08-26: its v0.33.2 binary made both exports inert and it kept
   reading local SQLite — the missing `store:` header was the only visible tell.)

Topology — which projects exist, which switch moves which backend, what each machine is
wired to, and how package feedback flows end to end:
[docs/internal/message-plane-topology.md](docs/internal/message-plane-topology.md).

**Do not export `AILANG_STORAGE`** to reach the cloud inbox. It is a process-wide switch over *all
three* backends — coordinator, messaging, and observatory
([backend.go:83](internal/storage/backend.go#L83)) — so exporting it also moves eval banking and
coordinator task state to Firestore. `AILANG_MESSAGES_STORE` is the scoped selector; use it.

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
| **What did the PROVIDER actually see and return?** | OpenRouter **Broadcast** traces, in the prod observatory — `curl "https://dashboard.ailang.sunholo.com/api/observatory/spans?limit=1000&start_after=<RFC3339>&start_before=<RFC3339>"`. Each `LLM Generation` span carries ~92 attributes: the **full `rawRequest`** (so you can read the budget and reasoning config that actually went on the wire, not the one we declared), the completion text, the token split incl. `output_tokens.reasoning`, `finish_reason`, the provider host, and a `cancelled` flag. This is a SECOND, independent instrument on every OpenRouter call — use it before theorising from our own logs. |

The provider-side trace is the one that settles arguments our own logs cannot. On
2026-08-26 it showed that all five "pi:deepseek returned zero bytes" failures were one
mechanism — the model streamed only reasoning tokens and never emitted content — and that
**we** killed every one of them with a size ceiling measuring pi's quadratic event replay
rather than the model. Two prompt-level fixes had already been aimed at a model problem
that did not exist. The account key is an inference key, so `/api/v1/key` and
`/api/v1/generation?id=<gen-id>` work but `/api/v1/activity` returns 403 (needs a
management key) — go via the observatory, which is indexed by time anyway.

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
