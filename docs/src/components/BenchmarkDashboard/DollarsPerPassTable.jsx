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
  // langView: 'all' uses the pooled sweet_spot; 'ailang' / 'python' uses
  // models[name].sweet_spot_by_lang[langView]. Discover available langs
  // from the data so the toggle reflects what's actually computed.
  const [langView, setLangView] = useState('all');

  const availableLangs = useMemo(() => {
    const set = new Set();
    for (const stats of Object.values(models || {})) {
      const byLang = stats?.sweet_spot_by_lang;
      if (!byLang) continue;
      for (const k of Object.keys(byLang)) set.add(k);
    }
    return Array.from(set).sort();
  }, [models]);

  const rows = useMemo(() => {
    const data = [];
    for (const [name, stats] of Object.entries(models || {})) {
      // Pick which sweet-spot block to read: pooled vs per-lang.
      const ss = langView === 'all'
        ? stats.sweet_spot
        : (stats.sweet_spot_by_lang && stats.sweet_spot_by_lang[langView]);
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
        // "$-overhead" = median ratio of this-model-cost / cheapest-passer-cost,
        // computed per benchmark. 1.0 = matched the cheapest passer on every
        // benchmark this model passed. Distinguishes pricing tax from
        // inefficiency in concert with the tokens-overhead column.
        costOverhead: ss.cost_overhead_vs_best || 0,
        tokenOverhead: ss.token_overhead_vs_best || 0,
        // pareto_frontier is computed within the selected language's subset
        // when langView != 'all', so this flag is language-relative.
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
  }, [models, sortBy, sortDir, langView]);

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
      <div className={styles.sweetSpotHeader}>
        <h3>$/Pass Economics</h3>
        <div style={{ display: 'flex', gap: '1rem', alignItems: 'center', flexWrap: 'wrap' }}>
          {availableLangs.length > 0 && (
            <label className={styles.sweetSpotToggle}>
              Language:{' '}
              <select
                value={langView}
                onChange={(e) => setLangView(e.target.value)}
                style={{ marginLeft: '0.25rem' }}
              >
                <option value="all">All (pooled)</option>
                {availableLangs.map(l => (
                  <option key={l} value={l}>{l.charAt(0).toUpperCase() + l.slice(1)} only</option>
                ))}
              </select>
            </label>
          )}
          <label className={styles.sweetSpotToggle}>
            <input
              type="checkbox"
              checked={showRatio}
              onChange={(e) => setShowRatio(e.target.checked)}
            />
            Show as ratio vs cheapest
          </label>
        </div>
      </div>
      {langView !== 'all' && (
        <p className={styles.sweetSpotHeadlineNote}>
          Showing <strong>{langView}</strong> <strong>agent-mode</strong> runs only (standard 0-shot
          excluded, since it has no iteration loop and isn't comparable). Pareto frontier, $-Ovhd,
          and Tok-Ovhd are computed within the <strong>{langView}</strong>-only subset — a model can
          be on the frontier for one language but dominated for another.
        </p>
      )}

      {headlineRatio !== null && headlineRatio > 2 && (
        <p className={styles.sweetSpotHeadlineNote}>
          The most expensive passing model is <strong>{headlineRatio.toFixed(1)}×</strong> more expensive
          per success than the cheapest. Sort by $/pass to see the spread.
        </p>
      )}

      <div className={styles.tableWrapper}>
        <table className={styles.comparisonTable}>
          <thead>
            <tr>
              <th>Model</th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('dollars_per_pass')}
                style={{ textAlign: 'right' }}
              >
                {showRatio ? `Ratio${arrow('dollars_per_pass')}` : `$/pass${arrow('dollars_per_pass')}`}
              </th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('costOverhead')}
                style={{ textAlign: 'right' }}
                title="Median ratio of this model's cost per benchmark vs the cheapest passer of that same benchmark. 1.0× = matched the cheapest passer everywhere. High = expensive per token."
              >
                $-Ovhd{arrow('costOverhead')}
              </th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('tokenOverhead')}
                style={{ textAlign: 'right' }}
                title="Same shape as $-Ovhd, but using token counts. 1.0× = solved every benchmark in the fewest tokens. High = inefficient iteration (lots of wasted turns)."
              >
                Tok-Ovhd{arrow('tokenOverhead')}
              </th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('passRate')}
                style={{ textAlign: 'right' }}
              >
                Pass rate{arrow('passRate')}
              </th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('totalRuns')}
                style={{ textAlign: 'right' }}
              >
                Runs{arrow('totalRuns')}
              </th>
              <th
                className={styles.sweetSpotSortable}
                onClick={() => handleSort('totalCost')}
                style={{ textAlign: 'right' }}
              >
                Total spend{arrow('totalCost')}
              </th>
              <th style={{ textAlign: 'center' }}>Frontier</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(r => (
              <tr key={r.name}>
                <td>{r.displayName}</td>
                <td className={styles.sweetSpotNumCell}>
                  {showRatio
                    ? `${r.ratio.toFixed(1)}×`
                    : `$${r.dollarsPerPass.toFixed(4)}`}
                </td>
                <td className={styles.sweetSpotNumCell}>
                  {r.costOverhead > 0 ? (r.costOverhead >= 100 ? `${r.costOverhead.toFixed(0)}×` : `${r.costOverhead.toFixed(1)}×`) : '—'}
                </td>
                <td className={styles.sweetSpotNumCell}>
                  {r.tokenOverhead > 0 ? (r.tokenOverhead >= 100 ? `${r.tokenOverhead.toFixed(0)}×` : `${r.tokenOverhead.toFixed(1)}×`) : '—'}
                </td>
                <td className={styles.sweetSpotNumCell}>{(r.passRate * 100).toFixed(1)}%</td>
                <td className={styles.sweetSpotNumCell}>{r.totalRuns}</td>
                <td className={styles.sweetSpotNumCell}>${r.totalCost.toFixed(2)}</td>
                <td className={styles.sweetSpotCenterCell}>
                  {r.pareto
                    ? <span className={styles.sweetSpotFrontierYes}>✓</span>
                    : <span className={styles.sweetSpotFrontierNo}>—</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className={styles.sweetSpotFootnote}>
        <strong>$/pass</strong> = total cost / number of passing runs.{' '}
        <strong>$-Ovhd</strong> = median ratio of this model's cost vs the cheapest passer per benchmark
        (1.0× = matched the cheapest on every benchmark; high = expensive per token).{' '}
        <strong>Tok-Ovhd</strong> = same shape using token counts
        (1.0× = token-optimal; high = inefficient iteration).{' '}
        A model with <em>low Tok-Ovhd but high $-Ovhd</em> thinks efficiently but pays a per-token-pricing tax;
        a model with <em>low $-Ovhd but high Tok-Ovhd</em> is cheap because of its provider's pricing,
        not because it's solving things efficiently.{' '}
        <strong>Frontier</strong> ✓ means no other model has BOTH lower $/win AND lower median time-to-success.
      </p>
    </div>
  );
}
