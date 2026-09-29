package parser

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestProseTagsDoNotShareStorage(t *testing.T) {
	p := NewParser()
	path := filepath.Join("one", "two", "three", "tags.md")
	pathTags := make([]string, 3, 8)
	copy(pathTags, []string{"one", "two", "three"})
	p.pathTagsCache[filepath.Dir(path)] = pathTags

	p.parseLines(path, []byte("# First\n```sh\necho first\n```\n<!-- cheat -->\n#first\n# Second\n```sh\necho second\n```\n<!-- cheat -->\n#second\nend\n"))

	want := [][]string{
		{"one", "two", "three", "first"},
		{"one", "two", "three", "second"},
	}
	if len(p.index.Cheats) != len(want) {
		t.Fatalf("got %d cheats, want %d", len(p.index.Cheats), len(want))
	}
	for i, cheat := range p.index.Cheats {
		if !reflect.DeepEqual(cheat.Tags, want[i]) {
			t.Errorf("cheat %q tags = %v, want %v", cheat.Header, cheat.Tags, want[i])
		}
	}
}
