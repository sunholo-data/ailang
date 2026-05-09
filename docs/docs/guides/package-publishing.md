---
sidebar_position: 7
title: Publishing Your Package
---

# Publishing Your Package

Two tiers exist for sharing AILANG packages. Pick the one matching where you are in the lifecycle.

| Tier | When | Mechanism | Visible to others? |
|---|---|---|---|
| **1. Local development** | While editing the package | `path = "../path/to/pkg"` in host's `ailang.toml` | No — your machine only |
| **2. Published** | When the package is stable + reusable | `ailang publish` → AILANG registry | Yes — anyone can `"<your-namespace>/<pkg>" = "0.1.0"` |

The motoko-extension tutorial shows Tier 1 in detail. This guide covers Tier 2.

---

## Tier 2 in one screen

```bash
# 1. Your package directory must have a valid ailang.toml
cd ~/dev/myorg/my-extensions/packages/myorg-ext-foo
cat ailang.toml
# [package]
# name = "myorg/ext_foo"     ← namespace MUST match your authority (see below)
# version = "0.1.0"
# ...

# 2. Set the API key (one-time per shell)
export AILANG_REGISTRY_API_KEY=<key-issued-by-sunholo>

# 3. Dry-run first to see exactly what would upload
ailang publish --dry-run

# 4. Publish for real
ailang publish

# 5. Anyone (you, the host project, other contributors) can now depend on it
#    Replace your local path = "../..." with a registry version:
#    "myorg/ext_foo" = "0.1.0"
```

That's the whole flow. The rest of this guide explains the prerequisites and gotchas.

---

## Prerequisites

### 1. An API key

Today the registry uses a single shared API key for publish authorization. Contact [the AILANG maintainers](https://github.com/sunholo-data/ailang/issues) (or your existing channel — for arniwesth and other established partners, this is direct) to be issued the key.

> **Honest disclosure:** the current key is a superuser key — it can publish under any namespace. Per-namespace key scoping is on the roadmap (see [Limitations](#current-limitations) below). The key MUST be kept private — don't commit it, don't paste it in a PR, don't put it in a GitHub Action without using a secret.

### 2. A namespace

Your package name in `ailang.toml` must follow `<namespace>/<package_name>`:

```toml
[package]
name = "arniwesth/motoko_ext_openkb"   # arniwesth = the namespace
```

**Choose a stable namespace** — it ends up in every consumer's `ailang.toml` and lockfile. Common conventions:
- Your GitHub username (`arniwesth/...`)
- Your org name (`sunholo/...`, `acme/...`)
- A project-specific prefix (`motoko/...` if you maintain motoko-side packages independently)

Once published, **a name is permanent** — there's no rename.

### 3. A clean `ailang.toml`

Your package's `ailang.toml` must:
- Have `[package]` with `name`, `version`, `edition` set
- Declare `[exports].modules` listing every module that consumers should be able to `import`
- Declare `[effects].max` listing every effect your package uses (consumers' effect ceilings will reject your package if you under-declare)
- For any cross-package internal dependency, use `path = ...` during dev — `ailang publish` automatically rewrites these to registry versions in the published tarball (see [How publish handles path deps](#how-publish-handles-path-deps))

---

## Source-repo flexibility

**The registry is repo-agnostic.** Your package source can live anywhere:
- Your own GitHub repo (`github.com/arniwesth/my-extensions`)
- A monorepo with multiple packages (mirrors the [`sunholo-data/ailang-packages`](https://github.com/sunholo-data/ailang-packages) layout)
- A private GitLab/Bitbucket — the registry only sees the tarball you upload via `ailang publish`

A reasonable starting layout for one publisher with multiple packages:

```
github.com/arniwesth/motoko-extensions/
├── packages/
│   ├── motoko-ext-openkb/
│   │   ├── ailang.toml
│   │   ├── register.ail
│   │   └── ...
│   └── motoko-ext-other-thing/
│       └── ...
├── README.md
└── .github/workflows/publish.yml   # optional: auto-publish on tag push
```

You can mirror Sunholo's `motoko-ext-` naming convention so motoko_agent's registry-generator strips the prefix correctly — see the [build-an-extension tutorial](./build-a-motoko-extension.md#step-1-scaffold-the-package) for details.

---

## How `publish` handles path deps

A common confusion: your package depends on another package via `{ path = "../some-other-pkg" }` while you're editing both together. When you publish, those path refs would be invalid for anyone who downloads the tarball — they don't have your local sibling layout.

`ailang publish` handles this automatically:

1. Reads your `ailang.toml`
2. For each `path = ...` dep, looks up the actual published version from the path target's `ailang.toml`
3. Rewrites the dep to `"name" = "X.Y.Z"` in the **tarball's** `ailang.toml` (your local file is restored after upload)
4. Uploads the rewritten tarball

This means you can develop with sibling-checkout convenience and publish without manual edits — but it requires that **every transitive `path =` dependency is also published**. If you publish `motoko-ext-foo` while `motoko-ext-foo` depends on `path = "../motoko-ext-bar"` AND `motoko-ext-bar` isn't yet on the registry, the publish will fail.

**Order matters when releasing a package set:** publish leaves first.

---

## Publishing from CI

Once you're past the manual-publish stage, automate via GitHub Actions:

```yaml
# .github/workflows/publish.yml
name: Publish to AILANG registry
on:
  push:
    tags: ['v*']

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install ailang
        run: |
          curl -sSL https://github.com/sunholo-data/ailang/releases/latest/download/ailang-linux-amd64.tar.gz | tar xz
          sudo mv ailang /usr/local/bin/
      - name: Publish
        env:
          AILANG_REGISTRY_API_KEY: ${{ secrets.AILANG_REGISTRY_API_KEY }}
        run: |
          for pkg in packages/*/; do
            cd $pkg
            ailang publish || echo "skipped $pkg (already at this version?)"
            cd -
          done
```

Store the API key as a repo secret, not in the workflow file.

---

## Verifying your published package

```bash
# Look up your package
ailang pkg info myorg/ext_foo

# Force-fetch latest (bypass local cache)
ailang pkg info myorg/ext_foo --refresh

# In a consumer project, verify the dep resolves from registry
cd ../host-project
ailang lock
jq '[.packages[] | select(.name=="myorg/ext_foo") | {name, version, source}]' ailang.lock
# Expected: { "source": "registry" }
```

The package also auto-appears at `https://ailang.sunholo.com/docs/packages/<namespace>/<name>` after a doc-rebuild cycle (typically under one hour).

---

## Current limitations

These are real today. Known and tracked.

1. **Single shared API key.** No per-namespace scoping at the validator (see [`cmd/registry-validator/main.go:178`](https://github.com/sunholo-data/ailang/blob/dev/cmd/registry-validator/main.go#L178) — `// Step 5: Namespace auth — deferred (accept all publishers for now)`). Whoever holds the key can publish under any namespace. **Acceptable for trusted partners; not yet ready for an open ecosystem.** Tracked as `M-PKG-MULTI-NAMESPACE-AUTH` (planned).

2. **No version yanking yet.** Once published, a version is immutable — no way to retract a broken release. Workaround: publish a fixed `x.y.(z+1)`. Yanking is on the roadmap.

3. **No private registries.** All publishes go to the same Sunholo-hosted GCS bucket. For private packages, keep them as path-deps in a monorepo.

4. **No package owners other than via the API key.** Future namespace scoping will introduce an owner concept, but today there's no per-package ACL — the key is the only authority.

---

## Related guides

- [Build Your First motoko Extension](./build-a-motoko-extension.md) — covers the Tier 1 (local path) workflow end-to-end
- [Extension Packages reference](./extension-packages.md) — manifest schema + generator
- [Path vs registry checklist](./extension-packages.md#path-vs-registry-checklist) — the pre-PR audit you should run on every extension-host project
