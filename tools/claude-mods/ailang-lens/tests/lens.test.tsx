import { expect, test } from 'claude-code/testing'

import { relativeImports } from '../hooks/register'

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

test('relative imports resolve against the module folder', () => {
  expect(relativeImports('/r/sim/destination_stars.ail', 'module m\nimport ./data/stars (Star)\nimport ../shared/util\nimport pkg/x/y (z)\n'))
    .toEqual(['/r/sim/data/stars.ail', '/r/shared/util.ail'])
})

test('a package manifest edit re-checks the modules shown from that package', async ($, on) => {
  let isExported = false
  const IFACE_OK = JSON.stringify({ module: 'stapledons/sim/destination_stars', types: [], funcs: [], schema: 'ailang.iface/v1' })
  const NOT_EXPORTED = JSON.stringify({ passed: false, errors: [{ message: 'module loading error: module "stapledons/sim/data/stars" is not exported by package "stapledons/sim"' }] })
  on('fs.stat', async (_, e) => (e.path === '/r/sim/ailang.toml'
    ? { value: { kind: 'file' as const, size: 1, mtimeMs: isExported ? 2 : 1, isLink: false } }
    : e.path === '/r/sim/destination_stars.ail' ? { value: { kind: 'file' as const, size: 1, mtimeMs: 1, isLink: false } } : { deny: 'ENOENT' }))
  on('fs.read', async () => ({ value: 'module stapledons/sim/destination_stars\nimport ./data/stars (Star)\n' }))
  on('process.run', async (_, e) => (e.argv[1] === 'iface'
    ? out(0, IFACE_OK)
    : isExported ? out(0, '{"passed":true,"errors":[]}') : out(1, NOT_EXPORTED)))
  on('tool.call', async () => ({ result: 'written' }))
  on('ui.status', async () => ({ value: undefined }))
  on('ui.open', async () => ({ value: { isPlaced: true as const } }))

  await $.tool.call({ tool: 'Write', file_path: '/r/sim/destination_stars.ail', content: 'x' })
  const before = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await before.find({ text: /✗ fails/ })).toBeTruthy()
  await before.unmount()

  isExported = true
  await $.tool.call({ tool: 'Edit', file_path: '/r/sim/ailang.toml', old_string: 'a', new_string: 'b' })
  const after = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await after.find({ text: /✓ checks/ })).toBeTruthy()
})
