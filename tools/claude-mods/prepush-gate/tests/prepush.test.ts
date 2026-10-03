import { expect, test } from 'claude-code/testing'
import type { TestBody } from 'claude-code/testing'

type Answers = Record<string, { exitCode?: number; stdout?: string; stderr?: string }>

function setup(on: Parameters<TestBody>[1], answers: Answers, env: Record<string, string> = {}) {
  const ran: string[] = []
  let pushed = false
  on('session.cwd', async () => ({ value: '/repo' }))
  on('env.get', async (_, e) => ({ value: env[e.name] }))
  on('ui.status', async () => ({ value: undefined }))
  on('process.run', async (_, e) => {
    const key = e.argv.slice(0, 2).join(' ') + (e.argv[1] === '-n' ? ` ${e.argv[2]}` : '')
    ran.push(key)
    const a = answers[key] ?? {}
    return { value: { exitCode: a.exitCode ?? 0, stdout: a.stdout ?? '', stderr: a.stderr ?? '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('tool.call', async () => {
    pushed = true
    return { result: 'pushed' }
  })
  return { ran, wasPushed: () => pushed }
}

const GO = { 'git ls-files': { stdout: 'cmd/ailang/main.go\ninternal/x/x.go\n' } }

test('an unformatted Go file denies the push', async ($, on) => {
  const s = setup(on, { ...GO, 'gofmt -l': { stdout: 'internal/x/x.go\n' } })
  const res = await $.tool.call({ tool: 'Bash', command: 'git push origin dev' })
  expect(res.deny).toContain('gofmt: 1 unformatted Go file(s): internal/x/x.go')
  expect(s.wasPushed()).toBe(false)
})

test('a repo with no tracked Go roots pushes ungated', async ($, on) => {
  const s = setup(on, { 'git ls-files': { stdout: '' } })
  await $.tool.call({ tool: 'Bash', command: 'git push' })
  expect(s.wasPushed()).toBe(true)
  expect(s.ran).toEqual(['git ls-files'])
})

test('a make target the repo does not define is skipped', async ($, on) => {
  const s = setup(on, {
    ...GO,
    'make -n lint': { exitCode: 2, stderr: "make: *** No rule to make target `lint'.  Stop." },
    'make -n check-file-sizes': { exitCode: 2, stderr: "make: *** No rule to make target `check-file-sizes'.  Stop." },
  })
  await $.tool.call({ tool: 'Bash', command: 'gh pr create --fill' })
  expect(s.wasPushed()).toBe(true)
  expect(s.ran).not.toContain('make lint')
})

test('a failing lint denies with its tail', async ($, on) => {
  const s = setup(on, { ...GO, 'make lint': { exitCode: 2, stdout: 'internal/x/x.go:3: unused var y\n' } })
  const res = await $.tool.call({ tool: 'Bash', command: 'git push' })
  expect(res.deny).toContain('prepush gate failed (lint)')
  expect(res.deny).toContain('unused var y')
  expect(s.wasPushed()).toBe(false)
})

test('AILANG_SKIP_PREPUSH=1 in the launch environment skips every gate', async ($, on) => {
  const s = setup(on, { ...GO, 'gofmt -l': { stdout: 'internal/x/x.go\n' } }, { AILANG_SKIP_PREPUSH: '1' })
  await $.tool.call({ tool: 'Bash', command: 'git push' })
  expect(s.wasPushed()).toBe(true)
  expect(s.ran).toEqual([])
})

test('the hatch typed into the command does not open the gate', async ($, on) => {
  const s = setup(on, { ...GO, 'gofmt -l': { stdout: 'internal/x/x.go\n' } })
  await $.tool.call({ tool: 'Bash', command: 'AILANG_SKIP_PREPUSH=1 git push' })
  expect(s.wasPushed()).toBe(false)
})
