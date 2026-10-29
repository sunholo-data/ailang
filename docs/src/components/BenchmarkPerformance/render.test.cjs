// Render actual React components against a small saved-evidence fixture.
// No provider, browser service, live JSON or model call is involved.
const fs = require('node:fs');
const test = require('node:test');
const assert = require('node:assert/strict');
const babel = require('@babel/core');
const React = require('react');
const { renderToStaticMarkup } = require('react-dom/server');

require.extensions['.jsx'] = (module, filename) => {
  const output = babel.transformSync(fs.readFileSync(filename, 'utf8'), {
    filename, babelrc: false, configFile: false,
    presets: [[require.resolve('@babel/preset-react'), { runtime: 'automatic' }]],
    // Match the docs bundle's loose spread transform: iterable Set values
    // must be converted explicitly rather than treated as arrays.
    plugins: [[require.resolve('@babel/plugin-transform-spread'), { loose: true }],
      require.resolve('@babel/plugin-transform-modules-commonjs')],
  });
  module._compile(output.code, filename);
};
require.extensions['.css'] = module => { module.exports = { __esModule: true,
  default: new Proxy({}, { get: (_, name) => `style_${String(name)}` }) }; };
const { PerformanceView } = require('./index.jsx');
const Table = require('./ComparisonTable.jsx').default;
const Detail = require('./EvidenceDetail.jsx').default;
const { evidenceRows } = require('../../lib/benchmarkEvidence.mjs');
const s = (runs, passes, coreCompleted) => ({ runs, passes, coreCompleted, benchmarks: coreCompleted,
  firstAttemptPasses: passes, firstAttemptUnknown: 0, repairs: 0, versions: { 'v0.saved': runs },
  lastRun: '2026-10-13T12:00:00Z', latestVersion: 'v0.saved', costs: {}, timing: {}, outcomes: {} });
const fixture = () => ({ version: 'v0.publication', timestamp: '2026-10-23T12:00:00Z', history: [
  { version: 'v0.saved', timestamp: '2026-10-13', totalRuns: 49, languageStats: { ailang: { total_runs: 49, success_rate: 1 } }, ratings: { models: { rated: 2000 }, anchor_version: 'a' } },
], benchmarkEvidence: { schemaVersion: 1, requiredCoreIds: ['c1','c2'], rows: [
  { id: 'argon', model: 'gemini-4-argon', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: s(49,49,['c1','c2']) } } },
  { id: 'astra', model: 'gpt6-astra', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: s(3,3,[]) } } },
  { id: 'rated', model: 'rated', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: s(2,1,['c1','c2']) } } },
] }, ratings: { standard: { byLang: { ailang: { models: [{ id: 'rated', elo: 2000 }] } } } } });

test('production React view renders Argon, provisional Astra and overall plots', () => {
  const latest = fixture();
  const html = renderToStaticMarkup(React.createElement(PerformanceView, { latest, rows: evidenceRows(latest), source: 'fallback' }));
  assert.match(html, /Gemini 4 Argon/);
  assert.match(html, /49 \/ 49 runs/);
  assert.match(html, /0\/2 Core tasks/);
  assert.match(html, /Undefeated on measured tasks/);
  assert.match(html, /Static fallback copy/);
  assert.match(html, /Model ELO over releases/);
  assert.match(html, /AILANG and Python pass rates/);
  assert.match(html, /Version not retained|v0.saved/);
  assert(!html.includes('stale (fallback'), 'static fallback is not automatically stale');
  assert(html.indexOf('Gemini 4 Argon') < html.indexOf('Rated models'));
});

test('React scoped cells never borrow all-task cost or fabricate missing Python', () => {
  const latest = fixture(), row = latest.benchmarkEvidence.rows[0];
  row.scopes.all.ailang.costs = { metered: { runs: 49, complete: true, perSuccessUsd: 9.999 } };
  row.scopes.core = { ailang: s(2,2,['c1','c2']) };
  const rows = evidenceRows(latest);
  const html = renderToStaticMarkup(React.createElement(Table, { rows: [rows[0]], scope: 'core', lang: 'ailang', sort: 'elo', showPython: true, onSelect() {} }));
  assert(!html.includes('$9.999'));
  assert.match(html, /Not measured/);
  assert(!html.includes('<strong>0.0%</strong>'), 'absent Python is not a measured zero');
  const detail = renderToStaticMarkup(React.createElement(Detail, { row: rows[0], scope: 'frontier', lang: 'ailang', onClose() {} }));
  assert.match(detail, /No measurements retained/);
  rows[0].scopes.all.ailang.outputTokens = 0;
  const tokens = renderToStaticMarkup(React.createElement(Detail, { row: rows[0], scope: 'all', lang: 'ailang', onClose() {} }));
  assert.match(tokens, /Output tokens<strong>Not retained/);
});
