import React, { useMemo, useState } from 'react';
import styles from './styles.module.css';

// Mirror of formatModelName in ValueScoreTable / SpeedRadar so the leaderboard
// labels read consistently across charts.
function formatModelName(name) {
  let s = name;
  let suffix = '';
  if (s.startsWith('motoko-or-'))   { suffix = ' (motoko · OR)'; s = s.slice('motoko-or-'.length); }
  else if (s.startsWith('motoko-')) { suffix = ' (motoko)';      s = s.slice('motoko-'.length); }
  else if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-'))    { suffix = ' (agent)';     s = s.slice('opencode-'.length); }
  else if (s.startsWith('pi-'))     { suffix = ' (Pi)'; s = s.slice('pi-'.length); }
  else if (s.startsWith('or-'))     { suffix = ' (OR)'; s = s.slice('or-'.length); }
  s = s
    .replace(/^claude-/, 'Claude ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt5/, 'GPT-5')
    .replace(/^glm-/, 'GLM ')
    .replace(/^kimi-/, 'Kimi ')
    .replace(/^qwen3-/, 'Qwen3 ')
    .replace(/^gemma-/, 'Gemma ')
    .replace(/^gemma4-/, 'Gemma4 ')
    .replace(/^deepseek-/, 'DeepSeek ')
    .replace(/-/g, ' ');
  return s + suffix;
}

/**
 * DollarsPerPassTable — the headline economic comparison.
 *
 * For each model with a `sweet_spot` block in latest.json, render
 * (model, $/pass, pass rate, total spend, Pareto frontier badge).
 *
 * Default-sorted ascending by $/pass (cheapest first) — so the first
 * row IS the recommended cost-optimal model.
 *
 * Toggle "show as ratio" replaces the $/pass column with
 * "model.$ / cheapest.$" — surfaces the 12.4× spread between cheap and
 * expensive models that's invisible in absolute dollars.
 *
 * Reads `models[name].sweet_spot` from latest.json. No client-side
 * recomputation — the Go exporter pre-computes everything.
 */
export default function DollarsPerPassTable({ models }) {
  const [showRatio, setShowRatio] = useState(false);
  const [sortBy, setSortBy] = useState('dollars_per_pass');
  const [sortDir, setSortDir] = useState('asc');

  const rows = useMemo(() => {
    const data = [];
    for (const [name, stats] of Object.entries(models || {})) {
      const ss = stats.sweet_spot;
      if (!ss) continue;
      // Skip rows with zero pass rate — they don't have meaningful $/pass.
      if (!ss.dollars_per_pass || ss.dollars_per_pass <= 0) continue;
      const agg = stats.aggregates || {};
      data.push({
        name,
        displayName: formatModelName(name),
        dollarsPerPass: ss.dollars_per_pass,
        passRate: ss.pass_rate,
        totalRuns: ss.total_runs,
        totalCost: agg.totalCostUSD || (stats.agentStats?.avgCost || 0) * (ss.total_runs || 0),
        pareto: !!ss.pareto_frontier,
      });
    }
    if (data.length === 0) return [];

    const cheapest = Math.min(...data.map(r => r.dollarsPerPass));
    for (const r of data) {
      r.ratio = r.dollarsPerPass / cheapest;
    }

    const dir = sortDir === 'asc' ? 1 : -1;
    data.sort((a, b) => {
      const av = a[sortBy], bv = b[sortBy];
      if (av === bv) return 0;
      return av < bv ? -dir : dir;
    });
    return data;
  }, [models, sortBy, sortDir]);

  if (rows.length === 0) {
    return (
      <div className={styles.chartContainer}>
        <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>
          No models with sweet_spot data yet. Re-run <code>ailang eval-report</code> with v0.19.0+ to populate.
        </p>
      </div>
    );
  }

  const headlineRatio = rows.length > 1
    ? Math.max(...rows.map(r => r.ratio))
    : null;

  function handleSort(col) {
    if (sortBy === col) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(col);
      setSortDir(col === 'dollars_per_pass' ? 'asc' : 'desc');
    }
  }

  const arrow = (col) => sortBy === col ? (sortDir === 'asc' ? ' ▲' : ' ▼') : '';

  return (
    <div className={styles.chartContainer}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 12 }}>
        <h3 style={{ margin: 0 }}>$/Pass Economics</h3>
        <label style={{ fontSize: '0.85rem', cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={showRatio}
            onChange={(e) => setShowRatio(e.target.checked)}
            style={{ marginRight: 6 }}
          />
          Show as ratio vs cheapest
        </label>
      </div>

      {headlineRatio !== null && headlineRatio > 2 && (
        <p style={{ fontSize: '0.95rem', color: 'var(--ifm-color-emphasis-700)', marginBottom: 12 }}>
          The most expensive passing model is <strong>{headlineRatio.toFixed(1)}×</strong> more expensive
          per success than the cheapest. Sort by $/pass to see the spread.
        </p>
      )}

      <table className={styles.comparisonTable} style={{ width: '100%' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'left' }}>Model</th>
            <th onClick={() => handleSort('dollars_per_pass')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              {showRatio ? `Ratio${arrow('dollars_per_pass')}` : `$/pass${arrow('dollars_per_pass')}`}
            </th>
            <th onClick={() => handleSort('passRate')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              Pass rate{arrow('passRate')}
            </th>
            <th onClick={() => handleSort('totalRuns')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              Runs{arrow('totalRuns')}
            </th>
            <th onClick={() => handleSort('totalCost')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              Total spend{arrow('totalCost')}
            </th>
            <th style={{ textAlign: 'center' }}>Frontier</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.name}>
              <td>{r.displayName}</td>
              <td style={{ textAlign: 'right', fontFamily: 'monospace' }}>
                {showRatio
                  ? `${r.ratio.toFixed(1)}×`
                  : `$${r.dollarsPerPass.toFixed(4)}`}
              </td>
              <td style={{ textAlign: 'right' }}>{(r.passRate * 100).toFixed(1)}%</td>
              <td style={{ textAlign: 'right' }}>{r.totalRuns}</td>
              <td style={{ textAlign: 'right', fontFamily: 'monospace' }}>${r.totalCost.toFixed(2)}</td>
              <td style={{ textAlign: 'center' }}>
                {r.pareto ? <span style={{ color: 'var(--ifm-color-success)' }}>✓</span> : <span style={{ color: 'var(--ifm-color-emphasis-400)' }}>—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginTop: 8 }}>
        <strong>$/pass</strong> = total cost / number of passing runs.{' '}
        <strong>Frontier</strong> ✓ means no other model has BOTH lower $/win AND lower median time-to-success.
      </p>
    </div>
  );
}
