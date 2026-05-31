package resolver

import (
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestSearchCheats(t *testing.T) {
	c1 := &parser.Cheat{
		File:        "/path/to/project/docker.md",
		Header:      "Build image",
		Description: "Builds a docker image",
		Command:     "docker build -t app .",
		Tags:        []string{"container"},
	}
	c2 := &parser.Cheat{
		File:        "/path/to/project/k8s.md",
		Header:      "Deploy to k8s",
		Description: "Deploys to kubernetes",
		Command:     "kubectl apply -f .",
		Tags:        []string{"container", "deploy"},
	}

	cheats := []*parser.Cheat{c1, c2}

	// Test folder match
	if len(SearchCheats(cheats, []string{"project"})) != 2 {
		t.Error("Expected both cheats to match folder 'project'")
	}

	// Test file match
	if len(SearchCheats(cheats, []string{"docker"})) != 1 {
		t.Error("Expected only c1 to match 'docker'")
	}

	// Test header match
	if len(SearchCheats(cheats, []string{"deploy", "k8s"})) != 1 {
		t.Error("Expected only c2 to match 'deploy k8s'")
	}

	// Test tag match
	if len(SearchCheats(cheats, []string{"container"})) != 2 {
		t.Error("Expected both cheats to match tag 'container'")
	}
}

func TestFindExactHeaderMatch(t *testing.T) {
	c1 := &parser.Cheat{Header: "Build Image"}
	c2 := &parser.Cheat{Header: "Deploy App"}
	cheats := []*parser.Cheat{c1, c2}

	if match := FindExactHeaderMatch(cheats, "build image"); match != c1 {
		t.Error("Expected case-insensitive exact match for 'build image'")
	}
	if match := FindExactHeaderMatch(cheats, "deploy app"); match != c2 {
		t.Error("Expected case-insensitive exact match for 'deploy app'")
	}
	if match := FindExactHeaderMatch(cheats, "build"); match != nil {
		t.Error("Expected nil for partial match")
	}
}

func TestMatchesAllWords(t *testing.T) {
	text := "the quick brown fox"
	if !MatchesAllWords(text, []string{"quick", "fox"}) {
		t.Error("Expected text to match all words")
	}
	if MatchesAllWords(text, []string{"quick", "lazy"}) {
		t.Error("Expected text to not match missing words")
	}
}
