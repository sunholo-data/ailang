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
  POST /api/messages { inbox, title, content, from }
  │
  ▼
Cloud Coordinator (Cloud Run)
  │
  ├── 1. Stores message in Firestore (durable)
  ├── 2. Publishes Pub/Sub notification (trigger)
  ├── 3. Routes to agent based on inbox
  ├── 4. Dispatches Cloud Run Job
  │
  ▼
Agent executes task
  │
  ├── 5. Publishes completion to Pub/Sub
  │
  ▼
Your Client
  ← pulls from events subscription (real-time progress)
  ← pulls from messages subscription (completion notification)
```

**Key principle**: One HTTP call to send a message. The coordinator handles Firestore storage and Pub/Sub notification atomically.

## Prerequisites

- The AILANG coordinator deployed on Cloud Run (or running locally)
- An API key (`COORDINATOR_API_KEY`) if auth is enabled
- For receiving results: a Pub/Sub pull subscription (see [Provisioning Client Subscriptions](#provisioning-client-subscriptions-terraform))

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

**Response (201 Created)**:

```json
{
  "message_id": "550e8400-e29b-41d4-a716-446655440000",
  "inbox": "design-doc-creator",
  "status": "unread"
}
```

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

After sending a message, agents process it and store results as response messages. Choose the approach that best fits your client type:

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

| `event_type` | Description |
|--------------|-------------|
| `text` | Model reasoning / text output |
| `tool_use` | Agent invoked a tool (file edit, bash, etc.) |
| `tool_result` | Tool returned a result |

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
| `COORDINATOR_API_KEY` | REST API (Options 1) | Send and receive messages via HTTP |
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

1. **Always store in Firestore FIRST, then publish to Pub/Sub.** If Pub/Sub publish fails, the message is still safe.

2. **ACK messages after processing.** If your handler crashes before ACK, Pub/Sub redelivers automatically.

3. **Handle duplicate deliveries.** Pub/Sub guarantees at-least-once delivery, so your handler must be idempotent. Use `message_id` to deduplicate.

4. **Don't hold messages too long.** The ack deadline is 30s (pull) or 60s (push). If processing takes longer, extend the deadline or ACK immediately and process asynchronously.

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
