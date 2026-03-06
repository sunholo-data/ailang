package pubsub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// PushEnvelope is the JSON body POSTed by Pub/Sub to push subscription endpoints.
// See: https://cloud.google.com/pubsub/docs/push#receive_push
type PushEnvelope struct {
	Message      PushMessage `json:"message"`
	Subscription string      `json:"subscription"`
}

// PushMessage represents the message within a Pub/Sub push envelope.
type PushMessage struct {
	Data        string            `json:"data"` // base64-encoded payload
	MessageID   string            `json:"messageId"`
	Attributes  map[string]string `json:"attributes"`
	PublishTime string            `json:"publishTime"`
}

// DecodePushEnvelope reads a Pub/Sub push HTTP body, unmarshals the envelope,
// and base64-decodes the Data field. Returns the decoded data bytes,
// attributes map, and message ID.
func DecodePushEnvelope(body io.Reader) (data []byte, attrs map[string]string, messageID string, err error) {
	var envelope PushEnvelope
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return nil, nil, "", fmt.Errorf("decode push envelope: %w", err)
	}

	if envelope.Message.Data == "" {
		return nil, envelope.Message.Attributes, envelope.Message.MessageID, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		return nil, nil, "", fmt.Errorf("base64 decode push data: %w", err)
	}

	return decoded, envelope.Message.Attributes, envelope.Message.MessageID, nil
}
