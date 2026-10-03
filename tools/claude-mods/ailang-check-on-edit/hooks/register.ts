import type { EngineInterface, Register } from 'claude-code'

// The model's side of the lens: errors go into the tool result's `context`,
// which the model reads after the result and the person never sees. A passing
// check adds nothing. Advisory only (M-CLAUDE-CODE-MODS C1/C2).

const MAX_ERRORS = 5

/** Pure: "type error in m (decl 0): ... at [... at m.ail:3:8]: cannot unify ..." → "3:8 cannot unify ..." */
export function shortError(message: string): string {
  const where = message.match(/\.ail:(\d+:\d+)/)?.[1]
  const tail = message.includes(']: ')
    ? message.slice(message.lastIndexOf(']: ') + 3)
    : message.replace(/^type error in \S+ \(decl \d+\):\s*/, '')
  return (where ? `${where} ${tail}` : message).split('\n')[0]!.trim()
}

/** Pure: the context line for a check report, or undefined when it passed. */
export function contextFor(rel: string, stdout: string, exitCode: number): string | undefined {
  const at = stdout.indexOf('{')
  let report: { passed?: boolean; errors?: { message?: string }[] } | undefined
  try {
    report = at < 0 ? undefined : JSON.parse(stdout.slice(at))
  } catch {
    report = undefined
  }
  if (exitCode === 0 && report?.passed !== false) return undefined

  const errors = (report?.errors ?? []).map(err => shortError(err.message ?? 'error'))
  if (errors.length === 0) return `ailang check ${rel}: ✗ failed (exit ${exitCode}), no structured errors`
  const shown = errors.slice(0, MAX_ERRORS).map(err => `  ${err}`)
  const more = errors.length > MAX_ERRORS ? [`  … ${errors.length - MAX_ERRORS} more`] : []
  return [`ailang check ${rel}: ✗ ${errors.length} error${errors.length === 1 ? '' : 's'}`, ...shown, ...more].join('\n')
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

async function check($: EngineInterface, path: string, isFormatting: boolean): Promise<string | undefined> {
  const file = path.startsWith('/') ? path : `${await $.session.cwd()}/${path}`
  const cwd = await projectRoot($, file)
  const rel = file.slice(cwd.length + 1)
  try {
    if (isFormatting) await $.process.run(['ailang', 'fmt', '--write', rel], { cwd, timeoutMs: 20_000 })
    const run = await $.process.run(['ailang', 'check', '--json', '--quiet', rel], { cwd, timeoutMs: 60_000 })
    return contextFor(rel, run.stdout, run.exitCode)
  } catch (err) {
    return `ailang check ${rel}: did not finish (${String(err).slice(0, 80)})`
  }
}

export const register: Register = (on, options) => {
  const isFormatting = options.formatOnEdit === true

  on('tool.call', async ($, e, next) => {
    const ran = await next(e)
    const path = e.tool === 'Edit' || e.tool === 'Write' ? e.file_path : undefined
    if (!path?.endsWith('.ail') || ran.deny !== undefined || ran.isError) return ran

    const line = await check($, path, isFormatting)
    return line ? { ...ran, context: [...(ran.context ?? []), line] } : ran
  })
}
