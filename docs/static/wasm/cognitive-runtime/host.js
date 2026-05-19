// host.js — Cognitive OS WASM↔JS bridge (M-COG-RUNTIME-BROWSER M1, v0.21.x)
//
// Registers the 8 JS-callback globals that the WasmDOMHandler and
// WasmMsgHandler in cmd/wasm/effects_cognition.go expect:
//
//   ailangSetDOMApplyPatchHandler(fn)
//   ailangSetDOMApplyBatchHandler(fn)
//   ailangSetMsgSendHandler(fn)
//   ailangSetMsgRecvHandler(fn)
//
// (Subscribe handlers land in M4 alongside _cog_drain().)
//
// Each callback returns a structured response that the Go-side decoder
// (awaitJSResult + jsGetString/jsGetInt in cmd/wasm/effects.go) reads
// back as a *PatchResult / *BatchResult / *SendResult / *Message.
//
// Design freeze (do not change without updating the umbrella doc):
//   - Scoped regions only: one <div data-cog-region="${region}"> per agent
//   - Canonical layout: no Date.now() / Math.random() / browser auto-IDs
//   - Patch shape from WASM: {ctor: "AddPanel"|"UpdateNode"|"RemoveNode"|"AddTimeline", fields: [...]}
//
// Companion modules:
//   - canonical_dom.js — deterministic node-ID layer (loaded next)
//   - event_log_indexeddb.js — Sink for the cognitive event log (M3)
//   - replay.js — JSONL → DOM reconstruction (M3)
//
// Usage from index.html:
//
//   <script src="/wasm/wasm_exec.js"></script>
//   <script src="/js/ailang-repl.js"></script>
//   <script src="/wasm/cognitive-runtime/canonical_dom.js"></script>
//   <script src="/wasm/cognitive-runtime/host.js"></script>
//   <script>
//     const repl = new AilangREPL();
//     await repl.init();
//     CognitiveOS.attach({ rootSelector: '[data-cog-runtime-root]' });
//     await repl.loadModule('demo/agent', myAilangSource);
//   </script>

(function (global) {
  'use strict';

  // ========================================================================
  // Internal state — per-page, populated when CognitiveOS.attach() is called
  // ========================================================================

  const state = {
    root: null,                 // HTMLElement: top-level container for scoped regions
    regions: new Map(),         // RegionID (string) → HTMLElement
    sentEnvelopes: new Map(),   // msg_id → envelope (for self-recv when no peers)
    sender: null,               // string: this tab's NodeID (set in attach())
    canonical: null,            // CanonicalDOM instance (set in attach())
    clock: 0,                   // local Lamport-like counter for Sends without a Go-side stamp
    nextMsgId: 0,               // monotonic counter for envelope IDs
  };

  // ========================================================================
  // Sender identity — nanoid-like; persists in localStorage per origin
  // ========================================================================

  // generateSenderId produces a 64-bit-equivalent random identifier.
  // Uses crypto.getRandomValues for entropy; falls back to Math.random for
  // older runtimes (M-COG-MESH will tighten the security story; M1 is
  // deterministic-runtime, not adversarial-trust).
  function generateSenderId() {
    let bytes;
    if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
      bytes = new Uint8Array(8);
      crypto.getRandomValues(bytes);
    } else {
      bytes = new Uint8Array(8);
      for (let i = 0; i < 8; i++) bytes[i] = Math.floor(Math.random() * 256);
    }
    let hex = '';
    for (let i = 0; i < bytes.length; i++) {
      hex += bytes[i].toString(16).padStart(2, '0');
    }
    return 'tab_' + hex;
  }

  // resolveSender returns the per-tab NodeID. Uses sessionStorage (NOT
  // localStorage) so each browser tab gets its own identity — localStorage
  // is shared across all tabs on an origin, which would make every tab
  // appear as the same sender and break the BroadcastChannel self-loop
  // guard. sessionStorage persists across reloads of the same tab, which
  // is the right durability scope for a per-tab cognitive node.
  function resolveSender() {
    if (state.sender) return state.sender;
    try {
      const stored = sessionStorage.getItem('ailang_cog_sender');
      if (stored && stored.length > 0) {
        state.sender = stored;
        return stored;
      }
    } catch (_) { /* sessionStorage may throw in private mode */ }
    const fresh = generateSenderId() + '_' + Date.now().toString(36);
    state.sender = fresh;
    try { sessionStorage.setItem('ailang_cog_sender', fresh); } catch (_) {}
    return fresh;
  }

  // ========================================================================
  // Region management — agents address DOM by RegionID; the host enforces
  // the scoped-regions design freeze by refusing to patch outside.
  // ========================================================================

  function ensureRegion(regionId) {
    if (!state.root) {
      throw new Error('CognitiveOS not attached — call CognitiveOS.attach({ rootSelector: ... }) first');
    }
    let el = state.regions.get(regionId);
    if (el && el.isConnected) return el;
    // Look for an existing region declared in the HTML
    el = state.root.querySelector('[data-cog-region="' + cssEscape(regionId) + '"]');
    if (el) {
      state.regions.set(regionId, el);
      return el;
    }
    // Auto-create
    el = document.createElement('div');
    el.setAttribute('data-cog-region', regionId);
    state.root.appendChild(el);
    state.regions.set(regionId, el);
    return el;
  }

  // cssEscape is a tiny helper for selector safety. Browsers ship CSS.escape
  // but not all environments expose it; fall back to a conservative quote.
  function cssEscape(s) {
    if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') return CSS.escape(s);
    // Allow ASCII letters, digits, underscore, hyphen as-is; escape the rest.
    return String(s).replace(/[^a-zA-Z0-9_-]/g, function (ch) { return '\\' + ch; });
  }

  // ========================================================================
  // DOM patch application — delegates to the canonical_dom.js layer
  // for deterministic node-ID assignment + element creation.
  // ========================================================================

  function applyPatchInternal(regionId, patch) {
    if (!state.canonical) {
      throw new Error('CognitiveOS canonical_dom.js not loaded — patch dispatch unavailable');
    }
    if (!patch || typeof patch !== 'object') {
      throw new Error('CognitiveOS.applyPatch: patch must be an object, got ' + typeof patch);
    }
    const region = ensureRegion(regionId);
    const result = state.canonical.applyPatch(region, regionId, patch);
    return {
      node_id: result.nodeId,
      budget_remaining: -1, // unbounded in M1; M4 wires budget enforcement
    };
  }

  // ========================================================================
  // Msg fabric — local-tab queue with optional BroadcastChannel cross-tab
  // delivery (M2 wires the BroadcastChannel side).
  // ========================================================================

  const mailboxes = new Map();   // mailbox name → array of pending envelopes
  const mailboxWaiters = new Map(); // mailbox name → array of resolve fns
  const broadcastChannels = new Map(); // mailbox name → BroadcastChannel instance
  state.broadcastChannels = broadcastChannels; // expose for sendInternal's cross-tab dispatch

  // ensureBroadcastChannel lazy-creates a per-mailbox BroadcastChannel and
  // wires its onmessage handler to enqueueDelivery. Returns null on
  // engines that don't expose BroadcastChannel (older Safari, headless
  // environments without the Web API).
  function ensureBroadcastChannel(name) {
    if (broadcastChannels.has(name)) return broadcastChannels.get(name);
    if (typeof global.BroadcastChannel !== 'function') return null;
    let bc;
    try {
      bc = new global.BroadcastChannel(name);
    } catch (_) {
      return null;
    }
    bc.onmessage = function (ev) {
      // ev.data is the envelope the sender tab posted. Skip self-loop:
      // BroadcastChannel does NOT deliver to the sending tab per spec,
      // but defensive in case of polyfill quirks.
      const env = ev && ev.data;
      if (!env || typeof env !== 'object') return;
      if (env.from === state.sender) return;
      enqueueDelivery(env);
    };
    broadcastChannels.set(name, bc);
    return bc;
  }

  function enqueueDelivery(env) {
    const arr = mailboxes.get(env.to) || [];
    arr.push(env);
    mailboxes.set(env.to, arr);
    const waiters = mailboxWaiters.get(env.to);
    if (waiters && waiters.length > 0) {
      const resolve = waiters.shift();
      const next = arr.shift();
      mailboxes.set(env.to, arr);
      resolve(next);
    }
  }

  function sendInternal(to, payload) {
    state.clock += 1;
    state.nextMsgId += 1;
    const sender = resolveSender();
    const env = {
      msg_id: 'cog_' + state.nextMsgId + '_' + state.clock,
      from: sender,
      to: String(to),
      payload: String(payload),
      clock: state.clock,
    };
    state.sentEnvelopes.set(env.msg_id, env);
    // Same-tab delivery (loopback) so single-tab demos work without a peer
    enqueueDelivery(env);
    // M2: BroadcastChannel cross-tab delivery. Per spec, BroadcastChannel
    // does NOT deliver to the sending tab — so the loopback above is the
    // only path for same-tab Recv. The postMessage below propagates to
    // all OTHER tabs on the same origin that have called ensureBroadcastChannel.
    const bc = ensureBroadcastChannel(env.to);
    if (bc) {
      try { bc.postMessage(env); } catch (_) {}
    }
    return {
      msg_id: env.msg_id,
      clock: env.clock,
      budget_remaining: -1,
    };
  }

  function recvInternal(mailboxName) {
    // Eagerly bind a BroadcastChannel for this mailbox so cross-tab
    // arrivals show up even if no Send has happened yet in this tab.
    ensureBroadcastChannel(mailboxName);

    const arr = mailboxes.get(mailboxName) || [];
    if (arr.length > 0) {
      const env = arr.shift();
      mailboxes.set(mailboxName, arr);
      return Promise.resolve(env);
    }
    // Block via Promise — the WASM side awaits this via awaitJSResult.
    return new Promise(function (resolve) {
      const list = mailboxWaiters.get(mailboxName) || [];
      list.push(resolve);
      mailboxWaiters.set(mailboxName, list);
    });
  }

  // ========================================================================
  // Public API — attach() wires JS globals; the rest is auto-managed
  // ========================================================================

  function attach(opts) {
    opts = opts || {};
    const rootSelector = opts.rootSelector || '[data-cog-runtime-root]';
    state.root = document.querySelector(rootSelector);
    if (!state.root) {
      throw new Error('CognitiveOS.attach: no element matched ' + rootSelector);
    }
    if (!global.CanonicalDOM) {
      throw new Error('CognitiveOS.attach: canonical_dom.js must be loaded first');
    }
    state.canonical = new global.CanonicalDOM(state.root);
    resolveSender();

    // Register the 4 WASM-side bridges. Each global accepts a callback
    // matching the WasmDOMHandler / WasmMsgHandler shape in
    // cmd/wasm/effects_cognition.go.
    if (typeof global.ailangSetDOMApplyPatchHandler === 'function') {
      global.ailangSetDOMApplyPatchHandler(function (region, patch) {
        return applyPatchInternal(region, patch);
      });
    }
    if (typeof global.ailangSetDOMApplyBatchHandler === 'function') {
      global.ailangSetDOMApplyBatchHandler(function (region, patches) {
        const ids = [];
        for (let i = 0; i < patches.length; i++) {
          const r = applyPatchInternal(region, patches[i]);
          ids.push(r.node_id);
        }
        return { node_ids: ids, budget_remaining: -1 };
      });
    }
    if (typeof global.ailangSetMsgSendHandler === 'function') {
      global.ailangSetMsgSendHandler(function (to, payload) {
        return sendInternal(to, payload);
      });
    }
    if (typeof global.ailangSetMsgRecvHandler === 'function') {
      global.ailangSetMsgRecvHandler(function (mailboxName) {
        return recvInternal(mailboxName);
      });
    }
    return {
      sender: state.sender,
      root: state.root,
    };
  }

  // ========================================================================
  // Debug surface — exposed for the smoke-test harness + Playwright assertions
  // ========================================================================

  function debugSnapshot() {
    const regions = {};
    state.regions.forEach(function (el, id) {
      regions[id] = el.outerHTML;
    });
    return {
      sender: state.sender,
      clock: state.clock,
      regions: regions,
      mailboxes: Array.from(mailboxes.keys()),
    };
  }

  global.CognitiveOS = {
    attach: attach,
    debugSnapshot: debugSnapshot,
    // Direct-test bypass — lets the smoke harness drive applyPatch without
    // a full WASM REPL stack. Used by index.html before any AILANG code runs.
    _applyPatchDirect: function (region, patch) {
      return applyPatchInternal(region, patch);
    },
    _sendDirect: function (to, payload) {
      return sendInternal(to, payload);
    },
    _recvDirect: function (mailbox) {
      return recvInternal(mailbox);
    },
    _resetForTesting: function () {
      state.regions.clear();
      mailboxes.clear();
      mailboxWaiters.clear();
      state.sentEnvelopes.clear();
      state.clock = 0;
      state.nextMsgId = 0;
      if (state.root) state.root.innerHTML = '';
      // CRITICAL: also reset CanonicalDOM internal state — otherwise
      // regionParentHash retains entries across the reset and the next
      // _addPanel sees a stale parent-hash, breaking replay-determinism.
      // Re-create the instance to drop nodesById + regionParentHash
      // + re-inject the canonical style block.
      if (state.root) state.canonical = new global.CanonicalDOM(state.root);
      // Close BroadcastChannels so the next test session doesn't
      // inherit cross-tab subscriptions to dead handlers.
      broadcastChannels.forEach(function (bc) { try { bc.close(); } catch (_) {} });
      broadcastChannels.clear();
    },
  };
})(typeof window !== 'undefined' ? window : globalThis);
