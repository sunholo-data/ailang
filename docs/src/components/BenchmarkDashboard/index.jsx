import React, { useState, useEffect } from 'react';
import { TrendingUp, TrendingDown, Activity, Zap, CheckCircle, Lock, Target, Bot } from 'lucide-react';
import ModelChart from './ModelChart';
import ModelComparisonTable from './ModelComparisonTable';
import ModelTokenChart from './ModelTokenChart';
import LanguageChart from './LanguageChart';
import BenchmarkGallery from './BenchmarkGallery';
import SuccessTrend from './SuccessTrend';
import PerModelTrend from './PerModelTrend';
import ModelDeltaTrend from './ModelDeltaTrend';
import RadarCharts from './RadarCharts';
import AgentRadar from './AgentRadar';
import AxiomScorecard from './AxiomScorecard';
import styles from './styles.module.css';

// Tier order + labels for the M6 toggle. Core is the headline tier
// (primary metric), so it's the default selection when the tiers block
// is present in the dashboard JSON.
const TIER_ORDER = ['smoke', 'core', 'stretch', 'vision'];
const TIER_LABELS = {
  smoke: 'Smoke',
  core: 'Core',
  stretch: 'Stretch',
  vision: 'Vision',
};
const TIER_BLURBS = {
  smoke: 'Signal tier — fast sanity checks',
  core: 'Primary metric — expected AILANG floor',
  stretch: 'Headroom tier — hard contracts + polymorphism',
  vision: 'Aspirational — type-directed synthesis',
};

export default function BenchmarkDashboard() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedTier, setSelectedTier] = useState(null); // null = all tiers

  useEffect(() => {
    // Fetch benchmark data
    fetch('/benchmarks/latest.json')
      .then(res => {
        if (!res.ok) throw new Error('Failed to load benchmark data');
        return res.json();
      })
      .then(data => {
        setData(data);
        // Default to Core tier when the tiers block is present, so the
        // primary metric is the Core pass rate (M-EVAL-SUITE-PREP M6).
        if (data?.tiers?.core) {
          setSelectedTier('core');
        }
        setLoading(false);
      })
      .catch(err => {
        console.error('Error loading benchmarks:', err);
        setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className={styles.loading}>
        <Activity className={styles.spinner} />
        <p>Loading benchmark data...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.error}>
        <p>⚠️ Could not load benchmark data: {error}</p>
        <p className={styles.errorHint}>
          Try running: <code>ailang eval-report eval_results/baselines/VERSION VERSION --format=json</code>
        </p>
      </div>
    );
  }

  if (!data || !data.aggregates) {
    return (
      <div className={styles.error}>
        <p>⚠️ No benchmark data available</p>
        <p className={styles.errorHint}>
          Run <code>ailang eval-report eval_results/baselines/VERSION VERSION --format=json</code> to generate metrics.
        </p>
      </div>
    );
  }

  const { aggregates, models, benchmarks, version, totalRuns, history, languages, tiers } = data;

  // Use AILANG-specific metrics for the dashboard
  const ailangStats = languages?.ailang || aggregates;
  const pythonStats = languages?.python;
  const ailangRuns = ailangStats.total_runs || ailangStats.totalRuns || Math.floor(totalRuns / 2);
  const overallAilangSuccess = ailangStats.success_rate || ailangStats.finalSuccess || aggregates.finalSuccess;
  const overallPythonSuccess = pythonStats?.success_rate ?? 0;
  const ailangZeroShot = ailangStats.success_rate || ailangStats.zeroShotSuccess || aggregates.zeroShotSuccess;

  // Resolve the currently displayed tier (null = overall). Core is the
  // headline metric when the tiers block is present (M-EVAL-SUITE-PREP M6).
  const activeTier = selectedTier && tiers?.[selectedTier] ? tiers[selectedTier] : null;
  const ailangSuccess = activeTier ? activeTier.ailang_success_rate : overallAilangSuccess;
  const tierPythonSuccess = activeTier ? activeTier.python_success_rate : overallPythonSuccess;
  const tierAilangRuns = activeTier ? activeTier.ailang_runs : ailangRuns;
  const tierBenchCount = activeTier ? activeTier.benchmark_count : Object.keys(benchmarks || {}).length;

  // Filter benchmarks to the selected tier. The per-benchmark `tier`
  // field is written by ExportBenchmarkJSON from the YAML spec.
  const filteredBenchmarks = selectedTier && benchmarks
    ? Object.fromEntries(
        Object.entries(benchmarks).filter(([, b]) => b?.tier === selectedTier)
      )
    : benchmarks;

  // Calculate deltas vs Python baseline (tier-aware)
  const successDelta = (ailangSuccess - tierPythonSuccess) * 100;
  const tokenDelta = pythonStats ? ((ailangStats.avg_tokens - pythonStats.avg_tokens) / pythonStats.avg_tokens * 100) : 0;
  const tokenRatio = pythonStats ? (ailangStats.avg_tokens / pythonStats.avg_tokens) : 1;


  // Calculate trend (compare to previous version if available)
  let trend = null;
  if (history && history.length > 1) {
    const current = ailangSuccess;
    // Note: history[i].languages is a string (e.g., "ailang,python"), use languageStats for stats
    const previous = history[history.length - 2]?.languageStats?.ailang?.success_rate ||
                    history[history.length - 2]?.aggregates?.finalSuccess || 0;
    const diff = current - previous;
    if (Math.abs(diff) > 0.01) {
      trend = {
        direction: diff > 0 ? 'up' : 'down',
        value: Math.abs(diff * 100).toFixed(1)
      };
    }
  }

  const heroTitle = activeTier
    ? `${TIER_LABELS[selectedTier]} Pass Rate`
    : 'Success Rate';
  const heroSubtitle = activeTier
    ? `${successDelta >= 0 ? '+' : ''}${successDelta.toFixed(1)}% vs Python (${(tierPythonSuccess * 100).toFixed(1)}%)`
    : (pythonStats ? `${successDelta.toFixed(1)}% vs Python (${(overallPythonSuccess * 100).toFixed(1)}%)` : 'AILANG success rate');

  return (
    <div className={styles.dashboard}>
      {/* Tier Toggle (M6) */}
      {tiers && Object.keys(tiers).length > 0 && (
        <TierToggle
          tiers={tiers}
          selected={selectedTier}
          onSelect={setSelectedTier}
        />
      )}

      {/* Hero Metrics */}
      <div className={styles.heroSection}>
        <div className={styles.metricGrid}>
          <MetricCard
            icon={<CheckCircle />}
            title={heroTitle}
            value={`${(ailangSuccess * 100).toFixed(1)}%`}
            subtitle={heroSubtitle}
            trend={activeTier ? null : trend}
            large
          />
          <MetricCard
            icon={<Zap />}
            title="Output Tokens"
            value={Math.round(ailangStats.avg_tokens || ailangStats.avgTokens || (aggregates.totalTokens / totalRuns))}
            subtitle={pythonStats ? `${tokenRatio.toFixed(1)}x vs Python (${Math.round(pythonStats.avg_tokens)} tokens)` : 'Per AILANG run'}
            large
          />
          <MetricCard
            icon={<Activity />}
            title={activeTier ? `${TIER_LABELS[selectedTier]} Benchmarks` : 'Total Benchmarks'}
            value={activeTier ? tierBenchCount : ailangRuns}
            subtitle={activeTier ? `${tierAilangRuns} AILANG runs` : `Across ${Object.keys(models || {}).length} AI models`}
          />
          <MetricCard
            icon={<Target />}
            title="Zero-Shot Rate"
            value={`${((ailangStats.zero_shot_success || ailangZeroShot || 0) * 100).toFixed(1)}%`}
            subtitle="First-attempt success (no repair)"
          />
        </div>
      </div>

      {/* Per-Model Trend */}
      {history && history.length > 1 && history.some(h => h.modelStats) && (
        <div className={styles.section}>
          <h3>Success Rate by Model Over Time</h3>
          <p className={styles.sectionSubtitle}>
            Track how each AI model's performance evolves across AILANG versions
          </p>
          <PerModelTrend history={history} />
        </div>
      )}

      {/* Model Delta Trend */}
      {history && history.length > 1 && history.some(h => h.modelStats) && (
        <div className={styles.section}>
          <h3>AILANG vs Python Gap by Model</h3>
          <p className={styles.sectionSubtitle}>
            Positive values indicate AILANG outperforms Python for that model
          </p>
          <ModelDeltaTrend history={history} />
        </div>
      )}

      {/* Model Performance Chart */}
      {models && Object.keys(models).length > 0 && (
        <div className={styles.section}>
          <h3>Model Performance Comparison</h3>
          <ModelChart models={models} />
          <ModelComparisonTable models={models} />
        </div>
      )}

      {/* Language Comparison Chart */}
      {languages && Object.keys(languages).length > 1 && (
        <div className={styles.section}>
          <h3>AILANG vs Python Performance</h3>
          <p className={styles.sectionSubtitle}>
            Direct comparison of AI code generation success rates and efficiency
          </p>
          <LanguageChart languages={languages} />
        </div>
      )}

      {/* Model Token Usage Chart */}
      {models && Object.keys(models).length > 0 && (
        <div className={styles.section}>
          <h3>Token Usage & Cost by Model</h3>
          <p className={styles.sectionSubtitle}>
            Average output tokens and cost per benchmark run (excludes reasoning tokens for GPT-5)
          </p>
          <ModelTokenChart models={models} />
        </div>
      )}

      {/* Evaluation Modes Comparison */}
      <div className={styles.section}>
        <h3>Evaluation Approaches</h3>
        <p className={styles.sectionSubtitle}>
          Comparing 0-shot, self-repair, and multi-turn agent evaluation modes
        </p>
        <RadarCharts data={data} />
      </div>

      {/* Agent Performance Detail */}
      {aggregates.agentRuns > 0 && (
        <div className={styles.section}>
          <AgentRadar data={data} />
        </div>
      )}

      {/* Success Trend */}
      {history && history.length > 1 && (
        <div className={styles.section}>
          <h3>Success Rate Over Time</h3>
          <SuccessTrend history={history} languages={languages} />
        </div>
      )}

      {/* Benchmark Gallery (filtered by selected tier when active) */}
      {filteredBenchmarks && Object.keys(filteredBenchmarks).length > 0 && (
        <div className={styles.section}>
          <h3>
            Benchmark Results
            {activeTier && (
              <span className={styles.tierHeadline}>
                {' '}— showing {Object.keys(filteredBenchmarks).length} {TIER_LABELS[selectedTier]} benchmarks
              </span>
            )}
          </h3>
          <BenchmarkGallery benchmarks={filteredBenchmarks} />
        </div>
      )}

      {/* Axiom Compliance */}
      <div className={styles.section}>
        <h3>Design Axiom Compliance</h3>
        <p className={styles.sectionSubtitle}>
          Tracking AILANG's adherence to its 12 core design principles
        </p>
        <AxiomScorecard />
      </div>

      {/* Value Propositions */}
      <div className={styles.valueProps}>
        <ValueProp
          icon={<Lock size={32} />}
          title="Type Safety"
          description="Hindley-Milner inference catches errors before execution"
        />
        <ValueProp
          icon={<Zap size={32} />}
          title="Effect System"
          description="Explicit IO, FS, Net effects guide AI code generation"
        />
        <ValueProp
          icon={<Target size={32} />}
          title="Deterministic"
          description="Same input always produces same output"
        />
        <ValueProp
          icon={<Bot size={32} />}
          title="AI-Optimized"
          description="Designed for AI-assisted development"
        />
      </div>

      {/* CTA Section */}
      <div className={styles.ctaSection}>
        <h3>Try AILANG Today</h3>
        <p>Start building with AI-first functional programming</p>
        <div className={styles.ctaButtons}>
          <a href="/docs/guides/getting-started" className={styles.ctaButton}>
            Get Started
          </a>
          <a href="https://github.com/sunholo-data/ailang" className={styles.ctaButton + ' ' + styles.secondary}>
            View on GitHub
          </a>
        </div>
      </div>
    </div>
  );
}

// Sub-components

function MetricCard({ icon, title, value, subtitle, trend, large }) {
  return (
    <div className={`${styles.metricCard} ${large ? styles.metricCardLarge : ''}`}>
      <div className={styles.metricIcon}>{icon}</div>
      <div className={styles.metricContent}>
        <div className={styles.metricTitle}>{title}</div>
        <div className={styles.metricValue}>
          {value}
          {trend && (
            <span className={`${styles.trend} ${styles[trend.direction]}`}>
              {trend.direction === 'up' ? <TrendingUp size={20} /> : <TrendingDown size={20} />}
              {trend.value}%
            </span>
          )}
        </div>
        {subtitle && <div className={styles.metricSubtitle}>{subtitle}</div>}
      </div>
    </div>
  );
}

function ValueProp({ icon, title, description }) {
  return (
    <div className={styles.valueProp}>
      <div className={styles.valuePropIcon}>{icon}</div>
      <div className={styles.valuePropTitle}>{title}</div>
      <div className={styles.valuePropDescription}>{description}</div>
    </div>
  );
}

// TierToggle renders a chip row of tier filters. The selected tier
// drives the hero metric + benchmark gallery (M-EVAL-SUITE-PREP M6).
// An "All" chip resets to the overall view.
function TierToggle({ tiers, selected, onSelect }) {
  const label = selected && TIER_BLURBS[selected]
    ? TIER_BLURBS[selected]
    : 'Combined view across every tier';
  return (
    <div>
      <div className={styles.tierToggle}>
        <span className={styles.tierToggleLabel}>Tier:</span>
        <button
          type="button"
          className={`${styles.tierButton} ${selected === null ? styles.tierButtonActive : ''}`}
          onClick={() => onSelect(null)}
        >
          All
        </button>
        {TIER_ORDER.filter((t) => tiers[t]).map((t) => (
          <button
            key={t}
            type="button"
            className={`${styles.tierButton} ${selected === t ? styles.tierButtonActive : ''}`}
            onClick={() => onSelect(t)}
          >
            {TIER_LABELS[t]}
            <span className={styles.tierButtonCount}>({tiers[t].benchmark_count})</span>
          </button>
        ))}
      </div>
      <div className={styles.tierHeadline}>{label}</div>
    </div>
  );
}
