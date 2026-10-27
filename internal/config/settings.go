// Package config holds the application's global settings and the profiles
// store backed by an RDF file.
package config

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Store locations for the master password, as in the Chrome extension.
const (
	StoreMemory       = "memory"        // remembered while the application runs
	StoreMemoryExpire = "memory_expire" // remembered for ExpireMinutes
	StoreDisk         = "disk"          // saved (encrypted) in the settings file
	StoreNever        = "never"         // forgotten whenever the popup is hidden
)

// StoreLocations lists the store locations with display names.
var StoreLocations = []struct{ ID, Name string }{
	{StoreMemory, "Per Session"},
	{StoreMemoryExpire, "Per Session, Expire"},
	{StoreDisk, "On Disk"},
	{StoreNever, "Never Store"},
}

// Profile sort orders.
var SortOrders = []struct{ ID, Name string }{
	{"user_defined", "User-Defined Order"},
	{"alphabetical", "Alphabetical Order"},
	{"newest_first", "Newest First"},
	{"oldest_first", "Oldest First"},
}

// MasterHash is a salted PBKDF2 hash of the master password.
type MasterHash struct {
	Hash []byte `json:"hash"`
	Salt []byte `json:"salt"`
	Iter int    `json:"iter"`
}

// Settings are the global options.
type Settings struct {
	ProfilesFile          string      `json:"profiles_file"`
	ZoomLevel             int         `json:"zoom_level"`
	StoreLocation         string      `json:"store_location"`
	ExpireMinutes         int         `json:"expire_minutes"`
	SortProfiles          string      `json:"sort_profiles"`
	HideGeneratedPassword bool        `json:"hide_generated_password"`
	UseVerificationCode   bool        `json:"use_verification_code"`
	ShowPasswordStrength  bool        `json:"show_password_strength"`
	ShowModifiedTime      bool        `json:"show_modified_time"`
	MasterPasswordHash    *MasterHash `json:"master_password_hash,omitempty"`
	ShowUsername          bool        `json:"show_username"`
	StoredPassword        string      `json:"stored_password,omitempty"`
}

// DefaultSettings returns the settings used when none have been saved.
func DefaultSettings() Settings {
	return Settings{
		ZoomLevel:           100,
		StoreLocation:       StoreMemory,
		ExpireMinutes:       5,
		SortProfiles:        "user_defined",
		UseVerificationCode: true,
	}
}

// Dir returns the directory holding the settings (and, by default, the
// profiles file), creating it if necessary.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "pmtray")
	return dir, os.MkdirAll(dir, 0o700)
}

// DefaultProfilesFile is the profiles file used if the user hasn't chosen one.
func DefaultProfilesFile() string {
	dir, err := Dir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "profiles.rdf")
}

func settingsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// LoadSettings reads the settings file, returning defaults for anything
// missing.
func LoadSettings() (Settings, error) {
	s := DefaultSettings()
	path, err := settingsPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	if s.ZoomLevel < 50 || s.ZoomLevel > 300 {
		s.ZoomLevel = 100
	}
	if s.ExpireMinutes < 1 {
		s.ExpireMinutes = 5
	}
	return s, err
}

// Save writes the settings file.
func (s Settings) Save() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o600)
}

// ProfilesPath returns the profiles file in use.
func (s Settings) ProfilesPath() string {
	if s.ProfilesFile != "" {
		return s.ProfilesFile
	}
	return DefaultProfilesFile()
}

const hashIterations = 100000

// NewMasterHash hashes password for later verification.
func NewMasterHash(password string) (*MasterHash, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	h, err := pbkdf2.Key(sha256.New, password, salt, hashIterations, 32)
	if err != nil {
		return nil, err
	}
	return &MasterHash{Hash: h, Salt: salt, Iter: hashIterations}, nil
}

// Matches reports whether password matches the hash.
func (m *MasterHash) Matches(password string) bool {
	h, err := pbkdf2.Key(sha256.New, password, m.Salt, m.Iter, len(m.Hash))
	return err == nil && subtle.ConstantTimeCompare(h, m.Hash) == 1
}

// SetStoredPassword saves the master password, encrypted for the current
// user where the platform supports it.
func (s *Settings) SetStoredPassword(pw string) {
	if pw == "" {
		s.StoredPassword = ""
		return
	}
	enc, err := protect([]byte(pw))
	if err != nil {
		s.StoredPassword = ""
		return
	}
	s.StoredPassword = base64.StdEncoding.EncodeToString(enc)
}

// GetStoredPassword returns the saved master password, if any.
func (s Settings) GetStoredPassword() string {
	if s.StoredPassword == "" {
		return ""
	}
	enc, err := base64.StdEncoding.DecodeString(s.StoredPassword)
	if err != nil {
		return ""
	}
	pw, err := unprotect(enc)
	if err != nil {
		return ""
	}
	return string(pw)
}

// writeFileAtomic writes data to a temporary file then renames it over
// path, so readers never see a partially written file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmp.Name(), perm)
	return os.Rename(tmp.Name(), path)
}
