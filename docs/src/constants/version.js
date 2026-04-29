export const STABLE_RELEASE = 'v0.14.2';

// Teaching prompts are versioned independently from releases — a new prompt
// only ships when teachable surface (syntax, effects, builtins) changes.
// Active prompt is the source of truth in `prompts/versions.json` (`.active`).
// The generator script tools/generate-llms-txt.sh reads from there.
export const ACTIVE_PROMPT = 'v0.12.1';
