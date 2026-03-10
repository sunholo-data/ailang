---
sidebar_position: 22
title: Cloud Messaging Integration
description: How to integrate external clients with the AILANG Cloud Messaging Service via Google Cloud Pub/Sub
---

# Cloud Messaging Integration Guide

How to send messages to AILANG agents and receive results from an external client (mobile app, web app, CI/CD pipeline, or custom CLI).

## Architecture Overview

AILANG Cloud uses **Google Cloud Pub/Sub** as the message transport layer. Pub/Sub is a **notification layer**, not the primary store — message content lives in Firestore, and Pub/Sub carries lightweight notifications that trigger processing.

```
Your Client
  │
  ├── 1. Store message in Firestore (durable)
  ├── 2. Publish notification to Pub/Sub (trigger)
  │
  ▼
Cloud Coordinator (Cloud Run)
  │  ← receives push notification at /pubsub/push
  │
  ├── 3. Reads full message from Firestore
  ├── 4. Routes to agent based on inbox
  ├── 5. Dispatches Cloud Run Job
  │
  ▼
Agent executes task
  │
  ├── 6. Publishes completion to Pub/Sub
  │
  ▼
Your Client
  ← pulls from events subscription (real-time progress)
  ← pulls from messages subscription (completion notification)
```

**Key principle**: Messages are ALWAYS stored durably first (Firestore), then a Pub/Sub notification triggers processing. If Pub/Sub delivery fails, the message is still safe in Firestore.

## Prerequisites

- A Google Cloud project with Pub/Sub enabled
- Service account credentials with Pub/Sub Publisher/Subscriber roles
- The AILANG infrastructure deployed (topics and subscriptions exist via Terraform)

### GCP Project & Topic Prefix

All topic and subscription names follow the pattern `{prefix}-{base}`. The default prefix is `ailang`.

| Full Topic Name | Base Name | Purpose |
|-----------------|-----------|---------|
| `ailang-messages` | `messages` | Send messages to agents |
| `ailang-tasks` | `tasks` | Internal: coordinator dispatches jobs |
| `ailang-completions` | `completions` | Internal: jobs report completion |
| `ailang-events` | `events` | Real-time execution progress |
| `ailang-dead-letter` | `dead-letter` | Failed message sink |

**For clients, you only need two topics:**
1. **`ailang-messages`** — publish to this to send messages
2. **`ailang-events`** — subscribe to this for real-time progress (optional)

## Sending a Message

### Step 1: Store in Firestore

Store the full message in the `messages` collection:

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "from_agent": "my-client",
  "to_inbox": "design-doc-creator",
  "message_type": "request",
  "title": "Create design doc for caching feature",
  "payload": "We need a design document for implementing semantic caching...",
  "category": "feature",
  "status": "unread",
  "created_at": "2026-03-10T15:00:00Z"
}
```

**Required fields:**

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string (UUID) | Unique message identifier |
| `from_agent` | string | Your client identity (e.g., `"my-app"`, `"ci-pipeline"`) |
| `to_inbox` | string | Target agent inbox (see [Available Inboxes](#available-inboxes)) |
| `title` | string | Brief summary (shown in listings) |
| `payload` | string | Full message content / task description |
| `status` | string | Always `"unread"` for new messages |
| `created_at` | string (RFC3339) | Creation timestamp |

**Optional fields:**

| Field | Type | Description |
|-------|------|-------------|
| `message_type` | string | `"request"`, `"notification"`, `"response"` |
| `category` | string | `"bug"`, `"feature"`, `"general"`, `"research"` |
| `github_issue` | int | Linked GitHub issue number |
| `github_repo` | string | GitHub repo (e.g., `"owner/repo"`) |

### Step 2: Publish Notification to Pub/Sub

Publish a **lightweight notification** to the `ailang-messages` topic. The notification just carries the message ID — the coordinator fetches full content from Firestore.

**Pub/Sub message payload** (JSON):

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Pub/Sub message attributes** (key-value metadata for routing):

| Attribute | Value | Required |
|-----------|-------|----------|
| `inbox` | Target agent inbox (e.g., `"design-doc-creator"`) | Yes |
| `from_agent` | Your client identity | Yes |
| `workspace` | Project identifier (e.g., `"sunholo-data/ailang"`) | Recommended |
| `category` | Message category (e.g., `"feature"`) | Optional |
| `message_type` | Message type (e.g., `"request"`) | Optional |

**Ordering key**: Set to the `inbox` value. This ensures messages to the same agent are delivered in order.

### Example: Python Client

```python
from google.cloud import pubsub_v1, firestore
import json
import uuid
from datetime import datetime, timezone

PROJECT_ID = "your-gcp-project"
TOPIC_PREFIX = "ailang"

# Step 1: Store in Firestore
db = firestore.Client(project=PROJECT_ID)
message_id = str(uuid.uuid4())

db.collection("messages").document(message_id).set({
    "message_id": message_id,
    "from_agent": "my-python-app",
    "to_inbox": "design-doc-creator",
    "message_type": "request",
    "title": "Create design doc for caching",
    "payload": "We need semantic caching with TTL support...",
    "category": "feature",
    "status": "unread",
    "created_at": datetime.now(timezone.utc).isoformat(),
})

# Step 2: Publish notification
publisher = pubsub_v1.PublisherClient()
topic_path = publisher.topic_path(PROJECT_ID, f"{TOPIC_PREFIX}-messages")

future = publisher.publish(
    topic_path,
    data=json.dumps({"message_id": message_id}).encode("utf-8"),
    inbox="design-doc-creator",
    from_agent="my-python-app",
    workspace="sunholo-data/ailang",
    category="feature",
    message_type="request",
    ordering_key="design-doc-creator",
)
print(f"Published: {future.result()}")
```

### Example: Node.js Client

```javascript
const { PubSub } = require("@google-cloud/pubsub");
const { Firestore } = require("@google-cloud/firestore");
const { v4: uuidv4 } = require("uuid");

const PROJECT_ID = "your-gcp-project";
const TOPIC_PREFIX = "ailang";

async function sendMessage(inbox, title, content, category = "general") {
  const db = new Firestore({ projectId: PROJECT_ID });
  const pubsub = new PubSub({ projectId: PROJECT_ID });

  const messageId = uuidv4();

  // Step 1: Store in Firestore
  await db.collection("messages").doc(messageId).set({
    message_id: messageId,
    from_agent: "my-node-app",
    to_inbox: inbox,
    message_type: "request",
    title: title,
    payload: content,
    category: category,
    status: "unread",
    created_at: new Date().toISOString(),
  });

  // Step 2: Publish notification
  const topic = pubsub.topic(`${TOPIC_PREFIX}-messages`, {
    enableMessageOrdering: true,
  });

  await topic.publishMessage({
    data: Buffer.from(JSON.stringify({ message_id: messageId })),
    attributes: {
      inbox: inbox,
      from_agent: "my-node-app",
      workspace: "sunholo-data/ailang",
      category: category,
      message_type: "request",
    },
    orderingKey: inbox,
  });

  return messageId;
}
```

### Example: Go Client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "cloud.google.com/go/firestore"
    "cloud.google.com/go/pubsub"
    "github.com/google/uuid"
)

func sendMessage(ctx context.Context, inbox, title, content string) (string, error) {
    projectID := "your-gcp-project"
    prefix := "ailang"

    // Step 1: Store in Firestore
    fsClient, _ := firestore.NewClient(ctx, projectID)
    defer fsClient.Close()

    messageID := uuid.New().String()
    fsClient.Collection("messages").Doc(messageID).Set(ctx, map[string]interface{}{
        "message_id":   messageID,
        "from_agent":   "my-go-app",
        "to_inbox":     inbox,
        "message_type": "request",
        "title":        title,
        "payload":      content,
        "category":     "feature",
        "status":       "unread",
        "created_at":   time.Now().UTC().Format(time.RFC3339),
    })

    // Step 2: Publish notification
    psClient, _ := pubsub.NewClient(ctx, projectID)
    defer psClient.Close()

    topic := psClient.Topic(fmt.Sprintf("%s-messages", prefix))
    topic.EnableMessageOrdering = true

    data, _ := json.Marshal(map[string]string{"message_id": messageID})
    result := topic.Publish(ctx, &pubsub.Message{
        Data: data,
        Attributes: map[string]string{
            "inbox":        inbox,
            "from_agent":   "my-go-app",
            "workspace":    "sunholo-data/ailang",
            "category":     "feature",
            "message_type": "request",
        },
        OrderingKey: inbox,
    })

    serverID, err := result.Get(ctx)
    return serverID, err
}
```

### Example: curl (REST API)

```bash
# Get access token
TOKEN=$(gcloud auth print-access-token)
PROJECT_ID="your-gcp-project"

# Publish to Pub/Sub (notification only — store in Firestore separately)
# Data must be base64-encoded
DATA=$(echo -n '{"message_id":"550e8400-e29b-41d4-a716-446655440000"}' | base64)

curl -X POST \
  "https://pubsub.googleapis.com/v1/projects/${PROJECT_ID}/topics/ailang-messages:publish" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"messages\": [{
      \"data\": \"${DATA}\",
      \"attributes\": {
        \"inbox\": \"coordinator\",
        \"from_agent\": \"curl-client\",
        \"workspace\": \"sunholo-data/ailang\",
        \"category\": \"bug\"
      },
      \"orderingKey\": \"coordinator\"
    }]
  }"
```

## Receiving Results

### Option A: Pull Subscription (Recommended for Clients)

Create your own pull subscription on the `ailang-messages` topic to receive responses. Or use the pre-provisioned `ailang-messages-laptop` subscription if you're the only external client.

```python
from google.cloud import pubsub_v1
import json

PROJECT_ID = "your-gcp-project"
SUBSCRIPTION = "ailang-messages-laptop"  # Or your custom subscription

subscriber = pubsub_v1.SubscriberClient()
subscription_path = subscriber.subscription_path(PROJECT_ID, SUBSCRIPTION)

def callback(message):
    data = json.loads(message.data)
    attrs = message.attributes

    print(f"Message from: {attrs.get('from_agent')}")
    print(f"Inbox: {attrs.get('inbox')}")
    print(f"Message ID: {data.get('message_id')}")

    # Fetch full content from Firestore if needed
    # ...

    message.ack()  # Acknowledge (removes from queue)

streaming_pull = subscriber.subscribe(subscription_path, callback=callback)
print("Listening for messages...")
streaming_pull.result()  # Blocks forever
```

### Option B: Real-Time Event Streaming

Subscribe to the `ailang-events` topic for live execution progress (tool calls, model output, etc.):

```python
EVENTS_SUBSCRIPTION = "ailang-events-laptop"  # Or your custom subscription

def event_callback(message):
    attrs = message.attributes
    event_type = attrs.get("event_type")  # "text", "tool_use", "tool_result"
    task_id = attrs.get("task_id")

    event = json.loads(message.data)
    print(f"[{task_id}] {event_type}: {event}")

    message.ack()

streaming_pull = subscriber.subscribe(
    subscriber.subscription_path(PROJECT_ID, EVENTS_SUBSCRIPTION),
    callback=event_callback,
)
```

**Event types in the stream:**

| `event_type` | Description |
|--------------|-------------|
| `text` | Model reasoning / text output |
| `tool_use` | Agent invoked a tool (file edit, bash, etc.) |
| `tool_result` | Tool returned a result |

### Option C: Poll Firestore Directly

If you don't need real-time notifications, query Firestore for message status changes:

```python
# Check for responses to your messages
docs = db.collection("messages") \
    .where("from_agent", "==", "ailang-coordinator") \
    .where("to_inbox", "==", "my-client-inbox") \
    .where("status", "==", "unread") \
    .stream()

for doc in docs:
    msg = doc.to_dict()
    print(f"{msg['title']}: {msg['payload']}")
```

## Available Inboxes

Messages are routed to agents by `inbox` name. The coordinator matches inbox to agent configuration.

| Inbox | Agent | Purpose |
|-------|-------|---------|
| `coordinator` | General coordinator | Ad-hoc tasks (bug fixes, features, research) |
| `design-doc-creator` | Design Doc Creator | Creates design documents from requirements |
| `sprint-planner` | Sprint Planner | Creates sprint plans from design docs |
| `sprint-executor` | Sprint Executor | Implements approved sprint plans |
| `eval-runner` | Eval Runner (script) | Runs benchmark evaluations |
| `user` | Human developer | Messages for human review |

Custom agents can be added in `~/.ailang/config.yaml` — each agent watches its own inbox.

## Agent Chain Workflow

Agents can be chained: when one completes, it triggers the next. The standard chain is:

```
design-doc-creator → [Human Approval] → sprint-planner → [Human Approval] → sprint-executor → [Human Approval] → Merged
```

To kick off the full chain, send to `design-doc-creator`:

```python
send_message(
    inbox="design-doc-creator",
    title="Feature: Semantic Caching",
    content="Design and implement a semantic caching layer with TTL...",
    category="feature",
)
```

The coordinator handles the rest — each agent completes, requests approval, and triggers the next.

## Push Endpoint Format (For Server-Side Integration)

If you're building a server that receives Pub/Sub push messages (e.g., your own Cloud Run service subscribing to AILANG topics), here's the push envelope format:

### Pub/Sub Push Envelope

Pub/Sub POSTs this JSON to your endpoint:

```json
{
  "message": {
    "data": "eyJtZXNzYWdlX2lkIjoiNTUwZTg0MDAuLi4ifQ==",
    "messageId": "1234567890",
    "attributes": {
      "inbox": "design-doc-creator",
      "from_agent": "user",
      "workspace": "sunholo-data/ailang",
      "category": "feature",
      "message_type": "request"
    },
    "publishTime": "2026-03-10T15:00:00.000Z"
  },
  "subscription": "projects/PROJECT/subscriptions/your-subscription"
}
```

**`data` field**: Base64-encoded JSON. Decode to get:

```json
{"message_id": "550e8400-e29b-41d4-a716-446655440000"}
```

### Response Semantics

| HTTP Status | Pub/Sub Behavior |
|-------------|-----------------|
| **200** | ACK — message removed from queue |
| **500** | NACK — message retried with exponential backoff |

**Important**: Return 200 for malformed messages too (prevents infinite retry loops). Only return 500 for transient errors you want retried.

### Retry Policy

| Setting | Value |
|---------|-------|
| Min backoff | 10 seconds |
| Max backoff | 600 seconds (10 min) |
| Max delivery attempts | 5 |
| Dead letter | Messages moved to `ailang-dead-letter` topic after 5 failures |

## Authentication

### Application Default Credentials (Recommended)

On Google Cloud (Cloud Run, GCE, GKE), ADC works automatically:

```python
# No credentials needed — uses attached service account
publisher = pubsub_v1.PublisherClient()
```

### Service Account Key (Local Development)

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

### Required IAM Roles

| Role | Purpose |
|------|---------|
| `roles/pubsub.publisher` | Publish to `ailang-messages` topic |
| `roles/pubsub.subscriber` | Pull from subscriptions |
| `roles/datastore.user` | Read/write Firestore messages |

## Provisioning Client Subscriptions (Terraform)

Each client that needs to receive messages gets its own pull subscription, managed via Terraform in the `ailang-multivac` infrastructure repo. This ensures subscriptions are reproducible, version-controlled, and consistent across environments.

### Adding a New Client

Add an entry to the `client_subscriptions` variable in your Terraform configuration:

```hcl
# terraform/variables.tf
variable "client_subscriptions" {
  description = "Pull subscriptions for external clients on the messages topic"
  type = list(object({
    name               = string           # Client identifier (e.g., "stapledon", "mobile-app")
    inbox_filter       = optional(string)  # Only receive messages for this inbox (optional)
    ack_deadline       = optional(number, 30)
    retention_days     = optional(number, 7)
    message_ordering   = optional(bool, true)
  }))
  default = []
}
```

```hcl
# terraform/terraform.tfvars (or environment-specific .tfvars)
client_subscriptions = [
  {
    name         = "laptop"
    # No filter — receives all messages
  },
  {
    name         = "stapledon"
    inbox_filter = "stapledon"        # Only messages to stapledon inbox
  },
  {
    name         = "mobile-app"
    inbox_filter = "mobile-app"
    ack_deadline = 60                  # Longer ack for mobile (slower processing)
  },
  {
    name         = "ci-pipeline"
    inbox_filter = "ci-pipeline"
    retention_days = 1                 # CI doesn't need 7 days of history
  },
]
```

### Terraform Resource

```hcl
# terraform/pubsub.tf
resource "google_pubsub_subscription" "messages_client" {
  for_each = { for sub in var.client_subscriptions : sub.name => sub }

  name  = "${var.prefix}-messages-${each.value.name}"
  topic = google_pubsub_topic.messages.id

  enable_message_ordering    = each.value.message_ordering
  ack_deadline_seconds       = each.value.ack_deadline
  message_retention_duration = "${each.value.retention_days * 86400}s"

  # Optional: filter by inbox attribute
  dynamic "filter" {
    for_each = each.value.inbox_filter != null ? [each.value.inbox_filter] : []
    content {
      # Only deliver messages targeted at this client's inbox
      filter = "attributes.inbox = \"${filter.value}\""
    }
  }

  expiration_policy {
    ttl = ""  # Never expire
  }
}
```

This creates subscriptions named `{prefix}-messages-{client-name}`, e.g.:
- `ailang-messages-laptop`
- `ailang-messages-stapledon`
- `ailang-messages-mobile-app`

### What Each Field Does

| Field | Default | Description |
|-------|---------|-------------|
| `name` | (required) | Unique client identifier. Becomes part of the subscription name. |
| `inbox_filter` | `null` (no filter) | Pub/Sub server-side filter. When set, the client only receives messages where `attributes.inbox` matches. Reduces bandwidth and processing. |
| `ack_deadline` | `30` seconds | How long Pub/Sub waits for an ACK before redelivering. Increase for slow clients. |
| `retention_days` | `7` | How long unacked messages stay in the queue. Determines the offline window. |
| `message_ordering` | `true` | Deliver messages with the same `orderingKey` (inbox) in publish order. |

### Filter vs No Filter

**Without filter** (`inbox_filter = null`): Client receives ALL messages on the topic. Useful for monitoring dashboards or the primary developer laptop that needs visibility into everything.

**With filter** (`inbox_filter = "my-inbox"`): Client only receives messages where `attributes.inbox = "my-inbox"`. This is server-side filtering — filtered messages are never delivered, so there's no bandwidth or processing cost.

**Recommendation**: Always set a filter for production clients. Only omit it for admin/monitoring use cases.

### Ad-Hoc Subscriptions (gcloud)

For quick testing without a Terraform change:

```bash
gcloud pubsub subscriptions create ailang-messages-test-client \
  --topic=ailang-messages \
  --ack-deadline=30 \
  --message-retention-duration=7d \
  --enable-message-ordering \
  --filter='attributes.inbox = "test-inbox"'
```

**Note**: Ad-hoc subscriptions are not tracked in Terraform state and should be cleaned up after testing. For permanent clients, always use the Terraform approach above.

## Offline Behavior

Pull subscriptions queue messages while your client is offline:

| Subscription | Retention | Offline Window |
|--------------|-----------|----------------|
| `ailang-messages-*` | 7 days | Up to 7 days offline |
| `ailang-events-laptop` | 1 day | Up to 1 day of events |
| `ailang-events-dashboard` | 1 hour | Ephemeral, real-time only |

When your client reconnects, it automatically receives all queued messages.

## Error Handling Best Practices

1. **Always store in Firestore FIRST, then publish to Pub/Sub.** If Pub/Sub publish fails, the message is still safe.

2. **ACK messages after processing.** If your handler crashes before ACK, Pub/Sub redelivers automatically.

3. **Handle duplicate deliveries.** Pub/Sub guarantees at-least-once delivery, so your handler must be idempotent. Use `message_id` to deduplicate.

4. **Don't hold messages too long.** The ack deadline is 30s (pull) or 60s (push). If processing takes longer, extend the deadline or ACK immediately and process asynchronously.

## Quick Reference

### Publish a message (minimal)

```python
# Pub/Sub data payload
{"message_id": "<uuid>"}

# Pub/Sub attributes (routing metadata)
{"inbox": "<agent-inbox>", "from_agent": "<your-id>"}
```

### Topic naming

```
{prefix}-{base}
ailang-messages      # You publish here
ailang-events        # You subscribe here (optional)
ailang-tasks         # Internal only
ailang-completions   # Internal only
ailang-dead-letter   # Failed messages
```

### Subscription naming

```
{prefix}-messages-{client}
ailang-messages-laptop       # Developer laptop (Terraform-managed)
ailang-messages-stapledon    # External project client (Terraform-managed)
ailang-messages-mobile-app   # Mobile client (Terraform-managed)

{prefix}-events-{client}
ailang-events-laptop         # Laptop event stream (Terraform-managed)
ailang-events-dashboard      # Dashboard event stream (Terraform-managed)
```

All client subscriptions are provisioned via Terraform `client_subscriptions` variable. See [Provisioning Client Subscriptions](#provisioning-client-subscriptions-terraform).
