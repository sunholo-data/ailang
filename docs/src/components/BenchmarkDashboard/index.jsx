import React, { useState, useEffect } from 'react';
import { TrendingUp, TrendingDown, Activity, DollarSign, Zap, CheckCircle, Lock, Target, Bot } from 'lucide-react';
import ModelChart from './ModelChart';
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
          Try running: <code>make benchmark-dashboard</code>
        </p>
      </div>
    );
  }

  if (!data || !data.aggregates) {
    return (
      <div className={styles.error}>
        <p>⚠️ No benchmark data available</p>
        <p className={styles.errorHint}>
          Run <code>make benchmark-dashboard</code> to generate metrics.
        </p>
      </div>
    );
  }

  const { aggregates, models, benchmarks, version, totalRuns, history } = data;

  // Calculate trend (compare to previous version if available)
  let trend = null;
  if (history && history.length > 1) {
    const current = aggregates.finalSuccess || aggregates.zeroShotSuccess || 0;
    const previous = history[history.length - 2]?.aggregates?.finalSuccess || 0;
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
            value={`${(aggregates.finalSuccess * 100).toFixed(1)}%`}
            subtitle={`${(aggregates.zeroShotSuccess * 100).toFixed(1)}% on first try`}
            trend={trend}
            large
          />
          <MetricCard
            icon={<Activity />}
            title="Total Runs"
            value={totalRuns}
            subtitle={`Across ${Object.keys(models || {}).length} models`}
          />
          <MetricCard
            icon={<Zap />}
            title="Avg Tokens"
            value={Math.round(aggregates.totalTokens / totalRuns)}
            subtitle="Per successful run"
          />
          <MetricCard
            icon={<DollarSign />}
            title="Total Cost"
            value={`$${aggregates.totalCostUSD.toFixed(2)}`}
            subtitle={`$${(aggregates.totalCostUSD / totalRuns).toFixed(4)}/run`}
          />
        </div>
      </div>

      {/* Model Performance Chart */}
      {models && Object.keys(models).length > 0 && (
        <div className={styles.section}>
          <h3>Model Performance Comparison</h3>
          <ModelChart models={models} />
        </div>
      )}

      {/* Success Trend */}
      {history && history.length > 1 && (
        <div className={styles.section}>
          <h3>Success Rate Over Time</h3>
          <SuccessTrend history={history} />
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
