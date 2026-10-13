package executor

import "testing"

func TestEvaluateConditionConjunction(t *testing.T) {
	for _, tc := range []struct {
		name, condition string
		scope           map[string]string
		want            bool
	}{
		{"both true", "$env == prod && $region == eu", map[string]string{"env": "prod", "region": "eu"}, true},
		{"first false", "$env == prod && $region == eu", map[string]string{"env": "dev", "region": "eu"}, false},
		{"second false", "$env == prod && $region == eu", map[string]string{"env": "prod", "region": "us"}, false},
		{"both false", "$env == prod && $region == eu", map[string]string{"env": "dev", "region": "us"}, false},
		{"mixed operators", "$env != dev && $region == eu", map[string]string{"env": "prod", "region": "eu"}, true},
		{"three predicates", "$env == prod && $region != us && $ready", map[string]string{"env": "prod", "region": "eu", "ready": "yes"}, true},
		{"empty middle predicate", "yes && && yes", nil, false},
		{"empty last predicate", "yes &&", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateCondition(tc.condition, tc.scope); got != tc.want {
				t.Fatalf("EvaluateCondition(%q) = %v, want %v", tc.condition, got, tc.want)
			}
		})
	}
}

func TestEvaluateConditionLiteralOperands(t *testing.T) {
	for _, tc := range []struct {
		name, condition, value, expected string
		want                             bool
	}{
		{"equality containing operators", "$value == $expected", "a == b != c && d", "a == b != c && d", true},
		{"unequal values containing operators", "$value == $expected", "a == b", "a != b", false},
		{"inequality containing equality", "$value != other", "a == b", "", true},
		{"inequality containing operators", "$value != $expected", "a != b && c", "a != b && c", false},
		{"operator text is truthy", "$value", "==", "", true},
		{"conjunction text is truthy", "$value", "&&", "", true},
		{"nonempty value is truthy", "$value", "ready", "", true},
		{"empty value is false", "$value", "", "", false},
		{"whitespace value stays truthy", "$value", " \t ", "", true},
		{"literal whitespace is false", " \t ", "", "", false},
		{"comparison trims whitespace", "$value == $expected", " value ", "value", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope := map[string]string{"value": tc.value, "expected": tc.expected}
			if got := EvaluateCondition(tc.condition, scope); got != tc.want {
				t.Fatalf("EvaluateCondition(%q) = %v, want %v", tc.condition, got, tc.want)
			}
		})
	}
}
