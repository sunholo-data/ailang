import { useState, useEffect, useCallback } from 'react';
import { ApprovalHistoryEntry } from '../../../types';
import styles from './ApprovalHistory.module.css';

interface ApprovalHistoryProps {
  threadId?: string;
  limit?: number;
}

// Format timestamp to readable date
function formatTime(timestamp: number): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

// Get action color class
function getActionClass(action: string): string {
  switch (action) {
    case 'approved':
      return styles.approved;
    case 'rejected':
      return styles.rejected;
    case 'expired':
      return styles.expired;
    default:
      return styles.created;
  }
}

// Format cost
function formatCost(cost?: number): string {
  if (cost === undefined || cost === null) return '-';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
}

export function ApprovalHistory({ threadId, limit = 50 }: ApprovalHistoryProps) {
  const [entries, setEntries] = useState<ApprovalHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async () => {
    try {
      let url = `/api/approvals/history?limit=${limit}`;
      if (threadId) {
        url += `&thread_id=${threadId}`;
      }

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to fetch history: ${response.status}`);
      }

      const data = await response.json();
      setEntries(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history');
    } finally {
      setLoading(false);
    }
  }, [threadId, limit]);

  useEffect(() => {
    fetchHistory();
    // Refresh every minute
    const interval = setInterval(fetchHistory, 60000);
    return () => clearInterval(interval);
  }, [fetchHistory]);

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading approval history...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>{error}</div>
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className={styles.container}>
        <div className={styles.empty}>No approval history</div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <h3 className={styles.title}>Approval History</h3>
      <div className={styles.tableContainer}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Time</th>
              <th>Action</th>
              <th>Actor</th>
              <th>Proposal</th>
              <th>Impact</th>
              <th>Cost</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((entry) => (
              <tr key={entry.id}>
                <td className={styles.time}>{formatTime(entry.created_at)}</td>
                <td>
                  <span className={`${styles.action} ${getActionClass(entry.action)}`}>
                    {entry.action}
                  </span>
                </td>
                <td className={styles.actor}>{entry.actor}</td>
                <td className={styles.proposal}>
                  {entry.proposal ? (
                    entry.proposal.length > 50
                      ? `${entry.proposal.slice(0, 50)}...`
                      : entry.proposal
                  ) : (
                    '-'
                  )}
                </td>
                <td>
                  <span className={`${styles.impact} ${styles[`impact${entry.impact || 'low'}`]}`}>
                    {entry.impact || '-'}
                  </span>
                </td>
                <td className={styles.cost}>{formatCost(entry.estimated_cost)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
