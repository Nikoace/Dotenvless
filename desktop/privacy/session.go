package privacy

import (
	"errors"
	"time"
)

type Clipboard interface {
	SetSensitive([]byte) (uint32, error)
	ClearOwned(uint32) error
}

// Session methods and dispatched callbacks run on the UI event thread.
type Session struct {
	clipboard                  Clipboard
	dispatch                   func(func())
	after                      func(time.Duration, func()) func()
	OnError                    func(error)
	display                    func(string)
	stopReveal, stopCopy       func()
	generation, copyGeneration uint64
	sequence                   uint32
	owned, closed              bool
}

func New(c Clipboard, dispatch func(func())) *Session {
	return &Session{clipboard: c, dispatch: dispatch, after: func(d time.Duration, f func()) func() { t := time.AfterFunc(d, f); return func() { t.Stop() } }}
}
func (s *Session) Reveal(value []byte, display func(string)) {
	defer clear(value)
	if s.closed {
		return
	}
	s.Hide()
	s.display = display
	display(string(value))
	generation := s.generation
	s.stopReveal = s.after(15*time.Second, func() {
		s.dispatch(func() {
			if !s.closed && s.generation == generation {
				s.Hide()
			}
		})
	})
}
func (s *Session) Hide() {
	s.generation++
	if s.stopReveal != nil {
		s.stopReveal()
		s.stopReveal = nil
	}
	if s.display != nil {
		s.display("")
		s.display = nil
	}
}
func (s *Session) Copy(value []byte) error {
	defer clear(value)
	if s.closed {
		return errors.New("privacy session is closed")
	}
	if s.clipboard == nil {
		return errors.New("native clipboard is unavailable")
	}
	seq, e := s.clipboard.SetSensitive(value)
	if e != nil {
		return e
	}
	if s.stopCopy != nil {
		s.stopCopy()
	}
	s.sequence = seq
	s.owned = true
	s.copyGeneration++
	generation := s.copyGeneration
	s.stopCopy = s.after(30*time.Second, func() {
		s.dispatch(func() {
			if !s.closed && s.copyGeneration == generation {
				s.clearClipboard()
			}
		})
	})
	return nil
}
func (s *Session) clearClipboard() {
	if !s.owned {
		return
	}
	if e := s.clipboard.ClearOwned(s.sequence); e != nil {
		if s.OnError != nil {
			s.OnError(e)
		}
		return
	}
	s.owned = false
}
func (s *Session) Close() {
	s.Hide()
	s.closed = true
	s.copyGeneration++
	if s.stopCopy != nil {
		s.stopCopy()
		s.stopCopy = nil
	}
	s.clearClipboard()
}
