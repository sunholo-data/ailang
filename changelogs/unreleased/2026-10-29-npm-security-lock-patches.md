### Fixed — dashboard and documentation security dependencies (2026-10-29)

Update the dashboard lockfile to re2js 2.8.7 for [GHSA-gg6h-894w-4m9c](https://github.com/advisories/GHSA-gg6h-894w-4m9c), and the documentation lockfile to serialize-javascript 7.1.3 for [GHSA-m5mm-p8h7-m29v](https://github.com/advisories/GHSA-m5mm-p8h7-m29v), within their existing allowed ranges. re2js now requires Node 20 or later; the dashboard build uses Node 22.23.4. Dependency declarations and overrides are unchanged.
