package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestFooterPreservesOrdinaryMarkdown(t *testing.T) {
	for _, block := range []string{
		"## Greeting\n```sh\necho hello\n```",
		"Ordinary prose mentioning tags: [greeting]",
		"description: ordinary metadata",
		"description: |\n  tags: [greeting]",
		"metadata:\n  tags: [greeting]",
		"- tags: [greeting]",
		"tags: [unterminated",
	} {
		t.Run(block, func(t *testing.T) {
			input := "# Documentation\nIntro\n---\n" + block + "\n---\n"
			body, tags := extractFooterTags([]byte(input))
			if string(body) != input || len(tags) != 0 {
				t.Fatalf("extractFooterTags() = (%q, %v), want (%q, [])", body, tags, input)
			}
		})
	}
}

func TestFooterTagsPreserveCommandBetweenRules(t *testing.T) {
	for _, tc := range []struct {
		name, footer string
		wantTags     []string
	}{
		{"ordinary closing rule", "---\n", nil},
		{"inline tags", "---\ntags: [greeting, sample]\n---\n", []string{"greeting", "sample"}},
		{"case insensitive tags", "---\nTaGs: Greeting, Sample\n---\n", []string{"Greeting", "Sample"}},
		{"list tags", "---\ntags:\n  - greeting\n  - sample\n---\n", []string{"greeting", "sample"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commandBody := "# Documentation\nIntro\n---\n## Greeting\n```sh\necho hello\n```\n"
			input := commandBody + tc.footer
			body, tags := extractFooterTags([]byte(input))
			wantBody := commandBody
			if tc.wantTags == nil {
				wantBody = input
			}
			if string(body) != wantBody || !reflect.DeepEqual(tags, tc.wantTags) {
				t.Fatalf("extractFooterTags() = (%q, %v), want (%q, %v)", body, tags, wantBody, tc.wantTags)
			}
			path := filepath.Join(t.TempDir(), "rules.md")
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			index, err := NewParser().ParseSingleFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(index.Cheats) != 1 || index.Cheats[0].Command != "echo hello" {
				t.Fatalf("parsed cheats = %+v, want one echo hello command", index.Cheats)
			}
			if tc.wantTags != nil && (!slices.Contains(index.Cheats[0].Tags, "greeting") || !slices.Contains(index.Cheats[0].Tags, "sample")) {
				t.Fatalf("footer tags not applied: %v", index.Cheats[0].Tags)
			}
		})
	}
}
