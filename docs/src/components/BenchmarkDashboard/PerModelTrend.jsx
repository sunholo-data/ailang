import React, { useState, useEffect, useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import styles from './styles.module.css';
import { useEvents, annotationColor, groupByVersion, snapEventsToVersions } from './useEvents';
import { assignModelColors, getProvider } from './modelColors';

// Provider grouping — collapses 30+ per-model lines into one averaged line per
// provider. 'other' = cloud open-source (DeepSeek / GLM / MiniMax / Kimi via
// OpenRouter). 'local' = the on-device rig (qwen3/gemma4 through motoko/opencode/pi,
// $0/run) folded in from os/history.json.
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

// Metric options for the M4 dropdown (M-EVAL-COST-AND-SPEED-BUDGETS).
// Success rate has full history coverage. TTS / cost-per-success only land
// in history once M5 reruns the suite with the new measurement paths — until
// then we plot a single point per model at the latest version using the
// current `models` snapshot.
const METRIC_OPTIONS = [
  { id: 'successRate', label: 'Success Rate %', unit: '%', historic: true },
  { id: 'tts',         label: 'Time to Success (sec)', unit: 's', historic: false },
  { id: 'costPerSuccess', label: 'Cost per Success ($)', unit: '$', historic: false },
];

export default function PerModelTrend({ history, events, selectedTier, models: currentModels }) {
  const [selectedLanguage, setSelectedLanguage] = useState('ailang');
  const [selectedMetric, setSelectedMetric] = useState('successRate');
  const metric = METRIC_OPTIONS.find((m) => m.id === selectedMetric) || METRIC_OPTIONS[0];
  // Selected-models set: empty Set = "show all" (default). Clicking a chip
  // when the set is empty solos that model; subsequent clicks add or remove
  // from the selection. Clicking the last remaining selection clears back
  // to "show all" — lets the user focus one model then add peers to compare.
  const [selectedModels, setSelectedModels] = useState(() => new Set());
  // Default to provider grouping so the chart is readable; users can switch to per-model.
  const [groupBy, setGroupBy] = useState('provider');
  const isVisible = (m) => selectedModels.size === 0 || selectedModels.has(m);
  const toggleModel = (m) => {
    setSelectedModels((prev) => {
      const next = new Set(prev);
      if (next.has(m)) next.delete(m);
      else next.add(m);
      return next;
    });
  };
  const annotations = useEvents(events, { selectedTier });
  const eventsByVersion = groupByVersion(annotations, formatVersion);

  // Fold the on-device rig's version-trend (os/history.json) into the cloud
  // history so local model×harness combos (motoko/opencode/pi on qwen) appear as a
  // "Local agent" provider line + per-model breakdowns. Runtime fetch; a missing
  // file is a no-op. Local data is all-tier only, so it shows in the "All" view.
  const [osHistory, setOsHistory] = useState(null);
  useEffect(() => {
    fetch('/benchmarks/os/history.json')
      .then((r) => (r.ok ? r.json() : []))
      .then((h) => setOsHistory(Array.isArray(h) ? h : []))
      .catch(() => setOsHistory([]));
  }, []);
  const mergedHistory = useMemo(() => {
    if (!osHistory || osHistory.length === 0) return history;
    const osByVer = {};
    osHistory.forEach((e) => { if (e && e.ailang_version) osByVer[e.ailang_version] = e.rows || []; });
    return history.map((entry) => {
      const base = (entry.version || '').split('-')[0]; // v0.29.2 from v0.29.2-29-g…
      const rows = osByVer[entry.version] || osByVer[base];
      if (!rows || !rows.length) return entry;
      const modelStats = { ...(entry.modelStats || {}) };
      rows.forEach((r) => {
        if (!r || !r.model || !r.lang) return;
        const ms = {};
        for (const [l, rate] of Object.entries(r.lang)) {
          if (typeof rate === 'number') ms[l] = { successRate: rate, totalRuns: 1 };
        }
        modelStats[r.model] = ms;
      });
      return { ...entry, modelStats };
    });
  }, [history, osHistory]);

  // When a tier is selected, read history[i].tiers[t].modelStats instead of
  // the all-tier history[i].modelStats. Historic baselines without tier
  // data simply drop out of the view (connectNulls bridges the gap).
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
      Object.keys(ms).forEach(model => allModels.add(model));
    }
  });

  // Track api-error gate metadata per (version, model) so the tooltip can
  // explain why a dot is missing. Key: `${version}|${model}`.
  const apiErrorMeta = {};

  // Read a per-(version,model,lang) metric value. Success rate has complete
  // historic coverage; TTS / cost-per-success may not be in history yet
  // (added by M3 — pre-M3 baselines simply return null and the line skips).
  const valueFor = (langStats) => {
    if (!langStats) return null;
    if (selectedMetric === 'successRate') {
      return parseFloat((langStats.successRate * 100).toFixed(1));
    }
    if (selectedMetric === 'tts') {
      // Look for either snake_case or camelCase — M3 emits camelCase but
      // future history shapes might mirror the top-level efficiency block.
      const ms = langStats.medianTimeToSuccessMs ?? langStats.median_time_to_success_ms;
      return typeof ms === 'number' && ms > 0 ? parseFloat((ms / 1000).toFixed(2)) : null;
    }
    if (selectedMetric === 'costPerSuccess') {
      const usd = langStats.p90CostPerSuccess ?? langStats.p90_cost_per_success;
      return typeof usd === 'number' && usd > 0 ? parseFloat(usd.toFixed(4)) : null;
    }
    return null;
  };

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

        // API-error gate only meaningfully applies to success-rate (a 0%
        // due to quota errors is misleading). For latency/cost metrics
        // we just emit null when the underlying value isn't available.
        if (gated && selectedMetric === 'successRate') {
          apiErrorMeta[`${versionLabel}|${modelName}`] = { apiErrors, total };
          point[modelName] = null;
          return;
        }

        point[modelName] = valueFor(lang);
      });
    }

    return point;
  });

  // Latest-snapshot fallback for metrics that history hasn't been backfilled
  // for yet (TTS, cost-per-success). We graft the current `models` prop's
  // efficiency block onto the most recent chartData row so the user sees at
  // least one dot per model. The dropdown stays useful pre-M5.
  const usingSnapshotFallback =
    !metric.historic &&
    chartData.length > 0 &&
    currentModels &&
    chartData[chartData.length - 1] &&
    Array.from(allModels).every((m) => chartData[chartData.length - 1][m] == null);

  if (usingSnapshotFallback) {
    const latestRow = chartData[chartData.length - 1];
    Object.entries(currentModels).forEach(([modelName, modelData]) => {
      const eff = modelData?.efficiency;
      if (!eff) return;
      if (selectedMetric === 'tts') {
        const ms = eff.median_time_to_success_ms;
        if (typeof ms === 'number' && ms > 0) {
          latestRow[modelName] = parseFloat((ms / 1000).toFixed(2));
        }
      } else if (selectedMetric === 'costPerSuccess') {
        const usd = eff.p90_cost_per_success;
        if (typeof usd === 'number' && usd > 0) {
          latestRow[modelName] = parseFloat(usd.toFixed(4));
        }
      }
    });
    // Make sure the latest snapshot's models are in allModels so they get
    // chips + lines even if the older history shape didn't list them.
    Object.keys(currentModels).forEach((m) => allModels.add(m));
  }

  // Provider-grouped color assignment (Anthropic/OpenAI/Google shades) so
  // new models don't fall through to grey when the static palette runs out.
  // Computed after the snapshot fallback has potentially added models so
  // every chip + line gets a stable colour.
  const modelColors = assignModelColors(allModels);

  // Provider grouping: one line per provider = average of member models' metric
  // per version (nulls skipped). Precision follows the metric ($ keeps 4 dp).
  const providers = PROVIDER_ORDER.filter((p) => Array.from(allModels).some((m) => getProvider(m) === p));
  const providerChartData = chartData.map((pt) => {
    const out = { version: pt.version, date: pt.date };
    providers.forEach((p) => {
      const vals = Array.from(allModels)
        .filter((m) => getProvider(m) === p)
        .map((m) => pt[m])
        .filter((v) => v != null && !Number.isNaN(v));
      out[p] = vals.length
        ? parseFloat((vals.reduce((a, b) => a + b, 0) / vals.length).toFixed(selectedMetric === 'costPerSuccess' ? 4 : 1))
        : null;
    });
    return out;
  });
  const byProvider = groupBy === 'provider';
  const activeData = byProvider ? providerChartData : chartData;
  const seriesKeys = byProvider ? providers : Array.from(allModels).filter(isVisible);
  const seriesColor = (k) => (byProvider ? PROVIDER_COLOR[k] : (modelColors.get(k) || '#999'));
  const seriesLabel = (k) => (byProvider ? PROVIDER_LABEL[k] || k : formatModelName(k));

  // Snap annotations onto real x-axis points so events at versions with no
  // baseline (e.g. the v0.29.0 benchmark additions) still render.
  const snappedEvents = snapEventsToVersions(eventsByVersion, activeData.map((d) => d.version));

  // Format a metric value for display (tooltip/axis ticks).
  const formatValue = (value) => {
    if (value == null) return '—';
    if (selectedMetric === 'successRate') return `${value}%`;
    if (selectedMetric === 'tts') return `${value}s`;
    if (selectedMetric === 'costPerSuccess') return `$${value.toFixed(4)}`;
    return String(value);
  };

  // Custom tooltip — shows API-error gate info for models with null points
  // on this baseline (infra failures, not code-quality 0%s).
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const gatedHere = Array.from(allModels)
        .map((m) => ({ model: m, meta: apiErrorMeta[`${label}|${m}`] }))
        .filter((r) => r.meta);
      const eventsHere = snappedEvents.get(label) || [];
      // Sort entries best-at-top. For successRate higher = better;
      // for tts and costPerSuccess lower = better.
      const lowerIsBetter = selectedMetric === 'tts' || selectedMetric === 'costPerSuccess';
      const sortedPayload = [...payload]
        .filter((e) => e.value != null)
        .sort((a, b) => lowerIsBetter ? a.value - b.value : b.value - a.value);
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          {sortedPayload.map((entry, index) => (
            <p key={entry.name || index} className={styles.tooltipValue}>
              <span className={styles.tooltipDot} style={{backgroundColor: entry.color}} />
              {seriesLabel(entry.name)}: {formatValue(entry.value)}
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
      <div className={styles.chartHeader}>
        <div className={styles.chartTitle}>
          {metric.label} by Model Over Time
          {selectedTier && <span style={{ fontWeight: 400, color: 'var(--ifm-color-emphasis-600)', fontSize: '0.85em' }}>
            {' '}({selectedTier} tier)
          </span>}
        </div>
        <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center', flexWrap: 'wrap' }}>
          <select
            value={selectedMetric}
            onChange={(e) => setSelectedMetric(e.target.value)}
            style={{
              padding: '4px 8px',
              fontSize: '0.85rem',
              border: '1px solid var(--ifm-color-emphasis-300)',
              borderRadius: 4,
              background: 'var(--ifm-background-surface-color)',
              color: 'var(--ifm-color-emphasis-900)',
            }}
            aria-label="Metric"
          >
            {METRIC_OPTIONS.map((opt) => (
              <option key={opt.id} value={opt.id}>{opt.label}</option>
            ))}
          </select>
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
          <div className={styles.languageToggle} title="Group the lines by provider or show every model">
            <button
              className={groupBy === 'provider' ? styles.toggleActive : styles.toggleInactive}
              onClick={() => setGroupBy('provider')}
            >
              Provider
            </button>
            <button
              className={groupBy === 'model' ? styles.toggleActive : styles.toggleInactive}
              onClick={() => setGroupBy('model')}
            >
              Model
            </button>
          </div>
        </div>
      </div>
      {usingSnapshotFallback && (
        <div style={{
          margin: '0.5rem 0 1rem',
          padding: '0.5rem 0.75rem',
          background: 'var(--ifm-color-emphasis-100)',
          borderLeft: '3px solid #F59E0B',
          fontSize: '0.85rem',
          color: 'var(--ifm-color-emphasis-800)',
        }}>
          History doesn't carry this metric yet — showing the latest snapshot only.
          M5 will backfill once the suite reruns with the new measurement paths.
        </div>
      )}
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
            domain={selectedMetric === 'successRate' ? [0, 100] : ['auto', 'auto']}
            tickFormatter={(v) => formatValue(v)}
            label={{
              value:
                selectedMetric === 'successRate' ? 'Success Rate (%)'
                : selectedMetric === 'tts' ? 'Time to Success (sec)'
                : 'Cost / Success ($)',
              angle: -90,
              position: 'insideLeft',
            }}
          />
          <Tooltip content={<CustomTooltip />} wrapperStyle={{ zIndex: 1000, outline: 'none' }} />
          {Array.from(snappedEvents.entries()).map(([formattedVersion, evs]) => {
            const exists = chartData.some(d => d.version === formattedVersion);
            if (!exists) return null;
            // One dashed line per version, colored by the first event. Full
            // label text lives in the tooltip — stacking multiple labels on
            // the axis is unreadable when a release carries several events.
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
          <p className={styles.chipLegendHint}>Averaged per provider — switch to “Model” for per-model detail.</p>
          {providers.map((p) => (
            <span key={p} className={styles.legendChip} style={{ cursor: 'default' }}>
              <span className={styles.legendChipDot} style={{ backgroundColor: PROVIDER_COLOR[p] }} />
              {PROVIDER_LABEL[p]}
            </span>
          ))}
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
      )}
    </div>
  );
}
