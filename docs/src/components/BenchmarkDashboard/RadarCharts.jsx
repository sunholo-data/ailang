import React from 'react';
import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, Legend, Tooltip, ResponsiveContainer } from 'recharts';
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

  // 1. Zero-Shot Performance (first attempt only)
  const zeroShotData = [
    {
      metric: 'Success Rate (%)',
      AILANG: (languages.ailang?.zero_shot_success || aggregates.zeroShotSuccess || 0) * 100,
      Python: (languages.python?.zero_shot_success || 0) * 100
    },
    {
      metric: 'Avg Tokens (K)',
      AILANG: ((languages.ailang?.avg_tokens || 0) / 1000),
      Python: ((languages.python?.avg_tokens || 0) / 1000)
    },
    {
      metric: 'Avg Cost ($)',
      AILANG: (languages.ailang?.avg_cost_usd || 0) * 100, // Scale to cents for visibility
      Python: (languages.python?.avg_cost_usd || 0) * 100
    }
  ];

  // 2. With Repair (0-shot + self-repair)
  const withRepairData = [
    {
      metric: 'Final Success (%)',
      AILANG: (languages.ailang?.success_rate || aggregates.finalSuccess || 0) * 100,
      Python: (languages.python?.success_rate || 0) * 100
    },
    {
      metric: 'Repair Rate (%)',
      AILANG: (languages.ailang?.repair_success_rate || aggregates.repairSuccessRate || 0) * 100,
      Python: (languages.python?.repair_success_rate || 0) * 100
    },
    {
      metric: 'Total Tokens (K)',
      AILANG: ((languages.ailang?.total_tokens || 0) / 1000),
      Python: ((languages.python?.total_tokens || 0) / 1000)
    }
  ];

  // 3. Agent Evaluation (multi-turn)
  const agentData = hasAgentData ? [
    {
      metric: 'Success Rate (%)',
      AILANG: (languages.ailang?.agent_success_rate || 0) * 100,
      Python: (languages.python?.agent_success_rate || 0) * 100
    },
    {
      metric: 'Avg Turns',
      AILANG: languages.ailang?.agent_avg_turns || 0,
      Python: languages.python?.agent_avg_turns || 0
    },
    {
      metric: 'Avg Tokens (K)',
      AILANG: ((languages.ailang?.agent_avg_tokens || 0) / 1000),
      Python: ((languages.python?.agent_avg_tokens || 0) / 1000)
    }
  ] : [];

  // 4. Overall Comparison (all approaches normalized)
  const overallData = [
    {
      metric: '0-Shot Success',
      AILANG: (languages.ailang?.zero_shot_success || aggregates.zeroShotSuccess || 0) * 100,
      Python: (languages.python?.zero_shot_success || 0) * 100
    },
    {
      metric: '+Repair Success',
      AILANG: (languages.ailang?.success_rate || aggregates.finalSuccess || 0) * 100,
      Python: (languages.python?.success_rate || 0) * 100
    },
    {
      metric: 'Agent Success',
      AILANG: hasAgentData ? (languages.ailang?.agent_success_rate || 0) * 100 : 0,
      Python: hasAgentData ? (languages.python?.agent_success_rate || 0) * 100 : 0
    }
  ];

  return (
    <div className={styles.radarGrid}>
      {/* 1. Zero-Shot Performance */}
      <div className={styles.radarCard}>
        <h4 className={styles.radarTitle}>Zero-Shot (First Attempt)</h4>
        <p className={styles.radarSubtitle}>Single LLM call, no feedback</p>
        <ResponsiveContainer width="100%" height={300}>
          <RadarChart data={zeroShotData}>
            <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
            <PolarAngleAxis
              dataKey="metric"
              tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 11 }}
            />
            <PolarRadiusAxis
              angle={90}
              domain={[0, 'dataMax']}
              tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 9 }}
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
            <Legend wrapperStyle={{ fontSize: '11px' }} />
            <Tooltip content={<CustomTooltip />} />
          </RadarChart>
        </ResponsiveContainer>
      </div>

      {/* 2. With Self-Repair */}
      <div className={styles.radarCard}>
        <h4 className={styles.radarTitle}>With Self-Repair</h4>
        <p className={styles.radarSubtitle}>Compiler feedback + 1 retry</p>
        <ResponsiveContainer width="100%" height={300}>
          <RadarChart data={withRepairData}>
            <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
            <PolarAngleAxis
              dataKey="metric"
              tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 11 }}
            />
            <PolarRadiusAxis
              angle={90}
              domain={[0, 'dataMax']}
              tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 9 }}
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
            <Legend wrapperStyle={{ fontSize: '11px' }} />
            <Tooltip content={<CustomTooltip />} />
          </RadarChart>
        </ResponsiveContainer>
      </div>

      {/* 3. Agent Evaluation (multi-turn) */}
      <div className={styles.radarCard}>
        <h4 className={styles.radarTitle}>Agent (Multi-Turn)</h4>
        <p className={styles.radarSubtitle}>
          {hasAgentData ? `Avg ${aggregates.avgAgentTurns?.toFixed(1)} turns` : 'No data yet'}
        </p>
        {hasAgentData ? (
          <ResponsiveContainer width="100%" height={300}>
            <RadarChart data={agentData}>
              <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
              <PolarAngleAxis
                dataKey="metric"
                tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 11 }}
              />
              <PolarRadiusAxis
                angle={90}
                domain={[0, 'dataMax']}
                tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 9 }}
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
              <Legend wrapperStyle={{ fontSize: '11px' }} />
              <Tooltip content={<CustomTooltip />} />
            </RadarChart>
          </ResponsiveContainer>
        ) : (
          <div className={styles.noData}>
            <p>Agent evaluation data not available</p>
            <p className={styles.noDataHint}>Run eval with --agent flag</p>
          </div>
        )}
      </div>

      {/* 4. Overall Comparison */}
      <div className={styles.radarCard}>
        <h4 className={styles.radarTitle}>Overall Comparison</h4>
        <p className={styles.radarSubtitle}>Success rates across all modes</p>
        <ResponsiveContainer width="100%" height={300}>
          <RadarChart data={overallData}>
            <PolarGrid stroke="var(--ifm-color-emphasis-300)" />
            <PolarAngleAxis
              dataKey="metric"
              tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 11 }}
            />
            <PolarRadiusAxis
              angle={90}
              domain={[0, 100]}
              tick={{ fill: 'var(--ifm-color-emphasis-600)', fontSize: 9 }}
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
            <Legend wrapperStyle={{ fontSize: '11px' }} />
            <Tooltip content={<CustomTooltip />} />
          </RadarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
