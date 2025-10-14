import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function ModelChart({ models }) {
  // Transform data for recharts - now with per-language breakdown
  const chartData = Object.entries(models).map(([name, stats]) => {
    const shortName = formatModelName(name);
    const data = {
      name: shortName,
      fullName: name,
      runs: stats.totalRuns
    };

    // Use per-language stats if available (NEW DATA!)
    if (stats.languages && stats.languages.ailang && stats.languages.python) {
      data['AILANG'] = (stats.languages.ailang.successRate * 100).toFixed(1);
      data['Python'] = (stats.languages.python.successRate * 100).toFixed(1);
      data.ailangTokens = Math.round(stats.languages.ailang.avgTokens);
      data.pythonTokens = Math.round(stats.languages.python.avgTokens);
      data.ailangRuns = stats.languages.ailang.totalRuns;
      data.pythonRuns = stats.languages.python.totalRuns;
    } else {
      // Fallback to aggregates (old behavior)
      data['Zero-Shot'] = (stats.aggregates.zeroShotSuccess * 100).toFixed(1);
      data['Final (with repair)'] = (stats.aggregates.finalSuccess * 100).toFixed(1);
    }

    return data;
  });

  // Sort by AILANG success rate (or zero-shot if no language data)
  chartData.sort((a, b) => {
    const aVal = parseFloat(a['AILANG'] || a['Zero-Shot'] || 0);
    const bVal = parseFloat(b['AILANG'] || b['Zero-Shot'] || 0);
    return bVal - aVal;
  });

  // Check if we have language-specific data
  const hasLanguageData = chartData.some(d => d['AILANG'] !== undefined);

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data['AILANG'] !== undefined ? (
            <>
              <p className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: '#2e8555'}} />
                AILANG: {data['AILANG']}% ({data.ailangRuns} runs, {data.ailangTokens} tokens)
              </p>
              <p className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: '#25c2a0'}} />
                Python: {data['Python']}% ({data.pythonRuns} runs, {data.pythonTokens} tokens)
              </p>
              <p className={styles.tooltipRuns}>
                Gap: {(parseFloat(data['AILANG']) - parseFloat(data['Python'])).toFixed(1)}%
              </p>
            </>
          ) : (
            <>
              <p className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: '#2e8555'}} />
                Zero-Shot: {data['Zero-Shot']}%
              </p>
              <p className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: '#25c2a0'}} />
                Final: {data['Final (with repair)']}%
              </p>
              <p className={styles.tooltipRuns}>({data.runs} runs)</p>
            </>
          )}
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <ResponsiveContainer width="100%" height={350}>
        <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 60 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            dataKey="name"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            angle={-45}
            textAnchor="end"
            height={80}
          />
          <YAxis
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
            label={{ value: 'Success Rate (%)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="circle"
          />
          {hasLanguageData ? (
            <>
              <Bar
                dataKey="AILANG"
                fill="var(--ifm-color-primary-dark)"
                radius={[8, 8, 0, 0]}
              />
              <Bar
                dataKey="Python"
                fill="var(--ifm-color-success-dark)"
                radius={[8, 8, 0, 0]}
              />
            </>
          ) : (
            <>
              <Bar
                dataKey="Zero-Shot"
                fill="var(--ifm-color-primary-dark)"
                radius={[8, 8, 0, 0]}
              />
              <Bar
                dataKey="Final (with repair)"
                fill="var(--ifm-color-primary-light)"
                radius={[8, 8, 0, 0]}
              />
            </>
          )}
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function formatModelName(name) {
  // Format model names consistently with models.yml conventions
  // Use proper capitalization and version numbers
  // Check most specific patterns first
  if (name.includes('claude-sonnet-4-5')) return 'Claude Sonnet 4.5';
  if (name.includes('claude-sonnet')) return 'Claude Sonnet';
  if (name.includes('claude-opus')) return 'Claude Opus';
  if (name.includes('gpt-4o-mini')) return 'GPT-4o Mini';
  if (name.includes('gpt-5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5')) return 'GPT-5';
  if (name.includes('gpt-4o')) return 'GPT-4o';
  if (name.includes('gpt-4')) return 'GPT-4';
  if (name.includes('gemini-2-5-flash') || name.includes('gemini-2.5-flash')) return 'Gemini 2.5 Flash';
  if (name.includes('gemini-2-5-pro') || name.includes('gemini-2.5-pro')) return 'Gemini 2.5 Pro';
  if (name.includes('gemini-pro')) return 'Gemini Pro';
  if (name.includes('gemini')) return 'Gemini';

  // Fallback: capitalize first letter of each word
  return name.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
}
