package executor

import (
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// mockClipboard implements Clipboard interface for testing
type mockClipboard struct {
	lastCopied string
}

func (m *mockClipboard) Copy(text string) error {
	m.lastCopied = text
	return nil
}

func TestOutputWithMode_Copy(t *testing.T) {
	mockClip := &mockClipboard{}
	exec := NewExecutor(parser.NewCheatIndex()).WithClipboard(mockClip)

	testText := "echo hello"
	err := exec.OutputWithMode(testText, OutputCopy)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockClip.lastCopied != testText {
		t.Errorf("expected clipboard to have %q, got %q", testText, mockClip.lastCopied)
	}
}

func TestSubstituteVars_Dollar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		scope    map[string]string
		expected string
	}{
		{
			name:     "simple substitution",
			input:    "echo $var",
			scope:    map[string]string{"var": "hello"},
			expected: "echo hello",
		},
		{
			name:     "multiple substitutions",
			input:    "curl -u $user:$pass $url",
			scope:    map[string]string{"user": "admin", "pass": "secret", "url": "http://localhost"},
			expected: "curl -u admin:secret http://localhost",
		},
		{
			name:     "prefix collision prevention",
			input:    "echo $username and $user",
			scope:    map[string]string{"user": "bob", "username": "alice"},
			expected: "echo alice and bob",
		},
		{
			name:     "substitution happens inside shell quotes",
			input:    "curl -H '$Authorization_Bearer_token'",
			scope:    map[string]string{"Authorization_Bearer_token": "Authorization: Bearer token"},
			expected: "curl -H 'Authorization: Bearer token'",
		},
		{
			name:     "missing var is left as is",
			input:    "echo $missing",
			scope:    map[string]string{"other": "val"},
			expected: "echo $missing",
		},
		{
			name:     "angle brackets ignored in dollar mode",
			input:    "echo <host>",
			scope:    map[string]string{"host": "10.0.0.1"},
			expected: "echo <host>",
		},
		{
			name:     "braced shell vars ignored in dollar mode",
			input:    "echo ${host} $host",
			scope:    map[string]string{"host": "10.0.0.1"},
			expected: "echo ${host} 10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteVars(tt.input, tt.scope, "dollar")
			if got != tt.expected {
				t.Errorf("SubstituteVars(dollar) = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSubstituteVars_Angle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		scope    map[string]string
		expected string
	}{
		{
			name:     "simple angle substitution",
			input:    "echo <host>",
			scope:    map[string]string{"host": "10.0.0.1"},
			expected: "echo 10.0.0.1",
		},
		{
			name:     "dollar ignored in angle mode",
			input:    "echo $host",
			scope:    map[string]string{"host": "10.0.0.1"},
			expected: "echo $host",
		},
		{
			name:     "multiple angle vars",
			input:    "ssh -p <port> user@<host>",
			scope:    map[string]string{"port": "2222", "host": "10.0.0.1"},
			expected: "ssh -p 2222 user@10.0.0.1",
		},
		{
			name:     "preserves shell vars",
			input:    "echo $HOME <target>",
			scope:    map[string]string{"target": "10.0.0.1"},
			expected: "echo $HOME 10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteVars(tt.input, tt.scope, "angle")
			if got != tt.expected {
				t.Errorf("SubstituteVars(angle) = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSubstituteVars_Both(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		scope    map[string]string
		expected string
	}{
		{
			name:     "dollar and angle both replaced",
			input:    "curl $host:<port>",
			scope:    map[string]string{"host": "http://10.0.0.1", "port": "443"},
			expected: "curl http://10.0.0.1:443",
		},
		{
			name:     "same var in both syntaxes",
			input:    "echo $host and <host>",
			scope:    map[string]string{"host": "10.0.0.1"},
			expected: "echo 10.0.0.1 and 10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteVars(tt.input, tt.scope, "both")
			if got != tt.expected {
				t.Errorf("SubstituteVars(both) = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildFinalCommand(t *testing.T) {
	cheat := &parser.Cheat{
		Command: "echo $greeting \\$HOME",
		Scope: map[string]string{
			"greeting": "hello",
		},
	}

	exec := NewExecutor(parser.NewCheatIndex())
	got := exec.BuildFinalCommand(cheat)
	want := "echo hello $HOME"

	if got != want {
		t.Errorf("BuildFinalCommand() = %q, want %q", got, want)
	}
}

func TestRunShell(t *testing.T) {
	exec := NewExecutor(parser.NewCheatIndex())
	exec.shell = "bash" // enforce shell for test

	out, err := exec.RunShell("echo -n 'hello world'")
	if err != nil {
		t.Fatalf("RunShell failed: %v", err)
	}
	if out != "hello world" {
		t.Errorf("RunShell() = %q, want 'hello world'", out)
	}

	// Test trailing whitespace trimming
	out, err = exec.RunShell("echo '  hello whitespace  '")
	if err != nil {
		t.Fatalf("RunShell failed: %v", err)
	}
	if out != "hello whitespace" {
		t.Errorf("RunShell() trimmed = %q, want 'hello whitespace'", out)
	}

	// Test shell error
	_, err = exec.RunShell("exit 1")
	if err == nil {
		t.Errorf("expected RunShell to fail on exit 1")
	}
}

func TestExecute(t *testing.T) {
	exec := NewExecutor(parser.NewCheatIndex())
	exec.shell = "bash"

	// Just verifying it doesn't error on a simple command
	err := exec.Execute("true")
	if err != nil {
		t.Errorf("Execute('true') failed: %v", err)
	}

	err = exec.Execute("false")
	if err == nil {
		t.Errorf("Execute('false') expected error, got nil")
	}
}

func TestOutputWithMode_Exec(t *testing.T) {
	exec := NewExecutor(parser.NewCheatIndex())
	exec.shell = "bash"

	err := exec.OutputWithMode("true", OutputExec)
	if err != nil {
		t.Errorf("OutputWithMode(OutputExec) failed: %v", err)
	}
}

func TestOutputWithMode_Print(t *testing.T) {
	exec := NewExecutor(parser.NewCheatIndex())
	// Print does nothing but return nil
	err := exec.OutputWithMode("echo foo", OutputPrint)
	if err != nil {
		t.Errorf("OutputWithMode(OutputPrint) failed: %v", err)
	}
}
