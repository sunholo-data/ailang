import React, { useState, useEffect, useMemo, useCallback } from 'react';

// Benchmark Gallery (M-EVAL gallery redo). Two levels:
//  - Index: a filterable grid of compact benchmark cards.
//  - Detail (deep-linked via #bench-<id>): a "spec sheet" — the task prompt +
//    expected output, pass rates by language / model / harness, and a language-tab
//    solution browser. The ELO difficulty table links straight to a detail via the
//    same hash. Data comes from latest.json: benchmarks[id] (taskPrompt,
//    expectedStdout, languageStats, modelStats, agentStats, codeSamples) + ratings
//    (per-benchmark difficulty ELO/band).

const LANGS = ['ailang', 'python', 'javascript', 'go'];
const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_COLOR = { ailang: '#1D9E75', python: '#6b7280', javascript: '#d99a1c', go: '#2f80ce' };
const TIER_ORDER = ['smoke', 'core', 'stretch', 'frontier', 'vision'];
const BAND_COLOR = { Trivial: '#6b7280', Easy: '#639922', Moderate: '#BA7517', Hard: '#D85A30', 'Very hard': '#A32D2D' };
const bandColor = (b) => BAND_COLOR[b] || 'var(--ifm-color-emphasis-500)';

function isLocalAgent(id) {
  return /qwen3|gemma4/i.test(id || '') && !String(id).includes('-or-');
}

function modelShort(id) {
  if (isLocalAgent(id)) {
    let h = 'agent';
    if (id.startsWith('motoko-')) h = 'motoko';
    else if (id.startsWith('opencode-')) h = 'opencode';
    else if (id.startsWith('pi-')) h = 'Pi';
    const qm = /qwen3-(\d+)/.exec(id);
    const gm = /gemma4-(\d+)/.exec(id);
    return `${qm ? `Qwen3.${qm[1]}` : gm ? `Gemma4.${gm[1]}` : 'local'} · ${h}`;
  }
  return id
    .replace('claude-', 'Claude ')
    .replace('gemini-', 'Gemini ')
    .replace('opencode-or-', 'OC/')
    .replace('opencode-', 'OC/')
    .replace('gpt5', 'GPT-5')
    .replace(/-/g, ' ');
}

function firstLines(text, n) {
  if (!text) return '';
  return String(text).split('\n').filter((l) => l.trim()).slice(0, n).join(' ');
}

function hashId() {
  if (typeof window === 'undefined') return null;
  const m = /^#bench-(.+)$/.exec(window.location.hash || '');
  return m ? decodeURIComponent(m[1]) : null;
}

export default function BenchmarkGallery({ benchmarks, ratings }) {
  const [selected, setSelected] = useState(hashId());
  useEffect(() => {
    const onHash = () => setSelected(hashId());
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);
  const openDetail = useCallback((id) => { window.location.hash = `#bench-${encodeURIComponent(id)}`; }, []);
  const backToIndex = useCallback(() => {
    if (typeof window !== 'undefined') {
      window.history.pushState('', document.title, window.location.pathname + window.location.search);
      setSelected(null);
    }
  }, []);

  // Per-benchmark difficulty (ELO + band) from the ratings block; prefer standard,
  // fall back to agent so agent-only benchmarks still show a difficulty.
  const difficulty = useMemo(() => {
    const m = {};
    ['standard', 'agent'].forEach((mode) => {
      (ratings?.[mode]?.benchmarks || []).forEach((b) => {
        if (m[b.id] == null) m[b.id] = { elo: b.elo, band: b.band, mode };
      });
    });
    return m;
  }, [ratings]);

  const list = useMemo(
    () => Object.entries(benchmarks || {}).map(([id, b]) => ({ id, ...b, difficulty: difficulty[id] })),
    [benchmarks, difficulty],
  );

  if (selected && benchmarks && benchmarks[selected]) {
    return <Detail id={selected} b={benchmarks[selected]} difficulty={difficulty[selected]} onBack={backToIndex} />;
  }
  return <Index list={list} onOpen={openDetail} />;
}

// ---------------------------------------------------------------- Index

function Index({ list, onOpen }) {
  const [tier, setTier] = useState(null);
  const [tag, setTag] = useState(null);
  const [q, setQ] = useState('');
  const [sortBy, setSortBy] = useState('difficulty');

  const tiers = useMemo(
    () => [...new Set(list.map((x) => x.tier).filter(Boolean))].sort((a, b) => TIER_ORDER.indexOf(a) - TIER_ORDER.indexOf(b)),
    [list],
  );
  const tags = useMemo(() => [...new Set(list.flatMap((x) => x.tags || []))].sort(), [list]);

  const rows = useMemo(() => {
    const ql = q.trim().toLowerCase();
    const r = list.filter((x) =>
      (!tier || x.tier === tier) &&
      (!tag || (x.tags || []).includes(tag)) &&
      (!ql || x.id.toLowerCase().includes(ql) || (x.taskPrompt || '').toLowerCase().includes(ql)),
    );
    r.sort((a, b) => {
      if (sortBy === 'difficulty') return (b.difficulty?.elo || 0) - (a.difficulty?.elo || 0);
      if (sortBy === 'passRate') return (b.successRate || 0) - (a.successRate || 0);
      return a.id.localeCompare(b.id);
    });
    return r;
  }, [list, tier, tag, q, sortBy]);

  // Explicit margins (not flex gap) space the chips — some Docusaurus builds drop
  // inline flex-gap, which bunched every chip into one blob.
  const chip = (active) => ({
    display: 'inline-block', verticalAlign: 'middle', margin: '0 6px 6px 0',
    fontSize: 12, padding: '3px 11px', borderRadius: 20, cursor: 'pointer', whiteSpace: 'nowrap',
    border: `1px solid ${active ? 'var(--ifm-color-primary)' : 'var(--ifm-color-emphasis-300)'}`,
    background: active ? 'var(--ifm-color-primary)' : 'transparent',
    color: active ? '#fff' : 'var(--ifm-color-emphasis-800)',
  });

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap', marginBottom: 10 }}>
        <input
          type="text" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search benchmarks…"
          style={{ flex: '1 1 200px', minWidth: 160, padding: '6px 10px', borderRadius: 6, border: '1px solid var(--ifm-color-emphasis-300)', background: 'var(--ifm-background-color)', color: 'var(--ifm-font-color-base)' }}
        />
        <select value={sortBy} onChange={(e) => setSortBy(e.target.value)}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid var(--ifm-color-emphasis-300)', background: 'var(--ifm-background-color)', color: 'var(--ifm-font-color-base)' }}>
          <option value="difficulty">Hardest first</option>
          <option value="passRate">Highest pass rate</option>
          <option value="name">Name (A–Z)</option>
        </select>
      </div>
      <div style={{ marginBottom: 6, lineHeight: 2.2 }}>
        <span style={{ fontSize: 12, color: 'var(--ifm-color-emphasis-600)', marginRight: 8, verticalAlign: 'middle' }}>Tier</span>
        <button style={chip(!tier)} onClick={() => setTier(null)}>All</button>
        {tiers.map((t) => <button key={t} style={chip(tier === t)} onClick={() => setTier(tier === t ? null : t)}>{t}</button>)}
      </div>
      {tags.length > 0 && (
        <div style={{ marginBottom: 14, lineHeight: 2.2 }}>
          <span style={{ fontSize: 12, color: 'var(--ifm-color-emphasis-600)', marginRight: 8, verticalAlign: 'middle' }}>Tag</span>
          <button style={chip(!tag)} onClick={() => setTag(null)}>All</button>
          {tags.map((t) => <button key={t} style={chip(tag === t)} onClick={() => setTag(tag === t ? null : t)}>{t}</button>)}
        </div>
      )}
      <p style={{ fontSize: 13, color: 'var(--ifm-color-emphasis-600)', margin: '0 0 10px' }}>
        {rows.length} benchmark{rows.length === 1 ? '' : 's'} · click any card for the task, pass rates, and solutions
      </p>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))', gap: 12 }}>
        {rows.map((x) => <Card key={x.id} x={x} onOpen={onOpen} />)}
      </div>
    </div>
  );
}

function Card({ x, onOpen }) {
  const a = x.languageStats?.ailang?.successRate;
  const p = x.languageStats?.python?.successRate;
  return (
    <button
      onClick={() => onOpen(x.id)}
      style={{
        textAlign: 'left', cursor: 'pointer', font: 'inherit', color: 'inherit',
        background: 'var(--ifm-background-surface-color)', border: '1px solid var(--ifm-color-emphasis-200)',
        borderRadius: 12, padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 8,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <code style={{ fontSize: 13, fontWeight: 600, wordBreak: 'break-word' }}>{x.id}</code>
        {x.difficulty && (
          <span style={{ marginLeft: 'auto', flexShrink: 0, fontSize: 11, padding: '1px 8px', borderRadius: 20, background: `${bandColor(x.difficulty.band)}22`, color: bandColor(x.difficulty.band) }}>
            {x.difficulty.band}
          </span>
        )}
      </div>
      {x.taskPrompt && (
        <div style={{ fontSize: 12, color: 'var(--ifm-color-emphasis-600)', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
          {firstLines(x.taskPrompt, 2)}
        </div>
      )}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        <MiniBar label="AILANG" rate={a} />
        <MiniBar label="Python" rate={p} />
      </div>
      {(x.tags || []).length > 0 && (
        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
          {x.tags.map((t) => (
            <span key={t} style={{ fontSize: 10.5, padding: '1px 6px', borderRadius: 4, background: 'var(--ifm-color-emphasis-100)', color: 'var(--ifm-color-emphasis-700)' }}>{t}</span>
          ))}
        </div>
      )}
    </button>
  );
}

function MiniBar({ label, rate }) {
  if (typeof rate !== 'number') return null;
  const pct = Math.round(rate * 100);
  const color = pct >= 70 ? '#1D9E75' : pct >= 50 ? '#BA7517' : '#A32D2D';
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <span style={{ width: 50, fontSize: 11, color: 'var(--ifm-color-emphasis-600)' }}>{label}</span>
      <div style={{ flex: 1, height: 6, background: 'var(--ifm-color-emphasis-200)', borderRadius: 4, overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color }} />
      </div>
      <span style={{ width: 30, textAlign: 'right', fontSize: 11, fontWeight: 600 }}>{pct}</span>
    </div>
  );
}

// ---------------------------------------------------------------- Detail

function Detail({ id, b, difficulty, onBack }) {
  return (
    <div>
      <button onClick={onBack} style={{ font: 'inherit', cursor: 'pointer', background: 'none', border: 'none', color: 'var(--ifm-color-primary)', padding: 0, marginBottom: 12 }}>
        ← All benchmarks
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginBottom: 4 }}>
        <code style={{ fontSize: 18, fontWeight: 600 }}>{id}</code>
        {difficulty && (
          <span style={{ fontSize: 12, padding: '2px 10px', borderRadius: 20, background: `${bandColor(difficulty.band)}22`, color: bandColor(difficulty.band) }}>
            {difficulty.band} · ELO {Math.round(difficulty.elo)}
          </span>
        )}
        {b.tier && <span style={{ fontSize: 12, padding: '2px 9px', borderRadius: 6, background: 'var(--ifm-color-emphasis-100)', color: 'var(--ifm-color-emphasis-700)' }}>{b.tier}</span>}
      </div>
      {(b.tags || []).length > 0 && (
        <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap', marginBottom: 16 }}>
          {b.tags.map((t) => <span key={t} style={{ fontSize: 11, padding: '1px 7px', borderRadius: 4, background: 'var(--ifm-color-emphasis-100)', color: 'var(--ifm-color-emphasis-700)' }}>{t}</span>)}
        </div>
      )}

      <Spec prompt={b.taskPrompt} expected={b.expectedStdout} />
      <LangStats languageStats={b.languageStats} totalRuns={b.totalRuns} />
      <ModelStrip modelStats={b.modelStats} />
      <HarnessRow agentStats={b.agentStats} />
      <Solutions codeSamples={b.codeSamples} />
    </div>
  );
}

function Panel({ title, note, children }) {
  return (
    <div style={{ marginBottom: 18 }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
        <h4 style={{ margin: 0, fontSize: 15 }}>{title}</h4>
        {note && <span style={{ fontSize: 12, color: 'var(--ifm-color-emphasis-500)' }}>{note}</span>}
      </div>
      {children}
    </div>
  );
}

function CodeBlock({ text, minor }) {
  return (
    <pre style={{ margin: 0, padding: 12, borderRadius: 8, overflow: 'auto', background: 'var(--ifm-color-emphasis-100)', border: '1px solid var(--ifm-color-emphasis-200)' }}>
      <code style={{ fontFamily: 'var(--ifm-font-family-monospace)', fontSize: 12.5, lineHeight: 1.6, color: minor ? 'var(--ifm-color-emphasis-700)' : 'var(--ifm-font-color-base)', whiteSpace: 'pre-wrap' }}>{text}</code>
    </pre>
  );
}

function Spec({ prompt, expected }) {
  if (!prompt && !expected) return null;
  return (
    <Panel title="The task" note="from the benchmark spec">
      {prompt && <CodeBlock text={prompt.trim()} />}
      {expected && (
        <div style={{ marginTop: 8 }}>
          <div style={{ fontSize: 12, color: 'var(--ifm-color-emphasis-600)', marginBottom: 4 }}>Expected output</div>
          <CodeBlock text={expected.trim()} minor />
        </div>
      )}
    </Panel>
  );
}

function LangStats({ languageStats, totalRuns }) {
  const langs = LANGS.filter((l) => languageStats?.[l]);
  if (langs.length === 0) return null;
  return (
    <Panel title="Pass rate by language" note={totalRuns ? `${totalRuns} runs` : null}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
        {langs.map((l) => {
          const s = languageStats[l];
          const pct = Math.round((s.successRate || 0) * 100);
          return (
            <div key={l} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ width: 76, fontSize: 13, color: 'var(--ifm-color-emphasis-700)' }}>{LANG_LABEL[l]}</span>
              <div style={{ flex: 1, height: 20, background: 'var(--ifm-color-emphasis-200)', borderRadius: 6, overflow: 'hidden' }}>
                <div style={{ width: `${pct}%`, height: '100%', background: LANG_COLOR[l] }} />
              </div>
              <span style={{ width: 42, textAlign: 'right', fontSize: 13, fontWeight: 600 }}>{pct}%</span>
              {s.avgTokens ? <span style={{ width: 66, textAlign: 'right', fontSize: 12, color: 'var(--ifm-color-emphasis-500)' }}>{Math.round(s.avgTokens)} tok</span> : <span style={{ width: 66 }} />}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function ModelStrip({ modelStats }) {
  const models = useMemo(() => {
    if (!modelStats) return [];
    return Object.entries(modelStats)
      .map(([id, langs]) => ({ id, ailang: langs.ailang?.passRate, python: langs.python?.passRate, local: isLocalAgent(id) }))
      .sort((a, b) => (b.ailang ?? -1) - (a.ailang ?? -1));
  }, [modelStats]);
  if (models.length === 0) return null;
  const dot = (rate) => {
    if (typeof rate !== 'number') return <span style={{ color: 'var(--ifm-color-emphasis-400)' }}>—</span>;
    const pct = Math.round(rate * 100);
    const color = pct >= 70 ? '#1D9E75' : pct >= 40 ? '#BA7517' : '#A32D2D';
    return <span style={{ color, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{pct}%</span>;
  };
  return (
    <Panel title="By model" note={`${models.length} models · AILANG vs Python on this task`}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(210px, 1fr))', gap: '2px 16px' }}>
        {models.map((m) => (
          <div key={m.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '3px 6px', borderRadius: 6, background: m.local ? 'rgba(8,145,178,0.08)' : undefined, boxShadow: m.local ? 'inset 2px 0 0 #0891b2' : undefined }}>
            <span style={{ flex: 1, fontSize: 12.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={m.id}>{modelShort(m.id)}</span>
            <span style={{ width: 38, textAlign: 'right', fontSize: 12 }}>{dot(m.ailang)}</span>
            <span style={{ width: 38, textAlign: 'right', fontSize: 12 }}>{dot(m.python)}</span>
          </div>
        ))}
      </div>
      <p style={{ fontSize: 11.5, color: 'var(--ifm-color-emphasis-500)', margin: '6px 0 0' }}>
        Two columns: AILANG · Python pass rate. <span style={{ color: '#0891b2' }}>Cyan</span> = on-device GPU agent.
      </p>
    </Panel>
  );
}

function HarnessRow({ agentStats }) {
  const byHarness = agentStats?.ailang?.byHarness;
  if (!byHarness || Object.keys(byHarness).length === 0) return null;
  const rows = Object.entries(byHarness)
    .map(([h, s]) => ({ h, rate: s.successRate, runs: s.runs }))
    .sort((a, b) => (b.rate ?? -1) - (a.rate ?? -1));
  return (
    <Panel title="By harness" note="agent mode · AILANG">
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {rows.map((r) => {
          const pct = Math.round((r.rate || 0) * 100);
          const local = ['motoko', 'opencode', 'pi'].includes(r.h);
          return (
            <span key={r.h} title={`${r.runs} runs`} style={{ fontSize: 12, padding: '3px 10px', borderRadius: 20, border: `1px solid ${local ? '#0891b2' : 'var(--ifm-color-emphasis-300)'}`, background: local ? 'rgba(8,145,178,0.08)' : 'var(--ifm-background-surface-color)', color: local ? '#0e7490' : 'var(--ifm-font-color-base)' }}>
              {r.h} · {pct}%
            </span>
          );
        })}
      </div>
    </Panel>
  );
}

function Solutions({ codeSamples }) {
  const langs = LANGS.filter((l) => codeSamples?.[l]);
  const [active, setActive] = useState(langs[0]);
  useEffect(() => { if (langs.length && !langs.includes(active)) setActive(langs[0]); }, [langs, active]);
  if (langs.length === 0) return null;
  return (
    <Panel title="Sample solutions" note="one representative generated solution per language">
      <div style={{ display: 'flex', gap: 2, marginBottom: -1 }}>
        {langs.map((l) => (
          <button
            key={l} onClick={() => setActive(l)}
            style={{
              font: 'inherit', cursor: 'pointer', fontSize: 12.5, padding: '6px 12px',
              border: '1px solid var(--ifm-color-emphasis-200)',
              borderRadius: '8px 8px 0 0', marginRight: 2,
              background: active === l ? 'var(--ifm-color-emphasis-100)' : 'transparent',
              color: active === l ? 'var(--ifm-font-color-base)' : 'var(--ifm-color-emphasis-600)',
              fontWeight: active === l ? 600 : 400,
            }}
          >
            {LANG_LABEL[l]}
          </button>
        ))}
      </div>
      <CodeBlock text={codeSamples[active]} />
    </Panel>
  );
}
