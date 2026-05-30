package packmanifest

import (
	"testing"
	"time"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	m, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load on dir without manifest: %v", err)
	}
	if len(m.Packs) != 0 {
		t.Errorf("expected empty manifest, got %d packs", len(m.Packs))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{}
	m.Upsert(Entry{Name: "git", Repo: "https://github.com/x/git", InstalledAt: time.Now()})
	m.Upsert(Entry{Name: "docker", Repo: "https://github.com/x/docker", InstalledAt: time.Now()})

	if err := Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Packs) != 2 {
		t.Fatalf("got %d packs, want 2", len(got.Packs))
	}
	if !got.Has("git") || !got.Has("docker") {
		t.Errorf("manifest missing expected packs: %+v", got.Packs)
	}
	if got.Has("nope") {
		t.Error("Has returned true for an uninstalled pack")
	}
}

func TestUpsertReplacesByName(t *testing.T) {
	m := &Manifest{}
	m.Upsert(Entry{Name: "git", Repo: "old"})
	m.Upsert(Entry{Name: "git", Repo: "new"})

	if len(m.Packs) != 1 {
		t.Fatalf("expected 1 pack after upsert, got %d", len(m.Packs))
	}
	if m.Packs[0].Repo != "new" {
		t.Errorf("repo = %q, want new", m.Packs[0].Repo)
	}
}
