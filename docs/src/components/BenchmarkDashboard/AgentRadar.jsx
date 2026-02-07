import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Legend, Tooltip, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

function execLabel(executor) {
  return executor.charAt(0).toUpperCase() + executor.slice(1);
}

const EXECUTOR_COLORS = {
  claude: '#c96442',
  gemini: '#4285f4',
  unknown: '#888888',
};

export default function AgentRadar({ data }) {
  const hasAgentData = data && data.aggregates && data.aggregates.agentRuns > 0;
  const executors = data?.executors || {};
  const executorNames = data?.aggregates?.agentExecutors || Object.keys(executors);
  const multiExecutor = executorNames.length > 1;

  if (!hasAgentData) {
    return (
      <div className={styles.chartContainer}>
        <div className={styles.noData}>
          <p>Agent evaluation data not available yet.</p>
          <p className={styles.noDataHint}>
            Agent eval provides multi-turn iterative problem solving metrics.
          </p>
        </div>
      </div>
    );
  }

  // Build per-executor bar chart data for AILANG vs Python comparison
  const chartData = executorNames.map(exec => {
    const execData = executors[exec] || {};
    const langs = execData.languages || {};
    return {
      name: execLabel(exec),
      'AILANG Success': ((langs.ailang?.successRate || 0) * 100),
      'Python Success': ((langs.python?.successRate || 0) * 100),
      'AILANG Turns': langs.ailang?.avgTurns || 0,
      'Python Turns': langs.python?.avgTurns || 0,
    };
  });

  // Build per-model within executor data
  const modelData = [];
  for (const exec of executorNames) {
    const execData = executors[exec] || {};
    const models = execData.models || {};
    for (const [model, stats] of Object.entries(models)) {
      modelData.push({
        name: `${execLabel(exec)} / ${model}`,
        executor: exec,
        model,
        successRate: ((stats.successRate || 0) * 100),
        avgTurns: stats.avgTurns || 0,
        avgCost: stats.avgCost || 0,
        runs: stats.runs || 0,
      });
    }
  }

  const CustomTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{payload[0]?.payload?.name}</p>
          {payload.map((entry, index) => (
            <p key={index} className={styles.tooltipValue}>
              <span className={styles.tooltipDot} style={{backgroundColor: entry.color}} />
              {entry.name}: {entry.value.toFixed(1)}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <h3 className={styles.chartTitle}>Agent Evaluation Performance</h3>
      <p className={styles.chartSubtitle}>
        Multi-turn iterative problem solving ({data.aggregates.agentRuns} runs, avg {data.aggregates.avgAgentTurns?.toFixed(1)} turns)
      </p>

      {/* Per-executor stat cards */}
      <div className={styles.agentStats}>
        {executorNames.map(exec => {
          const es = executors[exec] || {};
          return (
            <div key={exec} className={styles.statCard} style={{ borderLeft: `3px solid ${EXECUTOR_COLORS[exec] || '#888'}` }}>
              <div className={styles.statValue} style={{ fontSize: '1rem' }}>{execLabel(exec)}</div>
              <div className={styles.statLabel}>
                {es.runs || 0} runs | {((es.successRate || 0) * 100).toFixed(1)}% success | ${(es.avgCost || 0).toFixed(3)}/run
              </div>
            </div>
          );
        })}
      </div>

      {/* Per-executor success rate bar chart */}
      {multiExecutor && (
        <>
          <h4 style={{ textAlign: 'center', margin: '1.5rem 0 0.5rem' }}>Success Rate by Executor</h4>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
              <XAxis dataKey="name" tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }} />
              <YAxis domain={[0, 100]} tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 10 }} />
              <Tooltip content={<CustomTooltip />} />
              <Legend wrapperStyle={{ fontSize: '12px' }} />
              <Bar dataKey="AILANG Success" fill="var(--ifm-color-primary)" />
              <Bar dataKey="Python Success" fill="var(--ifm-color-success)" />
            </BarChart>
          </ResponsiveContainer>
        </>
      )}

      {/* Per-model success rate table */}
      {modelData.length > 0 && (
        <>
          <h4 style={{ textAlign: 'center', margin: '1.5rem 0 0.5rem' }}>Per-Model Breakdown</h4>
          <div className={styles.tableWrapper}>
            <table className={styles.comparisonTable}>
              <thead>
                <tr>
                  <th>Executor / Model</th>
                  <th>Runs</th>
                  <th>Success Rate</th>
                  <th>Avg Turns</th>
                  <th>Avg Cost</th>
                </tr>
              </thead>
              <tbody>
                {modelData.map(m => (
                  <tr key={m.name}>
                    <td>
                      <span style={{ color: EXECUTOR_COLORS[m.executor] || '#888', fontWeight: 600 }}>
                        {execLabel(m.executor)}
                      </span>
                      {' / '}{m.model}
                    </td>
                    <td className={styles.tableCell}>{m.runs}</td>
                    <td className={styles.tableCell}>{m.successRate.toFixed(1)}%</td>
                    <td className={styles.tableCell}>{m.avgTurns.toFixed(1)}</td>
                    <td className={styles.tableCell}>${m.avgCost.toFixed(3)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      <div className={styles.agentNotes}>
        <p><strong>What is Agent Evaluation?</strong></p>
        <p>
          Agent eval uses AI coding agents (Claude Code, Gemini CLI) in headless mode for multi-turn
          iterative problem solving. Unlike standard 0-shot evaluation, agents can explore, debug, and
          refine solutions over multiple turns.
        </p>
      </div>
    </div>
  );
}
