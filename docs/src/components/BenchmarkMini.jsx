import React, { useState, useEffect } from 'react';
import { TrendingUp, CheckCircle, Zap, Bot } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import Link from '@docusaurus/Link';

export default function BenchmarkMini() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/ailang/benchmarks/latest.json')
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        setData(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <section style={styles.container}>
        <div style={styles.loading}>Loading benchmarks...</div>
      </section>
    );
  }

  if (!data || !data.aggregates) {
    return null;
  }

  const { aggregates, version, totalRuns, history, languages } = data;
  const zeroShot = (aggregates.zeroShotSuccess * 100).toFixed(0);
  const final = (aggregates.finalSuccess * 100).toFixed(0);
  const agent = (aggregates.agentSuccessRate * 100).toFixed(0);

  // Prepare chart data from history
  const chartData = prepareChartData(history, languages);

  return (
    <section style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>Real Benchmark Results</h2>
        <p style={styles.subtitle}>
          {totalRuns} runs across 46 benchmarks • {version}
        </p>
      </div>

      <div style={styles.grid}>
        <MetricCard
          icon={<CheckCircle size={24} />}
          value={`${zeroShot}%`}
          label="Zero-Shot"
          sublabel="Works on first try"
          color="#2c7a7b"
        />
        <MetricCard
          icon={<Zap size={24} />}
          value={`${final}%`}
          label="Final Success"
          sublabel="After self-repair"
          color="#e73c17"
        />
        <MetricCard
          icon={<Bot size={24} />}
          value={`${agent}%`}
          label="Agent Mode"
          sublabel="Multi-turn completion"
          color="#6b46c1"
        />
      </div>

      {/* Trend Chart */}
      {chartData && chartData.length > 1 && (
        <div style={styles.chartSection}>
          <h3 style={styles.chartTitle}>Success Rate Over Time</h3>
          <p style={styles.chartSubtitle}>
            Tracking AI code generation success across AILANG versions
          </p>
          <div style={styles.chartWrapper}>
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 60 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(128,128,128,0.2)" />
                <XAxis
                  dataKey="version"
                  stroke="var(--ifm-font-color-secondary)"
                  tick={{ fill: 'var(--ifm-font-color-secondary)', fontSize: 11 }}
                  angle={-45}
                  textAnchor="end"
                  height={60}
                />
                <YAxis
                  stroke="var(--ifm-font-color-secondary)"
                  tick={{ fill: 'var(--ifm-font-color-secondary)', fontSize: 12 }}
                  domain={[0, 100]}
                  tickFormatter={(v) => `${v}%`}
                />
                <Tooltip content={<CustomTooltip />} />
                <Legend wrapperStyle={{ paddingTop: '10px' }} iconType="circle" />
                <Line
                  type="monotone"
                  dataKey="AILANG"
                  stroke="#e73c17"
                  strokeWidth={3}
                  dot={{ r: 4, fill: '#e73c17' }}
                  activeDot={{ r: 6 }}
                />
                {chartData.some(d => d.Python > 0) && (
                  <Line
                    type="monotone"
                    dataKey="Python"
                    stroke="#ffa726"
                    strokeWidth={3}
                    dot={{ r: 4, fill: '#ffa726' }}
                    activeDot={{ r: 6 }}
                  />
                )}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      <div style={styles.cta}>
        <Link to="/docs/benchmarks/performance" style={styles.link}>
          View Full Dashboard
          <TrendingUp size={16} style={{ marginLeft: '0.5rem' }} />
        </Link>
      </div>
    </section>
  );
}

function prepareChartData(history, languages) {
  if (!history || history.length === 0) return null;

  const validHistory = history.filter(h => {
    const date = new Date(h.timestamp);
    return date.getFullYear() > 2000;
  });

  const sortedHistory = [...validHistory].sort((a, b) => {
    return new Date(a.timestamp) - new Date(b.timestamp);
  });

  return sortedHistory.map((baseline, index) => {
    const isLatest = index === sortedHistory.length - 1;
    let ailangRate = 0;
    let pythonRate = 0;

    if (baseline.languageStats) {
      ailangRate = (baseline.languageStats.ailang?.success_rate || 0) * 100;
      pythonRate = (baseline.languageStats.python?.success_rate || 0) * 100;
    } else if (isLatest && languages) {
      ailangRate = (languages.ailang?.success_rate || 0) * 100;
      pythonRate = (languages.python?.success_rate || 0) * 100;
    } else {
      const rate = (baseline.successRate || baseline.aggregates?.finalSuccess || 0) * 100;
      ailangRate = rate;
    }

    return {
      version: formatVersion(baseline.version),
      'AILANG': parseFloat(ailangRate.toFixed(1)),
      'Python': parseFloat(pythonRate.toFixed(1)),
      date: baseline.timestamp ? new Date(baseline.timestamp).toLocaleDateString() : ''
    };
  });
}

function formatVersion(version) {
  if (!version) return '?';
  version = version.replace(/^v/, '');
  const parts = version.split('-');
  if (parts.length >= 3) return `v${parts[0]}`;
  return `v${version}`;
}

function CustomTooltip({ active, payload, label }) {
  if (active && payload && payload.length) {
    const data = payload[0].payload;
    return (
      <div style={styles.tooltip}>
        <div style={styles.tooltipLabel}>{label}</div>
        {data.date && <div style={styles.tooltipDate}>{data.date}</div>}
        <div style={styles.tooltipValue}>
          <span style={{ ...styles.tooltipDot, background: '#e73c17' }} />
          AILANG: {data['AILANG']}%
        </div>
        {data['Python'] > 0 && (
          <div style={styles.tooltipValue}>
            <span style={{ ...styles.tooltipDot, background: '#ffa726' }} />
            Python: {data['Python']}%
          </div>
        )}
      </div>
    );
  }
  return null;
}

function MetricCard({ icon, value, label, sublabel, color }) {
  return (
    <div style={styles.card}>
      <div style={{ ...styles.iconWrapper, background: `${color}15`, color }}>
        {icon}
      </div>
      <div style={styles.value}>{value}</div>
      <div style={styles.label}>{label}</div>
      <div style={styles.sublabel}>{sublabel}</div>
    </div>
  );
}

const styles = {
  container: {
    padding: '4rem 2rem',
    background: 'var(--ifm-background-surface-color)',
  },
  header: {
    textAlign: 'center',
    marginBottom: '2.5rem',
  },
  title: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 800,
    fontSize: 'clamp(1.75rem, 4vw, 2.5rem)',
    marginBottom: '0.5rem',
  },
  subtitle: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '1rem',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
    gap: '1.5rem',
    maxWidth: '700px',
    margin: '0 auto 3rem',
  },
  card: {
    background: 'var(--ifm-background-color)',
    borderRadius: '16px',
    padding: '1.5rem',
    textAlign: 'center',
    border: '1px solid rgba(128, 128, 128, 0.1)',
  },
  iconWrapper: {
    width: '48px',
    height: '48px',
    borderRadius: '12px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '0 auto 1rem',
  },
  value: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 800,
    fontSize: '2.25rem',
    lineHeight: 1,
    marginBottom: '0.5rem',
  },
  label: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 600,
    fontSize: '0.95rem',
    marginBottom: '0.25rem',
  },
  sublabel: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.8rem',
  },
  chartSection: {
    maxWidth: '800px',
    margin: '0 auto 2rem',
  },
  chartTitle: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 700,
    fontSize: '1.25rem',
    textAlign: 'center',
    marginBottom: '0.5rem',
  },
  chartSubtitle: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.9rem',
    textAlign: 'center',
    marginBottom: '1.5rem',
  },
  chartWrapper: {
    background: 'var(--ifm-background-color)',
    borderRadius: '16px',
    padding: '1.5rem',
    border: '1px solid rgba(128, 128, 128, 0.1)',
  },
  cta: {
    textAlign: 'center',
    marginTop: '1rem',
  },
  link: {
    display: 'inline-flex',
    alignItems: 'center',
    color: 'var(--ifm-color-primary)',
    fontWeight: 600,
    textDecoration: 'none',
    padding: '0.75rem 1.5rem',
    borderRadius: '8px',
    border: '1px solid var(--ifm-color-primary)',
    transition: 'all 0.2s ease',
  },
  loading: {
    textAlign: 'center',
    padding: '3rem',
    color: 'var(--ifm-font-color-secondary)',
  },
  tooltip: {
    background: 'var(--ifm-background-color)',
    border: '1px solid rgba(128, 128, 128, 0.2)',
    borderRadius: '8px',
    padding: '0.75rem 1rem',
    boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
  },
  tooltipLabel: {
    fontWeight: 700,
    marginBottom: '0.25rem',
  },
  tooltipDate: {
    fontSize: '0.8rem',
    color: 'var(--ifm-font-color-secondary)',
    marginBottom: '0.5rem',
  },
  tooltipValue: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    fontSize: '0.9rem',
  },
  tooltipDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    display: 'inline-block',
  },
};
