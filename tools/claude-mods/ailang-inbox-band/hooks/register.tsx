import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { InboxMessage } from '../types'

// For the person only (M-CLAUDE-CODE-MODS C3): message text is external
// content from other agents and GitHub, so nothing here reaches the model.
// The /ail-inbox command's reply carries a count, never a title or body.

const PANE = 'ailang-inbox-band'
const unread = atom({ plugin: 'ailang-inbox-band', key: 'unread' } as const, [])
const toasted = atom({ plugin: 'ailang-inbox-band', key: 'toasted' } as const, [])
const openId = atom({ plugin: 'ailang-inbox-band', key: 'openId' } as const, '')
const error = atom({ plugin: 'ailang-inbox-band', key: 'error' } as const, '')

type Raw = { id: string; from_agent?: string; to_inbox?: string; title?: string; payload?: string; created_at?: string }

export function parseList(stdout: string): InboxMessage[] {
  // The JSON starts on its own line; a version warning may precede it.
  const at = stdout.search(/^(\[|null)/m)
  if (at < 0) return []
  const list = JSON.parse(stdout.slice(at)) as Raw[] | null
  return (list ?? []).map(m => ({
    id: m.id,
    from: m.from_agent ?? '?',
    inbox: m.to_inbox ?? '?',
    title: (m.title ?? '').trim() || '(no title)',
    body: (m.payload ?? '').slice(0, 1500),
    createdAt: m.created_at ?? '',
  }))
}

export function age(createdAt: string, now: number): string {
  const ms = now - Date.parse(createdAt)
  if (!Number.isFinite(ms) || ms < 0) return ''
  const minutes = Math.floor(ms / 60_000)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return hours < 48 ? `${hours}h` : `${Math.floor(hours / 24)}d`
}

function inboxesOf(options: Record<string, unknown>): string[] {
  return String(options.inboxes ?? 'user').split(',').map(s => s.trim()).filter(Boolean)
}

// The backlog at the first poll of a load is shown in the band, not toasted.
let hasPolled = false

async function poll($: EngineInterface, inboxes: readonly string[]): Promise<InboxMessage[]> {
  const found: InboxMessage[] = []
  for (const inbox of inboxes) {
    try {
      const run = await $.process.run(
        ['ailang', 'messages', 'list', '--unread', '--json', '--limit', '20', '--inbox', inbox],
        { timeoutMs: 10_000 },
      )
      if (run.exitCode !== 0) throw new Error(run.stderr.trim().split('\n').pop() || `exit ${run.exitCode}`)
      found.push(...parseList(run.stdout))
    } catch (err) {
      await update($, error, () => `${inbox}: ${String(err).slice(0, 120)}`)
      return read($, unread)
    }
  }
  await update($, error, () => '')

  const byNewest = [...new Map(found.map(m => [m.id, m])).values()]
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
  await update($, unread, () => byNewest)

  // The backlog at the first poll is shown in the band, not toasted.
  const seen = new Set(await read($, toasted))
  const fresh = byNewest.filter(m => !seen.has(m.id))
  if (hasPolled && fresh.length > 0) {
    $.ui.toast(fresh.length === 1 ? `📬 ${fresh[0]!.from}: ${fresh[0]!.title}` : `📬 ${fresh.length} new messages`)
  }
  hasPolled = true
  await update($, toasted, list => [...list, ...fresh.map(m => m.id)].slice(-500))
  return byNewest
}

async function ack($: EngineInterface, id: string): Promise<void> {
  const run = await $.process.run(['ailang', 'messages', 'ack', id], { timeoutMs: 10_000 })
  if (run.exitCode !== 0) {
    $.ui.toast(`ack failed: ${run.stderr.trim().slice(0, 80)}`)
    return
  }
  await update($, unread, list => list.filter(m => m.id !== id))
}

export const register: Register = (on, options) => {
  const inboxes = inboxesOf(options)
  const pollMs = Math.max(10, Number(options.pollSeconds ?? 60)) * 1000
  hasPolled = false

  on('session.start', async ($, e, next) => {
    await $.command.register({ name: 'ail-inbox', description: 'Show unread AILANG messages' })
    void poll($, inboxes)
    $.clock.every(pollMs, () => void poll($, inboxes))
    return next(e)
  })

  on('command.run', { command: 'ail-inbox' }, async $ => {
    const list = await poll($, inboxes)
    await $.ui.open({ id: PANE, title: 'AILANG inbox' })
    return { text: `${list.length} unread in ${inboxes.join(', ')}.` }
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    const list = await read($, unread)
    if (e.props.hasSurvey || list.length === 0) return next(e)

    const { Box, Button, Text } = $.ui.resolve(e)
    const newest = list[0]!
    return (
      <Box>
        <Text>
          <Text color="cyan">{`📬 ${list.length} unread`}</Text>
          <Text dimColor>{` · ${newest.inbox} — `}</Text>
          <Text wrap="truncate-end">{newest.title}</Text>
        </Text>
        <Text> </Text>
        <Button key="open" label="Open" dimColor onPress={() => $.ui.open({ id: PANE, title: 'AILANG inbox' })} />
      </Box>
    )
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Button, Text } = $.ui.resolve(e)
    const [list, opened, failure, now] = await Promise.all([
      read($, unread), read($, openId), read($, error), $.clock.now(),
    ])

    return (
      <Box flexDirection="column">
        {failure !== '' && <Text color="red">{`inbox: ${failure}`}</Text>}
        {list.length === 0 && <Text dimColor>{`No unread messages in ${inboxes.join(', ')}.`}</Text>}
        {list.map(m => (
          <Box flexDirection="column" marginBottom={1}>
            <Text bold wrap="truncate-end">{m.title}</Text>
            <Box>
              <Text dimColor>{`${m.from} → ${m.inbox} · ${age(m.createdAt, now)}  `}</Text>
              <Button
                key={`read-${m.id}`}
                label={opened === m.id ? 'Hide' : 'Read'}
                dimColor
                onPress={() => update($, openId, id => (id === m.id ? '' : m.id))}
              />
              <Text> </Text>
              <Button key={`ack-${m.id}`} label="Ack" onPress={() => ack($, m.id)} />
            </Box>
            {opened === m.id && <Text wrap="wrap">{m.body}</Text>}
          </Box>
        ))}
      </Box>
    )
  })
}
