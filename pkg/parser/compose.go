package parser

import (
	"regexp"
)

var combinedRegex = regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)|<([a-zA-Z_][a-zA-Z0-9_]*)>`)

// ExtractVars finds all variables in a command string. It respects the provided
// flags for dollar ($var) and angle bracket (<var>) syntaxes. It returns a
// deduplicated list of variable names in the exact order they appear from
// left to right.
func ExtractVars(command string, allowDollar, allowAngle bool) []string {
	varMap := make(map[string]bool)
	var vars []string

	matches := combinedRegex.FindAllStringSubmatch(command, -1)
	for _, match := range matches {
		var name string
		if match[1] != "" && allowDollar {
			name = match[1]
		} else if match[2] != "" && allowAngle {
			name = match[2]
		}

		if name != "" && !varMap[name] {
			varMap[name] = true
			vars = append(vars, name)
		}
	}

	return vars
}
