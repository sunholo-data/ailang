package effects

import (
	"testing"
	"time"
)

func TestTerminalDecoderSplitAndEscape(t *testing.T) {
	d := &terminalDecoder{}
	now := time.Now()
	if err := d.feed([]byte{0xe2, 0x82}, now); err != nil {
		t.Fatal(err)
	}
	if ev, err := d.next(now, false); ev != nil || err != nil {
		t.Fatalf("partial UTF8: %v %v", ev, err)
	}
	_ = d.feed([]byte{0xac, 27, '['}, now)
	ev, err := d.next(now, false)
	if err != nil || ev == nil || ev.String() != "Key(Text(€))" {
		t.Fatalf("text: %v %v", ev, err)
	}
	if ev, err = d.next(now, false); ev != nil || err != nil {
		t.Fatalf("partial CSI: %v %v", ev, err)
	}
	_ = d.feed([]byte{'A'}, now.Add(10*time.Millisecond))
	ev, err = d.next(now, false)
	if err != nil || ev.String() != "Key(Up)" {
		t.Fatalf("up: %v %v", ev, err)
	}
	_ = d.feed([]byte{27}, now)
	if ev, _ = d.next(now.Add(29*time.Millisecond), false); ev != nil {
		t.Fatal("Escape resolved early")
	}
	ev, err = d.next(now.Add(30*time.Millisecond), false)
	if err != nil || ev.String() != "Key(Escape)" {
		t.Fatalf("escape: %v %v", ev, err)
	}
}

func TestTerminalDecoderRejectsUnsafeAndPartialEOF(t *testing.T) {
	for _, input := range [][]byte{{0xff}, {27, '[', '?', '9', '9', 'h'}, {0xe2}, {27, '['}} {
		d := &terminalDecoder{}
		err := d.feed(input, time.Now())
		if err == nil {
			_, err = d.next(time.Now(), true)
		}
		if err == nil {
			t.Fatalf("accepted malformed/unsafe bytes: %v", input)
		}
	}
	d := &terminalDecoder{}
	if err := d.feed(append([]byte{27, '['}, make([]byte, 65)...), time.Now()); err == nil {
		t.Fatal("oversized control accepted")
	}
}

func TestTerminalDecoderLateContinuationCannotExtendDeadline(t *testing.T) {
	now := time.Now()
	d := &terminalDecoder{}
	_ = d.feed([]byte("\x1b["), now)
	_ = d.feed([]byte("A"), now.Add(31*time.Millisecond))
	if _, err := d.next(now.Add(31*time.Millisecond), false); err == nil {
		t.Fatal("late completed CSI escaped ambiguity deadline")
	}
	d = &terminalDecoder{}
	_ = d.feed([]byte("\x1b"), now)
	_ = d.feed([]byte("[A"), now.Add(31*time.Millisecond))
	event, err := d.next(now.Add(31*time.Millisecond), false)
	if err != nil || event.String() != "Key(Escape)" {
		t.Fatalf("late bytes prevented lone Escape: %v %v", event, err)
	}
}
