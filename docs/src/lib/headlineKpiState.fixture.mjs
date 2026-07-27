// Data-logic fixture for the headline-KPI card state derivation
// (M-COST-PER-SUCCESS-KPI, M3). Runs under plain Node (no jest/babel needed):
//
//   node docs/src/lib/headlineKpiState.fixture.mjs
//
// Exercises the four render states + the fallback-source signal. This is the
// browserless verification the sprint requires; the JSX card is verified
// separately by the Docusaurus build (syntax/schema).

import assert from 'node:assert/strict';
import {
  deriveHeadlineKpiState,
  formatUSD4,
  KPI_STATE_ABSENT,
  KPI_STATE_INCOMPLETE,
  KPI_STATE_ZERO_DENOM,
  KPI_STATE_AVAILABLE,
} from './headlineKpiState.js';

let passed = 0;
function check(name, fn) {
  fn();
  passed++;
  console.log(`  ok - ${name}`);
}

// ---- available ----
check('available: real number, gcs source (no fallback badge)', () => {
  const latest = {
    headlineKpis: {
      costPerVerifiedSuccess: {
        baseline_id: 'v1.0',
        available: true,
        total_runs: 4,
        verified_successes: 2,
        known_cost_usd: 0.4,
        quota_stages: 1,
        unknown_stages: 0,
        cost_per_verified_success_usd: 0.2,
        reason: '',
      },
    },
  };
  const s = deriveHeadlineKpiState(latest, 'gcs');
  assert.equal(s.state, KPI_STATE_AVAILABLE);
  assert.equal(s.isFallback, false);
  assert.equal(s.valueUSD, 0.2);
  assert.equal(s.verifiedSuccesses, 2);
  assert.equal(s.totalRuns, 4);
  assert.equal(formatUSD4(s.valueUSD), '$0.2000');
});

// ---- incomplete (unknown cost) ----
check('incomplete: available=false with unknown cost => INCOMPLETE, never $0', () => {
  const latest = {
    headlineKpis: {
      costPerVerifiedSuccess: {
        baseline_id: 'v1.0',
        available: false,
        reason: 'unknown_cost',
        total_runs: 3,
        verified_successes: 2,
        unknown_stages: 1,
        cost_per_verified_success_usd: 0, // must NOT be shown
      },
    },
  };
  const s = deriveHeadlineKpiState(latest, 'gcs');
  assert.equal(s.state, KPI_STATE_INCOMPLETE);
  assert.equal(s.valueUSD, null); // never surfaces the $0
  assert.equal(s.unknownStages, 1);
});

// ---- zero denominator ----
check('zero denominator: paid runs but 0 verified successes => ZERO_DENOM', () => {
  const latest = {
    headlineKpis: {
      costPerVerifiedSuccess: {
        baseline_id: 'v1.0',
        available: false,
        reason: 'zero_denominator',
        total_runs: 5,
        verified_successes: 0,
        cost_per_verified_success_usd: 0,
      },
    },
  };
  const s = deriveHeadlineKpiState(latest, 'gcs');
  assert.equal(s.state, KPI_STATE_ZERO_DENOM);
  assert.equal(s.totalRuns, 5);
  assert.equal(s.valueUSD, null);
});

// ---- absent ----
check('absent: no headlineKpis object => ABSENT (degrade, not break)', () => {
  const s = deriveHeadlineKpiState({ models: {}, aggregates: {} }, 'gcs');
  assert.equal(s.state, KPI_STATE_ABSENT);
  assert.equal(s.kpi, null);
});

check('absent: null latest.json => ABSENT', () => {
  const s = deriveHeadlineKpiState(null, undefined);
  assert.equal(s.state, KPI_STATE_ABSENT);
});

// ---- fallback source signal ----
check('fallback: static-copy source sets isFallback=true (badge trigger)', () => {
  const latest = {
    headlineKpis: {
      costPerVerifiedSuccess: {
        baseline_id: 'v1.0', available: true,
        total_runs: 1, verified_successes: 1, known_cost_usd: 0.1,
        cost_per_verified_success_usd: 0.1,
      },
    },
  };
  const gcs = deriveHeadlineKpiState(latest, 'gcs');
  const fb = deriveHeadlineKpiState(latest, 'fallback');
  assert.equal(gcs.isFallback, false);
  assert.equal(fb.isFallback, true);
  // fallback badge is independent of KPI completeness — an available KPI can
  // still be served stale.
  assert.equal(fb.state, KPI_STATE_AVAILABLE);
});

// ---- formatUSD4 guards ----
check('formatUSD4: null/NaN => em dash', () => {
  assert.equal(formatUSD4(null), '—');
  assert.equal(formatUSD4(NaN), '—');
  assert.equal(formatUSD4(1.5), '$1.5000');
});

console.log(`\nheadlineKpiState.fixture: ${passed} checks passed`);
