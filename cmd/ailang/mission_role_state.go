package main

import (
	"context"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"time"
)

func runDurableMissionRole(ctx context.Context, req dispatch.Request, receipt, dbPath string, models *modelreg.ModelsConfig, factory dispatch.Factory) (*dispatch.Report, error) {
	return runMissionRoleWithHeartbeat(ctx, req, receipt, dbPath, models, factory, 10*time.Second)
}
func runMissionRoleWithHeartbeat(ctx context.Context, req dispatch.Request, receipt, dbPath string, models *modelreg.ModelsConfig, factory dispatch.Factory, heartbeatInterval time.Duration) (*dispatch.Report, error) {
	return iteration.RunRoleWithHeartbeat(ctx, req, receipt, dbPath, models, factory, heartbeatInterval)
}
