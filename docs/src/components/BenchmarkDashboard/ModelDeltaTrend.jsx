import React, { useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import styles from './styles.module.css';
import { useEvents, annotationColor, groupByVersion } from './useEvents';
import { assignModelColors } from './modelColors';

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

export default function ModelDeltaTrend({ history, events, selectedTier }) {
  // Show the same events on both trend charts so release context is
  // consistent at a glance; taxonomy events get grouped+summarized in the
  // tooltip rather than filtered out.
  const annotations = useEvents(events, { selectedTier });
  const eventsByVersion = groupByVersion(annotations, formatVersion);
  // Same solo-then-add semantics as PerModelTrend: empty set = show all.
  const [selectedModels, setSelectedModels] = useState(() => new Set());
  const isVisible = (m) => selectedModels.size === 0 || selectedModels.has(m);
  const toggleModel = (m) => {
    setSelectedModels((prev) => {
      const next = new Set(prev);
      if (next.has(m)) next.delete(m);
      else next.add(m);
      return next;
    });
  };

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

  // Provider-grouped color assignment — see modelColors.js.
  const modelColors = assignModelColors(allModels);

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
      const eventsHere = eventsByVersion.get(label) || [];
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          {payload.map((entry, index) => {
            const value = entry.value;
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
          {eventsHere.length > 0 && (
            <>
              <p className={styles.tooltipRuns} style={{marginTop: '8px', fontSize: '11px', color: '#666'}}>
                Release events:
              </p>
              {eventsHere.map((ev, i) => (
                <p key={i} className={styles.tooltipValue} style={{fontSize: '11px'}}>
                  <span className={styles.tooltipDot} style={{backgroundColor: annotationColor(ev)}} />
                  {ev.label}
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
          {Array.from(eventsByVersion.entries()).map(([formattedVersion, evs]) => {
            const exists = chartData.some(d => d.version === formattedVersion);
            if (!exists) return null;
            const color = annotationColor(evs[0]);
            const marker = evs.length > 1 ? `● ${evs.length}` : '●';
            return (
              <ReferenceLine
                key={`ev-${formattedVersion}`}
                x={formattedVersion}
                stroke={color}
                strokeDasharray="4 4"
                label={{ value: marker, position: 'top', fill: color, fontSize: 12, fontWeight: 600 }}
              />
            );
          })}
          <Tooltip content={<CustomTooltip />} wrapperStyle={{ zIndex: 1000, outline: 'none' }} />
          {Array.from(allModels)
            .filter(isVisible)
            .map(modelName => (
              <Line
                key={modelName}
                type="linear"
                dataKey={modelName}
                stroke={modelColors.get(modelName) || '#999'}
                strokeWidth={2}
                dot={{ r: 4 }}
                activeDot={{ r: 6 }}
                connectNulls
              />
          ))}
        </LineChart>
      </ResponsiveContainer>
      <div className={styles.chipLegend}>
        <p className={styles.chipLegendHint}>
          {selectedModels.size === 0
            ? 'Click a model to focus it; click more to compare.'
            : `Showing ${selectedModels.size} of ${allModels.size} — click more to add, or click again to remove.`}
        </p>
        {Array.from(allModels).map((modelName) => {
          const active = isVisible(modelName);
          const color = modelColors.get(modelName) || '#999';
          return (
            <button
              key={modelName}
              type="button"
              className={`${styles.legendChip} ${active ? '' : styles.legendChipHidden}`}
              onClick={() => toggleModel(modelName)}
            >
              <span className={styles.legendChipDot} style={{ backgroundColor: color }} />
              {formatModelName(modelName)}
            </button>
          );
        })}
        {selectedModels.size > 0 && (
          <button
            type="button"
            className={styles.legendChip}
            onClick={() => setSelectedModels(new Set())}
            title="Reset to show all models"
          >
            Reset
          </button>
        )}
      </div>
    </div>
  );
}
