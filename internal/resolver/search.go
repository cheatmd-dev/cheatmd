package resolver

import (
	"path/filepath"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// SearchCheats filters a slice of cheats returning only those that match all query words.
// Query words must be pre-lowercased.
func SearchCheats(cheats []*parser.Cheat, words []string) []*parser.Cheat {
	var matched []*parser.Cheat
	for _, c := range cheats {
		if MatchesQuery(c, words) {
			matched = append(matched, c)
		}
	}
	return matched
}

// MatchesQuery returns true if the cheat's metadata contains all query words.
// Query words must be pre-lowercased.
func MatchesQuery(cheat *parser.Cheat, words []string) bool {
	for _, word := range words {
		if !ContainsWord(cheat, word) {
			return false
		}
	}
	return true
}

// ContainsWord returns true if the cheat's metadata (folder, file, header, description, command, tags)
// contains the given lowercase word.
func ContainsWord(cheat *parser.Cheat, word string) bool {
	folder := strings.ToLower(filepath.Base(filepath.Dir(cheat.File)))
	file := strings.ToLower(strings.TrimSuffix(filepath.Base(cheat.File), filepath.Ext(cheat.File)))

	if strings.Contains(folder, word) ||
		strings.Contains(file, word) ||
		strings.Contains(strings.ToLower(cheat.Header), word) ||
		strings.Contains(strings.ToLower(cheat.Description), word) ||
		strings.Contains(strings.ToLower(cheat.Command), word) {
		return true
	}

	for _, tag := range cheat.Tags {
		if strings.Contains(strings.ToLower(tag), word) {
			return true
		}
	}
	return false
}

// FindExactHeaderMatch returns the first cheat with an exact case-insensitive header match.
func FindExactHeaderMatch(cheats []*parser.Cheat, query string) *parser.Cheat {
	for _, mc := range cheats {
		if strings.EqualFold(mc.Header, query) {
			return mc
		}
	}
	return nil
}

// MatchesAllWords returns true if the given text contains all the given words.
// The text and words must be pre-lowercased.
func MatchesAllWords(text string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}
