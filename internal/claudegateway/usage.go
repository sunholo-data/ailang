package claudegateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

func parseUsage(raw json.RawMessage, outputRequired bool) (modelreg.ClaudeUsage, error) {
	var u modelreg.ClaudeUsage
	obj, err := decodeObject(raw)
	if err != nil {
		return u, err
	}
	allowed := map[string]bool{"input_tokens": true, "output_tokens": true, "cache_read_input_tokens": true, "cache_creation_input_tokens": true, "cache_creation": true, "server_tool_use": true, "service_tier": true, "inference_geo": true}
	for k := range obj {
		if !allowed[k] {
			return u, errors.New("unsupported usage category")
		}
	}
	read := func(name string, required bool) (int64, error) {
		b, ok := obj[name]
		if !ok {
			if required {
				return 0, errors.New("incomplete usage")
			}
			return 0, nil
		}
		var n int64
		if err := json.Unmarshal(b, &n); err != nil || n < 0 {
			return 0, errors.New("invalid usage")
		}
		return n, nil
	}
	if u.InputTokens, err = read("input_tokens", true); err != nil {
		return u, err
	}
	if u.OutputTokens, err = read("output_tokens", outputRequired); err != nil {
		return u, err
	}
	if u.CacheReadTokens, err = read("cache_read_input_tokens", false); err != nil {
		return u, err
	}
	total, err := read("cache_creation_input_tokens", false)
	if err != nil {
		return u, err
	}
	if total > 0 {
		c, err := decodeObject(obj["cache_creation"])
		if err != nil {
			return u, errors.New("cache TTL usage missing")
		}
		if len(c) != 2 {
			return u, errors.New("cache TTL categories unsupported")
		}
		if err = json.Unmarshal(c["ephemeral_5m_input_tokens"], &u.CacheWrite5mTokens); err != nil {
			return u, err
		}
		if err = json.Unmarshal(c["ephemeral_1h_input_tokens"], &u.CacheWrite1hTokens); err != nil {
			return u, err
		}
		if u.CacheWrite5mTokens < 0 || u.CacheWrite1hTokens < 0 || u.CacheWrite5mTokens > total || u.CacheWrite1hTokens != total-u.CacheWrite5mTokens {
			return u, errors.New("cache usage inconsistent")
		}
	} else if b, ok := obj["cache_creation"]; ok && string(b) != "null" {
		var counts map[string]int64
		if err = json.Unmarshal(b, &counts); err != nil {
			return u, err
		}
		for _, n := range counts {
			if n != 0 {
				return u, errors.New("cache usage inconsistent")
			}
		}
	}
	if b, ok := obj["server_tool_use"]; ok && string(b) != "null" {
		var counts map[string]int64
		if err = json.Unmarshal(b, &counts); err != nil {
			return u, err
		}
		for _, n := range counts {
			if n != 0 {
				return u, errors.New("unsupported paid tool usage")
			}
		}
	}
	if b, ok := obj["service_tier"]; ok {
		var tier string
		if json.Unmarshal(b, &tier) != nil || tier != "standard" {
			return u, errors.New("unsupported usage service tier")
		}
	}
	if b, ok := obj["inference_geo"]; ok {
		var geo string
		if json.Unmarshal(b, &geo) != nil || (geo != "global" && geo != "") {
			return u, errors.New("unsupported inference residency")
		}
	}
	return u, nil
}
func messageUsage(raw []byte, model string, maxTokens int) (modelreg.ClaudeUsage, string, error) {
	var u modelreg.ClaudeUsage
	obj, err := decodeObject(raw)
	if err != nil {
		return u, "", err
	}
	var id, responseModel, typ string
	json.Unmarshal(obj["id"], &id)
	json.Unmarshal(obj["model"], &responseModel)
	json.Unmarshal(obj["type"], &typ)
	if id == "" || responseModel != model || typ != "message" {
		return u, "", errors.New("invalid upstream message identity")
	}
	u, err = parseUsage(obj["usage"], true)
	if err != nil || u.OutputTokens > int64(maxTokens) {
		return u, "", errors.New("invalid upstream usage")
	}
	return u, id, nil
}
func relayStream(w http.ResponseWriter, body io.Reader, model string, maxTokens int) (modelreg.ClaudeUsage, string, error) {
	var u modelreg.ClaudeUsage
	var id string
	started, delta, stopped := false, false, false
	// ReadSlice relays even very long tool content incrementally. Only the usage
	// event frame is accumulated, with a finite bound; full responses are not buffered.
	reader := bufio.NewReaderSize(body, 32<<10)
	var event bytes.Buffer
	process := func() error {
		raw := event.String()
		event.Reset()
		var data []string
		var eventType string
		for _, line := range strings.Split(raw, "\n") {
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if len(data) == 0 {
			return nil
		}
		obj, err := decodeObject([]byte(strings.Join(data, "\n")))
		if err != nil {
			return err
		}
		var typ string
		json.Unmarshal(obj["type"], &typ)
		if eventType != "" && eventType != typ {
			return errors.New("event type mismatch")
		}
		if stopped {
			return errors.New("event after message_stop")
		}
		switch typ {
		case "message_start":
			if started {
				return errors.New("duplicate message_start")
			}
			msg, err := decodeObject(obj["message"])
			if err != nil {
				return err
			}
			var responseModel string
			json.Unmarshal(msg["id"], &id)
			json.Unmarshal(msg["model"], &responseModel)
			if id == "" || responseModel != model {
				return errors.New("invalid stream identity")
			}
			u, err = parseUsage(msg["usage"], true)
			if err != nil {
				return err
			}
			started = true
		case "message_delta":
			if !started {
				return errors.New("unexpected message_delta")
			}
			delta = true
			usage, err := decodeObject(obj["usage"])
			if err != nil {
				return err
			}
			// Input/cache fields may be repeated only with identical values; output is
			// cumulative, not a delta to add twice.
			for k, b := range usage {
				switch k {
				case "output_tokens":
					var output int64
					if err = json.Unmarshal(b, &output); err != nil || output < u.OutputTokens {
						return errors.New("nonmonotonic output usage")
					}
					u.OutputTokens = output
				case "input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "cache_creation", "server_tool_use", "service_tier", "inference_geo":
					start := map[string]any{"input_tokens": u.InputTokens, "output_tokens": u.OutputTokens, "cache_read_input_tokens": u.CacheReadTokens, "cache_creation_input_tokens": u.CacheWrite5mTokens + u.CacheWrite1hTokens, "cache_creation": map[string]int64{"ephemeral_5m_input_tokens": u.CacheWrite5mTokens, "ephemeral_1h_input_tokens": u.CacheWrite1hTokens}}
					start[k] = json.RawMessage(b)
					encoded, _ := json.Marshal(start)
					check, err := parseUsage(encoded, true)
					if err != nil || check != u {
						return errors.New("inconsistent stream usage")
					}
				default:
					return errors.New("unsupported stream usage category")
				}
			}
			if _, ok := usage["output_tokens"]; !ok || u.OutputTokens < 0 || u.OutputTokens > int64(maxTokens) {
				return errors.New("missing or impossible stream output usage")
			}
		case "message_stop":
			if !started || !delta {
				return errors.New("truncated stream")
			}
			stopped = true
		case "ping":
		case "content_block_start", "content_block_delta", "content_block_stop":
			if !started || delta {
				return errors.New("invalid content sequence")
			}
		default:
			return errors.New("unsupported stream event")
		}
		return nil
	}
	for {
		line, readErr := reader.ReadSlice('\n')
		if len(line) > 0 {
			if event.Len()+len(line) > 8<<20 {
				return u, id, errors.New("stream event too large")
			}
			event.Write(line)
			if _, err := w.Write(line); err != nil {
				return u, id, err
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			if string(line) == "\n" || string(line) == "\r\n" {
				if err := process(); err != nil {
					return u, id, err
				}
			}
		}
		if readErr == bufio.ErrBufferFull {
			continue
		}
		if readErr == io.EOF {
			if event.Len() != 0 || !stopped {
				return u, id, errors.New("truncated stream")
			}
			return u, id, nil
		}
		if readErr != nil {
			return u, id, readErr
		}
	}
}
