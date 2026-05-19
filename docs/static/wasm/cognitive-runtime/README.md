# Cognitive OS Browser Runtime (M-COG-RUNTIME-BROWSER, v0.21.x)

Browser-side JS substrate that lights up the [M-COG-RUNTIME WASM bridges](https://github.com/sunholo-data/ailang/blob/dev/cmd/wasm/effects_cognition.go).

## Files

| File | Role | Loaded |
|------|------|--------|
| `canonical_dom.js` | Deterministic DOM layer: content-hash node IDs, idempotent patches, no time/random deps | 1st |
| `host.js` | WASM↔JS bridge: registers `ailangSet*Handler` callbacks; manages scoped regions + sender identity | 2nd |
| `event_log_indexeddb.js` | (M3) IndexedDB persistence sink for the cognitive event log | later |
| `replay.js` | (M3) JSONL → DOM reconstruction | later |
| `scheduler.js` | (M3) Microtask-based JS scheduler mirroring Go-side ordering | later |

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

- **M1** (shipped, this directory): host.js + canonical_dom.js + smoke index.html
- **M2** (next): BroadcastChannel cross-tab wire-up
- **M3** (later): IndexedDB sink + replay engine + JS scheduler
- **M4** (later): Subscribe ops + `_cog_drain()` builtin
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
