/**
 * useBudgetStatus - Hook to fetch budget status from AILANG bridge
 * Demonstrates AILANG dogfooding: contracts (requires) and effect budgets
 */
import { useState, useEffect, useMemo } from 'react';
import { ControlPlaneFilters, buildFilterQueryString } from '../features/controlplane/types/filters';

interface BudgetConfig {
  workspaceBudget: number;
  dailyBudget: number;
  taskMaxCost: number;
  warningThreshold: number;
  providerBudgets?: Record<string, ProviderBudget>;
}

interface ProviderBudget {
  dailyBudget: number;
  taskMaxCost: number;
  hardLimit: boolean;
  warningThreshold: number;
}

interface BudgetStatusData {
  allowed: boolean;
  remainingWorkspace: number;
  remainingDaily: number;
  warningLevel: 'ok' | 'warning' | 'critical' | 'exceeded';
  message: string;
}

interface BudgetUsage {
  workspaceSpend: number;
  dailySpend: number;
  usagePercent: number;
}

interface ProviderUsage {
  spend: number;
  budget: number;
  usagePercent: number;
  warningLevel: 'ok' | 'warning' | 'critical' | 'exceeded';
  hardLimit: boolean;
}

export interface BurnRateInfo {
  costPerHour: number;
  hoursUntilExhaustion: number; // -1 if no burn rate
  windowHours: number;
}

interface BudgetResponse {
  config: BudgetConfig;
  status: BudgetStatusData;
  usage: BudgetUsage;
  burnRate: BurnRateInfo;
  byProvider?: Record<string, ProviderUsage>;
  usingAilang: boolean;
}

interface UseBudgetStatusResult {
  budget: BudgetResponse | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
}

export function useBudgetStatus(
  refreshInterval = 120000,
  filters?: ControlPlaneFilters
): UseBudgetStatusResult {
  const [budget, setBudget] = useState<BudgetResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  // Build query string from filters (memoized to prevent unnecessary re-renders)
  const queryString = useMemo(() => {
    if (!filters) return '';
    // Only include budget-relevant filters (not status, sort, search)
    const budgetFilters: ControlPlaneFilters = {
      provider: filters.provider,
      workspace: filters.workspace,
      model: filters.model,
      start_date: filters.start_date,
      end_date: filters.end_date,
    };
    return buildFilterQueryString(budgetFilters);
  }, [filters?.provider, filters?.workspace, filters?.model, filters?.start_date, filters?.end_date]);

  const fetchBudget = async () => {
    try {
      const response = await fetch(`/api/budget/status${queryString}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data = await response.json();
      setBudget(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBudget();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchBudget, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [refreshInterval, queryString]);

  return { budget, loading, error, refetch: fetchBudget };
}

export default useBudgetStatus;
