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

  return (
    <div className={styles.card}>
      <div className={styles.cardHeader}>
        <Shield className={styles.icon} style={{ color }} />
        <h3>Design Axiom Compliance</h3>
        <span className={styles.badge} style={{ backgroundColor: color, color: '#fff' }}>
          {grade}
        </span>
      </div>
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
