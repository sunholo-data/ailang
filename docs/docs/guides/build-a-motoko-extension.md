---
sidebar_position: 6
title: Build Your First motoko Extension
tier: experimental
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Build Your First motoko Extension

A 3-step walkthrough for creating a new `motoko_agent` extension as a reusable AILANG package (`motoko-ext-openkb`). See [Extension Packages](./extension-packages.md) for the full manifest and generator reference.

## Before You Start

**An extension is a separate AILANG package** in [`ailang-packages/`](https://github.com/sunholo-data/ailang-packages), never a folder inside `motoko_agent/src/core/ext/`. Keeping it separate preserves independent versioning, reuse, and automated dispatch generation.

Prerequisites: `ailang` binary (`ailang --version`), sibling checkouts of `motoko_agent` and `ailang-packages`, and a working `make check_core` in `motoko_agent`.

## Step 1: Scaffold the Package

```bash
cd ../ailang-packages
ailang init motoko-extension \
  --name arniwesth/motoko_ext_openkb \
  --tools "OpenKBSearch,OpenKBList" \
  --effects "FS,Process,Env"
```

This generates `packages/motoko-ext-openkb/` ready for `ailang lock` and `ailang check`:

| Generated file | Role |
|---|---|
| `ailang.toml` | Registry dependencies, `[exports]`, `[effects].max` |
| `register.ail` | Canonical `register_with_config` wrapper |
| `types.ail` | Exported extension types |
| `openkb.ail` | `ExtensionHooks` implementation (with no-op defaults) |
| `README.md` | Package overview |

## Step 2: Implement Hooks & Types

Edit `openkb.ail` to implement `on_describe_tools` (returning `[ToolSchema]` schemas) and `on_tool_handle` (matching `call.name` and returning `Handled(result)` or `Delegate`). Leave unused hooks as no-op defaults, update `types.ail`, and run `ailang check register.ail` after edits.

## Step 3: Wire into `motoko_agent`

Add the local path dependency during development in `motoko_agent/ailang.toml`:

```toml
[dependencies]
"arniwesth/motoko_ext_openkb" = { path = "../ailang-packages/packages/motoko-ext-openkb" }

[extensions]
packages = [
  "arniwesth/motoko_ext_openkb@0.1.0",
]
```

```bash
ailang lock
ailang generate-extension-registry   # regenerates src/core/ext/registry_generated.ail
make check_core
```

Before opening a PR, publish the extension with [`ailang publish`](./package-publishing.md) and swap the `path = ...` entry to `"0.1.0"`.

## Common Pitfalls (Auto-Prevented by Scaffolder)

| Mistake | Symptom | Prevented by `ailang init motoko-extension` |
|---|---|---|
| Placing code in `motoko_agent/src/core/ext/` | Cannot version or share | Always outputs to `packages/motoko-ext-<name>/` |
| Omitting `motoko-ext-` / `motoko_ext_` infix | Key derivation doesn't strip prefix | Validates `motoko_ext_` infix |
| Hand-editing `registry_generated.ail` | Overwritten by generator | Scaffolder leaves host registry alone |
| Leaving sibling `path = "../..."` in prod | CI clones lack sibling checkout | Scaffolder pins registry `motoko_ext_abi` |

## Going Further

- Reference extensions: [`motoko-ext-exa-search`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-exa-search), [`motoko-ext-omnigraph`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-omnigraph)
- Hook contract: [`pkg/sunholo/motoko_ext_abi/types.ExtensionHooks`](https://github.com/sunholo-data/ailang-packages/blob/main/packages/motoko-ext-abi/types.ail)
- Publishing guide: [Publishing Your Package](./package-publishing.md)

<details>
<summary>Appendix: Manual scaffolding file templates (types.ail, openkb.ail, register.ail)</summary>

```ailang
module sunholo/motoko_ext_openkb/types

export type KBHit = {
  doc_path: string,
  excerpt: string,
  score: float
}
```

```ailang parse
module sunholo/motoko_ext_openkb/openkb

import pkg/sunholo/motoko_ext_abi/types (
  ExtensionHooks, ExtCtx, ToolHandleDecision,
  ResponseInterceptDecision, FinalizeDecision, PromptPatch, BudgetPatch,
  Allow, Delegate, NoIntercept, NoDecision
)

export func make_hooks(workdir: string) -> ExtensionHooks ! {Env, FS} {
  let policy = \_ctx _call. Allow;
  let handle = func(_ctx: ExtCtx, call: { id: string, name: string, arguments: string })
    -> ToolHandleDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
    Delegate
  };
  let no_intercept = func(_ctx: ExtCtx, _text: string)
    -> ResponseInterceptDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} { NoIntercept };
  let no_solver = func(_ctx: ExtCtx, _text: string)
    -> FinalizeDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} { NoDecision };
  {
    id: "openkb",
    provided_tools: ["OpenKBSearch"],
    on_describe_tools: \_ . [],
    on_build_system_prompt: func(_ctx: ExtCtx) -> PromptPatch { { prepend: [], append: [] } },
    on_budget_plan: func(_ctx: ExtCtx, _plan: a) -> BudgetPatch ! {Env, FS} {
      { requested_total: None, requested_solver: None, requested_verifier: None }
    },
    on_tool_policy: policy,
    on_tool_handle: handle,
    on_response_intercept: no_intercept,
    on_solver_candidate: no_solver
  }
}
```

```ailang parse
module sunholo/motoko_ext_openkb/register

import std/env (getEnvOr)
import pkg/sunholo/motoko_ext_abi/types (ExtensionHooks)
import pkg/sunholo/motoko_ext_openkb/openkb (make_hooks)

export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  let workdir = getEnvOr("MOTOKO_WORKDIR", ".");
  make_hooks(workdir)
}
```

</details>
