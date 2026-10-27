package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/abudden/pmtray/internal/config"
	"github.com/abudden/pmtray/internal/pwm"
	"github.com/abudden/pmtray/internal/rdf"
)

// newTestApp starts the application with a fresh configuration directory
// whose profiles file holds profiles.
func newTestApp(t *testing.T, profiles []pwm.Profile) (*App, fyne.App) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	if profiles != nil {
		path := config.DefaultProfilesFile()
		if err := os.WriteFile(path, rdf.Marshal(profiles), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a := test.NewTempApp(t)
	app := New(a)
	t.Cleanup(func() { close(app.stop) })
	return app, a
}

func testProfiles() []pwm.Profile {
	p := pwm.DefaultProfiles()
	bank := pwm.NewProfile()
	bank.Title = "Bank"
	bank.SiteList = "*bank* /^https://secure\\./"
	bank.SelectedCharset = "0123456789"
	bank.URLSubdomain = true
	return append(p, bank)
}

func focused(p *popup) fyne.Focusable {
	return p.win.Canvas().Focused()
}

func TestFocusOnShow(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup

	app.ShowPopup()
	if focused(p) != p.master {
		t.Fatalf("expected master password focus, got %T", focused(p))
	}
	test.Type(p.master, "secret")
	if !p.valid || p.generated.Text != pwm.NewProfile().GeneratePassword("", "secret", "") {
		t.Errorf("generated %q valid=%v", p.generated.Text, p.valid)
	}
	if p.verify.Text != pwm.VerificationCode("secret") {
		t.Errorf("verification code %q", p.verify.Text)
	}

	p.url.SetText("https://example.com/")
	p.hide()
	app.ShowPopup()
	if focused(p) != p.url {
		t.Fatalf("expected URL focus, got %T", focused(p))
	}
	if p.url.SelectedText() != "https://example.com/" {
		t.Errorf("URL text not selected: %q", p.url.SelectedText())
	}
}

func TestTabOrder(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup
	app.ShowPopup()
	c := p.win.Canvas()
	want := []fyne.Focusable{p.url, p.used, p.profile, p.generated}
	for i, w := range want {
		c.FocusNext()
		if c.Focused() != w {
			t.Fatalf("tab %d: got %T, want %T", i+1, c.Focused(), w)
		}
	}
}

func TestPasteURL(t *testing.T) {
	app, a := newTestApp(t, testProfiles())
	p := app.popup
	app.ShowPopup()
	test.Type(p.master, "secret")
	p.win.Canvas().Focus(p.url)

	a.Clipboard().SetContent("https://www.mybank.co.uk/login?x=1")
	p.url.TypedShortcut(&fyne.ShortcutPaste{Clipboard: a.Clipboard()})

	if p.url.Text != "https://www.mybank.co.uk/login?x=1" {
		t.Errorf("URL %q", p.url.Text)
	}
	if focused(p) != p.generated {
		t.Errorf("focus after paste: %T", focused(p))
	}
	if p.profile.Selected != "Bank" {
		t.Errorf("profile %q", p.profile.Selected)
	}
	if p.used.Text != "www.mybank.co.uk" {
		t.Errorf("used text %q", p.used.Text)
	}
	bank := testProfiles()[2]
	want := bank.GeneratePassword("www.mybank.co.uk", "secret", "")
	if p.generated.Text != want {
		t.Errorf("generated %q want %q", p.generated.Text, want)
	}

	// Ctrl-C in the generated field copies the password and hides the popup.
	a.Clipboard().SetContent("")
	p.generated.TypedShortcut(&fyne.ShortcutCopy{Clipboard: a.Clipboard()})
	if got := a.Clipboard().Content(); got != want {
		t.Errorf("clipboard %q", got)
	}

	// Choosing another profile recalculates the used text from the URL.
	p.profile.SetSelected("Default")
	if p.used.Text != "mybank.co.uk" || p.url.Text == "" {
		t.Errorf("after profile change: used %q url %q", p.used.Text, p.url.Text)
	}

	// The URL and used text are remembered when the popup is reopened.
	p.hide()
	app.ShowPopup()
	if p.url.Text != "https://www.mybank.co.uk/login?x=1" || p.used.Text != "mybank.co.uk" ||
		p.profile.Selected != "Default" {
		t.Errorf("state not kept: %q %q %q", p.url.Text, p.used.Text, p.profile.Selected)
	}
}

func TestTypedUsedText(t *testing.T) {
	app, _ := newTestApp(t, testProfiles())
	p := app.popup
	app.ShowPopup()
	p.url.SetText("https://example.com/")
	if p.profile.Selected != "Default" || p.used.Text != "example.com" {
		t.Fatalf("profile %q used %q", p.profile.Selected, p.used.Text)
	}

	p.win.Canvas().Focus(p.used)
	selectAll(&p.used.Entry)
	test.Type(p.used, "bank")
	if p.url.Text != "" {
		t.Errorf("URL not cleared: %q", p.url.Text)
	}
	if p.profile.Selected != "Bank" {
		t.Errorf("profile %q", p.profile.Selected)
	}
	test.Type(p.used, "x")
	if p.profile.Selected != "Bank" || p.used.Text != "bankx" {
		t.Errorf("profile %q used %q", p.profile.Selected, p.used.Text)
	}
	// A typed used text is kept when changing profile.
	p.profile.SetSelected("Alphanumeric")
	if p.used.Text != "bankx" {
		t.Errorf("used text replaced: %q", p.used.Text)
	}
	selectAll(&p.used.Entry)
	test.Type(p.used, "other")
	if p.profile.Selected != "Default" {
		t.Errorf("profile %q", p.profile.Selected)
	}
}

func TestProfilesFileReload(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup
	path := app.store.Path()

	profiles := testProfiles()
	profiles[0].Title = "Renamed"
	if err := os.WriteFile(path, rdf.Marshal(profiles), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(5 * time.Second)
	os.Chtimes(path, future, future)

	app.ShowPopup() // also checks for changes
	if len(p.profile.Options) != 3 || p.profile.Options[0] != "Renamed" {
		t.Errorf("options %q", p.profile.Options)
	}
	p.url.SetText("https://secure.example.com/")
	if p.profile.Selected != "Bank" {
		t.Errorf("profile %q", p.profile.Selected)
	}
}

func TestOptionsSaveProfile(t *testing.T) {
	app, _ := newTestApp(t, nil)
	app.showOptions()
	o := app.options
	o.selectProfile(1)
	o.name.SetText("Changed")
	o.sites.SetText("*.example.org")
	o.saveProfile()

	data, err := os.ReadFile(app.store.Path())
	if err != nil {
		t.Fatal(err)
	}
	got, err := rdf.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got[1].Title != "Changed" || got[1].SiteList != "*.example.org" || got[1].Timestamp == 0 {
		t.Errorf("saved %+v", got[1])
	}
	if app.popup.profile.Options[1] != "Changed" {
		t.Errorf("popup not updated: %q", app.popup.profile.Options)
	}

	// Switching to a new file creates it with the current profiles.
	newPath := filepath.Join(t.TempDir(), "new.rdf")
	o.switchFile(newPath)
	if app.store.Path() != newPath || app.settings.ProfilesFile != newPath {
		t.Errorf("not switched: %s", app.store.Path())
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Error(err)
	}
}

func TestShowUsernameSetting(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup
	app.settings.ShowUsername = true
	app.settingsChanged()
	app.ShowPopup()
	test.Type(p.master, "secret")
	c := p.win.Canvas()
	c.Focus(p.profile)
	c.FocusNext()
	if c.Focused() != p.username {
		t.Fatalf("username not after profile: %T", c.Focused())
	}
	test.Type(p.username, "bob")
	want := pwm.NewProfile().GeneratePassword("", "secret", "bob")
	if p.generated.Text != want {
		t.Errorf("got %q want %q", p.generated.Text, want)
	}
}

func TestConfirmationAndNeverStore(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup
	app.settings.UseVerificationCode = false
	app.settings.StoreLocation = config.StoreNever
	app.settingsChanged()
	app.ShowPopup()

	test.Type(p.master, "secret")
	if p.valid || p.generated.Text != "Passwords Don't Match" {
		t.Errorf("generated %q", p.generated.Text)
	}
	c := p.win.Canvas()
	c.FocusNext()
	if c.Focused() != p.confirm {
		t.Fatalf("confirmation not after password: %T", c.Focused())
	}
	test.Type(p.confirm, "secret")
	if !p.valid {
		t.Errorf("generated %q", p.generated.Text)
	}

	p.hide()
	if app.master != "" || p.master.Text != "" || p.confirm.Text != "" {
		t.Error("master password kept with Never Store")
	}
	app.ShowPopup()
	if focused(p) != p.master {
		t.Errorf("focus %T", focused(p))
	}
}

func TestMasterPasswordHash(t *testing.T) {
	app, _ := newTestApp(t, nil)
	p := app.popup
	h, err := config.NewMasterHash("right")
	if err != nil {
		t.Fatal(err)
	}
	app.settings.MasterPasswordHash = h
	app.settingsChanged()
	app.ShowPopup()
	test.Type(p.master, "wrong")
	if p.generated.Text != "Master Password Mismatch" {
		t.Errorf("generated %q", p.generated.Text)
	}
	p.master.SetText("right")
	if !p.valid {
		t.Errorf("generated %q", p.generated.Text)
	}
}
