import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

function formatModelName(name) {
  // Surface harness + provider as explicit suffixes. See
  // BenchmarkExplorer/index.jsx::modelShort for the canonical version.
  let s = name;
  let suffix = '';
  if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-')) { suffix = ' (agent)';     s = s.slice('opencode-'.length); }
  else if (s.startsWith('pi-'))       { suffix = ' (Pi)';        s = s.slice('pi-'.length); }
  else if (s.startsWith('or-'))       { suffix = ' (OR)';        s = s.slice('or-'.length); }
  s = s
    .replace(/^claude-/, 'Claude ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt5/, 'GPT-5')
    .replace(/^minimax-/, 'MiniMax ')
    .replace(/^glm-/, 'GLM ')
    .replace(/^kimi-/, 'Kimi ')
    .replace(/^qwen3-/, 'Qwen3 ')
    .replace(/^gemma4-/, 'Gemma4 ')
    .replace(/^deepseek-/, 'DeepSeek ')
    .replace(/-/g, ' ');
  return s + suffix;
}

export default function ModelTokenChart({ models }) {
  // Convert models data to chart format
  const chartData = Object.entries(models)
    // Only show models with language breakdown (has output token data)
    .filter(([name, stats]) => stats.languages && stats.languages.ailang && stats.languages.python)
    .map(([name, stats]) => {
      // Use output tokens from language breakdown (actual code generated)
      // Average AILANG and Python output tokens - with safe access
      const ailang = stats.languages?.ailang;
      const python = stats.languages?.python;
      const ailangTokens = ailang?.avgTokens || 0;
      const pythonTokens = python?.avgTokens || 0;
      const avgOutputTokens = (ailangTokens + pythonTokens) / 2;

      const avgCost = (stats.aggregates?.totalCostUSD || 0) / (stats.totalRuns || 1);

      return {
        name: formatModelName(name),
        'Avg Output Tokens': Math.round(avgOutputTokens),
        'Cost per Run ($)': parseFloat((avgCost * 1000).toFixed(3)), // Show as milli-dollars for better scale
        fullCost: avgCost, // Keep actual cost for tooltip
        ailangTokens: Math.round(ailangTokens),
        pythonTokens: Math.round(pythonTokens),
      };
    });

  // Sort by tokens (descending)
  chartData.sort((a, b) => b['Avg Output Tokens'] - a['Avg Output Tokens']);

  const CustomTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{data.name}</p>
          <p className={styles.tooltipValue}>
            Avg Tokens: <strong>{data['Avg Output Tokens']}</strong>
          </p>
          <p className={styles.tooltipRuns}>
            (AILANG: {data.ailangTokens}, Python: {data.pythonTokens})
          </p>
          <p className={styles.tooltipValue}>
            Cost/Run: <strong>${data.fullCost.toFixed(6)}</strong>
          </p>
          <p className={styles.tooltipRuns}>
            (${(data.fullCost * 1000).toFixed(3)} per 1000 runs)
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <div className={styles.chartTitle}>Token Usage by Model</div>
      <ResponsiveContainer width="100%" height={350}>
        <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 80 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-300)" />
          <XAxis
            dataKey="name"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            angle={-45}
            textAnchor="end"
            height={80}
          />
          <YAxis
            yAxisId="left"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
            label={{ value: 'Output Tokens', angle: -90, position: 'insideLeft' }}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
            label={{ value: 'Cost (milli-$ per run)', angle: 90, position: 'insideRight' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend wrapperStyle={{ paddingTop: '20px' }} />
          <Bar
            yAxisId="left"
            dataKey="Avg Output Tokens"
            fill="var(--ifm-color-primary)"
            radius={[8, 8, 0, 0]}
          />
          <Bar
            yAxisId="right"
            dataKey="Cost per Run ($)"
            fill="var(--ifm-color-success)"
            radius={[8, 8, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
      <div className={styles.chartNote}>
        💡 Output tokens = actual code generated (excludes reasoning tokens for GPT-5)
      </div>
    </div>
  );
}
