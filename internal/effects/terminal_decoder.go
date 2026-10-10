package effects

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/eval"
	"time"
	"unicode/utf8"
)

const terminalEscapeDelay = 30 * time.Millisecond
const terminalControlLimit = 64

type terminalDecoder struct {
	pending       []byte
	escapeAt      time.Time
	expiredEscape int
}

func terminalTag(name string, fields ...eval.Value) eval.Value {
	module, typeName := "std/terminal", ""
	switch name {
	case "Some", "None":
		module, typeName = "std/option", "Option"
	case "Ok", "Err":
		module, typeName = "std/result", "Result"
	case "TerminalSession":
		typeName = "TerminalSession"
	case "Text", "Up", "Down", "Left", "Right", "Enter", "Escape", "Backspace", "Tab", "Home", "End", "PageUp", "PageDown", "Delete":
		typeName = "TerminalKey"
	case "Key", "Resize", "EndOfInput", "Interrupted", "Idle":
		typeName = "TerminalEvent"
	case "Unsupported", "NotTTY", "Busy", "InvalidSession", "InvalidTimeout", "InputFailure", "QueryFailure", "CleanupFailure":
		typeName = "TerminalError"
	}
	return &eval.TaggedValue{CtorName: name, ModulePath: module, TypeName: typeName, Fields: fields}
}
func terminalText(text string) eval.Value {
	return terminalTag("Key", terminalTag("Text", &eval.StringValue{Value: text}))
}
func terminalKey(name string) eval.Value { return terminalTag("Key", terminalTag(name)) }

func (d *terminalDecoder) feed(data []byte, now time.Time) error {
	if len(d.pending) > 0 && d.pending[0] == 27 && !d.escapeAt.IsZero() && now.Sub(d.escapeAt) >= terminalEscapeDelay {
		if len(d.pending) == 1 {
			d.expiredEscape = 1
		} else {
			complete := false
			for _, b := range d.pending[2:] {
				if b >= 0x40 && b <= 0x7e {
					complete = true
					break
				}
			}
			if !complete {
				d.expiredEscape = 2
			}
		}
	}
	if len(d.pending)+len(data) > 4096 {
		return fmt.Errorf("terminal input queue exceeds 4096 bytes")
	}
	if len(d.pending) == 0 && len(data) > 0 && data[0] == 27 {
		d.escapeAt = now
	}
	d.pending = append(d.pending, data...)
	if len(d.pending) > terminalControlLimit && d.pending[0] == 27 {
		// Only the undecided sequence is bounded; complete keys may be queued.
		for i, b := range d.pending[2:] {
			if b >= 0x40 && b <= 0x7e {
				if i+3 > terminalControlLimit {
					return fmt.Errorf("control sequence exceeds 64 bytes")
				}
				return nil
			}
			if i+3 > terminalControlLimit {
				return fmt.Errorf("control sequence exceeds 64 bytes")
			}
		}
		return fmt.Errorf("control sequence exceeds 64 bytes")
	}
	return nil
}
func (d *terminalDecoder) consume(n int, now time.Time) {
	d.pending = d.pending[n:]
	d.escapeAt = time.Time{}
	d.expiredEscape = 0
	if len(d.pending) > 0 && d.pending[0] == 27 {
		d.escapeAt = now
	}
}
func (d *terminalDecoder) next(now time.Time, eof bool) (eval.Value, error) {
	if len(d.pending) == 0 {
		return nil, nil
	}
	b := d.pending[0]
	if b == 27 {
		return d.escape(now, eof)
	}
	names := map[byte]string{'\r': "Enter", '\n': "Enter", 127: "Backspace", 8: "Backspace", '\t': "Tab", 4: "EndOfInput", 3: "Interrupted"}
	if name, ok := names[b]; ok {
		d.consume(1, now)
		if name == "EndOfInput" || name == "Interrupted" {
			return terminalTag(name), nil
		}
		return terminalKey(name), nil
	}
	if b < 32 {
		return nil, fmt.Errorf("unsupported control byte 0x%02x", b)
	}
	if !utf8.FullRune(d.pending) {
		if eof {
			return nil, fmt.Errorf("incomplete UTF-8 at EOF")
		}
		return nil, nil
	}
	r, n := utf8.DecodeRune(d.pending)
	if r == utf8.RuneError && n == 1 {
		return nil, fmt.Errorf("invalid UTF-8")
	}
	if r >= 0x80 && r <= 0x9f {
		return nil, fmt.Errorf("unsupported control character U+%04X", r)
	}
	text := string(d.pending[:n])
	d.consume(n, now)
	return terminalText(text), nil
}
func (d *terminalDecoder) escape(now time.Time, eof bool) (eval.Value, error) {
	if d.expiredEscape == 1 {
		d.consume(1, now)
		return terminalKey("Escape"), nil
	}
	if d.expiredEscape == 2 {
		return nil, fmt.Errorf("incomplete escape sequence at ambiguity deadline")
	}

	if d.escapeAt.IsZero() {
		d.escapeAt = now
	}
	if len(d.pending) == 1 {
		if !eof && now.Sub(d.escapeAt) < terminalEscapeDelay {
			return nil, nil
		}
		d.consume(1, now)
		return terminalKey("Escape"), nil
	}
	if d.pending[1] != '[' && d.pending[1] != 'O' {
		return nil, fmt.Errorf("unsupported escape sequence")
	}
	end := 0
	for i := 2; i < len(d.pending); i++ {
		b := d.pending[i]
		if i >= terminalControlLimit {
			return nil, fmt.Errorf("control sequence exceeds 64 bytes")
		}
		if b >= 0x40 && b <= 0x7e {
			end = i + 1
			break
		}
		if b < 0x20 || b > 0x3f {
			return nil, fmt.Errorf("malformed control sequence")
		}
	}
	if end == 0 {
		if eof || now.Sub(d.escapeAt) >= terminalEscapeDelay {
			return nil, fmt.Errorf("incomplete escape sequence")
		}
		return nil, nil
	}
	keys := map[string]string{"[A": "Up", "[B": "Down", "[C": "Right", "[D": "Left", "[H": "Home", "[F": "End", "OA": "Up", "OB": "Down", "OC": "Right", "OD": "Left", "OH": "Home", "OF": "End", "[1~": "Home", "[4~": "End", "[7~": "Home", "[8~": "End", "[5~": "PageUp", "[6~": "PageDown", "[3~": "Delete"}
	name, ok := keys[string(d.pending[1:end])]
	if !ok {
		return nil, fmt.Errorf("unsupported control sequence")
	}
	d.consume(end, now)
	return terminalKey(name), nil
}
