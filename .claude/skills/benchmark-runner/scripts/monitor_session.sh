#!/bin/bash
# Monitor active benchmark sessions
# Usage: monitor_session.sh [--watch|-w] [--full|-f]

BENCH_DIR="/Users/mark/dev/sunholo/ai-coding-lang-bench"

if [ ! -f "$BENCH_DIR/monitor.sh" ]; then
  echo "Error: monitor.sh not found in $BENCH_DIR"
  exit 1
fi

cd "$BENCH_DIR"
exec ./monitor.sh "$@"
