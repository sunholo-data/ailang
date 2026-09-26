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

## What `publish` checks — the quality report (v0.40.0)

`ailang publish` (and `--dry-run`) runs the same quality report the registry validator runs
on upload, and refuses on the same `PUBnnn` codes **before** a tarball is built. You can run
it on its own:

```bash
ailang pkg quality .            # human summary
ailang pkg quality --json .     # schema ailang.package-quality/v1
ailang pkg quality --strict .   # warn-level badges become gates (exit 2)
ailang pkg quality --no-run .   # skip executing tests and _smoke.ail
```

Every section carries a **provenance**:

| Provenance | Sections | Who computes it | Can it block a publish? |
|---|---|---|---|
| `server` | compile · contracts (Z3) · interface identity · effects · release · docs · style | you locally **and** the validator, identically | yes — these are the registry's gates |
| `attested` | tests · `_smoke.ail` | **only your machine** (the validator never executes package code) | locally yes; at the registry never — it is banked with `attested_by` = your key owner and shown as a badge |

Gates vs badges depend on `[stability] level`: at `experimental` only compile failures,
refuted contracts (`PUB006`), release-description problems and identity skew block; at
`stable`/`frozen` the badges (uncontracted exports, tests, effect ceiling) become gates too.

### Every version describes itself

Two things are required from v0.41.0 (badges in v0.40.0):

```toml
# ailang.toml
[release]
kind = "fix"     # security | fix | feature | breaking
```

```markdown
# CHANGELOG.md
## 0.8.2
- token refresh no longer logs the refresh token on failure
```

`ailang init package` scaffolds both. The section for the version being published must
exist and be non-empty (`PUB001`); the kind must be declared (`PUB002`). `CHANGELOG.md`
ships in the tarball; `ailang pkg versions <name>` prints the kind and notes per version.
The kind is your *claim* — a later release of the ladder checks it against the measured
change class, so `security` is the label that is verified hardest.

### Contracts are now counted

The validator verifies `requires`/`ensures` contracts package-wide (`ailang verify
--package .`) and banks `contracts_verified/contracts_total` in `metadata.json` and
`contracts_total` in the index. A refuted contract is a gate everywhere.

### Interface identity (v2)

Alongside the manifest-level `interface_hash`, the validator now banks a
signature-sensitive `interface_hash_v2` and the exported `interface_signatures`. If your
`ailang` computes a different v2 hash than the validator's, the publish is refused with
`PUB005` naming both — upgrade the publisher. A package whose v2 identity cannot be built
gets a badge today (shadow mode); `GET /api/stats` reports `v2_clean_streak`.

### The package inbox

Every published package has an agent inbox, `pkg:<vendor>/<name>`, derived from
`[metadata] repository` — a GitHub tree URL such as
`https://github.com/sunholo-data/ailang-packages/tree/main/packages/gcp-auth`. Without a
parseable URL **no agent is derived** — the repository is not a function of the package name —
and `PUB021` warns until you add it. A `pkg:` inbox for a package that is not in the registry is
served by nothing either — a typo stays visible. See the autonomous-package-updates guide.

---

## Prerequisites

### 1. An API key

Publishing needs a key in `AILANG_REGISTRY_API_KEY`. There are two kinds:

| Kind | Who holds it | Can write |
|---|---|---|
| **Scoped key** (`ailr_…`) | each publisher | only the package globs it was minted with, e.g. `daneel/*` or `sunholo/daneel_*` |
| **Superuser key** | the AILANG maintainers | every namespace, plus `unpublish` of anything, `rebuild-index`, and minting scoped keys |

Ask [the AILANG maintainers](https://github.com/sunholo-data/ailang/issues) (or your existing channel) for a scoped key for your namespace. Reads never need a key — the registry bucket is public.

A maintainer mints one with:

```bash
export AILANG_REGISTRY_API_KEY=<superuser key>
ailang pkg key create --owner daneel --scope 'daneel/*' --note "Daneel's publish key"
# ✓ Minted key for daneel (scopes: daneel/*)
#   id:  3f9c…        ← for `ailang pkg key revoke <id>`
#   key: ailr_…       ← shown ONCE; hand over on a private channel
ailang pkg key list
```

The key MUST be kept private — don't commit it, don't paste it in a PR, don't put it in a GitHub Action without using a secret. A scoped key publishing outside its scope gets a 403 that names the owner and the package; the scopes are enforced on `publish` and `unpublish` alike, and `published_by` in the package metadata is stamped with the key's owner.

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

You can mirror Sunholo's `motoko-ext-` naming convention so motoko_agent's registry-generator strips the prefix correctly — see the [build-an-extension tutorial](./build-a-motoko-extension.md#step-1-scaffold-the-package-one-command) for details.

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

## For maintainers: issuing and managing keys

Keys are minted centrally (CRAN-style) — there is no signup. The superuser key in Secret
Manager (`ailang-registry-api-key`, project `ailang-registry`) is the only thing that can mint.

### What a publisher may write

A key carries one or more **scopes**, globs over the full `vendor/name`. Two shapes cover
the cases we actually have:

| Situation | Scope | Effect |
|---|---|---|
| Their own namespace — they publish whatever they like there | `daneel/*` | any package under `daneel/` |
| A package of ours we hand to them | `sunholo/daneel_tools` | that one package, nothing else under `sunholo/` |
| A family of ours we hand to them | `sunholo/daneel_*` | every `sunholo/daneel_…` package |

Combine with repeated `--scope`:

```bash
ailang pkg key create --owner daneel \
  --scope 'daneel/*' \
  --scope 'sunholo/daneel_tools' \
  --note "Daneel: own namespace + the docs tooling we delegated"
```

Rules of thumb:

- **Namespace = owner name.** The `--owner` you record and the vendor they publish under
  should match (`--owner daneel` ↔ `daneel/*`). `published_by` in metadata is stamped with
  the owner, so provenance reads cleanly.
- **Never hand out `sunholo/*`.** Delegate specific packages or a prefix; the wildcard on our
  own namespace is superuser territory.
- **Nothing stops two keys covering the same package.** Scopes are permissions, not ownership
  records. Before minting a scope inside a namespace someone else already holds, check
  `ailang pkg key list` — it is the only registry of who can write what.

### Changing what someone can publish

There is deliberately no "edit scopes" — mint a replacement and revoke the old one, so the
change is a new key in the holder's hands and an audit row, not a silent widening:

```bash
ailang pkg key list                          # find the id
ailang pkg key create --owner daneel --scope 'daneel/*' --scope 'sunholo/new_thing'
ailang pkg key revoke <old-id>               # after they have switched
```

Revocation is immediate (checked on every write) and permanent — a revoked id cannot be
re-enabled. Keys are stored as SHA-256 only; a lost key is re-minted, never recovered.

### The superuser key

- Existing holders keep working unchanged. It is still the break-glass and the minting authority.
- **Rotate it once every current holder has a scoped key** — the point of scoping is that only
  maintainers hold `*`. Rotation: add a new version to the `ailang-registry-api-key` secret in
  `ailang-registry`, mirror it into `ailang-multivac`'s copy (recipe in that repo's
  `terraform/secrets.tf`), then roll the validator revision so the new version is read — the
  env is resolved at instance start, not on every request.
- The fleet's own publish jobs (`ailang-multivac/terraform/cloud_run_jobs.tf`) currently use
  the superuser key. They should be moved to a scoped `sunholo/*`-shaped key before rotation
  so a leaked job env cannot mint keys.

## Current limitations

These are real today. Known and tracked.

1. **No self-registration.** Only a maintainer can mint a scoped key (CRAN-style curation); there is no signup flow. Scopes are globs, not ownership records — two keys can be minted over the same namespace, and nothing stops a maintainer from doing so.

2. **No version yanking yet.** Once published, a version is immutable — no way to retract a broken release. Workaround: publish a fixed `x.y.(z+1)`. Yanking is on the roadmap.

3. **No private registries.** All publishes go to the same Sunholo-hosted GCS bucket. For private packages, keep them as path-deps in a monorepo.

4. **The key is the only identity.** `published_by` records the key's owner, but there is no owner concept beyond that — no transfer, no co-owners, no per-package ACL other than the key scopes.

---

## Related guides

- [Build Your First motoko Extension](./build-a-motoko-extension.md) — covers the Tier 1 (local path) workflow end-to-end
- [Extension Packages reference](./extension-packages.md) — manifest schema + generator
- [Path vs registry checklist](./extension-packages.md#path-vs-registry-checklist) — the pre-PR audit you should run on every extension-host project
