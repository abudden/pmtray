package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abudden/pmtray/internal/pwm"
	"github.com/abudden/pmtray/internal/rdf"
)

func TestStoreCreatesAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "profiles.rdf")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Profiles(); len(got) != 2 || got[1].Title != "Alphanumeric" {
		t.Fatalf("unexpected defaults: %+v", got)
	}
	if s.CheckForChanges() {
		t.Error("reported change after own save")
	}

	// Simulate an external edit.
	p := pwm.DefaultProfiles()
	p[0].Title = "Edited"
	if err := os.WriteFile(path, rdf.Marshal(p), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	os.Chtimes(path, future, future)
	if !s.CheckForChanges() {
		t.Fatal("external change not detected")
	}
	if s.Profiles()[0].Title != "Edited" {
		t.Errorf("not reloaded: %+v", s.Profiles()[0])
	}

	// A broken file keeps the previous profiles.
	os.WriteFile(path, []byte("garbage <"), 0o600)
	future = future.Add(2 * time.Second)
	os.Chtimes(path, future, future)
	if s.CheckForChanges() || s.Err() == nil || s.Profiles()[0].Title != "Edited" {
		t.Error("broken file handling")
	}
}

func TestSwitchTo(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(filepath.Join(dir, "a.rdf"))
	if err != nil {
		t.Fatal(err)
	}
	p := s.Profiles()
	p[0].Title = "Mine"
	s.Save(p)
	b := filepath.Join(dir, "b.rdf")
	if err := s.SwitchTo(b); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(b); err != nil || s.Profiles()[0].Title != "Mine" {
		t.Error("new file not created with current profiles")
	}
	os.WriteFile(filepath.Join(dir, "bad.rdf"), []byte("x"), 0o600)
	if err := s.SwitchTo(filepath.Join(dir, "bad.rdf")); err == nil || s.Path() != b {
		t.Error("switching to a broken file should fail and keep the old file")
	}
}

func TestSortedIndices(t *testing.T) {
	p := []pwm.Profile{{Title: "Z", Timestamp: 5}, {Title: "b", Timestamp: 1}, {Title: "A", Timestamp: 3}}
	check := func(order string, want ...int) {
		got := SortedIndices(p, order)
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %v want %v", order, got, want)
				return
			}
		}
	}
	check("user_defined", 0, 1, 2)
	check("alphabetical", 0, 2, 1)
	check("newest_first", 0, 2, 1)
	check("oldest_first", 0, 1, 2)
}

func TestMasterHash(t *testing.T) {
	h, err := NewMasterHash("secret")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Matches("secret") || h.Matches("Secret") {
		t.Error("hash mismatch")
	}
}
