package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func desktopFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "autostart", "pmtray.desktop"), nil
}

// AutostartSupported reports whether SetAutostart works on this platform.
func AutostartSupported() bool { return true }

// Autostart reports whether the application starts at login.
func Autostart() bool {
	path, err := desktopFile()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// SetAutostart enables or disables starting the application at login.
func SetAutostart(enable bool) error {
	path, err := desktopFile()
	if err != nil {
		return err
	}
	if !enable {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	entry := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=PasswordMaker Tray\nExec=%q\nX-GNOME-Autostart-enabled=true\n", exe)
	return os.WriteFile(path, []byte(entry), 0o644)
}
