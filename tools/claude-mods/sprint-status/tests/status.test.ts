import { expect, test } from 'claude-code/testing'

const sprint = (id: string, status: string, passes: (boolean | null)[]) => JSON.stringify({
  sprint_id: id, status, features: passes.map((p, i) => ({ id: `M${i + 1}_STEP`, passes: p })),
})

test('the newest in-progress sprint goes on the status line', async ($, on) => {
  const files: Record<string, { body: string; mtime: number }> = {
    'sprint_OLD.json': { body: sprint('M-OLD', 'in_progress', [true, null]), mtime: 1 },
    'sprint_NEW.json': { body: sprint('M-NEW', 'in_progress', [true, true, false, null]), mtime: 3 },
    'sprint_DONE.json': { body: sprint('M-DONE', 'completed', [true]), mtime: 5 },
  }
  const name = (path: string) => path.split('/').pop()!
  on('session.cwd', async () => ({ value: '/repo' }))
  on('clock.now', async () => ({ value: 1_000_000 }))
  on('fs.list', async () => ({
    value: Object.entries(files).map(([n, f]) => ({ name: n, kind: 'file' as const, size: 1, mtimeMs: f.mtime, isLink: false })),
  }))
  on('fs.read', async (_, e) => ({ value: files[name(e.path)]!.body }))
  on('session.start', async () => ({ cwd: '/repo' }) as never)
  const shown: (string | undefined)[] = []
  on('ui.status', async (_, e) => {
    shown.push((e as { text?: string }).text)
    return { value: undefined }
  })

  await $.session.start({ source: 'startup', cwd: '/repo' } as never)
  expect(shown.at(-1)).toBe('sprint M-NEW 2/4 · M3_STEP')
})
