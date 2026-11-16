package apiserver

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// unwrapMCPResult gives an MCP tool that returns Result[T, E] the same meaning
// a @route handler already has (resultErrStatus): Ok(v) answers with v, and
// Err(e) is a tool error (isError: true) whose text is e. Before this, a
// Result crossed MCP as {"__tag": "Err", ...} in a successful result, so a
// client could not tell a refusal from an answer. Any other value passes
// through unchanged. A non-nil *CallToolResult means the value was an Err.
func unwrapMCPResult(v eval.Value) (eval.Value, *mcp.CallToolResult) {
	tagged, ok := v.(*eval.TaggedValue)
	if !ok || tagged.TypeName != "Result" {
		return v, nil
	}
	switch tagged.CtorName {
	case "Ok":
		if len(tagged.Fields) == 0 {
			return &eval.UnitValue{}, nil
		}
		return tagged.Fields[0], nil
	case "Err":
		if len(tagged.Fields) == 0 {
			return nil, mcpError("error")
		}
		return nil, mcpError(errPayloadText(tagged.Fields[0]))
	}
	return v, nil
}

// errPayloadText renders an Err payload as the tool error's text: a string as
// itself, anything else as JSON.
func errPayloadText(p eval.Value) string {
	if s, ok := p.(*eval.StringValue); ok {
		return s.Value
	}
	goVal, err := embed.ToGo(p)
	if err != nil {
		return fmt.Sprintf("error (unconvertible payload: %v)", err)
	}
	b, err := json.Marshal(goVal)
	if err != nil {
		return fmt.Sprintf("error (unserializable payload: %v)", err)
	}
	return string(b)
}
