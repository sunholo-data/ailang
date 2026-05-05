import React from 'react';
import {
  Scatter,
  XAxis,
  YAxis,
  ZAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Line,
  ComposedChart,
} from 'recharts';
import styles from './styles.module.css';

// Same formatter every other v0.15.0 dashboard component uses (b5d9866e).
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

// Harness colour (matches the ai-coding-lang-bench convention used by the
// daemon dashboards). standard-only API models (no agent_cli) get a neutral grey.
const HARNESS_COLOR = {
  claude:   '#EF4444', // red
  gemini:   '#3B82F6', // blue
  codex:    '#10B981', // green
  opencode: '#F59E0B', // orange
  pi:       '#8B5CF6', // purple
  api:      '#6B7280', // grey (standard-only / direct API)
};

function harnessFor(modelMeta) {
  return modelMeta?.agent_cli || 'api';
}

/**
 * Compute the Pareto frontier for cost-vs-time minimisation. A point is on
 * the frontier if no other point has both lower cost AND lower-or-equal time
 * (and is not the same point). After sorting by ascending cost, we keep
 * points whose time is strictly less than the running min — classic
 * monotone-staircase frontier.
 */
function paretoFrontier(points) {
  const sorted = [...points].sort((a, b) => a.x - b.x || a.y - b.y);
  const frontier = [];
  let bestY = Infinity;
  for (const p of sorted) {
    if (p.y < bestY) {
      frontier.push(p);
      bestY = p.y;
    }
  }
  return frontier;
}

/** Find the dominator for an off-frontier point (any frontier point with
 *  lower-or-equal cost AND lower time, preferring the cheapest such). */
function dominatorFor(point, frontier) {
  const candidates = frontier.filter(
    (f) => f.name !== point.name && f.x <= point.x && f.y <= point.y && (f.x < point.x || f.y < point.y),
  );
  if (candidates.length === 0) return null;
  candidates.sort((a, b) => a.x - b.x || a.y - b.y);
  return candidates[0];
}

/**
 * CostSpeedFrontier — Pareto scatter chart of $/success vs sec/success.
 *
 * Inputs (per model from latest.json):
 *   x = efficiency.p90_cost_per_success     (USD)
 *   y = efficiency.median_time_to_success_ms / 1000  (seconds)
 *   marker size = aggregates.finalSuccess (0..1)
 *   colour = agent_cli harness (claude/gemini/codex/opencode/pi/api)
 *
 * The frontier line connects models that aren't dominated — these are your
 * efficient choices. Off-frontier models pay either more dollars or more
 * seconds (or both) for the same success rate as a frontier model.
 */
export default function CostSpeedFrontier({ models }) {
  if (!models || Object.keys(models).length === 0) {
    return (
      <div className={styles.chartContainer}>
        <p>No model data available for cost/speed frontier.</p>
      </div>
    );
  }

  const points = Object.entries(models)
    .map(([name, m]) => {
      const eff = m?.efficiency;
      if (!eff) return null;
      const cost = eff.p90_cost_per_success;
      const ttsMs = eff.median_time_to_success_ms;
      const success = m?.aggregates?.finalSuccess ?? 0;
      if (typeof cost !== 'number' || cost <= 0) return null;
      if (typeof ttsMs !== 'number' || ttsMs <= 0) return null;
      const harness = harnessFor(m);
      return {
        name,
        label: formatModelName(name),
        x: cost,
        y: ttsMs / 1000,
        successRate: success,
        // Marker size — recharts ZAxis maps to area; pick a sane domain
        // (success rates 0..1 → marker areas 80..400 px²).
        z: 80 + success * 320,
        harness,
        color: HARNESS_COLOR[harness] || HARNESS_COLOR.api,
      };
    })
    .filter(Boolean);

  if (points.length === 0) {
    return (
      <div className={styles.chartContainer}>
        <h3>Cost vs Speed Frontier</h3>
        <p className={styles.sectionSubtitle}>
          No efficiency data yet. M3 emits the per-model efficiency block; M5
          will populate it after running with the new measurement paths.
        </p>
      </div>
    );
  }

  const frontier = paretoFrontier(points);
  const frontierNames = new Set(frontier.map((p) => p.name));
  const dominated = points
    .filter((p) => !frontierNames.has(p.name))
    .map((p) => ({ point: p, dominator: dominatorFor(p, frontier) }))
    .filter(({ dominator }) => dominator);

  // Group points by harness for separate Scatter series — recharts colours
  // each Scatter as one colour, so we render one Scatter per harness.
  const byHarness = {};
  for (const p of points) {
    if (!byHarness[p.harness]) byHarness[p.harness] = [];
    byHarness[p.harness].push(p);
  }

  // Frontier line — recharts needs a sorted dataset whose x/y match the
  // scatter axes. We sort by ascending x (cost) and use a Line in a
  // ComposedChart-style layering trick: actually ScatterChart supports a
  // Line child via syncing dataset, but the simplest robust approach is
  // ComposedChart with two series: line for frontier, scatter for points.
  const frontierSorted = [...frontier].sort((a, b) => a.x - b.x);

  // Custom tooltip — show all dimensions including success rate.
  const renderTooltip = ({ active, payload }) => {
    if (!active || !payload || !payload.length) return null;
    const p = payload[0]?.payload;
    if (!p) return null;
    return (
      <div className={styles.chartTooltip}>
        <p className={styles.tooltipLabel}>{p.label}</p>
        <p className={styles.tooltipValue}>
          <span className={styles.tooltipDot} style={{ backgroundColor: p.color }} />
          Harness: {p.harness}
        </p>
        <p className={styles.tooltipValue}>Cost / success: ${p.x.toFixed(4)}</p>
        <p className={styles.tooltipValue}>Time / success: {p.y.toFixed(1)} s</p>
        <p className={styles.tooltipValue}>Success rate: {(p.successRate * 100).toFixed(1)}%</p>
        {frontierNames.has(p.name) && (
          <p className={styles.tooltipValue} style={{ marginTop: 6, fontWeight: 600, color: '#10B981' }}>
            On Pareto frontier
          </p>
        )}
      </div>
    );
  };

  return (
    <div className={styles.chartContainer}>
      <div className={styles.chartTitle}>Cost vs Speed Frontier</div>
      <p className={styles.sectionSubtitle}>
        Each dot is a model. <strong>X = $/success (log)</strong>,{' '}
        <strong>Y = seconds/success (log)</strong>, marker size = success rate,
        colour = harness. The dashed line connects the Pareto-efficient picks —
        any model off the line costs more dollars or seconds (or both) than a
        frontier model with comparable success.
      </p>
      <ResponsiveContainer width="100%" height={460}>
        <ComposedChart margin={{ top: 20, right: 30, bottom: 50, left: 60 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            type="number"
            dataKey="x"
            name="Cost per success"
            scale="log"
            domain={['auto', 'auto']}
            tickFormatter={(v) => `$${Number(v).toFixed(v < 0.1 ? 3 : 2)}`}
            label={{ value: '$ / success (log)', position: 'insideBottom', offset: -10 }}
            stroke="var(--ifm-color-emphasis-600)"
          />
          <YAxis
            type="number"
            dataKey="y"
            name="Seconds to success"
            scale="log"
            domain={['auto', 'auto']}
            tickFormatter={(v) => `${Number(v).toFixed(v < 1 ? 2 : 0)}s`}
            label={{ value: 'sec / success (log)', angle: -90, position: 'insideLeft' }}
            stroke="var(--ifm-color-emphasis-600)"
          />
          <ZAxis type="number" dataKey="z" range={[80, 400]} />
          <Tooltip content={renderTooltip} cursor={{ strokeDasharray: '3 3' }} />
          <Legend />
          {/* Frontier line — drawn first so scatter dots overlay. Plotted via
              a Line series whose data is a separate sorted slice. */}
          <Line
            type="monotone"
            data={frontierSorted}
            dataKey="y"
            name="Pareto frontier"
            stroke="#10B981"
            strokeWidth={2}
            strokeDasharray="6 4"
            dot={false}
            isAnimationActive={false}
            legendType="line"
          />
          {Object.entries(byHarness).map(([harness, pts]) => (
            <Scatter
              key={harness}
              name={harness === 'api' ? 'API (direct)' : harness}
              data={pts}
              fill={HARNESS_COLOR[harness] || HARNESS_COLOR.api}
              shape="circle"
            />
          ))}
        </ComposedChart>
      </ResponsiveContainer>
      <div style={{
        marginTop: '1rem',
        padding: '1rem',
        background: 'var(--ifm-color-emphasis-100)',
        borderLeft: '4px solid #10B981',
        borderRadius: 4,
        fontSize: '0.85rem',
        lineHeight: 1.5,
      }}>
        <strong>Frontier:</strong>{' '}
        {frontier.length > 0
          ? frontier.map((f) => f.label).join(' · ')
          : '(no points)'}
        .
        {dominated.length > 0 && (
          <div style={{ marginTop: 8 }}>
            <strong>Dominated models (off the frontier):</strong>
            <ul style={{ margin: '4px 0', paddingLeft: 20 }}>
              {dominated.map(({ point, dominator }) => (
                <li key={point.name}>
                  {point.label} — dominated by {dominator.label}{' '}
                  (cheaper or faster at comparable success)
                </li>
              ))}
            </ul>
          </div>
        )}
        {dominated.length === 0 && (
          <div style={{ marginTop: 8 }}>
            All models are on the frontier — no clear loser at the current data resolution.
          </div>
        )}
      </div>
    </div>
  );
}
