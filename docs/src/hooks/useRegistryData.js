import { useState, useEffect, useCallback } from 'react';

const REGISTRY_API = typeof window !== 'undefined'
  ? (window.__AILANG_REGISTRY_API || 'https://ailang-registry-validator-mdpoxgrptq-ew.a.run.app')
  : '';

const CACHE_PREFIX = 'ailang_registry_';
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

function getCached(key) {
  try {
    const raw = localStorage.getItem(CACHE_PREFIX + key);
    if (!raw) return null;
    const { data, ts } = JSON.parse(raw);
    if (Date.now() - ts > CACHE_TTL) return null;
    return data;
  } catch {
    return null;
  }
}

function setCache(key, data) {
  try {
    localStorage.setItem(CACHE_PREFIX + key, JSON.stringify({ data, ts: Date.now() }));
  } catch {
    // localStorage full or unavailable
  }
}

/**
 * Generic registry data hook — tries live API, falls back to static snapshot,
 * then localStorage cache. Implements stale-while-revalidate.
 */
export function useRegistryData(path, staticFallback = null) {
  const [data, setData] = useState(() => getCached(path) || staticFallback);
  const [loading, setLoading] = useState(!data);
  const [error, setError] = useState(null);
  const [stale, setStale] = useState(!!getCached(path));

  useEffect(() => {
    if (!REGISTRY_API) return;

    let cancelled = false;
    const url = `${REGISTRY_API}${path}`;

    fetch(url, { signal: AbortSignal.timeout(8000) })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((freshData) => {
        if (cancelled) return;
        setData(freshData);
        setStale(false);
        setLoading(false);
        setCache(path, freshData);
      })
      .catch((err) => {
        if (cancelled) return;
        // Try static fallback
        if (!data && staticFallback) {
          setData(staticFallback);
        } else if (!data) {
          // Try fetching from static build snapshot
          fetch(`/registry${path.replace('/api', '')}`)
            .then(r => r.ok ? r.json() : Promise.reject())
            .then(staticData => {
              if (!cancelled) {
                setData(staticData);
                setStale(true);
              }
            })
            .catch(() => {
              if (!cancelled) setError(err.message);
            });
        } else {
          setStale(true);
        }
        setLoading(false);
      });

    return () => { cancelled = true; };
  }, [path]);

  return { data, loading, error, stale };
}

/** Fetch the full package index. */
export function usePackageIndex() {
  return useRegistryData('/api/packages');
}

/** Fetch detail for a specific package (all versions + dependents). */
export function usePackageDetail(vendor, name) {
  return useRegistryData(`/api/packages/${vendor}/${name}`);
}

/** Fetch ecosystem statistics. */
export function useEcosystemStats() {
  return useRegistryData('/api/stats');
}
