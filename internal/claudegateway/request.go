package claudegateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// decodeObject rejects duplicate keys at every depth. Parsing and forwarding
// different interpretations of a billable feature is never acceptable.
func decodeObject(raw []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := k.(string)
				if !ok || seen[s] {
					return errors.New("duplicate JSON key")
				}
				seen[s] = true
				if err = walk(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, errors.New("expected JSON object")
	}
	return obj, nil
}
func validateMessage(raw []byte) (model string, maxTokens int, stream bool, err error) {
	obj, err := decodeObject(raw)
	if err != nil {
		return "", 0, false, err
	}
	allowed := map[string]bool{"model": true, "max_tokens": true, "messages": true, "system": true, "tools": true, "tool_choice": true, "temperature": true, "top_p": true, "top_k": true, "stop_sequences": true, "stream": true, "metadata": true, "thinking": true, "output_config": true, "cache_control": true}
	for k := range obj {
		if !allowed[k] {
			return "", 0, false, fmt.Errorf("unsupported request field %s", k)
		}
	}
	if err = json.Unmarshal(obj["model"], &model); err != nil || model != "claude-haiku-5-5" {
		return "", 0, false, errors.New("unsupported model")
	}
	if err = json.Unmarshal(obj["max_tokens"], &maxTokens); err != nil || maxTokens <= 0 || maxTokens > 128000 {
		return "", 0, false, errors.New("invalid max_tokens")
	}
	if s, ok := obj["stream"]; ok {
		if err = json.Unmarshal(s, &stream); err != nil {
			return "", 0, false, err
		}
	}
	var tools []map[string]json.RawMessage
	if b, ok := obj["tools"]; ok {
		if err = json.Unmarshal(b, &tools); err != nil {
			return "", 0, false, err
		}
		for _, tool := range tools {
			if cache, ok := tool["cache_control"]; ok {
				if err = validateCache(cache); err != nil {
					return "", 0, false, err
				}
			}
			if _, ok := tool["type"]; ok {
				return "", 0, false, errors.New("server tools unsupported")
			}
			if strict, ok := tool["strict"]; ok {
				var v bool
				if json.Unmarshal(strict, &v) != nil {
					return "", 0, false, errors.New("invalid strict client tool")
				}
			}
			for k := range tool {
				if k != "name" && k != "description" && k != "input_schema" && k != "cache_control" && k != "strict" {
					return "", 0, false, fmt.Errorf("unsupported tool field %s", k)
				}
			}
		}
	}
	if b, ok := obj["thinking"]; ok {
		if err = validateThinking(b, maxTokens); err != nil {
			return "", 0, false, err
		}
	}
	if b, ok := obj["output_config"]; ok {
		if err = validateOutputConfig(b); err != nil {
			return "", 0, false, err
		}
	}
	if b, ok := obj["tool_choice"]; ok {
		if err = validateToolChoice(b); err != nil {
			return "", 0, false, err
		}
	}
	if b, ok := obj["metadata"]; ok {
		if err = validateMetadata(b); err != nil {
			return "", 0, false, err
		}
	}
	// Images, documents, server tools, fallbacks and context compaction need a
	// separate proved billing bound. Client tool schemas/inputs remain arbitrary.
	var messageObjects []json.RawMessage
	if err = json.Unmarshal(obj["messages"], &messageObjects); err != nil {
		return "", 0, false, errors.New("messages required")
	}
	for _, message := range messageObjects {
		if _, err = supportedObject(message, "role", "content"); err != nil {
			return "", 0, false, err
		}
	}
	var messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err = json.Unmarshal(obj["messages"], &messages); err != nil || len(messages) == 0 {
		return "", 0, false, errors.New("messages required")
	}
	for _, m := range messages {
		if m.Role != "user" && m.Role != "assistant" {
			return "", 0, false, errors.New("unsupported role")
		}
		if err = validateContent(m.Content); err != nil {
			return "", 0, false, err
		}
	}
	if b, ok := obj["system"]; ok {
		if err = validateContent(b); err != nil {
			return "", 0, false, err
		}
	}
	if b, ok := obj["cache_control"]; ok {
		if err = validateCache(b); err != nil {
			return "", 0, false, err
		}
	}
	return model, maxTokens, stream, nil
}
func validateContent(raw json.RawMessage) error {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return nil
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return errors.New("unsupported content")
	}
	for _, b := range blocks {
		var typ string
		json.Unmarshal(b["type"], &typ)
		switch typ {
		case "text", "thinking", "redacted_thinking", "tool_use":
		case "tool_result":
			if c, ok := b["content"]; ok {
				if err := validateContent(c); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unsupported content type %s", typ)
		}
		keys := map[string][]string{
			"text":              {"type", "text", "cache_control"},
			"thinking":          {"type", "thinking", "signature", "cache_control"},
			"redacted_thinking": {"type", "data", "cache_control"},
			"tool_use":          {"type", "id", "name", "input", "cache_control"},
			"tool_result":       {"type", "tool_use_id", "content", "is_error", "cache_control"},
		}
		allowed := map[string]bool{}
		for _, key := range keys[typ] {
			allowed[key] = true
		}
		for key := range b {
			if !allowed[key] {
				return fmt.Errorf("unsupported content field %s", key)
			}
		}
		if c, ok := b["cache_control"]; ok {
			if err := validateCache(c); err != nil {
				return err
			}
		}
	}
	return nil
}
func validateCache(raw json.RawMessage) error {
	var c struct {
		Type string `json:"type"`
		TTL  string `json:"ttl"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil || c.Type != "ephemeral" || (c.TTL != "" && c.TTL != "5m" && c.TTL != "1h") {
		return errors.New("unsupported cache contract")
	}
	return nil
}
