package runlife

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Session watches an inherited owner channel before compilation or admission.
// Bind and stop share a barrier: a stop can never admit a later execution owner.
type Session struct {
	request, receipt, lifeline *os.File
	exit                       func(int)
	fallback                   func()
	mu                         sync.Mutex
	stopped, completed         bool
	cleanup                    func(context.Context) error
	failure                    error
	once                       sync.Once
}

var active struct {
	sync.Mutex
	s *Session
}

// Bootstrap consumes and clears private metadata. Endpoints become close-on-exec
// immediately; only Prepare can explicitly transfer fresh authority to a child.
func Bootstrap(meta string, exit func(int), fallback func()) (*Session, error) {
	_ = os.Unsetenv(envKey)
	if meta == "" {
		return nil, nil
	}
	parts := strings.Split(meta, ",")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, fmt.Errorf("invalid lifetime descriptors")
	}
	files := make([]*os.File, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		fd, err := strconv.Atoi(part)
		if err != nil || fd < 3 || seen[fd] {
			for _, f := range files {
				_ = f.Close()
			}
			return nil, fmt.Errorf("invalid lifetime descriptor")
		}
		seen[fd] = true
		f := os.NewFile(uintptr(fd), "runtime-lifetime")
		if f == nil {
			return nil, fmt.Errorf("invalid lifetime pipe")
		}
		closeOnExec(fd)
		st, err := f.Stat()
		if err != nil || st.Mode()&os.ModeNamedPipe == 0 {
			_ = f.Close()
			for _, prev := range files {
				_ = prev.Close()
			}
			return nil, fmt.Errorf("lifetime descriptor is not a pipe")
		}
		files = append(files, f)
	}
	s := &Session{request: files[0], receipt: files[1], exit: exit, fallback: fallback}
	if len(files) == 3 {
		s.lifeline = files[2]
	}
	if err := writeFrame(s.receipt, frame{Kind: "ready"}); err != nil {
		s.closeFiles()
		return nil, err
	}
	active.Lock()
	active.s = s
	active.Unlock()
	go func() {
		f, err := readFrame(s.request)
		if err != nil || f.Kind != "stop" {
			s.stop(ShutdownTimeout, false)
			return
		}
		s.stop(time.Duration(f.RemainingNS), false)
	}()
	if s.lifeline != nil {
		go func() { var b [1]byte; _, _ = s.lifeline.Read(b[:]); s.stop(ShutdownTimeout, true) }()
	}
	return s, nil
}

func (s *Session) Bind(cleanup func(context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped || s.completed {
		return fmt.Errorf("runtime owner is shutting down")
	}
	s.cleanup = cleanup
	return nil
}
func Bind(cleanup func(context.Context) error) error {
	active.Lock()
	s := active.s
	active.Unlock()
	if s == nil {
		return nil
	}
	return s.Bind(cleanup)
}
func Complete(err error) {
	active.Lock()
	s := active.s
	active.Unlock()
	if s != nil {
		s.Complete(err)
	}
}
func RecordFailure(err error) {
	if err == nil {
		return
	}
	active.Lock()
	s := active.s
	active.Unlock()
	if s != nil {
		s.mu.Lock()
		s.failure = errors.Join(s.failure, err)
		s.mu.Unlock()
	}
}

func (s *Session) stop(remaining time.Duration, lostSupervisor bool) {
	s.mu.Lock()
	if s.stopped || s.completed {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	cleanup := s.cleanup
	s.mu.Unlock()
	// A stuck application or cleanup cannot disable the inherited lifeline.
	watchdog := time.AfterFunc(remaining, func() {
		if lostSupervisor && s.fallback != nil {
			s.fallback()
		}
		s.exit(1)
	})
	ctx, cancel := context.WithTimeout(context.Background(), remaining)
	defer cancel()
	var err error
	if cleanup != nil {
		err = cleanup(ctx)
	}
	if lostSupervisor {
		watchdog.Stop()
		if s.fallback != nil {
			s.fallback()
		}
		s.exit(1)
		return
	}
	s.Complete(err)
	watchdog.Stop()
	code := 0
	if err != nil {
		code = 1
	}
	s.exit(code)
}
func (s *Session) Complete(err error) {
	s.once.Do(func() {
		s.mu.Lock()
		s.completed = true
		err = errors.Join(s.failure, err)
		s.mu.Unlock()
		f := frame{Kind: "receipt"}
		if err != nil {
			f.Error = "owned runtime cleanup failed"
		}
		_ = writeFrame(s.receipt, f)
		s.closeFiles()
	})
}
func (s *Session) closeFiles() {
	_ = s.request.Close()
	_ = s.receipt.Close()
	if s.lifeline != nil {
		_ = s.lifeline.Close()
	}
}

// Admit serializes a policy admission decision against inherited stop. No
// admission may be published after cancellation has closed the runtime gate.
func (s *Session) Admit(action func()) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped || s.completed {
		return false
	}
	action()
	return true
}
func Admit(action func()) bool {
	active.Lock()
	s := active.s
	active.Unlock()
	if s == nil {
		action()
		return true
	}
	return s.Admit(action)
}
