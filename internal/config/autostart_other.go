//go:build !windows && !linux

package config

import "errors"

// AutostartSupported reports whether SetAutostart works on this platform.
func AutostartSupported() bool { return false }

// Autostart reports whether the application starts at login.
func Autostart() bool { return false }

// SetAutostart enables or disables starting the application at login.
func SetAutostart(bool) error { return errors.New("not supported on this platform") }
