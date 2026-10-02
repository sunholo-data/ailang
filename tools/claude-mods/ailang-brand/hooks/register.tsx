import type { EngineInterface, Register } from 'claude-code'

const PANE = 'ailang-brand'

// Present / past pairs; the engine samples its own word per turn, and we map
// it onto this list by hash so one turn keeps one word.
const WORDS: readonly (readonly [string, string])[] = [
  ['Unifying', 'Unified'],
  ['Inferring effects', 'Inferred effects'],
  ['Elaborating', 'Elaborated'],
  ['Monomorphizing', 'Monomorphized'],
  ['Pattern-matching', 'Pattern-matched'],
  ['Checking contracts', 'Checked contracts'],
  ['Lowering to bytecode', 'Lowered to bytecode'],
  ['Folding', 'Folded'],
  ['Normalizing', 'Normalized'],
  ['Resolving modules', 'Resolved modules'],
]

/** Pure: a stable index for a word. */
export function pick(word: string): readonly [string, string] {
  let h = 0
  for (const c of word) h = (h * 31 + c.charCodeAt(0)) >>> 0
  return WORDS[h % WORDS.length]!
}

let version = ''

async function readVersion($: EngineInterface): Promise<void> {
  try {
    const run = await $.process.run(['ailang', '--version'], { timeoutMs: 5_000 })
    version = run.stdout.match(/AILANG (v[0-9][^\s-]*)/)?.[1] ?? ''
  } catch {
    version = ''
  }
}

export const register: Register = on => {
  version = ''

  on('session.start', async ($, e, next) => {
    await $.command.register({ name: 'ailang', description: 'AILANG: logo, version and the mods you have' })
    await readVersion($)
    return next(e)
  })

  on('command.run', { command: 'ailang' }, async $ => {
    await $.ui.open({ id: PANE, title: 'AILANG' })
    return { text: `AILANG ${version || '(ailang not on PATH)'}` }
  })

  // Beside "? for shortcuts" under the prompt.
  on('ui.render', { component: 'PromptHint' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, tail: `${e.props.tail ?? ''}  λ AILANG${version ? ` ${version}` : ''}` } }))

  on('ui.render', { component: 'Spinner' }, ($, e, next) =>
    e.props.message ? next(e) : next({ ...e, props: { ...e.props, word: pick(e.props.word)[0] } }))

  on('ui.render', { component: 'TurnDuration' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, word: pick(e.props.word)[1] } }))

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const elements = $.ui.resolve(e)
    const { Box, Text } = elements
    // Image is on the terminal and desktop tables; the terminal shows the
    // picture where it speaks the kitty graphics protocol, the alt text elsewhere.
    const logo = 'Image' in elements
      ? <elements.Image
          source={{ png: (await $.fs.read(`${$.plugin.root}/assets/ailang-logo.png`, { as: 'bytes' })).base64 }}
          columns={24} rows={11} alt="λ AILANG" />
      : <Text bold color="cyan">λ</Text>
    return (
      <Box flexDirection="column" alignItems="center">
        {logo}
        <Text bold>{`AILANG ${version}`}</Text>
        <Text dimColor>The AI-first programming language</Text>
        <Text dimColor>/ail-lens · /ail-inbox · ailang prompt</Text>
      </Box>
    )
  })
}
