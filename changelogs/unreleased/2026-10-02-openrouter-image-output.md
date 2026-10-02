### Added — OpenRouter image output and a per-call model for `callImage*` (#1500, #1496) (2026-10-02)

`AI.callImage` and `AI.callImageBase64` now work on an OpenRouter-bound handler.
The adapter no longer refuses image requests. It sends the normal chat
completion with `"modalities": ["image","text"]` and `image_config`
(`aspect_ratio`, and `mime_type` mapped to `output_format`), then decodes the
first base64 data-URL in `choices[0].message.images`. That gives image output
from models such as `google/gemini-3-pro-image` and `openai/gpt-5-image` on
the same key, with the cost in the same `usage.cost` figure as text calls.
These cases are typed errors that name the model: no image returned, a remote
image URL, a malformed data URL, and an unsupported `mime_type`. Text request
bodies are byte-identical to before. OpenRouter `Step`/`StreamStep` now reject
image output with an error instead of returning text.

The image options JSON accepts a `"model"` key that applies to that call
only. It is resolved through the same `ModelResolver` as `step()`: friendly
`models.yml` names resolve to api_names, and a model on another provider
fails with `ModelNotAllowed`. Without the key, the bound `--ai` model is used
as before. Design: `design_docs/implemented/v0_51_1/m-openrouter-image-output.md`.

### Added — reference-conditioned image generation: `callImageWithRefs` / `callImageBase64WithRefs` (#1496) (2026-10-02)

New `std/ai` functions take reference images as well as the prompt, for edits
and for keeping one character the same across generations:
`callImageWithRefs(prompt, refs: [ImagePart], output_path, options) -> string ! {AI, FS}`
and `callImageBase64WithRefs(prompt, refs: [ImagePart], options) -> string ! {AI}`.
The refs use the same `ImagePart` record as `step()` vision input (base64 or
data-URI `source`, plus `mime`). Gemini sends them as `inlineData` parts and
OpenRouter as `image_url` content parts. Anthropic, OpenAI-direct, Ollama and
config-driven providers, and any handler without the capability, fail with
"not supported" rather than dropping the references. `--ai-stub` returns the
fixed 1x1 PNG for both functions.
