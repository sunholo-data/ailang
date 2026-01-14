/**
 * IterationBadge.tsx - Shows iteration status for retriggered tasks
 *
 * Displays "Iteration N/3" badge with warning styling on final attempt.
 * Part of M-DASHBOARD-APPROVAL-INTEGRATION multi-channel workflow.
 */

import React from 'react';
import './IterationBadge.css';

interface IterationBadgeProps {
  iteration: number;
  maxIterations?: number;
  className?: string;
}

export const IterationBadge: React.FC<IterationBadgeProps> = ({
  iteration,
  maxIterations = 3,
  className = '',
}) => {
  const isFinalAttempt = iteration >= maxIterations;
  const isRetry = iteration > 1;

  if (!isRetry) {
    // Don't show badge for first attempt
    return null;
  }

  return (
    <span
      className={`iteration-badge ${isFinalAttempt ? 'final-attempt' : ''} ${className}`}
      title={isFinalAttempt ? 'Final attempt - no more retries allowed' : `Retry ${iteration - 1} of ${maxIterations - 1}`}
    >
      <span className="iteration-icon">
        {isFinalAttempt ? '⚠️' : '🔄'}
      </span>
      <span className="iteration-text">
        Iteration {iteration}/{maxIterations}
      </span>
      {isFinalAttempt && (
        <span className="final-warning">Final attempt</span>
      )}
    </span>
  );
};

export default IterationBadge;
