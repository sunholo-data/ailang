import type { EngineInterface, Register } from 'claude-code'

import { goRoots, isPushCommand, makeTargetDefined, skipRequested } from './core'

// Same gates and same refusals as .pi/extensions/prepush-gate.ts. The escape
// hatch is read from Claude Code's own launch environment ($.env), so an agent
// writing `AILANG_SKIP_PREPUSH=1 git push` does not open it. Not a security
// boundary (M-CLAUDE-CODE-MODS C1): it saves a red CI round-trip, nothing more.

async function gateFailure($: EngineInterface): Promise<string | undefined> {
  if (skipRequested({ AILANG_SKIP_PREPUSH: await $.env.get('AILANG_SKIP_PREPUSH') })) return undefined
  const cwd = await $.session.cwd()
  const run = (argv: string[], timeoutMs: number) => $.process.run(argv, { cwd, timeoutMs })

  // A gate for a toolchain the repo does not use has nothing to check.
  const tracked = await run(['git', 'ls-files', 'cmd/*.go', 'internal/*.go', 'cmd/**/*.go', 'internal/**/*.go'], 15_000)
  const roots = goRoots(tracked.stdout)
  if (roots.length === 0) return undefined

  const gofmt = await run(['gofmt', '-l', ...roots.map(r => `${r}/`)], 30_000)
  const unformatted = gofmt.stdout.split('\n').filter(l => l.trim() !== '')
  if (unformatted.length > 0) {
    return `gofmt: ${unformatted.length} unformatted Go file(s): ${unformatted.slice(0, 3).join(', ')}. Run gofmt -w on them, then push again.`
  }

  const defined = async (target: string) => {
    const probe = await run(['make', '-n', target], 15_000)
    return makeTargetDefined(target, probe.exitCode, `${probe.stderr}\n${probe.stdout}`)
  }
  if (await defined('lint')) {
    const lint = await run(['make', 'lint'], 180_000)
    if (lint.exitCode !== 0) {
      const tail = (lint.stderr || lint.stdout).split('\n').filter(Boolean).slice(-6).join('\n')
      return `prepush gate failed (lint):\n${tail.slice(0, 2000)}\n\nFix locally, then push again.`
    }
  }
  if (await defined('check-file-sizes')) {
    const sizes = await run(['make', 'check-file-sizes'], 15_000)
    if (sizes.exitCode !== 0) return `prepush gate failed (file sizes >800 lines):\n${sizes.stdout.slice(-400)}`
  }
  return undefined
}

export const register: Register = on => {
  on('tool.call', { tool: 'Bash' }, async ($, e, next) => {
    if (!isPushCommand(e.command)) return next(e)

    $.ui.status('prepush gate: running CI gates…')
    const failure = await gateFailure($).finally(() => $.ui.status(undefined))
    return failure ? { deny: `${$.plugin.name}: ${failure}` } : next(e)
  })
}
