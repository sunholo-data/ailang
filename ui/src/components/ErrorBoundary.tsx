import React from 'react';

interface ErrorBoundaryProps {
  children: React.ReactNode;
  /** Optional fallback UI. Defaults to a simple error message. */
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Catches render errors in child components and displays a fallback UI
 * instead of crashing the entire dashboard.
 *
 * Wrap around sections that may receive malformed data (JSON payloads,
 * WebSocket events, API responses rendered inline).
 */
export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('[ErrorBoundary] Caught render error:', error.message, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }
      return (
        <div style={{ padding: '12px', color: '#e57373', background: '#2a1a1a', borderRadius: '6px', margin: '8px 0', fontSize: '13px' }}>
          <strong>Render error:</strong> {this.state.error?.message || 'Unknown error'}
          <br />
          <button
            onClick={() => this.setState({ hasError: false, error: null })}
            style={{ marginTop: '8px', padding: '4px 12px', cursor: 'pointer', background: '#333', color: '#ccc', border: '1px solid #555', borderRadius: '4px' }}
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
