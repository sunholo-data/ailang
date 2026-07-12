import React, { useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ReferenceLine } from 'recharts';
import styles from './styles.module.css';
import { useEvents, annotationColor, groupByVersion } from './useEvents';
import { useOSHistory } from './osHistory';

export default function SuccessTrend({ history, languages, events, selectedTier, coverage }) {
  const annotations = useEvents(events, { selectedTier });
  const eventsByVersion = groupByVersion(annotations, formatVersion);

  // On-device rig: a "Local agent" aggregate AILANG line (mean of the local models
  // per version) so the whole-page trend shows how on-device tracks vs cloud.
  const osHistory = useOSHistory();
  const osByVer = useMemo(() => {
    const m = {};
    (osHistory || []).forEach((e) => { if (e && e.ailang_version) m[e.ailang_version] = e.rows || []; });
    return m;
  }, [osHistory]);
  const localModelNames = useMemo(() => {
    const s = new Set();
    Object.values(osByVer).forEach((rows) => rows.forEach((r) => r && r.model && s.add(r.model)));
    return s;
  }, [osByVer]);
  // Incomplete while any local model is still under-covered (auto-clears via coverage).
  const anyLocalIncomplete = [...localModelNames].some((m) => (coverage ? coverage.isProvisional(m) : true));
  // Filter out entries with invalid timestamps (0001-01-01 means no timestamp)
  const validHistory = history.filter(h => {
    const date = new Date(h.timestamp);
    return date.getFullYear() > 2000; // Only show entries with real timestamps
  });

  // Sort history by timestamp (oldest first for proper trend display)
  const sortedHistory = [...validHistory].sort((a, b) => {
    const dateA = new Date(a.timestamp);
    const dateB = new Date(b.timestamp);
    return dateA - dateB;
  });

  // Transform history data for recharts. When selectedTier is set, prefer
  // the tier-scoped snapshot (baseline.tiers[t]) so this chart updates
  // alongside TierToggle instead of showing all-tier numbers.
  const chartData = sortedHistory.map((baseline, index) => {
    const langs = baseline.languages || '';
    const isLatest = index === sortedHistory.length - 1;

    let ailangRate = 0;
    let pythonRate = 0;

    const tierSnap = selectedTier ? baseline.tiers?.[selectedTier] : null;

    if (tierSnap) {
      ailangRate = (tierSnap.ailang_success_rate || 0) * 100;
      pythonRate = (tierSnap.python_success_rate || 0) * 100;
    } else if (baseline.languageStats) {
      ailangRate = (baseline.languageStats.ailang?.success_rate || 0) * 100;
      pythonRate = (baseline.languageStats.python?.success_rate || 0) * 100;
    } else if (isLatest && languages) {
      // Fallback: Use top-level language stats for latest version
      ailangRate = (languages.ailang?.success_rate || 0) * 100;
      pythonRate = (languages.python?.success_rate || 0) * 100;
    } else if (langs === 'ailang') {
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = combinedRate;
      pythonRate = 0;
    } else if (langs === 'python') {
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = 0;
      pythonRate = combinedRate;
    } else {
      const combinedRate = (baseline.successRate || 0) * 100;
      ailangRate = combinedRate;
      pythonRate = combinedRate;
    }

    const base = (baseline.version || '').split('-')[0];
    const osRows = osByVer[baseline.version] || osByVer[base] || [];
    const localRates = osRows.map((r) => r && r.lang && r.lang.ailang).filter((v) => typeof v === 'number');
    const localMean = localRates.length
      ? parseFloat(((localRates.reduce((a, b) => a + b, 0) / localRates.length) * 100).toFixed(1))
      : null;

    return {
      version: formatVersion(baseline.version),
      'AILANG': parseFloat(ailangRate.toFixed(1)),
      'Python': parseFloat(pythonRate.toFixed(1)),
      'Local agent': localMean,
      date: baseline.timestamp ? new Date(baseline.timestamp).toLocaleDateString() : ''
    };
  });

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      const eventsHere = eventsByVersion.get(label) || [];
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          {data.date && <p className={styles.tooltipDate}>{data.date}</p>}
          <p className={styles.tooltipValue}>
            <span className={styles.tooltipDot} style={{backgroundColor: '#2e8555'}} />
            AILANG: {data['AILANG']}%
          </p>
          {data['Python'] > 0 && (
            <p className={styles.tooltipValue}>
              <span className={styles.tooltipDot} style={{backgroundColor: '#ffa726'}} />
              Python: {data['Python']}%
            </p>
          )}
          {eventsHere.length > 0 && (
            <>
              <p className={styles.tooltipRuns} style={{marginTop: '8px', fontSize: '11px', color: '#666'}}>
                Release events:
              </p>
              {eventsHere.map((ev, i) => (
                <p key={i} className={styles.tooltipValue} style={{fontSize: '11px'}}>
                  <span className={styles.tooltipDot} style={{backgroundColor: annotationColor(ev)}} />
                  {ev.label}
                </p>
              ))}
            </>
          )}
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.chartContainer}>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis
            dataKey="version"
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)', fontSize: 12 }}
            angle={-45}
            textAnchor="end"
            height={80}
          />
          <YAxis
            stroke="var(--ifm-color-emphasis-600)"
            tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
            domain={[0, 100]}
            label={{ value: 'Success Rate (%)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip content={<CustomTooltip />} wrapperStyle={{ zIndex: 1000, outline: 'none' }} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="circle"
          />
          {Array.from(eventsByVersion.entries()).map(([formattedVersion, evs]) => {
            const exists = chartData.some(d => d.version === formattedVersion);
            if (!exists) return null;
            const color = annotationColor(evs[0]);
            const marker = evs.length > 1 ? `● ${evs.length}` : '●';
            return (
              <ReferenceLine
                key={`ev-${formattedVersion}`}
                x={formattedVersion}
                stroke={color}
                strokeDasharray="4 4"
                label={{ value: marker, position: 'top', fill: color, fontSize: 12, fontWeight: 600 }}
              />
            );
          })}
          <Line
            type="monotone"
            dataKey="AILANG"
            stroke="var(--ifm-color-primary)"
            strokeWidth={3}
            dot={{ r: 5 }}
            activeDot={{ r: 7 }}
          />
          <Line
            type="monotone"
            dataKey="Python"
            stroke="#ffa726"
            strokeWidth={3}
            dot={{ r: 5 }}
            activeDot={{ r: 7 }}
          />
          <Line
            type="monotone"
            dataKey="Local agent"
            name={anyLocalIncomplete ? 'Local agent *' : 'Local agent'}
            stroke="#0891b2"
            strokeWidth={3}
            strokeDasharray="5 3"
            dot={{ r: 4 }}
            activeDot={{ r: 6 }}
            connectNulls
          />
        </LineChart>
      </ResponsiveContainer>
      {anyLocalIncomplete && chartData.some((d) => d['Local agent'] != null) && (
        <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)', marginTop: 8 }}>
          * <strong>Local agent</strong> is the on-device rig&apos;s mean AILANG rate — over an
          <strong> incomplete</strong> benchmark set so far; it fills in as the rotation runs.
        </p>
      )}
    </div>
  );
}

function formatVersion(version) {
  // Shorten version strings for display
  if (!version) return 'Unknown';

  // Remove 'v' prefix if present
  version = version.replace(/^v/, '');

  // For git versions like "0.3.0-35-g3530d07", show "v0.3.0-35"
  const parts = version.split('-');
  if (parts.length >= 3) {
    return `v${parts[0]}-${parts[1]}`;
  }

  // For simple versions, show as-is
  return `v${version}`;
}
