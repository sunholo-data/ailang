package apiserver

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// TRANSITIONAL - deleted in M2
type ToolDescriptor = protocol.ToolDescriptor

// TRANSITIONAL - deleted in M2
type AuthorizedSurface struct {
	*protocol.AuthorizedSurface
	tools []ToolDescriptor
}

// TRANSITIONAL - deleted in M2
var ErrCallbackCapacity = protocol.ErrCallbackCapacity

// TRANSITIONAL - deleted in M2
func callerSurface(v []ToolDescriptor) (*AuthorizedSurface, error) {
	surface, err := protocol.CallerSurface(v)
	if err != nil {
		return nil, err
	}
	return &AuthorizedSurface{AuthorizedSurface: surface, tools: surface.All()}, nil
}

// TRANSITIONAL - deleted in M2
func validateMCPName(v string) error { return protocol.ValidateMCPName(v) }

// TRANSITIONAL - deleted in M2
var mcpToolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// TRANSITIONAL - deleted in M2
type a2aRequest = protocol.A2ARequest

// TRANSITIONAL - deleted in M2
type a2aTaskSendParams = protocol.A2ATaskSendParams

// TRANSITIONAL - deleted in M2
type a2aMessage = protocol.A2AMessage

// TRANSITIONAL - deleted in M2
//
//nolint:unused // Preserved until embedded A2A wire machinery moves in M2.
type a2aContent = protocol.A2AContent

// TRANSITIONAL - deleted in M2
func a2aError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	protocol.A2AError(w, id, code, msg)
}

// TRANSITIONAL - deleted in M2
func a2aResult(w http.ResponseWriter, id json.RawMessage, result any) {
	protocol.A2AResult(w, id, result)
}

// TRANSITIONAL - deleted in M2
func requestID(body []byte) json.RawMessage { return protocol.RequestID(body) }

// TRANSITIONAL - deleted in M2
func writeMCPEnvelope(w http.ResponseWriter, id json.RawMessage, msg string) {
	protocol.WriteMCPEnvelope(w, id, msg)
}

// TRANSITIONAL - deleted in M2
func callbackMessage(err error) string { return protocol.CallbackMessage(err) }

// TRANSITIONAL - deleted in M2
func authorizationStatus(err error) int { return protocol.AuthorizationStatus(err) }
