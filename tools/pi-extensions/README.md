# pi extensions — mission executor containment

pi is the mission's sprint executor (`pi:openrouter/deepseek/deepseek-v4-flash-0731`).
It runs with **full user permissions** from a git worktree, and containment has been the
directive's scope fence plus the controller's post-hoc `git -C <main-checkout> status
--short` review — prose plus an audit, with nothing enforcing it. Iteration 168 showed the
cost: a killed executor kept running and overwrote a verified tree mid-evaluation.

pi's own docs name this as extension territory ("permission gates, path protection,
sandboxing"), so containment goes here rather than into a VM.

## Two layers, because one is not enough

| Tool | Fenced by | Mechanism |
|---|---|---|
| `bash` | upstream `sandbox/` example extension | `@anthropic-ai/sandbox-runtime` → Seatbelt (`sandbox-exec`) on macOS |
| `write`, `edit` | **`worktree-fence.ts` (this dir)** | `tool_call` hook, allow-list on the resolved path |
| `read` | — | not a write risk; see Limitations |

**The upstream sandbox extension fences only `bash`.** It registers a replacement
`bash (sandboxed)` tool and hooks `user_bash`; `write` and `edit` are Node `fs` calls
inside the pi process, which is *not* sandboxed, so they bypass it entirely. Verified by
reading its source — that gap is why this extension exists. Use both.

## worktree-fence.ts

Allow-list, not deny-list. The upstream `protected-paths` example blocks a few known-bad
substrings; wrong shape here. We don't know every path worth protecting, but we do know
the one path that is legitimate.

```bash
cd "$WT" && PI_FENCE_ROOT="$WT" pi --mode json --no-session \
  -e tools/pi-extensions/worktree-fence.ts --model "$MODEL" -p "$PROMPT"
```

Root is `$PI_FENCE_ROOT`, else cwd. Headless-safe: it only ever blocks or allows, never
prompts (`ctx.hasUI` guards the notification), because a prompt in the mission's
non-interactive path would wedge the loop.

Fails closed: a write tool whose path argument it cannot find is refused, not waved through.

### Tests

```bash
cd tools/pi-extensions && bun run worktree-fence.test.ts
```

18 arms, driving the real extension through a fake `pi` object. Covers `..` escapes,
symlink escapes, the macOS `/tmp` → `/private/tmp` realpath trap, `/a/bc`-vs-`/a/b`
prefix confusion, fail-closed shapes, and pass-through for non-write tools.

Not wired into `make ci` — it needs `bun`, which CI does not carry. Run it by hand when
changing the fence.

### One trap worth keeping

An early version resolved **relative** paths against the fence root instead of the process
cwd. pi passes relative paths (`{"path":"dbg.txt"}`), so `dbg.txt` resolved to
`<root>/dbg.txt` — inside the fence, allowed — while pi wrote it to `<cwd>/dbg.txt`,
outside. **The unit tests were green throughout**, because they set `root == cwd` and so
could not distinguish the two bases. Only a live run caught it. The suite now has an
explicit `root != cwd` arm, verified to fail when the bug is reintroduced.

## Limitations — read before trusting this

- **Not an exfiltration control.** `read` is unfenced, and pi's own model calls are outside
  the sandbox by design (that is what keeps OpenRouter reachable). This confines *writes*.
- **`bash` needs the separate sandbox extension.** This file deliberately does not parse
  shell; without that extension a `bash` tool call can still write anywhere.
- **Not yet wired into the mission recipe.** Doing that is a `mission-control/SKILL.md`
  edit; both missions run on a schedule and Gate 5 may write that file, so it wants a clean
  window.
