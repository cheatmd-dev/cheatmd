package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// makeTarball builds a gzip+tar archive from name->content entries, mimicking
// GitHub's top-level wrapper directory.
func makeTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func TestExtractMarkdownTarball(t *testing.T) {
	tb := makeTarball(t, map[string]string{
		"owner-repo-abc123/git.md":         "# git",
		"owner-repo-abc123/nested/k8s.md":  "# k8s",
		"owner-repo-abc123/README.txt":     "ignore me",
		"owner-repo-abc123/LICENSE":        "ignore me",
		"owner-repo-abc123/assets/pic.png": "binary",
	})

	dest := t.TempDir()
	n, err := extractMarkdownTarball(bytes.NewReader(tb), "", dest)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if n != 2 {
		t.Fatalf("extracted %d files, want 2", n)
	}

	if got, _ := os.ReadFile(filepath.Join(dest, "git.md")); string(got) != "# git" {
		t.Errorf("git.md content = %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "nested", "k8s.md")); string(got) != "# k8s" {
		t.Errorf("nested/k8s.md content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.txt")); !os.IsNotExist(err) {
		t.Error("README.txt should not have been extracted")
	}
}

func TestExtractMarkdownTarballSubdir(t *testing.T) {
	tb := makeTarball(t, map[string]string{
		"owner-repo-abc123/top.md":          "top",
		"owner-repo-abc123/cheats/git.md":   "git",
		"owner-repo-abc123/cheats/sub/d.md": "d",
	})

	dest := t.TempDir()
	n, err := extractMarkdownTarball(bytes.NewReader(tb), "cheats", dest)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if n != 2 {
		t.Fatalf("extracted %d files, want 2", n)
	}
	// subdir prefix should be stripped from the output paths.
	if _, err := os.Stat(filepath.Join(dest, "git.md")); err != nil {
		t.Errorf("expected git.md at dest root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "top.md")); !os.IsNotExist(err) {
		t.Error("top.md is outside subdir and should be skipped")
	}
}

func TestExtractMarkdownTarballZipSlip(t *testing.T) {
	tb := makeTarball(t, map[string]string{
		"owner-repo-abc123/../../evil.md": "pwned",
	})

	dest := t.TempDir()
	if _, err := extractMarkdownTarball(bytes.NewReader(tb), "", dest); err == nil {
		t.Fatal("expected zip-slip guard to reject path-escaping entry")
	}
}

func TestParseGitHubRepo(t *testing.T) {
	cases := map[string][2]string{
		"https://github.com/owner/repo":     {"owner", "repo"},
		"https://github.com/owner/repo.git": {"owner", "repo"},
		"github.com/owner/repo":             {"owner", "repo"},
		"owner/repo":                        {"owner", "repo"},
	}
	for in, want := range cases {
		o, r, err := parseGitHubRepo(in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", in, err)
			continue
		}
		if o != want[0] || r != want[1] {
			t.Errorf("%q: got %s/%s, want %s/%s", in, o, r, want[0], want[1])
		}
	}

	if _, _, err := parseGitHubRepo("notarepo"); err == nil {
		t.Error("expected error for malformed repo")
	}
}
