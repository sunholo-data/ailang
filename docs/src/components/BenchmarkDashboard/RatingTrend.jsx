import React, { useMemo, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';
import { assignModelColors, getProvider } from './modelColors';

// RatingTrend — ELO over AILANG versions (M-EVAL-ROLLING-ELO M4).
//
// Two series that answer different questions and are deliberately NOT mixed on
// one axis:
//   · Model capability (anchored placement fit) — "how strong is each model"
//   · Language-direction index (direction fit, bridge models held fixed) —
//     "how hard is AILANG for a constant panel of models". FALLING = the
//     language/prompt got easier = progress. Shown with an inverted axis so
//     "up = better" holds for both views.
//
// Data: history[].ratings, published per release. Entries predating the rolling
// series simply have no `ratings` key and are skipped — never interpolated.
//
// Follows the m-eval-os-version-trend-redesign rules: props only (parent already
// runtime-fetched latest.json), no build-time import, no cache-busting, folded
// into the proven dashboard rather than a new standalone page.
// Provider vocabulary is shared verbatim with PerModelTrend so the page reads
// consistently: same grouping, same colours, same chip filters (Mark, 2026-08-27).
const PROVIDER_ORDER = ['anthropic', 'openai', 'google', 'other', 'local'];
const PROVIDER_LABEL = { anthropic: 'Anthropic', openai: 'OpenAI', google: 'Google', other: 'Open-source', local: 'Local agent' };
const PROVIDER_COLOR = { anthropic: '#E67E22', openai: '#16a34a', google: '#2E86DE', other: '#8b5cf6', local: '#0891b2' };

function formatModelName(name) {
  let s = name;
  let suffix = '';
  if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-')) { suffix = ' (agent)'; s = s.slice('opencode-'.length); }
  else if (s.startsWith('motoko-')) { suffix = ' (motoko)'; s = s.slice('motoko-'.length); }
  else if (s.startsWith('pi-')) { suffix = ' (Pi)'; s = s.slice('pi-'.length); }
  else if (s.startsWith('or-')) { suffix = ' (OR)'; s = s.slice('or-'.length); }
  return s + suffix;
}
// Release order must come from the VERSION, not the timestamp: baselines get
// re-banked and back-filled, so timestamps do not increase monotonically with
// release (observed 2026-08-27 — the x-axis came out shuffled). Same helper
// shape as OSReleaseTrend's semverKey.
function semverKey(v) {
  const p = String(v || '').replace(/^v/, '').split('.').map((n) => parseInt(n, 10) || 0);
  return p[0] * 1e6 + (p[1] || 0) * 1e3 + (p[2] || 0);
}

export default function RatingTrend({ history }) {
  const [view, setView] = useState('direction');
  // Default to provider grouping so the chart is readable; users can switch to
  // per-model — identical behaviour to PerModelTrend above it.
  const [groupBy, setGroupBy] = useState('provider');
  const [selectedProviders, setSelectedProviders] = useState(() => new Set());
  const [selectedModels, setSelectedModels] = useState(() => new Set());
  const isProviderVisible = (p) => selectedProviders.size === 0 || selectedProviders.has(p);
  const isModelVisible = (m) => selectedModels.size === 0 || selectedModels.has(m);
  const toggleProvider = (p) => setSelectedProviders((prev) => {
    const next = new Set(prev);
    if (next.has(p)) { next.delete(p); } else { next.add(p); }
    return next;
  });
  const toggleModel = (m) => setSelectedModels((prev) => {
    const next = new Set(prev);
    if (next.has(m)) { next.delete(m); } else { next.add(m); }
    return next;
  });

  const points = useMemo(() => {
    return (history || [])
      .filter((h) => h && h.ratings)
      .map((h) => ({
        version: h.version,
        timestamp: h.timestamp,
        directionIndex: h.ratings.direction_index || null,
        byTier: h.ratings.direction_by_tier || {},
        models: h.ratings.models || {},
        anchor: h.ratings.anchor_version,
        panel: h.ratings.panel_version,
        trials: h.ratings.trials,
      }))
      .sort((a, b) => semverKey(a.version) - semverKey(b.version));
  }, [history]);

  // EVERY model that has ever been rated — no cap and no coverage filter.
  // Earlier revisions filtered here, which structurally hid the current fleet
  // (today's models only exist in the last few baselines). Legibility is now
  // handled the way the rest of the page handles it: provider grouping by
  // default, with chip filters to drill in.
  const modelNames = useMemo(() => {
    const seen = new Set();
    points.forEach((p) => Object.keys(p.models).forEach((m) => seen.add(m)));
    return Array.from(seen).sort();
  }, [points]);

  const providers = useMemo(
    () => PROVIDER_ORDER.filter((p) => modelNames.some((m) => getProvider(m) === p)),
    [modelNames],
  );
  const modelColors = useMemo(() => assignModelColors(modelNames), [modelNames]);

  const chartData = useMemo(
    () => points.map((p) => {
      const row = { version: p.version, directionIndex: p.directionIndex };
      modelNames.forEach((m) => { row[m] = p.models[m] ?? null; });
      return row;
    }),
    [points, modelNames],
  );

  // Provider view: one line per provider = mean of that provider's rated models
  // in that release (same construction as PerModelTrend's providerChartData).
  const providerChartData = useMemo(
    () => chartData.map((pt) => {
      const row = { version: pt.version, directionIndex: pt.directionIndex };
      providers.forEach((p) => {
        const vals = modelNames
          .filter((m) => getProvider(m) === p)
          .map((m) => pt[m])
          .filter((v) => v != null);
        row[p] = vals.length ? Math.round(vals.reduce((a, b) => a + b, 0) / vals.length) : null;
      });
      return row;
    }),
    [chartData, providers, modelNames],
  );

  const byProvider = groupBy === 'provider';
  const modelData = byProvider ? providerChartData : chartData;
  const seriesKeys = byProvider
    ? providers.filter(isProviderVisible)
    : modelNames.filter(isModelVisible);
  const seriesColor = (k) => (byProvider ? PROVIDER_COLOR[k] : (modelColors.get(k) || '#999'));
  const seriesLabel = (k) => (byProvider ? (PROVIDER_LABEL[k] || k) : formatModelName(k));

  if (points.length < 2) {
    return (
      <div className={styles.chartContainer}>
        <h3>Rating trend over releases</h3>
        <p className={styles.chartNote}>
          Not enough published rating history yet ({points.length} release
          {points.length === 1 ? '' : 's'}). Each release adds a point once its linking run
          publishes a stamped rating snapshot.
        </p>
      </div>
    );
  }

  const latest = points[points.length - 1];
  const first = points[0];
  // Points that actually carry a direction index. Historical releases carry the
  // model half only (the index is a stamped release-time measurement and is
  // never backfilled), so this is usually empty until the first linking run.
  const dirPoints = points.filter((p) => p.directionIndex != null);
  const firstDir = dirPoints[0];
  const lastDir = dirPoints[dirPoints.length - 1];
  const delta = dirPoints.length >= 2 ? lastDir.directionIndex - firstDir.directionIndex : null;

  return (
    <div className={styles.chartContainer}>
      <h3>Rating trend over releases</h3>
      <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '0.75rem' }}>
        <button
          type="button"
          onClick={() => setView('direction')}
          className={view === "direction" ? styles.tierButtonActive : styles.tierButton}
        >
          Language direction
        </button>
        <button
          type="button"
          onClick={() => setView('models')}
          className={view === "models" ? styles.tierButtonActive : styles.tierButton}
        >
          Model capability
        </button>
      </div>

      {view === 'direction' ? (
        <>
          <p className={styles.chartNote}>
            Mean fitted difficulty of the fixed direction panel, with the bridge models held
            constant. <strong>Lower is better</strong> — it means the same models find AILANG
            easier. Axis is inverted so improvement reads upward.
            {delta !== null && (
              <> Since {firstDir.version}: <strong>{delta < 0 ? '↓' : '↑'} {Math.abs(delta).toFixed(1)} ELO</strong>
                {delta < 0 ? ' (easier — improving)' : ' (harder)'}.</>
            )}
          </p>
          {dirPoints.length === 0 && (
            <p className={styles.chartNote}>
              <strong>No release has published a direction index yet</strong>, so there is nothing
              to plot here. It is produced by the per-release linking run and is deliberately
              never backfilled — inventing one for a past release would be fabrication. The
              <em> Model capability</em> tab has data now.
            </p>
          )}
          {dirPoints.length > 0 && (
          <ResponsiveContainer width="100%" height={320}>
            <LineChart data={chartData} margin={{ top: 8, right: 24, bottom: 8, left: 8 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="version" />
              <YAxis reversed domain={['auto', 'auto']} label={{ value: 'Panel difficulty (ELO, lower = easier)', angle: -90, position: 'insideLeft', style: { fontSize: 11 } }} />
              <Tooltip formatter={(v) => (v == null ? '—' : Number(v).toFixed(1))} />
              <Legend />
              <Line type="monotone" dataKey="directionIndex" name="Language-direction index" stroke="#16a34a" strokeWidth={2} connectNulls={false} />
            </LineChart>
          </ResponsiveContainer>
          )}
        </>
      ) : (
        <>
          <p className={styles.chartNote}>
            Anchored model capability. Comparable across releases because the anchor panel pins
            the scale ({latest.anchor || 'anchor'}); an unanchored fit is only comparable within
            one run.
          </p>
          <div className={styles.languageToggle} title="Group the lines by provider or show every model">
            <button
              className={byProvider ? styles.toggleActive : styles.toggleInactive}
              onClick={() => setGroupBy('provider')}
            >
              Provider
            </button>
            <button
              className={!byProvider ? styles.toggleActive : styles.toggleInactive}
              onClick={() => setGroupBy('model')}
            >
              Model
            </button>
          </div>
          <ResponsiveContainer width="100%" height={340}>
            <LineChart data={modelData} margin={{ top: 8, right: 24, bottom: 8, left: 8 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="version" />
              <YAxis domain={['auto', 'auto']} label={{ value: 'Model ELO', angle: -90, position: 'insideLeft', style: { fontSize: 11 } }} />
              <Tooltip formatter={(v, n) => [v == null ? '—' : Number(v).toFixed(0), seriesLabel(n)]} />
              {seriesKeys.map((k) => (
                <Line key={k} type="monotone" dataKey={k} name={k} stroke={seriesColor(k)} strokeWidth={2} dot={!byProvider} connectNulls={false} />
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
              {providers.map((p) => (
                <button
                  key={p}
                  type="button"
                  className={`${styles.legendChip} ${isProviderVisible(p) ? '' : styles.legendChipHidden}`}
                  onClick={() => toggleProvider(p)}
                >
                  <span className={styles.legendChipDot} style={{ backgroundColor: PROVIDER_COLOR[p] }} />
                  {PROVIDER_LABEL[p]}
                </button>
              ))}
              {selectedProviders.size > 0 && (
                <button type="button" className={styles.legendChip} onClick={() => setSelectedProviders(new Set())} title="Reset to show all providers">
                  Reset
                </button>
              )}
            </div>
          ) : (
            <div className={styles.chipLegend}>
              <p className={styles.chipLegendHint}>
                {selectedModels.size === 0
                  ? `All ${modelNames.length} rated models — click any to focus.`
                  : `Showing ${selectedModels.size} of ${modelNames.length} — click more to add, or click again to remove.`}
              </p>
              {modelNames.map((m) => (
                <button
                  key={m}
                  type="button"
                  className={`${styles.legendChip} ${isModelVisible(m) ? '' : styles.legendChipHidden}`}
                  onClick={() => toggleModel(m)}
                >
                  <span className={styles.legendChipDot} style={{ backgroundColor: modelColors.get(m) || '#999' }} />
                  {formatModelName(m)}
                </button>
              ))}
              {selectedModels.size > 0 && (
                <button type="button" className={styles.legendChip} onClick={() => setSelectedModels(new Set())} title="Reset to show all models">
                  Reset
                </button>
              )}
            </div>
          )}
        </>
      )}

      <p className={styles.chartNote} style={{ fontSize: '0.8rem', opacity: 0.8 }}>
        {latest.panel && <>Panel {latest.panel}. </>}
        {latest.trials ? <>{latest.trials} trials in the latest release. </> : null}
        Releases without a published rating snapshot are skipped, never interpolated.
      </p>
    </div>
  );
}
