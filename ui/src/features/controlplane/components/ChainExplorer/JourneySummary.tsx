/**
 * JourneySummary - Compact narrative of a chain's execution journey.
 * Fetches from GET /api/chains/{id}/journey and renders a summary line
 * with step pills showing status, agent, and cost.
 *
 * M-CHAIN-JOURNEY: "The journey of how code gets written"
 */
import React, { useState, useEffect } from 'react';
import { formatCost } from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

interface JourneyStep {
  stage_number: number;
  agent_id: string;
  action: string;
  status: string;
  approval_status?: string;
  iteration: number;
  feedback?: string;
  error_excerpt?: string;
  cost: number;
  duration_ms: number;
}

interface JourneyData {
  chain_id: string;
  steps: JourneyStep[];
  summary: string;
}

interface JourneySummaryProps {
  chainId: string;
}

const STATUS_DOT: Record<string, string> = {
  completed: '#3fb950',
  failed: '#f85149',
  running: '#d29922',
  awaiting_approval: '#58a6ff',
};

export const JourneySummary: React.FC<JourneySummaryProps> = ({ chainId }) => {
  const [journey, setJourney] = useState<JourneyData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    fetch(`/api/chains/${encodeURIComponent(chainId)}/journey`)
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (!cancelled) {
          setJourney(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setLoading(false);
      });

    return () => { cancelled = true; };
  }, [chainId]);

  if (loading) {
    return <div className={styles.journeySkeleton}>Loading journey...</div>;
  }
  if (!journey || !journey.steps || journey.steps.length === 0) {
    return null;
  }

  return (
    <div className={styles.journeyContainer}>
      <div className={styles.journeySummaryLine}>{journey.summary}</div>
      <div className={styles.journeySteps}>
        {journey.steps.map((step, i) => (
          <div key={i} className={styles.journeyStep}>
            <span
              className={styles.journeyDot}
              style={{ backgroundColor: STATUS_DOT[step.status] || '#8b949e' }}
            />
            <span className={styles.journeyAction}>{step.action}</span>
            {step.iteration > 1 && (
              <span className={styles.journeyIteration}>x{step.iteration}</span>
            )}
            {step.cost > 0 && (
              <span className={styles.journeyCost}>{formatCost(step.cost)}</span>
            )}
            {step.error_excerpt && (
              <span className={styles.journeyError} title={step.error_excerpt}>!</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default JourneySummary;
