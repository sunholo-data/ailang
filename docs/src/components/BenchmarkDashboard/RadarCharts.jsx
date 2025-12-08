import React from 'react';
import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, Legend, Tooltip, ResponsiveContainer } from 'recharts';
import ComparisonTable from './ComparisonTable';
import RepairEffectiveness from './RepairEffectiveness';
import styles from './styles.module.css';

export default function RadarCharts({ data }) {
  const languages = data.languages || {};
  const aggregates = data.aggregates || {};
  const hasAgentData = aggregates.agentRuns > 0;

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

  // Chart 1: Success Rate across all approaches
  // Use _comparable metrics when agent data exists (same benchmarks only for fair comparison)
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
    },
    {
      metric: 'Agent',
      AILANG: hasAgentData ? (languages.ailang?.agent_success_rate || 0) * 100 : 0,
      Python: hasAgentData ? (languages.python?.agent_success_rate || 0) * 100 : 0
    }
  ];

  // Chart 2: Total Tokens across all approaches
  const totalTokensData = [
    {
      metric: '0-Shot',
      AILANG: hasAgentData
        ? (languages.ailang?.zero_shot_avg_tokens_comparable || 0)
        : (languages.ailang?.zero_shot_avg_tokens || 0),
      Python: hasAgentData
        ? (languages.python?.zero_shot_avg_tokens_comparable || 0)
        : (languages.python?.zero_shot_avg_tokens || 0)
    },
    {
      metric: 'With Repair',
      AILANG: hasAgentData
        ? (languages.ailang?.final_success_avg_tokens_comparable || 0)
        : (languages.ailang?.final_success_avg_tokens || 0),
      Python: hasAgentData
        ? (languages.python?.final_success_avg_tokens_comparable || 0)
        : (languages.python?.final_success_avg_tokens || 0)
    },
    {
      metric: 'Agent',
      AILANG: hasAgentData ? (languages.ailang?.agent_avg_tokens || 0) : 0,
      Python: hasAgentData ? (languages.python?.agent_avg_tokens || 0) : 0
    }
  ];

  // Chart 3: Cost per 1K runs across all approaches
  const costData = [
    {
      metric: '0-Shot',
      AILANG: hasAgentData
        ? (languages.ailang?.zero_shot_avg_cost_comparable || 0) * 1000
        : (languages.ailang?.zero_shot_avg_cost || 0) * 1000,
      Python: hasAgentData
        ? (languages.python?.zero_shot_avg_cost_comparable || 0) * 1000
        : (languages.python?.zero_shot_avg_cost || 0) * 1000
    },
    {
      metric: 'With Repair',
      AILANG: hasAgentData
        ? (languages.ailang?.final_success_avg_cost_comparable || 0) * 1000
        : (languages.ailang?.final_success_avg_cost || 0) * 1000,
      Python: hasAgentData
        ? (languages.python?.final_success_avg_cost_comparable || 0) * 1000
        : (languages.python?.final_success_avg_cost || 0) * 1000
    },
    {
      metric: 'Agent',
      AILANG: hasAgentData ? (languages.ailang?.agent_avg_cost || 0) * 1000 : 0,
      Python: hasAgentData ? (languages.python?.agent_avg_cost || 0) * 1000 : 0
    }
  ];

  // Get agent benchmark list from data
  const agentBenchmarks = aggregates.agentBenchmarks || [];

  return (
    <>
      {hasAgentData && agentBenchmarks.length > 0 && (
        <div className={styles.benchmarkNote}>
          <strong>Note:</strong> Agent comparison uses only benchmarks tested in agent mode: {agentBenchmarks.join(', ')}
        </div>
      )}

      {/* Success Rate Radar Chart */}
      <div className={styles.radarCard} style={{ maxWidth: '600px', margin: '2rem auto' }}>
        <h4 className={styles.radarTitle}>Success Rate (%)</h4>
        <p className={styles.radarSubtitle}>
          {hasAgentData ? 'Comparing on same benchmarks' : 'Comparing all evaluation approaches'}
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
