package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/spf13/viper"
)

func TestCLIConfigurationErrors(t *testing.T) {
	oldConfig := *config.Get()
	benchmark := rootCmd.PersistentFlags().Lookup("benchmark")
	oldBenchmark, oldChanged := benchmark.Value.String(), benchmark.Changed
	t.Cleanup(func() {
		*config.Get() = oldConfig
		viper.Reset()
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		_ = benchmark.Value.Set(oldBenchmark)
		benchmark.Changed = oldChanged
	})

	dir := t.TempDir()
	configPath := filepath.Join(dir, "cheatmd.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "greeting.md")
	if err := os.WriteFile(path, []byte("# Greeting\n```sh\necho hello\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, command := range []struct {
		name string
		args []string
		want string
	}{
		{"root", []string{"--benchmark", path}, "Loaded 1 cheats"},
		{"dump", []string{"dump", path}, `"command": "echo hello"`},
		{"widget", []string{"widget", "bash"}, "_cheatmd_widget"},
	} {
		t.Run(command.name, func(t *testing.T) {
			for _, historyMax := range []string{"not-an-integer", "42"} {
				t.Run(historyMax, func(t *testing.T) {
					viper.Reset()
					_ = benchmark.Value.Set("false")
					benchmark.Changed = false
					viper.SetConfigFile(configPath)
					t.Setenv("CHEATMD_HISTORY_MAX", historyMax)
					var out, stderr bytes.Buffer
					rootCmd.SetOut(&out)
					rootCmd.SetErr(&stderr)
					rootCmd.SetArgs(command.args)
					err := rootCmd.Execute()
					if historyMax == "not-an-integer" {
						if err == nil || !strings.Contains(err.Error(), "history_max") {
							t.Errorf("expected history_max config error, got %v", err)
						}
						if out.Len() != 0 {
							t.Errorf("command ran despite invalid configuration: %s", out.String())
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(out.String(), command.want) {
						t.Errorf("output %q does not contain %q", out.String(), command.want)
					}
					if got := config.Get().HistoryMax; got != 42 {
						t.Errorf("history_max = %d, want 42", got)
					}
				})
			}
		})
	}
}
