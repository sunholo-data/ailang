package codex

import (
	"context"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	spanExecTurn  = "exec.turn"
	attrTurnNum   = "turn.number"
	attrSessionID = "session.id"
	fmtTurnHeader = "\n[TURN %d]\n"
)

// startTurn opens a new turn span, fires OnTurnStart, and appends the turn
// header to transcriptBuf. It ends any in-progress turn span first.
func startTurn(
	ctx context.Context,
	n int,
	sessionID string,
	prev *trace.Span,
	buf *strings.Builder,
	handler executor.EventHandler,
) trace.Span {
	if prev != nil && *prev != nil {
		(*prev).End()
	}
	_, s := telemetry.StartSpan(ctx, codexTracer, spanExecTurn,
		trace.WithAttributes(
			attribute.Int(attrTurnNum, n),
			attribute.String(attrSessionID, sessionID),
		),
	)
	handler.OnTurnStart(n)
	buf.WriteString(fmt.Sprintf(fmtTurnHeader, n))
	return s
}
