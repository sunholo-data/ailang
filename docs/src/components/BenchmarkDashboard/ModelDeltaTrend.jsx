import React, { useState, useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import styles from './styles.module.css';
import { useEvents, annotationColor, groupByVersion, snapEventsToVersions } from './useEvents';
import { assignModelColors, getProvider } from './modelColors';
import { useOSHistory, mergeOSHistory } from './osHistory';

// Provider grouping — collapses the 30+ per-model lines into one averaged line per
// provider. 'other' = cloud open-source (OpenRouter). 'local' = the on-device rig
// (qwen3/gemma4 via motoko/opencode/pi), folded in from os/history.json.
const PROVIDER_ORDER = ['anthropic', 'openai', 'google', 'other', 'local'];
const PROVIDER_LABEL = { anthropic: 'Anthropic', openai: 'OpenAI', google: 'Google', other: 'Open-source', local: 'Local agent' };
const PROVIDER_COLOR = { anthropic: '#E67E22', openai: '#16a34a', google: '#2E86DE', other: '#8b5cf6', local: '#0891b2' };

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

export default function ModelDeltaTrend({ history, events, selectedTier, coverage }) {
  // Show the same events on both trend charts so release context is
  // consistent at a glance; taxonomy events get grouped+summarized in the
  // tooltip rather than filtered out.
  const annotations = useEvents(events, { selectedTier });
  const eventsByVersion = groupByVersion(annotations, formatVersion);
  // Default to provider grouping so the chart is readable at a glance; users can
  // switch to per-model for detail.
  const [groupBy, setGroupBy] = useState('provider');
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
  // Same solo-then-add focus behaviour for the PROVIDER legend (empty = show all).
  const [selectedProviders, setSelectedProviders] = useState(() => new Set());
  const isProviderVisible = (p) => selectedProviders.size === 0 || selectedProviders.has(p);
  const toggleProvider = (p) => {
    setSelectedProviders((prev) => {
      const next = new Set(prev);
      if (next.has(p)) next.delete(p);
      else next.add(p);
      return next;
    });
  };

  // Fold the on-device rig history in as a "Local agent" provider (shared helper).
  const osHistory = useOSHistory();
  const mergedHistory = useMemo(() => mergeOSHistory(history, osHistory), [history, osHistory]);
  const localIncomplete = (m) => getProvider(m) === 'local' && (coverage ? coverage.isProvisional(m) : true);

  // Tier-scoped source: when a tier is selected read the per-tier snapshot
  // so the gap updates when the user flips between Core/Stretch.
  const tierScopedStats = (entry) => {
    if (selectedTier) return entry.tiers?.[selectedTier]?.modelStats || null;
    return entry.modelStats || null;
  };

  // Filter out entries with invalid timestamps or no model data
  const validHistory = mergedHistory.filter(h => {
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
      // Local models are SHOWN (with a "*" while coverage is incomplete) so the
      // on-device gap is visible; cloud models are always full-coverage here.
      Object.keys(ms).forEach(model => allModels.add(model));
    }
  });

  // Provider-grouped color assignment — see modelColors.js.
  const modelColors = assignModelColors(allModels);
  const anyLocalIncomplete = Array.from(allModels).some(localIncomplete);

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

  // Provider grouping: one line per provider = average of its member models'
  // delta at each version (nulls skipped so gated points don't drag the mean).
  const providers = PROVIDER_ORDER.filter((p) => Array.from(allModels).some((m) => getProvider(m) === p));
  const providerChartData = chartData.map((pt) => {
    const out = { version: pt.version, date: pt.date };
    providers.forEach((p) => {
      const vals = Array.from(allModels)
        .filter((m) => getProvider(m) === p)
        .map((m) => pt[m])
        .filter((v) => v != null && !Number.isNaN(v));
      out[p] = vals.length ? parseFloat((vals.reduce((a, b) => a + b, 0) / vals.length).toFixed(1)) : null;
    });
    return out;
  });

  const byProvider = groupBy === 'provider';
  const activeData = byProvider ? providerChartData : chartData;
  const seriesKeys = byProvider ? providers.filter(isProviderVisible) : Array.from(allModels).filter(isVisible);
  const seriesColor = (k) => (byProvider ? PROVIDER_COLOR[k] : (modelColors.get(k) || '#999'));
  const seriesLabel = (k) => (byProvider ? PROVIDER_LABEL[k] || k : formatModelName(k));

  // Snap annotations onto real x-axis points so events at versions with no
  // baseline (e.g. the v0.29.0 benchmark additions) still render.
  const snappedEvents = snapEventsToVersions(eventsByVersion, activeData.map((d) => d.version));

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const eventsHere = snappedEvents.get(label) || [];
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          {payload.map((entry, index) => {
            const value = entry.value;
            return (
              <p key={index} className={styles.tooltipValue}>
                <span className={styles.tooltipDot} style={{backgroundColor: entry.color}} />
                {seriesLabel(entry.name)}: {value >= 0 ? '+' : ''}{value}%
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
      <div style={{ display: 'flex', gap: 6, alignItems: 'center', margin: '8px 0 4px' }}>
        <span style={{ fontSize: '0.82em', color: 'var(--ifm-color-emphasis-600)' }}>Group by:</span>
        {['provider', 'model'].map((g) => (
          <button
            key={g}
            type="button"
            onClick={() => setGroupBy(g)}
            style={{
              padding: '3px 11px', cursor: 'pointer', borderRadius: 6, fontSize: '0.85em', fontWeight: 600,
              border: '1px solid var(--ifm-color-emphasis-300)',
              background: groupBy === g ? 'var(--ifm-color-primary)' : 'transparent',
              color: groupBy === g ? '#fff' : 'var(--ifm-color-emphasis-800)',
            }}
          >{g === 'provider' ? 'Provider' : 'Model'}</button>
        ))}
      </div>
      <ResponsiveContainer width="100%" height={400}>
        <LineChart data={activeData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
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
          {Array.from(snappedEvents.entries()).map(([formattedVersion, evs]) => {
            const exists = chartData.some(d => d.version === formattedVersion);
            if (!exists) return null;
            const color = annotationColor(evs[0]);
            // Clean dot marker; event details + count live in the hover tooltip.
            const marker = '●';
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
          {seriesKeys.map((key) => (
            <Line
              key={key}
              type="linear"
              dataKey={key}
              stroke={seriesColor(key)}
              strokeWidth={byProvider ? 3 : 2}
              dot={{ r: byProvider ? 3 : 4 }}
              activeDot={{ r: 6 }}
              connectNulls
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
      {byProvider ? (
        <div className={styles.chipLegend}>
          <p className={styles.chipLegendHint}>
            {selectedProviders.size === 0
              ? 'Averaged per provider — click one to focus it; switch to “Model” for per-model detail.'
              : `Showing ${selectedProviders.size} of ${providers.length} — click more to add, or click again to remove.`}
          </p>
          {providers.map((p) => {
            const active = isProviderVisible(p);
            return (
              <button
                key={p}
                type="button"
                className={`${styles.legendChip} ${active ? '' : styles.legendChipHidden}`}
                onClick={() => toggleProvider(p)}
              >
                <span className={styles.legendChipDot} style={{ backgroundColor: PROVIDER_COLOR[p] }} />
                {PROVIDER_LABEL[p]}{p === 'local' && anyLocalIncomplete ? ' *' : ''}
              </button>
            );
          })}
          {selectedProviders.size > 0 && (
            <button
              type="button"
              className={styles.legendChip}
              onClick={() => setSelectedProviders(new Set())}
              title="Reset to show all providers"
            >
              Reset
            </button>
          )}
          {anyLocalIncomplete && (
            <p className={styles.chipLegendHint} style={{ width: '100%', marginTop: 4 }}>
              * on-device scores are over an <strong>incomplete</strong> benchmark set so far — they fill
              in as the rotation runs.
            </p>
          )}
        </div>
      ) : (
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
                {formatModelName(modelName)}{localIncomplete(modelName) ? ' *' : ''}
              </button>
            );
          })}
          {anyLocalIncomplete && (
            <p className={styles.chipLegendHint} style={{ width: '100%', marginTop: 4 }}>
              * on-device scores are over an <strong>incomplete</strong> benchmark set so far — they fill
              in as the rotation runs.
            </p>
          )}
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
      )}
    </div>
  );
}
