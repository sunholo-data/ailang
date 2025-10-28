import React from 'react';
import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, Legend, Tooltip, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function AgentRadar({ data }) {
  // Check if we have agent data
  const hasAgentData = data && data.aggregates && data.aggregates.agentRuns > 0;

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

  const languages = data.languages || {};

  // Build radar chart data - normalize to 0-100 scale
  const maxTurns = Math.max(
    languages.ailang?.agent_avg_turns || 0,
    languages.python?.agent_avg_turns || 0
  );

  const maxTokens = Math.max(
    languages.ailang?.agent_avg_tokens || 0,
    languages.python?.agent_avg_tokens || 0
  );

  const chartData = [
    {
      metric: 'Avg Turns',
      AILANG: languages.ailang?.agent_avg_turns || 0,
      Python: languages.python?.agent_avg_turns || 0,
      fullMax: maxTurns
    },
    {
      metric: 'Avg Tokens (K)',
      AILANG: ((languages.ailang?.agent_avg_tokens || 0) / 1000),
      Python: ((languages.python?.agent_avg_tokens || 0) / 1000),
      fullMax: maxTokens / 1000
    },
    {
      metric: 'Success Rate (%)',
      AILANG: (languages.ailang?.agent_success_rate || 0) * 100,
      Python: (languages.python?.agent_success_rate || 0) * 100,
      fullMax: 100
    }
  ];

  const CustomTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{data.metric}</p>
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
        Multi-turn iterative problem solving (avg {data.aggregates.avgAgentTurns?.toFixed(1)} turns)
      </p>

      <ResponsiveContainer width="100%" height={400}>
        <RadarChart data={chartData}>
          <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
          <PolarAngleAxis
            dataKey="metric"
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
          />
          <PolarRadiusAxis
            angle={90}
            domain={[0, 'dataMax']}
            tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 10 }}
          />
          <Radar
            name="AILANG"
            dataKey="AILANG"
            stroke="var(--ifm-color-primary)"
            fill="var(--ifm-color-primary)"
            fillOpacity={0.3}
          />
          <Radar
            name="Python"
            dataKey="Python"
            stroke="var(--ifm-color-success)"
            fill="var(--ifm-color-success)"
            fillOpacity={0.3}
          />
          <Legend />
          <Tooltip content={<CustomTooltip />} />
        </RadarChart>
      </ResponsiveContainer>

      <div className={styles.agentStats}>
        <div className={styles.statCard}>
          <div className={styles.statValue}>{data.aggregates.agentRuns}</div>
          <div className={styles.statLabel}>Agent Runs</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statValue}>
            {((data.aggregates.agentSuccessRate || 0) * 100).toFixed(1)}%
          </div>
          <div className={styles.statLabel}>Success Rate</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statValue}>
            {(data.aggregates.avgAgentTurns || 0).toFixed(1)}
          </div>
          <div className={styles.statLabel}>Avg Turns</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statValue}>
            {((data.aggregates.agentTotalTokens || 0) / 1000).toFixed(0)}K
          </div>
          <div className={styles.statLabel}>Total Tokens</div>
        </div>
      </div>

      <div className={styles.agentNotes}>
        <p><strong>What is Agent Evaluation?</strong></p>
        <p>
          Agent eval uses Claude Code in headless mode for multi-turn iterative problem solving.
          Unlike standard 0-shot evaluation, agents can explore, debug, and refine solutions
          over multiple turns, providing insights into real-world development workflows.
        </p>
      </div>
    </div>
  );
}
