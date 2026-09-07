package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	msg         string
	out         io.Writer
	interactive bool

	mu      sync.Mutex
	stopped bool
	done    chan struct{}
	started time.Time
}

func NewSpinner(msg string) *Spinner {
	return newSpinner(msg, os.Stderr, isTerminal(os.Stderr))
}

func (pr *Printer) Spinner(msg string) *Spinner {
	return newSpinner(msg, pr.errw(), isTerminal(pr.errw()))
}

func newSpinner(msg string, out io.Writer, interactive bool) *Spinner {
	return &Spinner{msg: msg, out: out, interactive: interactive, done: make(chan struct{})}
}

func (s *Spinner) Start() {
	s.started = time.Now()
	if !s.interactive {
		_, _ = fmt.Fprintf(s.out, "%s...\n", s.msg)
		return
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					return
				}
				elapsed := time.Since(s.started).Round(time.Second)
				_, _ = fmt.Fprintf(s.out, "\r\033[K%s %s… (%s)", spinnerFrames[frame%len(spinnerFrames)], s.msg, elapsed)
				s.mu.Unlock()
				frame++
			}
		}
	}()
}

func (s *Spinner) UpdateMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.msg = msg
	if !s.interactive {
		_, _ = fmt.Fprintf(s.out, "%s...\n", msg)
	}
}

func (s *Spinner) Succeed(msg string) { s.finish("✓", msg) }

func (s *Spinner) Fail(msg string) { s.finish("✗", msg) }

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.done)
	if s.interactive {
		_, _ = fmt.Fprint(s.out, "\r\033[K")
	}
}

func (s *Spinner) finish(mark, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.stopped = true
	close(s.done)
	if s.interactive {
		_, _ = fmt.Fprint(s.out, "\r\033[K")
	}
	_, _ = fmt.Fprintf(s.out, "%s %s\n", mark, msg)
}
