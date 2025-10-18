import React, { useState, useEffect } from 'react';
import { TrendingUp, TrendingDown, Activity, DollarSign, Zap, CheckCircle, Lock, Target, Bot } from 'lucide-react';
import ModelChart from './ModelChart';
import ModelTokenChart from './ModelTokenChart';
import LanguageChart from './LanguageChart';
import BenchmarkGallery from './BenchmarkGallery';
import SuccessTrend from './SuccessTrend';
import styles from './styles.module.css';

export default function BenchmarkDashboard() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    // Fetch benchmark data
    fetch('/ailang/benchmarks/latest.json')
      .then(res => {
        if (!res.ok) throw new Error('Failed to load benchmark data');
        return res.json();
      })
      .then(data => {
        setData(data);
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

  const { aggregates, models, benchmarks, version, totalRuns, history, languages } = data;

  // Use AILANG-specific metrics for the dashboard
  const ailangStats = languages?.ailang || aggregates;
  const pythonStats = languages?.python;
  const ailangRuns = ailangStats.total_runs || ailangStats.totalRuns || Math.floor(totalRuns / 2);
  const ailangSuccess = ailangStats.success_rate || ailangStats.finalSuccess || aggregates.finalSuccess;
  const ailangZeroShot = ailangStats.success_rate || ailangStats.zeroShotSuccess || aggregates.zeroShotSuccess;

  // Calculate deltas vs Python baseline
  const successDelta = pythonStats ? ((ailangSuccess - pythonStats.success_rate) * 100) : 0;
  const tokenDelta = pythonStats ? ((ailangStats.avg_tokens - pythonStats.avg_tokens) / pythonStats.avg_tokens * 100) : 0;
  const tokenRatio = pythonStats ? (ailangStats.avg_tokens / pythonStats.avg_tokens) : 1;

  // Calculate trend (compare to previous version if available)
  let trend = null;
  if (history && history.length > 1) {
    const current = ailangSuccess;
    const previous = history[history.length - 2]?.languages?.ailang?.success_rate ||
                    history[history.length - 2]?.aggregates?.finalSuccess || 0;
    const diff = current - previous;
    if (Math.abs(diff) > 0.01) {
      trend = {
        direction: diff > 0 ? 'up' : 'down',
        value: Math.abs(diff * 100).toFixed(1)
      };
    }
  }

  return (
    <div className={styles.dashboard}>
      {/* Hero Metrics */}
      <div className={styles.heroSection}>
        <div className={styles.metricGrid}>
          <MetricCard
            icon={<CheckCircle />}
            title="Success Rate"
            value={`${(ailangSuccess * 100).toFixed(1)}%`}
            subtitle={pythonStats ? `${successDelta.toFixed(1)}% vs Python (${(pythonStats.success_rate * 100).toFixed(1)}%)` : 'AILANG success rate'}
            trend={trend}
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
            title="Total Benchmarks"
            value={ailangRuns}
            subtitle={`Across ${Object.keys(models || {}).length} AI models`}
          />
          <MetricCard
            icon={<DollarSign />}
            title="Cost Efficiency"
            value={pythonStats ? `${tokenRatio.toFixed(1)}x` : 'N/A'}
            subtitle={pythonStats ? `More expensive than Python` : 'Cost comparison'}
          />
        </div>
      </div>

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

      {/* Model Performance Chart */}
      {models && Object.keys(models).length > 0 && (
        <div className={styles.section}>
          <h3>Model Performance Comparison</h3>
          <ModelChart models={models} />
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

      {/* Success Trend */}
      {history && history.length > 1 && (
        <div className={styles.section}>
          <h3>Success Rate Over Time</h3>
          <SuccessTrend history={history} languages={languages} />
        </div>
      )}

      {/* Benchmark Gallery */}
      {benchmarks && Object.keys(benchmarks).length > 0 && (
        <div className={styles.section}>
          <h3>Benchmark Results</h3>
          <BenchmarkGallery benchmarks={benchmarks} />
        </div>
      )}

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
          <a href="/ailang/docs/guides/getting-started" className={styles.ctaButton}>
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
