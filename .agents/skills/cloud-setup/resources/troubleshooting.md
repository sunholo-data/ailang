# Cloud Setup Troubleshooting

Symptoms seen in Anthropic-hosted Claude Code cloud sessions, with the fix. Run
`scripts/verify.sh` first — it names most of these directly.

## `ailang: command not found`

`make quick-install` installs to `$(go env GOPATH)/bin` and links `~/.local/bin`; neither is
on the container's PATH. `setup.sh` links `/usr/local/bin/ailang`. Rerun it, or:

```bash
make quick-install && ln -sf "$(go env GOPATH)/bin/ailang" /usr/local/bin/ailang
```

## `ailang --version` says `AILANG dev`

The binary was built without the version ldflags (`go build ./cmd/ailang`). Use
`make quick-install` or `make build`, which pass `LDFLAGS` from the Makefile.

## GCP: `CREDENTIALS_MISSING` / `No credentialed accounts`

The container has no GCP identity of its own. Add `GCP_SA_KEY_JSON` (full service-account key
JSON) to the environment's secrets and rerun `setup.sh`. It writes the key to
`~/.config/gcloud/application_default_credentials.json` — the ADC well-known path, which the
Go client libraries read with no further config — and activates it for gcloud/bq/gsutil.

The harness sets a `CLOUDSDK_AUTH_ACCESS_TOKEN` placeholder with no GCP identity behind it:
`gcloud auth print-access-token` succeeds, every API call fails `CREDENTIALS_MISSING`
(measured 2026-10-16). gcloud prefers that variable over an activated account, so if gcloud
still fails after activation, test with `env -u CLOUDSDK_AUTH_ACCESS_TOKEN gcloud ...`. The Go
clients (and so `ailang messages`) read ADC, not that variable. (Not yet confirmed with a real
key — record the result here once it is.)

## `ailang messages list` shows nothing / no `store: gcp (...)` header

You are reading the container's empty local SQLite. Set `AILANG_STORAGE_MESSAGING=gcp` and
`AILANG_MESSAGES_PROJECT=ailang-multivac` in the environment, and make sure the GCP credential
above works. `AILANG_MESSAGES_STORE` and friends are retired and hard-error by design.

## `go: downloading go1.x ... toolchain not available` / `requires go >= 1.x`

Something set `GOTOOLCHAIN=local` (the old version of this skill did). Unset it; Go's default
fetches the toolchain `go.mod` asks for through proxy.golang.org.

## `go mod download` stalls or `stream error ... INTERNAL_ERROR`

proxy.golang.org resets HTTP/2 streams under load (seen from Cloud Build 2026-09-29). Use
`GOPROXY='https://proxy.golang.org|direct'` — the `|` falls back on any error, a `,` only on
404/410 — and retry. `GOPROXY=direct` alone runs into GitHub's anonymous clone throttle.

## TLS errors, 403/407 from the proxy, transfers cut off

All egress goes through the agent proxy. Read `/root/.ccr/README.md` and
`curl -sS "$HTTPS_PROXY/__agentproxy/status"`. Never disable TLS verification or unset
`HTTPS_PROXY`. A host denied outright means the environment's network level is too narrow —
set it to Full.

## `ailang ai-check` passes but verified nothing

z3 is missing, and contract verification is skipped silently. `apt-get install -y z3`
(`setup.sh` does this).

## The Stop hook pushed / rebased `dev`

`scripts/hooks/push_dev_on_stop.sh` runs in cloud sessions too. Set `AILANG_AUTOPUSH=0` and
`AILANG_AUTOREBASE=0` in the environment and push a branch deliberately.

## Disk full (`no space left on device`)

The writable disk is a fixed per-session allowance, so `df` is misleading. Delete build
artifacts and caches (`go clean -cache`, `bin/`, `node_modules/`), then retry.

## Quick diagnostics

```bash
.claude/skills/cloud-setup/scripts/verify.sh
go env GOVERSION GOTOOLCHAIN GOPROXY GOPATH
gcloud auth list && gcloud config list
ailang storage status
```
