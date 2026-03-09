# M-PUBSUB: Pub/Sub Cloud Messaging Transport

**Status**: Implemented
**Target**: v0.9.0
**Priority**: P0 (High)
**Estimated**: 3 weeks (Go code only; deployment infra in ailang-multivac)
**Dependencies**: Firestore backends (complete), M-CLOUD-INFRA (reference)
**Author**: Claude + Mark
**Created**: 2026-03-05

---

## Executive Summary

Add Google Cloud Pub/Sub as a **notification/transport layer** on top of Firestore storage. This enables event-driven messaging between the coordinator, Cloud Run Jobs, the dashboard, and the developer's laptop — replacing SQLite polling with push-based delivery. Messages remain durably stored in Firestore; Pub/Sub provides instant notification that new work is available.

**Key outcome**: The AILANG coordinator system works 24/7 on Cloud Run without the laptop being available, while the laptop can still send messages and receive real-time updates when online.

---

## Problem Statement

### Current State

The coordinator polls SQLite every 30 seconds for new messages:

```
Laptop (must be running)
├── Coordinator daemon polls collaboration.db every 30s
├── Executes tasks locally via Claude/Gemini CLI
├── HTTPBroadcaster POSTs events to localhost:1957
└── Everything stops when laptop sleeps
```

**Pain points:**
1. **Agents stop when laptop sleeps** — no 24/7 execution
2. **30s polling latency** — messages sit idle between polls
3. **Single machine** — can't scale execution across Cloud Run Jobs
4. **No remote access** — must be at laptop to send messages or check status
5. **All 20+ agents across 4 projects** (ailang, stapledons_voyage, TwilightGame, sunholo-websites) compete for one machine

### Target State

```
Cloud Run (always-on)
├── Coordinator subscribes to Pub/Sub — instant message delivery
├── Publishes tasks to Pub/Sub → Eventarc triggers Cloud Run Jobs
├── Events stream to dashboard + laptop via Pub/Sub
└── Works 24/7, laptop optional

Laptop (optional, bidirectional)
├── ailang messages send → Firestore + Pub/Sub publish
├── ailang messages watch --pubsub → pull subscription
└── Works behind NAT, queues 7 days while offline
```

---

## Architecture

### Core Principle: Pub/Sub as Notification, Firestore as Storage

```
┌──────────────────────────────────────────────────────────────────────┐
│                    DATA FLOW: DUAL-WRITE PATTERN                      │
│                                                                        │
│  ailang messages send "Fix parser bug" --inbox design-doc-creator     │
│           │                                                            │
│           ├──► Firestore: InsertInboxMessage() [DURABLE STORAGE]      │
│           │    • Searchable, dedup, dashboard, audit trail             │
│           │    • Source of truth for all reads                          │
│           │                                                            │
│           └──► Pub/Sub: Publish(ailang-messages) [NOTIFICATION]       │
│                • Attributes: {inbox, workspace, from_agent, category}  │
│                • Triggers coordinator immediately                      │
│                • 7-day retention, at-least-once delivery               │
│                                                                        │
│  Coordinator receives Pub/Sub notification:                            │
│  1. Reads full message from Firestore (NOT from Pub/Sub payload)      │
│  2. Creates task, publishes to ailang-tasks topic                     │
│  3. Acks the Pub/Sub message                                          │
│                                                                        │
│  Why this pattern:                                                     │
│  • Pub/Sub payload is small (just message_id + routing attributes)    │
│  • Firestore has the full message with envelope, embeddings, etc.     │
│  • Idempotent: re-delivery just re-reads from Firestore               │
│  • Search, dedup, dashboard all work via Firestore (no Pub/Sub deps)  │
└──────────────────────────────────────────────────────────────────────┘
```

### Full Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  LAPTOP (optional)                    CLOUD (always-on)                     │
│  ┌───────────────────┐               ┌─────────────────────────────────┐   │
│  │ ailang CLI         │               │     Cloud Run: Coordinator      │   │
│  │                    │  Pub/Sub      │     (ailang coordinator start)  │   │
│  │ messages send ─────┼──publish──►   │                                 │   │
│  │                    │               │  Subscribes to:                  │   │
│  │ messages watch ◄───┼──pull─────    │  • ailang-messages (new msgs)   │   │
│  │                    │               │  • ailang-completions (results)  │   │
│  │ messages list ─────┼──Firestore──► │                                 │   │
│  │ chains list ───────┼──Firestore──► │  Publishes to:                  │   │
│  └───────────────────┘               │  • ailang-tasks (dispatch jobs)  │   │
│                                       │  • ailang-events (dashboard)     │   │
│                                       └──────────┬──────────────────────┘   │
│                                                   │                          │
│                                                   │ Pub/Sub (ailang-tasks)   │
│                                                   │ via Eventarc             │
│                                                   ▼                          │
│                                       ┌─────────────────────────────────┐   │
│                                       │     Cloud Run Job: Agent         │   │
│                                       │     (ailang coordinator          │   │
│                                       │      execute-job)                │   │
│                                       │                                  │   │
│                                       │  • Reads task from Firestore     │   │
│                                       │  • git clone + branch            │   │
│                                       │  • Runs Claude/Gemini CLI        │   │
│                                       │  • Commits, pushes               │   │
│                                       │  • Publishes to completions      │   │
│                                       └─────────────────────────────────┘   │
│                                                                             │
│  STORAGE (shared)                     DASHBOARD (optional)                  │
│  ┌───────────────────┐               ┌─────────────────────────────────┐   │
│  │ Firestore          │               │     Cloud Run: Dashboard         │   │
│  │ • messages         │               │     (ailang serve)               │   │
│  │ • tasks            │               │                                  │   │
│  │ • approvals        │               │  Subscribes to ailang-events     │   │
│  │ • chains           │               │  → WebSocket to browser          │   │
│  └───────────────────┘               └─────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Topic & Subscription Design

### Single Topic with Attribute Filtering

Instead of per-agent topics (`ailang-inbox-design-doc-creator`, etc.), we use a **single `ailang-messages` topic** with message attributes for routing. This is better because:

1. **Agents are config-driven** — adding an agent in `config.yaml` shouldn't require Terraform changes
2. **Pub/Sub supports attribute filtering** on subscriptions — same routing, less infra
3. **Multi-project** — `workspace` attribute enables cross-project routing from day one
4. **Simpler** — 5 topics total instead of 20+

### Topics

| Topic | Purpose | Payload | Key Attributes |
|-------|---------|---------|----------------|
| `{prefix}-messages` | New message notifications | `{message_id}` | `inbox`, `workspace`, `from_agent`, `category` |
| `{prefix}-tasks` | Task dispatch to Cloud Run Jobs | `{task_id, agent_id}` | `agent_id`, `workspace`, `provider` |
| `{prefix}-completions` | Task results | `{task_id, status}` | `agent_id`, `workspace`, `status` |
| `{prefix}-events` | Real-time streaming | `{event_type, task_id, ...}` | `event_type`, `task_id`, `workspace` |
| `{prefix}-dead-letter` | Failed messages | Any | Original topic attribute preserved |

Default prefix: `ailang`. Configurable via `pubsub.topic_prefix`.

### Message Attributes (Multi-Project)

Every Pub/Sub message carries routing attributes:

```json
{
  "inbox": "design-doc-creator",
  "workspace": "sunholo-data/ailang",
  "from_agent": "user",
  "category": "feature",
  "message_type": "request",
  "message_id": "29404032-74b3-40c6-acc3-23d6bbe14b68"
}
```

The `workspace` attribute maps directly to the existing config.yaml workspace mappings:

| Workspace | Projects |
|-----------|----------|
| `sunholo-data/ailang` | ailang core agents |
| `sunholo-data/stapledons_voyage` | stapledons game agents |
| `MarkEdmondson1234/TwilightGame` | TwilightGame agents |
| `sunholo-data/sunholo-websites` | website builder |

### Subscriptions

| Subscription | Topic | Subscriber | Filter |
|-------------|-------|------------|--------|
| `{prefix}-messages-coordinator` | messages | Cloud Run coordinator | None (all messages) |
| `{prefix}-messages-laptop` | messages | Laptop CLI | None (all) or `workspace = X` |
| `{prefix}-tasks-executor` | tasks | Eventarc → Cloud Run Job | None |
| `{prefix}-completions-coordinator` | completions | Cloud Run coordinator | None |
| `{prefix}-events-dashboard` | events | Cloud Run dashboard | None |
| `{prefix}-events-laptop` | events | Laptop CLI | Optional workspace filter |

### Subscription Configuration

All subscriptions share:
- **Retry policy**: exponential backoff, 10s min, 600s max
- **Dead letter**: after 5 failed delivery attempts
- **Ack deadline**: 60s (messages), 600s (tasks)
- **Message retention**: 7 days
- **Ordering**: by `inbox` key (messages to same inbox delivered in order)

---

## Go Implementation

### New Package: `internal/pubsub/`

#### `client.go` — Pub/Sub Client Wrapper

```go
package pubsub

import (
    "context"
    "fmt"

    "cloud.google.com/go/pubsub"
)

// Client wraps the Google Cloud Pub/Sub client with AILANG conventions.
type Client struct {
    ps        *pubsub.Client
    projectID string
    prefix    string // topic name prefix (default: "ailang")
}

// NewClient creates a new Pub/Sub client.
// Uses Application Default Credentials.
func NewClient(ctx context.Context, projectID, prefix string) (*Client, error) {
    if prefix == "" {
        prefix = "ailang"
    }
    ps, err := pubsub.NewClient(ctx, projectID)
    if err != nil {
        return nil, fmt.Errorf("pubsub.NewClient: %w", err)
    }
    return &Client{ps: ps, projectID: projectID, prefix: prefix}, nil
}

// TopicName returns the full topic name with prefix.
func (c *Client) TopicName(name string) string {
    return fmt.Sprintf("%s-%s", c.prefix, name)
}

// Topic returns a topic reference (does NOT create it — Terraform manages topics).
func (c *Client) Topic(name string) *pubsub.Topic {
    return c.ps.Topic(c.TopicName(name))
}

// Subscription returns a subscription reference.
func (c *Client) Subscription(name string) *pubsub.Subscription {
    return c.ps.Subscription(fmt.Sprintf("%s-%s", c.prefix, name))
}

// Close closes the underlying Pub/Sub client.
func (c *Client) Close() error {
    return c.ps.Close()
}
```

#### `topics.go` — Constants and Payload Types

```go
package pubsub

// Topic base names (prefixed with client.prefix at runtime).
const (
    TopicMessages    = "messages"
    TopicTasks       = "tasks"
    TopicCompletions = "completions"
    TopicEvents      = "events"
    TopicDeadLetter  = "dead-letter"
)

// Subscription suffixes.
const (
    SubMessagesCoordinator    = "messages-coordinator"
    SubMessagesLaptop         = "messages-laptop"
    SubTasksExecutor          = "tasks-executor"
    SubCompletionsCoordinator = "completions-coordinator"
    SubEventsDashboard        = "events-dashboard"
    SubEventsLaptop           = "events-laptop"
)

// MessageNotification is the Pub/Sub payload for the messages topic.
// Intentionally minimal — full message content is in Firestore.
type MessageNotification struct {
    MessageID string `json:"message_id"`
}

// TaskDispatch is the Pub/Sub payload for the tasks topic.
type TaskDispatch struct {
    TaskID  string `json:"task_id"`
    AgentID string `json:"agent_id"`
}

// TaskCompletion is the Pub/Sub payload for the completions topic.
type TaskCompletion struct {
    TaskID     string `json:"task_id"`
    AgentID    string `json:"agent_id"`
    Status     string `json:"status"`      // "completed", "failed"
    BranchName string `json:"branch_name"` // Git branch with changes
    ErrorMsg   string `json:"error_msg,omitempty"`
}

// Attributes returns Pub/Sub message attributes for routing.
type MessageAttributes struct {
    Inbox       string `json:"inbox"`
    Workspace   string `json:"workspace"`
    FromAgent   string `json:"from_agent"`
    Category    string `json:"category"`
    MessageType string `json:"message_type"`
}

// ToMap converts to map[string]string for Pub/Sub attributes.
func (a MessageAttributes) ToMap() map[string]string {
    m := map[string]string{
        "inbox":     a.Inbox,
        "workspace": a.Workspace,
    }
    if a.FromAgent != "" {
        m["from_agent"] = a.FromAgent
    }
    if a.Category != "" {
        m["category"] = a.Category
    }
    if a.MessageType != "" {
        m["message_type"] = a.MessageType
    }
    return m
}
```

#### `publisher.go` — High-Level Publish Functions

```go
package pubsub

import (
    "context"
    "encoding/json"
    "fmt"

    gpubsub "cloud.google.com/go/pubsub"
)

// Publisher provides high-level publish functions for AILANG topics.
type Publisher struct {
    client *Client
    topics map[string]*gpubsub.Topic // cached topic handles
}

// NewPublisher creates a publisher.
func NewPublisher(client *Client) *Publisher {
    return &Publisher{
        client: client,
        topics: make(map[string]*gpubsub.Topic),
    }
}

// topic returns a cached topic handle.
func (p *Publisher) topic(name string) *gpubsub.Topic {
    if t, ok := p.topics[name]; ok {
        return t
    }
    t := p.client.Topic(name)
    t.EnableMessageOrdering = true
    p.topics[name] = t
    return t
}

// PublishMessage publishes a message notification.
// The message is already stored in Firestore — this just notifies subscribers.
func (p *Publisher) PublishMessage(ctx context.Context, messageID string, attrs MessageAttributes) error {
    payload, _ := json.Marshal(MessageNotification{MessageID: messageID})
    result := p.topic(TopicMessages).Publish(ctx, &gpubsub.Message{
        Data:        payload,
        Attributes:  attrs.ToMap(),
        OrderingKey: attrs.Inbox, // Messages to same inbox delivered in order
    })
    _, err := result.Get(ctx)
    if err != nil {
        return fmt.Errorf("publish message notification: %w", err)
    }
    return nil
}

// PublishTask publishes a task dispatch.
func (p *Publisher) PublishTask(ctx context.Context, taskID, agentID, workspace, provider string) error {
    payload, _ := json.Marshal(TaskDispatch{TaskID: taskID, AgentID: agentID})
    result := p.topic(TopicTasks).Publish(ctx, &gpubsub.Message{
        Data: payload,
        Attributes: map[string]string{
            "agent_id":  agentID,
            "workspace": workspace,
            "provider":  provider,
        },
        OrderingKey: agentID,
    })
    _, err := result.Get(ctx)
    if err != nil {
        return fmt.Errorf("publish task dispatch: %w", err)
    }
    return nil
}

// PublishCompletion publishes a task completion.
func (p *Publisher) PublishCompletion(ctx context.Context, completion TaskCompletion, workspace string) error {
    payload, _ := json.Marshal(completion)
    result := p.topic(TopicCompletions).Publish(ctx, &gpubsub.Message{
        Data: payload,
        Attributes: map[string]string{
            "agent_id":  completion.AgentID,
            "workspace": workspace,
            "status":    completion.Status,
        },
    })
    _, err := result.Get(ctx)
    if err != nil {
        return fmt.Errorf("publish completion: %w", err)
    }
    return nil
}

// PublishEvent publishes a real-time dashboard event.
func (p *Publisher) PublishEvent(ctx context.Context, eventJSON []byte, eventType, taskID, workspace string) error {
    result := p.topic(TopicEvents).Publish(ctx, &gpubsub.Message{
        Data: eventJSON,
        Attributes: map[string]string{
            "event_type": eventType,
            "task_id":    taskID,
            "workspace":  workspace,
        },
    })
    _, err := result.Get(ctx)
    if err != nil {
        return fmt.Errorf("publish event: %w", err)
    }
    return nil
}

// Stop flushes all pending messages and releases topic resources.
func (p *Publisher) Stop() {
    for _, t := range p.topics {
        t.Stop()
    }
}
```

#### `subscriber.go` — Pull Subscription Handler

```go
package pubsub

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    gpubsub "cloud.google.com/go/pubsub"
)

// MessageHandler processes a received Pub/Sub message.
// Return nil to ack, non-nil to nack (retry).
type MessageHandler func(ctx context.Context, data []byte, attrs map[string]string) error

// Subscriber manages pull subscriptions.
type Subscriber struct {
    client *Client
    logger *log.Logger
}

// NewSubscriber creates a subscriber.
func NewSubscriber(client *Client, logger *log.Logger) *Subscriber {
    return &Subscriber{client: client, logger: logger}
}

// Subscribe starts receiving messages on a subscription.
// Blocks until ctx is cancelled.
func (s *Subscriber) Subscribe(ctx context.Context, subName string, handler MessageHandler) error {
    sub := s.client.Subscription(subName)

    // Verify subscription exists
    exists, err := sub.Exists(ctx)
    if err != nil {
        return fmt.Errorf("check subscription %s: %w", subName, err)
    }
    if !exists {
        return fmt.Errorf("subscription %s does not exist (create via Terraform)", subName)
    }

    s.logger.Printf("Subscribing to %s", subName)
    return sub.Receive(ctx, func(ctx context.Context, msg *gpubsub.Message) {
        if err := handler(ctx, msg.Data, msg.Attributes); err != nil {
            s.logger.Printf("Message %s failed: %v (will retry)", msg.ID, err)
            msg.Nack()
            return
        }
        msg.Ack()
    })
}

// ReceiveOne receives a single message (useful for CLI watch mode).
func (s *Subscriber) ReceiveOne(ctx context.Context, subName string) ([]byte, map[string]string, error) {
    sub := s.client.Subscription(subName)
    sub.ReceiveSettings.MaxOutstandingMessages = 1

    var data []byte
    var attrs map[string]string

    cctx, cancel := context.WithCancel(ctx)
    err := sub.Receive(cctx, func(_ context.Context, msg *gpubsub.Message) {
        data = msg.Data
        attrs = msg.Attributes
        msg.Ack()
        cancel() // Stop after first message
    })
    if err != nil && ctx.Err() == nil {
        return nil, nil, err
    }
    return data, attrs, nil
}

// DecodeMessageNotification decodes a messages topic payload.
func DecodeMessageNotification(data []byte) (*MessageNotification, error) {
    var n MessageNotification
    if err := json.Unmarshal(data, &n); err != nil {
        return nil, err
    }
    return &n, nil
}

// DecodeTaskCompletion decodes a completions topic payload.
func DecodeTaskCompletion(data []byte) (*TaskCompletion, error) {
    var c TaskCompletion
    if err := json.Unmarshal(data, &c); err != nil {
        return nil, err
    }
    return &c, nil
}
```

### Coordinator Adapters

#### `internal/coordinator/pubsub_adapter.go` — PubSubInboxAdapter

Replaces `InboxMessageAdapter` in cloud mode. Same interface, different transport.

```go
package coordinator

import (
    "context"
    "sync"

    "github.com/sunholo/ailang/internal/messaging"
    ailpubsub "github.com/sunholo/ailang/internal/pubsub"
)

// PubSubInboxAdapter receives message notifications via Pub/Sub
// and reads full messages from Firestore.
//
// Implements coordinator.MessageStore.
type PubSubInboxAdapter struct {
    subscriber *ailpubsub.Subscriber
    msgStore   messaging.MessageStore // Firestore (source of truth for reads)
    inbox      string

    mu      sync.Mutex
    pending []*Message // Buffer for messages received from Pub/Sub
}

// NewPubSubInboxAdapter creates an adapter that receives via Pub/Sub,
// reads from Firestore.
func NewPubSubInboxAdapter(
    subscriber *ailpubsub.Subscriber,
    msgStore messaging.MessageStore,
    inbox string,
) *PubSubInboxAdapter {
    return &PubSubInboxAdapter{
        subscriber: subscriber,
        msgStore:   msgStore,
        inbox:      inbox,
        pending:    make([]*Message, 0),
    }
}

// StartReceiving begins the Pub/Sub subscription in a goroutine.
// Call this once during daemon initialization.
func (a *PubSubInboxAdapter) StartReceiving(ctx context.Context) {
    go func() {
        // Subscribe to messages for this inbox
        _ = a.subscriber.Subscribe(ctx, ailpubsub.SubMessagesCoordinator,
            func(ctx context.Context, data []byte, attrs map[string]string) error {
                notification, err := ailpubsub.DecodeMessageNotification(data)
                if err != nil {
                    return err // Nack — retry
                }

                // Read full message from Firestore
                msgs, err := a.msgStore.ListInboxMessages(messaging.InboxListOptions{
                    Inbox:      a.inbox,
                    UnreadOnly: true,
                    Limit:      1,
                    // Filter by message_id if ListInboxMessages supports it
                })
                if err != nil {
                    return err // Nack — retry
                }

                // Convert and buffer
                a.mu.Lock()
                defer a.mu.Unlock()
                for _, m := range msgs {
                    if m.ID == notification.MessageID || m.MessageID == notification.MessageID {
                        a.pending = append(a.pending, convertInboxMessage(m))
                    }
                }
                return nil // Ack
            })
    }()
}

// ListUnread returns buffered messages (received via Pub/Sub, read from Firestore).
// This is called by the daemon's poll loop, but messages arrive push-based.
func (a *PubSubInboxAdapter) ListUnread() ([]*Message, error) {
    a.mu.Lock()
    defer a.mu.Unlock()

    result := a.pending
    a.pending = make([]*Message, 0)
    return result, nil
}

// MarkAsRead marks a message as read in Firestore.
func (a *PubSubInboxAdapter) MarkAsRead(id string) error {
    return a.msgStore.MarkInboxMessageRead(id)
}

// convertInboxMessage converts messaging.InboxMessage to coordinator.Message.
// Same logic as InboxMessageAdapter.ListUnread().
func convertInboxMessage(m messaging.InboxMessage) *Message {
    githubIssue := 0
    if m.GitHubIssue != nil {
        githubIssue = *m.GitHubIssue
    }
    return &Message{
        ID:           m.ID,
        From:         m.FromAgent,
        Title:        m.Title,
        Content:      m.Payload,
        Type:         m.Category,
        Kind:         m.MessageType,
        GithubIssue:  githubIssue,
        GithubRepo:   m.GitHubRepo,
        ParentTaskID: m.ParentTaskID,
        ChainID:      m.ChainID,
        Iteration:    m.Iteration,
        CreatedAt:    m.CreatedAt,
    }
}

var _ MessageStore = (*PubSubInboxAdapter)(nil)
```

#### `internal/coordinator/pubsub_broadcaster.go` — PubSubBroadcaster

Replaces `HTTPBroadcaster` in cloud mode. Publishes events to Pub/Sub instead of HTTP POST.

```go
package coordinator

import (
    "context"
    "encoding/json"
    "log"

    ailpubsub "github.com/sunholo/ailang/internal/pubsub"
    "github.com/sunholo/ailang/internal/websocket"
)

// PubSubBroadcaster publishes task events to the ailang-events Pub/Sub topic.
// Dashboard and laptop subscribe to receive real-time updates.
type PubSubBroadcaster struct {
    publisher *ailpubsub.Publisher
    logger    *log.Logger
    ctx       context.Context
}

// NewPubSubBroadcaster creates a broadcaster that publishes to Pub/Sub.
func NewPubSubBroadcaster(ctx context.Context, publisher *ailpubsub.Publisher, logger *log.Logger) *PubSubBroadcaster {
    return &PubSubBroadcaster{
        publisher: publisher,
        logger:    logger,
        ctx:       ctx,
    }
}

// Broadcast publishes an event to Pub/Sub. Implements EventBroadcaster.
func (b *PubSubBroadcaster) Broadcast(event *websocket.TaskStreamEvent) {
    eventJSON, err := json.Marshal(event)
    if err != nil {
        b.logger.Printf("PubSubBroadcaster: marshal error: %v", err)
        return
    }

    if err := b.publisher.PublishEvent(
        b.ctx,
        eventJSON,
        string(event.StreamType),
        event.TaskID,
        event.Workspace,
    ); err != nil {
        b.logger.Printf("PubSubBroadcaster: publish error: %v", err)
        // Non-fatal: events are best-effort
    }
}

// BroadcastFunc returns the EventBroadcaster function.
func (b *PubSubBroadcaster) BroadcastFunc() EventBroadcaster {
    return b.Broadcast
}
```

### Daemon Mode Selection

#### Changes to `internal/coordinator/daemon_tasks_init.go`

```go
// In initTaskProcessing(), after loading config:

coordinatorMode := os.Getenv("COORDINATOR_MODE")

switch coordinatorMode {
case "cloud":
    // Pub/Sub mode: receive messages via subscription, dispatch via publish
    pubsubClient, err := ailpubsub.NewClient(ctx, os.Getenv("AILANG_CLOUD_PROJECT"), coordConfig.PubSub.TopicPrefix)
    if err != nil {
        return fmt.Errorf("failed to create Pub/Sub client: %w", err)
    }
    d.pubsubClient = pubsubClient
    d.publisher = ailpubsub.NewPublisher(pubsubClient)
    d.subscriber = ailpubsub.NewSubscriber(pubsubClient, d.logger)

    // Use PubSubInboxAdapter instead of InboxMessageAdapter
    for _, agent := range coordConfig.Agents {
        adapter := NewPubSubInboxAdapter(d.subscriber, d.msgStore, agent.Inbox)
        adapter.StartReceiving(d.ctx)
        d.inboxAdapters[agent.Inbox] = adapter // Same interface
    }

default: // "local" or ""
    // SQLite polling mode (existing behavior, unchanged)
    for _, agent := range coordConfig.Agents {
        d.inboxAdapters[agent.Inbox] = &InboxMessageAdapter{
            store: d.msgStore,
            inbox: agent.Inbox,
        }
    }
}
```

#### Changes to `initHTTPBroadcaster()`

```go
func (d *Daemon) initEventBroadcaster() error {
    coordinatorMode := os.Getenv("COORDINATOR_MODE")

    switch coordinatorMode {
    case "cloud":
        // Pub/Sub broadcaster
        broadcaster := NewPubSubBroadcaster(d.ctx, d.publisher, d.logger)
        d.SetEventBroadcaster(broadcaster.BroadcastFunc())
        d.logger.Printf("Pub/Sub event broadcaster initialized")

    default:
        // HTTP broadcaster (existing behavior)
        serverURL := DefaultServerURL()
        broadcaster := NewHTTPBroadcaster(serverURL, d.logger)
        if !broadcaster.CheckServerAvailable() {
            return fmt.Errorf("server not available at %s", serverURL)
        }
        d.SetEventBroadcaster(broadcaster.BroadcastFunc())
        d.logger.Printf("HTTP broadcaster initialized, streaming to %s", serverURL)
    }
    return nil
}
```

### CLI Integration

#### Changes to `cmd/ailang/messages_send.go`

After writing to Firestore, publish notification to Pub/Sub:

```go
// In runMessagesSend(), after store.InsertInboxMessage():

// Dual-write: also notify via Pub/Sub if configured
if pubsubEnabled() {
    publisher, err := getPubSubPublisher(ctx)
    if err != nil {
        // Non-fatal: message is already in Firestore
        fmt.Fprintf(os.Stderr, "Warning: Pub/Sub notification failed: %v\n", err)
    } else {
        attrs := pubsub.MessageAttributes{
            Inbox:       inbox,
            Workspace:   resolveWorkspace(inbox),
            FromAgent:   fromAgent,
            Category:    category,
            MessageType: messageType,
        }
        if err := publisher.PublishMessage(ctx, msg.ID, attrs); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Pub/Sub notification failed: %v\n", err)
        }
    }
}
```

#### New: `ailang messages watch --pubsub`

```go
// In messages watch command, add --pubsub flag:

if watchPubsub {
    client, _ := pubsub.NewClient(ctx, projectID, prefix)
    sub := pubsub.NewSubscriber(client, logger)

    fmt.Println("Watching for messages via Pub/Sub (Ctrl+C to stop)...")
    sub.Subscribe(ctx, pubsub.SubMessagesLaptop,
        func(ctx context.Context, data []byte, attrs map[string]string) error {
            notification, _ := pubsub.DecodeMessageNotification(data)
            // Read full message from Firestore
            msg, _ := store.GetInboxMessage(notification.MessageID)
            fmt.Printf("[%s] %s: %s\n", attrs["inbox"], msg.Title, truncate(msg.Payload, 80))
            return nil
        })
}
```

### New Subcommand: `ailang coordinator execute-job`

For Cloud Run Jobs — receives task from Pub/Sub, executes, publishes completion:

```go
// cmd/ailang/coordinator_cloud.go

func runExecuteJob(ctx context.Context) error {
    taskID := os.Getenv("AILANG_TASK_ID")
    if taskID == "" {
        return fmt.Errorf("AILANG_TASK_ID not set")
    }

    // 1. Read task from Firestore
    backends, _ := storage.NewBackends(ctx)
    task, _ := backends.Coordinator.GetTask(ctx, taskID)

    // 2. Idempotency check
    if task.Status != "pending" && task.Status != "queued" {
        log.Printf("Task %s already %s, skipping", taskID, task.Status)
        return nil
    }

    // 3. Mark as running (Firestore transaction prevents races)
    backends.Coordinator.MarkTaskRunning(ctx, taskID)

    // 4. Clone repo and create branch
    repoURL := resolveRepoURL(task.Workspace)
    workDir := "/workspace"
    exec.Command("git", "clone", repoURL, workDir).Run()
    branchName := fmt.Sprintf("coordinator/%s", taskID)
    exec.Command("git", "-C", workDir, "checkout", "-b", branchName).Run()

    // 5. Execute via provider (reuses existing executor code)
    executor, _ := executor.GlobalFactory().GetExecutor(task.Provider)
    result, err := executor.Execute(ctx, &executor.Task{
        Directive: task.Directive,
        Workspace: workDir,
    })

    // 6. Commit and push
    exec.Command("git", "-C", workDir, "add", "-A").Run()
    exec.Command("git", "-C", workDir, "commit", "-m", task.Title).Run()
    exec.Command("git", "-C", workDir, "push", "origin", branchName).Run()

    // 7. Publish completion
    publisher, _ := getPubSubPublisher(ctx)
    publisher.PublishCompletion(ctx, pubsub.TaskCompletion{
        TaskID:     taskID,
        AgentID:    task.AgentID,
        Status:     statusFromResult(result, err),
        BranchName: branchName,
    }, task.Workspace)

    return nil
}
```

---

## Configuration Changes

### `~/.ailang/config.yaml` additions

```yaml
# New section: Pub/Sub configuration
pubsub:
  enabled: false                    # Enable Pub/Sub transport layer
  project_id: ""                    # GCP project (defaults to AILANG_CLOUD_PROJECT)
  topic_prefix: "ailang"            # Prefix for all topic names
  laptop_subscription: true         # Create/use laptop pull subscription

coordinator:
  mode: local                       # "local" (SQLite polling) or "cloud" (Pub/Sub)
  # ... existing agent config unchanged ...
```

### Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `COORDINATOR_MODE` | `local` or `cloud` | Cloud Run only |
| `AILANG_CLOUD_PROJECT` | GCP project ID | Cloud/hybrid mode |
| `AILANG_STORAGE` | `gcp` or `hybrid` | Cloud Run only |
| `AILANG_TASK_ID` | Task ID for execute-job | Set by Eventarc |
| `GOOGLE_APPLICATION_CREDENTIALS` | Service account key | If not using Workload Identity |

---

## Laptop Hybrid Mode

### How It Works

The laptop connects to the same Firestore + Pub/Sub infrastructure as cloud:

| Operation | Transport | Works Offline? |
|-----------|-----------|----------------|
| `ailang messages send` | Firestore write + Pub/Sub publish | Firestore only (Pub/Sub skipped) |
| `ailang messages list` | Firestore read | No (needs Firestore) |
| `ailang messages watch --pubsub` | Pub/Sub pull subscription | Queues up to 7 days |
| `ailang chains list` | Firestore read | No |
| `ailang coordinator status` | Firestore read | No |

### Behind NAT

Pull subscriptions use outbound HTTPS to `pubsub.googleapis.com` — no inbound connections needed. Works from any network.

### Offline Reconnection

1. Laptop goes offline
2. Messages accumulate in Pub/Sub (7-day retention)
3. Laptop comes back online
4. `ailang messages watch --pubsub` pulls accumulated messages
5. All messages delivered in order (per-inbox ordering key)

---

## Idempotency

Pub/Sub provides at-least-once delivery. All handlers must be idempotent:

| Handler | Idempotency Strategy |
|---------|---------------------|
| Message received | Check if message already exists in Firestore (by message_id) |
| Task dispatch | Firestore transaction: `pending` → `running` CAS. Skip if already running/completed |
| Task completion | Check task status before updating. Re-completing is a no-op |
| Event broadcast | Events are fire-and-forget, duplicates are harmless |

---

## Cost Analysis

### Pub/Sub Costs (100 tasks/day)

| Topic | Messages/Month | Avg Size | Cost/Month |
|-------|---------------|----------|------------|
| messages | ~3,000 | 200B | ~$0.01 |
| tasks | ~3,000 | 500B | ~$0.01 |
| completions | ~3,000 | 300B | ~$0.01 |
| events | ~300,000 | 500B | ~$0.06 |
| **Total** | | | **~$0.10** |

Pub/Sub cost is negligible. The dominant costs are Cloud Run compute and AI provider API calls.

### Cloud Run Costs (reference, detailed in ailang-multivac)

| Service | Monthly Cost (estimated) |
|---------|------------------------|
| Coordinator (0.5 vCPU, always-on) | ~$20 |
| Dashboard (0.5 vCPU, always-on) | ~$25 |
| Agent Jobs (2 vCPU, ~100 jobs × 15 min avg) | ~$15 |
| **Total compute** | **~$60** |

---

## Implementation Phases

### Phase 1: `internal/pubsub/` Package (Week 1)

**~400 LOC**. Pure Pub/Sub client with no coordinator dependencies.

Files:
- `internal/pubsub/client.go` — Client wrapper
- `internal/pubsub/publisher.go` — Publish functions
- `internal/pubsub/subscriber.go` — Pull subscription handler
- `internal/pubsub/topics.go` — Constants and payload types
- `internal/pubsub/client_test.go` — Unit tests (with emulator)

Testing: `PUBSUB_EMULATOR_HOST=localhost:8085 go test ./internal/pubsub/...`

### Phase 2: Coordinator Adapters (Week 1–2)

**~300 LOC**. Wire Pub/Sub into coordinator daemon.

Files:
- `internal/coordinator/pubsub_adapter.go` — PubSubInboxAdapter
- `internal/coordinator/pubsub_broadcaster.go` — PubSubBroadcaster
- Modify `daemon_tasks_init.go` — COORDINATOR_MODE switch

Testing: Integration test with Pub/Sub emulator + Firestore emulator.

### Phase 3: `execute-job` Subcommand (Week 2)

**~350 LOC**. Cloud Run Job entry point.

Files:
- `cmd/ailang/coordinator_cloud.go` — execute-job command
- Modify completion handling in daemon

Testing: Local test with `AILANG_TASK_ID=... ailang coordinator execute-job`.

### Phase 4: CLI Integration (Week 2–3)

**~200 LOC**. Dual-write in `messages send`, `--pubsub` flag for `watch`.

Files:
- Modify `cmd/ailang/messages_send.go` — Pub/Sub publish after Firestore write
- Modify `cmd/ailang/messages.go` — `watch --pubsub` mode
- Modify `cmd/ailang/config.go` — Pub/Sub config loading

Testing: End-to-end: `ailang messages send` on laptop → Pub/Sub → coordinator processes.

### Phase 5: Testing & Documentation (Week 3)

- Integration tests with both emulators
- Update CLAUDE.md with Pub/Sub config section
- Create `docs/docs/guides/cloud-pubsub.md`

---

## `ailang-multivac` Repository Skeleton

The deployment infrastructure lives in a **separate private repository**:

```
ailang-multivac/
├── terraform/
│   ├── main.tf                 # Provider config, GCS state backend
│   ├── variables.tf            # project_id, region, topic_prefix, etc.
│   ├── outputs.tf              # Service URLs, subscription names
│   ├── pubsub.tf               # 5 topics + subscriptions + dead letter
│   ├── cloud_run.tf            # Coordinator + Dashboard services
│   ├── cloud_run_jobs.tf       # Agent executor job
│   ├── eventarc.tf             # ailang-tasks → Cloud Run Job trigger
│   ├── iam.tf                  # Service accounts + Pub/Sub + Firestore roles
│   ├── secrets.tf              # Secret Manager: API keys, GitHub PAT
│   └── firestore.tf            # Firestore database + indexes
├── docker/
│   ├── Dockerfile.coordinator  # FROM golang:1.23 → build ailang → run coordinator
│   ├── Dockerfile.agent        # Same + Claude CLI + Gemini CLI + git
│   └── Dockerfile.dashboard    # Build ailang + ui/ React app
├── config/
│   ├── config.cloud.yaml       # Cloud coordinator config (no local paths)
│   └── config.example.yaml     # Template for new deployments
├── scripts/
│   ├── deploy.sh               # terraform apply + docker build + push
│   ├── setup-secrets.sh        # Initial ANTHROPIC_API_KEY, GITHUB_TOKEN provisioning
│   └── teardown.sh             # Clean removal
├── .github/
│   └── workflows/
│       └── deploy.yml          # CI/CD: build images, push to Artifact Registry, deploy
├── Makefile                    # deploy, plan, destroy, logs, status
└── README.md                   # Setup guide
```

### Key Terraform Resources (Reference)

```hcl
# terraform/pubsub.tf

resource "google_pubsub_topic" "messages" {
  name = "${var.prefix}-messages"
  message_retention_duration = "604800s"  # 7 days
}

resource "google_pubsub_topic" "tasks" {
  name = "${var.prefix}-tasks"
  message_retention_duration = "604800s"
}

resource "google_pubsub_topic" "completions" {
  name = "${var.prefix}-completions"
}

resource "google_pubsub_topic" "events" {
  name = "${var.prefix}-events"
}

resource "google_pubsub_topic" "dead_letter" {
  name = "${var.prefix}-dead-letter"
}

# Coordinator subscription (Cloud Run Service)
resource "google_pubsub_subscription" "messages_coordinator" {
  name  = "${var.prefix}-messages-coordinator"
  topic = google_pubsub_topic.messages.name

  enable_message_ordering = true

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }

  ack_deadline_seconds          = 60
  message_retention_duration    = "604800s"
}

# Laptop subscription (pull-based)
resource "google_pubsub_subscription" "messages_laptop" {
  name  = "${var.prefix}-messages-laptop"
  topic = google_pubsub_topic.messages.name

  enable_message_ordering = true
  ack_deadline_seconds    = 30
  message_retention_duration = "604800s"
}
```

---

## Relationship to M-CLOUD-INFRA

This design doc is a **focused subset** of the comprehensive [M-CLOUD-INFRA](../v0_8_2/m-cloud-infra.md) design.

**Key divergence:** M-CLOUD-INFRA uses per-agent topics (`ailang-inbox-{agent-id}`). This doc uses a **single `ailang-messages` topic** with attribute filtering instead. Rationale: agents are config-driven (20+ agents across 4 projects), per-agent topics require Terraform changes for each new agent, and attribute filtering achieves the same routing with zero infra changes.

Specifically:

| M-CLOUD-INFRA Scope | This Doc |
|---------------------|----------|
| Pub/Sub messaging layer | **Yes** — full implementation |
| Firestore storage | Already implemented |
| Cloud Run Services | Referenced (deployment in ailang-multivac) |
| Cloud Run Jobs | `execute-job` command (deployment in ailang-multivac) |
| BigQuery observatory | Out of scope |
| Secret Manager | Out of scope (ailang-multivac) |
| Terraform IaC | Skeleton only (ailang-multivac) |
| CI/CD pipeline | Out of scope (ailang-multivac) |

Implementing M-PUBSUB first provides the foundation for M-CLOUD-INFRA deployment.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Pub/Sub message ordering not guaranteed across topics | Tasks may process before message fully written to Firestore | Dual-write pattern: Firestore write completes before Pub/Sub publish |
| Cloud Run cold starts add latency | First message after idle takes 5-10s | min-instances=1 for coordinator |
| Git clone in Cloud Run Jobs is slow for large repos | Job startup delayed | Shallow clone (`--depth 1`) or pre-cached base image |
| Pub/Sub emulator doesn't support all features | Integration tests may miss edge cases | Test critical paths against real Pub/Sub in staging |
| Duplicate Pub/Sub delivery | Task executed twice | Firestore transaction CAS on task status |
