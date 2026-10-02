### Added — `--ai-stub-fixtures <file>`: the AI stub replays recorded responses (#1497)

- `ailang run --ai-stub --ai-stub-fixtures fixtures.json` replays provider-shaped responses (text, `step`
  tool calls, images as base64, and typed `AIError`s) keyed by the sha256 of the canonical request: the
  prompt for single-shot and image calls, the last message's content for `step`. Entries name their request
  literally (`"prompt"`) or by `"sha256"`.
- A request with no fixture is an error whose message starts `E_AI_STUB_FIXTURE_MISS` and names the missing
  sha256; it never falls back to the default `{"kind":"Wait"}`. The file is validated at startup, and the
  flag without `--ai-stub` is refused. Format: the ai-effect guide.
