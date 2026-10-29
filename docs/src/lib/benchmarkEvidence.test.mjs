import test from 'node:test';
import assert from 'node:assert/strict';
import { evidenceRows, scopeStats, sortRows, capabilityGroup, isProvisional, costMetric, timingMetric } from './benchmarkEvidence.mjs';

const stats = (runs, passes, core = []) => ({ runs, passes, benchmarks: core, coreCompleted: core,
  versions: {}, costs: {}, timing: {}, outcomes: {} });
const latest = () => ({ benchmarkEvidence: { schemaVersion: 1, requiredCoreIds: ['c1', 'c2'], rows: [
  { id: 'argon', model: 'gemini-4-argon', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: stats(49, 49, ['c1', 'c2']) } } },
  { id: 'astra', model: 'gpt6-astra', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: stats(3, 3) } } },
  { id: 'rated', model: 'rated', mode: 'standard', harness: 'direct_api', scopes: { all: { ailang: stats(10, 9, ['c1','c2']), python: stats(5, 0) } } },
  { id: 'agent', model: 'gemini-agents-argon', mode: 'agent', harness: 'gemini_agents', scopes: { all: { ailang: stats(2, 2) } } },
] }, ratings: { standard: { byLang: { ailang: { models: [{ id: 'rated', elo: 2500 }] } } } } });

test('undefeated group first, complete Core before small sample, numeric ELO unchanged', () => {
  const rows = evidenceRows(latest()).filter(r => r.mode === 'standard');
  const ordered = sortRows(rows, { sort: 'elo', lang: 'ailang', scope: 'all' });
  assert.deepEqual(ordered.map(r => r.id), ['argon', 'astra', 'rated']);
  assert.equal(capabilityGroup(ordered[0], 'ailang'), 0);
  assert.equal(isProvisional(ordered[0], 'ailang'), false);
  assert.equal(isProvisional(ordered[1], 'ailang'), true);
  assert.equal(scopeStats(ordered[2], 'all', 'python').rate, 0);
  assert.equal(scopeStats(ordered[0], 'all', 'python'), null);
});

test('cost sorting is independent and cost/timing preserve scope and kind', () => {
  const fixture = latest();
  fixture.benchmarkEvidence.rows[0].scopes.all.ailang.costs = { unknown: { runs: 49, complete: true, perSuccessUsd: .07 } };
  fixture.benchmarkEvidence.rows[2].scopes.all.ailang.costs = { 'free-local': { runs: 10, complete: true, perSuccessUsd: 0 } };
  const rows = evidenceRows(fixture).filter(r => r.mode === 'standard');
  assert.equal(sortRows(rows, { sort: 'cost', lang: 'ailang', scope: 'all' })[0].id, 'rated');
  assert.equal(costMetric(scopeStats(rows[0], 'core', 'ailang')).value, null);
  const s = scopeStats(rows[0], 'all', 'ailang');
  assert.equal(costMetric(s).kind, 'unknown');
  s.timing = { recorded_duration: { samples: 49, medianMs: 43 } };
  assert.equal(timingMetric(s).kind, 'recorded_duration');
  s.costs.metered = { runs: 1, complete: true, perSuccessUsd: .1 };
  assert.equal(costMetric(s).value, null, 'mixed cost categories are not one price');
});

test('legacy single-language rows stay visible without borrowing pooled metrics', () => {
  const rows = evidenceRows({ models: { argon: { aggregates: { zeroShotSuccess: .9, totalCostUSD: 10 },
    languages: { ailang: { totalRuns: 49, successRate: 1 } } } }, tiers: {}, benchmarks: {} });
  assert.equal(rows.length, 1);
  assert.equal(scopeStats(rows[0], 'all', 'ailang').passes, 49);
  assert.equal(costMetric(scopeStats(rows[0], 'all', 'ailang')).value, null);
  assert.equal(scopeStats(rows[0], 'all', 'ailang').firstAttemptPasses, null);
  assert.equal(rows[0].legacy, true);
});

test('ratings are not borrowed between modes or multiple harness identities', () => {
  const fixture = latest();
  const duplicate = structuredClone(fixture.benchmarkEvidence.rows[2]);
  duplicate.id = 'rated-other'; duplicate.harness = 'other';
  fixture.benchmarkEvidence.rows.push(duplicate);
  const rows = evidenceRows(fixture);
  assert.equal(rows.find(r => r.id === 'rated').ratings.ailang, undefined);
  assert.equal(rows.find(r => r.id === 'agent').ratings.ailang, undefined);
});

test('legacy gallery agent coverage cannot certify direct-model Core coverage', () => {
  const rows = evidenceRows({ models: { astra: { agentStats: { runs: 30 },
    languages: { ailang: { totalRuns: 3, successRate: 1, agentRuns: 30, agentSuccessRate: .97 } } } },
    benchmarks: { core_task: { tier: 'core', modelStats: { astra: { ailang: { runs: 1 } } } } } });
  const direct = rows.find(r => r.mode === 'standard');
  assert.equal(scopeStats(direct, 'all', 'ailang').coreCompleted, null);
  assert.equal(isProvisional(direct, 'ailang'), true);
});

test('unsupported additive schema errors visibly rather than guessing', () => {
  assert.throws(() => evidenceRows({ benchmarkEvidence: { schemaVersion: 99 } }), /Unsupported/);
});
