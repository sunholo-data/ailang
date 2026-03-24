import React, { useState, useMemo } from 'react';
import { usePackageIndex, useEcosystemStats } from '@site/src/hooks/useRegistryData';
import { BarChart, Bar, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import styles from './styles.module.css';

const CHART_COLORS = ['#e73c17', '#2c7a7b', '#6b46c1', '#dd6b20', '#2b6cb0', '#38a169', '#d69e2e'];

function timeAgo(dateStr) {
  if (!dateStr) return '';
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86400000);
  if (days === 0) return 'today';
  if (days === 1) return '1d ago';
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

export default function PackageExplorer() {
  const { data: indexData, loading, error, stale } = usePackageIndex();
  const { data: stats } = useEcosystemStats();
  const [search, setSearch] = useState('');
  const [effectFilter, setEffectFilter] = useState('');
  const [stabilityFilter, setStabilityFilter] = useState('');
  const [expanded, setExpanded] = useState(null);

  const packages = indexData?.packages || [];

  // Collect unique effects and stabilities for filters
  const allEffects = useMemo(() => {
    const set = new Set();
    packages.forEach((p) => (p.effects || []).forEach((e) => set.add(e)));
    return ['', ...Array.from(set).sort()];
  }, [packages]);

  const allStabilities = useMemo(() => {
    const set = new Set();
    packages.forEach((p) => set.add(p.stability || 'experimental'));
    return ['', ...Array.from(set).sort()];
  }, [packages]);

  // Filter packages
  const filtered = useMemo(() => {
    return packages.filter((p) => {
      const q = search.toLowerCase();
      const matchesSearch = !q ||
        p.name.toLowerCase().includes(q) ||
        (p.ai_summary || '').toLowerCase().includes(q) ||
        (p.tags || []).some((t) => t.toLowerCase().includes(q));

      const matchesEffect = !effectFilter ||
        (effectFilter === 'Pure' ? (!p.effects || p.effects.length === 0) : (p.effects || []).includes(effectFilter));

      const matchesStability = !stabilityFilter ||
        (p.stability || 'experimental') === stabilityFilter;

      return matchesSearch && matchesEffect && matchesStability;
    });
  }, [packages, search, effectFilter, stabilityFilter]);

  if (loading) {
    return (
      <div className={styles.loading}>
        <div className={styles.spinner} />
        <p>Loading packages from registry...</p>
      </div>
    );
  }

  if (error && packages.length === 0) {
    return (
      <div className={styles.loading}>
        <p>Could not load registry data: {error}</p>
      </div>
    );
  }

  return (
    <div className={styles.explorer}>
      {stale && (
        <div className={styles.staleBanner}>
          Showing cached data — registry API may be temporarily unavailable.
        </div>
      )}

      {/* Stats Bar */}
      {stats && <StatsBar stats={stats} />}

      {/* Charts */}
      {stats && <StatsCharts stats={stats} />}

      {/* Search & Filters */}
      <div className={styles.searchBar}>
        <input
          type="text"
          className={styles.searchInput}
          placeholder="Search packages by name, description, or tag..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select
          className={styles.filterSelect}
          value={effectFilter}
          onChange={(e) => setEffectFilter(e.target.value)}
        >
          <option value="">All Effects</option>
          <option value="Pure">Pure</option>
          {allEffects.filter(Boolean).map((e) => (
            <option key={e} value={e}>{e}</option>
          ))}
        </select>
        <select
          className={styles.filterSelect}
          value={stabilityFilter}
          onChange={(e) => setStabilityFilter(e.target.value)}
        >
          <option value="">All Stability</option>
          {allStabilities.filter(Boolean).map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>

      <div className={styles.resultCount}>
        {filtered.length} of {packages.length} packages
      </div>

      {/* Package List */}
      <div className={styles.packageList}>
        {filtered.map((pkg) => {
          const shortName = pkg.name.split('/')[1];
          const isPure = !pkg.effects || pkg.effects.length === 0;
          const effects = isPure ? 'Pure' : pkg.effects.join(', ');

          return (
            <a
              key={pkg.name}
              href={`/docs/packages/${pkg.name}`}
              className={styles.packageListCard}
            >
              <div className={styles.listCardTop}>
                <div className={styles.listCardInfo}>
                  <div className={styles.cardHeader}>
                    <span className={styles.packageName}>{pkg.name}</span>
                    <span className={styles.version}>v{pkg.latest}</span>
                  </div>
                  <p className={styles.summary}>{pkg.ai_summary || 'No description'}</p>
                  <div className={styles.badges}>
                    <span className={`${styles.badge} ${
                      pkg.stability === 'stable' ? styles.stabilityStable :
                      pkg.stability === 'frozen' ? styles.stabilityFrozen :
                      styles.stabilityBadge
                    }`}>
                      {pkg.stability || 'experimental'}
                    </span>
                    <span className={`${styles.badge} ${isPure ? styles.effectPure : styles.effectBadge}`}>
                      {effects}
                    </span>
                    {(pkg.tags || []).slice(0, 4).map((tag) => (
                      <span key={tag} className={`${styles.badge} ${styles.tagBadge}`}>{tag}</span>
                    ))}
                  </div>
                </div>
                <div className={styles.listCardRight}>
                  <span className={styles.timeAgo}>{timeAgo(pkg.last_updated)}</span>
                </div>
              </div>
            </a>
          );
        })}
      </div>

      {filtered.length === 0 && (
        <div className={styles.emptyState}>
          <p>No packages match your search.</p>
        </div>
      )}
    </div>
  );
}

function StatsBar({ stats }) {
  return (
    <div className={styles.statsBar}>
      <div className={styles.statCard}>
        <div className={styles.statValue}>{stats.total_packages}</div>
        <div className={styles.statLabel}>Packages</div>
      </div>
      <div className={styles.statCard}>
        <div className={styles.statValue}>{stats.total_versions}</div>
        <div className={styles.statLabel}>Versions</div>
      </div>
      <div className={styles.statCard}>
        <div className={styles.statValue}>{stats.pure_packages}</div>
        <div className={styles.statLabel}>Pure (No Effects)</div>
      </div>
      <div className={styles.statCard}>
        <div className={styles.statValue}>{Math.round(stats.validation_pass_rate * 100)}%</div>
        <div className={styles.statLabel}>Validation Pass</div>
      </div>
    </div>
  );
}

function StatsCharts({ stats }) {
  const effectData = Object.entries(stats.effect_distribution || {})
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value);

  const stabilityData = Object.entries(stats.stability_breakdown || {})
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value);

  const agentHumanData = [
    { name: 'Agent', value: stats.agent_vs_human?.agent || 0 },
    { name: 'Human', value: stats.agent_vs_human?.human || 0 },
  ].filter(d => d.value > 0);

  // Cap at top 10 for scalability — works at any package count
  const topDeps = (stats.top_depended_on || []).slice(0, 10).map((d) => ({
    name: d.name.split('/')[1],
    value: d.dependent_count,
  }));

  if (effectData.length === 0 && topDeps.length === 0) return null;

  const barHeight = (data) => Math.max(120, data.length * 32 + 20);

  return (
    <div className={styles.chartsGrid}>
      {effectData.length > 0 && (
        <div className={styles.chartPanel}>
          <div className={styles.chartTitle}>Effect Usage</div>
          <ResponsiveContainer width="100%" height={barHeight(effectData)}>
            <BarChart data={effectData} layout="vertical" margin={{ left: 10, right: 20, top: 5, bottom: 5 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="name" width={50} tick={{ fontSize: 12, fontFamily: 'JetBrains Mono, monospace' }} axisLine={false} tickLine={false} />
              <Tooltip cursor={{ fill: 'var(--ifm-color-emphasis-100)' }} />
              <Bar dataKey="value" radius={[0, 4, 4, 0]} barSize={20}>
                {effectData.map((_, i) => (
                  <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {stabilityData.length > 0 && (
        <div className={styles.chartPanel}>
          <div className={styles.chartTitle}>Stability</div>
          <ResponsiveContainer width="100%" height={barHeight(stabilityData)}>
            <BarChart data={stabilityData} layout="vertical" margin={{ left: 10, right: 20, top: 5, bottom: 5 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="name" width={100} tick={{ fontSize: 12, fontFamily: 'JetBrains Mono, monospace' }} axisLine={false} tickLine={false} />
              <Tooltip cursor={{ fill: 'var(--ifm-color-emphasis-100)' }} />
              <Bar dataKey="value" fill="#2c7a7b" radius={[0, 4, 4, 0]} barSize={20} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {topDeps.length > 0 && (
        <div className={styles.chartPanel}>
          <div className={styles.chartTitle}>Most Depended On</div>
          <ResponsiveContainer width="100%" height={barHeight(topDeps)}>
            <BarChart data={topDeps} layout="vertical" margin={{ left: 10, right: 20, top: 5, bottom: 5 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="name" width={110} tick={{ fontSize: 11, fontFamily: 'JetBrains Mono, monospace' }} axisLine={false} tickLine={false} />
              <Tooltip cursor={{ fill: 'var(--ifm-color-emphasis-100)' }} />
              <Bar dataKey="value" fill="#6b46c1" radius={[0, 4, 4, 0]} barSize={20} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {agentHumanData.length > 0 && (
        <div className={styles.chartPanel}>
          <div className={styles.chartTitle}>Updates By</div>
          <ResponsiveContainer width="100%" height={barHeight(agentHumanData)}>
            <BarChart data={agentHumanData} layout="vertical" margin={{ left: 10, right: 20, top: 5, bottom: 5 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="name" width={60} tick={{ fontSize: 12, fontFamily: 'JetBrains Mono, monospace' }} axisLine={false} tickLine={false} />
              <Tooltip cursor={{ fill: 'var(--ifm-color-emphasis-100)' }} />
              <Bar dataKey="value" radius={[0, 4, 4, 0]} barSize={20}>
                {agentHumanData.map((d, i) => (
                  <Cell key={i} fill={d.name === 'Agent' ? '#e73c17' : '#2b6cb0'} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
