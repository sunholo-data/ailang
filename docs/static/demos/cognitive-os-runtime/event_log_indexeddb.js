// event_log_indexeddb.js — IndexedDB persistence sink (M3 Day 5, v0.21.x)
//
// Mirrors the Go-side internal/cognition.Sink interface from JavaScript:
// every cognitive event flowing through CognitiveOS gets transactionally
// written to IndexedDB so the log survives tab restarts and feeds the
// replay engine (replay.js, Day 6).
//
// Design freeze:
//   - Single object store "cognitive_events", primary key = monotonic counter
//   - Secondary index "by_clock_sender" on (clock, sender) for canonical-order queries
//   - DB-level version field with explicit upgrade callback for forward migration
//   - Writes serialized through a single Promise chain (no concurrent-write races)
//
// Wire shape (events flowing through `emit`):
//   {kind, clock, sender, ts_ms, ...variantFields}
//
// Schema matches internal/cognition/event_log.go's CognitiveEvent ADT —
// the Go-side ExportJSONL output and this JS sink produce structurally
// identical records (modulo language-native types).
//
// Usage from host.js:
//
//   const sink = new CognitiveEventLog({ dbName: 'ailang_cog_v1' });
//   await sink.open();
//   sink.emit({kind: 'MessageSent', clock: 1, sender: 'tab_a', ...});
//   const events = await sink.queryAll();
//   await sink.exportJSONL();  // Blob URL for download
//   await sink.clear();         // wipe (test reset)

(function (global) {
  'use strict';

  // ========================================================================
  // Schema constants — bumped on breaking IndexedDB layout changes
  // ========================================================================

  const SCHEMA_VERSION = 1;
  const STORE_NAME = 'cognitive_events';
  const INDEX_NAME = 'by_clock_sender';
  const DEFAULT_DB_NAME = 'ailang_cog_v1';

  // ========================================================================
  // CognitiveEventLog — wraps an IndexedDB database
  // ========================================================================

  function CognitiveEventLog(opts) {
    opts = opts || {};
    this.dbName = opts.dbName || DEFAULT_DB_NAME;
    this.db = null;
    this.writeChain = Promise.resolve(); // serializes Emit through one chain
    this.nextSeq = 0; // monotonic counter for primary keys (set during open())
  }

  // open initializes (or upgrades) the IndexedDB. Must be awaited before
  // any emit/query call. Returns a Promise<void>.
  //
  // The upgrade callback is the single point of truth for the schema:
  //   - v1: object store cognitive_events + index by_clock_sender
  //
  // Future migrations follow the standard IDB pattern: bump SCHEMA_VERSION
  // and add a `case oldVersion < N:` block in the onupgradeneeded callback.
  CognitiveEventLog.prototype.open = function () {
    const self = this;
    return new Promise(function (resolve, reject) {
      if (typeof indexedDB === 'undefined') {
        reject(new Error('IndexedDB unavailable in this runtime'));
        return;
      }
      const req = indexedDB.open(self.dbName, SCHEMA_VERSION);

      req.onupgradeneeded = function (ev) {
        const db = ev.target.result;
        const oldVersion = ev.oldVersion || 0;
        // Initial schema (v0 → v1)
        if (oldVersion < 1) {
          const store = db.createObjectStore(STORE_NAME, {
            keyPath: 'seq', // monotonic counter — assigned by emit()
          });
          store.createIndex(INDEX_NAME, ['clock', 'sender'], { unique: false });
        }
        // Future: if (oldVersion < 2) { ... migrate ... }
      };

      req.onsuccess = function (ev) {
        self.db = ev.target.result;
        // Determine next seq by reading the highest key currently in the store
        const tx = self.db.transaction(STORE_NAME, 'readonly');
        const store = tx.objectStore(STORE_NAME);
        const cursorReq = store.openCursor(null, 'prev'); // descending
        cursorReq.onsuccess = function (cev) {
          const c = cev.target.result;
          if (c) {
            self.nextSeq = (c.value.seq || 0) + 1;
          } else {
            self.nextSeq = 1;
          }
          resolve();
        };
        cursorReq.onerror = function () { resolve(); /* defensive: start at nextSeq=1 below */ };
      };

      req.onerror = function (ev) {
        reject(new Error('IndexedDB open failed: ' + (ev.target.error && ev.target.error.message)));
      };

      req.onblocked = function () {
        reject(new Error('IndexedDB open blocked — another tab holds an older version'));
      };
    });
  };

  // emit appends an event to the store transactionally. Returns a
  // Promise<seq> (the assigned monotonic key).
  //
  // Writes are serialized through this.writeChain so the transaction
  // ordering matches the call order. Per-emit transactions are tiny
  // (single put) — no batching, no risk of cross-event interleaving.
  CognitiveEventLog.prototype.emit = function (event) {
    const self = this;
    if (!event || typeof event !== 'object') {
      return Promise.reject(new Error('emit: event must be an object'));
    }
    if (typeof event.kind !== 'string' || event.kind.length === 0) {
      return Promise.reject(new Error('emit: event.kind missing'));
    }
    if (!self.db) {
      return Promise.reject(new Error('emit: open() not yet called'));
    }

    const chained = self.writeChain.then(function () {
      return new Promise(function (resolve, reject) {
        const seq = self.nextSeq++;
        const record = Object.assign({}, event, { seq: seq });
        // Defaults that simplify queries downstream
        if (typeof record.clock !== 'number') record.clock = 0;
        if (typeof record.sender !== 'string') record.sender = '';
        if (typeof record.ts_ms !== 'number') record.ts_ms = Date.now();

        const tx = self.db.transaction(STORE_NAME, 'readwrite');
        const store = tx.objectStore(STORE_NAME);
        const putReq = store.put(record);
        putReq.onsuccess = function () { resolve(seq); };
        putReq.onerror = function (ev) {
          reject(new Error('emit: put failed: ' + (ev.target.error && ev.target.error.message)));
        };
        tx.onerror = function (ev) {
          reject(new Error('emit: tx failed: ' + (ev.target.error && ev.target.error.message)));
        };
      });
    });
    // Replace the chain with one that swallows errors (so a single failed
    // write doesn't poison every subsequent emit) but still returns the
    // original promise to the caller for error handling.
    self.writeChain = chained.catch(function () { /* swallow for chain continuity */ });
    return chained;
  };

  // queryAll returns every event in monotonic seq order. The seq order
  // matches insertion order (since emit assigns seq monotonically), but
  // is NOT the canonical (clock, sender) replay order — callers that
  // need replay order should use queryCanonicalOrder.
  CognitiveEventLog.prototype.queryAll = function () {
    const self = this;
    if (!self.db) return Promise.reject(new Error('queryAll: open() not yet called'));
    return new Promise(function (resolve, reject) {
      const tx = self.db.transaction(STORE_NAME, 'readonly');
      const store = tx.objectStore(STORE_NAME);
      const events = [];
      const req = store.openCursor();
      req.onsuccess = function (ev) {
        const c = ev.target.result;
        if (c) {
          events.push(c.value);
          c.continue();
        } else {
          resolve(events);
        }
      };
      req.onerror = function (ev) {
        reject(new Error('queryAll: cursor failed: ' + (ev.target.error && ev.target.error.message)));
      };
    });
  };

  // queryCanonicalOrder returns every event in (clock, sender) order via
  // the by_clock_sender index. This is the order replay.js feeds to the
  // scheduler — matches the Go-side scheduler.RunFromLog sort key.
  CognitiveEventLog.prototype.queryCanonicalOrder = function () {
    const self = this;
    if (!self.db) return Promise.reject(new Error('queryCanonicalOrder: open() not yet called'));
    return new Promise(function (resolve, reject) {
      const tx = self.db.transaction(STORE_NAME, 'readonly');
      const store = tx.objectStore(STORE_NAME);
      const idx = store.index(INDEX_NAME);
      const events = [];
      const req = idx.openCursor();
      req.onsuccess = function (ev) {
        const c = ev.target.result;
        if (c) {
          events.push(c.value);
          c.continue();
        } else {
          resolve(events);
        }
      };
      req.onerror = function (ev) {
        reject(new Error('queryCanonicalOrder: index cursor failed: ' + (ev.target.error && ev.target.error.message)));
      };
    });
  };

  // count returns the number of events currently in the store.
  CognitiveEventLog.prototype.count = function () {
    const self = this;
    if (!self.db) return Promise.reject(new Error('count: open() not yet called'));
    return new Promise(function (resolve, reject) {
      const tx = self.db.transaction(STORE_NAME, 'readonly');
      const store = tx.objectStore(STORE_NAME);
      const req = store.count();
      req.onsuccess = function () { resolve(req.result); };
      req.onerror = function (ev) {
        reject(new Error('count: failed: ' + (ev.target.error && ev.target.error.message)));
      };
    });
  };

  // exportJSONL returns a Blob containing the cognitive event log in JSONL
  // format (one event per line, canonical (clock, sender) order). Matches
  // the Go-side EventLog.ExportJSONL byte-shape — events can be replayed
  // through the Go-side Replayer or vice versa.
  CognitiveEventLog.prototype.exportJSONL = function () {
    return this.queryCanonicalOrder().then(function (events) {
      const lines = events.map(function (e) {
        // Strip the JS-side `seq` field — it's a storage primary key,
        // not part of the canonical event shape. The Go-side JSONL
        // import skips unknown fields anyway, so this is defensive.
        const out = {};
        for (const k in e) {
          if (k !== 'seq' && Object.prototype.hasOwnProperty.call(e, k)) {
            out[k] = e[k];
          }
        }
        return JSON.stringify(out);
      });
      return new Blob([lines.join('\n')], { type: 'application/x-ndjson' });
    });
  };

  // clear wipes every event from the store. Used by tests and the
  // "Reset" demo button — the next open() will start with nextSeq=1.
  CognitiveEventLog.prototype.clear = function () {
    const self = this;
    if (!self.db) return Promise.reject(new Error('clear: open() not yet called'));
    return new Promise(function (resolve, reject) {
      const tx = self.db.transaction(STORE_NAME, 'readwrite');
      const store = tx.objectStore(STORE_NAME);
      const req = store.clear();
      req.onsuccess = function () {
        self.nextSeq = 1;
        resolve();
      };
      req.onerror = function (ev) {
        reject(new Error('clear: failed: ' + (ev.target.error && ev.target.error.message)));
      };
    });
  };

  // close releases the IndexedDB handle. Safe to call multiple times.
  CognitiveEventLog.prototype.close = function () {
    if (this.db) {
      try { this.db.close(); } catch (_) {}
      this.db = null;
    }
  };

  // Expose constants for tests
  CognitiveEventLog.SCHEMA_VERSION = SCHEMA_VERSION;
  CognitiveEventLog.STORE_NAME = STORE_NAME;
  CognitiveEventLog.INDEX_NAME = INDEX_NAME;

  global.CognitiveEventLog = CognitiveEventLog;
})(typeof window !== 'undefined' ? window : globalThis);
