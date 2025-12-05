# M-AGENT-MONITOR: Real-Time Agent & Process Monitoring

## Problem Statement

On November 29, 2025, two eval processes ran for 7+ hours consuming 300% CPU combined before being manually discovered. The existing UI had no visibility into:
- Running processes and their resource usage
- Claude session telemetry (tokens, cost, turns)
- Duration of running tasks

Without monitoring, runaway processes go undetected until the machine slows down.

## Solution Overview

Add a **Monitor tab** to the Collaboration Hub UI that displays real-time metrics:

| Metric | Source | Update Interval |
|--------|--------|-----------------|
| CPU % | OS process polling | 2s |
| Memory MB | OS process polling | 2s |
| Duration | Process start time | 1s |
| Turns/Steps | Claude NDJSON stream | Real-time |
| Tokens (in/out) | Claude NDJSON stream | Real-time |
| Cost ($) | Claude NDJSON stream | Real-time |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Collaboration Hub UI (Monitor Tab)                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Process 1   │  │ Process 2   │  │ + more...   │         │
│  │ CPU: 45%    │  │ CPU: 12%    │  │             │         │
│  │ Mem: 120MB  │  │ Mem: 85MB   │  │             │         │
│  │ Time: 2m30s │  │ Time: 45s   │  │             │         │
│  │ Turns: 5    │  │ Turns: 2    │  │             │         │
│  │ Cost: $0.12 │  │ Cost: $0.03 │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
         │                                      ▲
         │ Poll every 2s                        │ WebSocket events
         ▼                                      │
┌─────────────────────────────────────────────────────────────┐
│  ailang serve (HTTP Server)                                 │
│  ┌─────────────────────┐  ┌─────────────────────────────┐  │
│  │ GET /api/monitor    │  │ WebSocket /ws               │  │
│  │ - List processes    │  │ - telemetry_update events   │  │
│  │ - CPU/mem stats     │  │ - process_started           │  │
│  │ - Basic telemetry   │  │ - process_stopped           │  │
│  └─────────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
         │                                      ▲
         │ os.FindProcess                       │ Parse NDJSON
         ▼                                      │
┌─────────────────────────────────────────────────────────────┐
│  OS Process Table             Claude Subprocess Streams     │
│  - ailang-agent PIDs          - stdout NDJSON parsing      │
│  - ailang run PIDs            - TokenUsage, TotalCostUSD   │
└─────────────────────────────────────────────────────────────┘
```

## Implementation

### Phase 1: Process Stats API (Immediate)

Add `/api/monitor` endpoint that returns process stats:

```go
// internal/server/monitor.go

type ProcessStats struct {
    InstanceID  string    `json:"instance_id"`
    PID         int       `json:"pid"`
    StartedAt   time.Time `json:"started_at"`
    DurationSec int       `json:"duration_sec"`
    CPUPercent  float64   `json:"cpu_percent"`
    MemoryMB    float64   `json:"memory_mb"`
    Status      string    `json:"status"` // running, completed, failed
}

type MonitorResponse struct {
    Timestamp time.Time      `json:"timestamp"`
    Processes []ProcessStats `json:"processes"`
}

// GET /api/monitor
func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
    stats := s.collectProcessStats()
    json.NewEncoder(w).Encode(MonitorResponse{
        Timestamp: time.Now(),
        Processes: stats,
    })
}

// collectProcessStats gathers CPU/memory for tracked processes
func (s *Server) collectProcessStats() []ProcessStats {
    s.agentsMu.RLock()
    defer s.agentsMu.RUnlock()

    var stats []ProcessStats
    for _, agent := range s.agents {
        stat := ProcessStats{
            InstanceID:  agent.InstanceID,
            PID:         agent.PID,
            StartedAt:   agent.StartedAt,
            DurationSec: int(time.Since(agent.StartedAt).Seconds()),
            Status:      "running",
        }

        // Get CPU/memory from /proc or ps command
        cpu, mem := getProcessResourceUsage(agent.PID)
        stat.CPUPercent = cpu
        stat.MemoryMB = mem

        stats = append(stats, stat)
    }
    return stats
}

// getProcessResourceUsage uses ps command (cross-platform)
func getProcessResourceUsage(pid int) (cpu float64, memMB float64) {
    cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "%cpu,%mem,rss")
    output, err := cmd.Output()
    if err != nil {
        return 0, 0 // Process may have exited
    }
    // Parse output...
    return cpu, memMB
}
```

### Phase 2: Telemetry Streaming (v0.5.1)

Enhance spawned agents to report telemetry via WebSocket:

```go
// When spawning agent, capture and parse its stdout
cmd.Stdout = &telemetryParser{
    instanceID: instanceID,
    wsServer:   s.wsServer,
}

type telemetryParser struct {
    instanceID string
    wsServer   *websocket.Server
    turns      int
    tokens     TokenUsage
    cost       float64
}

func (t *telemetryParser) Write(p []byte) (n int, err error) {
    // Parse NDJSON line
    var event map[string]interface{}
    if json.Unmarshal(p, &event) == nil {
        // Extract telemetry from stream_event or result types
        if eventType, ok := event["type"].(string); ok {
            switch eventType {
            case "stream_event":
                // Extract turn count from message_start
                t.turns++
            case "result":
                // Extract final cost/tokens
                if cost, ok := event["total_cost_usd"].(float64); ok {
                    t.cost = cost
                }
            }

            // Broadcast telemetry update
            t.wsServer.Broadcast(websocket.Message{
                Type: "telemetry_update",
                Data: map[string]interface{}{
                    "instance_id": t.instanceID,
                    "turns":       t.turns,
                    "cost":        t.cost,
                },
            })
        }
    }
    return len(p), nil
}
```

### Phase 3: UI Monitor Component

React component for Monitor tab:

```tsx
// ui/src/components/Monitor/Monitor.tsx

interface ProcessStats {
  instance_id: string;
  pid: number;
  started_at: string;
  duration_sec: number;
  cpu_percent: number;
  memory_mb: number;
  status: string;
  // Telemetry (from WebSocket updates)
  turns?: number;
  tokens_in?: number;
  tokens_out?: number;
  cost?: number;
}

export const Monitor: React.FC = () => {
  const [processes, setProcesses] = useState<ProcessStats[]>([]);

  // Poll /api/monitor every 2s
  useEffect(() => {
    const poll = async () => {
      const res = await fetch('/api/monitor');
      const data = await res.json();
      setProcesses(data.processes);
    };
    poll();
    const interval = setInterval(poll, 2000);
    return () => clearInterval(interval);
  }, []);

  // Listen for WebSocket telemetry updates
  useEffect(() => {
    // Merge telemetry into process stats...
  }, []);

  return (
    <div className="monitor-grid">
      {processes.map(proc => (
        <ProcessCard key={proc.instance_id} process={proc} />
      ))}
    </div>
  );
};
```

## Configuration

```yaml
# monitor settings
monitor:
  poll_interval_ms: 2000    # How often to poll process stats
  history_duration_min: 60  # How long to keep completed process history
  cpu_warning_threshold: 80 # Highlight processes above this CPU%
  max_duration_warning_sec: 300  # Warn if process runs longer than this
```

## Alerts & Actions

The Monitor tab will support:
- **Visual alerts**: Red highlight for processes exceeding thresholds
- **Kill button**: Stop runaway processes from the UI
- **History view**: See recently completed processes with their final stats

## Migration

No breaking changes. New API endpoint and UI tab added.

## Testing

1. Start server with `ailang serve`
2. Spawn an agent from the UI
3. Verify Monitor tab shows process with updating stats
4. Kill process from Monitor tab
5. Verify process disappears and history shows completion

## References

- Incident: November 29, 2025 - 2 stuck eval processes
- Related: M-EVAL-TIMEOUT (process-level timeout design)
- Files: `internal/server/monitor.go`, `ui/src/components/Monitor/`
