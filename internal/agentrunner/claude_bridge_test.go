package agentrunner

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/agentprotocol"
)

func TestClaudeAgentHandler(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock agent file
	agentFile := tmpDir + "/test-agent.md"
	err := os.WriteFile(agentFile, []byte("# Test Agent\nThis is a test agent."), 0644)
	require.NoError(t, err)

	handler := NewClaudeAgentHandler(agentFile, tmpDir)

	msg := &agentprotocol.Envelope{
		MessageID:  "msg_001",
		FromAgent:  "sender",
		ToAgent:    "receiver",
		MessageType: "request",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Payload: map[string]interface{}{
			"task": "test",
		},
	}

	response, err := handler.HandleMessage(msg)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response, "status")
}

func TestSkillHandler(t *testing.T) {
	tmpDir := t.TempDir()

	handler := NewSkillHandler("test-skill", tmpDir)

	msg := &agentprotocol.Envelope{
		MessageID:  "msg_001",
		FromAgent:  "sender",
		ToAgent:    "receiver",
		MessageType: "request",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	response, err := handler.HandleMessage(msg)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "no_scripts", response["status"])
}

func TestCommandHandler(t *testing.T) {
	tmpDir := t.TempDir()

	handler := NewCommandHandler("echo", []string{"test"}, tmpDir)

	msg := &agentprotocol.Envelope{
		MessageID:  "msg_001",
		FromAgent:  "sender",
		ToAgent:    "receiver",
		MessageType: "request",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	response, err := handler.HandleMessage(msg)
	require.NoError(t, err)
	assert.Equal(t, "completed", response["status"])
	assert.Contains(t, response["output"], "test")
}

func TestFunctionHandler(t *testing.T) {
	handler := NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
		return map[string]interface{}{
			"status": "success",
			"from":   msg.FromAgent,
		}, nil
	})

	msg := &agentprotocol.Envelope{
		MessageID:  "msg_001",
		FromAgent:  "test-sender",
		ToAgent:    "receiver",
		MessageType: "request",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	response, err := handler.HandleMessage(msg)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "test-sender", response["from"])
}
