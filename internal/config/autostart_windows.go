package config

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValue = "PasswordMaker Tray"

// AutostartSupported reports whether SetAutostart works on this platform.
func AutostartSupported() bool { return true }

// Autostart reports whether the application starts at login.
func Autostart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValue)
	return err == nil
}

// SetAutostart enables or disables starting the application at login.
func SetAutostart(enable bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !enable {
		if err := k.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValue, `"`+exe+`"`)
}
