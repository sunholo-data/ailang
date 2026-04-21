import React, { useState } from 'react';
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

export default function PerModelTrend({ history, events, selectedTier }) {
  const [selectedLanguage, setSelectedLanguage] = useState('ailang');
  const annotations = useEvents(events, { selectedTier });

  // When a tier is selected, read history[i].tiers[t].modelStats instead of
  // the all-tier history[i].modelStats. Historic baselines without tier
  // data simply drop out of the view (connectNulls bridges the gap).
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

  // Track api-error gate metadata per (version, model) so the tooltip can
  // explain why a dot is missing. Key: `${version}|${model}`.
  const apiErrorMeta = {};

  // Transform history data for recharts. Apply API-error 0% gate: if ≥50%
  // of a model's runs on this baseline were api_error, null out the point
  // instead of plotting a misleading 0% (OpenAI key revoked, quota hit, etc).
  const chartData = sortedHistory.map(baseline => {
    const versionLabel = formatVersion(baseline.version);
    const point = {
      version: versionLabel,
      date: baseline.timestamp ? new Date(baseline.timestamp).toLocaleDateString() : '',
    };

    const ms = tierScopedStats(baseline);
    if (ms) {
      Object.entries(ms).forEach(([modelName, langStats]) => {
        const lang = langStats?.[selectedLanguage];
        if (!lang) return;

        const total = lang.totalRuns || 0;
        const apiErrors = lang.apiErrorCount || 0;
        const gated = total > 0 && apiErrors / total >= 0.5;

        if (gated) {
          apiErrorMeta[`${versionLabel}|${modelName}`] = { apiErrors, total };
          point[modelName] = null;
          return;
        }

        const successRate = lang.successRate * 100;
        point[modelName] = parseFloat(successRate.toFixed(1));
      });
    }

    return point;
  });

  // Custom tooltip — shows API-error gate info for models with null points
  // on this baseline (infra failures, not code-quality 0%s).
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const gatedHere = Array.from(allModels)
        .map((m) => ({ model: m, meta: apiErrorMeta[`${label}|${m}`] }))
        .filter((r) => r.meta);
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          {payload.map((entry, index) => (
            <p key={index} className={styles.tooltipValue}>
              <span className={styles.tooltipDot} style={{backgroundColor: entry.color}} />
              {formatModelName(entry.name)}: {entry.value}%
            </p>
          ))}
          {gatedHere.length > 0 && (
            <>
              <p className={styles.tooltipRuns} style={{marginTop: '8px', fontSize: '11px', color: '#666'}}>
                API-error gate (omitted from chart):
              </p>
              {gatedHere.map(({ model, meta }) => (
                <p key={model} className={styles.tooltipValue} style={{fontSize: '11px', color: '#666'}}>
                  {formatModelName(model)}: — (API errors: {meta.apiErrors}/{meta.total})
                </p>
              ))}
            </>
          )}
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <div className={styles.chartHeader}>
        <div className={styles.chartTitle}>
          Success Rate by Model Over Time
          {selectedTier && <span style={{ fontWeight: 400, color: 'var(--ifm-color-emphasis-600)', fontSize: '0.85em' }}>
            {' '}({selectedTier} tier)
          </span>}
        </div>
        <div className={styles.languageToggle}>
          <button
            className={selectedLanguage === 'ailang' ? styles.toggleActive : styles.toggleInactive}
            onClick={() => setSelectedLanguage('ailang')}
          >
            AILANG
          </button>
          <button
            className={selectedLanguage === 'python' ? styles.toggleActive : styles.toggleInactive}
            onClick={() => setSelectedLanguage('python')}
          >
            Python
          </button>
        </div>
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
            domain={[0, 100]}
            label={{ value: 'Success Rate (%)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="circle"
            formatter={(value) => formatModelName(value)}
          />
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
