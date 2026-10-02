#!/bin/bash
set -uo pipefail
src="$1"; p="$2"
S=$(mktemp -d /tmp/fleet15/su.XXXX)
awk '/^if \[ "\$RC" -ne 0 \]; then$/,/^exit "\$RC"$/' "$src" | sed '$d' > "$S/blk.sh"
[ -s "$S/blk.sh" ] || { echo EMPTY_EXTRACT; exit 1; }
ailang(){ echo SEND >>"$S/sends"; }; gh(){ echo GH >>"$S/sends"; }; log(){ echo "$*" >>"$S/log"; }
RC=1; STATE_DIR=$S; MISSION_NAME=t; MISSION_LOG_FILE=/nonexistent; pre_last_record=""; LOG=x; MSG_FROM=t; MISSION_GH_ISSUE=""
[ "$p" = unset ] || MC_PAUSED=$p
. "$S/blk.sh" 2>"$S/err"
echo "paused=$p sends=$(cat "$S/sends" 2>/dev/null | wc -l | tr -d ' ') stderr_bytes=$(wc -c < "$S/err" | tr -d ' ') log=[$(tr '\n' ';' < "$S/log" 2>/dev/null)]"
