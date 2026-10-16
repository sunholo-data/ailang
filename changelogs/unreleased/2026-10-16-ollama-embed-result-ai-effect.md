### Added
- `std/embedding` now exports `ollamaEmbed`, `ollamaEmbedMedia`, `ollamaEmbedResult` and `ollamaEmbedMediaResult`. Result variants return `Result[[float], string]` so programs can recover from Ollama outages or missing models.
- Embeddings support deterministic eight-dimensional `--ai-stub` vectors and version-1 `embed` fixtures for offline success/error replay.

### Changed
- **Breaking:** Ollama embedding builtins require AI instead of IO. Add AI to caller effect rows and capability grants. Every call charges one AI budget unit and respects operator ceilings; authorization failures remain runtime errors. Embeddings still call local Ollama independently of the chat provider and need no `--ai` model.
