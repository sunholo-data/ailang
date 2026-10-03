import { expect, test } from 'claude-code/testing'

const IFACE = JSON.stringify({
  module: 'demo/shop',
  types: [{ name: 'Order' }],
  funcs: [
    { name: 'total', type: '(Order)->int', effects: [], pure: true },
    { name: 'main', type: '(())->()!{IO,FS}', effects: ['IO', 'FS'], pure: true },
  ],
  schema: 'ailang.iface/v1',
})

const PANE = { plugin: 'ailang-lens', component: 'Pane', requestId: 'ailang-lens', props: { title: 'AILANG lens', isFocused: false, bodyColumns: 70, placement: 'dock', scroll: { offset: 0, bodyRows: 40 }, view: {} } } as const
const out = (exitCode: number, stdout: string) => ({ value: { exitCode, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })

const BROKEN = JSON.stringify({
  file: 'shop.ail',
  passed: false,
  error_count: 1,
  errors: [{ code: 'ERROR', message: 'type error in shop (decl 0): type unification failed at [return type annotation at shop.ail:3:8]: cannot unify type constructors: int vs string' }],
})

for (const surface of ['terminal', 'desktop'] as const) {
  test(`a .ail write fills the lens (${surface})`, async ($, on) => {
    let isBroken = false
    on('process.run', async (_, e) => {
      const sub = e.argv[1]
      if (sub === 'iface') return isBroken ? out(1, '') : out(0, `Warning: stdlib mismatch\n${IFACE}`)
      return isBroken ? out(1, BROKEN) : out(0, '{"passed":true,"errors":[]}')
    })
    on('tool.call', async () => ({ result: 'written' }))
    on('ui.status', async () => ({ value: undefined }))
    on('ui.open', async () => ({ value: { isPlaced: true as const } }))

    await $.tool.call({ tool: 'Write', file_path: 'shop.ail', content: 'module demo/shop' })
    const ui = await $.ui.mount({ ...PANE, surface })
    expect(await ui.find({ text: 'demo/shop' })).toBeTruthy()
    expect(await ui.find({ text: /!\{IO,FS\}/ })).toBeTruthy()
    expect(await ui.find({ text: /pure/ })).toBeTruthy()

    await ui.unmount()
    isBroken = true
    await $.tool.call({ tool: 'Write', file_path: 'shop.ail', content: 'broken' })
    const after = await $.ui.mount({ ...PANE, surface })
    expect(await after.find({ text: /3:8\s+cannot unify type constructors: int vs string/ })).toBeTruthy()
    // the last good signatures stay visible beside the error
    expect(await after.find({ text: /!\{IO,FS\}/ })).toBeTruthy()
  })
}

test('non-.ail edits are ignored', async ($, on) => {
  let runs = 0
  on('process.run', async () => { runs++; return out(0, '{}') })
  on('tool.call', async () => ({ result: 'written' }))
  await $.tool.call({ tool: 'Write', file_path: 'main.go', content: 'package main' })
  expect(runs).toBe(0)
})

test('/ail-lens on a missing relative path says so', async ($, on) => {
  on('fs.stat', async () => ({ deny: 'ENOENT' }))
  on('session.cwd', async () => ({ value: '/work/repo' }))
  const { text } = await $.command.run({ command: 'ail-lens', args: 'nope/missing.ail', origin: { kind: 'composer' }, presentation: { isFullscreen: false, columns: 120 } })
  expect(text).toContain('/work/repo/nope/missing.ail not found')
})
