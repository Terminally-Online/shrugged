package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var frames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner renders a single-line animated status on stderr that overwrites
// itself in place. It is safe for concurrent use.
type Spinner struct {
	mu      sync.Mutex
	w       io.Writer
	msg     string
	frame   int
	stop    chan struct{}
	stopped chan struct{}
	active  bool
	isTTY   bool
}

// NewSpinner creates a spinner that writes to stderr.
func NewSpinner() *Spinner {
	fi, _ := os.Stderr.Stat()
	isTTY := fi != nil && (fi.Mode()&os.ModeCharDevice) != 0

	return &Spinner{
		w:     os.Stderr,
		isTTY: isTTY,
	}
}

// Start begins the spinner with the given message.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		s.msg = msg
		return
	}

	s.msg = msg
	s.active = true
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	s.frame = 0

	if !s.isTTY {
		return
	}

	// Hide cursor
	fmt.Fprint(s.w, "\033[?25l")
	s.render()

	go func() {
		defer close(s.stopped)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.frame = (s.frame + 1) % len(frames)
				s.render()
				s.mu.Unlock()
			}
		}
	}()
}

// Update changes the spinner message while it's running.
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msg = msg
	if s.active && s.isTTY {
		s.render()
	}
}

// Stop clears the spinner line and restores the cursor. After Stop the
// terminal line is completely clean — no residual text.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false

	if s.isTTY {
		close(s.stop)
	}
	s.mu.Unlock()

	if s.isTTY {
		<-s.stopped
		// Clear line and restore cursor
		fmt.Fprint(s.w, "\r\033[K\033[?25h")
	}
}

// render writes the current frame + message. Caller must hold s.mu.
func (s *Spinner) render() {
	fmt.Fprintf(s.w, "\r\033[K%s %s", frames[s.frame], s.msg)
}
