package ui

import (
	"dotenvless/desktop/privacy"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Keep copy/cut out of Fyne's ordinary clipboard path, even when the eye is open.
type secretEntry struct {
	widget.Entry
	guard *privacy.Session
}

func newSecretEntry(guard *privacy.Session) *secretEntry {
	e := &secretEntry{guard: guard}
	e.Password = true
	e.ActionItem = widget.NewLabel("遮蔽")
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapBreak
	e.ExtendBaseWidget(e)
	e.SetMinRowsVisible(3)
	return e
}
func (e *secretEntry) TypedShortcut(shortcut fyne.Shortcut) {
	switch shortcut.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutCut:
		text := e.SelectedText()
		if text == "" {
			return
		}
		if err := e.guard.Copy([]byte(text)); err != nil {
			if e.guard.OnError != nil {
				e.guard.OnError(err)
			}
			return
		}
		if _, cut := shortcut.(*fyne.ShortcutCut); cut {
			e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
		}
	default:
		e.Entry.TypedShortcut(shortcut)
	}
}
func (e *secretEntry) TappedSecondary(*fyne.PointEvent) {}
