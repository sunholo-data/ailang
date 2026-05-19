# Cognitive OS Runtime — Public Demo (v0.21.x)

Live at: **https://ailang.sunholo.com/demos/cognitive-os-runtime/**

Interactive demonstration of the AILANG Cognitive OS substrate running entirely in your browser. No server. No network. No build step.

## What this demo proves

1. **Deterministic DOM mutation** — agents address scoped regions with structured patches (no raw HTML/JS injection); content-hashed node IDs survive across reloads.
2. **Cross-tab agent messaging** — `BroadcastChannel` Web API ferries typed envelopes between tabs with Lamport-clock preservation.
3. **Persistent cognitive event log** — every action writes a structured event to IndexedDB; the log survives tab restarts.
4. **Byte-identical replay** — capture an event log → reset state → replay → DOM reconstructed byte-identically. Verified via FNV-1a content hashing.
5. **Subscribe + drain** — DOM events flow back into AILANG closures via a queue+drain pattern that preserves single-threaded determinism.

## Mirrors the `wasm-step-byo-key` precedent

This demo follows the same file-shape conventions as the [BYO-key AI demo](https://ailang.sunholo.com/demos/wasm-step-byo-key/) (v0.19.0):

- Plain HTML page with `<script src>` includes — no bundler
- WASM REPL loaded from `/wasm/wasm_exec.js` + `/js/ailang-repl.js`
- Cognitive OS runtime modules under `/wasm/cognitive-runtime/` (loaded in dependency order)
- Per-tab sender identity in `sessionStorage`
- Layered effects: this demo shows DOM + Msg; the BYO-key demo shows AI; a future combined demo will use all three (`!: {AI, DOM, Msg, Cog}`)

## Source

Browser host JS: [`docs/static/wasm/cognitive-runtime/`](https://github.com/sunholo-data/ailang/tree/dev/docs/static/wasm/cognitive-runtime)
Go-side substrate: [`internal/cognition/`](https://github.com/sunholo-data/ailang/tree/dev/internal/cognition) + [`internal/effects/`](https://github.com/sunholo-data/ailang/tree/dev/internal/effects)
Design docs: [M-COG-RUNTIME (shipped)](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_21_0/m-cog-runtime.md) + [M-COG-RUNTIME-BROWSER (this sprint)](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-cog-runtime-browser.md)
