import React, { useState } from 'react';
import { ArrowUpDown, TrendingUp, TrendingDown } from 'lucide-react';
import styles from './styles.module.css';

function formatModelName(name) {
  // Check most specific patterns first
  if (name.includes('claude-sonnet-4-5')) return 'Claude Sonnet 4.5';
  if (name.includes('claude-haiku-4-5')) return 'Claude Haiku 4.5';
  if (name.includes('gpt-5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5')) return 'GPT-5';
  if (name.includes('gpt5-1-instant')) return 'Gpt5 1 Instant';
  if (name.includes('gpt5-1')) return 'Gpt5 1';
  if (name.includes('gemini-2-5-flash') || name.includes('gemini-2.5-flash')) return 'Gemini 2.5 Flash';
  if (name.includes('gemini-2-5-pro') || name.includes('gemini-2.5-pro')) return 'Gemini 2.5 Pro';
  if (name.includes('gemini-3-pro') || name.includes('gemini-3.0-pro')) return 'Gemini 3.0 Pro';
  // Fallback: capitalize first letter of each word
  return name.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
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
              <td className={styles.tableModelName}>{row.displayName}</td>
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
