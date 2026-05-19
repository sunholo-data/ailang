# Cognitive OS Browser Runtime (M-COG-RUNTIME-BROWSER, v0.21.x)

Browser-side JS substrate that lights up the [M-COG-RUNTIME WASM bridges](https://github.com/sunholo-data/ailang/blob/dev/cmd/wasm/effects_cognition.go).

## Files

| File | Role | Loaded |
|------|------|--------|
| `canonical_dom.js` | Deterministic DOM layer: content-hash node IDs, idempotent patches, no time/random deps | 1st |
| `event_log_indexeddb.js` | IndexedDB persistence sink for the cognitive event log | 2nd |
| `host.js` | WASM↔JS bridge: registers `ailangSet*Handler` callbacks; manages scoped regions + sender identity | 3rd |
| `replay.js` | JSONL → DOM reconstruction with content fidelity + byte-equality verification | 4th |
| `scheduler.js` | Microtask-based JS event loop with kind-filtered subscribers + re-entrance safety | 5th |

## Loading order

```html
<script src="/wasm/wasm_exec.js"></script>
<script src="/js/ailang-repl.js"></script>
<script src="/wasm/cognitive-runtime/canonical_dom.js"></script>
<script src="/wasm/cognitive-runtime/host.js"></script>
<script>
  (async () => {
    const repl = new AilangREPL();
    await repl.init();
    CognitiveOS.attach({ rootSelector: '[data-cog-runtime-root]' });
    await repl.loadModule('demo/agent', myAilangSource);
    await repl.call('demo/agent', 'main');
  })();
</script>
```

## Status

- **M1** ✅ Shipped — host.js + canonical_dom.js + smoke index.html (11/11 assertions pass)
- **M2** ✅ Shipped — BroadcastChannel cross-tab wire-up (20/20 + verified cross-tab delivery)
- **M3** ✅ Shipped — IndexedDB sink + replay engine + JS scheduler (byte-identical DOM verified)
- **M4** (next): Subscribe ops + `_cog_drain()` builtin
- **M5** (last): Trace extension + Playwright + public demo deploy

See [sprint plan](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-cog-runtime-browser-sprint-plan.md) and [design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-cog-runtime-browser.md).

## Local development

```bash
# Build WASM + serve docs/static
make wasm-serve

# Open the smoke test
open http://localhost:3000/wasm/cognitive-runtime/index.html
```

The smoke harness uses `CognitiveOS._applyPatchDirect()` to drive patches without a full WASM REPL stack — useful when iterating on the DOM layer before AILANG modules are wired up.

## Determinism guarantees

- Same `(region, ctor, fields, parent-hash)` → same node ID across reloads
- No `Date.now()` / `Math.random()` / `crypto` in the canonical DOM path
- Inline styles + system font stack → no FOUC, no network font fallback
- Animations + transitions disabled inside scoped regions

The replay engine in M3 verifies these guarantees via `regionHash()` content-hash equality after page refresh.

## Patch semantics: append-only with parent-hash chain

`AddPanel` and `AddTimeline` are **append-only**, not idempotent-on-content. Calling `AddPanel('A', 'B')` twice in sequence produces **two sibling panels**, not one — each gets a deterministic-but-distinct ID via the parent-hash chain.

This is the right model for an event-driven runtime: the cognitive event log replays each AddPanel as a distinct event, and replay-determinism means "same sequence of events → same DOM", not "duplicate events collapse".

If you want explicit replace-or-create semantics, use `UpdateNode(node_id, content)` with a stable `node_id`. `UpdateNode` IS idempotent — same `(node_id, content)` applied twice has no extra effect.
