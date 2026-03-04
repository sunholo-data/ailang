/**
 * Hook to track document visibility state.
 * Returns `true` when the tab is visible, `false` when hidden.
 * Use this to pause polling when the user isn't looking at the tab.
 *
 * M-PERF-OBSERVATORY Phase 4.1: Visibility-aware polling
 */
import { useState, useEffect } from 'react';

export function useDocumentVisibility(): boolean {
  const [isVisible, setIsVisible] = useState(() => !document.hidden);

  useEffect(() => {
    const handler = () => setIsVisible(!document.hidden);
    document.addEventListener('visibilitychange', handler);
    return () => document.removeEventListener('visibilitychange', handler);
  }, []);

  return isVisible;
}
