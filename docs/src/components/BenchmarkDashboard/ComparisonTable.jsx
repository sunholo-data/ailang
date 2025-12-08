import React from 'react';
import styles from './styles.module.css';

export default function ComparisonTable({ data }) {
  const languages = data.languages || {};
  const aggregates = data.aggregates || {};
  const hasAgentData = aggregates.agentRuns > 0;

  if (!hasAgentData) {
    return null;
  }

  // Helper function to format numbers
  const fmt = (value, decimals = 1) => {
    if (value == null) return 'N/A';
    return value.toFixed(decimals);
  };

  const fmtPct = (value, decimals = 1) => {
    if (value == null) return 'N/A';
    return `${(value * 100).toFixed(decimals)}%`;
  };

  const fmtCost = (value, decimals = 4) => {
    if (value == null) return 'N/A';
    return `$${value.toFixed(decimals)}`;
  };

  const renderLanguageRow = (lang, langData) => {
    const zeroShot = {
      success: langData.zero_shot_success_comparable,
      tokens: langData.zero_shot_avg_tokens_comparable,
      cost: langData.zero_shot_avg_cost_comparable,
      costPerSuccess: langData.zero_shot_cost_per_success
    };

    const withRepair = {
      success: langData.final_success_comparable,
      tokens: langData.final_success_avg_tokens_comparable,
      cost: langData.final_success_avg_cost_comparable,
      costPerSuccess: langData.final_cost_per_success_comparable
    };

    const agent = {
      success: langData.agent_success_rate,
      tokens: langData.agent_avg_tokens,
      cost: langData.agent_avg_cost,
      costPerSuccess: langData.agent_cost_per_success,
      avgTurns: langData.agent_avg_turns,
      avgTurnsSuccess: langData.agent_avg_turns_success,
      avgTurnsFailure: langData.agent_avg_turns_failure
    };

    const derived = {
      successGap: langData.agent_success_gap,
      impossibilityCoverage: langData.agent_impossibility_coverage,
      costRatio: langData.agent_cost_efficiency_ratio
    };

    return (
      <React.Fragment key={lang}>
        <tr className={styles.tableRowHeader}>
          <td colSpan="4" className={styles.languageHeader}>
            <strong>{lang.toUpperCase()}</strong>
          </td>
        </tr>

        {/* Success Rates */}
        <tr>
          <td>Success Rate</td>
          <td className={styles.tableCell}>{fmtPct(zeroShot.success)}</td>
          <td className={styles.tableCell}>{fmtPct(withRepair.success)}</td>
          <td className={styles.tableCell}>{fmtPct(agent.success)}</td>
        </tr>

        {/* Average Tokens */}
        <tr>
          <td>Avg Tokens</td>
          <td className={styles.tableCell}>{fmt(zeroShot.tokens, 0)}</td>
          <td className={styles.tableCell}>{fmt(withRepair.tokens, 0)}</td>
          <td className={styles.tableCell}>{fmt(agent.tokens, 0)}</td>
        </tr>

        {/* Average Cost */}
        <tr>
          <td>Avg Cost</td>
          <td className={styles.tableCell}>{fmtCost(zeroShot.cost)}</td>
          <td className={styles.tableCell}>{fmtCost(withRepair.cost)}</td>
          <td className={styles.tableCell}>{fmtCost(agent.cost, 2)}</td>
        </tr>

        {/* Cost per Success */}
        <tr className={styles.highlightRow}>
          <td><strong>Cost per Success</strong></td>
          <td className={styles.tableCell}>{fmtCost(zeroShot.costPerSuccess)}</td>
          <td className={styles.tableCell}>{fmtCost(withRepair.costPerSuccess)}</td>
          <td className={styles.tableCell}>{fmtCost(agent.costPerSuccess, 2)}</td>
        </tr>

        {/* Agent-specific metrics */}
        <tr>
          <td>Agent Avg Turns</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          <td className={styles.tableCell}>{fmt(agent.avgTurns)}</td>
        </tr>

        {agent.avgTurnsSuccess != null && (
          <tr>
            <td style={{ paddingLeft: '2rem', fontSize: '0.9em' }}>→ Success</td>
            <td className={styles.tableCell} colSpan="2">-</td>
            <td className={styles.tableCell}>{fmt(agent.avgTurnsSuccess)}</td>
          </tr>
        )}

        {agent.avgTurnsFailure != null && (
          <tr>
            <td style={{ paddingLeft: '2rem', fontSize: '0.9em' }}>→ Failure</td>
            <td className={styles.tableCell} colSpan="2">-</td>
            <td className={styles.tableCell}>{fmt(agent.avgTurnsFailure)}</td>
          </tr>
        )}

        {/* Derived metrics showing agent value */}
        <tr className={styles.derivedMetricsHeader}>
          <td colSpan="4"><strong>Agent Superiority Metrics</strong></td>
        </tr>

        <tr>
          <td>Success Gap (agent - 0-shot)</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          <td className={styles.tableCell}>
            <span className={derived.successGap > 0.3 ? styles.goodValue : ''}>
              {fmtPct(derived.successGap)}
            </span>
          </td>
        </tr>

        <tr>
          <td>Impossibility Coverage</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          <td className={styles.tableCell}>
            <span className={derived.impossibilityCoverage > 0.7 ? styles.goodValue : ''}>
              {fmtPct(derived.impossibilityCoverage)}
            </span>
          </td>
        </tr>

        <tr className={styles.highlightRow}>
          <td><strong>Cost Efficiency Ratio</strong></td>
          <td className={styles.tableCell} colSpan="2">-</td>
          <td className={styles.tableCell}>
            <span className={derived.costRatio < 5 ? styles.goodValue : styles.badValue}>
              {fmt(derived.costRatio)}x
            </span>
          </td>
        </tr>
      </React.Fragment>
    );
  };

  return (
    <div className={styles.comparisonTableContainer}>
      <h3>Approach Comparison</h3>
      <p className={styles.tableDescription}>
        Comparing evaluation approaches on the same {aggregates.agentBenchmarks?.length || 0} benchmarks.
        <strong> Cost Efficiency Ratio</strong>: Lower is better (agent justifies its cost).
        <strong> Impossibility Coverage</strong>: % of 0-shot failures that agent solves.
      </p>

      <div className={styles.tableWrapper}>
        <table className={styles.comparisonTable}>
          <thead>
            <tr>
              <th>Metric</th>
              <th>0-Shot</th>
              <th>With Repair</th>
              <th>Agent</th>
            </tr>
          </thead>
          <tbody>
            {languages.ailang && renderLanguageRow('ailang', languages.ailang)}
            {languages.python && renderLanguageRow('python', languages.python)}
          </tbody>
        </table>
      </div>

      <div className={styles.tableFootnote}>
        <p><strong>Interpreting Agent Superiority:</strong></p>
        <ul>
          <li><strong>Success Gap &gt; 30%</strong>: Agent provides significant value</li>
          <li><strong>Impossibility Coverage &gt; 70%</strong>: Agent solves most "impossible" problems</li>
          <li><strong>Cost Ratio &lt; 5x</strong>: Agent cost is justified by success improvement</li>
        </ul>
        <p>
          On harder benchmarks, these metrics should favor agent mode more strongly.
        </p>
      </div>
    </div>
  );
}
