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
| `bash` | `sandbox/` (this dir, adapted upstream) | `@anthropic-ai/sandbox-runtime` → Seatbelt (`sandbox-exec`) on macOS |
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
- **SM.B2a-class work (irreversible publish) still stays off this lane.** Sandboxing writes
  is not the same as bounding blast radius on a publish.

## sandbox/ — the bash layer

Upstream's example, adapted for pi 0.73.1. Two forced deltas, both recorded in the file
header: the package rename (`@earendil-works` → `@mariozechner`), and `CONFIG_DIR_NAME`,
which 0.73.1 does not export from the package root (only `"."` and `"./hooks"` are
exported) so it is inlined as `".pi"` — the value read out of the installed
`dist/config.js`. Dependency moved 0.0.26 → ^0.0.71; `SandboxManager.wrapWithSandbox`
and `initialize` were verified present in 0.0.71 before adopting.

Policy: `sandbox.mission.json` here is canonical; installed at
`~/.pi/extensions/sandbox.json`.

### Two gotchas that cost real debugging

1. **`mkdir -p /tmp/claude` is required.** sandbox-runtime pins `TMPDIR=/tmp/claude`
   inside the sandbox — it is built for Claude Code, which creates that directory. Under
   pi nothing does, and `go build` dies with `creating work dir: stat /tmp/claude: no such
   file or directory`. Your own `TMPDIR` does not survive into the sandbox, so exporting it
   is not a workaround.
2. **The Go caches must be in `allowWrite`.** With the stock `[".", "/tmp"]` policy,
   `go build` fails on `~/Library/Caches/go-build: operation not permitted`. Cache dirs
   cannot corrupt the repo, so widening to them costs nothing that matters — but it must be
   deliberate, and re-verified after: widening a policy is how holes get opened.

### Verified live (2026-08-11), running the skill's invocation verbatim

| check | result |
|---|---|
| bash write to `$HOME` | `Operation not permitted`, exit 1 |
| bash read of `secrets.env` | `Operation not permitted`, exit 1 |
| `write` tool to `/tmp` (outside `$WT`) | blocked by the fence |
| `go build ./internal/messaging/` | `BUILD_OK`, exit 0 |
