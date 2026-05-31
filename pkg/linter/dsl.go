package linter

import (
	"bufio"
	"os"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func findDSLRef(file, keyword, name string) (int, int) {
	f, err := os.Open(file)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	inCheat := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		if !inCheat {
			if content, ok := parser.ParseCheatSingleLine(line); ok {
				if dslLineMatches(content, keyword, name) {
					return lineNo, stringColumn(raw, name)
				}
				continue
			}
			if parser.IsCheatStart(line) {
				inCheat = true
			}
			continue
		}

		if parser.IsCheatEnd(line) {
			inCheat = false
			continue
		}
		if dslLineMatches(line, keyword, name) {
			return lineNo, stringColumn(raw, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0
	}

	return 0, 0
}

func dslLineMatches(line, keyword, name string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}
	gotKeyword, rest := parser.SplitFirstWord(line)
	gotName, _ := parser.SplitFirstWord(rest)
	return gotKeyword == keyword && gotName == name
}

func stringColumn(line, needle string) int {
	if idx := strings.Index(line, needle); idx >= 0 {
		return idx + 1
	}
	return 1
}
