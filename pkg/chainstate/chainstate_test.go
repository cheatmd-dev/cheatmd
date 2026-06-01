package chainstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveComplexState(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	state := &State{Projects: make(map[string]*ProjectState)}

	// Set up some initial state
	SetActive("/my/project1", "chainA", state)
	SetStep("/my/project1", "chainA", 2, state)
	SetStep("/my/project1", "chainB", 5, state)

	SetActive("/my/project2", "chainX", state)

	err := Save(path, state)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load it back
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if ActiveName("/my/project1", loaded) != "chainA" {
		t.Errorf("Expected active chain 'chainA', got '%s'", ActiveName("/my/project1", loaded))
	}
	if GetStep("/my/project1", "chainA", loaded) != 2 {
		t.Errorf("Expected step 2, got %d", GetStep("/my/project1", "chainA", loaded))
	}
	if GetStep("/my/project1", "chainB", loaded) != 5 {
		t.Errorf("Expected step 5, got %d", GetStep("/my/project1", "chainB", loaded))
	}
	if ActiveName("/my/project2", loaded) != "chainX" {
		t.Errorf("Expected active chain 'chainX', got '%s'", ActiveName("/my/project2", loaded))
	}
}

func TestClear(t *testing.T) {
	state := &State{Projects: make(map[string]*ProjectState)}
	SetStep("/root", "chain1", 1, state)
	SetActive("/root", "chain1", state)
	SetStep("/root", "chain2", 2, state)

	// Clear specific chain that is active
	Clear("/root", "chain1", state)

	if GetStep("/root", "chain1", state) != 0 {
		t.Errorf("Expected chain1 step to be 0")
	}
	if ActiveName("/root", state) != "" {
		t.Errorf("Expected active name to be cleared")
	}
	if GetStep("/root", "chain2", state) != 2 {
		t.Errorf("Expected chain2 step to remain 2")
	}

	// Clear whole project
	Clear("/root", "", state)
	if _, exists := state.Projects["/root"]; exists {
		t.Errorf("Expected project /root to be completely removed")
	}
}

func TestLoadEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Missing file should return empty state
	state, err := Load(filepath.Join(tmpDir, "missing.json"))
	if err != nil {
		t.Errorf("Missing file should not error, got: %v", err)
	}
	if state == nil || state.Projects == nil {
		t.Errorf("Missing file should return initialized empty state")
	}

	// 2. Corrupt file should return empty state without crashing
	path := filepath.Join(tmpDir, "corrupt.json")
	if err := os.WriteFile(path, []byte("{ invalid json"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	state, err = Load(path)
	if err != nil {
		t.Errorf("Corrupt file should not error out completely, should recover empty state. Got error: %v", err)
	}
	if state == nil || state.Projects == nil {
		t.Errorf("Corrupt file should return initialized empty state")
	}

	// 3. Empty file
	pathEmpty := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(pathEmpty, []byte(""), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	state, err = Load(pathEmpty)
	if err != nil {
		t.Errorf("Empty file should not error, got: %v", err)
	}
	if state == nil || state.Projects == nil {
		t.Errorf("Empty file should return initialized empty state")
	}
}
