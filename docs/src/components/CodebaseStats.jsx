import React, { useState, useEffect } from 'react';
import { Code, FileText, GitCommit, Users, Cpu, TrendingUp } from 'lucide-react';
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, BarChart, Bar, Legend
} from 'recharts';

export default function CodebaseStats() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/codebase_stats.json')
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
        <div style={styles.loading}>Loading codebase statistics...</div>
      </section>
    );
  }

  if (!data || !data.current) {
    return (
      <section style={styles.container}>
        <div style={styles.loading}>Statistics not available</div>
      </section>
    );
  }

  const { current, history } = data;
  const { lines, files, tokens, git } = current;

  // Prepare chart data from history
  const chartData = prepareChartData(history);
  const breakdownData = prepareBreakdownData(current);

  // Format large numbers
  const formatNumber = (n) => {
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
    if (n >= 1000) return `${(n / 1000).toFixed(0)}K`;
    return n.toString();
  };

  return (
    <section style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>Codebase Statistics</h2>
        <p style={styles.subtitle}>
          AI-assisted development metrics for {current.version}
        </p>
      </div>

      {/* Key Metrics Grid */}
      <div style={styles.grid}>
        <MetricCard
          icon={<Code size={24} />}
          value={formatNumber(lines.implementation_total)}
          label="Lines of Code"
          sublabel="Go + AILANG + TypeScript"
          color="#e73c17"
        />
        <MetricCard
          icon={<Cpu size={24} />}
          value={formatNumber(tokens.estimated_tokens)}
          label="Est. Tokens"
          sublabel="AI context size"
          color="#6b46c1"
        />
        <MetricCard
          icon={<FileText size={24} />}
          value={formatNumber(lines.documentation)}
          label="Docs Lines"
          sublabel={`${formatNumber(lines.design_docs)} design docs`}
          color="#2c7a7b"
        />
        <MetricCard
          icon={<GitCommit size={24} />}
          value={formatNumber(git.commits)}
          label="Commits"
          sublabel={`${git.contributors} contributors`}
          color="#d69e2e"
        />
      </div>

      {/* Growth Over Time Chart - Stacked Bar */}
      {chartData.length > 1 && (
        <div style={styles.chartSection}>
          <h3 style={styles.chartTitle}>
            <TrendingUp size={20} style={{ marginRight: '0.5rem' }} />
            Codebase Growth Over Time
          </h3>
          <div style={styles.chartWrapper}>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 60 }}>
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
                  tickFormatter={(v) => `${(v / 1000).toFixed(0)}K`}
                />
                <Tooltip content={<CustomTooltip />} />
                <Legend wrapperStyle={{ paddingTop: '10px' }} iconType="square" />
                <Bar dataKey="Go" stackId="code" fill="#e73c17" name="Go Code" />
                <Bar dataKey="AILANG" stackId="code" fill="#6b46c1" name="AILANG" />
                <Bar dataKey="Other" stackId="code" fill="#2c7a7b" name="TS/Shell" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Code Breakdown */}
      <div style={styles.chartSection}>
        <h3 style={styles.chartTitle}>Code Breakdown</h3>
        <div style={styles.chartWrapper}>
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={breakdownData} layout="vertical" margin={{ top: 10, right: 30, left: 100, bottom: 10 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(128,128,128,0.2)" />
              <XAxis
                type="number"
                stroke="var(--ifm-font-color-secondary)"
                tick={{ fill: 'var(--ifm-font-color-secondary)', fontSize: 12 }}
                tickFormatter={(v) => `${(v / 1000).toFixed(0)}K`}
              />
              <YAxis
                type="category"
                dataKey="name"
                stroke="var(--ifm-font-color-secondary)"
                tick={{ fill: 'var(--ifm-font-color-secondary)', fontSize: 12 }}
                width={90}
              />
              <Tooltip
                formatter={(value) => [`${value.toLocaleString()} lines`, 'Lines']}
                contentStyle={{
                  background: 'var(--ifm-background-color)',
                  border: '1px solid rgba(128,128,128,0.2)',
                  borderRadius: '8px',
                }}
              />
              <Bar dataKey="lines" fill="#e73c17" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Detailed Stats Table */}
      <div style={styles.tableSection}>
        <h3 style={styles.chartTitle}>Detailed Statistics</h3>
        <div style={styles.tableWrapper}>
          <table style={styles.table}>
            <thead>
              <tr>
                <th style={styles.th}>Category</th>
                <th style={styles.thRight}>Lines</th>
                <th style={styles.thRight}>Files</th>
              </tr>
            </thead>
            <tbody>
              <tr style={styles.tr}>
                <td style={styles.td}>Go Production Code</td>
                <td style={styles.tdRight}>{lines.go_production.toLocaleString()}</td>
                <td style={styles.tdRight}>{(files.go - Math.floor(files.go * 0.35)).toLocaleString()}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>Go Test Code</td>
                <td style={styles.tdRight}>{lines.go_test.toLocaleString()}</td>
                <td style={styles.tdRight}>{Math.floor(files.go * 0.35).toLocaleString()}</td>
              </tr>
              <tr style={styles.trHighlight}>
                <td style={styles.td}><strong>Total Go</strong></td>
                <td style={styles.tdRight}><strong>{lines.go_total.toLocaleString()}</strong></td>
                <td style={styles.tdRight}><strong>{files.go.toLocaleString()}</strong></td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>AILANG Examples</td>
                <td style={styles.tdRight}>{lines.ailang_examples.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.ailang.toLocaleString()}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>Standard Library</td>
                <td style={styles.tdRight}>{lines.ailang_stdlib.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.stdlib?.toLocaleString() || '-'}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>TypeScript/React</td>
                <td style={styles.tdRight}>{lines.typescript.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.typescript?.toLocaleString() || '-'}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>Shell Scripts</td>
                <td style={styles.tdRight}>{lines.shell.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.shell?.toLocaleString() || '-'}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>Documentation</td>
                <td style={styles.tdRight}>{lines.documentation.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.documentation.toLocaleString()}</td>
              </tr>
              <tr style={styles.tr}>
                <td style={styles.td}>Design Docs</td>
                <td style={styles.tdRight}>{lines.design_docs.toLocaleString()}</td>
                <td style={styles.tdRight}>{files.design_docs?.toLocaleString() || '-'}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Token Estimation Box */}
      <div style={styles.tokenBox}>
        <div style={styles.tokenContent}>
          <Cpu size={32} style={{ color: '#6b46c1' }} />
          <div>
            <div style={styles.tokenValue}>~{(tokens.estimated_tokens / 1000000).toFixed(1)}M tokens</div>
            <div style={styles.tokenLabel}>
              Estimated from {(tokens.total_characters / 1000000).toFixed(1)}M characters (~4 chars/token)
            </div>
          </div>
        </div>
        <p style={styles.tokenNote}>
          This represents the approximate context size if the entire codebase were fed to an AI model.
          The actual AILANG project uses AI-assisted development with Claude Code.
        </p>
      </div>

      <div style={styles.footer}>
        <p style={styles.footerText}>
          Last updated: {new Date(data.lastUpdated).toLocaleDateString()} |
          Version: {current.version} |
          Generated automatically during CI/CD
        </p>
      </div>
    </section>
  );
}

function prepareChartData(history) {
  if (!history || history.length === 0) return [];

  return history.map(entry => ({
    version: entry.version.replace(/^v/, ''),
    Go: entry.lines.go_total,
    AILANG: entry.lines.ailang_examples + entry.lines.ailang_stdlib,
    Other: entry.lines.typescript + entry.lines.shell,
    Total: entry.lines.implementation_total,
    date: new Date(entry.timestamp).toLocaleDateString(),
  })).sort((a, b) => {
    // Sort by version number
    const vA = a.version.split('.').map(Number);
    const vB = b.version.split('.').map(Number);
    for (let i = 0; i < 3; i++) {
      if ((vA[i] || 0) !== (vB[i] || 0)) return (vA[i] || 0) - (vB[i] || 0);
    }
    return 0;
  });
}

function prepareBreakdownData(current) {
  const { lines } = current;
  // Calculate "other docs" as everything not in specific categories
  const knownDocs = (lines.design_docs || 0) + (lines.website_docs || 0) +
                    (lines.prompts || 0) + (lines.changelog || 0);
  const otherDocs = Math.max(0, lines.documentation - knownDocs);

  return [
    { name: 'Go Production', lines: lines.go_production },
    { name: 'Go Tests', lines: lines.go_test },
    { name: 'Design Docs', lines: lines.design_docs },
    { name: 'Website Docs', lines: lines.website_docs || 0 },
    { name: 'Prompts', lines: lines.prompts || 0 },
    { name: 'Changelog', lines: lines.changelog || 0 },
    { name: 'Other Docs', lines: otherDocs },
    { name: 'Shell Scripts', lines: lines.shell },
    { name: 'AILANG', lines: lines.ailang_examples + lines.ailang_stdlib },
    { name: 'TypeScript', lines: lines.typescript },
  ].filter(item => item.lines > 0).sort((a, b) => b.lines - a.lines);
}

function CustomTooltip({ active, payload, label }) {
  if (active && payload && payload.length) {
    const data = payload[0].payload;
    return (
      <div style={styles.tooltip}>
        <div style={styles.tooltipLabel}>v{label}</div>
        <div style={styles.tooltipDate}>{data.date}</div>
        <div style={styles.tooltipValue}>
          <span style={{ ...styles.tooltipDot, background: '#e73c17' }} />
          Go: {data.Go.toLocaleString()}
        </div>
        <div style={styles.tooltipValue}>
          <span style={{ ...styles.tooltipDot, background: '#6b46c1' }} />
          AILANG: {data.AILANG.toLocaleString()}
        </div>
        <div style={styles.tooltipValue}>
          <span style={{ ...styles.tooltipDot, background: '#2c7a7b' }} />
          TS/Shell: {data.Other.toLocaleString()}
        </div>
        <div style={styles.tooltipTotal}>
          Total: {data.Total.toLocaleString()} lines
        </div>
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
    padding: '2rem 0',
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
    gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
    gap: '1.5rem',
    marginBottom: '3rem',
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
    fontSize: '2rem',
    lineHeight: 1,
    marginBottom: '0.5rem',
  },
  label: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 600,
    fontSize: '0.9rem',
    marginBottom: '0.25rem',
  },
  sublabel: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.8rem',
  },
  chartSection: {
    marginBottom: '2.5rem',
  },
  chartTitle: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 700,
    fontSize: '1.25rem',
    marginBottom: '1rem',
    display: 'flex',
    alignItems: 'center',
  },
  chartWrapper: {
    background: 'var(--ifm-background-color)',
    borderRadius: '16px',
    padding: '1.5rem',
    border: '1px solid rgba(128, 128, 128, 0.1)',
  },
  tableSection: {
    marginBottom: '2.5rem',
  },
  tableWrapper: {
    background: 'var(--ifm-background-color)',
    borderRadius: '16px',
    padding: '1.5rem',
    border: '1px solid rgba(128, 128, 128, 0.1)',
    overflowX: 'auto',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
  },
  th: {
    textAlign: 'left',
    padding: '0.75rem',
    borderBottom: '2px solid rgba(128, 128, 128, 0.2)',
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 600,
  },
  thRight: {
    textAlign: 'right',
    padding: '0.75rem',
    borderBottom: '2px solid rgba(128, 128, 128, 0.2)',
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 600,
  },
  tr: {
    borderBottom: '1px solid rgba(128, 128, 128, 0.1)',
  },
  trHighlight: {
    borderBottom: '1px solid rgba(128, 128, 128, 0.1)',
    background: 'rgba(231, 60, 23, 0.05)',
  },
  td: {
    padding: '0.75rem',
  },
  tdRight: {
    textAlign: 'right',
    padding: '0.75rem',
    fontFamily: 'monospace',
  },
  tokenBox: {
    background: 'linear-gradient(135deg, rgba(107, 70, 193, 0.1) 0%, rgba(231, 60, 23, 0.05) 100%)',
    borderRadius: '16px',
    padding: '2rem',
    marginBottom: '2rem',
    border: '1px solid rgba(107, 70, 193, 0.2)',
  },
  tokenContent: {
    display: 'flex',
    alignItems: 'center',
    gap: '1.5rem',
    marginBottom: '1rem',
  },
  tokenValue: {
    fontFamily: "'Montserrat', sans-serif",
    fontWeight: 800,
    fontSize: '1.75rem',
  },
  tokenLabel: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.9rem',
  },
  tokenNote: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.9rem',
    margin: 0,
    lineHeight: 1.6,
  },
  footer: {
    textAlign: 'center',
    paddingTop: '1rem',
    borderTop: '1px solid rgba(128, 128, 128, 0.1)',
  },
  footerText: {
    color: 'var(--ifm-font-color-secondary)',
    fontSize: '0.85rem',
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
  tooltipTotal: {
    marginTop: '0.5rem',
    paddingTop: '0.5rem',
    borderTop: '1px solid rgba(128, 128, 128, 0.2)',
    fontWeight: 600,
  },
  tooltipDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    display: 'inline-block',
  },
};
