import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import { isSweepingGitOp, parsePorcelain, unownedDirty } from './core'

// A visibility warning, not an authority check (M-CLAUDE-CODE-MODS C1): git
// itself says what is dirty; this session's Edit/Write calls say what it
// wrote. Bash writes are unknowable, and the warning says it is a heuristic.

const own = atom({ plugin: 'unowned-dirty', key: 'own' } as const, [])

/** Pure: absolute paths under `top` as repo-relative, as porcelain prints them. */
export function relativeTo(top: string, paths: readonly string[]): Set<string> {
  const prefix = top.endsWith('/') ? top : `${top}/`
  return new Set(paths.filter(p => p.startsWith(prefix)).map(p => p.slice(prefix.length)))
}

async function sweepWarning($: EngineInterface, command: string): Promise<string | undefined> {
  const cwd = await $.session.cwd()
  const top = await $.process.run(['git', 'rev-parse', '--show-toplevel'], { cwd, timeoutMs: 10_000 })
  if (top.exitCode !== 0) return undefined
  const status = await $.process.run(['git', 'status', '--porcelain'], { cwd, timeoutMs: 10_000 })
  if (status.exitCode !== 0) return undefined

  const unowned = unownedDirty(parsePorcelain(status.stdout), relativeTo(top.stdout.trim(), await read($, own)))
  if (unowned.length === 0) return undefined
  const named = unowned.slice(0, 5).join(', ') + (unowned.length > 5 ? `, +${unowned.length - 5} more` : '')
  return `[unowned-dirty heuristic] '${command.trim().slice(0, 60)}' may sweep ${unowned.length} dirty file(s) this session did not write: ${named}. Coordinate before stashing or committing others' in-flight work.`
}

export const register: Register = on => {
  on('tool.call', async ($, e, next) => {
    if (e.tool === 'Edit' || e.tool === 'Write') {
      const ran = await next(e)
      if (ran.deny === undefined && !ran.isError) {
        const path = e.file_path.startsWith('/') ? e.file_path : `${await $.session.cwd()}/${e.file_path}`
        await update($, own, list => (list.includes(path) ? list : [...list, path]))
      }
      return ran
    }
    if (e.tool !== 'Bash' || !isSweepingGitOp(e.command)) return next(e)

    // Read the tree before the command changes it; never block it.
    const warning = await sweepWarning($, e.command).catch(() => undefined)
    if (warning) $.ui.toast(warning)
    const ran = await next(e)
    return warning && ran.deny === undefined ? { ...ran, context: [...(ran.context ?? []), warning] } : ran
  })
}
