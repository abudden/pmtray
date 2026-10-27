package ui

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/abudden/pmtray/internal/config"
	"github.com/abudden/pmtray/internal/pwm"
)

const customCharset = "Custom"

// optionsWindow edits the profiles and global settings.
type optionsWindow struct {
	app *App
	win fyne.Window

	profiles []pwm.Profile
	selected int
	dirty    bool // the profile form has unsaved changes
	loading  bool // set while filling in widgets programmatically

	list        *widget.List
	status      *widget.Label
	name        *widget.Entry
	modifiedRow *fyne.Container
	modified    *widget.Label
	sites       *widget.Entry
	protocol    *widget.Check
	subdomain   *widget.Check
	domain      *widget.Check
	path        *widget.Check
	useText     *widget.Entry
	leetWhere   *widget.Select
	leetLevel   *widget.Select
	hash        *widget.Select
	length      *widget.Entry
	username    *widget.Entry
	modifier    *widget.Entry
	charset     *widget.Select
	custom      *widget.Entry
	prefix      *widget.Entry
	suffix      *widget.Entry
	description *widget.Entry

	fileLabel *widget.Label
	general   *fyne.Container
}

func newOptionsWindow(a *App) *optionsWindow {
	o := &optionsWindow{app: a}
	o.win = a.fyne.NewWindow("PasswordMaker Options")
	o.win.SetCloseIntercept(o.win.Hide)
	o.profiles = a.store.Profiles()

	tabs := container.NewAppTabs(
		container.NewTabItem("Profiles", o.buildProfilesTab()),
		container.NewTabItem("General Options", o.buildGeneralTab()),
	)
	o.win.SetContent(tabs)
	o.win.Resize(fyne.NewSize(700, 560))
	o.win.CenterOnScreen()
	o.selectProfile(0)
	return o
}

func (o *optionsWindow) show() {
	o.refreshGeneral()
	o.updateStatus()
	o.win.Show()
	fixScale(o.win, nil)
	o.win.RequestFocus()
}

func (o *optionsWindow) changed() {
	if !o.loading {
		o.dirty = true
	}
}

func (o *optionsWindow) entry(multiLine bool) *widget.Entry {
	e := widget.NewEntry()
	if multiLine {
		e = widget.NewMultiLineEntry()
		e.SetMinRowsVisible(3)
		e.Wrapping = fyne.TextWrapWord
	}
	e.OnChanged = func(string) { o.changed() }
	return e
}

func (o *optionsWindow) check(label string) *widget.Check {
	return widget.NewCheck(label, func(bool) { o.changed() })
}

func (o *optionsWindow) buildProfilesTab() fyne.CanvasObject {
	o.list = widget.NewList(
		func() int { return len(o.profiles) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(o.profiles) {
				obj.(*widget.Label).SetText(o.profiles[id].Title)
			}
		})
	o.list.OnSelected = func(id widget.ListItemID) {
		if o.loading || id == o.selected {
			return
		}
		if !o.dirty {
			o.selectProfile(id)
			return
		}
		previous := o.selected
		dialog.ShowConfirm("Unsaved changes",
			"Discard the changes to \""+o.profiles[previous].Title+"\"?",
			func(discard bool) {
				if discard {
					o.dirty = false
					o.selectProfile(id)
				} else {
					o.loading = true
					o.list.Select(previous)
					o.loading = false
				}
			}, o.win)
	}

	listButtons := container.NewGridWithColumns(3,
		widget.NewButton("Add", o.addProfile),
		widget.NewButton("Up", func() { o.moveProfile(-1) }),
		widget.NewButton("Down", func() { o.moveProfile(1) }),
	)
	left := container.NewBorder(nil, listButtons, nil, nil, o.list)

	o.name = o.entry(false)
	o.modified = widget.NewLabel("")
	o.sites = o.entry(true)
	o.sites.SetPlaceHolder("e.g. *.example.com /^https://login\\./")
	o.protocol = o.check("Protocol")
	o.subdomain = o.check("Subdomain(s)")
	o.domain = o.check("Domain")
	o.path = o.check("Port, path, query parameters")
	o.useText = o.entry(false)

	var leetNames []string
	for _, l := range pwm.L33tOptions {
		leetNames = append(leetNames, l.Name)
	}
	o.leetWhere = widget.NewSelect(leetNames, func(string) {
		o.changed()
		o.updateLeetLevel()
	})
	o.leetLevel = widget.NewSelect([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
		func(string) { o.changed() })

	var hashNames []string
	for _, h := range pwm.HashAlgorithms {
		hashNames = append(hashNames, h.Name)
	}
	o.hash = widget.NewSelect(hashNames, func(string) { o.changed() })
	o.length = o.entry(false)
	o.username = o.entry(false)
	o.modifier = o.entry(false)
	o.charset = widget.NewSelect(append(append([]string(nil), pwm.CharsetOptions...), customCharset),
		func(s string) {
			o.changed()
			o.updateCustomCharset()
		})
	o.custom = o.entry(false)
	o.prefix = o.entry(false)
	o.suffix = o.entry(false)
	o.description = o.entry(true)

	o.modifiedRow = container.New(layout.NewFormLayout(), formLabel("Profile Modified On:"), o.modified)

	form := container.New(layout.NewFormLayout(),
		formLabel("Name:"), o.name,
		formLabel("Use With Sites:"), o.sites,
		formLabel("URL Components:"), container.NewGridWithColumns(2, o.protocol, o.subdomain, o.domain, o.path),
		formLabel("Use This Text:"), o.useText,
		formLabel("Use l33t:"), o.leetWhere,
		formLabel("l33t Level:"), o.leetLevel,
		formLabel("Hash Algorithm:"), o.hash,
		formLabel("Password Length:"), o.length,
		formLabel("Username:"), o.username,
		formLabel("Modifier:"), o.modifier,
		formLabel("Characters:"), o.charset,
		widget.NewLabel(""), o.custom,
		formLabel("Password Prefix:"), o.prefix,
		formLabel("Password Suffix:"), o.suffix,
		formLabel("Notes/Description:"), o.description,
	)
	help := widget.NewLabel("Site patterns are separated by spaces. Use * and ? as wildcards, " +
		"or enclose a regular expression in slashes. The first profile is used when no pattern matches.")
	help.Wrapping = fyne.TextWrapWord
	help.Importance = widget.LowImportance

	save := widget.NewButton("Save", o.saveProfile)
	save.Importance = widget.HighImportance
	remove := widget.NewButton("Remove", o.removeProfile)
	remove.Importance = widget.DangerImportance
	buttons := container.NewHBox(save, remove)

	right := container.NewBorder(nil, buttons, nil, nil,
		container.NewVScroll(container.NewVBox(o.modifiedRow, form, help)))

	o.status = widget.NewLabel("")
	o.status.Wrapping = fyne.TextWrapWord
	split := container.NewHSplit(left, container.NewPadded(right))
	split.Offset = 0.25
	return container.NewBorder(nil, o.status, nil, nil, split)
}

func (o *optionsWindow) updateLeetLevel() {
	if o.leetWhere.SelectedIndex() <= 0 {
		o.leetLevel.Disable()
	} else {
		o.leetLevel.Enable()
	}
}

func (o *optionsWindow) updateCustomCharset() {
	if o.charset.Selected == customCharset {
		o.custom.Enable()
	} else {
		o.custom.Disable()
		o.custom.SetText(o.charset.Selected)
	}
}

// selectProfile shows profiles[id] in the form.
func (o *optionsWindow) selectProfile(id int) {
	if id < 0 || id >= len(o.profiles) {
		id = 0
	}
	o.selected = id
	o.loading = true
	defer func() { o.loading = false }()
	o.list.Select(id)

	p := o.profiles[id]
	o.name.SetText(p.Title)
	if p.Timestamp > 0 {
		o.modified.SetText(time.UnixMilli(p.Timestamp).Format("2006-01-02 15:04:05"))
	} else {
		o.modified.SetText("unknown")
	}
	if o.app.settings.ShowModifiedTime {
		o.modifiedRow.Show()
	} else {
		o.modifiedRow.Hide()
	}
	o.sites.SetText(strings.Join(p.SitePatterns(), "\n"))
	o.protocol.SetChecked(p.URLProtocol)
	o.subdomain.SetChecked(p.URLSubdomain)
	o.domain.SetChecked(p.URLDomain)
	o.path.SetChecked(p.URLPath)
	o.useText.SetText(p.StrUseText)
	o.leetWhere.SetSelectedIndex(0)
	for i, l := range pwm.L33tOptions {
		if l.ID == p.WhereToUseL33t {
			o.leetWhere.SetSelectedIndex(i)
		}
	}
	if p.L33tLevel >= 1 && p.L33tLevel <= 9 {
		o.leetLevel.SetSelectedIndex(p.L33tLevel - 1)
	} else {
		o.leetLevel.ClearSelected()
	}
	o.updateLeetLevel()
	o.hash.ClearSelected()
	for i, h := range pwm.HashAlgorithms {
		if h.ID == p.HashAlgorithm {
			o.hash.SetSelectedIndex(i)
		}
	}
	o.length.SetText(strconv.Itoa(p.PasswordLength))
	o.username.SetText(p.Username)
	o.modifier.SetText(p.Modifier)
	o.charset.SetSelected(customCharset)
	o.custom.SetText(p.SelectedCharset)
	for _, c := range pwm.CharsetOptions {
		if c == p.SelectedCharset {
			o.charset.SetSelected(c)
		}
	}
	o.updateCustomCharset()
	o.prefix.SetText(p.PasswordPrefix)
	o.suffix.SetText(p.PasswordSuffix)
	o.description.SetText(p.Description)
	o.dirty = false
	o.updateStatus()
}

// formProfile builds a profile from the form.
func (o *optionsWindow) formProfile() (pwm.Profile, error) {
	p := pwm.NewProfile()
	p.Title = strings.TrimSpace(o.name.Text)
	if p.Title == "" {
		return p, errors.New("please enter a name for the profile")
	}
	p.SiteList = strings.Join(strings.Fields(o.sites.Text), " ")
	p.URLProtocol = o.protocol.Checked
	p.URLSubdomain = o.subdomain.Checked
	p.URLDomain = o.domain.Checked
	p.URLPath = o.path.Checked
	p.StrUseText = o.useText.Text
	if i := o.leetWhere.SelectedIndex(); i >= 0 {
		p.WhereToUseL33t = pwm.L33tOptions[i].ID
	}
	p.L33tLevel = o.leetLevel.SelectedIndex() + 1
	if p.WhereToUseL33t != "off" && p.L33tLevel == 0 {
		return p, errors.New("please choose a l33t level")
	}
	i := o.hash.SelectedIndex()
	if i < 0 {
		return p, errors.New("please choose a hash algorithm")
	}
	p.HashAlgorithm = pwm.HashAlgorithms[i].ID
	n, err := strconv.Atoi(strings.TrimSpace(o.length.Text))
	if err != nil || n < 1 || n > 512 {
		return p, errors.New("the password length must be a number from 1 to 512")
	}
	p.PasswordLength = n
	p.Username = o.username.Text
	p.Modifier = o.modifier.Text
	p.SelectedCharset = o.custom.Text
	if len([]rune(p.SelectedCharset)) < 2 {
		return p, errors.New("the character set must contain at least two characters")
	}
	p.PasswordPrefix = o.prefix.Text
	p.PasswordSuffix = o.suffix.Text
	p.Description = o.description.Text
	p.Timestamp = time.Now().UnixMilli()
	return p, nil
}

// saveProfiles writes the profiles to the file and updates the windows.
func (o *optionsWindow) saveProfiles(profiles []pwm.Profile, selected int) bool {
	if err := o.app.store.Save(profiles); err != nil {
		dialog.ShowError(fmt.Errorf("saving %s: %w", o.app.store.Path(), err), o.win)
		return false
	}
	o.dirty = false
	o.selected = selected
	o.app.profilesChanged()
	return true
}

func (o *optionsWindow) saveProfile() {
	p, err := o.formProfile()
	if err != nil {
		dialog.ShowError(err, o.win)
		return
	}
	profiles := o.app.store.Profiles()
	if o.selected < len(profiles) {
		profiles[o.selected] = p
	} else {
		profiles = append(profiles, p)
	}
	o.saveProfiles(profiles, o.selected)
}

func (o *optionsWindow) addProfile() {
	add := func() {
		p := pwm.NewProfile()
		p.Title = "New Profile"
		p.Timestamp = time.Now().UnixMilli()
		profiles := append(o.app.store.Profiles(), p)
		o.saveProfiles(profiles, len(profiles)-1)
	}
	if o.dirty {
		dialog.ShowConfirm("Unsaved changes", "Discard the changes to the current profile?",
			func(ok bool) {
				if ok {
					add()
				}
			}, o.win)
		return
	}
	add()
}

func (o *optionsWindow) removeProfile() {
	profiles := o.app.store.Profiles()
	if len(profiles) <= 1 {
		dialog.ShowInformation("Remove profile", "The last profile can't be removed.", o.win)
		return
	}
	if o.selected >= len(profiles) {
		return
	}
	dialog.ShowConfirm("Remove profile", "Remove the profile \""+profiles[o.selected].Title+"\"?",
		func(ok bool) {
			if !ok {
				return
			}
			profiles = append(profiles[:o.selected], profiles[o.selected+1:]...)
			sel := o.selected
			if sel >= len(profiles) {
				sel = len(profiles) - 1
			}
			o.saveProfiles(profiles, sel)
		}, o.win)
}

func (o *optionsWindow) moveProfile(delta int) {
	if o.dirty {
		dialog.ShowInformation("Unsaved changes", "Please save or discard the changes to the profile first.", o.win)
		return
	}
	profiles := o.app.store.Profiles()
	from, to := o.selected, o.selected+delta
	if from >= len(profiles) || to < 0 || to >= len(profiles) {
		return
	}
	profiles[from], profiles[to] = profiles[to], profiles[from]
	o.saveProfiles(profiles, to)
}

// profilesChanged is called when the profiles have been saved or reloaded.
func (o *optionsWindow) profilesChanged() {
	o.profiles = o.app.store.Profiles()
	o.list.Refresh()
	if o.dirty {
		o.status.SetText("The profiles file has changed on disk. Saving will replace profile " +
			strconv.Itoa(o.selected+1) + " in the new version with this one.")
		return
	}
	o.selectProfile(o.selected)
}

func (o *optionsWindow) updateStatus() {
	msg := "Profiles file: " + o.app.store.Path()
	if err := o.app.store.Err(); err != nil {
		msg += "\nError reading the file (using the previous profiles): " + err.Error()
	}
	o.status.SetText(msg)
	if o.fileLabel != nil {
		o.fileLabel.SetText(o.app.store.Path())
	}
}

func (o *optionsWindow) buildGeneralTab() fyne.CanvasObject {
	o.general = container.NewVBox()
	o.refreshGeneral()
	return container.NewVScroll(container.NewPadded(o.general))
}

// refreshGeneral rebuilds the general options from the current settings.
func (o *optionsWindow) refreshGeneral() {
	if o.general == nil {
		return
	}
	a := o.app
	s := &a.settings
	update := func() { a.settingsChanged() }

	o.fileLabel = widget.NewLabel(a.store.Path())
	o.fileLabel.Wrapping = fyne.TextWrapBreak
	fileButtons := container.NewHBox(
		widget.NewButton("Use Existing File…", o.chooseExistingFile),
		widget.NewButton("Create New File…", o.chooseNewFile),
		widget.NewButton("Use Default Location", func() { o.switchFile(config.DefaultProfilesFile()) }),
	)

	var zooms []string
	for z := 100; z <= 210; z += 10 {
		zooms = append(zooms, strconv.Itoa(z))
	}
	zoom := widget.NewSelect(zooms, func(v string) {
		n, _ := strconv.Atoi(v)
		if n != s.ZoomLevel {
			s.ZoomLevel = n
			update()
		}
	})
	zoom.SetSelected(strconv.Itoa(s.ZoomLevel))

	expire := widget.NewEntry()
	expire.SetText(strconv.Itoa(s.ExpireMinutes))
	expire.OnChanged = func(v string) {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 720 && n != s.ExpireMinutes {
			s.ExpireMinutes = n
			update()
		}
	}
	expireRow := container.NewHBox(widget.NewLabel("Expire after"), expire, widget.NewLabel("minutes"))
	var storeNames []string
	for _, l := range config.StoreLocations {
		storeNames = append(storeNames, l.Name)
	}
	store := widget.NewSelect(storeNames, nil)
	for i, l := range config.StoreLocations {
		if l.ID == s.StoreLocation {
			store.SetSelectedIndex(i)
		}
	}
	setExpireVisible := func() {
		if s.StoreLocation == config.StoreMemoryExpire {
			expireRow.Show()
		} else {
			expireRow.Hide()
		}
	}
	setExpireVisible()
	store.OnChanged = func(string) {
		id := config.StoreLocations[store.SelectedIndex()].ID
		if id != s.StoreLocation {
			s.StoreLocation = id
			setExpireVisible()
			update()
		}
	}

	var sortNames []string
	for _, l := range config.SortOrders {
		sortNames = append(sortNames, l.Name)
	}
	sortSel := widget.NewSelect(sortNames, nil)
	for i, l := range config.SortOrders {
		if l.ID == s.SortProfiles {
			sortSel.SetSelectedIndex(i)
		}
	}
	sortSel.OnChanged = func(string) {
		s.SortProfiles = config.SortOrders[sortSel.SelectedIndex()].ID
		update()
	}

	boolCheck := func(label string, v *bool, after func()) *widget.Check {
		c := widget.NewCheck(label, nil)
		c.SetChecked(*v)
		c.OnChanged = func(b bool) {
			*v = b
			if after != nil {
				after()
			}
			update()
		}
		return c
	}

	keepHash := widget.NewCheck("Keep Master Password Hash", nil)
	keepHash.SetChecked(s.MasterPasswordHash != nil)
	keepHash.OnChanged = func(b bool) {
		if !b {
			s.MasterPasswordHash = nil
			update()
			return
		}
		o.askMasterHash(func(ok bool) {
			if !ok {
				keepHash.SetChecked(false)
			}
		})
	}

	items := []fyne.CanvasObject{
		formLabel("Profiles file"),
		o.fileLabel,
		fileButtons,
		widget.NewSeparator(),
		formLabel("General options"),
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Set Popup Zoom Level (%):"), zoom,
			widget.NewLabel("Save Password:"), container.NewVBox(store, expireRow),
			widget.NewLabel("Sort Profiles By:"), sortSel,
		),
		boolCheck("Visually Hide Generated Password in Popup", &s.HideGeneratedPassword, nil),
		boolCheck("Manually Verify Password Using 3 Letter Code", &s.UseVerificationCode, nil),
		boolCheck("Show Password Strength Meter in Popup", &s.ShowPasswordStrength, nil),
		boolCheck("Show the Most Recent Profile Modification Time", &s.ShowModifiedTime, func() {
			if s.ShowModifiedTime {
				o.modifiedRow.Show()
			} else {
				o.modifiedRow.Hide()
			}
		}),
		keepHash,
		boolCheck("Show Username Field in Popup", &s.ShowUsername, nil),
	}
	if config.AutostartSupported() {
		auto := widget.NewCheck("Start PasswordMaker When I Log In", nil)
		auto.SetChecked(config.Autostart())
		auto.OnChanged = func(b bool) {
			if err := config.SetAutostart(b); err != nil {
				dialog.ShowError(err, o.win)
			}
		}
		items = append(items, auto)
	}
	o.general.Objects = items
	o.general.Refresh()
}

func (o *optionsWindow) askMasterHash(done func(bool)) {
	pw := widget.NewPasswordEntry()
	confirm := widget.NewPasswordEntry()
	pw.SetText(o.app.master)
	d := dialog.NewForm("Keep Master Password Hash", "OK", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Master password", pw),
		widget.NewFormItem("Confirm", confirm),
	}, func(ok bool) {
		if !ok {
			done(false)
			return
		}
		if pw.Text == "" || pw.Text != confirm.Text {
			dialog.ShowError(errors.New("the passwords are empty or don't match"), o.win)
			done(false)
			return
		}
		h, err := config.NewMasterHash(pw.Text)
		if err != nil {
			dialog.ShowError(err, o.win)
			done(false)
			return
		}
		o.app.settings.MasterPasswordHash = h
		o.app.settingsChanged()
		done(true)
	}, o.win)
	d.Resize(fyne.NewSize(420, 200))
	d.Show()
}

func rdfFilter() storage.FileFilter {
	return storage.NewExtensionFileFilter([]string{".rdf"})
}

func (o *optionsWindow) startDir() fyne.ListableURI {
	dir := storage.NewFileURI(o.app.store.Path())
	parent, err := storage.Parent(dir)
	if err != nil {
		return nil
	}
	l, err := storage.ListerForURI(parent)
	if err != nil {
		return nil
	}
	return l
}

func (o *optionsWindow) chooseExistingFile() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		r.Close()
		o.switchFile(r.URI().Path())
	}, o.win)
	d.SetFilter(rdfFilter())
	if l := o.startDir(); l != nil {
		d.SetLocation(l)
	}
	d.Resize(o.win.Canvas().Size())
	d.Show()
}

func (o *optionsWindow) chooseNewFile() {
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		w.Close()
		path := w.URI().Path()
		// The dialog creates an empty file; replace it with the profiles.
		os.Remove(path)
		o.switchFile(path)
	}, o.win)
	d.SetFileName("profiles.rdf")
	d.SetFilter(rdfFilter())
	if l := o.startDir(); l != nil {
		d.SetLocation(l)
	}
	d.Resize(o.win.Canvas().Size())
	d.Show()
}

// switchFile makes path the profiles file: an existing file is loaded,
// otherwise it is created with the current profiles.
func (o *optionsWindow) switchFile(path string) {
	a := o.app
	if err := a.store.SwitchTo(path); err != nil {
		dialog.ShowError(err, o.win)
		return
	}
	if path == config.DefaultProfilesFile() {
		a.settings.ProfilesFile = ""
	} else {
		a.settings.ProfilesFile = path
	}
	if err := a.settings.Save(); err != nil {
		dialog.ShowError(err, o.win)
	}
	o.dirty = false
	a.profilesChanged()
	o.updateStatus()
}
