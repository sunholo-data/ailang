import React, { useEffect, useRef } from 'react';
import { TaskStreamEvent, TaskStreamEventType } from '../../types';
import styles from './TaskExecution.module.css';

interface StreamingLogProps {
  events: TaskStreamEvent[];
  maxLines?: number;
  autoScroll?: boolean;
}

const getEventIcon = (type: TaskStreamEventType): string => {
  switch (type) {
    case 'log':
    case 'stdout':
      return '>';
    case 'stderr':
      return '!';
    case 'tool_use':
      return '[T]';
    case 'thinking':
      return '...';
    case 'result':
      return '[R]';
    case 'error':
      return '[E]';
    case 'status':
      return '[S]';
    case 'metrics':
      return '[M]';
    default:
      return '';
  }
};

const getEventClass = (type: TaskStreamEventType): string => {
  switch (type) {
    case 'stderr':
    case 'error':
      return styles.logError;
    case 'tool_use':
      return styles.logTool;
    case 'thinking':
      return styles.logThinking;
    case 'result':
      return styles.logResult;
    case 'status':
      return styles.logStatus;
    case 'metrics':
      return styles.logMetrics;
    default:
      return styles.logStdout;
  }
};

const formatTimestamp = (ts: number): string => {
  const date = new Date(ts);
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
};

export const StreamingLog: React.FC<StreamingLogProps> = ({
  events,
  maxLines = 500,
  autoScroll = true,
}) => {
  const logRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);

  // Track scroll position
  useEffect(() => {
    const logElement = logRef.current;
    if (!logElement) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = logElement;
      isAtBottomRef.current = scrollHeight - scrollTop - clientHeight < 50;
    };

    logElement.addEventListener('scroll', handleScroll);
    return () => logElement.removeEventListener('scroll', handleScroll);
  }, []);

  // Auto-scroll when new events arrive
  useEffect(() => {
    if (autoScroll && isAtBottomRef.current && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [events, autoScroll]);

  // Limit displayed events
  const displayEvents = events.length > maxLines
    ? events.slice(-maxLines)
    : events;

  return (
    <div className={styles.streamingLog} ref={logRef}>
      {events.length === 0 && (
        <div className={styles.emptyLog}>
          Waiting for task events...
        </div>
      )}
      {displayEvents.map((event, index) => (
        <div key={`${event.timestamp}-${index}`} className={`${styles.logLine} ${getEventClass(event.event_type)}`}>
          <span className={styles.timestamp}>{formatTimestamp(event.timestamp)}</span>
          <span className={styles.icon}>{getEventIcon(event.event_type)}</span>
          {event.tool_name && (
            <span className={styles.toolName}>[{event.tool_name}]</span>
          )}
          <span className={styles.content}>
            {event.content || event.tool_input || event.status || ''}
          </span>
        </div>
      ))}
    </div>
  );
};

export default StreamingLog;
