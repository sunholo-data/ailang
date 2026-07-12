// Runtime-decoupled benchmark data fetch (M-EVAL-DATA-HOSTING-DECOUPLE).
//
// The rig syncs the benchmark JSONs to a private GCS bucket every cycle (~45 min),
// and the dashboard Cloud Run service exposes them at /benchmarks/<path> (public,
// Cache-Control max-age=60). Fetching from there means fresh rig/cloud data shows
// within ~1 min WITHOUT a Docusaurus rebuild + GitHub Pages deploy — the fix for
// the "edited data never shows" / stale-cache churn.
//
// Robust fallback: if the dashboard route is unreachable or returns non-2xx, we fall
// back to the in-build copy baked into the site's static dir (Fastly-served). So the
// page degrades to build-time data instead of hard-failing when Cloud Run is down.
//
// Returns a Response (whichever source answered), so call sites keep their existing
// `.then(r => r.json())` / `.then(r => r.ok ? r.json() : null)` chains unchanged —
// just swap `fetch('/benchmarks/X')` for `benchmarkFetch('X')`.

// Dashboard Cloud Run URL (run.app; swap for a custom domain later if desired).
const DASHBOARD_BASE = 'https://ailang-dev-dashboard-ejjw6zt3bq-ew.a.run.app';

// relPath is relative to the benchmarks root, e.g. 'latest.json' or 'os/history.json'
// (a leading slash or 'benchmarks/' prefix is tolerated and stripped).
export async function benchmarkFetch(relPath, { cacheBust = false, ...opts } = {}) {
  const rel = String(relPath).replace(/^\/+/, '').replace(/^benchmarks\//, '');
  const q = cacheBust ? `?v=${Date.now()}` : '';
  const remote = `${DASHBOARD_BASE}/benchmarks/${rel}${q}`;
  const local = `/benchmarks/${rel}${q}`;
  try {
    const r = await fetch(remote, opts);
    if (r.ok) return r;
  } catch (e) {
    // Network/CORS error reaching the dashboard — fall through to the in-build copy.
  }
  return fetch(local, opts);
}
