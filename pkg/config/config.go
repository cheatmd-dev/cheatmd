package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// defaultConfigTemplate is the starter config written by WriteDefaultConfig
// during first-run setup. Keeping it embedded keeps generated configs in sync
// with the documented example.
//
//go:embed default_config.yaml
var defaultConfigTemplate []byte

// ============================================================================
// Configuration Types
// ============================================================================

// Config holds the application configuration
type Config struct {
	Path                string `mapstructure:"path"`
	RegistryURL         string `mapstructure:"registry_url"`
	Output              string `mapstructure:"output"`
	Shell               string `mapstructure:"shell"`
	Editor              string `mapstructure:"editor"`
	PreHook             string `mapstructure:"pre_hook"`
	PostHook            string `mapstructure:"post_hook"`
	RequireCheatBlock   bool   `mapstructure:"require_cheat_block"`
	AllowUndeclaredVars bool   `mapstructure:"allow_undeclared_vars"`
	VarSyntax           string `mapstructure:"var_syntax"`
	AutoSelect          bool   `mapstructure:"auto_select"`
	AutoContinue        bool   `mapstructure:"auto_continue"`

	// Keybindings
	KeyWidget     string `mapstructure:"key_widget"`
	KeyOpen       string `mapstructure:"key_open"`
	KeySubstitute string `mapstructure:"key_substitute"`
	KeyPreview    string `mapstructure:"key_preview"`
	KeyHistory    string `mapstructure:"key_history"`

	// Execution history
	HistoryFile string `mapstructure:"history_file"`
	HistoryMax  int    `mapstructure:"history_max"`

	// Substitute search
	SubstituteSources []string `mapstructure:"substitute_sources"`

	// Display options
	ShowFolder    bool `mapstructure:"show_folder"`
	ShowFile      bool `mapstructure:"show_file"`
	PreviewHeight int  `mapstructure:"preview_height"`

	// Colors
	Colors ColorConfig `mapstructure:",squash"`

	// Columns
	Columns ColumnConfig `mapstructure:",squash"`
}

// ColorConfig holds all color settings
type ColorConfig struct {
	Header   string `mapstructure:"color_header"`
	Command  string `mapstructure:"color_command"`
	Desc     string `mapstructure:"color_desc"`
	Path     string `mapstructure:"color_path"`
	Border   string `mapstructure:"color_border"`
	Cursor   string `mapstructure:"color_cursor"`
	Selected string `mapstructure:"color_selected"`
	Dim      string `mapstructure:"color_dim"`
}

// ColumnConfig holds column width settings
type ColumnConfig struct {
	Gap     int `mapstructure:"column_gap"`
	Header  int `mapstructure:"column_header"`
	Desc    int `mapstructure:"column_desc"`
	Command int `mapstructure:"column_command"`
}

// ============================================================================
// Default Values
// ============================================================================

// Defaults for configuration
// DefaultRegistryURL is the canonical cheat-pack registry manifest. Override
// via the registry_url config key for private/self-hosted registries.
const DefaultRegistryURL = "https://raw.githubusercontent.com/cheatmd-dev/registry/main/registry.yaml"

var DefaultConfig = Config{
	Path:                ".",
	RegistryURL:         DefaultRegistryURL,
	Output:              "print",
	Shell:               "", // Set dynamically
	Editor:              "", // Empty means use system default (xdg-open/open/start)
	PreHook:             "",
	PostHook:            "",
	RequireCheatBlock:   false,
	AllowUndeclaredVars: false,
	VarSyntax:           "dollar",
	AutoSelect:          false,
	AutoContinue:        false,
	KeyWidget:           "\\C-g",  // Ctrl+G for shell widgets
	KeyOpen:             "ctrl+o", // Ctrl+O in TUI
	KeySubstitute:       "ctrl+t", // Ctrl+T opens substitute search during var resolution
	KeyPreview:          "ctrl+y", // Ctrl+Y opens markdown preview of current cheat's file
	KeyHistory:          "ctrl+h", // Ctrl+H opens execution history
	HistoryFile:         "",       // Empty -> $XDG_DATA_HOME/cheatmd/history.jsonl
	HistoryMax:          1000,
	SubstituteSources:   []string{"env", "history"},
	ShowFolder:          true,
	ShowFile:            true,
	PreviewHeight:       6,
	Colors: ColorConfig{
		Header:   "36",  // Cyan
		Command:  "32",  // Green
		Desc:     "246", // Light gray
		Path:     "33",  // Yellow
		Border:   "242", // Gray
		Cursor:   "212", // Pink
		Selected: "236", // Dark bg
		Dim:      "245", // Medium gray
	},
	Columns: ColumnConfig{
		Gap:     4,
		Header:  40,
		Desc:    40,
		Command: 60,
	},
}

// ============================================================================
// Global Config
// ============================================================================

// cfg is the global config instance
var cfg Config = DefaultConfig

// Init initializes configuration with viper
func Init() error {
	cfg = DefaultConfig // copy defaults

	shell := os.Getenv("SHELL")
	if shell != "" {
		cfg.Shell = shell
	}

	configureViper()

	var configErr error
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			configErr = fmt.Errorf("config file error: %w", err)
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	return configErr
}

// Get returns a pointer to the global configuration
func Get() *Config {
	return &cfg
}

// configureViper sets up viper configuration sources
func configureViper() {
	viper.SetConfigName("cheatmd")
	viper.SetConfigType("yaml")

	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "cheatmd"))
		viper.AddConfigPath(home)
	}

	viper.SetEnvPrefix("CHEATMD")
	viper.AutomaticEnv()
}

// ============================================================================
// Getters - Core Settings
// ============================================================================

// CheatsInstallDir returns the directory cheat packs should be installed into,
// and where installed-pack detection looks. It is the configured cheats path
// when the user has set one to a real directory; otherwise it falls back to
// DefaultCheatsDir(). This keeps "where packs land" aligned with "where cheats
// are browsed" (GetPath) instead of always using the XDG default.
//
// The default path is "." (browse the current directory); we treat that as
// "unset" for install purposes so packs never land in an arbitrary cwd.
func CheatsInstallDir() string {
	if p := Get().Path; p != "" && p != "." {
		return p
	}
	return DefaultCheatsDir()
}

// DefaultCheatsDir returns the directory where first-run setup installs
// starter cheat packs. Cheats are user data, so they live under
// $XDG_DATA_HOME/cheatmd/cheats (falling back to
// ~/.local/share/cheatmd/cheats), matching where history is stored.
func DefaultCheatsDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "cheatmd", "cheats")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "cheats"
	}
	return filepath.Join(home, ".local", "share", "cheatmd", "cheats")
}

// WriteDefaultConfig writes the embedded starter config to path, creating
// parent directories as needed. It refuses to overwrite an existing file.
//
// The template's "path:" line is rewritten to the resolved DefaultCheatsDir()
// so the config always points at the same directory first-run setup installs
// into, even when $XDG_DATA_HOME relocates it.
func WriteDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	content := withCheatsPath(string(defaultConfigTemplate), DefaultCheatsDir())

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// withCheatsPath replaces the first top-level "path:" entry in the template
// with the given cheats directory, leaving comments and other lines intact. If
// no such line exists (e.g. a hand-edited template), the template is returned
// unchanged.
func withCheatsPath(template, cheatsDir string) string {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "path:") {
			lines[i] = "path: " + cheatsDir
			break
		}
	}
	return strings.Join(lines, "\n")
}

// ============================================================================
// Helpers
// ============================================================================

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, path[1:])
	}
	return path
}

func VarSyntaxAllowsDollar() bool {
	syntax := Get().VarSyntax
	return syntax == "dollar" || syntax == "both" || syntax == ""
}

func VarSyntaxAllowsAngle() bool {
	syntax := Get().VarSyntax
	return syntax == "angle" || syntax == "both"
}

// DefaultConfigPath is the path to the user's config file
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "cheatmd.yaml"
	}
	return filepath.Join(home, ".config", "cheatmd", "cheatmd.yaml")
}
