import { expect, test } from 'claude-code/testing'

import { pick } from '../hooks/register'

test('pick is stable and two-way', () => {
  expect(pick('Sauteing')).toEqual(pick('Sauteing'))
  const [now, past] = pick('Baking')
  expect(now.length > 0 && past.length > 0).toBe(true)
})

test('the prompt hint carries the AILANG version', async ($, on) => {
  on('session.start', async () => ({ cwd: '/repo' }) as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('process.run', async () => ({ value: { exitCode: 0, stdout: 'AILANG v0.51.0\nCommit: abc\n', stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }))
  let seen: unknown
  on('ui.render', async (_, e) => {
    seen = (e as { props: { tail?: string } }).props.tail
    return null as never
  })
  await $.session.start({ source: 'startup', cwd: '/repo' } as never)
  await $.ui.render({ component: 'PromptHint', props: { isDraft: false, isWorking: false, hint: '? for shortcuts' } } as never).catch(() => undefined)
  expect(String(seen)).toContain('λ AILANG v0.51.0')
})
