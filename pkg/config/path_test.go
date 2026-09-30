package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfiguredHomePaths(t *testing.T) {
	old := *Get()
	t.Cleanup(func() { *Get() = old; viper.Reset() })
	file := filepath.Join(t.TempDir(), "cheatmd.yaml")
	if err := os.WriteFile(file, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, want string }{
		{"~", home},
		{"~/cheats", filepath.Join(home, "cheats")},
		{"~/first/../cheats", home + "/first/../cheats"},
		{"~,/second", home + ",/second"},
		{"~/first,/second/../third", home + "/first,/second/../third"},
		{"~/first,/second/../../third", home + "/first,/second/../../third"},
		{"~other/cheats", "~other/cheats"},
		{"cheats/~archive", "cheats/~archive"},
		{"$HOME/cheats", "$HOME/cheats"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			viper.Reset()
			viper.SetConfigFile(file)
			t.Setenv("CHEATMD_PATH", tc.path)
			if err := Init(); err != nil {
				t.Fatal(err)
			}
			if got := Get().Path; got != tc.want {
				t.Errorf("configured path = %q, want %q", got, tc.want)
			}
			if got := CheatsInstallDir(); got != tc.want {
				t.Errorf("install path = %q, want %q", got, tc.want)
			}
		})
	}
}
