package messaging

import (
	"fmt"
	"log"
	"time"
)

// Client provides an interface for AILANG instances to interact with the collaboration hub
type Client struct {
	store      *Store
	instanceID string
	pollTicker *time.Ticker
	stopCh     chan struct{}
}

// NewClient creates a new messaging client for an AILANG instance
func NewClient(dbPath string, instanceID string) (*Client, error) {
	store, err := OpenStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	return &Client{
		store:      store,
		instanceID: instanceID,
		stopCh:     make(chan struct{}),
	}, nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	if c.pollTicker != nil {
		c.pollTicker.Stop()
	}
	close(c.stopCh)
	return c.store.Close()
}

// PollMessages checks for new messages addressed to this instance
// Returns messages that haven't been acknowledged yet
func (c *Client) PollMessages() ([]*Message, error) {
	messages, err := c.store.GetMessages("ailang_instance", c.instanceID, "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to poll messages: %w", err)
	}

	// Convert []Message to []*Message
	result := make([]*Message, len(messages))
	for i := range messages {
		result[i] = &messages[i]
	}
	return result, nil
}

// StartPolling starts a background polling loop that checks for new messages every interval
// The callback is invoked with each batch of new messages
func (c *Client) StartPolling(interval time.Duration, callback func([]*Message) error) {
	c.pollTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-c.pollTicker.C:
				messages, err := c.PollMessages()
				if err != nil {
					log.Printf("Polling error: %v", err)
					continue
				}

				if len(messages) > 0 {
					if err := callback(messages); err != nil {
						log.Printf("Callback error: %v", err)
					}
				}

			case <-c.stopCh:
				return
			}
		}
	}()
}

// StopPolling stops the background polling loop
func (c *Client) StopPolling() {
	if c.pollTicker != nil {
		c.pollTicker.Stop()
	}
}

// PublishMessage sends a message to a recipient
func (c *Client) PublishMessage(threadID, toType, toID, kind, content string) (*Message, error) {
	msg, err := c.store.CreateMessage(
		threadID,
		"ailang_instance", c.instanceID, // from
		toType, toID, // to
		kind,
		content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}
	return msg, nil
}

// SendStatus sends a status update message to the human in a thread
func (c *Client) SendStatus(threadID, status string) (*Message, error) {
	return c.PublishMessage(threadID, "human", "user", "status", status)
}

// SendQuestion sends a question to the human in a thread
func (c *Client) SendQuestion(threadID, question string) (*Message, error) {
	return c.PublishMessage(threadID, "human", "user", "question", question)
}

// SendResult sends a completion result to the human in a thread
func (c *Client) SendResult(threadID, result string) (*Message, error) {
	return c.PublishMessage(threadID, "human", "user", "result", result)
}

// SendStatusToAgent sends a status update to another agent instance
func (c *Client) SendStatusToAgent(threadID, targetAgentID, status string) (*Message, error) {
	return c.PublishMessage(threadID, "ailang_instance", targetAgentID, "status", status)
}

// BroadcastStatus sends a status update to all agents watching a thread
func (c *Client) BroadcastStatus(threadID, status string) (*Message, error) {
	return c.PublishMessage(threadID, "broadcast", "", "status", status)
}

// AcknowledgeMessage marks a message as acknowledged
func (c *Client) AcknowledgeMessage(messageID string) error {
	if err := c.store.MarkAsAcked(messageID); err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}
	return nil
}

// ClaimMessage atomically claims a message for processing.
// This prevents other agents from processing the same message.
// Returns nil if successfully claimed, error if already claimed by another agent.
func (c *Client) ClaimMessage(messageID string) error {
	if err := c.store.ClaimMessage(messageID, c.instanceID); err != nil {
		return fmt.Errorf("failed to claim message: %w", err)
	}
	return nil
}

// RequestApproval creates an approval request for effect-gated actions
// Returns the approval ID that can be checked later
func (c *Client) RequestApproval(threadID string, effectDelta *EffectDelta, proposal, impact string, estimatedCost float64) (string, error) {
	approval, err := c.store.CreateApproval(threadID, c.instanceID, effectDelta, proposal, impact, estimatedCost)
	if err != nil {
		return "", fmt.Errorf("failed to request approval: %w", err)
	}

	// Also send a message to notify the human
	approvalMsg := fmt.Sprintf("Approval requested: %s (impact: %s, cost: $%.2f)", proposal, impact, estimatedCost)
	_, _ = c.PublishMessage(threadID, "human", "user", "approval_request", approvalMsg)

	return approval.ID, nil
}

// CheckApprovalStatus checks the status of an approval request
func (c *Client) CheckApprovalStatus(approvalID string) (string, error) {
	approval, err := c.store.GetApproval(approvalID)
	if err != nil {
		return "", fmt.Errorf("failed to get approval: %w", err)
	}
	return approval.Status, nil
}

// WaitForApproval waits for an approval request to be approved or rejected
// Returns true if approved, false if rejected
// Timeout after the specified duration
func (c *Client) WaitForApproval(approvalID string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := c.CheckApprovalStatus(approvalID)
		if err != nil {
			return false, err
		}

		switch status {
		case "approved":
			return true, nil
		case "rejected":
			return false, nil
		case "pending":
			// Keep waiting
			time.Sleep(1 * time.Second)
		default:
			return false, fmt.Errorf("unexpected approval status: %s", status)
		}
	}

	return false, fmt.Errorf("approval request timed out after %v", timeout)
}

// GetCapabilityToken retrieves the capability token for an approved request
func (c *Client) GetCapabilityToken(approvalID string) (string, error) {
	approval, err := c.store.GetApproval(approvalID)
	if err != nil {
		return "", fmt.Errorf("failed to get approval: %w", err)
	}

	if approval.Status != "approved" {
		return "", fmt.Errorf("approval not approved (status: %s)", approval.Status)
	}

	if approval.CapabilityToken == "" {
		return "", fmt.Errorf("no capability token available")
	}

	return approval.CapabilityToken, nil
}

// SubscribeToThread subscribes this instance to a thread
func (c *Client) SubscribeToThread(threadID string) error {
	if err := c.store.Subscribe(c.instanceID, threadID); err != nil {
		return fmt.Errorf("failed to subscribe to thread: %w", err)
	}
	return nil
}

// UpdateAckSeq updates the last acknowledged sequence number for a thread
func (c *Client) UpdateAckSeq(threadID string, ackSeq int) error {
	if err := c.store.UpdateAckSeq(c.instanceID, threadID, ackSeq); err != nil {
		return fmt.Errorf("failed to update ack seq: %w", err)
	}
	return nil
}

// GetThread retrieves thread information
func (c *Client) GetThread(threadID string) (*Thread, error) {
	thread, err := c.store.GetThread(threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	return thread, nil
}
