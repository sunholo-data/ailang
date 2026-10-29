package eval_harness

import (
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/executor"
)

// MaintenanceUsageCollector owns controller-only v2 receipt export for the finite
// runner. Missing parser fields remain unavailable; local edit intent is never
// inferred from tools, language or file-touch counts.
type MaintenanceUsageCollector struct {
	mu        sync.Mutex
	execution *MaintenanceExecution
	receipts  []MaintenanceUsageReceipt
	seen      map[string]bool
}

func NewMaintenanceUsageCollector(e *MaintenanceExecution) *MaintenanceUsageCollector {
	return &MaintenanceUsageCollector{execution: e, receipts: []MaintenanceUsageReceipt{}, seen: map[string]bool{}}
}

func (c *MaintenanceUsageCollector) CollectAttempt(trajectory, stage, attempt string, result *executor.Result, executeErr error) (MaintenanceAttempt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	row := MaintenanceAttempt{ID: attempt, ReceiptIDs: []string{}}
	if c.execution == nil || c.seen[attempt] {
		return row, fmt.Errorf("missing execution or duplicate attempt usage assignment")
	}
	matched := false
	for _, tr := range c.execution.Plan.Trajectories {
		if tr.ID == trajectory {
			for _, st := range tr.Stages {
				if st.ID == stage {
					matched = true
				}
			}
		}
	}
	if !matched {
		return row, fmt.Errorf("usage outside frozen schedule")
	}
	if result == nil || result.ExecutorVersion == "" || result.ExecutorVersion != c.execution.Plan.Harness {
		return row, fmt.Errorf("usage lacks matching observed harness identity")
	}
	rows, err := MaintenanceUsageReceipts(attempt, c.execution.Plan.SettingsDigest, c.execution.Plan.Model, result)
	if err != nil {
		return row, err
	}
	for _, receipt := range rows {
		if receipt.Role == "leaf" {
			row.ReceiptIDs = append(row.ReceiptIDs, receipt.ID)
		}
	}
	c.receipts = append(c.receipts, rows...)
	c.seen[attempt] = true
	return row, nil
}

func (c *MaintenanceUsageCollector) Bundle() MaintenanceUsageBundle {
	c.mu.Lock()
	defer c.mu.Unlock()
	return MaintenanceUsageBundle{Schema: "maintenance.usage/2", Receipts: append([]MaintenanceUsageReceipt{}, c.receipts...)}
}
func (c *MaintenanceUsageCollector) WriteNew(path string) error {
	return maintenanceWriteNewJSON(path, c.Bundle())
}
