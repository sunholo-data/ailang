export type LensFunc = { name: string; type: string; effects: string[] }

export type LensModule = {
  file: string
  module: string
  types: string[]
  funcs: LensFunc[]
  passed: boolean
  errors: string[]
  ms: number
  /** Package root the checks ran from (nearest ailang.toml, else .git). */
  root: string
  /** Newest mtime of the module, its package manifest/lock and its ./ imports. */
  stamp: number
}

declare module 'claude-code' {
  interface PluginState {
    'ailang-lens': { modules: LensModule[] }
  }
}
