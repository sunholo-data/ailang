import React, { useState, useEffect } from 'react';
import { ArrowUpDown, TrendingUp, TrendingDown } from 'lucide-react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import styles from './styles.module.css';

// Short label for an on-device model string, e.g.
// "motoko-local-qwen3-6-35b-a3b-mxfp8" -> "Qwen3.6". Falls back to the raw model.
function shortLocalModel(model) {
  const m = /qwen3-(\d+)/.exec(model || '');
  return m ? `Qwen3.${m[1]}` : (model || 'local');
}

function formatModelName(name) {
  // Surface harness + provider as explicit suffixes. See
  // BenchmarkExplorer/index.jsx::modelShort for the canonical version.
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

export default function ModelComparisonTable({ models, coverage, showLocalAgent = false }) {
  const [sortColumn, setSortColumn] = useState('ailangSuccess');
  const [sortDirection, setSortDirection] = useState('desc');

  // Optional on-device "Local GPU agent" row (M-EVAL): the rig's best agentic
  // config (opencode/pi/motoko on a local qwen) fetched from os/latest.json. It's
  // agent-mode + ~$0/run — a slow, free option — so we surface only its AILANG
  // success against the 0-shot cloud rows (the thesis), not a full head-to-head.
  const [localRow, setLocalRow] = useState(null);
  useEffect(() => {
    if (!showLocalAgent) return;
    let alive = true;
    benchmarkFetch('os/latest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then((os) => {
        if (!alive || !os || !Array.isArray(os.rows)) return;
        let best = null;
        for (const row of os.rows) {
          const a = row.lang && row.lang.ailang;
          if (typeof a !== 'number') continue;
          if (!best || a > best.ailang) best = { ailang: a, harness: row.harness, model: row.model };
        }
        if (best) {
          setLocalRow({
            ailangSuccess: best.ailang * 100,
            harness: best.harness || 'agent',
            model: shortLocalModel(best.model),
            configCount: os.rows.length,
            version: os.ailang_version || os.version || '',
          });
        }
      })
      .catch(() => { /* os data optional — omit the row on failure */ });
    return () => { alive = false; };
  }, [showLocalAgent]);

  // Transform models data into table rows
  const tableData = Object.entries(models)
    .filter(([name, stats]) => {
      return stats.languages && stats.languages.ailang && stats.languages.python;
    })
    .map(([name, stats]) => {
      const ailang = stats.languages?.ailang;
      const python = stats.languages?.python;
      const ailangSuccess = (ailang?.successRate || 0) * 100;
      const pythonSuccess = (python?.successRate || 0) * 100;
      const ailangTokens = ailang?.avgTokens || 0;
      const pythonTokens = python?.avgTokens || 1; // Avoid div by zero
      const gap = ailangSuccess - pythonSuccess;

      return {
        modelName: name,
        displayName: formatModelName(name),
        ailangSuccess: ailangSuccess,
        ailangRuns: ailang?.totalRuns || 0,
        ailangTokens: Math.round(ailangTokens),
        pythonSuccess: pythonSuccess,
        pythonRuns: python?.totalRuns || 0,
        pythonTokens: Math.round(python?.avgTokens || 0),
        gap: gap,
        tokenRatio: ailangTokens / pythonTokens,
      };
    });

  // Sample-size sanity check: lang_harness_suite models (claude-haiku-4-5,
  // opencode-haiku) only ran the core tier (~23 benchmarks per language)
  // while agent_suite/extended_suite models ran core+stretch (~34 per
  // language). Same percentages computed on different benchmark sets are not
  // directly comparable — flag the partial samples so readers don't conclude
  // "haiku is the best model" when it actually skipped the hardest 11 tasks.
  const fullRuns = Math.max(0, ...tableData.map(r => Math.max(r.ailangRuns, r.pythonRuns)));
  for (const row of tableData) {
    const minRuns = Math.min(row.ailangRuns || 0, row.pythonRuns || 0);
    row.isPartialSample = fullRuns > 0 && minRuns > 0 && minRuns < fullRuns;
    row.runs = minRuns;
    // M-EVAL-VALIDITY-DISCIPLINE (W2): prefer TRUE benchmark coverage (distinct
    // benchmarks from the ratings block) over the run-count heuristic when present.
    row.benchmarks = coverage ? coverage.benchmarksFor(row.modelName) : null;
    row.provisional = coverage ? coverage.isProvisional(row.modelName) : row.isPartialSample;
  }
  const maxCoverage = coverage ? coverage.maxCoverage : fullRuns;

  // Sort table data
  const sortedData = [...tableData].sort((a, b) => {
    let aVal = a[sortColumn];
    let bVal = b[sortColumn];

    if (sortDirection === 'asc') {
      return aVal > bVal ? 1 : -1;
    } else {
      return aVal < bVal ? 1 : -1;
    }
  });

  const handleSort = (column) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('desc');
    }
  };

  const SortIcon = ({ column }) => {
    if (sortColumn !== column) {
      return <ArrowUpDown size={14} className={styles.sortIconInactive} />;
    }
    return sortDirection === 'asc' ?
      <TrendingUp size={14} className={styles.sortIconActive} /> :
      <TrendingDown size={14} className={styles.sortIconActive} />;
  };

  return (
    <div className={styles.tableContainer}>
      <table className={styles.comparisonTable}>
        <thead>
          <tr>
            <th className={styles.tableHeaderSticky}>Model</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>AILANG</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>Python</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>Comparison</th>
          </tr>
          <tr>
            <th className={styles.tableHeaderSticky}></th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('ailangSuccess')}>
              % <SortIcon column="ailangSuccess" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('ailangTokens')}>
              Tok <SortIcon column="ailangTokens" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('pythonSuccess')}>
              % <SortIcon column="pythonSuccess" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('pythonTokens')}>
              Tok <SortIcon column="pythonTokens" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('gap')}>
              Gap <SortIcon column="gap" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('tokenRatio')}>
              Ratio <SortIcon column="tokenRatio" />
            </th>
          </tr>
        </thead>
        <tbody>
          {localRow && (
            <tr style={{ background: 'var(--ifm-color-info-contrast-background, rgba(8,145,178,0.10))', borderLeft: '4px solid #0891b2' }}>
              <td className={styles.tableModelName}>
                <span
                  title={`Best of ${localRow.configCount} on-device agent configs (${localRow.harness} · ${localRow.model}${localRow.version ? ', ' + localRow.version : ''}). Agent-mode, multi-turn, runs on the local GPU at ~$0/run — slow but free. Not directly comparable to the 0-shot cloud rows.`}
                  style={{ fontWeight: 700, cursor: 'help' }}
                >
                  🖥️ Local GPU agent
                </span>
                <span style={{ marginLeft: 6, fontSize: '0.7em', color: 'var(--ifm-color-emphasis-600)' }}>
                  {localRow.harness} · {localRow.model} · agent · ~$0
                </span>
              </td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: localRow.ailangSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    localRow.ailangSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}>
                  {localRow.ailangSuccess.toFixed(1)}
                </span>
              </td>
              <td className={styles.tableNumber} title="No token data for on-device runs">—</td>
              <td className={styles.tableNumber} title="Local agent runs AILANG only">—</td>
              <td className={styles.tableNumber}>—</td>
              <td className={styles.tableNumber}>—</td>
              <td className={styles.tableNumber}>—</td>
            </tr>
          )}
          {sortedData.map((row) => (
            <tr key={row.modelName} style={row.provisional ? { opacity: 0.6, fontStyle: 'italic' } : undefined}>
              <td className={styles.tableModelName}>
                {row.displayName}
                {row.provisional && (
                  <span
                    title={row.benchmarks != null
                      ? `Provisional: ran ${row.benchmarks}/${maxCoverage} benchmarks so far — pass-rate not comparable to full-coverage models until the rotation fills coverage in.`
                      : `Partial sample: ran ${row.runs}/${fullRuns} benchmarks (core tier only). Pass-rate not directly comparable to models that ran the full core+stretch suite.`}
                    style={{
                      marginLeft: 6,
                      fontSize: '0.7em',
                      padding: '1px 5px',
                      borderRadius: 3,
                      background: 'var(--ifm-color-warning-lightest)',
                      color: 'var(--ifm-color-warning-darkest)',
                      border: '1px solid var(--ifm-color-warning-light)',
                      verticalAlign: 'middle',
                      cursor: 'help',
                      fontWeight: 600,
                    }}
                  >
                    ⚠ {row.benchmarks != null ? `${row.benchmarks}/${maxCoverage}` : `${row.runs}/${fullRuns}`}
                  </span>
                )}
              </td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: row.ailangSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    row.ailangSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}>
                  {row.ailangSuccess.toFixed(1)}
                </span>
              </td>
              <td className={styles.tableNumber}>{row.ailangTokens}</td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: row.pythonSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    row.pythonSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}>
                  {row.pythonSuccess.toFixed(1)}
                </span>
              </td>
              <td className={styles.tableNumber}>{row.pythonTokens}</td>
              <td className={styles.tableNumber}>
                <span className={row.gap >= 0 ? styles.gapPositive : styles.gapNegative}>
                  {row.gap >= 0 ? '+' : ''}{row.gap.toFixed(1)}
                </span>
              </td>
              <td className={styles.tableNumber}>
                <span className={row.tokenRatio > 1 ? styles.ratioHigher : styles.ratioLower}>
                  {row.tokenRatio.toFixed(2)}x
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className={styles.tableFootnote}>
        💡 <strong>Gap</strong> = AILANG - Python success % (positive = better) · <strong>Ratio</strong> = AILANG/Python tokens (lower = more efficient) · <strong>Tok</strong> = avg output tokens
        {localRow && (
          <>
            {' · '}<strong>🖥️ Local GPU agent</strong> = best on-device config (qwen via an agentic harness), agent-mode + ~$0/run. Shown for the free-local-option thesis — <em>agent-mode, so not directly comparable to the 0-shot cloud rows</em>.
          </>
        )}
      </div>
    </div>
  );
}
