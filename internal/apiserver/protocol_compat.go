package apiserver

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

func validateMCPName(v string) error { return protocol.ValidateMCPName(v) }

var mcpToolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type a2aRequest = protocol.A2ARequest

type a2aTaskSendParams = protocol.A2ATaskSendParams

func a2aError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	protocol.A2AError(w, id, code, msg)
}

func a2aResult(w http.ResponseWriter, id json.RawMessage, result any) {
	protocol.A2AResult(w, id, result)
}
