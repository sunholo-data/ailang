export type InboxMessage = {
  id: string
  from: string
  inbox: string
  title: string
  body: string
  createdAt: string
}

declare module 'claude-code' {
  interface PluginState {
    'ailang-inbox-band': {
      unread: InboxMessage[]
      toasted: string[]
      openId: string
      error: string
    }
  }
}
