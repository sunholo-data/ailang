/**
 * Control Plane filter types for interactive filtering
 */

// Status filter values
export type StatusFilter = 'all' | 'running' | 'pending' | 'completed' | 'failed';

// Sort field options
export type SortField = 'timestamp' | 'turns' | 'cost' | 'tokens' | 'duration';

// Sort order options
export type SortOrder = 'asc' | 'desc';

// Filter parameters that can be passed to Control Plane API endpoints
export interface ControlPlaneFilters {
  source_type?: string; // eval, coordinator, direct_api, local, other
  provider?: string;    // claude, gemini, openai, etc.
  model?: string;       // claude-sonnet-4-5, gemini-2-5-pro, etc.
  workspace?: string;   // workspace ID
  start_date?: string;  // YYYY-MM-DD format for time range filter (inclusive)
  end_date?: string;    // YYYY-MM-DD format for time range filter (inclusive)
  status?: StatusFilter; // Filter by task/span status
  search?: string;      // Search query for filtering by name/content
  sort?: SortField;     // Sort by: timestamp, turns, cost, tokens, duration
  order?: SortOrder;    // Sort order: asc, desc
}

// Check if any filters are active
export function hasActiveFilters(filters: ControlPlaneFilters): boolean {
  return !!(
    filters.source_type ||
    filters.provider ||
    filters.model ||
    filters.workspace ||
    filters.start_date ||
    filters.end_date ||
    (filters.status && filters.status !== 'all') ||
    filters.search
  );
}

// Check if a time range filter is active
export function hasTimeRangeFilter(filters: ControlPlaneFilters): boolean {
  return !!(filters.start_date || filters.end_date);
}

// Build query string from filters
export function buildFilterQueryString(filters: ControlPlaneFilters): string {
  const params = new URLSearchParams();
  if (filters.source_type) params.set('source_type', filters.source_type);
  if (filters.provider) params.set('provider', filters.provider);
  if (filters.model) params.set('model', filters.model);
  if (filters.workspace) params.set('workspace', filters.workspace);
  if (filters.start_date) params.set('start_date', filters.start_date);
  if (filters.end_date) params.set('end_date', filters.end_date);
  if (filters.status && filters.status !== 'all') params.set('status', filters.status);
  if (filters.search) params.set('search', filters.search);
  if (filters.sort) params.set('sort', filters.sort);
  if (filters.order) params.set('order', filters.order);
  const queryString = params.toString();
  return queryString ? `?${queryString}` : '';
}

// Format date for display (e.g., "Jan 5, 2026")
function formatDateForDisplay(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00'); // Ensure local timezone
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

// Get human-readable filter description
export function getFilterDescription(filters: ControlPlaneFilters): string {
  const parts: string[] = [];
  if (filters.source_type) {
    const labels: Record<string, string> = {
      eval: 'Eval Benchmarks',
      coordinator: 'Coordinator Tasks',
      direct_api: 'Direct API Calls',
      local: 'Local Usage',
      other: 'Other',
    };
    parts.push(labels[filters.source_type] || filters.source_type);
  }
  if (filters.provider) parts.push(filters.provider);
  if (filters.model) parts.push(filters.model);
  if (filters.workspace) parts.push(`Workspace: ${filters.workspace}`);

  // Add time range description
  if (filters.start_date && filters.end_date) {
    if (filters.start_date === filters.end_date) {
      parts.push(formatDateForDisplay(filters.start_date));
    } else {
      parts.push(`${formatDateForDisplay(filters.start_date)} - ${formatDateForDisplay(filters.end_date)}`);
    }
  } else if (filters.start_date) {
    parts.push(`From ${formatDateForDisplay(filters.start_date)}`);
  } else if (filters.end_date) {
    parts.push(`Until ${formatDateForDisplay(filters.end_date)}`);
  }

  return parts.join(' > ') || 'All Data';
}

// Create filter for a specific date (single day)
export function createDateFilter(date: string): Partial<ControlPlaneFilters> {
  return {
    start_date: date,
    end_date: date,
  };
}

// Create filter for a date range
export function createDateRangeFilter(startDate: string, endDate: string): Partial<ControlPlaneFilters> {
  return {
    start_date: startDate,
    end_date: endDate,
  };
}

// Clear time range from filters while preserving other filters
export function clearTimeRangeFilter(filters: ControlPlaneFilters): ControlPlaneFilters {
  const { start_date, end_date, ...rest } = filters;
  return rest;
}

// Merge filters (new filters override existing ones)
export function mergeFilters(
  existing: ControlPlaneFilters,
  newFilters: Partial<ControlPlaneFilters>
): ControlPlaneFilters {
  return { ...existing, ...newFilters };
}
