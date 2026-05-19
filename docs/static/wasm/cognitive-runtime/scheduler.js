// scheduler.js — Microtask-based JS event loop (M3 Day 7, v0.21.x)
//
// JS-side counterpart to internal/cognition/scheduler.go. Provides a
// single-threaded deterministic dispatch surface for browser-originated
// events (DOM clicks, BroadcastChannel arrivals, IndexedDB write
// callbacks) so the ordering matches the Go-side scheduler when both
// observe the same canonical (Lamport clock, sender NodeID) input.
//
// Key properties locked by the design freeze:
//
//   - Microtask-based, NEVER setTimeout/setInterval — preserves
//     replay determinism (setTimeout jitter would break it).
//   - Single-threaded: every handler runs to completion before the
//     next is dispatched. Handlers can call back into Subscribe /
//     Unsubscribe / Dispatch without lock-related deadlocks (the
//     mutation is staged into the next microtask, not applied mid-dispatch).
//   - Kind-filtered subscriptions: handlers register against a specific
//     event kind ("PatchApplied", "MessageReceived", etc.) or "" (all).
//   - Stable subscriber order: subscriptions get monotonically-increasing
//     IDs; dispatch iterates in registration order — matches the
//     Go-side scheduler.Dispatch + Subscribe semantics.
//
// This file does NOT replace replay.js's inline dispatch — replay
// already handles canonical ordering by sorting before iterating.
// Scheduler is a runtime abstraction for live events (DOM clicks,
// BroadcastChannel arrivals) where ordering must be enforced
// dynamically. M4 (Subscribe ops) is the primary consumer.
//
// Loaded by index.html after host.js. Exposes global CognitiveScheduler.

(function (global) {
  'use strict';

  // ========================================================================
  // CognitiveScheduler — single-threaded microtask-based event loop
  // ========================================================================

  function CognitiveScheduler(opts) {
    opts = opts || {};
    this.subs = new Map();            // id → {kind, handler}
    this.nextId = 0;                  // monotonic subscription ID
    this.pending = [];                // queue of events awaiting dispatch
    this.dispatching = false;         // re-entrance guard
    this.stopped = false;
    this.dispatched = [];             // capture log for determinism assertions
    this.captureDispatches = opts.captureDispatches !== false; // default true
  }

  // Subscribe registers a handler against an event kind. kind="" matches
  // every event. Returns a Subscription token used by Unsubscribe.
  CognitiveScheduler.prototype.subscribe = function (kind, handler) {
    if (typeof handler !== 'function') {
      // Defensive — silently ignore nil/non-function handlers. Matches
      // the Go-side scheduler's "return sentinel id -1" pattern.
      return { id: -1 };
    }
    this.nextId += 1;
    const id = this.nextId;
    this.subs.set(id, { kind: kind || '', handler: handler });
    return { id: id, kind: kind || '' };
  };

  // Unsubscribe removes a registered handler. Calling unsubscribe on an
  // unknown token is a no-op.
  CognitiveScheduler.prototype.unsubscribe = function (sub) {
    if (!sub || typeof sub.id !== 'number' || sub.id < 0) return;
    this.subs.delete(sub.id);
  };

  // Dispatch delivers an event to all matching subscribers in stable
  // registration order. If a dispatch is already in progress (e.g. a
  // handler called dispatch from within itself), the new event is
  // queued and processed after the current handler returns — this is
  // the re-entrance guard that keeps dispatch single-threaded.
  CognitiveScheduler.prototype.dispatch = function (event) {
    if (!event || typeof event.kind !== 'string') {
      // Defensive — skip malformed events without throwing.
      return;
    }
    if (this.stopped) return;
    if (this.dispatching) {
      // Re-entrant dispatch — queue for processing after the current
      // handler returns. Preserves single-threaded semantics.
      this.pending.push(event);
      return;
    }
    this.dispatching = true;
    try {
      this._dispatchNow(event);
      // Drain the pending queue if handlers re-dispatched anything.
      while (this.pending.length > 0) {
        const next = this.pending.shift();
        this._dispatchNow(next);
      }
    } finally {
      this.dispatching = false;
    }
  };

  // _dispatchNow does the actual handler invocation. Snapshot the
  // subscriber set first so handlers can call subscribe/unsubscribe
  // mid-dispatch without affecting the current iteration.
  CognitiveScheduler.prototype._dispatchNow = function (event) {
    // Stable iteration order via sorted IDs
    const ids = [];
    this.subs.forEach(function (_, id) { ids.push(id); });
    ids.sort(function (a, b) { return a - b; });

    for (let i = 0; i < ids.length; i++) {
      const sub = this.subs.get(ids[i]);
      // sub may have been unsubscribed by a previous handler in this
      // pass — skip it if so.
      if (!sub) continue;
      if (sub.kind !== '' && sub.kind !== event.kind) continue;
      try {
        sub.handler(event);
      } catch (err) {
        // Errors in one handler must not stop dispatch to the rest.
        // Log + continue. Matches the Go-side scheduler's tolerance
        // for handler failures.
        if (global.console && global.console.error) {
          global.console.error('[CognitiveScheduler] handler error:', err);
        }
      }
    }
    if (this.captureDispatches) {
      this.dispatched.push(event);
    }
  };

  // dispatchSoon schedules an event for dispatch on the next microtask.
  // Used by host.js when a DOM-event listener wants to enter the
  // scheduler's serial order — direct .dispatch() would inherit the
  // browser's event-handler call stack, which is undesirable for
  // determinism. Microtasks are processed in queue order before the
  // browser yields to the event loop, so handlers run with a clean
  // stack but predictable ordering.
  CognitiveScheduler.prototype.dispatchSoon = function (event) {
    const self = this;
    Promise.resolve().then(function () { self.dispatch(event); });
  };

  // dispatchedSnapshot returns a defensive copy of the dispatch capture
  // log. Used by determinism tests to assert two runs produce the same
  // dispatch sequence.
  CognitiveScheduler.prototype.dispatchedSnapshot = function () {
    return this.dispatched.slice();
  };

  // clearDispatched resets the capture log without affecting subscribers.
  CognitiveScheduler.prototype.clearDispatched = function () {
    this.dispatched = [];
  };

  // stop halts the scheduler. Subsequent dispatch calls become no-ops.
  // Idempotent.
  CognitiveScheduler.prototype.stop = function () {
    this.stopped = true;
    this.pending = [];
  };

  // ========================================================================
  // Equivalence check — verify two scheduler dispatch sequences match
  // ========================================================================

  // areEquivalent(a, b) compares two dispatched-event slices by
  // (kind, clock, sender) tuple. Returns {equal, mismatches: [{index, ...}]}.
  // Mirrors AreReplaysEquivalent / ReplayDivergence in
  // internal/cognition/replay.go.
  function areEquivalent(a, b) {
    const out = { equal: true, mismatches: [] };
    const min = Math.min(a.length, b.length);
    for (let i = 0; i < min; i++) {
      const x = a[i], y = b[i];
      if (x.kind !== y.kind || (x.clock || 0) !== (y.clock || 0) || (x.sender || '') !== (y.sender || '')) {
        out.equal = false;
        out.mismatches.push({ index: i, expected: { kind: x.kind, clock: x.clock, sender: x.sender }, actual: { kind: y.kind, clock: y.clock, sender: y.sender } });
      }
    }
    if (a.length !== b.length) {
      out.equal = false;
      out.mismatches.push({ index: min, expected: a.length > min ? a[min] : '(end)', actual: b.length > min ? b[min] : '(end)' });
    }
    return out;
  }

  // Export
  global.CognitiveScheduler = CognitiveScheduler;
  global.CognitiveScheduler.areEquivalent = areEquivalent;
})(typeof window !== 'undefined' ? window : globalThis);
