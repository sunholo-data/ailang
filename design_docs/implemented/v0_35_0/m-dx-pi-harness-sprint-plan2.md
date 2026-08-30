# Sprint Plan: M-DX-PI-HARNESS-P2 — Distribution v2: ailang pi install

## Summary
Ship the binary channel: embed the nine extensions plus README in the ailang binary (`pi_assets`), add `ailang pi install|uninstall|status` mirroring `ailang editor install vscode`, managed-file manifest for ~/.pi/agent/extensions, fleet image hook (closes F5).

**Design doc:** m-dx-pi-harness.md — Distribution v2 (ratified 2026-08-28)
**Duration:** 1 session (~400 LOC + tests)
**Risk Level:** Low — writes confined to ~/.pi/agent/extensions under a managed manifest

## Proposed Milestones

### M1: Embedded assets + install/uninstall/status (~2h)
**Estimated:** ~200 LOC
- `cmd/ailang/pi_assets/` — committed copy of .pi/extensions (non-dot files)
- `cmd/ailang/pi_setup.go`: install/uninstall/status + managed manifest (.ailang-managed.json: per-file sha256 + binary version); conflict policy (unmanaged/modified → .ailang-suggested/ + warning, never clobber)
- Go tests mirroring editor_vscode_test.go style (plan computation, conflict, idempotence)

**Acceptance criteria:**
- [x] install → files in ~/.pi/agent/extensions + manifest; re-run idempotent
- [x] user-modified managed file → .ailang-suggested/ copy + warning, original preserved
- [x] uninstall removes only managed files
- [x] status reports FRESH/DRIFT/UNMANAGED/MISSING per file

### M2: Makefile sync + CI drift-check (~0.5h)
**Estimated:** ~20 LOC
**Tasks:** `make pi-assets` (copy .pi/extensions → cmd/ailang/pi_assets, committed) + `make verify-pi-assets` drift check (diff -r, excluding dotfiles)

**Acceptance criteria:**
- [x] make pi-assets idempotent; verify-pi-assets catches a seeded drift

### M3: Fleet + docs (~0.5h)
**Estimated:** ~15 LOC
**Tasks:** Dockerfile.agent-pi adds `RUN ailang pi install`; README/CHANGELOG; design-doc distribution section updated to shipped

**Acceptance criteria:**
- [x] Dockerfile change present (image rebuild/publish stays human-owned)
- [x] README distribution guide rewritten around the binary channel
- [x] CHANGELOG entry
