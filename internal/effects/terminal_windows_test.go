//go:build windows

package effects

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestTerminalWindowsDiskIdentityAndBorrowedHandle(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	alias, err := os.Open(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer alias.Close()
	other, err := os.CreateTemp(t.TempDir(), "other-input")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	key, err := terminalDeviceKey(int(file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	aliasKey, err := terminalDeviceKey(int(alias.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := terminalDeviceKey(int(other.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if key != aliasKey {
		t.Fatalf("opened file aliases differ: %q/%q", key, aliasKey)
	}
	if key == otherKey {
		t.Fatalf("distinct files collide: %q", key)
	}
	if _, err := file.WriteString("still owned by caller"); err != nil {
		t.Fatalf("identity query invalidated borrowed handle: %v", err)
	}
}

func TestTerminalWindowsPipeIdentityAndInvalidHandle(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	first, err := terminalDeviceKey(int(input.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	second, err := terminalDeviceKey(int(input.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same pipe handle identity changed: %q/%q", first, second)
	}
	invalid := windows.InvalidHandle
	if _, err := terminalDeviceKey(int(invalid)); err == nil {
		t.Fatal("invalid handle identity silently accepted")
	}
	if terminalSupported() {
		t.Fatal("Windows file identity must not advertise native terminal support")
	}
}

func TestTerminalWindowsPipeLineIO(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	if _, err = output.WriteString("\nfinal"); err != nil {
		t.Fatal(err)
	}
	if err = output.Close(); err != nil {
		t.Fatal(err)
	}
	ctx := &EffContext{IOReader: input}
	for _, want := range []string{"Some()", "Some(final)", "None"} {
		value, err := ioReadLineOpt(ctx, nil)
		if err != nil || value.String() != want {
			t.Fatalf("Windows pipe line IO: %v %v want %s", value, err, want)
		}
	}
}

func TestTerminalWindowsCharacterDeviceLineIO(t *testing.T) {
	// NUL exercises FILE_TYPE_CHAR without requiring an attended console in CI.
	input, err := os.Open("NUL")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if _, err := terminalDeviceKey(int(input.Fd())); err != nil {
		t.Fatalf("character input incorrectly required disk metadata: %v", err)
	}
	value, err := ioReadLineOpt(&EffContext{IOReader: input}, nil)
	if err != nil || value.String() != "None" {
		t.Fatalf("character line IO: %v %v", value, err)
	}
	if _, err := input.Stat(); err != nil {
		t.Fatalf("borrowed character handle was closed: %v", err)
	}
}
