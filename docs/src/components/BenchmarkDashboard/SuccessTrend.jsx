import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function SuccessTrend({ history }) {
  // Sort history by timestamp (oldest first for proper trend display)
  const sortedHistory = [...history].sort((a, b) => {
    const dateA = new Date(a.timestamp);
    const dateB = new Date(b.timestamp);
    return dateA - dateB;
  });

  // Transform history data for recharts
  const chartData = sortedHistory.map(baseline => {
    // History entries have pre-calculated successRate from SuccessCount/TotalBenchmarks
    // The languages field indicates which language(s) the baseline ran for
    const successRate = (baseline.successRate || 0) * 100;
    const languages = baseline.languages || '';

    // If baseline only ran one language, show it in the appropriate column
    // If it ran both (comma-separated or empty string), show in both columns
    const isAilangOnly = languages === 'ailang';
    const isPythonOnly = languages === 'python';
    const isBoth = languages.includes(',') || languages === '';

    return {
      version: formatVersion(baseline.version),
      'AILANG': parseFloat((isAilangOnly || isBoth ? successRate : 0).toFixed(1)),
      'Python': parseFloat((isPythonOnly || isBoth ? successRate : 0).toFixed(1)),
      date: baseline.timestamp ? new Date(baseline.timestamp).toLocaleDateString() : ''
    };
  });

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          <p className={styles.tooltipValue}>
            <span className={styles.tooltipDot} style={{backgroundColor: '#2e8555'}} />
            AILANG: {data['AILANG']}%
          </p>
          {data['Python'] > 0 && (
            <p className={styles.tooltipValue}>
              <span className={styles.tooltipDot} style={{backgroundColor: '#ffa726'}} />
              Python: {data['Python']}%
            </p>
          )}
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            dataKey="version"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            angle={-45}
            textAnchor="end"
            height={80}
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
          <Line
            type="monotone"
            dataKey="AILANG"
            stroke="var(--ifm-color-primary)"
            strokeWidth={3}
            dot={{ r: 5 }}
            activeDot={{ r: 7 }}
          />
          <Line
            type="monotone"
            dataKey="Python"
            stroke="#ffa726"
            strokeWidth={3}
            dot={{ r: 5 }}
            activeDot={{ r: 7 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

function formatVersion(version) {
  // Shorten version strings for display
  if (!version) return 'Unknown';

  // Remove 'v' prefix if present
  version = version.replace(/^v/, '');

  // For git versions like "0.3.0-35-g3530d07", show "v0.3.0-35"
  const parts = version.split('-');
  if (parts.length >= 3) {
    return `v${parts[0]}-${parts[1]}`;
  }

  // For simple versions, show as-is
  return `v${version}`;
}
