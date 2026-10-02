import { expect, test } from 'claude-code/testing'

const msg = (id: string, title: string, createdAt: string) => ({
  id, message_id: id, from_agent: 'stapledons_godot', to_inbox: 'user', message_type: 'notification',
  title, payload: `body of ${id}`, status: 'unread', created_at: createdAt,
})
const out = (stdout: string, exitCode = 0) =>
  ({ value: { exitCode, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })
const RUN = { origin: { kind: 'composer' }, presentation: { isFullscreen: false, columns: 120 } } as const
const BAND = {
  plugin: 'ailang-inbox-band', component: 'AbovePrompt',
  props: { hasSurvey: false, isWorking: false, maxRows: 4, bodyColumns: 100, scroll: { offset: 0, bodyRows: 4 }, view: {} },
} as const

for (const surface of ['terminal', 'desktop'] as const) {
  test(`band shows the count and newest title (${surface})`, async ($, on) => {
    on('process.run', async () => out(JSON.stringify([
      msg('a', 'older report', '2026-10-01T10:00:00Z'),
      msg('b', 'strict VM: nested patterns unbound', '2026-10-02T07:11:39Z'),
    ])))
    on('ui.open', async () => ({ value: { isPlaced: true as const } }))
    on('ui.toast', async () => ({ value: undefined }))

    const { text } = await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
    expect(text).toBe('2 unread in user.')
    expect(text).not.toContain('strict VM') // C3: no message text in what the model may read

    const band = await $.ui.mount({ ...BAND, surface })
    expect(await band.find({ text: /2 unread/ })).toBeTruthy()
    expect(await band.find({ text: 'strict VM: nested patterns unbound' })).toBeTruthy()
  })
}

test('no unread: the band draws nothing of its own', async ($, on) => {
  on('process.run', async () => out('null\n'))
  on('ui.open', async () => ({ value: { isPlaced: true as const } }))
  let isPassedOn = false
  on('ui.render', async () => {
    isPassedOn = true
    return null as never
  })

  await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
  await $.ui.mount({ ...BAND, surface: 'terminal' }).catch(() => undefined)
  expect(isPassedOn).toBe(true)
})

test('Ack runs messages ack and drops the row', async ($, on) => {
  const calls: string[][] = []
  on('process.run', async (_, e) => {
    calls.push([...e.argv])
    return e.argv[2] === 'ack' ? out('') : out(JSON.stringify([msg('a', 'one', '2026-10-02T07:00:00Z')]))
  })
  on('ui.open', async () => ({ value: { isPlaced: true as const } }))
  on('clock.now', async () => ({ value: Date.parse('2026-10-02T09:00:00Z') }))

  await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
  const pane = await $.ui.mount({
    plugin: 'ailang-inbox-band', surface: 'terminal', component: 'Pane', requestId: 'ailang-inbox-band',
    props: { title: 'AILANG inbox', isFocused: true, bodyColumns: 80, placement: 'dock', scroll: { offset: 0, bodyRows: 30 }, view: {} },
  })
  expect(await pane.find({ text: /stapledons_godot → user · 2h/ })).toBeTruthy()
  await pane.press({ key: 'ack-a' })

  expect(calls.some(argv => argv.join(' ') === 'ailang messages ack a')).toBe(true)
  expect(await pane.find({ text: /No unread messages/ })).toBeTruthy()
})

test('a new message is toasted once; the first backlog is not', async ($, on) => {
  let list = [msg('a', 'backlog', '2026-10-02T06:00:00Z')]
  on('process.run', async () => out(JSON.stringify(list)))
  on('ui.open', async () => ({ value: { isPlaced: true as const } }))
  const toasts: string[] = []
  on('ui.toast', async (_, e) => {
    toasts.push(String((e as { text?: string }).text ?? JSON.stringify(e)))
    return { value: undefined }
  })

  await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
  expect(toasts.length).toBe(0)

  list = [msg('b', 'fresh one', '2026-10-02T08:00:00Z'), ...list]
  await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
  await $.command.run({ command: 'ail-inbox', args: '', ...RUN })
  expect(toasts.length).toBe(1)
  expect(toasts[0]).toContain('fresh one')
})
