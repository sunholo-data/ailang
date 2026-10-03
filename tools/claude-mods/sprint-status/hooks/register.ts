import type { EngineInterface, Register } from 'claude-code'

const DIR = '.ailang/state/sprints'
const MIN_GAP_MS = 30_000

type Sprint = { sprint_id?: string; status?: string; features?: { id?: string; passes?: boolean | null }[] }

/** Pure: the status-line text for one sprint, or undefined when it is not in progress. */
export function summarise(sprint: Sprint): string | undefined {
  if (sprint.status !== 'in_progress') return undefined
  const features = sprint.features ?? []
  const passed = features.filter(f => f.passes === true).length
  const next = features.find(f => f.passes !== true)?.id
  return `sprint ${sprint.sprint_id ?? '?'} ${passed}/${features.length}${next ? ` · ${next}` : ''}`
}

let lastRun = 0

async function refresh($: EngineInterface): Promise<void> {
  const now = await $.clock.now()
  if (now - lastRun < MIN_GAP_MS) return
  lastRun = now

  const root = await $.session.cwd()
  let entries
  try {
    entries = (await $.fs.list(`${root}/${DIR}`)).filter(f => f.kind === 'file' && f.name.endsWith('.json'))
  } catch {
    $.ui.status(undefined) // not an AILANG checkout
    return
  }
  for (const f of entries.sort((a, b) => b.mtimeMs - a.mtimeMs).slice(0, 10)) {
    const path = `${root}/${DIR}/${f.name}`
    try {
      const text = summarise(JSON.parse(await $.fs.read(path)) as Sprint)
      if (text) {
        $.ui.status(text)
        return
      }
    } catch {
      // a half-written or hand-edited file: skip it
    }
  }
  $.ui.status(undefined)
}

export const register: Register = on => {
  lastRun = 0

  on('session.start', async ($, e, next) => {
    await refresh($).catch(() => undefined)
    return next(e)
  })

  on('turn.complete', async ($, e, next) => {
    const done = await next(e)
    await refresh($).catch(() => undefined)
    return done
  })
}
