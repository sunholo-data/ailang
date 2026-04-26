import React from 'react';
import styles from './styles.module.css';

function execLabel(executor) {
  return executor.charAt(0).toUpperCase() + executor.slice(1);
}

export default function ComparisonTable({ data }) {
  const languages = data.languages || {};
  const aggregates = data.aggregates || {};
  const executors = data.executors || {};
  const hasAgentData = aggregates.agentRuns > 0;
  const executorNames = (aggregates.agentExecutors || Object.keys(executors)).filter(e => e !== 'unknown');
  const multiExecutor = executorNames.length > 1;

  if (!hasAgentData) {
    return null;
  }

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

  // Get per-executor agent stats for a language
  const getExecStats = (langData, exec) => {
    return {
      success: langData[`agent_success_rate_${exec}`],
      tokens: langData[`agent_avg_tokens_${exec}`],
      cost: langData[`agent_avg_cost_${exec}`],
      runs: langData[`agent_runs_${exec}`],
    };
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

    const agentBlended = {
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
          <td colSpan={multiExecutor ? 3 + executorNames.length : 4} className={styles.languageHeader}>
            <strong>{lang.toUpperCase()}</strong>
          </td>
        </tr>

        {/* Success Rates */}
        <tr>
          <td>Success Rate</td>
          <td className={styles.tableCell}>{fmtPct(zeroShot.success)}</td>
          <td className={styles.tableCell}>{fmtPct(withRepair.success)}</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              return <td key={exec} className={styles.tableCell}>{fmtPct(es.success)}</td>;
            })
          ) : (
            <td className={styles.tableCell}>{fmtPct(agentBlended.success)}</td>
          )}
        </tr>

        {/* Average Tokens */}
        <tr>
          <td>Avg Tokens</td>
          <td className={styles.tableCell}>{fmt(zeroShot.tokens, 0)}</td>
          <td className={styles.tableCell}>{fmt(withRepair.tokens, 0)}</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              return <td key={exec} className={styles.tableCell}>{fmt(es.tokens, 0)}</td>;
            })
          ) : (
            <td className={styles.tableCell}>{fmt(agentBlended.tokens, 0)}</td>
          )}
        </tr>

        {/* Average Cost */}
        <tr>
          <td>Avg Cost</td>
          <td className={styles.tableCell}>{fmtCost(zeroShot.cost)}</td>
          <td className={styles.tableCell}>{fmtCost(withRepair.cost)}</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              return <td key={exec} className={styles.tableCell}>{fmtCost(es.cost, 2)}</td>;
            })
          ) : (
            <td className={styles.tableCell}>{fmtCost(agentBlended.cost, 2)}</td>
          )}
        </tr>

        {/* Cost per Success */}
        <tr className={styles.highlightRow}>
          <td><strong>Cost per Success</strong></td>
          <td className={styles.tableCell}>{fmtCost(zeroShot.costPerSuccess)}</td>
          <td className={styles.tableCell}>{fmtCost(withRepair.costPerSuccess)}</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              const cps = es.success > 0 ? es.cost / es.success : null;
              return <td key={exec} className={styles.tableCell}>{fmtCost(cps, 2)}</td>;
            })
          ) : (
            <td className={styles.tableCell}>{fmtCost(agentBlended.costPerSuccess, 2)}</td>
          )}
        </tr>

        {/* Agent-specific metrics - blended (turn metrics don't vary much per executor) */}
        <tr>
          <td>Agent Avg Turns</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const turns = langData[`agent_avg_turns_${exec}`];
              return <td key={exec} className={styles.tableCell}>{fmt(turns)}</td>;
            })
          ) : (
            <td className={styles.tableCell}>{fmt(agentBlended.avgTurns)}</td>
          )}
        </tr>

        {!multiExecutor && agentBlended.avgTurnsSuccess != null && (
          <tr>
            <td style={{ paddingLeft: '2rem', fontSize: '0.9em' }}>&#8594; Success</td>
            <td className={styles.tableCell} colSpan="2">-</td>
            <td className={styles.tableCell}>{fmt(agentBlended.avgTurnsSuccess)}</td>
          </tr>
        )}

        {!multiExecutor && agentBlended.avgTurnsFailure != null && (
          <tr>
            <td style={{ paddingLeft: '2rem', fontSize: '0.9em' }}>&#8594; Failure</td>
            <td className={styles.tableCell} colSpan="2">-</td>
            <td className={styles.tableCell}>{fmt(agentBlended.avgTurnsFailure)}</td>
          </tr>
        )}

        {/* Derived metrics — per-executor where computable, blended for impossibility coverage */}
        <tr className={styles.derivedMetricsHeader}>
          <td colSpan={multiExecutor ? 3 + executorNames.length : 4}><strong>Agent Superiority Metrics</strong></td>
        </tr>

        {/* Success Gap = agent_success - zero_shot_success_comparable_<exec>.
            Uses the per-executor comparable baseline when available so the
            denominator matches that executor's actual benchmark set. Falls back
            to subtracting the blended zero_shot if the per-executor field is
            absent (e.g. older baselines). */}
        <tr>
          <td>Success Gap (agent - 0-shot)</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const fairGap = langData[`agent_success_gap_${exec}`];
              const es = getExecStats(langData, exec);
              const fallback = (es.success != null && zeroShot.success != null) ? es.success - zeroShot.success : null;
              const gap = fairGap != null ? fairGap : fallback;
              return (
                <td key={exec} className={styles.tableCell}>
                  <span className={gap > 0.3 ? styles.goodValue : (gap < 0 ? styles.badValue : '')}>
                    {fmtPct(gap)}
                  </span>
                </td>
              );
            })
          ) : (
            <td className={styles.tableCell}>
              <span className={derived.successGap > 0.3 ? styles.goodValue : ''}>
                {fmtPct(derived.successGap)}
              </span>
            </td>
          )}
        </tr>

        {/* Impossibility Coverage — only blended (requires per-benchmark data not exported per-executor) */}
        <tr>
          <td>Impossibility Coverage <span style={{ fontSize: '0.75em', color: 'var(--ifm-color-emphasis-500)' }}>(blended)</span></td>
          <td className={styles.tableCell} colSpan="2">-</td>
          <td className={styles.tableCell} colSpan={multiExecutor ? executorNames.length : 1} style={{ textAlign: 'center' }}>
            <span className={derived.impossibilityCoverage > 0.7 ? styles.goodValue : ''}>
              {fmtPct(derived.impossibilityCoverage)}
            </span>
          </td>
        </tr>

        {/* Cost Efficiency Ratio = (agent_cost/success) / zero_shot_cost/success.
            Uses per-executor comparable baseline when available — same fairness
            argument as Success Gap above. */}
        <tr className={styles.highlightRow}>
          <td><strong>Cost Efficiency Ratio</strong></td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const fairRatio = langData[`agent_cost_efficiency_ratio_${exec}`];
              const es = getExecStats(langData, exec);
              const cps = (es.success > 0 && es.cost != null) ? es.cost / es.success : null;
              const fallback = (cps != null && zeroShot.costPerSuccess > 0) ? cps / zeroShot.costPerSuccess : null;
              const ratio = fairRatio != null ? fairRatio : fallback;
              return (
                <td key={exec} className={styles.tableCell}>
                  <span className={ratio != null && ratio < 5 ? styles.goodValue : styles.badValue}>
                    {ratio != null ? `${fmt(ratio)}x` : 'N/A'}
                  </span>
                </td>
              );
            })
          ) : (
            <td className={styles.tableCell}>
              <span className={derived.costRatio < 5 ? styles.goodValue : styles.badValue}>
                {fmt(derived.costRatio)}x
              </span>
            </td>
          )}
        </tr>
      </React.Fragment>
    );
  };

  return (
    <div className={styles.comparisonTableContainer}>
      <h3>Approach Comparison</h3>
      <p className={styles.tableDescription}>
        Comparing evaluation approaches on the same {aggregates.agentBenchmarks?.length || 0} benchmarks.
        {multiExecutor && (
          <> Per-executor breakdown shows {executorNames.map(e => execLabel(e)).join(' vs ')}.</>
        )}
        {' '}<strong>Cost Efficiency Ratio</strong>: Lower is better.
        {' '}<strong>Impossibility Coverage</strong>: % of 0-shot failures that agent solves.
      </p>

      <div className={styles.tableWrapper}>
        <table className={styles.comparisonTable}>
          <thead>
            <tr>
              <th>Metric</th>
              <th>0-Shot</th>
              <th>With Repair</th>
              {multiExecutor ? (
                executorNames.map(exec => (
                  <th key={exec}>Agent ({execLabel(exec)})</th>
                ))
              ) : (
                <th>Agent</th>
              )}
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
          <li><strong>Impossibility Coverage &gt; 70%</strong>: Agent solves most &quot;impossible&quot; problems</li>
          <li><strong>Cost Ratio &lt; 5x</strong>: Agent cost is justified by success improvement</li>
        </ul>
      </div>
    </div>
  );
}
