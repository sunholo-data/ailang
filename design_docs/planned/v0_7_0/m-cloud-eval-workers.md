# M-CLOUD-EVAL: Distributed Cloud Evaluation Workers

**Status**: Planned
**Target**: v0.6.2+
**Priority**: P1 - Medium-High
**Created**: 2025-12-29
**Part of**: [Global Collaboration Hub](global-collaboration-hub.md)

**Dependencies**:
- M-AI-OLLAMA (v0.6.2) - COMPLETE - Unified AI providers + Ollama support
- Global Collaboration Hub infrastructure - IN PROGRESS
- `internal/eval_harness/` - COMPLETE - Local eval framework

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Job specs immutable, results reproducible with seeds |
| A2: Replayability | +1 | Full job history persisted in Firestore |
| A3: Effect Legibility | 0 | Infrastructure feature, not language-level |
| A4: Explicit Authority | +1 | IAM-based access, workers have scoped permissions |
| A5: Bounded Verification | +1 | Each job validates compile/runtime/output independently |
| A6: Safe Concurrency | +1 | Pub/Sub guarantees at-least-once, deduplication via job ID |
| A7: Machines First | +1 | JSON job specs, no human prose in execution path |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Detailed cost tracking per job + infrastructure costs |
| A10: Composability | +1 | Extends existing eval harness, reuses Hub infrastructure |
| A11: Structured Failure | +1 | Job failures isolated, retryable, categorized |
| A12: System Boundary | +1 | Clear local/cloud boundary via mode flag |

**Net Score: +10** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Jobs are reproducible (seed, prompt version tracked)
- [x] A3 (Effects): No hidden effects; all cloud operations explicit
- [x] A4 (Authority): Workers can't access anything outside their scope
- [x] A7 (Machines First): All communication via structured JSON

---

## Problem Statement

**Current State (v0.6.2)**:
- `ailang eval-suite` runs locally with semaphore-based parallelization (default: 10 concurrent)
- All models (OpenAI, Anthropic, Gemini, Ollama) execute on local machine
- Local Ollama models are **very slow** without GPU (minutes per benchmark)
- No distributed execution - limited to single machine's resources
- API rate limits hit when running from single source IP

**Bottlenecks Identified**:

| Bottleneck | Impact | Cloud Solution |
|------------|--------|----------------|
| **Local Ollama speed** | 5-10 min/benchmark | GPU-accelerated Cloud Run Jobs |
| **Single-source API limits** | Rate limiting errors | Multiple worker IPs |
| **Serial execution** | Long baseline runs (4+ hours) | Horizontal scaling to 50+ workers |
| **No crash recovery** | Lost progress on failure | Persistent job queue + checkpointing |
| **Resource contention** | Blocks local development | Offload to cloud |

**Benchmark Numbers (Local vs Projected Cloud)**:

| Model Type | Local (M1 Mac) | Cloud (T4 GPU) | Speedup |
|------------|----------------|----------------|---------|
| Ollama codellama (7B) | 180s/benchmark | 15s/benchmark | **12x** |
| Ollama deepseek-coder (6.7B) | 200s/benchmark | 18s/benchmark | **11x** |
| Cloud APIs (GPT-5, Claude) | 3-5s/benchmark | 3-5s/benchmark | 1x (same) |
| **Full baseline (264 jobs)** | 4-6 hours | 15-20 min | **15-20x** |

---

## Goals

**Primary Goal**: Enable distributed, GPU-accelerated evaluation runs using the Global Collaboration Hub infrastructure.

**Success Metrics**:
- Full baseline (264 benchmarks × 6 models) completes in <30 minutes
- Ollama model benchmarks run 10x+ faster with GPU
- Zero job loss during worker failures (at-least-once delivery)
- Cost under $5 for a full baseline run
- Seamless fallback to local execution when cloud unavailable

---

## Solution Design

### Overview

Extend the Global Collaboration Hub with **Eval Workers** - specialized Cloud Run Jobs that execute benchmark evaluations.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         CLOUD EVAL ARCHITECTURE                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  LOCAL (ailang eval-suite --cloud)                                              │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Job Orchestrator                                                          │  │
│  │  ├── Generate job specs from benchmarks/*.yml                              │  │
│  │  ├── Publish jobs to Pub/Sub                                               │  │
│  │  ├── Monitor progress via Firestore                                        │  │
│  │  └── Aggregate results when complete                                       │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                              │                                                   │
│                              ▼                                                   │
│  CLOUD                                                                          │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                        Pub/Sub: eval-jobs                                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │  │
│  │  │ Job: fizz   │  │ Job: fact   │  │ Job: tree   │  │ Job: ...    │       │  │
│  │  │ gpt5-mini   │  │ ollama:cll  │  │ claude-4.5  │  │             │       │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘       │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                              │                                                   │
│       ┌──────────────────────┼──────────────────────┐                           │
│       ▼                      ▼                      ▼                            │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐                   │
│  │ Worker Pool  │      │ Worker Pool  │      │ Worker Pool  │                   │
│  │ (CPU-only)   │      │ (GPU: T4)    │      │ (GPU: L4)    │                   │
│  │              │      │              │      │              │                   │
│  │ • OpenAI     │      │ • Ollama 7B  │      │ • Ollama 70B │                   │
│  │ • Anthropic  │      │ • Ollama 13B │      │ • Large LLMs │                   │
│  │ • Gemini     │      │              │      │              │                   │
│  │              │      │              │      │              │                   │
│  │ 20 workers   │      │ 10 workers   │      │ 2 workers    │                   │
│  └──────────────┘      └──────────────┘      └──────────────┘                   │
│                              │                                                   │
│                              ▼                                                   │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                        Result Storage                                      │  │
│  │  ┌─────────────────────────┐    ┌─────────────────────────────────────┐   │  │
│  │  │ Firestore               │    │ Cloud Storage                        │   │  │
│  │  │ • Job status            │    │ • Full result JSON                   │   │  │
│  │  │ • Progress tracking     │    │ • Generated code                     │   │  │
│  │  │ • Aggregated metrics    │    │ • Execution logs                     │   │  │
│  │  └─────────────────────────┘    └─────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

#### 1. Job-Level Parallelization (Not Batch-Level)

**Decision**: Each job (1 benchmark × 1 model × 1 language) is an independent unit.

**Why**:
- Maximum parallelization (264 jobs can run across 50+ workers)
- Fine-grained failure isolation (1 job fails, others continue)
- Natural load balancing (fast jobs don't wait for slow jobs)
- Easy retry logic (retry individual failed jobs)

**Job Spec**:
```json
{
  "job_id": "eval_v0.6.2_fizzbuzz_ailang_ollama-codellama_1703847600",
  "version": "1.0",
  "eval_version": "v0.6.2",

  "benchmark": {
    "id": "fizzbuzz",
    "language": "ailang",
    "prompt_file": "benchmarks/fizzbuzz/prompt_ailang.md",
    "expected_output_hash": "sha256:abc123..."
  },

  "model": {
    "id": "ollama-codellama",
    "provider": "ollama",
    "model_name": "codellama:7b-instruct",
    "requires_gpu": true,
    "min_vram_gb": 8
  },

  "config": {
    "timeout_seconds": 300,
    "self_repair": true,
    "max_repair_attempts": 3,
    "seed": 42
  },

  "routing": {
    "pool": "gpu-t4",  // or "cpu-only", "gpu-l4"
    "priority": "normal"  // or "high", "low"
  },

  "metadata": {
    "created_at": "2025-12-29T10:00:00Z",
    "created_by": "local-orchestrator",
    "baseline_run": true
  }
}
```

#### 2. Three Worker Pools by Hardware

**Why separate pools**:
- Cloud APIs (OpenAI, Anthropic, Gemini) don't need GPU - cheaper workers
- Small Ollama models (7B-13B) run well on T4 GPUs ($0.35/hr)
- Large Ollama models (70B) need L4/A100 GPUs ($1.25+/hr)

**Pool Configuration**:
```yaml
worker_pools:
  cpu-only:
    description: "Cloud API models (OpenAI, Anthropic, Gemini)"
    cloud_run_job: "eval-worker-cpu"
    resources:
      cpu: "2"
      memory: "4Gi"
    max_workers: 30
    models:
      - gpt5
      - gpt5-mini
      - gpt5-2
      - claude-sonnet-4-5
      - claude-haiku-4-5
      - gemini-2-5-pro
      - gemini-2-5-flash
    cost_per_hour: 0.05  # Cloud Run CPU pricing

  gpu-t4:
    description: "Small Ollama models (7B-13B parameters)"
    cloud_run_job: "eval-worker-gpu-t4"
    resources:
      cpu: "4"
      memory: "16Gi"
      gpu: "nvidia-tesla-t4"
      gpu_count: 1
    max_workers: 10
    models:
      - ollama-codellama      # 7B
      - ollama-deepseek-coder # 6.7B
      - ollama-qwen-coder     # 7B
      - ollama-mistral        # 7B
    cost_per_hour: 0.35  # Cloud Run GPU pricing

  gpu-l4:
    description: "Large Ollama models (30B-70B parameters)"
    cloud_run_job: "eval-worker-gpu-l4"
    resources:
      cpu: "8"
      memory: "32Gi"
      gpu: "nvidia-l4"
      gpu_count: 1
    max_workers: 5
    models:
      - ollama-codellama-70b
      - ollama-deepseek-coder-33b
    cost_per_hour: 1.25  # Cloud Run L4 pricing
```

#### 3. Ollama as Sidecar (Not Shared Service)

**Decision**: Each GPU worker runs Ollama as a sidecar container.

**Why**:
- No cold-start model loading delays (model pre-loaded in container)
- Isolated GPU memory per worker
- Stateless workers - easy horizontal scaling
- No shared Ollama service bottleneck

**Container Architecture**:
```yaml
# eval-worker-gpu-t4.yaml
apiVersion: run.googleapis.com/v1
kind: Job
metadata:
  name: eval-worker-gpu-t4
spec:
  template:
    spec:
      containers:
        # Main worker container
        - name: eval-worker
          image: gcr.io/ailang/eval-worker:latest
          env:
            - name: OLLAMA_HOST
              value: "http://localhost:11434"
            - name: WORKER_POOL
              value: "gpu-t4"
          resources:
            limits:
              cpu: "4"
              memory: "8Gi"

        # Ollama sidecar with pre-loaded model
        - name: ollama
          image: gcr.io/ailang/ollama-codellama:7b-instruct
          resources:
            limits:
              cpu: "2"
              memory: "8Gi"
              nvidia.com/gpu: "1"
          volumeMounts:
            - name: ollama-models
              mountPath: /root/.ollama

      nodeSelector:
        cloud.google.com/gke-accelerator: nvidia-tesla-t4

      volumes:
        - name: ollama-models
          emptyDir: {}

      timeoutSeconds: 600  # 10 min per job
      serviceAccountName: eval-worker-sa
```

**Pre-baked Model Images**:
```dockerfile
# Dockerfile.ollama-codellama
FROM ollama/ollama:latest

# Pre-pull model during build (cached in image)
RUN ollama pull codellama:7b-instruct

# Keep container running for sidecar pattern
CMD ["serve"]
```

#### 4. Result Flow: Pub/Sub → Worker → Firestore + GCS

**Sequence**:
```
1. Orchestrator publishes job to Pub/Sub (eval-jobs topic)
2. Worker pulls job from subscription
3. Worker executes benchmark
4. Worker writes result to:
   - Firestore: Status, metrics (for real-time tracking)
   - Cloud Storage: Full result JSON (for archival)
5. Worker acks message
6. Orchestrator polls Firestore for completion
7. Orchestrator downloads results from GCS
```

**Firestore Schema** (extends Hub schema):
```
hub/{project-id}/
├── eval_runs/{run-id}
│   ├── status: "running" | "completed" | "failed"
│   ├── version: "v0.6.2"
│   ├── created_at: timestamp
│   ├── completed_at: timestamp
│   ├── total_jobs: 264
│   ├── completed_jobs: 180
│   ├── failed_jobs: 2
│   └── cost_usd: 2.34
│
└── eval_jobs/{job-id}
    ├── run_id: "run_v0.6.2_baseline_1703847600"
    ├── status: "pending" | "running" | "completed" | "failed"
    ├── pool: "gpu-t4"
    ├── worker_id: "worker-abc123"
    ├── started_at: timestamp
    ├── completed_at: timestamp
    ├── result_gcs_path: "gs://ailang-evals/results/..."
    ├── metrics: {
    │     compile_ok: true,
    │     runtime_ok: true,
    │     stdout_ok: false,
    │     duration_ms: 15234,
    │     tokens_input: 2048,
    │     tokens_output: 512,
    │     cost_usd: 0.0
    │   }
    └── error: null | "timeout" | "compile_error" | ...
```

**Cloud Storage Structure**:
```
gs://ailang-evals/
├── results/
│   └── {run-id}/
│       └── {job-id}.json          # Full RunMetrics JSON
├── code/
│   └── {run-id}/
│       └── {job-id}.ail           # Generated code
└── logs/
    └── {run-id}/
        └── {job-id}.log           # Execution logs
```

---

### CLI Integration

**New flags for `ailang eval-suite`**:

```bash
# Run evaluation in cloud
ailang eval-suite \
  --cloud \                          # Enable cloud execution
  --cloud-project sunholo-data \     # GCP project
  --cloud-region us-central1 \       # Region for workers
  --parallel 50 \                    # Max concurrent workers
  --output eval_results/v0.6.2 \     # Local output (downloaded from GCS)
  --models gpt5-mini,ollama-codellama \
  --benchmarks fizzbuzz,factorial

# Monitor running cloud eval
ailang eval-suite --cloud --status   # Show progress

# Resume failed jobs
ailang eval-suite --cloud --resume run_v0.6.2_baseline_1703847600

# List recent cloud runs
ailang eval-suite --cloud --list

# Download results from completed run
ailang eval-suite --cloud --download run_v0.6.2_baseline_1703847600 \
  --output eval_results/v0.6.2
```

**Implementation in `cmd/ailang/eval_suite.go`**:

```go
// Cloud eval flags
cloud := fs.Bool("cloud", false, "Run evaluation in cloud (requires GCP setup)")
cloudProject := fs.String("cloud-project", "", "GCP project ID")
cloudRegion := fs.String("cloud-region", "us-central1", "Cloud region")
cloudStatus := fs.Bool("status", false, "Show cloud run status")
cloudResume := fs.String("resume", "", "Resume run ID")
cloudDownload := fs.String("download", "", "Download results from run ID")
cloudList := fs.Bool("list", false, "List recent cloud runs")

if *cloud {
    // Initialize cloud orchestrator
    orchestrator, err := cloud_eval.NewOrchestrator(cloud_eval.Config{
        ProjectID:    *cloudProject,
        Region:       *cloudRegion,
        MaxWorkers:   *parallel,
        OutputDir:    *outputDir,
    })
    if err != nil {
        log.Fatalf("Failed to initialize cloud eval: %v", err)
    }

    if *cloudStatus {
        return orchestrator.ShowStatus()
    }
    if *cloudList {
        return orchestrator.ListRuns()
    }
    if *cloudDownload != "" {
        return orchestrator.DownloadResults(*cloudDownload)
    }
    if *cloudResume != "" {
        return orchestrator.ResumeRun(*cloudResume)
    }

    // Start new cloud eval run
    runID, err := orchestrator.StartRun(specs, models, langs)
    if err != nil {
        log.Fatalf("Failed to start cloud eval: %v", err)
    }

    // Wait for completion with progress updates
    return orchestrator.WaitForCompletion(runID)
}
```

---

### Worker Implementation

**`internal/cloud_eval/worker/worker.go`**:

```go
package worker

import (
    "context"
    "encoding/json"

    "cloud.google.com/go/pubsub"
    "cloud.google.com/go/firestore"
    "cloud.google.com/go/storage"

    "github.com/sunholo-data/ailang/internal/ai"
    "github.com/sunholo-data/ailang/internal/eval_harness"
)

type Worker struct {
    projectID    string
    pool         string
    workerID     string

    pubsub       *pubsub.Client
    firestore    *firestore.Client
    storage      *storage.Client

    aiProvider   ai.Provider
    runner       eval_harness.Runner
}

func (w *Worker) Run(ctx context.Context) error {
    sub := w.pubsub.Subscription(fmt.Sprintf("eval-jobs-%s", w.pool))

    return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
        var job JobSpec
        if err := json.Unmarshal(msg.Data, &job); err != nil {
            msg.Nack()
            return
        }

        // Update status to running
        w.updateJobStatus(ctx, job.JobID, "running")

        // Execute benchmark
        result, err := w.executeBenchmark(ctx, &job)
        if err != nil {
            w.updateJobStatus(ctx, job.JobID, "failed", err.Error())
            msg.Ack() // Don't retry - mark as failed
            return
        }

        // Store result
        if err := w.storeResult(ctx, &job, result); err != nil {
            msg.Nack() // Retry
            return
        }

        // Update status to completed
        w.updateJobStatus(ctx, job.JobID, "completed")

        msg.Ack()
    })
}

func (w *Worker) executeBenchmark(ctx context.Context, job *JobSpec) (*eval_harness.RunMetrics, error) {
    // Load benchmark spec
    spec, err := eval_harness.LoadBenchmarkSpec(job.Benchmark.ID)
    if err != nil {
        return nil, fmt.Errorf("load spec: %w", err)
    }

    // Get AI provider for model
    provider, err := ai.GetProvider(job.Model.Provider)
    if err != nil {
        return nil, fmt.Errorf("get provider: %w", err)
    }

    // Create agent
    agent := eval_harness.NewAIAgent(job.Model.ID, provider)

    // Get language runner
    runner, err := eval_harness.GetRunner(job.Benchmark.Language)
    if err != nil {
        return nil, fmt.Errorf("get runner: %w", err)
    }

    // Generate code
    prompt := spec.PromptForLanguage(job.Benchmark.Language)
    code, genMetrics, err := agent.GenerateCode(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("generate code: %w", err)
    }

    // Run code
    execResult, err := runner.Run(ctx, code, job.Config.TimeoutSeconds)
    if err != nil {
        // Don't fail on execution errors - record them
    }

    // Build metrics
    metrics := &eval_harness.RunMetrics{
        ID:            job.Benchmark.ID,
        Lang:          job.Benchmark.Language,
        Model:         job.Model.ID,
        InputTokens:   genMetrics.InputTokens,
        OutputTokens:  genMetrics.OutputTokens,
        CompileOK:     execResult.CompileOK,
        RuntimeOK:     execResult.RuntimeOK,
        StdoutOK:      execResult.StdoutOK,
        DurationMS:    execResult.DurationMS,
        GeneratedCode: code,
        Stderr:        execResult.Stderr,
        Stdout:        execResult.Stdout,
    }

    // Self-repair if enabled
    if job.Config.SelfRepair && !metrics.StdoutOK {
        repairRunner := eval_harness.NewRepairRunner(agent, runner, job.Config.MaxRepairAttempts)
        metrics, err = repairRunner.Repair(ctx, spec, metrics)
    }

    return metrics, nil
}
```

---

### Cost Model

**Per-Run Cost Breakdown** (264 jobs, full baseline):

| Component | Count | Unit Cost | Total |
|-----------|-------|-----------|-------|
| **CPU Workers** (cloud APIs) | 180 jobs × 5s avg | $0.05/hr = $0.000014/s | $0.01 |
| **GPU-T4 Workers** (small Ollama) | 60 jobs × 15s avg | $0.35/hr = $0.000097/s | $0.09 |
| **GPU-L4 Workers** (large Ollama) | 24 jobs × 30s avg | $1.25/hr = $0.000347/s | $0.25 |
| **Pub/Sub** | 264 msgs × 2 | $0.04/100K msgs | $0.00 |
| **Firestore** | 528 writes + 1000 reads | $0.18/100K ops | $0.00 |
| **Cloud Storage** | 264 × 5KB results | $0.02/GB | $0.00 |
| **API costs** (OpenAI/Anthropic/Gemini) | Varies by model | See models.yml | $2-5 |
| **Total Infrastructure** | | | **~$0.35** |
| **Total with API costs** | | | **$2.50-5.50** |

**Comparison**:
| Mode | Time | Cost | Notes |
|------|------|------|-------|
| Local (no GPU) | 4-6 hours | API only ($2-5) | Blocks machine |
| Local (with GPU) | 2-3 hours | API only ($2-5) | Requires GPU hardware |
| **Cloud** | **15-30 min** | **$2.50-5.50** | Fast + doesn't block |

---

### Implementation Plan

**Phase 1: Core Infrastructure** (~8 hours)
- [ ] Create `internal/cloud_eval/` package structure
- [ ] Implement `Orchestrator` - job generation, Pub/Sub publishing
- [ ] Implement `Worker` - job execution, result storage
- [ ] Add Firestore schema for eval_runs and eval_jobs
- [ ] Create Cloud Storage bucket structure

**Phase 2: Worker Containers** (~6 hours)
- [ ] Create base `eval-worker` Dockerfile
- [ ] Create `ollama-codellama:7b-instruct` pre-baked image
- [ ] Create `ollama-deepseek-coder:6.7b` pre-baked image
- [ ] Configure Cloud Run Jobs with GPU support
- [ ] Test sidecar pattern locally with Docker Compose

**Phase 3: CLI Integration** (~4 hours)
- [ ] Add `--cloud` flag to `eval-suite`
- [ ] Implement `--status`, `--list`, `--download` commands
- [ ] Add progress bar for cloud execution
- [ ] Implement `--resume` for failed runs

**Phase 4: Monitoring & Polish** (~4 hours)
- [ ] Add Cloud Monitoring dashboards
- [ ] Implement cost tracking per run
- [ ] Add timeout handling and dead letter queue
- [ ] Write integration tests

**Phase 5: Documentation** (~2 hours)
- [ ] Update eval guide with cloud setup
- [ ] Document cost model
- [ ] Add troubleshooting guide

---

### Files to Create

```
internal/cloud_eval/
├── orchestrator.go     # Job generation, progress tracking (~300 LOC)
├── pubsub.go           # Pub/Sub client wrapper (~150 LOC)
├── firestore.go        # Firestore state management (~200 LOC)
├── storage.go          # Cloud Storage result handling (~100 LOC)
├── config.go           # Worker pool configuration (~80 LOC)
└── worker/
    ├── worker.go       # Main worker loop (~250 LOC)
    ├── executor.go     # Benchmark execution (~200 LOC)
    └── main.go         # Worker entrypoint (~50 LOC)

infra/eval/
├── terraform/
│   ├── main.tf         # Core infrastructure (~150 LOC)
│   ├── pubsub.tf       # Pub/Sub topics/subs (~50 LOC)
│   ├── cloud_run.tf    # Cloud Run Jobs (~200 LOC)
│   └── variables.tf    # Configuration (~50 LOC)
└── docker/
    ├── Dockerfile.worker           # Base worker image
    ├── Dockerfile.ollama-codellama # Pre-baked Ollama
    └── docker-compose.yml          # Local testing
```

**Estimated Total**: ~1,500 LOC Go + ~500 LOC Terraform/Docker

---

### Relationship to Other Components

**Reuses from Global Collaboration Hub**:
- Pub/Sub infrastructure (same project, different topics)
- Firestore database (extends schema)
- Cloud Storage bucket structure
- IAM roles and service accounts
- Authentication flow

**Independent from Coordinator**:
- Eval workers are simpler (no git worktrees needed)
- No human-in-the-loop approval (automated benchmarks)
- Different workflow (batch processing vs. task execution)
- Separate Cloud Run Jobs (can run concurrently)

**Extends Eval Harness**:
- Same `BenchmarkSpec` format
- Same `RunMetrics` output
- Same language runners (Python, AILANG)
- Adds distributed execution layer

---

### Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| GPU quota limits | Medium | Blocks scaling | Request quota increase, use multiple regions |
| Ollama sidecar startup time | Medium | Slow jobs | Pre-warm containers, keep minimum instances |
| API rate limits | Low | Slows execution | Distribute across workers/IPs |
| Cold start latency | Medium | First job slow | Keep warm pool, minimize container size |
| Cost overruns | Low | Budget exceeded | Implement budget limits, alerts |
| Result loss | Low | Need re-run | Ack only after GCS write confirmed |

---

### Open Questions

1. **Model pre-loading strategy**: Should we pre-load all models in all GPU containers, or have model-specific containers?
   - Current choice: Model-specific containers for faster startup

2. **Spot instances**: Should GPU workers use preemptible/spot instances for cost savings?
   - Trade-off: 60-90% cheaper but can be interrupted

3. **Multi-region**: Should we distribute workers across regions for API rate limit avoidance?
   - Adds complexity but may be needed for large-scale runs

4. **Result caching**: Should we cache results for identical (benchmark, model, prompt_version) combinations?
   - Would speed up re-runs but complicates cache invalidation

---

### Success Criteria

- [ ] `ailang eval-suite --cloud` completes 264-job baseline in <30 minutes
- [ ] Ollama models run 10x+ faster than local (non-GPU) execution
- [ ] Zero job loss after simulated worker failure
- [ ] Cost per full baseline under $6 (including API costs)
- [ ] Dashboard shows real-time progress
- [ ] Seamless fallback to local execution when `--cloud` omitted

---

**Document created**: 2025-12-29
**Last updated**: 2025-12-29
