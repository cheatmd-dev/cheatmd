package parser

// WalkVars visits complete variable references in source order. Offsets are byte
// offsets into command. Dollar references escaped by an odd number of preceding
// backslashes and shell ${...} forms are left to the shell.
func WalkVars(command string, allowDollar, allowAngle bool, visit func(start, end int, name string)) {
	backslashes := 0
	for i := 0; i < len(command); i++ {
		if command[i] == '\\' {
			backslashes++
			continue
		}
		escaped := backslashes%2 != 0
		backslashes = 0
		dollar := command[i] == '$' && allowDollar && !escaped
		angle := command[i] == '<' && allowAngle
		if !dollar && !angle {
			continue
		}
		start := i
		j := i + 1
		if j >= len(command) || !IsVarChar(command[j], true) {
			continue
		}
		j++
		for j < len(command) && IsVarChar(command[j], false) {
			j++
		}
		name := command[i+1 : j]
		if angle {
			if j >= len(command) || command[j] != '>' {
				continue
			}
			j++
		}
		visit(start, j, name)
		i = j - 1
	}
}

// ExtractVars finds all variables in a command string. It respects the provided
// flags for dollar ($var) and angle bracket (<var>) syntaxes. It returns a
// deduplicated list of variable names in the exact order they appear from
// left to right.
func ExtractVars(command string, allowDollar, allowAngle bool) []string {
	varMap := make(map[string]bool)
	var vars []string

	WalkVars(command, allowDollar, allowAngle, func(_, _ int, name string) {
		if !varMap[name] {
			varMap[name] = true
			vars = append(vars, name)
		}
	})

	return vars
}
