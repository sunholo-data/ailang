import React, { useState, useMemo } from 'react';
import styles from './styles.module.css';

// Unified formatModelName mirrors v0.15.0 hotfix.
function formatModelName(name) {
  let s = name;
  let suffix = '';
  if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-')) { suffix = ' (agent)';     s = s.slice('opencode-'.length); }
  else if (s.startsWith('pi-'))       { suffix = ' (Pi)';        s = s.slice('pi-'.length); }
  else if (s.startsWith('or-'))       { suffix = ' (OR)';        s = s.slice('or-'.length); }
  s = s
    .replace(/^claude-/, 'Claude ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt5/, 'GPT-5')
    .replace(/^minimax-/, 'MiniMax ')
    .replace(/^glm-/, 'GLM ')
    .replace(/^kimi-/, 'Kimi ')
    .replace(/^qwen3-/, 'Qwen3 ')
    .replace(/^gemma4-/, 'Gemma4 ')
    .replace(/^deepseek-/, 'DeepSeek ')
    .replace(/-/g, ' ');
  return s + suffix;
}

/**
 * ValueScoreTable — combined cost/quality/speed leaderboard.
 *
 * Score = (elo / best_elo)^N / (cost_per_success × (1 + median_TTS_seconds / 60))
 *
 * Quality is the AILANG ELO (normalised to the best model in view), not raw pass
 * rate — ELO separates the strong models a saturated pass rate collapses together.
 * N=1 leans toward cheap models; N=3 weights quality more heavily.
 * The "Pareto" column flags non-dominated models — i.e. no other model is
 * BOTH cheaper AND higher-ELO.
 *
 * Reads three sources from latest.json per model:
 *   - aggregates.finalSuccess + aggregates.totalCostUSD + totalRuns → pass%, $/success
 *   - efficiency.median_time_to_success_ms → speed factor
 *   - reliability for adjusted (api-error-excluded) pass rate when available
 */
export default function ValueScoreTable({ models, mode = 'standard', coverage, ratings }) {
  const [weighting, setWeighting] = useState(2); // N
  const [sortBy, setSortBy] = useState('score');
  const [sortDir, setSortDir] = useState('desc');
  const isAgent = mode === 'agent';

  // Per-model AILANG ELO for this mode (prefer the AILANG-specific fit). ELO is the
  // quality basis — it discriminates the top models a raw pass rate saturates.
  const eloOf = useMemo(() => {
    const r = (ratings && ratings[mode]) || {};
    const block = (r.byLang && r.byLang.ailang) || r;
    const m = {};
    (block.models || []).forEach((x) => { if (x && x.id != null) m[x.id] = x.elo; });
    return m;
  }, [ratings, mode]);

  const rows = useMemo(() => {
    const data = [];
    for (const [name, stats] of Object.entries(models || {})) {
      const agg = stats.aggregates || {};
      const as = stats.agentStats || null;
      // Per-mode fields — never blend standard and agent (standard 0-shot vs agent multi-turn).
      const eff = (isAgent ? stats.efficiencyAgent : stats.efficiencyStandard) || {};

      let passRate;
      let totalRuns;
      let totalCost;
      if (isAgent) {
        if (!as || !as.runs) continue;
        totalRuns = as.runs;
        passRate = as.successRate ?? 0;
        totalCost = (as.avgCost || 0) * totalRuns;
      } else {
        totalRuns = agg.totalRuns || stats.totalRuns || 0;
        passRate = agg.finalSuccess ?? 0;
        totalCost = agg.totalCostUSD || 0;
      }
      const successes = passRate * totalRuns;
      const costPerSuccess = successes > 0 ? totalCost / successes : null;
      const ttsMs = eff.median_time_to_success_ms || 0;
      const ttsSec = ttsMs / 1000;

      if (totalRuns < 10 || passRate <= 0 || costPerSuccess == null) continue; // skip low-sample models
      const elo = eloOf[name];
      if (elo == null) continue; // need an ELO rating to score quality

      const timeFactor = 1 + ttsSec / 60; // 1-min baseline; lower is better
      data.push({
        name,
        elo,
        passRate,
        costPerSuccess,
        ttsSec,
        timeFactor,
        totalRuns,
        totalCost,
        // M-EVAL-VALIDITY-DISCIPLINE (W2): under-covered models earn no medal and
        // sort last — a 9-benchmark model must not win "best value" over a 56.
        provisional: coverage ? coverage.isProvisional(name) : false,
        benchmarks: coverage ? coverage.benchmarksFor(name) : null,
      });
    }

    // Pareto frontier: dominated if any other has higher ELO AND lower cost.
    for (const m of data) {
      m.pareto = !data.some(o =>
        o.name !== m.name && o.elo >= m.elo && o.costPerSuccess < m.costPerSuccess
      );
    }

    // Weighted value score. Quality = ELO normalised to the best model in view
    // (elo/maxElo, ~0.5–1.0), so N still tunes cost-vs-quality the way it did with
    // pass rate — but now it reflects the ELO gaps a saturated pass rate hides.
    const maxElo = Math.max(1, ...data.map(m => m.elo || 0));
    for (const m of data) {
      const q = m.elo / maxElo;
      m.score = costPerSuccessSafe(m) > 0
        ? Math.pow(q, weighting) / (costPerSuccessSafe(m) * m.timeFactor)
        : 0;
    }

    // Sort
    const sorted = [...data].sort((a, b) => {
      // Provisional (low-coverage) models always sort last so they can't take a medal.
      if (a.provisional !== b.provisional) return a.provisional ? 1 : -1;
      const dir = sortDir === 'desc' ? -1 : 1;
      if (sortBy === 'name') return a.name.localeCompare(b.name) * dir;
      return ((a[sortBy] ?? 0) - (b[sortBy] ?? 0)) * dir;
    });
    return sorted;
  }, [models, weighting, sortBy, sortDir, isAgent, coverage, eloOf]);

  const handleSort = (col) => {
    if (sortBy === col) setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
    else { setSortBy(col); setSortDir('desc'); }
  };

  const sortIndicator = (col) => sortBy === col ? (sortDir === 'desc' ? ' ↓' : ' ↑') : '';

  return (
    <div className={styles.chartContainer} style={{ padding: '16px', maxWidth: '100%' }}>
      <h3 style={{ marginTop: 0 }}>Value Score Leaderboard</h3>
      <p style={{ fontSize: '0.9em', color: 'var(--ifm-color-emphasis-700)' }}>
        Combined cost-quality-speed score:{' '}
        <code>(ELO / best_ELO)<sup>N</sup> / (cost_per_success × (1 + sec_to_solution/60))</code>.{' '}
        Higher = better. Quality is <strong>AILANG ELO</strong> — it separates the top models a raw
        pass rate saturates. <strong>Pareto ⭐</strong> models are not dominated by any other on the
        cost+ELO plane (nothing is both cheaper AND higher-ELO).
      </p>

      <div style={{ marginBottom: 12 }}>
        <label style={{ marginRight: 8, fontWeight: 600 }}>Quality weighting (N):</label>
        {[1, 2, 3, 4].map(n => (
          <button
            key={n}
            onClick={() => setWeighting(n)}
            style={{
              marginRight: 6,
              padding: '4px 10px',
              background: weighting === n ? 'var(--ifm-color-primary)' : 'var(--ifm-color-emphasis-200)',
              color: weighting === n ? 'white' : 'var(--ifm-color-emphasis-800)',
              border: 'none',
              borderRadius: 4,
              cursor: 'pointer',
              fontSize: '0.85em',
            }}
          >
            N={n} {n === 1 ? '(cost)' : n === 2 ? '(balanced)' : n === 3 ? '(quality+)' : '(quality)'}
          </button>
        ))}
      </div>

      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
          <thead>
            <tr style={{ background: 'var(--ifm-color-emphasis-100)', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
              <th style={th} onClick={() => handleSort('name')}>Model{sortIndicator('name')}</th>
              <th style={th} onClick={() => handleSort('elo')}>ELO{sortIndicator('elo')}</th>
              <th style={th} onClick={() => handleSort('passRate')}>Pass %{sortIndicator('passRate')}</th>
              <th style={th} onClick={() => handleSort('costPerSuccess')}>$/success{sortIndicator('costPerSuccess')}</th>
              <th style={th} onClick={() => handleSort('ttsSec')}>Med. Time{sortIndicator('ttsSec')}</th>
              <th style={th} onClick={() => handleSort('score')}>Score (N={weighting}){sortIndicator('score')}</th>
              <th style={th}>Frontier</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((m, idx) => (
              <tr key={m.name} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)', ...(m.provisional ? { opacity: 0.6, fontStyle: 'italic' } : {}) }}>
                <td style={td}>
                  <span style={{ marginRight: 6 }}>
                    {!m.provisional && sortBy === 'score' && idx === 0 ? '🥇' : !m.provisional && sortBy === 'score' && idx === 1 ? '🥈' : !m.provisional && sortBy === 'score' && idx === 2 ? '🥉' : ''}
                  </span>
                  {formatModelName(m.name)}
                  {m.provisional && (
                    <span
                      title={`Provisional: ran ${m.benchmarks ?? '?'}/${coverage ? coverage.maxCoverage : '?'} benchmarks — not comparable to full-coverage models; no medal awarded.`}
                      style={{ marginLeft: 6, fontSize: '0.7em', padding: '1px 5px', borderRadius: 3, background: 'var(--ifm-color-warning-lightest)', color: 'var(--ifm-color-warning-darkest)', border: '1px solid var(--ifm-color-warning-light)', cursor: 'help', fontWeight: 600, fontStyle: 'normal' }}
                    >
                      ⚠ {m.benchmarks != null ? `${m.benchmarks}/${coverage ? coverage.maxCoverage : '?'}` : 'low cov'}
                    </span>
                  )}
                </td>
                <td style={{ ...tdNum, fontWeight: 600 }}>{Math.round(m.elo)}</td>
                <td style={tdNum}>{(m.passRate * 100).toFixed(1)}%</td>
                <td style={tdNum}>${m.costPerSuccess.toFixed(4)}</td>
                <td style={tdNum}>{m.ttsSec > 0 ? `${formatSeconds2sf(m.ttsSec)}s` : '—'}</td>
                <td style={{ ...tdNum, fontWeight: 600 }}>{m.score.toFixed(1)}</td>
                <td style={td}>
                  {m.pareto ? (
                    <span title="Non-dominated: no other model is both cheaper AND higher-ELO">⭐ Pareto</span>
                  ) : (
                    <span style={{ color: 'var(--ifm-color-emphasis-500)' }}>—</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div style={{ marginTop: 12, fontSize: '0.85em', color: 'var(--ifm-color-emphasis-700)' }}>
        Time component uses median wall-clock-to-success across runs (falls back to total
        duration when SuccessAtMs not measured). Models with &lt;10 runs are filtered.
      </div>
    </div>
  );
}

function costPerSuccessSafe(m) { return Math.max(1e-6, m.costPerSuccess || 0); }

// formatSeconds2sf — display wall-clock with 2 significant figures so
// sub-second values don't disappear (the standard-eval rows would
// otherwise round to "0s" and look like missing data).
//   0.156s → "0.16"
//   1.23s  → "1.2"
//   12.4s  → "12"
//   100s   → "100"
function formatSeconds2sf(s) {
  if (s <= 0) return '0';
  if (s >= 100) return s.toFixed(0);
  if (s >= 10) return s.toFixed(0);
  if (s >= 1) return s.toFixed(1);
  return s.toFixed(2);
}

const th = {
  padding: '8px 12px',
  textAlign: 'left',
  cursor: 'pointer',
  fontSize: '0.85em',
  fontWeight: 600,
  whiteSpace: 'nowrap',
  userSelect: 'none',
};

const td = {
  padding: '8px 12px',
  textAlign: 'left',
};

const tdNum = {
  ...td,
  textAlign: 'right',
  fontVariantNumeric: 'tabular-nums',
};
