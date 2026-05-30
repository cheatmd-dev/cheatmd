package registry

import "testing"

const sampleYAML = `
version: 1
packs:
  - name: git
    repo: https://github.com/cheatmd-dev/cheats-git
    description: Git everyday commands
    official: true
  - name: docker
    repo: https://github.com/cheatmd-dev/cheats-docker
    description: Docker commands
    official: false
  - name: ""
    repo: https://github.com/cheatmd-dev/bad
    description: missing name, should be skipped
`

func TestParse(t *testing.T) {
	reg, err := Parse([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if reg.Version != 1 {
		t.Errorf("Version = %d, want 1", reg.Version)
	}

	// The empty-name entry must be dropped.
	if len(reg.Packs) != 2 {
		t.Fatalf("got %d packs, want 2", len(reg.Packs))
	}

	if reg.Packs[0].Name != "git" {
		t.Errorf("first pack name = %q, want git", reg.Packs[0].Name)
	}
}

func TestOfficialPacks(t *testing.T) {
	reg, err := Parse([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	official := reg.OfficialPacks()
	if len(official) != 1 {
		t.Fatalf("got %d official packs, want 1", len(official))
	}
	if official[0].Name != "git" {
		t.Errorf("official pack = %q, want git", official[0].Name)
	}
}

func TestParseRejectsEmptyRegistry(t *testing.T) {
	if _, err := Parse([]byte("version: 1\npacks: []\n")); err == nil {
		t.Fatal("expected error for empty registry, got nil")
	}
}

func TestSelect(t *testing.T) {
	reg, err := Parse([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Order is preserved as requested, not as listed in the registry.
	got, err := reg.Select([]string{"docker", "git"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(got) != 2 || got[0].Name != "docker" || got[1].Name != "git" {
		t.Fatalf("Select returned %+v, want [docker git]", got)
	}

	if _, err := reg.Select([]string{"git", "nope"}); err == nil {
		t.Error("expected error for unknown pack name")
	}
}
