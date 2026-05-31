package resolver

import (
	"regexp"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// FindMatchingCheat finds a cheat whose command pattern matches the input.
// It builds a regex from the cheat command (replacing $var with capture groups)
// and returns the most specific match.
func FindMatchingCheat(cheats []*parser.Cheat, input string) *parser.Cheat {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var best *parser.Cheat
	bestScore := 0
	for _, cheat := range cheats {
		pattern, _, score := buildMatchPatternWithScore(cheat.Command)
		if pattern.MatchString(input) && score > bestScore {
			best = cheat
			bestScore = score
		}
	}
	return best
}

// buildMatchPattern converts a command template to a regex pattern for matching.
func buildMatchPattern(cmd string) (*regexp.Regexp, []string) {
	pattern, varOrder, _ := buildMatchPatternWithScore(cmd)
	return pattern, varOrder
}

func buildMatchPatternWithScore(cmd string) (*regexp.Regexp, []string, int) {
	allMatches := extractVariableMatches(cmd)
	var varOrder []string
	literalScore := 0

	var result strings.Builder
	result.WriteString(`^\s*`)
	lastEnd := 0

	for i, match := range allMatches {
		varStart, varEnd, varName := extractMatchBounds(cmd, match)

		if varStart > lastEnd {
			literal := cmd[lastEnd:varStart]
			literalScore += len(strings.TrimSpace(literal))
			result.WriteString(regexp.QuoteMeta(literal))
		}

		varOrder = append(varOrder, varName)
		lastEnd = appendRegexForVariable(&result, cmd, varStart, varEnd, i, allMatches)
	}

	if lastEnd < len(cmd) {
		literal := cmd[lastEnd:]
		literalScore += len(strings.TrimSpace(literal))
		result.WriteString(regexp.QuoteMeta(literal))
	}
	result.WriteString(`\s*$`)

	re, err := regexp.Compile(result.String())
	if err != nil {
		return regexp.MustCompile(`^$`), nil, 0
	}
	return re, varOrder, literalScore
}

func extractVariableMatches(cmd string) [][]int {
	var parts []string
	if config.VarSyntaxAllowsDollar() {
		parts = append(parts, `\$(\w+)`)
	}
	if config.VarSyntaxAllowsAngle() {
		parts = append(parts, `<(\w+)>`)
	}
	if len(parts) == 0 {
		parts = append(parts, `\$(\w+)`)
	}
	varPattern := regexp.MustCompile(strings.Join(parts, "|"))
	return varPattern.FindAllStringSubmatchIndex(cmd, -1)
}

func extractMatchBounds(cmd string, match []int) (int, int, string) {
	varStart := match[0]
	varEnd := match[1]
	var varName string
	for j := 2; j < len(match); j += 2 {
		if match[j] != -1 {
			varName = cmd[match[j]:match[j+1]]
			break
		}
	}
	return varStart, varEnd, varName
}

func appendRegexForVariable(result *strings.Builder, cmd string, varStart, varEnd, matchIndex int, allMatches [][]int) int {
	beforeVar := cmd[:varStart]
	afterVar := cmd[varEnd:]

	if strings.HasSuffix(beforeVar, `"`) && strings.HasPrefix(afterVar, `"`) {
		current := result.String()
		if strings.HasSuffix(current, `"`) {
			result.Reset()
			result.WriteString(current[:len(current)-1])
		}
		result.WriteString(`"([^"]*)"`)
		return varEnd + 1
	} else if strings.HasSuffix(beforeVar, `'`) && strings.HasPrefix(afterVar, `'`) {
		current := result.String()
		if strings.HasSuffix(current, `'`) {
			result.Reset()
			result.WriteString(current[:len(current)-1])
		}
		result.WriteString(`'([^']*)'`)
		return varEnd + 1
	}

	isLastVar := matchIndex == len(allMatches)-1
	remainingText := strings.TrimSpace(cmd[varEnd:])
	if isLastVar && remainingText == "" {
		result.WriteString(`(.+)`)
	} else {
		nextLiteralStart := varEnd
		nextLiteralEnd := len(cmd)
		if matchIndex+1 < len(allMatches) {
			nextLiteralEnd = allMatches[matchIndex+1][0]
		}
		nextLiteral := strings.TrimSpace(cmd[nextLiteralStart:nextLiteralEnd])

		if nextLiteral != "" {
			result.WriteString(`(.+?)`)
		} else {
			result.WriteString(`(\S+)`)
		}
	}
	return varEnd
}

// PrefillScopeFromMatch extracts variable values from the matched command and
// writes them into cheat.Scope.
func PrefillScopeFromMatch(cheat *parser.Cheat, input string) {
	input = strings.TrimSpace(input)
	pattern, varNames := buildMatchPattern(cheat.Command)
	if pattern == nil || len(varNames) == 0 {
		return
	}

	matches := pattern.FindStringSubmatch(input)
	if matches == nil {
		return
	}

	if cheat.Scope == nil {
		cheat.Scope = make(map[string]string)
	}

	for i, name := range varNames {
		if i+1 >= len(matches) {
			continue
		}
		if _, exists := cheat.Scope[name]; !exists {
			cheat.Scope[name] = matches[i+1]
		}
	}
}

// InferDependentVars reverse-engineers dependent variables from literal values.
func InferDependentVars(cheat *parser.Cheat, index *parser.CheatIndex) {
	if len(cheat.Scope) == 0 {
		return
	}

	varDefs := executor.CollectVarDefinitions(cheat, index)

	changed := true
	for changed {
		changed = false
		for varName, prefillValue := range cheat.Scope {
			defs, ok := varDefs[varName]
			if !ok {
				continue
			}

			if inferFromDefinitions(cheat, defs, prefillValue) {
				changed = true
			}
		}
	}
}

func inferFromDefinitions(cheat *parser.Cheat, defs []parser.VarDef, prefillValue string) bool {
	changed := false
	for _, def := range defs {
		if def.Literal == "" || def.Condition == "" {
			continue
		}

		condVar, condOp, condValue := parseCondition(def.Condition)
		if condVar == "" {
			continue
		}

		if _, exists := cheat.Scope[condVar]; exists {
			continue
		}

		literalResult := executor.SubstituteVars(def.Literal, cheat.Scope, "dollar")

		if strings.Contains(literalResult, "$") {
			extracted := extractEmbeddedVars(def.Literal, prefillValue, cheat.Scope)
			for k, v := range extracted {
				if _, exists := cheat.Scope[k]; !exists {
					cheat.Scope[k] = v
					changed = true
				}
			}
			literalResult = executor.SubstituteVars(def.Literal, cheat.Scope, "dollar")
		}

		if literalResult == prefillValue && condOp == "==" {
			cheat.Scope[condVar] = condValue
			changed = true
		}
	}
	return changed
}

func parseCondition(cond string) (varName, op, value string) {
	cond = strings.TrimSpace(cond)

	if idx := strings.Index(cond, "=="); idx != -1 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		if strings.HasPrefix(left, "$") {
			return left[1:], "==", right
		}
	}

	if idx := strings.Index(cond, "!="); idx != -1 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		if strings.HasPrefix(left, "$") {
			return left[1:], "!=", right
		}
	}

	return "", "", ""
}

func extractEmbeddedVars(template, actual string, existingScope map[string]string) map[string]string {
	result := make(map[string]string)

	pattern := template
	for k, v := range existingScope {
		pattern = strings.ReplaceAll(pattern, "$"+k, regexp.QuoteMeta(v))
	}

	varPattern := regexp.MustCompile(`\$(\w+)`)
	varMatches := varPattern.FindAllStringSubmatchIndex(pattern, -1)
	if len(varMatches) == 0 {
		return result
	}

	var regexParts strings.Builder
	regexParts.WriteString("^")
	lastEnd := 0
	var varNames []string

	for i, match := range varMatches {
		varStart := match[0]
		varEnd := match[1]
		varName := pattern[match[2]:match[3]]

		if varStart > lastEnd {
			regexParts.WriteString(regexp.QuoteMeta(pattern[lastEnd:varStart]))
		}

		if i == len(varMatches)-1 && varEnd == len(pattern) {
			regexParts.WriteString(`(.+)`)
		} else {
			regexParts.WriteString(`(.+?)`)
		}
		varNames = append(varNames, varName)
		lastEnd = varEnd
	}
	if lastEnd < len(pattern) {
		regexParts.WriteString(regexp.QuoteMeta(pattern[lastEnd:]))
	}
	regexParts.WriteString("$")

	re, err := regexp.Compile(regexParts.String())
	if err != nil {
		return result
	}

	matches := re.FindStringSubmatch(actual)
	if matches == nil {
		return result
	}

	for i, name := range varNames {
		if i+1 < len(matches) {
			result[name] = matches[i+1]
		}
	}

	return result
}
