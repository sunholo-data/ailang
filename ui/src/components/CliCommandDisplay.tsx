/**
 * CliCommandDisplay - Shows equivalent ailang CLI command for current view
 *
 * Displays the CLI command that produces the same data as the current
 * dashboard visualization, enabling easy verification and debugging.
 */
import React, { useState, useCallback } from 'react';
import styles from './CliCommandDisplay.module.css';

export interface CliCommandDisplayProps {
  /** The CLI command to display */
  command: string;
  /** Optional label for the command context */
  label?: string;
  /** Whether to start collapsed (default: true) */
  defaultCollapsed?: boolean;
  /** Compact mode for inline display */
  compact?: boolean;
}

export const CliCommandDisplay: React.FC<CliCommandDisplayProps> = ({
  command,
  label = 'CLI Command',
  defaultCollapsed = true,
  compact = false,
}) => {
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy command:', err);
    }
  }, [command]);

  if (compact) {
    return (
      <div className={styles.compactContainer}>
        <code className={styles.compactCommand}>{command}</code>
        <button
          className={styles.copyBtn}
          onClick={handleCopy}
          title={copied ? 'Copied!' : 'Copy to clipboard'}
        >
          {copied ? '✓' : '⎘'}
        </button>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <button
        className={styles.header}
        onClick={() => setIsCollapsed(!isCollapsed)}
        aria-expanded={!isCollapsed}
      >
        <span className={styles.icon}>{isCollapsed ? '▸' : '▾'}</span>
        <span className={styles.label}>{label}</span>
        <span className={styles.preview}>
          {isCollapsed && <code className={styles.previewCode}>{command.substring(0, 40)}...</code>}
        </span>
      </button>

      {!isCollapsed && (
        <div className={styles.body}>
          <div className={styles.commandWrapper}>
            <code className={styles.command}>{command}</code>
            <button
              className={`${styles.copyBtn} ${styles.copyBtnLarge}`}
              onClick={handleCopy}
              title={copied ? 'Copied!' : 'Copy to clipboard'}
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <p className={styles.hint}>
            Run this command in terminal to get the same data as JSON
          </p>
        </div>
      )}
    </div>
  );
};

export default CliCommandDisplay;
