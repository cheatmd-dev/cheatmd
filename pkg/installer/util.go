package installer

import (
	"archive/tar"
	"path/filepath"
	"strings"
)

// normalizeSubdir cleans a subdir filter to a slash path, treating an empty or
// "." value as "no filter".
func normalizeSubdir(subdir string) string {
	s := filepath.ToSlash(filepath.Clean(strings.TrimSpace(subdir)))
	if s == "." {
		return ""
	}
	return s
}

// markdownRelPath reports whether a tar entry is a .md file we should keep and,
// if so, returns its destination-relative path: the GitHub wrapper directory
// stripped, and the subdir prefix (when set) removed.
func markdownRelPath(hdr *tar.Header, subdir string) (rel string, keep bool) {
	if hdr.Typeflag != tar.TypeReg {
		return "", false
	}
	if !strings.EqualFold(filepath.Ext(hdr.Name), ".md") {
		return "", false
	}

	rel = stripFirstPathComponent(hdr.Name)
	if rel == "" {
		return "", false
	}
	if subdir == "" {
		return rel, true
	}
	if !strings.HasPrefix(rel+"/", subdir+"/") {
		return "", false
	}
	return strings.TrimPrefix(rel, subdir+"/"), true
}

// stripFirstPathComponent removes the leading path segment (the GitHub tarball
// wrapper dir), returning a forward-slash relative path.
func stripFirstPathComponent(name string) string {
	name = filepath.ToSlash(name)
	_, rest, found := strings.Cut(name, "/")
	if !found {
		return ""
	}
	return rest
}

// withinDir reports whether target resolves to a path inside dir.
func withinDir(dir, target string) bool {
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
