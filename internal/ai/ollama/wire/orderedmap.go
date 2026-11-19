package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

// errMalformedObject is the text the SDK's JSON object walker
// (github.com/buger/jsonparser, under its ordered map) returned for a
// non-object value. It reaches callers verbatim (a tool call whose
// arguments arrive as a JSON string fails the stream with it), so it is
// reproduced exactly.
var errMalformedObject = errors.New("Value looks like object, but can't find closing '}' symbol") //nolint:staticcheck // ST1005: wire-compatible text

// orderedMap is a JSON object that keeps its keys in document order, as the
// SDK's internal/orderedmap (github.com/wk8/go-ordered-map/v2) did. null marks
// a map that was decoded from JSON null: the SDK then held a nil inner map
// and marshalled it back as null.
type orderedMap[V any] struct {
	keys []string
	vals map[string]V
	null bool
}

// set inserts or updates a key; an update keeps the key's first position.
func (m *orderedMap[V]) set(k string, v V) {
	if m.vals == nil {
		m.vals = map[string]V{}
	}
	if _, ok := m.vals[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.vals[k] = v
}

// MarshalJSON writes keys in insertion order. Keys are escaped the way
// easyjson's jwriter escaped them under the SDK (control characters other
// than \t \n \r as \u00XX — encoding/json would write \b and \f); values go
// through encoding/json.
func (m *orderedMap[V]) MarshalJSON() ([]byte, error) {
	if m == nil || m.null {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeKey(&buf, k)
		buf.WriteByte(':')
		v, err := json.Marshal(m.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON reads a JSON object in document order. encoding/json has
// already validated data by the time it calls this, so the only failures
// are a non-object value and a value that does not decode into V. Keys set
// before a failing value stay set, as they did under the SDK.
func (m *orderedMap[V]) UnmarshalJSON(data []byte) error {
	*m = orderedMap[V]{}
	if string(data) == "null" {
		m.null = true
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return errMalformedObject
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := tok.(string)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		var v V
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		m.set(key, v)
	}
	return nil
}

const hexDigits = "0123456789abcdef"

// writeKey is easyjson's jwriter.Writer.String with HTML escaping on.
func writeKey(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	p := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' && c != '<' && c != '>' && c != '&' {
				i++
				continue
			}
			buf.WriteString(s[p:i])
			switch c {
			case '\t':
				buf.WriteString(`\t`)
			case '\r':
				buf.WriteString(`\r`)
			case '\n':
				buf.WriteString(`\n`)
			case '\\':
				buf.WriteString(`\\`)
			case '"':
				buf.WriteString(`\"`)
			default:
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigits[c>>4])
				buf.WriteByte(hexDigits[c&0xf])
			}
			i++
			p = i
			continue
		}
		r, w := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && w == 1 {
			buf.WriteString(s[p:i])
			buf.WriteString(`�`)
			i++
			p = i
			continue
		}
		if r == ' ' || r == ' ' {
			buf.WriteString(s[p:i])
			buf.WriteString(`\u202`)
			buf.WriteByte(hexDigits[r&0xf])
			i += w
			p = i
			continue
		}
		i += w
	}
	buf.WriteString(s[p:])
	buf.WriteByte('"')
}
