package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abudden/pmtray/internal/pwm"
	"github.com/abudden/pmtray/internal/rdf"
)

// Store holds the profiles, kept in sync with an RDF file on disk.
type Store struct {
	mu       sync.Mutex
	path     string
	profiles []pwm.Profile
	modTime  time.Time
	size     int64
	lastErr  error
}

// OpenStore loads the profiles from path, creating the file with default
// profiles if it doesn't exist.
func OpenStore(path string) (*Store, error) {
	s := &Store{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		s.profiles = pwm.DefaultProfiles()
		return s, s.Save(s.profiles)
	}
	if err := s.reload(); err != nil {
		s.profiles = pwm.DefaultProfiles()
		return s, err
	}
	return s, nil
}

// Path returns the file backing the store.
func (s *Store) Path() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path
}

// Profiles returns a copy of the profiles in file order.
func (s *Store) Profiles() []pwm.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]pwm.Profile(nil), s.profiles...)
}

// Err returns the error from the most recent attempt to read the file.
func (s *Store) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

func (s *Store) reload() error {
	fi, err := os.Stat(s.path)
	if err != nil {
		s.lastErr = err
		return err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		s.lastErr = err
		return err
	}
	s.modTime, s.size = fi.ModTime(), fi.Size()
	profiles, err := rdf.Parse(data)
	if err != nil {
		s.lastErr = fmt.Errorf("%s: %w", s.path, err)
		return s.lastErr
	}
	s.profiles = profiles
	s.lastErr = nil
	return nil
}

// CheckForChanges re-reads the file if it has changed on disk since it was
// last read or written, and reports whether the profiles were reloaded.
// If the file can't be parsed the current profiles are kept.
func (s *Store) CheckForChanges() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	fi, err := os.Stat(s.path)
	if err != nil {
		return false // deleted or inaccessible: keep what we have
	}
	if fi.ModTime().Equal(s.modTime) && fi.Size() == s.size {
		return false
	}
	return s.reload() == nil
}

// Save writes profiles to the file.
func (s *Store) Save(profiles []pwm.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := writeFileAtomic(s.path, rdf.Marshal(profiles), 0o600); err != nil {
		return err
	}
	s.profiles = append([]pwm.Profile(nil), profiles...)
	if fi, err := os.Stat(s.path); err == nil {
		s.modTime, s.size = fi.ModTime(), fi.Size()
	}
	s.lastErr = nil
	return nil
}

// SwitchTo makes the store use a different file. An existing file is
// loaded; otherwise it is created containing the current profiles.
func (s *Store) SwitchTo(path string) error {
	s.mu.Lock()
	current := s.profiles
	s.mu.Unlock()
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := rdf.Parse(data); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.path = path
		return s.reload()
	}
	s.mu.Lock()
	s.path = path
	s.mu.Unlock()
	return s.Save(current)
}

// Watch polls the file for changes until stop is closed, calling onChange
// (from another goroutine) whenever the profiles are reloaded.
func (s *Store) Watch(interval time.Duration, stop <-chan struct{}, onChange func()) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if s.CheckForChanges() {
				onChange()
			}
		}
	}
}

// SortedIndices returns the indices of profiles in display order. The first
// profile always comes first; the rest are ordered by sortOrder.
func SortedIndices(profiles []pwm.Profile, sortOrder string) []int {
	idx := make([]int, len(profiles))
	for i := range idx {
		idx[i] = i
	}
	if len(idx) < 2 {
		return idx
	}
	rest := idx[1:]
	switch sortOrder {
	case "alphabetical":
		sort.SliceStable(rest, func(a, b int) bool {
			return strings.ToUpper(profiles[rest[a]].Title) < strings.ToUpper(profiles[rest[b]].Title)
		})
	case "newest_first":
		sort.SliceStable(rest, func(a, b int) bool {
			return profiles[rest[a]].Timestamp > profiles[rest[b]].Timestamp
		})
	case "oldest_first":
		sort.SliceStable(rest, func(a, b int) bool {
			return profiles[rest[a]].Timestamp < profiles[rest[b]].Timestamp
		})
	}
	return idx
}
