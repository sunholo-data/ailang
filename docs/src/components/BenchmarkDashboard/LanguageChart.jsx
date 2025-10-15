import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function LanguageChart({ languages }) {
  if (!languages || Object.keys(languages).length === 0) {
    return null;
  }

  // Get Python as baseline
  const pythonStats = languages.python;
  const ailangStats = languages.ailang;

  if (!pythonStats || !ailangStats) {
    return null;
  }

  // Calculate deltas (AILANG vs Python baseline)
  const successDelta = ((ailangStats.success_rate - pythonStats.success_rate) * 100).toFixed(1);
  const tokenDelta = ((ailangStats.avg_tokens - pythonStats.avg_tokens) / pythonStats.avg_tokens * 100).toFixed(1);

  // Transform data for recharts
  const chartData = [
    {
      name: 'Python',
      'Success Rate (%)': parseFloat((pythonStats.success_rate * 100).toFixed(1)),
      'Avg Output Tokens': Math.round(pythonStats.avg_tokens),
      runs: pythonStats.total_runs
    },
    {
      name: 'AILANG',
      'Success Rate (%)': parseFloat((ailangStats.success_rate * 100).toFixed(1)),
      'Avg Output Tokens': Math.round(ailangStats.avg_tokens),
      runs: ailangStats.total_runs,
      successDelta,
      tokenDelta
    }
  ];

  // Custom tooltip with delta
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const isAilang = label === 'AILANG';
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}><strong>{label}</strong></p>
          <p className={styles.tooltipValue}>
            Success Rate: <strong>{data['Success Rate (%)']}%</strong>
            {isAilang && (
              <span className={styles.tooltipDelta} style={{color: parseFloat(data.successDelta) < 0 ? '#d32f2f' : '#2e8555'}}>
                {' '}({data.successDelta > 0 ? '+' : ''}{data.successDelta}% vs Python)
              </span>
            )}
          </p>
          <p className={styles.tooltipValue}>
            Output Tokens: <strong>{data['Avg Output Tokens']}</strong>
            {isAilang && (
              <span className={styles.tooltipDelta} style={{color: parseFloat(data.tokenDelta) > 0 ? '#d32f2f' : '#2e8555'}}>
                {' '}({data.tokenDelta > 0 ? '+' : ''}{data.tokenDelta}% vs Python)
              </span>
            )}
          </p>
          <p className={styles.tooltipRuns}>{data.runs} runs</p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <div className={styles.chartTitle}>Success Rate Comparison</div>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            dataKey="name"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
          />
          <YAxis
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
            domain={[0, 100]}
            label={{ value: 'Success Rate (%)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="circle"
          />
          <Bar
            dataKey="Success Rate (%)"
            fill="var(--ifm-color-primary)"
            radius={[8, 8, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
