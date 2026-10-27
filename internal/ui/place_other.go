//go:build !windows

package ui

import "fyne.io/fyne/v2"

// placeWindow positions the popup; elsewhere the window manager decides.
func placeWindow(fyne.Window) {}

// fixScale corrects the window's scale where needed (only on Windows).
func fixScale(fyne.Window, func()) {}
