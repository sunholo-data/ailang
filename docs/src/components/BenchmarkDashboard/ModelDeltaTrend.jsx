import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ReferenceLine } from 'recharts';
import styles from './styles.module.css';
import { useEvents, annotationColor } from './useEvents';

function formatModelName(name) {
  // Check most specific patterns first
  if (name.includes('claude-sonnet-4-5')) return 'Claude Sonnet 4.5';
  if (name.includes('claude-haiku-4-5')) return 'Claude Haiku 4.5';
  if (name.includes('gpt-5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5')) return 'GPT-5';
  if (name.includes('gpt5-1-instant')) return 'Gpt5 1 Instant';
  if (name.includes('gpt5-1')) return 'Gpt5 1';
  if (name.includes('gemini-2-5-flash') || name.includes('gemini-2.5-flash')) return 'Gemini 2.5 Flash';
  if (name.includes('gemini-2-5-pro') || name.includes('gemini-2.5-pro')) return 'Gemini 2.5 Pro';
  if (name.includes('gemini-3-pro') || name.includes('gemini-3.0-pro')) return 'Gemini 3.0 Pro';
  // Fallback: capitalize first letter of each word
  return name.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
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

// Color palette for models (distinct colors)
const MODEL_COLORS = {
  'gpt5-1': '#FF6B6B',
  'gpt5-mini': '#FFA07A',
  'gpt5-1-instant': '#FFD700',
  'claude-sonnet-4-5': '#4ECDC4',
  'claude-haiku-4-5': '#95E1D3',
  'gemini-2-5-pro': '#9B59B6',
  'gemini-2-5-flash': '#C39BD3',
  'gemini-3-pro': '#8E44AD',
};

export default function ModelDeltaTrend({ history, events, selectedTier }) {
  // ModelDeltaTrend shows the per-model AILANG−Python gap; taxonomy-only
  // events don't shift that curve (they only change which benchmarks bin
  // where), so filter them out — otherwise the chart is noisy.
  const annotations = useEvents(events, {
    kinds: ['benchmark_add', 'benchmark_remove', 'prompt'],
    selectedTier,
  });

  // Tier-scoped source: when a tier is selected read the per-tier snapshot
  // so the gap updates when the user flips between Core/Stretch.
  const tierScopedStats = (entry) => {
    if (selectedTier) return entry.tiers?.[selectedTier]?.modelStats || null;
    return entry.modelStats || null;
  };

  // Filter out entries with invalid timestamps or no model data
  const validHistory = history.filter(h => {
    const date = new Date(h.timestamp);
    return date.getFullYear() > 2000 && tierScopedStats(h);
  });

  // Sort history by timestamp (oldest first for proper trend display)
  const sortedHistory = [...validHistory].sort((a, b) => {
    const dateA = new Date(a.timestamp);
    const dateB = new Date(b.timestamp);
    return dateA - dateB;
  });

  // Get list of all models that appear in history (tier-scoped)
  const allModels = new Set();
  sortedHistory.forEach(entry => {
    const ms = tierScopedStats(entry);
    if (ms) {
      Object.keys(ms).forEach(model => allModels.add(model));
    }
  });

  // Transform history data for recharts - calculate delta (AILANG - Python).
  // Same api-error gate as PerModelTrend: if either side's run is dominated
  // by infra failures, omit the point so the delta doesn't spike spuriously.
  const chartData = sortedHistory.map(baseline => {
    const point = {
      version: formatVersion(baseline.version),
      date: baseline.timestamp ? new Date(baseline.timestamp).toLocaleDateString() : '',
    };

    const ms = tierScopedStats(baseline);
    if (ms) {
      Object.entries(ms).forEach(([modelName, langStats]) => {
        if (langStats?.ailang && langStats?.python) {
          const ail = langStats.ailang;
          const py = langStats.python;
          const ailGated = (ail.totalRuns || 0) > 0 && (ail.apiErrorCount || 0) / ail.totalRuns >= 0.5;
          const pyGated = (py.totalRuns || 0) > 0 && (py.apiErrorCount || 0) / py.totalRuns >= 0.5;
          if (ailGated || pyGated) {
            point[modelName] = null;
            return;
          }
          const ailangRate = (ail.successRate || 0) * 100;
          const pythonRate = (py.successRate || 0) * 100;
          const delta = ailangRate - pythonRate;
          point[modelName] = parseFloat(delta.toFixed(1));
        }
      });
    }

    return point;
  });

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          {payload.map((entry, index) => {
            const value = entry.value;
            const color = value >= 0 ? '#2e8555' : '#e74c3c';
            return (
              <p key={index} className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: entry.color}} />
                {formatModelName(entry.name)}: {value >= 0 ? '+' : ''}{value}%
              </p>
            );
          })}
          <p className={styles.tooltipRuns} style={{marginTop: '8px', fontSize: '11px', color: '#666'}}>
            Positive = AILANG better, Negative = Python better
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <div className={styles.chartTitle}>
        AILANG vs Python Gap by Model
        {selectedTier && <span style={{ fontWeight: 400, color: 'var(--ifm-color-emphasis-600)', fontSize: '0.85em' }}>
          {' '}({selectedTier} tier)
        </span>}
      </div>
      <div className={styles.chartSubtitle}>
        Positive = AILANG performs better · Negative = Python performs better
      </div>
      <ResponsiveContainer width="100%" height={400}>
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
            label={{ value: 'Gap (AILANG - Python) %', angle: -90, position: 'insideLeft' }}
          />
          <ReferenceLine y={0} stroke="#666" strokeDasharray="3 3" strokeWidth={1} />
          {annotations.map(ann => {
            const formattedVersion = formatVersion(ann.version);
            const exists = chartData.some(d => d.version === formattedVersion);
            const color = annotationColor(ann);
            return exists ? (
              <ReferenceLine
                key={`${ann.version}-${ann.kind || 'event'}-${ann.label}`}
                x={formattedVersion}
                stroke={color}
                strokeDasharray="4 4"
                label={{ value: ann.label, position: 'top', fill: color, fontSize: 11 }}
              />
            ) : null;
          })}
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="circle"
            formatter={(value) => formatModelName(value)}
          />
          {Array.from(allModels).map(modelName => (
            <Line
              key={modelName}
              type="linear"
              dataKey={modelName}
              stroke={MODEL_COLORS[modelName] || '#999'}
              strokeWidth={2}
              dot={{ r: 4 }}
              activeDot={{ r: 6 }}
              connectNulls
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
