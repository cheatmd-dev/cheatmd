package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/spf13/viper"
)

func TestHomePathInputs(t *testing.T) {
	old := *config.Get()
	t.Cleanup(func() { *config.Get() = old; viper.Reset() })
	dir := t.TempDir()
	path := filepath.Join(dir, "greeting.md")
	if err := os.WriteFile(path, []byte("# Greeting\n```sh\necho hello\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(home, path)
	if err != nil {
		t.Fatal(err)
	}
	homePath := "~/" + filepath.ToSlash(relative)
	expandedPath := home + "/" + filepath.ToSlash(relative)
	configFile := filepath.Join(dir, "cheatmd.yaml")
	if err := os.WriteFile(configFile, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, configured := range []bool{false, true} {
		name := "explicit"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			args, composeFile := []string{homePath}, homePath
			viper.Reset()
			viper.SetConfigFile(configFile)
			if configured {
				t.Setenv("CHEATMD_PATH", homePath)
				args, composeFile = nil, ""
			} else {
				t.Setenv("CHEATMD_PATH", ".")
			}
			if err := config.Init(); err != nil {
				t.Fatal(err)
			}
			t.Run("root", func(t *testing.T) {
				got, err := resolveCheatPath(args)
				if err != nil || got != path {
					t.Fatalf("resolved path = %q, %v; want %q", got, err, path)
				}
			})
			t.Run("dump", func(t *testing.T) {
				index, err := parseDumpIndex(args)
				if err != nil {
					t.Fatal(err)
				}
				if len(index.Cheats) != 1 || index.Cheats[0].Command != "echo hello" {
					t.Fatalf("unexpected cheats: %+v", index.Cheats)
				}
			})
			t.Run("compose", func(t *testing.T) {
				got, err := determineComposeTargetFile(composeFile)
				if err != nil || got != expandedPath {
					t.Fatalf("compose target = %q, %v; want %q", got, err, expandedPath)
				}
			})
		})
	}

	for _, tc := range []struct{ name, path, want string }{
		{"absolute second path", homePath + ",/unused/second.md", expandedPath},
		{"parent in second path", homePath + ",/second/../../third.md", expandedPath},
		{"bare home first", "~,/second", filepath.Join(home, "snippets.md")},
	} {
		t.Run("compose configured path list/"+tc.name, func(t *testing.T) {
			viper.Reset()
			viper.SetConfigFile(configFile)
			t.Setenv("CHEATMD_PATH", tc.path)
			if err := config.Init(); err != nil {
				t.Fatal(err)
			}
			got, err := determineComposeTargetFile("")
			if err != nil || got != tc.want {
				t.Fatalf("compose target = %q, %v; want first path %q", got, err, tc.want)
			}
		})
	}
}
