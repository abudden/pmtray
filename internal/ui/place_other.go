//go:build !windows

package ui

import "fyne.io/fyne/v2"

// placeWindow positions the popup; elsewhere the window manager decides.
func placeWindow(fyne.Window) {}
