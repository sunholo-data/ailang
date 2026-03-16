---
sidebar_position: 22
title: Cloud Messaging Integration
description: How to send messages to AILANG agents and receive results — REST API, Firestore, and Pub/Sub integration options
---

# Cloud Messaging Integration Guide

How to send messages to AILANG agents and receive results from an external client (mobile app, web app, CI/CD pipeline, or custom CLI).

## Architecture Overview

The coordinator runs on Cloud Run and exposes a REST API for message ingestion. Clients send a single HTTP POST — the coordinator handles storage (Firestore) and notification (Pub/Sub) internally.

```
Your Client
  │
  POST /api/messages { inbox, title, content, from, [anthropic_api_key] }
  │
  ▼
Cloud Coordinator (Cloud Run)
  │
  ├── 1. Stores message in Firestore (durable)
  ├── 1b. If API key provided: encrypt with KMS, cache in memory (10min TTL)
  ├── 2. Publishes Pub/Sub notification (trigger)
  ├── 3. Routes to agent based on inbox
  ├── 4. Dispatches Cloud Run Job (OAuth or API Key mode)
  │       env: AILANG_GIT_MODE, AILANG_PUSH_BRANCH, AILANG_TASK_ID
  │
  ▼
Agent executes task (git guardrails enforced by PreToolUse hook)
  │
  ├── 5a. Streams progress events to Pub/Sub (text, tool_use, tool_result, error)
  ├── 5b. Pushes changes to git (branch or direct to main for skip_approval agents)
  ├── 6. Publishes TaskCompletion to Pub/Sub
  │
  ▼
Dashboard (Cloud Run) subscribes to ailang-events-dashboard
  │
  ├── PubSubEventSubscriber pulls events → BroadcastTaskEvent() to WebSocket clients
  │
  ▼
Coordinator receives completion (ailang-completions topic)
  │
  ├── 7. Updates task status (completed/failed/pending_approval)
  ├── 8. Posts completion message to agent inbox (with correlation_id)
  │
  ▼
Your Client
  ← GET /api/messages + filter by correlation_id (polling, recommended for completion)
  ← OR Dashboard WebSocket /ws (real-time streaming of all events)
  ← OR Firestore onSnapshot (real-time completion)
  ← OR Pub/Sub pull subscription (backend services)
```

**Key principle**: One HTTP call to send a message. The coordinator handles Firestore storage and Pub/Sub notification atomically.

## Prerequisites

- The AILANG coordinator deployed on Cloud Run (or running locally)
- An API key (`COORDINATOR_API_KEY`) if auth is enabled
- **REST API clients** (recommended): No additional dependencies — just an HTTP client
- **Firestore clients**: Firestore SDK + `roles/datastore.user` IAM role
- **Pub/Sub clients**: Pub/Sub SDK + a Terraform-managed subscription (see [Provisioning Client Subscriptions](#provisioning-client-subscriptions-terraform))

## Sending a Message

### Option 1: REST API (Recommended)

Send a single HTTP POST to the coordinator. No GCP SDKs required — any HTTP client works.

**Endpoint**: `POST /api/messages`

**Headers**:
- `Content-Type: application/json`
- `Authorization: Bearer <COORDINATOR_API_KEY>` (if auth is configured)

**Request body**:

```json
{
  "inbox": "design-doc-creator",
  "title": "Feature: Semantic Caching",
  "content": "Design and implement a semantic caching layer with TTL...",
  "from": "my-client",
  "category": "feature",
  "message_type": "request"
}
```

**Required fields**:

| Field | Type | Description |
|-------|------|-------------|
| `inbox` | string | Target agent inbox (see [Available Inboxes](#available-inboxes)) |
| `title` | string | Brief summary (shown in listings) |
| `content` | string | Full message content / task description |
| `from` | string | Your client identity (e.g., `"my-app"`, `"ci-pipeline"`) |

**Optional fields**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `category` | string | `"general"` | `"bug"`, `"feature"`, `"general"`, `"research"` |
| `message_type` | string | `"request"` | `"request"`, `"notification"`, `"response"` |
| `github_issue` | int | | Linked GitHub issue number |
| `github_repo` | string | | GitHub repo (e.g., `"owner/repo"`) |
| `anthropic_api_key` | string | | Your Anthropic API key for pay-per-token mode (see [Bring Your Own Key](#bring-your-own-key-byok)) |

**Response (201 Created)**:

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "inbox": "design-doc-creator",
  "status": "unread"
}
```

**Deriving task_id**: The coordinator creates a task with a deterministic ID: `task_id = "task-" + message_id[:8]`. For example, message_id `29404032-74b3-...` → task_id `task-29404032`. Use this to filter WebSocket events for your task (see [Live Build Progress](#live-build-progress-via-websocket-dashboard)).

**Error responses**:
- `400` — Missing required field or invalid JSON
- `401` — Invalid or missing API key (when auth configured)
- `503` — Message store not available

#### Example: curl

```bash
COORDINATOR_URL="https://your-coordinator.run.app"  # Or http://localhost:8080

curl -X POST "${COORDINATOR_URL}/api/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${COORDINATOR_API_KEY}" \
  -d '{
    "inbox": "design-doc-creator",
    "title": "Feature: Semantic Caching",
    "content": "Design and implement semantic caching with TTL support...",
    "from": "my-script",
    "category": "feature"
  }'
```

#### Example: Python

```python
import requests

COORDINATOR_URL = "https://your-coordinator.run.app"
API_KEY = "your-api-key"

resp = requests.post(
    f"{COORDINATOR_URL}/api/messages",
    headers={"Authorization": f"Bearer {API_KEY}"},
    json={
        "inbox": "design-doc-creator",
        "title": "Feature: Semantic Caching",
        "content": "Design and implement semantic caching with TTL...",
        "from": "my-python-app",
        "category": "feature",
    },
)
resp.raise_for_status()
print(f"Created: {resp.json()['message_id']}")
```

#### Example: Node.js

```javascript
const resp = await fetch(`${COORDINATOR_URL}/api/messages`, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: `Bearer ${API_KEY}`,
  },
  body: JSON.stringify({
    inbox: "design-doc-creator",
    title: "Feature: Semantic Caching",
    content: "Design and implement semantic caching with TTL...",
    from: "my-node-app",
    category: "feature",
  }),
});
const { message_id } = await resp.json();
console.log(`Created: ${message_id}`);
```

#### Example: Go

```go
body, _ := json.Marshal(map[string]string{
    "inbox":   "design-doc-creator",
    "title":   "Feature: Semantic Caching",
    "content": "Design and implement semantic caching with TTL...",
    "from":    "my-go-app",
    "category": "feature",
})

req, _ := http.NewRequest("POST", coordinatorURL+"/api/messages", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+apiKey)

resp, err := http.DefaultClient.Do(req)
```

### Bring Your Own Key (BYOK)

By default, cloud agents authenticate using OAuth credentials managed by the coordinator (internal workloads). External users can instead provide their own Anthropic API key for **pay-per-token** execution.

To use BYOK mode, include `anthropic_api_key` in your message:

```bash
curl -X POST "${COORDINATOR_URL}/api/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${COORDINATOR_API_KEY}" \
  -d '{
    "inbox": "sprint-executor",
    "title": "Implement caching layer",
    "content": "Add Redis-backed caching to the API endpoints...",
    "from": "external-client",
    "anthropic_api_key": "sk-ant-api03-YOUR-KEY-HERE"
  }'
```

```python
resp = requests.post(
    f"{COORDINATOR_URL}/api/messages",
    headers={"Authorization": f"Bearer {API_KEY}"},
    json={
        "inbox": "sprint-executor",
        "title": "Implement caching layer",
        "content": "Add Redis-backed caching to the API endpoints...",
        "from": "external-client",
        "anthropic_api_key": "sk-ant-api03-YOUR-KEY-HERE",
    },
)
```

**How it works:**

1. The coordinator encrypts your API key with [Cloud KMS](https://cloud.google.com/security/products/security-key-management) before dispatch
2. The encrypted key is passed as a Cloud Run Job environment override
3. The agent executor decrypts the key at runtime using KMS
4. Claude Code reads `ANTHROPIC_API_KEY` natively (pay-per-token billing to your account)

**Security properties:**

- Your API key is **never persisted** to Firestore, Secret Manager, or any database
- It is held in an **in-memory cache** (10-minute TTL) on the coordinator only
- Cloud Audit Logs see only **KMS ciphertext**, never plaintext
- The coordinator can only encrypt (not decrypt); the agent can only decrypt (not encrypt)
- The KMS key auto-rotates every 90 days

**Auth mode comparison:**

| Property | OAuth (default) | API Key (BYOK) |
|----------|----------------|----------------|
| Billing | Coordinator's Anthropic account | Your Anthropic account |
| Credentials | OAuth token from Secret Manager | User-provided `sk-ant-...` key |
| Cloud Run Job | `agent-executor` | `agent-executor-apikey` |
| Audit log exposure | N/A (token in Secret Manager) | KMS-encrypted ciphertext only |
| Use case | Internal/team workloads | External users, per-project billing |

:::caution
If the coordinator processes your message before your API key's 10-minute cache TTL expires, execution proceeds normally. If the coordinator is under heavy load and the cache entry expires, the task will **fail explicitly** rather than silently falling back to OAuth. Retry the message if this occurs.
:::

### Option 2: Direct Firestore + Pub/Sub (Advanced)

For advanced use cases where you need direct control over storage and notification, you can bypass the REST API and write to Firestore + Pub/Sub directly. This requires GCP client SDKs (Firestore + Pub/Sub).

**Topics** (all follow the pattern `{prefix}-{base}`, default prefix: `ailang`):

| Full Topic Name | Purpose |
|-----------------|---------|
| `ailang-messages` | Publish message notifications here |
| `ailang-events` | Subscribe for real-time execution progress |
| `ailang-tasks` | Internal: coordinator dispatches jobs |
| `ailang-completions` | Internal: jobs report completion |

#### Step 1: Store in Firestore

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

#### Step 2: Publish Notification to Pub/Sub

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

#### Example: Python Client

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

#### Example: Node.js Client

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

#### Example: Go Client

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

#### Example: curl (REST API)

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

After sending a message, agents process it and store results as response messages. Choose the approach that best fits your client type.

> **Most common case:** If you just need to know when an agent finishes, see [Completion Notifications](#completion-notifications) — poll the agent's inbox and match by `correlation_id`.

| Approach | Best For | Dependencies | Latency |
|----------|----------|-------------|---------|
| **REST API Polling** | Scripts, CLI tools, simple integrations | HTTP client only | Seconds (poll interval) |
| **Firestore onSnapshot** | Web apps, mobile apps needing real-time | Firestore SDK | Sub-second |
| **Pub/Sub Pull** | Backend services, always-on consumers | Pub/Sub SDK + Terraform | Sub-second |

### Option 1: REST API Polling (Recommended)

The simplest approach — poll `GET /api/messages` with filters. No GCP SDKs required.

**Endpoint:** `GET /api/messages`

**Query Parameters:**

| Parameter | Example | Description |
|-----------|---------|-------------|
| `inbox` | `?inbox=my-client` | Filter by target inbox |
| `status` | `?status=unread` | Filter by status (`unread`, `read`, `archived`) |
| `from` | `?from=coordinator` | Filter by sender agent |
| `limit` | `?limit=20` | Max results (default: 50) |
| `collapsed` | `?collapsed=true` | Hide deduplicated messages |

**Response:**
```json
{
  "messages": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "from_agent": "design-doc-creator",
      "to_inbox": "my-client",
      "title": "Design Doc: Semantic Caching",
      "payload": "Full design document content...",
      "status": "unread",
      "category": "feature",
      "message_type": "response",
      "created_at": "2026-03-10T15:30:00Z"
    }
  ],
  "count": 1,
  "limit": 50
}
```

#### curl

```bash
# Check for unread messages in your inbox
curl -s "${COORDINATOR_URL}/api/messages?inbox=my-client&status=unread" \
  -H "Authorization: Bearer ${API_KEY}" | jq .

# All messages from a specific agent
curl -s "${COORDINATOR_URL}/api/messages?from=design-doc-creator" \
  -H "Authorization: Bearer ${API_KEY}" | jq .
```

#### Python

```python
import requests
import time

COORDINATOR_URL = "https://your-coordinator.run.app"
API_KEY = "your-api-key"
HEADERS = {"Authorization": f"Bearer {API_KEY}"}

def poll_messages(inbox, interval=10):
    """Poll for new messages with backoff."""
    while True:
        resp = requests.get(
            f"{COORDINATOR_URL}/api/messages",
            params={"inbox": inbox, "status": "unread"},
            headers=HEADERS,
        )
        resp.raise_for_status()
        data = resp.json()

        for msg in data["messages"]:
            print(f"[{msg['from_agent']}] {msg['title']}")
            print(f"  {msg['payload'][:200]}...")
            # Process the message...

        time.sleep(interval)

poll_messages("my-client")
```

#### Node.js

```javascript
const COORDINATOR_URL = "https://your-coordinator.run.app";
const API_KEY = "your-api-key";

async function pollMessages(inbox, intervalMs = 10000) {
  while (true) {
    const resp = await fetch(
      `${COORDINATOR_URL}/api/messages?inbox=${inbox}&status=unread`,
      { headers: { Authorization: `Bearer ${API_KEY}` } }
    );
    const data = await resp.json();

    for (const msg of data.messages) {
      console.log(`[${msg.from_agent}] ${msg.title}`);
      // Process the message...
    }

    await new Promise((r) => setTimeout(r, intervalMs));
  }
}

pollMessages("my-client");
```

#### Go

```go
func pollMessages(coordinatorURL, apiKey, inbox string) error {
    client := &http.Client{Timeout: 10 * time.Second}
    for {
        req, _ := http.NewRequest("GET",
            fmt.Sprintf("%s/api/messages?inbox=%s&status=unread", coordinatorURL, inbox), nil)
        req.Header.Set("Authorization", "Bearer "+apiKey)

        resp, err := client.Do(req)
        if err != nil {
            log.Printf("poll error: %v", err)
            time.Sleep(10 * time.Second)
            continue
        }

        var result struct {
            Messages []map[string]interface{} `json:"messages"`
            Count    int                      `json:"count"`
        }
        json.NewDecoder(resp.Body).Decode(&result)
        resp.Body.Close()

        for _, msg := range result.Messages {
            fmt.Printf("[%s] %s\n", msg["from_agent"], msg["title"])
        }

        time.Sleep(10 * time.Second)
    }
}
```

### Option 2: Firestore onSnapshot (Real-Time, Web/Mobile)

For web or mobile apps that need instant updates, use Firestore's real-time listener. Messages arrive within milliseconds of being stored.

```javascript
import { initializeApp } from "firebase/app";
import {
  getFirestore,
  collection,
  query,
  where,
  onSnapshot,
} from "firebase/firestore";

const app = initializeApp({ projectId: "your-gcp-project" });
const db = getFirestore(app);

// Listen for new unread messages in your inbox
const q = query(
  collection(db, "inbox_messages"),
  where("to_inbox", "==", "my-client"),
  where("status", "==", "unread")
);

const unsubscribe = onSnapshot(q, (snapshot) => {
  snapshot.docChanges().forEach((change) => {
    if (change.type === "added") {
      const msg = change.doc.data();
      console.log(`New message: [${msg.from_agent}] ${msg.title}`);
      console.log(`Content: ${msg.payload}`);
    }
  });
});
```

**Python (Firestore watch):**

```python
from google.cloud import firestore

db = firestore.Client(project="your-gcp-project")

def on_snapshot(doc_snapshot, changes, read_time):
    for change in changes:
        if change.type.name == "ADDED":
            msg = change.document.to_dict()
            print(f"New: [{msg['from_agent']}] {msg['title']}")

query = db.collection("inbox_messages") \
    .where("to_inbox", "==", "my-client") \
    .where("status", "==", "unread")

query.on_snapshot(on_snapshot)
```

**Requirements:** Firestore SDK (`firebase` for web, `google-cloud-firestore` for Python/Go) and `roles/datastore.user` IAM role.

### Option 3: Pub/Sub Pull Subscription (Backend Services)

For always-on backend services, subscribe to the `ailang-messages` Pub/Sub topic. Messages queue while your service is offline (up to 7 days). Requires a Terraform-managed subscription (see [Provisioning Client Subscriptions](#provisioning-client-subscriptions-terraform)).

```python
from google.cloud import pubsub_v1
import json

PROJECT_ID = "your-gcp-project"
SUBSCRIPTION = "ailang-messages-my-client"  # Your Terraform-provisioned subscription

subscriber = pubsub_v1.SubscriberClient()
subscription_path = subscriber.subscription_path(PROJECT_ID, SUBSCRIPTION)

def callback(message):
    data = json.loads(message.data)
    attrs = message.attributes

    print(f"Message from: {attrs.get('from_agent')}")
    print(f"Inbox: {attrs.get('inbox')}")
    print(f"Message ID: {data.get('message_id')}")

    # Fetch full content from Firestore or REST API
    # The Pub/Sub notification contains only the message_id —
    # fetch the full payload via GET /api/messages or Firestore.

    message.ack()  # Acknowledge (removes from queue)

streaming_pull = subscriber.subscribe(subscription_path, callback=callback)
print("Listening for messages...")
streaming_pull.result()  # Blocks forever
```

**Requirements:** Pub/Sub SDK, Terraform subscription, `roles/pubsub.subscriber` IAM role.

### Real-Time Event Streaming (All Options)

Regardless of which receiving approach you use, you can also subscribe to the `ailang-events` topic for live execution progress (tool calls, model output, etc.):

```python
EVENTS_SUBSCRIPTION = "ailang-events-my-client"  # Your Terraform-provisioned subscription

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

| `event_type` | Description | Rate Limit / Truncation |
|--------------|-------------|------------------------|
| `text` | Model reasoning / text output | 1 per 500ms, truncated to 2000 chars |
| `tool_use` | Agent invoked a tool (file edit, bash, etc.) | Tool input truncated to 1000 chars |
| `tool_result` | Tool returned a result | Output truncated to 2000 chars |
| `turn_start` | Agent turn begins | Always broadcast |
| `turn_end` | Agent turn completes | Always broadcast |
| `status` | Task status change (running, completed, failed) | Always broadcast, includes tokens/cost |
| `error` | Error during execution | Always broadcast |

:::info
The cloud executor rate-limits `text` events to 1 per 500ms to avoid flooding Pub/Sub. All events are logged to Cloud Logging (stderr) regardless of rate limiting. The `status` event includes final metrics: `tokens_in`, `tokens_out`, `cost`, and `duration_sec`.
:::

### Live Build Progress via WebSocket (Dashboard)

The AILANG Dashboard provides a WebSocket endpoint for real-time task streaming without GCP SDKs. Connect to the dashboard's WebSocket and receive `TaskStreamEvent` messages as tasks execute.

**Dashboard URL**: `wss://your-dashboard.run.app/ws` (or `ws://localhost:1957/ws` locally)

**How it works**: The cloud executor publishes **all mid-execution events** (text, tool_use, tool_result, error, turn_start, turn_end, status) to the `ailang-events` Pub/Sub topic via `PubSubBroadcaster`. The dashboard's `PubSubEventSubscriber` pulls from the `ailang-events-dashboard` subscription and calls `BroadcastTaskEvent()` to relay events to all connected WebSocket clients.

**Full streaming chain**:

```
Cloud Run Job (agent executor)
  │
  cloudEventHandler receives executor callbacks:
  │  OnText()       → rate-limited 1 per 500ms
  │  OnToolUse()    → tool name + summary
  │  OnToolResult() → tool output (truncated to 2000 chars)
  │  OnError()      → error message
  │  OnMetrics()    → final status with tokens/cost
  │
  ▼
PubSubBroadcaster.Broadcast()
  │  Serializes TaskStreamEvent → publishes to ailang-events topic
  │  Attributes: event_type, task_id, workspace
  │
  ▼
Dashboard (PubSubEventSubscriber)
  │  Pulls from ailang-events-dashboard subscription
  │  Deserializes → wsServer.BroadcastTaskEvent()
  │
  ▼
WebSocket /ws endpoint
  │  Broadcasts to all connected clients
  │
  ▼
Your Client (browser, portal, CLI)
```

**Authentication**: When `COORDINATOR_API_KEY` is set (cloud deployments), external WebSocket clients must connect with `?token=API_KEY` query parameter. The same API key used for `POST /api/messages` works for WebSocket. The embedded dashboard UI (same-origin) connects without a token. In local mode (no key configured), all connections are open.

#### Connecting message_id to task_id

When you send a message via `POST /api/messages`, the response returns a `message_id` (UUID). The coordinator creates a task with a **deterministic** task_id derived from that UUID:

```
task_id = "task-" + message_id.substring(0, 8)
```

For example, if `POST /api/messages` returns:

```json
{ "message_id": "29404032-74b3-40c6-acc3-23d6bbe14b68", "inbox": "sprint-executor", "status": "unread" }
```

The task_id will be `task-29404032`. Your client can derive this immediately — no polling or extra API call needed.

#### End-to-End Flow

```
Portal/Client                  Coordinator              Dashboard
  │                               │                        │
  POST /api/messages ───────────► │                        │
  │                               │                        │
  ◄── { message_id: "2940..." }   │                        │
  │                               │                        │
  │  task_id = "task-" +          │ processes message      │
  │    message_id[:8]             │ creates task-29404032  │
  │  = "task-29404032"            │ dispatches to agent    │
  │                               │                        │
  ws://dashboard/ws ──────────────────────────────────────► │
  │  filter by task_id            │                        │
  │                               │  agent executes...     │
  │◄── task_stream: text ─────────────────────────────────  │
  │◄── task_stream: tool_use ─────────────────────────────  │
  │◄── task_stream: status=completed ─────────────────────  │
  │                               │                        │
  │  Task complete — agent has pushed to GitHub             │
  │  Load preview / fetch results                          │
```

**Timing note**: There is a brief delay between the `POST /api/messages` response and the first task_stream event (the coordinator needs to process the message and dispatch the task). Show a "Queued" state in your UI until the first `status: "running"` event arrives.

#### JavaScript Example (Full Flow)

```javascript
const COORDINATOR_URL = "https://your-coordinator.run.app";
const DASHBOARD_URL = "wss://your-dashboard.run.app/ws";
const API_KEY = "your-api-key";

// 1. Send message and derive task_id
async function submitBuild(inbox, title, content) {
  const response = await fetch(`${COORDINATOR_URL}/api/messages`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${API_KEY}`,
    },
    body: JSON.stringify({ inbox, title, content, from: "portal" }),
  });

  const { message_id } = await response.json();
  const task_id = `task-${message_id.substring(0, 8)}`;

  return { message_id, task_id };
}

// 2. Connect to WebSocket and filter by task_id
function watchTask(task_id, callbacks) {
  const ws = new WebSocket(`${DASHBOARD_URL}?token=${API_KEY}`);

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type !== "task_stream") return;
    if (msg.data.task_id !== task_id) return;

    const { stream_type, text, tool_name, status, agent_id } = msg.data;

    switch (stream_type) {
      case "text":
        callbacks.onText?.(text);
        break;
      case "tool_use":
        callbacks.onToolUse?.(tool_name, msg.data.tool_input);
        break;
      case "tool_result":
        callbacks.onToolResult?.(tool_name, msg.data.tool_output);
        break;
      case "status":
        callbacks.onStatus?.(status);
        if (status === "completed" || status === "failed") {
          ws.close();
        }
        break;
      case "error":
        callbacks.onError?.(msg.data.error_msg);
        break;
    }
  };

  return ws;
}

// 3. Usage
const { task_id } = await submitBuild(
  "sprint-executor",
  "Build landing page",
  "Create a responsive landing page with hero section..."
);

watchTask(task_id, {
  onText: (text) => appendToLog(text),
  onToolUse: (tool) => showStep(`Running: ${tool}`),
  onStatus: (status) => updateBuildStatus(status),
  onError: (err) => showError(err),
});
```

#### Python Example (Full Flow)

```python
import asyncio
import json
import httpx
import websockets

COORDINATOR_URL = "https://your-coordinator.run.app"
DASHBOARD_URL = "wss://your-dashboard.run.app/ws"
API_KEY = "your-api-key"

async def submit_and_watch(inbox: str, title: str, content: str):
    # 1. Send message
    async with httpx.AsyncClient() as client:
        resp = await client.post(
            f"{COORDINATOR_URL}/api/messages",
            json={"inbox": inbox, "title": title, "content": content, "from": "portal"},
            headers={"Authorization": f"Bearer {API_KEY}"},
        )
        message_id = resp.json()["message_id"]

    # 2. Derive task_id
    task_id = f"task-{message_id[:8]}"
    print(f"Submitted: message_id={message_id}, task_id={task_id}")

    # 3. Watch WebSocket for events (token required for external clients)
    async with websockets.connect(f"{DASHBOARD_URL}?token={API_KEY}") as ws:
        async for raw in ws:
            msg = json.loads(raw)
            if msg.get("type") != "task_stream":
                continue
            data = msg["data"]
            if data.get("task_id") != task_id:
                continue

            stream_type = data.get("stream_type")
            if stream_type == "text":
                print(data.get("text", ""), end="")
            elif stream_type == "tool_use":
                print(f"\n[Tool] {data.get('tool_name')}")
            elif stream_type == "status":
                print(f"\n[Status] {data.get('status')}")
                if data.get("status") in ("completed", "failed"):
                    break

asyncio.run(submit_and_watch(
    "sprint-executor",
    "Build landing page",
    "Create a responsive landing page with hero section..."
))
```

#### WebSocket Message Format

```json
{
  "type": "task_stream",
  "data": {
    "task_id": "task-29404032",
    "stream_type": "text",
    "turn_num": 3,
    "text": "Let me fix the parser...",
    "agent_id": "sprint-executor",
    "workspace": "/workspace/project",
    "status": "running"
  }
}
```

**TaskStreamEvent fields:**

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Task identifier (`"task-" + message_id[:8]`) |
| `stream_type` | string | Event type (see table above) |
| `turn_num` | int | Agent turn number |
| `text` | string | Model text output (for `text` events) |
| `tool_name` | string | Tool name (for `tool_use` / `tool_result`) |
| `tool_input` | string | Tool input JSON (truncated to 1000 chars) |
| `tool_output` | string | Tool output (truncated to 2000 chars) |
| `status` | string | `running`, `completed`, `failed` (for `status` events) |
| `error_msg` | string | Error message (for `error` events) |
| `tokens_in` | int | Input tokens used |
| `tokens_out` | int | Output tokens used |
| `cost` | float | Execution cost in USD |
| `agent_id` | string | Agent identifier (e.g., `"sprint-executor"`) |
| `workspace` | string | Working directory path |

#### Comparison: WebSocket vs Pub/Sub for Events

| Aspect | Dashboard WebSocket | Pub/Sub Pull |
|--------|-------------------|--------------|
| **SDK Required** | WebSocket client (built into browsers) | Google Cloud Pub/Sub SDK |
| **Best For** | Web dashboards, browser apps, portals | Backend services, CI/CD pipelines |
| **Message Delivery** | Broadcast to all connected clients | Load-balanced across subscribers |
| **Persistence** | None (live stream only) | 1-hour retention in subscription |
| **Auth** | API key via `?token=API_KEY` query parameter | GCP IAM (`roles/pubsub.subscriber`) |
| **message_id → task_id** | Client derives: `"task-" + id[:8]` | Same derivation, or filter by attributes |

## Available Inboxes

Messages are routed to agents by `inbox` name. The coordinator matches inbox to agent configuration.

| Inbox | Agent | Purpose |
|-------|-------|---------|
| `coordinator` | General coordinator | Ad-hoc tasks (bug fixes, features, research) |
| `design-doc-creator` | Design Doc Creator | Creates design documents from requirements |
| `sprint-planner` | Sprint Planner | Creates sprint plans from design docs |
| `sprint-executor` | Sprint Executor | Implements approved sprint plans |
| `website-builder` | Website Builder | Builds websites from briefs, pushes directly to GitHub |
| `eval-runner` | Eval Runner (script) | Runs benchmark evaluations |
| `user` | Human developer | Messages for human review |

Custom agents can be added in `~/.ailang/config.yaml` — each agent watches its own inbox.

### Skip-Approval Agents (Direct Push)

Some agents are configured with `skip_approval: true` to bypass the human approval step. These agents push their changes directly to a target branch (e.g., `main`) instead of creating a `coordinator/{taskID}` branch.

**How it works:**

1. Agent config has `skip_approval: true` and `merge_branch: main`
2. Coordinator sets `AILANG_PUSH_BRANCH=main` on the Cloud Run Job
3. The job skips `git checkout -b coordinator/{taskID}` and works directly on the cloned branch
4. Changes are pushed directly to the target branch (e.g., `main`)
5. CompletionHandler marks the task as `completed` (not `pending_approval`)

**Current skip-approval agents:**

| Agent | Target Branch | Repo | Use Case |
|-------|--------------|------|----------|
| `website-builder` | `main` | `sunholo-data/sunholo-websites` | GitHub Pages sites — user-specific subdirectories, no review needed |

**Agent config example** (`config.cloud.yaml`):

```yaml
- id: website-builder
  label: "Website Builder"
  inbox: website-builder
  workspace: sunholo-data/sunholo-websites
  model: sonnet
  skip_approval: true    # Push directly, no human approval
  merge_branch: main     # Target branch for direct push
  auto_merge: false
```

### Git Guardrails

Cloud agent executors enforce git guardrails via a PreToolUse hook (`ailang_bootstrap/scripts/hooks/git_guard.sh`). This prevents agents from creating rogue branches, pushing to wrong remotes, or wasting turns on merge conflicts.

**Why hooks, not settings.json**: The agent runs with `--dangerously-skip-permissions` which bypasses all `settings.json` deny rules. PreToolUse hooks are the only enforcement mechanism that still fires in this mode.

:::warning
The hook script must NOT use `jq` or `grep -P` — neither is installed in the agent container (Debian slim). Use `grep -E` and `sed` only.
:::

**`AILANG_GIT_MODE` environment variable** (set on Cloud Run Jobs):

| Mode | Reads | Commits | Push | Branch creation | Use case |
|------|-------|---------|------|----------------|----------|
| `guardrails` (default) | Yes | Yes | Only to expected branch | Blocked | Most agents |
| `strict` | Yes | No | No | Blocked | Read-only agents |
| `permissive` | Yes | Yes | Yes | Yes | Legacy/escape hatch |
| (not set) | N/A | N/A | N/A | N/A | Local dev (no-op) |

**Decision matrix:**

| Git Command | `guardrails` | `strict` | `permissive` |
|---|---|---|---|
| `git status/diff/log` | Allow | Allow | Allow |
| `git add/commit/stash` | Allow | Deny | Allow |
| `git push origin $PUSH_BRANCH` | Allow | Deny | Allow |
| `git push origin other-branch` | **Deny** | Deny | Allow |
| `git push --force` | **Deny** | Deny | **Deny** |
| `git checkout -b new-branch` | **Deny** | Deny | Allow |
| `git remote add/set-url` | **Deny** | Deny | Allow |
| `git reset --hard` | **Deny** | Deny | **Deny** |

When a command is blocked, the hook returns a structured deny message that the AI reads and self-corrects from:

```
BLOCKED: Creating new branches is not allowed. Work on the current branch ('main').
The executor manages branch creation.
```

**Per-agent configuration** (`config.cloud.yaml`):

```yaml
- id: website-builder
  git_mode: strict        # Read-only git access
- id: sprint-executor
  git_mode: guardrails    # Can commit and push to expected branch
```

The executor's post-AI phase handles all git push operations deterministically — the agent should focus on creating/editing files and committing locally.

**Agent executor environment variables** (set on Cloud Run Job):

| Variable | Default | Description |
|----------|---------|-------------|
| `AILANG_GIT_MODE` | `guardrails` | Git guardrails mode — per-agent override via `git_mode` in config |
| `AILANG_PUSH_BRANCH` | `coordinator/{taskID}` | Target branch for push validation |
| `AILANG_TASK_ID` | (set by coordinator) | Task ID for branch naming |
| `AILANG_REPO_URL` | (set by coordinator) | Git repo URL for the workspace |

## Agent Chain Workflow

Agents can be chained: when one completes, it triggers the next. The standard chain is:

```
design-doc-creator → [Human Approval] → sprint-planner → [Human Approval] → sprint-executor → [Human Approval] → Merged
```

To kick off the full chain, send to `design-doc-creator`:

```bash
curl -X POST "${COORDINATOR_URL}/api/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d '{
    "inbox": "design-doc-creator",
    "title": "Feature: Semantic Caching",
    "content": "Design and implement a semantic caching layer with TTL...",
    "from": "my-client",
    "category": "feature"
  }'
```

The coordinator handles the rest — each agent completes, requests approval, and triggers the next.

## Completion Notifications

When an agent finishes a task (success or failure), the CompletionHandler posts a **completion message** to the agent's inbox. This enables external clients to detect task completion by polling `GET /api/messages`.

### How It Works

1. Cloud Run Job finishes → publishes `TaskCompletion` to `ailang-completions` topic
2. CompletionHandler receives the completion (via push or pull subscription)
3. Handler updates task status (`completed` or `failed`)
4. Handler posts an inbox message with `message_type: "completion"` and a `correlation_id` linking back to the original request

### Completion Message Format

```json
{
  "id": "auto-generated-uuid",
  "from_agent": "website-builder",
  "to_inbox": "website-builder",
  "message_type": "completion",
  "title": "Task task-29404032: completed",
  "payload": "{\"task_id\":\"task-29404032\",\"agent_id\":\"website-builder\",\"status\":\"completed\",\"branch_name\":\"main\",\"error_msg\":\"\"}",
  "correlation_id": "29404032-74b3-40c6-acc3-23d6bbe14b68"
}
```

**Key fields:**

| Field | Description |
|-------|-------------|
| `message_type` | Always `"completion"` for these notifications |
| `correlation_id` | The original `message_id` from the `POST /api/messages` request |
| `payload` | JSON with `task_id`, `agent_id`, `status`, `branch_name`, `error_msg` |

### Polling for Completion

To check if your task has completed, poll `GET /api/messages` and match by `correlation_id`:

```javascript
// After submitting a build via POST /api/messages
const { message_id } = await submitResponse.json();

// Poll for completion
async function waitForCompletion(messageId, inbox, intervalMs = 5000) {
  while (true) {
    const resp = await fetch(
      `${COORDINATOR_URL}/api/messages?inbox=${inbox}&from=${inbox}`,
      { headers: { Authorization: `Bearer ${API_KEY}` } }
    );
    const data = await resp.json();

    // Find completion message matching our original request
    const completion = data.messages.find(
      (msg) =>
        msg.message_type === "completion" &&
        msg.correlation_id === messageId
    );

    if (completion) {
      const payload = JSON.parse(completion.payload);
      return payload; // { task_id, agent_id, status, branch_name, error_msg }
    }

    await new Promise((r) => setTimeout(r, intervalMs));
  }
}

const result = await waitForCompletion(message_id, "website-builder");
if (result.status === "completed") {
  console.log(`Site pushed to branch: ${result.branch_name}`);
} else {
  console.error(`Build failed: ${result.error_msg}`);
}
```

**Note:** The `GET /api/messages` endpoint does not currently support filtering by `correlation_id` or `message_type` as query parameters. Clients must filter results client-side. The result set is small enough that this is efficient for typical workloads.

## Website Builder Integration

The website-builder agent builds websites from text briefs and pushes them directly to GitHub Pages.

### Architecture

```
Portal (GitHub Pages SPA)
  │
  ├── POST /api/build { title, content }
  ├── WSS /api/ws (live streaming)
  │
  ▼
Express Sidecar (Cloud Run)
  │
  ├── POST /api/messages { inbox: "website-builder", title, content, from: "sidecar" }
  ├── WSS proxy → DASHBOARD_URL/ws?token=COORDINATOR_API_KEY
  │
  ▼
Coordinator (Cloud Run)
  │
  ├── Creates task-{id}
  ├── Dispatches Cloud Run Job with AILANG_PUSH_BRANCH=main
  │
  ▼
Cloud Run Job (website-builder agent)
  │
  ├── Clones sunholo-data/sunholo-websites (branch: main)
  ├── Runs Claude CLI to build HTML/CSS
  ├── Streams progress → PubSubBroadcaster → ailang-events topic
  ├── git add + commit + push to main
  ├── Publishes TaskCompletion to ailang-completions
  │
  ▼                                          ▼
Coordinator                           Dashboard
  │                                     │
  ├── Marks task completed              ├── PubSubEventSubscriber pulls events
  ├── Posts completion message          ├── Broadcasts to WebSocket /ws
  │                                     │
  ▼                                     ▼
Sidecar polls GET /api/messages    Sidecar proxies WSS to portal
  │                                     │
  ├── Finds completion                  ├── Portal shows live activity:
  ├── Returns result to portal          │   text, tool_use, status, error
  │                                     │
  ▼                                     ▼
Portal loads preview               Portal shows "Build complete"
from GitHub Pages                  (completedViaWs fast-path)
```

### Sidecar Endpoints

The Express.js sidecar mediates between the portal and the coordinator:

| Sidecar Endpoint | Maps To | Purpose |
|-----------------|---------|---------|
| `POST /api/build` | `POST /api/messages` on coordinator | Submit build brief |
| `GET /api/status` | `GET /api/messages` on coordinator | Poll for completion |
| `WSS /api/ws` | `WSS DASHBOARD_URL/ws?token=API_KEY` | Proxy live streaming events |

**Field mapping** (sidecar → coordinator):

| Sidecar sends | Coordinator expects | Match? |
|---------------|-------------------|--------|
| `inbox` | `inbox` | Exact |
| `title` | `title` | Exact |
| `content` | `content` | Exact |
| `from` | `from` | Exact |

### Sidecar WebSocket Proxy (Live Streaming)

The sidecar proxies WebSocket connections from the portal to the dashboard, enabling real-time build progress without exposing the dashboard URL or API key to the browser.

```
Portal (BuildStep.vue)
  │
  wss://sidecar-url/api/ws
  │
  ▼
Sidecar (Express.js on Cloud Run)
  │
  Upgrades connection, proxies to:
  wss://DASHBOARD_URL/ws?token=COORDINATOR_API_KEY
  │
  ▼
Dashboard (AILANG Coordinator)
  │
  PubSubEventSubscriber → BroadcastTaskEvent() → WebSocket broadcast
  │
  ▼
Portal receives task_stream messages:
  • text: AI reasoning output (truncated to 200 chars in UI)
  • tool_use: tool name with friendly labels ("Writing file...")
  • status: completed/failed → triggers final pollStatus()
  • error: error messages
```

**Portal event handling** (BuildStep.vue):

The portal's `connectTaskStream(taskId)` converts the sidecar HTTP URL to WebSocket (`https:` → `wss:`, appends `/ws`) and handles four event types:

| `stream_type` | Portal behavior |
|---------------|----------------|
| `text` | Appends to activity log (truncated to 200 chars) |
| `tool_use` | Shows friendly tool label (e.g., "Writing file...") |
| `status` | Sets `completedViaWs = true`, triggers `pollStatus()` for final payload |
| `error` | Logs error message |

The portal keeps the last 30 activity log entries and uses `completedViaWs` as a fast-path to skip polling — when a `status: "completed"` event arrives via WebSocket, it immediately fetches the completion payload rather than waiting for the next poll cycle.

**Dual completion path**: The portal supports both WebSocket completion (fast, sub-second) and polling completion (fallback, 5-second intervals). The WebSocket path fires first when streaming is active; polling acts as a safety net if the WebSocket connection drops.

### Environment Variables

The sidecar needs these environment variables:

| Variable | Example | Description |
|----------|---------|-------------|
| `COORDINATOR_URL` | `https://coordinator.run.app` | Coordinator Cloud Run service URL |
| `COORDINATOR_API_KEY` | `sk-...` | API key for coordinator auth |
| `DASHBOARD_URL` | `https://dashboard.run.app` | Dashboard URL for WebSocket proxy upstream |

### End-to-End Example (JavaScript)

```javascript
// Portal sends brief to sidecar
const buildResp = await fetch("/api/build", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    inbox: "website-builder",
    title: "Build Acme Corp landing page",
    content: "Create a modern landing page for Acme Corp with hero section, features grid, and CTA...",
    from: "portal",
  }),
});
const { message_id } = await buildResp.json();

// Poll sidecar for completion (sidecar forwards to coordinator)
const pollInterval = setInterval(async () => {
  const statusResp = await fetch(`/api/status?inbox=website-builder`);
  const data = await statusResp.json();

  const completion = data.messages?.find(
    (msg) => msg.correlation_id === message_id && msg.message_type === "completion"
  );

  if (completion) {
    clearInterval(pollInterval);
    const result = JSON.parse(completion.payload);

    if (result.status === "completed") {
      // Load preview from GitHub Pages
      window.location.href = `https://sunholo-data.github.io/sunholo-websites/${sitePath}/`;
    } else {
      showError(`Build failed: ${result.error_msg}`);
    }
  }
}, 5000);
```

### Deployment Checklist

To enable the website-builder flow:

- [ ] Deploy coordinator with completion notification support (v0.9.1+)
- [ ] Deploy agent executor Cloud Run Job with `AILANG_PUSH_BRANCH` support
- [ ] Upload config: `make config-upload` (adds `skip_approval: true` to website-builder)
- [ ] Verify sidecar has `COORDINATOR_URL` and `COORDINATOR_API_KEY` env vars
- [ ] Agent executor has `GITHUB_TOKEN` from Secret Manager for git push

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

### Coordinator API Key

The REST API (`/api/messages`) is protected by a single `COORDINATOR_API_KEY` set on the Cloud Run service. All clients share this key. Inbox-level filtering provides functional isolation — each client only queries their own inbox.

For stronger per-user isolation, deploy separate coordinator instances with distinct API keys. The architecture supports this — Firestore and Pub/Sub are shared with workspace-based routing, so each coordinator instance serves its own set of workspaces.

### Application Default Credentials (For Pub/Sub / Firestore)

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

| Role | When Needed | Purpose |
|------|-------------|---------|
| `COORDINATOR_API_KEY` | REST API (Option 1) | Send and receive messages via HTTP |
| `roles/datastore.user` | Firestore (Option 2) | Real-time message listener |
| `roles/pubsub.subscriber` | Pub/Sub (Option 3) | Pull from subscriptions |
| `roles/pubsub.publisher` | Direct Pub/Sub send | Publish to `ailang-messages` topic (advanced) |

**Minimum for REST API clients:** Only the `COORDINATOR_API_KEY` bearer token. No GCP IAM roles required.

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

1. **Use the REST API unless you need real-time.** The coordinator handles Firestore storage and Pub/Sub notification atomically — one fewer failure mode for your client.

2. **Handle duplicate deliveries.** Both Pub/Sub (at-least-once) and polling (if you don't track what you've processed) can deliver duplicates. Use `message_id` to deduplicate.

3. **For direct Firestore + Pub/Sub clients:** Always store in Firestore FIRST, then publish to Pub/Sub. If Pub/Sub publish fails, the message is still safe.

4. **For Pub/Sub consumers:** ACK messages after processing. If your handler crashes before ACK, Pub/Sub redelivers automatically. Don't hold messages beyond the ack deadline (30s pull / 60s push).

## Common Pitfalls

These are real issues encountered while building the [Website Builder integration](#website-builder-integration). Check this list before debugging unexpected behavior.

### 1. Polling the wrong inbox for completions

Completion notifications go to the **agent's own inbox**, not a client-specific inbox. If your agent is `website-builder`, poll `inbox=website-builder`.

```javascript
// ✅ Correct — poll the agent's inbox
const resp = await fetch(`${COORDINATOR_URL}/api/messages?inbox=website-builder&status=unread`);

// ❌ Wrong — "portal" is not where completions land
const resp = await fetch(`${COORDINATOR_URL}/api/messages?inbox=portal&status=unread`);
```

The coordinator's `CompletionHandler` posts to `ToInbox: task.AgentID` — always the agent that ran the task.

### 2. Inventing client-side correlation IDs

Use the `message_id` returned by `POST /api/messages` for correlation. Don't generate your own IDs — they won't match the `correlation_id` on completion messages.

```javascript
// ✅ Correct — use the coordinator's message_id
const { message_id } = await resp.json();
// ... later, match: msg.correlation_id === message_id

// ❌ Wrong — frontend-generated ID won't appear in completion
const myId = crypto.randomUUID();
```

### 3. Forgetting to parse `payload`

The `payload` field on completion messages is a **JSON string**, not an object. Parse it before accessing fields.

```javascript
// ✅ Correct
const payload = JSON.parse(completion.payload);
console.log(payload.status); // "completed"

// ❌ Wrong — payload is a string, not an object
console.log(completion.payload.status); // undefined
```

### 4. Checking the wrong status value

The coordinator sends `status: "completed"` (past tense). Some integrations also accept `"complete"` as a fallback — check both if unsure:

```javascript
if (payload.status === "completed" || payload.status === "complete") {
  // Task finished successfully
}
```

### 5. Expecting the sidecar to commit (for skip_approval agents)

Agents with `skip_approval: true` push changes **directly to GitHub** from the Cloud Run Job. The sidecar does not perform a save/commit step. Don't wait for a sidecar-side git operation that will never happen.

### 6. Not allowing time for GitHub Pages deployment

After the agent commits, GitHub Pages needs time to rebuild and deploy (typically 30–120 seconds). Poll the live URL with retries rather than loading immediately after completion.

## Quick Reference

### Send a message (REST API — recommended)

```bash
curl -X POST "${COORDINATOR_URL}/api/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d '{"inbox":"INBOX","title":"TITLE","content":"CONTENT","from":"CLIENT_ID"}'
```

### Receive messages (REST API — recommended)

```bash
# Unread messages in your inbox
curl -s "${COORDINATOR_URL}/api/messages?inbox=MY_INBOX&status=unread" \
  -H "Authorization: Bearer ${API_KEY}"

# All messages from a specific agent
curl -s "${COORDINATOR_URL}/api/messages?from=design-doc-creator&limit=10" \
  -H "Authorization: Bearer ${API_KEY}"
```

### Send + watch build progress (end-to-end)

```bash
# 1. Send message, get message_id
MESSAGE_ID=$(curl -s -X POST "${COORDINATOR_URL}/api/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${API_KEY}" \
  -d '{"inbox":"sprint-executor","title":"Build page","content":"...","from":"portal"}' \
  | jq -r '.message_id')

# 2. Derive task_id (deterministic: "task-" + first 8 chars of message_id)
TASK_ID="task-${MESSAGE_ID:0:8}"
echo "Watching task: ${TASK_ID}"

# 3. Connect to dashboard WebSocket and filter by task_id
websocat "wss://YOUR_DASHBOARD_URL/ws" | jq --arg tid "$TASK_ID" \
  'select(.type == "task_stream" and .data.task_id == $tid) | .data'
```

```javascript
// JavaScript: send message, derive task_id, watch WebSocket
const resp = await fetch(`${COORDINATOR_URL}/api/messages`, {
  method: "POST",
  headers: { "Content-Type": "application/json", "Authorization": `Bearer ${API_KEY}` },
  body: JSON.stringify({ inbox: "sprint-executor", title: "Build", content: "...", from: "portal" }),
});
const { message_id } = await resp.json();
const task_id = `task-${message_id.substring(0, 8)}`;

const ws = new WebSocket("wss://YOUR_DASHBOARD_URL/ws");
ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  if (msg.type === "task_stream" && msg.data.task_id === task_id) {
    console.log(msg.data.stream_type, msg.data);
  }
};
```

### Topic naming (for direct Pub/Sub integration)

```
{prefix}-{base}
ailang-messages      # Publish notifications here (or use REST API instead)
ailang-events        # Subscribe here for real-time progress
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
