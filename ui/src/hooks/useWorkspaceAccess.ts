/**
 * useWorkspaceAccess - Hook for fetching accessible workspaces and their roles
 *
 * Returns list of workspaces the current user can access:
 * - For authenticated users: public workspaces + granted workspaces with roles
 * - For unauthenticated users: only public workspaces
 */
import { useState, useEffect, useCallback } from 'react';

export interface AccessibleWorkspace {
  id: string;
  name?: string;
  role: string; // "Viewer" | "Approver" | "" (for public)
  is_public: boolean;
}

interface UseWorkspaceAccessResult {
  workspaces: AccessibleWorkspace[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
  // Helper to get role for a workspace
  getRole: (workspaceId: string) => string | undefined;
  // Helper to check if user has access
  hasAccess: (workspaceId: string) => boolean;
}

export function useWorkspaceAccess(): UseWorkspaceAccessResult {
  const [workspaces, setWorkspaces] = useState<AccessibleWorkspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkspaces = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/workspaces');

      if (!response.ok) {
        if (response.status === 403) {
          setError('Access denied');
          setWorkspaces([]);
          return;
        }
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Handle both old format (string[]) and new format (AccessibleWorkspace[])
      if (Array.isArray(data)) {
        if (data.length === 0) {
          setWorkspaces([]);
        } else if (typeof data[0] === 'string') {
          // Old format: convert to new format (assume public access)
          setWorkspaces(
            data.map((id: string) => ({
              id,
              name: id,
              role: '',
              is_public: true,
            }))
          );
        } else {
          // New format
          setWorkspaces(data);
        }
      } else {
        setWorkspaces([]);
      }
    } catch (err) {
      console.error('Failed to fetch workspaces:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
      setWorkspaces([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWorkspaces();
  }, [fetchWorkspaces]);

  const getRole = useCallback(
    (workspaceId: string): string | undefined => {
      const workspace = workspaces.find((w) => w.id === workspaceId);
      return workspace?.role;
    },
    [workspaces]
  );

  const hasAccess = useCallback(
    (workspaceId: string): boolean => {
      // If no workspaces loaded, allow all (permissive fallback)
      if (workspaces.length === 0) return true;
      return workspaces.some((w) => w.id === workspaceId);
    },
    [workspaces]
  );

  return {
    workspaces,
    loading,
    error,
    refresh: fetchWorkspaces,
    getRole,
    hasAccess,
  };
}

export default useWorkspaceAccess;
