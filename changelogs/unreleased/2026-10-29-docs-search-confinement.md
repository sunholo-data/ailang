### Fixed

- Refuse the GitHub docs-search backend under `AILANG_AGENT_POLICY` before credential discovery, cached results, requests or cache writes, including direct/reused Go backends and policy-tool children. Local SimHash search remains available with provisioned documentation; refusal hints direct agents to local docs or the operator. Explicit neural and cache-management modes retain their separately scoped behavior.
