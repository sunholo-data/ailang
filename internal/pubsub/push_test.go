package pubsub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodePushEnvelope_Valid(t *testing.T) {
	payload := `{"message_id":"abc-123"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	envelope := PushEnvelope{
		Message: PushMessage{
			Data:      encoded,
			MessageID: "pub-msg-001",
			Attributes: map[string]string{
				"inbox":      "design-doc-creator",
				"from_agent": "user",
			},
			PublishTime: "2026-03-06T15:00:00Z",
		},
		Subscription: "projects/my-project/subscriptions/ailang-dev-messages-coordinator",
	}

	body, _ := json.Marshal(envelope)
	data, attrs, msgID, err := DecodePushEnvelope(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != payload {
		t.Errorf("data = %q, want %q", string(data), payload)
	}
	if msgID != "pub-msg-001" {
		t.Errorf("messageID = %q, want %q", msgID, "pub-msg-001")
	}
	if attrs["inbox"] != "design-doc-creator" {
		t.Errorf("attrs[inbox] = %q, want %q", attrs["inbox"], "design-doc-creator")
	}
	if attrs["from_agent"] != "user" {
		t.Errorf("attrs[from_agent] = %q, want %q", attrs["from_agent"], "user")
	}
}

func TestDecodePushEnvelope_EmptyData(t *testing.T) {
	envelope := PushEnvelope{
		Message: PushMessage{
			Data:      "",
			MessageID: "pub-msg-002",
			Attributes: map[string]string{
				"inbox": "coordinator",
			},
		},
	}

	body, _ := json.Marshal(envelope)
	data, attrs, msgID, err := DecodePushEnvelope(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data != nil {
		t.Errorf("data = %v, want nil for empty data", data)
	}
	if msgID != "pub-msg-002" {
		t.Errorf("messageID = %q, want %q", msgID, "pub-msg-002")
	}
	if attrs["inbox"] != "coordinator" {
		t.Errorf("attrs[inbox] = %q, want %q", attrs["inbox"], "coordinator")
	}
}

func TestDecodePushEnvelope_MalformedJSON(t *testing.T) {
	_, _, _, err := DecodePushEnvelope(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDecodePushEnvelope_InvalidBase64(t *testing.T) {
	envelope := PushEnvelope{
		Message: PushMessage{
			Data:      "not-valid-base64!!!",
			MessageID: "pub-msg-003",
		},
	}

	body, _ := json.Marshal(envelope)
	_, _, _, err := DecodePushEnvelope(strings.NewReader(string(body)))
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodePushEnvelope_NoAttributes(t *testing.T) {
	payload := `{"task_id":"task-001"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	envelope := PushEnvelope{
		Message: PushMessage{
			Data:      encoded,
			MessageID: "pub-msg-004",
		},
	}

	body, _ := json.Marshal(envelope)
	data, attrs, msgID, err := DecodePushEnvelope(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != payload {
		t.Errorf("data = %q, want %q", string(data), payload)
	}
	if msgID != "pub-msg-004" {
		t.Errorf("messageID = %q, want %q", msgID, "pub-msg-004")
	}
	if len(attrs) != 0 {
		t.Errorf("attrs = %v, want empty", attrs)
	}
}
