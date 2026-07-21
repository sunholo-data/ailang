// Single source of truth for how ON-DEVICE (local) models are identified, named,
// rated and caveated across EVERY benchmark page.
//
// Why this exists (audit, 2026-07-21): local-model data had drifted into five
// different rendered pass rates for the SAME model on the SAME release, six
// different display names, five ad-hoc "is this local?" regexes, and two
// different definitions of "provisional". A reader comparing on-device to cloud
// got a ~10pp swing depending which page they landed on.
//
// Rules, in order of importance:
//   1. RATE — always `agentSuccessRate` from latest.json (runs-based). That is the
//      same accumulator the cloud numbers use, so local-vs-cloud is like-for-like.
//      os/latest.json divides by TRIALS instead and is published on a different
//      cadence, so its rates DRIFT from latest.json. Use os/* for tiers/languages
//      (which latest.json doesn't carry) — never for a headline rate.
//   2. IDENTITY — `provider_type === 'local'`, which is already in the data.
//      Never a regex on the model id.
//   3. COVERAGE — one threshold, from coverageGate. Not a per-component variant.

// Canonical caveat. If you are rendering a local number next to a cloud number,
// this string (or LocalCloudBadge) must appear somewhere the reader can see.
export const LOCAL_CAVEAT =
  'On-device GPU agent: a local model run through an agentic harness — slow, ~$0/run. ' +
  'Agent-mode numbers are not directly comparable to hosted 0-shot models.';

// Identity. `data` is the parsed latest.json. Falls back to a name match ONLY for
// callers working from os/latest.json rows, which carry no provider_type — extend
// the publisher rather than this list if a new local model appears.
const FALLBACK_LOCAL = /^(motoko-local-|pi-|opencode-)/;

export function isLocalModel(id, data) {
  if (!id) return false;
  const entry = data && data.agentModels && data.agentModels[id];
  if (entry && entry.provider_type) return entry.provider_type === 'local';
  const plain = data && data.models && data.models[id];
  if (plain && plain.provider_type) return plain.provider_type === 'local';
  return FALLBACK_LOCAL.test(id);
}

// Display name. One spelling everywhere. Keeps the harness visible, because the
// whole point of the on-device roster is comparing harnesses on identical weights
// — "Qwen3.6" alone is ambiguous across three rows.
const HARNESS_OF = [
  [/^motoko-local-/, 'motoko'],
  [/^motoko-/, 'motoko'],
  [/^opencode-/, 'opencode'],
  [/^pi-/, 'pi'],
];

export function formatLocalName(id) {
  if (!id) return '';
  let harness = null;
  let rest = id;
  for (const [re, h] of HARNESS_OF) {
    if (re.test(id)) {
      harness = h;
      rest = id.replace(re, '');
      break;
    }
  }
  const model = rest
    .replace(/-35b-a3b-mxfp8$/, '')
    .replace(/^qwen3-6$/, 'Qwen3.6')
    .replace(/^qwen3-5$/, 'Qwen3.5');
  return harness ? `${model} · ${harness}` : model;
}

// Canonical rate + its denominator. Returns null when the model has no agent entry,
// so callers can omit the row rather than invent a number.
//
// `adjusted` (success/(runs-apiErrors)) is deliberately NOT the headline: it reads
// as a higher score for a model that errored more, which is exactly backwards when
// sitting next to cloud rows. Exposed so a surface can show it as a secondary note.
export function localRate(data, id, lang = 'ailang') {
  const entry = data && data.agentModels && data.agentModels[id];
  const l = entry && entry.languages && entry.languages[lang];
  if (!l) return null;
  const rate = l.agentSuccessRate != null ? l.agentSuccessRate : l.successRate;
  if (rate == null) return null;
  return {
    rate,
    runs: l.agentRuns != null ? l.agentRuns : l.totalRuns,
    adjusted: l.agentSuccessRateAdjusted,
    apiErrors: l.agentApiErrors || 0,
  };
}

// A local model's cost is ~$0, which breaks any cost-derived ranking: dividing by
// it yields a score orders of magnitude above every cloud model, and awarding it a
// Pareto/efficiency marker silently STRIPS that marker from cloud models that
// earned it. Cost-efficiency rankings must exclude local models rather than let
// them win by dividing by zero.
export function excludeFromCostRanking(id, data) {
  return isLocalModel(id, data);
}
