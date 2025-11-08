import React, { useState } from 'react';
import { MessageCenter } from './components/MessageCenter/MessageCenter';
import { ApprovalQueue } from './components/ApprovalQueue/ApprovalQueue';
import { Approval } from './types';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'messages' | 'approvals'>('messages');
  const [approvals, setApprovals] = useState<Approval[]>([]);

  // WebSocket URL - dynamically use current host/port
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const websocketUrl = `${protocol}//${window.location.host}/ws`;
  const instanceId = 'user'; // or get from auth

  const handleApprove = async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(
        `/api/approvals/${approvalId}/approve`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ notes }),
        }
      );

      if (!response.ok) {
        console.error('Failed to approve:', await response.text());
        return;
      }

      // Update local state
      setApprovals((prev) =>
        prev.map((a) =>
          a.id === approvalId
            ? { ...a, status: 'approved', reviewed_by: 'user', review_notes: notes }
            : a
        )
      );
    } catch (error) {
      console.error('Error approving:', error);
    }
  };

  const handleReject = async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(
        `/api/approvals/${approvalId}/reject`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ notes }),
        }
      );

      if (!response.ok) {
        console.error('Failed to reject:', await response.text());
        return;
      }

      // Update local state
      setApprovals((prev) =>
        prev.map((a) =>
          a.id === approvalId
            ? { ...a, status: 'rejected', reviewed_by: 'user', review_notes: notes }
            : a
        )
      );
    } catch (error) {
      console.error('Error rejecting:', error);
    }
  };

  // Fetch approvals from backend API
  React.useEffect(() => {
    const fetchApprovals = async () => {
      try {
        const response = await fetch('/api/approvals?status=pending');
        if (!response.ok) {
          console.error('Failed to fetch approvals:', await response.text());
          return;
        }
        const data: Approval[] = await response.json();
        setApprovals(data);
      } catch (error) {
        console.error('Error fetching approvals:', error);
      }
    };

    fetchApprovals();

    // Poll for new approvals every 5 seconds
    const interval = setInterval(fetchApprovals, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="app">
      <header className="app-header">
        <h1>🤖 AILANG Collaboration Hub</h1>
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'messages' ? 'active' : ''}`}
            onClick={() => setActiveTab('messages')}
          >
            💬 Messages
          </button>
          <button
            className={`tab ${activeTab === 'approvals' ? 'active' : ''}`}
            onClick={() => setActiveTab('approvals')}
          >
            🔒 Approvals
            {approvals && approvals.filter((a) => a.status === 'pending').length > 0 && (
              <span className="badge">
                {approvals.filter((a) => a.status === 'pending').length}
              </span>
            )}
          </button>
        </div>
      </header>

      <main className="app-content">
        {activeTab === 'messages' ? (
          <MessageCenter websocketUrl={websocketUrl} instanceId={instanceId} />
        ) : (
          <ApprovalQueue
            approvals={approvals}
            onApprove={handleApprove}
            onReject={handleReject}
          />
        )}
      </main>

      <style>{`
        .app {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: #f5f5f5;
        }

        .app-header {
          background: white;
          border-bottom: 2px solid #e0e0e0;
          padding: 1rem 2rem;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
        }

        .app-header h1 {
          font-size: 1.5rem;
          font-weight: 600;
          margin-bottom: 1rem;
          color: #333;
        }

        .tabs {
          display: flex;
          gap: 0.5rem;
        }

        .tab {
          padding: 0.75rem 1.5rem;
          background: #f5f5f5;
          border: none;
          border-radius: 8px 8px 0 0;
          cursor: pointer;
          font-size: 0.9375rem;
          font-weight: 500;
          color: #666;
          transition: all 0.2s;
          position: relative;
        }

        .tab:hover {
          background: #e0e0e0;
          color: #333;
        }

        .tab.active {
          background: #007bff;
          color: white;
        }

        .badge {
          position: absolute;
          top: 0.25rem;
          right: 0.25rem;
          background: #dc3545;
          color: white;
          padding: 0.125rem 0.375rem;
          border-radius: 10px;
          font-size: 0.75rem;
          font-weight: 600;
        }

        .app-content {
          flex: 1;
          overflow: hidden;
        }
      `}</style>
    </div>
  );
};
