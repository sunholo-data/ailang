package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// TelemetryReporter sends telemetry updates to the collaboration hub server
// This allows eval suite and other external processes to report their Claude usage
type TelemetryReporter struct {
	serverURL  string
	instanceID string
	pid        int
	enabled    bool

	mu        sync.Mutex
	turns     int
	tokensIn  int
	tokensOut int
	cost      float64
	status    string
}

// telemetryRequest matches the server's TelemetryReportRequest
type telemetryRequest struct {
	PID        int     `json:"pid"`
	InstanceID string  `json:"instance_id"`
	Turns      int     `json:"turns"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	Cost       float64 `json:"cost"`
	Status     string  `json:"status"`
}

// NewTelemetryReporter creates a new telemetry reporter
// serverURL should be like "http://localhost:8090" (the collaboration hub)
// If serverURL is empty, checks AILANG_HUB_URL environment variable
// If neither is set, telemetry reporting is disabled
func NewTelemetryReporter(serverURL, instanceID string) *TelemetryReporter {
	if serverURL == "" {
		serverURL = config.HubURL()
	}

	enabled := serverURL != ""

	return &TelemetryReporter{
		serverURL:  serverURL,
		instanceID: instanceID,
		pid:        os.Getpid(),
		enabled:    enabled,
		status:     "running",
	}
}

// SetStatus sets the process status (running, completed, error)
func (t *TelemetryReporter) SetStatus(status string) {
	t.mu.Lock()
	t.status = status
	t.mu.Unlock()
	t.sendUpdate()
}

// sendUpdate sends the current telemetry to the server
func (t *TelemetryReporter) sendUpdate() {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	req := telemetryRequest{
		PID:        t.pid,
		InstanceID: t.instanceID,
		Turns:      t.turns,
		TokensIn:   t.tokensIn,
		TokensOut:  t.tokensOut,
		Cost:       t.cost,
		Status:     t.status,
	}
	t.mu.Unlock()

	// Send in background to not block the main execution
	go func() {
		body, err := json.Marshal(req)
		if err != nil {
			return // Silent fail - telemetry is best-effort
		}

		httpReq, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/telemetry", t.serverURL), bytes.NewReader(body))
		if err != nil {
			return // Silent fail
		}
		httpReq.Header.Set("Content-Type", "application/json")
		config.AttachHubToken(httpReq.Header) // hub-class ingest route
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			return // Silent fail
		}
		resp.Body.Close()
	}()
}
