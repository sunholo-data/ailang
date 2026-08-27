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
      .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
  }, [history]);

  const modelNames = useMemo(() => {
    const counts = {};
    points.forEach((p) => Object.keys(p.models).forEach((m) => { counts[m] = (counts[m] || 0) + 1; }));
    // Only models present in a majority of points — a one-off model produces a
    // single dot and reads as a broken line.
    return Object.keys(counts)
      .filter((m) => counts[m] >= Math.max(2, Math.ceil(points.length / 2)))
      .sort();
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
  const delta = latest.directionIndex && first.directionIndex
    ? latest.directionIndex - first.directionIndex
    : null;

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
              <> Since {first.version}: <strong>{delta < 0 ? '↓' : '↑'} {Math.abs(delta).toFixed(1)} ELO</strong>
                {delta < 0 ? ' (easier — improving)' : ' (harder)'}.</>
            )}
          </p>
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
