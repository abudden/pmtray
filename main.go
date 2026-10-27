// Command pmtray is a PasswordMaker password generator that lives in the
// notification area (system tray).
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/abudden/pmtray/internal/config"
	"github.com/abudden/pmtray/internal/single"
	"github.com/abudden/pmtray/internal/ui"
)

func main() {
	showNow := flag.Bool("show", false, "show the popup on start-up")
	flag.Parse()

	dir, err := config.Dir()
	if err != nil {
		log.Fatal(err)
	}
	_, statErr := os.Stat(filepath.Join(dir, "settings.json"))
	firstRun := os.IsNotExist(statErr)

	var pm *ui.App
	ready := make(chan struct{})
	ok, err := single.Listen(dir, func() {
		<-ready
		fyne.Do(pm.ShowPopup)
	})
	if err != nil {
		log.Printf("single instance check: %v", err)
	}
	if !ok {
		return // the running instance will show its popup
	}

	a := app.NewWithID("org.passwordmaker.pmtray")
	pm = ui.New(a)
	close(ready)
	if *showNow || firstRun {
		a.Lifecycle().SetOnStarted(pm.ShowPopup)
	}
	a.Run()
}
