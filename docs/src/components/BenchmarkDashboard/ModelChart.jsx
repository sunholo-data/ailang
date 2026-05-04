import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

export default function ModelChart({ models }) {
  // Transform data for recharts - now with per-language breakdown
  const chartData = Object.entries(models)
    // Filter: Only show models with both AILANG and Python data
    .filter(([name, stats]) => {
      return stats.languages && stats.languages.ailang && stats.languages.python;
    })
    .map(([name, stats]) => {
      const shortName = formatModelName(name);
      const ailang = stats.languages?.ailang;
      const python = stats.languages?.python;
      const data = {
        name: shortName,
        fullName: name,
        runs: stats.totalRuns
      };

      // Use per-language stats with safe access
      data['AILANG'] = ((ailang?.successRate || 0) * 100).toFixed(1);
      data['Python'] = ((python?.successRate || 0) * 100).toFixed(1);
      data.ailangTokens = Math.round(ailang?.avgTokens || 0);
      data.pythonTokens = Math.round(python?.avgTokens || 0);
      data.ailangRuns = ailang?.totalRuns || 0;
      data.pythonRuns = python?.totalRuns || 0;

      return data;
    });

  // Sort by AILANG success rate (highest first)
  chartData.sort((a, b) => {
    return parseFloat(b['AILANG']) - parseFloat(a['AILANG']);
  });

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
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
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function formatModelName(name) {
  // Surface harness + provider as explicit suffixes. Mirrors
  // BenchmarkExplorer/index.jsx::modelShort so labels are consistent across
  // charts and tables. Examples:
  //   opencode-or-glm-5    → "GLM 5 (agent · OR)"
  //   opencode-sonnet-4-6  → "Sonnet 4.6 (agent)"
  //   or-glm-5             → "GLM 5 (OR)"
  //   claude-sonnet-4-6    → "Claude Sonnet 4.6"
  //   gpt5-4-mini          → "GPT-5 4 mini"
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
