---
sidebar_position: 6
title: Build Your First motoko Extension
---

# Build Your First motoko Extension

A 7-step walkthrough for creating a new motoko_agent extension as a reusable AILANG package. We'll build `motoko-ext-openkb` — a tool that lets the agent search a local knowledge base directory.

If you've read [Extension Packages](./extension-packages.md) (the reference) and want a guided tutorial instead, you're in the right place.

---

## Before you start

**The single most important rule:** an extension is a **separate AILANG package** that lives in [`ailang-packages/`](https://github.com/sunholo-data/ailang-packages), not a folder inside `motoko_agent/src/core/ext/`. This isn't pedantry — putting it inside motoko_agent breaks versioning, prevents reuse, and forces you to hand-edit `registry_generated.ail` (which the generator will overwrite next time you run it).

You'll need:
- AILANG `>= 0.18.4` (`ailang --version`)
- `motoko_agent` and `ailang-packages` checked out as siblings (typical layout: `~/dev/myorg/{motoko_agent,ailang-packages}/`)
- A working `make check_core` in motoko_agent

---

## Step 1: Scaffold the package

Create the directory under `ailang-packages/packages/`:

```bash
cd ../ailang-packages/packages
mkdir motoko-ext-openkb
cd motoko-ext-openkb
```

Naming convention: **`motoko-ext-<short_name>`** with hyphens. The `motoko-ext-` prefix is automatically stripped by the registry generator's short-name derivation, leaving `openkb` as the dispatch key.

Create `ailang.toml`:

```toml
[package]
name = "sunholo/motoko_ext_openkb"   # underscores in module-style name; matches dependency declaration
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/motoko_ext_abi" = { path = "../motoko-ext-abi" }   # path = OK while developing the package itself

[effects]
max = ["Process", "FS", "Env"]   # everything your extension needs at MOST

[exports]
modules = [
  "sunholo/motoko_ext_openkb/register",   # MUST end in /register
  "sunholo/motoko_ext_openkb/openkb",
  "sunholo/motoko_ext_openkb/types"
]
```

> **Why `_ext_` matters.** The generator strips `motoko-ext-` from package short-names. If you name it `motoko_openkb` instead, the dispatch key becomes `motoko_openkb` (ugly) instead of `openkb` (clean). Arni's first attempt used `motoko_openkb` — easy to miss.

---

## Step 2: Define your types

Create `types.ail`:

```ailang
module sunholo/motoko_ext_openkb/types

-- The shape of an OpenKB search result.
export type KBHit = {
  doc_path: string,
  excerpt: string,
  score: float
}
```

Verify it type-checks:

```bash
ailang check types.ail   # should print "✓ No errors found!"
```

---

## Step 3: Implement your tool

Create `openkb.ail`:

```ailang
module sunholo/motoko_ext_openkb/openkb

import std/process (exec, ProcessOutput)
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)
import std/string (length, contains)
import pkg/sunholo/motoko_ext_abi/types (
  ExtensionHooks, ExtCtx, ToolPolicyDecision, ToolHandleDecision,
  ResponseInterceptDecision, FinalizeDecision, PromptPatch, BudgetPatch,
  Allow, Delegate, NoIntercept, NoDecision
)

-- Your tool's handler. Receives the call envelope, returns a result string.
func handle_search(workdir: string, query: string) -> string ! {Process} {
  -- placeholder: replace with real KB-search logic
  "[openkb] would search '${query}' in ${workdir}/.openkb"
}

-- The hook factory exported for register.ail to call.
export func make_hooks(workdir: string) -> ExtensionHooks ! {Env, FS} {
  let policy = \_ctx _call. Allow;

  let handle = func(_ctx: ExtCtx, call: { id: string, name: string, arguments: string })
    -> ToolHandleDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
    if call.name == "OpenKBSearch"
    then {
      -- Decode call.arguments JSON, run handle_search, return Allow with result
      Delegate   -- replace with actual handle invocation
    }
    else Delegate
  };

  let no_intercept = func(_ctx: ExtCtx, _text: string)
    -> ResponseInterceptDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} { NoIntercept };

  let no_solver = func(_ctx: ExtCtx, _text: string)
    -> FinalizeDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} { NoDecision };

  {
    id: "openkb",                              -- becomes the dispatch key
    provided_tools: ["OpenKBSearch"],          -- tools your extension contributes
    on_describe_tools: \_ . [/* tool schema */],
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

> Filling in every hook field is mandatory — but you can no-op the ones you don't care about. The pattern above shows the minimum boilerplate.

---

## Step 4: Write the `register` entry point

Create `register.ail`:

```ailang
module sunholo/motoko_ext_openkb/register

import std/env (getEnvOr)
import pkg/sunholo/motoko_ext_abi/types (ExtensionHooks)
import pkg/sunholo/motoko_ext_openkb/openkb (make_hooks)

-- The function name `register_with_config` is the canonical contract:
-- the registry generator emits a call to this exact name.
export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  let workdir = getEnvOr("MOTOKO_WORKDIR", ".");
  make_hooks(workdir)
}
```

Type-check the whole package:

```bash
ailang lock                           # resolve motoko-ext-abi
ailang check register.ail             # ✓ No errors found!
```

---

## Step 5: Wire it into motoko_agent

Open `motoko_agent/ailang.toml` and add:

```toml
[dependencies]
# ... existing entries ...
# WHILE DEVELOPING — point at your local checkout
"sunholo/motoko_ext_openkb" = { path = "../ailang-packages/packages/motoko-ext-openkb" }

[extensions]
packages = [
  # ... existing entries ...
  "sunholo/motoko_ext_openkb@0.1.0",
]
```

> **Path vs registry.** During development, `{ path = "..." }` lets motoko consume your in-progress edits without a publish round-trip. Before opening a PR or shipping, swap to `"sunholo/motoko_ext_openkb" = "0.1.0"` and re-lock. See the [path vs registry checklist](./extension-packages.md#path-vs-registry-checklist).

---

## Step 6: Regenerate the registry

**Do not hand-edit `src/core/ext/registry_generated.ail`.** Run:

```bash
cd ../motoko_agent
ailang lock
ailang generate-extension-registry
```

Verify the output:

```bash
grep -A 1 "openkb" src/core/ext/registry_generated.ail
# Should show the import + the else-if branch the generator added
```

**Common mistake:** people hand-edit `src/core/ext/registry.ail` (the older hand-curated dispatch file). That file is being phased out — `registry_generated.ail` is the source of truth. If your changes get clobbered when you re-run `lock`, you're editing the wrong file.

---

## Step 7: Configure + test

Add a config block in `.motoko/config/default/openkb.json` (or wherever your motoko config is):

```json
{
  "openkb": {
    "wiki_dir": ".openkb",
    "timeout_ms": 60000
  }
}
```

Type-check and run:

```bash
make check_core   # 23+/23+ modules type-check, including the new generated registry
make run TASK="search the openkb for AILANG syntax"   # smoke test
```

If `make check_core` fails, diff `src/core/ext/registry_generated.ail` against your last commit — most likely the generator added/removed a line that conflicts with a stale hand-edit.

---

## Common pitfalls (the "first attempt" failures)

These are the four mistakes that wreck most first attempts. Each was hit at least once in the wild.

| Mistake | Symptom | Fix |
|---|---|---|
| Putting the extension inside `motoko_agent/src/core/ext/openkb/` | Can't version it; conflicts with `generate-extension-registry`; can't be reused by other motoko hosts | Put it in `ailang-packages/packages/motoko-ext-openkb/` |
| Naming it `motoko_openkb` instead of `motoko-ext-openkb` | Short-name derivation produces ugly key (`motoko_openkb` instead of `openkb`) | Use the `motoko-ext-` prefix; underscores in `[package].name` field but hyphens in directory name |
| Hand-editing `src/core/ext/registry.ail` (or `registry_generated.ail`) | Changes vanish next time someone runs `ailang generate-extension-registry` | Always regenerate; never hand-edit the dispatch file |
| Leaving `path = "../..."` in the host `ailang.toml` for production | Lockfile bakes in your absolute path; PR/CI clones break | Swap to registry version + re-lock before opening PR |

---

## Going further

- Real reference implementation: [`motoko-ext-exa-search`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-exa-search) — closest in shape to what most third-party extensions need
- Multi-tool example with budget hooks: [`motoko-ext-omnigraph`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-omnigraph)
- Hook contract reference: [`pkg/sunholo/motoko_ext_abi/types.ExtensionHooks`](https://github.com/sunholo-data/ailang-packages/blob/main/packages/motoko-ext-abi/types.ail)
- Publishing your extension to the AILANG package registry: see [Package Publishing](./package-publishing.md)
