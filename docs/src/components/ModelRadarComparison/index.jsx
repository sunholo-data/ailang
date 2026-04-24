import React, { useState } from 'react';
import { Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import styles from './styles.module.css';

function formatModelName(name) {
  // Check most specific patterns first for proper display names
  if (name.includes('claude-opus-4-7')) return 'Claude Opus 4.7';
  if (name.includes('claude-opus-4-6')) return 'Claude Opus 4.6';
  if (name.includes('claude-opus-4-5')) return 'Claude Opus 4.5';
  if (name.includes('claude-sonnet-4-6')) return 'Claude Sonnet 4.6';
  if (name.includes('claude-sonnet-4-5')) return 'Claude Sonnet 4.5';
  if (name.includes('claude-haiku-4-5')) return 'Claude Haiku 4.5';
  if (name.includes('gpt5-4-mini') || name.includes('gpt5-4mini')) return 'GPT-5.4 Mini';
  if (name.includes('gpt5-4')) return 'GPT-5.4';
  if (name.includes('gpt5-2-codex')) return 'GPT-5.2 Codex';
  if (name.includes('gpt5-2-instant')) return 'GPT-5.2 Instant';
  if (name.includes('gpt5-2')) return 'GPT-5.2';
  if (name.includes('gpt5-1-instant')) return 'GPT-5.1 Instant';
  if (name.includes('gpt5-1-codex')) return 'GPT-5.1 Codex';
  if (name.includes('gpt5-1')) return 'GPT-5.1';
  if (name.includes('gpt-5-mini') || name.includes('gpt5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5') || name.includes('gpt5')) return 'GPT-5';
  if (name.includes('gemini-3-1-pro')) return 'Gemini 3.1 Pro';
  if (name.includes('gemini-3-flash')) return 'Gemini 3.0 Flash';
  if (name.includes('gemini-3-pro') || name.includes('gemini-3.0-pro')) return 'Gemini 3.0 Pro';
  if (name.includes('gemini-2-5-flash') || name.includes('gemini-2.5-flash')) return 'Gemini 2.5 Flash';
  if (name.includes('gemini-2-5-pro') || name.includes('gemini-2.5-pro')) return 'Gemini 2.5 Pro';
  // Fallback: capitalize first letter of each word
  return name.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
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
    fetch('/benchmarks/latest.json')
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

    // Cost per 1000 successful runs per language (in dollars)
    // Estimate language cost proportionally based on token usage
    const ailangData = modelData.languages.ailang || {};
    const pythonData = modelData.languages.python || {};

    const totalModelTokens = modelData.aggregates?.totalTokens || 1;
    const ailangTotalTokens = (ailangData.avgTokens || 0) * (ailangData.totalRuns || 0);
    const pythonTotalTokens = (pythonData.avgTokens || 0) * (pythonData.totalRuns || 0);

    const ailangCostProportion = ailangTotalTokens / totalModelTokens;
    const pythonCostProportion = pythonTotalTokens / totalModelTokens;

    const ailangEstimatedCost = (modelData.aggregates?.totalCostUSD || 0) * ailangCostProportion;
    const pythonEstimatedCost = (modelData.aggregates?.totalCostUSD || 0) * pythonCostProportion;

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
      // Delta metrics (AILANG relative to Python baseline)
      'Success Gap': pythonSuccess - ailangSuccess, // Positive = Python is better
      'Token Delta': ((pythonTokens - ailangTokens) / pythonTokens) * 100, // Positive = AILANG uses fewer tokens
      // Cost efficiency per language (cost per 1000 successful runs in dollars)
      'AILANG Cost ($)': ailangCostPer1000,
      'Python Cost ($)': pythonCostPer1000,
      // Reliability per language (infra uptime, not code quality)
      'AILANG Uptime': ailangUptime,
      'Python Uptime': pythonUptime,
    };
  });

  // Custom tooltip formatter to round values
  const formatTooltip = (value, name) => {
    if (typeof value !== 'number') return value;
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
            <strong>Red = success gap, Blue = token delta.</strong> Success gap shows Python success - AILANG success (positive = AILANG behind). Token delta shows token savings (positive = AILANG uses fewer tokens than Python). Goal: minimize red, maximize blue. The <strong style={{color: '#10B981'}}>green dashed line</strong> at 0% marks parity with Python.
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
        <div className={styles.chartCard}>
          <h3>Cost per 1000 Successes</h3>
          <p className={styles.subtitle}>Dollar cost for 1000 successful benchmarks</p>
          <ResponsiveContainer width="100%" height={350}>
            <RadarChart data={radarData}>
              <PolarGrid />
              <PolarAngleAxis dataKey="model" />
              <PolarRadiusAxis angle={90} domain={[0, 2]} />
              <Tooltip formatter={formatTooltip} />
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
            <strong>Purple = AILANG cost, Green = Python cost (in dollars).</strong> Shows cost for 1000 successful benchmark runs per language. Lower = better value. Example: $0.50 means 1000 successful benchmarks cost fifty cents. Ideal for budget-conscious model selection.
          </div>
        </div>

      </div>
    </div>
  );
}
