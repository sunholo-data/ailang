package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	gpubsub "cloud.google.com/go/pubsub"
)

// Publisher provides high-level publish functions for AILANG Pub/Sub topics.
type Publisher struct {
	client *Client

	mu     sync.Mutex
	topics map[string]*gpubsub.Topic // cached topic handles
}

// NewPublisher creates a publisher from a client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{
		client: client,
		topics: make(map[string]*gpubsub.Topic),
	}
}

// topic returns a cached topic handle with message ordering enabled.
func (p *Publisher) topic(baseName string) *gpubsub.Topic {
	p.mu.Lock()
	defer p.mu.Unlock()

	if t, ok := p.topics[baseName]; ok {
		return t
	}
	t := p.client.Topic(baseName)
	t.EnableMessageOrdering = true
	p.topics[baseName] = t
	return t
}

// PublishMessage publishes a message notification to the messages topic.
// The actual message content is already stored in Firestore — this notification
// tells the coordinator (or laptop) that a new message is available.
func (p *Publisher) PublishMessage(ctx context.Context, messageID string, attrs MessageAttributes) error {
	payload, err := json.Marshal(MessageNotification{MessageID: messageID})
	if err != nil {
		return fmt.Errorf("marshal message notification: %w", err)
	}

	result := p.topic(TopicMessages).Publish(ctx, &gpubsub.Message{
		Data:        payload,
		Attributes:  attrs.ToMap(),
		OrderingKey: attrs.Inbox, // Messages to same inbox delivered in order
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish message notification (id=%s): %w", messageID, err)
	}
	return nil
}

// CascadeEnvelopeFields are the cascade-specific routing/classification fields
// the coordinator extracts at dispatch time. These map to attributes on the
// outgoing Pub/Sub message AND get embedded as JSON in the message data field
// (see PublishCascadeWithEnvelope) so the cloud coordinator can decide whether
// a deterministic bump is sufficient or AI escalation is needed without having
// to fetch the full envelope from a separate store.
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
type CascadeEnvelopeFields struct {
	RootPackage       string   `json:"root_package"` // vendor/name@version
	ChangeClass       string   `json:"change_class"` // "A" (content-only), "B" (additive), "C" (interface change)
	FromVersion       string   `json:"from_version"`
	ToVersion         string   `json:"to_version"`
	FromInterfaceHash string   `json:"from_interface_hash"`
	ToInterfaceHash   string   `json:"to_interface_hash"`
	FromContentHash   string   `json:"from_content_hash"`
	ToContentHash     string   `json:"to_content_hash"`
	EffectsWidened    bool     `json:"effects_widened"`
	PrevEffectCeiling []string `json:"prev_effect_ceiling,omitempty"`
	NewEffectCeiling  []string `json:"new_effect_ceiling,omitempty"`
}

// CascadeMessageData is the JSON shape of the Pub/Sub message data field
// for cascade messages. M-PKG-CASCADE-DETERMINISTIC-FIRST: previously this
// was just `{"message_id": "..."}` (a pointer); now it is self-contained so
// the cloud coordinator can dispatch deterministically without a separate
// fetch from an inbox store.
type CascadeMessageData struct {
	MessageID string                 `json:"message_id"`
	Envelope  *CascadeEnvelopeFields `json:"envelope,omitempty"`
}

// PublishCascade publishes a cascade-trigger notification to the cascade topic.
// (M-PKG-AUTONOMOUS-CASCADE-SAFE M2) The cascade topic's publish IAM is
// restricted to the coordinator service account at the GCP layer, so this
// method's success implicitly proves the caller is authorized — a stranger
// via the public MCP cannot reach this topic.
//
// Stamps `source=cascade` and `root_package=<vendor>/<name>@<version>` as
// message attributes alongside the standard inbox routing fields. The
// receiving pkg-* agent's pkg-update.md template uses {{.Source}} to
// distinguish authoritative bumps from public-routed feedback.
//
// For backwards compatibility this method accepts only the basic root_package
// signal. New code path is PublishCascadeWithEnvelope which carries the full
// hash + change_class context for deterministic dispatch.
func (p *Publisher) PublishCascade(ctx context.Context, messageID string, attrs MessageAttributes, rootPackage string) error {
	return p.PublishCascadeWithEnvelope(ctx, messageID, attrs, &CascadeEnvelopeFields{RootPackage: rootPackage})
}

// PublishCascadeWithEnvelope publishes a cascade-trigger notification with the
// full envelope (hashes, change_class, effect deltas) embedded in the message
// data field AND surfaced as Pub/Sub attributes. The cloud coordinator can
// then make deterministic dispatch decisions (deterministic toml bump vs AI
// escalation) without fetching from a separate store.
//
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func (p *Publisher) PublishCascadeWithEnvelope(ctx context.Context, messageID string, attrs MessageAttributes, env *CascadeEnvelopeFields) error {
	if env == nil {
		env = &CascadeEnvelopeFields{}
	}

	data := CascadeMessageData{
		MessageID: messageID,
		Envelope:  env,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal cascade message data: %w", err)
	}

	// Stamp the source so the receiving template guard can recognize it.
	// We always force this regardless of what the caller passed; cascade
	// topic publishes are authoritative bumps by construction.
	attrs.Source = SourceCascade

	attrMap := attrs.ToMap()
	// Surface envelope fields as attributes too — the cloud coordinator can
	// route based on these without decoding the data payload (cheap path).
	if env.RootPackage != "" {
		attrMap["root_package"] = env.RootPackage
	}
	if env.ChangeClass != "" {
		attrMap["change_class"] = env.ChangeClass
	}
	if env.FromVersion != "" {
		attrMap["from_version"] = env.FromVersion
	}
	if env.ToVersion != "" {
		attrMap["to_version"] = env.ToVersion
	}
	if env.EffectsWidened {
		attrMap["effects_widened"] = "true"
	}

	result := p.topic(TopicCascade).Publish(ctx, &gpubsub.Message{
		Data:        payload,
		Attributes:  attrMap,
		OrderingKey: attrs.Inbox, // Cascade messages to same dependent in order
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish cascade notification (id=%s, root=%s): %w", messageID, env.RootPackage, err)
	}
	return nil
}

// PublishTask publishes a task dispatch to the tasks topic.
// This triggers a Cloud Run Job via Eventarc to execute the task.
func (p *Publisher) PublishTask(ctx context.Context, taskID, agentID, workspace, provider string) error {
	payload, err := json.Marshal(TaskDispatch{TaskID: taskID, AgentID: agentID})
	if err != nil {
		return fmt.Errorf("marshal task dispatch: %w", err)
	}

	result := p.topic(TopicTasks).Publish(ctx, &gpubsub.Message{
		Data: payload,
		Attributes: map[string]string{
			"agent_id":  agentID,
			"workspace": workspace,
			"provider":  provider,
		},
		OrderingKey: agentID, // Tasks for same agent delivered in order
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish task dispatch (task=%s, agent=%s): %w", taskID, agentID, err)
	}
	return nil
}

// PublishCompletion publishes a task completion to the completions topic.
func (p *Publisher) PublishCompletion(ctx context.Context, completion TaskCompletion, workspace string) error {
	payload, err := json.Marshal(completion)
	if err != nil {
		return fmt.Errorf("marshal task completion: %w", err)
	}

	result := p.topic(TopicCompletions).Publish(ctx, &gpubsub.Message{
		Data: payload,
		Attributes: map[string]string{
			"agent_id":  completion.AgentID,
			"workspace": workspace,
			"status":    completion.Status,
		},
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish completion (task=%s): %w", completion.TaskID, err)
	}
	return nil
}

// PublishEvent publishes a real-time event to the events topic.
// Events are consumed by the dashboard and laptop for live streaming.
func (p *Publisher) PublishEvent(ctx context.Context, eventJSON []byte, eventType, taskID, workspace string) error {
	result := p.topic(TopicEvents).Publish(ctx, &gpubsub.Message{
		Data: eventJSON,
		Attributes: map[string]string{
			"event_type": eventType,
			"task_id":    taskID,
			"workspace":  workspace,
		},
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish event (type=%s, task=%s): %w", eventType, taskID, err)
	}
	return nil
}

// PublishApproval publishes a secret-approval push request to the approvals
// topic (M-SECRET-REMOTE-APPROVAL-WIRING). notificationJSON is a marshalled
// notify.Notification (built dashboard-side with signed Approve/Deny action
// URLs); it carries the reference, purpose, and agent — never a resolved value.
// The kind="approval" attribute lets the coordinator's /pubsub/push handler
// route it to the ntfy bridge.
func (p *Publisher) PublishApproval(ctx context.Context, approvalID, approvalType, agentID string, notificationJSON []byte) error {
	result := p.topic(TopicApprovals).Publish(ctx, &gpubsub.Message{
		Data: notificationJSON,
		Attributes: map[string]string{
			"kind":          "approval",
			"approval_id":   approvalID,
			"approval_type": approvalType,
			"agent_id":      agentID,
		},
		OrderingKey: approvalID,
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish approval (id=%s): %w", approvalID, err)
	}
	return nil
}

// Stop flushes all pending messages and releases topic resources.
func (p *Publisher) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.topics {
		t.Stop()
	}
}
