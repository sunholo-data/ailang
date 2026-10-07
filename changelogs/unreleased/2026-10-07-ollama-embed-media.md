### Added — `_ollama_embed_media`: image and audio embeddings via Ollama

- `_ollama_embed_media(model, text, image, audio) -> list[float] ! {IO}` sends one multimodal input
  (`{text, image, audio}`) to Ollama's embed API, for models with the vision or audio capability such
  as `embeddinggemma-2` (needs Ollama >= 0.36). Empty text or empty bytes means that part is absent;
  all three empty is an error. Image and audio are raw bytes (`std/fs.readFileRaw`); the builtin
  base64-encodes them.
- The vector shares the model's text space, so it compares directly with `_ollama_embed` output from the
  same model: an image can be searched with a text query.
- `_ollama_embed(model, text)` is unchanged. Media calls get a 120 s timeout (text stays at 30 s) because
  vision/audio models load slower. Requested by email-parse (PR #35, resumable re-embed onto a new model).
