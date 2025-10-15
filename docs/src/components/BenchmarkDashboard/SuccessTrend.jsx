import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function SuccessTrend({ history, languages }) {
  // Filter out entries with invalid timestamps (0001-01-01 means no timestamp)
  const validHistory = history.filter(h => {
    const date = new Date(h.timestamp);
    return date.getFullYear() > 2000; // Only show entries with real timestamps
  });

  // Sort history by timestamp (oldest first for proper trend display)
  const sortedHistory = [...validHistory].sort((a, b) => {
    const dateA = new Date(a.timestamp);
    const dateB = new Date(b.timestamp);
    return dateA - dateB;
  });

  // Transform history data for recharts
  const chartData = sortedHistory.map((baseline, index) => {
    const langs = baseline.languages || '';
    const isLatest = index === sortedHistory.length - 1;

    let ailangRate = 0;
    let pythonRate = 0;

    // Check if this baseline has per-language stats (new format)
    if (baseline.languageStats) {
      ailangRate = (baseline.languageStats.ailang?.success_rate || 0) * 100;
      pythonRate = (baseline.languageStats.python?.success_rate || 0) * 100;
    } else if (isLatest && languages) {
      // Fallback: Use top-level language stats for latest version
      ailangRate = (languages.ailang?.success_rate || 0) * 100;
      pythonRate = (languages.python?.success_rate || 0) * 100;
    } else if (langs === 'ailang') {
      // AILANG-only baseline
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = combinedRate;
      pythonRate = 0;
    } else if (langs === 'python') {
      // Python-only baseline
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = 0;
      pythonRate = combinedRate;
    } else {
      // Both languages - use combined rate for both (legacy behavior)
      // This shouldn't happen with new export format
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = combinedRate;
      pythonRate = combinedRate;
    }

    return {
      version: formatVersion(baseline.version),
      'AILANG': parseFloat(ailangRate.toFixed(1)),
      'Python': parseFloat(pythonRate.toFixed(1)),
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
