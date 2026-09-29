package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestLiteralDefinitionUsesDollarSyntaxWithAngleCommand(t *testing.T) {
	old := config.Get().VarSyntax
	t.Cleanup(func() { config.Get().VarSyntax = old })
	config.Get().VarSyntax = "angle"
	m, index := setupTestModel()
	cheat := index.Cheats[0]
	cheat.Command = "echo <b>"
	cheat.Vars = []parser.VarDef{
		{Name: "a", Literal: "alpha"},
		{Name: "b", Literal: "$a-beta"},
	}
	m.selected = cheat

	m.startVarResolution()

	if got := cheat.Scope["b"]; got != "alpha-beta" {
		t.Fatalf("literal value = %q, want alpha-beta", got)
	}
	if got := executor.NewExecutor(index).BuildFinalCommand(cheat); got != "echo alpha-beta" {
		t.Fatalf("command = %q, want echo alpha-beta", got)
	}
}

type definitionShellExecutor struct {
	mockExecutor
	command string
}

func (e *definitionShellExecutor) RunShell(command string) (string, error) {
	e.command = command
	return "alpha-beta", nil
}

func TestShellDefinitionUsesDollarSyntaxWithAngleCommand(t *testing.T) {
	old := config.Get().VarSyntax
	t.Cleanup(func() { config.Get().VarSyntax = old })
	config.Get().VarSyntax = "angle"
	m, index := setupTestModel()
	cheat := index.Cheats[0]
	cheat.Command = "echo <b>"
	cheat.Vars = []parser.VarDef{
		{Name: "a", Literal: "alpha"},
		{Name: "b", Shell: "printf '%s' '$a-beta'"},
	}
	shell := &definitionShellExecutor{}
	m.executor = shell
	m.selected = cheat

	cmd := m.startVarResolution()
	if cmd == nil {
		t.Fatal("shell definition returned no command")
	}
	m.Update(cmd())
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if shell.command != "printf '%s' 'alpha-beta'" {
		t.Fatalf("shell command = %q, want substituted dollar reference", shell.command)
	}
	if got := executor.NewExecutor(index).BuildFinalCommand(cheat); got != "echo alpha-beta" {
		t.Fatalf("command = %q, want echo alpha-beta", got)
	}
}
