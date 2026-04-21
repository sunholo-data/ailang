// Provider-grouped color palettes for the per-model trend charts.
//
// Hue = provider, shade = model index within provider (sorted by name for
// stable assignment). New models inherit a sensible color automatically
// instead of falling through to grey.
//
// Palettes are chosen to be distinguishable at typical line-chart widths
// and to loosely echo each vendor's brand:
//   Anthropic → warm peach / amber
//   OpenAI    → emerald / teal
//   Google    → blue → violet (Gemini gradient)

const PALETTES = {
  anthropic: ['#CC785C', '#E89471', '#F2B79A', '#8A4B35'],
  openai:    ['#10A37F', '#14B8A6', '#5EEAD4', '#047857'],
  google:    ['#4285F4', '#8B5CF6', '#A78BFA', '#1E40AF'],
  other:     ['#6B7280', '#9CA3AF', '#4B5563', '#D1D5DB'],
};

export function getProvider(modelName) {
  const n = (modelName || '').toLowerCase();
  if (n.includes('claude') || n.includes('sonnet') || n.includes('opus') || n.includes('haiku')) {
    return 'anthropic';
  }
  if (n.includes('gpt') || n.includes('openai') || n.includes('o1') || n.includes('o3')) {
    return 'openai';
  }
  if (n.includes('gemini') || n.includes('bard') || n.includes('google')) {
    return 'google';
  }
  return 'other';
}

// assignModelColors takes the set of model names visible on a chart and
// returns a Map<modelName, hex>. Models are grouped by provider and sorted
// alphabetically inside each group so the same model always gets the same
// shade across renders (no flicker when toggling chips).
export function assignModelColors(models) {
  const byProvider = { anthropic: [], openai: [], google: [], other: [] };
  Array.from(models).forEach((m) => {
    byProvider[getProvider(m)].push(m);
  });
  Object.values(byProvider).forEach((arr) => arr.sort());

  const map = new Map();
  Object.entries(byProvider).forEach(([prov, arr]) => {
    const palette = PALETTES[prov];
    arr.forEach((m, i) => {
      map.set(m, palette[i % palette.length]);
    });
  });
  return map;
}
