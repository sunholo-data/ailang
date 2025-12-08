import { useState, useEffect } from 'react';
import { ConnectionState, subscribeToGlobalConnection } from '../../hooks/useWebSocket';
import styles from './ConnectionStatus.module.css';

export function ConnectionStatus() {
  const [state, setState] = useState<ConnectionState>('disconnected');
  const [attempts, setAttempts] = useState(0);

  useEffect(() => {
    return subscribeToGlobalConnection((newState, newAttempts) => {
      setState(newState);
      setAttempts(newAttempts);
    });
  }, []);

  // Don't show when connected (clean UI)
  if (state === 'connected') {
    return (
      <div className={`${styles.indicator} ${styles.connected}`} title="Connected">
        <span className={styles.dot} />
      </div>
    );
  }

  const getStatusText = () => {
    switch (state) {
      case 'connecting':
        return 'Connecting...';
      case 'reconnecting':
        return `Reconnecting... (${attempts})`;
      case 'disconnected':
        return attempts > 0 ? 'Disconnected' : 'Offline';
      default:
        return 'Unknown';
    }
  };

  const getStatusClass = () => {
    switch (state) {
      case 'connecting':
      case 'reconnecting':
        return styles.connecting;
      case 'disconnected':
        return styles.disconnected;
      default:
        return '';
    }
  };

  return (
    <div className={`${styles.indicator} ${getStatusClass()}`} title={getStatusText()}>
      <span className={`${styles.dot} ${state === 'connecting' || state === 'reconnecting' ? styles.pulsing : ''}`} />
      <span className={styles.text}>{getStatusText()}</span>
    </div>
  );
}
