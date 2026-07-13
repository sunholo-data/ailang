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
  LabelList,
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

const HARNESS_COLOR = {
  claude:   '#EF4444',
  gemini:   '#3B82F6',
  codex:    '#10B981',
  opencode: '#F59E0B',
  pi:       '#8B5CF6',
  api:      '#6B7280',
};

function harnessFor(modelMeta) {
  return modelMeta?.agent_cli || 'api';
}

// 2-sig-fig wall-clock display so sub-second standard-eval values don't
// collapse to "0s" and look like missing data on the speed axis.
function formatSeconds2sf(s) {
  if (s <= 0) return '0';
  if (s >= 10) return s.toFixed(0);
  if (s >= 1) return s.toFixed(1);
  return s.toFixed(2);
}

/**
 * shortLabel — compact display name for inline scatter labels.
 * Goes for ~10–14 chars so labels don't overlap dots much.
 *   claude-opus-4-7         → Opus 4.7
 *   claude-sonnet-4-6       → Sonnet 4.6
 *   gpt5-5                  → GPT-5.5
 *   gpt5-4-mini             → GPT-5.4 mini
 *   gemini-3-1-pro          → Gem 3.1 Pro
 *   gemini-3-flash          → Gem 3 Flash
 *   opencode-or-glm-5       → GLM 5 (OC)
 *   opencode-or-minimax-m2-7→ MiniMax (OC)
 *   opencode-sonnet-4-6     → Sonnet 4.6 (OC)
 *   or-glm-5                → GLM 5
 */
function shortLabel(name) {
  if (name.startsWith('opencode-or-')) {
    const rest = name.slice('opencode-or-'.length);
    return shortFamily(rest) + ' (OC)';
  }
  if (name.startsWith('opencode-')) {
    const rest = name.slice('opencode-'.length);
    return shortFamily(rest) + ' (OC)';
  }
  if (name.startsWith('or-')) {
    const rest = name.slice('or-'.length);
    return shortFamily(rest);
  }
  if (name.startsWith('pi-')) {
    const rest = name.slice('pi-'.length);
    return shortFamily(rest) + ' (Pi)';
  }
  return shortFamily(name);
}

function shortFamily(s) {
  // claude-opus-4-7 → Opus 4.7
  if (s.startsWith('claude-')) {
    return s.slice('claude-'.length).replace(/-/g, ' ').replace(/(\d) (\d)/g, '$1.$2').replace(/^./, c => c.toUpperCase());
  }
  // gpt5-5 → GPT-5.5; gpt5-4-mini → GPT-5.4 mini
  if (s.startsWith('gpt5')) {
    return s.replace(/^gpt5/, 'GPT-5').replace(/-(\d)/g, '.$1').replace(/-/g, ' ');
  }
  // gemini-3-1-pro → Gem 3.1 Pro; gemini-3-flash → Gem 3 Flash
  if (s.startsWith('gemini-')) {
    return s.replace('gemini-', 'Gem ').replace(/-(\d)/g, '.$1').replace(/-/g, ' ').replace(/\b(\w)/g, (m, c, i) => i === 0 ? m : c.toUpperCase());
  }
  // glm-5 → GLM 5; glm-4-7-flash → GLM 4.7 Flash
  if (s.startsWith('glm-')) {
    return s.replace('glm-', 'GLM ').replace(/-(\d)/g, '.$1').replace(/-/g, ' ').replace(/\b(\w)/g, (m, c, i) => i <= 4 ? m : c.toUpperCase());
  }
  // minimax-m2-7 → MiniMax M2.7
  if (s.startsWith('minimax-')) {
    return s.replace('minimax-', 'MiniMax ').replace(/-(\d)/g, '.$1');
  }
  if (s.startsWith('kimi-')) {
    return s.replace('kimi-', 'Kimi ').replace(/-(\d)/g, '.$1');
  }
  if (s.startsWith('deepseek-')) {
    return s.replace('deepseek-', 'DeepSeek ').replace(/-(\d)/g, '.$1');
  }
  if (s.startsWith('gemma4-')) {
    return s.replace('gemma4-', 'Gemma 4 ').replace(/-(\d)/g, '.$1');
  }
  if (s.startsWith('qwen3-')) {
    return s.replace('qwen3-', 'Qwen3 ').replace(/-(\d)/g, '.$1');
  }
  return s.replace(/-/g, ' ');
}

/**
 * Maximise-quality, minimise-x Pareto frontier.
 * A point is on the frontier if no other point has lower-or-equal x AND
 * higher quality (strictly). Walk in ascending-x order, keeping running max-y.
 */
function paretoFrontier(points) {
  const sorted = [...points].sort((a, b) => a.x - b.x);
  const frontier = [];
  let maxY = -Infinity;
  for (const p of sorted) {
    if (p.y > maxY) {
      frontier.push(p);
      maxY = p.y;
    }
  }
  return frontier;
}

/**
 * QualityScatter — Industry-standard "score vs cost" or "score vs speed" plot.
 *
 * Props:
 *   models    — { [name]: { aggregates, efficiency, agent_cli, ... } }
 *   xMetric   — 'cost' | 'speed'
 *   minRuns   — filter models with fewer total runs (default 10)
 */
export default function QualityScatter({ models, xMetric = 'cost', mode = 'standard', minRuns = 10, coverage, ratings }) {
  const isCost = xMetric === 'cost';
  const isAgent = mode === 'agent';

  // Quality axis = per-model AILANG ELO for this mode. ELO spreads the strong models
  // that a raw pass rate saturates into one corner. Prefer the AILANG-specific ELO;
  // fall back to the combined rating.
  const eloOf = {};
  {
    const r = (ratings && ratings[mode]) || {};
    const block = (r.byLang && r.byLang.ailang) || r;
    (block.models || []).forEach((m) => { if (m && m.id != null) eloOf[m.id] = m.elo; });
  }

  const points = [];
  for (const [name, stats] of Object.entries(models || {})) {
    const agg = stats.aggregates || {};
    const as = stats.agentStats || null;
    // Per-mode speed (standard 0-shot is ~ms; agent multi-turn is ~seconds) — never blend.
    const eff = (isAgent ? stats.efficiencyAgent : stats.efficiencyStandard) || {};

    let passRate;
    let totalRuns;
    let x = 0;
    if (isAgent) {
      // AGENT mode: only agent fields; skip models that didn't run agent.
      if (!as || !as.runs) continue;
      totalRuns = as.runs;
      passRate = as.successRate ?? 0;
      if (passRate <= 0) continue;
      if (isCost) {
        const totalCost = (as.avgCost || 0) * totalRuns;
        const successes = passRate * totalRuns;
        x = successes > 0 ? totalCost / successes : 0;
      } else {
        x = (eff.median_time_to_success_ms || 0) / 1000;
      }
    } else {
      // STANDARD mode: only standard fields; skip models that didn't run standard.
      // No fallback to agent — that was the standard/agent mixing bug.
      passRate = agg.finalSuccess ?? 0;
      totalRuns = agg.totalRuns || stats.totalRuns || 0;
      if (passRate <= 0) continue;
      if (isCost) {
        const totalCost = agg.totalCostUSD || 0;
        const successes = passRate * totalRuns;
        x = successes > 0 ? totalCost / successes : 0;
      } else {
        x = (eff.median_time_to_success_ms || 0) / 1000;
      }
    }
    if (totalRuns < minRuns || passRate <= 0 || x <= 0) continue;
    const elo = eloOf[name];
    if (elo == null) continue; // no ELO rating for this model in this mode — omit

    const harness = harnessFor(stats);
    points.push({
      name,
      shortName: formatModelName(name),
      labelName: shortLabel(name),  // compact inline label
      x,
      y: elo,                  // quality = AILANG ELO
      passPct: passRate * 100, // kept for the tooltip
      runs: totalRuns,
      harness,
      color: HARNESS_COLOR[harness] || HARNESS_COLOR.api,
      // M-EVAL-VALIDITY-DISCIPLINE (W2): under-covered models can't define the frontier.
      provisional: coverage ? coverage.isProvisional(name) : false,
    });
  }

  if (points.length === 0) {
    return (
      <div className={styles.chartContainer} style={{ padding: 16 }}>
        <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>
          No model data available yet (need ≥{minRuns} runs per model + {isCost ? 'cost' : 'speed'} data).
        </p>
      </div>
    );
  }

  // Pareto frontier: maximise pass-rate while minimising x.
  // Carry full payload through so the tooltip shows model name when the
  // line vertex is the closest hover target.
  const frontier = paretoFrontier(points.filter(p => !p.provisional));
  const frontierLine = frontier.map(p => ({ ...p }));

  // Label only the value-frontier models inline — labeling every dot overlaps badly
  // once the strong models cluster. Everything else keeps its dot + hover tooltip.
  const frontierNames = new Set(frontier.map(p => p.name));
  for (const p of points) p.plotLabel = frontierNames.has(p.name) ? p.labelName : '';

  // Bucket by harness for separate Scatter series (so Legend works per-harness)
  const byHarness = {};
  for (const p of points) {
    if (!byHarness[p.harness]) byHarness[p.harness] = [];
    byHarness[p.harness].push(p);
  }

  // Compute domain — log-scale with 10% padding
  const xs = points.map(p => p.x);
  const xMin = Math.min(...xs) * 0.7;
  const xMax = Math.max(...xs) * 1.3;

  // Y (ELO) domain — round out to the nearest 50 with a little headroom so points
  // don't sit on the axis edge.
  const ys = points.map(p => p.y);
  const yMin = Math.floor(Math.min(...ys) / 50) * 50 - 25;
  const yMax = Math.ceil(Math.max(...ys) / 50) * 50 + 25;

  const xLabel = isCost ? 'Cost per success ($, log scale)' : 'Median time to solution (sec, log scale)';

  const xFormatter = isCost
    ? (v) => `$${v.toFixed(3)}`
    : (v) => `${formatSeconds2sf(v)}s`;

  return (
    <div className={styles.chartContainer}>
      <ResponsiveContainer width="100%" height={420}>
        <ComposedChart margin={{ top: 24, right: 30, left: 30, bottom: 60 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            type="number"
            dataKey="x"
            scale="log"
            domain={[xMin, xMax]}
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            label={{ value: xLabel, position: 'insideBottom', offset: -10, fill: 'var(--ifm-color-emphasis-700)' }}
            tickFormatter={xFormatter}
          />
          <YAxis
            type="number"
            dataKey="y"
            domain={[yMin, yMax]}
            allowDecimals={false}
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            label={{ value: 'AILANG ELO', angle: -90, position: 'insideLeft', fill: 'var(--ifm-color-emphasis-700)' }}
            tickFormatter={(v) => Math.round(v)}
          />
          <ZAxis type="number" range={[60, 60]} />
          <Tooltip
            cursor={{ strokeDasharray: '3 3' }}
            content={({ active, payload }) => {
              if (!active || !payload || payload.length === 0) return null;
              const p = payload[0].payload;
              return (
                <div className={styles.chartTooltip}>
                  <div style={{ fontWeight: 600 }}>{p.shortName}</div>
                  <div>AILANG ELO: {Math.round(p.y)}</div>
                  <div>Pass rate: {p.passPct.toFixed(1)}%</div>
                  <div>{isCost ? `Cost / success: $${p.x.toFixed(4)}` : `Time / success: ${formatSeconds2sf(p.x)}s`}</div>
                  <div style={{ fontSize: '0.85em', color: 'var(--ifm-color-emphasis-600)' }}>
                    Harness: {p.harness} · {p.runs} runs
                  </div>
                </div>
              );
            }}
          />
          <Legend wrapperStyle={{ paddingTop: 16 }} />

          {/* Pareto frontier line connects best-in-class points */}
          <Line
            type="linear"
            data={frontierLine}
            dataKey="y"
            stroke="#10B981"
            strokeDasharray="5 3"
            strokeWidth={2}
            dot={false}
            activeDot={false}
            legendType="none"
            name="Pareto frontier"
            isAnimationActive={false}
          />

          {/* One Scatter series per harness for color-coded legend.
              Inline LabelList gives every dot a readable model tag — this is
              the LMArena-style identification users expect on score-vs-cost
              plots, and keeps the tooltip as the deeper-info layer. */}
          {Object.entries(byHarness).map(([h, pts]) => (
            <Scatter
              key={h}
              name={h}
              data={pts}
              fill={HARNESS_COLOR[h] || HARNESS_COLOR.api}
              shape="circle"
            >
              <LabelList
                dataKey="plotLabel"
                position="right"
                offset={8}
                style={{
                  fontSize: 11,
                  fill: 'var(--ifm-color-emphasis-800)',
                  fontWeight: 600,
                  pointerEvents: 'none',
                }}
              />
            </Scatter>
          ))}
        </ComposedChart>
      </ResponsiveContainer>

      <div style={{ marginTop: 8, fontSize: '0.85em', color: 'var(--ifm-color-emphasis-700)', padding: '0 16px' }}>
        <strong>Read NW-corner-up:</strong> high ELO + {isCost ? 'low cost' : 'low time'} = better.
        The green dashed line is the Pareto frontier — its models are non-dominated and are the ones
        labeled; hover any dot for its name, ELO, and {isCost ? 'cost' : 'time'}. Color codes harness.
        {isCost && ' Lower-left corner = cheap but weaker; upper-right = expensive flagships.'}
        {!isCost && ' Faster solutions are leftward; slower (multi-turn agent loops) rightward.'}
      </div>
    </div>
  );
}
