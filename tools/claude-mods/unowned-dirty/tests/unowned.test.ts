import { expect, test } from 'claude-code/testing'
import type { TestBody } from 'claude-code/testing'

const out = (stdout: string) =>
  ({ value: { exitCode: 0, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })

function setup(on: Parameters<TestBody>[1], porcelain: string) {
  const toasts: string[] = []
  on('session.cwd', async () => ({ value: '/repo' }))
  on('process.run', async (_, e) => (e.argv[1] === 'rev-parse' ? out('/repo\n') : out(porcelain)))
  on('tool.call', async () => ({ result: 'ok' }))
  on('ui.toast', async (_, e) => {
    toasts.push(JSON.stringify(e))
    return { value: undefined }
  })
  return toasts
}

test('a sweep names the dirty files this session did not write', async ($, on) => {
  const toasts = setup(on, ' M src/mine.ail\n M internal/theirs.go\n?? notes.txt\n')
  await $.tool.call({ tool: 'Write', file_path: '/repo/src/mine.ail', content: 'x' })

  const ran = await $.tool.call({ tool: 'Bash', command: 'git add -A' })
  expect(ran.deny).toBeUndefined()
  expect(ran.context?.[0]).toContain('may sweep 2 dirty file(s)')
  expect(ran.context?.[0]).toContain('internal/theirs.go, notes.txt')
  expect(ran.context?.[0]).not.toContain('mine.ail')
  expect(toasts.length).toBe(1)
})

test('silent when every dirty file is the session\'s own', async ($, on) => {
  const toasts = setup(on, ' M src/mine.ail\n')
  await $.tool.call({ tool: 'Edit', file_path: '/repo/src/mine.ail', old_string: 'a', new_string: 'b' })

  const ran = await $.tool.call({ tool: 'Bash', command: 'git add src/mine.ail' })
  expect(ran.context ?? []).toEqual([])
  expect(toasts.length).toBe(0)
})

test('non-sweeping git commands are not inspected', async ($, on) => {
  let runs = 0
  on('process.run', async () => {
    runs++
    return out('')
  })
  on('tool.call', async () => ({ result: 'ok' }))
  await $.tool.call({ tool: 'Bash', command: 'git status' })
  expect(runs).toBe(0)
})
