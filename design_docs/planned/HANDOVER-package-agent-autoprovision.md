# HANDOVER — auto-provisioning an agent inbox when a package appears on the registry

**For:** whoever is building the `package-descriptions` / registry→agent path.
**From:** the message-plane session, 2026-09-17. Everything below was measured
today while adding one such agent by hand and making the lane a default.

## Why this is worth automating

`sunholo/docparse` had been a published package for months with **no agent**. A
report addressed to it was accepted, filed, and dispatched nowhere:

```
! NOTHING   docparse   accepted and never dispatched — no agent, not declared human-triage
```

It surfaced only because a triage pass read the message by hand. `sunholo/discord`
(v0.2.1) is in the same state right now. `ailang messages inboxes` is the
instrument that shows it — every inbox, and what sending there actually does.

## What is already automatic, and what is not

| | automatic? |
|---|---|
| package usable as a dependency | ✅ publishing is enough |
| an agent watching `pkg:sunholo/<name>` | ❌ a hand-written registry entry |
| the tool lane that agent runs on | ✅ since 2026-09-17 — a `pkg:` inbox defaults to `ailang_only` (`GetEffectiveToolPolicy`) |
| `policy_path` for that lane | ❌ **and dispatch now refuses without it** |

That last row is the trap for a generator. `tool_policy: ailang_only` takes the
shell away; `policy_path` is what gives back the ability to execute anything. An
agent with the first and not the second can read and write files and nothing
else — `ailang_run` default-denies. Since the default landed, dispatch fails
closed with a named reason rather than sending such a job, and `coordinator lint`
has a `lane-has-policy` rule that reads the EFFECTIVE policy. **A generated entry
must emit `policy_path: /etc/ailang-config/policies/pkg-ailang-only.toml`.**

## The four fields a generator cannot derive, and must look up

1. **The repo.** It is not a function of the package name. Measured:
   `sunholo/email` → `sunholo-data/email-parse`; `sunholo/ailang_parse` →
   `sunholo-data/ailang-parse` (no subdirectory, the repo IS the package);
   `sunholo/docparse` → `sunholo-data/docparse`; `sunholo/duckdb` →
   `sunholo-data/ailang-packages` with `subdirectory: packages/duckdb`. Any
   naming rule gets at least two of those wrong. The package's `ailang.toml`
   location in its own repo is the only reliable source.
2. **`subdirectory`** — set for a monorepo package, omitted when the repo root is
   the package root. It scopes both the agent's CWD and the deterministic
   `ailang publish` directory, so a wrong value clones the right repo and finds
   nothing (this exact bug sent every `pkg-sunholo-ailang-parse` dispatch to a
   path that does not exist).
3. **`artifact_patterns`** — leaving it unset means `**/*`, which bounds nothing,
   and the merge guard reads the DECLARED list. Repo-is-the-package → `**/*` is
   honest; monorepo package → `packages/<name>/**/*`.
4. **`merge_branch`** — `main` for the package repos, not the `dev` default.

## Inbox spelling

From the registry name with **underscores** (`FormatPackageInbox`), never the
repo directory's hyphens: a hyphen in an import path parses as subtraction and
fails with `PAR_HYPHEN_IN_IMPORT`. A hyphenated inbox mints a phantom that
nothing watches — that is how ten `pkg:sunholo/ailang-parse` tickets went
nowhere, and why the config keeps a short list of sender typos DELIBERATELY
undeclared so they keep bouncing visibly.

## Deploy path (two triggers, easy to mix up)

- `config/config.cloud.yaml` → the **agents** trigger (`ailang-multivac-agents-*`),
  which fires on that file alone and is the fast path (~2 min/rung).
- `config/templates/*.md`, `config/policies/*.toml` → the **config** trigger,
  which ignores `config.cloud.yaml` and root `*.md`.

Both ladder by branch push: `dev` → `dev:test` → `dev:prod`. Check
`origin/prod..origin/dev` first — it has carried other people's unfinished
terraform.

## Verification that actually proves it

```bash
ailang coordinator lint                                  # 7 rules, incl. lane-has-policy
ailang coordinator agent-check <id> --repo-config <multivac>/config/config.cloud.yaml
ailang messages inboxes | grep <inbox>                   # must read DISPATCHES
```

`agent-check` is the one that catches what a generator gets wrong: it verifies
the merge branch exists, that the fleet token can push to that repo, that no
config key is dead, and that the live registry matches the file. Without
`--repo-config` two of its eight checks report "unverifiable", which is not a
pass.

## One judgement call to make explicitly

Auto-provisioning means a newly published package immediately has an agent that
can be sent work. Every package agent is `auto_merge: false`,
`skip_approval: false`, so nothing lands without a human — but it will open PRs
and spend model budget on whatever arrives. Decide whether provisioning is
automatic on publish, or generated-and-proposed (a PR against
`config.cloud.yaml`) for an operator to merge. The second is what the current
approval posture implies.

## Decision + implementation (M-PKG-QUALITY-LADDER S1 M6, PR #1255, 2026-09-17)

**Automatic existence, human-gated landing.** Provisioning is a *runtime derivation*, not a
config mutation: `coordinator.package_agent_template` (a section, not an agent — it serves no
inbox) is cloned per package in the registry index at load and every 10 minutes
(`internal/coordinator/package_agents.go`). Derived agents inherit the template's
`auto_merge: false` / `skip_approval: false`, so nothing lands without a human — the posture
this doc says the second option implies — while a newly published package can be sent work
immediately. Hand-written `agents:` entries always win.

How the four look-ups are answered: **not from the name — from `metadata.repository`**, the
GitHub tree URL every publish banks (`sunholo/email` → `sunholo-data/email-parse` +
`packages/email`; `sunholo/ailang_parse` → repo root, `**/*`; `sunholo/duckdb` → the monorepo +
`packages/duckdb`). `merge_branch` follows the URL's `/tree/<branch>`. A package without a
parseable URL still gets an agent (on the template workspace) and `ailang pkg quality` flags it
(`PUB021`); a `pkg:` inbox for a package **not in the registry** is served by nothing, so the
deliberate typo list keeps bouncing. Inbox spelling is `FormatPackageInbox(registry name)` —
underscores.

The `policy_path` trap is refused structurally: a template on a non-`full` lane with no
`policy_path` derives nothing and logs why (`templateUsable`). Verification remains
`coordinator lint` + `agent-check --repo-config` on the template and `messages inboxes`
reading `DISPATCHES … (derived from registry)`.
