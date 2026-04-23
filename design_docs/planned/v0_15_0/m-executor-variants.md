# M-EXECUTOR-VARIANTS: Per-Agent Executor Image Variants

**Status**: Planned
**Target**: v1.1.0
**Priority**: P2 — quality-of-life; current baked-in-Go image is a temporary workaround
**Estimated**: 1–2 days
**Source**: Agent OOM when downloading Go at runtime; different agents need different tool sets

---

## Problem

All agents currently run the same executor image (`agent:latest`). This image must carry
every tool any agent might need. Right now that means Node.js + Claude CLI + Gemini CLI +
Go toolchain — a ~2GB image that takes longer to pull and wastes memory on agents that
only edit ailang packages (which need no Go compiler).

As more agent types are added (Python, Rust, data science), the single image grows
unbounded and cold-start time degrades for all agents.

---

## Solution: Variant field in agent config + dispatcher image override

Each agent in `config.cloud.yaml` declares an `executor_variant`. The dispatcher
looks up the image for that variant from a static map and passes it as a container
override when creating the Cloud Run Job execution. No new job templates needed.

### Variants (initial set)

| Variant | Image tag | Contents | Use case |
|---------|-----------|----------|----------|
| `default` | `agent:latest` | Node + Claude CLI + git + ailang binary | Package authors, docs, web agents |
| `go` | `agent-go:latest` | `default` + Go toolchain | ailang repo, any Go codebase |

New variants can be added without infra changes — just a new Dockerfile and a new entry
in the dispatcher map.

---

## Implementation

### 1. Agent config (`config/config.cloud.yaml`)

Add optional `executor_variant` field per agent. Omitting it defaults to `"default"`.

```yaml
agents:
  sprint-executor:
    executor_variant: go      # needs go build/test
  design-doc-creator:
    executor_variant: default # edits markdown only
  stapledon-executor:
    executor_variant: go
  website-builder:
    executor_variant: default
```

### 2. Coordinator config struct (`internal/coordinator/config.go`)

```go
type AgentConfig struct {
    // ... existing fields ...
    ExecutorVariant string `yaml:"executor_variant"` // "", "default", "go"
}
```

### 3. Dispatcher (`internal/dispatch/cloudrun/dispatcher.go`)

Add variant → image resolution before launching the execution:

```go
var variantImages = map[string]string{
    "":        "agent:latest",
    "default": "agent:latest",
    "go":      "agent-go:latest",
}

func imageForVariant(variant, imageBase string) string {
    tag, ok := variantImages[variant]
    if !ok {
        tag = "agent:latest"
    }
    // imageBase = "europe-west1-docker.pkg.dev/{project}/ailang"
    return imageBase + "/" + tag
}
```

Pass as a container override in the `RunV2JobsService.Run` call:

```go
overrides := &runpb.RunJobRequest_Overrides{
    ContainerOverrides: []*runpb.RunJobRequest_Overrides_ContainerOverride{
        {
            Name:  containerName,
            Image: imageForVariant(params.ExecutorVariant, imageBase),
            Env:   envOverrides,
        },
    },
}
```

### 4. New Dockerfile (`docker/Dockerfile.agent-go`)

Thin layer on top of the default image — just adds the Go toolchain:

```dockerfile
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent:latest

# Go toolchain (matches go.mod version in ailang repo)
COPY --from=golang:1.25-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/ailang/go"
```

This keeps the default image lean and lets `agent-go` track Go version changes
in one place.

### 5. CI: build both images (`cloudbuild-images.yaml` + `cloudbuild-trigger-ailang.yaml`)

Add a parallel build step for `agent-go` alongside the existing `agent` step:

```yaml
- name: 'gcr.io/cloud-builders/docker'
  id: build-agent-go
  waitFor: ['build-agent']   # agent-go FROM agent:latest, so must follow
  args:
    - build
    - --build-arg=PROJECT=${_TARGET_PROJECT}
    - -t
    - ${_REGION}-docker.pkg.dev/${_TARGET_PROJECT}/ailang/agent-go:latest
    - -f
    - /workspace/ailang/docker/Dockerfile.agent-go
    - /workspace/ailang
```

Note the `waitFor: ['build-agent']` — `agent-go` builds FROM the just-pushed `agent:latest`,
so it must come after.

---

## Migration path

1. Revert the Go toolchain addition from `Dockerfile.agent` (keep `agent:latest` lean)
2. Add `Dockerfile.agent-go` as described above
3. Update CI to build both
4. Wire `executor_variant: go` into config for Go-repo agents
5. Deploy — `agent:latest` shrinks back, `agent-go:latest` carries Go

Until step 5 is deployed, the current monolithic image works fine (just oversized).

---

## Non-goals

- Dynamic image selection at task dispatch time by the coordinator (over-engineered)
- Per-task image overrides in the API (security surface, not needed)
- Variant-specific resource limits (handle via separate job templates if ever needed)
