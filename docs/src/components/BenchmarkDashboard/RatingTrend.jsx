import React, { useMemo, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import styles from './styles.module.css';

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
const MODEL_COLORS = ['#2563eb', '#16a34a', '#db2777', '#ea580c', '#7c3aed', '#0891b2', '#ca8a04', '#dc2626'];
// Release order must come from the VERSION, not the timestamp: baselines get
// re-banked and back-filled, so timestamps do not increase monotonically with
// release (observed 2026-08-27 — the x-axis came out shuffled). Same helper
// shape as OSReleaseTrend's semverKey.
const MAX_LINES = 8;
function semverKey(v) {
  const p = String(v || '').replace(/^v/, '').split('.').map((n) => parseInt(n, 10) || 0);
  return p[0] * 1e6 + (p[1] || 0) * 1e3 + (p[2] || 0);
}

export default function RatingTrend({ history }) {
  const [view, setView] = useState('direction');

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

  const modelNames = useMemo(() => {
    // Selection favours the CURRENT fleet. A pure "appears in >=50% of releases"
    // rule silently excluded every model we actually run today (they only exist
    // in the last few baselines) and drew a chart of retired models — reported
    // 2026-08-27. Rule now: models from the most recent release first, then the
    // best-covered historical models to fill the remaining slots.
    if (points.length === 0) return [];
    const counts = {};
    points.forEach((p) => Object.keys(p.models).forEach((m) => { counts[m] = (counts[m] || 0) + 1; }));
    const latest = points[points.length - 1];
    const current = Object.keys(latest.models || {}).sort(
      (a, b) => (latest.models[b] || 0) - (latest.models[a] || 0),
    );
    const historical = Object.keys(counts)
      .filter((m) => !current.includes(m) && counts[m] >= 2)
      .sort((a, b) => counts[b] - counts[a]);
    return [...current, ...historical].slice(0, MAX_LINES);
  }, [points]);

  const chartData = useMemo(
    () => points.map((p) => {
      const row = { version: p.version, directionIndex: p.directionIndex };
      modelNames.forEach((m) => { row[m] = p.models[m] ?? null; });
      return row;
    }),
    [points, modelNames],
  );

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
          <ResponsiveContainer width="100%" height={320}>
            <LineChart data={chartData} margin={{ top: 8, right: 24, bottom: 8, left: 8 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="version" />
              <YAxis domain={['auto', 'auto']} label={{ value: 'Model ELO', angle: -90, position: 'insideLeft', style: { fontSize: 11 } }} />
              <Tooltip formatter={(v) => (v == null ? '—' : Number(v).toFixed(1))} />
              <Legend />
              {modelNames.map((m, i) => (
                <Line key={m} type="monotone" dataKey={m} name={m} stroke={MODEL_COLORS[i % MODEL_COLORS.length]} strokeWidth={2} dot connectNulls={false} />
              ))}
            </LineChart>
          </ResponsiveContainer>
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
