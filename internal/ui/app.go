// Package ui implements the notification area icon, popup and options
// window.
package ui

import (
	_ "embed"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/systray"

	"github.com/abudden/pmtray/internal/config"
)

//go:embed assets/icon.png
var iconPNG []byte

var appIcon = fyne.NewStaticResource("icon.png", iconPNG)

// App ties together the profiles store, settings and windows.
type App struct {
	fyne     fyne.App
	store    *config.Store
	settings config.Settings
	popup    *popup
	options  *optionsWindow

	master string      // the master password, while remembered
	expiry *time.Timer // forgets the master password (StoreMemoryExpire)
	stop   chan struct{}
}

// New creates the application. a must be a desktop fyne.App.
func New(a fyne.App) *App {
	app := &App{fyne: a, stop: make(chan struct{})}
	a.SetIcon(appIcon)

	var err error
	app.settings, err = config.LoadSettings()
	if err != nil {
		log.Printf("loading settings: %v", err)
	}
	app.store, err = config.OpenStore(app.settings.ProfilesPath())
	if err != nil {
		log.Printf("loading profiles: %v", err)
	}
	// Write the settings file so later runs know this isn't the first.
	if err := app.settings.Save(); err != nil {
		log.Printf("saving settings: %v", err)
	}
	if app.settings.StoreLocation == config.StoreDisk {
		app.master = app.settings.GetStoredPassword()
	}
	a.Settings().SetTheme(zoomTheme{scale: app.scale()})

	app.popup = newPopup(app)

	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(appIcon)
		desk.SetSystemTrayMenu(fyne.NewMenu("PasswordMaker",
			fyne.NewMenuItem("Show PasswordMaker", app.ShowPopup),
			fyne.NewMenuItem("Options…", app.showOptions),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", app.quit),
		))
		systray.SetTooltip("PasswordMaker")
		systray.SetOnTapped(func() { fyne.Do(app.ShowPopup) })
	}

	go app.store.Watch(time.Second, app.stop, func() {
		fyne.Do(app.profilesChanged)
	})
	return app
}

// scale returns the zoom level as a factor.
func (a *App) scale() float32 {
	return float32(a.settings.ZoomLevel) / 100
}

// ShowPopup shows the popup window.
func (a *App) ShowPopup() {
	if a.store.CheckForChanges() {
		a.profilesChanged()
	}
	a.popup.show()
}

func (a *App) showOptions() {
	if a.options == nil {
		a.options = newOptionsWindow(a)
	}
	a.options.show()
}

func (a *App) quit() {
	close(a.stop)
	a.popupHidden()
	a.fyne.Quit()
}

// profilesChanged updates the windows after the profiles have changed.
func (a *App) profilesChanged() {
	a.popup.profilesChanged()
	if a.options != nil {
		a.options.profilesChanged()
	}
}

// settingsChanged saves the settings and updates the windows.
func (a *App) settingsChanged() {
	if err := a.settings.Save(); err != nil {
		log.Printf("saving settings: %v", err)
	}
	a.fyne.Settings().SetTheme(zoomTheme{scale: a.scale()})
	a.popup.setProfiles(a.store.Profiles()) // sort order may have changed
	a.popup.rebuild()
	a.applyStoreLocation()
}

// setMaster records a new master password.
func (a *App) setMaster(pw string) {
	a.master = pw
	a.applyStoreLocation()
}

// applyStoreLocation handles remembering the master password according to
// the "Save Password" setting.
func (a *App) applyStoreLocation() {
	if a.expiry != nil {
		a.expiry.Stop()
		a.expiry = nil
	}
	s := &a.settings
	if s.StoreLocation == config.StoreMemoryExpire && a.master != "" {
		a.expiry = time.AfterFunc(time.Duration(s.ExpireMinutes)*time.Minute, func() {
			fyne.Do(a.forgetMaster)
		})
	}
	if s.StoreLocation != config.StoreDisk && s.StoredPassword != "" {
		s.StoredPassword = ""
		if err := s.Save(); err != nil {
			log.Printf("saving settings: %v", err)
		}
	}
}

func (a *App) forgetMaster() {
	a.master = ""
	a.popup.syncMaster()
	a.popup.regenerate()
}

// popupHidden is called when the popup is closed.
func (a *App) popupHidden() {
	switch a.settings.StoreLocation {
	case config.StoreNever:
		a.forgetMaster()
	case config.StoreDisk:
		if a.settings.GetStoredPassword() != a.master {
			a.settings.SetStoredPassword(a.master)
			if err := a.settings.Save(); err != nil {
				log.Printf("saving settings: %v", err)
			}
		}
	}
}
