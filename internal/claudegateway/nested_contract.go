package claudegateway

import (
	"encoding/json"
	"errors"
	"fmt"
)

func supportedObject(raw json.RawMessage, allowed ...string) (map[string]json.RawMessage, error) {
	obj, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, k := range allowed {
		names[k] = true
	}
	for k := range obj {
		if !names[k] {
			return nil, fmt.Errorf("unsupported nested field %s", k)
		}
	}
	return obj, nil
}
func enum(raw json.RawMessage, allowed ...string) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return errors.New("invalid feature value")
	}
	for _, x := range allowed {
		if value == x {
			return nil
		}
	}
	return errors.New("unsupported feature value")
}
func validateThinking(raw json.RawMessage, maxTokens int) error {
	obj, err := supportedObject(raw, "type", "budget_tokens", "display")
	if err != nil {
		return err
	}
	if err = enum(obj["type"], "enabled", "adaptive", "disabled"); err != nil {
		return err
	}
	var typ string
	json.Unmarshal(obj["type"], &typ)
	if b, ok := obj["budget_tokens"]; ok {
		var n int
		if typ != "enabled" || json.Unmarshal(b, &n) != nil || n < 1024 || n >= maxTokens {
			return errors.New("unsupported thinking budget")
		}
	} else if typ == "enabled" {
		return errors.New("thinking budget required")
	}
	if b, ok := obj["display"]; ok {
		if typ == "disabled" {
			return errors.New("disabled thinking display unsupported")
		}
		if err = enum(b, "summarized", "omitted"); err != nil {
			return err
		}
	}
	return nil
}
func validateOutputConfig(raw json.RawMessage) error {
	obj, err := supportedObject(raw, "effort")
	if err != nil {
		return err
	}
	if b, ok := obj["effort"]; ok {
		return enum(b, "low", "medium", "high", "max")
	}
	return nil
}
func validateToolChoice(raw json.RawMessage) error {
	obj, err := supportedObject(raw, "type", "name", "disable_parallel_tool_use")
	if err != nil {
		return err
	}
	if err = enum(obj["type"], "auto", "any", "tool", "none"); err != nil {
		return err
	}
	var typ string
	json.Unmarshal(obj["type"], &typ)
	name, hasName := obj["name"]
	if typ == "tool" {
		var n string
		if !hasName || json.Unmarshal(name, &n) != nil || n == "" {
			return errors.New("tool name required")
		}
	} else if hasName {
		return errors.New("unexpected tool choice name")
	}
	if b, ok := obj["disable_parallel_tool_use"]; ok {
		var v bool
		if json.Unmarshal(b, &v) != nil {
			return errors.New("invalid parallel tool choice")
		}
	}
	return nil
}
func validateMetadata(raw json.RawMessage) error {
	obj, err := supportedObject(raw, "user_id")
	if err != nil {
		return err
	}
	if b, ok := obj["user_id"]; ok {
		var v string
		if json.Unmarshal(b, &v) != nil || len(v) > 4096 {
			return errors.New("invalid metadata user_id")
		}
	}
	return nil
}
