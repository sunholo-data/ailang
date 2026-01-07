/**
 * DetailPanel - Slide-in panel for viewing details of agents, events, etc.
 */
import React from 'react';
import type { Agent, DetailPanelState, HeatmapCell, EventMessage, TrustCapability } from './types';
import { getTrustLevel } from './utils';
import styles from '../ControlPlane.module.css';

export interface DetailPanelProps {
  state: DetailPanelState;
  onClose: () => void;
  agents: Agent[];
  trustCapabilities?: TrustCapability[];
  onTrustChange?: (name: string, score: number) => void;
}

export const DetailPanel: React.FC<DetailPanelProps> = ({
  state,
  onClose,
  agents,
  trustCapabilities,
  onTrustChange,
}) => {
  if (!state.type || !state.id) return null;

  const renderContent = () => {
    switch (state.type) {
      case 'agent': {
        const agent = agents.find(a => a.id === state.id);
        if (!agent) return <div>Agent not found</div>;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Status</h4>
              <span className={`${styles.detailStatus} ${styles[`status${agent.status.charAt(0).toUpperCase() + agent.status.slice(1)}`]}`}>
                {agent.status.toUpperCase()}
              </span>
            </div>
            <div className={styles.detailSection}>
              <h4>Statistics</h4>
              <div className={styles.detailStats}>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Tasks</span>
                  <span className={styles.detailStatValue}>{agent.taskCount}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Trust Score</span>
                  <span className={styles.detailStatValue}>{agent.trustScore}%</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Cost</span>
                  <span className={styles.detailStatValue}>${agent.cost.toFixed(2)}</span>
                </div>
              </div>
            </div>
            {trustCapabilities && onTrustChange && (
              <div className={styles.detailSection}>
                <h4>Trust Configuration</h4>
                <div className={styles.trustCapabilities}>
                  {trustCapabilities.map((cap) => {
                    const level = getTrustLevel(cap.score);
                    // Convert 'low-risk' to 'LowRisk' for CSS class
                    const levelClass = level.split('-').map(s => s.charAt(0).toUpperCase() + s.slice(1)).join('');
                    return (
                      <div key={cap.name} className={styles.trustRow}>
                        <div className={styles.trustCapLabel}>
                          <span className={styles.trustCapIcon}>{cap.icon}</span>
                          <span className={styles.trustCapName}>{cap.name}</span>
                          <span className={`${styles.trustValue} ${styles[`trust${levelClass}`]}`}>
                            {cap.score}%
                          </span>
                        </div>
                        <div className={styles.trustSliderContainer}>
                          <input
                            type="range"
                            min="0"
                            max="100"
                            value={cap.score}
                            onChange={(e) => onTrustChange(cap.name, parseInt(e.target.value))}
                            className={styles.trustSlider}
                          />
                          <div
                            className={`${styles.trustFill} ${styles[`trust${levelClass}`]}`}
                            style={{ width: `${cap.score}%` }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
            <div className={styles.detailSection}>
              <h4>Actions</h4>
              <div className={styles.detailActions}>
                <button className={styles.actionBtn}>View Messages</button>
                <button className={styles.actionBtn}>View Traces</button>
                <button className={`${styles.actionBtn} ${styles.actionDanger}`}>Pause Agent</button>
              </div>
            </div>
          </div>
        );
      }
      case 'date': {
        const cell = state.data as HeatmapCell;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Activity Summary</h4>
              <div className={styles.detailStats}>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Tasks</span>
                  <span className={styles.detailStatValue}>{cell.taskCount}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Cost</span>
                  <span className={styles.detailStatValue}>${cell.cost.toFixed(3)}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Success</span>
                  <span className={styles.detailStatValue}>{(cell.successRate * 100).toFixed(0)}%</span>
                </div>
              </div>
            </div>
            <div className={styles.detailSection}>
              <h4>Actions</h4>
              <div className={styles.detailActions}>
                <button className={styles.actionBtn}>View Tasks</button>
                <button className={styles.actionBtn}>View Traces</button>
                <button className={styles.actionBtn}>Export Report</button>
              </div>
            </div>
          </div>
        );
      }
      case 'event': {
        const event = state.data as EventMessage;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Event Details</h4>
              <div className={styles.eventDetail}>
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Type</span>
                  <span className={styles.eventValue}>{event.type}</span>
                </div>
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Source</span>
                  <span className={styles.eventValue}>{event.source}</span>
                </div>
                {event.target && (
                  <div className={styles.eventRow}>
                    <span className={styles.eventLabel}>Target</span>
                    <span className={styles.eventValue}>{event.target}</span>
                  </div>
                )}
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Time</span>
                  <span className={styles.eventValue}>{new Date(event.timestamp).toLocaleString()}</span>
                </div>
              </div>
            </div>
            <div className={styles.detailSection}>
              <h4>Content</h4>
              <pre className={styles.eventContent}>{event.content}</pre>
            </div>
          </div>
        );
      }
      default:
        return <div>Unknown detail type</div>;
    }
  };

  return (
    <div className={styles.detailPanel}>
      <div className={styles.detailHeader}>
        <h3 className={styles.detailTitle}>
          {state.type === 'agent' && `Agent: ${state.id}`}
          {state.type === 'date' && `Date: ${state.id}`}
          {state.type === 'event' && `Event: ${state.id.slice(0, 8)}`}
        </h3>
        <button className={styles.detailClose} onClick={onClose}>✕</button>
      </div>
      {renderContent()}
    </div>
  );
};

export default DetailPanel;
