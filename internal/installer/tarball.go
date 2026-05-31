package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheatmd-dev/cheatmd/internal/httputil"
	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

// tarballTimeout bounds the HTTPS fallback download.
const tarballTimeout = 60 * time.Second

// installViaTarball downloads the GitHub default-branch tarball and extracts
// its .md files into target. GitHub-only (clone covers other hosts).
func installViaTarball(ctx context.Context, pack registry.Pack, target string) error {
	owner, repo, err := parseGitHubRepo(pack.Repo)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, tarballTimeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball", owner, repo)
	respBody, err := httputil.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}
	defer respBody.Close()

	n, err := extractMarkdownTarball(respBody, pack.Subdir, target)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no .md cheats found in %s", pack.Repo)
	}
	return nil
}

// extractMarkdownTarball reads a gzip-compressed tarball from r and writes its
// .md files into dest. GitHub tarballs wrap everything in a top-level
// "<owner>-<repo>-<sha>/" directory, which we strip; subdir (if set) further
// restricts which files are kept. Returns the count extracted.
func extractMarkdownTarball(r io.Reader, subdir, dest string) (int, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	subdir = normalizeSubdir(subdir)

	count := 0
	var totalBytes int64
	const maxFiles = 5000
	const maxTotalBytes = 50 * 1024 * 1024 // 50MB
	const maxFileSize = 5 * 1024 * 1024    // 5MB

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("read tar: %w", err)
		}

		rel, keep := markdownRelPath(hdr, subdir)
		if !keep {
			continue
		}

		if count >= maxFiles {
			return count, fmt.Errorf("tarball exceeds maximum allowed files (%d)", maxFiles)
		}
		if hdr.Size > maxFileSize {
			return count, fmt.Errorf("tar entry %q exceeds maximum file size (5MB)", hdr.Name)
		}
		totalBytes += hdr.Size
		if totalBytes > maxTotalBytes {
			return count, fmt.Errorf("tarball exceeds maximum extracted size (50MB)")
		}

		if err := writeTarEntry(tr, hdr, dest, rel); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// writeTarEntry writes one tar entry to dest/rel, guarding against path
// traversal (zip-slip) and capping the copy to the declared size to avoid
// decompression bombs.
func writeTarEntry(tr *tar.Reader, hdr *tar.Header, dest, rel string) error {
	outPath := filepath.Join(dest, filepath.FromSlash(rel))
	if !withinDir(dest, outPath) {
		return fmt.Errorf("tar entry escapes destination: %q", hdr.Name)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, io.LimitReader(tr, hdr.Size)); err != nil {
		return err
	}
	return nil
}

// parseGitHubRepo extracts owner/repo from a GitHub repo URL or "owner/repo".
func parseGitHubRepo(repo string) (owner, name string, err error) {
	s := strings.TrimSpace(repo)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.Trim(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("tarball fallback supports github.com repos only, got %q", repo)
	}
	return parts[0], parts[1], nil
}
