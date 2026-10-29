import test from 'node:test';
import assert from 'node:assert/strict';
import { historyPoints, eloSeries } from './benchmarkHistory.mjs';

const snapshot = (version, anchor, value) => ({ version, timestamp: '2026-10-23',
  tiers: { core: { ailang_success_rate: 0, ailang_runs: 3, python_success_rate: 1, python_runs: 0 } },
  ratings: { anchor_version: anchor, models: value == null ? {} : { model: value } }, modelStats: { model: {} } });

test('release order is semantic, missing is null, measured zero remains zero', () => {
  const latest = { history: [snapshot('v0.10.0', 'a', 2000), snapshot('v0.9.0', 'a', null)] };
  const points = historyPoints(latest, 'core');
  assert.deepEqual(points.map(p => p.version), ['v0.9.0', 'v0.10.0']);
  assert.equal(points[0].ailang, 0);
  assert.equal(points[0].python, null);
  assert.equal(points[0].direction, null);
  assert.equal(latest.history[0].version, 'v0.10.0', 'saved snapshots not mutated');
});

test('ELO segments do not bridge aliases, missing samples or anchor changes', () => {
  const points = historyPoints({ history: [snapshot('v1.0.0','a',2000), snapshot('v1.0.1','b',2100), snapshot('v1.0.2','b',null)] });
  const { data, series } = eloSeries(points, ['model', 'new-model']);
  assert.equal(series.length, 2);
  assert.notEqual(series[0].key, series[1].key);
  assert.equal(data[0][series[1].key], undefined);
  assert.equal(data[1][series[0].key], undefined);
  assert.equal(data[2][series[1].key], undefined);
});

test('suite events retain their recorded version and task scope', () => {
  const h = snapshot('v1.0.0', 'a', 2000);
  h.tiers.core.benchmark_count = 23;
  const latest = { history: [h], events: [
    { version: 'v1.0.0', label: 'Core changed', affects_tiers: ['core'] },
    { version: 'v1.0.0', label: 'Frontier changed', affects_tiers: ['frontier'] },
    { version: 'v0.9.0', label: 'Earlier change' },
  ] };
  const [p] = historyPoints(latest, 'core');
  assert.equal(p.taskCount, 23);
  assert.deepEqual(p.events.map(e => e.label), ['Core changed']);
});
