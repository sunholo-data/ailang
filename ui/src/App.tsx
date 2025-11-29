import React, { useState } from 'react';
import { MessageCenter } from './components/MessageCenter/MessageCenter';
import { ApprovalQueue } from './components/ApprovalQueue/ApprovalQueue';
import { Monitor } from './components/Monitor/Monitor';
import { Approval } from './types';

// Icons as inline SVGs for a clean, professional look
const Icons = {
  messages: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  ),
  shield: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  ),
  activity: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  ),
  logo: (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v12M6 12h12" />
      <circle cx="12" cy="12" r="3" fill="currentColor" />
    </svg>
  ),
};

interface AgentInfo {
  id: string;
  last_active?: number;
}

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'messages' | 'approvals' | 'monitor'>('messages');
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [targetAgent, setTargetAgent] = useState<string>('my-agent');
  const [knownAgents, setKnownAgents] = useState<AgentInfo[]>([]);
  const [customAgent, setCustomAgent] = useState<string>('');
  const [showCustomInput, setShowCustomInput] = useState<boolean>(false);

  // WebSocket URL - dynamically use current host/port
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const websocketUrl = `${protocol}//${window.location.host}/ws`;

  // Fetch known agents
  React.useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = await fetch('/api/agents');
        if (response.ok) {
          const agents: AgentInfo[] = await response.json();
          setKnownAgents(agents);
          // Set first agent as default if available
          if (agents.length > 0 && !targetAgent) {
            setTargetAgent(agents[0].id);
          }
        }
      } catch (error) {
        console.error('Error fetching agents:', error);
      }
    };
    fetchAgents();
    const interval = setInterval(fetchAgents, 10000); // Refresh every 10s
    return () => clearInterval(interval);
  }, []);

  const handleAgentSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    if (value === '__custom__') {
      setShowCustomInput(true);
    } else {
      setTargetAgent(value);
      setShowCustomInput(false);
    }
  };

  const handleCustomAgentSubmit = () => {
    if (customAgent.trim()) {
      setTargetAgent(customAgent.trim());
      setShowCustomInput(false);
      setCustomAgent('');
    }
  };

  // Check if agent was active recently (within last 30 seconds)
  const isAgentActive = (agent: AgentInfo): boolean => {
    if (!agent.last_active) return false;
    const now = Date.now();
    const threshold = 30000; // 30 seconds
    return (now - agent.last_active) < threshold;
  };

  const getAgentStatus = (agent: AgentInfo): string => {
    if (isAgentActive(agent)) return '●'; // Green dot in dropdown
    return '○'; // Empty dot
  };

  const handleApprove = async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(`/api/approvals/${approvalId}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      });

      if (!response.ok) {
        console.error('Failed to approve:', await response.text());
        return;
      }

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
      const response = await fetch(`/api/approvals/${approvalId}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      });

      if (!response.ok) {
        console.error('Failed to reject:', await response.text());
        return;
      }

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
    const interval = setInterval(fetchApprovals, 5000);
    return () => clearInterval(interval);
  }, []);

  const pendingCount = approvals?.filter((a) => a.status === 'pending').length || 0;

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-brand">
          <div className="brand-logo">{Icons.logo}</div>
          <div className="brand-text">
            <h1>AILANG</h1>
            <span className="brand-subtitle">Collaboration Hub</span>
          </div>
        </div>

        <nav className="header-nav">
          <button
            className={`nav-tab ${activeTab === 'messages' ? 'active' : ''}`}
            onClick={() => setActiveTab('messages')}
          >
            <span className="nav-icon">{Icons.messages}</span>
            <span className="nav-label">Messages</span>
          </button>
          <button
            className={`nav-tab ${activeTab === 'approvals' ? 'active' : ''}`}
            onClick={() => setActiveTab('approvals')}
          >
            <span className="nav-icon">{Icons.shield}</span>
            <span className="nav-label">Approvals</span>
            {pendingCount > 0 && (
              <span className="nav-badge">{pendingCount}</span>
            )}
          </button>
          <button
            className={`nav-tab ${activeTab === 'monitor' ? 'active' : ''}`}
            onClick={() => setActiveTab('monitor')}
          >
            <span className="nav-icon">{Icons.activity}</span>
            <span className="nav-label">Monitor</span>
          </button>
        </nav>

        <div className="header-meta">
          <div className="agent-selector">
            <label className="agent-label">Target:</label>
            {showCustomInput ? (
              <div className="custom-agent-input">
                <input
                  type="text"
                  value={customAgent}
                  onChange={(e) => setCustomAgent(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleCustomAgentSubmit()}
                  className="agent-input"
                  placeholder="agent-id"
                  autoFocus
                />
                <button onClick={handleCustomAgentSubmit} className="agent-apply">Add</button>
                <button onClick={() => setShowCustomInput(false)} className="agent-cancel">Cancel</button>
              </div>
            ) : (
              <>
                <select
                  value={targetAgent}
                  onChange={handleAgentSelect}
                  className="agent-select"
                >
                  {knownAgents.map((agent) => (
                    <option key={agent.id} value={agent.id}>
                      {getAgentStatus(agent)} {agent.id}
                    </option>
                  ))}
                  {!knownAgents.find(a => a.id === targetAgent) && targetAgent && (
                    <option value={targetAgent}>○ {targetAgent}</option>
                  )}
                  <option value="__custom__">+ Add custom...</option>
                </select>
                {knownAgents.find(a => a.id === targetAgent) && (
                  <span className={`agent-status ${isAgentActive(knownAgents.find(a => a.id === targetAgent)!) ? 'active' : 'inactive'}`}>
                    {isAgentActive(knownAgents.find(a => a.id === targetAgent)!) ? 'Online' : 'Offline'}
                  </span>
                )}
              </>
            )}
          </div>
          <span className="version-tag">v0.5.0</span>
        </div>
      </header>

      <main className="app-content">
        {activeTab === 'messages' && (
          <MessageCenter websocketUrl={websocketUrl} instanceId={targetAgent} />
        )}
        {activeTab === 'approvals' && (
          <ApprovalQueue
            approvals={approvals}
            onApprove={handleApprove}
            onReject={handleReject}
          />
        )}
        {activeTab === 'monitor' && (
          <Monitor />
        )}
      </main>

      <style>{`
        .app {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: var(--bg-base);
          color: var(--text-primary);
        }

        /* Header */
        .app-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          height: 60px;
          padding: 0 var(--space-6);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        /* Brand */
        .header-brand {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .brand-logo {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          border-radius: var(--radius-lg);
          color: var(--text-inverse);
          box-shadow: var(--shadow-glow);
        }

        .brand-text h1 {
          font-size: var(--text-lg);
          font-weight: var(--font-bold);
          letter-spacing: -0.02em;
          color: var(--text-primary);
          line-height: 1;
          margin-bottom: 2px;
        }

        .brand-subtitle {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.1em;
        }

        /* Navigation */
        .header-nav {
          display: flex;
          gap: var(--space-1);
          background: var(--bg-base);
          padding: var(--space-1);
          border-radius: var(--radius-lg);
        }

        .nav-tab {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-2) var(--space-4);
          background: transparent;
          color: var(--text-secondary);
          font-family: var(--font-sans);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          position: relative;
        }

        .nav-tab:hover {
          color: var(--text-primary);
          background: var(--bg-hover);
        }

        .nav-tab.active {
          color: var(--color-primary);
          background: var(--bg-elevated);
        }

        .nav-tab.active::after {
          content: '';
          position: absolute;
          bottom: -1px;
          left: 50%;
          transform: translateX(-50%);
          width: 20px;
          height: 2px;
          background: var(--color-primary);
          border-radius: var(--radius-full);
        }

        .nav-icon {
          display: flex;
          align-items: center;
        }

        .nav-label {
          display: block;
        }

        .nav-badge {
          display: flex;
          align-items: center;
          justify-content: center;
          min-width: 18px;
          height: 18px;
          padding: 0 var(--space-1);
          background: var(--color-danger);
          color: white;
          font-size: 11px;
          font-weight: var(--font-bold);
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.8; }
        }

        /* Header Meta */
        .header-meta {
          display: flex;
          align-items: center;
          gap: var(--space-4);
        }

        .agent-selector {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .agent-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          white-space: nowrap;
        }

        .custom-agent-input {
          display: flex;
          align-items: center;
          gap: var(--space-1);
        }

        .agent-input {
          padding: var(--space-1) var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          width: 120px;
          transition: all var(--transition-fast);
        }

        .agent-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .agent-select {
          padding: var(--space-1) var(--space-3);
          padding-right: var(--space-6);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          cursor: pointer;
          appearance: none;
          background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
          background-repeat: no-repeat;
          background-position: right var(--space-2) center;
          min-width: 140px;
          transition: all var(--transition-fast);
        }

        .agent-select:hover {
          border-color: var(--color-primary);
        }

        .agent-select:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .agent-apply {
          padding: var(--space-1) var(--space-2);
          background: var(--color-primary);
          color: var(--text-inverse);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .agent-apply:hover {
          background: var(--color-primary-light);
        }

        .agent-cancel {
          padding: var(--space-1) var(--space-2);
          background: transparent;
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .agent-cancel:hover {
          background: var(--bg-hover);
          color: var(--text-primary);
        }

        .agent-status {
          font-size: var(--text-xs);
          padding: 2px var(--space-2);
          border-radius: var(--radius-full);
          font-weight: var(--font-medium);
        }

        .agent-status.active {
          background: rgba(46, 160, 67, 0.15);
          color: var(--color-success);
        }

        .agent-status.inactive {
          background: var(--bg-elevated);
          color: var(--text-tertiary);
        }

        .version-tag {
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border-radius: var(--radius-sm);
          border: 1px solid var(--border-subtle);
        }

        /* Content */
        .app-content {
          flex: 1;
          overflow: hidden;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .app-header {
            padding: 0 var(--space-4);
          }

          .brand-text {
            display: none;
          }

          .nav-label {
            display: none;
          }

          .nav-tab {
            padding: var(--space-2) var(--space-3);
          }

          .version-tag {
            display: none;
          }
        }
      `}</style>
    </div>
  );
};
