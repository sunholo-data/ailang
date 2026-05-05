import React from 'react';
import {
  Radar,
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from 'recharts';
import styles from './styles.module.css';

// Mirror of the canonical formatModelName in BenchmarkExplorer / ModelRadarComparison
// (v0.15.0 hotfix b5d9866e). Kept identical so all radar/legend labels read the same.
function formatModelName(name) {
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

/**
 * SpeedRadar — Polar radar chart of median time-to-success per model.
 *
 * Inputs: `models` from latest.json. For each model we read
 * `efficiency.median_time_to_success_ms` (M3 output) and convert to seconds
 * for display. Lower is better — a tight inner ring means everyone is fast.
 *
 * Outlier handling mirrors the v0.15.0 cost-radar fix in
 * ModelRadarComparison/index.jsx (commit b5d9866e): wide dynamic range
 * (sub-second API models vs minute-scale agent harnesses) collapses every
 * other spoke. We clip the *display* value at 5× median (min cap 30 s) and
 * surface real values in the tooltip + an outlier list under the chart.
 */
export default function SpeedRadar({ models }) {
  if (!models || Object.keys(models).length === 0) {
    return (
      <div className={styles.chartContainer}>
        <p>No model data available for speed radar.</p>
      </div>
    );
  }

  // Filter to models with efficiency.median_time_to_success_ms > 0 AND
  // total runs ≥ 10 (avoid noise from one-off baselines).
  const eligible = Object.entries(models)
    .filter(([, m]) => {
      const tts = m?.efficiency?.median_time_to_success_ms;
      const runs = m?.totalRuns ?? m?.aggregates?.totalRuns ?? 0;
      return typeof tts === 'number' && tts > 0 && runs >= 10;
    })
    .map(([name, m]) => ({
      name,
      ttsSec: m.efficiency.median_time_to_success_ms / 1000,
    }))
    .sort((a, b) => a.name.localeCompare(b.name));

  if (eligible.length === 0) {
    return (
      <div className={styles.chartContainer}>
        <h3>Median Time to Success</h3>
        <p className={styles.sectionSubtitle}>
          No efficiency data yet. M3 emits this block; M5 will populate it once
          the new measurement paths run end-to-end.
        </p>
      </div>
    );
  }

  // Outlier clipping (b5d9866e pattern).
  const sortedTimes = eligible.map((e) => e.ttsSec).sort((a, b) => a - b);
  const median = sortedTimes[Math.floor(sortedTimes.length / 2)];
  const capValue = Math.max(median * 5, 30); // never below 30s cap

  const radarData = eligible.map((e) => ({
    model: formatModelName(e.name),
    'Time to Success (s)': Math.min(e.ttsSec, capValue),
    _ttsReal: e.ttsSec,
  }));

  const outliers = eligible.filter((e) => e.ttsSec > capValue);

  const formatTooltip = (value, name, props) => {
    if (typeof value !== 'number') return value;
    const real = props?.payload?._ttsReal;
    const realStr = (typeof real === 'number') ? real.toFixed(1) : value.toFixed(1);
    return value < (real ?? Infinity)
      ? `${realStr}s (clipped at ${capValue.toFixed(0)}s)`
      : `${realStr}s`;
  };

  return (
    <div className={styles.chartContainer} style={{ marginTop: '2rem' }}>
      <div className={styles.chartTitle}>Median Time to Success (lower = better)</div>
      <p className={styles.sectionSubtitle}>
        Wall-clock seconds from prompt start to first passing run, per model.
        Display capped at <strong>{capValue.toFixed(0)}s</strong> (5× median) so
        sub-second API models stay readable when an agent harness pulls the
        scale out to minutes.
      </p>
      <ResponsiveContainer width="100%" height={400}>
        <RadarChart data={radarData}>
          <PolarGrid />
          <PolarAngleAxis dataKey="model" tick={{ fontSize: 11 }} />
          <PolarRadiusAxis angle={90} domain={[0, capValue]} />
          <Tooltip formatter={formatTooltip} />
          <Legend />
          <Radar
            name="Time to Success (s)"
            dataKey="Time to Success (s)"
            stroke="#F59E0B"
            fill="#F59E0B"
            fillOpacity={0.3}
            strokeWidth={3}
          />
        </RadarChart>
      </ResponsiveContainer>
      <div style={{
        marginTop: '1rem',
        padding: '1rem',
        background: 'var(--ifm-color-emphasis-100)',
        borderLeft: '4px solid #F59E0B',
        borderRadius: 4,
        fontSize: '0.85rem',
        lineHeight: 1.5,
      }}>
        <strong>Inner ring = faster.</strong> A pulled-out spoke is a slow model,
        usually because it's an agent harness (multi-turn) or running on a slow
        provider tier. Direct-API models (claude-opus-4-7, gpt5-5, gemini-3-1-pro)
        typically render in &lt;1 s; agent-wrapped runs (opencode-*) sit in the
        30–60 s range.
        {outliers.length > 0 && (
          <div style={{ marginTop: 8 }}>
            <strong>Outliers (real values):</strong>
            <ul style={{ margin: '4px 0', paddingLeft: 20 }}>
              {outliers.map((o) => (
                <li key={o.name}>
                  {formatModelName(o.name)}: {o.ttsSec.toFixed(1)}s
                  {(o.name.includes('opencode') || o.name.includes('agent')) &&
                    ' — agent harness amplifies wall-clock (multi-turn loop)'}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
