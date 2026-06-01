// Package chainstate manages persistent state for multi-step chained cheats.
// It tracks the user's progress through complex workflows by recording which
// step of a chain they are currently executing and persisting this state to disk.
package chainstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// State represents the overall persisted chain state across all projects.
type State struct {
	Projects map[string]*ProjectState `json:"projects"`
}

// ProjectState holds the chain tracking information for a specific project root.
type ProjectState struct {
	ActiveChain string         `json:"active_chain,omitempty"`
	Chains      map[string]int `json:"chains,omitempty"`
}

// DefaultPath returns the default filesystem path for the chainstate JSON file.
func DefaultPath() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "cheatmd", "chains.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "cheatmd", "chains.json"), nil
}

// Load reads the chainstate from the specified path, migrating older flat formats
// if necessary, and returns the structured State object.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{Projects: make(map[string]*ProjectState)}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err == nil {
		if state.Projects == nil {
			state.Projects = make(map[string]*ProjectState)
		}
		return &state, nil
	}

	// Unmarshal failed, return empty state
	return &State{Projects: make(map[string]*ProjectState)}, nil
}

func getOrCreateProject(state *State, root string) *ProjectState {
	cleanRoot := filepath.Clean(root)
	p, ok := state.Projects[cleanRoot]
	if !ok {
		p = &ProjectState{Chains: make(map[string]int)}
		state.Projects[cleanRoot] = p
	}
	if p.Chains == nil {
		p.Chains = make(map[string]int)
	}
	return p
}

// Save marshals and writes the given State to the specified path.
func Save(path string, state *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// ActiveName returns the currently active chain name for the given project root.
// Returns an empty string if no chain is active.
func ActiveName(root string, state *State) string {
	if state == nil || state.Projects == nil {
		return ""
	}
	if p, ok := state.Projects[filepath.Clean(root)]; ok {
		return p.ActiveChain
	}
	return ""
}

// SetActive sets the currently active chain name for the given project root.
func SetActive(root, name string, state *State) {
	if state == nil {
		return
	}
	p := getOrCreateProject(state, root)
	p.ActiveChain = name
}

// Clear removes all stored steps and active state for the specified chain within a project.
func Clear(root, name string, state *State) {
	if state == nil || state.Projects == nil {
		return
	}
	cleanRoot := filepath.Clean(root)
	p, ok := state.Projects[cleanRoot]
	if !ok {
		return
	}
	if name == "" {
		// Clear all chains and active status for this project
		delete(state.Projects, cleanRoot)
		return
	}
	delete(p.Chains, name)
	if p.ActiveChain == name {
		p.ActiveChain = ""
	}
}

// GetStep returns the current step integer for the specified chain within a project.
func GetStep(root, name string, state *State) int {
	if state == nil || state.Projects == nil {
		return 0
	}
	if p, ok := state.Projects[filepath.Clean(root)]; ok {
		return p.Chains[name]
	}
	return 0
}

// SetStep records the current step integer for the specified chain within a project.
func SetStep(root, name string, step int, state *State) {
	if state == nil {
		return
	}
	p := getOrCreateProject(state, root)
	p.Chains[name] = step
}
