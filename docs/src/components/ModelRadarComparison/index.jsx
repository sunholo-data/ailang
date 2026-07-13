import React, { useState } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import { Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import styles from './styles.module.css';

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

/**
 * ModelRadarComparison - Three focused radar plots with models as spokes
 *
 * Split into 3 charts for clarity:
 * 1. AILANG Performance (0-shot vs with-repair)
 * 2. Python Performance (0-shot vs with-repair)
 * 3. Efficiency Comparison (cost vs tokens)
 */
export default function ModelRadarComparison() {
  // Load benchmark data
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  React.useEffect(() => {
    benchmarkFetch('latest.json')
      .then(res => res.json())
      .then(json => setData(json))
      .catch(err => setError(err.message));
  }, []);

  if (error) return <div className={styles.error}>Error loading data: {error}</div>;
  if (!data) return <div className={styles.loading}>Loading benchmark data...</div>;

  // Normalize values to 0-100 scale
  const normalize = (value, min, max) => {
    if (max === min) return 50;
    return ((value - min) / (max - min)) * 100;
  };

  // Calculate min/max for normalization
  const models = Object.keys(data.models).sort((a, b) => a.localeCompare(b));

  // Transform data: each model becomes a spoke (axis)
  // Filter to only include models that have language data, cap at 12 by run count for readability
  const modelsWithLanguages = models
    .filter(model => {
      const modelData = data.models[model];
      return modelData?.languages?.ailang && modelData?.languages?.python && modelData?.aggregates;
    })
    .sort((a, b) => {
      const runsA = data.models[a]?.aggregates?.totalRuns || 0;
      const runsB = data.models[b]?.aggregates?.totalRuns || 0;
      return runsB - runsA;
    })
    .slice(0, 12)
    .sort((a, b) => a.localeCompare(b)); // restore alphabetical order for consistent spoke layout

  const radarData = modelsWithLanguages.map(model => {
    const modelData = data.models[model];
    const ailangSuccess = (modelData.languages?.ailang?.successRate || 0) * 100;
    const pythonSuccess = (modelData.languages?.python?.successRate || 0) * 100;

    // Calculate zero-shot rates (approximation based on repair impact)
    const repairBoost = modelData.aggregates?.finalSuccess / (modelData.aggregates?.zeroShotSuccess || 1);
    const ailangZeroShot = ailangSuccess / (repairBoost || 1);
    const pythonZeroShot = pythonSuccess / (repairBoost || 1);

    // AILANG tokens vs Python tokens (per language)
    const ailangTokens = modelData.languages?.ailang?.avgTokens || 0;
    const pythonTokens = modelData.languages?.python?.avgTokens || 0;

    // Cost per 1000 successful runs per language (in dollars).
    //
    // Prior implementation prorated cost by OUTPUT-token share — but
    // aggregates.totalTokens is INPUT+OUTPUT+cache, while languages.{lang}.avgTokens
    // is output-only. That skewed the proportion massively for Opus-class models
    // (huge input-cache usage drives totalTokens up, shrinking the output-only
    // numerator's share to ~1%) vs Haiku-class models (almost all output, so the
    // share goes to ~35%). Result: Haiku appeared MORE expensive per success
    // than Opus, which is the opposite of reality.
    //
    // Correct approach: use the explicit per-language avgCost when the schema
    // has it (agent-only models / recent baselines have it), else fall back to
    // (overall_cost / overall_runs) × lang_runs — assuming equal cost per run
    // across languages, which is accurate for standard eval (input prompt sizes
    // are similar) and a reasonable approximation for agent eval too.
    const ailangData = modelData.languages.ailang || {};
    const pythonData = modelData.languages.python || {};

    const totalModelRuns =
      modelData.aggregates?.totalRuns ||
      modelData.totalRuns ||
      ((ailangData.totalRuns || 0) + (pythonData.totalRuns || 0)) ||
      0;
    const totalModelCost = modelData.aggregates?.totalCostUSD || 0;
    const costPerRun = totalModelRuns > 0 ? totalModelCost / totalModelRuns : 0;

    // Prefer explicit per-language avgCost when present.
    const ailangCostPerRun = ailangData.avgCost ?? costPerRun;
    const pythonCostPerRun = pythonData.avgCost ?? costPerRun;

    const ailangEstimatedCost = ailangCostPerRun * (ailangData.totalRuns || 0);
    const pythonEstimatedCost = pythonCostPerRun * (pythonData.totalRuns || 0);

    const ailangSuccessCount = (ailangData.successRate || 0) * (ailangData.totalRuns || 0);
    const pythonSuccessCount = (pythonData.successRate || 0) * (pythonData.totalRuns || 0);

    // Cost per success, then multiply by 1000 to get cost per 1000 successes (in dollars)
    const ailangCostPer1000 = ailangSuccessCount > 0 ? (ailangEstimatedCost / ailangSuccessCount) * 1000 : 0;
    const pythonCostPer1000 = pythonSuccessCount > 0 ? (pythonEstimatedCost / pythonSuccessCount) * 1000 : 0;

    // Reliability (M-DASH-V2): per-language uptime = 1 - api_error_rate.
    // api errors = provider quota / auth / 5xx, NOT code quality.
    const rel = modelData.reliability || {};
    const ailangRuns = ailangData.totalRuns || 0;
    const pythonRuns = pythonData.totalRuns || 0;
    const ailangUptime = ailangRuns > 0
      ? (1 - ((rel.ailangApiError || 0) / ailangRuns)) * 100
      : 100;
    const pythonUptime = pythonRuns > 0
      ? (1 - ((rel.pythonApiError || 0) / pythonRuns)) * 100
      : 100;

    return {
      model: formatModelName(model),
      // AILANG metrics
      'Zero-Shot': ailangZeroShot,
      'With Repair': ailangSuccess,
      // Python metrics
      'Python 0-Shot': pythonZeroShot,
      'Python w/Repair': pythonSuccess,
      // Delta metrics (AILANG relative to Python baseline). These can run far
      // outside the [-50,50] axis (e.g. a very verbose AILANG model burning 4x
      // Python's tokens = -300%+). Plot a CLAMPED value so the polygon stays
      // readable, but keep the raw value (_raw*) for the tooltip so the true
      // magnitude isn't silently hidden at the axis edge.
      'Success Gap': Math.max(-50, Math.min(50, pythonSuccess - ailangSuccess)), // + = Python ahead
      'Token Delta': Math.max(-50, Math.min(50, pythonTokens > 0 ? ((pythonTokens - ailangTokens) / pythonTokens) * 100 : 0)), // + = AILANG uses fewer
      _rawSuccessGap: pythonSuccess - ailangSuccess,
      _rawTokenDelta: pythonTokens > 0 ? ((pythonTokens - ailangTokens) / pythonTokens) * 100 : 0,
      // Cost efficiency per language (cost per 1000 successful runs in dollars)
      'AILANG Cost ($)': ailangCostPer1000,
      'Python Cost ($)': pythonCostPer1000,
      // Reliability per language (infra uptime, not code quality)
      'AILANG Uptime': ailangUptime,
      'Python Uptime': pythonUptime,
    };
  });

  // Custom tooltip formatter to round values
  const formatTooltip = (value, name, item) => {
    if (typeof value !== 'number') return value;
    // Delta radars clamp to the [-50,50] axis; surface the TRUE magnitude here
    // so an off-scale value (e.g. Fable's -323% token delta) isn't hidden at
    // the axis edge reading as "no delta".
    const payload = item && item.payload;
    if (payload && name === 'Success Gap (%)' && typeof payload._rawSuccessGap === 'number') {
      const raw = payload._rawSuccessGap;
      return `${raw.toFixed(1)}%${Math.abs(raw) > 50 ? ' (off-scale, clamped)' : ''}`;
    }
    if (payload && name === 'Token Delta (%)' && typeof payload._rawTokenDelta === 'number') {
      const raw = payload._rawTokenDelta;
      return `${raw.toFixed(1)}%${Math.abs(raw) > 50 ? ' (off-scale, clamped)' : ''}`;
    }
    // Use 2 decimal places for money (cost per success)
    if (name && name.includes('Cost')) {
      return value.toFixed(2);
    }
    // Use 1 decimal place for percentages
    return value.toFixed(1);
  };

  return (
    <div className={styles.container}>
      <div className={styles.chartsGrid}>

        {/* Chart 1: AILANG Performance */}
        <div className={styles.chartCard}>
          <h3>AILANG Performance</h3>
          <p className={styles.subtitle}>Success rate across models</p>
          <ResponsiveContainer width="100%" height={350}>
            <RadarChart data={radarData}>
              <PolarGrid />
              <PolarAngleAxis dataKey="model" />
              <PolarRadiusAxis angle={90} domain={[0, 100]} />
              <Tooltip formatter={formatTooltip} />
              <Legend />
              <Radar
                name="Zero-Shot"
                dataKey="Zero-Shot"
                stroke="#A78BFA"
                fill="#A78BFA"
                fillOpacity={0.3}
                strokeWidth={2}
              />
              <Radar
                name="With Repair"
                dataKey="With Repair"
                stroke="#8B5CF6"
                fill="#8B5CF6"
                fillOpacity={0.4}
                strokeWidth={3}
              />
            </RadarChart>
          </ResponsiveContainer>
          <div className={styles.chartNote}>
            <strong>Purple = AILANG success rates.</strong> Outer line (darker) shows performance with M-EVAL-LOOP self-repair. Inner line (lighter) shows zero-shot performance.
          </div>
        </div>

        {/* Chart 2: AILANG vs Python Gap */}
        <div className={styles.chartCard}>
          <h3>AILANG vs Python Gap</h3>
          <p className={styles.subtitle}>How close is AILANG to Python baseline?</p>
          <ResponsiveContainer width="100%" height={350}>
            <RadarChart data={radarData.map(d => ({ ...d, 'Parity (0%)': 0 }))}>
              <PolarGrid
                stroke="var(--ifm-color-emphasis-300)"
                radialLines={true}
                gridType="polygon"
              />
              <PolarAngleAxis
                dataKey="model"
                tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 11 }}
              />
              <PolarRadiusAxis
                angle={90}
                domain={[-50, 50]}
                tick={{
                  fill: 'var(--ifm-color-emphasis-700)',
                  fontSize: 10
                }}
                tickFormatter={(value) => {
                  if (value === 0) return '0% (PARITY)';
                  return `${value > 0 ? '+' : ''}${value}%`;
                }}
                ticks={[-50, -25, 0, 25, 50]}
                axisLine={{ stroke: 'var(--ifm-color-emphasis-700)', strokeWidth: 2 }}
              />
              <Tooltip formatter={formatTooltip} />
              <Legend />
              {/* Reference line at 0% - shows where AILANG matches Python */}
              <Radar
                name="Parity Line (0%)"
                dataKey="Parity (0%)"
                stroke="#10B981"
                fill="none"
                strokeWidth={3}
                strokeDasharray="8 4"
                dot={false}
              />
              <Radar
                name="Success Gap (%)"
                dataKey="Success Gap"
                stroke="#EF4444"
                fill="#EF4444"
                fillOpacity={0.3}
                strokeWidth={3}
              />
              <Radar
                name="Token Delta (%)"
                dataKey="Token Delta"
                stroke="#3B82F6"
                fill="#3B82F6"
                fillOpacity={0.2}
                strokeWidth={2}
                strokeDasharray="5 5"
              />
            </RadarChart>
          </ResponsiveContainer>
          <div className={styles.chartNote}>
            <strong>Red = success gap, Blue = token delta.</strong> Success gap shows Python success - AILANG success (positive = AILANG behind). Token delta shows token savings (positive = AILANG uses fewer tokens than Python). Goal: minimize red, maximize blue. The <strong style={{color: '#10B981'}}>green dashed line</strong> at 0% marks parity with Python. Values beyond ±50% are clamped to the axis edge — <strong>hover any point for the true figure</strong> (e.g. a very verbose model can burn 300%+ more AILANG tokens).
          </div>
        </div>

        {/* Chart 4: API Reliability (M-DASH-V2) */}
        <div className={styles.chartCard}>
          <h3>API Reliability</h3>
          <p className={styles.subtitle}>Infra uptime per language (quota/auth/5xx, not code quality)</p>
          <ResponsiveContainer width="100%" height={350}>
            <RadarChart data={radarData}>
              <PolarGrid />
              <PolarAngleAxis dataKey="model" />
              <PolarRadiusAxis angle={90} domain={[0, 100]} />
              <Tooltip formatter={formatTooltip} />
              <Legend />
              <Radar
                name="AILANG Uptime (%)"
                dataKey="AILANG Uptime"
                stroke="#8B5CF6"
                fill="#8B5CF6"
                fillOpacity={0.3}
                strokeWidth={3}
              />
              <Radar
                name="Python Uptime (%)"
                dataKey="Python Uptime"
                stroke="#10B981"
                fill="#10B981"
                fillOpacity={0.25}
                strokeWidth={2}
                strokeDasharray="5 5"
              />
            </RadarChart>
          </ResponsiveContainer>
          <div className={styles.chartNote}>
            <strong>100% = no API errors.</strong> A pulled-in spoke means the provider
            returned quota/auth/5xx errors — NOT that the model produced bad code. Useful
            for spotting provider-side issues (e.g. Gemini free-tier quota) that would
            otherwise masquerade as code quality failures.
          </div>
        </div>

        {/* Chart 3: Cost Efficiency */}
        {(() => {
          // Cost-per-1000 has wide dynamic range — direct API models are
          // ~$0.50–$5 while opencode-wrapped agent models can be $50–$300
          // (multi-turn loops resend full context each turn). One outlier
          // collapses every other spoke to ~zero, so we cap the *display*
          // value at the median × 5 and surface the real number in the
          // tooltip + an outlier list under the chart.
          const allCosts = radarData.flatMap(d => [d['AILANG Cost ($)'], d['Python Cost ($)']]).filter(v => v > 0).sort((a, b) => a - b);
          const median = allCosts.length ? allCosts[Math.floor(allCosts.length / 2)] : 0;
          const capValue = Math.max(median * 5, 5); // never below $5 cap
          const cappedRadarData = radarData.map(d => ({
            ...d,
            'AILANG Cost ($)': Math.min(d['AILANG Cost ($)'] || 0, capValue),
            'Python Cost ($)': Math.min(d['Python Cost ($)'] || 0, capValue),
            // keep originals for the tooltip
            _ailangCostReal: d['AILANG Cost ($)'],
            _pythonCostReal: d['Python Cost ($)'],
          }));
          // Outliers are points whose REAL value exceeds the display cap.
          // Use the capped dataset's _*_Real fields so the outlier list shows
          // the actual numbers (not the clipped ones).
          const outliers = cappedRadarData.filter(d =>
            (d._ailangCostReal > capValue) || (d._pythonCostReal > capValue)
          );
          const formatCostTooltip = (value, name, props) => {
            if (typeof value !== 'number') return value;
            const real = name === 'AILANG Cost ($)'
              ? props?.payload?._ailangCostReal
              : props?.payload?._pythonCostReal;
            const realStr = (typeof real === 'number') ? real.toFixed(2) : value.toFixed(2);
            return value < real ? `${realStr} (clipped at ${capValue.toFixed(0)})` : realStr;
          };
          return (
            <div className={styles.chartCard}>
              <h3>Cost per 1000 Successes</h3>
              <p className={styles.subtitle}>Dollar cost for 1000 successful benchmarks (display capped at ${capValue.toFixed(0)} for readability)</p>
              <ResponsiveContainer width="100%" height={350}>
                <RadarChart data={cappedRadarData}>
                  <PolarGrid />
                  <PolarAngleAxis dataKey="model" />
                  <PolarRadiusAxis angle={90} domain={[0, capValue]} />
                  <Tooltip formatter={formatCostTooltip} />
                  <Legend />
                  <Radar
                    name="AILANG Cost ($)"
                    dataKey="AILANG Cost ($)"
                    stroke="#8B5CF6"
                    fill="#8B5CF6"
                    fillOpacity={0.3}
                    strokeWidth={3}
                  />
                  <Radar
                    name="Python Cost ($)"
                    dataKey="Python Cost ($)"
                    stroke="#10B981"
                    fill="#10B981"
                    fillOpacity={0.3}
                    strokeWidth={2}
                    strokeDasharray="5 5"
                  />
                </RadarChart>
              </ResponsiveContainer>
              <div className={styles.chartNote}>
                <strong>Purple = AILANG cost, Green = Python cost (in dollars).</strong> Cost for 1000 successful benchmark runs per language; lower = better value. Display clipped at <strong>${capValue.toFixed(0)}</strong> (5× median) so non-outlier models stay readable.
                {outliers.length > 0 && (
                  <div style={{ marginTop: 8 }}>
                    <strong>Outliers (real values):</strong>
                    <ul style={{ margin: '4px 0', paddingLeft: 20 }}>
                      {outliers.map(o => (
                        <li key={o.model}>
                          {o.model}: AILANG ${(o['_ailangCostReal'] || 0).toFixed(2)} · Python ${(o['_pythonCostReal'] || 0).toFixed(2)}
                          {(o.model.includes('opencode') || o.model.includes('agent')) && ' — agent harness amplifies cost (multi-turn re-sends full context)'}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          );
        })()}

      </div>
    </div>
  );
}
