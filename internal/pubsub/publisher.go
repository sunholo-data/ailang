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

// Stop flushes all pending messages and releases topic resources.
func (p *Publisher) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range p.topics {
		t.Stop()
	}
}
