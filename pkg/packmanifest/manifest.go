// Package packmanifest records which cheat packs have been installed into a
// cheats directory. It is the source of truth for "is this pack installed?",
// replacing fragile folder-name guessing: a directory only counts as an
// installed pack if it was recorded here by the installer.
//
// The manifest is stored as ".cheatmd-packs.json" at the root of the cheats
// directory. The cheat parser only reads .md files, so the manifest is ignored
// during normal browsing.
package packmanifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// FileName is the manifest's basename within the cheats directory.
const FileName = ".cheatmd-packs.json"

// Entry records a single installed pack.
type Entry struct {
	Name        string    `json:"name"`
	Repo        string    `json:"repo"`
	InstalledAt time.Time `json:"installed_at"`
}

// Manifest is the set of installed packs in a cheats directory.
type Manifest struct {
	Packs []Entry `json:"packs"`
}

// pathFor returns the manifest file path for a cheats directory.
func pathFor(dir string) string {
	return filepath.Join(dir, FileName)
}

// Load reads the manifest from dir. A missing manifest is not an error: it
// returns an empty manifest so callers can treat "no file" as "nothing
// installed".
func Load(dir string) (*Manifest, error) {
	data, err := os.ReadFile(pathFor(dir))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Manifest{}, nil
		}
		return nil, fmt.Errorf("read pack manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse pack manifest: %w", err)
	}
	return &m, nil
}

// Save writes the manifest to dir, creating the directory if needed.
func Save(dir string, m *Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create cheats dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode pack manifest: %w", err)
	}
	if err := os.WriteFile(pathFor(dir), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write pack manifest: %w", err)
	}
	return nil
}

// Has reports whether a pack with the given name is recorded as installed.
func (m *Manifest) Has(name string) bool {
	for _, e := range m.Packs {
		if e.Name == name {
			return true
		}
	}
	return false
}

// Get returns the entry for the given pack name, or nil if not installed.
func (m *Manifest) Get(name string) *Entry {
	for i := range m.Packs {
		if m.Packs[i].Name == name {
			return &m.Packs[i]
		}
	}
	return nil
}

// Upsert records (or refreshes) an installed pack, replacing any existing
// entry with the same name.
func (m *Manifest) Upsert(e Entry) {
	for i := range m.Packs {
		if m.Packs[i].Name == e.Name {
			m.Packs[i] = e
			return
		}
	}
	m.Packs = append(m.Packs, e)
}

// Remove deletes a pack from the manifest.
func (m *Manifest) Remove(name string) {
	for i := range m.Packs {
		if m.Packs[i].Name == name {
			m.Packs = append(m.Packs[:i], m.Packs[i+1:]...)
			return
		}
	}
}
