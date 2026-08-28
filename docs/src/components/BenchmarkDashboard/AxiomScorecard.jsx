import React, { useState, useEffect } from 'react';
import { Shield, CheckCircle, AlertCircle, XCircle, ChevronDown, ChevronUp } from 'lucide-react';
import styles from './styles.module.css';

export default function AxiomScorecard() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    fetch('/benchmarks/axiom_scorecard.json')
      .then(res => {
        if (!res.ok) throw new Error('Failed to load axiom scorecard');
        return res.json();
      })
      .then(data => {
        setData(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Error loading axiom scorecard:', err);
        setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <Shield className={styles.icon} />
          <h3>Design Axiom Compliance</h3>
        </div>
        <div className={styles.cardContent}>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return null; // Silently fail if scorecard not available
  }

  const { summary, axioms } = data;
  const percentage = summary.percentage;

  // Get grade and color
  const getGrade = (pct) => {
    if (pct >= 90) return { grade: 'A', color: '#22c55e' };
    if (pct >= 80) return { grade: 'B', color: '#06b6d4' };
    if (pct >= 70) return { grade: 'C', color: '#eab308' };
    if (pct >= 60) return { grade: 'D', color: '#f97316' };
    return { grade: 'F', color: '#ef4444' };
  };

  const { grade, color } = getGrade(percentage);

  // Axiom order
  const axiomOrder = [
    'A1_determinism', 'A2_replayability', 'A3_effect_legibility',
    'A4_explicit_authority', 'A5_bounded_verification', 'A6_safe_concurrency',
    'A7_machines_first', 'A8_minimal_syntax', 'A9_cost_visibility',
    'A10_composability', 'A11_structured_failure', 'A12_system_boundary',
  ];

  const getStatusIcon = (status) => {
    switch (status) {
      case 'strong':
        return <CheckCircle className={styles.iconSmall} style={{ color: '#22c55e' }} />;
      case 'partial':
        return <AlertCircle className={styles.iconSmall} style={{ color: '#eab308' }} />;
      default:
        return <XCircle className={styles.iconSmall} style={{ color: '#ef4444' }} />;
    }
  };

  // Staleness: this card is fed by a STATIC file (no runtime refresh path), so
  // it silently showed v0.15.0/2026-05-04 numbers for ~4 months and ~19 releases
  // with nothing saying so (M-EVAL-ROLLING-ELO M5). Until it is wired into the
  // publish pipeline, say how old it is rather than implying it is current.
  const scorecardAge = (() => {
    if (!data?.timestamp) return null;
    const then = new Date(data.timestamp);
    if (!(then.getFullYear() > 2000)) return null;
    const days = Math.floor((Date.now() - then.getTime()) / 86400000);
    return { days, when: then.toISOString().slice(0, 10) };
  })();
  const isStale = scorecardAge && scorecardAge.days > 45;

  return (
    <div className={styles.card}>
      <div className={styles.cardHeader}>
        <Shield className={styles.icon} style={{ color }} />
        <h3>Design Axiom Compliance</h3>
        <span className={styles.badge} style={{ backgroundColor: color, color: '#fff' }}>
          {grade}
        </span>
      </div>
      {scorecardAge && (
        <div
          style={{
            fontSize: '0.8rem',
            opacity: 0.85,
            padding: '0 1rem',
            fontWeight: isStale ? 600 : 400,
          }}
          title={
            isStale
              ? 'This scorecard is published as a static file and is not refreshed by the benchmark pipeline.'
              : undefined
          }
        >
          {isStale ? '⚠ ' : ''}
          Scorecard data: {data.version || 'unknown version'}, {scorecardAge.when}
          {isStale ? ` — ${scorecardAge.days} days old (static file, not auto-refreshed)` : ''}
        </div>
      )}
      <div className={styles.cardContent}>
        {/* Progress bar */}
        <div className={styles.progressContainer}>
          <div className={styles.progressBar}>
            <div
              className={styles.progressFill}
              style={{ width: `${percentage}%`, backgroundColor: color }}
            />
          </div>
          <span className={styles.progressLabel}>
            {summary.totalScore}/{summary.maxScore} ({percentage.toFixed(0)}%)
          </span>
        </div>

        {/* Summary stats */}
        <div className={styles.axiomStats}>
          <div className={styles.statItem}>
            <CheckCircle className={styles.iconSmall} style={{ color: '#22c55e' }} />
            <span>{summary.strongCount} strong</span>
          </div>
          <div className={styles.statItem}>
            <AlertCircle className={styles.iconSmall} style={{ color: '#eab308' }} />
            <span>{summary.partialCount} partial</span>
          </div>
          <div className={styles.statItem}>
            <XCircle className={styles.iconSmall} style={{ color: '#ef4444' }} />
            <span>{summary.violationCount} violations</span>
          </div>
        </div>

        {/* Expandable axiom list */}
        <button
          className={styles.expandButton}
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? 'Hide details' : 'Show all axioms'}
          {expanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>

        {expanded && (
          <div className={styles.axiomList}>
            {axiomOrder.map(key => {
              const axiom = axioms[key];
              if (!axiom) return null;
              const id = key.split('_')[0];
              return (
                <div key={key} className={styles.axiomItem}>
                  <div className={styles.axiomId}>{id}</div>
                  <div className={styles.axiomName}>{axiom.name}</div>
                  <div className={styles.axiomScore}>
                    {getStatusIcon(axiom.status)}
                    <span>{axiom.score}/{axiom.maxScore}</span>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <a
          href="/docs/references/axioms"
          className={styles.link}
        >
          View axiom definitions
        </a>
      </div>
    </div>
  );
}
