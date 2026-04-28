#!/bin/bash
# check_public_feedback.sh - SessionStart hook → Public Feedback inbox check
#
# Queries the prod Firestore (ailang-multivac project, inbox_messages
# collection, to_inbox=public-feedback) and emits a session-start summary
# when there are unread submissions from external agents via the public
# MCP (https://mcp.ailang.sunholo.com/mcp/).
#
# Why a dedicated script instead of `ailang messages list`?
# The CLI's openStore() ignores AILANG_STORAGE=gcp and always opens local
# SQLite — it can't see the cloud-side public-feedback inbox at all.
# Until that's fixed (M-MESSAGES-CLI-GCP follow-up), this script reads
# Firestore directly via Application Default Credentials.
#
# Hidden when:
#   - python3 is missing
#   - google-cloud-firestore not installed
#   - ADC not configured (silently skipped — first-time devs see nothing)
#   - 0 unread feedback in the inbox
#
# Cap: prints up to 5 most recent unread items.

set -euo pipefail

# Drain stdin so the hook chain doesn't block on it.
cat > /dev/null || true

# Bail silently if no python3 or no google-cloud-firestore — the hook chain
# should never block on optional tooling.
if ! command -v python3 >/dev/null 2>&1; then
    exit 0
fi

# Try the query; suppress stderr so missing creds / no internet don't spam
# the session start. Output is plain text, one summary line per message.
RESULT=$(python3 - 2>/dev/null <<'PYEOF' || true
import sys
try:
    from google.cloud import firestore
except ImportError:
    sys.exit(0)

try:
    db = firestore.Client(project='ailang-multivac')
    docs = list(
        db.collection('inbox_messages')
        .where('to_inbox', '==', 'public-feedback')
        .where('status', '==', 'unread')
        .stream()
    )
except Exception:
    # No ADC, no network, IAM denied, etc. — silent skip.
    sys.exit(0)

if not docs:
    sys.exit(0)

# Sort newest-first and take top 5.
docs.sort(key=lambda d: d.to_dict().get('created_at') or '', reverse=True)
print(f"COUNT={len(docs)}")
for d in docs[:5]:
    data = d.to_dict()
    msg_id = data.get('message_id', '?')
    title = (data.get('title') or '(untitled)').replace('\n', ' ')[:80]
    category = data.get('category', '?')
    from_agent = data.get('from_agent', '?')
    created = data.get('created_at')
    age = ''
    if created:
        from datetime import datetime, timezone
        try:
            now = datetime.now(timezone.utc)
            delta = now - created
            hours = delta.total_seconds() / 3600
            if hours < 1:
                age = f"{int(delta.total_seconds() / 60)}m ago"
            elif hours < 24:
                age = f"{int(hours)}h ago"
            else:
                age = f"{int(hours / 24)}d ago"
        except Exception:
            age = ''
    print(f"  - [{category}] {msg_id} • {age}")
    print(f"    {title}")
PYEOF
)

if [ -z "$RESULT" ]; then
    exit 0
fi

COUNT=$(echo "$RESULT" | head -1 | sed 's/COUNT=//')
ITEMS=$(echo "$RESULT" | tail -n +2)

cat <<EOF
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✉️  PUBLIC FEEDBACK INBOX: $COUNT unread submission(s) via mcp.ailang.sunholo.com
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

$ITEMS

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 Read full items in PROD Firestore:
   python3 -c "from google.cloud import firestore; db=firestore.Client(project='ailang-multivac'); [print(d.to_dict().get('payload','')[:400]+'\n---') for d in db.collection('inbox_messages').where('to_inbox','==','public-feedback').where('status','==','unread').stream()]"
   Mark read: db.collection('inbox_messages').document(<id>).update({'status': 'read'})
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF
