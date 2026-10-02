import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { LensFunc, LensModule } from '../types'

const PANE = 'ailang-lens'
const modules = atom({ plugin: 'ailang-lens', key: 'modules' } as const, [])

// First JSON object in a stream that may carry warnings before it.
function parseJson(text: string): any {
  const at = text.indexOf('{')
  if (at < 0) return undefined
  try {
    return JSON.parse(text.slice(at))
  } catch {
    return undefined
  }
}

function typeName(t: unknown): string {
  if (typeof t === 'string') return t
  if (t && typeof t === 'object' && 'name' in t) return String((t as { name: unknown }).name)
  return JSON.stringify(t)
}

// "type error in m (decl 0): type unification failed at [... at m.ail:3:8]: cannot unify ..."
// → "3:8 cannot unify ..."
function shortError(message: string): string {
  const where = message.match(/\.ail:(\d+:\d+)/)?.[1]
  const tail = message.includes(']: ')
    ? message.slice(message.lastIndexOf(']: ') + 3)
    : message.replace(/^type error in \S+ \(decl \d+\):\s*/, '')
  return where ? `${where}  ${tail}` : message
}

async function exists($: EngineInterface, path: string): Promise<boolean> {
  try {
    await $.fs.stat(path)
    return true
  } catch {
    return false
  }
}

// ailang names a module by its path from the working directory (MOD010), so
// run it from the project root: the nearest ailang.toml, else .git, else the
// file's own folder.
async function projectRoot($: EngineInterface, file: string): Promise<string> {
  const dirs: string[] = []
  for (let dir = file.slice(0, file.lastIndexOf('/')); dir; dir = dir.slice(0, dir.lastIndexOf('/'))) {
    dirs.push(dir)
  }
  for (const marker of ['ailang.toml', '.git']) {
    for (const dir of dirs) {
      if (await exists($, `${dir}/${marker}`)) return dir
    }
  }
  return dirs[0] ?? '.'
}

async function mtime($: EngineInterface, file: string): Promise<number> {
  try {
    return (await $.fs.stat(file)).mtimeMs
  } catch {
    return 0
  }
}

async function analyse($: EngineInterface, file: string, previous?: LensModule): Promise<LensModule> {
  const started = Date.now()
  const mtimeMs = await mtime($, file)
  const cwd = file.startsWith('/') ? await projectRoot($, file) : undefined
  const rel = cwd ? file.slice(cwd.length + 1) : file
  const [iface, check] = await Promise.all([
    $.process.run(['ailang', 'iface', rel], { cwd, timeoutMs: 20_000 }),
    $.process.run(['ailang', 'check', '--json', '--quiet', rel], { cwd, timeoutMs: 60_000 }),
  ])
  const shape = iface.exitCode === 0 ? parseJson(iface.stdout) : undefined
  const report = parseJson(check.stdout)

  const errors: string[] = report?.errors?.length
    ? report.errors.map((err: { message?: string }) => shortError(err.message ?? 'error'))
    : check.exitCode === 0
      ? []
      : [shortError((check.stderr || check.stdout).trim().split('\n').pop() ?? 'check failed')]

  // A broken edit keeps the last good shape on screen, beside the errors.
  const funcs: LensFunc[] = shape
    ? (shape.funcs ?? []).map((f: LensFunc) => ({ name: f.name, type: f.type, effects: f.effects ?? [] }))
    : (previous?.funcs ?? [])

  return {
    file,
    module: shape?.module ?? previous?.module ?? (file.split('/').pop() ?? file).replace(/\.ail$/, ''),
    types: shape ? (shape.types ?? []).map(typeName) : (previous?.types ?? []),
    funcs,
    passed: errors.length === 0 && check.exitCode === 0,
    errors,
    ms: Date.now() - started,
    mtimeMs,
  }
}

async function refresh($: EngineInterface, file: string): Promise<LensModule> {
  const known = await read($, modules)
  const lens = await analyse($, file, known.find(m => m.file === file))
  await update($, modules, list => [lens, ...list.filter(m => m.file !== file)].slice(0, 8))

  const effects = [...new Set(lens.funcs.flatMap(f => f.effects))].sort()
  $.ui.status(
    lens.passed
      ? `ail ✓ ${lens.module} · ${lens.funcs.length} fn · ${effects.length ? `!{${effects.join(',')}}` : 'pure'}`
      : `ail ✗ ${lens.module} · ${lens.errors.length} error${lens.errors.length === 1 ? '' : 's'}`,
  )
  return lens
}

// Edit/Write are seen directly; anything else that touches a tracked file
// (Bash, the person's editor) is caught by its mtime after a Bash call and
// at the end of each turn.
async function refreshStale($: EngineInterface): Promise<void> {
  for (const known of await read($, modules)) {
    if ((await mtime($, known.file)) !== known.mtimeMs) await refresh($, known.file)
  }
}

export const register: Register = on => {
  on('session.start', async ($, e, next) => {
    await $.command.register({
      name: 'ail-lens',
      description: 'Open the AILANG lens, or analyse a .ail file into it',
      argumentHint: '[file.ail]',
    })
    return next(e)
  })

  on('command.run', { command: 'ail-lens' }, async ($, e) => {
    const arg = e.args.trim()
    if (!arg) {
      await refreshStale($)
      await $.ui.open({ id: PANE, title: 'AILANG lens' })
      return { text: 'AILANG lens opened.' }
    }
    if (!arg.endsWith('.ail')) return { text: `ail-lens: ${arg} is not a .ail file.` }

    // A relative path is the person's, typed against the session's folder.
    const file = arg.startsWith('/') ? arg : `${await $.session.cwd()}/${arg.replace(/^\.\//, '')}`
    if (!(await exists($, file))) return { text: `ail-lens: ${file} not found.` }

    const lens = await refresh($, file)
    await $.ui.open({ id: PANE, title: 'AILANG lens' })
    return { text: `AILANG lens: ${lens.module} ${lens.passed ? '✓' : `✗ ${lens.errors.length} error(s)`}` }
  })

  // Every successful edit of a .ail file re-reads its interface and type-checks it.
  on('tool.call', async ($, e, next) => {
    const ran = await next(e)
    if (e.tool === 'Bash') {
      await refreshStale($).catch(() => undefined)
      return ran
    }
    const file = (e.tool === 'Edit' || e.tool === 'Write') ? e.file_path : undefined
    if (!file?.endsWith('.ail') || ran.deny !== undefined || ran.isError) return ran

    try {
      await refresh($, file)
      void $.ui.open({ id: PANE, title: 'AILANG lens' })
    } catch (err) {
      $.ui.status(`ail lens: ${String(err).slice(0, 60)}`)
    }
    return ran
  })

  on('turn.complete', async ($, e, next) => {
    const done = await next(e)
    await refreshStale($).catch(() => undefined)
    return done
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Text } = $.ui.resolve(e)
    const [current, ...older] = await read($, modules)

    if (!current) {
      return (
        <Box flexDirection="column">
          <Text dimColor>No AILANG edited yet.</Text>
          <Text dimColor>Edit a .ail file, or /ail-lens path/to/file.ail</Text>
        </Box>
      )
    }

    const width = Math.max(...current.funcs.map(f => f.name.length), 4)
    return (
      <Box flexDirection="column">
        <Text>
          <Text bold>{current.module}</Text>
          <Text color={current.passed ? 'green' : 'red'}>{current.passed ? '  ✓ checks' : '  ✗ fails'}</Text>
          <Text dimColor>{`  ${current.ms}ms`}</Text>
        </Text>

        {current.types.length > 0 && (
          <Text>
            <Text dimColor>types </Text>
            <Text color="magenta">{current.types.join('  ')}</Text>
          </Text>
        )}

        <Box flexDirection="column" marginTop={1}>
          {current.funcs.map(f => (
            <Box flexDirection="column">
              <Text>
                <Text bold>{f.name.padEnd(width)}</Text>
                <Text color={f.effects.length ? 'yellow' : 'green'}>
                  {f.effects.length ? `  !{${f.effects.join(',')}}` : '  pure'}
                </Text>
              </Text>
              <Text dimColor wrap="truncate-end">{`  ${f.type}`}</Text>
            </Box>
          ))}
        </Box>

        {current.errors.length > 0 && (
          <Box flexDirection="column" marginTop={1}>
            {current.errors.map(err => (
              <Text color="red" wrap="wrap">{`✗ ${err}`}</Text>
            ))}
            {current.funcs.length > 0 && <Text dimColor>(signatures above are the last good shape)</Text>}
          </Box>
        )}

        {older.length > 0 && (
          <Box flexDirection="column" marginTop={1}>
            <Text dimColor>recent</Text>
            {older.map(m => (
              <Text dimColor>
                {`${m.passed ? '✓' : '✗'} ${m.module} · ${m.funcs.length} fn`}
              </Text>
            ))}
          </Box>
        )}
      </Box>
    )
  })
}
