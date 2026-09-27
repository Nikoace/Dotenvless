package privacy

import (
	"testing"
	"time"
)

type fakeClipboard struct {
	seq    uint32
	text   string
	clears int
}

func (c *fakeClipboard) SetSensitive(v []byte) (uint32, error) {
	c.seq++
	c.text = string(v)
	return c.seq, nil
}
func (c *fakeClipboard) ClearOwned(seq uint32) error {
	if c.seq == seq {
		c.text = ""
		c.clears++
	}
	return nil
}
func TestRevealExpiryAndFocusInvalidation(t *testing.T) {
	c := &fakeClipboard{}
	s := New(c, func(f func()) { f() })
	var callbacks []func()
	var durations []time.Duration
	s.after = func(d time.Duration, f func()) func() {
		callbacks = append(callbacks, f)
		durations = append(durations, d)
		return func() {}
	}
	shown := ""
	s.Reveal([]byte("GENERATED_ONE"), func(v string) { shown = v })
	if shown != "GENERATED_ONE" || len(callbacks) != 1 || durations[0] != 15*time.Second {
		t.Fatal("reveal did not schedule expiry")
	}
	s.Hide()
	if shown != "" {
		t.Fatal("focus/navigation hide failed")
	}
	s.Reveal([]byte("GENERATED_TWO"), func(v string) { shown = v })
	callbacks[0]()
	if shown != "GENERATED_TWO" {
		t.Fatal("old timer hid new reveal")
	}
	callbacks[1]()
	if shown != "" {
		t.Fatal("reveal survived expiry")
	}
}
func TestCopyExpiryPreservesNewClipboardAndCloseClearsOwned(t *testing.T) {
	c := &fakeClipboard{}
	s := New(c, func(f func()) { f() })
	var callbacks []func()
	s.after = func(d time.Duration, f func()) func() {
		if d != 30*time.Second {
			t.Fatal("wrong clipboard lifetime")
		}
		callbacks = append(callbacks, f)
		return func() {}
	}
	if e := s.Copy([]byte("GENERATED_CLIP")); e != nil {
		t.Fatal(e)
	}
	if c.text != "GENERATED_CLIP" || len(callbacks) != 1 {
		t.Fatal("copy missing")
	}
	c.SetSensitive([]byte("USER_NEW_CONTENT"))
	callbacks[0]()
	if c.text != "USER_NEW_CONTENT" {
		t.Fatal("user clipboard replaced")
	}
	s.Copy([]byte("GENERATED_OWNED"))
	s.Close()
	if c.text != "" {
		t.Fatal("exit left owned clipboard")
	}
}
