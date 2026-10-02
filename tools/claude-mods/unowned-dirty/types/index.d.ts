/** Absolute paths this session wrote through Edit or Write. */
export type OwnFiles = string[]

declare module 'claude-code' {
  interface PluginState {
    'unowned-dirty': { own: OwnFiles }
  }
}
