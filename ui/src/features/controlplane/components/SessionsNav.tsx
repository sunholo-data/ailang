/**
 * SessionsNav - Sessions list for right sidebar
 * Shows Claude Code sessions with tool summary on expand
 */
import React, { useState } from 'react';
import { useSessionsData, useSessionTools, formatRelativeTime, formatWorkspace } from '../hooks/useSessionsData';
import type { Session, ToolSummaryItem } from '../hooks/useSessionsData';
import styles from '../ControlPlane.module.css';

// Tool icons
const toolIcons: Record<string, string> = {
  Bash: '$',
  Read: '◉',
  Write: '◈',
  Edit: '✎',
  Grep: '⌕',
  Glob: '❖',
  Task: '◎',
  WebFetch: '↗',
  WebSearch: '⌕',
  AskUserQuestion: '?',
  TodoWrite: '☑',
};

interface SessionItemProps {
  session: Session;
  isSelected: boolean;
  onSelect: () => void;
  toolsSummary: ToolSummaryItem[] | null;
}

const SessionItem: React.FC<SessionItemProps> = ({
  session,
  isSelected,
  onSelect,
  toolsSummary,
}) => {
  const workspaceName = formatWorkspace(session.workspace);
  const timeAgo = formatRelativeTime(session.started_at);
  const sessionIdShort = session.session_id.substring(0, 8);

  return (
    <div className={styles.sessionItemWrapper}>
      <div
        className={`${styles.sessionItem} ${isSelected ? styles.sessionItemSelected : ''}`}
        onClick={onSelect}
      >
        <div className={styles.sessionMain}>
          <span className={styles.sessionId}>{sessionIdShort}...</span>
          <span className={styles.sessionWorkspace}>{workspaceName}</span>
        </div>
        <div className={styles.sessionMeta}>
          <span className={styles.sessionTime}>{timeAgo}</span>
          <span className={styles.sessionToolCount}>{session.tool_count} tools</span>
        </div>
      </div>
      {isSelected && toolsSummary && toolsSummary.length > 0 && (
        <div className={styles.toolsSummary}>
          {toolsSummary.map((tool) => (
            <div key={tool.tool_name} className={styles.toolSummaryItem}>
              <span className={styles.toolIcon}>{toolIcons[tool.tool_name] || '·'}</span>
              <span className={styles.toolName}>{tool.tool_name}</span>
              <span className={styles.toolCount}>
                {tool.count}
                {tool.success_count < tool.count && (
                  <span className={styles.toolErrors}>
                    ({tool.count - tool.success_count} err)
                  </span>
                )}
              </span>
              {tool.details && tool.details.length > 0 && (
                <span className={styles.toolDetails} title={tool.details.join(', ')}>
                  {tool.details.slice(0, 3).map(d => formatDetail(d)).join(', ')}
                  {tool.details.length > 3 && ` +${tool.details.length - 3}`}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Format tool detail (shorten paths, patterns)
function formatDetail(detail: string): string {
  if (!detail) return '';
  // For file paths, show just the filename
  if (detail.includes('/')) {
    const parts = detail.split('/');
    return parts[parts.length - 1];
  }
  // Truncate long strings
  if (detail.length > 30) {
    return detail.substring(0, 27) + '...';
  }
  return detail;
}

export interface SessionsNavProps {
  onSessionSelect?: (sessionId: string) => void;
}

export const SessionsNav: React.FC<SessionsNavProps> = ({ onSessionSelect }) => {
  const [expanded, setExpanded] = useState(true);
  const [selectedSession, setSelectedSession] = useState<string | null>(null);
  const { sessions, loading } = useSessionsData({ limit: 10, refreshInterval: 30000 });
  const { tools: toolsSummary } = useSessionTools(selectedSession, { summary: true });

  const handleSessionSelect = (sessionId: string) => {
    if (selectedSession === sessionId) {
      setSelectedSession(null);
    } else {
      setSelectedSession(sessionId);
      onSessionSelect?.(sessionId);
    }
  };

  if (sessions.length === 0 && !loading) {
    return null; // Don't show section if no sessions
  }

  return (
    <div className={styles.sessionsNav}>
      <div
        className={styles.sessionsHeader}
        onClick={() => setExpanded(!expanded)}
      >
        <span className={`${styles.sessionsChevron} ${expanded ? styles.sessionsChevronOpen : ''}`}>
          ▸
        </span>
        <span className={styles.sessionsTitle}>SESSIONS</span>
        <span className={styles.sessionsCount}>
          {loading ? '...' : sessions.length}
        </span>
      </div>
      {expanded && (
        <div className={styles.sessionsList}>
          {loading ? (
            <div className={styles.sessionsLoading}>Loading sessions...</div>
          ) : sessions.length === 0 ? (
            <div className={styles.sessionsEmpty}>No sessions yet</div>
          ) : (
            sessions.map((s) => (
              <SessionItem
                key={s.session_id}
                session={s}
                isSelected={selectedSession === s.session_id}
                onSelect={() => handleSessionSelect(s.session_id)}
                toolsSummary={
                  selectedSession === s.session_id
                    ? (toolsSummary as ToolSummaryItem[])
                    : null
                }
              />
            ))
          )}
        </div>
      )}
    </div>
  );
};

export default SessionsNav;
