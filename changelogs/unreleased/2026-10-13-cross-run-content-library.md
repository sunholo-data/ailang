### Cross-run content libraries

- Add opt-in SQLite SharedMem and namespace-partitioned SharedIndex stores via `--sharedmem-db` / `AILANG_SHAREDMEM_DB` and `--sharedindex-db` / `AILANG_SHAREDINDEX_DB`, with capability checks and owned cleanup.
- Add `std/ai.embed(text)` with typed AI errors, configured Gemini/OpenAI/Ollama providers, explicit offline stub, in-memory content-hash caching, and optional `--embed-cache-db` / `AILANG_EMBED_CACHE_DB` persistence.
- Demonstrate a two-process content library and correct the semantic caching guide's default-persistence claim. ANN and read-only capabilities remain deferred.
