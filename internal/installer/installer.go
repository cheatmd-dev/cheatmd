// Package installer installs cheat packs from the registry onto the local
// machine. It prefers the user's git binary (shallow clone) and falls back to
// downloading the repository tarball over HTTPS when git is unavailable.
package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"regexp"

	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

// Install places pack's markdown cheats under destDir/<pack.Name>.
//
// It first tries `git clone --depth 1`; if git is not on PATH (or the clone
// fails), it falls back to downloading the GitHub tarball. Only .md files are
// kept, optionally restricted to pack.Subdir.

var validPackName = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

func Install(ctx context.Context, pack registry.Pack, destDir string) error {
	if !validPackName.MatchString(pack.Name) {
		return fmt.Errorf("invalid pack name %q: must contain only alphanumeric, dash, or underscore characters", pack.Name)
	}

	target := filepath.Join(destDir, pack.Name)
	if !withinDir(destDir, target) {
		return fmt.Errorf("invalid pack name %q: resolves outside destination directory", pack.Name)
	}
	backup := target + ".bak"

	// If the pack exists (e.g. during an update), move it aside so we can
	// cleanly overwrite it without leaving stale files.
	hasOld := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("backup existing pack: %w", err)
		}
		hasOld = true
		defer os.RemoveAll(backup)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		if hasOld {
			os.Rename(backup, target)
		}
		return fmt.Errorf("create install dir: %w", err)
	}

	// Helper to handle restoration on failure.
	restoreOnFail := func(err error) error {
		if err != nil {
			os.RemoveAll(target)
			if hasOld {
				os.Rename(backup, target)
			}
		}
		return err
	}

	// Without git, the tarball download is the only option.
	if !hasGit() {
		return restoreOnFail(installViaTarball(ctx, pack, target))
	}

	gitErr := installViaGit(ctx, pack, target)
	if gitErr == nil {
		return restoreOnFail(nil)
	}

	// git is present but failed; fall back to the tarball, surfacing both
	// failures if that fails too.
	if tarErr := installViaTarball(ctx, pack, target); tarErr != nil {
		return restoreOnFail(fmt.Errorf("git clone failed (%v) and tarball fallback failed: %w", gitErr, tarErr))
	}
	return restoreOnFail(nil)
}
