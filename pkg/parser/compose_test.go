package parser

import (
	"reflect"
	"testing"
)

func TestExtractVars(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "dollar syntax",
			command: "curl -X POST $url -H 'Auth: $token'",
			want:    []string{"url", "token"},
		},
		{
			name:    "angle syntax",
			command: "nmap -p <port> <host>",
			want:    []string{"port", "host"},
		},
		{
			name:    "mixed syntax",
			command: "ssh $user@<host> -p $port",
			want:    []string{"user", "host", "port"},
		},
		{
			name:    "duplicates removed",
			command: "echo $var <var> $var2 $var",
			want:    []string{"var", "var2"},
		},
		{
			name:    "no variables",
			command: "ls -la /tmp",
			want:    nil,
		},
		{
			name:    "escaped variables",
			command: "echo \\$var",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVars(tt.command, true, true)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractVars() = %v, want %v", got, tt.want)
			}
		})
	}
}
