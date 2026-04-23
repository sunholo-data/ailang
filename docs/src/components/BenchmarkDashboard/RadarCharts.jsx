import React from 'react';
import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, Legend, Tooltip, ResponsiveContainer } from 'recharts';
import ComparisonTable from './ComparisonTable';
import RepairEffectiveness from './RepairEffectiveness';
import styles from './styles.module.css';

// Capitalize executor name for display
function execLabel(executor) {
  return executor.charAt(0).toUpperCase() + executor.slice(1);
}

export default function RadarCharts({ data }) {
  const languages = data.languages || {};
  const aggregates = data.aggregates || {};
  const executors = data.executors || {};
  const hasAgentData = aggregates.agentRuns > 0;
  const executorNames = (aggregates.agentExecutors || Object.keys(executors)).filter(e => e !== 'unknown');
  const multiExecutor = executorNames.length > 1;

  // Custom tooltip for all radar charts
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

  // Build success rate data with per-executor agent axes when multiple executors exist
  const successRateData = [
    {
      metric: '0-Shot',
      AILANG: hasAgentData
        ? (languages.ailang?.zero_shot_success_comparable || 0) * 100
        : (languages.ailang?.zero_shot_success || 0) * 100,
      Python: hasAgentData
        ? (languages.python?.zero_shot_success_comparable || 0) * 100
        : (languages.python?.zero_shot_success || 0) * 100
    },
    {
      metric: 'With Repair',
      AILANG: hasAgentData
        ? (languages.ailang?.final_success_comparable || 0) * 100
        : (languages.ailang?.success_rate || 0) * 100,
      Python: hasAgentData
        ? (languages.python?.final_success_comparable || 0) * 100
        : (languages.python?.success_rate || 0) * 100
    }
  ];

  if (hasAgentData && multiExecutor) {
    // Add one axis per executor
    for (const exec of executorNames) {
      successRateData.push({
        metric: `Agent (${execLabel(exec)})`,
        AILANG: (languages.ailang?.[`agent_success_rate_${exec}`] || 0) * 100,
        Python: (languages.python?.[`agent_success_rate_${exec}`] || 0) * 100
      });
    }
  } else if (hasAgentData) {
    // Single executor - show as plain "Agent"
    successRateData.push({
      metric: 'Agent',
      AILANG: (languages.ailang?.agent_success_rate || 0) * 100,
      Python: (languages.python?.agent_success_rate || 0) * 100
    });
  }

  // Get agent benchmark list from data
  const agentBenchmarks = aggregates.agentBenchmarks || [];

  return (
    <>
      {hasAgentData && agentBenchmarks.length > 0 && (
        <div className={styles.benchmarkNote}>
          <strong>Note:</strong> Comparison uses only the {agentBenchmarks.length} benchmarks tested in agent mode.
          {multiExecutor && (
            <> Agents: {executorNames.map(e => execLabel(e)).join(', ')}.</>
          )}
        </div>
      )}

      {/* Success Rate Radar Chart */}
      <div className={styles.radarCard} style={{ maxWidth: '600px', margin: '2rem auto' }}>
        <h4 className={styles.radarTitle}>Success Rate (%)</h4>
        <p className={styles.radarSubtitle}>
          {hasAgentData
            ? multiExecutor
              ? 'Comparing evaluation approaches per executor'
              : 'Comparing on same benchmarks'
            : 'Comparing all evaluation approaches'}
        </p>
        <ResponsiveContainer width="100%" height={350}>
          <RadarChart data={successRateData}>
            <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
            <PolarAngleAxis
              dataKey="metric"
              tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            />
            <PolarRadiusAxis
              angle={90}
              domain={[0, 100]}
              tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 10 }}
              tickFormatter={(value) => value.toFixed(1)}
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
            <Legend wrapperStyle={{ fontSize: '12px' }} />
            <Tooltip content={<CustomTooltip />} />
          </RadarChart>
        </ResponsiveContainer>
      </div>

      {/* Comparison Table - replaces token/cost radars */}
      <ComparisonTable data={data} />

      {/* Repair Effectiveness Analysis */}
      <RepairEffectiveness data={data} />
    </>
  );
}
