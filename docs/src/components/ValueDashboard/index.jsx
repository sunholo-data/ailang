import React, { useEffect, useState } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import QualityScatter from '@site/src/components/BenchmarkDashboard/QualityScatter';
import ValueScoreTable from '@site/src/components/BenchmarkDashboard/ValueScoreTable';
import { buildCoverage } from '@site/src/components/BenchmarkDashboard/coverageGate';

import dashboardStyles from '@site/src/components/BenchmarkDashboard/styles.module.css';

/**
 * ValueDashboard — dedicated page for cost / quality / speed analysis.
 *
 * Three lenses on the same baseline data:
 *   1. Pass Rate vs Cost  (LMArena-style "score vs $")
 *   2. Pass Rate vs Speed (interactive vs batch tradeoff)
 *   3. Weighted Value Score table with N=1..4 quality weighting
 *
 * Reads from /benchmarks/latest.json (same source as the main Model Leaderboard).
 */
export default function ValueDashboard() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [mode, setMode] = useState('standard');

  useEffect(() => {
    benchmarkFetch('latest.json')
      .then(res => {
        if (!res.ok) throw new Error('Failed to load benchmark data');
        return res.json();
      })
      .then(setData)
      .catch(err => {
        console.error('Error loading benchmarks:', err);
        setError(err.message);
      });
  }, []);

  if (error) return <div className={dashboardStyles.error}>Error: {error}</div>;
  if (!data) return <div className={dashboardStyles.loading}>Loading benchmark data...</div>;

  const standardModels = data.models || {};
  // Agent set = agent-only models + dual-mode models (those that also ran agent).
  const agentModels = {
    ...(data.agentModels || {}),
    ...Object.fromEntries(
      Object.entries(standardModels).filter(([, m]) => m.agentStats && m.agentStats.runs)
    ),
  };
  const isAgent = mode === 'agent';
  const view = isAgent ? agentModels : standardModels;
  // M-EVAL-VALIDITY-DISCIPLINE (W2): coverage lookup so the value tables/scatter
  // can flag under-covered models provisional (no medals / dimmed) — a 9-benchmark
  // model must not win "best value" over one measured on the full 56.
  const coverage = buildCoverage(data.ratings);

  const tab = (m) => ({
    padding: '4px 14px', cursor: 'pointer', borderRadius: 6, fontWeight: 600,
    border: '1px solid var(--ifm-color-emphasis-300)',
    background: m === mode ? 'var(--ifm-color-primary)' : 'transparent',
    color: m === mode ? '#fff' : 'var(--ifm-color-emphasis-800)',
  });

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 14 }}>
        <button style={tab('standard')} onClick={() => setMode('standard')}>Standard</button>
        <button style={tab('agent')} onClick={() => setMode('agent')}>Agent</button>
        <span style={{ fontSize: '0.85em', color: 'var(--ifm-color-emphasis-600)' }}>
          {isAgent
            ? 'Agent mode — multi-turn agentic CLI (slower, higher cost/turns).'
            : 'Standard mode — 0-shot + self-repair via API (sub-second).'}
        </span>
      </div>

      <div className={dashboardStyles.section}>
        <h2>Quality (ELO) vs Cost <small>({isAgent ? 'agent' : 'standard'})</small></h2>
        <p className={dashboardStyles.sectionSubtitle}>
          Quality is the <strong>AILANG ELO rating</strong> — it separates the strong models that a raw
          pass rate saturates into one corner. NW corner = best value (high ELO, low cost). The dashed
          green line is the <strong>Pareto frontier</strong>; frontier models are labeled — hover any dot for the rest.
        </p>
        <QualityScatter models={view} xMetric="cost" mode={mode} coverage={coverage} ratings={data.ratings} />
      </div>

      <div className={dashboardStyles.section}>
        <h2>Quality (ELO) vs Speed <small>({isAgent ? 'agent' : 'standard'})</small></h2>
        <p className={dashboardStyles.sectionSubtitle}>
          AILANG ELO vs median time-to-success, <strong>split by mode</strong> — standard 0-shot is
          sub-second; agent multi-turn loops are seconds, so they are never blended. NW corner =
          fastest high-ELO models.
        </p>
        <QualityScatter models={view} xMetric="speed" mode={mode} coverage={coverage} ratings={data.ratings} />
      </div>

      <div className={dashboardStyles.section}>
        <ValueScoreTable models={view} mode={mode} coverage={coverage} ratings={data.ratings} />
        <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-500)' }}>
          Value Score reflects the selected <strong>{isAgent ? 'agent' : 'standard'}</strong> mode
          (toggle above).
        </p>
      </div>

      <div className={dashboardStyles.section}>
        <h3>How to read these charts</h3>
        <ul>
          <li>
            <strong>Cost-pure (N=1):</strong> Use to find the cheapest model that meets your
            quality bar. Best for batch processing or screening pipelines.
          </li>
          <li>
            <strong>Balanced (N=2):</strong> Default recommendation — squares the pass-rate so
            quality drops cost less than savings drop value. Good for general production.
          </li>
          <li>
            <strong>Quality-weighted (N=3, N=4):</strong> Use when accuracy matters more than
            spend — e.g. customer-facing code gen, regression-critical paths.
          </li>
          <li>
            <strong>Pareto frontier:</strong> Models on the dashed line are <em>provably optimal</em>
            for some tradeoff. Models off the frontier are <em>strictly dominated</em> (some other
            model is both cheaper/faster AND higher pass-rate).
          </li>
        </ul>

        <p style={{ marginTop: '1em' }}>
          <strong>Score formula:</strong>{' '}
          <code>pass_rate<sup>N</sup> / (cost_per_success × (1 + median_TTS_seconds / 60))</code>
        </p>
      </div>
    </div>
  );
}
