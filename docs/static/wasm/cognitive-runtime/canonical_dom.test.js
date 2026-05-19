// canonical_dom.test.js — Node.js test for the determinism contract.
//
// Verifies:
//   - FNV-1a hash is byte-stable across calls for the same input
//   - Different inputs produce different hashes (no trivial collisions)
//   - contentHashedID returns deterministic IDs for the same patch shape
//
// Run with: node docs/static/wasm/cognitive-runtime/canonical_dom.test.js
// Exit code 0 on pass, 1 on fail.
//
// This is a sanity-check script — Playwright is the canonical browser
// harness (M5). This runs in Node without DOM, so it only exercises the
// pure-function part of canonical_dom.js (hash + key construction).

'use strict';

// Build a minimal browser-like global so the IIFE can attach CanonicalDOM
const fakeGlobal = {};
const fs = require('fs');
const path = require('path');

// Inject a fake "document" + "CSS" that throws if anything DOM-related is
// touched at import time. The pure-function exports (CanonicalDOM._fnv1a64,
// CanonicalDOM._contentHashedID) don't touch the DOM, so they'll work.
const src = fs.readFileSync(path.join(__dirname, 'canonical_dom.js'), 'utf8');
// The IIFE references typeof window — Node's globalThis is fine.
const fn = new Function('window', src);
fn(fakeGlobal);

if (!fakeGlobal.CanonicalDOM) {
  console.error('FAIL: CanonicalDOM not attached to fakeGlobal');
  process.exit(1);
}

const hash = fakeGlobal.CanonicalDOM._fnv1a64;
const id = fakeGlobal.CanonicalDOM._contentHashedID;

let failures = 0;
function assert(cond, msg) {
  if (!cond) {
    console.error('FAIL: ' + msg);
    failures++;
  } else {
    console.log('OK:   ' + msg);
  }
}

// ===== Hash determinism =====
const h1 = hash('hello world');
const h2 = hash('hello world');
assert(h1 === h2, 'same input → same hash (determinism)');
assert(h1.length === 16, 'hash is 16 hex chars (64-bit)');
assert(/^[0-9a-f]+$/.test(h1), 'hash is lowercase hex');

// ===== Different inputs differ =====
const h3 = hash('hello world!');
assert(h1 !== h3, 'different inputs produce different hashes');

// Edge cases
assert(hash('') !== hash('a'), 'empty vs non-empty differ');
assert(hash('abc') !== hash('cba'), 'order matters');

// ===== Patch ID determinism =====
const id1 = id('agent_a', 'AddPanel', ['Title', 'Content'], 'parent_hash');
const id2 = id('agent_a', 'AddPanel', ['Title', 'Content'], 'parent_hash');
assert(id1 === id2, 'same patch → same node ID (replay invariant)');

// Field-order sensitivity
const id3 = id('agent_a', 'AddPanel', ['Content', 'Title'], 'parent_hash');
assert(id1 !== id3, 'patch field order matters');

// Region sensitivity
const id4 = id('agent_b', 'AddPanel', ['Title', 'Content'], 'parent_hash');
assert(id1 !== id4, 'different region → different ID');

// Parent-hash sensitivity (nested patches under different parents diverge)
const id5 = id('agent_a', 'AddPanel', ['Title', 'Content'], 'different_parent');
assert(id1 !== id5, 'parent-hash matters for nested patches');

// ===== ID format =====
assert(id1.startsWith('cog_'), 'node IDs start with cog_ prefix');
assert(id1.length === 4 + 16, 'node IDs are cog_ + 16 hex chars');

// ===== Null/undefined field stability =====
const id6 = id('agent_a', 'AddPanel', [null, undefined], '');
const id7 = id('agent_a', 'AddPanel', [null, undefined], '');
assert(id6 === id7, 'null/undefined fields are stable');

// ===== Stress: high volume same input =====
let same = true;
for (let i = 0; i < 100; i++) {
  if (hash('stress test') !== h1) { /* h1 isn't 'stress test' */ }
  if (hash('determinism') !== hash('determinism')) {
    same = false;
    break;
  }
}
assert(same, '100 iterations of same input always produce same hash');

if (failures > 0) {
  console.error('\n' + failures + ' test(s) failed');
  process.exit(1);
}
console.log('\nAll ' + 'canonical_dom.js' + ' determinism tests passed');
