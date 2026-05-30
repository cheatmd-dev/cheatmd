// Package registry fetches and parses the CheatMD cheat-pack registry: a
// remote YAML manifest that lists installable cheat repositories ("packs").
//
// The registry is hosted in its own repo (see the cheatmd.registry scaffold);
// the tool fetches it over HTTPS during first-run setup so users can pick
// official packs of cheats to install.
package registry

import (
	"context"
	"fmt"
	"github.com/cheatmd-dev/cheatmd/pkg/httputil"
	"io"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"
)

// Pack describes a single installable cheat repository in the registry.
type Pack struct {
	// Name is a short identifier, also used as the install subdirectory name.
	Name string `yaml:"name"`
	// Repo is the clonable repository URL (e.g. https://github.com/owner/repo).
	Repo string `yaml:"repo"`
	// Description is a one-line human summary shown in the picker.
	Description string `yaml:"description"`
	// Official marks packs as officially maintained, pre-selected during first-run.
	Official bool `yaml:"official"`
	// Subdir optionally restricts installation to .md files under this path
	// within the repo. Empty means the whole repo.
	Subdir string `yaml:"subdir"`
}

// Registry is the parsed manifest.
type Registry struct {
	Version int    `yaml:"version"`
	Packs   []Pack `yaml:"packs"`
}

// defaultTimeout bounds the registry fetch so first-run setup never hangs.
const defaultTimeout = 15 * time.Second

// Fetch retrieves and parses the registry manifest at url.
func Fetch(ctx context.Context, url string) (*Registry, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	respBody, err := httputil.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch registry: %w", err)
	}
	defer respBody.Close()

	data, err := io.ReadAll(respBody)
	if err != nil {
		return nil, fmt.Errorf("read registry body: %w", err)
	}

	return Parse(data)
}

// Parse unmarshals registry YAML and validates required fields. It is split
// from Fetch so it can be unit-tested without a network round-trip.
func Parse(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	valid := reg.Packs[:0]
	for _, p := range reg.Packs {
		p.Name = strings.TrimSpace(p.Name)
		p.Repo = strings.TrimSpace(p.Repo)
		if p.Name == "" || p.Repo == "" {
			// Skip malformed entries rather than failing the whole registry.
			continue
		}
		valid = append(valid, p)
	}
	reg.Packs = valid

	if len(reg.Packs) == 0 {
		return nil, fmt.Errorf("registry contains no valid packs")
	}
	return &reg, nil
}

// OfficialPacks returns the packs flagged as official packs.
func (r *Registry) OfficialPacks() []Pack {
	var out []Pack
	for _, p := range r.Packs {
		if p.Official {
			out = append(out, p)
		}
	}
	return out
}

// Select resolves pack names to their entries, preserving the order given. It
// errors on the first unknown name.
func (r *Registry) Select(names []string) ([]Pack, error) {
	byName := make(map[string]Pack, len(r.Packs))
	for _, p := range r.Packs {
		byName[p.Name] = p
	}

	out := make([]Pack, 0, len(names))
	for _, name := range names {
		p, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("unknown pack %q", name)
		}
		out = append(out, p)
	}
	return out, nil
}
