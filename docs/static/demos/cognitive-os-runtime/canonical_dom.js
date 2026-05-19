// canonical_dom.js — Deterministic DOM layer (M-COG-RUNTIME-BROWSER M1, v0.21.x)
//
// Implements the "canonical DOM" design freeze:
//
//   - Content-hash node IDs: hash(region, ctor, JSON.stringify(fields), parent-hash)
//   - No Date.now() / Math.random() / browser-supplied auto-IDs
//   - Layout determinism: inline styles, system font stack, no animations
//   - Same {region, ctor, fields, parent} input → byte-identical DOM output
//
// The hash function is a pure JS FNV-1a 64-bit variant. Crypto-grade
// hashes (SHA-256 via SubtleCrypto) are async — they'd force every patch
// to return a Promise, defeating the synchronous WasmDOMHandler.ApplyPatch
// contract. FNV-1a is cryptographically weak (collisions possible under
// adversarial input) but deterministic and fast, which is what we need
// for replay equality. M-COG-MESH may switch to BLAKE3 for cross-device
// replay where adversarial input becomes a concern.
//
// Wire format (from cmd/wasm/effects_cognition.go domPatchToJS):
//
//   {ctor: "AddPanel",     fields: ["title", "content"]}
//   {ctor: "UpdateNode",   fields: ["node_id", "content"]}
//   {ctor: "RemoveNode",   fields: ["node_id"]}
//   {ctor: "AddTimeline",  fields: ["title"]}
//
// Loaded by host.js (must precede). Exposes global CanonicalDOM constructor.

(function (global) {
  'use strict';

  // ========================================================================
  // FNV-1a 64-bit hash — split into two 32-bit halves for JS integer safety
  // ========================================================================
  //
  // JS numbers are IEEE-754 doubles → only 53-bit safe integers. Naively
  // computing a 64-bit FNV-1a in one accumulator loses precision near
  // the prime multiplication. We track two 32-bit halves and combine
  // them at the end into a hex string.

  const FNV_OFFSET_HI = 0xcbf29ce4;
  const FNV_OFFSET_LO = 0x84222325;
  const FNV_PRIME_HI = 0x00000100;
  const FNV_PRIME_LO = 0x000001b3;

  function fnv1a64(str) {
    let hi = FNV_OFFSET_HI;
    let lo = FNV_OFFSET_LO;
    for (let i = 0; i < str.length; i++) {
      // XOR with byte (treat string as UTF-16 code units; M1 sticks to
      // ASCII-clean RegionIDs and field content for stability).
      lo ^= str.charCodeAt(i);
      // 64-bit multiply by FNV prime, split into 32-bit halves.
      // (lo + hi * 2^32) * (PRIME_LO + PRIME_HI * 2^32)
      //   = lo*PRIME_LO + (lo*PRIME_HI + hi*PRIME_LO) * 2^32 + hi*PRIME_HI * 2^64
      // We drop the 2^64 term (modulo 2^64).
      const lolo = lo * FNV_PRIME_LO;
      const lohi = lo * FNV_PRIME_HI;
      const hilo = hi * FNV_PRIME_LO;
      // Use BigInt? No — explicit u32 arithmetic is faster + portable.
      const lolo_lo = lolo >>> 0;
      const lolo_hi = Math.floor(lolo / 0x100000000) >>> 0;
      const new_lo = lolo_lo;
      const new_hi = (lolo_hi + (lohi >>> 0) + (hilo >>> 0)) >>> 0;
      lo = new_lo;
      hi = new_hi;
    }
    // Format as 16-char hex
    return pad8Hex(hi) + pad8Hex(lo);
  }

  function pad8Hex(n) {
    const s = (n >>> 0).toString(16);
    return ('00000000' + s).slice(-8);
  }

  // canonicalKey serializes a patch's identity into a stable string.
  // Order of fields in JSON.stringify(arr) is deterministic for arrays
  // (insertion order); for objects we'd need explicit sorting, but the
  // patch wire shape is positional (fields is an Array).
  function canonicalKey(region, ctor, fields, parentHash) {
    // Stringify fields conservatively. JSON.stringify handles undefined
    // by omitting → we coerce to "" to keep the hash stable.
    const f = fields.map(function (v) {
      if (v === undefined || v === null) return '';
      return String(v);
    });
    return region + '|' + ctor + '|' + JSON.stringify(f) + '|' + (parentHash || '');
  }

  // contentHashedID returns a node ID stable across runs for the same
  // (region, ctor, fields, parentHash) input.
  function contentHashedID(region, ctor, fields, parentHash) {
    const key = canonicalKey(region, ctor, fields, parentHash);
    return 'cog_' + fnv1a64(key);
  }

  // ========================================================================
  // CanonicalDOM — applies typed patches to a root element
  // ========================================================================

  function CanonicalDOM(root) {
    this.root = root;
    this.nodesById = new Map();  // node_id → HTMLElement (so UpdateNode/RemoveNode can find them)
    this.regionParentHash = new Map(); // regionId → most recent parent-hash for stable nesting
    // Inject inline-style block once for determinism (no FOUC, no font fallback drift)
    this._injectCanonicalStyles();
  }

  CanonicalDOM.prototype._injectCanonicalStyles = function () {
    if (this.root.querySelector('style[data-cog-canonical]')) return;
    const style = document.createElement('style');
    style.setAttribute('data-cog-canonical', '');
    // System font stack — no web-font network requests, no FOUC.
    // Disable animations/transitions to prevent layout flicker affecting
    // hash inputs (we hash content, not layout — but consistent visual
    // is part of the demo determinism story).
    style.textContent = [
      '[data-cog-region] { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; ',
      '  line-height: 1.5; color: #1f2933; }',
      '[data-cog-region] * { animation: none !important; transition: none !important; }',
      '[data-cog-region] .cog-panel { border: 1px solid #cbd2d9; border-radius: 4px; padding: 12px; margin: 8px 0; background: #fff; }',
      '[data-cog-region] .cog-panel-title { font-weight: 600; margin-bottom: 6px; }',
      '[data-cog-region] .cog-panel-content { font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace; font-size: 0.875em; }',
      '[data-cog-region] .cog-timeline { border-left: 3px solid #4dabf7; padding-left: 12px; margin: 8px 0; }',
    ].join('\n');
    document.head.appendChild(style);
  };

  // applyPatch is the core dispatch. patch shape:
  //   {ctor: <variant>, fields: <positional args>}
  // Returns {nodeId} (caller wraps into the full Go-side response).
  CanonicalDOM.prototype.applyPatch = function (regionEl, regionId, patch) {
    if (!patch || typeof patch.ctor !== 'string' || !Array.isArray(patch.fields)) {
      throw new Error('CanonicalDOM.applyPatch: malformed patch ' + JSON.stringify(patch));
    }
    const parentHash = this.regionParentHash.get(regionId) || regionId;
    switch (patch.ctor) {
      case 'AddPanel': return this._addPanel(regionEl, regionId, patch.fields, parentHash);
      case 'UpdateNode': return this._updateNode(patch.fields);
      case 'RemoveNode': return this._removeNode(patch.fields);
      case 'AddTimeline': return this._addTimeline(regionEl, regionId, patch.fields, parentHash);
      default: throw new Error('CanonicalDOM.applyPatch: unknown ctor ' + patch.ctor);
    }
  };

  CanonicalDOM.prototype._addPanel = function (regionEl, regionId, fields, parentHash) {
    const title = fields[0] != null ? String(fields[0]) : '';
    const content = fields[1] != null ? String(fields[1]) : '';
    const nodeId = contentHashedID(regionId, 'AddPanel', fields, parentHash);

    // Idempotent: if a node with this exact ID already exists, reuse it.
    // Same patch applied twice → byte-identical DOM (replay invariant).
    let panel = this.nodesById.get(nodeId);
    if (panel && panel.isConnected) {
      return { nodeId: nodeId };
    }

    panel = document.createElement('div');
    panel.className = 'cog-panel';
    panel.setAttribute('data-cog-node', nodeId);

    const titleEl = document.createElement('div');
    titleEl.className = 'cog-panel-title';
    titleEl.textContent = title;
    panel.appendChild(titleEl);

    const contentEl = document.createElement('div');
    contentEl.className = 'cog-panel-content';
    contentEl.textContent = content;
    panel.appendChild(contentEl);

    regionEl.appendChild(panel);
    this.nodesById.set(nodeId, panel);
    this.regionParentHash.set(regionId, nodeId);
    return { nodeId: nodeId };
  };

  CanonicalDOM.prototype._addTimeline = function (regionEl, regionId, fields, parentHash) {
    const title = fields[0] != null ? String(fields[0]) : '';
    const nodeId = contentHashedID(regionId, 'AddTimeline', fields, parentHash);

    let timeline = this.nodesById.get(nodeId);
    if (timeline && timeline.isConnected) return { nodeId: nodeId };

    timeline = document.createElement('div');
    timeline.className = 'cog-timeline';
    timeline.setAttribute('data-cog-node', nodeId);

    const titleEl = document.createElement('div');
    titleEl.className = 'cog-timeline-title';
    titleEl.textContent = title;
    timeline.appendChild(titleEl);

    regionEl.appendChild(timeline);
    this.nodesById.set(nodeId, timeline);
    this.regionParentHash.set(regionId, nodeId);
    return { nodeId: nodeId };
  };

  CanonicalDOM.prototype._updateNode = function (fields) {
    const targetId = fields[0] != null ? String(fields[0]) : '';
    const newContent = fields[1] != null ? String(fields[1]) : '';
    const node = this.nodesById.get(targetId);
    if (!node || !node.isConnected) {
      // Unknown target — UpdateNode is idempotent-on-missing per the spec
      // (better to be a silent no-op than to throw and break replay).
      return { nodeId: targetId };
    }
    // Update the .cog-panel-content child if present, otherwise the node's own textContent.
    const contentEl = node.querySelector('.cog-panel-content');
    if (contentEl) {
      contentEl.textContent = newContent;
    } else {
      node.textContent = newContent;
    }
    return { nodeId: targetId };
  };

  CanonicalDOM.prototype._removeNode = function (fields) {
    const targetId = fields[0] != null ? String(fields[0]) : '';
    const node = this.nodesById.get(targetId);
    if (node && node.isConnected) {
      node.remove();
      this.nodesById.delete(targetId);
    }
    return { nodeId: targetId };
  };

  // ========================================================================
  // Content-hash equality — used by replay tests to verify reconstruction
  // ========================================================================
  //
  // Returns a stable hash of the entire region's DOM tree (recursively
  // serialized). Replay tests assert hashEquals(captured) === hashEquals(replayed).

  CanonicalDOM.prototype.regionHash = function (regionId) {
    const region = this.root.querySelector('[data-cog-region="' + regionId.replace(/"/g, '\\"') + '"]');
    if (!region) return null;
    return fnv1a64(serializeForHash(region));
  };

  function serializeForHash(el) {
    // Element shape: <tag attrs>...</tag> with children serialized recursively.
    // We exclude attributes that the canonical DOM doesn't generate (i.e.
    // browser-injected ones like layout caches). Only data-cog-node and class
    // are in canonical scope.
    if (el.nodeType === 3 /* TEXT_NODE */) {
      return el.textContent;
    }
    if (el.nodeType !== 1 /* ELEMENT_NODE */) {
      return '';
    }
    const tag = el.tagName.toLowerCase();
    const nodeId = el.getAttribute('data-cog-node') || '';
    const cls = el.getAttribute('class') || '';
    let out = '<' + tag + ' n=' + nodeId + ' c=' + cls + '>';
    for (let i = 0; i < el.childNodes.length; i++) {
      out += serializeForHash(el.childNodes[i]);
    }
    out += '</' + tag + '>';
    return out;
  }

  // Expose for testing
  CanonicalDOM._fnv1a64 = fnv1a64;
  CanonicalDOM._contentHashedID = contentHashedID;

  global.CanonicalDOM = CanonicalDOM;
})(typeof window !== 'undefined' ? window : globalThis);
