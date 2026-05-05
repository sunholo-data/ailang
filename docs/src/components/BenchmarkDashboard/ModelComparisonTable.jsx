import React, { useState } from 'react';
import { ArrowUpDown, TrendingUp, TrendingDown } from 'lucide-react';
import styles from './styles.module.css';

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

export default function ModelComparisonTable({ models }) {
  const [sortColumn, setSortColumn] = useState('ailangSuccess');
  const [sortDirection, setSortDirection] = useState('desc');

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
  }

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
          {sortedData.map((row) => (
            <tr key={row.modelName}>
              <td className={styles.tableModelName}>
                {row.displayName}
                {row.isPartialSample && (
                  <span
                    title={`Partial sample: ran ${row.runs}/${fullRuns} benchmarks (core tier only). Pass-rate not directly comparable to models that ran the full core+stretch suite.`}
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
                    ⚠ {row.runs}/{fullRuns}
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
      </div>
    </div>
  );
}
