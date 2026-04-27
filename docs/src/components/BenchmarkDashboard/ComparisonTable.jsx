import React from 'react';
import styles from './styles.module.css';

function execLabel(executor) {
  return executor.charAt(0).toUpperCase() + executor.slice(1);
}

// Display priority: ADJUSTED is the headline rate (true model strength),
// raw is the secondary annotation only shown when meaningfully different.
// Returns { primary, secondaryLabel } — both nullable.
function pickPrimary(raw, adjusted) {
  if (raw == null && adjusted == null) return { primary: null, secondaryLabel: null };
  // No adjustment available, just show raw.
  if (adjusted == null) return { primary: raw, secondaryLabel: null };
  if (raw == null) return { primary: adjusted, secondaryLabel: null };
  // Trivial difference — show raw only (no benefit to flipping).
  if (Math.abs(adjusted - raw) < 0.01) return { primary: raw, secondaryLabel: null };
  // Meaningful difference — adjusted is primary, raw is secondary.
  return { primary: adjusted, secondaryLabel: `raw ${Math.round(raw * 100)}%` };
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
      successAdjusted: langData[`agent_success_rate_adjusted_${exec}`],
      apiErrors: langData[`agent_api_errors_${exec}`],
      apiErrorRate: langData[`agent_api_error_rate_${exec}`],
      tokens: langData[`agent_avg_tokens_${exec}`],
      cost: langData[`agent_avg_cost_${exec}`],
      runs: langData[`agent_runs_${exec}`],
    };
  };

  // Render a cell with ADJUSTED as primary (true model strength) + (raw X%)
  // annotation showing the unfiltered rate when api_errors are dragging it down.
  const renderRateWithAdj = (raw, adjusted, key) => {
    const { primary, secondaryLabel } = pickPrimary(raw, adjusted);
    return (
      <td key={key} className={styles.tableCell}>
        {fmtPct(primary)}
        {secondaryLabel && (
          <span style={{ display: 'block', fontSize: '0.75em', color: 'var(--ifm-color-emphasis-500)', fontStyle: 'italic' }}>
            ({secondaryLabel})
          </span>
        )}
      </td>
    );
  };

  const renderLanguageRow = (lang, langData) => {
    const zeroShot = {
      success: langData.zero_shot_success_comparable,
      successAdjusted: langData.zero_shot_success_comparable_adjusted,
      tokens: langData.zero_shot_avg_tokens_comparable,
      cost: langData.zero_shot_avg_cost_comparable,
      costPerSuccess: langData.zero_shot_cost_per_success
    };

    const withRepair = {
      success: langData.final_success_comparable,
      successAdjusted: langData.final_success_comparable_adjusted,
      tokens: langData.final_success_avg_tokens_comparable,
      cost: langData.final_success_avg_cost_comparable,
      costPerSuccess: langData.final_cost_per_success_comparable
    };

    const agentBlended = {
      success: langData.agent_success_rate,
      successAdjusted: langData.agent_success_rate_adjusted,
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

        {/* Success Rates — primary number is raw; (adj X%) shows model strength
            when api_errors are excluded. Big delta = infrastructure dragging
            the rate down. Small/no delta = api_errors weren't a factor. */}
        <tr>
          <td>Success Rate</td>
          {renderRateWithAdj(zeroShot.success, zeroShot.successAdjusted, 'zs')}
          {renderRateWithAdj(withRepair.success, withRepair.successAdjusted, 'wr')}
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              return renderRateWithAdj(es.success, es.successAdjusted, exec);
            })
          ) : (
            renderRateWithAdj(agentBlended.success, agentBlended.successAdjusted, 'ag')
          )}
        </tr>

        {/* Average Tokens */}
        <tr>
          <td title="CLI-reported tokens. Codex/Gemini count the full re-sent context per turn (tool schemas + reasoning trace); Claude/opencode count only billable tokens. Use Avg Cost for cross-harness comparisons.">
            Avg Tokens <span style={{ fontSize: '0.75em', color: 'var(--ifm-color-emphasis-500)', fontWeight: 400 }}>(CLI-reported)</span>
          </td>
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

        {/* API error rate — exposes infrastructure failures (quota, CLI version,
            harness crashes) so reviewers can see why raw rates dipped. */}
        <tr>
          <td style={{ fontSize: '0.9em', color: 'var(--ifm-color-emphasis-700)' }}>API Error Rate</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const es = getExecStats(langData, exec);
              const rate = es.apiErrorRate;
              if (rate == null) return <td key={exec} className={styles.tableCell}>-</td>;
              return (
                <td key={exec} className={styles.tableCell} style={{ fontSize: '0.85em' }}>
                  <span className={rate > 0.2 ? styles.badValue : ''}>
                    {fmtPct(rate)}
                  </span>
                  {es.apiErrors > 0 && (
                    <span style={{ display: 'block', fontSize: '0.85em', color: 'var(--ifm-color-emphasis-500)' }}>
                      ({es.apiErrors} runs)
                    </span>
                  )}
                </td>
              );
            })
          ) : (
            <td className={styles.tableCell} style={{ fontSize: '0.85em' }}>
              {fmtPct(langData.agent_api_error_rate)}
            </td>
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

        {/* Success Gap = adjusted_agent_success - adjusted_zero_shot_success.
            Prefers the per-executor adjusted gap when available (apples-to-apples
            with the headline Success Rate row above). Falls back through:
            (a) per-exec adjusted, (b) per-exec raw, (c) computed from
            adjusted-agent-vs-blended-adjusted-zero-shot. */}
        <tr>
          <td>Success Gap (agent - 0-shot)</td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const adjGap = langData[`agent_success_gap_adjusted_${exec}`];
              const rawGap = langData[`agent_success_gap_${exec}`];
              const es = getExecStats(langData, exec);
              // Final fallback: use whichever rate is primary on the headline row
              // (adjusted preferred) and subtract the matching zero-shot baseline.
              const agentRate = es.successAdjusted ?? es.success;
              const baseRate = zeroShot.successAdjusted ?? zeroShot.success;
              const computed = (agentRate != null && baseRate != null) ? agentRate - baseRate : null;
              const gap = adjGap ?? rawGap ?? computed;
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

        {/* Cost Efficiency Ratio: (agent cost-per-success) / (0-shot cost-per-success).
            Adjusted variant uses adjusted success rates so the denominator matches
            the headline. Lower = agent justifies its cost. */}
        <tr className={styles.highlightRow}>
          <td><strong>Cost Efficiency Ratio</strong></td>
          <td className={styles.tableCell} colSpan="2">-</td>
          {multiExecutor ? (
            executorNames.map(exec => {
              const adjRatio = langData[`agent_cost_efficiency_ratio_adjusted_${exec}`];
              const rawRatio = langData[`agent_cost_efficiency_ratio_${exec}`];
              const ratio = adjRatio ?? rawRatio;
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

  const matchedModels = aggregates.agentModels || [];

  return (
    <div className={styles.comparisonTableContainer}>
      <h3>Approach Comparison</h3>
      <p className={styles.tableDescription}>
        Comparing evaluation approaches on the same {aggregates.agentBenchmarks?.length || 0} benchmarks
        {matchedModels.length > 0 && (
          <> using the same {matchedModels.length} models on both sides:{' '}
            <code style={{ fontSize: '0.85em' }}>{matchedModels.join(', ')}</code>.
            {' '}<em>(0-shot results from flagship-only models like Opus/Pro are excluded so the comparison is apples-to-apples.)</em></>
        )}
        {matchedModels.length === 0 && '.'}
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
        <p><strong>Reading the table:</strong></p>
        <ul>
          <li><strong>Headline rates exclude API errors</strong>: Each success rate
              shown is the &quot;adjusted&quot; rate (passes / non-api-error runs) — the
              true model strength when infrastructure works. <code>(raw X%)</code> in
              parentheses shows the unfiltered rate before excluding infra failures
              (quota, CLI version mismatches, harness crashes). The
              <strong> API Error Rate</strong> row shows the magnitude of infra noise per harness.</li>
          <li><strong>Avg Tokens differ across harnesses</strong> because each CLI
              counts differently — Codex/Gemini include the full re-sent context per
              turn (incl. tool schemas + reasoning trace); Claude Code/opencode count
              only billable tokens. <strong>Avg Cost</strong> and <strong>Cost per Success</strong>
              are the directly comparable metrics across harnesses.</li>
          <li><strong>Success Gap &gt; 30%</strong>: Agent provides significant value</li>
          <li><strong>Impossibility Coverage &gt; 70%</strong>: Agent solves most &quot;impossible&quot; problems</li>
          <li><strong>Cost Ratio &lt; 5x</strong>: Agent cost is justified by success improvement</li>
        </ul>
      </div>
    </div>
  );
}
