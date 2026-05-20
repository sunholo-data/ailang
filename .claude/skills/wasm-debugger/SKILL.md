---
name: wasm-debugger
description: Diagnose AILANG WASM browser failures — frozen pages, silent loadModule hangs, type-checker stack overflows, missing effect handler bridges, version drift between wasm/ailang.wasm and the .ail source. Use when a browser demo using AILANG WASM behaves differently than the CLI, or when the page freezes without a console error.
---

# WASM Debugger

Diagnose failures in browser-side AILANG WASM that **don't reproduce on the CLI**. Most painful class of bug in this codebase: the page freezes, no console error, no banner, DevTools won't open. This skill walks the canonical diagnostic path that's been proven in the field (most recently the 2026-05-20 `cognitive_commons/services/citizen.ail` stack overflow).

## When to invoke

User says any of:
- "the browser demo freezes"
- "WASM init failed" / "NetworkError when attempting to fetch resource"
- "AILANG works on CLI but not in the browser"
- "the page is slow to load" (when WASM is in the boot path)
- "loadModule never returns"
- "we broke it somehow in code we added"

Or you spot these symptoms in user-supplied logs:
- `repl.loadModule(...)` is logged but nothing else
- Server log shows the boot fetch sequence stops mid-stream
- "Maximum call stack size exceeded" in DevTools (if DevTools is reachable at all)

## Core insight

> **The CLI's `ailang check` is necessary but NOT sufficient for WASM-targeted modules.** Native Go grows goroutine stacks dynamically; WASM is bound by the host JS engine's ~10-15K call-stack frames. Type-checker recursion that's fine native can blow the WASM stack 80-120 seconds later, locking the page main thread with no error.
>
> **Always exercise WASM the way the browser does** — see `scripts/wasm-loadmodule-harness.js` in the demos repo.

## The diagnostic ladder

Walk these in order. Stop at the first one that pins the bug.

### Step 1: Is the server actually serving?

Surprisingly common false alarm. The default `python3 -m http.server` is single-threaded and wedges on large WASM transfers under concurrent fetches.

```bash
# Verify port + responsiveness
lsof -ti :8765
curl -sI http://localhost:<port>/<demo>/wasm/ailang.wasm | head -3

# Test concurrency (sim browser boot: WASM + small files in parallel)
(curl -so /dev/null -w "wasm:  %{http_code} %{time_total}s\n" -m 30 .../wasm/ailang.wasm &
 curl -so /dev/null -w "html:  %{http_code} %{time_total}s\n" -m 30 .../ &
 wait)
```

"Empty reply from server" = wedged. Fix: switch the dev server to `ThreadingHTTPServer` (see `demos/scripts/serve.sh` for the canonical pattern).

### Step 2: Is the wasm binary fresh?

The wasm in `demos/wasm/ailang.wasm` is built from the AILANG source repo (`make build-wasm`). It drifts. Stale wasm against new `.ail` syntax = silent loadModule hang.

```bash
# Canonical check (if the demos repo has it):
bash /Users/mark/dev/sunholo/demos/scripts/check-wasm-freshness.sh

# Otherwise compare versions manually:
strings demos/wasm/ailang.wasm | grep -oE 'v0\.[0-9]+\.[0-9]+-[0-9]+-g[0-9a-f]+' | head -1
ailang --version | head -1
```

Mismatch → rebuild:
```bash
cd /Users/mark/dev/sunholo/ailang
make build-wasm
cp bin/ailang.wasm /Users/mark/dev/sunholo/demos/wasm/ailang.wasm
```

### Step 3: Run the headless WASM harness

This is the single most valuable diagnostic. Runs the actual `wasm/ailang.wasm` in Node, calls `ailangLoadModule` for each `.ail` file the demo boots, times each, fails loudly with the actual error message.

```bash
node /Users/mark/dev/sunholo/demos/scripts/wasm-loadmodule-harness.js
```

Exit codes:
- `0` — all modules loaded cleanly. The freeze is NOT in module loading. Look elsewhere (effect handler wiring, event loop, ailangCall path).
- `2` — module hung past 15s budget. Pure hang.
- `3` — module threw, almost always "Maximum call stack size exceeded". WASM type-checker recursion overflow. See Step 4.
- `4` — module returned `success: false`. That's a normal AILANG type error; look at the error string.

If you're debugging a demo other than `cognitive_commons`, copy the harness and update the `modules` array. The script is ~150 LOC, self-contained.

### Step 4: Stack overflow in WASM type-checker

If Step 3 returns "Maximum call stack size exceeded": it's the bug class first documented in `debug-notes/wasm-citizen-stack-overflow.md`. The CLI passes because Go natively grows goroutine stacks. WASM uses the JS call stack.

**Diagnostic checklist on the offending `.ail` file:**

1. Triple-nested `match` (match inside match inside match)?
2. Multiple back-to-back matches on the same tagged-union value with record-field access (e.g. `Ok(s) => s.x`, then `Ok(s) => s.y`)?
3. Many intra-package imports with destructured symbols?
4. `M-TYPECHECK-NO-AUTO-UNWRAP-RESULT` analysis active on receiver shapes?

Workarounds (in order of preference):
1. **Flatten nested matches** into sequential let-bindings.
2. **Collapse repeated matches on the same value** into a single match returning a record/tuple. Helper function with one match, downstream code reads fields.
3. **Split the function** into smaller top-level functions — each gets its own type-check pass.
4. **Stub** the offending module: keep exported signatures, trivial bodies. Tracks the upstream fix.

If none of those help, the bug is upstream — see [m-wasm-typecheck-iterative.md](../../design_docs/planned/v1_1_0/m-wasm-typecheck-iterative.md) for the design doc on making the type-checker iterative for WASM targets.

### Step 5: Effect-handler wiring failure (`! {DOM}`, `! {Msg}`, etc.)

If the harness passes but the running demo still mis-behaves, the issue is at effect-handler binding time:

```js
// Check these globals exist on `window` after WASM init:
ailangSetDOMApplyPatchHandler
ailangSetDOMApplyBatchHandler
ailangSetMsgSendHandler
ailangSetMsgRecvHandler
ailangSetDOMSubscribeHandler
ailangSetMsgSubscribeHandler
ailangSetAIHandler
ailangSetAIStepHandler
ailangGrantCapability
```

If `typeof window.ailangSetDOMApplyPatchHandler !== "function"`, the WASM was built without the cognition handlers — rebuild with the current `cmd/wasm/effects_cognition.go`.

If handler exists but the AILANG-side `.ail` call hangs: the JS handler might be waiting on a Promise that never resolves. Look for handlers that route through `awaitJSResult` in `cmd/wasm/effects_helpers.go` — those are points where Go-WASM blocks on a JS callback.

### Step 6: Surface what's actually happening

If the browser is locked and DevTools won't open, you need a visible-without-DevTools surfacing path. The cognitive_commons demo has these patterns worth reusing:

- **Red boot banner** pinned `position:fixed; top:0; z-index:99999` — captures `window.onerror` + `window.onunhandledrejection` + per-phase try/catch around each boot step.
- **localStorage persistence** — every phase boundary writes to `localStorage.<demo>_boot_diag`. Survives page hangs and reloads.
- **A `diag.html` companion page** — renders the persisted diagnostic in plain text with a "Copy all" button. Reachable even when the main page is frozen.
- **A `reset.html` companion page** — clears IDB + state without needing to navigate the broken page.

Reference implementation: `demos/cognitive_commons/{index.html (showBootError + bootStep + bootLog), diag.html, reset.html}`.

## Anti-patterns (don't do these)

- ❌ "Just trust `ailang check`" — it doesn't share the WASM stack limit.
- ❌ "Just trust `scripts/check_demos.sh`" — same. CLI-only.
- ❌ Increasing `node --stack-size=...` to make the harness pass. If the recursion is pathologically slow (quadratic or worse), more stack just delays the symptom.
- ❌ Adding silent fallbacks when a fetch/loadModule fails. Surface every error to the boot banner; the user shouldn't have to open DevTools to find them.
- ❌ Wrapping loadModule in `setTimeout(..., 0)` to "yield" — loadModule is synchronous Go-side; the yield happens BEFORE the call, doesn't help.

## After the fix

When the underlying issue is resolved (rebuild, code change, upstream fix landed):

1. Re-run `node scripts/wasm-loadmodule-harness.js` — exit 0.
2. Reload the demo in Firefox + Chrome; confirm boot diagnostic shows `setAilangStatus("ready", ...)` in the banner.
3. If you stubbed a module: restore the real implementation from the `.orig` copy, harness again.
4. Update the postmortem doc in the demos repo with the resolution.

## Reference docs

- [debug-notes/wasm-citizen-stack-overflow.md](../../../demos/debug-notes/wasm-citizen-stack-overflow.md) — first instance of this bug class, full diagnostic trail.
- [m-wasm-typecheck-iterative.md](../../design_docs/planned/v1_1_0/m-wasm-typecheck-iterative.md) — the upstream fix design.
- [internal/types/](../../internal/types/) — type-checker source. M-TYPECHECK-NO-AUTO-UNWRAP-RESULT is the May 2026 change that amplified the recursion depth.
- [cmd/wasm/effects_cognition.go](../../cmd/wasm/effects_cognition.go) — DOM/Msg bridge implementation.
