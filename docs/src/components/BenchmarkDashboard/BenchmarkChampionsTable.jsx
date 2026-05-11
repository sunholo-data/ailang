import React, { useMemo, useState } from 'react';
import styles from './styles.module.css';

function formatModelName(name) {
  if (!name) return '';
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
 * BenchmarkChampionsTable — per-benchmark cheapest-pass + fastest-pass winners.
 *
 * Reads `sweet_spot_global.champions[]` (computed by the Go exporter from
 * BuildSweetSpot's per-benchmark champion identification).
 *
 * Each row: benchmark | cheapest model + cost | fastest model + TTS.
 * Sortable by any column. Default sort: benchmark name (alpha).
 *
 * Operator use: "I have benchmark X — which model should I use for cost?
 * For speed?" — direct answer in one row.
 */
export default function BenchmarkChampionsTable({ sweetSpotGlobal }) {
  const [sortBy, setSortBy] = useState('benchmark_id');
  const [sortDir, setSortDir] = useState('asc');

  const rows = useMemo(() => {
    const champions = sweetSpotGlobal?.champions || [];
    return [...champions].sort((a, b) => {
      const av = a[sortBy], bv = b[sortBy];
      if (av === bv) return 0;
      const dir = sortDir === 'asc' ? 1 : -1;
      return av < bv ? -dir : dir;
    });
  }, [sweetSpotGlobal, sortBy, sortDir]);

  if (rows.length === 0) {
    return (
      <div className={styles.chartContainer}>
        <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>
          No benchmark champions data yet. Re-run <code>ailang eval-report</code> with v0.19.0+ to populate.
        </p>
      </div>
    );
  }

  function handleSort(col) {
    if (sortBy === col) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(col);
      setSortDir(col === 'benchmark_id' ? 'asc' : 'asc');
    }
  }

  const arrow = (col) => sortBy === col ? (sortDir === 'asc' ? ' ▲' : ' ▼') : '';

  return (
    <div className={styles.chartContainer}>
      <h3 style={{ margin: '0 0 8px 0' }}>Cheapest / Fastest Pass per Benchmark</h3>
      <p style={{ fontSize: '0.9rem', color: 'var(--ifm-color-emphasis-700)', marginBottom: 12 }}>
        For each benchmark, the model that passes for the lowest cost and the model
        that passes the fastest. Same model can win both columns when it dominates.
      </p>

      <table className={styles.comparisonTable} style={{ width: '100%' }}>
        <thead>
          <tr>
            <th onClick={() => handleSort('benchmark_id')} style={{ textAlign: 'left', cursor: 'pointer' }}>
              Benchmark{arrow('benchmark_id')}
            </th>
            <th onClick={() => handleSort('cheapest_model')} style={{ textAlign: 'left', cursor: 'pointer' }}>
              Cheapest model{arrow('cheapest_model')}
            </th>
            <th onClick={() => handleSort('cheapest_cost_usd')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              Cost{arrow('cheapest_cost_usd')}
            </th>
            <th onClick={() => handleSort('fastest_model')} style={{ textAlign: 'left', cursor: 'pointer' }}>
              Fastest model{arrow('fastest_model')}
            </th>
            <th onClick={() => handleSort('fastest_tts_ms')} style={{ textAlign: 'right', cursor: 'pointer' }}>
              TTS{arrow('fastest_tts_ms')}
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map(r => {
            const sameWinner = r.cheapest_model === r.fastest_model;
            return (
              <tr key={r.benchmark_id}>
                <td><code>{r.benchmark_id}</code></td>
                <td>{formatModelName(r.cheapest_model)}</td>
                <td style={{ textAlign: 'right', fontFamily: 'monospace' }}>
                  ${(r.cheapest_cost_usd || 0).toFixed(4)}
                </td>
                <td style={{
                  color: sameWinner ? 'var(--ifm-color-emphasis-500)' : undefined,
                  fontStyle: sameWinner ? 'italic' : undefined,
                }}>
                  {sameWinner ? `↑ (same)` : formatModelName(r.fastest_model)}
                </td>
                <td style={{ textAlign: 'right', fontFamily: 'monospace' }}>
                  {((r.fastest_tts_ms || 0) / 1000).toFixed(1)}s
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
