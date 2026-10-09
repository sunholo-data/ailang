package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStderrGuard_Boundaries(t *testing.T) {
	for _, term := range []string{"", "\n", "\r", "\v", "\f", "\x1c", "\x1d", "\x1e", "\u0085", "\u2028", "\u2029"} {
		for _, token := range []string{"policy:", "policy-result:"} {
			input := "ordinary" + term + token + " {}"
			if term == "" {
				input = token + " {}"
			}
			want := strings.Replace(input, token, "worker: "+token, 1)
			for split := 0; split <= len(input); split++ {
				var out bytes.Buffer
				g := guardWorkerStderr(&out)
				_, _ = g.Write([]byte(input[:split]))
				_, _ = g.Write([]byte(input[split:]))
				g.Flush()
				if out.String() != want {
					t.Fatalf("term %q split %d: %q want %q", term, split, out.String(), want)
				}
			}
			var out bytes.Buffer
			g := guardWorkerStderr(&out)
			for i := range []byte(input) {
				_, _ = g.Write([]byte{input[i]})
			}
			g.Flush()
			if out.String() != want {
				t.Fatalf("byte chunks: %q", out.String())
			}
		}
	}
}

func TestStderrGuard_PreservationAndFlush(t *testing.T) {
	for _, input := range []string{"", "pol", "policy", "policy-resul", "policyX: {}", "x policy: {}", "\xe2policy: {}", "\xc2policy: {}", strings.Repeat("x", 100000) + "policy:"} {
		var out bytes.Buffer
		g := guardWorkerStderr(&out)
		_, _ = g.Write([]byte(input))
		g.Flush()
		g.Flush()
		if out.String() != input {
			t.Fatalf("got %q want %q", out.String(), input)
		}
	}
}

func TestStderrGuard_SupervisorFreshLine(t *testing.T) {
	for _, input := range []string{"", "tail", "tail\r", "tail\u2028", "pol", "tail\n"} {
		var out bytes.Buffer
		g := guardWorkerStderr(&out)
		_, _ = g.Write([]byte(input))
		g.Flush()
		g.supervisorLine("policy: %s\n", "{}")
		g.supervisorLine("policy-result: %s\n", "{}")
		want := input
		if input != "" && !strings.HasSuffix(input, "\n") {
			want += "\n"
		}
		want += "policy: {}\npolicy-result: {}\n"
		if out.String() != want {
			t.Fatalf("got %q want %q", out.String(), want)
		}
	}
}

func TestStderrGuard_CancellationFlush(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var out bytes.Buffer
	g := guardWorkerStderr(&out)
	var wg sync.WaitGroup
	wg.Add(1)
	read := make(chan struct{})
	go func() {
		defer wg.Done()
		defer g.Flush()
		buf := make([]byte, 3)
		n, _ := r.Read(buf)
		_, _ = g.Write(buf[:n])
		close(read)
		_, _ = io.Copy(g, r)
	}()
	if _, err = w.Write([]byte("pol")); err != nil {
		t.Fatal(err)
	}
	<-read
	// An open writer forces drainOutput's cancellation/closed-pipe path.
	drainOutput(&wg, time.Millisecond, r)
	if out.String() != "pol" {
		t.Fatalf("lost held prefix: %q", out.String())
	}
}
