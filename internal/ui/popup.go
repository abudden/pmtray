package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/abudden/pmtray/internal/config"
	"github.com/abudden/pmtray/internal/pwm"
)

const popupWidth = 400

// popup is the window shown when the tray icon is clicked.
type popup struct {
	app *App
	win fyne.Window

	master      *hookEntry
	confirm     *hookEntry
	verify      *widget.Label
	url         *hookEntry
	used        *hookEntry
	profile     *widget.Select
	username    *hookEntry
	generated   *readOnlyEntry
	genTheme    *foregroundTheme
	genOverride *container.ThemeOverride
	showPw      *widget.Button
	strength    *widget.ProgressBar
	copyBtn     *widget.Button

	// State, kept while the application is running.
	urlText      string
	usedText     string
	manualUsed   bool // usedText was typed rather than derived from urlText
	profileIdx   int  // index into profiles
	confirmText  string
	usernameText string
	revealed     bool // generated password shown despite HideGeneratedPassword

	profiles []pwm.Profile
	order    []int // display position -> index into profiles
	updating bool  // set while changing widgets programmatically
	password string
	valid    bool // password holds a generated password

	checkedMaster string // cache for the master password hash check
	masterMatches bool
}

func newPopup(a *App) *popup {
	p := &popup{app: a}
	p.win = a.fyne.NewWindow("PasswordMaker")
	p.win.SetFixedSize(true)
	p.win.SetCloseIntercept(p.hide)

	escape := func(k *fyne.KeyEvent) bool {
		if k.Name == fyne.KeyEscape {
			p.hide()
			return true
		}
		return false
	}

	p.master = newHookPasswordEntry()
	p.master.onKey = escape
	p.master.OnChanged = func(s string) {
		if p.updating {
			return
		}
		p.app.setMaster(s)
		p.regenerate()
	}
	p.master.OnSubmitted = func(string) { p.submit() }

	p.confirm = newHookPasswordEntry()
	p.confirm.onKey = escape
	p.confirm.OnChanged = func(s string) {
		p.confirmText = s
		p.regenerate()
	}
	p.confirm.OnSubmitted = func(string) { p.submit() }

	p.verify = widget.NewLabel("")
	p.verify.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	p.url = newHookEntry()
	p.url.SetPlaceHolder("Paste from address bar")
	p.url.onKey = escape
	p.url.OnChanged = p.onURLChanged
	p.url.OnSubmitted = func(string) { p.submit() }
	p.url.afterShortcut = func(s fyne.Shortcut) {
		if _, ok := s.(*fyne.ShortcutPaste); ok {
			p.win.Canvas().Focus(p.generated)
		}
	}

	p.used = newHookEntry()
	p.used.onKey = escape
	p.used.OnChanged = p.onUsedChanged
	p.used.OnSubmitted = func(string) { p.submit() }

	p.profile = widget.NewSelect(nil, func(string) {
		if p.updating {
			return
		}
		p.onProfileSelected(p.profile.SelectedIndex())
	})

	p.username = newHookEntry()
	p.username.SetPlaceHolder("Set in Profile Settings")
	p.username.onKey = escape
	p.username.OnChanged = func(s string) {
		if p.updating {
			return
		}
		p.usernameText = s
		p.regenerate()
	}
	p.username.OnSubmitted = func(string) { p.submit() }

	p.generated = newReadOnlyEntry()
	p.generated.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	p.generated.onCopy = p.copyAndHide
	p.generated.onKey = func(k *fyne.KeyEvent) bool {
		switch k.Name {
		case fyne.KeyReturn, fyne.KeyEnter:
			p.copyAndHide()
			return true
		}
		return escape(k)
	}
	p.genTheme = &foregroundTheme{}
	p.genOverride = container.NewThemeOverride(p.generated, p.genTheme)

	p.showPw = widget.NewButton("Show Password", func() {
		p.revealed = true
		p.rebuild()
	})
	p.strength = widget.NewProgressBar()
	p.strength.TextFormatter = func() string {
		return fmt.Sprintf("Strength: %.0f", p.strength.Value*100)
	}
	p.copyBtn = widget.NewButton("Copy", p.copyAndHide)

	p.setProfiles(a.store.Profiles())
	// A master password restored from disk needs no confirmation.
	p.confirmText = a.master
	p.confirm.SetText(a.master)
	p.rebuild()
	p.win.CenterOnScreen()
	return p
}

func formLabel(text string) fyne.CanvasObject {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

// rebuild lays out the window according to the current settings.
func (p *popup) rebuild() {
	s := p.app.settings
	p.syncMaster()

	title := widget.NewLabel("PasswordMaker")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.SizeName = theme.SizeNameSubHeadingText
	header := container.NewHBox(widget.NewIcon(appIcon), title)

	// The order of the rows is also the tab order.
	rows := []fyne.CanvasObject{formLabel("Password:"), p.master}
	if p.needConfirmation() {
		rows = append(rows, formLabel("Confirmation:"), p.confirm)
	}
	if s.UseVerificationCode {
		rows = append(rows, formLabel("Verification Code:"), p.verify)
	}
	rows = append(rows,
		formLabel("Full URL:"), p.url,
		formLabel("Used text:"), p.used,
		formLabel("Use Profile:"), p.profile)
	if s.ShowUsername {
		rows = append(rows, formLabel("Username:"), p.username)
	}
	form := container.New(layout.NewFormLayout(), rows...)

	hidden := s.HideGeneratedPassword && !p.revealed

	content := container.NewVBox(header, form, formLabel("Generated Password:"), p.genOverride)
	if hidden {
		content.Add(p.showPw)
	}
	if s.ShowPasswordStrength && !hidden {
		content.Add(p.strength)
	}
	content.Add(container.NewHBox(p.copyBtn, layout.NewSpacer(),
		widget.NewButton("Options", p.app.showOptions)))

	padded := container.NewPadded(content)
	p.win.SetContent(padded)
	p.win.Resize(fyne.NewSize(popupWidth*p.app.scale(), padded.MinSize().Height))
	p.regenerate()
}

func (p *popup) needConfirmation() bool {
	s := p.app.settings
	return !s.UseVerificationCode && s.MasterPasswordHash == nil
}

// setProfiles updates the profile list, keeping the selected profile if it
// still exists.
func (p *popup) setProfiles(profiles []pwm.Profile) {
	var current string
	if p.profileIdx < len(p.profiles) {
		current = p.profiles[p.profileIdx].Title
	}
	p.profiles = profiles
	p.order = config.SortedIndices(profiles, p.app.settings.SortProfiles)

	labels := make([]string, len(p.order))
	seen := map[string]int{}
	for i, idx := range p.order {
		title := profiles[idx].Title
		seen[title]++
		if seen[title] > 1 {
			title = fmt.Sprintf("%s (%d)", title, seen[title])
		}
		labels[i] = title
	}
	p.updating = true
	p.profile.SetOptions(labels)
	p.updating = false

	idx := -1
	for i := range profiles {
		if profiles[i].Title == current {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = p.matchProfile()
	}
	p.selectProfile(idx)
	if !p.manualUsed {
		p.setUsedFromProfile()
	}
}

// profilesChanged is called when the profiles file has been reloaded.
func (p *popup) profilesChanged() {
	p.setProfiles(p.app.store.Profiles())
	p.regenerate()
}

// matchProfile returns the profile whose site list matches the current
// URL (or typed used text), defaulting to the first profile.
func (p *popup) matchProfile() int {
	text := strings.TrimSpace(p.urlText)
	if p.manualUsed {
		text = p.usedText
	}
	if text == "" {
		return 0
	}
	if i := pwm.MatchProfile(p.profiles, text); i >= 0 {
		return i
	}
	return 0
}

func (p *popup) currentProfile() *pwm.Profile {
	if p.profileIdx >= len(p.profiles) {
		p.profileIdx = 0
	}
	return &p.profiles[p.profileIdx]
}

// selectProfile selects profiles[idx] without triggering change handlers.
func (p *popup) selectProfile(idx int) {
	if idx < 0 || idx >= len(p.profiles) {
		idx = 0
	}
	p.profileIdx = idx
	p.updating = true
	defer func() { p.updating = false }()
	for pos, i := range p.order {
		if i == idx {
			p.profile.SetSelectedIndex(pos)
		}
	}
	p.usernameText = p.profiles[idx].Username
	p.username.SetText(p.usernameText)
}

func (p *popup) setText(e *hookEntry, text string) {
	if e.Text == text {
		return
	}
	p.updating = true
	e.SetText(text)
	p.updating = false
}

// setUsedFromProfile derives the used text from the URL using the current
// profile's settings.
func (p *popup) setUsedFromProfile() {
	prof := p.currentProfile()
	if u := strings.TrimSpace(p.urlText); u != "" {
		p.usedText = prof.UsedText(u)
	} else {
		p.usedText = prof.StrUseText
	}
	p.setText(p.used, p.usedText)
}

func (p *popup) onURLChanged(s string) {
	if p.updating {
		return
	}
	p.urlText = s
	if strings.TrimSpace(s) != "" {
		p.manualUsed = false
	}
	if !p.manualUsed {
		p.selectProfile(p.matchProfile())
		p.setUsedFromProfile()
	}
	p.regenerate()
}

func (p *popup) onUsedChanged(s string) {
	if p.updating {
		return
	}
	p.usedText = s
	p.manualUsed = true
	if p.urlText != "" {
		p.urlText = ""
		p.setText(p.url, "")
	}
	p.selectProfile(p.matchProfile())
	p.regenerate()
}

func (p *popup) onProfileSelected(pos int) {
	if pos < 0 || pos >= len(p.order) {
		return
	}
	p.selectProfile(p.order[pos])
	if !p.manualUsed {
		p.setUsedFromProfile()
	}
	p.regenerate()
}

// masterOK reports whether the master password matches the saved hash,
// if there is one.
func (p *popup) masterOK(m string) bool {
	h := p.app.settings.MasterPasswordHash
	if h == nil {
		return true
	}
	if m != p.checkedMaster || p.checkedMaster == "" {
		p.checkedMaster = m
		p.masterMatches = h.Matches(m)
	}
	return p.masterMatches
}

// regenerate updates the generated password and related fields.
func (p *popup) regenerate() {
	s := p.app.settings
	m := p.app.master
	prof := p.currentProfile()

	if s.UseVerificationCode && m != "" {
		p.verify.SetText(pwm.VerificationCode(m))
	} else {
		p.verify.SetText("")
	}

	username := prof.Username
	if s.ShowUsername {
		username = p.usernameText
	}

	p.valid = false
	var text string
	st := statusError
	switch {
	case m == "":
		text, st = "Please Enter Password", statusInfo
	case !p.masterOK(m):
		text = "Master Password Mismatch"
	case p.needConfirmation() && m != p.confirmText:
		text = "Passwords Don't Match"
	default:
		p.password = prof.GeneratePassword(p.usedText, m, username)
		text, st, p.valid = p.password, statusOK, true
	}
	if !p.valid {
		p.password = ""
	}

	p.generated.Password = s.HideGeneratedPassword && !p.revealed && p.valid
	p.generated.SetText(text)
	if p.genTheme.status != st {
		p.genTheme.status = st
		p.genOverride.Refresh()
	}
	if p.valid {
		p.copyBtn.Enable()
		p.strength.SetValue(float64(pwm.PasswordStrength(p.password).Strength) / 100)
	} else {
		p.copyBtn.Disable()
		p.strength.SetValue(0)
	}
}

// syncMaster updates the master password field from the application state.
func (p *popup) syncMaster() {
	p.setText(p.master, p.app.master)
	if p.app.master == "" {
		p.confirmText = ""
		p.confirm.SetText("")
	}
}

func (p *popup) submit() {
	switch {
	case p.valid:
		p.copyAndHide()
	case p.needConfirmation() && p.app.master != "":
		p.win.Canvas().Focus(p.confirm)
	}
}

func (p *popup) copyAndHide() {
	if !p.valid {
		return
	}
	p.app.fyne.Clipboard().SetContent(p.password)
	p.hide()
}

func (p *popup) show() {
	p.syncMaster()
	p.regenerate()
	placeWindow(p.win)
	p.win.Show()
	placeWindow(p.win)
	p.win.RequestFocus()

	c := p.win.Canvas()
	if p.app.master == "" {
		c.Focus(p.master)
	} else {
		c.Focus(p.url)
		selectAll(&p.url.Entry)
	}
}

func (p *popup) hide() {
	p.win.Hide()
	if p.revealed {
		p.revealed = false
		p.rebuild()
	}
	p.app.popupHidden()
}
