// replay.js — Cognitive event log replay engine (M3 Day 6, v0.21.x)
//
// Reconstructs the prior session's DOM by replaying its cognitive event
// log through the canonical DOM dispatch path. This is the JS-side
// counterpart to internal/cognition/replay.go — same canonical
// (clock, sender) ordering, same dispatch semantics, same byte-equality
// invariant on the resulting DOM.
//
// Two input sources:
//   - IndexedDB: replay.fromEventLog(log) — pulls via log.queryCanonicalOrder()
//   - File: replay.fromJSONLBlob(blob) — for cross-machine replay verification
//     (load a .jsonl exported from another tab/machine and reconstruct here)
//
// Two output modes:
//   - Apply-to-DOM: actually mutates state.canonical via host.js. Used for
//     "rebuild the prior session's DOM after page reload".
//   - Dry-run: returns the event sequence + each event's resulting node_id
//     for assertion-only tests (compare two runs without DOM side effects).
//
// Byte-equality assertion:
//   replay.verifyByteEquality(captured: regionId → contentHash,
//                              replayed: regionId → contentHash) → boolean
//
// Loaded by index.html after host.js + canonical_dom.js + event_log_indexeddb.js.
// Exposes global CognitiveReplay namespace.

(function (global) {
  'use strict';

  // ========================================================================
  // Canonical-order sort — matches internal/cognition/clock.go:CompareEvents
  // ========================================================================
  //
  // Events with the same Lamport clock are tiebroken by sender NodeID
  // (lex order). When both match, original insertion order is preserved
  // via stable-sort semantics. Returns a NEW array — does not mutate input.

  function canonicalSort(events) {
    const out = events.slice();
    out.sort(function (a, b) {
      const ac = a.clock || 0, bc = b.clock || 0;
      if (ac !== bc) return ac - bc;
      const as = a.sender || '', bs = b.sender || '';
      if (as < bs) return -1;
      if (as > bs) return 1;
      return 0; // tied — preserve order (Array.prototype.sort is stable in modern engines)
    });
    return out;
  }

  // ========================================================================
  // Event dispatchers — call the same paths real-time agents use
  // ========================================================================

  // dispatchEvent reapplies a single event against the canonical DOM via
  // the active CognitiveOS host. PatchApplied events drive the DOM
  // reconstruction; MessageSent/MessageReceived/TraceCaptured/CapabilityExceeded
  // are passive log entries — they don't affect the DOM, so we just
  // count them for the dry-run summary.
  //
  // Returns {kind, node_id} where node_id is the assigned ID (or null
  // for non-DOM events). Throws if dispatch fails — callers wrap.
  function dispatchEvent(ev) {
    if (!ev || typeof ev.kind !== 'string') {
      throw new Error('dispatchEvent: malformed event ' + JSON.stringify(ev));
    }
    switch (ev.kind) {
      case 'PatchApplied': {
        if (!global.CognitiveOS || typeof global.CognitiveOS._applyPatchDirect !== 'function') {
          throw new Error('dispatchEvent: CognitiveOS host not attached');
        }
        // Reconstruct the patch wire shape from the event fields.
        const patch = patchFromEvent(ev);
        const result = global.CognitiveOS._applyPatchDirect(ev.region, patch);
        return { kind: ev.kind, node_id: result.node_id, expected_node_id: ev.node_id };
      }
      case 'MessageSent':
      case 'MessageReceived':
      case 'TraceCaptured':
      case 'CapabilityExceeded':
        // Passive log entries — no DOM side effect. Future M-COG-RUNTIME-BROWSER
        // Subscribe wiring (M4) may re-invoke onMessage callbacks here.
        return { kind: ev.kind, node_id: null };
      default:
        // Forward-compat: unknown event kinds (from M-COG-MEMORY or
        // M-COG-MESH) are silently ignored during replay.
        return { kind: ev.kind, node_id: null };
    }
  }

  // patchFromEvent reconstructs the {ctor, fields} patch wire shape from
  // a PatchApplied event.
  //
  // M-COG-RUNTIME-BROWSER M3 Day 6: when the event has a `fields` array
  // (the JS-side extension over the Go-side PatchAppliedEvent struct),
  // use it directly — gives byte-identical content reconstruction.
  //
  // Legacy fallback: events without `fields` (older logs or Go-side
  // emissions before the schema extension) fall back to placeholder
  // content. This preserves DOM SHAPE byte-equality but not content;
  // the legacy path is documented as a follow-up gap.
  function patchFromEvent(ev) {
    if (Array.isArray(ev.fields) && ev.fields.length > 0) {
      return { ctor: ev.patch_type, fields: ev.fields.slice() };
    }
    // Legacy fallback — schema didn't include fields. Use the recorded
    // node_id as the content stand-in for shape-only reconstruction.
    switch (ev.patch_type) {
      case 'AddPanel':
        return { ctor: 'AddPanel', fields: ['(replayed)', ev.node_id || ''] };
      case 'UpdateNode':
        return { ctor: 'UpdateNode', fields: [ev.node_id || '', '(replayed)'] };
      case 'RemoveNode':
        return { ctor: 'RemoveNode', fields: [ev.node_id || ''] };
      case 'AddTimeline':
        return { ctor: 'AddTimeline', fields: ['(replayed)'] };
      default:
        return { ctor: 'Unknown', fields: [] };
    }
  }

  // ========================================================================
  // Public replay API
  // ========================================================================

  // fromEventLog reads canonical-order events from a CognitiveEventLog
  // (IndexedDB) and dispatches them. Returns a Promise<ReplayResult>.
  //
  // opts:
  //   - resetFirst: bool (default true) — clear DOM + canonical state
  //     before replay so the reconstruction starts from a clean slate
  //   - dryRun: bool (default false) — count events but don't apply
  //     them to the DOM (useful for analysis without side effects)
  function fromEventLog(log, opts) {
    if (!log || typeof log.queryCanonicalOrder !== 'function') {
      return Promise.reject(new Error('fromEventLog: log must have queryCanonicalOrder()'));
    }
    opts = opts || {};
    return log.queryCanonicalOrder().then(function (events) {
      return runReplay(events, opts);
    });
  }

  // fromJSONLBlob reads JSONL from a Blob (e.g. a downloaded export) and
  // dispatches the events. Used for cross-machine replay verification.
  function fromJSONLBlob(blob, opts) {
    if (!blob || typeof blob.text !== 'function') {
      return Promise.reject(new Error('fromJSONLBlob: blob must support .text()'));
    }
    opts = opts || {};
    return blob.text().then(function (text) {
      const events = [];
      const lines = text.split('\n');
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        if (line.length === 0) continue;
        try {
          events.push(JSON.parse(line));
        } catch (e) {
          throw new Error('fromJSONLBlob: line ' + (i + 1) + ' invalid JSON: ' + e.message);
        }
      }
      return runReplay(events, opts);
    });
  }

  // runReplay is the shared dispatch core. Returns a ReplayResult:
  //   {
  //     total: N,                          // events read
  //     applied: N,                        // events with side effects (PatchApplied)
  //     skipped: N,                        // events of unknown/passive kind
  //     dispatched: [{kind, node_id, expected_node_id}, ...]
  //   }
  function runReplay(events, opts) {
    opts = opts || {};
    const resetFirst = opts.resetFirst !== false; // default true
    const dryRun = !!opts.dryRun;

    if (resetFirst && global.CognitiveOS && typeof global.CognitiveOS._resetForTesting === 'function') {
      global.CognitiveOS._resetForTesting();
    }

    const ordered = canonicalSort(events);
    const result = { total: ordered.length, applied: 0, skipped: 0, dispatched: [] };

    for (let i = 0; i < ordered.length; i++) {
      const ev = ordered[i];
      if (dryRun) {
        result.dispatched.push({ kind: ev.kind, node_id: null, expected_node_id: ev.node_id || null });
        result.skipped++;
        continue;
      }
      try {
        const out = dispatchEvent(ev);
        result.dispatched.push(out);
        if (out.node_id !== null) result.applied++;
        else result.skipped++;
      } catch (e) {
        result.dispatched.push({ kind: ev.kind, error: e.message });
        result.skipped++;
      }
    }
    return result;
  }

  // ========================================================================
  // Byte-equality verification — content-hash all regions and compare
  // ========================================================================

  // captureRegionHashes returns a map of regionId → contentHash for every
  // active region in the canonical DOM. Used to snapshot DOM state
  // before/after a replay for equality assertion.
  function captureRegionHashes() {
    const result = {};
    if (!global.CognitiveOS || !global.CognitiveOS._canonical) {
      // Best-effort fallback: probe the CanonicalDOM instance via the
      // CognitiveOS public surface. M3 doesn't expose the instance
      // directly, but the debugSnapshot reveals known regions.
      const snap = (global.CognitiveOS && global.CognitiveOS.debugSnapshot) ? global.CognitiveOS.debugSnapshot() : { regions: {} };
      for (const regionId in snap.regions) {
        // Compute a simple content hash from the outerHTML — matches what
        // CanonicalDOM.regionHash would produce for the same input.
        // (For perfect parity with CanonicalDOM.regionHash, callers can
        // expose the canonical instance directly in M4.)
        result[regionId] = simpleStringHash(snap.regions[regionId]);
      }
    }
    return result;
  }

  // simpleStringHash is a deterministic 32-bit hash for outerHTML strings.
  // Used as a content fingerprint when CanonicalDOM.regionHash isn't
  // directly accessible.
  function simpleStringHash(s) {
    let h = 0x811c9dc5; // FNV-1a 32-bit offset
    for (let i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i);
      h = (h * 0x01000193) >>> 0;
    }
    return ('00000000' + h.toString(16)).slice(-8);
  }

  // verifyByteEquality compares two regionId→hash maps. Returns
  // {equal: bool, mismatches: [{regionId, expected, actual}]}.
  function verifyByteEquality(captured, replayed) {
    const result = { equal: true, mismatches: [] };
    // Check captured → replayed
    for (const regionId in captured) {
      if (!Object.prototype.hasOwnProperty.call(captured, regionId)) continue;
      const expected = captured[regionId];
      const actual = replayed[regionId];
      if (expected !== actual) {
        result.equal = false;
        result.mismatches.push({ regionId: regionId, expected: expected, actual: actual });
      }
    }
    // Check replayed → captured (regions present in replay but not original)
    for (const regionId in replayed) {
      if (!Object.prototype.hasOwnProperty.call(replayed, regionId)) continue;
      if (!Object.prototype.hasOwnProperty.call(captured, regionId)) {
        result.equal = false;
        result.mismatches.push({ regionId: regionId, expected: '(missing)', actual: replayed[regionId] });
      }
    }
    return result;
  }

  // ========================================================================
  // Export
  // ========================================================================

  global.CognitiveReplay = {
    fromEventLog: fromEventLog,
    fromJSONLBlob: fromJSONLBlob,
    captureRegionHashes: captureRegionHashes,
    verifyByteEquality: verifyByteEquality,
    // Exposed for direct testing
    _canonicalSort: canonicalSort,
    _dispatchEvent: dispatchEvent,
    _patchFromEvent: patchFromEvent,
    _simpleStringHash: simpleStringHash,
  };
})(typeof window !== 'undefined' ? window : globalThis);
