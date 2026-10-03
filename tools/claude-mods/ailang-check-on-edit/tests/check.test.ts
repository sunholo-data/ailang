import { expect, test } from 'claude-code/testing'
import type { TestBody } from 'claude-code/testing'

const out = (exitCode: number, stdout: string) =>
  ({ value: { exitCode, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })
const FAIL = JSON.stringify({
  file: 'shop.ail', passed: false, error_count: 1,
  errors: [{ code: 'ERROR', message: 'type error in shop (decl 0): type unification failed at [return type annotation at shop.ail:3:8]: cannot unify type constructors: int vs string', file: 'shop.ail' }],
})
const PASS = JSON.stringify({ file: 'shop.ail', passed: true, error_count: 0, errors: [] })

function setup(on: Parameters<TestBody>[1], check: ReturnType<typeof out>) {
  const argvs: string[][] = []
  on('fs.stat', async (_, e) => e.path === '/repo/.git'
    ? { value: { kind: 'dir' as const, size: 0, mtimeMs: 0, isLink: false } }
    : { deny: 'ENOENT' })
  on('process.run', async (_, e) => {
    argvs.push([...e.argv, `@${e.init?.cwd}`])
    return check
  })
  on('tool.call', async () => ({ result: 'The file has been updated successfully.' }))
  return argvs
}

test('a failing .ail edit hands the model its errors', async ($, on) => {
  const argvs = setup(on, out(1, FAIL))
  const ran = await $.tool.call({ tool: 'Write', file_path: '/repo/src/shop.ail', content: 'x' })
  expect(ran.context).toEqual(['ailang check src/shop.ail: ✗ 1 error\n  3:8 cannot unify type constructors: int vs string'])
  expect(argvs[0]).toEqual(['ailang', 'check', '--json', '--quiet', 'src/shop.ail', '@/repo'])
})

test('a passing edit adds nothing', async ($, on) => {
  setup(on, out(0, PASS))
  const ran = await $.tool.call({ tool: 'Write', file_path: '/repo/src/shop.ail', content: 'x' })
  expect(ran.context ?? []).toEqual([])
})

test('non-.ail edits never run ailang', async ($, on) => {
  const argvs = setup(on, out(1, FAIL))
  await $.tool.call({ tool: 'Write', file_path: '/repo/main.go', content: 'package main' })
  expect(argvs.length).toBe(0)
})

test('formatOnEdit runs fmt before check', { options: { formatOnEdit: true } }, async ($, on) => {
  const argvs = setup(on, out(0, PASS))
  await $.tool.call({ tool: 'Write', file_path: '/repo/src/shop.ail', content: 'x' })
  expect(argvs.map(a => a[1])).toEqual(['fmt', 'check'])
})
