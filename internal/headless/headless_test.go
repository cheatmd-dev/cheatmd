package headless

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

type mockHeadlessExecutor struct {
	shellResult string
	shellErr    error
	finalCmd    string
	outputErr   error
}

func (m *mockHeadlessExecutor) RunShell(command string) (string, error) {
	return m.shellResult, m.shellErr
}

func (m *mockHeadlessExecutor) BuildFinalCommand(cheat *parser.Cheat) string {
	if m.finalCmd != "" {
		return m.finalCmd
	}
	return cheat.Command
}

func (m *mockHeadlessExecutor) OutputWithMode(command string, mode executor.OutputMode) error {
	return m.outputErr
}

func TestRunHeadlessSuccess(t *testing.T) {
	cheat := &parser.Cheat{
		File:    "test.md",
		Header:  "Test Headless",
		Command: "ping $ip",
		Vars: []parser.VarDef{
			{Name: "ip"},
		},
	}

	index := parser.NewCheatIndex()
	index.Cheats = []*parser.Cheat{cheat}

	// 1. Mock Stdin and Stdout Pipes
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = rIn

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = wOut

	// 2. Prepare Mock responses
	resBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"values": map[string]string{
				"ip": "127.0.0.1",
			},
		},
		"id": 1,
	})
	_, _ = wIn.Write(append(resBytes, '\n'))
	_ = wIn.Close() // Close stdin writing so Scanner doesn't hang

	exec := &mockHeadlessExecutor{
		finalCmd: "ping 127.0.0.1",
	}

	// Channel to capture stdout asynchronously
	outChan := make(chan string)
	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, rOut)
		outChan <- buf.String()
	}()

	// 3. Run Headless function
	err = Run(index, exec, "Test Headless", "")
	_ = wOut.Close() // Close stdout writing so copy goroutine finishes

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	capturedStdout := <-outChan

	// 4. Parse captured stdout lines
	lines := strings.Split(strings.TrimSpace(capturedStdout), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 output frames, got: %d (%q)", len(lines), capturedStdout)
	}

	// First line must be prompt request
	var promptReq struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Variables []struct {
				Name string `json:"name"`
			} `json:"variables"`
		} `json:"params"`
		Id int `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &promptReq); err != nil {
		t.Fatalf("failed to unmarshal prompt request: %v", err)
	}
	if promptReq.Method != "prompt" || len(promptReq.Params.Variables) == 0 || promptReq.Params.Variables[0].Name != "ip" {
		t.Errorf("unexpected prompt request: %+v", promptReq)
	}

	// Second line must be completed execution frame
	var completed struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Status  string `json:"status"`
			Command string `json:"command"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &completed); err != nil {
		t.Fatalf("failed to unmarshal completed frame: %v", err)
	}
	if completed.Method != "completed" || completed.Params.Status != "success" || completed.Params.Command != "ping 127.0.0.1" {
		t.Errorf("unexpected completed frame: %+v", completed)
	}
}

func TestRunHeadlessConditionalDependencies(t *testing.T) {
	cheat := &parser.Cheat{
		File:    "test.md",
		Header:  "Test Dynamic Resolution",
		Command: "curl https://$domain/api -u $user $auth_flags",
		Vars: []parser.VarDef{
			{Name: "domain"},
			{Name: "user"},
			{Name: "auth_method"},
			{Name: "credential", Condition: "$auth_method != oauth"},
			{Name: "auth_flags", Literal: "-p $credential", Condition: "$auth_method == password"},
			{Name: "auth_flags", Literal: "-H 'Authorization: Bearer $credential'", Condition: "$auth_method == token"},
			{Name: "auth_flags", Literal: "--oauth2-bearer", Condition: "$auth_method == oauth"},
		},
	}

	index := parser.NewCheatIndex()
	index.Cheats = []*parser.Cheat{cheat}

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = rIn

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = wOut

	// 1. Prepare Stdin replies for multi-pass dynamic prompts
	// Pass 1: resolve domain, user, and auth_method
	resBytes1, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"values": map[string]string{
				"domain":      "api.example.com",
				"user":        "admin",
				"auth_method": "password",
			},
		},
		"id": 1,
	})

	// Pass 2: resolve credential (now that auth_method == password is known)
	resBytes2, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"values": map[string]string{
				"credential": "secret123",
			},
		},
		"id": 1,
	})

	_, _ = wIn.Write(append(resBytes1, '\n'))
	_, _ = wIn.Write(append(resBytes2, '\n'))
	_ = wIn.Close()

	exec := &mockHeadlessExecutor{
		finalCmd: "curl https://api.example.com/api -u admin -p secret123",
	}

	outChan := make(chan string)
	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, rOut)
		outChan <- buf.String()
	}()

	err = Run(index, exec, "Test Dynamic Resolution", "")
	_ = wOut.Close()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	capturedStdout := <-outChan
	lines := strings.Split(strings.TrimSpace(capturedStdout), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 output frames (2 prompts + 1 completed), got: %d (%q)", len(lines), capturedStdout)
	}

	// Parse first completed frame and verify it matches the correct final execution command
	var completed struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Status  string `json:"status"`
			Command string `json:"command"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &completed); err != nil {
		t.Fatalf("failed to unmarshal completed frame: %v", err)
	}
	if completed.Params.Command != "curl https://api.example.com/api -u admin -p secret123" {
		t.Errorf("unexpected command: %q", completed.Params.Command)
	}
}

func TestRunHeadlessRPCLimit(t *testing.T) {
	cheat := &parser.Cheat{
		File:    "test.md",
		Header:  "Test Headless Limit",
		Command: "echo $payload",
		Vars: []parser.VarDef{
			{Name: "payload"},
		},
	}

	index := parser.NewCheatIndex()
	index.Cheats = []*parser.Cheat{cheat}

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = rIn

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = wOut

	largePayload := strings.Repeat("a", 70000)

	resBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"values": map[string]string{
				"payload": largePayload,
			},
		},
		"id": 1,
	})
	go func() {
		_, _ = wIn.Write(append(resBytes, '\n'))
		_ = wIn.Close()
	}()

	exec := &mockHeadlessExecutor{
		finalCmd: "echo " + largePayload,
	}

	outChan := make(chan string)
	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, rOut)
		outChan <- buf.String()
	}()

	err = Run(index, exec, "Test Headless Limit", "")
	_ = wOut.Close()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	<-outChan
}
