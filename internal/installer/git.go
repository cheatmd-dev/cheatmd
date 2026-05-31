package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

// hasGit reports whether a git binary is available on PATH.
func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// installViaGit shallow-clones the repo into a temp dir, then copies its .md
// files into target. Cloning into a temp dir (rather than target directly)
// keeps the install atomic-ish and lets us drop the .git metadata.
func installViaGit(ctx context.Context, pack registry.Pack, target string) error {
	tmp, err := os.MkdirTemp("", "cheatmd-clone-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", pack.Repo, tmp)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", pack.Repo, err)
	}

	root := tmp
	if pack.Subdir != "" {
		root = filepath.Join(tmp, filepath.Clean(pack.Subdir))
	}

	n, err := copyMarkdown(root, target)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no .md cheats found in %s", pack.Repo)
	}
	return nil
}

// copyMarkdown walks srcRoot and copies every .md file into dest, preserving
// the relative directory structure under srcRoot. Returns the count copied.
func copyMarkdown(srcRoot, dest string) (int, error) {
	count := 0
	err := filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != srcRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		outPath := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}
