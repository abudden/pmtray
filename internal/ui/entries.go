package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hookEntry is an Entry whose key and shortcut handling can be intercepted.
type hookEntry struct {
	widget.Entry
	// onKey is called before the default key handling; returning true
	// suppresses it.
	onKey func(*fyne.KeyEvent) bool
	// afterShortcut is called after the default shortcut handling.
	afterShortcut func(fyne.Shortcut)
}

func newHookEntry() *hookEntry {
	e := &hookEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func newHookPasswordEntry() *hookEntry {
	e := newHookEntry()
	e.Password = true
	return e
}

func (e *hookEntry) TypedKey(k *fyne.KeyEvent) {
	if e.onKey != nil && e.onKey(k) {
		return
	}
	e.Entry.TypedKey(k)
}

func (e *hookEntry) TypedShortcut(s fyne.Shortcut) {
	e.Entry.TypedShortcut(s)
	if e.afterShortcut != nil {
		e.afterShortcut(s)
	}
}

// selectAll selects all the entry's text, placing the cursor at the end.
func selectAll(e *widget.Entry) {
	e.TypedShortcut(&fyne.ShortcutSelectAll{})
}

// readOnlyEntry is a focusable entry whose text can be selected and copied
// but not edited.
type readOnlyEntry struct {
	widget.Entry
	onKey  func(*fyne.KeyEvent) bool
	onCopy func()
}

func newReadOnlyEntry() *readOnlyEntry {
	e := &readOnlyEntry{}
	e.ExtendBaseWidget(e)
	return e
}

var navigationKeys = map[fyne.KeyName]bool{
	fyne.KeyLeft: true, fyne.KeyRight: true, fyne.KeyUp: true, fyne.KeyDown: true,
	fyne.KeyHome: true, fyne.KeyEnd: true, fyne.KeyPageUp: true, fyne.KeyPageDown: true,
}

func (e *readOnlyEntry) TypedRune(rune) {}

func (e *readOnlyEntry) TypedKey(k *fyne.KeyEvent) {
	if e.onKey != nil && e.onKey(k) {
		return
	}
	if navigationKeys[k.Name] {
		e.Entry.TypedKey(k)
	}
}

func (e *readOnlyEntry) TypedShortcut(s fyne.Shortcut) {
	switch s := s.(type) {
	case *fyne.ShortcutCopy:
		if e.onCopy != nil {
			e.onCopy()
			return
		}
		e.Entry.TypedShortcut(s)
	case *fyne.ShortcutSelectAll:
		e.Entry.TypedShortcut(s)
	case *desktop.CustomShortcut:
		if navigationKeys[s.KeyName] {
			e.Entry.TypedShortcut(s)
		}
	}
}

// TappedSecondary suppresses the context menu, which offers Cut and Paste.
func (e *readOnlyEntry) TappedSecondary(*fyne.PointEvent) {}
